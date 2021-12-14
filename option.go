package log

import (
    "github.com/weitrue/log/core"
    "github.com/weitrue/log/field"
    "github.com/weitrue/log/level"
    "github.com/weitrue/log/writer"
    "time"
)


// An Option configures a Logger.
type Option interface {
    Apply(*Logger)
}

// optionFunc wraps a func so it satisfies the Option interface.
type optionFunc func(*Logger)

func (f optionFunc) Apply(log *Logger) {
    f(log)
}

// WrapCore wraps or replaces the Logger's underlying zapcore.Core.
func WrapCore(f func(core.Core) core.Core) Option {
    return optionFunc(func(log *Logger) {
        log.core = f(log.core)
    })
}


// Fields 添加 fields 到 logger 中，类似 With 操作
func Fields(fs ...field.Field) Option {
    return optionFunc(func(log *Logger) {
        log.core = log.core.With(fs)
    })
}

// ErrorOutput 设置 log 内部错误输出缓冲区
func ErrorOutput(output writer.WriteSyncer) Option {
    return optionFunc(func(log *Logger) {
        log.errorOutput = output
    })
}


// Development 开发者模式的配置，主要是该配置下，Critical 会触发 panic
func Development() Option {
    return optionFunc(func(log *Logger) {
        log.development = true
    })
}

// AddCaller Logger记录日志时，自动 添加调用函数信息
func AddCaller() Option {
    return optionFunc(func(log *Logger) {
        log.addCaller = true
    })
}

// AddCallerSkip increases the number of callers skipped by caller annotation
// (as enabled by the AddCaller option). When building wrappers around the
// Logger and SugaredLogger, supplying this Option prevents zap from always
// reporting the wrapper code as the caller.
func AddCallerSkip(skip int) Option {
    return optionFunc(func(log *Logger) {
        log.callerSkip += skip
    })
}

// AddStacktrace configures the Logger to record a stack trace for all messages at
// or above a given level.
func AddStacktrace(lvl level.LevelEnabler) Option {
    return optionFunc(func(log *Logger) {
        log.addStack = lvl
    })
}

// Location 配置日志时间时区
func Location(location *time.Location) Option {
    return optionFunc(func(log *Logger) {
        log.Location = location
    })
}





