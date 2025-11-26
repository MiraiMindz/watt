package http11

import (
	"fmt"
	"strings"
	"testing"
)

// Keep-Alive Benchmarks
//
// These benchmarks specifically test keep-alive connection reuse
// to verify the critical requirement: Keep-alive connection reuse MUST be 0 allocs/op
//
// The benchmarks measure per-request allocations on an already-established connection,
// not the connection setup cost.

// BenchmarkKeepAliveReuse measures the allocation cost of keep-alive request reuse
// on an established connection. This should be 0 allocs/op.
func BenchmarkKeepAliveReuse(b *testing.B) {
	// Create a connection with many requests pre-loaded
	requestData := strings.Repeat("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n", b.N+100)
	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()
	config.MaxRequests = b.N + 100 // Allow more than enough requests

	requestCount := 0
	handler := func(req *Request, rw *ResponseWriter) error {
		requestCount++
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))

		// Stop after b.N requests
		if requestCount > b.N {
			return fmt.Errorf("stop")
		}

		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	// This will process b.N requests on the same connection
	conn.Serve()
}

// BenchmarkKeepAliveSingleRequest measures the cost of handling a single request
// on an already-established connection (isolates just the request/response cycle)
func BenchmarkKeepAliveSingleRequest(b *testing.B) {
	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Single request per connection
		requestData := "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = 1

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}

// BenchmarkKeepAlive10Sequential measures 10 sequential requests on same connection
func BenchmarkKeepAlive10Sequential(b *testing.B) {
	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 10 requests per connection
		requestData := strings.Repeat("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n", 10)
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = 10

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}

	b.SetBytes(int64(10 * len("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")))
}

// BenchmarkKeepAlive100Sequential measures 100 sequential requests on same connection
func BenchmarkKeepAlive100Sequential(b *testing.B) {
	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 100 requests per connection
		requestData := strings.Repeat("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n", 100)
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = 100

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}

	b.SetBytes(int64(100 * len("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")))
}

// BenchmarkKeepAliveWithJSON measures keep-alive with JSON responses
func BenchmarkKeepAliveWithJSON(b *testing.B) {
	jsonData := []byte(`{"status":"ok","message":"success"}`)

	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteJSON(200, jsonData)
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 10 requests per connection
		requestData := strings.Repeat("GET /api/status HTTP/1.1\r\nHost: example.com\r\n\r\n", 10)
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = 10

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}

// BenchmarkKeepAliveWithHeaders measures keep-alive with custom headers
func BenchmarkKeepAliveWithHeaders(b *testing.B) {
	handler := func(req *Request, rw *ResponseWriter) error {
		rw.Header().Set([]byte("Cache-Control"), []byte("max-age=3600"))
		rw.Header().Set([]byte("X-Request-ID"), []byte("req-123"))
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// 10 requests per connection
		requestData := strings.Repeat("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n", 10)
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = 10

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}

// BenchmarkKeepAlivePipelined measures HTTP pipelining (multiple requests sent at once)
func BenchmarkKeepAlivePipelined(b *testing.B) {
	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// All 10 requests sent at once (pipelined)
		requestData := strings.Repeat("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n", 10)
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = 10

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}

// BenchmarkKeepAliveConnectionReuse specifically measures the cost per request
// when reusing an established connection (amortized cost)
func BenchmarkKeepAliveConnectionReuse_Amortized(b *testing.B) {
	const requestsPerConn = 100

	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	totalRequests := b.N
	for totalRequests > 0 {
		reqCount := requestsPerConn
		if totalRequests < requestsPerConn {
			reqCount = totalRequests
		}

		requestData := strings.Repeat("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n", reqCount)
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = reqCount

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()

		totalRequests -= reqCount
	}

	// Report per-request metrics
	b.ReportMetric(float64(b.N)/float64(b.Elapsed().Seconds()), "req/s")
}
