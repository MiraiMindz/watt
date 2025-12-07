# Pooling Strategies for Zero-Allocation Code

This document provides detailed guidance on implementing object pooling in Capacitor DAL components.

## Core Pooling Patterns

### 1. sync.Pool for Small Objects

Use `sync.Pool` for frequently allocated small objects:

```go
// Buffer pool for serialization
var bufferPool = sync.Pool{
    New: func() interface{} {
        return new(bytes.Buffer)
    },
}

func useBuffer() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset() // Clear before returning
        bufferPool.Put(buf)
    }()

    // Use buffer
    buf.WriteString("data")
}
```

**When to use:**
- Objects < 1MB
- High allocation frequency (>1000/sec)
- Objects can be reset/reused

**When NOT to use:**
- Large objects (>1MB) - memory fragmentation
- Objects with complex cleanup
- Rarely allocated objects

### 2. Per-CPU Pools for Hot Paths

For extreme performance, use per-CPU pools to eliminate lock contention:

```go
import "runtime"

type PerCPUPool[T any] struct {
    pools []*sync.Pool
    mask  int
}

func NewPerCPUPool[T any](newFn func() T) *PerCPUPool[T] {
    numCPU := runtime.NumCPU()
    // Round up to next power of 2
    size := 1
    for size < numCPU {
        size <<= 1
    }

    p := &PerCPUPool[T]{
        pools: make([]*sync.Pool, size),
        mask:  size - 1,
    }

    for i := 0; i < size; i++ {
        p.pools[i] = &sync.Pool{
            New: func() interface{} {
                return newFn()
            },
        }
    }

    return p
}

func (p *PerCPUPool[T]) Get() T {
    pid := runtime_procPin() // Pin to current CPU
    pool := p.pools[pid&p.mask]
    runtime_procUnpin()

    return pool.Get().(T)
}

func (p *PerCPUPool[T]) Put(x T) {
    pid := runtime_procPin()
    pool := p.pools[pid&p.mask]
    runtime_procUnpin()

    pool.Put(x)
}

// Usage
var entryPool = NewPerCPUPool(func() *Entry {
    return new(Entry)
})
```

### 3. Ring Buffer Pools for Predictable Allocation

For predictable allocation patterns, use ring buffers:

```go
type RingPool[T any] struct {
    items   []T
    newFn   func() T
    head    uint64
    mask    uint64
    maxSize int
}

func NewRingPool[T any](size int, newFn func() T) *RingPool[T] {
    // Round up to power of 2
    actualSize := 1
    for actualSize < size {
        actualSize <<= 1
    }

    pool := &RingPool[T]{
        items:   make([]T, actualSize),
        newFn:   newFn,
        mask:    uint64(actualSize - 1),
        maxSize: actualSize,
    }

    // Preallocate
    for i := 0; i < actualSize; i++ {
        pool.items[i] = newFn()
    }

    return pool
}

func (p *RingPool[T]) Get() T {
    idx := atomic.AddUint64(&p.head, 1) - 1
    return p.items[idx&p.mask]
}

func (p *RingPool[T]) Put(item T) {
    // No-op in ring buffer - items stay in pool
}
```

**When to use:**
- Fixed maximum concurrency
- Predictable access patterns
- Objects that can be reused without explicit reset

### 4. Slab Allocation for Same-Size Objects

For many same-size allocations, use slab allocator:

```go
type Slab struct {
    itemSize int
    slabSize int
    slabs    [][]byte
    free     [][]byte
    mu       sync.Mutex
}

func NewSlab(itemSize, initialSlabs int) *Slab {
    slabSize := 64 * 1024 // 64KB slabs
    return &Slab{
        itemSize: itemSize,
        slabSize: slabSize,
        slabs:    make([][]byte, 0, initialSlabs),
        free:     make([][]byte, 0, slabSize/itemSize*initialSlabs),
    }
}

func (s *Slab) Alloc() []byte {
    s.mu.Lock()
    defer s.mu.Unlock()

    if len(s.free) == 0 {
        s.allocateSlab()
    }

    item := s.free[len(s.free)-1]
    s.free = s.free[:len(s.free)-1]
    return item
}

func (s *Slab) Free(item []byte) {
    s.mu.Lock()
    s.free = append(s.free, item)
    s.mu.Unlock()
}

func (s *Slab) allocateSlab() {
    slab := make([]byte, s.slabSize)
    s.slabs = append(s.slabs, slab)

    for i := 0; i < len(slab); i += s.itemSize {
        s.free = append(s.free, slab[i:i+s.itemSize])
    }
}
```

## Pooling Decision Tree

```
Is the object frequently allocated?
├─ No → Don't pool
└─ Yes → Continue

Is the object < 1MB?
├─ No → Use custom allocation strategy
└─ Yes → Continue

Is allocation in a hot path (>10k ops/sec)?
├─ No → Use sync.Pool
└─ Yes → Continue

Is there high contention (>8 cores)?
├─ No → Use sync.Pool
└─ Yes → Use Per-CPU Pool

Is allocation pattern predictable?
├─ Yes → Consider Ring Buffer Pool
└─ No → Use Per-CPU Pool
```

## Common Pooling Anti-Patterns

### Anti-Pattern 1: Pooling Large Objects
```go
// BAD: Pools large objects
var hugePool = sync.Pool{
    New: func() interface{} {
        return make([]byte, 10*1024*1024) // 10MB
    },
}
```

**Why it's bad:** Large objects cause memory fragmentation and GC pressure.

**Better:** Use a free list with explicit memory management.

### Anti-Pattern 2: Not Resetting Objects
```go
// BAD: Doesn't reset
func badUse() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf) // Forgot to reset!

    buf.WriteString("data")
}
```

**Why it's bad:** Next Get() will have leftover data.

**Better:**
```go
func goodUse() {
    buf := bufferPool.Get().(*bytes.Buffer)
    defer func() {
        buf.Reset()
        bufferPool.Put(buf)
    }()

    buf.WriteString("data")
}
```

### Anti-Pattern 3: Holding Pool Objects Too Long
```go
// BAD: Holds object across goroutine
func badAsync() {
    buf := bufferPool.Get().(*bytes.Buffer)

    go func() {
        defer bufferPool.Put(buf)
        // Long running task
        process(buf)
    }()
}
```

**Why it's bad:** Starves pool, defeats purpose.

**Better:** Copy data, return object immediately.

## Performance Monitoring

Add metrics to your pools:

```go
type MeteredPool[T any] struct {
    pool     *sync.Pool
    gets     uint64
    puts     uint64
    news     uint64 // Number of new allocations
}

func (p *MeteredPool[T]) Get() T {
    atomic.AddUint64(&p.gets, 1)
    v := p.pool.Get()
    if v == nil {
        atomic.AddUint64(&p.news, 1)
        return p.newFn()
    }
    return v.(T)
}

func (p *MeteredPool[T]) Stats() PoolStats {
    gets := atomic.LoadUint64(&p.gets)
    news := atomic.LoadUint64(&p.news)

    return PoolStats{
        Gets:    gets,
        News:    news,
        HitRate: float64(gets-news) / float64(gets),
    }
}
```

## Benchmarking Pools

Always benchmark your pool implementation:

```go
func BenchmarkPoolVsNew(b *testing.B) {
    b.Run("New", func(b *testing.B) {
        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            buf := new(bytes.Buffer)
            buf.WriteString("test")
            _ = buf.Bytes()
        }
    })

    b.Run("Pool", func(b *testing.B) {
        pool := &sync.Pool{
            New: func() interface{} {
                return new(bytes.Buffer)
            },
        }

        b.ReportAllocs()
        for i := 0; i < b.N; i++ {
            buf := pool.Get().(*bytes.Buffer)
            buf.WriteString("test")
            _ = buf.Bytes()
            buf.Reset()
            pool.Put(buf)
        }
    })
}
```

Expected results:
```
BenchmarkPoolVsNew/New-8      5000000    250 ns/op    112 B/op    2 allocs/op
BenchmarkPoolVsNew/Pool-8    20000000     80 ns/op      0 B/op    0 allocs/op
```

## Integration with Capacitor

When creating a pooled cache:

```go
type PooledCache[K comparable, V any] struct {
    data  map[K]*entry[V]
    pool  *sync.Pool
    mu    sync.RWMutex
}

func NewPooledCache[K comparable, V any]() *PooledCache[K, V] {
    return &PooledCache[K, V]{
        data: make(map[K]*entry[V]),
        pool: &sync.Pool{
            New: func() interface{} {
                return new(entry[V])
            },
        },
    }
}

func (c *PooledCache[K, V]) Get(ctx context.Context, key K) (V, error) {
    c.mu.RLock()
    e := c.data[key]
    c.mu.RUnlock()

    if e == nil {
        var zero V
        return zero, ErrNotFound
    }

    return e.value, nil
}

func (c *PooledCache[K, V]) Set(ctx context.Context, key K, value V, opts ...SetOption) error {
    e := c.pool.Get().(*entry[V])
    e.value = value

    c.mu.Lock()
    if old := c.data[key]; old != nil {
        // Return old entry to pool
        var zero V
        old.value = zero
        c.pool.Put(old)
    }
    c.data[key] = e
    c.mu.Unlock()

    return nil
}
```

## Summary

- Use `sync.Pool` for most cases
- Use Per-CPU pools for extreme hot paths
- Use ring buffers for predictable patterns
- Always reset objects before returning to pool
- Measure pool effectiveness with metrics
- Benchmark before and after pooling
