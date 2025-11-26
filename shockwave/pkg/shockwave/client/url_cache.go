package client

import (
	"net/url"
	"sync"
)

// URLCacheEntry represents a cached parsed URL.
type URLCacheEntry struct {
	scheme string
	host   string
	port   string
	path   string
	query  string

	// For LRU tracking
	prev *URLCacheEntry
	next *URLCacheEntry
	key  string
}

// URLCache is a thread-safe LRU cache for parsed URLs.
// This eliminates the 1-2 allocations from url.Parse for frequently accessed URLs.
//
// Performance: 0 allocs/op on cache hit (vs 2 allocs for url.Parse)
type URLCache struct {
	mu sync.RWMutex

	// Cache storage
	entries map[string]*URLCacheEntry
	pool    sync.Pool // Pool for cache entries

	// LRU list
	head *URLCacheEntry
	tail *URLCacheEntry

	// Configuration
	maxSize int
	size    int

	// Stats
	hits   uint64
	misses uint64
}

const (
	// DefaultURLCacheSize is the default maximum number of cached URLs
	DefaultURLCacheSize = 1024
)

var globalURLCache = NewURLCache(DefaultURLCacheSize)

// NewURLCache creates a new URL cache with the specified maximum size.
func NewURLCache(maxSize int) *URLCache {
	c := &URLCache{
		entries: make(map[string]*URLCacheEntry, maxSize),
		maxSize: maxSize,
	}

	c.pool = sync.Pool{
		New: func() interface{} {
			return &URLCacheEntry{}
		},
	}

	return c
}

// Get retrieves a cached parsed URL.
// Returns nil if not found.
//
// Allocation behavior: 0 allocs/op on hit
func (c *URLCache) Get(urlStr string) *URLCacheEntry {
	c.mu.RLock()
	entry, ok := c.entries[urlStr]
	c.mu.RUnlock()

	if !ok {
		c.mu.Lock()
		c.misses++
		c.mu.Unlock()
		return nil
	}

	// Move to front (most recently used)
	c.mu.Lock()
	c.hits++
	c.moveToFront(entry)
	c.mu.Unlock()

	return entry
}

// Put adds a parsed URL to the cache.
//
// Allocation behavior: 0 allocs/op on pool hit
func (c *URLCache) Put(urlStr, scheme, host, port, path, query string) *URLCacheEntry {
	c.mu.Lock()
	defer c.mu.Unlock()

	// Check if already exists
	if entry, ok := c.entries[urlStr]; ok {
		c.moveToFront(entry)
		return entry
	}

	// Evict if necessary
	if c.size >= c.maxSize {
		c.evictLRU()
	}

	// Get entry from pool
	entry := c.pool.Get().(*URLCacheEntry)
	entry.key = urlStr
	entry.scheme = scheme
	entry.host = host
	entry.port = port
	entry.path = path
	entry.query = query

	// Add to cache
	c.entries[urlStr] = entry
	c.size++

	// Add to front of LRU list
	c.addToFront(entry)

	return entry
}

// ParseURL parses a URL using the cache.
// This is the main entry point - it checks cache first, falls back to url.Parse.
//
// Allocation behavior: 0 allocs/op on cache hit, 2 allocs/op on miss
func (c *URLCache) ParseURL(urlStr string) (scheme, host, port, path, query string, err error) {
	// Check cache first
	if entry := c.Get(urlStr); entry != nil {
		return entry.scheme, entry.host, entry.port, entry.path, entry.query, nil
	}

	// Cache miss - parse URL
	u, err := url.Parse(urlStr)
	if err != nil {
		return "", "", "", "", "", err
	}

	scheme = u.Scheme
	if scheme == "" {
		scheme = "http"
	}

	host = u.Hostname()
	port = u.Port()

	// Default ports
	if port == "" {
		if scheme == "https" {
			port = "443"
		} else {
			port = "80"
		}
	}

	path = u.Path
	if path == "" {
		path = "/"
	}

	query = u.RawQuery

	// Add to cache
	c.Put(urlStr, scheme, host, port, path, query)

	return scheme, host, port, path, query, nil
}

// moveToFront moves an entry to the front of the LRU list.
func (c *URLCache) moveToFront(entry *URLCacheEntry) {
	if entry == c.head {
		return
	}

	// Remove from current position
	if entry.prev != nil {
		entry.prev.next = entry.next
	}
	if entry.next != nil {
		entry.next.prev = entry.prev
	}
	if entry == c.tail {
		c.tail = entry.prev
	}

	// Add to front
	entry.prev = nil
	entry.next = c.head
	if c.head != nil {
		c.head.prev = entry
	}
	c.head = entry

	if c.tail == nil {
		c.tail = entry
	}
}

// addToFront adds an entry to the front of the LRU list.
func (c *URLCache) addToFront(entry *URLCacheEntry) {
	entry.prev = nil
	entry.next = c.head

	if c.head != nil {
		c.head.prev = entry
	}
	c.head = entry

	if c.tail == nil {
		c.tail = entry
	}
}

// evictLRU evicts the least recently used entry.
func (c *URLCache) evictLRU() {
	if c.tail == nil {
		return
	}

	// Remove from map
	delete(c.entries, c.tail.key)
	c.size--

	// Remove from list
	evicted := c.tail
	c.tail = evicted.prev
	if c.tail != nil {
		c.tail.next = nil
	} else {
		c.head = nil
	}

	// Return to pool
	evicted.prev = nil
	evicted.next = nil
	evicted.key = ""
	c.pool.Put(evicted)
}

// Stats returns cache statistics.
func (c *URLCache) Stats() (hits, misses uint64, size int) {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.hits, c.misses, c.size
}

// Clear clears the cache.
func (c *URLCache) Clear() {
	c.mu.Lock()
	defer c.mu.Unlock()

	c.entries = make(map[string]*URLCacheEntry, c.maxSize)
	c.head = nil
	c.tail = nil
	c.size = 0
	c.hits = 0
	c.misses = 0
}

// GetGlobalURLCache returns the global URL cache instance.
func GetGlobalURLCache() *URLCache {
	return globalURLCache
}
