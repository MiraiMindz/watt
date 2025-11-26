# CRITICAL SECURITY FIXES REQUIRED

## DO NOT USE IN PRODUCTION UNTIL THESE ARE FIXED

---

## P0 - CRITICAL (Fix Before ANY Production Use)

### 1. HTTP Request Smuggling - CL.TE Attack ⚠️ CRITICAL

**File**: `parser.go:319-346`

**Issue**: Requests with both `Content-Length` and `Transfer-Encoding` are accepted with both fields set.

**Fix**:
```go
func (p *Parser) processSpecialHeader(req *Request, name, value []byte) error {
    // Content-Length
    if bytesEqualCaseInsensitive(name, headerContentLength) {
        // SECURITY: Reject if Transfer-Encoding already set
        if len(req.TransferEncoding) > 0 {
            return ErrRequestSmuggling
        }
        contentLength, err := parseContentLength(value)
        if err != nil {
            return ErrInvalidContentLength
        }
        req.ContentLength = contentLength
        return nil
    }

    // Transfer-Encoding
    if bytesEqualCaseInsensitive(name, headerTransferEncoding) {
        // SECURITY: Reject if Content-Length already set
        if req.ContentLength > 0 {
            return ErrRequestSmuggling
        }
        if bytesEqualCaseInsensitive(value, headerChunked) {
            req.TransferEncoding = []string{"chunked"}
        }
        return nil
    }

    // Connection
    if bytesEqualCaseInsensitive(name, headerConnection) {
        if bytesEqualCaseInsensitive(value, headerClose) {
            req.Close = true
        }
        return nil
    }

    return nil
}
```

**Add to errors.go**:
```go
ErrRequestSmuggling = errors.New("http11: conflicting Content-Length and Transfer-Encoding")
```

**Test**: `TestSecurity_RequestSmuggling_CLTE` (already exists)

---

### 2. Duplicate Content-Length Attack ⚠️ CRITICAL

**File**: `parser.go:259-315`

**Issue**: Multiple `Content-Length` headers with different values are accepted.

**Fix**:
```go
func (p *Parser) parseHeaders(req *Request, buf []byte) error {
    pos := 0
    seenContentLength := false
    var firstContentLength int64

    for {
        if pos >= len(buf) {
            break
        }

        if pos+1 < len(buf) && buf[pos] == '\r' && buf[pos+1] == '\n' {
            break
        }

        lineEnd := bytes.Index(buf[pos:], []byte("\r\n"))
        if lineEnd == -1 {
            return ErrInvalidHeader
        }
        lineEnd += pos

        line := buf[pos:lineEnd]
        colonIdx := bytes.IndexByte(line, ':')
        if colonIdx == -1 {
            return ErrInvalidHeader
        }

        name := line[:colonIdx]
        value := line[colonIdx+1:]

        // SECURITY: Validate no CRLF in name or value
        if bytes.IndexByte(name, '\r') != -1 || bytes.IndexByte(name, '\n') != -1 {
            return ErrInvalidHeader
        }
        if bytes.IndexByte(value, '\r') != -1 || bytes.IndexByte(value, '\n') != -1 {
            return ErrInvalidHeader
        }

        // SECURITY: Validate no whitespace in header name
        if bytes.IndexByte(name, ' ') != -1 || bytes.IndexByte(name, '\t') != -1 {
            return ErrInvalidHeader
        }

        value = trimLeadingSpace(value)
        value = trimTrailingSpace(value)

        // SECURITY: Special handling for Content-Length to detect duplicates
        if bytesEqualCaseInsensitive(name, headerContentLength) {
            cl, err := parseContentLength(value)
            if err != nil {
                return ErrInvalidContentLength
            }

            if seenContentLength {
                // Duplicate Content-Length - must have same value or reject
                if cl != firstContentLength {
                    return ErrDuplicateContentLength
                }
                // Same value - skip (don't add duplicate)
                pos = lineEnd + 2
                continue
            }

            seenContentLength = true
            firstContentLength = cl
        }

        if err := req.Header.Add(name, value); err != nil {
            return err
        }

        if err := p.processSpecialHeader(req, name, value); err != nil {
            return err
        }

        pos = lineEnd + 2
    }

    return nil
}
```

**Add to errors.go**:
```go
ErrDuplicateContentLength = errors.New("http11: conflicting duplicate Content-Length headers")
```

**Test**: `TestSecurity_RequestSmuggling_DualContentLength` (already exists)

---

### 3. Header CRLF Injection ⚠️ HIGH

**File**: `parser.go:259-315`

**Issue**: CRLF characters in header values are not validated.

**Fix**: Already included in fix #2 above:
```go
// SECURITY: Validate no CRLF in name or value
if bytes.IndexByte(name, '\r') != -1 || bytes.IndexByte(name, '\n') != -1 {
    return ErrInvalidHeader
}
if bytes.IndexByte(value, '\r') != -1 || bytes.IndexByte(value, '\n') != -1 {
    return ErrInvalidHeader
}

// SECURITY: Validate no whitespace in header name (before colon per RFC)
if bytes.IndexByte(name, ' ') != -1 || bytes.IndexByte(name, '\t') != -1 {
    return ErrInvalidHeader
}
```

**Test**: `TestSecurity_HeaderInjection_CRLF` (already exists)

---

### 4. Excessive URI Length DoS ⚠️ MEDIUM

**File**: `parser.go:193-251`

**Issue**: Request lines can exceed 8KB limit if headers are small.

**Fix**:
```go
func (p *Parser) parseRequestLine(req *Request, buf []byte) (int, error) {
    lineEnd := bytes.Index(buf, []byte("\r\n"))
    if lineEnd == -1 {
        return 0, ErrInvalidRequestLine
    }

    // SECURITY: Enforce request line size limit BEFORE parsing
    if lineEnd > MaxRequestLineSize {
        return 0, ErrRequestLineTooLarge
    }

    line := buf[:lineEnd]

    // ... rest of parsing unchanged ...
}
```

**Test**: `TestSecurity_VeryLongURI` (already exists)

---

## Combined Fix (All P0 Issues)

Apply these changes to `parser.go`:

```go
// parseHeaders - Updated version with all security fixes
func (p *Parser) parseHeaders(req *Request, buf []byte) error {
    pos := 0
    seenContentLength := false
    var firstContentLength int64

    for {
        if pos >= len(buf) {
            break
        }

        if pos+1 < len(buf) && buf[pos] == '\r' && buf[pos+1] == '\n' {
            break
        }

        lineEnd := bytes.Index(buf[pos:], []byte("\r\n"))
        if lineEnd == -1 {
            return ErrInvalidHeader
        }
        lineEnd += pos

        line := buf[pos:lineEnd]
        colonIdx := bytes.IndexByte(line, ':')
        if colonIdx == -1 {
            return ErrInvalidHeader
        }

        name := line[:colonIdx]
        value := line[colonIdx+1:]

        // SECURITY FIX #3: Validate no CRLF in name or value
        if bytes.IndexByte(name, '\r') != -1 || bytes.IndexByte(name, '\n') != -1 {
            return ErrInvalidHeader
        }
        if bytes.IndexByte(value, '\r') != -1 || bytes.IndexByte(value, '\n') != -1 {
            return ErrInvalidHeader
        }

        // SECURITY FIX #3: Validate no whitespace in header name
        if bytes.IndexByte(name, ' ') != -1 || bytes.IndexByte(name, '\t') != -1 {
            return ErrInvalidHeader
        }

        value = trimLeadingSpace(value)
        value = trimTrailingSpace(value)

        // SECURITY FIX #2: Special handling for Content-Length
        if bytesEqualCaseInsensitive(name, headerContentLength) {
            cl, err := parseContentLength(value)
            if err != nil {
                return ErrInvalidContentLength
            }

            if seenContentLength {
                if cl != firstContentLength {
                    return ErrDuplicateContentLength
                }
                pos = lineEnd + 2
                continue
            }

            seenContentLength = true
            firstContentLength = cl
        }

        if err := req.Header.Add(name, value); err != nil {
            return err
        }

        if err := p.processSpecialHeader(req, name, value); err != nil {
            return err
        }

        pos = lineEnd + 2
    }

    return nil
}

// parseRequestLine - Updated with URI length check
func (p *Parser) parseRequestLine(req *Request, buf []byte) (int, error) {
    lineEnd := bytes.Index(buf, []byte("\r\n"))
    if lineEnd == -1 {
        return 0, ErrInvalidRequestLine
    }

    // SECURITY FIX #4: Enforce request line size limit
    if lineEnd > MaxRequestLineSize {
        return 0, ErrRequestLineTooLarge
    }

    line := buf[:lineEnd]

    spaceIdx := bytes.IndexByte(line, ' ')
    if spaceIdx == -1 {
        return 0, ErrInvalidRequestLine
    }

    methodBytes := line[:spaceIdx]
    req.MethodID = ParseMethodID(methodBytes)
    if req.MethodID == MethodUnknown {
        return 0, ErrInvalidMethod
    }
    req.methodBytes = methodBytes

    line = line[spaceIdx+1:]
    spaceIdx = bytes.IndexByte(line, ' ')
    if spaceIdx == -1 {
        return 0, ErrInvalidRequestLine
    }

    uriBytes := line[:spaceIdx]

    queryIdx := bytes.IndexByte(uriBytes, '?')
    if queryIdx != -1 {
        req.pathBytes = uriBytes[:queryIdx]
        req.queryBytes = uriBytes[queryIdx+1:]
    } else {
        req.pathBytes = uriBytes
        req.queryBytes = nil
    }

    if len(req.pathBytes) == 0 {
        return 0, ErrInvalidPath
    }
    if req.pathBytes[0] != '/' && req.pathBytes[0] != '*' {
        return 0, ErrInvalidPath
    }

    line = line[spaceIdx+1:]
    req.protoBytes = line

    if !bytes.Equal(line, http11Bytes) {
        return 0, ErrInvalidProtocol
    }

    return lineEnd + 2, nil
}

// processSpecialHeader - Updated with smuggling detection
func (p *Parser) processSpecialHeader(req *Request, name, value []byte) error {
    // SECURITY FIX #1: Content-Length
    if bytesEqualCaseInsensitive(name, headerContentLength) {
        // Reject if Transfer-Encoding already set
        if len(req.TransferEncoding) > 0 {
            return ErrRequestSmuggling
        }
        contentLength, err := parseContentLength(value)
        if err != nil {
            return ErrInvalidContentLength
        }
        req.ContentLength = contentLength
        return nil
    }

    // SECURITY FIX #1: Transfer-Encoding
    if bytesEqualCaseInsensitive(name, headerTransferEncoding) {
        // Reject if Content-Length already set
        if req.ContentLength > 0 {
            return ErrRequestSmuggling
        }
        if bytesEqualCaseInsensitive(value, headerChunked) {
            req.TransferEncoding = []string{"chunked"}
        }
        return nil
    }

    // Connection
    if bytesEqualCaseInsensitive(name, headerConnection) {
        if bytesEqualCaseInsensitive(value, headerClose) {
            req.Close = true
        }
        return nil
    }

    return nil
}
```

**Add to errors.go**:
```go
// Security errors
var (
    // ErrRequestSmuggling indicates conflicting Content-Length and Transfer-Encoding
    ErrRequestSmuggling = errors.New("http11: conflicting Content-Length and Transfer-Encoding headers")

    // ErrDuplicateContentLength indicates multiple Content-Length headers with different values
    ErrDuplicateContentLength = errors.New("http11: conflicting duplicate Content-Length headers")
)
```

---

## Verification

After applying fixes, run:

```bash
# Run security tests
go test -v -run TestSecurity

# Expected output: ALL PASS

# Run all tests
go test ./...

# Run with race detector
go test -race ./...

# Run benchmarks to ensure no performance regression
go test -bench=. -benchmem > new.txt
# Compare with baseline to ensure <5% regression
```

---

## Estimated Time to Fix

- **Security fixes**: 90 minutes
- **Testing**: 30 minutes
- **Documentation**: 15 minutes
- **Total**: ~2.5 hours

---

## Post-Fix Validation Checklist

- [ ] All security tests pass
- [ ] All existing tests still pass
- [ ] Benchmarks show <5% regression
- [ ] Manual testing with curl shows correct rejection of attacks
- [ ] Code review by security-aware developer
- [ ] Fuzz testing (optional but recommended)

---

## Attack Test Cases (After Fix)

Test these scenarios - all should be REJECTED:

```bash
# Test 1: CL.TE Attack
echo -ne 'POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 6\r\nTransfer-Encoding: chunked\r\n\r\n0\r\n\r\n' | nc localhost 8080
# Expected: Connection closed with error

# Test 2: Duplicate Content-Length
echo -ne 'POST / HTTP/1.1\r\nHost: localhost\r\nContent-Length: 10\r\nContent-Length: 20\r\n\r\n' | nc localhost 8080
# Expected: Connection closed with error

# Test 3: CRLF Injection
echo -ne 'GET / HTTP/1.1\r\nHost: localhost\r\nX-Test: value\r\nInjected: header\r\n\r\n' | nc localhost 8080
# Expected: Connection closed with error

# Test 4: Excessive URI
python3 -c "print('GET /' + 'a'*10000 + ' HTTP/1.1\r\nHost: localhost\r\n\r\n')" | nc localhost 8080
# Expected: Connection closed with error
```

---

## References

- **RFC 7230**: https://www.rfc-editor.org/rfc/rfc7230
- **HTTP Request Smuggling**: https://portswigger.net/web-security/request-smuggling
- **HTTP Desync Attacks**: https://portswigger.net/research/http-desync-attacks

---

**Status**: FIXES REQUIRED BEFORE PRODUCTION USE
**Priority**: P0 - CRITICAL
**Assigned To**: Development Team
**Due Date**: ASAP
