---
name: concurrency-patterns
description: Expert in concurrent programming patterns including lock-free data structures, memory ordering, synchronization primitives, and work scheduling. Use when implementing multi-threaded components or optimizing for concurrent access.
allowed-tools: Read, Write, Edit, Bash, Grep, Glob
---

# Concurrency Patterns Skill

## Purpose
This skill provides expertise in designing and implementing correct, efficient concurrent systems in Go, with focus on lock-free algorithms, memory ordering, and performance.

## When to Use This Skill
- Implementing lock-free data structures
- Optimizing synchronization for high contention
- Designing work-stealing schedulers
- Building concurrent queues and pools
- Understanding memory ordering and atomics

## Core Principles

### 1. Correctness First, Performance Second
Concurrent code is hard to debug. Always:
1. Write clear, obviously correct code
2. Document synchronization invariants
3. Test with race detector
4. Benchmark before optimizing
5. Use simplest correct solution

### 2. The Concurrency Hierarchy
Choose the simplest tool that works:
```
1. No sharing (best)
2. sync.Once, atomic values (simple)
3. sync.Mutex (standard)
4. sync.RWMutex (read-heavy)
5. Channels (communication)
6. Sharded locks (reduce contention)
7. Lock-free (when proven needed)
```

### 3. Memory Ordering Matters
Go's memory model guarantees:
- Synchronization operations (locks, channels, atomics) create happens-before relationships
- Without synchronization, reordering can occur
- Race detector finds most issues

## Lock-Free Data Structures

### 1. Lock-Free Stack (LIFO)

```go
package lockfree

import (
    "sync/atomic"
    "unsafe"
)

// Stack is a lock-free LIFO stack.
// Safe for concurrent use.
type Stack[T any] struct {
    top atomic.Pointer[node[T]]
}

type node[T any] struct {
    value T
    next  *node[T]
}

func NewStack[T any]() *Stack[T] {
    return &Stack[T]{}
}

func (s *Stack[T]) Push(value T) {
    n := &node[T]{value: value}

    for {
        old := s.top.Load()
        n.next = old

        if s.top.CompareAndSwap(old, n) {
            return
        }
        // CAS failed, retry
    }
}

func (s *Stack[T]) Pop() (T, bool) {
    for {
        old := s.top.Load()
        if old == nil {
            var zero T
            return zero, false
        }

        if s.top.CompareAndSwap(old, old.next) {
            return old.value, true
        }
        // CAS failed, retry
    }
}
```

### 2. Lock-Free Queue (MPMC)

```go
// Queue is a lock-free bounded MPMC queue.
type Queue[T any] struct {
    _       [8]uint64 // Padding
    head    atomic.Uint64
    _       [7]uint64 // Padding (cache line)
    tail    atomic.Uint64
    _       [7]uint64 // Padding
    mask    uint64
    slots   []slot[T]
}

type slot[T any] struct {
    _       [8]uint64 // Padding
    seq     atomic.Uint64
    value   T
}

func NewQueue[T any](size int) *Queue[T] {
    // Round up to power of 2
    size = roundUpPow2(size)

    q := &Queue[T]{
        mask:  uint64(size - 1),
        slots: make([]slot[T], size),
    }

    // Initialize sequences
    for i := range q.slots {
        q.slots[i].seq.Store(uint64(i))
    }

    return q
}

func (q *Queue[T]) Enqueue(value T) bool {
    for {
        head := q.head.Load()
        slot := &q.slots[head&q.mask]
        seq := slot.seq.Load()

        switch diff := seq - head; {
        case diff == 0:
            // Slot is empty, try to claim it
            if q.head.CompareAndSwap(head, head+1) {
                slot.value = value
                slot.seq.Store(head + 1)
                return true
            }
        case diff < 0:
            // Queue is full
            return false
        default:
            // Another thread claimed this slot, retry
        }
    }
}

func (q *Queue[T]) Dequeue() (T, bool) {
    for {
        tail := q.tail.Load()
        slot := &q.slots[tail&q.mask]
        seq := slot.seq.Load()

        switch diff := seq - (tail + 1); {
        case diff == 0:
            // Slot has data, try to claim it
            if q.tail.CompareAndSwap(tail, tail+1) {
                value := slot.value
                slot.seq.Store(tail + q.mask + 1)
                return value, true
            }
        case diff < 0:
            // Queue is empty
            var zero T
            return zero, false
        default:
            // Another thread claimed this slot, retry
        }
    }
}
```

### 3. RCU (Read-Copy-Update)

```go
// RCU allows lock-free reads with infrequent updates.
type RCU[T any] struct {
    current atomic.Pointer[T]
    mu      sync.Mutex // Only for writers
}

func NewRCU[T any](initial T) *RCU[T] {
    r := &RCU[T]{}
    r.current.Store(&initial)
    return r
}

// Read returns the current value (lock-free).
func (r *RCU[T]) Read() *T {
    return r.current.Load()
}

// Update atomically replaces the value.
// Only one writer at a time (serialized by mutex).
func (r *RCU[T]) Update(fn func(*T) *T) {
    r.mu.Lock()
    defer r.mu.Unlock()

    old := r.current.Load()
    new := fn(old)
    r.current.Store(new)

    // In a real RCU, we'd defer freeing 'old' until
    // all readers are done (grace period)
}
```

## Synchronization Patterns

### 1. Sharded Mutex (Reduce Contention)

```go
// ShardedMutex reduces contention by splitting into multiple locks.
type ShardedMutex struct {
    shards []sync.Mutex
    mask   uint64
}

func NewShardedMutex(shards int) *ShardedMutex {
    shards = roundUpPow2(shards)
    return &ShardedMutex{
        shards: make([]sync.Mutex, shards),
        mask:   uint64(shards - 1),
    }
}

func (m *ShardedMutex) Lock(key uint64) {
    m.shards[key&m.mask].Lock()
}

func (m *ShardedMutex) Unlock(key uint64) {
    m.shards[key&m.mask].Unlock()
}

// Usage:
// sm := NewShardedMutex(64)
// sm.Lock(hash(key))
// defer sm.Unlock(hash(key))
```

### 2. Reader-Priority RWMutex

```go
// FastRWMutex optimized for read-heavy workloads.
type FastRWMutex struct {
    readerCount atomic.Int32
    writerWait  atomic.Bool
    writerLock  sync.Mutex
}

func (rw *FastRWMutex) RLock() {
    for {
        if rw.writerWait.Load() {
            // Writer waiting, yield
            runtime.Gosched()
            continue
        }

        rw.readerCount.Add(1)

        // Check again after incrementing
        if rw.writerWait.Load() {
            rw.readerCount.Add(-1)
            runtime.Gosched()
            continue
        }

        return
    }
}

func (rw *FastRWMutex) RUnlock() {
    rw.readerCount.Add(-1)
}

func (rw *FastRWMutex) Lock() {
    rw.writerLock.Lock()
    rw.writerWait.Store(true)

    // Wait for readers to drain
    for rw.readerCount.Load() > 0 {
        runtime.Gosched()
    }
}

func (rw *FastRWMutex) Unlock() {
    rw.writerWait.Store(false)
    rw.writerLock.Unlock()
}
```

### 3. Semaphore for Rate Limiting

```go
type Semaphore struct {
    permits chan struct{}
}

func NewSemaphore(n int) *Semaphore {
    return &Semaphore{
        permits: make(chan struct{}, n),
    }
}

func (s *Semaphore) Acquire() {
    s.permits <- struct{}{}
}

func (s *Semaphore) Release() {
    <-s.permits
}

func (s *Semaphore) TryAcquire() bool {
    select {
    case s.permits <- struct{}{}:
        return true
    default:
        return false
    }
}

// Usage:
// sem := NewSemaphore(10) // Max 10 concurrent
// sem.Acquire()
// defer sem.Release()
```

## Work Scheduling

### 1. Work-Stealing Scheduler

```go
type Scheduler struct {
    workers []*Worker
    global  *Queue[Task]
}

type Worker struct {
    id     int
    local  *Deque[Task]  // Local work queue
    others []*Deque[Task] // Other workers' queues
}

type Task func()

func NewScheduler(numWorkers int) *Scheduler {
    s := &Scheduler{
        workers: make([]*Worker, numWorkers),
        global:  NewQueue[Task](1024),
    }

    // Create workers
    for i := 0; i < numWorkers; i++ {
        s.workers[i] = &Worker{
            id:    i,
            local: NewDeque[Task](256),
        }
    }

    // Link workers for stealing
    for i := range s.workers {
        s.workers[i].others = make([]*Deque[Task], 0, numWorkers-1)
        for j := range s.workers {
            if i != j {
                s.workers[i].others = append(s.workers[i].others, s.workers[j].local)
            }
        }
    }

    return s
}

func (w *Worker) Run() {
    for {
        // 1. Try local queue (LIFO for cache locality)
        if task, ok := w.local.PopBottom(); ok {
            task()
            continue
        }

        // 2. Try stealing from others (FIFO for fairness)
        for _, other := range w.others {
            if task, ok := other.PopTop(); ok {
                task()
                break
            }
        }

        // 3. Try global queue
        // 4. Sleep if no work
    }
}
```

### 2. Batch Processor

```go
type BatchProcessor[T any] struct {
    batchSize int
    timeout   time.Duration
    process   func([]T) error

    mu     sync.Mutex
    items  []T
    timer  *time.Timer
}

func NewBatchProcessor[T any](size int, timeout time.Duration, fn func([]T) error) *BatchProcessor[T] {
    return &BatchProcessor[T]{
        batchSize: size,
        timeout:   timeout,
        process:   fn,
        items:     make([]T, 0, size),
    }
}

func (b *BatchProcessor[T]) Add(item T) error {
    b.mu.Lock()
    defer b.mu.Unlock()

    b.items = append(b.items, item)

    if len(b.items) >= b.batchSize {
        return b.flush()
    }

    if b.timer == nil {
        b.timer = time.AfterFunc(b.timeout, func() {
            b.mu.Lock()
            defer b.mu.Unlock()
            b.flush()
        })
    }

    return nil
}

func (b *BatchProcessor[T]) flush() error {
    if len(b.items) == 0 {
        return nil
    }

    if b.timer != nil {
        b.timer.Stop()
        b.timer = nil
    }

    err := b.process(b.items)
    b.items = b.items[:0]
    return err
}
```

## Memory Ordering and Atomics

### 1. Atomic Operations

```go
import "sync/atomic"

// Common atomic patterns
var counter atomic.Int64

// Increment
counter.Add(1)

// Load
value := counter.Load()

// Store
counter.Store(42)

// Compare-and-swap
old := counter.Load()
swapped := counter.CompareAndSwap(old, old+1)

// Swap (atomic exchange)
oldValue := counter.Swap(100)
```

### 2. Atomic Pointers

```go
type Config struct {
    Timeout time.Duration
    MaxConn int
}

var config atomic.Pointer[Config]

// Initialize
config.Store(&Config{
    Timeout: time.Second,
    MaxConn: 100,
})

// Read (lock-free)
cfg := config.Load()
timeout := cfg.Timeout

// Update (rare)
newConfig := &Config{
    Timeout: 2 * time.Second,
    MaxConn: 200,
}
config.Store(newConfig)
```

### 3. Memory Fences

```go
// Go's atomic operations include appropriate memory barriers
// No explicit fences needed in most cases

// Example: Lazy initialization with double-checked locking
type Cache struct {
    data atomic.Pointer[Data]
    mu   sync.Mutex
}

func (c *Cache) Get() *Data {
    // Fast path: check without lock
    if d := c.data.Load(); d != nil {
        return d // Acquire barrier ensures we see initialized data
    }

    // Slow path: initialize
    c.mu.Lock()
    defer c.mu.Unlock()

    // Check again while holding lock
    if d := c.data.Load(); d != nil {
        return d
    }

    // Initialize
    d := &Data{/* ... */}
    c.data.Store(d) // Release barrier ensures others see initialized data
    return d
}
```

## Common Concurrency Patterns

### 1. Double-Checked Locking (Initialization)

```go
type Service struct {
    client atomic.Pointer[Client]
    mu     sync.Mutex
}

func (s *Service) GetClient() *Client {
    if c := s.client.Load(); c != nil {
        return c
    }

    s.mu.Lock()
    defer s.mu.Unlock()

    if c := s.client.Load(); c != nil {
        return c
    }

    c := &Client{/* ... */}
    s.client.Store(c)
    return c
}
```

### 2. Fan-Out/Fan-In

```go
func fanOut[T any](items []T, workers int, process func(T) error) error {
    errCh := make(chan error, len(items))
    sem := make(chan struct{}, workers)

    for _, item := range items {
        item := item // Capture

        sem <- struct{}{} // Acquire
        go func() {
            defer func() { <-sem }() // Release
            errCh <- process(item)
        }()
    }

    // Wait for all
    for i := 0; i < len(items); i++ {
        if err := <-errCh; err != nil {
            return err
        }
    }

    return nil
}
```

### 3. Pipeline Pattern

```go
func pipeline[T, U, V any](
    input <-chan T,
    stage1 func(T) U,
    stage2 func(U) V,
) <-chan V {
    middle := make(chan U, 10)
    output := make(chan V, 10)

    // Stage 1
    go func() {
        defer close(middle)
        for item := range input {
            middle <- stage1(item)
        }
    }()

    // Stage 2
    go func() {
        defer close(output)
        for item := range middle {
            output <- stage2(item)
        }
    }()

    return output
}
```

## Testing Concurrent Code

### 1. Race Detector

```bash
# ALWAYS test concurrent code with race detector
go test -race ./...

# Run benchmarks with race detector
go test -race -bench=.
```

### 2. Stress Testing

```go
func TestQueueConcurrent(t *testing.T) {
    q := NewQueue[int](1024)

    const (
        producers = 10
        consumers = 10
        items     = 10000
    )

    var wg sync.WaitGroup

    // Producers
    for i := 0; i < producers; i++ {
        wg.Add(1)
        go func(id int) {
            defer wg.Done()
            for j := 0; j < items; j++ {
                for !q.Enqueue(id*items + j) {
                    runtime.Gosched()
                }
            }
        }(i)
    }

    // Consumers
    seen := make([]atomic.Bool, producers*items)
    for i := 0; i < consumers; i++ {
        wg.Add(1)
        go func() {
            defer wg.Done()
            for {
                if v, ok := q.Dequeue(); ok {
                    if seen[v].Swap(true) {
                        t.Errorf("duplicate value: %d", v)
                    }
                } else {
                    return
                }
            }
        }()
    }

    wg.Wait()

    // Verify all items seen
    for i, s := range seen {
        if !s.Load() {
            t.Errorf("missing value: %d", i)
        }
    }
}
```

### 3. Deadlock Detection

```go
func TestNoDeadlock(t *testing.T) {
    done := make(chan bool)
    go func() {
        // Code that might deadlock
        operationUnderTest()
        done <- true
    }()

    select {
    case <-done:
        // OK
    case <-time.After(5 * time.Second):
        t.Fatal("potential deadlock detected")
    }
}
```

## Performance Checklist

- [ ] Profile shows contention on locks
- [ ] Workload is actually concurrent (multi-core benefit)
- [ ] Lock-free algorithm is proven correct
- [ ] Tested with race detector
- [ ] Stress tested with high goroutine count
- [ ] Benchmarked against simpler solutions
- [ ] No premature optimization

## Common Pitfalls

1. **Data Races** - Use race detector religiously
2. **ABA Problem** - Track version counters in CAS loops
3. **False Sharing** - Pad hot atomics to cache lines
4. **Lock Granularity** - Too coarse (contention) or too fine (overhead)
5. **Goroutine Leaks** - Always have exit conditions
6. **Starvation** - Ensure fairness in lock-free algorithms

## Resources

- [Go Memory Model](https://go.dev/ref/mem)
- [Concurrency is not Parallelism](https://go.dev/blog/waza-talk)
- [Lock-Free Algorithms](https://preshing.com/20120612/an-introduction-to-lock-free-programming/)
- [The Art of Multiprocessor Programming](https://www.amazon.com/Art-Multiprocessor-Programming-Maurice-Herlihy/dp/0123973376)

---

**Remember**: The best concurrent code is simple, obviously correct code. Complexity is the enemy of correctness.
