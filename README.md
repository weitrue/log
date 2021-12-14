# logger

logger 日志基础模块,该模块以 zap 开源模块为基础进行二次开发。

## 安装

`go get github.com/weitrue/log`

## 接口总览

> NOTICE: log 库使用`atomic.Value`来处理全局log接口函数，以期避免`DATA RACE`问题，因此会有些微性能影响，我们推荐自行创建出`Logger`对象，并自行在程序中传递。  
> 一般使用`New`构建的`Logger`也可以通过`GetLogger`函数来获取`Logger`对象。
```go
// Debug 用于调试信息输出
Debug(msg string, fields ...Field)
// Info 普通消息记录
Info(msg string, fields ...Field)
// Warn 告警信息记录
Warn(msg string, fields ...Field)
// Error 错误信息记录
Error(msg string, fields ...Field)
// Critical 异常信息记录
Critical(msg string, fields ...Field)
// Fixed 固定信息记录，用于服务运行状态或者输出加载配置信息等等。
Fixed(msg string, fields ...Field)
```

## 使用示例

使用全局logger对象
```go
import "github.com/weitrue/log"
// 记录日志
log.Info("field log",
    String("new-s", "val-s"),
)
// Output:
// {"level":"INFO","ts":"2019-01-01T09:12:34.483+08:00","log":"lognametest","msg":"field log","new-s":"val-s"}


```

```go
// 创建默认 配置,输出到标准输出
cfg := log.NewProductionConfig(os.Stdout)
cfg.Name = "lognametest"

// 创建 日志对象
logger, err := log.New(cfg)

// 记录日志
logger.Info("field log",
    String("new-s", "val-s"),
)
// Output:
// {"level":"INFO","ts":"2019-01-01T09:12:34.483+08:00","log":"lognametest","msg":"field log","new-s":"val-s"}

```

记录 caller 和 stack

```go
cfg := log.NewProductionConfig()

// 创建 日志对象
// AddCaller 自动添加 调用函数信息：文件名：行数
// AddStacktrace Error以上级别自动添加 调用栈信息
logger, err := log.New(cfg, AddCaller(), AddStacktrace(ERROR))
assert.NoError(t, err)
logger.Info("field log",
    String("new-s", "val-s"),
)
// Output:
// {"level":"INFO","ts":"2019-01-01T09:12:34.483+0800","caller":"log/log_test.go:134","msg":"field log","new-s":"val-s"}

logger.Error("err-log",
    String("new-s", "val-s"),
)
// Output:
// {"level":"ERROR","ts":"2019-01-01T09:12:34.483+0800","caller":"log/log_test.go:145","msg":"err-log","new-s":"val-s","stack":"github.com/weitrue/log.TestSimpleLog.func3\n\tG:/code/go/src/github.com/weitrue/log/log_test.go:145\ntesting.tRunner\n\tC:/Go/src/testing/testing.go:827"}

logger.Critical("err-log",
    String("new-s", "val-s"),
)
// Output:
// {"level":"CRITICAL","ts":"2019-01-01T09:12:34.483+0800","caller":"log/zap.go:37","msg":"err-log","new-s":"val-s","stack":"github.com/weitrue/log.(*Logger).Critical\n\tG:/code/go/src/github.com/weitrue/log/zap.go:37\ngithub.com/weitrue/log.TestSimpleLog.func3\n\tG:/code/go/src/github.com/weitrue/log/log_test.go:151\ntesting.tRunner\n\tC:/Go/src/testing/testing.go:827"}

// 手动输出栈信息
logger.Info("err-log",
    log.Stack(""),
)
```

使用manager获取logger
```go

// 获取全局默认对象
// DefaultName = "root"
logger,existed := log.GetLogger(DefaultName)
// 自定义创建的logger
// 在使用New进行创建 logger 时，根据配置一般会自动调用
// log.RegisterLogger()注册到全局管理器中。
// 如果没有，可以手动调用log.RegisterLogger或者log.ReplaceLogger
// log.ReplaceLogger主要用于更新logger的时候，强制删除旧管理器上的对象，放入新的logger。

```

日志对象固定输出某些额外字段数据

```go
// With 操作会复制出一个logger，不影响 原始 logger
newLog := log.With(String("tag", "serverRun"))
newLog.Error("more error")
// Output:
// {"level":"ERROR","ts":"2019-01-01T09:12:34.567+08:00","msg":"more error","tag":"serverRun"}

// 后续使用 newlog 来输出日志，都会携带 "tag":"testInerface"
```

收集field，最后统一输出
```go
fieldArr := log.CollectField(String("k1","v1"))

fieldArr.Append(String("k2","v2"),Float64("f3",123.4))
fieldArr.Append(Int("k4",567))
l.Info("Debug log",fieldArr...)
// output:
// {"level":"INFO","ts":"2019-01-01T09:12:34.456+08:00","msg":"Debug log","k1":"v1","k2":"v2","f3":123.4,"k4":567}
		
```

自定义log配置

```go

config.Config{
    // 日志输出等级设置，一般情况下，低于该等级的日志不输出
    Level:       level.NewAtomicLevelAt(INFO),
    // Name 日志名称，用于多个日志对象输出时，识别日志分类.
    // 默认为"",如果设置该字段，并且 encoderConfig 中 NameKey 也设置的则在日志数据中输出 Name。
    Name:"mylog",
    // 设置 日志 ID，该值不为空时，才进行注册创建的日志对象到 全局管理器中。一般情况下，ID和Name的值一致。
    ID:"mylog",
    // ForceReplace 如果注册 日志对象到 全局管理器冲突时，是否强制替换旧日志对象
    ForceReplace: true,
    // EnableCaller 日志数据自动记录调用文件名和调用位置行数.
    // 默认情况下，不会进行记录的。
    EnableCaller:true,
    // EnableStacktrace 开启自动获取调用堆栈信息.
    // 开启的情况下，stacktraces 在 development 模式下会捕获 WarnLevel 级别以上的堆栈信息，
    // 在 production 场景中则是捕获 ErrorLevel 级别以上的堆栈信息。
    EnableStacktrace: true,
    // Development 调整 log 为开发模式，主要调整 异常栈捕获流程和 Critical 的行为。
    // 当设置为 true 时， Critical 会触发 panic 操作
    Development: false,
    // Encoding 设置日志编码器. 默认设置有 "json" 和 "console",
    // 通过 RegisterEncoder 设置自定义编码器.
    Encoding:         "json",
    // InitialFields 初始字段设置，一般用于设置每条日志都会记录的默认数据，比如 服务名
    // 也可以在 log 创建后，通过 With 来增加默认日志数据
    InitialFields: map[string]interface{}{
      "global_trace_id":"1234567",
    },
    Writer: os.Stdout,
    EncoderConfig:    config.EncoderConfig{
        // 记录时间的Key
        TimeKey:        "T",
        // 记录日志等级的Key
        LevelKey:       "L",
        // 记录日志名的Key
        NameKey:        "N",
        // 默认记录 记录调用函数数据的Key
        CallerKey:      "C",
        // 默认记录 记录调日志消息的Key
        MessageKey:     "M",
        // 默认记录 栈信息 的Key
        StacktraceKey:  "S",
        // 默认换行符
        LineEnding:     encoder.DefaultLineEnding,
        // 将 日志等级序列化成 字段串 的编码器
        EncodeLevel:    encoder.CapitalLevelEncoder,
        // 将 日志时间 序列化成 字段串 的编码器
        EncodeTime:     encoder.RFC3339TimeEncoder,
        // 将 数据duration 数据序列化成 字段串 的编码器
        EncodeDuration: encoder.StringDurationEncoder,
        // 将 caller 数据序列化成 字段串 的编码器
        EncodeCaller:   encoder.ShortCallerEncoder,
    },
}


```

关于发送到ES的日志格式/配置：
可以使用默认配置`NewProductionWithESConfig`来创建日志对象配置
该操作设置了encoder配置，将`NameKey`设置为`@fluentd_tag`，用于 ES 索引,
如果`log.Name`不方便设置，可以通过`log.With`操作或者`New`的时候`Fields`的`Options`操作来手动设置该额外字段数据。

`ES`的时间支持`RFC3339`，因此默认时间`encoder`是`RFC3339TimeEncoder`
`TimeKey`默认设置为`log_time`，部分日志使用的是`generated_time`，如果有需要，可以自行修改。

##### 使用 flumefilewriter

```go
/**
	创建WriteHandle,其中前七个为必要参数 :
	1.RootPath :指包含分片目录的根目录,每个分片目录下必须有multiplexing or replicating文件夹
	2.tempFilePath : 临时的分片目录,如果所有分片目录大于maxFileCount,log文件直接写到该目录下
	3.tableName : 服务名
	4.sendingMode : 日志发送方式：multiplexing or replicating(发送到es以及hdfs)
	5.selectorType : ES:es集群,HDFS:hdfs一级分区,HDFS2:hdfs二级分区
	6.isFile : flume channel 使用的通道类型，分为file（true:不可丢）,memory（false:可丢）
	7.isJson : 半结构化数据（true:json），结构化数据（false:txt）
	8.opts :可选的配置方法
		WriteFileTime(t time.Duration)     // 设置写入日志文件的间隔时间   -- 默认5分钟
		FlashSliceDirTime(t time.Duration)  // 设置刷新分片目录间隔时间    -- 默认8分钟
		MaxFileCount(maxCount int)       // 分片目录下最大文件数量      -- 默认1000
		MaxLogCount(maxCount int)        // 单个文件,最大日志数量       -- 默认10000
		MoveTempFile()					   // 开启 移动临时文件协程      -- 默认关闭
		MoveTempFileTime(t time.Duration)  // 设置移动临时文件的时间       -- 默认8分钟
 */
wh :=flumefilewriter.NewWriteHandle(RootPath string, MuchInforFile string, tableName string, sendingMode SendMode,
selectorType SelectorType, isFile bool, isJson bool, opts ...DialOption)

// 例如创建一个服务名为tableName的日志写入,每10分钟写入一次文件,
// 每1000s刷新一次目录缓存,且检查临时目录
wh,err :=flumefilewriter.NewWriteHandle(
	"Data/flume","temp.file","tableName", flumefilewriter.Replicating, flumefilewriter.ES, true, true,
	flumefilewriter.FlashSliceDirTime(1000*time.Second),flumefilewriter.WriteFileTime(10*60*time.Second),MoveTempFile())


```

注意：

* 在linux环境下，使用flumeWrite写入，会在内存中占用大量cache（linux系统在文件读写时会写入内存缓存，导致“看上去”可用内存会减少）
* 若服务对内存敏感，请测试大批量日志写入后服务是否正常

> 另以act_log 请求日志为例：
> 1. 有区服act区分的请求使用：
> flumefilewriter.Replicating, flumefilewriter.ES 
> （Replicating 是复制流会发生到es和hdfs存储,使用这个配置后flumefilewriter.ES 这个配置相当于无效了）
> 2. 无区服act 请求使用
> flumefilewriter.Multiplexing, flumefilewriter.ES 
> （这个配置Multiplexing 是选择流 ，flumefilewriter.ES 是选择发送到es上，flumefilewriter.HDFS1 和flumefilewriter.HDFS2 分别是发送到HDFS一级分区和二级分区）	
  	


其他方法： wh.Close()  //调用此方法,不是立即关闭日志写入,而是立即将缓存中的日志数据写入到文件中,用于服务伸缩等情况.  

###### 非核心日志使用 flumefilewriter须知。 


 
* 行为日志（act_log）

| 字段名       | 类型     | 是否必填 | 最大长度  | 描述描述          | 示例值                                      |
| --------- | ------ | ---- | ----- | ------------- | ---------------------------------------- |
| act_id    | string | 是    | 32    | 接口标识          | 13604                                    |
| act_stat  | int    | 是    | int32 | 这个请求成功了传成功的状态码（成功的状态码传固定值1），失败了传失败的状态码（失败状态码可以自定义） | 0                                        |
| req       | string | 是    | text  | 请求内容          | {"traceid": "673c6222b00b4e148f58de86f50e198c", "ver": 1.3,"ip": "192.168.5.51"} |
| res       | string | 是    | 无长度限制 | 响应内容          |                                          |
| proc_time | int    | 是    | int32 | 处理时间（单位毫秒）    | 60                                       |
| user_id   | string | 是    | 32    | 角色标识          | iW53j3LbHjKvmkeeciSbcn                   |
| cp_app_id | int    | 是    | int32 | 内容提供商应用标识     | 1014                                     |
| log_time  | string  | 是   | 无长度限制  | 日志产生时间(转换为东八区时间进行存储，服务端自行转换) |"2019-01-01 00:00:00"  |
| reserve   | string | 是    | 无长度限制 | 保留字段      |          ""                                |

* 数值日志 (var_log)

| 字段名        | 类型     | 是否必填 | 最大长度  | 描述            | 示例值                    |
| ---------- | ------ | ---- | ----- | ------------- | ---------------------- |
| field_name | string | 是    | 32    | 重要数值字段名称（积分变化等）| RoleExp                |
| old_val    | int    | 是    | int64 | 原始值           | 200                    |
| new_val    | int    | 是    | int64 | 目标值           | 300                    |
| scene      | string | 是    | 32    | 数值变化场景        | 11403                  |
| remark     | string | 是    | 无长度限制 | 备注            | 领取成就奖励                 |
| user_id    | string | 是    | 32    | 角色标识          | iW53j3LbHjKvmkeeciSbcn |
| cp_app_id  | int    | 是    | int32 | 内容提供商应用标识     | 1014                   |
| log_time  | string  | 是    |   无长度限制  | 日志产生时间(转换为东八区时间进行存储，服务端自行转换) |"2019-01-01 00:00:00"  |
| reserve    | string | 是    | 无长度限制 | 保留字段      |        ""                |

* 写文件需要挂载nfs,命令如下

```
在本地创建/data1目录，并将/data1挂载到nfs集群。

以下是阿里云nfs测试地址,以及挂载命令
sudo mount -t nfs -o vers=4.0,noresvport 16413486e3-tjn76.cn-beijing.nas.aliyuncs.com: /data1/

卸载
umount /data1
```



**特殊情况**：

为了解决在线上环境中，由于读写文件导致内存缓存使用过高的情况，根据要求提出优化方案，在linux环境下，默认开启特殊操作，在操作文件时，使用`POSIX_FADV_DONTNEED`系统功能。

同时提供开关函数用于根据实际情况自定义。

相关测试报告在：`doc/gos线上集群cache过高问题分析-排查.pdf`

```go
// 开启 Fadvise 处理，默认处理。
flumefilewriter.EnableFadvise()

// 关闭 Fadvise 处理
flumefilewriter.DisableFadvise()
```
  
##### 使用syslog上传日志到kibana

除了`flumewriter`是把数据上传到`kibana`外，`syslog`的方式是目前使用较多的老方式。

注意使用`NewTcpSyslog2`时，会尝试从缓存中获取`Syslogwriter`对象，或者创建一个新的，并对该对象`引用计数+1`，
该逻辑是为了在更新`Syslogwriter`时，方便回收不用的资源。
在创建获取新的`syslogwriter`后，调用`ClearSyslogWriter`以便回收。

> 目前我们推荐使用`NewTcpSyslog2`来创建`syslog`的`writer`，新的`syslog`我们优化了逻辑，默认的还可以使用老版本的`NewTcpSyslog`。
> 
> 同时，`syslog`正在慢慢替换到`flumewriter`，该方案更稳定，也更推荐使用。

```go
// 这边 syslog 的 "INFO" 使用示例值即可，用于 syslog 创建对象使用，与log的 level 无关。
syslogger, err := writer.NewTcpSyslog2("127.0.0.1:514")

// 创建 log 配置
// 设置多个输出
cfg := log.NewProductionConfig(w,syslogger)

logger, err := log.New(cfg)

logger.Info("syslog info")
// Output: console
// {"level":"INFO","ts":"2019-01-01T09:12:34.483+08:00","msg":"syslog info"}
```

高级用法：参考 log_test 中的 `advanced configuration`测试用例，创建一个 logger ，并配置多种 encoder 编码器以及多种输出。

## 关于日志时区

默认日志都使用系统本地时区。

如果需要特殊时区，那么`Logger`以及`flumefilewriter`支持单独设置时区。

时区数据通过`time.LoadLocation`进行创建，具体时区配置参考：[List_of_tz_database_time_zones](https://en.wikipedia.org/wiki/List_of_tz_database_time_zones)

```go
Beijing, err : = time.LoadLocation("Asia/Shanghai")
```

Logger

`Logger`的时区设置将影响日志数据的`timeKey`对应的值。

```go
cfg := NewProductionConfig(os.Stdout)
// 通过 Location 设置创建的 Logger 的时区，这边设置为 北京时间
l, err := New(cfg,Location(Beijing))
```

flumefilewriter

`flumefilewriter`的时区设置将影响日志文件名上的日期。

```go
wh, err := flumefilewriter.NewWriteHandle(rootPath, tempPath, "session_log_test",
			Replicating, ES, true, true,
			FlashSliceDirTime(200*time.Millisecond),
			WriteFileTime(1*time.Second),
			MaxFileCount(fileCount),
			MaxLogCount(maxLogCount), // 最后日志缓存数量
			MoveTempFile(),
			MoveTempFileTime(1*time.Second),
			// 通过 Location 设置创建的 Logger 的时区，这边设置为 北京时间
			Location(Beijing))
```


## log额外字段数据

log通过传递各种 Field 进行记录额外数据。

常用 Field 说明:
其他类型可以直接引用 field 包下的字段，或者 zap 包下的字段

PS:特别注意，当使用Any或者其他方式记录相同字段不同数据类型时，会导致kibana无法检索相关日志信息，造成日志缺失的情况。

```go
// Any 可以加载各种常见数据
Any
// 记录 堆栈信息
Stack

// 常见输出字段类型，
String 
Bool
Int
Int64
Float64
Int32
Uint64

// Time 记录时间数据
Time
// 记录 byte 数据，并转换成 string
ByteString
// 记录 error 数据
Error

``` 

## 特殊Logger

log库另外实现了特殊log接口。

这个接口的性能比较弱。

```go
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
```

通过最上面的log示例可以方便转到到有特殊log函数的对象。
```go

// 创建默认 配置 
cfg := log.NewProductionConfig()
cfg.Name = "lognametest"

// 创建 日志对象
logger, err := log.New(cfg)

// 切换到 wlog

wlog := logger.Switch()

wlog.Info("Info log","infok","infov")
// Output:
// {"level":"INFO","ts":"2019-01-01T09:12:34.567+08:00","msg":"info log","infok":"infov"}


// 使用 with 操作来设置 额外字段数据
wlogWithField := wlog.With("extra_key","format type with key")
// 使用 format 操作
wlogWithField.Infof("info log:%s,%d","info_key",123)
// Output:
// {"level":"DEBUG","ts":"2019-01-01T09:12:34.567+08:00","msg":"info log:info_key,123","extra_key":"format type with key"}
        
```

## 日志记录出错处理

当出现异常时，会调用`utils.ErrorOutput`输出异常信息。

可以进行全局变量替换，修改成自己的错误处理函数。

### flumefilewriter

- 在写入文件数据异常时，会将临时数据写入到`/tmp/taotie.log/`目录中做备份。
