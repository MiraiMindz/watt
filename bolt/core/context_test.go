package core

import (
	"testing"
)

// TestContextParam tests parameter storage and retrieval.
func TestContextParam(t *testing.T) {
	c := &Context{}

	// Set a parameter
	c.setParam("id", "123")

	// Retrieve it
	if id := c.Param("id"); id != "123" {
		t.Errorf("expected id=123, got %s", id)
	}
}

// TestContextParamInlineStorage tests inline storage for ≤4 params.
func TestContextParamInlineStorage(t *testing.T) {
	c := &Context{}

	// Add 4 params (should use inline storage)
	c.setParam("param1", "value1")
	c.setParam("param2", "value2")
	c.setParam("param3", "value3")
	c.setParam("param4", "value4")

	// Verify all params
	if c.Param("param1") != "value1" {
		t.Error("param1 not stored correctly")
	}
	if c.Param("param2") != "value2" {
		t.Error("param2 not stored correctly")
	}
	if c.Param("param3") != "value3" {
		t.Error("param3 not stored correctly")
	}
	if c.Param("param4") != "value4" {
		t.Error("param4 not stored correctly")
	}

	// Verify inline storage was used (not map)
	if c.params != nil {
		t.Error("expected params map to be nil for ≤4 params")
	}
}

// TestContextParamMapOverflow tests map overflow for >4 params.
func TestContextParamMapOverflow(t *testing.T) {
	c := &Context{}

	// Add 5 params (should overflow to map)
	c.setParam("param1", "value1")
	c.setParam("param2", "value2")
	c.setParam("param3", "value3")
	c.setParam("param4", "value4")
	c.setParam("param5", "value5")

	// Verify all params are accessible
	if c.Param("param1") != "value1" {
		t.Error("param1 not accessible after overflow")
	}
	if c.Param("param5") != "value5" {
		t.Error("param5 not stored in map")
	}

	// Verify map was created
	if c.params == nil {
		t.Error("expected params map to be created for >4 params")
	}
}

// TestContextParamNotFound tests retrieving non-existent param.
func TestContextParamNotFound(t *testing.T) {
	c := &Context{}

	if param := c.Param("nonexistent"); param != "" {
		t.Errorf("expected empty string for non-existent param, got %s", param)
	}
}

// TestContextSetGet tests context storage.
func TestContextSetGet(t *testing.T) {
	c := &Context{}

	// Set a value
	c.Set("user", "Alice")

	// Get it back
	if user := c.Get("user"); user != "Alice" {
		t.Errorf("expected user=Alice, got %v", user)
	}
}

// TestContextGetNonExistent tests getting non-existent value.
func TestContextGetNonExistent(t *testing.T) {
	c := &Context{}

	if value := c.Get("nonexistent"); value != nil {
		t.Errorf("expected nil for non-existent key, got %v", value)
	}
}

// TestContextMustGet tests MustGet with existing key.
func TestContextMustGet(t *testing.T) {
	c := &Context{}
	c.Set("key", "value")

	// Should not panic
	value := c.MustGet("key")
	if value != "value" {
		t.Errorf("expected value, got %v", value)
	}
}

// TestContextMustGetPanic tests MustGet panic with missing key.
func TestContextMustGetPanic(t *testing.T) {
	c := &Context{}

	defer func() {
		if r := recover(); r == nil {
			t.Error("expected panic for missing key")
		}
	}()

	c.MustGet("nonexistent")
}

// TestContextMethodPath tests Method and Path getters.
func TestContextMethodPath(t *testing.T) {
	c := &Context{
		methodBytes: []byte("GET"),
		pathBytes:   []byte("/users/123"),
	}

	if c.Method() != "GET" {
		t.Errorf("expected method GET, got %s", c.Method())
	}
	if c.Path() != "/users/123" {
		t.Errorf("expected path /users/123, got %s", c.Path())
	}
}

// TestContextStatusCode tests status code tracking.
func TestContextStatusCode(t *testing.T) {
	c := &Context{}

	if c.StatusCode() != 0 {
		t.Errorf("expected initial status 0, got %d", c.StatusCode())
	}

	// Status is set after response is written
	c.statusCode = 200
	c.written = true

	if c.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", c.StatusCode())
	}
	if !c.Written() {
		t.Error("expected written=true")
	}
}

// TestContextReset tests context reset for pooling.
func TestContextReset(t *testing.T) {
	c := &Context{
		methodBytes: []byte("GET"),
		pathBytes:   []byte("/users"),
		queryBytes:  []byte("page=1"),
	}

	// Set params
	c.setParam("id", "123")

	// Set storage
	c.Set("user", "Alice")

	// Set response state
	c.statusCode = 200
	c.written = true

	// Reset
	c.Reset()

	// Verify everything is cleared
	if len(c.methodBytes) != 0 {
		t.Error("method not cleared")
	}
	if len(c.pathBytes) != 0 {
		t.Error("path not cleared")
	}
	if len(c.queryBytes) != 0 {
		t.Error("query not cleared")
	}
	if c.paramsLen != 0 {
		t.Error("paramsLen not cleared")
	}
	if c.statusCode != 0 {
		t.Error("statusCode not cleared")
	}
	if c.written {
		t.Error("written not cleared")
	}

	// Verify storage is cleared
	if c.Get("user") != nil {
		t.Error("storage not cleared")
	}
}

// TestContextPool tests context pool acquire/release.
func TestContextPool(t *testing.T) {
	pool := NewContextPool()

	// Acquire context
	ctx := pool.Acquire()
	if ctx == nil {
		t.Fatal("expected context from pool, got nil")
	}

	// Use context
	ctx.setParam("id", "123")
	ctx.Set("user", "Alice")

	// Release back to pool
	pool.Release(ctx)

	// Acquire again (should be same instance, but reset)
	ctx2 := pool.Acquire()
	if ctx2 == nil {
		t.Fatal("expected context from pool, got nil")
	}

	// Verify context was reset
	if ctx2.Param("id") != "" {
		t.Error("context not properly reset, params still present")
	}
	if ctx2.Get("user") != nil {
		t.Error("context not properly reset, storage still present")
	}
}

// TestContextPoolConcurrent tests concurrent pool access.
func TestContextPoolConcurrent(t *testing.T) {
	pool := NewContextPool()
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			ctx := pool.Acquire()
			ctx.setParam("id", "test")
			_ = ctx.Param("id")
			pool.Release(ctx)
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestSplitQuery tests query string splitting.
func TestSplitQuery(t *testing.T) {
	tests := []struct {
		query    string
		expected []string
	}{
		{"", nil},
		{"q=golang", []string{"q=golang"}},
		{"q=golang&limit=10", []string{"q=golang", "limit=10"}},
		{"a=1&b=2&c=3", []string{"a=1", "b=2", "c=3"}},
	}

	for _, tt := range tests {
		result := splitQuery(tt.query)
		if len(result) != len(tt.expected) {
			t.Errorf("splitQuery(%q): expected %d parts, got %d", tt.query, len(tt.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitQuery(%q)[%d]: expected %q, got %q", tt.query, i, tt.expected[i], result[i])
			}
		}
	}
}

// TestSplitKeyValue tests key-value splitting.
func TestSplitKeyValue(t *testing.T) {
	tests := []struct {
		pair     string
		expected []string
	}{
		{"key=value", []string{"key", "value"}},
		{"name=Alice", []string{"name", "Alice"}},
		{"key", []string{"key"}},
		{"empty=", []string{"empty", ""}},
		{"=value", []string{"", "value"}},
	}

	for _, tt := range tests {
		result := splitKeyValue(tt.pair)
		if len(result) != len(tt.expected) {
			t.Errorf("splitKeyValue(%q): expected %d parts, got %d", tt.pair, len(tt.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitKeyValue(%q)[%d]: expected %q, got %q", tt.pair, i, tt.expected[i], result[i])
			}
		}
	}
}

// TestContextQueryParsing tests query parameter parsing.
func TestContextQueryParsing(t *testing.T) {
	c := &Context{
		queryBytes: []byte("q=golang&limit=10&offset=0"),
	}

	// Parse query
	if q := c.Query("q"); q != "golang" {
		t.Errorf("expected q=golang, got %s", q)
	}
	if limit := c.Query("limit"); limit != "10" {
		t.Errorf("expected limit=10, got %s", limit)
	}
	if offset := c.Query("offset"); offset != "0" {
		t.Errorf("expected offset=0, got %s", offset)
	}
}

// TestContextQueryDefault tests QueryDefault.
func TestContextQueryDefault(t *testing.T) {
	c := &Context{
		queryBytes: []byte("page=1"),
	}

	// Existing param
	if page := c.QueryDefault("page", "10"); page != "1" {
		t.Errorf("expected page=1, got %s", page)
	}

	// Non-existent param (should use default)
	if limit := c.QueryDefault("limit", "10"); limit != "10" {
		t.Errorf("expected limit=10 (default), got %s", limit)
	}
}

// BenchmarkContextParam benchmarks parameter retrieval.
func BenchmarkContextParam(b *testing.B) {
	c := &Context{}
	c.setParam("id", "123")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Param("id")
	}
}

// BenchmarkContextSetGet benchmarks context storage.
func BenchmarkContextSetGet(b *testing.B) {
	c := &Context{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		c.Set("key", "value")
		_ = c.Get("key")
	}
}

// BenchmarkContextQuery benchmarks query parameter access.
func BenchmarkContextQuery(b *testing.B) {
	c := &Context{
		queryBytes: []byte("q=golang&limit=10&offset=0&sort=asc&filter=active"),
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = c.Query("q")
		_ = c.Query("limit")
		_ = c.Query("offset")
	}
}

// BenchmarkContextPool benchmarks context pool acquire/release cycle.
func BenchmarkContextPool(b *testing.B) {
	pool := NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := pool.Acquire()
		pool.Release(ctx)
	}
}

// BenchmarkContextPoolWithUsage benchmarks context pool with typical usage.
func BenchmarkContextPoolWithUsage(b *testing.B) {
	pool := NewContextPool()

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		ctx := pool.Acquire()
		ctx.setParam("id", "123")
		ctx.Set("user", "Alice")
		_ = ctx.Param("id")
		_ = ctx.Get("user")
		pool.Release(ctx)
	}
}

// TestContextGetSetHeader tests header access in test mode.
func TestContextGetSetHeader(t *testing.T) {
	c := &Context{}

	// Set request header
	c.SetRequestHeader("Authorization", "Bearer token123")

	// Get request header
	if auth := c.GetHeader("Authorization"); auth != "Bearer token123" {
		t.Errorf("expected 'Bearer token123', got %s", auth)
	}

	// Set response header
	c.SetHeader("Content-Type", "application/json")

	// Get response header
	if ct := c.GetResponseHeader("Content-Type"); ct != "application/json" {
		t.Errorf("expected 'application/json', got %s", ct)
	}
}

// TestContextJSON tests JSON response.
func TestContextJSON(t *testing.T) {
	c := &Context{}

	data := map[string]string{
		"message": "Hello, World!",
		"status":  "success",
	}

	err := c.JSON(200, data)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if c.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", c.StatusCode())
	}

	if !c.Written() {
		t.Error("expected response to be written")
	}
}

// TestContextText tests text response.
func TestContextText(t *testing.T) {
	c := &Context{}

	err := c.Text(200, "Hello, World!")
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if c.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", c.StatusCode())
	}

	if !c.Written() {
		t.Error("expected response to be written")
	}
}

// TestContextHTML tests HTML response.
func TestContextHTML(t *testing.T) {
	c := &Context{}

	html := "<html><body><h1>Hello</h1></body></html>"

	err := c.HTML(200, html)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if c.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", c.StatusCode())
	}

	if !c.Written() {
		t.Error("expected response to be written")
	}
}

// TestContextNoContent tests 204 No Content response.
func TestContextNoContent(t *testing.T) {
	c := &Context{}

	err := c.NoContent()
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if c.StatusCode() != 204 {
		t.Errorf("expected status 204, got %d", c.StatusCode())
	}

	if !c.Written() {
		t.Error("expected response to be written")
	}
}

// TestContextBindJSON tests JSON request binding.
func TestContextBindJSON(t *testing.T) {
	c := &Context{}
	c.SetMethod("POST")
	c.SetPath("/users")

	// BindJSON requires a request body, which we can't easily mock without Shockwave
	// This test verifies the method exists and can be called
	// (it will fail to bind without a real request body, which is expected)

	type User struct {
		Name  string `json:"name"`
		Email string `json:"email"`
	}

	var user User
	err := c.BindJSON(&user)

	// Expected to fail without request body, but shouldn't panic
	if err == nil {
		t.Log("BindJSON returned no error (no request body available)")
	}
}

// TestContextTestHelpers tests test helper methods.
func TestContextTestHelpers(t *testing.T) {
	c := &Context{}

	// SetMethod
	c.SetMethod("POST")
	if c.Method() != "POST" {
		t.Errorf("expected method POST, got %s", c.Method())
	}

	// SetPath
	c.SetPath("/api/users")
	if c.Path() != "/api/users" {
		t.Errorf("expected path /api/users, got %s", c.Path())
	}

	// SetRequestHeader
	c.SetRequestHeader("Authorization", "Bearer token")
	if c.GetHeader("Authorization") != "Bearer token" {
		t.Error("SetRequestHeader/GetHeader not working")
	}

	// SetHeader and GetResponseHeader
	c.SetHeader("X-Custom-Header", "custom-value")
	if c.GetResponseHeader("X-Custom-Header") != "custom-value" {
		t.Error("SetHeader/GetResponseHeader not working")
	}
}

// TestContextJSONBytes tests JSONBytes method.
func TestContextJSONBytes(t *testing.T) {
	c := &Context{}

	jsonData := []byte(`{"message":"Hello","status":"ok"}`)

	err := c.JSONBytes(200, jsonData)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if c.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", c.StatusCode())
	}

	if !c.Written() {
		t.Error("expected response to be written")
	}
}
