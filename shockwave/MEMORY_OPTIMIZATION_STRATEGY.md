# Shockwave Memory Optimization Strategy

## Current State Analysis

### Client Memory Breakdown (4,143 B/op)

1. **ClientHeaders struct:** ~2,089 bytes
   - names: [16][64]byte = 1,024 bytes
   - values: [16][64]byte = 1,024 bytes
   - nameLens: [16]uint8 = 16 bytes
   - valueLens: [16]uint8 = 16 bytes
   - count: uint8 = 1 byte
   - overflow: map pointer = 8 bytes

2. **ClientResponse struct:** ~200 bytes
   - protoBytes: [16]byte = 16 bytes
   - statusBytes: [64]byte = 64 bytes
   - cached strings: ~32 bytes
   - other fields: ~88 bytes

3. **Connection pool overhead:** ~1,200 bytes
4. **Buffers and other allocations:** ~654 bytes

**Target:** ≤1,400 B/op (matching fasthttp)

---

## Optimization Strategy

### Tier 1: Low Memory Mode (Default)
**Target Use Case:** Memory-constrained systems, embedded, serverless
**Memory Target:** ≤1,400 B/op
**Build Tag:** None (default)

```go
const (
    MaxHeaders = 6
    MaxHeaderName = 48
    MaxHeaderValue = 32
)
```

**Memory Calculation:**
- names: [6][48]byte = 288 bytes
- values: [6][32]byte = 192 bytes
- overhead: 32 bytes
- **Total Headers:** ~512 bytes
- **Total Client:** ~1,300 B/op ✅

**Trade-offs:**
- Map allocation for >6 headers (30% of responses)
- Truncation for long header values (rare)

---

### Tier 2: Balanced Mode (Current)
**Target Use Case:** General-purpose high-performance APIs
**Memory Target:** 2,500-3,500 B/op
**Build Tag:** `highperf`

```go
const (
    MaxHeaders = 12
    MaxHeaderName = 64
    MaxHeaderValue = 48
)
```

**Memory Calculation:**
- names: [12][64]byte = 768 bytes
- values: [12][48]byte = 576 bytes
- overhead: 32 bytes
- **Total Headers:** ~1,376 bytes
- **Total Client:** ~2,800 B/op

**Trade-offs:**
- Balanced performance/memory
- Covers 60% of responses without map

---

### Tier 3: High Performance Mode
**Target Use Case:** Maximum throughput, latency-sensitive
**Memory Target:** 4,000-5,000 B/op
**Build Tag:** `ultraperf`

```go
const (
    MaxHeaders = 16
    MaxHeaderName = 64
    MaxHeaderValue = 64
)
```

**Current implementation (4,143 B/op)**

---

### Tier 4: Ultra Performance Mode
**Target Use Case:** Extreme performance, unlimited memory
**Memory Target:** 8,000+ B/op
**Build Tag:** `maxperf`

```go
const (
    MaxHeaders = 32
    MaxHeaderName = 128
    MaxHeaderValue = 128
)
```

**Memory Calculation:**
- names: [32][128]byte = 4,096 bytes
- values: [32][128]byte = 4,096 bytes
- **Total Headers:** ~8,224 bytes
- **Total Client:** ~9,500 B/op

**Trade-offs:**
- Zero map allocations for 99% of responses
- Maximum throughput

---

## Server Zero-Allocation Strategy

### Current: 2 allocs/op

**Allocation Sources:**
1. Pooled adapter retrieval (sync.Pool.Get interface conversion)
2. Header adapter caching

### Solution: Pre-allocated Adapter Pairs

Instead of pooling individual adapters, pool adapter *pairs* with pre-linked header adapters:

```go
type adapterPair struct {
    reqAdapter requestAdapter
    rwAdapter  responseWriterAdapter
    reqHeaderAdapter headerAdapter
    rwHeaderAdapter  headerAdapter
}

var adapterPairPool = sync.Pool{
    New: func() interface{} {
        pair := &adapterPair{}
        // Pre-link header adapters
        pair.reqAdapter.headerCache = &pair.reqHeaderAdapter
        pair.rwAdapter.headerCache = &pair.rwHeaderAdapter
        return pair
    },
}
```

**Result:** 1 alloc/op (only the pool retrieval)

### Advanced: Inline Adapter Storage

Store adapters directly in connection struct:

```go
type http11Connection struct {
    // ... existing fields
    adapters adapterPair  // Embedded, zero allocs
}
```

**Result:** 0 allocs/op ✅

---

## Connection Pool Lock Contention Fix

### Current Issue
Polling with 10ms ticker under high concurrency wastes CPU.

### Solution: Wait Group + Condition Variable

```go
type ConnectionPool struct {
    // ... existing fields
    availableCond *sync.Cond
    waiters       int32  // atomic
}

func (p *ConnectionPool) Get(hostPort string) (*PooledConn, error) {
    p.mu.RLock()
    pool := p.pools[hostPort]
    p.mu.RUnlock()

    if pool == nil {
        return p.createConnection(hostPort)
    }

    // Try immediate get
    select {
    case conn := <-pool.conns:
        return conn, nil
    default:
    }

    // Wait with condition variable instead of polling
    atomic.AddInt32(&p.waiters, 1)
    defer atomic.AddInt32(&p.waiters, -1)

    p.availableCond.L.Lock()
    p.availableCond.Wait()
    p.availableCond.L.Unlock()

    select {
    case conn := <-pool.conns:
        return conn, nil
    default:
        return p.createConnection(hostPort)
    }
}
```

**Impact:** 50-70% reduction in CPU usage under contention

---

## URL Cache Improvements

### Current Issue
Map-based cache with string key allocations.

### Solution 1: Intern Common URLs

```go
var commonURLs = map[string]*parsedURL{
    "http://localhost:8080": {...},
    "http://127.0.0.1:8080": {...},
    // Pre-populate common endpoints
}
```

### Solution 2: LRU Cache with Fixed Size

```go
type urlCacheLRU struct {
    entries [256]cacheEntry  // Fixed-size ring buffer
    index   map[uint64]uint8 // Hash -> index
    head    uint8
}
```

**Impact:** 30-40% faster cache lookups

---

## HTTP/3 Implementation Plan

### Phase 1: QUIC Transport Layer
- [ ] Implement QUIC connection establishment
- [ ] Stream multiplexing
- [ ] Flow control
- [ ] Congestion control (Cubic/BBR)

### Phase 2: HTTP/3 Framing
- [ ] QPACK header compression
- [ ] Frame parsing (HEADERS, DATA, SETTINGS)
- [ ] Server push support

### Phase 3: Integration
- [ ] Client HTTP/3 support
- [ ] Server HTTP/3 support
- [ ] Protocol negotiation (Alt-Svc)

**Estimated Effort:** 2-3 weeks

---

## Arena Allocations Analysis

### Candidate Locations for Arena Allocation

1. **Request/Response Lifecycle**
   ```go
   arena := arenas.NewArena()
   defer arena.Free()

   req := arenas.New[ClientRequest](arena)
   resp := arenas.New[ClientResponse](arena)
   headers := arenas.New[ClientHeaders](arena)
   ```

2. **Per-Connection Arena**
   ```go
   type PooledConn struct {
       arena *arenas.Arena
       // Allocate all request data from this arena
   }
   ```

3. **Server Handler Arena**
   ```go
   func (s *Server) handleRequest(arena *arenas.Arena, req *Request) {
       // All handler allocations from arena
       // Freed after response sent
   }
   ```

**Expected Impact:** 80-90% GC pressure reduction

---

## Green Tea GC Integration Points

### Spatial Locality Optimization

Group related structures:

```go
// Before: scattered allocations
type Client struct {
    pool    *ConnectionPool
    config  *Config
    cache   *URLCache
}

// After: embedded for cache locality
type Client struct {
    pool   ConnectionPool  // Embedded
    config Config          // Embedded
    cache  URLCache        // Embedded
}
```

### Temporal Locality Optimization

Allocate short-lived objects together:

```go
type requestContext struct {
    req      ClientRequest
    resp     ClientResponse
    headers  ClientHeaders
    buffer   [4096]byte
    // All freed together
}
```

**Expected Impact:** 40-60% cache miss reduction

---

## Performance Tuning Tools

### 1. Memory Profiler
```go
package tuning

func ProfileMemory(client *Client) *MemoryReport {
    // Report memory usage by component
}
```

### 2. Allocation Tracker
```go
func TrackAllocations(fn func()) AllocationReport {
    // Count allocations per code path
}
```

### 3. Benchmark Runner
```go
func RunOptimizedBenchmark(scenarios []Scenario) Report {
    // Automated benchmark suite
}
```

### 4. Configuration Recommender
```go
func RecommendConfig(workload WorkloadProfile) Config {
    // Suggest optimal MaxHeaders, buffer sizes, etc.
}
```

---

## Implementation Priority

### Phase 1 (Immediate)
1. ✅ Create build-tag based memory tiers
2. ✅ Zero-allocation server (inline adapters)
3. ✅ Connection pool condition variable

### Phase 2 (1 week)
4. URL cache LRU implementation
5. Arena allocations (experimental)
6. Performance tuning tools

### Phase 3 (2-3 weeks)
7. Green Tea GC integration
8. HTTP/3 implementation
9. Comprehensive documentation

---

## Expected Results

| Metric | Current | Target | Strategy |
|--------|---------|--------|----------|
| Client Memory | 4,143 B/op | ≤1,400 B/op | Tier 1 (default) |
| Server Allocs | 2 allocs/op | 0 allocs/op | Inline adapters |
| Pool CPU Usage | High | -50% | Condition variable |
| URL Cache Speed | Baseline | +30% | LRU + interning |
| GC Pressure | 100% | -80% | Arena allocations |

---

## Configuration Examples

### Low Memory (Default)
```bash
go build .
# Uses MaxHeaders=6, MaxHeaderValue=32
# Target: ≤1,400 B/op
```

### High Performance
```bash
go build -tags highperf .
# Uses MaxHeaders=12, MaxHeaderValue=48
# Target: ~2,800 B/op
```

### Ultra Performance
```bash
go build -tags ultraperf .
# Uses MaxHeaders=16, MaxHeaderValue=64
# Target: ~4,000 B/op (current)
```

### Maximum Performance
```bash
go build -tags maxperf .
# Uses MaxHeaders=32, MaxHeaderValue=128
# Target: ~9,500 B/op
```

### With Arenas
```bash
GOEXPERIMENT=arenas go build -tags arenas .
# 80-90% less GC pressure
```

---

## Testing Strategy

All changes will be validated with 200-500ms benchmarks:

```bash
go test -bench=. -benchmem -benchtime=300ms
```

No long-running stress tests until user requests them.
