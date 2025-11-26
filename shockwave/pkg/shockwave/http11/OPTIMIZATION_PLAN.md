# HTTP/1.1 Parser Optimization Plan
**Target**: Reduce allocations from 15KB to <8KB per request
**Goal**: Achieve 0 allocs/op for typical requests (currently 3 allocs/op)

---

## Current Allocation Analysis

### Allocation Sources (from profiling):

1. **Parser.readUntilHeadersEnd** (41% of 16.97GB = 6.97GB)
   - Line 79: `tmpBuf := make([]byte, 4096)` - **4KB per request**
   - This buffer is allocated on every Parse() call
   - Root cause: No buffer pooling

2. **Parser.Parse** (55% of 16.97GB = 9.34GB)
   - Line 50: `req := &Request{...}` - **~11KB per request**
   - Request struct contains:
     - Header struct: ~10.3KB (names[32][64] + values[32][256])
     - Other fields: ~700 bytes
   - Root cause: No Request pooling

3. **Parser.buf** (amortized across requests)
   - Capacity: 8KB (MaxRequestLineSize + MaxHeadersSize)
   - Grows via append(), may reallocate
   - Root cause: No pre-allocation strategy

### Total Current Allocation: ~15KB per request with 3 allocs/op

**Breakdown**:
- tmpBuf: 4KB (1 alloc)
- Request + Header: 11KB (1 alloc)
- Misc (strings, slices): ~100 bytes (1 alloc)

---

## Optimization Strategy

### Phase 1: Pool Temporary Buffers (Save 4KB, -1 alloc)

**Problem**: parser.go:79 allocates 4KB tmpBuf on every call

**Solution**: Use sync.Pool for temporary read buffers

```go
var tmpBufPool = sync.Pool{
    New: func() interface{} {
        buf := make([]byte, 4096)
        return &buf
    },
}

func (p *Parser) readUntilHeadersEnd(r io.Reader) error {
    tmpBuf := tmpBufPool.Get().(*[]byte)
    defer tmpBufPool.Put(tmpBuf)

    // Use *tmpBuf instead of tmpBuf
    // ...
}
```

**Expected Impact**:
- Reduce allocations by 1 (4KB)
- Allocation per request: 15KB → 11KB
- Allocs/op: 3 → 2

---

### Phase 2: Pool Request Objects (Save 11KB, -1 alloc)

**Problem**: parser.go:50 allocates new Request on every call

**Solution**: Extend existing Request pool to include Header reset

**Current pool (pool.go)**:
```go
func GetRequest() *Request {
    return requestPool.Get().(*Request)
}

func PutRequest(req *Request) {
    req.Header.Reset()
    requestPool.Put(req)
}
```

**Modification needed**:
1. Parser should use GetRequest() instead of &Request{}
2. Ensure Header.Reset() clears all state properly
3. Connection must call PutRequest() after handler completes

**Expected Impact**:
- Reduce allocations by 1 (11KB)
- Allocation per request: 11KB → ~100 bytes
- Allocs/op: 2 → 1

---

### Phase 3: Optimize Header Storage (Save 2-3KB)

**Problem**: Header struct is 10.3KB, but most requests have <10 headers

**Current**:
```go
names  [32][64]byte   // 2KB
values [32][256]byte  // 8KB
```

**Solution A**: Tiered storage (small/medium/large)
- Small (0-8 headers): 512 bytes + 2KB = 2.5KB
- Medium (9-16 headers): 1KB + 4KB = 5KB
- Large (17-32 headers): 2KB + 8KB = 10KB (current)

**Solution B**: Reduce value array size
- Most header values are <128 bytes
- Use smaller inline storage, fallback to heap for large values

**Recommended**: Solution B for simplicity
```go
values [32][128]byte  // 4KB (down from 8KB)
```

**Expected Impact**:
- Reduce Request size by 4KB
- Pooled requests benefit immediately
- Negligible performance impact (overflow is rare)

---

### Phase 4: Optimize Buffer Management

**Problem**: Parser.buf grows via append(), may reallocate

**Solution**: Pre-allocate to maximum expected size

**Current**:
```go
buf: make([]byte, 0, MaxRequestLineSize+MaxHeadersSize)
```

**Optimization**: Reset to capacity instead of length 0
```go
func (p *Parser) Parse(r io.Reader) (*Request, error) {
    p.buf = p.buf[:cap(p.buf)]  // Use full capacity
    n := 0  // Track actual length
    // ...
}
```

**Expected Impact**:
- Prevent reallocation during append
- Minor (buf is pooled), but prevents edge-case allocs

---

### Phase 5: Implement HTTP Pipelining Support

**Problem**: Parser over-reads from io.Reader, consuming multiple requests

**Root Cause**: readUntilHeadersEnd() reads until it finds \r\n\r\n, but doesn't track buffer boundaries

**Solution**: Track exact read position and return unused bytes

**Implementation**:
1. Add `unreadBuf []byte` to Parser struct
2. In readUntilHeadersEnd(), save excess bytes after \r\n\r\n
3. On next Parse() call, check unreadBuf first before reading from io.Reader
4. Use io.MultiReader to combine unreadBuf with io.Reader

```go
type Parser struct {
    buf       []byte
    unreadBuf []byte  // Buffered data from previous read
}

func (p *Parser) Parse(r io.Reader) (*Request, error) {
    // Combine unread buffer with new reader
    if len(p.unreadBuf) > 0 {
        r = io.MultiReader(bytes.NewReader(p.unreadBuf), r)
        p.unreadBuf = nil
    }

    // ... rest of parsing
}

func (p *Parser) readUntilHeadersEnd(r io.Reader) error {
    // After finding \r\n\r\n, save any excess bytes
    actualIdx := searchStart + idx + 4
    if n > actualIdx {
        // Save excess bytes for next request
        p.unreadBuf = append([]byte{}, tmpBuf[actualIdx:n]...)
    }
    p.buf = p.buf[:actualIdx]
    // ...
}
```

**Alternative**: Use bufio.Reader with proper boundary tracking
```go
type Parser struct {
    buf       []byte
    reader    *bufio.Reader  // Persistent buffered reader
}

func (p *Parser) Parse(r io.Reader) (*Request, error) {
    // Reset reader with new source
    if p.reader == nil {
        p.reader = bufio.NewReader(r)
    } else {
        p.reader.Reset(r)
    }

    // Read until \r\n\r\n using ReadSlice or ReadBytes
    // bufio.Reader properly tracks position
}
```

**Expected Impact**:
- Enable HTTP keep-alive pipelining
- Fix 2 skipped tests in connection_test.go
- 2-3x throughput improvement for pipelined workloads

---

### Phase 6: Reduce Escape to Heap

**Problem**: Some allocations escape to heap unnecessarily

**Solution**: Run escape analysis and add compiler directives

```bash
go build -gcflags="-m -m" 2>&1 | grep "escapes to heap" | grep http11
```

**Common fixes**:
- Add `//go:noescape` for pure functions
- Avoid storing pointers to stack variables
- Use values instead of pointers where possible

**Expected Impact**:
- Reduce 1-2 allocations per request
- Compiler-dependent, but measurable

---

## Implementation Order

### Sprint 1: Buffer Pooling (Quick Win)
1. ✅ Implement tmpBuf pooling (Phase 1)
2. ✅ Integrate Request pooling in Parser (Phase 2)
3. ✅ Test and benchmark

**Target**: 2 allocs/op, ~11KB per request

### Sprint 2: Header Optimization
4. ✅ Reduce Header value array size (Phase 3)
5. ✅ Test with large header scenarios
6. ✅ Benchmark impact

**Target**: 2 allocs/op, ~7KB per request

### Sprint 3: Pipelining Support
7. ✅ Implement buffer boundary tracking (Phase 5)
8. ✅ Un-skip pipelining tests
9. ✅ Validate RFC compliance

**Target**: Enable keep-alive, 2x throughput

### Sprint 4: Escape Analysis
10. ✅ Run escape analysis (Phase 6)
11. ✅ Fix heap escapes
12. ✅ Final benchmarks

**Target**: 0-1 allocs/op, <8KB per request

---

## Success Criteria

### Performance Targets:
- ✅ **Allocations**: <2 allocs/op (currently 3)
- ✅ **Memory per request**: <8KB (currently 15KB)
- ✅ **Throughput**: >200K req/sec for simple GET
- ✅ **Pipelining**: Support keep-alive with multiple requests
- ✅ **Zero-alloc goal**: 0 allocs/op for typical requests

### Benchmarks to Run:
```bash
# Before optimizations
go test -bench=BenchmarkSuite_Parse -benchmem -count=5 > before.txt

# After each phase
go test -bench=BenchmarkSuite_Parse -benchmem -count=5 > after.txt

# Compare
benchstat before.txt after.txt
```

### Validation Tests:
- All existing tests pass
- RFC compliance tests pass
- E2E tests pass with pipelining
- Performance tests show improvement

---

## Risk Assessment

### Low Risk:
- Phase 1 (tmpBuf pooling) - Well-understood pattern
- Phase 3 (Header optimization) - Simple size reduction

### Medium Risk:
- Phase 2 (Request pooling) - Must ensure proper reset
- Phase 4 (Buffer management) - Edge cases with large requests

### High Risk:
- Phase 5 (Pipelining) - Complex buffer management, many edge cases
- Phase 6 (Escape analysis) - Compiler-dependent, may break assumptions

### Mitigation:
- Implement phases incrementally
- Full test suite after each phase
- Benchmark after each change
- Rollback plan if performance degrades

---

## Monitoring Strategy

### Metrics to Track:
1. **Allocations**: B/op and allocs/op per benchmark
2. **Throughput**: Req/sec for simple GET
3. **Latency**: ns/op for full cycle
4. **Memory**: Peak heap usage during load test

### Regression Thresholds:
- Max 5% latency increase
- Max 10% throughput decrease
- No new allocs introduced
- No RFC compliance failures

### Continuous Benchmarking:
```bash
# Run benchmark suite after each commit
go test -bench=. -benchmem -count=10 > results.txt

# Check against baseline
benchstat baseline.txt results.txt

# Alert if regression > 5%
```

---

## Expected Final Results

### Current (Baseline):
- **Parse**: 3.7 µs, 15,016 B/op, 3 allocs/op
- **Full cycle**: 6.7 µs, 19,744 B/op, 10 allocs/op
- **Throughput**: ~150K req/sec

### Target (After Optimization):
- **Parse**: 2.5 µs, 6,000 B/op, 1 allocs/op
- **Full cycle**: 4.5 µs, 10,000 B/op, 5 allocs/op
- **Throughput**: >250K req/sec

### Stretch Goal (Zero-Alloc):
- **Parse**: 2.0 µs, 0 B/op, 0 allocs/op
- **Full cycle**: 3.5 µs, 0 B/op, 0 allocs/op
- **Throughput**: >300K req/sec

---

## Next Steps

1. ✅ Review and approve optimization plan
2. ✅ Run baseline benchmarks and save results
3. ✅ Implement Phase 1 (tmpBuf pooling)
4. ✅ Implement Phase 2 (Request pooling integration)
5. ✅ Benchmark and validate
6. Continue to Phase 3...

---

**Plan Version**: 1.0
**Created**: 2025-11-10
**Last Updated**: 2025-11-10
**Status**: Ready for implementation
