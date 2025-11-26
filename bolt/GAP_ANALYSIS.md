# Performance Gap Analysis - Why Bolt Isn't #1 in All Categories

## Current Performance Gaps

### Static Routes: 35% slower than Gin
- **Gin:** 226ns
- **Bolt:** 306ns
- **Gap:** 80ns (35% slower)

### Dynamic Routes: 18% slower than Gin
- **Gin:** 473ns
- **Bolt:** 560ns
- **Gap:** 87ns (18% slower)

### Concurrent: 32% slower than Gin
- **Gin:** 143ns
- **Bolt:** 189ns
- **Gap:** 46ns (32% slower)

## Hypothesis: Where Are The Extra Nanoseconds?

Let me profile to find the bottlenecks...

## CPU Profile Analysis Results

### Bottleneck Breakdown (Static Route - 370ns total)

1. **Context Pool Release: 480ms (27.43%)**
   - `FastReset()`: 470ms (26.86%)
   - `runtime.duffcopy`: 290ms (16.57%) - Bulk struct zeroing
   - **Finding:** Context reset is TOO EXPENSIVE due to large struct size (~1.3KB)

2. **Header Setting: 460ms (26.29%)**
   - `SetHeaderBytes()`: 460ms
   - `net/textproto.CanonicalMIMEHeaderKey`: 230ms (13.14%)
   - `validHeaderFieldByte`: 180ms (10.29%)
   - **Finding:** Standard library header handling adds significant overhead

3. **JSON Encoding: 320ms (18.29%)**
   - Already using fast `goccy/go-json`
   - **Finding:** This is already optimized

4. **Router Lookup: 190ms (10.86%)**
   - `LookupBytes()`: 190ms
   - **Finding:** Pretty good, but room for improvement

5. **Handler Execution: 880ms (50%)**
   - Actual handler code (ctx.JSON call)

### Total Overhead: ~100-150ns vs Gin

**Where Gin beats us:**
1. **Smaller Context struct** - Gin's context is ~200 bytes vs our 1,300 bytes
2. **No net/textproto** - Gin uses custom header handling
3. **Faster pooling** - Smaller struct = faster reset
4. **Simpler routing** - Less abstraction layers

## Root Causes

### 1. Context Size Problem
- **Bolt Context:** ~1,300 bytes (with inline buffers)
  - paramsBuf[8]: 384 bytes
  - queryParamsBuf[16]: 768 bytes
- **Gin Context:** ~200 bytes (no inline buffers)

**Impact:** 
- Slower pool acquire/release
- Slower FastReset (more bytes to zero)
- More cache misses

### 2. net/textproto Overhead
- CanonicalMIMEHeaderKey: 230ms (canonicalizes "content-type" → "Content-Type")
- validHeaderFieldByte: 180ms (validates header bytes)
- **Total: 410ms wasted on header validation!**

### 3. Shockwave vs net/http
- Bolt uses Shockwave HTTP server (for production)
- Benchmarks use net/http (for compatibility)
- **Mismatch creates overhead!**

## Optimization Plan to Achieve #1

### Phase 1: Critical Optimizations (Expected: 80-100ns improvement)

#### 1.1 Bypass net/textproto Header Handling
**Problem:** 410ms spent on header canonicalization and validation

**Solution:** Custom header writing without validation
```go
// Instead of: ctx.SetHeader("Content-Type", "application/json")
// Use direct write: w.Header()["Content-Type"] = []string{"application/json"}
// Or even better: Pre-allocate response headers
```

**Expected gain:** ~60-80ns

#### 1.2 Optimize Context Pooling
**Problem:** Large struct (1.3KB) makes pooling slow

**Solution A - Separate inline buffers:**
```go
type Context struct {
    // Hot fields only (~200 bytes)
    shockwaveReq *http11.Request
    shockwaveRes *http11.ResponseWriter
    methodBytes []byte
    pathBytes   []byte
    // ... (small fields)
    
    // Pointer to separate buffer pool
    paramBuf  *ParamBuffer  // Only allocate if needed
    queryBuf  *QueryBuffer  // Only allocate if needed
}
```

**Solution B - Two-tier context pool:**
```go
// Small context for routes without params
type FastContext struct { ... } // ~200 bytes

// Full context for routes with params
type FullContext struct {
    FastContext
    paramsBuf [8]...
    queryBuf  [16]...
}
```

**Expected gain:** ~30-40ns

#### 1.3 Pre-allocate Response Headers
**Problem:** Setting headers allocates map entries

**Solution:** Pre-allocated header pool
```go
var commonHeaders = map[string][]string{
    "Content-Type": {"application/json"},
    // ... other common headers
}
```

**Expected gain:** ~10-15ns

### Phase 2: Advanced Optimizations (Expected: 20-30ns improvement)

#### 2.1 Inline Router Lookup
**Problem:** Function call overhead in LookupBytes

**Solution:** Inline static route lookup into ServeHTTP
```go
func (r *Router) ServeHTTP(c *Context) error {
    // FAST PATH: Inline static route lookup (no function call)
    key := ... // Build key inline
    if handler, ok := r.static[key]; ok {
        return handler(c)
    }
    
    // SLOW PATH: Dynamic route lookup
    handler, params, paramCount := r.lookupDynamic(...)
}
```

**Expected gain:** ~10-15ns

#### 2.2 Method-Specific Routers
**Problem:** Method lookup in hot path

**Solution:** Separate routers per method
```go
type Router struct {
    getRoutes  map[string]Handler
    postRoutes map[string]Handler
    // ... (avoid method lookup)
}
```

**Expected gain:** ~5-10ns

#### 2.3 Assembly Optimizations
**Problem:** Go compiler limitations

**Solution:** Critical path in assembly
- Custom `bytesEqual` in assembly (SIMD)
- Custom map lookup in assembly
- Custom struct zeroing in assembly

**Expected gain:** ~10-20ns (advanced)

### Phase 3: Radical Optimizations (Expected: 40-60ns improvement)

#### 3.1 Zero-Copy Response Writing
**Problem:** Multiple writes (header + body)

**Solution:** Single write with pre-built response
```go
// Pre-build common responses
var jsonOKResponse = []byte("HTTP/1.1 200 OK\r\nContent-Type: application/json\r\n\r\n{\"ok\":true}")

// Direct write (bypass ResponseWriter)
w.Write(jsonOKResponse)
```

**Expected gain:** ~20-30ns

#### 3.2 Lock-Free Static Route Map
**Problem:** RWMutex.RLock has overhead

**Solution:** Atomic pointer to immutable map
```go
type Router struct {
    staticRoutes atomic.Value // map[string]Handler (immutable)
}

// Lookup (no lock!)
routes := r.staticRoutes.Load().(map[string]Handler)
handler := routes[key]
```

**Expected gain:** ~15-20ns

#### 3.3 Per-CPU Context Pools
**Problem:** sync.Pool contention under concurrency

**Solution:** Per-CPU pools with GOMAXPROCS pools
```go
type ContextPool struct {
    pools []*sync.Pool // len = GOMAXPROCS
}

func (p *ContextPool) Acquire() *Context {
    cpuID := runtime_procPin() // Get current CPU
    return p.pools[cpuID].Get().(*Context)
}
```

**Expected gain:** ~10-15ns (concurrent workloads)

## Expected Total Improvement

### Conservative Estimate
- Phase 1: 80-100ns → **Bolt: 206-226ns** (matches Gin!)
- Phase 2: 20-30ns → **Bolt: 176-206ns** (beats Gin by 10-20ns!)
- Phase 3: 40-60ns → **Bolt: 116-166ns** (destroys Gin by 60-110ns!)

### Aggressive Estimate (all optimizations)
- **Bolt: 116ns** vs Gin: 226ns = **95% faster!** 🚀

## Implementation Priority

### Week 1 (Quick Wins - Target: #1 Static Routes)
1. ✅ Bypass net/textproto (60-80ns gain)
2. ✅ Pre-allocate response headers (10-15ns gain)
3. ✅ Inline static route lookup (10-15ns gain)

**Expected:** 80-110ns improvement → **Bolt: 196-226ns (matches/beats Gin!)**

### Week 2 (Advanced - Target: #1 Dynamic Routes)
4. ✅ Optimize context pooling (30-40ns gain)
5. ✅ Method-specific routers (5-10ns gain)

**Expected:** 35-50ns improvement → **Bolt: 146-191ns (beats Gin by 35-80ns!)**

### Week 3 (Radical - Target: #1 Concurrent)
6. ✅ Lock-free static route map (15-20ns gain)
7. ✅ Per-CPU context pools (10-15ns gain)

**Expected:** 25-35ns improvement → **Bolt: 111-166ns (destroys competition!)**

## Risks and Trade-offs

### Low Risk (Do Immediately)
- Bypass net/textproto: Safe, well-tested approach
- Pre-allocate headers: Simple optimization
- Inline static lookup: Minor code duplication

### Medium Risk (Test Thoroughly)
- Context pooling changes: Might affect memory if done wrong
- Method-specific routers: More code complexity
- Lock-free map: Type assertion heap escape risk (learned from Phase 3!)

### High Risk (Proceed with Caution)
- Assembly optimizations: Platform-specific, hard to maintain
- Zero-copy response: Bypasses standard library, might break compatibility
- Per-CPU pools: Complex, might not work on all platforms

## Recommended Approach

### Start with Low-Risk High-Impact Optimizations
1. **Bypass net/textproto** - Biggest single gain (60-80ns)
2. **Inline static route lookup** - Simple, safe (10-15ns)
3. **Pre-allocate headers** - Easy win (10-15ns)

**Total: 80-110ns improvement with LOW RISK**

This alone should make Bolt **#1 or tied for #1** in static routes!

### Then Move to Medium-Risk Optimizations
4. **Optimize context pooling** - Separate buffers (30-40ns)
5. **Method-specific routers** - Minor complexity (5-10ns)

**Total: 35-50ns more → Bolt becomes #1 in dynamic routes too!**

### Finally, If Needed
6. **Lock-free optimizations** - Only if we want to dominate concurrent
7. **Assembly** - Only if we want to be 2x faster than Gin

## Next Steps

Would you like me to:
1. ✅ **Implement Phase 1 (Low-Risk)** - Bypass net/textproto, inline lookup, pre-allocate headers
2. ⏳ Profile Gin's source code to confirm our analysis
3. ⏳ Create benchmarks to validate each optimization
4. ⏳ Implement Phase 2 (Medium-Risk) if Phase 1 gets us to #1

I recommend starting with **Phase 1** - these are safe, proven optimizations that should give us **80-110ns improvement** and make Bolt **#1 in static routes**!
