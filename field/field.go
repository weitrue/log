package field

import (
    "github.com/weitrue/log/stacktrace"
    "go.uber.org/zap"
    "go.uber.org/zap/zapcore"
)

// Field is an alias for Field. Aliasing this type dramatically
// improves the navigability of this package's API documentation.
type Field = zap.Field


// Field 设置 ==============================================================

// Skip constructs a no-op field, which is often useful when handling invalid
// inputs in other Field constructors.
var Skip = zap.Skip

// Binary constructs a field that carries an opaque binary blob.
//
// Binary data is serialized in an encoding-appropriate format. For example,
// zap's JSON encoder base64-encodes binary blobs. To log UTF-8 encoded text,
// use ByteString.
var Binary = zap.Binary

// Bool constructs a field that carries a bool.
var Bool = zap.Bool

// ByteString constructs a field that carries UTF-8 encoded text as a []byte.
// To log opaque binary blobs (which aren't necessarily valid UTF-8), use
// Binary.
var ByteString = zap.ByteString

// Complex128 constructs a field that carries a complex number. Unlike most
// numeric fields, this costs an allocation (to convert the complex128 to
// interface{}).
var Complex128 = zap.Complex128

// Complex64 constructs a field that carries a complex number. Unlike most
// numeric fields, this costs an allocation (to convert the complex64 to
// interface{}).
var Complex64 = zap.Complex64

// Float64 constructs a field that carries a float64. The way the
// floating-point value is represented is encoder-dependent, so marshaling is
// necessarily lazy.
var Float64 = zap.Float64

// Float32 constructs a field that carries a float32. The way the
// floating-point value is represented is encoder-dependent, so marshaling is
// necessarily lazy.
var Float32 = zap.Float32

// Int constructs a field with the given key and value.
var Int = zap.Int

// Int64 constructs a field with the given key and value.
var Int64 = zap.Int64

// Int32 constructs a field with the given key and value.
var Int32 = zap.Int32

// Int16 constructs a field with the given key and value.
var Int16 = zap.Int16

// Int8 constructs a field with the given key and value.
var Int8 = zap.Int8

// String constructs a field with the given key and value.
var String = zap.String

// Uint constructs a field with the given key and value.
var Uint = zap.Uint

// Uint64 constructs a field with the given key and value.
var Uint64 = zap.Uint64

// Uint32 constructs a field with the given key and value.
var Uint32 = zap.Uint32

// Uint16 constructs a field with the given key and value.
var Uint16 = zap.Uint16

// Uint8 constructs a field with the given key and value.

var Uint8 = zap.Uint8

// Uintptr constructs a field with the given key and value.
var Uintptr = zap.Uintptr

// Reflect constructs a field with the given key and an arbitrary object. It uses
// an encoding-appropriate, reflection-based function to lazily serialize nearly
// any object into the logging context, but it's relatively slow and
// allocation-heavy. Outside tests, Any is always a better choice.
//
// If encoding fails (e.g., trying to serialize a map[int]string to JSON), Reflect
// includes the error message in the final log output.
var Reflect = zap.Reflect

// Namespace creates a named, isolated scope within the log's context. All
// subsequent fields will be added to the new namespace.
//
// This helps prevent key collisions when injecting loggers into sub-components
// or third-party libraries.

var Namespace = zap.Namespace

// Stringer constructs a field with the given key and the output of the value's
// String method. The Stringer's String method is called lazily.
var Stringer = zap.Stringer

// Time constructs a Field with the given key and value. The encoder
// controls how the time is serialized.
var Time = zap.Time

// Duration constructs a field with the given key and value. The encoder
// controls how the duration is serialized.

var Duration = zap.Duration

// Object constructs a field with the given key and ObjectMarshaler. It
// provides a flexible, but still type-safe and efficient, way to add map- or
// struct-like user-defined types to the logging context. The struct's
// MarshalLogObject method is called lazily.

var Object = zap.Object

// Any takes a key and an arbitrary value and chooses the best way to represent
// them as a field, falling back to a reflection-based approach only if
// necessary.
//
// Since byte/uint8 and rune/int32 are aliases, Any can't differentiate between
// them. To minimize surprises, []byte values are treated as binary blobs, byte
// values are treated as uint8, and runes are always treated as integers.
// PS:特别注意，当使用Any或者其他方式记录相同字段不同数据类型时，会导致kibana无法检索相关日志信息，造成日志缺失的情况。
var Any = zap.Any

var Array = zap.Array
var Error = zap.Error
var NamedError = zap.NamedError

// Stack 获取堆栈信息
func Stack(key string)Field {
    // 将 堆栈信息 作为字符串返回会导致 alloc 内存分配，
    // 但是我们无法扩展  Field 联合结构体 来包含 []byte 字段。
    // 由于获取 堆栈跟踪 消耗非常大（~10us），所以额外的 alloc 是可以接受的。
    return Field{Key: key, Type: zapcore.StringType, Integer:STACK_TYPE_INT,String: stacktrace.TakeStacktraceSkip(1,0)}
}

// StackSkip 获取堆栈信息
// skip 跳过多少个栈尾 数据，一般根据需要，设置 3 左右
// duplicateFrameNum 当使用递归或这链式调用等场景时，设置这个参数来过滤重复栈，
// 比如为1时，表示如果当前栈与上一个栈一致，则当前栈不输出；为2 时，表示当前栈与上上个栈一致时，当前栈不输出
func StackSkip(key string,skip,duplicateFrameSkip int)Field {
    // 将 堆栈信息 作为字符串返回会导致 alloc 内存分配，
    // 但是我们无法扩展  Field 联合结构体 来包含 []byte 字段。
    // 由于获取 堆栈跟踪 消耗非常大（~10us），所以额外的 alloc 是可以接受的。
    return Field{Key: key, Type: zapcore.StringType, Integer:STACK_TYPE_INT,String: stacktrace.TakeStacktraceSkip(skip+1,duplicateFrameSkip)}
}
const (
    // STACK_TYPE_INT 该字段用于设置到 Stack field 对应的 integer 字段数据中，用于在 encoder 中识别该field是 stack，以便进一步处理。
    STACK_TYPE_INT = -101
)