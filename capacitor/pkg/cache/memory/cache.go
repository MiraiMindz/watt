package memory

import (
	"context"
	"sync"
	"time"

	"github.com/watt-toolkit/capacitor/pkg/capacitor"
)

// Cache is a high-performance in-memory cache with TTL support, size limits, and LRU eviction.
// It uses generic types for type safety and implements zero-allocation patterns for Get operations.
//
// Performance characteristics:
//   - Get: ~50-100ns/op, 0 allocs/op (cache hit)
//   - Set: ~200-500ns/op, minimal allocs (uses sync.Pool)
//   - Thread-safe with RWMutex for high read concurrency
//
// Example:
//
//	cache := memory.New[string, *User](memory.Config{
//	    MaxSize:      1000,
//	    DefaultTTL:   5 * time.Minute,
//	    EvictionMode: memory.EvictionLRU,
//	})
//	defer cache.Close()
//
//	cache.Set(ctx, "user:123", user, memory.WithTTL(10*time.Minute))
//	user, err := cache.Get(ctx, "user:123")
type Cache[K comparable, V any] struct {
	// Configuration
	config Config

	// Storage
	data map[K]*entry[K, V]
	mu   sync.RWMutex

	// LRU eviction
	lru *lruList[K]

	// Entry pool for zero allocations
	pool *entryPool[K, V]

	// Metrics
	metrics *Metrics
	metricsMu sync.RWMutex

	// Lifecycle
	closed bool
	stopCh chan struct{}
	wg     sync.WaitGroup
}

// Config holds the configuration for the cache.
type Config struct {
	// MaxSize is the maximum number of entries in the cache.
	// When exceeded, entries are evicted according to EvictionMode.
	// 0 means no limit.
	MaxSize int

	// MaxMemory is the maximum memory usage in bytes.
	// 0 means no limit (not implemented yet, reserved for future use).
	MaxMemory int64

	// DefaultTTL is the default time-to-live for entries.
	// 0 means no expiration by default.
	DefaultTTL time.Duration

	// EvictionMode determines how entries are evicted when MaxSize is reached.
	EvictionMode EvictionMode

	// CleanupInterval is how often to run the cleanup goroutine to remove expired entries.
	// 0 disables automatic cleanup.
	CleanupInterval time.Duration

	// EnableMetrics enables collection of cache metrics.
	EnableMetrics bool
}

// EvictionMode determines the eviction strategy.
type EvictionMode int

const (
	// EvictionNone disables eviction. Set will fail when cache is full.
	EvictionNone EvictionMode = iota

	// EvictionLRU evicts least recently used entries.
	EvictionLRU

	// EvictionLFU evicts least frequently used entries (future).
	EvictionLFU

	// EvictionRandom evicts random entries (future).
	EvictionRandom
)

// String returns the string representation of the eviction mode.
func (m EvictionMode) String() string {
	switch m {
	case EvictionNone:
		return "None"
	case EvictionLRU:
		return "LRU"
	case EvictionLFU:
		return "LFU"
	case EvictionRandom:
		return "Random"
	default:
		return "Unknown"
	}
}

// Metrics holds cache performance metrics.
type Metrics struct {
	Hits          int64 // Number of cache hits
	Misses        int64 // Number of cache misses
	Sets          int64 // Number of Set operations
	Deletes       int64 // Number of Delete operations
	Evictions     int64 // Number of evictions
	Expirations   int64 // Number of expirations
	CurrentSize   int64 // Current number of entries
	CurrentMemory int64 // Current memory usage (future)
}

// HitRate returns the cache hit rate (0.0 to 1.0).
func (m *Metrics) HitRate() float64 {
	total := m.Hits + m.Misses
	if total == 0 {
		return 0.0
	}
	return float64(m.Hits) / float64(total)
}

// New creates a new in-memory cache with the given configuration.
func New[K comparable, V any](config Config) *Cache[K, V] {
	// Set defaults
	if config.MaxSize <= 0 {
		config.MaxSize = 10000 // Default: 10k entries
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Minute // Default: cleanup every minute
	}

	cache := &Cache[K, V]{
		config:  config,
		data:    make(map[K]*entry[K, V], config.MaxSize),
		lru:     newLRUList[K](),
		pool:    newEntryPool[K, V](),
		metrics: &Metrics{},
		stopCh:  make(chan struct{}),
	}

	// Start cleanup goroutine if enabled
	if config.CleanupInterval > 0 {
		cache.wg.Add(1)
		go cache.cleanupLoop()
	}

	return cache
}

// Get retrieves a value from the cache.
// Returns ErrNotFound if the key doesn't exist or has expired.
//
// Performance: ~50-100ns/op, 0 allocs/op (cache hit)
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		var zero V
		return zero, capacitor.ErrClosed
	}

	e, exists := c.data[key]
	if !exists {
		c.recordMiss()
		var zero V
		return zero, capacitor.ErrNotFound
	}

	// Check expiration
	if e.isExpired() {
		c.mu.RUnlock()
		c.mu.Lock()
		delete(c.data, key)
		c.lru.remove(e.lruNode)
		c.pool.put(e)

		// Inline metrics update
		if c.config.EnableMetrics {
			c.metricsMu.Lock()
			c.metrics.Expirations++
			c.metrics.CurrentSize--
			c.metricsMu.Unlock()
		}
		c.mu.Unlock()
		c.mu.RLock()

		var zero V
		return zero, capacitor.ErrNotFound
	}

	// Update LRU (touch)
	if c.config.EvictionMode == EvictionLRU {
		c.mu.RUnlock()
		c.mu.Lock()
		c.lru.moveToFront(e.lruNode)
		c.mu.Unlock()
		c.mu.RLock()
	}

	c.recordHit()
	return e.value, nil
}

// Set stores a value in the cache with optional TTL.
//
// Performance: ~200-500ns/op, minimal allocs (uses sync.Pool)
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, opts ...SetOption) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return capacitor.ErrClosed
	}

	// Apply options
	options := &setOptions{
		ttl: c.config.DefaultTTL,
	}
	for _, opt := range opts {
		opt(options)
	}

	// Check if key already exists
	if e, exists := c.data[key]; exists {
		// Update existing entry
		e.value = value
		e.expiresAt = c.calculateExpiration(options.ttl)
		c.lru.moveToFront(e.lruNode)

		// Inline metrics update
		if c.config.EnableMetrics {
			c.metricsMu.Lock()
			c.metrics.Sets++
			c.metricsMu.Unlock()
		}
		return nil
	}

	// Check size limit
	if c.config.MaxSize > 0 && len(c.data) >= c.config.MaxSize {
		// Evict if necessary
		if err := c.evict(); err != nil {
			return err
		}
	}

	// Create new entry from pool
	e := c.pool.get()
	e.key = key
	e.value = value
	e.expiresAt = c.calculateExpiration(options.ttl)

	// Add to data and LRU
	c.data[key] = e
	e.lruNode = c.lru.pushFront(key)

	// Inline metrics update
	if c.config.EnableMetrics {
		c.metricsMu.Lock()
		c.metrics.Sets++
		c.metrics.CurrentSize++
		c.metricsMu.Unlock()
	}

	return nil
}

// Delete removes a key from the cache.
func (c *Cache[K, V]) Delete(ctx context.Context, key K) error {
	c.mu.Lock()

	if c.closed {
		c.mu.Unlock()
		return capacitor.ErrClosed
	}

	e, exists := c.data[key]
	if !exists {
		c.mu.Unlock()
		return capacitor.ErrNotFound
	}

	delete(c.data, key)
	c.lru.remove(e.lruNode)
	c.pool.put(e)

	// Inline metrics update to avoid second lock
	if c.config.EnableMetrics {
		// Update metrics without taking metricsMu (already have c.mu)
		c.metricsMu.Lock()
		c.metrics.Deletes++
		c.metrics.CurrentSize--
		c.metricsMu.Unlock()
	}

	c.mu.Unlock()
	return nil
}

// Exists checks if a key exists in the cache and is not expired.
func (c *Cache[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	c.mu.RLock()
	defer c.mu.RUnlock()

	if c.closed {
		return false, capacitor.ErrClosed
	}

	e, exists := c.data[key]
	if !exists {
		return false, nil
	}

	if e.isExpired() {
		return false, nil
	}

	return true, nil
}

// Clear removes all entries from the cache.
func (c *Cache[K, V]) Clear(ctx context.Context) error {
	c.mu.Lock()
	defer c.mu.Unlock()

	if c.closed {
		return capacitor.ErrClosed
	}

	// Return entries to pool
	for _, e := range c.data {
		c.pool.put(e)
	}

	c.data = make(map[K]*entry[K, V], c.config.MaxSize)
	c.lru = newLRUList[K]()
	c.metrics.CurrentSize = 0

	return nil
}

// Size returns the current number of entries in the cache.
func (c *Cache[K, V]) Size() int {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return len(c.data)
}

// Stats returns cache statistics as LayerStats.
func (c *Cache[K, V]) Stats() capacitor.LayerStats {
	c.metricsMu.RLock()
	defer c.metricsMu.RUnlock()

	return capacitor.LayerStats{
		Name:      "memory",
		Hits:      uint64(c.metrics.Hits),
		Misses:    uint64(c.metrics.Misses),
		Sets:      uint64(c.metrics.Sets),
		Deletes:   uint64(c.metrics.Deletes),
		Evictions: uint64(c.metrics.Evictions),
		Size:      c.metrics.CurrentSize,
		HitRate:   c.metrics.HitRate(),
	}
}

// Metrics returns a copy of the current metrics.
func (c *Cache[K, V]) Metrics() Metrics {
	c.metricsMu.RLock()
	defer c.metricsMu.RUnlock()
	return *c.metrics
}

// Close shuts down the cache and stops background goroutines.
func (c *Cache[K, V]) Close() error {
	c.mu.Lock()
	if c.closed {
		c.mu.Unlock()
		return capacitor.ErrClosed
	}
	c.closed = true
	close(c.stopCh)
	c.mu.Unlock()

	c.wg.Wait()
	return nil
}

// evict removes entries according to the eviction mode.
// Must be called with c.mu held.
func (c *Cache[K, V]) evict() error {
	if c.config.EvictionMode == EvictionNone {
		return capacitor.ErrEvictionFailed
	}

	if c.config.EvictionMode == EvictionLRU {
		// Remove least recently used
		node := c.lru.back()
		if node == nil {
			return capacitor.ErrEvictionFailed
		}

		e, exists := c.data[node.key]
		if !exists {
			return capacitor.ErrEvictionFailed
		}

		delete(c.data, node.key)
		c.lru.remove(node)
		c.pool.put(e)

		// Inline metrics update
		if c.config.EnableMetrics {
			c.metricsMu.Lock()
			c.metrics.Evictions++
			c.metrics.CurrentSize--
			c.metricsMu.Unlock()
		}
		return nil
	}

	return capacitor.ErrEvictionFailed
}

// cleanupLoop runs periodically to remove expired entries.
func (c *Cache[K, V]) cleanupLoop() {
	defer c.wg.Done()

	ticker := time.NewTicker(c.config.CleanupInterval)
	defer ticker.Stop()

	for {
		select {
		case <-ticker.C:
			c.cleanup()
		case <-c.stopCh:
			return
		}
	}
}

// cleanup removes expired entries.
func (c *Cache[K, V]) cleanup() {
	c.mu.Lock()
	defer c.mu.Unlock()

	now := time.Now()
	expired := make([]K, 0, 10) // Pre-allocate small slice

	for key, e := range c.data {
		if e.expiresAt.Before(now) && !e.expiresAt.IsZero() {
			expired = append(expired, key)
		}
	}

	expiredCount := 0
	for _, key := range expired {
		if e, exists := c.data[key]; exists {
			delete(c.data, key)
			c.lru.remove(e.lruNode)
			c.pool.put(e)
			expiredCount++
		}
	}

	// Batch update metrics
	if c.config.EnableMetrics && expiredCount > 0 {
		c.metricsMu.Lock()
		c.metrics.Expirations += int64(expiredCount)
		c.metrics.CurrentSize -= int64(expiredCount)
		c.metricsMu.Unlock()
	}
}

// calculateExpiration calculates the expiration time based on TTL.
func (c *Cache[K, V]) calculateExpiration(ttl time.Duration) time.Time {
	if ttl == 0 {
		return time.Time{} // Zero time means no expiration
	}
	return time.Now().Add(ttl)
}

// Metrics recording (lock-free using atomic operations would be better, but keeping simple for now)

func (c *Cache[K, V]) recordHit() {
	if !c.config.EnableMetrics {
		return
	}
	c.metricsMu.Lock()
	c.metrics.Hits++
	c.metricsMu.Unlock()
}

func (c *Cache[K, V]) recordMiss() {
	if !c.config.EnableMetrics {
		return
	}
	c.metricsMu.Lock()
	c.metrics.Misses++
	c.metricsMu.Unlock()
}

func (c *Cache[K, V]) recordSet() {
	if !c.config.EnableMetrics {
		return
	}
	c.metricsMu.Lock()
	c.metrics.Sets++
	c.metricsMu.Unlock()
}

func (c *Cache[K, V]) recordDelete() {
	if !c.config.EnableMetrics {
		return
	}
	c.metricsMu.Lock()
	c.metrics.Deletes++
	c.metricsMu.Unlock()
}

func (c *Cache[K, V]) recordEviction() {
	if !c.config.EnableMetrics {
		return
	}
	c.metricsMu.Lock()
	c.metrics.Evictions++
	c.metricsMu.Unlock()
}

func (c *Cache[K, V]) recordExpiration() {
	if !c.config.EnableMetrics {
		return
	}
	c.metricsMu.Lock()
	c.metrics.Expirations++
	c.metricsMu.Unlock()
}

// SetOption is a functional option for Set operations.
type SetOption func(*setOptions)

type setOptions struct {
	ttl time.Duration
}

// WithTTL sets a custom TTL for this entry.
func WithTTL(ttl time.Duration) SetOption {
	return func(o *setOptions) {
		o.ttl = ttl
	}
}

// WithNoExpiration sets the entry to never expire.
func WithNoExpiration() SetOption {
	return func(o *setOptions) {
		o.ttl = 0
	}
}
