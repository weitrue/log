package log

import (
    "fmt"
    "io/ioutil"
    "os"
    "runtime"
    "sort"
    "time"

    "github.com/weitrue/log/config"
    "github.com/weitrue/log/core"
    "github.com/weitrue/log/encoder"
    "github.com/weitrue/log/entry"
    "github.com/weitrue/log/field"
    "github.com/weitrue/log/level"
    "github.com/weitrue/log/writer"
)

var (
    DefaultLevel = "INFO"
)


// IBaseLogger 基础日志接口
type IBaseLogger interface {
    // Debug 用于调试信息输出
    Debug(msg string, fields ...field.Field)
    // Info 普通消息记录
    Info(msg string, fields ...field.Field)
    // Warn 告警信息记录
    Warn(msg string, fields ...field.Field)
    // Error 错误信息记录
    Error(msg string, fields ...field.Field)
    // Critical 异常信息记录
    Critical(msg string, fields ...field.Field)
    // Fixed 固定信息记录，用于服务运行状态或者输出加载配置信息等等。
    Fixed(msg string, fields ...field.Field)
}

// ILogger 日志接口
type ILogger interface {
    IBaseLogger

    // Sync 同步操作,刷新缓冲区，建议退出或删除 logger 前调用一次。
    Sync() error
    // Named 日志对象命名
    Named(s string)
    // With 附加信息到日志对象，并返回
    With(fields ...field.Field) *Logger
}

func New(cfg config.Config, options ...Option) (*Logger, error) {
    // 创建 数据编码器
    enc, err := encoder.NewEncoder(cfg.Encoding, cfg.EncoderConfig)
    if err != nil {
        return nil, err
    }

    // 开启 开发模式
    if cfg.Development {
        options = append(options, Development())
    }

    // 设置 caller，用于记录调用函数文件和调用位置行数
    if cfg.EnableCaller {
        options = append(options, AddCaller())
    }

    // 设置 调用栈信息 捕获处理等级
    stackLevel := level.StackLevel{Level:CRITICAL}
    if cfg.EnableStacktrace {
        if cfg.Development {
            stackLevel = level.StackLevel{Level:WARN}
        }
    }

    // callerSkip +1 主要是因为这边 logger 套了一层 zap.Logger所以需要 +1
    options = append([]Option{AddCallerSkip(1),AddStacktrace(stackLevel)}, options...)
    // if cfg.Sampling != nil {
    //	options = append(options, WrapCore(func(core Core) Core {
    //		return zapcore.NewSampler(core, time.Second, int(cfg.Sampling.Initial), int(cfg.Sampling.Thereafter))
    //	}))
    // }

    if len(cfg.InitialFields) > 0 {
        fs := make([]field.Field, 0, len(cfg.InitialFields))
        keys := make([]string, 0, len(cfg.InitialFields))
        for k := range cfg.InitialFields {
            keys = append(keys, k)
        }
        sort.Strings(keys)
        for _, k := range keys {
            fs = append(fs, Any(k, cfg.InitialFields[k]))
        }
        options = append(options, Fields(fs...))
    }

    var iw writer.WriteSyncer
    var l *Logger
    iw = cfg.Writer

    iCore := core.NewCore(enc, iw, cfg.Level)
    l = NewWithCore(iCore, options...)


    if cfg.Name != ""{
        l.Named(cfg.Name)
    }
    // 注册 logger 到全局管理器中
    if cfg.ID != ""{
        l.ID = cfg.ID
        err = registerLogger(l,cfg.ForceReplace)
    }

    return l, err
}

// NewWithCore 通过 自定义 core 创建 logger
// 自定义 core 可以实现 输出不同等级日志到不同输入源中
func NewWithCore(core core.Core, options ...Option) *Logger {
    if core == nil {
        return NewNop(options...)
    }
    log := &Logger{
        core:        core,
        errorOutput: writer.Lock(os.Stderr),
        addStack:   CRITICAL + 1,
        //     // 默认使用本地时区
        Location: Local,
    }
    return log.WithOptions(options...)
}
// NewNop returns a no-op Logger. It never writes out logs or internal errors,
// and it never runs user-defined hooks.
//
// Using WithOptions to replace the Core or error output of a no-op Logger can
// re-enable logging.
func NewNop(options ...Option) *Logger {
    log :=&Logger{
        core:       core.NewNopCore(),
        errorOutput: writer.AddSync(ioutil.Discard),
        addStack:    CRITICAL,
        //     // 默认使用本地时区
        Location: Local,
    }

    return log.WithOptions(options...)
}

// Logger 日志结构体
type Logger struct {
    // core  日志对象 核心，用于编码日志数据。
    core core.Core
    // ID logger 标志，用于在 logger 管理器中识别该 Logger
    // 一般情况下 ID 与 Name 一样。
    ID string
    development bool
    // Name 本日志对象的名称，用于在 日志输出时，设置 NameKey 的值
    Name        string
    // 错误输出，主要用于 logger 内部错误时进行输出。
    errorOutput writer.WriteSyncer

    // 是否 输出调用函数信息
    addCaller bool
    // 是否输出错误栈信息
    addStack  level.LevelEnabler
    // 输出栈信息时，需要跳过多少个 栈信息
    callerSkip int
    // Location 日志时区，在创建时，注意该配置需要设置，否则将会出现异常。
    Location *time.Location
}

func (l *Logger) Named(s string){

    // if s == "" {
    //     return l
    // }
    // log := l.clone()
    // if log.Name == "" {
    //     log.Name = s
    // } else {
    //     log.Name = strings.Join([]string{log.Name, s}, ".")
    // }
    l.Name = s
    // return log
}
func (l *Logger) With(fields ...field.Field) *Logger {
    if len(fields) == 0 {
        return l
    }
    log := l.clone()
    log.core =log.core.With(fields)
    return log
}
// WithOptions clones the current Logger, applies the supplied Options, and
// returns the resulting Logger. It's safe to use concurrently.
func (log *Logger) WithOptions(opts ...Option) *Logger {
    c := log.clone()
    for _, opt := range opts {
        opt.Apply(c)
    }
    return c
}

// Debug 调试信息
func (l *Logger) Debug(msg string, fields ...field.Field) {
    if ce := l.Check(DEBUG, msg); ce != nil {
        ce.Write(fields...)
    }
}

// Info 默认的消息
func (l *Logger) Info(msg string, fields ...field.Field) {
    if ce := l.Check(INFO, msg); ce != nil {
        ce.Write(fields...)
    }
}

// Warn 警告级别消息
func (l *Logger) Warn(msg string, fields ...field.Field) {
    if ce := l.Check(WARN, msg); ce != nil {
        ce.Write(fields...)
    }
}

// Error 错误级别的消息，不会输出堆栈信息（如果设置了自动记录堆栈）
func (l *Logger) Error(msg string, fields ...field.Field) {
    if ce := l.Check(ERROR, msg); ce != nil {
        ce.Write(fields...)
    }
}

// Critical 程序异常或崩溃记录信息，输出堆栈信息（如果设置了自动记录堆栈）
func (l *Logger) Critical(msg string, fields ...field.Field) {

    if ce := l.Check(CRITICAL, msg); ce != nil {

        ce.Write(fields...)
    }
    return
}

// Fixed 固定消息，主要用于输出服务运行日志等等，等级最高，且不输出堆栈信息
func (l *Logger) Fixed(msg string, fields ...field.Field) {
    if ce := l.Check(FIXED, msg); ce != nil {
        ce.Write(fields...)
    }
    return
}

// Sync 需要忽略错误
// https://github.com/uber-go/zap/issues/328
func (l *Logger) Sync() error {
    return l.core.Sync()
}
// Core returns the Logger's underlying zapcore.Core.
func (l *Logger) Core() core.Core {
    return l.core
}

func (l *Logger) clone() *Logger {
    copyLog := *l
    return &copyLog
}

func (l *Logger) Switch() *WLogger {
    cloneCore := l.clone()
    cloneCore.callerSkip += 2
    return &WLogger{base:cloneCore}
}

func (l *Logger) Check(lvl level.Level, msg string) *core.CheckedEntry {
    // check must always be called directly by a method in the Logger interface
    // (e.g., Check, Info, Fatal).
    const callerSkipOffset = 1

    // Create basic checked entry thru the core; this will be non-nil if the
    // log message will actually be written somewhere.
    ent := entry.Entry{
        LoggerName: l.Name,
        Time:       time.Now().In(l.Location),
        Level:      lvl,
        Message:    msg,
    }
    ce := l.core.Check(ent, nil)
    willWrite := ce != nil

    // 设置 终端 行为，比如在开发模式下 CRITICAL
    switch ent.Level {
    case CRITICAL:
        if l.development {
            ce = ce.Should(ent, core.WriteThenPanic)
        }
    }

    // Only do further annotation if we're going to write this message; checked
    // entries that exist only for terminal behavior don't benefit from
    // annotation.
    if !willWrite {
        return ce
    }

    // 设置 错误输出 到 CheckedEntry 对象中
    ce.ErrorOutput = l.errorOutput
    if l.addCaller {
        ce.Entry.Caller = core.NewEntryCaller(runtime.Caller(l.callerSkip + callerSkipOffset))
        if !ce.Entry.Caller.Defined {
            fmt.Fprintf(l.errorOutput, "%v Logger.check error: failed to get caller\n", time.Now().UTC())
            l.errorOutput.Sync()
        }
    }
    if l.addStack.Enabled(ce.Entry.Level) {
        ce.Entry.Stack = StackSkip("",l.callerSkip+1,0).String
    }

    return ce
}