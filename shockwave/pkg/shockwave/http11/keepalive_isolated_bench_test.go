package http11

import (
	"bytes"
	"strings"
	"testing"
)

// Isolated Keep-Alive Benchmarks
//
// These benchmarks isolate the keep-alive request/response cycle
// from connection setup and I/O overhead to measure the true
// allocation cost of keep-alive reuse.

// BenchmarkKeepAliveIsolated_ParseAndWrite measures just parse + write
// without connection overhead.
func BenchmarkKeepAliveIsolated_ParseAndWrite(b *testing.B) {
	requestData := []byte("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")

	var buf bytes.Buffer
	parser := GetParser()
	defer PutParser(parser)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Parse
		r := bytes.NewReader(requestData)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		// Write response
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		rw.Flush()

		// Return to pools
		PutResponseWriter(rw)
		PutRequest(req)
	}
}

// BenchmarkKeepAliveIsolated_HandlerCycle measures the handler cycle
// with pooled objects (simulates keep-alive reuse).
func BenchmarkKeepAliveIsolated_HandlerCycle(b *testing.B) {
	requestData := []byte("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")

	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Get parser from pool (simulates keep-alive connection)
		parser := GetParser()

		// Parse request
		r := bytes.NewReader(requestData)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		// Get response writer from pool
		buf.Reset()
		rw := GetResponseWriter(&buf)

		// Handle (simple handler)
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		rw.Flush()

		// Return to pools (simulates keep-alive reuse)
		PutResponseWriter(rw)
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkKeepAliveIsolated_PoolOperations measures just pool get/put operations
func BenchmarkKeepAliveIsolated_PoolOperations(b *testing.B) {
	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parser := GetParser()
		req := GetRequest()
		rw := GetResponseWriter(&buf)

		PutResponseWriter(rw)
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkKeepAliveIsolated_JSON measures keep-alive with JSON response
func BenchmarkKeepAliveIsolated_JSON(b *testing.B) {
	requestData := []byte("GET /api/data HTTP/1.1\r\nHost: example.com\r\n\r\n")
	jsonData := []byte(`{"status":"ok"}`)

	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		parser := GetParser()

		r := bytes.NewReader(requestData)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteJSON(200, jsonData)

		PutResponseWriter(rw)
		PutRequest(req)
		PutParser(parser)
	}
}

// BenchmarkKeepAliveIsolated_ReuseParser measures reusing same parser
// (true keep-alive scenario where parser persists across requests)
func BenchmarkKeepAliveIsolated_ReuseParser(b *testing.B) {
	requestData := []byte("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")

	var buf bytes.Buffer
	parser := GetParser()
	defer PutParser(parser)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Parse (parser is reused, not from pool each time)
		r := bytes.NewReader(requestData)
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		// Write
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		rw.Flush()

		PutResponseWriter(rw)
		PutRequest(req)
	}
}

// BenchmarkKeepAliveIsolated_MinimalPath measures the absolute minimum path
func BenchmarkKeepAliveIsolated_MinimalPath(b *testing.B) {
	// Pre-parse a request
	requestData := []byte("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n")
	parser := GetParser()
	defer PutParser(parser)

	var buf bytes.Buffer

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Parse
		r := strings.NewReader(string(requestData))
		req, err := parser.Parse(r)
		if err != nil {
			b.Fatal(err)
		}

		// Minimal response
		buf.Reset()
		rw := GetResponseWriter(&buf)
		rw.WriteHeader(200)
		rw.Flush()

		PutResponseWriter(rw)
		PutRequest(req)
	}
}
