package memory

import (
	"context"
	"testing"
	"time"

	"github.com/watt-toolkit/capacitor/pkg/capacitor"
)

func TestCache_BasicOperations(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0, // No expiration
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Test Set and Get
	if err := cache.Set(ctx, "key1", 100); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != 100 {
		t.Errorf("Get returned %d, want 100", val)
	}

	// Test Exists
	exists, err := cache.Exists(ctx, "key1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Exists returned false for existing key")
	}

	// Test Delete
	if err := cache.Delete(ctx, "key1"); err != nil {
		t.Fatalf("Delete failed: %v", err)
	}

	// Verify deleted
	_, err = cache.Get(ctx, "key1")
	if err != capacitor.ErrNotFound {
		t.Errorf("Get after delete returned %v, want ErrNotFound", err)
	}
}

func TestCache_TTL(t *testing.T) {
	cache := New[string, string](Config{
		MaxSize:       100,
		DefaultTTL:    100 * time.Millisecond,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set with default TTL
	if err := cache.Set(ctx, "key1", "value1"); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "value1" {
		t.Errorf("Get returned %s, want value1", val)
	}

	// Wait for expiration
	time.Sleep(150 * time.Millisecond)

	// Should be expired
	_, err = cache.Get(ctx, "key1")
	if err != capacitor.ErrNotFound {
		t.Errorf("Get after expiration returned %v, want ErrNotFound", err)
	}
}

func TestCache_CustomTTL(t *testing.T) {
	cache := New[string, string](Config{
		MaxSize:       100,
		DefaultTTL:    1 * time.Hour, // Long default
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set with custom short TTL
	if err := cache.Set(ctx, "key1", "value1", WithTTL(50*time.Millisecond)); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Should exist immediately
	_, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should be expired
	_, err = cache.Get(ctx, "key1")
	if err != capacitor.ErrNotFound {
		t.Errorf("Get after expiration returned %v, want ErrNotFound", err)
	}
}

func TestCache_NoExpiration(t *testing.T) {
	cache := New[string, string](Config{
		MaxSize:       100,
		DefaultTTL:    100 * time.Millisecond,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set with no expiration
	if err := cache.Set(ctx, "key1", "value1", WithNoExpiration()); err != nil {
		t.Fatalf("Set failed: %v", err)
	}

	// Wait longer than default TTL
	time.Sleep(200 * time.Millisecond)

	// Should still exist
	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != "value1" {
		t.Errorf("Get returned %s, want value1", val)
	}
}

func TestCache_LRUEviction(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       3, // Small cache for testing
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	cache.Set(ctx, "key1", 1)
	cache.Set(ctx, "key2", 2)
	cache.Set(ctx, "key3", 3)

	// Access key1 to make it most recently used
	cache.Get(ctx, "key1")

	// Add key4, should evict key2 (least recently used)
	cache.Set(ctx, "key4", 4)

	// key2 should be evicted
	_, err := cache.Get(ctx, "key2")
	if err != capacitor.ErrNotFound {
		t.Errorf("key2 should be evicted, got error: %v", err)
	}

	// key1, key3, key4 should exist
	if _, err := cache.Get(ctx, "key1"); err != nil {
		t.Errorf("key1 should exist: %v", err)
	}
	if _, err := cache.Get(ctx, "key3"); err != nil {
		t.Errorf("key3 should exist: %v", err)
	}
	if _, err := cache.Get(ctx, "key4"); err != nil {
		t.Errorf("key4 should exist: %v", err)
	}
}

func TestCache_UpdateExisting(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set initial value
	cache.Set(ctx, "key1", 100)

	// Update value
	cache.Set(ctx, "key1", 200)

	// Should have new value
	val, err := cache.Get(ctx, "key1")
	if err != nil {
		t.Fatalf("Get failed: %v", err)
	}
	if val != 200 {
		t.Errorf("Get returned %d, want 200", val)
	}

	// Size should still be 1
	if cache.Size() != 1 {
		t.Errorf("Size = %d, want 1", cache.Size())
	}
}

func TestCache_Clear(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Add entries
	for i := 0; i < 10; i++ {
		cache.Set(ctx, string(rune('a'+i)), i)
	}

	if cache.Size() != 10 {
		t.Errorf("Size before clear = %d, want 10", cache.Size())
	}

	// Clear
	if err := cache.Clear(ctx); err != nil {
		t.Fatalf("Clear failed: %v", err)
	}

	if cache.Size() != 0 {
		t.Errorf("Size after clear = %d, want 0", cache.Size())
	}
}

func TestCache_Metrics(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Perform operations
	cache.Set(ctx, "key1", 1)
	cache.Set(ctx, "key2", 2)
	cache.Get(ctx, "key1")                 // Hit
	cache.Get(ctx, "nonexistent")          // Miss
	cache.Delete(ctx, "key2")

	metrics := cache.Metrics()

	if metrics.Sets != 2 {
		t.Errorf("Sets = %d, want 2", metrics.Sets)
	}
	if metrics.Hits != 1 {
		t.Errorf("Hits = %d, want 1", metrics.Hits)
	}
	if metrics.Misses != 1 {
		t.Errorf("Misses = %d, want 1", metrics.Misses)
	}
	if metrics.Deletes != 1 {
		t.Errorf("Deletes = %d, want 1", metrics.Deletes)
	}
	if metrics.CurrentSize != 1 {
		t.Errorf("CurrentSize = %d, want 1", metrics.CurrentSize)
	}

	hitRate := metrics.HitRate()
	if hitRate != 0.5 {
		t.Errorf("HitRate = %f, want 0.5", hitRate)
	}
}

func TestCache_Cleanup(t *testing.T) {
	cache := New[string, string](Config{
		MaxSize:         100,
		DefaultTTL:      50 * time.Millisecond,
		EvictionMode:    EvictionLRU,
		CleanupInterval: 25 * time.Millisecond, // Fast cleanup for testing
		EnableMetrics:   true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Add entries
	cache.Set(ctx, "key1", "value1")
	cache.Set(ctx, "key2", "value2")
	cache.Set(ctx, "key3", "value3")

	// Wait for expiration and cleanup
	time.Sleep(100 * time.Millisecond)

	// All should be cleaned up
	if cache.Size() != 0 {
		t.Errorf("Size after cleanup = %d, want 0", cache.Size())
	}

	metrics := cache.Metrics()
	if metrics.Expirations != 3 {
		t.Errorf("Expirations = %d, want 3", metrics.Expirations)
	}
}

func TestCache_Close(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})

	ctx := context.Background()

	// Add entry
	cache.Set(ctx, "key1", 100)

	// Close
	if err := cache.Close(); err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Operations after close should fail
	if err := cache.Set(ctx, "key2", 200); err != capacitor.ErrClosed {
		t.Errorf("Set after close returned %v, want ErrClosed", err)
	}

	_, err := cache.Get(ctx, "key1")
	if err != capacitor.ErrClosed {
		t.Errorf("Get after close returned %v, want ErrClosed", err)
	}

	// Double close should return error
	if err := cache.Close(); err != capacitor.ErrClosed {
		t.Errorf("Second close returned %v, want ErrClosed", err)
	}
}

func TestCache_EvictionNone(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       2,
		DefaultTTL:    0,
		EvictionMode:  EvictionNone,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	cache.Set(ctx, "key1", 1)
	cache.Set(ctx, "key2", 2)

	// Try to add third item
	err := cache.Set(ctx, "key3", 3)
	if err != capacitor.ErrEvictionFailed {
		t.Errorf("Set when full returned %v, want ErrEvictionFailed", err)
	}
}

func TestEvictionMode_String(t *testing.T) {
	tests := []struct {
		mode EvictionMode
		want string
	}{
		{EvictionNone, "None"},
		{EvictionLRU, "LRU"},
		{EvictionLFU, "LFU"},
		{EvictionRandom, "Random"},
		{EvictionMode(99), "Unknown"},
	}

	for _, tt := range tests {
		if got := tt.mode.String(); got != tt.want {
			t.Errorf("EvictionMode(%d).String() = %s, want %s", tt.mode, got, tt.want)
		}
	}
}

func TestCache_ConcurrentAccess(t *testing.T) {
	cache := New[int, int](Config{
		MaxSize:       1000,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Run concurrent operations
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(id int) {
			for j := 0; j < 100; j++ {
				key := id*100 + j
				cache.Set(ctx, key, key)
				cache.Get(ctx, key)
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 10; i++ {
		<-done
	}

	// Verify metrics are consistent
	metrics := cache.Metrics()
	if metrics.Sets != 1000 {
		t.Errorf("Concurrent sets = %d, want 1000", metrics.Sets)
	}
}

func TestCache_Stats(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Perform operations
	cache.Set(ctx, "key1", 1)
	cache.Get(ctx, "key1")
	cache.Get(ctx, "missing")

	stats := cache.Stats()

	if stats.Sets != 1 {
		t.Errorf("Stats.Sets = %d, want 1", stats.Sets)
	}
	if stats.Hits != 1 {
		t.Errorf("Stats.Hits = %d, want 1", stats.Hits)
	}
	if stats.Misses != 1 {
		t.Errorf("Stats.Misses = %d, want 1", stats.Misses)
	}
	if stats.HitRate != 0.5 {
		t.Errorf("Stats.HitRate = %f, want 0.5", stats.HitRate)
	}
}
