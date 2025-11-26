# WebSocket Benchmark Analysis

## Gorilla WebSocket Performance Results

### 1. Echo Server (Round-trip latency)
- **Performance**: 6,600 - 7,100 ns/op
- **Throughput**: ~5 MB/s
- **Allocations**: 1,088 B/op, 5 allocs/op
- **Analysis**: Each message round-trip requires 5 allocations

### 2. Broadcast (1-to-10 clients)
- **Performance**: 50,000 - 53,000 ns/op
- **Throughput**: ~3.5 MB/s
- **Allocations**: 5,768 B/op, 23 allocs/op
- **Analysis**: Broadcasting scales poorly with allocations

### 3. Throughput (One-way streaming)
- **Performance**: 1,350 - 1,380 ns/op
- **Throughput**: 741-760 MB/s
- **Allocations**: 2,870 B/op, 4-5 allocs/op
- **Analysis**: Best case scenario for sustained throughput

### 4. Concurrent Connections (100 parallel)
- **Performance**: 2,300 - 2,400 ns/op per message
- **Throughput**: 1,900-1,970 MB/s aggregate
- **Allocations**: 1,095 B/op, 5 allocs/op
- **Analysis**: Good concurrency scaling

### 5. Large Message (1MB)
- **Performance**: 5.8ms - 42ms (high variance)
- **Throughput**: 96-357 MB/s
- **Allocations**: 10.5 MB/op, 59 allocs/op
- **Analysis**: Copies entire message, causes huge allocations

### 6. Ping/Pong (Control frames)
- **Performance**: 13,500 - 14,500 ns/op
- **Allocations**: 1,040 B/op, 13 allocs/op
- **Analysis**: Control frames are expensive

### 7. Frame Parsing (Low-level)
- **Performance**: 9.8 - 10.1 ns/op 🎯
- **Throughput**: 1,100+ MB/s
- **Allocations**: 0 B/op, 0 allocs/op ✅
- **Analysis**: Zero-allocation parsing achieved!

## Key Findings

### Strengths of Gorilla WebSocket
1. **Frame parsing is already optimized** (10 ns/op, 0 allocs)
2. **Good concurrent scaling** (2.3µs per message at 100 connections)
3. **Decent throughput** for streaming (750 MB/s)

### Weaknesses to Target
1. **Echo requires 5 allocations** per round-trip
2. **Large messages copy entire payload** (10.5 MB allocation for 1 MB message)
3. **Control frames are expensive** (13 allocations for ping/pong)
4. **Broadcasting doesn't scale** (23 allocations for 10 clients)

## Shockwave WebSocket Targets

### Phase 1: Beat Gorilla by 2x
| Operation | Gorilla | Shockwave Target | Improvement |
|-----------|---------|------------------|-------------|
| Echo | 6,800 ns | **< 3,400 ns** | 2x faster |
| Broadcast (10) | 51,000 ns | **< 25,000 ns** | 2x faster |
| Throughput | 1,370 ns | **< 700 ns** | 2x faster |
| Large Message | 10 MB alloc | **< 1 MB alloc** | 10x less memory |
| Ping/Pong | 13 allocs | **< 2 allocs** | 6x fewer |

### Phase 2: Zero-Allocation Goals
- [ ] Zero-copy message forwarding
- [ ] Ring buffer for frame assembly
- [ ] Pooled frame objects
- [ ] Direct syscall writes

### Phase 3: Industry Leading
- [ ] SIMD masking/unmasking
- [ ] Lock-free message routing
- [ ] Kernel bypass with io_uring
- [ ] < 100 ns frame parsing

## Implementation Strategy

### Critical Optimizations
1. **Zero-copy forwarding**: Use sendfile/splice for large messages
2. **Ring buffers**: Avoid allocations for frame assembly
3. **Direct writes**: Bypass Go's buffering for control frames
4. **SIMD unmasking**: Vectorize XOR operations

### Memory Management
```go
// Current gorilla approach (allocates)
msg := make([]byte, len(data))
copy(msg, data)

// Shockwave approach (zero-copy)
msg := bufferPool.Get()
defer bufferPool.Put(msg)
// Use msg directly, no copy
```

## Benchmark Comparison

| Feature | gorilla/websocket | Shockwave Target | fasthttp-ws |
|---------|------------------|------------------|-------------|
| Has WebSocket | ✅ | ✅ | ❌ (extension) |
| Zero-alloc parsing | ✅ (10 ns) | ✅ (< 10 ns) | N/A |
| Zero-copy forward | ❌ | ✅ | N/A |
| SIMD optimization | ❌ | ✅ | N/A |
| Integrated HTTP | ❌ (separate) | ✅ (unified) | ✅ |

## Success Metrics

```bash
# Current gorilla/websocket
BenchmarkGorillaWebSocketEcho-8        150000   6800 ns/op   1088 B/op   5 allocs/op
BenchmarkGorillaWebSocketThroughput-8  850000   1370 ns/op   2870 B/op   4 allocs/op

# Target for Shockwave
BenchmarkShockwaveWebSocketEcho-8      300000   3400 ns/op      0 B/op   0 allocs/op
BenchmarkShockwaveWebSocketThroughput-8 1700000  700 ns/op      0 B/op   0 allocs/op
```

## Conclusion

WebSocket benchmarking reveals:
1. **Frame parsing is already fast** (10 ns, zero alloc)
2. **Major gains possible** in message handling (5+ allocations)
3. **Zero-copy critical** for large messages
4. **Integration opportunity** with HTTP layer for efficiency

Shockwave can achieve 2-10x improvement by:
- Eliminating allocations in message path
- Zero-copy forwarding
- Unified HTTP/WebSocket handling
- SIMD optimizations for masking