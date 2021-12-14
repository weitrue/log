package flumefilewriter

import (
	"fmt"
	"os"
	"strconv"
	"strings"
)

// POSIX_FADV_DONTNEED 用于 Fadvise 操作的第四个参数，表示指定的数据近期不会被访问，立即从page cache 中丢弃数据。
// posix_fadvise 是linux上对文件进行预取的系统调用，其中第四个参数int advice为预取的方式。
const POSIX_FADV_DONTNEED = 4

func NewError(prefix string, err error) error {
	return fmt.Errorf("%s: %v", prefix, err)
}

func MergeError(errs ...error) (err error) {

	for i, e := range errs {
		if e != nil {
			err = NewError(strconv.Itoa(i), err)
		}
	}
	return

}

// 快速版读取文件夹
func readDir(path string) (fileInfos []os.FileInfo, err error) {
	file, err := os.Open(path)
	if err != nil {
		return nil, err
	}
	fileInfos, err = file.Readdir(-1)
	file.Close()
	return
}

func IsWriterTmpFile(fileName string) bool {

	return strings.HasSuffix(fileName, logWriterTmpSuffix)
}

func TempFile2LogFile(fileName string) string {
	index := strings.Index(fileName, logWriterTmpPrefixSmall)
	if index > 0 {
		return fileName[:index]
	}
	return fileName

}
func IsSliceDir(dir string) bool {
	if dir != Multiplexing.toString() && dir != Replicating.toString() {
		return false
	}
	return true
}

// RenameTempSuffixFile 重命名临时文件，不忽略temp目录
func RenameTempSuffixFile(wh writeHandle) (renameCount int) {
	logFileDirs := wh.GetAllLogFileDir()

	for _, logFileDir := range logFileDirs {
		// 检查缓存临时文件，并重命名失败就失败，影响不大
		err := wh.RenameTempFile(logFileDir)
		if err == nil {
			renameCount++
		}
	}

	return
}

// 针对Fadvise进行的特殊设置
var useFadvise = true

// EnableFadvise 开启 Fadvise 处理
// 当前仅针对文件处理时，删除系统缓存。
func EnableFadvise(){
	useFadvise = true
}
// DisableFadvise 关闭 Fadvise 处理
func DisableFadvise(){
	useFadvise = false
}