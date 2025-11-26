package http11

import (
	"bytes"
	"strconv"
	"strings"
	"sync"
	"testing"
)

// TestIntegrationFullRequestResponseCycle tests the complete HTTP/1.1 flow:
// Parse request -> Process -> Write response -> Pool cleanup
func TestIntegrationFullRequestResponseCycle(t *testing.T) {
	// Sample HTTP request
	requestData := "GET /api/users?page=1&limit=10 HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: TestClient/1.0\r\n" +
		"Accept: application/json\r\n" +
		"Authorization: Bearer token123\r\n" +
		"\r\n"

	// Get pooled parser
	parser := GetParser()
	defer PutParser(parser)

	// Parse the request
	req, err := parser.Parse(strings.NewReader(requestData))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify parsed request
	if req.MethodID != MethodGET {
		t.Errorf("Method = %d, want %d", req.MethodID, MethodGET)
	}

	if string(req.PathBytes()) != "/api/users" {
		t.Errorf("Path = %s, want /api/users", req.PathBytes())
	}

	if string(req.QueryBytes()) != "page=1&limit=10" {
		t.Errorf("Query = %s, want page=1&limit=10", req.QueryBytes())
	}

	// Verify headers
	host := req.Header.Get([]byte("Host"))
	if string(host) != "example.com" {
		t.Errorf("Host header = %s, want example.com", host)
	}

	userAgent := req.Header.Get([]byte("User-Agent"))
	if string(userAgent) != "TestClient/1.0" {
		t.Errorf("User-Agent = %s, want TestClient/1.0", userAgent)
	}

	// Get pooled response writer
	var buf bytes.Buffer
	rw := GetResponseWriter(&buf)
	defer PutResponseWriter(rw)

	// Build JSON response
	responseBody := []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}],"total":2}`)

	// Write response
	rw.Header().Set(headerContentType, contentTypeJSONUTF8)
	rw.Header().Set(headerServer, []byte("Shockwave/1.0"))
	rw.Header().Set(headerConnection, headerKeepAlive)

	err = rw.WriteJSON(200, responseBody)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Verify response output
	output := buf.String()

	// Check status line
	if !strings.Contains(output, "HTTP/1.1 200 OK") {
		t.Error("Response missing status line")
	}

	// Check headers
	if !strings.Contains(output, "Content-Type: application/json; charset=utf-8") {
		t.Error("Response missing Content-Type header")
	}

	if !strings.Contains(output, "Server: Shockwave/1.0") {
		t.Error("Response missing Server header")
	}

	if !strings.Contains(output, "Connection: keep-alive") {
		t.Error("Response missing Connection header")
	}

	// Check body
	if !strings.Contains(output, `{"users":`) {
		t.Error("Response missing JSON body")
	}

	// Verify bytes written
	if rw.BytesWritten() != int64(len(responseBody)) {
		t.Errorf("BytesWritten = %d, want %d", rw.BytesWritten(), len(responseBody))
	}
}

// TestIntegrationPOSTRequestWithBody tests POST request with body
func TestIntegrationPOSTRequestWithBody(t *testing.T) {
	requestBody := `{"username":"alice","email":"alice@example.com"}`
	requestData := "POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: " + strconv.Itoa(len(requestBody)) + "\r\n" +
		"\r\n" +
		requestBody

	parser := GetParser()
	defer PutParser(parser)

	req, err := parser.Parse(strings.NewReader(requestData))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify method
	if !req.IsPOST() {
		t.Error("Request should be POST")
	}

	// Verify content length header was parsed
	contentLength := req.Header.Get([]byte("Content-Length"))
	if string(contentLength) != strconv.Itoa(len(requestBody)) {
		t.Errorf("Content-Length header = %s, want %d", contentLength, len(requestBody))
	}

	// Verify content length field
	if req.ContentLength != int64(len(requestBody)) {
		t.Errorf("ContentLength = %d, want %d", req.ContentLength, len(requestBody))
	}

	// Note: Body reading would require buffering support in the parser
	// to preserve data read beyond the header boundary. This is a known
	// limitation of the current streaming parser implementation.

	// Send response
	var buf bytes.Buffer
	rw := GetResponseWriter(&buf)
	defer PutResponseWriter(rw)

	responseBody := []byte(`{"id":123,"username":"alice","email":"alice@example.com"}`)
	err = rw.WriteJSON(201, responseBody)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "HTTP/1.1 201 Created") {
		t.Error("Response should have 201 Created status")
	}

	if !strings.Contains(output, `"id":123`) {
		t.Error("Response missing created resource")
	}
}

// TestIntegrationErrorResponse tests error response handling
func TestIntegrationErrorResponse(t *testing.T) {
	requestData := "GET /api/nonexistent HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	parser := GetParser()
	defer PutParser(parser)

	req, err := parser.Parse(strings.NewReader(requestData))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify request
	if string(req.PathBytes()) != "/api/nonexistent" {
		t.Errorf("Path = %s, want /api/nonexistent", req.PathBytes())
	}

	// Send 404 error
	var buf bytes.Buffer
	rw := GetResponseWriter(&buf)
	defer PutResponseWriter(rw)

	err = rw.WriteError(404, "Resource not found")
	if err != nil {
		t.Fatalf("WriteError failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "HTTP/1.1 404 Not Found") {
		t.Error("Response should have 404 status")
	}

	if !strings.Contains(output, "Resource not found") {
		t.Error("Response missing error message")
	}
}

// TestIntegrationMultipleHeadersAndLargeResponse tests handling many headers
func TestIntegrationMultipleHeadersAndLargeResponse(t *testing.T) {
	// Build request with many headers
	var requestBuilder strings.Builder
	requestBuilder.WriteString("GET /api/data HTTP/1.1\r\n")
	requestBuilder.WriteString("Host: example.com\r\n")

	// Add 20 custom headers
	for i := 1; i <= 20; i++ {
		requestBuilder.WriteString("X-Custom-Header-")
		requestBuilder.WriteString(string(rune('0' + i%10)))
		requestBuilder.WriteString(": value")
		requestBuilder.WriteString(string(rune('0' + i%10)))
		requestBuilder.WriteString("\r\n")
	}
	requestBuilder.WriteString("\r\n")

	parser := GetParser()
	defer PutParser(parser)

	req, err := parser.Parse(strings.NewReader(requestBuilder.String()))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify header count (21 total: Host + 20 custom)
	if req.Header.Len() != 21 {
		t.Errorf("Header count = %d, want 21", req.Header.Len())
	}

	// Build large response (1KB JSON)
	var responseBodyBuilder strings.Builder
	responseBodyBuilder.WriteString(`{"items":[`)
	for i := 0; i < 50; i++ {
		if i > 0 {
			responseBodyBuilder.WriteString(",")
		}
		responseBodyBuilder.WriteString(`{"id":`)
		responseBodyBuilder.WriteString(string(rune('0' + i%10)))
		responseBodyBuilder.WriteString(`,"value":"data"}`)
	}
	responseBodyBuilder.WriteString(`]}`)

	responseBody := []byte(responseBodyBuilder.String())

	var buf bytes.Buffer
	rw := GetResponseWriter(&buf)
	defer PutResponseWriter(rw)

	// Add response headers
	rw.Header().Set([]byte("Cache-Control"), []byte("max-age=3600"))
	rw.Header().Set([]byte("X-Request-ID"), []byte("req-12345"))
	rw.Header().Set([]byte("X-Response-Time"), []byte("42ms"))

	err = rw.WriteJSON(200, responseBody)
	if err != nil {
		t.Fatalf("WriteJSON failed: %v", err)
	}

	// Verify response
	output := buf.String()
	if !strings.Contains(output, "Cache-Control: max-age=3600") {
		t.Error("Response missing Cache-Control header")
	}

	if len(output) < len(responseBody)+100 { // Headers + body
		t.Error("Response seems too short")
	}
}

// TestIntegrationConcurrentRequestProcessing tests concurrent request handling
func TestIntegrationConcurrentRequestProcessing(t *testing.T) {
	const goroutines = 50
	const iterations = 100

	var wg sync.WaitGroup
	wg.Add(goroutines)

	errors := make(chan error, goroutines*iterations)

	for g := 0; g < goroutines; g++ {
		go func(gid int) {
			defer wg.Done()

			for i := 0; i < iterations; i++ {
				// Build request
				requestData := "GET /api/test?id=" + string(rune('0'+gid%10)) + " HTTP/1.1\r\n" +
					"Host: example.com\r\n" +
					"X-Goroutine-ID: " + string(rune('0'+gid%10)) + "\r\n" +
					"\r\n"

				// Get pooled parser
				parser := GetParser()

				// Parse request
				req, err := parser.Parse(strings.NewReader(requestData))
				if err != nil {
					errors <- err
					PutParser(parser)
					continue
				}

				// Verify request
				if req.MethodID != MethodGET {
					errors <- ErrInvalidMethod
				}

				// Get pooled response writer
				var buf bytes.Buffer
				rw := GetResponseWriter(&buf)

				// Write response
				responseBody := []byte(`{"status":"ok"}`)
				err = rw.WriteJSON(200, responseBody)
				if err != nil {
					errors <- err
				}

				// Return to pools
				PutResponseWriter(rw)
				PutParser(parser)
			}
		}(g)
	}

	wg.Wait()
	close(errors)

	// Check for any errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Concurrent test error: %v", err)
		errorCount++
		if errorCount >= 10 { // Limit error output
			break
		}
	}

	if errorCount > 0 {
		t.Errorf("Total errors in concurrent test: %d", errorCount)
	}
}

// TestIntegrationRequestClone tests request cloning for persistence
func TestIntegrationRequestClone(t *testing.T) {
	requestData := "GET /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Authorization: Bearer token\r\n" +
		"\r\n"

	parser := GetParser()

	req, err := parser.Parse(strings.NewReader(requestData))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Clone the request for persistence
	clonedReq := req.Clone()

	// Return parser to pool (this invalidates original req buffer)
	PutParser(parser)

	// Verify cloned request is still valid
	if clonedReq.MethodID != MethodGET {
		t.Error("Cloned request lost method")
	}

	if clonedReq.Path() != "/api/users" {
		t.Errorf("Cloned request path = %s, want /api/users", clonedReq.Path())
	}

	authHeader := clonedReq.Header.Get([]byte("Authorization"))
	if string(authHeader) != "Bearer token" {
		t.Error("Cloned request lost headers")
	}
}

// TestIntegrationHTMLResponse tests HTML response writing
func TestIntegrationHTMLResponse(t *testing.T) {
	requestData := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Accept: text/html\r\n" +
		"\r\n"

	parser := GetParser()
	defer PutParser(parser)

	req, err := parser.Parse(strings.NewReader(requestData))
	if err != nil {
		t.Fatalf("Parse failed: %v", err)
	}

	// Verify Accept header
	accept := req.Header.Get([]byte("Accept"))
	if string(accept) != "text/html" {
		t.Errorf("Accept = %s, want text/html", accept)
	}

	// Send HTML response
	var buf bytes.Buffer
	rw := GetResponseWriter(&buf)
	defer PutResponseWriter(rw)

	htmlBody := []byte(`<!DOCTYPE html><html><head><title>Test</title></head><body><h1>Hello, World!</h1></body></html>`)
	err = rw.WriteHTML(200, htmlBody)
	if err != nil {
		t.Fatalf("WriteHTML failed: %v", err)
	}

	output := buf.String()
	if !strings.Contains(output, "Content-Type: text/html") {
		t.Error("Response missing HTML Content-Type")
	}

	if !strings.Contains(output, "<h1>Hello, World!</h1>") {
		t.Error("Response missing HTML body")
	}
}

// TestIntegrationPoolWarmupAndReuse tests pool warmup and object reuse
func TestIntegrationPoolWarmupAndReuse(t *testing.T) {
	// Warmup pools
	WarmupPools(10)

	// Get pool stats
	stats := GetPoolStats()
	if len(stats) != 7 {
		t.Errorf("Expected 7 pools, got %d", len(stats))
	}

	// Verify all pool types are present
	poolNames := make(map[string]bool)
	for _, stat := range stats {
		poolNames[stat.Name] = true
	}

	expectedPools := []string{"Request", "ResponseWriter", "Parser", "Buffer", "LargeBuffer", "BufioReader", "BufioWriter"}
	for _, name := range expectedPools {
		if !poolNames[name] {
			t.Errorf("Missing pool: %s", name)
		}
	}

	// Test rapid Get/Put cycles (simulating high-throughput server)
	for i := 0; i < 100; i++ {
		req := GetRequest()
		req.MethodID = MethodGET
		req.pathBytes = []byte("/test")
		PutRequest(req)

		rw := GetResponseWriter(nil)
		PutResponseWriter(rw)

		parser := GetParser()
		PutParser(parser)
	}

	// Verify we can still get objects from pools
	req := GetRequest()
	if req == nil {
		t.Error("Failed to get request from pool after warmup")
	}
	PutRequest(req)
}

// Benchmarks for integration tests

func BenchmarkIntegrationFullCycle(b *testing.B) {
	requestData := "GET /api/users?page=1 HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: Benchmark\r\n" +
		"Accept: application/json\r\n" +
		"\r\n"

	responseBody := []byte(`{"users":[{"id":1,"name":"Alice"}]}`)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Parse request
		parser := GetParser()
		req, err := parser.Parse(strings.NewReader(requestData))
		if err != nil {
			b.Fatalf("Parse failed: %v", err)
		}

		// Write response
		var buf bytes.Buffer
		rw := GetResponseWriter(&buf)
		rw.WriteJSON(200, responseBody)

		// Cleanup
		PutResponseWriter(rw)
		PutParser(parser)

		_ = req
	}
}

func BenchmarkIntegrationConcurrentFullCycle(b *testing.B) {
	requestData := "GET /api/test HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	responseBody := []byte(`{"status":"ok"}`)

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			parser := GetParser()
			req, _ := parser.Parse(strings.NewReader(requestData))

			var buf bytes.Buffer
			rw := GetResponseWriter(&buf)
			rw.WriteJSON(200, responseBody)

			PutResponseWriter(rw)
			PutParser(parser)

			_ = req
		}
	})
}
