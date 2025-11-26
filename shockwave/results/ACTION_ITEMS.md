# Shockwave Performance Optimization - Action Items

## Priority 0: Critical Fixes (Week 1)

### 1. Fix Keep-Alive Connection Reuse Bug
**Status**: CRITICAL - Tests failing
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http11/connection.go`
**Issue**:
- `TestConnectionKeepAlive` fails - handler only called once instead of twice
- `TestConnectionMaxRequests` fails - same issue
- Connection reuse not working correctly

**Debug Steps**:
```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http11
go test -v -run=TestConnectionKeepAlive
go test -v -run=TestConnectionMaxRequests
```

**Expected Fix**:
- Investigate `Connection.Serve()` method
- Check request counter and loop logic
- Verify EOF handling
- Ensure connection stays open between requests

**Validation**:
- [ ] Tests pass
- [ ] Manual test with keep-alive client
- [ ] Benchmark shows improved performance

**Estimated Time**: 1-2 days
**Assignee**: TBD
**Due Date**: Nov 15, 2025

---

### 2. Fix Benchmark OOM Crash
**Status**: CRITICAL - Blocks benchmark suite
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http11/keepalive_bench_test.go:21`
**Issue**:
- `BenchmarkKeepAliveReuse` tries to allocate ~40GB
- Caused by `strings.Repeat()` with extreme iteration count
- Prevents full benchmark suite completion

**Root Cause**:
```go
// Line 21 in keepalive_bench_test.go
data := strings.Repeat("x", b.N)  // b.N can be 1,000,000,000
```

**Fix**:
```go
// Limit data size to reasonable amount
maxSize := 1024 * 1024  // 1 MB
dataSize := b.N
if dataSize > maxSize {
    dataSize = maxSize
}
data := strings.Repeat("x", dataSize)
```

**Validation**:
```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http11
go test -bench=BenchmarkKeepAliveReuse -benchtime=3s
```

**Estimated Time**: 1 hour
**Assignee**: TBD
**Due Date**: Nov 14, 2025

---

### 3. Profile and Identify Server's 4 Allocations
**Status**: CRITICAL - Blocks optimization
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/server/`
**Issue**: Server has 4 allocations per request, fasthttp has 0

**Profiling Commands**:
```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/server

# Memory profiling
go test -bench=BenchmarkServer_SimpleGET -memprofile=mem.prof -benchtime=3s

# Analyze allocations
go tool pprof -alloc_objects -top mem.prof
go tool pprof -alloc_space -top mem.prof
go tool pprof -alloc_objects -list=. mem.prof

# Escape analysis
go test -gcflags="-m -m" 2>&1 | tee escape_analysis.txt
grep "escapes to heap" escape_analysis.txt
```

**Analysis Checklist**:
- [ ] Identify source of allocation 1
- [ ] Identify source of allocation 2
- [ ] Identify source of allocation 3
- [ ] Identify source of allocation 4
- [ ] Document root causes
- [ ] Plan fixes for each

**Expected Output**:
Document in `/home/mirai/Documents/Programming/Projects/watt/shockwave/results/server_allocation_analysis.md`

**Estimated Time**: 4-8 hours
**Assignee**: TBD
**Due Date**: Nov 16, 2025

---

## Priority 1: Server Optimization (Weeks 2-3)

### 4. Eliminate Server Allocation #1
**Status**: HIGH - Depends on task #3
**Files**: TBD (from profiling)
**Target**: Remove first identified allocation

**Common Causes**:
- Interface boxing
- String conversions
- Header map allocations
- Response buffer creation

**Strategy**:
- Replace with pooled objects
- Use inline buffers
- Avoid interface conversions
- Pre-allocate fixed-size structures

**Validation**:
```bash
go test -bench=BenchmarkServer_SimpleGET -benchmem
# Check: 3 allocs/op (down from 4)
```

**Estimated Time**: 2-3 days
**Assignee**: TBD
**Due Date**: Nov 20, 2025

---

### 5. Eliminate Server Allocation #2
**Status**: HIGH - Depends on task #4
**Target**: Remove second identified allocation
**Estimated Time**: 2-3 days
**Due Date**: Nov 23, 2025

---

### 6. Eliminate Server Allocation #3
**Status**: HIGH - Depends on task #5
**Target**: Remove third identified allocation
**Estimated Time**: 2-3 days
**Due Date**: Nov 26, 2025

---

### 7. Eliminate Server Allocation #4
**Status**: HIGH - Depends on task #6
**Target**: Remove fourth identified allocation
**Estimated Time**: 2-3 days
**Due Date**: Nov 29, 2025

**Final Validation**:
```bash
go test -bench=BenchmarkServer_SimpleGET -benchmem
# Target: 0 allocs/op, ~10.5 µs/op
```

---

## Priority 1: Client Optimization (Weeks 3-4)

### 8. Optimize Client Response Parsing
**Status**: HIGH - Biggest client bottleneck
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/client/response.go`
**Issue**: Response parsing has 5 allocations, slowest part of client

**Current Performance**:
```
BenchmarkResponseParsing   539.2 ns/op   114 B/op   5 allocs/op
```

**Target Performance**:
```
BenchmarkResponseParsing   320 ns/op     48 B/op    1 allocs/op
```

**Strategy**:
1. Use pooled buffers instead of `make([]byte, ...)`
2. Parse in-place without intermediate allocations
3. Reuse header storage from pool
4. Avoid string conversions during parse

**Implementation Steps**:
- [ ] Profile current response parsing
- [ ] Identify each of 5 allocations
- [ ] Replace with pooled/inline alternatives
- [ ] Benchmark improvement
- [ ] Validate correctness with tests

**Validation**:
```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/client
go test -bench=BenchmarkResponseParsing -benchmem
go test -bench=BenchmarkClientGET -benchmem
# Check total client allocs: 21 → 17
```

**Estimated Time**: 5-7 days
**Assignee**: TBD
**Due Date**: Dec 6, 2025

---

### 9. Reduce Client Buffer Allocation Sizes
**Status**: MEDIUM - Quick win
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/client/buffer.go`
**Issue**: Fixed 6.5 KB buffer per request, wasteful for small requests

**Current**:
```go
buf := make([]byte, 6580)  // Always 6.5 KB
```

**Target**:
```go
// Adaptive sizing based on expected response
func getResponseBuffer(expectedSize int) []byte {
    if expectedSize < 1024 {
        return smallPool.Get().([]byte)    // 512 B
    } else if expectedSize < 4096 {
        return mediumPool.Get().([]byte)   // 2 KB
    } else {
        return largePool.Get().([]byte)    // 8 KB
    }
}
```

**Expected Gain**: 40-60% memory reduction

**Validation**:
```bash
go test -bench=BenchmarkClientGET -benchmem
# Check B/op: 2,692 → ~1,500
```

**Estimated Time**: 2-3 days
**Assignee**: TBD
**Due Date**: Dec 9, 2025

---

### 10. Implement Inline Header Storage
**Status**: MEDIUM - Reduce allocations
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/client/headers.go`
**Issue**: Header slice allocation for most requests

**Current**:
```go
type Request struct {
    headers []Header  // Always heap allocated
}
```

**Target**:
```go
type Request struct {
    inlineHeaders [8]Header  // Inline for ≤8 headers (80% of requests)
    extraHeaders  []Header    // Only allocate if >8 headers
    headerCount   int
}
```

**Expected Gain**: -2 allocations per request for most traffic

**Validation**:
```bash
go test -bench=BenchmarkClientWithHeaders -benchmem
# Check allocs: 27 → 25
```

**Estimated Time**: 3-4 days
**Assignee**: TBD
**Due Date**: Dec 13, 2025

---

## Priority 2: Protocol Optimizations (Weeks 5-6)

### 11. Improve HTTP/2 HPACK Decode Performance
**Status**: MEDIUM - HTTP/2 bottleneck
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http2/hpack.go`
**Issue**: Decode is 2-5x slower than encode

**Current Performance**:
```
BenchmarkHuffmanEncode/long    293.8 ns/op   204 MB/s
BenchmarkHuffmanDecode/long    704.3 ns/op    65 MB/s  (3x slower)
```

**Target Performance**:
```
BenchmarkHuffmanDecode/long    350 ns/op     130 MB/s  (2x faster)
```

**Strategy**:
1. Profile Huffman decode function
2. Optimize bit manipulation
3. Use lookup tables for common patterns
4. Consider SIMD optimizations

**Validation**:
```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http2
go test -bench=BenchmarkHuffman -benchmem
```

**Estimated Time**: 1 week
**Assignee**: TBD
**Due Date**: Dec 20, 2025

---

### 12. Optimize HTTP/1.1 Chunked Transfer Encoding
**Status**: LOW - Edge case optimization
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http11/chunked.go`
**Issue**: Moderate performance, not zero-alloc

**Current Performance**:
```
BenchmarkChunkedReader_Large   2,508 ns/op   4,308 B/op   14 allocs/op
```

**Target Performance**:
```
BenchmarkChunkedReader_Large   1,500 ns/op   1,024 B/op    2 allocs/op
```

**Estimated Time**: 3-4 days
**Assignee**: TBD
**Due Date**: Dec 27, 2025

---

## Priority 3: Advanced Features (Weeks 7-8+)

### 13. Implement and Benchmark Arena Allocator Mode
**Status**: LOW - Experimental feature
**Files**: New files under `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/memory/`
**Goal**: Provide arena allocation option for zero-GC request handling

**Requirements**:
- Build tag: `arenas`
- Requires: `GOEXPERIMENT=arenas`
- Full request lifecycle in arena
- Benchmark vs standard pooling

**Estimated Time**: 2-3 weeks
**Due Date**: Jan 17, 2026

---

### 14. Complete HTTP/3 Implementation
**Status**: LOW - Future feature
**Files**: `/home/mirai/Documents/Programming/Projects/watt/shockwave/pkg/shockwave/http3/`
**Goal**: Production-ready HTTP/3 with zero-alloc paths

**Estimated Time**: 4-6 weeks
**Due Date**: Feb 28, 2026

---

## Continuous Tasks

### 15. Weekly Performance Regression Testing
**Frequency**: Every Monday
**Command**:
```bash
cd /home/mirai/Documents/Programming/Projects/watt/shockwave
./scripts/run_benchmarks.sh > results/weekly_$(date +%Y%m%d).txt
benchstat results/baseline.txt results/weekly_$(date +%Y%m%d).txt
```

**Success Criteria**: No regressions >2%

---

### 16. Monthly Competitive Benchmarking
**Frequency**: First of each month
**Compare**: Shockwave vs fasthttp vs net/http
**Track**: Progress towards fasthttp parity

---

## Success Metrics

### Week 4 Milestone (Dec 11, 2025)
- [ ] Keep-alive tests pass
- [ ] All benchmarks run without crashes
- [ ] Server: 0 allocations per request
- [ ] Server latency: <11 µs

### Week 8 Milestone (Jan 8, 2026)
- [ ] Client: ≤15 allocations per request
- [ ] Client latency: <32 µs
- [ ] Memory: <1.5 KB per client request
- [ ] 90% feature parity with fasthttp

### Production Readiness (Feb 1, 2026)
- [ ] Performance matches or exceeds fasthttp
- [ ] All tests passing
- [ ] Zero critical bugs
- [ ] Documentation complete
- [ ] Case studies from beta users

---

## Resource Requirements

### Development Environment
- Go 1.22+
- Linux for socket optimizations
- 16+ GB RAM for profiling
- CPU profiler access

### Tools Needed
- `go tool pprof`
- `benchstat`
- `perf` (Linux)
- Memory profilers

### Time Commitment
- Weeks 1-4: Full-time (critical path)
- Weeks 5-8: Part-time (optimizations)
- Ongoing: 10-20% (maintenance)

---

## Risk Assessment

### High Risk
1. **Server allocations difficult to eliminate**: Mitigation - Start early, seek help
2. **Keep-alive fix requires protocol changes**: Mitigation - Design review
3. **Performance targets unrealistic**: Mitigation - Adjust if needed

### Medium Risk
1. **Client optimization takes longer than expected**: Mitigation - Prioritize impact
2. **HPACK optimization requires deep expertise**: Mitigation - Research, consult experts

### Low Risk
1. **Buffer management straightforward**: Known solution
2. **Benchmark fixes easy**: Simple code changes

---

## Communication Plan

### Weekly Updates
- Progress on action items
- Benchmark results
- Blockers and risks

### Milestone Reviews
- Full performance report
- Comparison with targets
- Adjust priorities as needed

---

**Document Created**: 2025-11-13
**Next Review**: 2025-11-20 (after Week 1 tasks)
**Owner**: TBD
**Status**: ACTIVE
