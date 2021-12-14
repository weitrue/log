package encoder

import (
	"fmt"
	"sync"

	"github.com/weitrue/log/bufferpool"
	"github.com/weitrue/log/config"
	"github.com/weitrue/log/entry"
	"github.com/weitrue/log/field"
	"go.uber.org/zap/buffer"
	"go.uber.org/zap/zapcore"
)

/*
根据 zapcore 中的 console_encoder 进行改造，主要调整处理 stack 部分

*/

const (
	ConsoleEncoding = "console"
)

var _sliceEncoderPool = sync.Pool{
	New: func() interface{} {
		return &SliceArrayEncoder{elems: make([]interface{}, 0, 2)}
	},
}

func getSliceEncoder() *SliceArrayEncoder {
	return _sliceEncoderPool.Get().(*SliceArrayEncoder)
}

func putSliceEncoder(e *SliceArrayEncoder) {
	e.elems = e.elems[:0]
	_sliceEncoderPool.Put(e)
}

type ConsoleEncoder struct {
	*JsonEncoder
}

// NewConsoleEncoder creates an encoder whose output is designed for human -
// rather than machine - consumption. It serializes the core log entry data
// (message, level, timestamp, etc.) in a plain-text format and leaves the
// structured context as JSON.
//
// Note that although the console encoder doesn't use the keys specified in the
// encoder configuration, it will omit any element whose key is set to the empty
// string.
func NewConsoleEncoder(cfg config.EncoderConfig) Encoder {
	return ConsoleEncoder{newJSONEncoder(cfg,false)}
}

func (c ConsoleEncoder) Clone() Encoder {
	return ConsoleEncoder{c.JsonEncoder.Clone().(*JsonEncoder)}

}

func (c ConsoleEncoder) EncodeEntry(ent entry.Entry, fields []field.Field) (*buffer.Buffer, error) {
	line := bufferpool.Get()

	// 我们不希望引用和转义 entry 的元数据（如果它被编码为字符串），这意味着我们不能使用 JSON 编码器。 最简单的选择是使用内存编码器和fmt.Fprint。
	// 如果这成为性能瓶颈，我们可以为我们的纯文本格式实现ArrayEncoder。

	// 下面操作将 Time 等数据存储到 slice encoder 中，然后通过 fmt.Fprint 直接输出
	arr := getSliceEncoder()
	if c.TimeKey != "" && c.EncodeTime != nil {
		c.EncodeTime(ent.Time, arr)
	}
	if c.LevelKey != "" && c.EncodeLevel != nil {
		c.EncodeLevel(ent.Level, arr)
	}
	if ent.LoggerName != "" && c.NameKey != "" {
		nameEncoder := c.EncodeName

		if nameEncoder == nil {
			// Fall back to FullNameEncoder for backward compatibility.
			nameEncoder = FullNameEncoder
		}

		nameEncoder(ent.LoggerName, arr)
	}
	if ent.Caller.Defined && c.CallerKey != "" && c.EncodeCaller != nil {
		c.EncodeCaller(ent.Caller, arr)
	}
	for i := range arr.elems {
		if i > 0 {
			line.AppendByte('\t')
		}
		fmt.Fprint(line, arr.elems[i])
	}
	putSliceEncoder(arr)

	// Add the message itself.
	if c.MessageKey != "" {
		c.addTabIfNecessary(line)
		line.AppendString(ent.Message)
	}

	// Add any structured context.
	c.writeContext(&ent,line, fields)

	// If there's no stacktrace key, honor that; this allows users to force
	// single-line output.
	if ent.Stack != "" && c.StacktraceKey != "" {
		line.AppendByte('\n')
		line.AppendString(ent.Stack)
	}

	if c.LineEnding != "" {
		line.AppendString(c.LineEnding)
	} else {
		line.AppendString(DefaultLineEnding)
	}
	return line, nil
}

func (c ConsoleEncoder) writeContext(ent *entry.Entry,line *buffer.Buffer, extra []field.Field) {
	context := c.JsonEncoder.Clone().(*JsonEncoder)
	defer context.Buf.Free()

	for _,f := range extra {
		// 处理 stack
		if f.Integer == field.STACK_TYPE_INT && f.Type == zapcore.StringType{
			ent.Stack = f.String
		}else{
			f.AddTo(context)
		}
	}
	context.closeOpenNamespaces()
	if context.Buf.Len() == 0 {
		return
	}

	c.addTabIfNecessary(line)
	line.AppendByte('{')
	line.Write(context.Buf.Bytes())
	line.AppendByte('}')
}

func (c ConsoleEncoder) addTabIfNecessary(line *buffer.Buffer) {
	if line.Len() > 0 {
		line.AppendByte('\t')
	}
}
