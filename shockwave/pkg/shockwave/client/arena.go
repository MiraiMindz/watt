// +build arenas

package client

import (
	"arena"
	"sync"
)

// ArenaPool manages arenas for request-scoped allocations.
// This enables bulk allocation and bulk freeing for minimal GC pressure.
//
// Build with: GOEXPERIMENT=arenas go build -tags arenas
//
// Performance: Can reduce allocations to near-zero by allocating everything
// in the same arena and freeing it all at once when the request is done.
type ArenaPool struct {
	pool sync.Pool
}

var globalArenaPool = &ArenaPool{
	pool: sync.Pool{
		New: func() interface{} {
			return arena.NewArena()
		},
	},
}

// GetArena returns a pooled arena.
//
// Allocation behavior: 0 allocs/op on pool hit
func GetArena() *arena.Arena {
	return globalArenaPool.pool.Get().(*arena.Arena)
}

// PutArena returns an arena to the pool after freeing all allocations.
//
// Allocation behavior: 0 allocs/op
func PutArena(a *arena.Arena) {
	if a != nil {
		a.Free()
		globalArenaPool.pool.Put(a)
	}
}

// ArenaRequest wraps a ClientRequest with arena allocations.
type ArenaRequest struct {
	*ClientRequest
	arena *arena.Arena
}

// NewArenaRequest creates a request using arena allocation.
//
// Allocation behavior: All request allocations happen in arena (bulk freed later)
func NewArenaRequest() *ArenaRequest {
	a := GetArena()
	req := arena.New[ClientRequest](a)
	return &ArenaRequest{
		ClientRequest: req,
		arena:         a,
	}
}

// Free returns the arena to the pool.
func (ar *ArenaRequest) Free() {
	PutArena(ar.arena)
}

// ArenaResponse wraps a ClientResponse with arena allocations.
type ArenaResponse struct {
	*ClientResponse
	arena *arena.Arena
}

// NewArenaResponse creates a response using arena allocation.
//
// Allocation behavior: All response allocations happen in arena (bulk freed later)
func NewArenaResponse() *ArenaResponse {
	a := GetArena()
	resp := arena.New[ClientResponse](a)
	return &ArenaResponse{
		ClientResponse: resp,
		arena:          a,
	}
}

// Free returns the arena to the pool.
func (ar *ArenaResponse) Free() {
	PutArena(ar.arena)
}

// ArenaHeaders wraps ClientHeaders with arena allocations.
type ArenaHeaders struct {
	*ClientHeaders
	arena *arena.Arena
}

// NewArenaHeaders creates headers using arena allocation.
//
// Allocation behavior: All header allocations happen in arena (bulk freed later)
func NewArenaHeaders() *ArenaHeaders {
	a := GetArena()
	headers := arena.New[ClientHeaders](a)
	return &ArenaHeaders{
		ClientHeaders: headers,
		arena:         a,
	}
}

// Free returns the arena to the pool.
func (ah *ArenaHeaders) Free() {
	PutArena(ah.arena)
}
