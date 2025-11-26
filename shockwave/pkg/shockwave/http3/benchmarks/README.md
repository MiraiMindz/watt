# HTTP/3 Benchmarks

This directory contains benchmarks comparing Shockwave's HTTP/3 implementation against other libraries.

## Internal Benchmarks

Benchmarks measuring Shockwave's HTTP/3 performance in isolation:

```bash
go test -bench=. -benchmem github.com/yourusername/shockwave/pkg/shockwave/http3
```

### Results (11th Gen Intel Core i7-1165G7 @ 2.80GHz)

| Benchmark | ops/sec | ns/op | B/op | allocs/op |
|-----------|---------|-------|------|-----------|
| **Header Encoding** | 2.74M | 432.0 | 192 | 2 |
| **Header Decoding** | 2.82M | 412.4 | 656 | 13 |
| **Frame Serialization** | 6.75M | 178.6 | 1160 | 2 |
| **Frame Parsing** | 5.21M | 232.4 | 1083 | 6 |
| **Full Request/Response** | 600K | 1920 | 3592 | 38 |

## Comparative Benchmarks

### vs nghttp3

To run comparative benchmarks against nghttp3:

```bash
./benchmarks/run_nghttp3_comparison.sh
```

This requires:
- nghttp3 installed (`nghttp3-git` on Arch Linux)
- C compiler for cgo
- Benchmark server implementation

## Benchmark Methodology

### Header Encoding/Decoding
- Encodes 6 common HTTP headers using QPACK
- Measures compression ratio and throughput
- Tests static table utilization

### Frame Serialization/Parsing
- Serializes and parses 1KB DATA frames
- Measures zero-copy performance
- Tests frame validation

### Full Request/Response
- Complete HTTP/3 transaction cycle
- Includes header encoding, frame serialization, parsing, and decoding
- Simulates realistic workload

## Performance Goals

| Operation | Target | Status |
|-----------|--------|--------|
| Header encoding | < 500 ns/op | ✅ 432 ns/op |
| Header decoding | < 500 ns/op | ✅ 412 ns/op |
| Frame operations | < 300 ns/op | ✅ 178-232 ns/op |
| Request/Response | < 2μs/op | ✅ 1.92 μs/op |

## Optimization Notes

### Zero-Allocation Targets
- Header encoding: Currently 2 allocs, target 0
- Header decoding: Currently 13 allocs, needs reduction
- Frame serialization: Currently 2 allocs, target 1 (output buffer only)
- Frame parsing: Currently 6 allocs, target 2 (frame struct + data)

### Future Improvements
1. **Header Encoding**: Reduce allocations through buffer pooling
2. **Header Decoding**: Implement zero-copy string extraction
3. **Frame Operations**: Use sync.Pool for frame structs
4. **Huffman Coding**: Add Huffman compression for better compression ratios

## Profiling

Generate CPU profile:
```bash
go test -bench=BenchmarkHTTP3FullRequestResponse -cpuprofile=cpu.prof
go tool pprof -http=:8080 cpu.prof
```

Generate memory profile:
```bash
go test -bench=BenchmarkHTTP3FullRequestResponse -memprofile=mem.prof
go tool pprof -http=:8080 mem.prof
```
