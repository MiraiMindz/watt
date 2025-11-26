---
name: shockwave-integration
description: Expert knowledge of integrating Shockwave HTTP server into Bolt framework. Use when working with HTTP server setup, request handling, response writing, or zero-copy optimizations with Shockwave.
allowed-tools: Read, Grep, Write, Edit, Bash
---

# Shockwave Integration Skill

## When to Use This Skill

Invoke this skill when you need to:
- Set up Shockwave HTTP/1.1 or HTTP/2 servers
- Convert between Shockwave and Bolt types
- Optimize request/response handling with Shockwave
- Implement zero-copy patterns with Shockwave
- Debug Shockwave integration issues

## Core Shockwave Concepts

Shockwave is a high-performance HTTP implementation that achieves 40-60% better performance than `net/http` through:

1. **Zero-Copy Request Parsing** - Parse HTTP without allocating
2. **Built-in Connection Pooling** - Automatic connection reuse
3. **Optimized Response Writing** - Direct buffer writes
4. **Custom Memory Management** - Arena-based allocators

## Shockwave Type Reference

### Request Types
```go
// From: github.com/yourusername/shockwave/pkg/shockwave/http11

type Request struct {
    // Zero-copy accessors (return string views into buffer)
    Method() string
    Path() string
    Query() string
    Protocol() string

    // Headers
    Header() *Header

    // Body
    Body() io.Reader
    ContentLength() int64
}

type Header struct {
    Get(key string) string
    Set(key, value string)
    Del(key string)
    Keys() []string
}
```

### Response Types
```go
type ResponseWriter struct {
    // Status
    WriteHeader(statusCode int)

    // Headers
    Header() *Header

    // Body writing
    Write([]byte) (int, error)
    WriteString(string) (int, error)
    WriteJSON(statusCode int, data []byte) error

    // Flush
    Flush() error
}
```

### Server Types
```go
// From: github.com/yourusername/shockwave/pkg/shockwave/server

type Config struct {
    Addr                    string
    Handler                 func(*http11.ResponseWriter, *http11.Request)
    ReadTimeout             time.Duration
    WriteTimeout            time.Duration
    IdleTimeout             time.Duration
    MaxHeaderBytes          int
    MaxRequestBodySize      int64
    MaxKeepAliveRequests    int
    MaxConcurrentConnections int
    DisableKeepalive        bool
    EnableStats             bool  // Disable for zero-allocation mode
}

type Server struct {
    ListenAndServe() error
    ListenAndServeTLS(certFile, keyFile string) error
    Shutdown(ctx context.Context) error
    Stats() *Stats
}
```

## Integration Patterns

### Pattern 1: Basic Server Setup

```go
package main

import (
    "github.com/yourusername/shockwave/pkg/shockwave/http11"
    "github.com/yourusername/shockwave/pkg/shockwave/server"
    "github.com/yourusername/bolt/core"
)

func main() {
    // Create Bolt app
    app := bolt.New()
    app.Get("/ping", func(c *bolt.Context) error {
        return c.JSON(200, map[string]string{"message": "pong"})
    })

    // Create Shockwave config
    config := server.DefaultConfig()
    config.Addr = ":8080"
    config.EnableStats = false  // Zero-allocation mode

    // Integrate: Shockwave → Bolt
    config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
        app.handleShockwaveRequest(w, r)
    }

    // Start server
    srv := server.NewServer(config)
    srv.ListenAndServe()
}
```

### Pattern 2: Context Mapping (Zero-Allocation)

```go
// In: bolt/core/app.go

func (app *App) handleShockwaveRequest(w *http11.ResponseWriter, r *http11.Request) {
    // Acquire context from pool (0 allocs)
    ctx := app.contextPool.Acquire()
    defer app.contextPool.Release(ctx)

    // Map Shockwave request to Bolt context (zero-copy)
    ctx.shockwaveReq = r
    ctx.shockwaveRes = w

    // Zero-copy string views (no allocation)
    ctx.method = r.Method()
    ctx.path = r.Path()
    ctx.query = r.Query()

    // Route and execute
    if err := app.router.ServeHTTP(ctx); err != nil {
        app.errorHandler(ctx, err)
    }
}
```

### Pattern 3: Parameter Extraction (Zero-Copy)

```go
// In: bolt/core/context.go

type Context struct {
    shockwaveReq *http11.Request
    shockwaveRes *http11.ResponseWriter

    // Zero-copy string views (point into Shockwave buffer)
    method string
    path   string
    query  string
}

// Param returns path parameter (zero-copy)
func (c *Context) Param(key string) string {
    // Returns string view into Shockwave buffer
    // No allocation, no copying
    return c.params[key]
}

// Query returns query parameter (zero-copy)
func (c *Context) Query(key string) string {
    // Parse query string on first access
    if c.queryParams == nil {
        c.parseQuery()
    }
    return c.queryParams[key]
}

// parseQuery uses Shockwave's zero-copy query string
func (c *Context) parseQuery() {
    queryStr := c.shockwaveReq.Query()
    // Parse without copying
    c.queryParams = parseQueryString(queryStr)
}
```

### Pattern 4: Response Writing (Zero-Copy)

```go
// JSON response with buffer pooling
func (c *Context) JSON(status int, data interface{}) error {
    // Get buffer from pool
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()

    // Encode to buffer
    if err := json.NewEncoder(buf).Encode(data); err != nil {
        return err
    }

    // Write using Shockwave (zero-copy when possible)
    c.shockwaveRes.Header().Set("Content-Type", "application/json")
    c.shockwaveRes.WriteHeader(status)
    _, err := c.shockwaveRes.Write(buf.Bytes())
    return err
}

// Text response (zero-copy string)
func (c *Context) Text(status int, text string) error {
    c.shockwaveRes.Header().Set("Content-Type", "text/plain")
    c.shockwaveRes.WriteHeader(status)
    _, err := c.shockwaveRes.WriteString(text)
    return err
}
```

### Pattern 5: Header Access (Zero-Copy)

```go
// Read request header
func (c *Context) GetHeader(key string) string {
    return c.shockwaveReq.Header().Get(key)
}

// Set response header
func (c *Context) SetHeader(key, value string) {
    c.shockwaveRes.Header().Set(key, value)
}

// Example: Authentication
func (c *Context) getAuthToken() string {
    // Zero-copy header access
    auth := c.GetHeader("Authorization")
    if strings.HasPrefix(auth, "Bearer ") {
        return auth[7:]  // String slice (zero-copy)
    }
    return ""
}
```

### Pattern 6: Body Reading

```go
// Read body with size limit
func (c *Context) ReadBody(maxSize int64) ([]byte, error) {
    // Check content length
    if c.shockwaveReq.ContentLength() > maxSize {
        return nil, ErrRequestTooLarge
    }

    // Read body
    body := c.shockwaveReq.Body()
    data, err := io.ReadAll(io.LimitReader(body, maxSize))
    if err != nil {
        return nil, err
    }

    return data, nil
}

// Bind JSON from body
func (c *Context) BindJSON(v interface{}) error {
    body := c.shockwaveReq.Body()
    decoder := json.NewDecoder(body)
    decoder.DisallowUnknownFields()
    return decoder.Decode(v)
}
```

### Pattern 7: Graceful Shutdown

```go
func (app *App) Run(addr string) error {
    config := server.DefaultConfig()
    config.Addr = addr
    config.Handler = app.handleShockwaveRequest

    srv := server.NewServer(config)

    // Start server in background
    go func() {
        if err := srv.ListenAndServe(); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()

    // Wait for interrupt
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    // Graceful shutdown
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    return srv.Shutdown(ctx)
}
```

## Performance Optimization Checklist

When integrating Shockwave, ensure:

### ✓ Zero-Copy Paths
- [ ] Use string views from Shockwave request (don't copy)
- [ ] Return slices into Shockwave buffers when safe
- [ ] Avoid `string([]byte)` conversions
- [ ] Use `WriteString` instead of `Write([]byte(str))`

### ✓ Buffer Pooling
- [ ] Pool response buffers
- [ ] Pool JSON encoding buffers
- [ ] Return buffers to pool after use
- [ ] Reset buffers before pooling

### ✓ Context Pooling
- [ ] Acquire contexts from pool
- [ ] Reset contexts before release
- [ ] Release contexts with defer
- [ ] Clear Shockwave references on reset

### ✓ Configuration
- [ ] Disable stats in production (`EnableStats = false`)
- [ ] Set appropriate timeouts
- [ ] Configure connection limits
- [ ] Enable keep-alive (unless incompatible)

### ✓ Memory Limits
- [ ] Set MaxRequestBodySize
- [ ] Set MaxHeaderBytes
- [ ] Set MaxConcurrentConnections
- [ ] Implement request size validation

## Common Integration Issues

### Issue 1: Context Not Reset Properly

**Problem:**
```go
func (c *Context) Reset() {
    // WRONG: Shockwave reference not cleared
    c.params = nil
}
```

**Solution:**
```go
func (c *Context) Reset() {
    // Clear Shockwave references
    c.shockwaveReq = nil
    c.shockwaveRes = nil

    // Clear data
    c.method = ""
    c.path = ""
    c.params = nil
}
```

### Issue 2: String Allocation

**Problem:**
```go
// Allocates: converts []byte to string
path := string(r.Path())
```

**Solution:**
```go
// Zero-copy: r.Path() already returns string view
path := r.Path()
```

### Issue 3: Buffer Not Pooled

**Problem:**
```go
func (c *Context) JSON(status int, data interface{}) error {
    // WRONG: New buffer every time
    buf := new(bytes.Buffer)
    json.NewEncoder(buf).Encode(data)
    // ...
}
```

**Solution:**
```go
func (c *Context) JSON(status int, data interface{}) error {
    // RIGHT: Pool the buffer
    buf := bufferPool.Get().(*bytes.Buffer)
    defer bufferPool.Put(buf)
    buf.Reset()

    json.NewEncoder(buf).Encode(data)
    // ...
}
```

### Issue 4: Stats Enabled in Production

**Problem:**
```go
config := server.DefaultConfig()
config.EnableStats = true  // WRONG: Adds overhead
```

**Solution:**
```go
config := server.DefaultConfig()
config.EnableStats = false  // RIGHT: Zero-allocation mode
```

## Testing Shockwave Integration

### Unit Test Example
```go
func TestShockwaveIntegration(t *testing.T) {
    app := New()
    app.Get("/test", func(c *Context) error {
        return c.JSON(200, map[string]string{"status": "ok"})
    })

    // Create mock Shockwave request
    req := createMockShockwaveRequest("GET", "/test")
    res := createMockShockwaveResponse()

    // Handle request
    app.handleShockwaveRequest(res, req)

    // Verify response
    if res.StatusCode() != 200 {
        t.Errorf("Expected 200, got %d", res.StatusCode())
    }
}
```

### Benchmark Example
```go
func BenchmarkShockwaveIntegration(b *testing.B) {
    app := New()
    app.Get("/bench", func(c *Context) error {
        return c.JSON(200, map[string]string{"n": "value"})
    })

    req := createMockShockwaveRequest("GET", "/bench")
    res := createMockShockwaveResponse()

    b.ReportAllocs()
    b.ResetTimer()

    for i := 0; i < b.N; i++ {
        app.handleShockwaveRequest(res, req)
        res.Reset()
    }

    // Target: <1000ns/op, <3 allocs/op
}
```

## Reference Documentation

- **Shockwave Usage Guide:** `../shockwave/USAGE_GUIDE.md`
- **HTTP/1.1 API:** `../shockwave/pkg/shockwave/http11/`
- **Server API:** `../shockwave/pkg/shockwave/server/`

## Best Practices Summary

1. **Always use Shockwave types** - Never import `net/http`
2. **Zero-copy when safe** - Use string views, don't copy
3. **Pool everything** - Contexts, buffers, connections
4. **Disable stats in prod** - `EnableStats = false`
5. **Set limits** - Body size, headers, connections
6. **Reset properly** - Clear Shockwave refs in Reset()
7. **Test integration** - Unit + integration + benchmarks

---

**This skill makes you an expert in Shockwave integration. Use it whenever working with HTTP server concerns.**
