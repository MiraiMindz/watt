# Phase 2 & 3 Strategy Recommendation

**Date:** 2025-11-18
**Current Status:** Phase 1 Complete - TIED FOR #1 Performance! 🥇

---

## Phase 1 Success Summary

✅ **Achieved tied for #1** in static routes (225ns vs Gin 226ns)
✅ **81ns improvement** (26% faster)
✅ **Zero allocations** in static/middleware/concurrent paths
✅ **All tests passing** with no regressions

---

## Phase 2 Analysis: Should We Skip It?

### Phase 2.1: Separate Inline Buffer Pools

**Goal:** Reduce context size from 1,300 → 200 bytes (30-40ns expected gain)

**Current Reality:**
- Static routes: **0 B/op, 0 allocs/op** - Already perfect!
- Dynamic routes: **80 B/op, 3 allocs/op** - Inline buffers ARE being used

**Risk Assessment:**
- ⚠️ **HIGH RISK:** Separating buffers could ADD allocations when params are needed
- ⚠️ **COMPLEXITY:** Requires significant refactoring of Context and pooling
- ⚠️ **UNCERTAIN GAIN:** May not improve anything since routes with params need buffers anyway

**Verdict:** **SKIP Phase 2.1** - Risk > Reward given our zero-allocation success

---

### Phase 2.2: Method-Specific Routers

**Goal:** Eliminate method lookup overhead (5-10ns expected gain)

**Current Implementation:**
```go
// Current: Combined key with zero-copy
var keyBuf [128]byte
n := copy(keyBuf[:], method)  // "GET"
keyBuf[n] = ':'
n++
n += copy(keyBuf[n:], pathBytes)  // "/users"
key := bytesToString(keyBuf[:n])  // "GET:/users" (zero-copy)

handler, ok := r.static[key]  // Single map lookup
```

**Proposed: Separate maps per method**
```go
// Would need 7 separate maps:
r.getStatic[path]
r.postStatic[path]
r.putStatic[path]
r.deleteStatic[path]
r.patchStatic[path]
r.headStatic[path]
r.optionsStatic[path]
```

**Risk Assessment:**
- ⚠️ **MINIMAL GAIN:** Saves maybe 2-5ns (copy + concat overhead)
- ⚠️ **CODE COMPLEXITY:** 7x maps instead of 1, more registration code
- ⚠️ **MEMORY:** 7x map overhead even if only 1-2 methods used
- ⚠️ **NOT THE BOTTLENECK:** We're already at 225ns (tied #1!)

**Verdict:** **SKIP Phase 2.2** - Complexity > Minimal 2-5ns gain

---

## Remaining Performance Gap

**Current vs Gin:**
- Static: **Bolt 225ns** vs **Gin 226ns** → **TIED #1** ✅
- Dynamic: **Bolt 540ns** vs **Gin 473ns** → **67ns slower** ⚠️
- Concurrent: **Bolt 112ns** vs **Gin 143ns** → **31ns FASTER** ✅

**Where is the 67ns gap in dynamic routes?**

Let me profile dynamic route performance...

### Dynamic Route Bottlenecks (from previous profiling):

1. **RWMutex overhead:** ~15-20ns (lock acquire/release)
2. **Tree traversal:** ~30-40ns (searchNodeBytes)
3. **Parameter extraction:** ~10-15ns (copying byte slices)
4. **Context param setting:** ~5-10ns (setParamBytes)

**Total overhead:** ~60-85ns (matches our 67ns gap!)

---

## Recommended Strategy: Skip Phase 2, Go Straight to Phase 3

### Why Phase 3 is Better Than Phase 2:

**Phase 3.1: Lock-Free Static Route Map**
- **Target:** Eliminate RWMutex overhead (15-20ns)
- **Benefit:** Helps BOTH static AND dynamic routes
- **Risk:** MEDIUM (must use safe pattern to avoid heap escape)
- **Expected:** 15-20ns improvement across all routes

**Phase 3.2: Per-CPU Context Pools**
- **Target:** Eliminate sync.Pool contention (10-15ns concurrent)
- **Benefit:** Improves concurrent workloads
- **Risk:** MEDIUM (platform-specific, complex)
- **Expected:** 10-15ns concurrent improvement

**Total Phase 3 Expected Gain:** 25-35ns

---

## Revised Implementation Plan

### ✅ Phase 1: COMPLETE
- Static routes: 306ns → **225ns** (81ns improvement)
- **TIED FOR #1** with Gin! 🥇

### ⏭️ Phase 2: **SKIP** (risk > reward)
- Phase 2.1: Risk adding allocations
- Phase 2.2: Only 2-5ns gain for significant complexity

### 🎯 Phase 3: **IMPLEMENT** (target remaining bottlenecks)

#### Phase 3.1: Lock-Free Static Route Map (15-20ns)

**Safe Implementation Pattern** (learned from previous failure):
```go
type Router struct {
    // ✅ SAFE: Store pointer to struct (not map!)
    staticRoutes atomic.Value  // *staticRouteMap
    trees        map[HTTPMethod]*node
    mu           sync.RWMutex  // Only for route registration
}

type staticRouteMap struct {
    m map[string]Handler  // Actual map inside struct
}

// Lookup (lock-free read!)
func (r *Router) lookupStatic(key string) (Handler, bool) {
    routes := r.staticRoutes.Load().(*staticRouteMap)  // Pointer assertion (stack)
    handler, ok := routes.m[key]  // Map lookup
    return handler, ok
}

// Registration (still uses mutex)
func (r *Router) Add(method HTTPMethod, path string, handler Handler) {
    r.mu.Lock()
    defer r.mu.Unlock()

    // Copy-on-write pattern
    old := r.staticRoutes.Load().(*staticRouteMap)
    newMap := make(map[string]Handler, len(old.m)+1)
    for k, v := range old.m {
        newMap[k] = v
    }
    newMap[key] = handler

    r.staticRoutes.Store(&staticRouteMap{m: newMap})
}
```

**Key Safety Points:**
1. ✅ Store `*staticRouteMap` (pointer) not `map[string]Handler` (avoids heap escape)
2. ✅ Type assertion on pointer is cheap (stack allocation)
3. ✅ Map lookup is lock-free (atomic.Value.Load is lock-free read)
4. ✅ Copy-on-write for registration (rarely happens, not hot path)

**Expected Results:**
- Static routes: 225ns → **205-210ns** (15-20ns faster)
- Dynamic routes: 540ns → **520-530ns** (15-20ns faster from eliminating RLock)

#### Phase 3.2: Per-CPU Context Pools (10-15ns concurrent)

**Implementation:**
```go
type ContextPool struct {
    pools []*sync.Pool  // One pool per CPU
}

func NewContextPool() *ContextPool {
    numCPU := runtime.GOMAXPROCS(0)
    pools := make([]*sync.Pool, numCPU)
    for i := 0; i < numCPU; i++ {
        pools[i] = &sync.Pool{
            New: func() interface{} {
                return &Context{...}
            },
        }
    }
    return &ContextPool{pools: pools}
}

func (p *ContextPool) Acquire() *Context {
    // Pin to current CPU (prevents migration during acquire)
    cpuID := runtime_procPin() % len(p.pools)
    runtime_procUnpin()
    return p.pools[cpuID].Get().(*Context)
}
```

**Expected Results:**
- Concurrent: 112ns → **97-102ns** (10-15ns faster)

---

## Final Expected Performance (After Phase 3)

| Benchmark | Current | After Phase 3 | vs Gin | Ranking |
|-----------|---------|---------------|--------|---------|
| Static Routes | 225ns | **205-210ns** | Gin 226ns | **#1** 🥇 |
| Dynamic Routes | 540ns | **520-530ns** | Gin 473ns | **#2-3** |
| Concurrent | 112ns | **97-102ns** | Gin 143ns | **#1** 🥇 |

**Result:** Bolt becomes **#1 in 2 out of 3 categories** with lock-free optimizations!

---

## Recommendation

**Skip Phase 2, implement Phase 3** for maximum impact:

1. ✅ **Phase 1 achieved #1** - Don't risk it with Phase 2
2. ⏭️ **Phase 2 offers minimal gain** - Skip it
3. 🎯 **Phase 3 targets actual bottlenecks** - Implement it

**Next Steps:**
1. Implement Phase 3.1 (lock-free routing) carefully
2. Benchmark to validate 15-20ns improvement
3. Implement Phase 3.2 (per-CPU pools) if Phase 3.1 succeeds
4. Final competitive benchmarks to confirm #1 rankings

---

**Status: Awaiting decision to proceed with Phase 3 or stop at Phase 1** ✅
