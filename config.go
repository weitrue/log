package log

import (
    "github.com/weitrue/log/config"
    "github.com/weitrue/log/encoder"
    "github.com/weitrue/log/level"
    "github.com/weitrue/log/writer"
    "go.uber.org/zap/zapcore"
)


// NewProductionEncoderConfig returns an opinionated EncoderConfig for
// production environments.
func NewProductionEncoderConfig() config.EncoderConfig {
    return config.EncoderConfig{
        TimeKey:        "generated_time",
        LevelKey:       "level",
        NameKey:        "log",
        CallerKey:      "caller",
        MessageKey:     "msg",
        StacktraceKey:  "stack",
        LineEnding:     encoder.DefaultLineEnding,
        EncodeLevel:    encoder.CapitalLevelEncoder,
        EncodeTime:     encoder.RFC3339TimeEncoder,
        // EncodeTime:     encoder.EpochTimeIntEncoder,
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }
}

// NewProductionEncoderWithESConfig 针对常用的 ES日志（格式）
// 将 NameKey 设置为 @fluentd_tag，用于 ES 索引
// 如果 log.Name不方便设置，可以通过 log.With操作或者New的时候的Fields的Options操作来手动设置该额外字段数据
// ES 的时间支持 RFC3339
// TimeKey 默认设置为 log_time ，部分日志使用的是 generated_time
func NewProductionEncoderWithESConfig() config.EncoderConfig {
    return config.EncoderConfig{
        TimeKey:        "generated_time",
        LevelKey:       "level",
        NameKey:        "@fluentd_tag",
        CallerKey:      "caller",
        MessageKey:     "msg",
        StacktraceKey:  "stack",
        LineEnding:     encoder.DefaultLineEnding,
        EncodeLevel:    encoder.CapitalLevelEncoder,
        EncodeTime:     encoder.RFC3339TimeEncoder,
        EncodeDuration: zapcore.SecondsDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }
}

// NewProductionWithESConfig 针对 ES 日志格式设置 encoder 配置，
// 细节参考 NewProductionEncoderWithESConfig 的注释说明
func NewProductionWithESConfig(writers ...writer.WriteSyncer) config.Config {
    cfg :=  config.Config{
        Level:       level.NewAtomicLevelAt(INFO),
        Development: false,
        Encoding:         "json",
        EncoderConfig:    NewProductionEncoderWithESConfig(),
    }

    if len(writers) >0 {
        cfg.Writer = writer.NewMultiWriteSyncer(writers...)
    }

    return cfg
}
// NewProductionConfig is a reasonable production logging configuration.
// Logging is enabled at InfoLevel and above.
//
// It uses a JSON encoder, writes to standard error, and enables sampling.
// Stacktraces are automatically included on logs of ErrorLevel and above.
func NewProductionConfig(writers ...writer.WriteSyncer) config.Config {
    cfg :=  config.Config{
        Level:       level.NewAtomicLevelAt(INFO),
        Development: false,
        // Sampling: &SamplingConfig{
        //    Initial:    100,
        //    Thereafter: 100,
        // },
        EnableCaller:true,
        Encoding:         "json",
        EncoderConfig:    NewProductionEncoderConfig(),
    }

    if len(writers) >0 {
        cfg.Writer = writer.NewMultiWriteSyncer(writers...)
    }

    return cfg
}

// NewDevelopmentEncoderConfig returns an opinionated EncoderConfig for
// development environments.
func NewDevelopmentEncoderConfig() config.EncoderConfig {
    return config.EncoderConfig{
        // Keys can be anything except the empty string.
        TimeKey:        "generated_time",
        LevelKey:       "level",
        NameKey:        "log",
        CallerKey:      "caller",
        MessageKey:     "msg",
        StacktraceKey:  "stack",
        LineEnding:     encoder.DefaultLineEnding,
        EncodeLevel:    encoder.CapitalLevelEncoder,
        EncodeTime:     encoder.RFC3339TimeEncoder,
        EncodeDuration: encoder.StringDurationEncoder,
        EncodeCaller:   zapcore.ShortCallerEncoder,
    }
}

// NewDevelopmentConfig is a reasonable development logging configuration.
// Logging is enabled at DebugLevel and above.
//
// It enables development mode (which makes DPanicLevel logs panic), uses a
// console encoder, writes to standard error, and disables sampling.
// Stacktraces are automatically included on logs of WarnLevel and above.
func NewDevelopmentConfig(writers ...writer.WriteSyncer) config.Config {
    cfg :=  config.Config{
        Level:            level.NewAtomicLevelAt(DEBUG),
        // Development 调整 log 为开发模式，主要调整 异常栈捕获流程和 Critical 的行为。
        //    // 当设置为 true 时， Critical 会触发 panic 操作
        Development:      true,
        EnableCaller:true,
        Encoding:         "console",
        EncoderConfig:    NewDevelopmentEncoderConfig(),
    }
    if len(writers) >0 {
        cfg.Writer = writer.NewMultiWriteSyncer(writers...)
    }

    return cfg
}