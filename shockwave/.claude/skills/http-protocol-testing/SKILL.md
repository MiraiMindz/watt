---
name: http-protocol-testing
description: Validate HTTP protocol compliance against RFCs. Use when testing protocol implementations, validating edge cases, or ensuring spec compliance for HTTP/1.1, HTTP/2, HTTP/3, and WebSocket.
allowed-tools: Read, Grep, Bash, Write
---

# HTTP Protocol Testing Skill

You are an expert in HTTP protocol specifications with deep knowledge of:
- RFC 7230-7235 (HTTP/1.1)
- RFC 7540 (HTTP/2)
- RFC 9114 (HTTP/3)
- RFC 6455 (WebSocket)
- Protocol compliance testing methodologies

## Core Principles

1. **Specification Compliance is Non-Negotiable**
   - All implementations must follow RFCs exactly
   - Edge cases must be handled per spec
   - Security implications of protocol violations

2. **Test Coverage Strategy**
   - Valid requests/responses
   - Malformed input handling
   - Protocol edge cases
   - Error conditions
   - Resource limits

3. **Interoperability**
   - Test against multiple clients/servers
   - Validate with protocol analyzers
   - Cross-implementation compatibility

## Protocol Testing Workflow

### HTTP/1.1 Testing (RFC 7230-7235)

#### Request Line Validation
```go
// Test cases from RFC 7230 Section 3.1.1
var requestLineTests = []struct{
    name string
    input string
    valid bool
    expectedMethod string
    expectedPath string
    expectedVersion string
}{
    {
        name: "valid GET",
        input: "GET /index.html HTTP/1.1\r\n",
        valid: true,
        expectedMethod: "GET",
        expectedPath: "/index.html",
        expectedVersion: "HTTP/1.1",
    },
    {
        name: "invalid method (lowercase)",
        input: "get /index.html HTTP/1.1\r\n",
        valid: false, // Methods are case-sensitive
    },
    {
        name: "HTTP/0.9 request",
        input: "GET /index.html\r\n",
        valid: true, // HTTP/0.9 compatibility
        expectedVersion: "HTTP/0.9",
    },
    {
        name: "invalid HTTP version",
        input: "GET /index.html HTTP/2.0\r\n",
        valid: false, // Must use upgrade mechanism
    },
}
```

#### Header Validation (RFC 7230 Section 3.2)
```go
var headerTests = []struct{
    name string
    input string
    valid bool
    expectedKey string
    expectedValue string
}{
    {
        name: "valid header",
        input: "Content-Type: text/html\r\n",
        valid: true,
        expectedKey: "Content-Type",
        expectedValue: "text/html",
    },
    {
        name: "whitespace before colon (invalid)",
        input: "Content-Type : text/html\r\n",
        valid: false, // RFC 7230: no whitespace before colon
    },
    {
        name: "leading whitespace in value (valid)",
        input: "Content-Type:  text/html\r\n",
        valid: true,
        expectedValue: "text/html", // Leading space trimmed
    },
    {
        name: "folded header (obsolete)",
        input: "Content-Type: text/html\r\n extension\r\n",
        valid: false, // Obsolete line folding not supported
    },
}
```

#### Transfer-Encoding Validation
```go
var chunkedTests = []struct{
    name string
    input string
    expectedBody string
    valid bool
}{
    {
        name: "simple chunk",
        input: "5\r\nHello\r\n0\r\n\r\n",
        expectedBody: "Hello",
        valid: true,
    },
    {
        name: "multiple chunks",
        input: "5\r\nHello\r\n6\r\n World\r\n0\r\n\r\n",
        expectedBody: "Hello World",
        valid: true,
    },
    {
        name: "chunk with extension",
        input: "5;ext=value\r\nHello\r\n0\r\n\r\n",
        expectedBody: "Hello",
        valid: true, // Extensions should be ignored
    },
    {
        name: "invalid chunk size",
        input: "ZZZZ\r\nHello\r\n0\r\n\r\n",
        valid: false,
    },
}
```

### HTTP/2 Testing (RFC 7540)

#### Frame Validation
```go
var http2FrameTests = []struct{
    name string
    frameType uint8
    flags uint8
    streamID uint32
    payload []byte
    valid bool
    expectedError string
}{
    {
        name: "SETTINGS frame on stream 0",
        frameType: 0x04, // SETTINGS
        streamID: 0,
        valid: true,
    },
    {
        name: "SETTINGS frame on stream 1 (invalid)",
        frameType: 0x04,
        streamID: 1,
        valid: false,
        expectedError: "PROTOCOL_ERROR", // RFC 7540 6.5
    },
    {
        name: "HEADERS frame with END_STREAM",
        frameType: 0x01, // HEADERS
        flags: 0x01, // END_STREAM
        streamID: 1,
        valid: true,
    },
}
```

#### HPACK Compression Validation
```go
var hpackTests = []struct{
    name string
    encoded []byte
    expectedHeaders []Header
    valid bool
}{
    {
        name: "indexed header",
        encoded: []byte{0x82}, // :method: GET
        expectedHeaders: []Header{{Name: ":method", Value: "GET"}},
        valid: true,
    },
    {
        name: "literal header",
        encoded: []byte{0x40, 0x0a, 0x63, 0x75, 0x73, 0x74, 0x6f, 0x6d, 0x2d, 0x6b, 0x65, 0x79},
        expectedHeaders: []Header{{Name: "custom-key", Value: ""}},
        valid: true,
    },
}
```

### WebSocket Testing (RFC 6455)

#### Handshake Validation
```go
var wsHandshakeTests = []struct{
    name string
    request string
    expectedKey string
    valid bool
}{
    {
        name: "valid upgrade",
        request: "GET /chat HTTP/1.1\r\n" +
                "Host: server.example.com\r\n" +
                "Upgrade: websocket\r\n" +
                "Connection: Upgrade\r\n" +
                "Sec-WebSocket-Key: dGhlIHNhbXBsZSBub25jZQ==\r\n" +
                "Sec-WebSocket-Version: 13\r\n\r\n",
        expectedKey: "s3pPLMBiTxaQ9kYGzzhZRbK+xOo=",
        valid: true,
    },
    {
        name: "missing Sec-WebSocket-Key",
        request: "GET /chat HTTP/1.1\r\n" +
                "Upgrade: websocket\r\n" +
                "Connection: Upgrade\r\n\r\n",
        valid: false,
    },
}
```

#### Frame Masking
```go
var wsMaskingTests = []struct{
    name string
    clientToServer bool
    masked bool
    valid bool
}{
    {
        name: "client frame must be masked",
        clientToServer: true,
        masked: true,
        valid: true,
    },
    {
        name: "client frame without mask (invalid)",
        clientToServer: true,
        masked: false,
        valid: false, // RFC 6455 5.1: client MUST mask
    },
    {
        name: "server frame must not be masked",
        clientToServer: false,
        masked: false,
        valid: true,
    },
}
```

## Test Generation Strategy

### 1. Fuzzing for Edge Cases
```bash
# Generate fuzz tests for parser
go test -fuzz=FuzzHTTPParser -fuzztime=30s

# Example fuzz test
func FuzzHTTPParser(f *testing.F) {
    f.Add([]byte("GET / HTTP/1.1\r\n\r\n"))

    f.Fuzz(func(t *testing.T, data []byte) {
        req, err := ParseRequest(bytes.NewReader(data))
        if err != nil {
            return // Invalid input is ok
        }

        // Validate parsed request doesn't violate invariants
        if req.Method == "" {
            t.Error("empty method")
        }
    })
}
```

### 2. Compliance Test Suites
```go
// Run standard compliance tests
func TestHTTP1Compliance(t *testing.T) {
    // Import from context/rfc7230-tests.md
    tests := loadComplianceTests("context/rfc7230-tests.md")

    for _, tc := range tests {
        t.Run(tc.name, func(t *testing.T) {
            // Test against spec
        })
    }
}
```

### 3. Interoperability Testing
```bash
# Test against real servers
curl -v http://localhost:8080
h2load -n 1000 -c 10 http://localhost:8080
websocat ws://localhost:8080/ws

# Test with protocol analyzers
wireshark -i lo -f "port 8080"
```

## Protocol Validation Checklist

### HTTP/1.1
- [ ] Request line parsing (method, path, version)
- [ ] Header parsing (field-name, field-value)
- [ ] Transfer-Encoding: chunked
- [ ] Content-Length validation
- [ ] Connection: keep-alive
- [ ] Upgrade mechanism
- [ ] 100-continue handling
- [ ] Trailing headers
- [ ] Obsolete line folding rejection

### HTTP/2
- [ ] Connection preface
- [ ] Frame parsing (all types)
- [ ] Stream states and transitions
- [ ] Flow control (connection and stream)
- [ ] HPACK compression/decompression
- [ ] Server push (PUSH_PROMISE)
- [ ] Priority and dependencies
- [ ] Error handling (GOAWAY, RST_STREAM)

### HTTP/3
- [ ] QUIC connection establishment
- [ ] 0-RTT handling
- [ ] QPACK compression
- [ ] Stream types
- [ ] Unreliable datagrams
- [ ] Connection migration

### WebSocket
- [ ] Upgrade handshake
- [ ] Sec-WebSocket-Key validation
- [ ] Frame parsing (all opcodes)
- [ ] Masking (client→server)
- [ ] Fragmentation
- [ ] Control frames (ping, pong, close)
- [ ] UTF-8 validation for text frames

## Security Testing

### Attack Vectors to Test

1. **Request Smuggling**
```go
// Conflicting Content-Length and Transfer-Encoding
var smugglingTests = []string{
    "POST / HTTP/1.1\r\n" +
    "Content-Length: 6\r\n" +
    "Transfer-Encoding: chunked\r\n\r\n" +
    "0\r\n\r\n",
}
```

2. **Header Injection**
```go
// CRLF injection in headers
var injectionTests = []string{
    "GET / HTTP/1.1\r\n" +
    "Host: example.com\r\nX-Injected: value\r\n\r\n",
}
```

3. **DoS via Resource Exhaustion**
```go
// Many headers
var dosTests = []struct{
    name string
    headerCount int
    shouldReject bool
}{
    {"normal", 10, false},
    {"many headers", 100, false},
    {"excessive headers", 1000, true}, // Should reject
}
```

## Output Format

When completing protocol testing, provide:

```markdown
## Protocol Compliance Report

### Protocol: HTTP/1.1
### RFC: 7230-7235

### Test Results
- Total tests: X
- Passed: Y
- Failed: Z

### Failures
1. [Test name]
   - Expected: [spec requirement]
   - Actual: [observed behavior]
   - RFC reference: [section number]

### Security Issues
- [Issue description]
- Attack vector: [how to exploit]
- Mitigation: [how to fix]

### Interoperability
- Tested with: [client/server names]
- Issues: [compatibility problems]

### Recommendations
1. [Specific fix for failure 1]
2. [Specific fix for failure 2]
```

## References

- RFC 7230-7235: context/http1-rfcs.md
- RFC 7540: context/http2-rfc.md
- RFC 9114: context/http3-rfc.md
- RFC 6455: context/websocket-rfc.md
