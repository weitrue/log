package log

import (
    "fmt"

    "github.com/weitrue/log/field"
    "github.com/weitrue/log/level"
    "go.uber.org/multierr"
    "go.uber.org/zap/zapcore"
)

const (
    _oddNumberErrMsg    = "Ignored key without a value."
    _nonStringKeyErrMsg = "Ignored key-value pairs with non-string keys."
)

// IBaseLogger 基础日志接口
type IBaseWLogger interface {
    // Debug 用于调试信息输出
    Debug(msg string, keysAndValues ...interface{})
    Debugf(template string, args ...interface{})
    // Info 普通消息记录
    Info(msg string, keysAndValues ...interface{})
    Infof(template string, args ...interface{})
    // Warn 告警信息记录
    Warn(msg string, keysAndValues ...interface{})
    Warnf(template string, args ...interface{})

    // Error 错误信息记录
    Error(msg string, keysAndValues ...interface{})
    Errorf(template string, args ...interface{})

    // Critical 异常信息记录
    Critical(msg string, keysAndValues ...interface{})
    Criticalf(template string, args ...interface{})
    // Fixed 固定信息记录，用于服务运行状态或者输出加载配置信息等等。
    Fixed(msg string, keysAndValues ...interface{})
    Fixedf(template string, args ...interface{})
}

// ILogger 日志接口
type IWLogger interface {
    IBaseWLogger

    // Sync 同步操作,刷新缓冲区
    Sync() error
    // Named 日志对象命名
    Named(s string) *WLogger
    // With 附加信息到日志对象，并返回
    With(args ...interface{}) *WLogger
}

// WLogger world logger
// 该loger下，Debug 等函数的keysAndValues会自动解析为field，因此 kv值需要成对。
// 对于 Debugf 的操作，不再支持 field 数据。跟 prinf 的使用是一样的。
type WLogger struct {
    base *Logger
    Level level.AtomicLevel
}

func (l *WLogger) Debug(msg string, keysAndValues ...interface{}) {
    l.log(DEBUG, msg, nil, keysAndValues)
}

// Debugf uses fmt.Sprintf to log a templated message.
func (s *WLogger) Debugf(template string, args ...interface{}) {
    s.log(DEBUG, template, args, nil)
}

func (l *WLogger) Info(msg string, keysAndValues ...interface{}) {
    l.log(INFO, msg, nil, keysAndValues)
}
// Infof uses fmt.Sprintf to log a templated message.
func (s *WLogger) Infof(template string, args ...interface{}) {
    s.log(INFO, template, args, nil)
}
func (l *WLogger) Warn(msg string, keysAndValues ...interface{}) {
    l.log(WARN, msg, nil, keysAndValues)
}
// Warnf uses fmt.Sprintf to log a templated message.
func (s *WLogger) Warnf(template string, args ...interface{}) {
    s.log(WARN, template, args, nil)
}
func (l *WLogger) Error(msg string, keysAndValues ...interface{}) {
    l.log(ERROR, msg, nil, keysAndValues)

}
// Errorf uses fmt.Sprintf to log a templated message.
func (s *WLogger) Errorf(template string, args ...interface{}) {
    s.log(ERROR, template, args, nil)
}
func (l *WLogger) Critical(msg string, keysAndValues ...interface{}) {
    l.log(CRITICAL, msg, nil, keysAndValues)
}
// Criticalf uses fmt.Sprintf to log a templated message.
func (s *WLogger) Criticalf(template string, args ...interface{}) {
    s.log(CRITICAL, template, args, nil)
}
func (l *WLogger) Fixed(msg string, keysAndValues ...interface{}) {
    l.log(FIXED, msg, nil, keysAndValues)
}
// Fixedf uses fmt.Sprintf to log a templated message.
func (s *WLogger) Fixedf(template string, args ...interface{}) {
    s.log(FIXED, template, args, nil)
}
// Sync 需要忽略错误
// https://github.com/uber-go/zap/issues/328
func (l *WLogger) Sync() error {
    return l.base.Sync()
}

func (l *WLogger) Named(name string) {
     l.base.Named(name)
}

func (l *WLogger) With(args ...interface{}) *WLogger {
    return &WLogger{base: l.base.With(l.sweetenFields(args)...)}
}
func (l *WLogger) Switch() *Logger {
    base := l.base.clone()
    base.callerSkip -= 2
    return base
}
func (l *WLogger) log(lvl zapcore.Level, template string, fmtArgs []interface{}, context []interface{}) {
    // If logging at this level is completely disabled, skip the overhead of
    // string formatting.
    if lvl < CRITICAL && !l.base.Core().Enabled(lvl) {
        return
    }

    // Format with Sprint, Sprintf, or neither.
    msg := template
    // 用于实现 infof()操作.format
    if msg == "" && len(fmtArgs) > 0 {
       msg = fmt.Sprint(fmtArgs...)
    } else if msg != "" && len(fmtArgs) > 0 {
       msg = fmt.Sprintf(template, fmtArgs...)
    }

    if ce := l.base.Check(lvl, msg); ce != nil {
        ce.Write(l.sweetenFields(context)...)
    }
}

func (l *WLogger) sweetenFields(args []interface{}) []field.Field {
    if len(args) == 0 {
        return nil
    }

    // 预分配 map 空间。
    fields := make([]field.Field, 0, len(args))
    var invalid invalidPairs

    for i := 0; i < len(args); {
        // This is a strongly-typed field. Consume it and move on.
        if f, ok := args[i].(field.Field); ok {
            fields = append(fields, f)
            i++
            continue
        }

        // 确保 拓展 的 keyval 是成对的.
        if i == len(args)-1 {
            l.base.Error(_oddNumberErrMsg, Any("ignored", args[i]))
            break
        }

        // 使用此值和下一个值，将它们视为键值对。
        // 如果 key 不是字符串，则将此对数据添加到无效数据的切片中。
        key, val := args[i], args[i+1]
        if keyStr, ok := key.(string); !ok {
            // 可能会出现后续错误，因此事先分配一次。
            if cap(invalid) == 0 {
                invalid = make(invalidPairs, 0, len(args)/2)
            }
            invalid = append(invalid, invalidPair{i, key, val})
        } else {
            fields = append(fields, Any(keyStr, val))
        }
        i += 2
    }

    // If we encountered any invalid key-value pairs, log an error.
    if len(invalid) > 0 {
        l.base.Error(_nonStringKeyErrMsg, field.Array("invalid", invalid))
    }
    return fields
}

type invalidPair struct {
    position   int
    key, value interface{}
}

func (p invalidPair) MarshalLogObject(enc zapcore.ObjectEncoder) error {
    enc.AddInt64("position", int64(p.position))
    Any("key", p.key).AddTo(enc)
    Any("value", p.value).AddTo(enc)
    return nil
}

type invalidPairs []invalidPair

func (ps invalidPairs) MarshalLogArray(enc zapcore.ArrayEncoder) error {
    var err error
    for i := range ps {
        err = multierr.Append(err, enc.AppendObject(ps[i]))
    }
    return err
}