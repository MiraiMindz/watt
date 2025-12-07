package memory

import (
	"sync"
	"time"
)

// entry represents a cache entry with metadata.
// Entries are pooled using sync.Pool to minimize allocations.
type entry[K comparable, V any] struct {
	key       K
	value     V
	expiresAt time.Time
	lruNode   *lruNode[K]
}

// isExpired checks if the entry has expired.
func (e *entry[K, V]) isExpired() bool {
	if e.expiresAt.IsZero() {
		return false // No expiration
	}
	return time.Now().After(e.expiresAt)
}

// reset clears the entry for reuse.
func (e *entry[K, V]) reset() {
	var zeroK K
	var zeroV V
	e.key = zeroK
	e.value = zeroV
	e.expiresAt = time.Time{}
	e.lruNode = nil
}

// entryPool manages a pool of cache entries.
// Uses sync.Pool for zero-allocation Get operations.
type entryPool[K comparable, V any] struct {
	pool *sync.Pool
}

// newEntryPool creates a new entry pool.
func newEntryPool[K comparable, V any]() *entryPool[K, V] {
	return &entryPool[K, V]{
		pool: &sync.Pool{
			New: func() interface{} {
				return &entry[K, V]{}
			},
		},
	}
}

// get retrieves an entry from the pool.
// The entry is zeroed and ready for use.
func (p *entryPool[K, V]) get() *entry[K, V] {
	e := p.pool.Get().(*entry[K, V])
	e.reset() // Ensure clean state
	return e
}

// put returns an entry to the pool.
// The entry is reset before being returned.
func (p *entryPool[K, V]) put(e *entry[K, V]) {
	e.reset()
	p.pool.Put(e)
}
