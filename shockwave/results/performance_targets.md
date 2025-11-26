# Shockwave Performance Target Report

## Executive Summary

Based on comprehensive benchmarking of competitor HTTP libraries, we have established clear performance targets for Shockwave. Our analysis shows that fasthttp achieves **~9x better performance** than net/http in simple GET requests, with **zero allocations** compared to net/http's 109 allocations per request.

## Benchmark Results Summary

### Simple GET Request Performance

| Library | ns/op | Allocations | Relative Speed |
|---------|-------|------------|----------------|
| net/http | ~41,000 | 57-109 | 1.0x (baseline) |
| fasthttp | ~4,700 | 0 | 8.7x faster |

**Shockwave Target**: < 2,500 ns/op with 0 allocations (2x faster than fasthttp)

### Request Parsing Performance

| Library | ns/op | B/op | allocs/op |
|---------|-------|------|-----------|
| net/http | ~2,700 | 5,242 | 13 |
| fasthttp | ~1,900 | 4,241 | 3 |

**Shockwave Target**: < 1,000 ns/op with 0 allocations

### WebSocket Performance

| Library | ns/op | B/op | allocs/op |
|---------|-------|------|-----------|
| gorilla/websocket | ~9,300 | 1,088 | 5 |

**Shockwave Target**: < 5,000 ns/op with 0-2 allocations

### Response Writing Performance

| Library | ns/op | B/op | allocs/op |
|---------|-------|------|-----------|
| net/http | ~300 | varies | varies |
| fasthttp | ~325 | 0 | 0 |

**Shockwave Target**: < 150 ns/op with pre-compiled responses

### Header Processing (30 headers)

| Library | ns/op | B/op | allocs/op |
|---------|-------|------|-----------|
| net/http | ~7,200 | 8,979 | 72 |
| fasthttp | ~2,900 | 4,240 | 3 |

**Shockwave Target**: < 1,500 ns/op with inline header storage (0 allocs for ≤32 headers)

## Key Performance Gaps Identified

### 1. Memory Allocations
- **net/http**: 57-109 allocations per simple request
- **fasthttp**: 0 allocations for simple requests
- **Gap**: net/http allocates on every request, fasthttp reuses buffers
- **Shockwave Strategy**: Arena allocations + object pooling

### 2. Request Parsing
- **net/http**: Uses reflection and interfaces extensively
- **fasthttp**: Direct byte manipulation
- **Shockwave Strategy**: Zero-copy parsing with state machines

### 3. Connection Management
- **net/http**: High overhead for keep-alive (~36ms for 1000 requests)
- **fasthttp**: Efficient connection pooling
- **Shockwave Strategy**: Lock-free connection pools

## Shockwave Performance Contract

### Phase 1: HTTP/1.1 (Beat net/http by 5x)
- [ ] Simple GET: < 8,000 ns/op (5x faster than net/http)
- [ ] Zero allocations for requests with ≤32 headers
- [ ] Response writing: < 200 ns/op for pre-compiled responses
- [ ] Keep-alive: 0 allocations per reused connection

### Phase 2: Match fasthttp
- [ ] Simple GET: < 4,700 ns/op
- [ ] Request parsing: < 1,900 ns/op
- [ ] Concurrent handling: Linear scaling to 100 connections

### Phase 3: Industry Leading
- [ ] Simple GET: < 2,500 ns/op (2x faster than fasthttp)
- [ ] HTTP/2: > 300K req/sec multiplexed
- [ ] WebSocket: < 100 ns/op frame parsing
- [ ] Zero-copy sendfile for responses > 100KB

## Implementation Priorities

### Critical Optimizations
1. **Zero-allocation parsing**: Use stack-allocated buffers
2. **Pre-compiled constants**: Status lines, common headers
3. **Direct syscalls**: Bypass Go runtime where beneficial
4. **Arena allocation**: Eliminate GC pressure in hot paths

### Socket-Level Optimizations
1. **TCP_QUICKACK**: Reduce latency for small requests
2. **TCP_NODELAY**: Disable Nagle's algorithm
3. **TCP_FASTOPEN**: Reduce connection establishment time
4. **SO_REUSEPORT**: Better load distribution

## Success Metrics

### Minimum Viable Performance (MVP)
- Beat net/http by 2x in all benchmarks
- Zero allocations for common operations
- < 100μs P99 latency for simple requests

### Target Performance
- Match or beat fasthttp in all benchmarks
- Support 1M+ concurrent connections
- < 50μs P99 latency

### Stretch Goals
- 10M+ requests/second on modern hardware
- < 10μs P50 latency
- Industry-leading performance in all categories

## Validation Strategy

1. **Continuous Benchmarking**: Run benchmarks on every commit
2. **Regression Detection**: Fail CI on >5% performance regression
3. **Profile-Guided Optimization**: Use pprof data to guide development
4. **Real-World Testing**: Benchmark against production workloads

## Conclusion

The benchmarks clearly show:
- **fasthttp is 8-10x faster than net/http**
- **Zero allocations are achievable and critical**
- **Pre-compilation and pooling are key strategies**

Shockwave will achieve industry-leading performance by:
1. Starting with fasthttp's proven techniques
2. Adding advanced memory management (arenas, green tea GC)
3. Leveraging modern CPU features and syscalls
4. Maintaining net/http compatibility via adapter layer

**Next Step**: Implement Phase 1 - Zero-allocation HTTP/1.1 parser