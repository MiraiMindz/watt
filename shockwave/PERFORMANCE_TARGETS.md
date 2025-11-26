# Shockwave Performance Targets - Quick Reference

## 🎯 Target Metrics

| Operation | net/http (baseline) | fasthttp (current best) | gorilla/websocket | Shockwave Target | Improvement |
|-----------|-------------------|------------------------|-------------------|------------------|-------------|
| **Simple GET** | 41,200 ns/op | 4,680 ns/op | N/A | **< 2,500 ns/op** | 16x faster than net/http |
| **Request Parsing** | 2,700 ns/op | 1,900 ns/op | N/A | **< 1,000 ns/op** | 2.7x faster |
| **Response Writing** | 300 ns/op | 325 ns/op | N/A | **< 150 ns/op** | 2x faster |
| **WebSocket Echo** | N/A | N/A | 6,800 ns/op | **< 3,400 ns/op** | 2x faster |
| **WS Throughput** | N/A | N/A | 1,370 ns/op | **< 700 ns/op** | 2x faster |
| **WS Frame Parse** | N/A | N/A | 10 ns/op | **< 10 ns/op** | Match |
| **Keep-Alive (1K reqs)** | 36 ms total | ~5 ms total | N/A | **< 3 ms total** | 12x faster |

## 🚫 Zero Allocation Targets

| Component | Requirement |
|-----------|------------|
| Request parsing | 0 allocs for ≤32 headers |
| Response writing | 0 allocs for pre-compiled responses |
| Keep-alive | 0 allocs per connection reuse |
| Header lookup | 0 allocs (const string comparison) |
| Buffer management | 0 allocs (pooling) |

## 📊 Benchmark Baselines

```bash
# Current Competition (verified with benchstat)
net/http: 41.20µs | 4,404 B/op | 57 allocs/op
fasthttp:  4.68µs |     0 B/op |  0 allocs/op

# Performance Gap: 8.8x
```

## 🔧 Implementation Checklist

### Phase 1: Beat net/http (5x faster)
- [ ] Zero-allocation parser
- [ ] Pre-compiled status lines
- [ ] Inline header storage
- [ ] Basic buffer pooling

### Phase 2: Match fasthttp
- [ ] Advanced pooling
- [ ] Direct syscalls
- [ ] Lock-free structures
- [ ] Connection multiplexing

### Phase 3: Industry Leading
- [ ] Arena allocations
- [ ] SIMD parsing
- [ ] Zero-copy I/O
- [ ] Custom memory allocator

## 📈 Success Metrics

```go
// Benchmark must show:
BenchmarkShockwaveSimpleGET-8  400000  2500 ns/op  0 B/op  0 allocs/op
BenchmarkShockwaveParser-8     1000000 1000 ns/op  0 B/op  0 allocs/op
BenchmarkShockwaveKeepAlive-8  100000  3000 ns/op  0 B/op  0 allocs/op
```

## 🏃 Run Benchmarks

```bash
# Compare against competitors
go test -bench=. -benchmem -count=10 ./benchmarks/competitors

# Verify with benchstat
benchstat old.txt new.txt

# Check for regressions
go test -bench=. -benchmem | grep "allocs/op: 0"
```

---

**Remember**: Every allocation matters. Profile everything. The benchmark is truth.