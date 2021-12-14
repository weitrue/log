package syslog

import (
	"bytes"
	"strings"
	"sync"
	"time"
)

var (
	_pool = NewPool()
)

const _size = 1024 // by default, create 1 KiB buffers

// A Pool is a type-safe wrapper around a sync.Pool.
type Pool struct {
	p *sync.Pool
}

// NewPool constructs a new Pool.
func NewPool() Pool {
	return Pool{p: &sync.Pool{
		New: func() interface{} {

			return bytes.NewBuffer(make([]byte, 0, _size))
		},
	}}
}

// Get retrieves a Buffer from the pool, creating one if necessary.
func (p Pool) Get() *bytes.Buffer {
	buf := p.p.Get().(*bytes.Buffer)
	buf.Reset()
	return buf
}

func (p Pool) Put(buf *bytes.Buffer) {
	p.p.Put(buf)
}

var strBufPool = sync.Pool{
	New: func() interface{} {
		return &strings.Builder{}
	},
}

func GetStrBuf() *strings.Builder {
	cache := strBufPool.Get().(*strings.Builder)
	return cache
}

func PutStrBuf(cache *strings.Builder) {
	cache.Reset()
	strBufPool.Put(cache)
}

var timerPool = sync.Pool{
	New: func() interface{} {
		return time.NewTimer(time.Second)
	},
}

func GetTimer() *time.Timer {
	cache := timerPool.Get().(*time.Timer)
	return cache
}

func PutTimer(cache *time.Timer) {
	cache.Stop()
	timerPool.Put(cache)
}

var bytePool = sync.Pool{
	New: func() interface{} {
		bs := make([]byte, 0)
		return &bs
	},
}

func GetByte() *[]byte {
	return bytePool.Get().(*[]byte)
}

func PutByte(bs *[]byte) {
	*bs = (*bs)[:0]
	bytePool.Put(bs)
}
