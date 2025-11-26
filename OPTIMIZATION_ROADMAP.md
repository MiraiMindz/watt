# Comprehensive Optimization Roadmap
## Bolt & Shockwave Performance Optimization Plan

**Date:** 2025-11-18
**Goal:** Achieve #1 ranking in both CPU performance and Memory efficiency
**Timeline:** 8 weeks (4 weeks per project, can run in parallel)

---

## Executive Summary

### Current State
- **Bolt:** 3rd place overall (loses to Gin and Echo)
- **Shockwave:** Competitive with fasthttp (not consistently #1)

### Critical Findings
- **Bolt Paradox:** Pure router is 6ns/op but full stack is 1,581ns/op (262x overhead!)
- **Root Causes Identified:**
  - Context pooling overhead: ~423ns
  - Shockwave adapter inefficiency: Unknown overhead
  - Excessive memory allocations in dynamic routes
  - Concurrent handling bottlenecks

### Target State
- **Bolt:** <500ns/op for static routes, <1,400ns/op for dynamic routes
- **Shockwave:** Match or beat fasthttp in all categories
- **Both:** Zero allocations in hot paths

---

# PART 1: BOLT FRAMEWORK OPTIMIZATION

## Phase 1: Critical Path Analysis (Week 1)

### Task 1.1: Profile Full Stack Overhead
**Location:** `bolt/benchmarks/`

**Actions:**
```bash
cd /home/mirai/Documents/Programming/Projects/watt/bolt
go test -bench=BenchmarkFull_Bolt_StaticRoute -cpuprofile=cpu.prof -memprofile=mem.prof
go tool pprof -http=:8080 cpu.prof
```

**Expected Findings:**
- Identify where 262x overhead comes from (6ns → 1,581ns)
- Measure exact overhead of:
  - Context pool acquire/release
  - Shockwave adapter
  - Middleware execution
  - Response writing

### Task 1.2: Create Overhead Breakdown Report
**File to create:** `bolt/docs/overhead_analysis.md`

**Document:**
- Overhead per component (ns and %)
- Allocation sources (B/op breakdown)
- Lock contention points
- String conversion hotspots

---

## Phase 2: Context Pool Optimization (Week 1-2)

### Issue: Context Pool Adds ~423ns Overhead

**File:** `bolt/core/context_pool.go`

### Optimization 2.1: Eliminate Reset() Overhead

**Current Implementation:**
```go
func (ctx *Context) Reset() {
    ctx.req = nil
    ctx.resp = nil
    ctx.params = nil
    ctx.query = nil
    ctx.store = nil
    ctx.path = ""
    ctx.method = ""
    // ... many more fields
}
```

**Problem:** Too many field assignments

**Optimized Implementation:**
```go
// Use bulk zeroing instead of field-by-field assignment
func (ctx *Context) Reset() {
    // Zero the entire struct at once
    *ctx = Context{
        // Only reinitialize critical fields
        paramsBuf: ctx.paramsBuf, // Reuse inline arrays
        queryBuf:  ctx.queryBuf,
    }
}
```

**Expected Improvement:** 423ns → 50ns (8.5x faster)

### Optimization 2.2: Pre-Warm Pool

**File:** `bolt/core/context_pool.go`

**Add:**
```go
func (p *ContextPool) Warmup(count int) {
    ctxs := make([]*Context, count)
    for i := 0; i < count; i++ {
        ctxs[i] = p.Acquire()
    }
    for _, ctx := range ctxs {
        p.Release(ctx)
    }
}

// In app.go initialization
func New() *App {
    app := &App{
        ctxPool: NewContextPool(),
        // ...
    }
    app.ctxPool.Warmup(1000) // Pre-allocate 1000 contexts
    return app
}
```

**Expected Improvement:** Eliminates cold start allocations

---

## Phase 3: Shockwave Adapter Optimization (Week 2)

### Issue: Unknown Overhead in Shockwave Adapter

**File:** `bolt/shockwave/adapter.go` (111 lines)

### Optimization 3.1: Zero-Copy Request Mapping

**Current Approach:** Review `handleShockwaveRequest()` in `core/app.go`

**Optimization:**
```go
// Eliminate any intermediate allocations
func (app *App) handleShockwaveRequest(shockReq *shockwave.Request, shockResp shockwave.ResponseWriter) {
    // Acquire context ONCE
    ctx := app.ctxPool.Acquire()

    // Direct pointer assignment (zero-copy)
    ctx.req = shockReq
    ctx.resp = shockResp

    // Use byte slices directly (no string conversion)
    ctx.methodBytes = shockReq.MethodBytes()
    ctx.pathBytes = shockReq.PathBytes()

    // Defer release with fast path
    handler := app.router.LookupBytes(ctx.methodBytes, ctx.pathBytes, &ctx.paramsBuf)
    if handler == nil {
        ctx.resp.WriteHeader(404)
        app.ctxPool.Release(ctx)
        return
    }

    // Execute handler
    if err := handler(ctx); err != nil {
        app.config.ErrorHandler(ctx, err)
    }

    app.ctxPool.Release(ctx)
}
```

**Key Changes:**
- No string allocations (use byte slices throughout)
- Fast-path 404 handling
- Direct pointer assignment
- Early release on error path

**Expected Improvement:** Reduce adapter overhead to <50ns

---

## Phase 4: Parameter Extraction Optimization (Week 2)

### Issue: Dynamic Routes Use 351 B/op vs 160 B/op (Echo)

**File:** `bolt/core/context.go`

### Optimization 4.1: Increase Inline Parameter Storage

**Current:**
```go
type Context struct {
    paramsBuf [4]paramPair // Only 4 params inline
    // ...
}
```

**Optimized:**
```go
type Context struct {
    paramsBuf [8]paramPair // Double to 8 params inline
    queryBuf  [16]queryPair // Increase query params too
    // ...
}
```

**Rationale:**
- 95% of routes have ≤8 params
- Cost: 64 bytes more per Context (acceptable)
- Benefit: Zero allocations for 95% of requests

### Optimization 4.2: Optimize paramPair Structure

**Current:**
```go
type paramPair struct {
    key string
    val string
}
```

**Optimized:**
```go
type paramPair struct {
    keyBytes []byte  // Zero-copy from router
    valBytes []byte  // Zero-copy from path
    keyCached string // Lazy string conversion
    valCached string
}

func (p *paramPair) Key() string {
    if p.keyCached == "" {
        p.keyCached = bytesToString(p.keyBytes)
    }
    return p.keyCached
}

func (p *paramPair) Value() string {
    if p.valCached == "" {
        p.valCached = bytesToString(p.valBytes)
    }
    return p.valCached
}
```

**Benefits:**
- Zero allocations when using byte methods
- Lazy string conversion only when needed
- Cache strings to avoid repeated conversions

**Expected Improvement:** 351 B/op → 100 B/op (3.5x reduction)

---

## Phase 5: Middleware Optimization (Week 3)

### Issue: Middleware Uses 271 B/op vs 64 B/op (Gin)

**File:** `bolt/core/app.go`

### Optimization 5.1: Pre-Compile Middleware Chain

**Current:**
```go
// Middleware wrapped at registration time
func (app *App) Get(path string, handler Handler) *ChainLink {
    finalHandler := handler
    for i := len(app.middleware) - 1; i >= 0; i-- {
        finalHandler = app.middleware[i](finalHandler)
    }
    app.router.Add(MethodGet, path, finalHandler)
    // ...
}
```

**Problem:** Creates new closure on every request

**Optimized:**
```go
// Store middleware chain as array, execute in loop
type Route struct {
    handler    Handler
    middleware []Middleware
}

func (app *App) executeRoute(ctx *Context, route *Route) error {
    // Execute middleware chain without closures
    handler := route.handler
    for i := len(route.middleware) - 1; i >= 0; i-- {
        handler = route.middleware[i](handler)
    }
    return handler(ctx)
}
```

**Expected Improvement:** 271 B/op → 64 B/op (4.2x reduction)

### Optimization 5.2: Optimize Built-in Middleware

**Files:** `bolt/middleware/*.go`

**Changes:**
- Use pre-compiled byte slice constants
- Eliminate string allocations in logger
- Use context.store efficiently (pre-allocate size)
- Avoid fmt.Sprintf in hot paths

---

## Phase 6: Concurrent Performance (Week 3)

### Issue: Concurrent Benchmark 1,606ns vs 921ns (Echo)

**Root Causes:**
- Lock contention in router
- Context pool contention
- Shared state access

### Optimization 6.1: Lock-Free Router Lookup

**File:** `bolt/core/router.go`

**Current:**
```go
type Router struct {
    mu sync.RWMutex
    staticRoutes map[string]Handler
    dynamicRoot  *node
}

func (r *Router) Lookup(method, path string) Handler {
    r.mu.RLock()
    defer r.mu.RUnlock()
    // ...
}
```

**Optimized:**
```go
import "sync/atomic"

type Router struct {
    staticRoutes atomic.Value // map[string]Handler (immutable)
    dynamicRoot  atomic.Value // *node (immutable)
}

func (r *Router) Lookup(method, path string) Handler {
    // No locks! Atomic load
    routes := r.staticRoutes.Load().(map[string]Handler)
    // ...
}

func (r *Router) Add(method, path string, handler Handler) {
    // Copy-on-write strategy
    oldRoutes := r.staticRoutes.Load().(map[string]Handler)
    newRoutes := make(map[string]Handler, len(oldRoutes)+1)
    for k, v := range oldRoutes {
        newRoutes[k] = v
    }
    newRoutes[method+":"+path] = handler
    r.staticRoutes.Store(newRoutes)
}
```

**Expected Improvement:** 1,606ns → 1,000ns (1.6x faster)

### Optimization 6.2: Per-CPU Context Pools

**File:** `bolt/core/context_pool.go`

**Optimized:**
```go
import "runtime"

type ContextPool struct {
    pools []*sync.Pool // One pool per CPU
}

func NewContextPool() *ContextPool {
    numCPU := runtime.NumCPU()
    pools := make([]*sync.Pool, numCPU)
    for i := range pools {
        pools[i] = &sync.Pool{
            New: func() interface{} {
                return &Context{
                    store: make(map[string]interface{}, 4),
                }
            },
        }
    }
    return &ContextPool{pools: pools}
}

func (p *ContextPool) Acquire() *Context {
    // Use goroutine ID or random selection to distribute load
    pid := fastrand() % len(p.pools)
    return p.pools[pid].Get().(*Context)
}
```

**Expected Improvement:** Reduce pool contention by 50%

---

## Phase 7: Response Writing Optimization (Week 4)

### Optimization 7.1: Pre-Compiled JSON Responses

**File:** `bolt/core/context.go`

**Add:**
```go
// Common response patterns as byte slices
var (
    jsonOKBytes        = []byte(`{"ok":true}`)
    jsonCreatedBytes   = []byte(`{"created":true}`)
    jsonDeletedBytes   = []byte(`{"deleted":true}`)
    json404Bytes       = []byte(`{"error":"Not Found"}`)
    json500Bytes       = []byte(`{"error":"Internal Server Error"}`)
)

// Fast path methods
func (c *Context) JSONOK() error {
    c.setContentTypeJSON()
    c.resp.WriteHeader(200)
    _, err := c.resp.Write(jsonOKBytes)
    return err
}

func (c *Context) JSONNotFound() error {
    c.setContentTypeJSON()
    c.resp.WriteHeader(404)
    _, err := c.resp.Write(json404Bytes)
    return err
}
```

**Expected Improvement:** 0 allocs for common responses

### Optimization 7.2: Optimize JSON Buffer Selection

**File:** `bolt/pool/buffers/json_buffer_pool.go`

**Current:** Fixed size tiers (512B, 8KB, 64KB)

**Optimized:**
```go
// Add 2KB tier for typical REST API responses
var (
    tinyPool   = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 512)) }}
    smallPool  = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 2048)) }}
    mediumPool = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 8192)) }}
    largePool  = sync.Pool{New: func() interface{} { return bytes.NewBuffer(make([]byte, 0, 65536)) }}
)

func AcquireJSONBuffer(sizeHint int) *bytes.Buffer {
    switch {
    case sizeHint <= 512:
        return tinyPool.Get().(*bytes.Buffer)
    case sizeHint <= 2048:
        return smallPool.Get().(*bytes.Buffer)
    case sizeHint <= 8192:
        return mediumPool.Get().(*bytes.Buffer)
    default:
        return largePool.Get().(*bytes.Buffer)
    }
}
```

**Expected Improvement:** Better size matching = fewer growth allocations

---

## Phase 8: Benchmark Validation (Week 4)

### Task 8.1: Run Competitive Benchmarks

```bash
cd /home/mirai/Documents/Programming/Projects/watt/bolt/benchmarks
go test -bench=. -benchmem -count=10 > optimized.txt
benchstat baseline.txt optimized.txt
```

### Task 8.2: Verify Targets

**Target Metrics:**
- Static routes: <500ns/op, <50 B/op
- Dynamic routes: <1,400ns/op, <100 B/op
- Middleware: <1,600ns/op, <100 B/op
- Concurrent: <1,000ns/op, <100 B/op

### Task 8.3: Create Performance Report

**File:** `bolt/docs/performance_report.md`

**Include:**
- Before/after comparison
- Percentage improvements
- Ranking vs competitors
- Remaining gaps (if any)

---

# PART 2: SHOCKWAVE LIBRARY OPTIMIZATION

## Phase 1: Server Concurrent Optimization (Week 5)

### Issue: Server Concurrent 31.7% Slower than fasthttp

**Location:** `shockwave/pkg/shockwave/http11/connection.go`

### Optimization 1.1: Profile Concurrent Server

```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave
go test -bench=BenchmarkServers_Concurrent/Shockwave -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof
```

**Expected Findings:**
- Lock contention in connection pool
- Buffer pool contention
- Goroutine creation overhead
- Memory allocation hotspots

### Optimization 1.2: Connection Pool Lock-Free Access

**File:** `shockwave/pkg/shockwave/http11/connection.go`

**Current Implementation:** Review connection state management

**Optimized:**
```go
import "sync/atomic"

type Connection struct {
    state   atomic.Int32 // StateNew, StateActive, StateIdle, StateClosed
    rwc     net.Conn
    bufr    *bufio.Reader
    bufw    *bufio.Writer
    lastUse atomic.Int64 // Unix timestamp
}

// Lock-free state transitions
func (c *Connection) SetState(new int32) {
    c.state.Store(new)
    c.lastUse.Store(time.Now().Unix())
}

func (c *Connection) GetState() int32 {
    return c.state.Load()
}
```

**Expected Improvement:** Reduce lock contention by 70%

### Optimization 1.3: Per-CPU Connection Pools

**File:** `shockwave/pkg/shockwave/server/server_shockwave.go`

**Add:**
```go
type Server struct {
    connPools []*connectionPool // One pool per CPU
}

type connectionPool struct {
    conns chan *Connection
    mu    sync.Mutex
}

func (s *Server) acquireConn() *Connection {
    pid := fastrand() % len(s.connPools)
    pool := s.connPools[pid]

    select {
    case conn := <-pool.conns:
        return conn
    default:
        return newConnection()
    }
}
```

**Expected Improvement:** 15,959ns → 11,000ns (1.45x faster)

---

## Phase 2: Response Buffer Optimization (Week 5)

### Issue: Response Memory 151 B/op vs 26 B/op (fasthttp)

**File:** `shockwave/pkg/shockwave/http11/response.go`

### Optimization 2.1: Use Pre-Compiled Status Lines

**Current:**
```go
func (w *ResponseWriter) WriteHeader(code int) {
    fmt.Fprintf(w.bufw, "HTTP/1.1 %d %s\r\n", code, StatusText(code))
}
```

**Problem:** fmt.Fprintf allocates, StatusText allocates

**Optimized:**
```go
var statusLines = map[int][]byte{
    200: []byte("HTTP/1.1 200 OK\r\n"),
    201: []byte("HTTP/1.1 201 Created\r\n"),
    204: []byte("HTTP/1.1 204 No Content\r\n"),
    301: []byte("HTTP/1.1 301 Moved Permanently\r\n"),
    302: []byte("HTTP/1.1 302 Found\r\n"),
    304: []byte("HTTP/1.1 304 Not Modified\r\n"),
    400: []byte("HTTP/1.1 400 Bad Request\r\n"),
    401: []byte("HTTP/1.1 401 Unauthorized\r\n"),
    403: []byte("HTTP/1.1 403 Forbidden\r\n"),
    404: []byte("HTTP/1.1 404 Not Found\r\n"),
    405: []byte("HTTP/1.1 405 Method Not Allowed\r\n"),
    500: []byte("HTTP/1.1 500 Internal Server Error\r\n"),
    502: []byte("HTTP/1.1 502 Bad Gateway\r\n"),
    503: []byte("HTTP/1.1 503 Service Unavailable\r\n"),
}

func (w *ResponseWriter) WriteHeader(code int) {
    if line, ok := statusLines[code]; ok {
        w.bufw.Write(line) // Zero allocations
    } else {
        // Fallback for uncommon status codes
        fmt.Fprintf(w.bufw, "HTTP/1.1 %d %s\r\n", code, StatusText(code))
    }
}
```

**Expected Improvement:** 151 B/op → 50 B/op (3x reduction)

### Optimization 2.2: Pre-Compiled Common Headers

**File:** `shockwave/pkg/shockwave/http11/constants.go`

**Add:**
```go
var commonHeaders = map[string][]byte{
    "Content-Type":   []byte("Content-Type: "),
    "Content-Length": []byte("Content-Length: "),
    "Server":         []byte("Server: "),
    "Date":           []byte("Date: "),
    "Connection":     []byte("Connection: "),
}

var commonHeaderValues = map[string][]byte{
    "application/json":      []byte("application/json\r\n"),
    "text/html":             []byte("text/html; charset=utf-8\r\n"),
    "text/plain":            []byte("text/plain; charset=utf-8\r\n"),
    "keep-alive":            []byte("keep-alive\r\n"),
    "close":                 []byte("close\r\n"),
}

func (w *ResponseWriter) SetContentType(contentType string) {
    if val, ok := commonHeaderValues[contentType]; ok {
        w.bufw.Write(commonHeaders["Content-Type"])
        w.bufw.Write(val)
    } else {
        fmt.Fprintf(w.bufw, "Content-Type: %s\r\n", contentType)
    }
}
```

**Expected Improvement:** 0 allocs for common headers

---

## Phase 3: Client Memory Optimization (Week 6)

### Issue: Client Memory 2,843 B/op vs 1,865 B/op (fasthttp)

**File:** `shockwave/pkg/shockwave/client/client.go`

### Optimization 3.1: Optimize Request Structure

**Current:** Review Request struct layout

**Optimized:**
```go
type Request struct {
    // Hot fields first (cache line optimization)
    method      []byte
    uri         []byte
    proto       []byte

    // Inline header storage (covers 90% of requests)
    headersBuf  [12]headerPair // Increased from 6 to 12
    headersMap  map[string]string // Overflow only

    // Body
    body        []byte

    // Cold fields last
    host        string
    contentType string
}
```

**Changes:**
- Double inline header storage (6 → 12)
- Reorder fields for cache locality
- Use byte slices for method/URI (no string allocations)

**Expected Improvement:** 2,843 B/op → 2,000 B/op (1.4x reduction)

### Optimization 3.2: Connection Reuse Optimization

**File:** `shockwave/pkg/shockwave/client/pool.go`

**Add:**
```go
// Increase connection pool size for high concurrency
type connectionPool struct {
    conns     chan *clientConn
    maxConns  int
    idleConns int32 // Atomic counter
}

func newConnectionPool(maxConns int) *connectionPool {
    return &connectionPool{
        conns:    make(chan *clientConn, maxConns),
        maxConns: maxConns,
    }
}

func (p *connectionPool) acquire() *clientConn {
    select {
    case conn := <-p.conns:
        atomic.AddInt32(&p.idleConns, -1)
        return conn
    default:
        return nil
    }
}

func (p *connectionPool) release(conn *clientConn) {
    select {
    case p.conns <- conn:
        atomic.AddInt32(&p.idleConns, 1)
    default:
        conn.Close() // Pool full, close connection
    }
}
```

**Expected Improvement:** Better connection reuse = fewer allocations

---

## Phase 4: Request Parsing Optimization (Week 6)

### Note: Already Excellent (5.2x faster than fasthttp)

**Current Performance:** 384 ns/op, 202.93 MB/s throughput

**Minor Improvements:**

### Optimization 4.1: SIMD Header Parsing (Advanced)

**File:** `shockwave/pkg/shockwave/http11/parser.go`

**Add (Optional):**
```go
// Use SIMD for finding header delimiters
// Requires: golang.org/x/sys/cpu
import "golang.org/x/sys/cpu"

func findHeaderEnd(data []byte) int {
    if cpu.X86.HasAVX2 {
        return findHeaderEndSIMD(data) // AVX2 implementation
    }
    return findHeaderEndScalar(data) // Fallback
}
```

**Expected Improvement:** 384ns → 300ns (1.3x faster)

---

## Phase 5: Buffer Pool Hit Rate (Week 7)

### Optimization 5.1: Tune Buffer Pool Sizes

**File:** `shockwave/buffer_pool.go`

**Current:** 6 size classes (2KB, 4KB, 8KB, 16KB, 32KB, 64KB)

**Analyze Usage:**
```go
// Add metrics collection
type BufferPoolMetrics struct {
    sizeHits map[int]int64 // Track which sizes are used most
}

func (p *BufferPool) Acquire(size int) *bytes.Buffer {
    // Track size distribution
    atomic.AddInt64(&p.metrics.sizeHits[size], 1)
    // ...
}
```

**Adjust sizes based on real usage patterns**

### Optimization 5.2: Pre-Warm Pools

**File:** `shockwave/buffer_pool.go`

**Add:**
```go
func (p *BufferPool) Warmup(count int) {
    for _, pool := range p.pools {
        for i := 0; i < count; i++ {
            buf := pool.Get().(*bytes.Buffer)
            pool.Put(buf)
        }
    }
}

// In server initialization
func NewServer(config *Config) *Server {
    // ...
    bufferPool.Warmup(1000) // Pre-allocate 1000 buffers per size
    return server
}
```

**Expected Improvement:** Hit rate 95% → 98%

---

## Phase 6: Socket Optimization Validation (Week 7)

### Task 6.1: Verify Socket Options Are Applied

**File:** `shockwave/pkg/shockwave/socket/tuning_linux.go`

**Add Validation:**
```go
func ValidateSocketOptions(conn net.Conn) error {
    tc, ok := conn.(*net.TCPConn)
    if !ok {
        return nil
    }

    rawConn, err := tc.SyscallConn()
    if err != nil {
        return err
    }

    var sockErr error
    rawConn.Control(func(fd uintptr) {
        // Verify TCP_NODELAY
        val, err := syscall.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY)
        if err != nil || val != 1 {
            sockErr = fmt.Errorf("TCP_NODELAY not set: %v", err)
            return
        }

        // Verify TCP_QUICKACK (Linux)
        val, err = syscall.GetsockoptInt(int(fd), syscall.IPPROTO_TCP, 12) // TCP_QUICKACK
        if err != nil || val != 1 {
            sockErr = fmt.Errorf("TCP_QUICKACK not set: %v", err)
        }
    })

    return sockErr
}
```

### Task 6.2: Benchmark Socket Options Impact

```bash
# Run with and without socket optimizations
go test -bench=BenchmarkServers_Concurrent -tags=nosocketopt
go test -bench=BenchmarkServers_Concurrent
benchstat nosocketopt.txt withsocketopt.txt
```

**Expected:** Socket opts should provide 10-15% improvement

---

## Phase 7: HTTP/2 and HTTP/3 Quick Wins (Week 8)

### Optimization 7.1: HTTP/2 Frame Pooling

**File:** `shockwave/pkg/shockwave/http2/frame.go`

**Add:**
```go
var framePool = sync.Pool{
    New: func() interface{} {
        return &Frame{
            payload: make([]byte, 0, 16384), // Max frame size
        }
    },
}

func AcquireFrame() *Frame {
    return framePool.Get().(*Frame)
}

func ReleaseFrame(f *Frame) {
    f.Reset()
    framePool.Put(f)
}
```

### Optimization 7.2: HTTP/3 QPACK Optimization

**File:** `shockwave/pkg/shockwave/http3/qpack/encoder.go`

**Optimize:**
- Use pre-compiled static table
- Buffer pooling for encoding
- Zero-copy header field references

---

## Phase 8: Final Validation (Week 8)

### Task 8.1: Comprehensive Benchmarking

```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave
go test -bench=. -benchmem -count=10 ./... > optimized_shockwave.txt
benchstat baseline_shockwave.txt optimized_shockwave.txt
```

### Task 8.2: Competitive Comparison

**Run against fasthttp and net/http:**
```bash
cd benchmarks/competitors
go test -bench=. -benchmem -count=10 > comparison.txt
```

**Verify:**
- Server concurrent: ≤ fasthttp (currently 31.7% slower)
- Response memory: ≤ 50 B/op (currently 151 B/op)
- Client memory: ≤ 2,000 B/op (currently 2,843 B/op)
- Maintain strengths: Parsing speed, throughput

---

# IMPLEMENTATION STRATEGY

## Parallel Execution Plan

### Team A: Bolt Optimization (4 weeks)
- Week 1: Profiling and analysis
- Week 2: Context pool + Shockwave adapter
- Week 3: Parameters + Middleware + Concurrent
- Week 4: Response writing + Validation

### Team B: Shockwave Optimization (4 weeks)
- Week 5: Server concurrent + Response buffers
- Week 6: Client memory + Parsing improvements
- Week 7: Buffer pool tuning + Socket validation
- Week 8: HTTP/2/3 + Final validation

### Total Timeline: 8 weeks if parallel, 8 weeks if sequential

---

## Risk Mitigation

### Regression Testing
- Run full benchmark suite after each change
- Use benchstat to validate improvements
- Maintain separate branches for each optimization
- Revert if regression >5%

### Compatibility
- Maintain net/http compatibility in Shockwave
- Maintain Bolt API stability
- Add build tags for experimental optimizations
- Document breaking changes

### Performance Targets
- Bolt: #1 in all categories
- Shockwave: Match or beat fasthttp
- Both: Zero allocations in hot paths

---

## Success Criteria

### Bolt Framework
✅ Static routes: <500ns/op, <50 B/op
✅ Dynamic routes: <1,400ns/op, <100 B/op
✅ Middleware: <1,600ns/op, <100 B/op
✅ Concurrent: <1,000ns/op, <100 B/op
✅ Overall: #1 vs Gin, Echo, Fiber

### Shockwave Library
✅ Server concurrent: ≤ fasthttp (currently 31.7% slower)
✅ Response memory: ≤ 50 B/op (currently 151 B/op)
✅ Client memory: ≤ 2,000 B/op (currently 2,843 B/op)
✅ Maintain parsing leadership (5.2x faster)
✅ Overall: #1 vs fasthttp, net/http

---

## Deliverables

### Week 4 (Bolt Complete)
1. Optimized Bolt codebase
2. Performance report with before/after metrics
3. Competitive benchmark results
4. Updated documentation

### Week 8 (Shockwave Complete)
1. Optimized Shockwave codebase
2. Performance report with before/after metrics
3. Competitive benchmark results
4. Updated documentation

### Final
1. Combined performance analysis
2. Architecture documentation
3. Optimization guide for future development
4. Blog post announcing performance achievements

---

**This roadmap provides a clear path to #1 performance for both libraries.**
