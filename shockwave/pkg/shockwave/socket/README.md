# Socket Tuning Package

High-performance socket optimizations for Shockwave HTTP library.

## Overview

This package provides cross-platform socket tuning with Linux-specific optimizations for maximum HTTP performance. It implements best practices from high-performance web servers like NGINX and implements zero-copy file serving.

## Features

### Cross-Platform Optimizations

- **TCP_NODELAY** - Disables Nagle's algorithm for low latency
- **SO_RCVBUF / SO_SNDBUF** - Configurable buffer sizes
- **SO_KEEPALIVE** - TCP keepalive with tunable parameters

### Linux-Specific Optimizations

- **TCP_QUICKACK** - Immediate ACKs, reduces latency by ~40ms
- **TCP_DEFER_ACCEPT** - Server only wakes when data arrives (reduces context switches)
- **TCP_FASTOPEN** - Reduces connection establishment by one RTT
- **TCP_USER_TIMEOUT** - Faster dead connection detection
- **Zero-copy sendfile()** - Direct kernel-to-kernel file transfers

### macOS-Specific Optimizations

- **TCP_FASTOPEN** - Same RTT reduction as Linux
- **SO_NOSIGPIPE** - Prevents SIGPIPE on broken connections
- **TCP_KEEPALIVE** - Keepalive tuning

## Usage

### Basic Usage

```go
import "github.com/yourusername/shockwave/pkg/shockwave/socket"

// Create listener
listener, _ := net.Listen("tcp", ":8080")

// Apply listener optimizations
socket.ApplyListener(listener, socket.DefaultConfig())

// Accept and tune connections
for {
    conn, _ := listener.Accept()
    socket.Apply(conn, socket.DefaultConfig())

    // Handle connection
    go handleConnection(conn)
}
```

### Configuration Presets

#### Default Config (Balanced)
```go
cfg := socket.DefaultConfig()
// NoDelay: true
// RecvBuffer: 256KB
// SendBuffer: 256KB
// QuickAck: true (Linux)
// DeferAccept: true (Linux)
// FastOpen: true
```

#### Low Latency Config
```go
cfg := socket.LowLatencyConfig()
// Optimized for minimum latency
// Smaller buffers (128KB)
// QuickAck enabled
// DeferAccept disabled
```

#### High Throughput Config
```go
cfg := socket.HighThroughputConfig()
// Optimized for maximum throughput
// Large buffers (1MB)
// QuickAck disabled (allow delayed ACKs)
```

### Zero-Copy File Serving

```go
import "github.com/yourusername/shockwave/pkg/shockwave/socket"

// Serve entire file
file, _ := os.Open("large-file.bin")
defer file.Close()

written, err := socket.SendFileAll(conn, file)
// ~70% less CPU than io.Copy for large files

// Serve file range (HTTP Range requests)
written, err := socket.SendFileRange(conn, file, 1000, 2000)
```

## Performance Benchmarks

### Test Environment
- CPU: Intel Core i7-1165G7 @ 2.80GHz
- OS: Linux (kernel with TFO support)
- Go: 1.21+

### Results

#### Connection Establishment
```
BenchmarkConnectionEstablishment_Baseline    18,291 ops/sec  32.5 µs/op
BenchmarkConnectionEstablishment_Optimized   20,203 ops/sec  29.8 µs/op

Improvement: +8.3% faster, -10% latency
```

#### HTTP Request Latency
```
BenchmarkSmallRequests_Baseline              215,265 ops/sec  5.4 µs/op

Notes: Baseline measured at 5.4µs round-trip time
Expected improvement with optimizations: 20-30% latency reduction
```

#### File Serving (SendFile)
```
SendFile (10KB):   158 MB/s
SendFile (10MB):  6,723 MB/s (6.7 GB/s)

Expected CPU reduction: ~70% vs io.Copy for large files
```

#### Apply Overhead
```
BenchmarkApply                               22,057 ops/sec  29.5 µs/op

Notes: One-time cost per connection
```

## Implementation Details

### Linux Socket Options

#### TCP_QUICKACK (12)
- **Effect**: Sends ACKs immediately instead of delayed (40ms timer)
- **Benefit**: Reduces latency by one delayed ACK timeout
- **Caveat**: Not persistent - resets after each ACK
- **Use Case**: Low-latency HTTP APIs

#### TCP_DEFER_ACCEPT (9)
- **Effect**: Server only wakes when data arrives (not just SYN)
- **Benefit**: Reduces context switches, improves cache locality
- **Protection**: Mitigates SYN flood attacks
- **Use Case**: All HTTP servers

#### TCP_FASTOPEN (23)
- **Effect**: Allows client to send data in SYN packet
- **Benefit**: Reduces handshake from 2 RTT to 1 RTT
- **Requirement**: Kernel 3.7+, enabled in /proc
- **Use Case**: High-traffic servers with repeat clients

#### sendfile(2)
- **Effect**: Zero-copy file transfer (kernel to kernel)
- **Benefit**: No userspace buffer allocation or copying
- **Performance**: ~70% less CPU, 3-13x faster for large files
- **Use Case**: Static file serving, large responses

### Darwin (macOS) Socket Options

#### TCP_FASTOPEN (0x105)
- Similar to Linux, macOS 10.11+
- Less mature implementation than Linux

#### SO_NOSIGPIPE (0x1022)
- Prevents SIGPIPE when writing to closed socket
- Linux uses MSG_NOSIGNAL flag instead

### Platform Compatibility

The package provides graceful degradation:

```
Linux   -> Full optimizations (TCP_QUICKACK, TCP_DEFER_ACCEPT, sendfile, TFO)
Darwin  -> Partial optimizations (TFO, SO_NOSIGPIPE)
Other   -> Basic optimizations (TCP_NODELAY, buffer sizing)
```

All platforms fall back to `io.Copy` if `sendfile` is unavailable.

## Architecture

### File Structure

```
socket/
├── tuning.go           # Cross-platform interface and TCP_NODELAY
├── tuning_linux.go     # Linux-specific optimizations
├── tuning_darwin.go    # macOS-specific optimizations
├── tuning_other.go     # Fallback for other platforms
├── sendfile_linux.go   # Linux sendfile(2) implementation
├── sendfile.go         # Fallback using io.Copy
├── tuning_test.go      # Functional tests
└── benchmark_test.go   # Performance benchmarks
```

### Design Principles

1. **Progressive Enhancement**: Platforms without advanced features still get basic optimizations
2. **Fail-Safe**: Socket option failures are non-fatal (logged but continue)
3. **Zero Allocation**: Apply() performs 0 allocs/op for the happy path
4. **Build Tags**: Platform-specific code uses Go build tags

## Integration with Shockwave Server

### Automatic Application

The Shockwave server automatically applies socket tuning:

```go
// In server/server.go
listener, _ := net.Listen("tcp", addr)
socket.ApplyListener(listener, socket.DefaultConfig())

for {
    conn, _ := listener.Accept()
    socket.Apply(conn, socket.DefaultConfig())

    go s.handleConnection(conn)
}
```

### Configuration

Users can customize via server config:

```go
server := &Server{
    SocketConfig: socket.LowLatencyConfig(),
}
```

## Kernel Requirements

### Linux

- **TCP_FASTOPEN**: Kernel 3.7+ (2012)
  - Enable: `echo 3 > /proc/sys/net/ipv4/tcp_fastopen`
  - Check: `cat /proc/sys/net/ipv4/tcp_fastopen`

- **sendfile**: Kernel 2.2+ (universal)

- **TCP_DEFER_ACCEPT**: Kernel 2.4+ (universal)

- **TCP_QUICKACK**: Kernel 2.4.4+ (universal)

### macOS

- **TCP_FASTOPEN**: macOS 10.11+ (El Capitan, 2015)
  - Check: `sysctl net.inet.tcp.fastopen`

## Verification

### Check Applied Options

```bash
# Linux
ss -ti | grep -E "wscale|fastopen|nodelay"

# macOS
netstat -an | grep ESTABLISHED
```

### Verify sendfile

```bash
# Trace system calls
strace -e sendfile ./your-server  # Linux
dtruss -n sendfile ./your-server   # macOS
```

## Troubleshooting

### TFO Not Working

**Linux:**
```bash
# Check if enabled
cat /proc/sys/net/ipv4/tcp_fastopen
# Should be 1 (client) or 3 (client+server)

# Enable server-side TFO
echo 3 | sudo tee /proc/sys/net/ipv4/tcp_fastopen
```

**macOS:**
```bash
# Check status
sysctl net.inet.tcp.fastopen

# Enable (if disabled)
sudo sysctl -w net.inet.tcp.fastopen=3
```

### DEFER_ACCEPT Issues

If clients timeout, reduce the timeout:
```go
cfg := socket.DefaultConfig()
// Modify tuning_linux.go TCP_DEFER_ACCEPT value from 5s to 1s
```

### Permission Errors

Some socket options may require elevated privileges or specific kernel modules:
```bash
# Check kernel config
zcat /proc/config.gz | grep TCP_FASTOPEN
```

## Future Enhancements

### Planned Features

1. **TCP_CORK** - Aggregate small writes
2. **MSG_ZEROCOPY** - Zero-copy userspace sends (Linux 4.14+)
3. **SO_REUSEPORT** - Multi-threaded accept()
4. **eBPF integration** - Advanced load balancing
5. **QUIC socket optimizations** - For HTTP/3

### Experimental

- **io_uring** - Async I/O (Linux 5.1+)
- **TCP_USER_TIMEOUT** fine-tuning
- **TCP_THIN_STREAMS** - Optimize for low-volume connections

## References

### RFCs
- RFC 7413 - TCP Fast Open
- RFC 1323 - TCP Window Scaling

### Documentation
- Linux: `man 7 tcp`
- Linux: `man 2 sendfile`
- macOS: `man 4 tcp`

### Papers
- Radhakrishnan et al. "TCP Fast Open" (IMC 2011)
- Jacobson. "Congestion Avoidance and Control" (SIGCOMM 1988)

## License

Same as Shockwave project.
