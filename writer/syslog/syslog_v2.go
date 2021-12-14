package syslog

import (
	"errors"
	"fmt"
	"github.com/weitrue/log/utils"
	"go.uber.org/atomic"
	"io"
	"io/ioutil"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"syscall"
	"time"
)

var (
	ErrLoggerBusyNow = errors.New("syslog writer is busy now, please commit slowly")
	ErrLoggerStopped = errors.New("syslog writer stopped")
	ErrCacheNearFull = errors.New("syslog: cache buffer reached near full")
	ErrCacheFull     = errors.New("syslog: cache buffer reached full")
)

/*
相较于V1，变化如下：
  1、减少了一次内存拷贝，原版会在先拼接log字符串再放入字符串队列。新版使用一个大的buffer，
     从字符串队列获取到log字符串再按原版拼接顺序copy到大的byte buffer中。
  2、本地log缓存目录支持两层目录。单层目录在当日志文件较多时，读取目录的性能很差（读取目录会读取该目录下每个文件的inode信息）。
     假设一个日志文件叫xxxx123则存放的目录为/data/syslog_buffer/123/xxxx123。
  3、支持本地log缓存总大小限制。
  4、性能有所提升，主要是减少了内存拷贝和内存碎片，减少gc的压力。
*/
type SysLogHandleV2 struct {
	priority Priority //等级
	// TODO: 多等级支持
	priorityPreStr string
	raddr          string       //连接地址
	deamon         *utils.Int32 //后台

	connPool      *connPool   // 连接池
	logChan       chan []byte //缓存队列
	limit         chan int
	dialTimeoutFn dialFunc
	waitGroup     sync.WaitGroup //并发控制
	cacheDir      string         //缓存文件
	timeout       time.Duration  //发送超时时间
	lifeTime      int64          //连接最大生存时间,默认是100毫秒

	buffer           *[]byte // 发送缓冲区
	commitBufferSize int64   // 缓冲区发送的大小
	stopLoop         chan struct{}
	cacheQuota       int64        // 本地缓存配额
	cacheSize        atomic.Int64 // 当前本地缓存总大小
}

func (S *SysLogHandleV2) isNearFull() bool {
	return S.cacheSize.Load() > (S.cacheQuota * 90 / 100)
}

func (S *SysLogHandleV2) Write(b []byte) (n int, err error) {
	if S.deamon.Load() == 0 {
		return -1, ErrLoggerStopped
	}
	// 本地缓存即将写满，说明写速度过快，或者远程syslog故障，直接返回错误
	if S.isNearFull() {
		return -1, ErrCacheNearFull
	}
	select {
	case S.logChan <- b:
	default:
		// 直接发送到远程，不一定能成功
		S.emit(b)
		return -1, ErrLoggerBusyNow
	}

	return len(b), nil
}

func (S *SysLogHandleV2) Close() error {
	S.deamon.Store(0)
	<-S.stopLoop

	// 清空数据,不处理错误
	_ = S.Sync()
	//等待所有发送结束
	S.waitGroup.Wait()
	S.connPool.Close()

	return nil
}

func (S *SysLogHandleV2) Sync() error {
	loop := true
	for loop {
		select {
		case msg := <-S.logChan:
			S.writeBuffer(msg)
		default:
			loop = false
		}
	}

	S.flushBuffer()

	return nil
}

func (S *SysLogHandleV2) loopWrite() {
	defer close(S.stopLoop)
	defer utils.CatchPanic()
	timeout := time.Millisecond * S.timeout
	scanBufferTimer := time.NewTimer(timeout)

	for S.deamon.Load() > 0 {
		select {
		// 每五分钟检查一次缓存,如果不为空,则写入文件中
		case <-scanBufferTimer.C:
			// 超时的时候检查文件，超时意味着目前日志量不多
			// 在程序空闲的时候再去做这些事情
			S.flushBuffer()
			S.scanCache(false)

		case msg := <-S.logChan:
			S.writeBuffer(msg)
		}
		scanBufferTimer.Reset(timeout)
	}
}

func (S *SysLogHandleV2) writeBuffer(msg []byte) {
	*S.buffer = append(*S.buffer, S.priorityPreStr...)
	*S.buffer = append(*S.buffer, msg...)
	if msg[len(msg)-1] != '\n' {
		*S.buffer = append(*S.buffer, '\n')
	}

	if len(*S.buffer) > int(S.commitBufferSize) {
		S.flushBuffer()
	}
}

func (S *SysLogHandleV2) flushBuffer() {
	buff := S.buffer
	S.waitGroup.Add(1)
	go func() {
		S.emit(*buff)
		PutByte(buff)
		defer S.waitGroup.Add(-1)
	}()
	S.buffer = GetByte()
}

//emit调用情况如下
//1.init方法调用scanBuffer()方法时会进入一个死循环，里面一直扫描buffer,如果扫描到buffer里有数据的时候就会调用
//2.调用CLOSE方法关闭的时候会查看buffer里有没有数据，有的话会调用
//3.scanBuffer()方法中当空闲时候去调用scanFile方法扫描文件，当文件里有东西时调用
func (S *SysLogHandleV2) emit(b []byte) {
	defer utils.CatchPanic()

	conn := S.connPool.get()
	if conn == nil {
		S.writeFile(b)
		return
	}
	conn.setTimeout()
	_, err := conn.conn.Write(b)

	if err != nil {
		_, _ = utils.ErrorOutput("syslog send fail and write file:" + err.Error())
		conn.conn.Close()
		S.writeFile(b)
		return
	}

	S.connPool.put(conn)
	return
}

//写入文件的几种情况
//1.没有可用连接时
//2.使用链接发送日志到远端失败时
//3.使用链接发送日志到远端超时时
func (S *SysLogHandleV2) writeFile(data []byte) {
	if S.cacheSize.Load()+int64(len(data)) > S.cacheQuota {
		_, _ = utils.ErrorOutput(ErrCacheFull.Error())
	}
	var err error
	defer func() {
		if err != nil {
			_, _ = utils.ErrorOutput(err.Error())
		} else {
			S.cacheSize.Add(int64(len(data)))
		}
	}()
	now := time.Now().Unix()
	dirName := fmt.Sprintf("%s/%d", S.cacheDir, now%1000)
	_, err = os.Open(dirName)
	if err != nil {
		os.Mkdir(dirName, os.ModePerm)
	}
	fileName := fmt.Sprintf("%s/%d", dirName, now)
	f, err := os.OpenFile(fileName, os.O_WRONLY|os.O_CREATE|os.O_APPEND|syscall.O_ASYNC, os.ModePerm)
	if err != nil {
		return
	}
	n, err := f.Write(data)
	if err == nil && n < len(data) {
		err = io.ErrShortWrite
	}
	if err1 := f.Close(); err == nil {
		err = err1
	}
	return
}

func (S *SysLogHandleV2) loopCacheDir(dirName string, isGetSize bool) {
	subDirs, err := ioutil.ReadDir(dirName)
	if err != nil {
		_, _ = utils.ErrorOutput(err.Error())
		return
	}
	if len(subDirs) == 0 {
		if dirName != S.cacheDir {
			os.Remove(dirName)
		}
		return
	}
	for _, f := range subDirs {
		fileName := dirName + "/" + f.Name()
		if f.IsDir() {
			S.loopCacheDir(dirName+"/"+f.Name(), isGetSize)
			continue
		}
		if isGetSize {
			info, err := os.Stat(fileName)
			if err != nil {
				_, _ = utils.ErrorOutput(err.Error())
				continue
			}
			S.cacheSize.Add(info.Size())

		} else {
			content, err := ioutil.ReadFile(fileName)
			if err != nil {
				_, _ = utils.ErrorOutput(err.Error())
				continue
			}
			if len(content) > 0 {
				S.writeBuffer(content)
				S.cacheSize.Add(-int64(len(content)))
			}
			os.Remove(fileName)
		}

	}
}

func (S *SysLogHandleV2) SetDialTimeoutFn(fn dialFunc) {
	S.dialTimeoutFn = fn
}

//在空闲时候会扫描文件
func (S *SysLogHandleV2) scanCache(isGetSize bool) {
	S.loopCacheDir(S.cacheDir, isGetSize)
}

func (S *SysLogHandleV2) init() error {
	filePath, ok := os.LookupEnv("SYSLOG_BUFFER")
	if !ok {
		filePath = "/data/syslog_buffer2"
	}

	S.cacheDir = strings.TrimSuffix(filePath, "/")

	timeOut, ok := os.LookupEnv("SYSLOG_TIMEOUT")
	if !ok {
		timeOut = "3000"
	}
	timeout, _ := strconv.Atoi(timeOut)
	S.timeout = time.Duration(timeout)

	lifeTime, ok := os.LookupEnv("SYSLOG_CONN_LIFE_TIME")
	if !ok {
		lifeTime = "100"
	}
	S.lifeTime, _ = strconv.ParseInt(lifeTime, 10, 64)
	err := os.MkdirAll(S.cacheDir, os.ModePerm)
	if err != nil {
		return err
	}
	// scan and init current cache size
	S.scanCache(true)

	commitBufferSize, ok := os.LookupEnv("SYSLOG_COMMIT_BUFFER_SIZE")
	if !ok {
		commitBufferSize = "1048576"
	}
	S.commitBufferSize, _ = strconv.ParseInt(commitBufferSize, 10, 64)

	cacheQuota, ok := os.LookupEnv("SYSLOG_CACHE_QUOTA")
	if !ok {
		S.cacheQuota = 10 * 1024 * 1024 * 1024 // 10GB
	} else {
		S.cacheQuota, _ = strconv.ParseInt(cacheQuota, 10, 64)
	}

	S.connPool = &connPool{
		connQueue:     NewQueue(30, time.Millisecond*10),
		dialTimeoutFn: S.dialTimeoutFn,
		raddr:         S.raddr,
		lifeTime:      S.lifeTime,
		timeout:       S.timeout,
	}

	err = S.connPool.createConn()
	if err != nil {
		return err
	}
	go S.loopWrite()
	return nil
}

func NewSyslogHandleV2(raddr string, priority Priority, opts ...OptionFunc) (*SysLogHandleV2, error) {
	w := &SysLogHandleV2{
		priority:       LOG_LOCAL0 + priority,
		priorityPreStr: fmt.Sprintf("<%d>", LOG_LOCAL0+priority),
		raddr:          raddr,
		deamon:         utils.NewInt32(1),
		limit:          make(chan int, 30),
		logChan:        make(chan []byte, 100000),
		buffer:         GetByte(),
		dialTimeoutFn:  net.DialTimeout,
		stopLoop:       make(chan struct{}),
	}
	for _, opt := range opts {
		opt(w)
	}
	err := w.init()
	return w, err
}
