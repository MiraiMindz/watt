---
name: dal-generator
description: Generate high-performance DAL (Data Access Layer) implementations following Capacitor's zero-allocation philosophy. Use when creating new DAL interfaces, implementing cache layers, database backends, or DTO structures. Optimizes for CPU and memory efficiency with pooling patterns and generic type safety.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# DAL Generator Skill

This skill helps you generate high-performance Data Access Layer (DAL) implementations for the Capacitor project, following the zero-allocation philosophy and performance-first principles.

## Core Principles

When generating DAL code, always follow these principles:
1. **Zero Allocation in Hot Paths** - Use sync.Pool, reuse buffers, minimize heap allocations
2. **Generic Type Safety** - Leverage Go generics to eliminate type assertions and reflection
3. **Benchmark-Driven** - Include benchmarks with every implementation
4. **Composable Layers** - Each layer should be independent and swappable
5. **Context-Aware** - All operations accept context.Context for cancellation

## Quick Start Templates

### 1. Basic DAL Interface
Use this template when creating a new DAL interface:

```go
package capacitor

import "context"

// DAL provides generic data access operations.
// K must be comparable for map-based lookups.
// V can be any type.
type DAL[K comparable, V any] interface {
    // Get retrieves a value by key.
    // Returns ErrNotFound if key doesn't exist.
    Get(ctx context.Context, key K) (V, error)

    // Set stores a value with optional configuration.
    Set(ctx context.Context, key K, value V, opts ...SetOption) error

    // Delete removes a value by key.
    // Returns nil if key doesn't exist (idempotent).
    Delete(ctx context.Context, key K) error

    // Exists checks if a key exists without loading the value.
    Exists(ctx context.Context, key K) (bool, error)

    // Close releases resources.
    Close() error
}
```

### 2. Memory Cache Layer Template
Use this for in-memory caching implementations:

```go
package memory

import (
    "context"
    "sync"
    "time"
)

// Cache implements a high-performance in-memory cache with zero-allocation reads.
type Cache[K comparable, V any] struct {
    mu      sync.RWMutex
    data    map[K]*entry[V]
    pool    *sync.Pool
    maxSize int64

    // Metrics
    hits   uint64
    misses uint64
}

type entry[V any] struct {
    value     V
    expiresAt time.Time
    size      int64
}

// NewCache creates a new memory cache.
func NewCache[K comparable, V any](maxSize int64) *Cache[K, V] {
    return &Cache[K, V]{
        data:    make(map[K]*entry[V]),
        maxSize: maxSize,
        pool: &sync.Pool{
            New: func() interface{} {
                return new(entry[V])
            },
        },
    }
}

// Get retrieves a value from cache.
// Zero allocations for cache hits.
func (c *Cache[K, V]) Get(ctx context.Context, key K) (V, error) {
    c.mu.RLock()
    e, ok := c.data[key]
    c.mu.RUnlock()

    if !ok {
        atomic.AddUint64(&c.misses, 1)
        var zero V
        return zero, ErrNotFound
    }

    // Check expiration
    if !e.expiresAt.IsZero() && time.Now().After(e.expiresAt) {
        c.mu.Lock()
        delete(c.data, key)
        c.returnEntry(e)
        c.mu.Unlock()

        atomic.AddUint64(&c.misses, 1)
        var zero V
        return zero, ErrNotFound
    }

    atomic.AddUint64(&c.hits, 1)
    return e.value, nil
}

// Set stores a value in cache.
func (c *Cache[K, V]) Set(ctx context.Context, key K, value V, opts ...SetOption) error {
    options := defaultSetOptions()
    for _, opt := range opts {
        opt(&options)
    }

    e := c.getEntry()
    e.value = value
    if options.ttl > 0 {
        e.expiresAt = time.Now().Add(options.ttl)
    }
    e.size = options.size

    c.mu.Lock()
    if old, exists := c.data[key]; exists {
        c.returnEntry(old)
    }
    c.data[key] = e
    c.mu.Unlock()

    return nil
}

func (c *Cache[K, V]) getEntry() *entry[V] {
    return c.pool.Get().(*entry[V])
}

func (c *Cache[K, V]) returnEntry(e *entry[V]) {
    var zero V
    e.value = zero
    e.expiresAt = time.Time{}
    e.size = 0
    c.pool.Put(e)
}
```

### 3. Database Backend Template
Use this for database layer implementations:

```go
package database

import (
    "context"
    "database/sql"
    "fmt"
)

// SQLBackend provides a generic SQL database backend.
type SQLBackend[K comparable, V any] struct {
    db         *sql.DB
    tableName  string
    serializer Serializer[V]

    // Prepared statements
    getStmt    *sql.Stmt
    setStmt    *sql.Stmt
    deleteStmt *sql.Stmt
    existsStmt *sql.Stmt
}

// NewSQLBackend creates a SQL backend with prepared statements.
func NewSQLBackend[K comparable, V any](
    db *sql.DB,
    tableName string,
    serializer Serializer[V],
) (*SQLBackend[K, V], error) {
    backend := &SQLBackend[K, V]{
        db:         db,
        tableName:  tableName,
        serializer: serializer,
    }

    if err := backend.prepare(); err != nil {
        return nil, fmt.Errorf("failed to prepare statements: %w", err)
    }

    return backend, nil
}

func (b *SQLBackend[K, V]) prepare() error {
    var err error

    b.getStmt, err = b.db.Prepare(
        fmt.Sprintf("SELECT value FROM %s WHERE key = $1", b.tableName),
    )
    if err != nil {
        return err
    }

    b.setStmt, err = b.db.Prepare(
        fmt.Sprintf(
            "INSERT INTO %s (key, value) VALUES ($1, $2) ON CONFLICT (key) DO UPDATE SET value = $2",
            b.tableName,
        ),
    )
    if err != nil {
        return err
    }

    b.deleteStmt, err = b.db.Prepare(
        fmt.Sprintf("DELETE FROM %s WHERE key = $1", b.tableName),
    )
    if err != nil {
        return err
    }

    b.existsStmt, err = b.db.Prepare(
        fmt.Sprintf("SELECT EXISTS(SELECT 1 FROM %s WHERE key = $1)", b.tableName),
    )
    if err != nil {
        return err
    }

    return nil
}

func (b *SQLBackend[K, V]) Get(ctx context.Context, key K) (V, error) {
    var data []byte
    err := b.getStmt.QueryRowContext(ctx, key).Scan(&data)
    if err == sql.ErrNoRows {
        var zero V
        return zero, ErrNotFound
    }
    if err != nil {
        var zero V
        return zero, fmt.Errorf("query failed: %w", err)
    }

    value, err := b.serializer.Deserialize(data)
    if err != nil {
        var zero V
        return zero, fmt.Errorf("deserialization failed: %w", err)
    }

    return value, nil
}
```

## Generation Workflow

When asked to generate a new DAL component:

### Step 1: Understand Requirements
Ask these questions:
- What type of backend? (memory, disk, SQL, NoSQL, vector, etc.)
- What key and value types?
- What performance characteristics? (latency, throughput, size)
- What special features? (TTL, encryption, compression, validation)
- What eviction strategy? (LRU, LFU, TTL-based, size-based)

### Step 2: Choose Template
Select the appropriate template:
- **Memory Cache**: High-speed, volatile storage
- **Disk Cache**: Persistent, slower but larger
- **SQL Backend**: Relational data with transactions
- **NoSQL Backend**: Schema-less, horizontal scaling
- **Vector Backend**: Embedding storage and similarity search

### Step 3: Generate Implementation
1. Create the main struct with proper fields
2. Add sync.Pool for frequently allocated objects
3. Implement DAL interface methods
4. Add helper methods for resource management
5. Include proper error handling with wrapped errors
6. Add metrics collection (hits, misses, latency)

### Step 4: Generate Tests
Always include:
```go
func TestCacheBasicOperations(t *testing.T) {
    cache := NewCache[string, string](1024 * 1024) // 1MB
    defer cache.Close()

    ctx := context.Background()

    t.Run("Set and Get", func(t *testing.T) {
        err := cache.Set(ctx, "key1", "value1")
        if err != nil {
            t.Fatalf("Set failed: %v", err)
        }

        val, err := cache.Get(ctx, "key1")
        if err != nil {
            t.Fatalf("Get failed: %v", err)
        }

        if val != "value1" {
            t.Errorf("expected 'value1', got '%s'", val)
        }
    })

    t.Run("Get non-existent", func(t *testing.T) {
        _, err := cache.Get(ctx, "nonexistent")
        if !errors.Is(err, ErrNotFound) {
            t.Errorf("expected ErrNotFound, got %v", err)
        }
    })

    t.Run("Delete", func(t *testing.T) {
        cache.Set(ctx, "key2", "value2")
        err := cache.Delete(ctx, "key2")
        if err != nil {
            t.Fatalf("Delete failed: %v", err)
        }

        _, err = cache.Get(ctx, "key2")
        if !errors.Is(err, ErrNotFound) {
            t.Errorf("expected ErrNotFound after delete, got %v", err)
        }
    })
}
```

### Step 5: Generate Benchmarks
Include comprehensive benchmarks:
```go
func BenchmarkCacheGet(b *testing.B) {
    cache := NewCache[string, []byte](10 * 1024 * 1024) // 10MB
    defer cache.Close()

    ctx := context.Background()
    data := make([]byte, 1024) // 1KB value

    // Populate cache
    for i := 0; i < 1000; i++ {
        cache.Set(ctx, fmt.Sprintf("key%d", i), data)
    }

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        _, _ = cache.Get(ctx, "key500")
    }
}

func BenchmarkCacheSet(b *testing.B) {
    cache := NewCache[string, []byte](10 * 1024 * 1024)
    defer cache.Close()

    ctx := context.Background()
    data := make([]byte, 1024)

    b.ResetTimer()
    b.ReportAllocs()

    for i := 0; i < b.N; i++ {
        cache.Set(ctx, fmt.Sprintf("key%d", i%1000), data)
    }
}
```

## Zero-Allocation Patterns

### Pattern 1: Pool Everything
```go
// Pool for byte buffers
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func serializeWithPool(v interface{}) ([]byte, error) {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    enc := gob.NewEncoder(buf)
    if err := enc.Encode(v); err != nil {
        return nil, err
    }

    // Copy data before returning buffer to pool
    result := make([]byte, buf.Len())
    copy(result, buf.Bytes())
    return result, nil
}
```

### Pattern 2: Preallocate Slices
```go
// Bad: allocates on every call
func badGetAll() []Entry {
    var results []Entry
    for _, e := range data {
        results = append(results, e) // Allocates!
    }
    return results
}

// Good: preallocate
func goodGetAll() []Entry {
    results := make([]Entry, 0, len(data))
    for _, e := range data {
        results = append(results, e)
    }
    return results
}
```

### Pattern 3: Avoid Interface Conversion
```go
// Bad: boxing to interface
func badStore(key string, value interface{}) {
    data[key] = value // Box to interface
}

// Good: use generics
func goodStore[V any](key string, value V) {
    data[key] = value // No boxing
}
```

### Pattern 4: Reuse Connections
```go
type Backend struct {
    connPool *sql.DB // Connection pool
    stmtCache map[string]*sql.Stmt // Statement cache
}
```

## Common Patterns Reference

For detailed implementation patterns, see:
- [POOLING.md](POOLING.md) - Object pooling strategies
- [SERIALIZATION.md](SERIALIZATION.md) - Efficient serialization
- [METRICS.md](METRICS.md) - Performance metrics collection
- [VALIDATION.md](VALIDATION.md) - Input validation patterns

## Code Generation Scripts

Use the provided scripts for common generation tasks:

```bash
# Generate a new cache layer
./scripts/generate-cache.sh memory string []byte

# Generate a new database backend
./scripts/generate-backend.sh postgres int64 User

# Generate benchmark suite
./scripts/generate-benchmarks.sh Cache
```

## Best Practices Checklist

Before completing any DAL generation:
- [ ] Uses generics for type safety
- [ ] Includes sync.Pool for frequently allocated objects
- [ ] All exported methods have godoc comments
- [ ] Error handling with wrapped errors (fmt.Errorf with %w)
- [ ] Context.Context in all operations
- [ ] Proper resource cleanup (Close methods)
- [ ] Comprehensive test coverage (>85%)
- [ ] Benchmarks with allocation reporting
- [ ] Thread-safe (uses proper locking)
- [ ] Follows project naming conventions
- [ ] Includes usage examples
- [ ] Documents performance characteristics

## Integration with Capacitor

All generated components should integrate with the main Capacitor API:

```go
// User code
cache := memory.NewCache[string, User](10 * 1024 * 1024)
db := postgres.NewBackend[int64, User](sqlDB, "users", jsonSerializer)

dal := capacitor.New(
    capacitor.WithLayer("L1", cache),
    capacitor.WithLayer("persistent", db),
)
```

## Examples

For complete examples, see:
- [examples/memory-cache/](examples/memory-cache/) - In-memory cache implementation
- [examples/postgres-backend/](examples/postgres-backend/) - PostgreSQL backend
- [examples/multi-layer/](examples/multi-layer/) - Multi-layer composition

## Performance Targets

Generated code should meet these benchmarks:
- **Memory Cache Get**: < 100ns/op, 0 allocs/op
- **Memory Cache Set**: < 200ns/op, 1 alloc/op (for new entries)
- **Database Get**: < 1ms/op (with network)
- **Database Set**: < 2ms/op (with network)

Run comparative benchmarks against:
- Raw map access (for cache)
- Direct SQL queries (for database backends)
- Standard library equivalents

---

Always prioritize correctness over performance, but never implement without considering performance implications.
