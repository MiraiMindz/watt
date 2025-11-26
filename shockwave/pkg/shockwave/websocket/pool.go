package websocket

import (
	"sync"
)

// Buffer pools for common message sizes to reduce allocations.
// Provides pre-allocated buffers for 256B, 1KB, 4KB, and 16KB messages.

var (
	// Pool for 256-byte buffers
	pool256 = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 256)
			return &b
		},
	}

	// Pool for 1KB buffers
	pool1K = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 1024)
			return &b
		},
	}

	// Pool for 4KB buffers
	pool4K = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 4096)
			return &b
		},
	}

	// Pool for 16KB buffers
	pool16K = sync.Pool{
		New: func() interface{} {
			b := make([]byte, 16384)
			return &b
		},
	}

	// Pool for frame header buffers (14 bytes)
	headerPool = sync.Pool{
		New: func() interface{} {
			b := make([]byte, MaxFrameHeaderSize)
			return &b
		},
	}
)

// getBuffer returns a buffer from the appropriate pool for the given size.
// Returns nil if size is too large (>16KB) - caller should allocate directly.
func getBuffer(size int) *[]byte {
	switch {
	case size <= 256:
		return pool256.Get().(*[]byte)
	case size <= 1024:
		return pool1K.Get().(*[]byte)
	case size <= 4096:
		return pool4K.Get().(*[]byte)
	case size <= 16384:
		return pool16K.Get().(*[]byte)
	default:
		return nil // Too large, allocate directly
	}
}

// putBuffer returns a buffer to the appropriate pool.
func putBuffer(buf *[]byte) {
	if buf == nil {
		return
	}

	size := cap(*buf)
	switch {
	case size == 256:
		pool256.Put(buf)
	case size == 1024:
		pool1K.Put(buf)
	case size == 4096:
		pool4K.Put(buf)
	case size == 16384:
		pool16K.Put(buf)
	// Don't pool other sizes
	}
}

// getHeaderBuffer returns a pooled header buffer.
func getHeaderBuffer() *[]byte {
	return headerPool.Get().(*[]byte)
}

// putHeaderBuffer returns a header buffer to the pool.
func putHeaderBuffer(buf *[]byte) {
	if buf != nil {
		headerPool.Put(buf)
	}
}

// BufferPool provides a high-level interface for getting/putting buffers.
type BufferPool struct {
	// Enable/disable pooling globally (useful for testing)
	disabled bool
}

// DefaultBufferPool is the global buffer pool instance.
var DefaultBufferPool = &BufferPool{}

// Get returns a buffer of at least the given size.
// The returned buffer may be larger than requested.
// The caller must call Put() when done to return the buffer to the pool.
func (p *BufferPool) Get(size int) []byte {
	if p.disabled {
		return make([]byte, size)
	}

	buf := getBuffer(size)
	if buf == nil {
		// Size too large, allocate directly
		return make([]byte, size)
	}

	// Return a slice of the requested size
	return (*buf)[:size]
}

// Put returns a buffer to the pool.
// The buffer should not be used after calling Put.
func (p *BufferPool) Put(buf []byte) {
	if p.disabled || len(buf) == 0 {
		return
	}

	// Get the full capacity buffer
	fullBuf := buf[:cap(buf)]
	putBuffer(&fullBuf)
}

// GetExact returns a buffer of exactly the pooled size for the given size.
// Returns (buffer, true) if from pool, (nil, false) if too large.
func (p *BufferPool) GetExact(size int) ([]byte, bool) {
	if p.disabled {
		return nil, false
	}

	buf := getBuffer(size)
	if buf == nil {
		return nil, false
	}
	return *buf, true
}
