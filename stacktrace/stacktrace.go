package stacktrace

import (
	"runtime"
	"strings"
	"sync"

	"github.com/weitrue/log/bufferpool"
)

const _logPackage = "github.com/weitrue/log"

var (
	// SKIP_LOG_FRAME_FLAG 是否 忽略 本模块的相关 栈信息，可以在测试时设置为 false，输出 本模块相关栈信息。
	SKIP_LOG_FRAME_FLAG = true
	_stacktracePool = sync.Pool{
		New: func() interface{} {
			return newProgramCounters(64)
		},
	}

	// 下述两个检查主要是过滤栈信息中，本模块的相关代码
	// 检查 栈信息中 函数名的前缀
	_logStacktracePrefixes = []string{
		_logPackage,
		// "runtime.call",
		// "runtime.gopanic",
	}
	// 检查 栈信息中 函数名的前缀是否在 vendor 目录下
	// _LogStacktraceVendorContains = addPrefix("/vendor/", _logStacktracePrefixes...)
	_LogStacktraceVendorContains = []string{
		_logPackage+"/vendor/",
	}
)

// TakeStacktraceSkip 获取堆栈信息
// skip 跳过多少个栈尾 数据，一般根据需要，设置 3 左右
// duplicateFrameNum 当使用递归或这链式调用等场景时，设置这个参数来过滤重复栈，
// 比如为1时，表示如果当前栈与上一个栈一致，则当前栈不输出；为2 时，表示当前栈与上上个栈一致时，当前栈不输出
func TakeStacktraceSkip(skipCaller,duplicateFrameSkip int) string {
	buffer := bufferpool.Get()
	defer buffer.Free()
	programCounters := _stacktracePool.Get().(*programCounters)
	defer _stacktracePool.Put(programCounters)

	// skipLogFrame 跳过 log 本地函数栈，默认为 true；false 主要是用于 log 库测试用例。
	skipLogFrame := SKIP_LOG_FRAME_FLAG
	var numFrames int
	for {
		// Skip the call to runtime.Counters and takeStacktrace so that the
		// program counters start at the caller of takeStacktrace.
		numFrames = runtime.Callers(2+skipCaller, programCounters.pcs)
		if numFrames < len(programCounters.pcs) {
			break
		}
		// Don't put the too-short counter slice back into the pool; this lets
		// the pool adjust if we consistently take deep stacktraces.
		programCounters = newProgramCounters(len(programCounters.pcs) * 2)
	}

	i := 0
	frames := runtime.CallersFrames(programCounters.pcs[:numFrames])

	var duplicateFrames  = make([]runtime.Frame,duplicateFrameSkip)
	// Note: On the last iteration, frames.Next() returns false, with a valid
	// frame, but we ignore this frame. The last frame is a a runtime frame which
	// adds noise, since it's only either runtime.main or runtime.goexit.
	for frame, more := frames.Next(); more; frame, more = frames.Next() {
		if skipLogFrame && isLogFrame(frame.Function) {
			// skipLogFrame 栈尾检查是否是 log 库相关操作，如果是则忽略。
			// 过了栈尾之后，就不再处理了
			continue
		} else {
			skipLogFrame = false
		}
		// 在记录了 大于等于 duplicateFrameSkip 数量的栈信息后才开始检查重复栈
		if duplicateFrameSkip>0{
			// 用于比较的前一个栈的位置
			// 当前栈 位置 i = 2
			// 前duplicateFrameSkip个栈 位置 i-duplicateFrameSkip 0
			// %duplicateFrameSkip 用于计算循环队列的 真实 位置
			if i >= duplicateFrameSkip {
				preFrameIndex := (i-duplicateFrameSkip)%duplicateFrameSkip
				if duplicateFrames[preFrameIndex].File == frame.File && duplicateFrames[preFrameIndex].Line == frame.Line{
					continue
				}
			}

			duplicateFrames[i%duplicateFrameSkip] = frame
		}

		if i != 0 {
			buffer.AppendByte('\n')
		}
		i++

		fnName := StripPackage(frame.Function)
		buffer.AppendString(fnName)
		buffer.AppendByte('\n')
		buffer.AppendByte('\t')
		buffer.AppendString(frame.File)
		buffer.AppendByte(':')
		buffer.AppendInt(int64(frame.Line))
	}

	return buffer.String()
}

func TakeStacktrace() string {
	return TakeStacktraceSkip(0,0)
}

func isLogFrame(function string) bool {
	for _, prefix := range _logStacktracePrefixes {

		if strings.HasPrefix(function, prefix) {
			return true
		}
	}

	// We can't use a prefix match here since the location of the vendor
	// directory affects the prefix. Instead we do a contains match.
	for _, contains := range _LogStacktraceVendorContains {
		if strings.Contains(function, contains) {
			return true
		}
	}

	return false
}

type programCounters struct {
	pcs []uintptr
}

func newProgramCounters(size int) *programCounters {
	return &programCounters{make([]uintptr, size)}
}

func addPrefix(prefix string, ss ...string) []string {
	withPrefix := make([]string, len(ss))
	for i, s := range ss {
		withPrefix[i] = prefix + s
	}
	return withPrefix
}
// StripPackage strips the package name from the given Func.Name.
func StripPackage(n string) string {

	slashI := strings.LastIndex(n, "/")
	if slashI == -1 {
		slashI = 0 // for built-in packages
	}
	dotI := strings.Index(n[slashI:], ".")
	if dotI == -1 {
		return n
	}
	return n[slashI+dotI+1:]
}

