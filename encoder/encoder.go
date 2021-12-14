package encoder

import (
    "errors"
    "fmt"
    "github.com/weitrue/log/config"
    "github.com/weitrue/log/field"
    "github.com/weitrue/log/level"
    "go.uber.org/zap/zapcore"
    "sync"
    "time"
)

type Encoder = zapcore.Encoder
type PrimitiveArrayEncoder = zapcore.PrimitiveArrayEncoder
var DefaultLineEnding = zapcore.DefaultLineEnding
var StringDurationEncoder = zapcore.StringDurationEncoder
var ShortCallerEncoder = zapcore.ShortCallerEncoder

var (
    errNoEncoderNameSpecified = errors.New("no encoder name specified")

    encoders = map[string]func(config.EncoderConfig) (Encoder, error){
        ConsoleEncoding: func(encoderConfig config.EncoderConfig) (Encoder, error) {
            return NewConsoleEncoder(encoderConfig), nil
        },
        JsonEncoding: func(encoderConfig config.EncoderConfig) (Encoder, error) {
            return NewJSONEncoder(encoderConfig), nil
        },
    }
    _encoderMutex sync.RWMutex

)

// RegisterEncoder registers an encoder constructor, which the Config struct
// can then reference. By default, the "json" and "console" encoders are
// registered.
//
// Attempting to register an encoder whose name is already taken returns an
// error.
func RegisterEncoder(name string, constructor func(config.EncoderConfig) (Encoder, error)) error {
    _encoderMutex.Lock()
    defer _encoderMutex.Unlock()
    if name == "" {
        return errNoEncoderNameSpecified
    }
    if _, ok := encoders[name]; ok {
        return fmt.Errorf("encoder already registered for name %q", name)
    }
    encoders[name] = constructor
    return nil
}

// NewEncoder 创建一个日记数据编码器。主要对 message，time等字段数据进行序列化编码。
// 比如 json序列化编码器，或者直接输出日志信息的console 编码器。
func NewEncoder(name string, encoderConfig config.EncoderConfig) (Encoder, error) {
    _encoderMutex.RLock()
    defer _encoderMutex.RUnlock()
    if name == "" {
        return nil, errNoEncoderNameSpecified
    }
    constructor, ok := encoders[name]
    if !ok {
        return nil, fmt.Errorf("no encoder registered for name %q", name)
    }
    return constructor(encoderConfig)
}
// NewConsoleEncoder 控制台输出数据编码器
//var NewConsoleEncoder = zapcore.NewConsoleEncoder

// NewJSONEncoder 将日志数据编码为 json 字符串数据
//var NewJSONEncoder = zapcore.NewJSONEncoder

type ArrayMarshaler = zapcore.ArrayMarshaler
type ObjectMarshaler = zapcore.ObjectMarshaler

// FullNameEncoder 序列化 logger name.
var FullNameEncoder = zapcore.FullNameEncoder


// 时间序列化

// ISO8601TimeEncoder serializes a time.Time to a floating-point number of seconds
// since the Unix epoch.
// Output:2019-02-22T11:48:32.509+0800
var ISO8601TimeEncoder = zapcore.ISO8601TimeEncoder

var RFC3339MS = "2006-01-02T15:04:05.000Z07:00"

// RFC3339TimeEncoder FRC3339 格式化时间字符串，主要用于 kibana 时间排序。
// // Output:2019-02-22T11:48:32.509+08:00
func RFC3339TimeEncoder(t time.Time, enc PrimitiveArrayEncoder) {
    // 1 allocs
    enc.AppendString(t.Format(RFC3339MS))
}
// EpochMillisTimeIntEncoder 时间戳，毫秒级
func EpochMillisTimeIntEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
    nanos := t.UnixNano()
    millis := int64(nanos) / int64(time.Millisecond)
    enc.AppendInt64(millis)
}

// EpochTimeIntEncoder 时间戳，秒级
// Output:1550814085
func EpochTimeIntEncoder(t time.Time, enc zapcore.PrimitiveArrayEncoder) {
    nanos := t.UnixNano()
    sec := int64(nanos) / int64(time.Second)
    enc.AppendInt64(sec)
}


// CapitalLevelEncoder 序列化日志等级为大写字符串.
// 例如： InfoLevel 转为 INFO
func CapitalLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
    enc.AppendString(level.Level2CapitalName(l))
}

// LowercaseLevelEncoder 序列化日志等级为小写字符串.
// 例如： InfoLevel 转为 info
func LowercaseLevelEncoder(l zapcore.Level, enc zapcore.PrimitiveArrayEncoder) {
    enc.AppendString(level.Level2Name(l,false))
}

func AddFields(enc zapcore.ObjectEncoder, fields []field.Field) {
    for i := range fields {
        fields[i].AddTo(enc)
    }
}