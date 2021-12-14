package flumefilewriter

import (
	"bytes"
	"sync"
)

var(
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
