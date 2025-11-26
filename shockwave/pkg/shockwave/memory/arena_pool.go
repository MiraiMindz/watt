//go:build goexperiment.arenas
// +build goexperiment.arenas

package memory

import (
	"sync"
	"sync/atomic"
)

// ArenaRequestPool provides arena-based pooling integrated with HTTP request handling.
// This replaces sync.Pool with arena allocation for zero GC pressure.
//
// Build with: GOEXPERIMENT=arenas go build -tags arenas
type ArenaRequestPool struct {
	arenaPool *ArenaPool

	// Statistics
	stats ArenaPoolStats
}

// ArenaPoolStats tracks arena pool usage
type ArenaPoolStats struct {
	TotalRequests    atomic.Uint64
	TotalFrees       atomic.Uint64
	CurrentInUse     atomic.Int64
	BytesAllocated   atomic.Uint64
	ArenasCreated    atomic.Uint64
	ArenasReused     atomic.Uint64
}

// NewArenaRequestPool creates a new arena-based request pool
func NewArenaRequestPool() *ArenaRequestPool {
	return &ArenaRequestPool{
		arenaPool: NewArenaPool(),
	}
}

// GetRequestArena acquires an arena for a new HTTP request
func (p *ArenaRequestPool) GetRequestArena() *RequestArena {
	ra := NewRequestArena(p.arenaPool)

	// Update stats
	p.stats.TotalRequests.Add(1)
	p.stats.CurrentInUse.Add(1)

	// Check if arena was reused
	if ra.arena.arena != nil {
		p.stats.ArenasReused.Add(1)
	} else {
		p.stats.ArenasCreated.Add(1)
	}

	return ra
}

// PutRequestArena returns an arena to the pool
func (p *ArenaRequestPool) PutRequestArena(ra *RequestArena) {
	if ra == nil {
		return
	}

	// Update stats
	p.stats.TotalFrees.Add(1)
	p.stats.CurrentInUse.Add(-1)

	// Return arena to pool
	ra.Free(p.arenaPool)
}

// GetStats returns current pool statistics
func (p *ArenaRequestPool) GetStats() ArenaPoolStats {
	return p.stats
}

// HitRate returns the arena reuse rate (percentage)
func (s *ArenaPoolStats) HitRate() float64 {
	total := s.ArenasCreated.Load() + s.ArenasReused.Load()
	if total == 0 {
		return 0
	}
	return float64(s.ArenasReused.Load()) / float64(total) * 100.0
}

// GlobalArenaPool is the global arena pool instance
var GlobalArenaPool = NewArenaRequestPool()

// BatchAllocator provides batch allocation from arena for better cache locality
type BatchAllocator struct {
	arena *Arena

	// Current batch position
	offset int

	// Batch buffer (allocated in arena)
	batch []byte

	// Batch size (default 64KB)
	batchSize int
}

// NewBatchAllocator creates a batch allocator from an arena
func NewBatchAllocator(arena *Arena, batchSize int) *BatchAllocator {
	if batchSize == 0 {
		batchSize = 64 * 1024 // 64KB default
	}

	return &BatchAllocator{
		arena:     arena,
		batchSize: batchSize,
		offset:    0,
	}
}

// Allocate allocates n bytes from the current batch
// Falls back to arena allocation for large requests
func (ba *BatchAllocator) Allocate(n int) []byte {
	// If request is larger than batch size, allocate directly
	if n > ba.batchSize {
		return ba.arena.MakeSlice(n)
	}

	// Check if current batch has enough space
	if ba.batch == nil || ba.offset+n > len(ba.batch) {
		// Allocate new batch
		ba.batch = ba.arena.MakeSlice(ba.batchSize)
		ba.offset = 0
	}

	// Allocate from batch
	result := ba.batch[ba.offset : ba.offset+n]
	ba.offset += n

	return result
}

// Reset resets the batch allocator for reuse
func (ba *BatchAllocator) Reset() {
	ba.batch = nil
	ba.offset = 0
}

// ArenaHTTPRequest represents an HTTP request allocated entirely in an arena
type ArenaHTTPRequest struct {
	arena *Arena
	batch *BatchAllocator

	// Request components (all arena-allocated)
	Method []byte
	Path   []byte
	Proto  []byte

	// Headers (arena-allocated slices)
	HeaderKeys   [][]byte
	HeaderValues [][]byte

	// Body (arena-allocated)
	Body []byte

	// Metadata
	ContentLength int64
	Close         bool
}

// NewArenaHTTPRequest creates a new arena-allocated HTTP request
func NewArenaHTTPRequest(pool *ArenaPool) *ArenaHTTPRequest {
	arena := pool.Get()

	return &ArenaHTTPRequest{
		arena: arena,
		batch: NewBatchAllocator(arena, 64*1024), // 64KB batch
	}
}

// SetMethod sets the request method (arena-allocated)
func (r *ArenaHTTPRequest) SetMethod(method []byte) {
	r.Method = r.batch.Allocate(len(method))
	copy(r.Method, method)
}

// SetPath sets the request path (arena-allocated)
func (r *ArenaHTTPRequest) SetPath(path []byte) {
	r.Path = r.batch.Allocate(len(path))
	copy(r.Path, path)
}

// SetProto sets the protocol (arena-allocated)
func (r *ArenaHTTPRequest) SetProto(proto []byte) {
	r.Proto = r.batch.Allocate(len(proto))
	copy(r.Proto, proto)
}

// AddHeader adds a header (arena-allocated)
func (r *ArenaHTTPRequest) AddHeader(key, value []byte) {
	// Allocate key
	keyBuf := r.batch.Allocate(len(key))
	copy(keyBuf, key)

	// Allocate value
	valueBuf := r.batch.Allocate(len(value))
	copy(valueBuf, value)

	// Append to slices (slices themselves are not arena-allocated)
	r.HeaderKeys = append(r.HeaderKeys, keyBuf)
	r.HeaderValues = append(r.HeaderValues, valueBuf)
}

// SetBody sets the body (arena-allocated)
func (r *ArenaHTTPRequest) SetBody(body []byte) {
	if len(body) == 0 {
		r.Body = nil
		return
	}

	r.Body = r.batch.Allocate(len(body))
	copy(r.Body, body)
}

// Free returns the arena to the pool
func (r *ArenaHTTPRequest) Free(pool *ArenaPool) {
	pool.Put(r.arena)
}

// ArenaHTTPResponse represents an HTTP response allocated entirely in an arena
type ArenaHTTPResponse struct {
	arena *Arena
	batch *BatchAllocator

	// Response components
	StatusLine []byte

	// Headers
	HeaderKeys   [][]byte
	HeaderValues [][]byte

	// Body
	Body []byte
}

// NewArenaHTTPResponse creates a new arena-allocated HTTP response
func NewArenaHTTPResponse(pool *ArenaPool) *ArenaHTTPResponse {
	arena := pool.Get()

	return &ArenaHTTPResponse{
		arena: arena,
		batch: NewBatchAllocator(arena, 64*1024),
	}
}

// SetStatusLine sets the status line (arena-allocated)
func (r *ArenaHTTPResponse) SetStatusLine(status []byte) {
	r.StatusLine = r.batch.Allocate(len(status))
	copy(r.StatusLine, status)
}

// AddHeader adds a response header (arena-allocated)
func (r *ArenaHTTPResponse) AddHeader(key, value []byte) {
	keyBuf := r.batch.Allocate(len(key))
	copy(keyBuf, key)

	valueBuf := r.batch.Allocate(len(value))
	copy(valueBuf, value)

	r.HeaderKeys = append(r.HeaderKeys, keyBuf)
	r.HeaderValues = append(r.HeaderValues, valueBuf)
}

// SetBody sets the response body (arena-allocated)
func (r *ArenaHTTPResponse) SetBody(body []byte) {
	if len(body) == 0 {
		r.Body = nil
		return
	}

	r.Body = r.batch.Allocate(len(body))
	copy(r.Body, body)
}

// Free returns the arena to the pool
func (r *ArenaHTTPResponse) Free(pool *ArenaPool) {
	pool.Put(r.arena)
}

// ArenaConnectionPool manages per-connection arena allocations
// This is useful for HTTP/2 and WebSocket where multiple requests share a connection
type ArenaConnectionPool struct {
	pool sync.Pool
}

// NewArenaConnectionPool creates a new connection-scoped arena pool
func NewArenaConnectionPool() *ArenaConnectionPool {
	return &ArenaConnectionPool{
		pool: sync.Pool{
			New: func() interface{} {
				return NewArenaPool()
			},
		},
	}
}

// Get returns a per-connection arena pool
func (p *ArenaConnectionPool) Get() *ArenaPool {
	return p.pool.Get().(*ArenaPool)
}

// Put returns a connection arena pool
func (p *ArenaConnectionPool) Put(pool *ArenaPool) {
	// Note: We don't free arenas here, they're freed individually
	p.pool.Put(pool)
}
