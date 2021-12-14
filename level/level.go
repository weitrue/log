package level

import (
    "errors"
    "fmt"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
    "strings"
    "unsafe"
)

var errUnmarshalNilLevel = errors.New("can't unmarshal a nil *Level")

const (
    // DebugLevel 调试信息级别，一般用于开发环境，在生产环境中应该关闭。
    DebugLevel  Level = iota - 1
    // InfoLevel 默认信息级别
    InfoLevel
    // WarnLevel 警告信息级别，比 Info 级别更高，用于标注特殊，但又不需要立即处理的信息
    WarnLevel
    // ErrorLevel 错误信息级别。在程序出现异常时，应该使用这类等级进行记录。
    ErrorLevel
    // CriticalLevel 程序异常级别。一般用于程序异常时输出错误信息和堆栈信息。
    CriticalLevel
    // PanicLevel logs a message, then panics.
    // 占位，日志等级无需这么多等级。
    _PanicLevel
    // FatalLevel logs a message, then calls os.Exit(1).
    // 占位，日志等级无需这么多等级。
    _FatalLevel
    // FixedLevel 固定信息级别。主要特点是最高日志等级的正常日志。用于记录服务运行信息或其他加载信息等与普通信息区分开来的正常信息。
    FixedLevel

)
// Level 日志输出等级，级别越高越重要
type Level = zapcore.Level

/*

如果想要对当前 logger 对象调整 level 输出等级，需要使用 config.Level.SetLevel,该 Level 是AtomicLevel，支持原子操作，
并且相关 core 的 levelEnabler 都会调整。
如果你设置了多个 core ，并且 每个 Core 的 Level（或者 levelEnabler）不一定一样，那么你需要调用 log.WithOption ，通过 WrapCore 重新构建 core
或者通过 New 重新构建 logger

*/

// LevelEnablerFunc 用于方便的实现 LevelEnabler 接口的匿名函数
// 常用于需要分割不同日志等级数据到不同的存储上时，例如 info 数据输出到标准输出 stdout, error 以上数据输出到标准错误输出 stderr 上。
type LevelEnablerFunc = zap.LevelEnablerFunc

// AtomicLevel 是一个可动态更改日志记录级别的对象。 它允许你在运行时安全地更改 log 树（root logger和通过添加上下文创建的任何子项）的日志级别。
// AtomicLevel 本身是一个http.Handler，它为JSON端点提供服务以改变其级别。
// 必须使用 NewAtomicLevel 构造函数创建AtomicLevels以分配其内部原子指针。
type AtomicLevel struct {
    zap.AtomicLevel `yaml:",inline"`
}

func (lvl AtomicLevel) String() string {
    return Level2CapitalName(lvl.Level())
}
// UnmarshalText 反序列化 []byte 为 AtomicLevel
// 如： DEBUG，INFO，WARN，ERROR，CRITICAL，FIXED，也可以用小写
func (lvl *AtomicLevel) UnmarshalText(text []byte) error {
    var l Level
    sText := *(*string)(unsafe.Pointer(&text))
    l,ok := Name2Level(sText)
    if !ok{
        return errors.New("unmarshalText level:"+sText)
    }
    lvl.AtomicLevel = zap.NewAtomicLevelAt(l)
    lvl.SetLevel(l)
    return nil
}
func StringToSliceByte(s string) []byte {
    x := (*[2]uintptr)(unsafe.Pointer(&s))
    h := [3]uintptr{x[0], x[1], x[1]}
    return *(*[]byte)(unsafe.Pointer(&h))
}

// MarshalText 序列化 AtomicLevel 为 []byte 数据
// 如： DEBUG，INFO，WARN，ERROR，CRITICAL，FIXED
func (lvl AtomicLevel) MarshalText() (text []byte, err error) {
    return StringToSliceByte(Level2CapitalName(lvl.Level())),nil
}

type LevelEnabler = zapcore.LevelEnabler

// NewAtomicLevelAt 方便创建 AtomicLevel
func NewAtomicLevelAt(l Level) AtomicLevel {
    a := AtomicLevel{zap.NewAtomicLevel()}
    a.SetLevel(l)
    return a
}
// Name2Level 字符串转 level
func Name2Level(l string) (Level,bool) {
    l = strings.ToUpper(l)

    switch l {
    case "DEBUG":
        return DebugLevel,true
    case "INFO":
        return InfoLevel,true
    case "WARNING":
        return WarnLevel,true
    case "ERROR":
        return ErrorLevel,true
    case "CRITICAL":
        return CriticalLevel,true
    case "PANIC":
        return _PanicLevel,true
    case "FATAL":
        return _FatalLevel,true
    case "FIXED":
        return FixedLevel,true
    default:
        return InfoLevel,false
    }
}

// Level2Name level 转 字符串，可选参数为是否处理成大写字符串
func Level2Name(l Level,capital bool) string {
    //var capital bool
    //if len(capitals)>0{
    //    capital = capitals[0]
    //}
    //capital = true
    switch l {
    case DebugLevel:
        if capital{
            return "DEBUG"
        }
            return  "debug"
    case InfoLevel:
        if capital{
            return "INFO"
        }
            return  "info"
    case WarnLevel:
        if capital{
            return "WARN"
        }
        return  "warn"
    case ErrorLevel:
        if capital{
            return "ERROR"
        }
        return  "error"
    case CriticalLevel:
        if capital{
            return "CRITICAL"
        }
        return  "critical"
    case _PanicLevel:
        return "PANIC"
    case _FatalLevel:
        return "FATAL"
    case FixedLevel:
        if capital{
            return "FIXED"
        }
        return  "fixed"
    default:
        if capital{
            return fmt.Sprintf("LEVEL(%d)", l)
        }
        return  fmt.Sprintf("level(%d)", l)
    }
}

func Level2CapitalName(l Level) string {

    switch l {
    case DebugLevel:
            return "DEBUG"
    case InfoLevel:
            return "INFO"
    case WarnLevel:
            return "WARN"
    case ErrorLevel:
            return "ERROR"
    case CriticalLevel:
            return "CRITICAL"
    case _PanicLevel:
        return "PANIC"
    case _FatalLevel:
        return "FATAL"
    case FixedLevel:
            return "FIXED"
    default:
            return fmt.Sprintf("LEVEL(%d)", l)
    }
}


// 自定义 StackLevel，主要是自定义了FIXED 级别，但不希望捕获 stack
type StackLevel struct {
    Level
}

// Enabled 如果lv参数大于 当前日志等级，则返回 true，进行记录stack，否则返回false，不记录。
// 对于 FIXED 等级，不进行记录 stack。
func (l StackLevel) Enabled(lv Level) bool {
    if lv >= FixedLevel {
        return false
    }
    return lv >= l.Level
}