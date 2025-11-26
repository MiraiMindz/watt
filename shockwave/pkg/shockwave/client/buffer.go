package client

import (
	"sync"
)

// SmallBufferPool manages small buffers (512B) for minimal allocations.
// This is separate from the global buffer pool to avoid over-allocating.
//
// Performance: 0 allocs/op on pool hit
type SmallBufferPool struct {
	pool512  sync.Pool
	pool1K   sync.Pool
	pool2K   sync.Pool
	pool4K   sync.Pool
}

var globalSmallBufferPool = &SmallBufferPool{
	pool512: sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 512)
			return &buf
		},
	},
	pool1K: sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 1024)
			return &buf
		},
	},
	pool2K: sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 2048)
			return &buf
		},
	},
	pool4K: sync.Pool{
		New: func() interface{} {
			buf := make([]byte, 4096)
			return &buf
		},
	},
}

// GetSmallBuffer returns a pooled buffer of the appropriate size.
//
// Allocation behavior: 0 allocs/op on pool hit
func GetSmallBuffer(size int) *[]byte {
	if size <= 512 {
		return globalSmallBufferPool.pool512.Get().(*[]byte)
	} else if size <= 1024 {
		return globalSmallBufferPool.pool1K.Get().(*[]byte)
	} else if size <= 2048 {
		return globalSmallBufferPool.pool2K.Get().(*[]byte)
	} else {
		return globalSmallBufferPool.pool4K.Get().(*[]byte)
	}
}

// PutSmallBuffer returns a buffer to the pool.
//
// Allocation behavior: 0 allocs/op
func PutSmallBuffer(buf *[]byte) {
	if buf == nil {
		return
	}

	// Reset length but keep capacity
	*buf = (*buf)[:0]

	// Return to appropriate pool based on capacity
	switch cap(*buf) {
	case 512:
		globalSmallBufferPool.pool512.Put(buf)
	case 1024:
		globalSmallBufferPool.pool1K.Put(buf)
	case 2048:
		globalSmallBufferPool.pool2K.Put(buf)
	case 4096:
		globalSmallBufferPool.pool4K.Put(buf)
	}
}

// ByteSlicePool provides efficient pooling for byte slices with precise size control.
type ByteSlicePool struct {
	pools map[int]*sync.Pool
	mu    sync.RWMutex
}

// NewByteSlicePool creates a new byte slice pool.
func NewByteSlicePool() *ByteSlicePool {
	return &ByteSlicePool{
		pools: make(map[int]*sync.Pool),
	}
}

// Get returns a byte slice of at least the requested size.
// The returned slice may be larger than requested.
//
// Allocation behavior: 0 allocs/op on pool hit
func (p *ByteSlicePool) Get(size int) []byte {
	// Round up to power of 2
	roundedSize := nextPowerOf2(size)

	p.mu.RLock()
	pool, exists := p.pools[roundedSize]
	p.mu.RUnlock()

	if !exists {
		p.mu.Lock()
		// Double-check after acquiring write lock
		pool, exists = p.pools[roundedSize]
		if !exists {
			pool = &sync.Pool{
				New: func() interface{} {
					buf := make([]byte, roundedSize)
					return &buf
				},
			}
			p.pools[roundedSize] = pool
		}
		p.mu.Unlock()
	}

	bufPtr := pool.Get().(*[]byte)
	return (*bufPtr)[:size]
}

// Put returns a byte slice to the pool.
//
// Allocation behavior: 0 allocs/op
func (p *ByteSlicePool) Put(buf []byte) {
	if len(buf) == 0 {
		return
	}

	size := cap(buf)
	roundedSize := nextPowerOf2(size)

	p.mu.RLock()
	pool, exists := p.pools[roundedSize]
	p.mu.RUnlock()

	if !exists {
		return // Don't pool if size doesn't match
	}

	// Reset slice
	buf = buf[:0]
	pool.Put(&buf)
}

// nextPowerOf2 returns the next power of 2 >= n.
func nextPowerOf2(n int) int {
	if n <= 0 {
		return 1
	}

	// Check if already power of 2
	if n&(n-1) == 0 {
		return n
	}

	// Find next power of 2
	power := 1
	for power < n {
		power <<= 1
	}
	return power
}

// ResetSlice resets a slice to zero length while preserving capacity.
// This is useful for reusing slices without allocating.
//
// Allocation behavior: 0 allocs/op
func ResetSlice(s []byte) []byte {
	return s[:0]
}

// GrowSlice grows a slice to at least the requested capacity.
// Returns the same slice if capacity is sufficient, otherwise allocates.
//
// Allocation behavior: 0 allocs if capacity sufficient, 1 alloc otherwise
func GrowSlice(s []byte, minCap int) []byte {
	if cap(s) >= minCap {
		return s[:minCap]
	}

	// Allocate new slice with next power of 2 capacity
	newCap := nextPowerOf2(minCap)
	newSlice := make([]byte, minCap, newCap)
	copy(newSlice, s)
	return newSlice
}
