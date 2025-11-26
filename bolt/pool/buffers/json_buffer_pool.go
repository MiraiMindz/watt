package buffers

import (
	"bytes"
	"sync"
)

// Three-tier JSON buffer pooling for optimal memory reuse
// Automatically selects appropriate buffer size based on payload

var (
	// Small buffers: 512B (for small API responses)
	smallJSONPool = sync.Pool{
		New: func() interface{} {
			buf := bytes.NewBuffer(make([]byte, 0, 512))
			return buf
		},
	}

	// Medium buffers: 8KB (for typical API responses)
	mediumJSONPool = sync.Pool{
		New: func() interface{} {
			buf := bytes.NewBuffer(make([]byte, 0, 8192))
			return buf
		},
	}

	// Large buffers: 64KB (for large JSON payloads)
	largeJSONPool = sync.Pool{
		New: func() interface{} {
			buf := bytes.NewBuffer(make([]byte, 0, 65536))
			return buf
		},
	}
)

const (
	// Size thresholds for automatic tier selection
	smallBufferThreshold  = 512
	mediumBufferThreshold = 8192
)

// AcquireJSONBuffer acquires a buffer from the appropriate pool based on expected size.
//
// Size hints:
//   - 0 or unknown: Returns medium buffer (8KB) - good default
//   - <512B: Returns small buffer (512B)
//   - 512B-8KB: Returns medium buffer (8KB)
//   - >8KB: Returns large buffer (64KB)
//
// Performance: 0 allocs/op (buffer reuse from pool)
func AcquireJSONBuffer(sizeHint int) *bytes.Buffer {
	if sizeHint == 0 || sizeHint > smallBufferThreshold && sizeHint <= mediumBufferThreshold {
		// Default to medium for unknown sizes
		return mediumJSONPool.Get().(*bytes.Buffer)
	}

	if sizeHint <= smallBufferThreshold {
		return smallJSONPool.Get().(*bytes.Buffer)
	}

	// Large payloads
	return largeJSONPool.Get().(*bytes.Buffer)
}

// ReleaseJSONBuffer returns a buffer to the appropriate pool after resetting it.
//
// The buffer's capacity determines which pool it goes back to:
//   - ≤512B → small pool
//   - ≤8KB → medium pool
//   - >8KB → large pool
//
// Performance: 0 allocs/op (pool reuse)
func ReleaseJSONBuffer(buf *bytes.Buffer) {
	if buf == nil {
		return
	}

	// Reset buffer (keep capacity, clear length)
	buf.Reset()

	// Return to appropriate pool based on capacity
	cap := buf.Cap()

	if cap <= smallBufferThreshold {
		smallJSONPool.Put(buf)
	} else if cap <= mediumBufferThreshold {
		mediumJSONPool.Put(buf)
	} else {
		largeJSONPool.Put(buf)
	}
}

// AcquireSmallJSONBuffer explicitly acquires a small buffer (512B).
// Use when you know the response will be small (e.g., {"ok": true}).
func AcquireSmallJSONBuffer() *bytes.Buffer {
	return smallJSONPool.Get().(*bytes.Buffer)
}

// AcquireMediumJSONBuffer explicitly acquires a medium buffer (8KB).
// Use for typical API responses (user objects, small lists).
func AcquireMediumJSONBuffer() *bytes.Buffer {
	return mediumJSONPool.Get().(*bytes.Buffer)
}

// AcquireLargeJSONBuffer explicitly acquires a large buffer (64KB).
// Use for large payloads (long lists, nested objects, pagination).
func AcquireLargeJSONBuffer() *bytes.Buffer {
	return largeJSONPool.Get().(*bytes.Buffer)
}
