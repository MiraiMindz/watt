# Protocol Compliance Testing

Run comprehensive protocol compliance tests against HTTP RFCs.

## Your Task

1. Run all protocol tests:
   ```bash
   go test -v ./pkg/shockwave/http11 -run TestRFC
   go test -v ./pkg/shockwave/http2 -run TestRFC
   go test -v ./pkg/shockwave/websocket -run TestRFC
   ```

2. Test categories to verify:
   - **HTTP/1.1** (RFC 7230-7235):
     - Request line parsing
     - Header validation
     - Transfer-Encoding: chunked
     - Connection: keep-alive
     - Upgrade mechanism

   - **HTTP/2** (RFC 7540):
     - Frame parsing
     - Stream multiplexing
     - HPACK compression
     - Flow control
     - Server push

   - **WebSocket** (RFC 6455):
     - Upgrade handshake
     - Frame masking
     - Control frames
     - Fragmentation

3. Security compliance:
   - Request smuggling prevention
   - Header injection protection
   - DoS mitigation (header limits, body limits)
   - Malformed input rejection

4. For each failure:
   - RFC section violated
   - Test case that failed
   - Expected behavior
   - Actual behavior
   - Fix recommendation

5. Use external validators if available:
   ```bash
   # Test with curl
   curl -v http://localhost:8080

   # HTTP/2 with h2load
   h2load -n 1000 -c 10 http://localhost:8080

   # WebSocket with websocat
   websocat ws://localhost:8080/ws
   ```

6. Generate compliance report with pass/fail for each RFC section tested.

Invoke `http-protocol-testing` skill for detailed RFC validation.
