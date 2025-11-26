package core

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
	"sync"
	"testing"
	"time"
)

// Test utilities for integration testing with Shockwave server

// testServer wraps an App with test utilities.
type testServer struct {
	app  *App
	addr string
	url  string
}

// createTestServer creates and starts a test server on a random available port.
//
// The server runs in a background goroutine. Use Shutdown() to stop it.
//
// Example:
//
//	ts := createTestServer(t)
//	defer ts.Shutdown()
//	resp := ts.Get("/health")
func createTestServer(t *testing.T) *testServer {
	t.Helper()

	app := New()

	// Find available port
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("failed to find available port: %v", err)
	}
	addr := listener.Addr().String()
	listener.Close()

	ts := &testServer{
		app:  app,
		addr: addr,
		url:  "http://" + addr,
	}

	// Channel to signal server is fully started
	started := make(chan struct{})

	// Start server in background
	go func() {
		// Signal that goroutine has started and server is about to Listen
		close(started)

		if err := app.Listen(addr); err != nil {
			// Server stopped (expected during shutdown)
			t.Logf("server stopped: %v", err)
		}
	}()

	// Wait for goroutine to start
	<-started

	// Small delay to ensure server initialization completes
	time.Sleep(10 * time.Millisecond)

	// Wait for server to be ready
	if !waitForServer(addr, 5*time.Second) {
		t.Fatal("server failed to start")
	}

	return ts
}

// Shutdown gracefully shuts down the test server.
func (ts *testServer) Shutdown() error {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()
	return ts.app.Shutdown(ctx)
}

// Get makes a GET request to the test server.
func (ts *testServer) Get(path string) *testResponse {
	return ts.Request("GET", path, nil)
}

// Post makes a POST request with JSON body to the test server.
func (ts *testServer) Post(path string, body interface{}) *testResponse {
	jsonBody, _ := json.Marshal(body)
	return ts.Request("POST", path, bytes.NewReader(jsonBody))
}

// Put makes a PUT request with JSON body to the test server.
func (ts *testServer) Put(path string, body interface{}) *testResponse {
	jsonBody, _ := json.Marshal(body)
	return ts.Request("PUT", path, bytes.NewReader(jsonBody))
}

// Delete makes a DELETE request to the test server.
func (ts *testServer) Delete(path string) *testResponse {
	return ts.Request("DELETE", path, nil)
}

// Request makes an HTTP request to the test server.
func (ts *testServer) Request(method, path string, body io.Reader) *testResponse {
	req, err := http.NewRequest(method, ts.url+path, body)
	if err != nil {
		return &testResponse{err: err}
	}

	if body != nil {
		req.Header.Set("Content-Type", "application/json")
	}

	client := &http.Client{Timeout: 5 * time.Second}
	resp, err := client.Do(req)
	if err != nil {
		return &testResponse{err: err}
	}

	bodyBytes, _ := io.ReadAll(resp.Body)
	resp.Body.Close()

	return &testResponse{
		statusCode: resp.StatusCode,
		body:       bodyBytes,
		headers:    resp.Header,
	}
}

// testResponse wraps an HTTP response for testing.
type testResponse struct {
	statusCode int
	body       []byte
	headers    http.Header
	err        error
}

// AssertStatus asserts the response status code.
func (r *testResponse) AssertStatus(t *testing.T, expected int) *testResponse {
	t.Helper()
	if r.err != nil {
		t.Fatalf("request error: %v", r.err)
	}
	if r.statusCode != expected {
		t.Errorf("expected status %d, got %d", expected, r.statusCode)
	}
	return r
}

// AssertJSON asserts the response body matches expected JSON.
func (r *testResponse) AssertJSON(t *testing.T, expected interface{}) *testResponse {
	t.Helper()
	if r.err != nil {
		t.Fatalf("request error: %v", r.err)
	}

	var actual interface{}
	if err := json.Unmarshal(r.body, &actual); err != nil {
		t.Fatalf("failed to unmarshal response: %v\nBody: %s", err, r.body)
	}

	expectedJSON, _ := json.Marshal(expected)
	actualJSON, _ := json.Marshal(actual)

	if string(expectedJSON) != string(actualJSON) {
		t.Errorf("JSON mismatch\nExpected: %s\nActual:   %s", expectedJSON, actualJSON)
	}
	return r
}

// AssertHeader asserts a response header value.
func (r *testResponse) AssertHeader(t *testing.T, key, expected string) *testResponse {
	t.Helper()
	if r.err != nil {
		t.Fatalf("request error: %v", r.err)
	}
	actual := r.headers.Get(key)
	if actual != expected {
		t.Errorf("header %s: expected %s, got %s", key, expected, actual)
	}
	return r
}

// GetJSON unmarshals the response body into v.
func (r *testResponse) GetJSON(v interface{}) error {
	if r.err != nil {
		return r.err
	}
	return json.Unmarshal(r.body, v)
}

// waitForServer waits for the server to be ready to accept connections.
func waitForServer(addr string, timeout time.Duration) bool {
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		conn, err := net.DialTimeout("tcp", addr, 100*time.Millisecond)
		if err == nil {
			conn.Close()
			return true
		}
		time.Sleep(10 * time.Millisecond)
	}
	return false
}

// Integration Tests

// TestFullRequestCycle tests the complete request lifecycle from HTTP request to response.
//
// This tests:
//   - Shockwave server accepts connections
//   - Request routing works end-to-end
//   - Response is written correctly
//   - JSON encoding works
func TestFullRequestCycle(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	// Register route
	ts.app.Get("/hello", func(c *Context) error {
		return c.JSON(200, map[string]string{
			"message": "Hello, World!",
		})
	})

	// Make request
	resp := ts.Get("/hello")
	resp.AssertStatus(t, 200)
	resp.AssertJSON(t, map[string]string{
		"message": "Hello, World!",
	})
}

// TestStaticRoute tests simple GET request with static route.
func TestStaticRoute(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/ping", func(c *Context) error {
		return c.JSON(200, map[string]string{"status": "pong"})
	})

	resp := ts.Get("/ping")
	resp.AssertStatus(t, 200)
	resp.AssertJSON(t, map[string]string{"status": "pong"})
}

// TestDynamicRoute tests route with path parameters.
func TestDynamicRoute(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/users/:id", func(c *Context) error {
		return c.JSON(200, map[string]string{
			"id": c.Param("id"),
		})
	})

	resp := ts.Get("/users/123")
	resp.AssertStatus(t, 200)
	resp.AssertJSON(t, map[string]string{"id": "123"})
}

// TestMultipleParameters tests route with multiple path parameters.
func TestMultipleParameters(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/users/:userId/posts/:postId", func(c *Context) error {
		return c.JSON(200, map[string]string{
			"userId": c.Param("userId"),
			"postId": c.Param("postId"),
		})
	})

	resp := ts.Get("/users/42/posts/99")
	resp.AssertStatus(t, 200)
	resp.AssertJSON(t, map[string]string{
		"userId": "42",
		"postId": "99",
	})
}

// TestQueryParameters tests query string parsing.
func TestQueryParameters(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/search", func(c *Context) error {
		return c.JSON(200, map[string]string{
			"q":     c.Query("q"),
			"limit": c.Query("limit"),
			"page":  c.Query("page"),
		})
	})

	resp := ts.Get("/search?q=golang&limit=10&page=2")
	resp.AssertStatus(t, 200)
	resp.AssertJSON(t, map[string]string{
		"q":     "golang",
		"limit": "10",
		"page":  "2",
	})
}

// TestRequestBody tests POST request with JSON body.
func TestRequestBody(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	type CreateUserRequest struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	ts.app.Post("/users", func(c *Context) error {
		var req CreateUserRequest
		if err := c.BindJSON(&req); err != nil {
			return c.JSON(400, map[string]string{"error": "invalid json"})
		}

		return c.JSON(201, map[string]string{
			"name":  req.Name,
			"email": req.Email,
		})
	})

	body := CreateUserRequest{
		Name:  "Alice",
		Email: "alice@example.com",
	}

	resp := ts.Post("/users", body)
	resp.AssertStatus(t, 201)
	resp.AssertJSON(t, map[string]string{
		"name":  "Alice",
		"email": "alice@example.com",
	})
}

// TestMiddlewareStack tests global and route-specific middleware execution.
func TestMiddlewareStack(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	// Track execution order
	var mu sync.Mutex
	var execOrder []string

	// Global middleware 1
	globalMw1 := func(next Handler) Handler {
		return func(c *Context) error {
			mu.Lock()
			execOrder = append(execOrder, "global1-before")
			mu.Unlock()

			err := next(c)

			mu.Lock()
			execOrder = append(execOrder, "global1-after")
			mu.Unlock()
			return err
		}
	}

	// Global middleware 2
	globalMw2 := func(next Handler) Handler {
		return func(c *Context) error {
			mu.Lock()
			execOrder = append(execOrder, "global2-before")
			mu.Unlock()

			err := next(c)

			mu.Lock()
			execOrder = append(execOrder, "global2-after")
			mu.Unlock()
			return err
		}
	}

	// Route middleware
	routeMw := func(next Handler) Handler {
		return func(c *Context) error {
			mu.Lock()
			execOrder = append(execOrder, "route-before")
			mu.Unlock()

			err := next(c)

			mu.Lock()
			execOrder = append(execOrder, "route-after")
			mu.Unlock()
			return err
		}
	}

	ts.app.Use(globalMw1, globalMw2)

	ts.app.Get("/test", func(c *Context) error {
		mu.Lock()
		execOrder = append(execOrder, "handler")
		mu.Unlock()
		return c.JSON(200, map[string]string{"status": "ok"})
	}).Use(routeMw)

	resp := ts.Get("/test")
	resp.AssertStatus(t, 200)

	// Verify execution order
	// Route middleware executes OUTSIDE global middleware (wraps them)
	// This is correct: route middleware -> global1 -> global2 -> handler
	expected := []string{
		"route-before",
		"global1-before",
		"global2-before",
		"handler",
		"global2-after",
		"global1-after",
		"route-after",
	}

	mu.Lock()
	defer mu.Unlock()

	if len(execOrder) != len(expected) {
		t.Fatalf("expected %d middleware calls, got %d", len(expected), len(execOrder))
	}

	for i, exp := range expected {
		if execOrder[i] != exp {
			t.Errorf("execution[%d]: expected %s, got %s", i, exp, execOrder[i])
		}
	}
}

// TestErrorHandling tests error responses from handlers.
func TestErrorHandling(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/not-found", func(c *Context) error {
		return ErrNotFound
	})

	ts.app.Get("/bad-request", func(c *Context) error {
		return ErrBadRequest
	})

	ts.app.Get("/unauthorized", func(c *Context) error {
		return ErrUnauthorized
	})

	tests := []struct {
		path           string
		expectedStatus int
		expectedError  string
	}{
		{"/not-found", 404, "Not Found"},
		{"/bad-request", 400, "Bad Request"},
		{"/unauthorized", 401, "Unauthorized"},
	}

	for _, tt := range tests {
		t.Run(tt.path, func(t *testing.T) {
			resp := ts.Get(tt.path)
			resp.AssertStatus(t, tt.expectedStatus)
			resp.AssertJSON(t, map[string]string{"error": tt.expectedError})
		})
	}
}

// TestContextStorage tests passing data through context between middleware and handlers.
func TestContextStorage(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	// Middleware that sets user in context
	authMiddleware := func(next Handler) Handler {
		return func(c *Context) error {
			c.Set("user", "alice")
			c.Set("role", "admin")
			return next(c)
		}
	}

	ts.app.Use(authMiddleware)

	ts.app.Get("/profile", func(c *Context) error {
		user := c.Get("user").(string)
		role := c.Get("role").(string)

		return c.JSON(200, map[string]string{
			"user": user,
			"role": role,
		})
	})

	resp := ts.Get("/profile")
	resp.AssertStatus(t, 200)
	resp.AssertJSON(t, map[string]string{
		"user": "alice",
		"role": "admin",
	})
}

// TestConcurrentRequests tests handling 100 concurrent requests safely.
//
// Run with: go test -race ./core -run TestConcurrentRequests
func TestConcurrentRequests(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	// Request counter
	var counter int
	var mu sync.Mutex

	ts.app.Get("/increment", func(c *Context) error {
		mu.Lock()
		counter++
		current := counter
		mu.Unlock()

		// Simulate some work
		time.Sleep(1 * time.Millisecond)

		return c.JSON(200, map[string]int{"count": current})
	})

	// Make 100 concurrent requests
	const numRequests = 100
	var wg sync.WaitGroup
	errors := make(chan error, numRequests)

	for i := 0; i < numRequests; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()

			resp := ts.Get("/increment")
			if resp.err != nil {
				errors <- resp.err
				return
			}

			if resp.statusCode != 200 {
				errors <- fmt.Errorf("expected status 200, got %d", resp.statusCode)
				return
			}
		}()
	}

	wg.Wait()
	close(errors)

	// Check for errors
	for err := range errors {
		t.Errorf("concurrent request error: %v", err)
	}

	// Verify counter
	if counter != numRequests {
		t.Errorf("expected counter %d, got %d", numRequests, counter)
	}
}

// TestGracefulShutdown tests server graceful shutdown.
func TestGracefulShutdown(t *testing.T) {
	ts := createTestServer(t)

	ts.app.Get("/ping", func(c *Context) error {
		return c.JSON(200, map[string]string{"status": "pong"})
	})

	// Make a request to verify server is working
	resp := ts.Get("/ping")
	resp.AssertStatus(t, 200)

	// Shutdown server
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	// Give server a moment to ensure it's fully initialized
	time.Sleep(100 * time.Millisecond)

	err := ts.app.Shutdown(ctx)
	if err != nil {
		t.Errorf("shutdown error: %v", err)
	}

	// Verify server is stopped (connection should fail)
	time.Sleep(100 * time.Millisecond)
	resp = ts.Get("/ping")
	if resp.err == nil {
		t.Error("expected connection error after shutdown, got successful response")
	}
}

// TestCustomHeaders tests setting and reading custom headers.
func TestCustomHeaders(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/headers", func(c *Context) error {
		c.SetHeader("X-Custom-Header", "custom-value")
		c.SetHeader("X-Request-ID", "12345")
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	resp := ts.Get("/headers")
	resp.AssertStatus(t, 200)
	resp.AssertHeader(t, "X-Custom-Header", "custom-value")
	resp.AssertHeader(t, "X-Request-ID", "12345")
}

// TestDifferentHTTPMethods tests all HTTP methods.
func TestDifferentHTTPMethods(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/resource", func(c *Context) error {
		return c.JSON(200, map[string]string{"method": "GET"})
	})

	ts.app.Post("/resource", func(c *Context) error {
		return c.JSON(201, map[string]string{"method": "POST"})
	})

	ts.app.Put("/resource", func(c *Context) error {
		return c.JSON(200, map[string]string{"method": "PUT"})
	})

	ts.app.Delete("/resource", func(c *Context) error {
		return c.JSON(204, map[string]string{"method": "DELETE"})
	})

	tests := []struct {
		method         string
		expectedStatus int
		expectedMethod string
	}{
		{"GET", 200, "GET"},
		{"POST", 201, "POST"},
		{"PUT", 200, "PUT"},
		{"DELETE", 204, "DELETE"},
	}

	for _, tt := range tests {
		t.Run(tt.method, func(t *testing.T) {
			resp := ts.Request(tt.method, "/resource", nil)
			resp.AssertStatus(t, tt.expectedStatus)

			if tt.expectedStatus != 204 {
				resp.AssertJSON(t, map[string]string{"method": tt.expectedMethod})
			}
		})
	}
}

// TestNotFoundRoute tests 404 handling for non-existent routes.
func TestNotFoundRoute(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/exists", func(c *Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Request non-existent route
	resp := ts.Get("/does-not-exist")

	// Router should return 404
	if resp.statusCode != 404 {
		t.Errorf("expected 404 for non-existent route, got %d", resp.statusCode)
	}
}

// TestLargeJSONPayload tests handling large JSON responses.
func TestLargeJSONPayload(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/large", func(c *Context) error {
		// Create large response (10KB)
		data := make(map[string]string)
		for i := 0; i < 100; i++ {
			data[fmt.Sprintf("key_%d", i)] = fmt.Sprintf("value_%d_with_some_padding_to_increase_size", i)
		}
		return c.JSON(200, data)
	})

	resp := ts.Get("/large")
	resp.AssertStatus(t, 200)

	// Verify response is valid JSON
	var result map[string]string
	if err := resp.GetJSON(&result); err != nil {
		t.Errorf("failed to parse large JSON: %v", err)
	}

	if len(result) != 100 {
		t.Errorf("expected 100 keys, got %d", len(result))
	}
}

// BenchmarkIntegrationFullCycle benchmarks full request/response cycle.
func BenchmarkIntegrationFullCycle(b *testing.B) {
	// Create test app (not running server for benchmark)
	app := New()

	app.Get("/bench", func(c *Context) error {
		return c.JSON(200, map[string]string{"message": "hello"})
	})

	// Note: This benchmark doesn't use real HTTP server
	// For full integration benchmark, we'd need to start actual server
	// which adds significant overhead

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		// This benchmarks routing + handler execution
		// without HTTP server overhead
		ctx := app.contextPool.Acquire()
		ctx.methodBytes = []byte("GET")
		ctx.pathBytes = []byte("/bench")

		app.router.ServeHTTP(ctx)
		app.contextPool.Release(ctx)
	}
}

// TestJSONBytesIntegration tests the JSONBytes method with actual Shockwave server.
func TestJSONBytesIntegration(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	precomputedJSON := []byte(`{"status":"ok","precomputed":true,"value":42}`)
	ts.app.Get("/precomputed", func(c *Context) error {
		return c.JSONBytes(200, precomputedJSON)
	})

	resp := ts.Get("/precomputed")
	resp.AssertStatus(t, 200)

	var result map[string]interface{}
	if err := json.Unmarshal(resp.body, &result); err != nil {
		t.Fatalf("failed to unmarshal response: %v", err)
	}

	if result["status"] != "ok" {
		t.Errorf("expected status=ok, got %v", result["status"])
	}
	if result["precomputed"] != true {
		t.Errorf("expected precomputed=true, got %v", result["precomputed"])
	}
	if result["value"].(float64) != 42 {
		t.Errorf("expected value=42, got %v", result["value"])
	}

	// Verify Content-Type header
	contentType := resp.headers.Get("Content-Type")
	if contentType != "application/json" {
		t.Errorf("expected Content-Type=application/json, got %s", contentType)
	}
}

// TestTextIntegration tests the Text method with actual Shockwave server.
func TestTextIntegration(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/text", func(c *Context) error {
		return c.Text(200, "Hello, World! This is plain text.")
	})

	ts.app.Get("/text-multi-line", func(c *Context) error {
		return c.Text(200, "Line 1\nLine 2\nLine 3")
	})

	// Test simple text
	resp := ts.Get("/text")
	resp.AssertStatus(t, 200)

	expected := "Hello, World! This is plain text."
	if string(resp.body) != expected {
		t.Errorf("expected %q, got %q", expected, string(resp.body))
	}

	contentType := resp.headers.Get("Content-Type")
	if !strings.Contains(contentType, "text/plain") {
		t.Errorf("expected Content-Type to contain text/plain, got %s", contentType)
	}

	// Test multi-line text
	resp2 := ts.Get("/text-multi-line")
	resp2.AssertStatus(t, 200)

	expectedMulti := "Line 1\nLine 2\nLine 3"
	if string(resp2.body) != expectedMulti {
		t.Errorf("expected %q, got %q", expectedMulti, string(resp2.body))
	}
}

// TestHTMLIntegration tests the HTML method with actual Shockwave server.
func TestHTMLIntegration(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/html", func(c *Context) error {
		return c.HTML(200, "<h1>Hello, World!</h1>")
	})

	ts.app.Get("/html-full", func(c *Context) error {
		html := `<!DOCTYPE html>
<html>
<head><title>Test</title></head>
<body><h1>Welcome</h1><p>Test page</p></body>
</html>`
		return c.HTML(200, html)
	})

	// Test simple HTML
	resp := ts.Get("/html")
	resp.AssertStatus(t, 200)

	expected := "<h1>Hello, World!</h1>"
	if string(resp.body) != expected {
		t.Errorf("expected %q, got %q", expected, string(resp.body))
	}

	contentType := resp.headers.Get("Content-Type")
	if !strings.Contains(contentType, "text/html") {
		t.Errorf("expected Content-Type to contain text/html, got %s", contentType)
	}

	// Test full HTML document
	resp2 := ts.Get("/html-full")
	resp2.AssertStatus(t, 200)

	if !strings.Contains(string(resp2.body), "<!DOCTYPE html>") {
		t.Error("expected full HTML document")
	}
	if !strings.Contains(string(resp2.body), "<h1>Welcome</h1>") {
		t.Error("expected <h1>Welcome</h1> in response")
	}
}

// TestNoContentIntegration tests the NoContent method with actual Shockwave server.
func TestNoContentIntegration(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Delete("/resource/:id", func(c *Context) error {
		// Simulate deletion
		id := c.Param("id")
		if id == "" {
			return c.JSON(400, map[string]string{"error": "missing id"})
		}
		return c.NoContent()
	})

	ts.app.Put("/resource/:id/archive", func(c *Context) error {
		// Simulate archiving (204 response)
		return c.NoContent()
	})

	// Test DELETE with NoContent
	resp := ts.Delete("/resource/123")
	resp.AssertStatus(t, 204)

	if len(resp.body) != 0 {
		t.Errorf("expected empty body for 204, got %d bytes", len(resp.body))
	}

	// Test PUT with NoContent
	resp2 := ts.Put("/resource/456/archive", nil)
	resp2.AssertStatus(t, 204)

	if len(resp2.body) != 0 {
		t.Errorf("expected empty body for 204, got %d bytes", len(resp2.body))
	}
}

// TestResponseHeadersIntegration tests setting and reading response headers.
func TestResponseHeadersIntegration(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/headers", func(c *Context) error {
		// Set multiple custom headers
		c.SetHeader("X-Custom-Header", "custom-value")
		c.SetHeader("X-Request-ID", "req-12345")
		c.SetHeader("X-Server-Version", "1.0.0")
		c.SetHeader("Cache-Control", "no-cache, no-store, must-revalidate")

		return c.JSON(200, map[string]string{"status": "ok"})
	})

	resp := ts.Get("/headers")
	resp.AssertStatus(t, 200)

	// Verify all custom headers
	if resp.headers.Get("X-Custom-Header") != "custom-value" {
		t.Errorf("expected X-Custom-Header=custom-value, got %s", resp.headers.Get("X-Custom-Header"))
	}
	if resp.headers.Get("X-Request-ID") != "req-12345" {
		t.Errorf("expected X-Request-ID=req-12345, got %s", resp.headers.Get("X-Request-ID"))
	}
	if resp.headers.Get("X-Server-Version") != "1.0.0" {
		t.Errorf("expected X-Server-Version=1.0.0, got %s", resp.headers.Get("X-Server-Version"))
	}
	if resp.headers.Get("Cache-Control") != "no-cache, no-store, must-revalidate" {
		t.Errorf("expected Cache-Control header, got %s", resp.headers.Get("Cache-Control"))
	}
}

// TestRequestHeadersIntegration tests reading request headers with Shockwave.
func TestRequestHeadersIntegration(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	ts.app.Get("/check-auth", func(c *Context) error {
		auth := c.GetHeader("Authorization")
		userAgent := c.GetHeader("User-Agent")
		acceptLang := c.GetHeader("Accept-Language")
		contentType := c.GetHeader("Content-Type")

		return c.JSON(200, map[string]string{
			"auth":         auth,
			"user_agent":   userAgent,
			"accept_lang":  acceptLang,
			"content_type": contentType,
		})
	})

	// Create request with custom headers
	req, _ := http.NewRequest("GET", ts.url+"/check-auth", nil)
	req.Header.Set("Authorization", "Bearer secret-token-abc123")
	req.Header.Set("User-Agent", "BoltTestClient/2.0")
	req.Header.Set("Accept-Language", "en-US,en;q=0.9")
	req.Header.Set("Content-Type", "application/json")

	client := &http.Client{Timeout: 5 * time.Second}
	httpResp, err := client.Do(req)
	if err != nil {
		t.Fatalf("request failed: %v", err)
	}
	defer httpResp.Body.Close()

	body, _ := io.ReadAll(httpResp.Body)
	var result map[string]string
	if err := json.Unmarshal(body, &result); err != nil {
		t.Fatalf("failed to unmarshal: %v", err)
	}

	if result["auth"] != "Bearer secret-token-abc123" {
		t.Errorf("expected auth='Bearer secret-token-abc123', got %s", result["auth"])
	}
	if result["user_agent"] != "BoltTestClient/2.0" {
		t.Errorf("expected user_agent='BoltTestClient/2.0', got %s", result["user_agent"])
	}
	if result["accept_lang"] != "en-US,en;q=0.9" {
		t.Errorf("expected accept_lang='en-US,en;q=0.9', got %s", result["accept_lang"])
	}
	if result["content_type"] != "application/json" {
		t.Errorf("expected content_type='application/json', got %s", result["content_type"])
	}
}

// TestMixedResponseTypes tests using different response types in same app.
func TestMixedResponseTypes(t *testing.T) {
	ts := createTestServer(t)
	defer ts.Shutdown()

	// Register different response type handlers
	ts.app.Get("/api/json", func(c *Context) error {
		return c.JSON(200, map[string]interface{}{
			"type": "json",
			"data": []int{1, 2, 3},
		})
	})

	ts.app.Get("/api/text", func(c *Context) error {
		return c.Text(200, "Plain text response")
	})

	ts.app.Get("/api/html", func(c *Context) error {
		return c.HTML(200, "<div>HTML response</div>")
	})

	ts.app.Get("/api/bytes", func(c *Context) error {
		return c.JSONBytes(200, []byte(`{"type":"bytes"}`))
	})

	// Test each endpoint
	jsonResp := ts.Get("/api/json")
	jsonResp.AssertStatus(t, 200)
	if !strings.Contains(jsonResp.headers.Get("Content-Type"), "application/json") {
		t.Error("JSON endpoint should have application/json Content-Type")
	}

	textResp := ts.Get("/api/text")
	textResp.AssertStatus(t, 200)
	if string(textResp.body) != "Plain text response" {
		t.Errorf("unexpected text response: %s", string(textResp.body))
	}

	htmlResp := ts.Get("/api/html")
	htmlResp.AssertStatus(t, 200)
	if !strings.Contains(htmlResp.headers.Get("Content-Type"), "text/html") {
		t.Error("HTML endpoint should have text/html Content-Type")
	}

	bytesResp := ts.Get("/api/bytes")
	bytesResp.AssertStatus(t, 200)
	var bytesResult map[string]string
	if err := json.Unmarshal(bytesResp.body, &bytesResult); err != nil {
		t.Errorf("failed to parse bytes response: %v", err)
	}
	if bytesResult["type"] != "bytes" {
		t.Errorf("expected type=bytes, got %s", bytesResult["type"])
	}
}
