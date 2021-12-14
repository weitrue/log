// +build linux

package flumefilewriter

import (
	"golang.org/x/sys/unix"
	"os"
)

// FadvisePOSIXFADVDONTNEED 系统文件读取清理缓存操作
func FadvisePOSIXFADVDONTNEED(filename string) error {
	if useFadvise {
		f, err := os.OpenFile(filename, os.O_RDONLY, os.ModePerm)
		if err != nil {
			return err
		}
		// 4 == POSIX_FADV_DONTNEED
		err = unix.Fadvise(int(f.Fd()), 0, 0, POSIX_FADV_DONTNEED)
		if f != nil {
			f.Close()
		}
		return err
	}
	return nil
}
