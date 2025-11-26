# HTTP/3 QPACK Analysis & Optimization Opportunities
**Date**: 2025-11-19
**Platform**: Linux 6.17.8-zen1-1-zen, Intel i7-1165G7 @ 2.80GHz

---

## Executive Summary

HTTP/3 QPACK implementation is **functionally complete** with excellent compression and good baseline performance. However, it follows similar patterns to HTTP/2 HPACK **before optimization**, presenting clear opportunities for performance improvements using the same proven techniques.

**Key Findings:**
- ✅ RFC 9204 compliant QPACK encoder/decoder
- ✅ Static table (99 entries) and dynamic table support
- ✅ Huffman encoding/decoding
- ⚠️ **Allocation-heavy patterns** (similar to pre-optimized HPACK)
- ⚠️ **Buffer reuse opportunities** identified
- 🎯 **Estimated improvement potential: 15-20% faster, 30-40% fewer allocations**

---

## Current Performance Baseline

### QPACK Encoding Benchmarks

```
BenchmarkShockwaveHeaderEncoding-8    ~525 ns/op    192 B/op    2 allocs/op
```

**Workload**: 6 headers (`:method`, `:scheme`, `:authority`, `:path`, `user-agent`, `accept`)

**Analysis:**
- **2 allocations** per encode operation
  - 1 allocation: `bytes.NewBuffer(nil)` in `EncodeHeaders()` (encoder.go:27)
  - 1 allocation: `buf.Bytes()` return copy (encoder.go:45)
- **192 B/op** memory allocation
  - Includes buffer allocation and returned byte slice

---

## Implementation Analysis

### QPACK Encoder (`pkg/shockwave/http3/qpack/encoder.go`)

**Current Implementation Pattern:**

```go
func (enc *Encoder) EncodeHeaders(headers []Header) ([]byte, []byte, error) {
    buf := bytes.NewBuffer(nil)  // ⚠️ ALLOCATION #1: New buffer every call

    // ... encoding logic ...

    return buf.Bytes(), nil, nil  // ⚠️ ALLOCATION #2: Byte slice copy
}
```

**Identified Issues:**
1. **No buffer reuse**: Creates new `bytes.Buffer` on every encode
2. **Copy on return**: `buf.Bytes()` allocates and copies data
3. **No pooling**: Buffers are not pooled for reuse

**Similar to**: HTTP/2 HPACK encoder before optimization

---

### QPACK Decoder (`pkg/shockwave/http3/qpack/decoder.go`)

**Current Implementation Pattern:**

```go
func (d *Decoder) DecodeHeaders(headerBlock []byte) ([]Header, error) {
    r := bytes.NewReader(headerBlock)  // ⚠️ ALLOCATION: bytes.Reader

    var headers []Header               // ⚠️ No pre-allocation
    for r.Len() > 0 {
        header, err := d.decodeHeaderField(r)
        if err != nil {
            return nil, err
        }
        headers = append(headers, header)  // ⚠️ Potential reallocations
    }

    return headers, nil
}
```

**Identified Issues:**
1. **bytes.Reader allocation**: `bytes.NewReader()` allocates on every decode (line 42)
2. **No header slice pre-allocation**: Slice grows dynamically, causing reallocations
3. **No buffer reuse**: String decoding likely allocates intermediate buffers
4. **String conversions**: []byte → string conversions allocate (Go language requirement)

**Similar to**: HTTP/2 HPACK decoder before optimization

---

### Dynamic Table (`pkg/shockwave/http3/qpack/dynamic_table.go`)

**Current Implementation:**

```go
type DynamicTable struct {
    mu sync.RWMutex

    maxSize     uint64
    currentSize uint64

    entries     []DynamicTableEntry
    insertIndex uint64
    baseIndex   uint64

    capacity int
}
```

**Analysis:**
- ✅ Well-designed circular buffer
- ✅ Proper size management (RFC 9204 Section 3.2.1)
- ✅ Thread-safe with RWMutex
- ⚠️ Entry allocation on insert (unavoidable, but could pool entries)

---

## Optimization Opportunities

### Priority 1: Encoder Buffer Reuse ⚡ **HIGH IMPACT**

**Current Problem:**
```go
func (enc *Encoder) EncodeHeaders(headers []Header) ([]byte, []byte, error) {
    buf := bytes.NewBuffer(nil)  // NEW ALLOCATION EVERY CALL
    // ...
    return buf.Bytes(), nil, nil  // COPY ALLOCATION
}
```

**Proposed Solution:**

```go
type Encoder struct {
    dynamicTable *DynamicTable
    maxTableSize uint64

    // Add pooled buffer
    buf          *bytes.Buffer  // Reusable encode buffer
}

func (enc *Encoder) EncodeHeaders(headers []Header) ([]byte, []byte, error) {
    enc.buf.Reset()  // Reuse existing buffer

    // ... encoding logic ...

    // Return copy (required for safety, but reuse buffer for next call)
    result := make([]byte, enc.buf.Len())
    copy(result, enc.buf.Bytes())
    return result, nil, nil
}

// OR: Add EncodeInto() for zero-copy
func (enc *Encoder) EncodeInto(headers []Header, dest []byte) (int, []byte, error) {
    enc.buf.Reset()
    // ... encoding ...
    return copy(dest, enc.buf.Bytes()), nil, nil
}
```

**Expected Impact:**
- **Eliminate 1 allocation** (bytes.Buffer creation)
- **Reuse buffer memory** across multiple encode operations
- **Estimated**: 1-2 allocations → 1 allocation (50% reduction)

---

### Priority 2: Decoder Lightweight Reader ⚡ **HIGH IMPACT**

**Current Problem:**
```go
func (d *Decoder) DecodeHeaders(headerBlock []byte) ([]Header, error) {
    r := bytes.NewReader(headerBlock)  // ALLOCATION
    // ...
}
```

**Proposed Solution** (proven pattern from HTTP/2 HPACK):

```go
type qpackReader struct {
    data []byte
    pos  int
}

func (r *qpackReader) ReadByte() (byte, error) {
    if r.pos >= len(r.data) {
        return 0, io.EOF
    }
    b := r.data[r.pos]
    r.pos++
    return b, nil
}

func (r *qpackReader) UnreadByte() error {
    if r.pos <= 0 {
        return errors.New("cannot unread")
    }
    r.pos--
    return nil
}

func (r *qpackReader) Read(p []byte) (int, error) {
    if r.pos >= len(r.data) {
        return 0, io.EOF
    }
    n := copy(p, r.data[r.pos:])
    r.pos += n
    return n, nil
}

func (r *qpackReader) Len() int {
    return len(r.data) - r.pos
}

func (r *qpackReader) Reset(data []byte) {
    r.data = data
    r.pos = 0
}

// Decoder with reusable reader
type Decoder struct {
    dynamicTable *DynamicTable
    maxTableSize uint64

    reader       qpackReader  // Reusable lightweight reader
    headerBuf    []Header     // Pre-allocated header slice
}

func (d *Decoder) DecodeHeaders(headerBlock []byte) ([]Header, error) {
    d.reader.Reset(headerBlock)  // Zero-allocation reset
    d.headerBuf = d.headerBuf[:0]  // Reuse slice

    for d.reader.Len() > 0 {
        header, err := d.decodeHeaderField(&d.reader)
        if err != nil {
            return nil, err
        }
        d.headerBuf = append(d.headerBuf, header)
    }

    return d.headerBuf, nil
}
```

**Expected Impact:**
- **Eliminate bytes.Reader allocation**
- **Reuse header slice** across multiple decode operations
- **Same pattern proven in HTTP/2 HPACK** (eliminated 2 allocations)

---

### Priority 3: Unsafe String Conversion ⚡ **MODERATE IMPACT**

**Current Problem:**
String decoding likely uses standard `string([]byte)` conversion, which allocates.

**Proposed Solution:**

```go
// Import from pkg/shockwave/http2/unsafe.go or copy

//go:inline
func bytesToString(b []byte) string {
    return unsafe.String(unsafe.SliceData(b), len(b))
}

// Apply in string decoding (similar to HPACK)
func (d *Decoder) decodeString(r qpackReader, length int) (string, error) {
    // ... read bytes into buffer ...

    if huffman {
        return HuffmanDecode(d.stringBuf)
    }

    // Zero-copy conversion (SAFETY: string immediately copied to Header struct)
    return bytesToString(d.stringBuf), nil
}
```

**Safety Requirements** (proven in HTTP/2 HPACK):
1. String is immediately used to create a Header struct
2. Header struct stores the string (which copies it)
3. stringBuf is reused but not modified during string's lifetime
4. String is read-only (Go enforces immutability)

**Expected Impact:**
- **4-5% speedup** (based on HTTP/2 HPACK results)
- **No allocation reduction** (Go still counts string header allocations)

---

### Priority 4: Header Slice Pre-Allocation 📊 **MODERATE IMPACT**

**Current Problem:**
```go
var headers []Header  // Starts at capacity 0
for r.Len() > 0 {
    headers = append(headers, header)  // May reallocate multiple times
}
```

**Proposed Solution:**

```go
type Decoder struct {
    // ...
    headerBuf []Header  // Pre-allocated with reasonable capacity
}

func NewDecoder(maxTableCapacity uint64) *Decoder {
    return &Decoder{
        // ...
        headerBuf: make([]Header, 0, 32),  // Pre-allocate for 32 headers
    }
}

func (d *Decoder) DecodeHeaders(headerBlock []byte) ([]Header, error) {
    d.headerBuf = d.headerBuf[:0]  // Reset length, keep capacity

    for d.reader.Len() > 0 {
        header, err := d.decodeHeaderField(&d.reader)
        if err != nil {
            return nil, err
        }
        d.headerBuf = append(d.headerBuf, header)  // Rarely reallocates
    }

    return d.headerBuf, nil
}
```

**Expected Impact:**
- **Eliminate slice reallocation** for typical requests (≤32 headers)
- **Reuse slice memory** across multiple decode operations
- **Estimated**: 10-20% fewer allocations for large header sets

---

### Priority 5: Dynamic Table Entry Pooling 📦 **LOW IMPACT**

**Current Pattern:**
```go
func (dt *DynamicTable) Insert(name, value string) error {
    entry := DynamicTableEntry{  // Allocation
        Name:  name,
        Value: value,
        Size:  CalculateEntrySize(name, value),
    }
    // ...
}
```

**Proposed Solution:**

```go
var dynamicTableEntryPool = sync.Pool{
    New: func() interface{} { return &DynamicTableEntry{} },
}

func (dt *DynamicTable) Insert(name, value string) error {
    entry := dynamicTableEntryPool.Get().(*DynamicTableEntry)
    entry.Name = name
    entry.Value = value
    entry.Size = CalculateEntrySize(name, value)
    // ...
}

func (dt *DynamicTable) evictToSize(targetSize uint64) error {
    // When evicting, return entries to pool
    for /* eviction logic */ {
        evicted := dt.entries[0]
        dynamicTableEntryPool.Put(&evicted)
        // ...
    }
}
```

**Expected Impact:**
- **Reduce dynamic table insertion overhead**
- **Minimal benefit** for typical workloads (dynamic table rarely used)
- **Consider only if profiling shows dynamic table allocation hotspot**

---

## Comparison: QPACK vs HPACK Optimization Potential

| Metric | HPACK (Pre-Opt) | HPACK (Post-Opt) | Improvement | QPACK (Current) | QPACK (Estimated) |
|--------|-----------------|------------------|-------------|-----------------|-------------------|
| **Decode Speed** | 3600 ns/op | 3045 ns/op | **16% faster** | ~600 ns/op¹ | ~500 ns/op |
| **Decode Allocs** | 19 allocs | 17 allocs | **11% less** | ~5 allocs¹ | ~3 allocs |
| **Encode Speed** | ~300 ns/op² | ~250 ns/op² | **17% faster** | 525 ns/op | ~420 ns/op |
| **Encode Allocs** | 3 allocs² | 1 alloc² | **67% less** | 2 allocs | 1 alloc |

¹ Estimated from encoding benchmark
² Estimated from HPACK encoding patterns

**Conclusion**: QPACK should see **similar or better improvements** than HPACK.

---

## Implementation Strategy

### Phase 1: Encoder Buffer Reuse (Quick Win) 🚀
**Effort**: 2-3 hours
**Expected Impact**: 30-50% fewer allocations in encoding

1. Add `buf *bytes.Buffer` field to Encoder
2. Initialize in `NewEncoder()`
3. Use `buf.Reset()` instead of `bytes.NewBuffer(nil)`
4. Add `EncodeInto()` method for zero-copy option

**Files to Modify:**
- `pkg/shockwave/http3/qpack/encoder.go`

---

### Phase 2: Decoder Lightweight Reader + Pre-Allocation 🚀
**Effort**: 3-4 hours
**Expected Impact**: 20-30% fewer allocations, 10-15% faster decoding

1. Create `qpackReader` struct (copy from HTTP/2 byteReader)
2. Add `reader qpackReader` and `headerBuf []Header` to Decoder
3. Replace `bytes.NewReader()` with `reader.Reset()`
4. Pre-allocate `headerBuf` with capacity 32

**Files to Modify:**
- `pkg/shockwave/http3/qpack/decoder.go`

---

### Phase 3: Unsafe String Conversion ⚡
**Effort**: 1-2 hours
**Expected Impact**: 4-5% speedup

1. Copy `bytesToString()` from `pkg/shockwave/http2/unsafe.go`
   - OR: Create shared `pkg/shockwave/internal/unsafe.go`
2. Apply in string decoding paths
3. Document safety guarantees

**Files to Modify:**
- `pkg/shockwave/http3/qpack/decoder.go`
- Optionally create `pkg/shockwave/http3/qpack/unsafe.go`

---

### Phase 4: Benchmarking & Validation ✅
**Effort**: 2-3 hours

1. Run comprehensive benchmarks before and after
2. Use `benchstat` to validate improvements
3. Ensure no regressions in correctness tests
4. Update documentation

**Benchmarks to Run:**
- `BenchmarkShockwaveHeaderEncoding`
- `BenchmarkShockwaveHeaderDecodingSmall`
- `BenchmarkShockwaveHeaderDecodingLarge`
- `BenchmarkShockwaveStaticTableLookup`
- `BenchmarkShockwaveDynamicTableInsertion`
- `BenchmarkShockwaveHeaderReuse`

---

## Estimated Total Impact

### Performance Gains (Projected)

**Encoding:**
- Speed: **~525 ns/op → ~420 ns/op** (20% faster)
- Memory: **192 B/op → ~120 B/op** (37% less)
- Allocations: **2 allocs/op → 1 alloc/op** (50% fewer)

**Decoding (extrapolated from encoding):**
- Speed: **~600 ns/op → ~500 ns/op** (17% faster)
- Allocations: **~5 allocs/op → ~3 allocs/op** (40% fewer)

**Overall:**
- 🏆 **15-20% faster** QPACK operations
- 🏆 **30-40% fewer allocations**
- 🏆 **Reduced GC pressure**
- 🏆 **Better scalability under load**

---

## Production Readiness Assessment

### ✅ Current State: Production-Ready (Functional)

**Strengths:**
- [x] RFC 9204 compliant
- [x] Static table (99 entries)
- [x] Dynamic table with proper eviction
- [x] Huffman encoding/decoding
- [x] Thread-safe (RWMutex protection)
- [x] Comprehensive test coverage

**Weaknesses:**
- [ ] Allocation-heavy patterns (similar to pre-optimized HPACK)
- [ ] No buffer reuse
- [ ] No encoder/decoder pooling

---

### 🚀 After Optimization: Production-Ready (Performant)

**Additional Benefits:**
- [x] Minimal allocations
- [x] Buffer reuse for sustained performance
- [x] Competitive with best-in-class implementations
- [x] Proven optimization patterns from HTTP/2 HPACK

---

## Comparison: HTTP/2 vs HTTP/3 Optimization

### Similarities
1. **Same allocation patterns**: Both use `bytes.Buffer` and `bytes.Reader`
2. **Same optimization techniques**: Buffer reuse, lightweight readers, unsafe strings
3. **Same RFC structure**: Both use static/dynamic tables, integer encoding, string encoding

### Differences
1. **QPACK is simpler**: No bi-directional state synchronization in basic usage
2. **Smaller baseline**: QPACK already performs better (525ns vs 3600ns for HPACK)
3. **Fewer allocations baseline**: QPACK starts at 2 allocs (HPACK was 19 allocs)

### Conclusion
QPACK optimization will be **faster to implement** and **easier to validate** than HPACK, following the same proven patterns.

---

## Next Steps

### Immediate Actions

1. ✅ **Baseline benchmarks captured** (~525 ns/op, 192 B/op, 2 allocs/op)
2. ⏭ **Implement Phase 1**: Encoder buffer reuse
3. ⏭ **Implement Phase 2**: Decoder lightweight reader
4. ⏭ **Implement Phase 3**: Unsafe string conversion
5. ⏭ **Validate with benchmarks**: Run full benchmark suite
6. ⏭ **Document results**: Create HTTP3_OPTIMIZATION_RESULTS.md

### Long-Term Opportunities

1. **Competitive benchmarking**: vs nghttp3, quic-go QPACK
2. **Advanced optimizations**:
   - Zero-copy UDP integration
   - QUIC stream multiplexing tuning
   - Connection migration optimization
3. **Production deployment**: Real-world workload testing

---

## Lessons from HTTP/2 HPACK Optimization

### What Worked Well ✅
1. **Buffer reuse**: Massive impact with minimal complexity
2. **Lightweight readers**: Simple pattern, reliable gains
3. **Unsafe zero-copy**: Small but measurable improvement
4. **Incremental approach**: Optimize, measure, document, repeat

### What to Avoid ⚠️
1. **Premature optimization**: Measure first, optimize second
2. **Over-optimization**: Don't chase the last 1% if complexity doubles
3. **Breaking API**: Keep compatibility, add new methods if needed

### Application to QPACK
- ✅ Apply all proven techniques from HPACK
- ✅ Start with high-impact, low-complexity optimizations
- ✅ Validate each optimization with benchmarks
- ✅ Document everything for future maintenance

---

## Conclusion

**HTTP/3 QPACK is production-ready but not yet performance-optimized.**

The implementation follows the same patterns as HTTP/2 HPACK **before optimization**, which means we have a clear roadmap to achieve significant performance improvements using proven techniques.

**Estimated Total Effort**: 8-12 hours
**Estimated Total Impact**: 15-20% faster, 30-40% fewer allocations
**Risk**: Low (proven patterns from HTTP/2 HPACK)

**Recommendation**: Proceed with QPACK optimization using the phased approach outlined above. The investment is small, the techniques are proven, and the expected gains are substantial.

---

**QPACK optimization is the natural next step after HTTP/2 HPACK optimization.** 🚀

---

*Analysis Completed: 2025-11-19*
*Baseline Platform: Linux 6.17.8-zen1-1-zen, Intel i7-1165G7 @ 2.80GHz*
*Optimization Pattern: Proven techniques from HTTP/2 HPACK (16% faster, 11% fewer allocs)*
