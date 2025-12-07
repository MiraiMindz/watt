package capacitor

import (
	"context"
	"fmt"
	"sync"
	"testing"
)

// mockIterableLayer extends mockLayer with iteration support
type mockIterableLayer[K comparable, V any] struct {
	*mockLayer[K, V]
	supportsIteration bool
}

func newMockIterableLayer[K comparable, V any](name string, supportsIteration bool) *mockIterableLayer[K, V] {
	return &mockIterableLayer[K, V]{
		mockLayer:         newMockLayer[K, V](name),
		supportsIteration: supportsIteration,
	}
}

func (m *mockIterableLayer[K, V]) Range(ctx context.Context, f func(key K, value V) bool) error {
	if !m.supportsIteration {
		return ErrIterationNotSupported
	}

	m.mu.RLock()
	defer m.mu.RUnlock()

	if m.closed {
		return ErrClosed
	}

	for k, v := range m.data {
		if !f(k, v) {
			break
		}
	}

	return nil
}

// mockTxLayer extends mockLayer with transaction support
type mockTxLayer[K comparable, V any] struct {
	*mockLayer[K, V]
	supportsTx bool
}

func newMockTxLayer[K comparable, V any](name string, supportsTx bool) *mockTxLayer[K, V] {
	return &mockTxLayer[K, V]{
		mockLayer:  newMockLayer[K, V](name),
		supportsTx: supportsTx,
	}
}

type mockTx[K comparable, V any] struct {
	layer  *mockTxLayer[K, V]
	closed bool
	mu     sync.Mutex
}

func (m *mockTxLayer[K, V]) BeginTx(ctx context.Context, opts ...TxOption) (Tx[K, V], error) {
	if !m.supportsTx {
		return nil, ErrTxNotSupported
	}

	return &mockTx[K, V]{
		layer:  m,
		closed: false,
	}, nil
}

func (tx *mockTx[K, V]) Get(ctx context.Context, key K) (V, error) {
	return tx.layer.Get(ctx, key)
}

func (tx *mockTx[K, V]) Set(ctx context.Context, key K, value V, opts ...SetOption) error {
	return tx.layer.Set(ctx, key, value, opts...)
}

func (tx *mockTx[K, V]) Delete(ctx context.Context, key K) error {
	return tx.layer.Delete(ctx, key)
}

func (tx *mockTx[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	return tx.layer.Exists(ctx, key)
}

func (tx *mockTx[K, V]) GetMulti(ctx context.Context, keys []K) (map[K]V, map[K]error) {
	return tx.layer.GetMulti(ctx, keys)
}

func (tx *mockTx[K, V]) SetMulti(ctx context.Context, items map[K]V, opts ...SetOption) map[K]error {
	return tx.layer.SetMulti(ctx, items, opts...)
}

func (tx *mockTx[K, V]) DeleteMulti(ctx context.Context, keys []K) map[K]error {
	return tx.layer.DeleteMulti(ctx, keys)
}

func (tx *mockTx[K, V]) Commit(ctx context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.closed {
		return ErrClosed
	}

	tx.closed = true
	return nil
}

func (tx *mockTx[K, V]) Rollback(ctx context.Context) error {
	tx.mu.Lock()
	defer tx.mu.Unlock()

	if tx.closed {
		return nil // Idempotent
	}

	tx.closed = true
	return nil
}

func (tx *mockTx[K, V]) ID() string {
	return "mock-tx-id"
}

// Test iteration support
func TestMultiLayerDAL_Range(t *testing.T) {
	t.Run("RangeSuccess", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", false)
		l2 := newMockIterableLayer[string, int]("L2", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Populate L2
		l2.Set(ctx, "key1", 100)
		l2.Set(ctx, "key2", 200)
		l2.Set(ctx, "key3", 300)

		// Range over all items
		count := 0
		err := dal.Range(ctx, func(k string, v int) bool {
			count++
			return true
		})

		if err != nil {
			t.Fatalf("Range failed: %v", err)
		}
		if count != 3 {
			t.Errorf("Expected 3 items, got %d", count)
		}
	})

	t.Run("RangeEarlyExit", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Populate L1
		for i := 0; i < 10; i++ {
			l1.Set(ctx, fmt.Sprintf("key%d", i), i)
		}

		// Range with early exit
		count := 0
		err := dal.Range(ctx, func(k string, v int) bool {
			count++
			return count < 5 // Stop after 5 items
		})

		if err != nil {
			t.Fatalf("Range failed: %v", err)
		}
		if count != 5 {
			t.Errorf("Expected 5 items, got %d", count)
		}
	})

	t.Run("RangeNotSupported", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", false)
		l2 := newMockIterableLayer[string, int]("L2", false)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		err := dal.Range(ctx, func(k string, v int) bool {
			return true
		})

		if err != ErrIterationNotSupported {
			t.Errorf("Expected ErrIterationNotSupported, got %v", err)
		}
	})

	t.Run("RangeAfterClose", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		dal.Close()

		ctx := context.Background()

		err := dal.Range(ctx, func(k string, v int) bool {
			return true
		})

		if err != ErrClosed {
			t.Errorf("Expected ErrClosed, got %v", err)
		}
	})
}

// Test Keys iteration
func TestMultiLayerDAL_Keys(t *testing.T) {
	t.Run("KeysSuccess", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", false)
		l2 := newMockIterableLayer[string, int]("L2", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Populate L2
		expected := map[string]bool{
			"key1": true,
			"key2": true,
			"key3": true,
		}
		for k := range expected {
			l2.Set(ctx, k, 100)
		}

		// Get keys
		keysCh, err := dal.Keys(ctx)
		if err != nil {
			t.Fatalf("Keys failed: %v", err)
		}

		// Collect keys
		keys := make(map[string]bool)
		for key := range keysCh {
			keys[key] = true
		}

		if len(keys) != len(expected) {
			t.Errorf("Expected %d keys, got %d", len(expected), len(keys))
		}

		for k := range expected {
			if !keys[k] {
				t.Errorf("Missing key: %s", k)
			}
		}
	})

	t.Run("KeysNotSupported", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", false)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		_, err := dal.Keys(ctx)
		if err != ErrIterationNotSupported {
			t.Errorf("Expected ErrIterationNotSupported, got %v", err)
		}
	})

	t.Run("KeysAfterClose", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		dal.Close()

		ctx := context.Background()

		_, err := dal.Keys(ctx)
		if err != ErrClosed {
			t.Errorf("Expected ErrClosed, got %v", err)
		}
	})

	t.Run("KeysContextCancel", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		// Populate with many keys
		ctx := context.Background()
		for i := 0; i < 100; i++ {
			l1.Set(ctx, fmt.Sprintf("key%d", i), i)
		}

		// Create cancelable context
		ctx, cancel := context.WithCancel(context.Background())

		keysCh, err := dal.Keys(ctx)
		if err != nil {
			t.Fatalf("Keys failed: %v", err)
		}

		// Cancel immediately
		cancel()

		// Channel should close
		count := 0
		for range keysCh {
			count++
		}

		// Should get 0 or very few keys due to immediate cancel
		if count > 10 {
			t.Errorf("Expected few keys after cancel, got %d", count)
		}
	})
}

// Test Values iteration
func TestMultiLayerDAL_Values(t *testing.T) {
	t.Run("ValuesSuccess", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", false)
		l2 := newMockIterableLayer[string, int]("L2", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Populate L2
		l2.Set(ctx, "key1", 100)
		l2.Set(ctx, "key2", 200)
		l2.Set(ctx, "key3", 300)

		// Get values
		valuesCh, err := dal.Values(ctx)
		if err != nil {
			t.Fatalf("Values failed: %v", err)
		}

		// Collect values
		values := make(map[int]bool)
		for value := range valuesCh {
			values[value] = true
		}

		if len(values) != 3 {
			t.Errorf("Expected 3 values, got %d", len(values))
		}

		expected := map[int]bool{100: true, 200: true, 300: true}
		for v := range expected {
			if !values[v] {
				t.Errorf("Missing value: %d", v)
			}
		}
	})

	t.Run("ValuesNotSupported", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", false)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		_, err := dal.Values(ctx)
		if err != ErrIterationNotSupported {
			t.Errorf("Expected ErrIterationNotSupported, got %v", err)
		}
	})

	t.Run("ValuesAfterClose", func(t *testing.T) {
		l1 := newMockIterableLayer[string, int]("L1", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		dal.Close()

		ctx := context.Background()

		_, err := dal.Values(ctx)
		if err != ErrClosed {
			t.Errorf("Expected ErrClosed, got %v", err)
		}
	})
}

// Test transaction support
func TestMultiLayerDAL_BeginTx(t *testing.T) {
	t.Run("BeginTxSuccess", func(t *testing.T) {
		l1 := newMockTxLayer[string, int]("L1", false)
		l2 := newMockTxLayer[string, int]("L2", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		tx, err := dal.BeginTx(ctx)
		if err != nil {
			t.Fatalf("BeginTx failed: %v", err)
		}

		if tx == nil {
			t.Fatal("BeginTx returned nil transaction")
		}

		err = tx.Commit(ctx)
		if err != nil {
			t.Fatalf("Commit failed: %v", err)
		}
	})

	t.Run("BeginTxNotSupported", func(t *testing.T) {
		l1 := newMockTxLayer[string, int]("L1", false)
		l2 := newMockTxLayer[string, int]("L2", false)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		_, err := dal.BeginTx(ctx)
		if err != ErrTxNotSupported {
			t.Errorf("Expected ErrTxNotSupported, got %v", err)
		}
	})

	t.Run("BeginTxAfterClose", func(t *testing.T) {
		l1 := newMockTxLayer[string, int]("L1", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		dal.Close()

		ctx := context.Background()

		_, err := dal.BeginTx(ctx)
		if err != ErrClosed {
			t.Errorf("Expected ErrClosed, got %v", err)
		}
	})

	t.Run("BeginTxWithOptions", func(t *testing.T) {
		l1 := newMockTxLayer[string, int]("L1", true)

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		tx, err := dal.BeginTx(ctx, WithIsolation(IsolationSerializable), WithReadOnly())
		if err != nil {
			t.Fatalf("BeginTx with options failed: %v", err)
		}

		if tx == nil {
			t.Fatal("BeginTx returned nil transaction")
		}

		tx.Rollback(ctx)
	})
}

// Additional edge case tests
func TestMultiLayerDAL_EdgeCases(t *testing.T) {
	t.Run("SetWithSkipLayers", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")
		l3 := newMockLayer[string, int]("L3")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
				{Name: "L3", Layer: l3},
			},
			WriteThrough: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Set with skip L2
		err := dal.Set(ctx, "key1", 100, WithSkipLayers("L2"))
		if err != nil {
			t.Fatalf("Set failed: %v", err)
		}

		// Verify in L1 and L3, but not L2
		_, err1 := l1.Get(ctx, "key1")
		_, err2 := l2.Get(ctx, "key1")
		_, err3 := l3.Get(ctx, "key1")

		if err1 != nil {
			t.Error("Key should be in L1")
		}
		if err2 != ErrNotFound {
			t.Error("Key should not be in L2 (skipped)")
		}
		if err3 != nil {
			t.Error("Key should be in L3")
		}
	})

	t.Run("SetAllLayersFail", func(t *testing.T) {
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

		// Make all layers fail
		l1.failSet = true
		l2.failSet = true

		err := dal.Set(ctx, "key1", 100)
		if err == nil {
			t.Error("Set should fail when all layers fail")
		}
	})

	t.Run("DeleteAllLayersFail", func(t *testing.T) {
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

		// Make all layers fail
		l1.failDel = true
		l2.failDel = true

		err := dal.Delete(ctx, "key1")
		if err == nil {
			t.Error("Delete should fail when all layers fail")
		}
	})

	t.Run("GetMultiAllMiss", func(t *testing.T) {
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

		keys := []string{"key1", "key2", "key3"}
		values, errors := dal.GetMulti(ctx, keys)

		if len(values) != 0 {
			t.Error("Expected no values for all miss")
		}
		if len(errors) != 3 {
			t.Errorf("Expected 3 errors, got %d", len(errors))
		}
		for _, key := range keys {
			if errors[key] != ErrNotFound {
				t.Errorf("Expected ErrNotFound for %s, got %v", key, errors[key])
			}
		}
	})

	t.Run("SetMultiAllFail", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
			WriteThrough: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Make layer fail
		l1.failSet = true

		items := map[string]int{
			"key1": 100,
			"key2": 200,
		}

		errors := dal.SetMulti(ctx, items)
		if len(errors) != 2 {
			t.Errorf("Expected 2 errors, got %d", len(errors))
		}
	})

	t.Run("NoWritableLayers", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1, ReadOnly: true},
			},
			WriteThrough: true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Try to set when all layers are read-only
		err := dal.Set(ctx, "key1", 100)
		if err == nil {
			t.Error("Expected error when no writable layers")
		}
	})
}

// Concurrent access tests
func TestMultiLayerDAL_Concurrent(t *testing.T) {
	t.Run("ConcurrentGetSet", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")
		l2 := newMockLayer[string, int]("L2")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
				{Name: "L2", Layer: l2},
			},
			EnablePromotion: true,
			WriteThrough:    true,
		}

		dal, _ := NewMultiLayer(config)
		defer dal.Close()

		ctx := context.Background()

		// Run concurrent operations
		var wg sync.WaitGroup
		concurrency := 10
		operations := 100

		for i := 0; i < concurrency; i++ {
			wg.Add(1)
			go func(id int) {
				defer wg.Done()
				for j := 0; j < operations; j++ {
					key := fmt.Sprintf("key%d", j%10)
					value := id*operations + j

					// Mix of operations
					switch j % 3 {
					case 0:
						dal.Set(ctx, key, value)
					case 1:
						dal.Get(ctx, key)
					case 2:
						dal.Delete(ctx, key)
					}
				}
			}(i)
		}

		wg.Wait()

		// Verify DAL still functional
		err := dal.Set(ctx, "test", 999)
		if err != nil {
			t.Errorf("DAL not functional after concurrent access: %v", err)
		}
	})

	t.Run("ConcurrentClose", func(t *testing.T) {
		l1 := newMockLayer[string, int]("L1")

		config := Config[string, int]{
			Layers: []LayerConfig[string, int]{
				{Name: "L1", Layer: l1},
			},
		}

		dal, _ := NewMultiLayer(config)

		// Try closing from multiple goroutines
		var wg sync.WaitGroup
		for i := 0; i < 10; i++ {
			wg.Add(1)
			go func() {
				defer wg.Done()
				dal.Close()
			}()
		}

		wg.Wait()

		// Verify closed
		ctx := context.Background()
		_, err := dal.Get(ctx, "key")
		if err != ErrClosed {
			t.Errorf("Expected ErrClosed after concurrent close, got %v", err)
		}
	})
}

// Table-driven configuration validation tests
func TestMultiLayerDAL_ConfigValidation(t *testing.T) {
	tests := []struct {
		name        string
		config      Config[string, int]
		expectError bool
		errorMsg    string
	}{
		{
			name: "Valid configuration",
			config: Config[string, int]{
				Layers: []LayerConfig[string, int]{
					{Name: "L1", Layer: newMockLayer[string, int]("L1")},
				},
			},
			expectError: false,
		},
		{
			name: "Empty layers",
			config: Config[string, int]{
				Layers: []LayerConfig[string, int]{},
			},
			expectError: true,
			errorMsg:    "at least one layer",
		},
		{
			name: "Nil layer",
			config: Config[string, int]{
				Layers: []LayerConfig[string, int]{
					{Name: "L1", Layer: nil},
				},
			},
			expectError: true,
			errorMsg:    "cannot be nil",
		},
		{
			name: "Empty layer name",
			config: Config[string, int]{
				Layers: []LayerConfig[string, int]{
					{Name: "", Layer: newMockLayer[string, int]("L1")},
				},
			},
			expectError: true,
			errorMsg:    "name cannot be empty",
		},
		{
			name: "Duplicate layer names",
			config: Config[string, int]{
				Layers: []LayerConfig[string, int]{
					{Name: "L1", Layer: newMockLayer[string, int]("L1")},
					{Name: "L1", Layer: newMockLayer[string, int]("L1")},
				},
			},
			expectError: true,
			errorMsg:    "duplicate",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, err := NewMultiLayer(tt.config)

			if tt.expectError {
				if err == nil {
					t.Errorf("Expected error containing '%s', got nil", tt.errorMsg)
				} else if tt.errorMsg != "" && !contains(err.Error(), tt.errorMsg) {
					t.Errorf("Expected error containing '%s', got '%s'", tt.errorMsg, err.Error())
				}
			} else {
				if err != nil {
					t.Errorf("Expected no error, got %v", err)
				}
			}
		})
	}
}

func contains(s, substr string) bool {
	return len(s) >= len(substr) && (s == substr || len(s) > len(substr) && (s[:len(substr)] == substr || s[len(s)-len(substr):] == substr || containsMiddle(s, substr)))
}

func containsMiddle(s, substr string) bool {
	for i := 0; i <= len(s)-len(substr); i++ {
		if s[i:i+len(substr)] == substr {
			return true
		}
	}
	return false
}
