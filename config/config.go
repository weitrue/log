package config

import (
    "github.com/weitrue/log/level"
    "github.com/weitrue/log/writer"
    "go.uber.org/zap/zapcore"
)

type  EncoderConfig = zapcore.EncoderConfig

// Config 提供了一种构造记录器的声明方式。
// 它没有做任何使用New，Options和各种zapcore.WriteSyncer和zapcore.Core包装器无法完成的任务，但它是一种更简单的方法来切换常见选项。
// 请注意，Config仅故意支持最常用的选项。
// 可以使用更多不寻常的日志记录设置（记录到网络连接或消息队列，在多个文件之间拆分输出等），但需要直接使用zapcore软件包。
// 有关示例代码，请参阅包级别的BasicConfiguration和AdvancedConfiguration示例。
// 有关显示运行时日志级别更改的示例，请参阅AtomicLevel的文档。
type Config struct {
    // Name 日志名称，用于多个日志对象输出时，识别日志分类.
    // 默认为"",如果设置该字段，并且 encoderConfig 中 NameKey 也设置的则在日志数据中输出 Name。
    Name string `json:"name" yaml:"name"`
    // 设置 日志 ID，该值不为空时，才进行注册创建的日志对象到 全局管理器中。一般情况下，ID和Name的值一致。
    ID string `json:"id" yaml:"id"`
    // ForceReplace 如果注册 日志对象到 全局管理器冲突时，是否强制替换旧日志对象
    ForceReplace bool `json:"forceReplace" yaml:"forceReplace"`
    // Level 最低允许记录等级. 该对象为引用对象，
    // 可以通过调用 Config.Level.SetLevel 来动态调整等级，并影响 log 的对象树下的所有 log
    Level level.AtomicLevel `json:"level" yaml:"level"`

    // CustomLevelEnabler 自定义 LevelEnabler 函数，用于判断允许什么等级的日志进行输出。
    // 默认使用 Level 对象的 Enabled 函数（大于等于该级别才允许输出），如果设置该字段函数，则替换使用。
    // 且在Level进行重置时，该判断函数不会被覆盖。
    // CustomLevelEnabler level.LevelEnablerFunc
    // Development 调整 log 为开发模式，主要调整 异常栈捕获流程和 Critical 的行为。
    // 当设置为 true 时， Critical 会触发 panic 操作
    Development bool `json:"development" yaml:"development"`
    // EnableCaller 日志数据自动记录调用文件名和调用位置行数.
    // 默认情况下，不会进行记录的。
    EnableCaller bool `json:"enableCaller" yaml:"enableCaller"`
    // EnableStacktrace 开启自动获取调用堆栈信息.
    // 开启的情况下，stacktraces 在 development 模式下会捕获 WarnLevel 级别以上的堆栈信息，
    // 在 production 场景中则是捕获 ErrorLevel 级别以上的堆栈信息。
    EnableStacktrace bool `json:"enableStacktrace" yaml:"enableStacktrace"`

    // Sampling 统计重复日志，并优化.
    // 关闭该功能,TODO: 后续移植过来
    // Sampling *SamplingConfig `json:"sampling" yaml:"sampling"`

    // Encoding 设置日志编码器. 默认设置有 "json" 和 "console",
    // 通过 RegisterEncoder 设置自定义编码器.
    Encoding string `json:"encoding" yaml:"encoding"`
    // EncoderConfig sets options for the chosen encoder. See
    // zapcore.EncoderConfig for details.
    EncoderConfig EncoderConfig `json:"encoderConfig" yaml:"encoderConfig"`

    // InitialFields 初始字段设置，一般用于设置每条日志都会记录的默认数据，比如 服务名
    // 也可以在 log 创建后，通过 With 来增加默认日志数据
    InitialFields map[string]interface{} `json:"initialFields" yaml:"initialFields"`

    // Writer 自定义 writer，用于日志数据输出，可以实现多数据通道数据
    Writer writer.WriteSyncer
}


