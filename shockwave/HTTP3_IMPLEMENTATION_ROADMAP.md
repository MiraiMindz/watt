# HTTP/3 Implementation Roadmap
## QUIC-based HTTP for Shockwave

**Status:** Design Complete - Ready for Implementation
**Timeline:** 2-3 weeks for full implementation
**Priority:** High (requested feature)

---

## Current State

HTTP/3 skeletal structure exists in:
- `pkg/shockwave/http3/` - Basic scaffolding
- `pkg/shockwave/http3/quic/` - QUIC protocol stubs

**Missing Components:**
1. Complete QUIC implementation
2. QPACK header compression
3. HTTP/3 framing
4. 0-RTT support
5. Connection migration

---

## Phase 1: QUIC Transport (Week 1)

### 1.1 Connection Establishment
**File:** `pkg/shockwave/http3/quic/connection.go`

```go
type QUICConnection struct {
    conn        net.PacketConn
    remoteAddr  net.Addr
    streams     map[uint64]*Stream
    mu          sync.RWMutex

    // Congestion control
    congestion  *CongestionController
    rtt         time.Duration
    cwnd        uint64  // Congestion window

    // Crypto
    keys        *CryptoKeys
    tls         *tls.Conn
}

func (c *QUICConnection) Handshake() error {
    // 1-RTT or 0-RTT handshake
    // TLS 1.3 over QUIC
}
```

### 1.2 Stream Multiplexing
**File:** `pkg/shockwave/http3/quic/stream.go`

```go
type Stream struct {
    id          uint64
    conn        *QUICConnection
    sendBuf     *bytes.Buffer
    recvBuf     *bytes.Buffer
    state       StreamState
    flowControl *FlowController
}

func (s *Stream) Write(p []byte) (int, error) {
    // Fragment into STREAM frames
    // Apply flow control
    // Send with congestion control
}
```

### 1.3 Flow Control
**File:** `pkg/shockwave/http3/quic/flow_control.go`

```go
type FlowController struct {
    sent        uint64
    maxData     uint64
    consumed    uint64
    maxStreamData uint64
}

func (fc *FlowController) CanSend(n uint64) bool {
    return fc.sent + n <= fc.maxData
}
```

---

## Phase 2: HTTP/3 Framing (Week 2)

### 2.1 QPACK Header Compression
**File:** `pkg/shockwave/http3/qpack/encoder.go`

```go
type Encoder struct {
    dynamicTable *DynamicTable
    staticTable  *StaticTable
}

func (e *Encoder) Encode(headers [][2]string) ([]byte, error) {
    // Encode headers using QPACK
    // Reference static table
    // Update dynamic table
}
```

### 2.2 Frame Types
**File:** `pkg/shockwave/http3/frames.go`

```go
const (
    FrameData         = 0x00
    FrameHeaders      = 0x01
    FrameCancelPush   = 0x03
    FrameSettings     = 0x04
    FramePushPromise  = 0x05
    FrameGoAway       = 0x07
    FrameMaxPushID    = 0x0D
)

type Frame interface {
    Type() uint64
    Marshal() []byte
    Unmarshal([]byte) error
}
```

### 2.3 Request/Response Processing
**File:** `pkg/shockwave/http3/client.go`

```go
type HTTP3Client struct {
    conn     *quic.QUICConnection
    encoder  *qpack.Encoder
    decoder  *qpack.Decoder
    streams  map[uint64]*RequestStream
}

func (c *HTTP3Client) Do(req *Request) (*Response, error) {
    // 1. Open bidirectional stream
    stream := c.conn.OpenStream()

    // 2. Encode headers with QPACK
    headerBytes := c.encoder.Encode(req.Headers)

    // 3. Send HEADERS frame
    stream.WriteFrame(&HeadersFrame{
        Headers: headerBytes,
    })

    // 4. Send DATA frames
    stream.WriteFrame(&DataFrame{
        Data: req.Body,
    })

    // 5. Read response
    return c.readResponse(stream)
}
```

---

## Phase 3: Advanced Features (Week 3)

### 3.1 0-RTT (Zero Round Trip Time)
**File:** `pkg/shockwave/http3/zero_rtt.go`

```go
type SessionCache struct {
    sessions map[string]*SessionState
    mu       sync.RWMutex
}

func (c *HTTP3Client) DoWith0RTT(req *Request) (*Response, error) {
    // Check for cached session
    session := c.cache.Get(req.Host)
    if session == nil {
        return c.Do(req)  // Fallback to 1-RTT
    }

    // Send request immediately with 0-RTT
    // Replay protection required
    return c.do0RTT(req, session)
}
```

### 3.2 Connection Migration
**File:** `pkg/shockwave/http3/quic/migration.go`

```go
func (c *QUICConnection) Migrate(newAddr net.Addr) error {
    // 1. Validate new path
    // 2. Send PATH_CHALLENGE
    // 3. Wait for PATH_RESPONSE
    // 4. Update connection addr
    c.remoteAddr = newAddr
    return nil
}
```

### 3.3 Congestion Control
**File:** `pkg/shockwave/http3/quic/congestion.go`

```go
type CongestionController struct {
    algorithm string // "cubic", "bbr", "reno"
    cwnd      uint64
    ssthresh  uint64
    rtt       time.Duration
}

func (cc *CongestionController) OnAck(ackedBytes uint64) {
    switch cc.algorithm {
    case "cubic":
        cc.cubicOnAck(ackedBytes)
    case "bbr":
        cc.bbrOnAck(ackedBytes)
    }
}
```

---

## Integration with Existing Code

### Client Integration
**File:** `pkg/shockwave/client/client.go`

```go
type Client struct {
    preferHTTP3  bool
    http3Client  *http3.HTTP3Client
    // ... existing fields
}

func (c *Client) Do(req *ClientRequest) (*ClientResponse, error) {
    // Auto-detect HTTP/3 support via Alt-Svc
    if c.preferHTTP3 && c.supportsHTTP3(req.Host) {
        return c.http3Client.Do(req)
    }

    // Fallback to HTTP/1.1 or HTTP/2
    return c.doHTTP1(req)
}
```

### Server Integration
**File:** `pkg/shockwave/server/server.go`

```go
func (s *Server) ListenAndServeHTTP3() error {
    // Listen on UDP for QUIC
    conn, err := net.ListenPacket("udp", s.config.Addr)
    if err != nil {
        return err
    }

    // Start HTTP/3 server
    h3Server := http3.NewServer(s.config.Handler)
    return h3Server.Serve(conn)
}
```

---

## Performance Targets

### Latency
- **0-RTT:** < 10ms for cached sessions
- **1-RTT:** < 50ms for new connections
- **vs HTTP/2:** 20-40% faster on high-latency networks

### Throughput
- **Concurrent streams:** 100+ simultaneous requests
- **Data transfer:** > 1 Gbps on fast networks
- **CPU efficiency:** < 10% overhead vs HTTP/2

---

## Testing Strategy

### Unit Tests
```bash
go test ./pkg/shockwave/http3/... -v
```

### Integration Tests
```bash
go test ./pkg/shockwave/http3/integration_test.go -v
```

### Interoperability Tests
Test against:
- quic-go (reference implementation)
- Chrome/Chromium HTTP/3
- nginx with QUIC module
- Cloudflare QUIC

### Benchmark Tests
```bash
go test -bench=HTTP3 -benchmem -benchtime=300ms
```

---

## Dependencies

### External Libraries (Optional)
- `github.com/quic-go/quic-go` - Reference for spec compliance
- `github.com/lucas-clemente/quic-go` - Alternative implementation

### Pure Go Implementation (Preferred)
- Implement QUIC from scratch for full control
- Better integration with Shockwave's zero-allocation goals
- No external dependencies

---

## File Structure

```
pkg/shockwave/http3/
├── client.go              # HTTP/3 client
├── server.go              # HTTP/3 server
├── frames.go              # Frame definitions
├── frames_test.go
├── qpack/
│   ├── encoder.go         # QPACK encoder
│   ├── decoder.go         # QPACK decoder
│   ├── static_table.go    # Static table
│   ├── dynamic_table.go   # Dynamic table
│   └── qpack_test.go
├── quic/
│   ├── connection.go      # QUIC connection
│   ├── stream.go          # QUIC stream
│   ├── flow_control.go    # Flow control
│   ├── congestion.go      # Congestion control
│   ├── crypto.go          # Crypto (TLS 1.3)
│   ├── packet.go          # Packet encoding/decoding
│   ├── migration.go       # Connection migration
│   └── quic_test.go
└── integration_test.go    # Full stack tests
```

---

## Implementation Checklist

### Phase 1: QUIC Transport ✅
- [ ] Connection establishment (1-RTT)
- [ ] Stream multiplexing
- [ ] Flow control (stream + connection level)
- [ ] Loss detection and recovery
- [ ] Congestion control (Cubic)
- [ ] Packet pacing
- [ ] Connection migration
- [ ] Path validation

### Phase 2: HTTP/3 Framing ✅
- [ ] QPACK encoder
- [ ] QPACK decoder
- [ ] Static table
- [ ] Dynamic table
- [ ] Frame parsing (DATA, HEADERS, SETTINGS, etc.)
- [ ] Server push
- [ ] Prioritization

### Phase 3: Advanced Features ✅
- [ ] 0-RTT support
- [ ] Session resumption
- [ ] Alt-Svc header parsing
- [ ] BBR congestion control
- [ ] ECN support
- [ ] Datagram extension

---

## Expected Performance (Estimated)

### vs HTTP/2
```
Low latency (< 20ms RTT):    Similar performance
Medium latency (50-100ms):   20-30% faster
High latency (> 200ms):      40-50% faster (0-RTT)
Packet loss (1-5%):          30-40% better throughput
```

### Memory Usage
```
Connection overhead:  ~5 KB per connection
Stream overhead:      ~500 B per stream
QPACK tables:         ~16 KB (shared)
Total:               ~22 KB per client (acceptable)
```

### Allocations
```
With arenas:  2-3 allocs/op (QUIC framing)
Without:      8-10 allocs/op (header encoding)
```

---

## RFC Compliance

- **RFC 9114:** HTTP/3
- **RFC 9000:** QUIC Transport Protocol
- **RFC 9001:** Using TLS to Secure QUIC
- **RFC 9002:** QUIC Loss Detection and Congestion Control
- **RFC 7541:** HPACK (for comparison)
- **RFC 9204:** QPACK

---

## Next Steps

1. **Week 1:** Implement QUIC transport layer
   - Connection setup
   - Stream management
   - Flow control

2. **Week 2:** Implement HTTP/3 framing
   - QPACK compression
   - Frame encoding/decoding
   - Request/response handling

3. **Week 3:** Advanced features + testing
   - 0-RTT
   - Connection migration
   - Comprehensive testing

4. **Week 4:** Integration + benchmarking
   - Client/server integration
   - Performance tuning
   - Documentation

---

## Conclusion

HTTP/3 implementation will:
1. Reduce latency by 20-50% on high-latency networks
2. Improve resilience to packet loss
3. Enable 0-RTT for returning clients
4. Position Shockwave as a complete HTTP stack

**Recommendation:** Prioritize Phase 1 (QUIC transport) for immediate value, then iterate on advanced features based on user demand.
