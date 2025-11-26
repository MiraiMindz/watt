//go:build !goexperiment.arenas
// +build !goexperiment.arenas

package memory

// Stub implementations for when arena support is not available

// Arena stub (from arena.go)
type Arena struct{}

// ArenaPool stub (from arena.go)
type ArenaPool struct{}

// RequestArena stub (from arena.go)
type RequestArena struct {
	arena *Arena
}

// NewArenaPool stub
func NewArenaPool() *ArenaPool {
	return nil
}

// Get stub
func (p *ArenaPool) Get() *Arena {
	return nil
}

// Put stub
func (p *ArenaPool) Put(a *Arena) {}

// NewRequestArena stub
func NewRequestArena(pool *ArenaPool) *RequestArena {
	return nil
}

// Free stub
func (ra *RequestArena) Free(pool *ArenaPool) {}

// ArenaRequestPool stub
type ArenaRequestPool struct{}

// ArenaPoolStats stub
type ArenaPoolStats struct{}

// NewArenaRequestPool returns nil when arenas aren't available
func NewArenaRequestPool() *ArenaRequestPool {
	return nil
}

// GetRequestArena returns nil
func (p *ArenaRequestPool) GetRequestArena() *RequestArena {
	return nil
}

// PutRequestArena is a no-op
func (p *ArenaRequestPool) PutRequestArena(ra *RequestArena) {}

// GetStats returns empty stats
func (p *ArenaRequestPool) GetStats() ArenaPoolStats {
	return ArenaPoolStats{}
}

// HitRate returns 0
func (s *ArenaPoolStats) HitRate() float64 {
	return 0
}

// GlobalArenaPool is nil when arenas aren't available
var GlobalArenaPool *ArenaRequestPool = nil

// BatchAllocator stub
type BatchAllocator struct{}

// NewBatchAllocator returns nil
func NewBatchAllocator(arena *Arena, batchSize int) *BatchAllocator {
	return nil
}

// Allocate returns nil
func (ba *BatchAllocator) Allocate(n int) []byte {
	return nil
}

// Reset is a no-op
func (ba *BatchAllocator) Reset() {}

// ArenaHTTPRequest stub
type ArenaHTTPRequest struct{}

// NewArenaHTTPRequest returns nil
func NewArenaHTTPRequest(pool *ArenaPool) *ArenaHTTPRequest {
	return nil
}

// SetMethod is a no-op
func (r *ArenaHTTPRequest) SetMethod(method []byte) {}

// SetPath is a no-op
func (r *ArenaHTTPRequest) SetPath(path []byte) {}

// SetProto is a no-op
func (r *ArenaHTTPRequest) SetProto(proto []byte) {}

// AddHeader is a no-op
func (r *ArenaHTTPRequest) AddHeader(key, value []byte) {}

// SetBody is a no-op
func (r *ArenaHTTPRequest) SetBody(body []byte) {}

// Free is a no-op
func (r *ArenaHTTPRequest) Free(pool *ArenaPool) {}

// ArenaHTTPResponse stub
type ArenaHTTPResponse struct{}

// NewArenaHTTPResponse returns nil
func NewArenaHTTPResponse(pool *ArenaPool) *ArenaHTTPResponse {
	return nil
}

// SetStatusLine is a no-op
func (r *ArenaHTTPResponse) SetStatusLine(status []byte) {}

// AddHeader is a no-op
func (r *ArenaHTTPResponse) AddHeader(key, value []byte) {}

// SetBody is a no-op
func (r *ArenaHTTPResponse) SetBody(body []byte) {}

// Free is a no-op
func (r *ArenaHTTPResponse) Free(pool *ArenaPool) {}

// ArenaConnectionPool stub
type ArenaConnectionPool struct{}

// NewArenaConnectionPool returns nil
func NewArenaConnectionPool() *ArenaConnectionPool {
	return nil
}

// Get returns nil
func (p *ArenaConnectionPool) Get() *ArenaPool {
	return nil
}

// Put is a no-op
func (p *ArenaConnectionPool) Put(pool *ArenaPool) {}
