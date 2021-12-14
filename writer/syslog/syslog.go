package syslog

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"net"
	"os"
	"strconv"
	"strings"
	"sync"
	"time"
	"unsafe"

	"github.com/weitrue/log/utils"
)

type LogHandle interface {
	io.WriteCloser
	WriteString(s string) (n int, err error)
}

type Priority int

const (
	// Severity.

	// From /usr/include/sys/syslog.h.
	// These are the same on Linux, BSD, and OS X.
	LOG_EMERG Priority = iota
	LOG_ALERT
	LOG_CRIT
	LOG_ERR
	LOG_WARNING
	LOG_NOTICE
	LOG_INFO
	LOG_DEBUG
)

const (
	// Facility.

	// From /usr/include/sys/syslog.h.
	// These are the same up to LOG_FTP on Linux, BSD, and OS X.
	LOG_KERN Priority = iota << 3
	LOG_USER
	LOG_MAIL
	LOG_DAEMON
	LOG_AUTH
	LOG_SYSLOG
	LOG_LPR
	LOG_NEWS
	LOG_UUCP
	LOG_CRON
	LOG_AUTHPRIV
	LOG_FTP
	_ // unused
	_ // unused
	_ // unused
	_ // unused
	LOG_LOCAL0
	LOG_LOCAL1
	LOG_LOCAL2
	LOG_LOCAL3
	LOG_LOCAL4
	LOG_LOCAL5
	LOG_LOCAL6
	LOG_LOCAL7
)

/*
关于SysLogHandle解释
	在InitLogger()方法中调用Dial()函数创建SysLogHandle对象

关于数据流向的理解:
	在init()方法里调用S.scanBuffer()方法，S.scanBuffer()方法会进入死循环扫描缓存数据
	每次扫描当数据量大于S.batchSize(1000)或者扫描时间大于S.linger(3)时退出扫描，
		在每次扫描期间从S.buff里拿数据，如果拿成功则往buff对象里写数据，并且将计数器(count)+1
	每次扫描结束如果计数器(count)>0则调用S.emit(buff.Bytes())方法把数据发送到远端
	如果处于空闲状态(count < S.batchSize),则会去扫描文件,因为在某些发送失败的情况下会把数据写入文件，
	在scanFile()方法中也会调用S.emit()方法把数据发送到远端
	在emit()方法中调用conn := S.getConn()获得连接，然后调用_, err := conn.conn.Write(b)向连接传输数据

关于连接池的理解：
	创建连接的两种情况
		在Dial()函数中调用了init()函数，该函数调用createConn()建立第一个连接并且放到连接池中
		在getConn函数中会判断当连接池数量为0时再次创立连接
	关闭连接的两种情况
		在getConn函数中会判断当连接是否超时(调用isOld()，目前超时时间是100毫秒，由lifeTime确定),超时则关闭连接
		在Close()中关闭连接，此方法一般不会调用到
	综上所述连接数100毫秒更新一个连接
*/
type SysLogHandle struct {
	priority Priority //等级
	// priorityPreStr syslog 前缀
	// TODO: 多等级支持
	priorityPreStr string
	raddr          string       //连接地址
	deamon         *utils.Int32 //后台

	connPool      *connPool // 连接池
	buff          *queue    //缓存队列
	dialTimeoutFn dialFunc
	limit         chan int
	waitGroup     sync.WaitGroup //并发控制
	filePath      string         //缓存文件
	batchSize     int            // 批发条数
	linger        int64          //延时等待时间
	timeout       time.Duration  //发送超时时间
	lifeTime      int64          //连接最大生存时间,默认是100毫秒

	largeBuff chan bool // 标记当前有大量缓存(高并发状态,使其会迅速消耗缓存,直接写入log文件)
}

func (S *SysLogHandle) Write(b []byte) (n int, err error) {
	return S.WriteString(*(*string)(unsafe.Pointer(&b)))
}

func (S *SysLogHandle) WriteString(msg string) (n int, err error) {
	buf := GetStrBuf()
	buf.WriteString(S.priorityPreStr)
	buf.WriteString(msg)
	if !strings.HasSuffix(msg, "\n") {
		buf.WriteByte('\n')
	}

	//if !strings.HasSuffix(msg, "\n") {
	//	msg = msg + "\n"
	//}
	//message := S.priorityPreStr+msg
	if !S.buff.Put(buf) {
		return -1, errors.New("syslog writer buf is full")
	}
	// 如果缓存的日志文件已经较多,则不等待检查,直接写入文件
	if len(S.largeBuff) == 0 && S.buff.Size() > S.batchSize {
		S.largeBuff <- true
	}

	return len(msg), nil
}

func (S *SysLogHandle) Close() error {
	S.deamon.Store(0)
	// 清空数据,不处理错误
	_ = S.Sync()
	S.waitGroup.Wait() //等待所有发送结束
	S.buff.Close()
	S.connPool.Close()
	return nil
}
func (S *SysLogHandle) Sync() error {
	for S.buff.Size() > 0 {
		S.clearLargeBuff()
		S.writeData()
	}
	return nil
}

func (S *SysLogHandle) clearLargeBuff() {
	for {
		select {
		case <-S.largeBuff:
		default:
			return
		}
	}
}
func (S *SysLogHandle) scanBuffer() {
	defer utils.CatchPanic()
	// 关闭
	//defer func() {
	//	_ = S.Close()
	//}()
	timeout := time.Millisecond * S.timeout
	scanBufferTimer := time.NewTimer(timeout)

	for S.deamon.Load() > 0 {
		select {
		// 每五分钟检查一次缓存,如果不为空,则写入文件中
		case <-scanBufferTimer.C:
			S.writeData()
			// 超时的时候检查文件，超时意味着目前日志量不多
			// 在程序空闲的时候再去做这些事情
			S.scanFile()
		case <-S.largeBuff:
			// 循环写文件
			// 持续到buff内容小于最大限制
			// 同时清除largeBuff
			_ = S.Sync()
		}
		scanBufferTimer.Stop()
		scanBufferTimer.Reset(timeout)
	}
}

func (S *SysLogHandle) writeData() {

	bs := S.buff.Size()
	if bs > 0 {
		S.waitGroup.Add(1)
		//缓存信息超过1w,判断为高并发状态,此时需要检查目录下文件是否大于1000,且只取缓存中1w条日志
		if bs > S.batchSize {
			bs = S.batchSize
		}

		// 建立缓冲区,接收wh.buff
		buff := _pool.Get()
		for i := 0; i < bs; i++ {
			content, ok := S.buff.Get()
			if ok {
				buf := content.(*strings.Builder)
				buff.WriteString(buf.String())
				PutStrBuf(buf)
				//buff.WriteString(content.(string))
			} else {
				break
			}
		}

		S.emit(buff.Bytes())
		_pool.Put(buff)
	}
}

//emit调用情况如下
//1.init方法调用scanBuffer()方法时会进入一个死循环，里面一直扫描buffer,如果扫描到buffer里有数据的时候就会调用
//2.调用CLOSE方法关闭的时候会查看buffer里有没有数据，有的话会调用
//3.scanBuffer()方法中当空闲时候去调用scanFile方法扫描文件，当文件里有东西时调用
func (S *SysLogHandle) emit(b []byte) {
	defer utils.CatchPanic()
	defer S.waitGroup.Add(-1)
	select {
	case S.limit <- 1:
		defer func() {
			<-S.limit
		}()
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

	case <-time.After(time.Millisecond * 10):
		_, _ = utils.ErrorOutput("flow control write file")
		S.writeFile(b)
	}
	return
}

func (S *SysLogHandle) init() error {
	S.largeBuff = make(chan bool, 20)
	filePath, ok := os.LookupEnv("SYSLOG_BUFFER")
	if !ok {
		filePath = "/data/syslog_buffer"
	}

	S.filePath = strings.TrimSuffix(filePath, "/")
	batchSize, ok := os.LookupEnv("BATCH_SIZE")
	if !ok {
		batchSize = "1000"
	}
	S.batchSize, _ = strconv.Atoi(batchSize)

	Linger, ok := os.LookupEnv("Linger")
	if !ok {
		Linger = "3"
	}
	S.linger, _ = strconv.ParseInt(Linger, 10, 64)

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
	err := os.MkdirAll(S.filePath, os.ModePerm)
	if err != nil {
		return err
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
	go S.scanBuffer()
	return nil
}

func (S *SysLogHandle) SetDialTimeoutFn(fn dialFunc) {
	S.dialTimeoutFn = fn
}

func NewSyslogHandle(raddr string, priority Priority, opts ...OptionFunc) (*SysLogHandle, error) {
	w := &SysLogHandle{
		priority:       LOG_LOCAL0 + priority,
		priorityPreStr: fmt.Sprintf("<%d>", LOG_LOCAL0+priority),
		raddr:          raddr,
		buff:           NewQueue(100000, time.Millisecond*10),
		deamon:         utils.NewInt32(1),
		limit:          make(chan int, 30),
		dialTimeoutFn:  net.DialTimeout,
	}
	for _, opt := range opts {
		opt(w)
	}
	err := w.init()
	return w, err
}

//写入文件的几种情况
//1.没有可用连接时
//2.使用链接发送日志到远端失败时
//3.使用链接发送日志到远端超时时
func (S *SysLogHandle) writeFile(data []byte) {
	//f := getRandomString(32)
	fileName := fmt.Sprintf("%s/%d", S.filePath, time.Now().UnixNano())

	err := ioutil.WriteFile(fileName, data, os.ModePerm)
	if err != nil {
		_, _ = utils.ErrorOutput(err.Error())
	}
	return
}

//在空闲时候会扫描文件
func (S *SysLogHandle) scanFile() {
	filePath := S.filePath
	files, err := ioutil.ReadDir(filePath)
	if err != nil {
		_, _ = utils.ErrorOutput(err.Error())
		return
	}
	if len(files) == 0 {
		return
	}
	n := rand.Intn(len(files))
	name := files[n].Name()
	fileName := filePath + "/" + name
	content, err := ioutil.ReadFile(fileName)
	if err != nil {
		_, _ = utils.ErrorOutput(err.Error())
	}
	if len(content) > 0 {
		S.waitGroup.Add(1)
		go S.emit(content)
	}
	os.Remove(fileName)
}

//func getRandomString(length int) string {
//	str := []byte("0123456789abcdefghijklmnopqrstuvwxyzABCDEFGHIJKLMNOPQRSTUVWXYZ")
//	result := make([]byte, 32)
//	r := rand.New(rand.NewSource(time.Now().UnixNano()))
//	for i := 0; i < length; i++ {
//		result = append(result, str[r.Intn(len(str))])
//	}
//	return string(result)
//}
