package flumefilewriter

import (
	"time"

	"go.uber.org/atomic"
)

type writeHandle struct {
	Location *time.Location
	// 需要使用的缓存
	buff     chan []byte // 缓存日志信息
	sliceDir chan string // 缓存分区目录   --看需求中分片目录会增加,需要动态刷新
	// dirManager atomic.Value
	dirRRIndex atomic.Int64
	largeBuff  chan bool // 标记当前有大量缓存(高并发状态,使其会迅速消耗缓存,直接写入log文件)
	done       chan bool // 关闭各个协程的信号

	// 文件写入路径
	rootPath     string // 指包含分片目录的根目录
	tempFilePath string // 临时的分片目录,如果所有分片目录大于maxFileCount,log文件直接写到该目录下

	// 生成文件名规则
	tableName    string       // 服务名
	sendingMode  SendMode     // 日记发送方式：multiplexing or replicating(发送到es以及hdfs)
	selectorType SelectorType // 0,1,2 0:es集群,1:hdfs一级分区,2:hdfs二级分区
	isFile       bool         // flume channel 使用的通道类型，分为memory（可丢），file（不可丢），默认（file）
	isJson       bool         // 半结构化数据（json），结构化数据（txt，分隔符），默认（json数据）

	// 输出文件规则
	writeFileTime     time.Duration // 设置写入日志文件的间隔时间    -- 默认5分钟  间隔时间检查一次缓存,将日志写入文件
	flashSliceDirTime time.Duration // 设置刷新分片目录间隔时间      -- 默认8分钟  并发量高的情况下,设置此值越低write效率越高
	moveTempFileTime  time.Duration // 设置写入日志文件的间隔时间    -- 默认8分钟  间隔时间移动TempFile,需要设置MoveTempFile()
	maxFileCount      int           // 一个分区下最大文件数量
	maxLogCount       int           // 一个log文件下最大日志条数

	sliceDirCount  atomic.Int64 // 分片目录数量
	isMoveTempFile bool         // 是否监控并移动临时文件
	// 关闭标志
	isClose bool
}

type DialOption interface {
	apply(info *writeHandle)
}

type optionFunc func(*writeHandle)

func (f optionFunc) apply(log *writeHandle) {
	f(log)
}

// Location 设置 Location
func Location(location *time.Location) DialOption {
	return optionFunc(func(wh *writeHandle) {
		wh.Location = location
	})
}

// WriteFileTime 设置写入日志文件的间隔时间    -- 默认5分钟
func WriteFileTime(t time.Duration) DialOption {
	return setWriteFileTime{t: t}
}

type setWriteFileTime struct {
	t time.Duration
}

func (s setWriteFileTime) apply(info *writeHandle) {
	info.writeFileTime = s.t
}

// MoveTempFileTime 设置监控并移动临时日志文件的间隔时间    -- 默认8分钟
func MoveTempFileTime(t time.Duration) DialOption {
	return setMoveTempFileTime{t: t}
}

type setMoveTempFileTime struct {
	t time.Duration
}

func (s setMoveTempFileTime) apply(info *writeHandle) {
	info.moveTempFileTime = s.t
}

// FlashSliceDirTime 设置刷新分片目录间隔时间    -- 应 略小于 写入file时间*分片目录数量
func FlashSliceDirTime(t time.Duration) DialOption {
	return setflashSliceDirTime{t: t}
}

type setflashSliceDirTime struct {
	t time.Duration
}

func (s setflashSliceDirTime) apply(info *writeHandle) {
	info.flashSliceDirTime = s.t
}

// MaxFileCount 分片目录下最大文件数量      -- 默认1000
func MaxFileCount(maxCount int) DialOption {
	return setmaxFileCount{fileCount: maxCount}
}

type setmaxFileCount struct {
	fileCount int
}

func (s setmaxFileCount) apply(info *writeHandle) {
	info.maxFileCount = s.fileCount
}

// MaxLogCount 单个文件,最大日志数量     -- 默认10000
func MaxLogCount(maxCount int) DialOption {
	return setmaxLogCount{logCount: maxCount}
}

type setmaxLogCount struct {
	logCount int
}

func (s setmaxLogCount) apply(info *writeHandle) {
	info.maxLogCount = s.logCount
}

// MoveTempFile 可移动TempFile
func MoveTempFile() DialOption {
	return moveTempFile{}
}

type moveTempFile struct{}

func (s moveTempFile) apply(info *writeHandle) {
	info.isMoveTempFile = true
}

func defaultInfo() *writeHandle {
	return &writeHandle{
		writeFileTime:     5 * 60 * time.Second,
		flashSliceDirTime: 8 * 60 * time.Second,
		moveTempFileTime:  8 * 60 * time.Second,
		maxFileCount:      1000,
		maxLogCount:       10000,
		done:              make(chan bool, 5), // 关闭协程的信号
		// sliceDir:          make(chan string, 1000),   // 分片目录缓存
		largeBuff: make(chan bool, 20),       // 缓存过大的信号
		buff:      make(chan []byte, 100000), // 日志缓存
		// 默认本地时间时区
		Location: time.Local,
	}
}
