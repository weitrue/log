// +build !linux

package flumefilewriter

// FadvisePOSIXFADVDONTNEED 系统文件读取清理缓存操作
func FadvisePOSIXFADVDONTNEED(_ string) error {
	return nil
}
