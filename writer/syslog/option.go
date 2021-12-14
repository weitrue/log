package syslog

type OptionFunc func(handle SyslogHandleWriter)

func ResetDialTimeout(fn dialFunc) OptionFunc {
	return func(handle SyslogHandleWriter) {
		handle.SetDialTimeoutFn(fn)
	}
}
