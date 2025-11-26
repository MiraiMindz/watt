//go:build goexperiment.arenas

package memory

import (
	"arena"
	"sync"
)

// Arena provides arena-based memory allocation for HTTP requests
// This eliminates GC pressure by allocating all request-related memory
// in an arena that is freed all at once when the request completes.
//
// Build with: GOEXPERIMENT=arenas go build -tags arenas
type Arena struct {
	arena *arena.Arena
}

// ArenaPool manages a pool of arenas for reuse
type ArenaPool struct {
	pool sync.Pool
}

// NewArenaPool creates a new arena pool
func NewArenaPool() *ArenaPool {
	return &ArenaPool{
		pool: sync.Pool{
			New: func() interface{} {
				return &Arena{
					arena: arena.NewArena(),
				}
			},
		},
	}
}

// Get returns an arena from the pool
func (p *ArenaPool) Get() *Arena {
	return p.pool.Get().(*Arena)
}

// Put returns an arena to the pool
func (p *ArenaPool) Put(a *Arena) {
	// Free arena memory
	a.arena.Free()

	// Create new arena for reuse
	a.arena = arena.NewArena()

	p.pool.Put(a)
}

// MakeSlice allocates a byte slice in the arena
func (a *Arena) MakeSlice(size int) []byte {
	return arena.MakeSlice[byte](a.arena, size, size)
}

// MakeString allocates a string in the arena
func (a *Arena) MakeString(s string) string {
	// Allocate byte slice in arena
	b := arena.MakeSlice[byte](a.arena, len(s), len(s))
	copy(b, s)
	return string(b)
}

// Clone clones a byte slice into the arena
func (a *Arena) Clone(src []byte) []byte {
	dst := arena.MakeSlice[byte](a.arena, len(src), len(src))
	copy(dst, src)
	return dst
}

// Free explicitly frees the arena
func (a *Arena) Free() {
	a.arena.Free()
}

// RequestArena holds arena allocations for a single HTTP request
type RequestArena struct {
	arena *Arena

	// Request data allocated in arena
	Method string
	Path   string
	Proto  string
	Body   []byte

	// Headers allocated in arena
	HeaderKeys   []string
	HeaderValues []string
}

// NewRequestArena creates a new request arena
func NewRequestArena(pool *ArenaPool) *RequestArena {
	return &RequestArena{
		arena: pool.Get(),
	}
}

// Free returns the arena to the pool
func (ra *RequestArena) Free(pool *ArenaPool) {
	pool.Put(ra.arena)
}

// AllocateMethod allocates the method string in the arena
func (ra *RequestArena) AllocateMethod(method string) {
	ra.Method = ra.arena.MakeString(method)
}

// AllocatePath allocates the path string in the arena
func (ra *RequestArena) AllocatePath(path string) {
	ra.Path = ra.arena.MakeString(path)
}

// AllocateProto allocates the protocol string in the arena
func (ra *RequestArena) AllocateProto(proto string) {
	ra.Proto = ra.arena.MakeString(proto)
}

// AllocateBody allocates the body in the arena
func (ra *RequestArena) AllocateBody(body []byte) {
	ra.Body = ra.arena.Clone(body)
}

// AllocateHeader allocates a header key-value pair in the arena
func (ra *RequestArena) AllocateHeader(key, value string) {
	ra.HeaderKeys = append(ra.HeaderKeys, ra.arena.MakeString(key))
	ra.HeaderValues = append(ra.HeaderValues, ra.arena.MakeString(value))
}

// Stats provides arena allocation statistics
type Stats struct {
	// Total arenas created
	TotalArenas uint64

	// Total arenas freed
	TotalFreed uint64

	// Current arenas in use
	InUse uint64

	// Total bytes allocated
	BytesAllocated uint64
}

// GetStats returns arena statistics (placeholder for now)
func (p *ArenaPool) GetStats() Stats {
	// TODO: Implement statistics tracking
	return Stats{}
}
