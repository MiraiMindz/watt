package core

import (
	"testing"
)

// Test handler that returns nil
func testHandler(c *Context) error {
	return nil
}

// Test handler that sets a value
func testHandlerWithValue(value string) Handler {
	return func(c *Context) error {
		c.Set("value", value)
		return nil
	}
}

// TestNewRouter tests router creation.
func TestNewRouter(t *testing.T) {
	r := NewRouter()

	if r == nil {
		t.Fatal("expected router, got nil")
	}
	if r.static == nil {
		t.Error("expected static map to be initialized")
	}
	if r.trees == nil {
		t.Error("expected trees map to be initialized")
	}
}

// TestAddStaticRoute tests adding static routes.
func TestAddStaticRoute(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users", testHandler)

	// Verify static route was added
	key := "GET:/users"
	if _, ok := r.static[key]; !ok {
		t.Errorf("expected static route %s to be added", key)
	}
}

// TestAddDynamicRoute tests adding dynamic routes with parameters.
func TestAddDynamicRoute(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users/:id", testHandler)

	// Verify tree was created
	if r.trees[MethodGet] == nil {
		t.Error("expected tree for GET method")
	}
}

// TestLookupStaticRoute tests looking up static routes.
func TestLookupStaticRoute(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users", testHandlerWithValue("users"))
	r.Add(MethodPost, "/users", testHandlerWithValue("create-user"))

	// Test GET /users
	handler, params := r.Lookup(MethodGet, "/users")
	if handler == nil {
		t.Fatal("expected handler, got nil")
	}
	if params != nil {
		t.Error("expected no params for static route")
	}

	// Execute handler to verify it's the right one
	c := &Context{}
	handler(c)
	if c.Get("value") != "users" {
		t.Errorf("expected value 'users', got '%v'", c.Get("value"))
	}

	// Test POST /users
	handler, _ = r.Lookup(MethodPost, "/users")
	if handler == nil {
		t.Fatal("expected handler for POST, got nil")
	}
}

// TestLookupDynamicRoute tests looking up dynamic routes with parameters.
func TestLookupDynamicRoute(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users/:id", testHandler)

	handler, params := r.Lookup(MethodGet, "/users/123")

	if handler == nil {
		t.Fatal("expected handler, got nil")
	}
	if params == nil {
		t.Fatal("expected params, got nil")
	}
	if params["id"] != "123" {
		t.Errorf("expected id=123, got %s", params["id"])
	}
}

// TestLookupMultipleParams tests routes with multiple parameters.
func TestLookupMultipleParams(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users/:userId/posts/:postId", testHandler)

	handler, params := r.Lookup(MethodGet, "/users/123/posts/456")

	if handler == nil {
		t.Fatal("expected handler, got nil")
	}
	if params["userId"] != "123" {
		t.Errorf("expected userId=123, got %s", params["userId"])
	}
	if params["postId"] != "456" {
		t.Errorf("expected postId=456, got %s", params["postId"])
	}
}

// TestLookupWildcard tests wildcard routes.
func TestLookupWildcard(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/files/*filepath", testHandler)

	handler, params := r.Lookup(MethodGet, "/files/documents/report.pdf")

	if handler == nil {
		t.Fatal("expected handler, got nil")
	}
	if params["filepath"] != "documents/report.pdf" {
		t.Errorf("expected filepath=documents/report.pdf, got %s", params["filepath"])
	}
}

// TestLookupNotFound tests looking up non-existent routes.
func TestLookupNotFound(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users", testHandler)

	handler, params := r.Lookup(MethodGet, "/posts")

	if handler != nil {
		t.Error("expected nil handler for non-existent route")
	}
	if params != nil {
		t.Error("expected nil params for non-existent route")
	}
}

// TestLookupMethodNotAllowed tests accessing route with wrong method.
func TestLookupMethodNotAllowed(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users", testHandler)

	// Try POST on GET-only route
	handler, _ := r.Lookup(MethodPost, "/users")

	if handler != nil {
		t.Error("expected nil handler for wrong method")
	}
}

// TestRootPath tests root path handling.
func TestRootPath(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/", testHandler)

	handler, params := r.Lookup(MethodGet, "/")

	if handler == nil {
		t.Fatal("expected handler for root path")
	}
	if params != nil {
		t.Error("expected no params for root path")
	}
}

// TestSplitPath tests path splitting.
func TestSplitPath(t *testing.T) {
	tests := []struct {
		path     string
		expected []string
	}{
		{"/", []string{}},
		{"", []string{}},
		{"/users", []string{"users"}},
		{"/users/123", []string{"users", "123"}},
		{"/users/:id/posts", []string{"users", ":id", "posts"}},
		{"users/123", []string{"users", "123"}},    // No leading slash
		{"/users/", []string{"users"}},             // Trailing slash
		{"/users//posts", []string{"users", "posts"}}, // Double slash
	}

	for _, tt := range tests {
		result := splitPath(tt.path)
		if len(result) != len(tt.expected) {
			t.Errorf("splitPath(%q): expected %d segments, got %d", tt.path, len(tt.expected), len(result))
			continue
		}
		for i := range result {
			if result[i] != tt.expected[i] {
				t.Errorf("splitPath(%q)[%d]: expected %q, got %q", tt.path, i, tt.expected[i], result[i])
			}
		}
	}
}

// TestRouterConcurrency tests concurrent access to router.
func TestRouterConcurrency(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/users/:id", testHandler)
	r.Add(MethodGet, "/posts/:id", testHandler)

	// Concurrent lookups
	done := make(chan bool, 100)

	for i := 0; i < 100; i++ {
		go func(id int) {
			// Alternate between routes
			if id%2 == 0 {
				handler, params := r.Lookup(MethodGet, "/users/123")
				if handler == nil || params["id"] != "123" {
					t.Error("concurrent lookup failed for /users/123")
				}
			} else {
				handler, params := r.Lookup(MethodGet, "/posts/456")
				if handler == nil || params["id"] != "456" {
					t.Error("concurrent lookup failed for /posts/456")
				}
			}
			done <- true
		}(i)
	}

	// Wait for all goroutines
	for i := 0; i < 100; i++ {
		<-done
	}
}

// TestHybridRoutingPerformance tests that static routes are faster.
func TestHybridRoutingPerformance(t *testing.T) {
	r := NewRouter()
	r.Add(MethodGet, "/static", testHandler)       // Static
	r.Add(MethodGet, "/dynamic/:id", testHandler)  // Dynamic

	// This is just a smoke test - actual benchmarks are in BenchmarkRouter*
	handler, _ := r.Lookup(MethodGet, "/static")
	if handler == nil {
		t.Error("static route lookup failed")
	}

	handler, params := r.Lookup(MethodGet, "/dynamic/123")
	if handler == nil || params["id"] != "123" {
		t.Error("dynamic route lookup failed")
	}
}

// BenchmarkRouter_StaticRoute benchmarks static route lookup.
func BenchmarkRouter_StaticRoute(b *testing.B) {
	r := NewRouter()
	r.Add(MethodGet, "/users", testHandler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Lookup(MethodGet, "/users")
	}
}

// BenchmarkRouter_DynamicRoute benchmarks dynamic route lookup.
func BenchmarkRouter_DynamicRoute(b *testing.B) {
	r := NewRouter()
	r.Add(MethodGet, "/users/:id", testHandler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Lookup(MethodGet, "/users/123")
	}
}

// BenchmarkRouter_MultipleParams benchmarks route with multiple parameters.
func BenchmarkRouter_MultipleParams(b *testing.B) {
	r := NewRouter()
	r.Add(MethodGet, "/users/:userId/posts/:postId/comments/:commentId", testHandler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Lookup(MethodGet, "/users/123/posts/456/comments/789")
	}
}

// BenchmarkRouter_Wildcard benchmarks wildcard route lookup.
func BenchmarkRouter_Wildcard(b *testing.B) {
	r := NewRouter()
	r.Add(MethodGet, "/files/*filepath", testHandler)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r.Lookup(MethodGet, "/files/documents/reports/2024/report.pdf")
	}
}

// BenchmarkRouter_Add benchmarks adding routes.
func BenchmarkRouter_Add(b *testing.B) {
	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		r := NewRouter()
		r.Add(MethodGet, "/users/:id", testHandler)
	}
}
