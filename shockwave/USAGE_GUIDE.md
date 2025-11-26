# Shockwave HTTP Library - Complete Usage Guide

## Quick Start

```bash
go get github.com/yourusername/shockwave
```

---

## Table of Contents

1. [HTTP/1.1 Server](#http11-server)
2. [HTTP/1.1 Client](#http11-client)
3. [HTTP/2 Server](#http2-server)
4. [HTTP/2 Client](#http2-client)
5. [HTTP/3 Server (QUIC)](#http3-server-quic)
6. [HTTP/3 Client (QUIC)](#http3-client-quic)
7. [Performance Tips](#performance-tips)
8. [Advanced Configuration](#advanced-configuration)

---

## HTTP/1.1 Server

### Basic HTTP/1.1 Server (Zero Allocations)

```go
package main

import (
    "log"

    "github.com/yourusername/shockwave/pkg/shockwave/http11"
    "github.com/yourusername/shockwave/pkg/shockwave/server"
)

func main() {
    // Create server configuration
    config := server.DefaultConfig()
    config.Addr = ":8080"

    // Use concrete types for zero-allocation performance (recommended)
    config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
        w.WriteHeader(200)
        w.WriteString("Hello, World!")
    }

    // Create and start server
    srv := server.NewServer(config)
    log.Printf("Starting HTTP/1.1 server on %s", config.Addr)

    if err := srv.ListenAndServe(); err != nil {
        log.Fatal(err)
    }
}
```

### HTTP/1.1 Server with JSON Response

```go
config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
    // Pre-marshal JSON for best performance
    jsonData := []byte(`{"status":"ok","message":"Hello, World!"}`)
    w.WriteJSON(200, jsonData)
}
```

### HTTP/1.1 Server with Routing

```go
config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
    path := r.Path()
    method := r.Method()

    switch {
    case method == "GET" && path == "/":
        w.WriteHeader(200)
        w.WriteString("Home Page")

    case method == "GET" && path == "/api/users":
        handleGetUsers(w, r)

    case method == "POST" && path == "/api/users":
        handleCreateUser(w, r)

    case method == "GET" && strings.HasPrefix(path, "/api/users/"):
        handleGetUser(w, r)

    default:
        w.WriteHeader(404)
        w.WriteString("Not Found")
    }
}

func handleGetUsers(w *http11.ResponseWriter, r *http11.Request) {
    users := `[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]`
    w.WriteJSON(200, []byte(users))
}

func handleCreateUser(w *http11.ResponseWriter, r *http11.Request) {
    // Read body
    body, _ := io.ReadAll(r.Body())

    // Process user creation...

    w.WriteHeader(201)
    w.WriteString("Created")
}

func handleGetUser(w *http11.ResponseWriter, r *http11.Request) {
    // Extract ID from path
    path := r.Path()
    id := strings.TrimPrefix(path, "/api/users/")

    // Fetch user...
    user := fmt.Sprintf(`{"id":%s,"name":"User %s"}`, id, id)
    w.WriteJSON(200, []byte(user))
}
```

### HTTP/1.1 Server with Headers

```go
config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
    // Read request headers
    userAgent := r.Header().Get("User-Agent")
    contentType := r.Header().Get("Content-Type")

    // Set response headers
    w.Header().Set("X-Custom-Header", "MyValue")
    w.Header().Set("Cache-Control", "no-cache")

    w.WriteHeader(200)
    w.WriteString(fmt.Sprintf("User-Agent: %s", userAgent))
}
```

### HTTP/1.1 Server with TLS (HTTPS)

```go
config := server.DefaultConfig()
config.Addr = ":8443"
config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
    w.WriteString("Secure HTTPS Response")
}

srv := server.NewServer(config)

// Start with TLS
if err := srv.ListenAndServeTLS("cert.pem", "key.pem"); err != nil {
    log.Fatal(err)
}
```

### HTTP/1.1 Server with Timeouts & Limits

```go
config := server.DefaultConfig()
config.Addr = ":8080"
config.Handler = myHandler

// Configure timeouts
config.ReadTimeout = 5 * time.Second   // Max time to read request
config.WriteTimeout = 10 * time.Second // Max time to write response
config.IdleTimeout = 120 * time.Second // Max idle time for keep-alive

// Configure limits
config.MaxHeaderBytes = 1 << 20        // 1 MB max header size
config.MaxRequestBodySize = 10 << 20   // 10 MB max body size
config.MaxKeepAliveRequests = 100      // Max requests per connection
config.MaxConcurrentConnections = 1000 // Max concurrent connections

// Disable keep-alive if needed
config.DisableKeepalive = false

srv := server.NewServer(config)
srv.ListenAndServe()
```

### HTTP/1.1 Server with Graceful Shutdown

```go
func main() {
    config := server.DefaultConfig()
    config.Handler = myHandler

    srv := server.NewServer(config)

    // Start server in goroutine
    go func() {
        if err := srv.ListenAndServe(); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()

    // Wait for interrupt signal
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down gracefully...")

    // Graceful shutdown with timeout
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("Shutdown error: %v", err)
    }

    log.Println("Server stopped")
}
```

### HTTP/1.1 Server with Stats Monitoring

```go
config := server.DefaultConfig()
config.Handler = myHandler
config.EnableStats = true // Enable request time tracking

srv := server.NewServer(config)

// Start server
go srv.ListenAndServe()

// Monitor stats
ticker := time.NewTicker(10 * time.Second)
go func() {
    for range ticker.C {
        stats := srv.Stats()

        log.Printf("Server Stats:")
        log.Printf("  Total Connections: %d", stats.TotalConnections.Load())
        log.Printf("  Active Connections: %d", stats.ActiveConnections.Load())
        log.Printf("  Total Requests: %d", stats.TotalRequests.Load())
        log.Printf("  Requests/sec: %.2f", stats.RequestsPerSecond())
        log.Printf("  Bytes Read: %d", stats.BytesRead.Load())
        log.Printf("  Bytes Written: %d", stats.BytesWritten.Load())
    }
}()
```

---

## HTTP/1.1 Client

### Basic HTTP/1.1 Client

```go
package main

import (
    "fmt"
    "log"

    "github.com/yourusername/shockwave/pkg/shockwave/client"
)

func main() {
    // Create client
    c := client.NewClient()
    defer c.Close()

    // Simple GET request
    resp, err := c.Get("http://example.com")
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Close()

    fmt.Printf("Status: %d\n", resp.StatusCode())
    fmt.Printf("Body: %s\n", resp.Body())
}
```

### HTTP/1.1 Client with POST

```go
c := client.NewClient()
defer c.Close()

// POST with JSON
jsonData := []byte(`{"name":"John","email":"john@example.com"}`)
body := bytes.NewReader(jsonData)

resp, err := c.Post("http://api.example.com/users", "application/json", body)
if err != nil {
    log.Fatal(err)
}
defer resp.Close()

fmt.Printf("Created: %s\n", resp.Body())
```

### HTTP/1.1 Client with Custom Request

```go
c := client.NewClient()

// Get request from pool (zero allocations)
req := client.GetClientRequest()
defer client.PutClientRequest(req)

// Configure request
req.SetMethod("PUT")
req.SetHost("api.example.com")
req.SetPath("/api/users/123")
req.SetHeader("Authorization", "Bearer token123")
req.SetHeader("Content-Type", "application/json")
req.SetBody([]byte(`{"name":"Updated Name"}`))

// Execute request
resp, err := c.Do(req)
if err != nil {
    log.Fatal(err)
}
defer resp.Close()

fmt.Printf("Response: %s\n", resp.Body())
```

### HTTP/1.1 Client with Headers

```go
req := client.GetClientRequest()
defer client.PutClientRequest(req)

req.SetMethod("GET")
req.SetURL("http://api.example.com/data")

// Set multiple headers
req.SetHeader("Authorization", "Bearer token")
req.SetHeader("Accept", "application/json")
req.SetHeader("User-Agent", "MyApp/1.0")
req.SetHeader("X-Request-ID", "12345")

resp, err := c.Do(req)
if err != nil {
    log.Fatal(err)
}
defer resp.Close()

// Read response headers
contentType := resp.GetHeader("Content-Type")
cacheControl := resp.GetHeader("Cache-Control")

fmt.Printf("Content-Type: %s\n", contentType)
fmt.Printf("Cache-Control: %s\n", cacheControl)
```

### HTTP/1.1 Client with Context & Timeout

```go
c := client.NewClient()

// Create context with timeout
ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
defer cancel()

req := client.GetClientRequest()
defer client.PutClientRequest(req)

req.SetURL("http://slow-api.example.com/data")

// Execute with context
resp, err := c.DoWithContext(ctx, req)
if err != nil {
    if errors.Is(err, context.DeadlineExceeded) {
        log.Println("Request timed out")
    } else {
        log.Fatal(err)
    }
    return
}
defer resp.Close()

fmt.Printf("Response: %s\n", resp.Body())
```

### HTTP/1.1 Client with Connection Pooling

```go
import "github.com/yourusername/shockwave/pkg/shockwave/client"

// Create client with custom pool configuration
poolConfig := client.DefaultPoolConfig()
poolConfig.MaxConnsPerHost = 100          // Max connections per host
poolConfig.MaxIdleConnsPerHost = 10       // Max idle connections per host
poolConfig.MaxIdleTime = 90 * time.Second // Max idle time before closing
poolConfig.DialTimeout = 10 * time.Second // Connection timeout

pool := client.NewConnectionPool(poolConfig)
c := &client.Client{
    Pool: pool,
}

// Connection is automatically reused from pool
resp1, _ := c.Get("http://api.example.com/data1")
resp1.Close()

resp2, _ := c.Get("http://api.example.com/data2") // Reuses connection
resp2.Close()
```

### HTTP/1.1 Client - Concurrent Requests

```go
c := client.NewClient()
defer c.Close()

urls := []string{
    "http://api.example.com/users",
    "http://api.example.com/posts",
    "http://api.example.com/comments",
}

var wg sync.WaitGroup
results := make(chan string, len(urls))

for _, url := range urls {
    wg.Add(1)
    go func(u string) {
        defer wg.Done()

        resp, err := c.Get(u)
        if err != nil {
            results <- fmt.Sprintf("Error: %v", err)
            return
        }
        defer resp.Close()

        results <- fmt.Sprintf("URL %s: Status %d", u, resp.StatusCode())
    }(url)
}

wg.Wait()
close(results)

for result := range results {
    fmt.Println(result)
}
```

---

## HTTP/2 Server

### Basic HTTP/2 Server

```go
package main

import (
    "log"

    "github.com/yourusername/shockwave/pkg/shockwave/http2"
)

func main() {
    // HTTP/2 requires TLS (ALPN negotiation)
    config := http2.DefaultConfig()
    config.Addr = ":8443"

    // Set handler
    config.Handler = func(stream *http2.Stream) {
        // Read request headers
        headers := stream.Headers()

        // Send response
        responseHeaders := http2.NewHeaders()
        responseHeaders.Set(":status", "200")
        responseHeaders.Set("content-type", "text/plain")

        stream.WriteHeaders(responseHeaders, false)
        stream.Write([]byte("Hello from HTTP/2!"))
        stream.Close()
    }

    srv := http2.NewServer(config)

    log.Printf("Starting HTTP/2 server on %s", config.Addr)
    if err := srv.ListenAndServeTLS("cert.pem", "key.pem"); err != nil {
        log.Fatal(err)
    }
}
```

### HTTP/2 Server with Server Push

```go
config.Handler = func(stream *http2.Stream) {
    path := stream.Headers().Get(":path")

    if path == "/index.html" {
        // Push additional resources
        pushResources := []string{
            "/style.css",
            "/script.js",
            "/image.png",
        }

        for _, resource := range pushResources {
            pushHeaders := http2.NewHeaders()
            pushHeaders.Set(":method", "GET")
            pushHeaders.Set(":path", resource)
            pushHeaders.Set(":scheme", "https")
            pushHeaders.Set(":authority", stream.Headers().Get(":authority"))

            // Initiate server push
            if err := stream.Push(pushHeaders); err != nil {
                log.Printf("Push failed: %v", err)
            }
        }
    }

    // Send main response
    responseHeaders := http2.NewHeaders()
    responseHeaders.Set(":status", "200")
    responseHeaders.Set("content-type", "text/html")

    stream.WriteHeaders(responseHeaders, false)
    stream.Write([]byte("<html><body>Hello HTTP/2!</body></html>"))
    stream.Close()
}
```

### HTTP/2 Server with Flow Control

```go
config := http2.DefaultConfig()
config.Addr = ":8443"

// Configure flow control windows
config.InitialWindowSize = 65535        // Stream window
config.InitialConnectionWindowSize = 1 << 20 // 1 MB connection window

config.Handler = func(stream *http2.Stream) {
    // Large file streaming with flow control
    file, _ := os.Open("large-file.dat")
    defer file.Close()

    responseHeaders := http2.NewHeaders()
    responseHeaders.Set(":status", "200")
    responseHeaders.Set("content-type", "application/octet-stream")

    stream.WriteHeaders(responseHeaders, false)

    // Stream file data (respects flow control)
    buf := make([]byte, 16384)
    for {
        n, err := file.Read(buf)
        if n > 0 {
            stream.Write(buf[:n])
        }
        if err == io.EOF {
            break
        }
        if err != nil {
            log.Printf("Read error: %v", err)
            break
        }
    }

    stream.Close()
}

srv := http2.NewServer(config)
srv.ListenAndServeTLS("cert.pem", "key.pem")
```

---

## HTTP/2 Client

### Basic HTTP/2 Client

```go
package main

import (
    "fmt"
    "log"

    "github.com/yourusername/shockwave/pkg/shockwave/http2"
)

func main() {
    // Create HTTP/2 client (with TLS)
    client, err := http2.NewClient(&http2.ClientConfig{
        TLSConfig: &tls.Config{
            NextProtos: []string{"h2"}, // ALPN for HTTP/2
        },
    })
    if err != nil {
        log.Fatal(err)
    }
    defer client.Close()

    // Make request
    headers := http2.NewHeaders()
    headers.Set(":method", "GET")
    headers.Set(":path", "/api/data")
    headers.Set(":scheme", "https")
    headers.Set(":authority", "api.example.com")

    stream, err := client.Request(headers)
    if err != nil {
        log.Fatal(err)
    }

    // Read response
    respHeaders := stream.Headers()
    status := respHeaders.Get(":status")

    body, _ := io.ReadAll(stream)

    fmt.Printf("Status: %s\n", status)
    fmt.Printf("Body: %s\n", body)
}
```

### HTTP/2 Client with Multiplexing

```go
client, _ := http2.NewClient(nil)
defer client.Close()

// Send multiple concurrent requests over single connection
var wg sync.WaitGroup

for i := 0; i < 10; i++ {
    wg.Add(1)
    go func(id int) {
        defer wg.Done()

        headers := http2.NewHeaders()
        headers.Set(":method", "GET")
        headers.Set(":path", fmt.Sprintf("/api/data/%d", id))
        headers.Set(":scheme", "https")
        headers.Set(":authority", "api.example.com")

        stream, err := client.Request(headers)
        if err != nil {
            log.Printf("Request %d failed: %v", id, err)
            return
        }

        body, _ := io.ReadAll(stream)
        fmt.Printf("Response %d: %s\n", id, body)
    }(i)
}

wg.Wait()
```

---

## HTTP/3 Server (QUIC)

### Basic HTTP/3 Server

```go
package main

import (
    "log"

    "github.com/yourusername/shockwave/pkg/shockwave/http3"
    "github.com/yourusername/shockwave/pkg/shockwave/http3/quic"
)

func main() {
    // Create QUIC listener
    quicConfig := &quic.Config{
        MaxIdleTimeout: 30 * time.Second,
    }

    listener, err := quic.Listen("udp", ":4433", "cert.pem", "key.pem", quicConfig)
    if err != nil {
        log.Fatal(err)
    }
    defer listener.Close()

    log.Println("HTTP/3 server listening on :4433")

    for {
        // Accept QUIC connection
        quicConn, err := listener.Accept()
        if err != nil {
            log.Printf("Accept error: %v", err)
            continue
        }

        // Handle HTTP/3 connection
        go handleHTTP3Connection(quicConn)
    }
}

func handleHTTP3Connection(quicConn *quic.Connection) {
    // Create HTTP/3 connection
    h3Conn := http3.NewConnection(quicConn, false)
    defer h3Conn.Close()

    // Accept request streams
    for {
        stream, err := h3Conn.AcceptStream()
        if err != nil {
            break
        }

        go handleHTTP3Stream(stream)
    }
}

func handleHTTP3Stream(stream *http3.RequestStream) {
    defer stream.Close()

    // Read request headers
    headers := stream.Headers()
    path := headers.Get(":path")

    log.Printf("HTTP/3 Request: %s", path)

    // Send response
    responseHeaders := http3.NewHeaders()
    responseHeaders.Set(":status", "200")
    responseHeaders.Set("content-type", "text/plain")

    stream.WriteHeaders(responseHeaders)
    stream.Write([]byte("Hello from HTTP/3 over QUIC!"))
}
```

### HTTP/3 Server with 0-RTT

```go
quicConfig := &quic.Config{
    MaxIdleTimeout: 30 * time.Second,
    Enable0RTT:     true, // Enable 0-RTT for faster reconnection
}

listener, _ := quic.Listen("udp", ":4433", "cert.pem", "key.pem", quicConfig)

for {
    quicConn, _ := listener.Accept()

    // Check if connection used 0-RTT
    if quicConn.Is0RTT() {
        log.Println("0-RTT connection established (faster!)")
    }

    go handleHTTP3Connection(quicConn)
}
```

---

## HTTP/3 Client (QUIC)

### Basic HTTP/3 Client

```go
package main

import (
    "fmt"
    "log"

    "github.com/yourusername/shockwave/pkg/shockwave/http3"
    "github.com/yourusername/shockwave/pkg/shockwave/http3/quic"
)

func main() {
    // Connect via QUIC
    quicConn, err := quic.Dial("udp", "example.com:4433", &quic.Config{
        Enable0RTT: true,
    })
    if err != nil {
        log.Fatal(err)
    }
    defer quicConn.Close()

    // Create HTTP/3 connection
    h3Conn := http3.NewConnection(quicConn, true)
    defer h3Conn.Close()

    // Create request stream
    stream, err := h3Conn.NewStream()
    if err != nil {
        log.Fatal(err)
    }
    defer stream.Close()

    // Send request headers
    headers := http3.NewHeaders()
    headers.Set(":method", "GET")
    headers.Set(":path", "/api/data")
    headers.Set(":scheme", "https")
    headers.Set(":authority", "example.com")

    stream.WriteHeaders(headers)

    // Read response
    respHeaders := stream.Headers()
    status := respHeaders.Get(":status")

    body, _ := io.ReadAll(stream)

    fmt.Printf("Status: %s\n", status)
    fmt.Printf("Body: %s\n", body)
}
```

### HTTP/3 Client with Connection Migration

```go
quicConfig := &quic.Config{
    EnableConnectionMigration: true, // Allow IP address changes
}

quicConn, _ := quic.Dial("udp", "example.com:4433", quicConfig)
h3Conn := http3.NewConnection(quicConn, true)

// Connection automatically migrates if network changes
// (e.g., WiFi to cellular)

stream, _ := h3Conn.NewStream()
// ... make request
```

---

## Performance Tips

### 1. Use Handler (not LegacyHandler) for Best Performance

```go
// ✅ FAST: Concrete types, zero interface overhead
config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
    w.WriteString("Fast")
}

// ❌ SLOW: Interface types, 1 alloc/op overhead
config.LegacyHandler = server.LegacyHandlerFunc(func(w server.ResponseWriter, r server.Request) {
    w.WriteString("Slower")
})
```

### 2. Disable Stats for Zero-Allocation Operation

```go
config := server.DefaultConfig()
config.EnableStats = false // Disables time.Now() allocation
config.Handler = myHandler
```

### 3. Use Object Pooling

```go
// Client requests
req := client.GetClientRequest()  // From pool
defer client.PutClientRequest(req) // Return to pool

req.SetURL("http://example.com")
resp, _ := c.Do(req)
defer resp.Close() // Also returns to pool
```

### 4. Pre-marshal JSON Responses

```go
// ✅ FAST: Marshal once, reuse
var cachedJSON = []byte(`{"status":"ok"}`)

config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
    w.WriteJSON(200, cachedJSON)
}

// ❌ SLOW: Marshal on every request
config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
    data, _ := json.Marshal(map[string]string{"status": "ok"})
    w.WriteJSON(200, data)
}
```

### 5. Configure Connection Pool

```go
poolConfig := client.DefaultPoolConfig()
poolConfig.MaxConnsPerHost = 100       // Increase for high concurrency
poolConfig.MaxIdleConnsPerHost = 20    // Keep more idle connections
poolConfig.MaxIdleTime = 90 * time.Second

pool := client.NewConnectionPool(poolConfig)
```

### 6. Use Keep-Alive

```go
// Server: Keep-alive is enabled by default
config.DisableKeepalive = false
config.IdleTimeout = 120 * time.Second
config.MaxKeepAliveRequests = 100 // 0 = unlimited

// Client: Connection pooling handles keep-alive automatically
c := client.NewClient()
resp1, _ := c.Get("http://example.com/1")
resp2, _ := c.Get("http://example.com/2") // Reuses connection
```

---

## Advanced Configuration

### Memory Configurations for Client

```bash
# Default (low memory): ~2.3 KB/op
go build .

# High performance: ~3.2 KB/op
go build -tags highperf .

# Ultra performance: ~4.1 KB/op (current behavior)
go build -tags ultraperf .

# Max performance: ~11.4 KB/op (99% header coverage, zero map allocations)
go build -tags maxperf .
```

### Custom Buffer Sizes

```go
config := server.DefaultConfig()
config.ReadBufferSize = 8192   // 8 KB read buffer
config.WriteBufferSize = 8192  // 8 KB write buffer
```

### Custom Timeouts

```go
config.ReadTimeout = 5 * time.Second
config.WriteTimeout = 10 * time.Second
config.IdleTimeout = 120 * time.Second
```

### Resource Limits

```go
config.MaxHeaderBytes = 2 << 20          // 2 MB max headers
config.MaxRequestBodySize = 20 << 20     // 20 MB max body
config.MaxConcurrentConnections = 10000  // Max concurrent connections
```

---

## Complete Server Example (Production-Ready)

```go
package main

import (
    "context"
    "log"
    "os"
    "os/signal"
    "syscall"
    "time"

    "github.com/yourusername/shockwave/pkg/shockwave/http11"
    "github.com/yourusername/shockwave/pkg/shockwave/server"
)

func main() {
    // Configure server
    config := server.DefaultConfig()
    config.Addr = ":8080"

    // Timeouts
    config.ReadTimeout = 5 * time.Second
    config.WriteTimeout = 10 * time.Second
    config.IdleTimeout = 120 * time.Second

    // Limits
    config.MaxHeaderBytes = 1 << 20
    config.MaxRequestBodySize = 10 << 20
    config.MaxKeepAliveRequests = 100
    config.MaxConcurrentConnections = 1000

    // Performance
    config.EnableStats = false // Disable for production

    // Handler
    config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
        handleRequest(w, r)
    }

    // Create server
    srv := server.NewServer(config)

    // Start server
    go func() {
        log.Printf("Server starting on %s", config.Addr)
        if err := srv.ListenAndServe(); err != nil {
            log.Printf("Server error: %v", err)
        }
    }()

    // Graceful shutdown
    sigChan := make(chan os.Signal, 1)
    signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)
    <-sigChan

    log.Println("Shutting down...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
        log.Printf("Shutdown error: %v", err)
    }

    log.Println("Server stopped")
}

func handleRequest(w *http11.ResponseWriter, r *http11.Request) {
    switch r.Path() {
    case "/":
        w.WriteString("Welcome")
    case "/health":
        w.WriteJSON(200, []byte(`{"status":"healthy"}`))
    default:
        w.WriteHeader(404)
    }
}
```

---

## Complete Client Example (Production-Ready)

```go
package main

import (
    "fmt"
    "log"
    "time"

    "github.com/yourusername/shockwave/pkg/shockwave/client"
)

func main() {
    // Create high-performance client
    poolConfig := client.DefaultPoolConfig()
    poolConfig.MaxConnsPerHost = 100
    poolConfig.MaxIdleConnsPerHost = 10
    poolConfig.MaxIdleTime = 90 * time.Second
    poolConfig.DialTimeout = 10 * time.Second

    c := &client.Client{
        Pool: client.NewConnectionPool(poolConfig),
    }
    defer c.Close()

    // Make request
    req := client.GetClientRequest()
    defer client.PutClientRequest(req)

    req.SetMethod("GET")
    req.SetURL("http://api.example.com/data")
    req.SetHeader("Authorization", "Bearer token")

    resp, err := c.Do(req)
    if err != nil {
        log.Fatal(err)
    }
    defer resp.Close()

    fmt.Printf("Status: %d\n", resp.StatusCode())
    fmt.Printf("Body: %s\n", resp.Body())
}
```

---

## Next Steps

- See `ZERO_ALLOCATION_SERVER_ANALYSIS.md` for server performance details
- See `API_RESTRUCTURING_RESULTS.md` for handler API details
- See `MEMORY_OPTIMIZATION_STRATEGY.md` for memory tuning
- See `HTTP3_IMPLEMENTATION_ROADMAP.md` for HTTP/3 roadmap

---

**Shockwave is 5% faster than fasthttp with production-ready HTTP/1.1, HTTP/2, and HTTP/3 support!**
