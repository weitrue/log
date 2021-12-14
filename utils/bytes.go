package utils

import (
	"os"
	"strconv"
	"sync"
)
const (
	BYTE_1   = 1
	BYTE_2   = 2
	BYTE_4   = 4
	BYTE_64  = 64
	BYTE_128 = 128
	BYTE_4K  = 4 * 1024
	BYTE_64K = 64 * 1024
	BYTE_1M  = 1 * 1024 * 1024
	BYTE_4M  = 4 * 1024 * 1024
	BYTE_16M = 16 * 1024 * 1024
)

// DefaultBytePool 默认的 byte 对象池
var DefaultBytePool *BytePool

func init() {
	defaultPoolFlag := os.Getenv("GO_ENABLE_DEFAULT_BYTE_POOL")
	ok, _ := strconv.ParseBool(defaultPoolFlag)
	if defaultPoolFlag == "" || ok {
		DefaultBytePool = NewBytePool(BYTE_1, BYTE_4M, 2)
	}
}

type BytePool struct {
	classes     []sync.Pool
	classesSize []int
	n           int
	minSize     int
	maxSize     int
}

// NewBytePool 创建一个byte的对象池
// minSize 最小byte长度
// maxSize 最大长度
// factor 对象池大小增长步进
func NewBytePool(minSize, maxSize, factor int) *BytePool {
	n := 0
	for chunkSize := minSize; chunkSize <= maxSize; chunkSize *= factor {
		n++
	}
	pool := &BytePool{
		make([]sync.Pool, n),
		make([]int, n),
		n,
		minSize, maxSize,
	}
	n = 0
	for chunkSize := minSize; chunkSize <= maxSize; chunkSize *= factor {
		pool.classesSize[n] = chunkSize
		pool.classes[n].New = func(size int) func() interface{} {
			return func() interface{} {
				buf := make([]byte, size)
				return &buf
			}
		}(chunkSize)
		n++
	}
	return pool
}

func (pool *BytePool) getIndex(size int) int {
	// Define f(-1) == false and f(n) == true.
	// Invariant: f(i-1) == false, f(j) == true.
	i, j := 0, pool.n
	for i < j {
		h := int(uint(i+j) >> 1) // avoid overflow when computing h
		// i ≤ h < j
		if !(pool.classesSize[h] >= size) {
			i = h + 1 // preserves f(i-1) == false
		} else {
			j = h // preserves f(j) == true
		}
	}
	// i == j, f(i-1) == false, and f(j) (= f(i)) == true  =>  answer is i.
	return i
}

// Get try alloc a []byte from internal slab class if no free chunk in slab class Alloc will make one.
func (pool *BytePool) Get(size int) []byte {
	if size <= 0 {
		return []byte{}
	}
	if size <= pool.maxSize {
		mem := pool.classes[pool.getIndex(size)].Get().(*[]byte)
		return (*mem)[:size]
	}
	return make([]byte, size)
}

// GetReference 获取引用类型的 []byte 对象(*[]byte)，为了避免 []byte 对象在使用时，出现栈数据移动到堆数据的现象，导致出现频繁gc的情况。
// *[]byte 没有重新设置 cap 属性，因此获取对象后，为了避免异常情况，需要手动设置
// buf := DefaultBytePool.GetReference(size)
// buf2 := (*buf)[:size]
func (pool *BytePool) GetReference(size int) *[]byte {
	if size <= 0 {
		return &[]byte{}
	}
	if size <= pool.maxSize {

		mem := pool.classes[pool.getIndex(size)].Get().(*[]byte)
		//b := (*mem)[:size]
		//return &b
		return mem
	}
	byteSlice := make([]byte, size)
	return &byteSlice
}

// Put release a []byte that alloc from Pool.Alloc.
func (pool *BytePool) Put(mem []byte) {
	// if size := cap(mem); size <= pool.maxSize {
	// 	for i := 0; i < len(pool.classesSize); i++ {
	// 		if pool.classesSize[i] >= size {
	// 			pool.classes[i].Put(&mem)
	// 			return
	// 		}
	// 	}
	// }
	if len(mem) == 0 {
		return
	}
	capL := cap(mem)
	i := pool.getIndex(capL)
	if capL < pool.classesSize[i] {
		i--
		if i < 0 {
			return
		}
	}
	pool.classes[i].Put(&mem)
}

// PutReference 回收 *[]byte 对象
func (pool *BytePool) PutReference(mem *[]byte) {

	if len(*mem) == 0 {
		return
	}
	capL := cap(*mem)
	i := pool.getIndex(capL)
	if capL < pool.classesSize[i] {
		i--
		if i < 0 {
			return
		}
	}
	pool.classes[i].Put(mem)
}
func (pool *BytePool) N() int {
	return pool.n
}

func (pool *BytePool) MinSize() int {
	return pool.minSize
}

func (pool *BytePool) MaxSize() int {
	return pool.maxSize
}
