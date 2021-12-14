package utils

import (
	"fmt"
	"os"
	"runtime"
	"strings"
	"time"

	"go.uber.org/zap/zapcore"
)

// ErrorOutput 修改该函数可以切换错误输出方式
// 日志组件内部异常无法传递到外部，因此默认使用输出到标准错误中。
var ErrorOutput = func(msg string) (n int, err error) {
	caller:= zapcore.NewEntryCaller(runtime.Caller(1)).TrimmedPath()
	t := time.Now().Format("2006-01-02T15:04:05.000Z07:00")
	if strings.LastIndex(msg, "\n") > 0 {
		return fmt.Fprint(os.Stderr, t+" "+msg)
	}
	return fmt.Fprintln(os.Stderr, t+" ERROR "+caller+" "+msg)
}

func CatchPanic() {
	if xErr := recover(); xErr != nil {
		_, _ = ErrorOutput(fmt.Sprint(xErr))
	}
}
