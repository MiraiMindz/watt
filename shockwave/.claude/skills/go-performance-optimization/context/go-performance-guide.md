# Go Performance Optimization Guide for Shockwave

## Memory Allocation Hierarchy

### 1. Stack Allocation (Fastest)
- Function-local variables that don't escape
- Fixed-size arrays
- Primitive types
- Small structs passed by value

**Cost:** ~1-2 ns (register/stack access)

### 2. Arena Allocation (Shockwave-specific)
- Bulk allocation for request lifetime
- Zero GC pressure
- Freed all at once when request completes
- Enabled with `arenas` build tag

**Cost:** ~5-10 ns + zero GC overhead

### 3. Pooled Allocation
- `sync.Pool` for reusable objects
- Amortized cost near zero
- Subject to GC between uses

**Cost:** ~20-50 ns (pool get/put)

### 4. Heap Allocation (Slowest)
- Anything that escapes
- Dynamic sizes
- Interface conversions
- Return pointers to local variables

**Cost:** ~50-500 ns + GC overhead

## Hot Path Optimization Strategies

### Zero-Allocation Request Parsing
```go
type Request struct {
    // Inline array, no allocation for ≤32 headers
    headers [32]Header
    headerCount int

    // Inline buffer for small request lines
    methodBuf [16]byte
    pathBuf [1024]byte

    // Pointers to inline buffers (zero alloc)
    Method string
    Path string
}

func (r *Request) parseMethod(b []byte) {
    n := copy(r.methodBuf[:], b)
    r.Method = string(r.methodBuf[:n]) // Points to inline array
}
```

### Pre-compiled Constants Pattern
```go
// constants.go
var (
    // Both representations to avoid conversions
    httpVersion11Bytes = []byte("HTTP/1.1")
    httpVersion11 = "HTTP/1.1"

    status200Bytes = []byte("HTTP/1.1 200 OK\r\n")
    status404Bytes = []byte("HTTP/1.1 404 Not Found\r\n")

    headerContentType = []byte("Content-Type")
    headerContentLength = []byte("Content-Length")
)

// Usage: zero-allocation write
func (w *ResponseWriter) WriteStatus200() {
    w.buf.Write(status200Bytes) // No allocation
}
```

### Method Dispatch via Constants
```go
const (
    MethodGET = iota
    MethodPOST
    MethodPUT
    MethodDELETE
    MethodPATCH
    MethodHEAD
    MethodOPTIONS
)

// O(1) lookup table
var methodNames = [...]string{
    MethodGET: "GET",
    MethodPOST: "POST",
    MethodPUT: "PUT",
    // ...
}

// O(1) dispatch
func (s *Server) dispatch(methodID int, r *Request) {
    switch methodID {
    case MethodGET:
        s.handleGET(r)
    case MethodPOST:
        s.handlePOST(r)
    // ...
    }
}
```

### Buffer Pool Management
```go
// Size-specific pools for predictable allocation
var (
    pool2KB = sync.Pool{
        New: func() interface{} {
            buf := make([]byte, 2048)
            return &buf
        },
    }

    pool4KB = sync.Pool{
        New: func() interface{} {
            buf := make([]byte, 4096)
            return &buf
        },
    }
)

func getBuffer(size int) *[]byte {
    if size <= 2048 {
        return pool2KB.Get().(*[]byte)
    }
    if size <= 4096 {
        return pool4KB.Get().(*[]byte)
    }
    buf := make([]byte, size)
    return &buf
}

func putBuffer(buf *[]byte) {
    *buf = (*buf)[:0] // Reset length, keep capacity
    switch cap(*buf) {
    case 2048:
        pool2KB.Put(buf)
    case 4096:
        pool4KB.Put(buf)
    }
}
```

## Escape Analysis Patterns

### Common Escape Causes

#### 1. Interface Conversion
```go
// ❌ Escapes
func write(w io.Writer, data []byte) {
    w.Write(data) // w escapes
}

// ✅ Stays on stack
func write(w *bytes.Buffer, data []byte) {
    w.Write(data) // Concrete type
}
```

#### 2. Closure Capture
```go
// ❌ Escapes
func handler(r *Request) {
    go func() {
        process(r) // r escapes to heap
    }()
}

// ✅ Stays on stack
func handler(r *Request) {
    data := r.extract() // Copy what you need
    go func() {
        process(data) // Only data escapes
    }()
}
```

#### 3. Slice to Interface
```go
// ❌ Escapes
func send(items []Item) {
    log.Println(items) // items escapes
}

// ✅ Avoid if hot path
func send(items []Item) {
    // Don't log in hot path
}
```

## Socket-Level Optimizations (Linux)

### TCP_QUICKACK
- Sends ACKs immediately without 40ms delay
- Reduces latency by ~40ms per request
- Trade-off: slightly more CPU for ACK processing

```go
// In socket/tuning_linux.go
func SetTCPQuickAck(fd int) error {
    return syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP,
        unix.TCP_QUICKACK, 1)
}
```

### TCP_DEFER_ACCEPT
- Kernel doesn't wake server until data arrives
- Reduces overhead from SYN flood
- Saves context switches

```go
func SetTCPDeferAccept(fd int) error {
    // Wait up to 5 seconds for data
    return syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP,
        unix.TCP_DEFER_ACCEPT, 5)
}
```

### TCP_FASTOPEN
- Embeds data in SYN packet
- Reduces connection establishment from 1.5 RTT to 0.5 RTT
- Significant for short-lived connections

```go
func SetTCPFastOpen(fd int) error {
    return syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP,
        unix.TCP_FASTOPEN, 5) // Queue size
}
```

### Zero-Copy Sendfile
- Kernel transfers file → socket without user-space copy
- Eliminates ~70% CPU for file serving

```go
func sendFile(sock, file int, size int64) error {
    _, err := syscall.Sendfile(sock, file, nil, int(size))
    return err
}
```

## HTTP/2 Specific Optimizations

### HPACK Compression
- Dynamic table management
- Header compression ratio: 30-80%
- Trade-off: CPU for bandwidth

```go
type HPACKEncoder struct {
    dynamicTable []HeaderField
    size int
    maxSize int
}

// Compress headers using static and dynamic tables
func (e *HPACKEncoder) Encode(headers []Header) []byte {
    // Use indexed representation when possible
    // Add to dynamic table for repeated headers
}
```

### Stream Multiplexing
- Multiple requests on single connection
- Concurrent stream handling
- Flow control per stream

```go
type Connection struct {
    streams sync.Map // map[StreamID]*Stream
    flowControl *FlowController
}
```

## Profiling Cookbook

### CPU Profile
```bash
# Generate profile
go test -bench=BenchmarkHTTP11 -cpuprofile=cpu.prof

# Interactive analysis
go tool pprof cpu.prof
> top 20
> list FunctionName
> web

# Or web UI
go tool pprof -http=:8080 cpu.prof
```

### Memory Profile
```bash
# Allocation profile
go test -bench=BenchmarkHTTP11 -memprofile=mem.prof

# Analyze allocations
go tool pprof -alloc_space mem.prof
> top 20

# Analyze live heap
go tool pprof -inuse_space mem.prof
```

### Escape Analysis
```bash
# See all escapes
go build -gcflags="-m -m" ./... 2>&1 | tee escape.txt

# Filter specific package
go build -gcflags="-m -m" ./pkg/shockwave/http11 2>&1 | grep "escapes"

# Look for unexpected escapes
grep "escapes to heap" escape.txt | grep -v "interface"
```

### Benchmark Comparison
```bash
# Baseline
git checkout main
go test -bench=. -benchmem -count=10 > main.txt

# Your changes
git checkout feature/optimization
go test -bench=. -benchmem -count=10 > feature.txt

# Compare
benchstat main.txt feature.txt
```

## Performance Targets

### Shockwave Goals
- **Request parsing**: 0 allocs/op
- **Keep-alive reuse**: 0 allocs/op
- **Status line write**: 0 allocs/op
- **Header lookup**: <10ns/op
- **HTTP/1.1 throughput**: >500k req/s (single core)
- **HTTP/2 throughput**: >300k req/s (single core)
- **Latency p99**: <100μs (excluding handler)

### Acceptable Trade-offs
- CPU for memory (within reason)
- Code complexity for 10%+ performance gain
- Platform-specific code for major optimizations
- Experimental features (arenas) for 40%+ improvement

### Unacceptable Trade-offs
- Correctness for performance
- Security for performance
- Protocol compliance for performance
- Maintainability for <5% gain
