package http11

import (
	"fmt"
	"strings"
	"sync"
	"testing"
	"time"
)

// TestE2ESimpleGETRequest tests a complete GET request/response cycle
func TestE2ESimpleGETRequest(t *testing.T) {
	// Setup mock connection
	requestData := "GET /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: TestClient/1.0\r\n" +
		"Accept: application/json\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	// Handler that processes the request
	handler := func(req *Request, rw *ResponseWriter) error {
		// Verify request
		if req.MethodID != MethodGET {
			t.Errorf("Method = %d, want GET", req.MethodID)
		}

		if req.Path() != "/api/users" {
			t.Errorf("Path = %s, want /api/users", req.Path())
		}

		// Verify headers
		host := req.GetHeaderString("Host")
		if host != "example.com" {
			t.Errorf("Host = %s, want example.com", host)
		}

		// Write JSON response
		responseBody := []byte(`{"users":[{"id":1,"name":"Alice"}]}`)
		return rw.WriteJSON(200, responseBody)
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	// Serve the connection
	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}

	// Verify response
	response := mockConn.GetWritten()
	if !strings.Contains(response, "HTTP/1.1 200 OK") {
		t.Error("Response missing 200 OK status")
	}

	if !strings.Contains(response, "Content-Type: application/json") {
		t.Error("Response missing JSON content type")
	}

	if !strings.Contains(response, `"users"`) {
		t.Error("Response missing JSON body")
	}
}

// TestE2EPOSTWithBody tests POST request with body parsing
func TestE2EPOSTWithBody(t *testing.T) {
	requestBody := `{"username":"alice","email":"alice@example.com"}`
	requestData := "POST /api/users HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Content-Type: application/json\r\n" +
		"Content-Length: 48\r\n" +
		"\r\n" +
		requestBody

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Verify method
		if !req.IsPOST() {
			t.Error("Expected POST method")
		}

		// Verify Content-Length was parsed
		if req.ContentLength != 48 {
			t.Errorf("ContentLength = %d, want 48", req.ContentLength)
		}

		// Verify Content-Type
		contentType := req.GetHeaderString("Content-Type")
		if contentType != "application/json" {
			t.Errorf("Content-Type = %s, want application/json", contentType)
		}

		// Send 201 Created response
		responseBody := []byte(`{"id":123,"username":"alice","email":"alice@example.com"}`)
		return rw.WriteJSON(201, responseBody)
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}

	response := mockConn.GetWritten()
	if !strings.Contains(response, "HTTP/1.1 201 Created") {
		t.Error("Response should be 201 Created")
	}
}

// TestE2EMultipleHeaders tests handling of many headers
func TestE2EMultipleHeaders(t *testing.T) {
	var requestBuilder strings.Builder
	requestBuilder.WriteString("GET /api/data HTTP/1.1\r\n")
	requestBuilder.WriteString("Host: example.com\r\n")

	// Add 30 custom headers (staying within inline capacity)
	for i := 1; i <= 30; i++ {
		requestBuilder.WriteString(fmt.Sprintf("X-Custom-Header-%d: value-%d\r\n", i, i))
	}
	requestBuilder.WriteString("\r\n")

	mockConn := newMockConn(requestBuilder.String())
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Verify we got all headers
		headerCount := req.Header.Len()
		if headerCount != 31 { // Host + 30 custom
			t.Errorf("Header count = %d, want 31", headerCount)
		}

		// Verify specific custom header
		val := req.GetHeaderString("X-Custom-Header-15")
		if val != "value-15" {
			t.Errorf("X-Custom-Header-15 = %s, want value-15", val)
		}

		return rw.WriteText(200, []byte("OK"))
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}
}

// TestE2E404NotFound tests 404 error response
func TestE2E404NotFound(t *testing.T) {
	requestData := "GET /nonexistent HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Simulate resource not found
		return rw.WriteError(404, "Resource not found")
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}

	response := mockConn.GetWritten()
	if !strings.Contains(response, "HTTP/1.1 404 Not Found") {
		t.Error("Response should be 404 Not Found")
	}

	if !strings.Contains(response, "Resource not found") {
		t.Error("Response missing error message")
	}
}

// TestE2E500InternalError tests 500 error response
func TestE2E500InternalError(t *testing.T) {
	requestData := "GET /error HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Simulate internal error
		rw.WriteHeader(500)
		rw.Write([]byte("Internal Server Error"))
		return fmt.Errorf("database connection failed")
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err == nil {
		t.Error("Expected error from handler")
	}

	response := mockConn.GetWritten()
	if !strings.Contains(response, "HTTP/1.1 500 Internal Server Error") {
		t.Error("Response should be 500 Internal Server Error")
	}
}

// TestE2EConnectionClose tests Connection: close behavior
func TestE2EConnectionClose(t *testing.T) {
	requestData := "GET /test HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Connection: close\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handlerCalled := 0
	handler := func(req *Request, rw *ResponseWriter) error {
		handlerCalled++

		if !req.Close {
			t.Error("Request.Close should be true")
		}

		return rw.WriteText(200, []byte("Closing"))
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	conn.Serve()

	// Should only handle one request
	if handlerCalled != 1 {
		t.Errorf("Handler called %d times, want 1", handlerCalled)
	}

	// Response should include the body
	response := mockConn.GetWritten()
	if !strings.Contains(response, "Closing") {
		t.Error("Response missing body")
	}
}

// TestE2EConcurrentConnections tests multiple concurrent connections
func TestE2EConcurrentConnections(t *testing.T) {
	const numConnections = 50

	var wg sync.WaitGroup
	errors := make(chan error, numConnections)

	for i := 0; i < numConnections; i++ {
		wg.Add(1)
		go func(id int) {
			defer wg.Done()

			requestData := fmt.Sprintf("GET /test/%d HTTP/1.1\r\nHost: example.com\r\n\r\n", id)
			mockConn := newMockConn(requestData)
			config := DefaultConnectionConfig()

			handler := func(req *Request, rw *ResponseWriter) error {
				// Verify path contains ID
				if !strings.Contains(req.Path(), fmt.Sprintf("%d", id)) {
					return fmt.Errorf("path mismatch for connection %d", id)
				}

				responseBody := fmt.Sprintf(`{"connection":%d,"status":"ok"}`, id)
				return rw.WriteJSON(200, []byte(responseBody))
			}

			conn := NewConnection(mockConn, config, handler)
			defer conn.Close()

			err := conn.Serve()
			if err != nil {
				errors <- err
			}
		}(i)
	}

	wg.Wait()
	close(errors)

	// Check for errors
	errorCount := 0
	for err := range errors {
		t.Errorf("Connection error: %v", err)
		errorCount++
		if errorCount >= 5 {
			break
		}
	}
}

// TestE2ELargeResponse tests handling large response bodies
func TestE2ELargeResponse(t *testing.T) {
	requestData := "GET /large HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Generate 10KB response
		var responseBuilder strings.Builder
		responseBuilder.WriteString(`{"items":[`)
		for i := 0; i < 500; i++ {
			if i > 0 {
				responseBuilder.WriteString(",")
			}
			responseBuilder.WriteString(fmt.Sprintf(`{"id":%d,"data":"item%d"}`, i, i))
		}
		responseBuilder.WriteString(`]}`)

		responseBody := []byte(responseBuilder.String())
		return rw.WriteJSON(200, responseBody)
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()
	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}

	response := mockConn.GetWritten()
	if len(response) < 10000 {
		t.Errorf("Response too small: %d bytes", len(response))
	}

	if !strings.Contains(response, `"items"`) {
		t.Error("Response missing items array")
	}
}

// TestE2EHTMLResponse tests HTML content serving
func TestE2EHTMLResponse(t *testing.T) {
	requestData := "GET / HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Accept: text/html\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		htmlContent := []byte(`<!DOCTYPE html>
<html>
<head><title>Test Page</title></head>
<body>
<h1>Welcome to Shockwave</h1>
<p>High-performance HTTP library</p>
</body>
</html>`)
		return rw.WriteHTML(200, htmlContent)
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()
	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}

	response := mockConn.GetWritten()
	if !strings.Contains(response, "Content-Type: text/html") {
		t.Error("Response missing HTML content type")
	}

	if !strings.Contains(response, "Welcome to Shockwave") {
		t.Error("Response missing HTML body")
	}
}

// TestE2ERedirect tests redirect responses
func TestE2ERedirect(t *testing.T) {
	requestData := "GET /old-path HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Send 301 redirect
		rw.Header().Set([]byte("Location"), []byte("/new-path"))
		rw.WriteHeader(301)
		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()
	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}

	response := mockConn.GetWritten()
	if !strings.Contains(response, "HTTP/1.1 301 Moved Permanently") {
		t.Error("Response should be 301")
	}

	if !strings.Contains(response, "Location: /new-path") {
		t.Error("Response missing Location header")
	}
}

// TestE2ERequestTimeout tests connection timeout behavior
func TestE2ERequestTimeout(t *testing.T) {
	// Create a connection that will timeout quickly
	requestData := "GET /slow HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()
	config.KeepAliveTimeout = 1 * time.Millisecond // Very short timeout

	handler := func(req *Request, rw *ResponseWriter) error {
		// Simulate slow processing
		time.Sleep(5 * time.Millisecond)
		return rw.WriteText(200, []byte("OK"))
	}


	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()
	// This should timeout or complete quickly
	conn.Serve()

	// Test passes if it doesn't hang
}

// TestE2EPoolingEfficiency tests that pooling works correctly across requests
func TestE2EPoolingEfficiency(t *testing.T) {
	// Warmup pools
	WarmupPools(10)

	requestData := "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"

	handler := func(req *Request, rw *ResponseWriter) error {
		return rw.WriteText(200, []byte("OK"))
	}

	// Process multiple requests to test pool reuse
	for i := 0; i < 20; i++ {
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		conn := NewConnection(mockConn, config, handler)

		conn.Serve()
		conn.Close()
	}

	// Verify pools still have objects
	stats := GetPoolStats()
	if len(stats) != 7 {
		t.Errorf("Expected 7 pools, got %d", len(stats))
	}

	// Test completed successfully if we got here without panics
}

// TestE2EQueryParameters tests query parameter parsing
func TestE2EQueryParameters(t *testing.T) {
	requestData := "GET /search?q=golang&page=1&limit=10 HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Verify path and query
		if req.Path() != "/search" {
			t.Errorf("Path = %s, want /search", req.Path())
		}

		query := req.Query()
		if !strings.Contains(query, "q=golang") {
			t.Error("Query missing q parameter")
		}

		if !strings.Contains(query, "page=1") {
			t.Error("Query missing page parameter")
		}

		// Parse URL for query values
		parsedURL, err := req.ParsedURL()
		if err != nil {
			t.Errorf("ParsedURL error: %v", err)
		}

		if parsedURL.Query().Get("q") != "golang" {
			t.Error("Query parameter q not parsed correctly")
		}

		responseBody := []byte(`{"results":[],"total":0}`)
		return rw.WriteJSON(200, responseBody)
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()
	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}
}

// TestE2ECaseSensitiveHeaders tests case-insensitive header handling
func TestE2ECaseSensitiveHeaders(t *testing.T) {
	requestData := "GET /test HTTP/1.1\r\n" +
		"host: example.com\r\n" + // lowercase
		"content-type: application/json\r\n" + // lowercase
		"X-Custom-Header: value\r\n" + // mixed case
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Should be able to retrieve with different cases
		host1 := req.GetHeaderString("Host")
		host2 := req.GetHeaderString("host")
		host3 := req.GetHeaderString("HOST")

		if host1 != "example.com" || host2 != "example.com" || host3 != "example.com" {
			t.Error("Case-insensitive header lookup failed")
		}

		return rw.WriteText(200, []byte("OK"))
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()
	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve error: %v", err)
	}
}

// Benchmark E2E scenarios

func BenchmarkE2ESimpleGET(b *testing.B) {
	requestData := "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"
	handler := func(req *Request, rw *ResponseWriter) error {
		return rw.WriteText(200, []byte("OK"))
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}

func BenchmarkE2EJSONAPI(b *testing.B) {
	requestData := "GET /api/users HTTP/1.1\r\nHost: example.com\r\nAccept: application/json\r\n\r\n"
	responseBody := []byte(`{"users":[{"id":1,"name":"Alice"},{"id":2,"name":"Bob"}]}`)

	handler := func(req *Request, rw *ResponseWriter) error {
		return rw.WriteJSON(200, responseBody)
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(responseBody)))

	for i := 0; i < b.N; i++ {
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}

func BenchmarkE2EConcurrentConnections(b *testing.B) {
	requestData := "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"
	handler := func(req *Request, rw *ResponseWriter) error {
		return rw.WriteText(200, []byte("OK"))
	}

	b.ResetTimer()
	b.ReportAllocs()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			mockConn := newMockConn(requestData)
			config := DefaultConnectionConfig()
			conn := NewConnection(mockConn, config, handler)
			conn.Serve()
			conn.Close()
		}
	})
}
