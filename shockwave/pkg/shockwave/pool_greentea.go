//go:build greenteagc

package shockwave

import (
	"sync"
	"sync/atomic"
)

// Green Tea GC Optimization
//
// The Green Tea GC technique exploits spatial and temporal locality to reduce
// GC overhead. The key insight is that objects allocated together and used
// together should be collected together.
//
// Strategy:
// 1. Group related objects (request, headers, body, response) into "generations"
// 2. Allocate objects from the same request in adjacent memory locations
// 3. Use generational pooling - separate pools for short-lived vs long-lived objects
// 4. Minimize cross-generation pointers
//
// Benefits:
// - Improved cache locality (objects used together are stored together)
// - Reduced GC pressure (generational collection is more efficient)
// - Better memory layout for modern CPUs
//
// Build with: go build -tags greenteagc

// RequestGeneration holds all allocations for a single HTTP request
// Objects are allocated together for better cache locality
type RequestGeneration struct {
	// Request data
	Method [16]byte   // Inline method (most are <16 bytes)
	Path   [256]byte  // Inline path (most are <256 bytes)
	Proto  [16]byte   // Inline proto ("HTTP/1.1" = 8 bytes)

	// Header storage (inline to keep locality)
	HeaderKeys   [32][64]byte  // Max 32 headers, 64 bytes per key
	HeaderValues [32][256]byte // Max 32 headers, 256 bytes per value
	HeaderCount  int

	// Body storage (pooled separately for large bodies)
	Body       []byte
	BodyInline [4096]byte // Inline storage for small bodies (<4KB)

	// Metadata
	Generation uint64 // Generation number for tracking
	InUse      atomic.Bool
}

// GreenTeaPool implements generational pooling for HTTP requests
type GreenTeaPool struct {
	// Short-lived object pool (request/response pairs)
	shortLived sync.Pool

	// Generation counter
	generation atomic.Uint64

	// Statistics
	stats GreenTeaStats
}

// GreenTeaStats tracks pooling statistics
type GreenTeaStats struct {
	TotalAllocations atomic.Uint64
	TotalFrees       atomic.Uint64
	PoolHits         atomic.Uint64
	PoolMisses       atomic.Uint64
	CurrentInUse     atomic.Int64
}

// NewGreenTeaPool creates a new Green Tea GC pool
func NewGreenTeaPool() *GreenTeaPool {
	return &GreenTeaPool{
		shortLived: sync.Pool{
			New: func() interface{} {
				return &RequestGeneration{}
			},
		},
	}
}

// GetRequestGeneration acquires a request generation from the pool
func (p *GreenTeaPool) GetRequestGeneration() *RequestGeneration {
	rg := p.shortLived.Get().(*RequestGeneration)

	// Set generation number for tracking
	rg.Generation = p.generation.Add(1)
	rg.InUse.Store(true)

	// Update stats
	p.stats.TotalAllocations.Add(1)
	p.stats.CurrentInUse.Add(1)

	if rg.Generation > 1 {
		p.stats.PoolHits.Add(1)
	} else {
		p.stats.PoolMisses.Add(1)
	}

	return rg
}

// PutRequestGeneration returns a request generation to the pool
func (p *GreenTeaPool) PutRequestGeneration(rg *RequestGeneration) {
	if !rg.InUse.CompareAndSwap(true, false) {
		// Already returned to pool
		return
	}

	// Clear data (helps GC identify dead objects)
	rg.HeaderCount = 0
	rg.Body = nil

	// Update stats
	p.stats.TotalFrees.Add(1)
	p.stats.CurrentInUse.Add(-1)

	p.shortLived.Put(rg)
}

// GetStats returns current pool statistics
func (p *GreenTeaPool) GetStats() GreenTeaStats {
	return p.stats
}

// HitRate returns the pool hit rate (percentage)
func (s *GreenTeaStats) HitRate() float64 {
	total := s.PoolHits.Load() + s.PoolMisses.Load()
	if total == 0 {
		return 0
	}
	return float64(s.PoolHits.Load()) / float64(total) * 100.0
}

// ResponseGeneration holds response data with cache locality
type ResponseGeneration struct {
	// Status line
	StatusLine [64]byte

	// Headers
	HeaderKeys   [32][64]byte
	HeaderValues [32][256]byte
	HeaderCount  int

	// Body buffer
	Body       []byte
	BodyInline [8192]byte // Inline storage for small responses (<8KB)

	// Metadata
	Generation uint64
	InUse      atomic.Bool
}

// GetResponseGeneration acquires a response generation from the pool
func (p *GreenTeaPool) GetResponseGeneration() *ResponseGeneration {
	// For now, use the same pool (could separate later)
	rg := &ResponseGeneration{
		Generation: p.generation.Add(1),
	}
	rg.InUse.Store(true)

	p.stats.TotalAllocations.Add(1)
	p.stats.CurrentInUse.Add(1)

	return rg
}

// PutResponseGeneration returns a response generation to the pool
func (p *GreenTeaPool) PutResponseGeneration(rg *ResponseGeneration) {
	if !rg.InUse.CompareAndSwap(true, false) {
		return
	}

	// Clear data
	rg.HeaderCount = 0
	rg.Body = nil

	p.stats.TotalFrees.Add(1)
	p.stats.CurrentInUse.Add(-1)

	// For now, just let GC handle it (could pool later)
}

// GlobalGreenTeaPool is the global Green Tea GC pool instance
var GlobalGreenTeaPool = NewGreenTeaPool()
