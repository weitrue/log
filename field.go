package log

import (
    "github.com/weitrue/log/field"
)

// Any 输出任一类型数据
// 特别注意，当使用Any或者其他方式记录相同字段不同数据类型时，会导致kibana无法检索相关日志信息，造成日志缺失的情况。
var Any = field.Any
var Stack = field.Stack
var StackSkip = field.StackSkip


// 常见输出字段类型，其他类型可以直接引用 field 包下的字段，或者 zap 包下的字段
var String = field.String
var Bool = field.Bool
var Int = field.Int
var Int64 = field.Int64
var Float64 = field.Float64
var Int32 = field.Int32
var Uint64 = field.Uint64


var Time = field.Time
var ByteString = field.ByteString


