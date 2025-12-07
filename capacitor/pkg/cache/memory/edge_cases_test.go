package memory

import (
	"context"
	"testing"
	"time"

	"github.com/watt-toolkit/capacitor/pkg/capacitor"
)

// Test HitRate edge cases
func TestMetrics_HitRate_EdgeCases(t *testing.T) {
	// No operations
	m := &Metrics{}
	if rate := m.HitRate(); rate != 0.0 {
		t.Errorf("HitRate with no operations = %f, want 0.0", rate)
	}

	// Only hits
	m.Hits = 10
	if rate := m.HitRate(); rate != 1.0 {
		t.Errorf("HitRate with only hits = %f, want 1.0", rate)
	}

	// Only misses
	m = &Metrics{Misses: 5}
	if rate := m.HitRate(); rate != 0.0 {
		t.Errorf("HitRate with only misses = %f, want 0.0", rate)
	}

	// Mixed
	m = &Metrics{Hits: 7, Misses: 3}
	if rate := m.HitRate(); rate != 0.7 {
		t.Errorf("HitRate with 7 hits, 3 misses = %f, want 0.7", rate)
	}
}

// Test cache with zero CleanupInterval
func TestCache_NoCleanup(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:         100,
		DefaultTTL:      100 * time.Millisecond,
		EvictionMode:    EvictionLRU,
		CleanupInterval: 0, // Disabled
		EnableMetrics:   true,
	})
	defer cache.Close()

	ctx := context.Background()

	// Add expired entry
	cache.Set(ctx, "key1", 1)
	time.Sleep(150 * time.Millisecond)

	// Entry should be expired but not cleaned up automatically
	_, err := cache.Get(ctx, "key1")
	if err != capacitor.ErrNotFound {
		t.Errorf("Get on expired entry without cleanup = %v, want ErrNotFound", err)
	}

	metrics := cache.Metrics()
	// Should be marked as expiration when Get detects it
	if metrics.Expirations < 1 {
		t.Errorf("Expirations = %d, want >= 1", metrics.Expirations)
	}
}

// Test Delete on non-existent key
func TestCache_DeleteNonExistent(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: true,
	})
	defer cache.Close()

	ctx := context.Background()

	err := cache.Delete(ctx, "nonexistent")
	if err != capacitor.ErrNotFound {
		t.Errorf("Delete nonexistent = %v, want ErrNotFound", err)
	}
}

// Test Exists on expired entry
func TestCache_ExistsExpired(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:         100,
		DefaultTTL:      50 * time.Millisecond,
		EvictionMode:    EvictionLRU,
		CleanupInterval: 0,
		EnableMetrics:   true,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", 1)

	// Should exist immediately
	exists, err := cache.Exists(ctx, "key1")
	if err != nil || !exists {
		t.Errorf("Exists immediately after Set = %v, %v; want true, nil", exists, err)
	}

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Should not exist after expiration
	exists, err = cache.Exists(ctx, "key1")
	if err != nil || exists {
		t.Errorf("Exists after expiration = %v, %v; want false, nil", exists, err)
	}
}

// Test Clear after Close
func TestCache_ClearAfterClose(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:      100,
		DefaultTTL:   0,
		EvictionMode: EvictionLRU,
	})

	ctx := context.Background()
	cache.Set(ctx, "key1", 1)
	cache.Close()

	err := cache.Clear(ctx)
	if err != capacitor.ErrClosed {
		t.Errorf("Clear after Close = %v, want ErrClosed", err)
	}
}

// Test Exists after Close
func TestCache_ExistsAfterClose(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:      100,
		DefaultTTL:   0,
		EvictionMode: EvictionLRU,
	})

	ctx := context.Background()
	cache.Set(ctx, "key1", 1)
	cache.Close()

	_, err := cache.Exists(ctx, "key1")
	if err != capacitor.ErrClosed {
		t.Errorf("Exists after Close = %v, want ErrClosed", err)
	}
}

// Test eviction failure scenarios
func TestCache_EvictionFailure(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:      2,
		DefaultTTL:   0,
		EvictionMode: EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache
	cache.Set(ctx, "a", 1)
	cache.Set(ctx, "b", 2)

	// Manually corrupt LRU to force eviction failure
	cache.mu.Lock()
	cache.lru = newLRUList[string]() // Empty LRU list
	cache.mu.Unlock()

	// Try to add another item - should fail eviction
	err := cache.Set(ctx, "c", 3)
	if err != capacitor.ErrEvictionFailed {
		t.Errorf("Set when eviction fails = %v, want ErrEvictionFailed", err)
	}
}

// Test metrics with disabled metrics
func TestCache_MetricsDisabled(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:       100,
		DefaultTTL:    0,
		EvictionMode:  EvictionLRU,
		EnableMetrics: false, // Disabled
	})
	defer cache.Close()

	ctx := context.Background()

	// Perform operations
	cache.Set(ctx, "key1", 1)
	cache.Get(ctx, "key1")
	cache.Get(ctx, "missing")
	cache.Delete(ctx, "key1")

	// Metrics should all be zero
	metrics := cache.Metrics()
	if metrics.Hits != 0 || metrics.Misses != 0 || metrics.Sets != 0 || metrics.Deletes != 0 {
		t.Errorf("Metrics with EnableMetrics=false should be zero, got %+v", metrics)
	}
}

// Test Get on expired entry triggers cleanup
func TestCache_GetExpiresEntry(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:         100,
		DefaultTTL:      50 * time.Millisecond,
		EvictionMode:    EvictionLRU,
		CleanupInterval: 0, // Manual cleanup only
		EnableMetrics:   true,
	})
	defer cache.Close()

	ctx := context.Background()

	cache.Set(ctx, "key1", 1)

	// Wait for expiration
	time.Sleep(100 * time.Millisecond)

	// Get should detect expiration and clean up
	_, err := cache.Get(ctx, "key1")
	if err != capacitor.ErrNotFound {
		t.Errorf("Get expired = %v, want ErrNotFound", err)
	}

	// Entry should be removed from cache
	if cache.Size() != 0 {
		t.Errorf("Size after Get on expired = %d, want 0", cache.Size())
	}

	metrics := cache.Metrics()
	if metrics.Expirations != 1 {
		t.Errorf("Expirations = %d, want 1", metrics.Expirations)
	}
}

// Test LRU update on Get
func TestCache_GetUpdatesLRU(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:      3,
		DefaultTTL:   0,
		EvictionMode: EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Fill cache: c -> b -> a (most recent to least recent)
	cache.Set(ctx, "a", 1)
	cache.Set(ctx, "b", 2)
	cache.Set(ctx, "c", 3)

	// Access 'a' to make it most recent
	cache.Get(ctx, "a")

	// Add new entry - 'b' should be evicted (now least recent)
	cache.Set(ctx, "d", 4)

	// Verify 'b' was evicted
	_, err := cache.Get(ctx, "b")
	if err != capacitor.ErrNotFound {
		t.Error("b should have been evicted")
	}

	// Verify others exist
	if _, err := cache.Get(ctx, "a"); err != nil {
		t.Error("a should exist (was made most recent)")
	}
	if _, err := cache.Get(ctx, "c"); err != nil {
		t.Error("c should exist")
	}
	if _, err := cache.Get(ctx, "d"); err != nil {
		t.Error("d should exist (just added)")
	}
}

// Test default config values
func TestCache_DefaultConfig(t *testing.T) {
	cache := New[string, int](Config{
		// Only set required fields, let defaults apply
		EvictionMode: EvictionLRU,
	})
	defer cache.Close()

	if cache.config.MaxSize != 10000 {
		t.Errorf("Default MaxSize = %d, want 10000", cache.config.MaxSize)
	}

	if cache.config.CleanupInterval != 1*time.Minute {
		t.Errorf("Default CleanupInterval = %v, want 1m", cache.config.CleanupInterval)
	}
}

// Test Update with different TTL
func TestCache_UpdateWithDifferentTTL(t *testing.T) {
	cache := New[string, int](Config{
		MaxSize:      100,
		DefaultTTL:   1 * time.Hour,
		EvictionMode: EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Set with long TTL
	cache.Set(ctx, "key1", 1, WithTTL(1*time.Hour))

	// Update with short TTL
	cache.Set(ctx, "key1", 2, WithTTL(50*time.Millisecond))

	// Immediately should still exist
	val, err := cache.Get(ctx, "key1")
	if err != nil || val != 2 {
		t.Errorf("Get after update = %v, %v; want 2, nil", val, err)
	}

	// Wait for new TTL to expire
	time.Sleep(100 * time.Millisecond)

	// Should be expired now
	_, err = cache.Get(ctx, "key1")
	if err != capacitor.ErrNotFound {
		t.Errorf("Get after new TTL expires = %v, want ErrNotFound", err)
	}
}

// Test concurrent updates to same key
func TestCache_ConcurrentUpdates(t *testing.T) {
	cache := New[int, int](Config{
		MaxSize:      1000,
		DefaultTTL:   0,
		EvictionMode: EvictionLRU,
	})
	defer cache.Close()

	ctx := context.Background()

	// Many goroutines updating same key
	done := make(chan bool)
	for i := 0; i < 10; i++ {
		go func(val int) {
			for j := 0; j < 100; j++ {
				cache.Set(ctx, 1, val)
			}
			done <- true
		}(i)
	}

	for i := 0; i < 10; i++ {
		<-done
	}

	// Should have exactly one entry
	if cache.Size() != 1 {
		t.Errorf("Size after concurrent updates = %d, want 1", cache.Size())
	}

	// Value should be one of the written values
	val, err := cache.Get(ctx, 1)
	if err != nil {
		t.Errorf("Get after concurrent updates failed: %v", err)
	}
	if val < 0 || val >= 10 {
		t.Errorf("Get after concurrent updates = %d, want 0-9", val)
	}
}
