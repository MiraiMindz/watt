// Package memory provides advanced memory management strategies for Shockwave.
//
// Three allocation modes are available:
// 1. Standard - sync.Pool based (default)
// 2. Green Tea GC - Spatial/temporal locality optimization (build tag: greenteagc)
// 3. Arena - Zero GC pressure (build tag: arenas, GOEXPERIMENT=arenas)
package memory

import (
	"sync"
	"sync/atomic"
)

// GreenTeaAllocator provides batch allocation with spatial locality optimization.
// Objects allocated together are stored adjacently in memory for better cache performance.
type GreenTeaAllocator struct {
	// Current slab
	slab []byte

	// Offset into current slab
	offset int

	// Slab size (default 256KB)
	slabSize int

	// Slab pool for reuse
	slabPool *sync.Pool

	// Statistics
	slabsAllocated atomic.Uint64
	bytesAllocated atomic.Uint64
}

// NewGreenTeaAllocator creates a new Green Tea allocator
func NewGreenTeaAllocator(slabSize int) *GreenTeaAllocator {
	if slabSize == 0 {
		slabSize = 256 * 1024 // 256KB default
	}

	return &GreenTeaAllocator{
		slabSize: slabSize,
		slabPool: &sync.Pool{
			New: func() interface{} {
				buf := make([]byte, slabSize)
				return &buf
			},
		},
	}
}

// Allocate allocates n bytes with spatial locality
func (gta *GreenTeaAllocator) Allocate(n int) []byte {
	// For large allocations, use direct allocation
	if n > gta.slabSize/2 {
		buf := make([]byte, n)
		gta.bytesAllocated.Add(uint64(n))
		return buf
	}

	// Check if current slab has enough space
	if gta.slab == nil || gta.offset+n > len(gta.slab) {
		// Get new slab from pool
		slabPtr := gta.slabPool.Get().(*[]byte)
		gta.slab = *slabPtr
		gta.offset = 0
		gta.slabsAllocated.Add(1)

		// Verify slab is valid
		if len(gta.slab) == 0 {
			// Pool returned empty slice, create new one
			gta.slab = make([]byte, gta.slabSize)
		}
	}

	// Allocate from slab (actual zero-copy slice)
	result := gta.slab[gta.offset : gta.offset+n]
	gta.offset += n
	gta.bytesAllocated.Add(uint64(n))

	return result
}

// Reset resets the allocator and returns slabs to pool
func (gta *GreenTeaAllocator) Reset() {
	if gta.slab != nil {
		// Need to put a copy of the slice pointer to avoid issues
		slab := gta.slab
		gta.slabPool.Put(&slab)
		gta.slab = nil
	}
	gta.offset = 0
}

// GetStats returns allocation statistics
func (gta *GreenTeaAllocator) GetStats() (slabsAllocated, bytesAllocated uint64) {
	return gta.slabsAllocated.Load(), gta.bytesAllocated.Load()
}

// GreenTeaHTTPRequest represents an HTTP request using Green Tea GC
type GreenTeaHTTPRequest struct {
	allocator *GreenTeaAllocator

	// Request components (allocated from slab)
	Method []byte
	Path   []byte
	Proto  []byte

	// Headers
	HeaderKeys   [][]byte
	HeaderValues [][]byte

	// Body
	Body []byte

	// Metadata
	ContentLength int64
	Close         bool
}

// NewGreenTeaHTTPRequest creates a new Green Tea GC request
func NewGreenTeaHTTPRequest() *GreenTeaHTTPRequest {
	return &GreenTeaHTTPRequest{
		allocator: NewGreenTeaAllocator(256 * 1024), // 256KB slab
	}
}

// SetMethod sets the request method (slab-allocated)
func (r *GreenTeaHTTPRequest) SetMethod(method []byte) {
	r.Method = r.allocator.Allocate(len(method))
	copy(r.Method, method)
}

// SetPath sets the request path (slab-allocated)
func (r *GreenTeaHTTPRequest) SetPath(path []byte) {
	r.Path = r.allocator.Allocate(len(path))
	copy(r.Path, path)
}

// SetProto sets the protocol (slab-allocated)
func (r *GreenTeaHTTPRequest) SetProto(proto []byte) {
	r.Proto = r.allocator.Allocate(len(proto))
	copy(r.Proto, proto)
}

// AddHeader adds a header (slab-allocated)
func (r *GreenTeaHTTPRequest) AddHeader(key, value []byte) {
	keyBuf := r.allocator.Allocate(len(key))
	copy(keyBuf, key)

	valueBuf := r.allocator.Allocate(len(value))
	copy(valueBuf, value)

	r.HeaderKeys = append(r.HeaderKeys, keyBuf)
	r.HeaderValues = append(r.HeaderValues, valueBuf)
}

// SetBody sets the body (slab-allocated)
func (r *GreenTeaHTTPRequest) SetBody(body []byte) {
	if len(body) == 0 {
		r.Body = nil
		return
	}

	r.Body = r.allocator.Allocate(len(body))
	copy(r.Body, body)
}

// Free resets and returns slabs to pool
func (r *GreenTeaHTTPRequest) Free() {
	r.allocator.Reset()
}

// GreenTeaHTTPResponse represents an HTTP response using Green Tea GC
type GreenTeaHTTPResponse struct {
	allocator *GreenTeaAllocator

	// Response components
	StatusLine []byte

	// Headers
	HeaderKeys   [][]byte
	HeaderValues [][]byte

	// Body
	Body []byte
}

// NewGreenTeaHTTPResponse creates a new Green Tea GC response
func NewGreenTeaHTTPResponse() *GreenTeaHTTPResponse {
	return &GreenTeaHTTPResponse{
		allocator: NewGreenTeaAllocator(256 * 1024),
	}
}

// SetStatusLine sets the status line (slab-allocated)
func (r *GreenTeaHTTPResponse) SetStatusLine(status []byte) {
	r.StatusLine = r.allocator.Allocate(len(status))
	copy(r.StatusLine, status)
}

// AddHeader adds a response header (slab-allocated)
func (r *GreenTeaHTTPResponse) AddHeader(key, value []byte) {
	keyBuf := r.allocator.Allocate(len(key))
	copy(keyBuf, key)

	valueBuf := r.allocator.Allocate(len(value))
	copy(valueBuf, value)

	r.HeaderKeys = append(r.HeaderKeys, keyBuf)
	r.HeaderValues = append(r.HeaderValues, valueBuf)
}

// SetBody sets the response body (slab-allocated)
func (r *GreenTeaHTTPResponse) SetBody(body []byte) {
	if len(body) == 0 {
		r.Body = nil
		return
	}

	r.Body = r.allocator.Allocate(len(body))
	copy(r.Body, body)
}

// Free resets and returns slabs to pool
func (r *GreenTeaHTTPResponse) Free() {
	r.allocator.Reset()
}

// GreenTeaRequestPool provides pooled Green Tea allocators
type GreenTeaRequestPool struct {
	pool sync.Pool

	// Statistics
	stats GreenTeaRequestPoolStats
}

// GreenTeaRequestPoolStats tracks pool statistics
type GreenTeaRequestPoolStats struct {
	TotalRequests atomic.Uint64
	TotalFrees    atomic.Uint64
	CurrentInUse  atomic.Int64
	PoolHits      atomic.Uint64
	PoolMisses    atomic.Uint64
}

// NewGreenTeaRequestPool creates a new Green Tea request pool
func NewGreenTeaRequestPool() *GreenTeaRequestPool {
	return &GreenTeaRequestPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewGreenTeaHTTPRequest()
			},
		},
	}
}

// GetRequest acquires a Green Tea request from the pool
func (p *GreenTeaRequestPool) GetRequest() *GreenTeaHTTPRequest {
	req := p.pool.Get().(*GreenTeaHTTPRequest)

	// Update stats
	p.stats.TotalRequests.Add(1)
	p.stats.CurrentInUse.Add(1)

	// Track reuse
	slabs, _ := req.allocator.GetStats()
	if slabs > 0 {
		p.stats.PoolHits.Add(1)
	} else {
		p.stats.PoolMisses.Add(1)
	}

	return req
}

// PutRequest returns a Green Tea request to the pool
func (p *GreenTeaRequestPool) PutRequest(req *GreenTeaHTTPRequest) {
	if req == nil {
		return
	}

	// Reset allocator (returns slabs to pool)
	req.Free()

	// Clear slices
	req.HeaderKeys = nil
	req.HeaderValues = nil
	req.Body = nil

	// Update stats
	p.stats.TotalFrees.Add(1)
	p.stats.CurrentInUse.Add(-1)

	// Return to pool
	p.pool.Put(req)
}

// GetStats returns current pool statistics
func (p *GreenTeaRequestPool) GetStats() GreenTeaRequestPoolStats {
	return p.stats
}

// HitRate returns the pool hit rate (percentage)
func (s *GreenTeaRequestPoolStats) HitRate() float64 {
	total := s.PoolHits.Load() + s.PoolMisses.Load()
	if total == 0 {
		return 0
	}
	return float64(s.PoolHits.Load()) / float64(total) * 100.0
}

// GlobalGreenTeaRequestPool is the global Green Tea request pool
var GlobalGreenTeaRequestPool = NewGreenTeaRequestPool()
