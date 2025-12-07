// Package capacitor provides a high-performance Data Access Layer (DAL)
// with a zero-allocation philosophy for the WATT Toolkit.
package capacitor

import (
	"context"
	"errors"
)

// Common errors returned by DAL operations.
var (
	// ErrNotFound indicates the requested key was not found in any layer.
	ErrNotFound = errors.New("key not found")

	// ErrLayerNotFound indicates a specified layer does not exist.
	ErrLayerNotFound = errors.New("layer not found")

	// ErrClosed indicates the DAL has been closed and cannot be used.
	ErrClosed = errors.New("dal closed")

	// ErrInvalidKey indicates the provided key is invalid.
	ErrInvalidKey = errors.New("invalid key")

	// ErrInvalidValue indicates the provided value is invalid.
	ErrInvalidValue = errors.New("invalid value")

	// ErrTxNotSupported indicates the layer does not support transactions.
	ErrTxNotSupported = errors.New("transactions not supported")

	// ErrTxAlreadyStarted indicates a transaction is already in progress.
	ErrTxAlreadyStarted = errors.New("transaction already started")

	// ErrNoActiveTx indicates no transaction is currently active.
	ErrNoActiveTx = errors.New("no active transaction")

	// ErrIterationNotSupported indicates the layer does not support iteration.
	ErrIterationNotSupported = errors.New("iteration not supported")
)

// DAL provides generic data access operations across multiple storage layers.
// K must be comparable for map-based lookups.
// V can be any type.
//
// DAL supports multi-layer caching with automatic promotion and write-through semantics.
// Reads try layers in order (L1 -> L2 -> persistent), promoting hits to faster layers.
// Writes propagate to all configured layers.
type DAL[K comparable, V any] interface {
	// Get retrieves a value by key, trying each layer in order.
	// Returns ErrNotFound if the key doesn't exist in any layer.
	// On cache miss in upper layers but hit in lower layers, the value is promoted.
	Get(ctx context.Context, key K) (V, error)

	// Set stores a value across all layers with optional configuration.
	// Options can control TTL, size, and other layer-specific settings.
	Set(ctx context.Context, key K, value V, opts ...SetOption) error

	// Delete removes a value from all layers.
	// Returns nil if the key doesn't exist (idempotent operation).
	Delete(ctx context.Context, key K) error

	// Exists checks if a key exists in any layer without loading the value.
	// More efficient than Get when you only need to know if a key exists.
	Exists(ctx context.Context, key K) (bool, error)

	// GetFromLayer retrieves a value from a specific layer.
	// Returns ErrLayerNotFound if the layer doesn't exist.
	GetFromLayer(ctx context.Context, layer string, key K) (V, error)

	// Stats returns statistics for all layers.
	Stats() Stats

	// Batch Operations
	// These operations allow efficient processing of multiple keys at once.

	// GetMulti retrieves multiple values by their keys.
	// Returns a map of key to value for found keys and a map of key to error for failed retrievals.
	// This is more efficient than calling Get multiple times as it can batch requests to underlying layers.
	//
	// Example:
	//   keys := []K{"user:1", "user:2", "user:3"}
	//   values, errors := dal.GetMulti(ctx, keys)
	//   for key, value := range values {
	//       // Process found values
	//   }
	//   for key, err := range errors {
	//       // Handle errors (e.g., ErrNotFound)
	//   }
	GetMulti(ctx context.Context, keys []K) (map[K]V, map[K]error)

	// SetMulti stores multiple key-value pairs.
	// Returns a map of key to error for any failed Set operations.
	// Successful sets return nil error (not included in the map).
	// This is more efficient than calling Set multiple times as it can batch requests.
	//
	// Example:
	//   items := map[K]V{"user:1": user1, "user:2": user2}
	//   errors := dal.SetMulti(ctx, items)
	//   for key, err := range errors {
	//       log.Printf("Failed to set %v: %v", key, err)
	//   }
	SetMulti(ctx context.Context, items map[K]V, opts ...SetOption) map[K]error

	// DeleteMulti removes multiple keys.
	// Returns a map of key to error for any failed Delete operations.
	// Successful deletes return nil error (not included in the map).
	// This is idempotent - deleting non-existent keys is not an error.
	//
	// Example:
	//   keys := []K{"user:1", "user:2", "user:3"}
	//   errors := dal.DeleteMulti(ctx, keys)
	//   if len(errors) > 0 {
	//       // Handle failures
	//   }
	DeleteMulti(ctx context.Context, keys []K) map[K]error

	// Iteration Support
	// These operations allow efficient iteration over keys and values.
	// Not all layers may support iteration - check for ErrIterationNotSupported.

	// Range iterates over all key-value pairs in the DAL.
	// The function f is called for each key-value pair.
	// If f returns false, iteration stops.
	// Returns ErrIterationNotSupported if the underlying layer doesn't support iteration.
	//
	// Note: Range iterates over the first layer that supports iteration.
	// For multi-layer DALs, this typically means the persistent layer.
	//
	// Example:
	//   err := dal.Range(ctx, func(key K, value V) bool {
	//       fmt.Printf("%v: %v\n", key, value)
	//       return true // continue iteration
	//   })
	Range(ctx context.Context, f func(key K, value V) bool) error

	// Keys returns an iterator over all keys in the DAL.
	// The returned channel will be closed when iteration completes or context is cancelled.
	// Returns ErrIterationNotSupported if the underlying layer doesn't support iteration.
	//
	// Example:
	//   keys, err := dal.Keys(ctx)
	//   if err != nil {
	//       return err
	//   }
	//   for key := range keys {
	//       fmt.Println(key)
	//   }
	Keys(ctx context.Context) (<-chan K, error)

	// Values returns an iterator over all values in the DAL.
	// The returned channel will be closed when iteration completes or context is cancelled.
	// Returns ErrIterationNotSupported if the underlying layer doesn't support iteration.
	//
	// Note: Values without keys may be less useful - consider using Range instead.
	//
	// Example:
	//   values, err := dal.Values(ctx)
	//   if err != nil {
	//       return err
	//   }
	//   for value := range values {
	//       process(value)
	//   }
	Values(ctx context.Context) (<-chan V, error)

	// Transaction Support
	// These operations provide ACID transaction semantics where supported.
	// Not all layers support transactions - check for ErrTxNotSupported.

	// BeginTx starts a new transaction.
	// All subsequent operations will be part of this transaction until Commit or Rollback.
	// Returns ErrTxNotSupported if the underlying layer doesn't support transactions.
	// Returns ErrTxAlreadyStarted if a transaction is already in progress.
	//
	// Transactions provide:
	// - Atomicity: All operations succeed or all fail
	// - Consistency: Data remains valid
	// - Isolation: Concurrent transactions don't interfere
	// - Durability: Committed changes are persistent
	//
	// Example:
	//   tx, err := dal.BeginTx(ctx, TxOptions{Isolation: IsolationSerializable})
	//   if err != nil {
	//       return err
	//   }
	//   defer tx.Rollback(ctx) // Rollback if not committed
	//
	//   if err := tx.Set(ctx, "key", value); err != nil {
	//       return err
	//   }
	//   return tx.Commit(ctx)
	BeginTx(ctx context.Context, opts ...TxOption) (Tx[K, V], error)

	// Close releases all resources held by the DAL and its layers.
	// After Close, all operations will return ErrClosed.
	Close() error
}

// Layer represents a single storage tier in the DAL.
// Implementations must be thread-safe.
type Layer[K comparable, V any] interface {
	// Basic Operations

	// Get retrieves a value from this layer.
	Get(ctx context.Context, key K) (V, error)

	// Set stores a value in this layer.
	Set(ctx context.Context, key K, value V, opts ...SetOption) error

	// Delete removes a value from this layer.
	Delete(ctx context.Context, key K) error

	// Exists checks if a key exists in this layer.
	Exists(ctx context.Context, key K) (bool, error)

	// Stats returns statistics for this layer.
	Stats() LayerStats

	// Close releases resources held by this layer.
	Close() error

	// Batch Operations (optional - implement if the layer supports batching)

	// GetMulti retrieves multiple values. Returns nil if not supported.
	GetMulti(ctx context.Context, keys []K) (map[K]V, map[K]error)

	// SetMulti stores multiple values. Returns nil if not supported.
	SetMulti(ctx context.Context, items map[K]V, opts ...SetOption) map[K]error

	// DeleteMulti removes multiple keys. Returns nil if not supported.
	DeleteMulti(ctx context.Context, keys []K) map[K]error

	// Iteration (optional - implement if the layer supports iteration)

	// Range iterates over all key-value pairs. Returns ErrIterationNotSupported if not supported.
	Range(ctx context.Context, f func(key K, value V) bool) error

	// Transaction Support (optional - implement if the layer supports transactions)

	// BeginTx starts a transaction. Returns ErrTxNotSupported if not supported.
	BeginTx(ctx context.Context, opts ...TxOption) (Tx[K, V], error)
}

// Stats contains aggregate statistics across all DAL layers.
type Stats struct {
	// Layers maps layer name to its statistics.
	Layers map[string]LayerStats

	// TotalHits is the sum of all layer hits.
	TotalHits uint64

	// TotalMisses is the sum of all layer misses.
	TotalMisses uint64

	// HitRate is the overall cache hit rate (0.0 to 1.0).
	HitRate float64
}

// LayerStats contains statistics for a single layer.
type LayerStats struct {
	// Name of this layer (e.g., "L1", "L2", "persistent").
	Name string

	// Hits is the number of successful Get operations.
	Hits uint64

	// Misses is the number of failed Get operations.
	Misses uint64

	// Sets is the number of Set operations.
	Sets uint64

	// Deletes is the number of Delete operations.
	Deletes uint64

	// Evictions is the number of items evicted (if applicable).
	Evictions uint64

	// Size is the current number of items in this layer.
	Size int64

	// Bytes is the approximate memory/disk usage in bytes.
	Bytes int64

	// HitRate is the cache hit rate for this layer (0.0 to 1.0).
	HitRate float64
}

// SetOption is a function that configures Set operations.
type SetOption func(*SetOptions)

// SetOptions contains configuration for Set operations.
type SetOptions struct {
	// TTL specifies how long the value should be retained.
	// Zero means no expiration (use layer default).
	TTL int64 // nanoseconds

	// Size is the approximate size of the value in bytes.
	// Used for size-based eviction policies.
	Size int64

	// Priority affects eviction order (higher = keep longer).
	// Interpretation is implementation-specific.
	Priority int

	// SkipLayers specifies layers to skip for this operation.
	SkipLayers []string
}

// WithTTL sets the time-to-live for a value.
func WithTTL(ttl int64) SetOption {
	return func(opts *SetOptions) {
		opts.TTL = ttl
	}
}

// WithSize sets the size hint for a value.
func WithSize(size int64) SetOption {
	return func(opts *SetOptions) {
		opts.Size = size
	}
}

// WithPriority sets the eviction priority for a value.
func WithPriority(priority int) SetOption {
	return func(opts *SetOptions) {
		opts.Priority = priority
	}
}

// WithSkipLayers specifies layers to skip for this operation.
func WithSkipLayers(layers ...string) SetOption {
	return func(opts *SetOptions) {
		opts.SkipLayers = append(opts.SkipLayers, layers...)
	}
}

// defaultSetOptions returns the default SetOptions.
func defaultSetOptions() SetOptions {
	return SetOptions{
		TTL:      0,
		Size:     0,
		Priority: 0,
	}
}

// Tx represents a database transaction.
// All operations within a transaction are atomic - they all succeed or all fail.
// Transactions must be explicitly committed or rolled back.
//
// Example usage:
//
//	tx, err := dal.BeginTx(ctx)
//	if err != nil {
//	    return err
//	}
//	defer tx.Rollback(ctx) // Rollback if not committed
//
//	if err := tx.Set(ctx, "key1", value1); err != nil {
//	    return err
//	}
//	if err := tx.Set(ctx, "key2", value2); err != nil {
//	    return err
//	}
//
//	return tx.Commit(ctx) // Commit all changes
type Tx[K comparable, V any] interface {
	// Get retrieves a value within the transaction.
	// Reads see writes made earlier in this transaction.
	Get(ctx context.Context, key K) (V, error)

	// Set stores a value within the transaction.
	// Changes are not visible to other transactions until Commit.
	Set(ctx context.Context, key K, value V, opts ...SetOption) error

	// Delete removes a value within the transaction.
	// Deletions are not visible to other transactions until Commit.
	Delete(ctx context.Context, key K) error

	// Exists checks if a key exists within the transaction.
	// Sees changes made earlier in this transaction.
	Exists(ctx context.Context, key K) (bool, error)

	// Batch operations within transaction

	// GetMulti retrieves multiple values within the transaction.
	GetMulti(ctx context.Context, keys []K) (map[K]V, map[K]error)

	// SetMulti stores multiple values within the transaction.
	SetMulti(ctx context.Context, items map[K]V, opts ...SetOption) map[K]error

	// DeleteMulti removes multiple keys within the transaction.
	DeleteMulti(ctx context.Context, keys []K) map[K]error

	// Commit commits the transaction, making all changes permanent.
	// After Commit, the transaction cannot be used.
	// Returns ErrClosed if the transaction was already committed or rolled back.
	Commit(ctx context.Context) error

	// Rollback rolls back the transaction, discarding all changes.
	// After Rollback, the transaction cannot be used.
	// Rollback is safe to call multiple times (idempotent).
	Rollback(ctx context.Context) error

	// ID returns a unique identifier for this transaction.
	// Useful for logging and debugging.
	ID() string
}

// IsolationLevel defines the isolation level for transactions.
// Higher isolation levels provide stronger guarantees but may reduce concurrency.
type IsolationLevel int

const (
	// IsolationDefault uses the layer's default isolation level.
	IsolationDefault IsolationLevel = iota

	// IsolationReadUncommitted allows dirty reads (uncommitted changes from other transactions).
	// Highest concurrency, lowest consistency.
	IsolationReadUncommitted

	// IsolationReadCommitted prevents dirty reads but allows non-repeatable reads.
	// A transaction sees only committed changes.
	IsolationReadCommitted

	// IsolationRepeatableRead prevents dirty and non-repeatable reads.
	// Reads within a transaction see a consistent snapshot.
	IsolationRepeatableRead

	// IsolationSerializable provides full isolation - transactions appear to execute serially.
	// Lowest concurrency, highest consistency.
	IsolationSerializable
)

// String returns the string representation of the isolation level.
func (i IsolationLevel) String() string {
	switch i {
	case IsolationDefault:
		return "Default"
	case IsolationReadUncommitted:
		return "ReadUncommitted"
	case IsolationReadCommitted:
		return "ReadCommitted"
	case IsolationRepeatableRead:
		return "RepeatableRead"
	case IsolationSerializable:
		return "Serializable"
	default:
		return "Unknown"
	}
}

// TxOption configures transaction behavior.
type TxOption func(*TxOptions)

// TxOptions contains configuration for transactions.
type TxOptions struct {
	// Isolation specifies the transaction isolation level.
	// Default is IsolationDefault (uses layer's default).
	Isolation IsolationLevel

	// ReadOnly indicates this is a read-only transaction.
	// Read-only transactions may have better performance.
	ReadOnly bool

	// Timeout specifies the maximum duration for the transaction.
	// Zero means no timeout (use context for timeouts instead).
	Timeout int64 // nanoseconds
}

// WithIsolation sets the transaction isolation level.
//
// Example:
//
//	tx, err := dal.BeginTx(ctx, WithIsolation(IsolationSerializable))
func WithIsolation(level IsolationLevel) TxOption {
	return func(opts *TxOptions) {
		opts.Isolation = level
	}
}

// WithReadOnly marks the transaction as read-only.
// Read-only transactions cannot perform writes but may have better performance.
//
// Example:
//
//	tx, err := dal.BeginTx(ctx, WithReadOnly())
func WithReadOnly() TxOption {
	return func(opts *TxOptions) {
		opts.ReadOnly = true
	}
}

// WithTimeout sets the transaction timeout.
//
// Example:
//
//	tx, err := dal.BeginTx(ctx, WithTimeout(30*time.Second))
func WithTimeout(timeout int64) TxOption {
	return func(opts *TxOptions) {
		opts.Timeout = timeout
	}
}

// defaultTxOptions returns the default transaction options.
func defaultTxOptions() TxOptions {
	return TxOptions{
		Isolation: IsolationDefault,
		ReadOnly:  false,
		Timeout:   0,
	}
}

// BatchResult represents the result of a batch operation.
// It allows checking success/failure for each key without allocating error maps when all succeed.
type BatchResult[K comparable] struct {
	// Errors maps keys to their errors. Nil if all operations succeeded.
	Errors map[K]error

	// SuccessCount is the number of successful operations.
	SuccessCount int

	// FailureCount is the number of failed operations.
	FailureCount int
}

// OK returns true if all operations in the batch succeeded.
func (br *BatchResult[K]) OK() bool {
	return br.FailureCount == 0
}

// Error returns the first error encountered, or nil if all succeeded.
// This allows BatchResult to satisfy the error interface for simple error checking.
func (br *BatchResult[K]) Error() error {
	if br.Errors == nil || len(br.Errors) == 0 {
		return nil
	}
	// Return first error found
	for _, err := range br.Errors {
		if err != nil {
			return err
		}
	}
	return nil
}
