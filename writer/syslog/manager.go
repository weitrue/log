package syslog

import (
	"errors"
	"fmt"
	"go.uber.org/atomic"
	"io"
	"sync"
)

var uswManager = map[string]*UniqueSyslogWriter{}
var uswManagerLock = &sync.Mutex{}

type SyslogHandleWriter interface {
	io.WriteCloser
	Sync() error
	SetDialTimeoutFn(dialFunc)
}

type UniqueSyslogWriter struct {
	SyslogHandleWriter
	id string
	// 引用计数
	count *atomic.Int32
}

func (usw *UniqueSyslogWriter) Close() (err error) {
	r := usw.count.Dec()
	// 引用删除，调用 客户端 的 close 方法
	if r <= 0 {
		uswManagerLock.Lock()
		delete(uswManager, usw.id)
		uswManagerLock.Unlock()

		return usw.SyslogHandleWriter.Close()
	}
	return
}

func (usw *UniqueSyslogWriter) Reference() {
	usw.count.Inc()
}
func (usw *UniqueSyslogWriter) DeReference() {
	usw.count.Dec()
}
func (usw *UniqueSyslogWriter) Referenced() int32 {
	return usw.count.Load()
}
func (usw *UniqueSyslogWriter) Sync() error {

	return usw.SyslogHandleWriter.Sync()
}

// ClearSyslogWriter 清理不用的 syslog
func ClearSyslogWriter() {
	uswManagerLock.Lock()
	defer uswManagerLock.Unlock()
	for _, w := range uswManager {
		r := w.count.Dec()
		// 引用删除，调用 客户端 的 close 方法
		if r <= 0 {
			_ = w.SyslogHandleWriter.Close()
			delete(uswManager, w.id)
		}
	}
}

var syslogLevM = map[string]Priority{
	"DEBUG":    LOG_DEBUG,
	"INFO":     LOG_INFO,
	"ERROR":    LOG_ERR,
	"WARNING":  LOG_WARNING,
	"CRITICAL": LOG_CRIT,
	"FIXED":    LOG_ALERT,
}

func DialByLevel(ver int, network, raddr string, level string, opts ...OptionFunc) (*UniqueSyslogWriter, error) {
	uswManagerLock.Lock()
	defer uswManagerLock.Unlock()

	id := fmt.Sprintf("v%d-%s", ver, raddr)
	w, ok := uswManager[id]
	// client 已经存在
	if ok {
		w.Reference()
		return w, nil
	}

	l, ok := syslogLevM[level]
	if !ok {
		l = LOG_INFO
	}

	sysLog, err := Dial(ver, network, raddr, l, opts...)
	if err != nil {
		if sysLog != nil {
			_ = sysLog.Close()
		}
		return nil, err
	}

	w = &UniqueSyslogWriter{sysLog, id, atomic.NewInt32(1)}
	uswManager[id] = w
	return w, nil
}

func Dial(ver int, network, raddr string, priority Priority, opts ...OptionFunc) (SyslogHandleWriter, error) {
	if priority < LOG_EMERG || priority > LOG_DEBUG {
		return nil, errors.New("log/syslog: invalid priority")
	}

	if network != "tcp" {
		return nil, errors.New("syslog only support tcp")
	}

	var w SyslogHandleWriter
	var err error
	switch ver {
	case 1:
		w, err = NewSyslogHandle(raddr, priority, opts...)
	case 2:
		w, err = NewSyslogHandleV2(raddr, priority, opts...)
	}

	return w, err
}

// NewTcpSyslog 创建 syslog writer
func NewTcpSyslog(raddr string, opts ...OptionFunc) (*UniqueSyslogWriter, error) {
	return DialByLevel(1, "tcp", raddr, "INFO", opts...)
}

func NewTcpSyslog2(raddr string, opts ...OptionFunc) (*UniqueSyslogWriter, error) {
	return DialByLevel(2, "tcp", raddr, "INFO", opts...)
}
