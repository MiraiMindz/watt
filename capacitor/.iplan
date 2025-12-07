# Capacitor Implementation Plan

## Overview
Capacitor is Watt's high-performance data access layer optimized for framework internals, route storage, and metadata management. Like a capacitor stores electrical charge for quick release, Capacitor stores and retrieves framework data with minimal latency and maximum efficiency.

## Core Principles
- **Cache-Friendly Structures**: Data layouts optimized for CPU cache
- **Zero-Copy Access**: Direct memory access where possible
- **Predictable Performance**: No hidden allocations or GC pressure
- **SIMD-Ready**: Vectorizable operations for batch queries

## Phase 1: Foundation (Week 1-2)

### 1.1 Core Data Structures
```go
// Route storage with arena backing
type RouteCapacitor struct {
    arena        *arena.Arena     // Long-lived arena from Electron
    staticRoutes *FlatHashMap     // O(1) static route lookup
    dynRoutes    *CompactRadix    // Cache-friendly radix tree
    metadata     *DenseArray      // Packed route metadata
}

// Flat hash map for static routes
type FlatHashMap struct {
    keys   []string    // Contiguous key storage
    values []Handler   // Contiguous handler storage
    hashes []uint64    // Pre-computed hashes
}

// Compact radix tree for dynamic routes
type CompactRadix struct {
    nodes    []RadixNode  // Node pool
    edges    []Edge       // Edge pool
    params   []Param      // Parameter pool
    freeList []uint32     // Free node indices
}
```

### 1.2 Memory Management Integration
- Integrate with Electron's arena allocators
- Implement pool-based allocation strategies
- Design zero-copy access patterns
- Create benchmark suite for memory patterns

### 1.3 Two-Step Route Analysis
```go
type RouteAnalyzer struct {
    staticCount  int
    dynamicCount int
    paramCount   int
    maxDepth     int
}

// Phase 1: Count everything
func (a *RouteAnalyzer) AnalyzeRoute(path string, method string)

// Phase 2: Allocate exact sizes
func (c *RouteCapacitor) Allocate(analysis RouteAnalysis)
```

## Phase 2: Core Implementation (Week 3-4)

### 2.1 FlatHashMap Implementation
- Implement collision-free hash function
- Design resize-free structure with pre-allocation
- Add SIMD-accelerated batch lookups
- Optimize for cache line alignment

### 2.2 CompactRadix Implementation
- Design cache-optimized node layout
- Implement edge compression
- Add parameter extraction logic
- Optimize traversal for CPU branch prediction

### 2.3 Middleware Storage
```go
type MiddlewareCapacitor struct {
    chains      []CompiledChain  // Pre-compiled chains
    registry    *DenseArray      // Middleware registry
    routeMap    []uint32         // Route → Chain mapping
}
```

## Phase 3: Advanced Features (Week 5-6)

### 3.1 Metadata Cache
```go
type MetadataCapacitor struct {
    arena    *arena.Arena
    docs     *CompactMap[string, RouteDoc]
    tags     *StringInterner      // Deduplicated strings
    params   *ParamCache          // Path parameter metadata
}
```

### 3.2 SIMD Optimizations
- Implement vectorized string comparisons
- Add batch hash computations
- Design parallel route lookups
- Create SIMD fallbacks for non-AVX systems

### 3.3 Lock-Free Operations
- Implement RCU (Read-Copy-Update) for updates
- Design wait-free readers
- Add epoch-based memory reclamation
- Create concurrent test suite

## Phase 4: Integration & Testing (Week 7-8)

### 4.1 Bolt Integration
- Replace existing router storage
- Integrate with middleware system
- Add configuration API
- Maintain backward compatibility

### 4.2 Performance Testing
- Benchmark vs existing router
- Profile cache misses and branch mispredictions
- Optimize hot paths based on profiling
- Create performance regression tests

### 4.3 Production Hardening
- Add panic recovery
- Implement capacity monitoring
- Create diagnostic endpoints
- Add observability hooks

## API Design

### Initialization
```go
capacitor := capacitor.New(capacitor.Config{
    EnableArenas:     true,
    PreallocateSize:  1000,  // Expected route count
    EnableSIMD:       runtime.GOARCH == "amd64",
    CacheLineSize:    64,
})

app := bolt.New(bolt.WithCapacitor(capacitor))
```

### Internal API
```go
// Fast route lookup (zero allocations)
handler, params := capacitor.Lookup(method, path)

// Batch operations (SIMD-accelerated)
results := capacitor.BatchLookup(requests)

// Metadata access
doc := capacitor.GetDoc(routeID)

// Statistics
stats := capacitor.Stats() // Memory usage, hit rates, etc.
```

## Performance Targets
- Static route lookup: <100ns
- Dynamic route lookup: <200ns
- Zero allocations for lookups
- 75% reduction in route storage memory
- Support for 100K+ routes without degradation

## Testing Strategy
1. Unit tests for each data structure
2. Fuzz testing for route patterns
3. Concurrency stress tests
4. Memory leak detection
5. Performance regression tests
6. Integration tests with Bolt

## Documentation Requirements
- API reference documentation
- Performance tuning guide
- Migration guide from standard router
- Benchmark comparison results
- Architecture decision records (ADRs)

## Success Metrics
- [ ] Static route lookup <100ns
- [ ] Zero allocation lookups achieved
- [ ] 75% memory reduction vs baseline
- [ ] SIMD optimizations provide 2x speedup for batch ops
- [ ] All integration tests passing
- [ ] Production deployment ready

## Dependencies
- Electron package (arena allocators, utilities)
- SIMD libraries (golang.org/x/sys/cpu)
- Testing frameworks
- Benchmark tools

## Risk Mitigation
- **Risk**: SIMD not available on all architectures
  - **Mitigation**: Automatic fallback to scalar operations
- **Risk**: Arena allocator instability
  - **Mitigation**: Optional arena usage with standard allocator fallback
- **Risk**: Lock-free complexity
  - **Mitigation**: Start with mutex-based, optimize later
- **Risk**: Breaking API changes
  - **Mitigation**: Maintain compatibility layer during transition