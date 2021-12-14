package writer

import (
    "go.uber.org/multierr"
    "go.uber.org/zap/zapcore"
    "io"
)

type WriteSyncer = zapcore.WriteSyncer
// sync 需要忽略错误
// https://github.com/uber-go/zap/issues/328
var AddSync = zapcore.AddSync
var Lock = zapcore.Lock



//var NewMultiWriteSyncer = zapcore.NewMultiWriteSyncer


type multiWriteSyncer []WriteSyncer

// NewMultiWriteSyncer creates a WriteSyncer that duplicates its writes
// and sync calls, much like io.MultiWriter.
func NewMultiWriteSyncer(ws ...WriteSyncer) WriteSyncer {
    if len(ws) == 1 {
        return ws[0]
    }
    // Copy to protect against https://github.com/golang/go/issues/7809
    return multiWriteSyncer(append([]WriteSyncer(nil), ws...))
}

// See https://golang.org/src/io/multi.go
// When not all underlying syncers write the same number of bytes,
// the smallest number is returned even though Write() is called on
// all of them.
func (ws multiWriteSyncer) Write(p []byte) (n int, err error) {
    for _, w := range ws {
        n, err = w.Write(p)

        if err != nil {

            return
        }
        if n < len(p) {
            err = io.ErrShortWrite
            return
        }
    }
    return len(p), nil
}

func (ws multiWriteSyncer) Sync() error {
    var err error
    for _, w := range ws {
        err = multierr.Append(err, w.Sync())
    }
    return err
}

// A WriteSyncer is an io.Writer that can also flush any buffered data. Note
// that *os.File (and thus, os.Stderr and os.Stdout) implement WriteSyncer.
type WriteCloseSyncer interface {
    io.Writer
    Sync() error
    io.Closer
}