package flumefilewriter

import (
	"errors"
	"fmt"
	"io"
	"io/ioutil"
	"math/rand"
	"os"
	"path/filepath"
	"runtime"
	"strconv"
	"strings"
	"time"

	"github.com/weitrue/go.uuid"
	"github.com/weitrue/log/utils"
)

// NewWriteHandle creat flume writer handle
func NewWriteHandle(RootPath string, TempFilePath string, tableName string, sendingMode SendMode, selectorType SelectorType, isFile bool, isJson bool, opts ...DialOption) (*writeHandle, error) {
	wh := defaultInfo()
	for _, opt := range opts {
		opt.apply(wh)
	}

	wh.rootPath = RootPath
	wh.tempFilePath = TempFilePath
	wh.tableName = tableName //表名
	wh.sendingMode = sendingMode
	wh.selectorType = selectorType
	wh.isFile = isFile
	wh.isJson = isJson

	if err := wh.checkInitDir(); err != nil {
		return nil, err
	}
	if wh.tempFilePath == "" {
		// 使用系统临时文件
		wh.tempFilePath = os.TempDir()
	}
	rand.Seed(time.Now().Unix())
	if wh.sliceDirCount.Load() > 0 {
		wh.dirRRIndex.Store(int64(rand.Uint32()) % wh.sliceDirCount.Load())
	}

	// 每五分钟检查一次日志缓存,如果不为空,则写入文件中
	go wh.monitorBuffer()
	// 刷新缓存分区目录
	go wh.flashSliceDir()
	// 检查临时目录下文件数
	if wh.isMoveTempFile {
		go wh.monitorTemp()
	}
	// 初始化时 异步 Rename 避免阻塞
	go RenameTempSuffixFile(*wh)
	return wh, nil
}

func (wh *writeHandle) formatLogFileName() (fileName string) {
	// 服务名
	fileName = wh.tableName + "."
	// 0,1,2 0:es集群,1:hdfs一级分区,2:hdfs二级分区
	fileName += strconv.Itoa(int(wh.selectorType)) + "."
	// 时间
	// 使用UTC
	// fileName += time.Unix(time.Now().Unix(), 0).Format("2006-01-02") + "."
	fileName += time.Now().In(wh.Location).Format("2006-01-02") + "."
	// 使用的通道类型，分为memory（可丢），file（不可丢），默认（file）
	if wh.isFile {
		fileName += "1."
	} else {
		fileName += "0."
	}
	// 半结构化数据（json），结构化数据（txt，分隔符），默认（json数据）
	if wh.isJson {
		fileName += "1."
	} else {
		fileName += "0."
	}
	fileName += uuid.Must(uuid.NewV4()).String()
	return
}

func (wh *writeHandle) formatFileNameWithDir(dir string) (pwd string) {
	// 服务名
	fileName := wh.formatLogFileName()
	pwd = filepath.Join(dir, fileName)
	return

}
func (wh *writeHandle) formatBaseDirPath(root string, shard string, mode SendMode) (pwd string) {
	return filepath.Join(root, shard, mode.toString())
}

// 通过配置的命名规则,生成文件名
func (wh *writeHandle) formatFileName() (pwd string) {
	var (
		preDir     string
		isMuchFile bool
	)
	// 获取一个合适的分片路径,如果没有则返回文件数过多的bool值
	preDir, isMuchFile = wh.getFilePath()

	// 如果当前分片目录下文件数量较多
	if isMuchFile {
		preDir = wh.formatBaseDirPath(wh.tempFilePath, "", wh.sendingMode)
	}
	pwd = wh.formatFileNameWithDir(preDir)
	return pwd
}

// Sync 同步数据，写入，并作一些特殊处理
// 这边尝试进行移动临时数据
func (wh *writeHandle) Sync() error {
	err := wh.EndToFlush()
	if err != nil {
		return err
	}
	wh.moveTempFile()

	return nil
}

// LargeToFlush 高并发下Flush
func (wh *writeHandle) LargeToFlush() error {
	for len(wh.buff) > wh.maxLogCount/2 {
		wh.clearLargeBuff()
		wh.writeFile()
	}
	return nil
}

// EndToFlush 结束状态Flush 一直刷新,直到wh.buff无数据,用于log优雅关闭
func (wh *writeHandle) EndToFlush() error {
	// 有数据就刷新
	for len(wh.buff) > 0 {
		wh.clearLargeBuff()
		wh.writeFile()
	}
	return nil
}


// 写入 log 到缓存中
func (wh *writeHandle) Write(data []byte) (n int, err error) {
	if wh.isClose {
		return -1, errors.New("writer closed")
	}
	msg := string(data)
	if !strings.HasSuffix(msg, "\n") {
		msg = msg + "\n"
	}
	// select {
	// case wh.buff <- []byte(msg):
	// default:
	// 	 return -1, errors.New("writer buffer is full")
	// }

	wh.buff <- []byte(msg)

	// 如果缓存的日志文件已经较多,则不等待检查,直接写入文件
	if len(wh.largeBuff) == 0 && len(wh.buff) > wh.maxLogCount {
		wh.largeBuff <- true
	}
	return len(msg), nil
}

func (wh *writeHandle) clearLargeBuff() {
	for {
		select {
		case <-wh.largeBuff:
		default:
			return
		}
	}
}

// 监控缓存,当缓存大于1w或间隔,输出日志文件
func (wh *writeHandle) monitorBuffer() {
	defer utils.CatchPanic()

	writeFileTime := time.NewTimer(wh.writeFileTime)
	for {

		select {
		// 每五分钟检查一次缓存,如果不为空,则写入文件中
		case <-writeFileTime.C:
			wh.writeFile()
		case <-wh.largeBuff:
			// 循环写文件
			// 持续到buff内容小于最大限制
			// 同时清除largeBuff

			if len(wh.buff) >= wh.maxLogCount {
				_ = wh.LargeToFlush()
			}

		case <-wh.done:
			writeFileTime.Stop()

			return
		}
		writeFileTime.Reset(wh.writeFileTime)
	}
}

// 如果缓存信息不为空,则将其写入到文件中
func (wh *writeHandle) writeFile() {
	if len(wh.buff) > 0 {
		var (
			fileName string // 文件名
			writeErr error
			// writeN    int
			logLength   int // 日志数量
			logMsgCount int
		)
		// 缓存信息超过1w,判断为高并发状态,此时需要检查目录下文件是否大于1000,且只取缓存中1w条日志
		if logLength = len(wh.buff); logLength > wh.maxLogCount {
			logLength = wh.maxLogCount
		}

		// 建立缓冲区,接收wh.buff
		buff := _pool.Get()
		for i := 0; i < logLength; i++ {
			select {
			case msg := <-wh.buff:
				buff.Write(msg)
				logMsgCount++
			default:
				// 跳过，有多少写多少
				i = logLength
			}
		}
		d := buff.Bytes()
		countSuffix := strconv.Itoa(logMsgCount)

		if len(d) > 0 {
			// 重试所有目录
			retryCount := int(wh.sliceDirCount.Load())
			for i := 0; i < retryCount; i++ {
				// 获取文件路径和文件名
				fileName = wh.formatFileName() + "." + countSuffix
				_, err := WriteFile(fileName, d, os.ModePerm)
				if err != nil {
					if _, ok := err.(*os.PathError); ok {
						freshDirErr := wh.freshDir()
						if freshDirErr != nil {
							writeErr = MergeError(writeErr, fmt.Errorf("freshDir:%v:%v", freshDirErr, err))
							break
						}
					}
				} else {
					// 重试写入 成功
					writeErr = nil
					break
				}
				writeErr = MergeError(writeErr, err)
			}
			// 最终还是报错
			if writeErr != nil {
				// 不处理 fmt 错误
				_, _ = utils.ErrorOutput(NewError("WriteFile", writeErr).Error())
				// 写临时目录了
				// 不处理 fmt 错误
				_, _ = wh.write2SysTemp(d, countSuffix)
			}
		}
		_pool.Put(buff)
	}
}

func (wh *writeHandle) write2SysTemp(d []byte, logMsgCount string) (int, error) {
	preDir := filepath.Join(os.TempDir(), "taotie.log")
	err := os.Mkdir(preDir, os.ModePerm)
	// 这都不能写
	if err != nil {
		return utils.ErrorOutput(string(d))
	}
	pwd := wh.formatFileNameWithDir(preDir) + "." + logMsgCount
	n, err := WriteFile(pwd, d, os.ModePerm)
	// 这都不能写
	if err != nil {
		return utils.ErrorOutput(string(d))
	}
	return n, err
}

// 刷新SliceDir的缓存
func (wh *writeHandle) flashSliceDir() {
	defer utils.CatchPanic()

	// err := wh.putSliceDir()
	// if err != nil {
	//	 utils.ErrorOutput( err)
	// }
	// 每十分钟检查一次,如果缓存的分区目录数小于200,则将ROOTPath下的分区放到SliceDir中
	flashSliceDirTime := time.NewTimer(wh.flashSliceDirTime)
	for {

		select {
		case <-flashSliceDirTime.C:
			err := wh.freshDir()
			if err != nil {
				_, _ = utils.ErrorOutput(err.Error())
			}
		case <-wh.done:
			flashSliceDirTime.Stop()
			return
		}
		flashSliceDirTime.Reset(wh.flashSliceDirTime)
	}
}

func (wh *writeHandle) IsMyFile(fileName string) bool {
	return strings.HasPrefix(fileName, wh.tableName)
}

// func (wh *writeHandle) CountRootPathDir(fileInfos []os.FileInfo) []DirItem {
// 	dirs := make([]DirItem, 0, len(fileInfos))
// 	for _, v := range fileInfos {
// 		dirName := v.Name()
// 		if strings.HasSuffix(dirName, "temp") {
// 			continue
// 		}
// 		if v.IsDir() {
// 			item := DirItem{
// 				Name:      dirName,
// 				FileCount: utils.NewInt32(0),
// 			}
// 			dirs = append(dirs, item)
// 		}
//
// 	}
// 	return dirs
// }

// 获取 rootPath 下的分片目录 只需要获取其大小即可
func (wh *writeHandle) freshDir(fileInfos ...os.FileInfo) (err error) {
	if len(fileInfos) <= 0 {
		// TODO： 如果目录被删除了，那么需要做处理
		fileInfos, err = readDir(wh.rootPath)
		if err != nil {
			return err
		}
	}

	// dLen := len(fileInfos)
	if len(fileInfos) == 0 {
		return errors.New("RootPath is an empty directory")
	}
	// dirs := wh.CountRootPathDir(fileInfos)

	// 对比旧数据，如果旧数据比较大，则替换,去掉 temp
	// wh.dirManager.Store(dirs)
	// 只需要知道文件夹数量即可,名称一定为整数
	wh.sliceDirCount.Store(int64(len(fileInfos)) - 1)

	return nil
}

// func (wh *writeHandle) getDirCache() []DirItem {
// 	return wh.dirManager.Load().([]DirItem)
// }
//
// // 获取一个分片目录
// func (wh *writeHandle) getShardDir() DirItem {
// 	index := wh.dirRRIndex.Add(1)
// 	m := wh.getDirCache()
// 	dirName := m[index%uint32(len(m))]
// 	return dirName
// }

// 获取一个分片目录,确认了 分片目录名必为 数字 (1..2..3..这种)所以直接根据sliceDirCount取余
// 如 root 目录下有十个文件 1-9为分片目录及一个temp 文件 (已确定)
// sliceDirCount大小为 9 ,则取余会得到 0-8 ,需对结果+1,再转为string,则为分片目录
func (wh *writeHandle) getShardDir() string {
	return strconv.FormatInt((wh.dirRRIndex.Inc()%wh.sliceDirCount.Load())+1, 10)
}

// 高并发状态下,检查分片目录下的文件数是否大于1000(配置)
func (wh *writeHandle) checkSliceDir(sliceDir string) (int, bool) {

	fileList, err := readDir(sliceDir)
	l := len(fileList)
	if err != nil {
		_, _ = utils.ErrorOutput(err.Error())
		return l, true
	}
	// 如果当前目录下,文件数大于1000,则返回true
	if l >= wh.maxFileCount {
		return l, true
	}
	return l, false
}

// 获取一个分片目录下文件数小于MaxFileCount的 路径
func (wh *writeHandle) getFilePath() (tempFilePath string, isMuchFile bool) {
	var shardDir string
	// c        int

	retryCount := int(wh.sliceDirCount.Load())
	for i := 0; i < retryCount; i++ {
		shardDir = wh.getShardDir()
		// 拼接FilePath和Name
		// 分片目录
		// 发送模式
		tempFilePath = wh.formatBaseDirPath(wh.rootPath, shardDir, wh.sendingMode)

		// 判断该路径下文件数是否大于1000
		_, isMuchFile = wh.checkSliceDir(tempFilePath)
		// 以前存储了目录下文件数,但是未使用,简化 去掉
		// shardDir.FileCount.Store(int32(c))
		if isMuchFile {
			continue
		}
		return tempFilePath, isMuchFile
	}
	isMuchFile = true
	return tempFilePath, isMuchFile
}

// 检查临时文件下的文件数
func (wh *writeHandle) monitorTemp() {
	defer utils.CatchPanic()

	moveTempFileTime := time.NewTimer(wh.moveTempFileTime)
	for {

		select {
		case <-moveTempFileTime.C:
			wh.moveTempFile()
		case <-wh.done:
			moveTempFileTime.Stop()
			return
		}

		moveTempFileTime.Reset(wh.moveTempFileTime)
	}
}

// 移动临时文件夹下的日志至flume分区
func (wh *writeHandle) moveTempFile() {
	if !wh.isMoveTempFile {
		return
	}

	// 读取临时目录下的发送方式
	tempFilePath := filepath.Join(wh.tempFilePath, wh.sendingMode.toString())

	// 读取replicating或multiplexing目录
	// 读取目录下的临时文件
	err := wh.RenameTempFile(tempFilePath)
	if err != nil {
		_, _ = utils.ErrorOutput(NewError("moveTempFile", err).Error())
	}
}

// Close 将剩余缓存写入文件,临时目录下的文件移动到分片目录
func (wh *writeHandle) Close() error {
	// TODO: 释放 chan 资源
	if !wh.isClose {
		wh.isClose = true
		wh.done <- true
		wh.done <- true
		wh.done <- true

	}
	wh.Sync()

	return nil
}

func (wh *writeHandle) GetAllLogFileDir() (l []string) {
	// 获取分片目录
	dirList, err := readDir(wh.rootPath)
	if err != nil {
		return nil
	}

	// 检查每个分片目录下是否含有 sendmod
	for _, shard := range dirList {
		// /data/test/1
		sendModList, _ := readDir(filepath.Join(wh.rootPath, shard.Name()))
		for _, sendModDir := range sendModList {
			// /data/test/1/multiplexing
			sendModeName := sendModDir.Name()
			if sendModeName == wh.sendingMode.toString() {
				l = append(l, filepath.Join(wh.rootPath, shard.Name(), sendModeName))
			}
		}
	}
	return
}

// 初始化时检查路径参数,每个分片目录和临时目录至少有一种 send mod
func (wh *writeHandle) checkInitDir() error {
	var isSliceDir bool
	// 获取分片目录
	dirList, err := readDir(wh.rootPath)
	if err != nil {
		return err
	}
	if len(dirList) == 0 {
		return errors.New("rootPath is an empty directory")
	}
	// 检查每个分片目录下是否含有 sendmod
	for _, shard := range dirList {
		if strings.HasSuffix(shard.Name(), "temp") {
			continue
		}
		// /data/test/1
		sendModList, _ := readDir(filepath.Join(wh.rootPath, shard.Name()))
		// isSliceDir = false
		for _, sendModDir := range sendModList {
			// /data/test/1/multiplexing
			sendModeName := sendModDir.Name()
			if sendModeName == wh.sendingMode.toString() {
				isSliceDir = true
				// 失败就失败，影响不大 不在初始化时Rename 避免阻塞
				// _ = wh.RenameTempFile(filepath.Join(wh.rootPath, shard.Name(), sendModeName))
				break
			}
		}
		// 分片目录遍历完.也没有 sendMod 目录,则返回err
		if !isSliceDir {
			return errors.New("not sendMod directory")
		}
	}
	// 刷新文件夹状态 只需要获取其下文件夹数量即可
	err = wh.freshDir(dirList...)
	if err != nil {
		return err
	}

	// 再检查临时目录
	tempDirList, err := readDir(wh.tempFilePath)
	if err != nil {
		return err
	}
	if len(tempDirList) == 0 {
		return errors.New("tempFilePath is an empty directory")
	}
	isSliceDir = false
	for _, sendMod := range tempDirList {
		sendModeName := sendMod.Name()
		if sendModeName == wh.sendingMode.toString() {
			isSliceDir = true
			break
		}
	}
	if !isSliceDir {
		return errors.New("TempPath not have slice directory")
	}
	return nil
}

func (wh *writeHandle) RenameTempFile(currentFilePath string) (err error) {
	var (
		fileInfos []os.FileInfo
		// move or rename
		moveFile bool
	)
	fileInfos, err = readDir(currentFilePath)
	if err != nil {
		return
	}

	if strings.HasPrefix(currentFilePath, wh.tempFilePath) {
		moveFile = true
	}
	now := time.Now()
	for _, fv := range fileInfos {
		fileName := fv.Name()

		// 如果是临时写入缓存文件，则忽略
		// 或者是其他log的文件，也忽略, tableName 判断
		if !wh.IsMyFile(fileName) {
			continue
		}

		if IsWriterTmpFile(fileName) {
			// 临时文件大于写文件时间2倍时，将其改名
			if now.Sub(fv.ModTime()) > wh.writeFileTime*2 {
				fileName = TempFile2LogFile(fileName)
			} else {
				continue
			}
		}

		if !moveFile && fileName == fv.Name() {
			// 名字相同，当前目录又不在临时目录，则是正常文件，不必处理
			continue
		}

		oldPath := filepath.Join(currentFilePath, fv.Name())
		newPath := filepath.Join(currentFilePath, fileName)
		if moveFile {
			// 获取合适的分片路径,没有则跳过这些文件的移动
			filePath, isMuch := wh.getFilePath()
			if isMuch {
				break
			}
			newPath = filepath.Join(filePath, fileName)
		}

		// 可能被其他程序处理，忽略错误
		_ = os.Rename(oldPath, newPath)

	}
	return
}

var chmodSupported = false

var logWriterTmpPrefix = "__*.tmp"
var logWriterTmpPrefixSmall = "__"
var logWriterTmpSuffix = ".tmp"

// ResetLogWriterTmpFormat 设置默认的临时文件格式*为随机数ID，避免文件名冲突
func ResetLogWriterTmpFormat(format string) {
	logWriterTmpPrefix = format
	index := strings.Index(logWriterTmpPrefix, "*")
	if index > 0 {
		logWriterTmpPrefixSmall = logWriterTmpPrefix[:index]
	}
}

// ResetLogWriterTmpSuffix 设置临时文件后缀，用于避免调整
func ResetLogWriterTmpSuffix(format string) {
	logWriterTmpSuffix = format
}

// ResetChmodSupported 设置是否开启日志文件权限设置，开启后所有日志文件设置 0777 权限
func ResetChmodSupported(v bool) {
	if v && runtime.GOOS != "windows" {
		chmodSupported = v
	} else {
		chmodSupported = false
	}
}

// WriteFile is a drop-in replacement for ioutil.WriteFile;
// but writeFile writes data to a temporary file first and
// only upon success renames that file to filename.
// TODO(gri) This can be removed if #17869 is accepted and
// implemented.
// using gofmt writeFile logic
func WriteFile(filename string, data []byte, perm os.FileMode) (int, error) {
	// open temp file
	// 记录文件原始名称
	//

	f, err := ioutil.TempFile(filepath.Dir(filename), filepath.Base(filename)+logWriterTmpPrefix)
	if err != nil {
		return 0, NewError("TempFile", err)
	}
	backendName := f.Name()
	if chmodSupported {
		err = f.Chmod(perm)
		if err != nil {
			// 错误不影响
			_ = f.Close()

			// TODO: 错误聚合处理
			// 删除文件，错误不影响
			_ = os.Remove(backendName)
			return 0, NewError("Chmod", err)
		}
	}

	// write data to temp file
	n, err1 := f.Write(data)
	if err1 == nil && n < len(data) {
		err1 = io.ErrShortWrite
	}

	err2 := f.Close()
	err = MergeError(err1, err2)
	if err != nil {
		// 删除文件，错误不影响
		_ = os.Remove(backendName)
		return n, NewError("Write Close", err)
	}
	// 不用处理,进行特殊的清理缓存操作
	_ = FadvisePOSIXFADVDONTNEED(backendName)

	// 将临时文件重命名为正式文件。
	// 如果重命名失败，则不进行处理
	err = os.Rename(backendName, filename)
	if err != nil {
		// 删除文件，错误不影响
		_ = os.Remove(backendName)
		return 0, NewError("Rename", err)
	}
	return n, nil
}
