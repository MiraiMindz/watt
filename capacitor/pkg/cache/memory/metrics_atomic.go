package memory

import "sync/atomic"

// AtomicMetrics holds cache performance metrics using lock-free atomic operations.
// This provides zero-contention metrics updates at the cost of relaxed consistency.
//
// All fields use atomic operations for lock-free updates. Read operations may
// observe slightly stale values, but this is acceptable for metrics.
type AtomicMetrics struct {
	hits        atomic.Int64 // Number of cache hits
	misses      atomic.Int64 // Number of cache misses
	sets        atomic.Int64 // Number of Set operations
	deletes     atomic.Int64 // Number of Delete operations
	evictions   atomic.Int64 // Number of evictions
	expirations atomic.Int64 // Number of expirations
	currentSize atomic.Int64 // Current number of entries
}

// NewAtomicMetrics creates a new lock-free metrics instance.
func NewAtomicMetrics() *AtomicMetrics {
	return &AtomicMetrics{}
}

// RecordHit atomically increments the hit counter.
//
//go:inline
func (m *AtomicMetrics) RecordHit() {
	m.hits.Add(1)
}

// RecordMiss atomically increments the miss counter.
//
//go:inline
func (m *AtomicMetrics) RecordMiss() {
	m.misses.Add(1)
}

// RecordSet atomically increments the set counter.
//
//go:inline
func (m *AtomicMetrics) RecordSet() {
	m.sets.Add(1)
}

// RecordDelete atomically increments the delete counter.
//
//go:inline
func (m *AtomicMetrics) RecordDelete() {
	m.deletes.Add(1)
}

// RecordEviction atomically increments the eviction counter.
//
//go:inline
func (m *AtomicMetrics) RecordEviction() {
	m.evictions.Add(1)
}

// RecordExpiration atomically increments the expiration counter.
//
//go:inline
func (m *AtomicMetrics) RecordExpiration() {
	m.expirations.Add(1)
}

// IncrementSize atomically increments the current size.
//
//go:inline
func (m *AtomicMetrics) IncrementSize() {
	m.currentSize.Add(1)
}

// DecrementSize atomically decrements the current size.
//
//go:inline
func (m *AtomicMetrics) DecrementSize() {
	m.currentSize.Add(-1)
}

// AddExpirations atomically adds multiple expirations (for batch cleanup).
//
//go:inline
func (m *AtomicMetrics) AddExpirations(count int64) {
	m.expirations.Add(count)
	m.currentSize.Add(-count)
}

// Snapshot returns a point-in-time snapshot of all metrics.
// Values may not be perfectly consistent with each other due to concurrent updates.
func (m *AtomicMetrics) Snapshot() Metrics {
	return Metrics{
		Hits:        m.hits.Load(),
		Misses:      m.misses.Load(),
		Sets:        m.sets.Load(),
		Deletes:     m.deletes.Load(),
		Evictions:   m.evictions.Load(),
		Expirations: m.expirations.Load(),
		CurrentSize: m.currentSize.Load(),
	}
}

// HitRate returns the cache hit rate (0.0 to 1.0).
// This is a computed value based on current hit and miss counts.
func (m *AtomicMetrics) HitRate() float64 {
	hits := m.hits.Load()
	misses := m.misses.Load()
	total := hits + misses
	if total == 0 {
		return 0.0
	}
	return float64(hits) / float64(total)
}

// Reset atomically resets all metrics to zero.
// This is useful for benchmarking or resetting statistics.
func (m *AtomicMetrics) Reset() {
	m.hits.Store(0)
	m.misses.Store(0)
	m.sets.Store(0)
	m.deletes.Store(0)
	m.evictions.Store(0)
	m.expirations.Store(0)
	m.currentSize.Store(0)
}

// GetHits returns the current hit count.
func (m *AtomicMetrics) GetHits() int64 {
	return m.hits.Load()
}

// GetMisses returns the current miss count.
func (m *AtomicMetrics) GetMisses() int64 {
	return m.misses.Load()
}

// GetSets returns the current set count.
func (m *AtomicMetrics) GetSets() int64 {
	return m.sets.Load()
}

// GetDeletes returns the current delete count.
func (m *AtomicMetrics) GetDeletes() int64 {
	return m.deletes.Load()
}

// GetEvictions returns the current eviction count.
func (m *AtomicMetrics) GetEvictions() int64 {
	return m.evictions.Load()
}

// GetExpirations returns the current expiration count.
func (m *AtomicMetrics) GetExpirations() int64 {
	return m.expirations.Load()
}

// GetCurrentSize returns the current cache size.
func (m *AtomicMetrics) GetCurrentSize() int64 {
	return m.currentSize.Load()
}
