package encoder

import (
    "go.uber.org/zap/zapcore"
    "time"
)

// SliceArrayEncoder is an ArrayEncoder backed by a simple []interface{}. Like
// the MapObjectEncoder, it's not designed for production use.
type SliceArrayEncoder struct {
    elems []interface{}
}

func (s *SliceArrayEncoder) AppendArray(v zapcore.ArrayMarshaler) error {
    enc := &SliceArrayEncoder{}
    err := v.MarshalLogArray(enc)
    s.elems = append(s.elems, enc.elems)
    return err
}

func (s *SliceArrayEncoder) AppendObject(v zapcore.ObjectMarshaler) error {
    m := zapcore.NewMapObjectEncoder()
    err := v.MarshalLogObject(m)
    s.elems = append(s.elems, m.Fields)
    return err
}

func (s *SliceArrayEncoder) AppendReflected(v interface{}) error {
    s.elems = append(s.elems, v)
    return nil
}

func (s *SliceArrayEncoder) AppendBool(v bool)              { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendByteString(v []byte)      { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendComplex128(v complex128)  { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendComplex64(v complex64)    { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendDuration(v time.Duration) { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendFloat64(v float64)        { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendFloat32(v float32)        { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendInt(v int)                { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendInt64(v int64)            { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendInt32(v int32)            { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendInt16(v int16)            { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendInt8(v int8)              { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendString(v string)          { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendTime(v time.Time)         { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendUint(v uint)              { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendUint64(v uint64)          { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendUint32(v uint32)          { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendUint16(v uint16)          { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendUint8(v uint8)            { s.elems = append(s.elems, v) }
func (s *SliceArrayEncoder) AppendUintptr(v uintptr)        { s.elems = append(s.elems, v) }

