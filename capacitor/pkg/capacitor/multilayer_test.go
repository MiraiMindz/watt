package capacitor

import (
	"context"
	"errors"
	"sync"
	"testing"
	"time"
)

// mockLayer is a simple in-memory layer for testing.
type mockLayer[K comparable, V any] struct {
	name    string
	data    map[K]V
	mu      sync.RWMutex
	stats   LayerStats
	closed  bool
	failGet bool
	failSet bool
	failDel bool
}

func newMockLayer[K comparable, V any](name string) *mockLayer[K, V] {
	return &mockLayer[K, V]{
		name: name,
		data: make(map[K]V),
		stats: LayerStats{
			Name: name,
		},
	}
}

func (m *mockLayer[K, V]) Get(ctx context.Context, key K) (V, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		var zero V
		return zero, ErrClosed
	}

	if m.failGet {
		var zero V
		return zero, errors.New("mock get error")
	}

	value, ok := m.data[key]
	if !ok {
		m.stats.Misses++
		var zero V
		return zero, ErrNotFound
	}

	m.stats.Hits++
	return value, nil
}

func (m *mockLayer[K, V]) Set(ctx context.Context, key K, value V, opts ...SetOption) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	if m.failSet {
		return errors.New("mock set error")
	}

	m.data[key] = value
	m.stats.Sets++
	m.stats.Size = int64(len(m.data))
	return nil
}

func (m *mockLayer[K, V]) Delete(ctx context.Context, key K) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if m.closed {
		return ErrClosed
	}

	if m.failDel {
		return errors.New("mock delete error")
	}

	if _, exists := m.data[key]; !exists {
		return ErrNotFound
	}

	delete(m.data, key)
	m.stats.Deletes++
	m.stats.Size = int64(len(m.data))
	return nil
}

func (m *mockLayer[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return false, ErrClosed
	}

	_, exists := m.data[key]
	return exists, nil
}

func (m *mockLayer[K, V]) Stats() LayerStats {
	m.mu.RLock()
	defer m.mu.RUnlock()

	stats := m.stats
	total := stats.Hits + stats.Misses
	if total > 0 {
		stats.HitRate = float64(stats.Hits) / float64(total)
	}
	return stats
}

func (m *mockLayer[K, V]) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()

	m.closed = true
	return nil
}

func (m *mockLayer[K, V]) GetMulti(ctx context.Context, keys []K) (map[K]V, map[K]error) {
	// Not implemented - return nil to indicate no batch support
	return nil, nil
}

func (m *mockLayer[K, V]) SetMulti(ctx context.Context, items map[K]V, opts ...SetOption) map[K]error {
	// Not implemented - return nil to indicate no batch support
	return nil
}

func (m *mockLayer[K, V]) DeleteMulti(ctx context.Context, keys []K) map[K]error {
	// Not implemented - return nil to indicate no batch support
	return nil
}

func (m *mockLayer[K, V]) Range(ctx context.Context, f func(key K, value V) bool) error {
	return ErrIterationNotSupported
}

func (m *mockLayer[K, V]) BeginTx(ctx context.Context, opts ...TxOption) (Tx[K, V], error) {
	return nil, ErrTxNotSupported
}

// Test NewMultiLayer
func TestNewMultiLayer(t *testing.T) {
	t.Run("ValidConfig", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1, TTL: time.Minute},
				{Name: "L2", Layer: l2, TTL: time.Hour},
			},
			EnableMetrics:   true,
			EnablePromotion: true,
			WriteThrough:    true,
		}

		dal, err := NewMultiLayer(config)
		if err != nil {
			t.Fatalf("NewMultiLayer failed: %v", err)
		}
		if dal == nil {
			t.Fatal("NewMultiLayer returned nil")
		}
		defer dal.Close()
	})

	t.Run("EmptyConfig", func(t *testing.T) {
		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{},
		}

		_, err := NewMultiLayer(config)
		if err == nil {
			t.Fatal("Expected error for empty config")
		}
	})

	t.Run("DuplicateLayerNames", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L1") // Same name

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L1", Layer: l2}, // Duplicate
			},
		}

		_, err := NewMultiLayer(config)
		if err == nil {
			t.Fatal("Expected error for duplicate layer names")
		}
	})
}

// Test Get operation
func TestMultiLayerDAL_Get(t *testing.T) {
	t.Run("GetFromL1", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set in L1
		l1.Set(ctx, "key1", 100)

		value, err := dal.Get(ctx, "key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if value != 100 {
			t.Errorf("Expected 100, got %d", value)
		}
	})

	t.Run("GetFromL2WithPromotion", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			EnablePromotion: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set only in L2
		l2.Set(ctx, "key1", 200)

		value, err := dal.Get(ctx, "key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if value != 200 {
			t.Errorf("Expected 200, got %d", value)
		}

		// Verify promotion to L1
		l1Value, err := l1.Get(ctx, "key1")
		if err != nil {
			t.Errorf("Value not promoted to L1: %v", err)
		}
		if l1Value != 200 {
			t.Errorf("Promoted value incorrect: expected 200, got %d", l1Value)
		}
	})

	t.Run("GetFromL2WithoutPromotion", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			EnablePromotion: false,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set only in L2
		l2.Set(ctx, "key1", 200)

		value, err := dal.Get(ctx, "key1")
		if err != nil {
			t.Fatalf("Get failed: %v", err)
		}
		if value != 200 {
			t.Errorf("Expected 200, got %d", value)
		}

		// Verify NO promotion to L1
		_, err = l1.Get(ctx, "key1")
		if err != ErrNotFound {
			t.Error("Value should not be promoted to L1")
		}
	})

	t.Run("GetNotFound", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		_, err := dal.Get(ctx, "nonexistent")
		if err != ErrNotFound {
			t.Errorf("Expected ErrNotFound, got %v", err)
		}
	})

	t.Run("GetAfterClose", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		dal.Close()

		ctx := context.Background()
		_, err := dal.Get(ctx, "key1")
		if err != ErrClosed {
			t.Errorf("Expected ErrClosed, got %v", err)
		}
	})

	t.Run("GetWithLayerFailure", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set in L2
		l2.Set(ctx, "key1", 300)

		// Make L1 fail
		l1.failGet = true

		// Should still get from L2
		value, err := dal.Get(ctx, "key1")
		if err != nil {
			t.Fatalf("Get should succeed from L2: %v", err)
		}
		if value != 300 {
			t.Errorf("Expected 300, got %d", value)
		}
	})
}

// Test Set operation
func TestMultiLayerDAL_Set(t *testing.T) {
	t.Run("SetWithWriteThrough", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			WriteThrough: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		err := dal.Set(ctx, "key1", 100)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Verify in both layers
		l1Val, err1 := l1.Get(ctx, "key1")
		l2Val, err2 := l2.Get(ctx, "key1")

		if err1 != nil || err2 != nil {
			t.Fatalf("Value not in both layers: L1=%v, L2=%v", err1, err2)
		}
		if l1Val != 100 || l2Val != 100 {
			t.Errorf("Values incorrect: L1=%d, L2=%d", l1Val, l2Val)
		}
	})

	t.Run("SetWithoutWriteThrough", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			WriteThrough: false,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		err := dal.Set(ctx, "key1", 100)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Verify only in L1
		l1Val, err1 := l1.Get(ctx, "key1")
		_, err2 := l2.Get(ctx, "key1")

		if err1 != nil {
			t.Fatalf("Value not in L1: %v", err1)
		}
		if err2 != ErrNotFound {
			t.Error("Value should not be in L2")
		}
		if l1Val != 100 {
			t.Errorf("L1 value incorrect: %d", l1Val)
		}
	})

	t.Run("SetReadOnlyLayer", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2, ReadOnly: true},
			},
			WriteThrough: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		err := dal.Set(ctx, "key1", 100)
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Verify only in L1 (L2 is read-only)
		l1Val, err1 := l1.Get(ctx, "key1")
		_, err2 := l2.Get(ctx, "key1")

		if err1 != nil {
			t.Fatalf("Value not in L1: %v", err1)
		}
		if err2 != ErrNotFound {
			t.Error("Value should not be in read-only L2")
		}
		if l1Val != 100 {
			t.Errorf("L1 value incorrect: %d", l1Val)
		}
	})

	t.Run("SetWithPartialFailure", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			WriteThrough: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Make L2 fail
		l2.failSet = true

		// Should still succeed (L1 succeeded)
		err := dal.Set(ctx, "key1", 100)
		if err != nil {
			t.Fatalf("Set should succeed with partial failure: %v", err)
		}

		// Verify in L1
		l1Val, err1 := l1.Get(ctx, "key1")
		if err1 != nil || l1Val != 100 {
			t.Error("Value should be in L1")
		}
	})
}

// Test Delete operation
func TestMultiLayerDAL_Delete(t *testing.T) {
	t.Run("DeleteFromAllLayers", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set in both layers
		l1.Set(ctx, "key1", 100)
		l2.Set(ctx, "key1", 100)

		err := dal.Delete(ctx, "key1")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deleted from both
		_, err1 := l1.Get(ctx, "key1")
		_, err2 := l2.Get(ctx, "key1")

		if err1 != ErrNotFound || err2 != ErrNotFound {
			t.Error("Key should be deleted from all layers")
		}
	})

	t.Run("DeleteNonExistent", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Delete should be idempotent
		err := dal.Delete(ctx, "nonexistent")
		if err != nil {
			t.Errorf("Delete of non-existent key should succeed: %v", err)
		}
	})

	t.Run("DeleteReadOnlyLayer", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2, ReadOnly: true},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set in both layers
		l1.Set(ctx, "key1", 100)
		l2.Set(ctx, "key1", 100)

		err := dal.Delete(ctx, "key1")
		if err != nil {
			t.Fatalf("Delete failed: %v", err)
		}

		// Verify deleted from L1 only
		_, err1 := l1.Get(ctx, "key1")
		l2Val, err2 := l2.Get(ctx, "key1")

		if err1 != ErrNotFound {
			t.Error("Key should be deleted from L1")
		}
		if err2 != nil || l2Val != 100 {
			t.Error("Key should remain in read-only L2")
		}
	})
}

// Test Exists operation
func TestMultiLayerDAL_Exists(t *testing.T) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Set in L2 only
	l2.Set(ctx, "key1", 100)

	exists, err := dal.Exists(ctx, "key1")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if !exists {
		t.Error("Key should exist")
	}

	exists, err = dal.Exists(ctx, "nonexistent")
	if err != nil {
		t.Fatalf("Exists failed: %v", err)
	}
	if exists {
		t.Error("Key should not exist")
	}
}

// Test GetFromLayer
func TestMultiLayerDAL_GetFromLayer(t *testing.T) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Set in L2 only
	l2.Set(ctx, "key1", 200)

	// Get from L1 (should fail)
	_, err := dal.GetFromLayer(ctx, "L1", "key1")
	if err != ErrNotFound {
		t.Errorf("Expected ErrNotFound from L1, got %v", err)
	}

	// Get from L2 (should succeed)
	value, err := dal.GetFromLayer(ctx, "L2", "key1")
	if err != nil {
		t.Fatalf("GetFromLayer(L2) failed: %v", err)
	}
	if value != 200 {
		t.Errorf("Expected 200, got %d", value)
	}

	// Get from non-existent layer
	_, err = dal.GetFromLayer(ctx, "L3", "key1")
	if err != ErrLayerNotFound {
		t.Errorf("Expected ErrLayerNotFound, got %v", err)
	}
}

// Test Stats
func TestMultiLayerDAL_Stats(t *testing.T) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
		EnablePromotion: true,
	}

	dal, _ := NewMultiLayer(config)
	defer dal.Close()

	ctx := context.Background()

	// Generate some operations
	l1.Set(ctx, "key1", 100)
	l2.Set(ctx, "key2", 200)

	dal.Get(ctx, "key1") // L1 hit
	dal.Get(ctx, "key2") // L2 hit (promotes to L1)
	dal.Get(ctx, "key3") // Miss in both

	stats := dal.Stats()

	if len(stats.Layers) != 2 {
		t.Errorf("Expected 2 layers in stats, got %d", len(stats.Layers))
	}

	if stats.TotalHits == 0 {
		t.Error("Expected some hits")
	}

	if stats.TotalMisses == 0 {
		t.Error("Expected some misses")
	}

	if stats.HitRate <= 0 || stats.HitRate > 1 {
		t.Errorf("Invalid hit rate: %f", stats.HitRate)
	}
}

// Test batch operations
func TestMultiLayerDAL_BatchOperations(t *testing.T) {
	t.Run("GetMulti", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			EnablePromotion: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set values in different layers
		l1.Set(ctx, "key1", 100)
		l2.Set(ctx, "key2", 200)

		keys := []string{"key1", "key2", "key3"}
		values, errors := dal.GetMulti(ctx, keys)

		if len(values) != 2 {
			t.Errorf("Expected 2 values, got %d", len(values))
		}
		if values["key1"] != 100 {
			t.Error("key1 value incorrect")
		}
		if values["key2"] != 200 {
			t.Error("key2 value incorrect")
		}
		if errors["key3"] != ErrNotFound {
			t.Error("key3 should have ErrNotFound")
		}
	})

	t.Run("SetMulti", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			WriteThrough: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		items := map[string]int{
			"key1": 100,
			"key2": 200,
		}

		errors := dal.SetMulti(ctx, items)
		if len(errors) > 0 {
			t.Errorf("SetMulti had errors: %v", errors)
		}

		// Verify in both layers
		for key, expectedValue := range items {
			l1Val, _ := l1.Get(ctx, key)
			l2Val, _ := l2.Get(ctx, key)
			if l1Val != expectedValue || l2Val != expectedValue {
				t.Errorf("Values not set correctly for %s", key)
			}
		}
	})

	t.Run("DeleteMulti", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set values
		l1.Set(ctx, "key1", 100)
		l1.Set(ctx, "key2", 200)
		l2.Set(ctx, "key1", 100)
		l2.Set(ctx, "key2", 200)

		keys := []string{"key1", "key2"}
		errors := dal.DeleteMulti(ctx, keys)

		if len(errors) > 0 {
			t.Errorf("DeleteMulti had errors: %v", errors)
		}

		// Verify deleted from both layers
		for _, key := range keys {
			_, err1 := l1.Get(ctx, key)
			_, err2 := l2.Get(ctx, key)
			if err1 != ErrNotFound || err2 != ErrNotFound {
				t.Errorf("Key %s not deleted from all layers", key)
			}
		}
	})
}

// Test Close
func TestMultiLayerDAL_Close(t *testing.T) {
	l1 := newMockLayer[string, int]("L1")
	l2 := newMockLayer[string, int]("L2")

	config := Config[string, int]{
		Layers: []LayerConfig[string, int]{
			{Name: "L1", Layer: l1},
			{Name: "L2", Layer: l2},
		},
	}

	dal, _ := NewMultiLayer(config)

	err := dal.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Verify layers are closed
	if !l1.closed || !l2.closed {
		t.Error("Layers should be closed")
	}

	// Verify operations return ErrClosed
	ctx := context.Background()
	_, err = dal.Get(ctx, "key1")
	if err != ErrClosed {
		t.Errorf("Expected ErrClosed after Close, got %v", err)
	}

	// Close again should return ErrClosed
	err = dal.Close()
	if err != ErrClosed {
		t.Errorf("Expected ErrClosed on second Close, got %v", err)
	}
}
