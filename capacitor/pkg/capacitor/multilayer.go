package capacitor

import (
	"context"
	"fmt"
	"sync"
)

// MultiLayerDAL implements the DAL interface with support for multiple storage layers.
// It provides automatic cache promotion, write-through semantics, and graceful error handling.
//
// Performance characteristics:
//   - Get: O(n) worst case where n = number of layers (tries each layer in order)
//   - Set: O(n) for write-through (writes to all layers)
//   - Delete: O(n) (deletes from all layers)
//   - Thread-safe with RWMutex for closed state
//
// Architecture:
//   - Layers are tried in order (L1 -> L2 -> ... -> persistent)
//   - Cache hits in slower layers are promoted to faster layers
//   - Writes propagate to all non-read-only layers
//   - Layer failures don't stop operations (graceful degradation)
type MultiLayerDAL[K comparable, V any] struct {
	config Config[K, V]
	closed bool
	mu     sync.RWMutex
}

// NewMultiLayer creates a new multi-layer DAL from the given configuration.
// Returns an error if the configuration is invalid.
//
// Example:
//
//	config := Config[string, User]{
//	    Layers: []LayerConfig[string, User]{
//	        {Name: "L1", Layer: memoryCache, TTL: 1 * time.Minute},
//	        {Name: "L2", Layer: redisCache, TTL: 1 * time.Hour},
//	        {Name: "persistent", Layer: database, TTL: 0},
//	    },
//	    EnableMetrics:   true,
//	    EnablePromotion: true,
//	    WriteThrough:    true,
//	}
//	dal, err := NewMultiLayer(config)
func NewMultiLayer[K comparable, V any](config Config[K, V]) (*MultiLayerDAL[K, V], error) {
	if err := config.Validate(); err != nil {
		return nil, fmt.Errorf("invalid config: %w", err)
	}

	return &MultiLayerDAL[K, V]{
		config: config,
		closed: false,
	}, nil
}

// Get retrieves a value by trying each layer in order.
// If found in a slower layer and promotion is enabled, the value is promoted to faster layers.
//
// Returns:
//   - The value if found in any layer
//   - ErrNotFound if not found in any layer
//   - ErrClosed if the DAL is closed
//
// Performance: O(n) where n is the number of layers before a hit.
// With promotion enabled, subsequent Gets will be O(1) from L1.
func (d *MultiLayerDAL[K, V]) Get(ctx context.Context, key K) (V, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		var zero V
		return zero, ErrClosed
	}
	d.mu.RUnlock()

	// Try each layer in order
	for i, layerCfg := range d.config.Layers {
		value, err := layerCfg.Layer.Get(ctx, key)
		if err == nil {
			// Found! Promote to faster layers if enabled
			if d.config.EnablePromotion && i > 0 {
				d.promoteValue(ctx, key, value, i)
			}
			return value, nil
		}

		// Continue to next layer on not found
		if err == ErrNotFound {
			continue
		}

		// For other errors, log but continue trying remaining layers
		// This provides graceful degradation if a layer is down
	}

	var zero V
	return zero, ErrNotFound
}

// promoteValue writes a value to all faster layers (L1, L2, ..., up to foundAtIndex).
// This is called when a cache miss in upper layers but hit in a lower layer occurs.
// Errors during promotion are silently ignored to avoid disrupting the Get operation.
//
//go:inline
func (d *MultiLayerDAL[K, V]) promoteValue(ctx context.Context, key K, value V, foundAtIndex int) {
	// Write to all faster layers
	for i := 0; i < foundAtIndex; i++ {
		layerCfg := d.config.Layers[i]

		// Skip read-only layers
		if layerCfg.ReadOnly {
			continue
		}

		// Build set options with layer-specific TTL
		var opts []SetOption
		if layerCfg.TTL > 0 {
			opts = append(opts, WithTTL(int64(layerCfg.TTL)))
		}

		// Best effort - ignore errors during promotion
		_ = layerCfg.Layer.Set(ctx, key, value, opts...)
	}
}

// Set stores a value across all layers (or only L1 if write-through is disabled).
// Returns the first error encountered, but attempts to write to all layers.
//
// Behavior:
//   - WriteThrough=true: writes to all non-read-only layers
//   - WriteThrough=false: writes only to first layer (L1)
//   - Layer failures are collected but don't stop subsequent writes
//
// Returns:
//   - nil if at least one layer succeeded
//   - Error if all layers failed or DAL is closed
//
// Performance: O(n) for write-through, O(1) otherwise.
func (d *MultiLayerDAL[K, V]) Set(ctx context.Context, key K, value V, opts ...SetOption) error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return ErrClosed
	}
	d.mu.RUnlock()

	// Parse options
	options := defaultSetOptions()
	for _, opt := range opts {
		opt(&options)
	}

	// Build skip set for efficient lookup
	skipSet := make(map[string]bool, len(options.SkipLayers))
	for _, name := range options.SkipLayers {
		skipSet[name] = true
	}

	var firstErr error
	successCount := 0

	// Write-through to all layers or just L1
	layerCount := 1
	if d.config.WriteThrough {
		layerCount = len(d.config.Layers)
	}

	for i := 0; i < layerCount; i++ {
		layerCfg := d.config.Layers[i]

		// Skip read-only layers
		if layerCfg.ReadOnly {
			continue
		}

		// Skip layers in skip list
		if skipSet[layerCfg.Name] {
			continue
		}

		// Build layer-specific options
		layerOpts := make([]SetOption, 0, len(opts)+1)
		layerOpts = append(layerOpts, opts...)

		// Apply layer-specific TTL if not overridden
		if options.TTL == 0 && layerCfg.TTL > 0 {
			layerOpts = append(layerOpts, WithTTL(int64(layerCfg.TTL)))
		}

		// Attempt write
		err := layerCfg.Layer.Set(ctx, key, value, layerOpts...)
		if err == nil {
			successCount++
		} else if firstErr == nil {
			firstErr = err
		}
	}

	// Success if at least one layer succeeded
	if successCount > 0 {
		return nil
	}

	// All layers failed
	if firstErr != nil {
		return fmt.Errorf("set failed on all layers: %w", firstErr)
	}

	return fmt.Errorf("no writable layers available")
}

// Delete removes a value from all layers.
// Returns nil even if some layers fail, as long as at least one succeeds.
// This provides idempotent delete semantics.
//
// Returns:
//   - nil if at least one layer succeeded or key doesn't exist
//   - Error if all layers failed
//   - ErrClosed if DAL is closed
//
// Performance: O(n) where n = number of layers.
func (d *MultiLayerDAL[K, V]) Delete(ctx context.Context, key K) error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return ErrClosed
	}
	d.mu.RUnlock()

	var firstErr error
	successCount := 0
	notFoundCount := 0

	// Delete from all layers
	for _, layerCfg := range d.config.Layers {
		// Skip read-only layers
		if layerCfg.ReadOnly {
			continue
		}

		err := layerCfg.Layer.Delete(ctx, key)
		if err == nil || err == ErrNotFound {
			successCount++
			if err == ErrNotFound {
				notFoundCount++
			}
		} else if firstErr == nil {
			firstErr = err
		}
	}

	// Success if at least one layer succeeded
	if successCount > 0 {
		return nil
	}

	// All layers failed
	if firstErr != nil {
		return fmt.Errorf("delete failed on all layers: %w", firstErr)
	}

	return nil
}

// Exists checks if a key exists in any layer.
// Returns true if found in any layer.
//
// Performance: O(n) worst case, O(1) best case if in L1.
func (d *MultiLayerDAL[K, V]) Exists(ctx context.Context, key K) (bool, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return false, ErrClosed
	}
	d.mu.RUnlock()

	// Check each layer
	for _, layerCfg := range d.config.Layers {
		exists, err := layerCfg.Layer.Exists(ctx, key)
		if err == nil && exists {
			return true, nil
		}
		// Continue on error or not found
	}

	return false, nil
}

// GetFromLayer retrieves a value from a specific layer by name.
// Returns ErrLayerNotFound if the layer doesn't exist.
//
// This is useful for debugging or when you need to check a specific layer.
func (d *MultiLayerDAL[K, V]) GetFromLayer(ctx context.Context, layer string, key K) (V, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		var zero V
		return zero, ErrClosed
	}
	d.mu.RUnlock()

	// Find the layer
	for _, layerCfg := range d.config.Layers {
		if layerCfg.Name == layer {
			return layerCfg.Layer.Get(ctx, key)
		}
	}

	var zero V
	return zero, ErrLayerNotFound
}

// Stats returns aggregate statistics across all layers.
//
// The returned Stats includes:
//   - Per-layer statistics in the Layers map
//   - Total hits/misses across all layers
//   - Overall hit rate
func (d *MultiLayerDAL[K, V]) Stats() Stats {
	d.mu.RLock()
	defer d.mu.RUnlock()

	stats := Stats{
		Layers:      make(map[string]LayerStats, len(d.config.Layers)),
		TotalHits:   0,
		TotalMisses: 0,
		HitRate:     0,
	}

	// Collect stats from each layer
	for _, layerCfg := range d.config.Layers {
		layerStats := layerCfg.Layer.Stats()
		stats.Layers[layerCfg.Name] = layerStats

		stats.TotalHits += layerStats.Hits
		stats.TotalMisses += layerStats.Misses
	}

	// Calculate overall hit rate
	total := stats.TotalHits + stats.TotalMisses
	if total > 0 {
		stats.HitRate = float64(stats.TotalHits) / float64(total)
	}

	return stats
}

// GetMulti retrieves multiple values efficiently.
// Returns a map of found values and a map of errors for failed retrievals.
//
// For each key:
//   - Tries layers in order (L1 -> L2 -> persistent)
//   - Promotes to faster layers if found in slower layer
//   - Returns ErrNotFound if not found in any layer
//
// This is more efficient than calling Get multiple times if the underlying
// layers support batch operations.
func (d *MultiLayerDAL[K, V]) GetMulti(ctx context.Context, keys []K) (map[K]V, map[K]error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		errors := make(map[K]error, len(keys))
		for _, key := range keys {
			errors[key] = ErrClosed
		}
		return nil, errors
	}
	d.mu.RUnlock()

	values := make(map[K]V, len(keys))
	errors := make(map[K]error)

	// Try to use batch operations if supported
	// Otherwise fall back to individual Gets
	remaining := make([]K, 0, len(keys))
	for _, key := range keys {
		remaining = append(remaining, key)
	}

	// Try each layer in order
	for i, layerCfg := range d.config.Layers {
		if len(remaining) == 0 {
			break
		}

		// Try batch operation first
		batchValues, _ := layerCfg.Layer.GetMulti(ctx, remaining)

		if batchValues != nil {
			// Layer supports batching
			for k, v := range batchValues {
				values[k] = v

				// Promote to faster layers if enabled
				if d.config.EnablePromotion && i > 0 {
					d.promoteValue(ctx, k, v, i)
				}
			}

			// Update remaining keys
			newRemaining := make([]K, 0, len(remaining))
			for _, key := range remaining {
				if _, found := batchValues[key]; !found {
					newRemaining = append(newRemaining, key)
				}
			}
			remaining = newRemaining
		} else {
			// Fall back to individual Gets
			newRemaining := make([]K, 0, len(remaining))
			for _, key := range remaining {
				value, err := layerCfg.Layer.Get(ctx, key)
				if err == nil {
					values[key] = value

					// Promote if enabled
					if d.config.EnablePromotion && i > 0 {
						d.promoteValue(ctx, key, value, i)
					}
				} else if err != ErrNotFound {
					// Store non-NotFound errors but continue
					newRemaining = append(newRemaining, key)
				} else {
					newRemaining = append(newRemaining, key)
				}
			}
			remaining = newRemaining
		}
	}

	// Any remaining keys were not found
	for _, key := range remaining {
		errors[key] = ErrNotFound
	}

	return values, errors
}

// SetMulti stores multiple key-value pairs efficiently.
// Returns a map of errors for any failed operations.
//
// Behavior:
//   - Attempts to use batch operations if supported by layers
//   - Falls back to individual Sets if batching not supported
//   - Respects WriteThrough configuration
//   - Collects errors but doesn't fail fast
func (d *MultiLayerDAL[K, V]) SetMulti(ctx context.Context, items map[K]V, opts ...SetOption) map[K]error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		errors := make(map[K]error, len(items))
		for k := range items {
			errors[k] = ErrClosed
		}
		return errors
	}
	d.mu.RUnlock()

	errors := make(map[K]error)

	// Determine layers to write to
	layerCount := 1
	if d.config.WriteThrough {
		layerCount = len(d.config.Layers)
	}

	// Track successes per key
	successes := make(map[K]int, len(items))

	for i := 0; i < layerCount; i++ {
		layerCfg := d.config.Layers[i]

		// Skip read-only layers
		if layerCfg.ReadOnly {
			continue
		}

		// Build layer-specific options
		layerOpts := make([]SetOption, 0, len(opts)+1)
		layerOpts = append(layerOpts, opts...)

		// Apply layer-specific TTL if not overridden
		options := defaultSetOptions()
		for _, opt := range opts {
			opt(&options)
		}
		if options.TTL == 0 && layerCfg.TTL > 0 {
			layerOpts = append(layerOpts, WithTTL(int64(layerCfg.TTL)))
		}

		// Try batch operation first
		layerBatchErrors := layerCfg.Layer.SetMulti(ctx, items, layerOpts...)

		if layerBatchErrors != nil {
			// Layer supports batching
			for k := range items {
				if _, hasError := layerBatchErrors[k]; !hasError {
					successes[k]++
				}
			}
		} else {
			// Fall back to individual Sets
			for k, v := range items {
				err := layerCfg.Layer.Set(ctx, k, v, layerOpts...)
				if err == nil {
					successes[k]++
				} else if i == 0 {
					// Store first layer errors
					errors[k] = err
				}
			}
		}
	}

	// Build final error map (only keys that failed on all layers)
	finalErrors := make(map[K]error)
	for k := range items {
		if successes[k] == 0 {
			if err, exists := errors[k]; exists {
				finalErrors[k] = err
			} else {
				finalErrors[k] = fmt.Errorf("set failed on all layers")
			}
		}
	}

	return finalErrors
}

// DeleteMulti removes multiple keys efficiently.
// Returns a map of errors for any failed operations.
//
// This is idempotent - deleting non-existent keys is not an error.
func (d *MultiLayerDAL[K, V]) DeleteMulti(ctx context.Context, keys []K) map[K]error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		errors := make(map[K]error, len(keys))
		for _, key := range keys {
			errors[key] = ErrClosed
		}
		return errors
	}
	d.mu.RUnlock()

	errors := make(map[K]error)
	successes := make(map[K]int, len(keys))

	// Delete from all layers
	for _, layerCfg := range d.config.Layers {
		// Skip read-only layers
		if layerCfg.ReadOnly {
			continue
		}

		// Try batch operation first
		batchErrors := layerCfg.Layer.DeleteMulti(ctx, keys)

		if batchErrors != nil {
			// Layer supports batching
			for _, k := range keys {
				if _, hasError := batchErrors[k]; !hasError {
					successes[k]++
				}
			}
		} else {
			// Fall back to individual Deletes
			for _, k := range keys {
				err := layerCfg.Layer.Delete(ctx, k)
				if err == nil || err == ErrNotFound {
					successes[k]++
				} else if successes[k] == 0 {
					errors[k] = err
				}
			}
		}
	}

	// Build final error map (only keys that failed on all layers)
	finalErrors := make(map[K]error)
	for _, k := range keys {
		if successes[k] == 0 {
			if err, exists := errors[k]; exists {
				finalErrors[k] = err
			} else {
				finalErrors[k] = fmt.Errorf("delete failed on all layers")
			}
		}
	}

	return finalErrors
}

// Range iterates over all key-value pairs in the DAL.
// Uses the first layer that supports iteration (typically the persistent layer).
//
// Returns ErrIterationNotSupported if no layer supports iteration.
func (d *MultiLayerDAL[K, V]) Range(ctx context.Context, f func(key K, value V) bool) error {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return ErrClosed
	}
	d.mu.RUnlock()

	// Try each layer until we find one that supports iteration
	// Typically this will be the persistent layer
	for _, layerCfg := range d.config.Layers {
		err := layerCfg.Layer.Range(ctx, f)
		if err == nil {
			return nil
		}
		if err != ErrIterationNotSupported {
			return err
		}
	}

	return ErrIterationNotSupported
}

// Keys returns a channel of all keys in the DAL.
// Uses the first layer that supports iteration.
//
// The returned channel will be closed when iteration completes or context is cancelled.
func (d *MultiLayerDAL[K, V]) Keys(ctx context.Context) (<-chan K, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return nil, ErrClosed
	}
	d.mu.RUnlock()

	ch := make(chan K)

	// Find a layer that supports iteration
	var iterLayer Layer[K, V]
	for _, layerCfg := range d.config.Layers {
		err := layerCfg.Layer.Range(ctx, func(k K, v V) bool { return false })
		if err != ErrIterationNotSupported {
			iterLayer = layerCfg.Layer
			break
		}
	}

	if iterLayer == nil {
		close(ch)
		return ch, ErrIterationNotSupported
	}

	// Start iteration in background
	go func() {
		defer close(ch)
		_ = iterLayer.Range(ctx, func(k K, v V) bool {
			select {
			case ch <- k:
				return true
			case <-ctx.Done():
				return false
			}
		})
	}()

	return ch, nil
}

// Values returns a channel of all values in the DAL.
// Uses the first layer that supports iteration.
//
// The returned channel will be closed when iteration completes or context is cancelled.
func (d *MultiLayerDAL[K, V]) Values(ctx context.Context) (<-chan V, error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return nil, ErrClosed
	}
	d.mu.RUnlock()

	ch := make(chan V)

	// Find a layer that supports iteration
	var iterLayer Layer[K, V]
	for _, layerCfg := range d.config.Layers {
		err := layerCfg.Layer.Range(ctx, func(k K, v V) bool { return false })
		if err != ErrIterationNotSupported {
			iterLayer = layerCfg.Layer
			break
		}
	}

	if iterLayer == nil {
		close(ch)
		return ch, ErrIterationNotSupported
	}

	// Start iteration in background
	go func() {
		defer close(ch)
		_ = iterLayer.Range(ctx, func(k K, v V) bool {
			select {
			case ch <- v:
				return true
			case <-ctx.Done():
				return false
			}
		})
	}()

	return ch, nil
}

// BeginTx starts a transaction on the first layer that supports transactions.
// Typically this will be the persistent storage layer.
//
// Returns ErrTxNotSupported if no layer supports transactions.
//
// Note: Transactions only apply to the persistent layer, not cache layers.
// Cache invalidation during rollback is not automatic.
func (d *MultiLayerDAL[K, V]) BeginTx(ctx context.Context, opts ...TxOption) (Tx[K, V], error) {
	d.mu.RLock()
	if d.closed {
		d.mu.RUnlock()
		return nil, ErrClosed
	}
	d.mu.RUnlock()

	// Try each layer until we find one that supports transactions
	for _, layerCfg := range d.config.Layers {
		tx, err := layerCfg.Layer.BeginTx(ctx, opts...)
		if err == nil {
			return tx, nil
		}
		if err != ErrTxNotSupported {
			return nil, err
		}
	}

	return nil, ErrTxNotSupported
}

// Close closes all layers and marks the DAL as closed.
// After Close, all operations will return ErrClosed.
//
// Errors from closing individual layers are collected and returned.
// The DAL is marked as closed even if some layers fail to close.
func (d *MultiLayerDAL[K, V]) Close() error {
	d.mu.Lock()
	defer d.mu.Unlock()

	if d.closed {
		return ErrClosed
	}

	d.closed = true

	var firstErr error

	// Close all layers
	for _, layerCfg := range d.config.Layers {
		if err := layerCfg.Layer.Close(); err != nil && firstErr == nil {
			firstErr = err
		}
	}

	return firstErr
}
