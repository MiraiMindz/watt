package memory_test

import (
	"context"
	"fmt"
	"time"

	"github.com/watt-toolkit/capacitor/pkg/cache/memory"
)

// Example demonstrating basic cache usage
func ExampleCache_basic() {
	// Create a cache with 1000 entry limit
	cache := memory.New[string, int](memory.Config{
		MaxSize:      1000,
		DefaultTTL:   5 * time.Minute,
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set a value
	cache.Set(ctx, "user:123", 42)

	// Get the value
	value, err := cache.Get(ctx, "user:123")
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("Value: %d\n", value)

	// Output:
	// Value: 42
}

// Example demonstrating TTL (time-to-live)
func ExampleCache_ttl() {
	cache := memory.New[string, string](memory.Config{
		MaxSize:      100,
		DefaultTTL:   1 * time.Hour, // Default: 1 hour
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set with default TTL (1 hour)
	cache.Set(ctx, "session:abc", "user123")

	// Set with custom TTL (5 minutes)
	cache.Set(ctx, "temp:xyz", "temporary", memory.WithTTL(5*time.Minute))

	// Set with no expiration
	cache.Set(ctx, "permanent", "forever", memory.WithNoExpiration())

	fmt.Println("Values set with different TTLs")

	// Output:
	// Values set with different TTLs
}

// Example demonstrating LRU eviction
func ExampleCache_lru() {
	// Small cache for demonstration
	cache := memory.New[string, int](memory.Config{
		MaxSize:      3, // Only 3 entries
		DefaultTTL:   0, // No expiration
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	cache.Set(ctx, "a", 1)
	cache.Set(ctx, "b", 2)
	cache.Set(ctx, "c", 3)

	// Access 'a' to make it recently used
	cache.Get(ctx, "a")

	// Add new entry - 'b' will be evicted (least recently used)
	cache.Set(ctx, "d", 4)

	// Check what's in cache
	_, err := cache.Get(ctx, "b")
	if err != nil {
		fmt.Println("b was evicted")
	}

	_, err = cache.Get(ctx, "a")
	if err == nil {
		fmt.Println("a still in cache")
	}

	// Output:
	// b was evicted
	// a still in cache
}

// Example demonstrating cache metrics
func ExampleCache_metrics() {
	cache := memory.New[string, int](memory.Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  memory.EvictionLRU,
		EnableMetrics: true, // Enable metrics
	})
	defer cache.Close()

	ctx := context.Background()

	// Perform operations
	cache.Set(ctx, "key1", 100)
	cache.Set(ctx, "key2", 200)
	cache.Get(ctx, "key1")      // Hit
	cache.Get(ctx, "missing")   // Miss
	cache.Delete(ctx, "key2")

	// Get metrics
	metrics := cache.Metrics()
	fmt.Printf("Hits: %d\n", metrics.Hits)
	fmt.Printf("Misses: %d\n", metrics.Misses)
	fmt.Printf("Hit Rate: %.1f%%\n", metrics.HitRate()*100)
	fmt.Printf("Sets: %d\n", metrics.Sets)
	fmt.Printf("Deletes: %d\n", metrics.Deletes)
	fmt.Printf("Current Size: %d\n", metrics.CurrentSize)

	// Output:
	// Hits: 1
	// Misses: 1
	// Hit Rate: 50.0%
	// Sets: 2
	// Deletes: 1
	// Current Size: 1
}

// Example demonstrating update operations
func ExampleCache_update() {
	cache := memory.New[string, string](memory.Config{
		MaxSize:      100,
		DefaultTTL:   0,
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set initial value
	cache.Set(ctx, "status", "pending")
	val, _ := cache.Get(ctx, "status")
	fmt.Printf("Initial: %s\n", val)

	// Update value
	cache.Set(ctx, "status", "completed")
	val, _ = cache.Get(ctx, "status")
	fmt.Printf("Updated: %s\n", val)

	// Output:
	// Initial: pending
	// Updated: completed
}

// Example demonstrating cache clearing
func ExampleCache_clear() {
	cache := memory.New[string, int](memory.Config{
		MaxSize:      100,
		DefaultTTL:   0,
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Add entries
	for i := 0; i < 10; i++ {
		cache.Set(ctx, fmt.Sprintf("key%d", i), i)
	}

	fmt.Printf("Size before clear: %d\n", cache.Size())

	// Clear all entries
	cache.Clear(ctx)

	fmt.Printf("Size after clear: %d\n", cache.Size())

	// Output:
	// Size before clear: 10
	// Size after clear: 0
}

// Example demonstrating exists check
func ExampleCache_exists() {
	cache := memory.New[string, string](memory.Config{
		MaxSize:      100,
		DefaultTTL:   0,
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "user:123", "john")

	exists, _ := cache.Exists(ctx, "user:123")
	fmt.Printf("user:123 exists: %v\n", exists)

	exists, _ = cache.Exists(ctx, "user:999")
	fmt.Printf("user:999 exists: %v\n", exists)

	// Output:
	// user:123 exists: true
	// user:999 exists: false
}

// Example demonstrating eviction mode
func ExampleEvictionMode() {
	modes := []memory.EvictionMode{
		memory.EvictionNone,
		memory.EvictionLRU,
		memory.EvictionLFU,
		memory.EvictionRandom,
	}

	for _, mode := range modes {
		fmt.Println(mode.String())
	}

	// Output:
	// None
	// LRU
	// LFU
	// Random
}

// Example demonstrating cache with struct values
func ExampleCache_struct() {
	type User struct {
		ID    int64
		Name  string
		Email string
	}

	cache := memory.New[int64, *User](memory.Config{
		MaxSize:      1000,
		DefaultTTL:   10 * time.Minute,
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Store user
	user := &User{
		ID:    123,
		Name:  "John Doe",
		Email: "john@example.com",
	}
	cache.Set(ctx, 123, user)

	// Retrieve user
	retrieved, err := cache.Get(ctx, 123)
	if err != nil {
		fmt.Printf("Error: %v\n", err)
		return
	}

	fmt.Printf("User: %s (%s)\n", retrieved.Name, retrieved.Email)

	// Output:
	// User: John Doe (john@example.com)
}

// Example demonstrating automatic cleanup
func ExampleCache_cleanup() {
	cache := memory.New[string, string](memory.Config{
		MaxSize:         100,
		DefaultTTL:      100 * time.Millisecond,
		EvictionMode:    memory.EvictionLRU,
		CleanupInterval: 50 * time.Millisecond, // Cleanup every 50ms
		EnableMetrics:   true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Add entries
	cache.Set(ctx, "key1", "value1")
	cache.Set(ctx, "key2", "value2")

	fmt.Printf("Initial size: %d\n", cache.Size())

	// Wait for expiration and cleanup
	time.Sleep(200 * time.Millisecond)

	fmt.Printf("After cleanup: %d\n", cache.Size())

	metrics := cache.Metrics()
	fmt.Printf("Expirations: %d\n", metrics.Expirations)

	// Output:
	// Initial size: 2
	// After cleanup: 0
	// Expirations: 2
}

// Example demonstrating cache statistics
func ExampleCache_stats() {
	cache := memory.New[string, int](memory.Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  memory.EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Perform operations
	cache.Set(ctx, "a", 1)
	cache.Set(ctx, "b", 2)
	cache.Get(ctx, "a")
	cache.Get(ctx, "missing")

	stats := cache.Stats()
	fmt.Printf("Hit Rate: %.0f%%\n", stats.HitRate*100)
	fmt.Printf("Total Operations: %d\n", stats.Hits+stats.Misses)

	// Output:
	// Hit Rate: 50%
	// Total Operations: 2
}

// Example demonstrating concurrent access
func ExampleCache_concurrent() {
	cache := memory.New[int, int](memory.Config{
		MaxSize:      1000,
		DefaultTTL:   0,
		EvictionMode: memory.EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Concurrent writes
	done := make(chan bool)
	for i := 0; i < 5; i++ {
		go func(id int) {
			for j := 0; j < 10; j++ {
				cache.Set(ctx, id*10+j, id*10+j)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 5; i++ {
		<-done
	}

	fmt.Printf("Final size: %d\n", cache.Size())

	// Output:
	// Final size: 50
}

// Example demonstrating no eviction mode
func ExampleCache_noEviction() {
	cache := memory.New[string, int](memory.Config{
		MaxSize:      2,
		DefaultTTL:   0,
		EvictionMode: memory.EvictionNone, // No eviction
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	cache.Set(ctx, "a", 1)
	cache.Set(ctx, "b", 2)

	// Try to add third item
	err := cache.Set(ctx, "c", 3)
	if err != nil {
		fmt.Println("Cache full, cannot add more entries")
	}

	fmt.Printf("Size: %d\n", cache.Size())

	// Output:
	// Cache full, cannot add more entries
	// Size: 2
}
