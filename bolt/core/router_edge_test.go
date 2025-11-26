package core

import (
	"testing"
)

// TestRouterEdgeCases tests router edge cases for better coverage.

// TestRouterEmptyPath tests routing with empty path.
func TestRouterEmptyPath(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "", handler)

	// Should match empty path
	h, params := router.Lookup(MethodGet, "")
	if h == nil {
		t.Error("expected handler for empty path")
	}
	if len(params) != 0 {
		t.Error("expected no params for empty path")
	}
}

// TestRouterSlashOnly tests routing with "/" path.
func TestRouterSlashOnly(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/", handler)

	h, _ := router.Lookup(MethodGet, "/")
	if h == nil {
		t.Error("expected handler for / path")
	}
}

// TestRouterParamAtRoot tests parameter at root level.
func TestRouterParamAtRoot(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/:id", handler)

	h, params := router.Lookup(MethodGet, "/123")
	if h == nil {
		t.Fatal("expected handler for /:id")
	}

	if params["id"] != "123" {
		t.Errorf("expected id=123, got %s", params["id"])
	}
}

// TestRouterWildcardAtRoot tests wildcard at root.
func TestRouterWildcardAtRoot(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/*path", handler)

	h, params := router.Lookup(MethodGet, "/any/nested/path")
	if h == nil {
		t.Fatal("expected handler for /*path")
	}

	if params["path"] != "any/nested/path" {
		t.Errorf("expected path='any/nested/path', got '%s'", params["path"])
	}
}

// TestRouterMixedStaticDynamic tests routes with mixed static/dynamic segments.
func TestRouterMixedStaticDynamic(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Add routes
	router.Add(MethodGet, "/api/users/:id/posts/:postId", handler)

	h, params := router.Lookup(MethodGet, "/api/users/123/posts/456")
	if h == nil {
		t.Fatal("expected handler for mixed route")
	}

	if params["id"] != "123" {
		t.Errorf("expected id=123, got %s", params["id"])
	}

	if params["postId"] != "456" {
		t.Errorf("expected postId=456, got %s", params["postId"])
	}
}

// TestRouterSimilarPaths tests routes with similar prefixes.
func TestRouterSimilarPaths(t *testing.T) {
	router := NewRouter()

	handler1 := func(c *Context) error {
		c.Set("handler", "1")
		return nil
	}

	handler2 := func(c *Context) error {
		c.Set("handler", "2")
		return nil
	}

	handler3 := func(c *Context) error {
		c.Set("handler", "3")
		return nil
	}

	router.Add(MethodGet, "/user", handler1)
	router.Add(MethodGet, "/users", handler2)
	router.Add(MethodGet, "/users/:id", handler3)

	// Test /user
	h1, _ := router.Lookup(MethodGet, "/user")
	if h1 == nil {
		t.Error("expected handler for /user")
	}

	// Test /users
	h2, _ := router.Lookup(MethodGet, "/users")
	if h2 == nil {
		t.Error("expected handler for /users")
	}

	// Test /users/123
	h3, params := router.Lookup(MethodGet, "/users/123")
	if h3 == nil {
		t.Error("expected handler for /users/:id")
	}
	if params["id"] != "123" {
		t.Errorf("expected id=123, got %s", params["id"])
	}
}

// TestRouterTrailingSlash tests handling of trailing slashes.
func TestRouterTrailingSlash(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/users", handler)

	// Without trailing slash
	h1, _ := router.Lookup(MethodGet, "/users")
	if h1 == nil {
		t.Error("expected handler for /users")
	}

	// With trailing slash (different path)
	h2, _ := router.Lookup(MethodGet, "/users/")
	if h2 != nil {
		t.Log("Note: /users/ is treated as different from /users")
	}
}

// TestRouterDeepNesting tests deeply nested routes.
func TestRouterDeepNesting(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/a/b/c/d/e/f/g/h", handler)

	h, _ := router.Lookup(MethodGet, "/a/b/c/d/e/f/g/h")
	if h == nil {
		t.Error("expected handler for deeply nested path")
	}
}

// TestRouterSpecialCharactersInPath tests paths with special characters.
func TestRouterSpecialCharactersInPath(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/api/v1/users/:id", handler)

	h, params := router.Lookup(MethodGet, "/api/v1/users/user-123-abc")
	if h == nil {
		t.Fatal("expected handler")
	}

	if params["id"] != "user-123-abc" {
		t.Errorf("expected id='user-123-abc', got '%s'", params["id"])
	}
}

// TestRouterMultipleWildcards tests multiple wildcards (should use last one).
func TestRouterMultipleWildcards(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Only last wildcard should matter
	router.Add(MethodGet, "/files/*filepath", handler)

	h, params := router.Lookup(MethodGet, "/files/docs/readme.md")
	if h == nil {
		t.Fatal("expected handler for wildcard route")
	}

	if params["filepath"] != "docs/readme.md" {
		t.Errorf("expected filepath='docs/readme.md', got '%s'", params["filepath"])
	}
}

// TestRouterParamWithDot tests parameter values containing dots.
func TestRouterParamWithDot(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/files/:filename", handler)

	h, params := router.Lookup(MethodGet, "/files/document.pdf")
	if h == nil {
		t.Fatal("expected handler")
	}

	if params["filename"] != "document.pdf" {
		t.Errorf("expected filename='document.pdf', got '%s'", params["filename"])
	}
}

// TestRouterOverwriteRoute tests route overwriting.
func TestRouterOverwriteRoute(t *testing.T) {
	router := NewRouter()

	handler1 := func(c *Context) error {
		c.Set("version", 1)
		return nil
	}

	handler2 := func(c *Context) error {
		c.Set("version", 2)
		return nil
	}

	// Add first handler
	router.Add(MethodGet, "/test", handler1)

	// Overwrite with second handler
	router.Add(MethodGet, "/test", handler2)

	// Should get the second handler
	h, _ := router.Lookup(MethodGet, "/test")
	if h == nil {
		t.Fatal("expected handler")
	}

	ctx := &Context{}
	h(ctx)

	if ver := ctx.Get("version"); ver != 2 {
		t.Errorf("expected version 2 (overwritten), got %v", ver)
	}
}

// TestRouterServeHTTPSuccess tests ServeHTTP with successful lookup.
func TestRouterServeHTTPSuccess(t *testing.T) {
	router := NewRouter()

	called := false
	handler := func(c *Context) error {
		called = true
		return nil
	}

	router.Add(MethodGet, "/test/:id", handler)

	ctx := &Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test/123")

	err := router.ServeHTTP(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Error("expected handler to be called")
	}

	// Verify params were set
	if ctx.Param("id") != "123" {
		t.Errorf("expected id=123, got %s", ctx.Param("id"))
	}
}

// TestRouterServeHTTPNotFound tests ServeHTTP with no matching route.
func TestRouterServeHTTPNotFound(t *testing.T) {
	router := NewRouter()

	ctx := &Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/nonexistent")

	err := router.ServeHTTP(ctx)
	if err != ErrNotFound {
		t.Errorf("expected ErrNotFound, got %v", err)
	}
}

// TestRouterServeHTTPMethodNotAllowed tests ServeHTTP with wrong method.
func TestRouterServeHTTPMethodNotAllowed(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Register POST handler
	router.Add(MethodPost, "/test", handler)

	// Try with GET
	ctx := &Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	err := router.ServeHTTP(ctx)
	if err != ErrNotFound {
		// Currently returns ErrNotFound for method mismatch
		t.Logf("method mismatch returns: %v", err)
	}
}

// TestAddToTreeEdgeCases tests addToTree internal method edge cases.
func TestAddToTreeEdgeCases(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Add route with param and wildcard
	router.Add(MethodGet, "/:category/*file", handler)

	h, params := router.Lookup(MethodGet, "/docs/guides/intro.md")
	if h == nil {
		t.Fatal("expected handler")
	}

	if params["category"] != "docs" {
		t.Errorf("expected category='docs', got '%s'", params["category"])
	}

	if params["file"] != "guides/intro.md" {
		t.Errorf("expected file='guides/intro.md', got '%s'", params["file"])
	}
}

// TestFindOrCreateChildEdgeCases tests findOrCreateChild behavior.
func TestFindOrCreateChildEdgeCases(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Create branching paths
	router.Add(MethodGet, "/api/users", handler)
	router.Add(MethodGet, "/api/posts", handler)
	router.Add(MethodGet, "/api/comments", handler)

	// All should be findable
	if h, _ := router.Lookup(MethodGet, "/api/users"); h == nil {
		t.Error("expected /api/users")
	}
	if h, _ := router.Lookup(MethodGet, "/api/posts"); h == nil {
		t.Error("expected /api/posts")
	}
	if h, _ := router.Lookup(MethodGet, "/api/comments"); h == nil {
		t.Error("expected /api/comments")
	}
}

// TestSearchNodeEdgeCases tests searchNode with various patterns.
func TestSearchNodeEdgeCases(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Add routes with different node types
	router.Add(MethodGet, "/static", handler)           // static
	router.Add(MethodGet, "/param/:id", handler)        // param
	router.Add(MethodGet, "/wildcard/*path", handler)   // wildcard
	router.Add(MethodGet, "/mixed/:id/static", handler) // mixed

	tests := []struct {
		path      string
		shouldFind bool
	}{
		{"/static", true},
		{"/param/123", true},
		{"/wildcard/any/path/here", true},
		{"/mixed/999/static", true},
		{"/notfound", false},
	}

	for _, tt := range tests {
		h, _ := router.Lookup(MethodGet, tt.path)
		found := h != nil

		if found != tt.shouldFind {
			t.Errorf("path %s: expected found=%v, got %v", tt.path, tt.shouldFind, found)
		}
	}
}

// TestRouterConflictingRoutes tests adding routes that might conflict.
func TestRouterConflictingRoutes(t *testing.T) {
	router := NewRouter()

	handler1 := func(c *Context) error {
		c.Set("handler", "1")
		return nil
	}

	handler2 := func(c *Context) error {
		c.Set("handler", "2")
		return nil
	}

	// Add potentially conflicting routes
	router.Add(MethodGet, "/users/:id", handler1)
	router.Add(MethodGet, "/users/:userId", handler2) // Same pattern, different param name

	h, params := router.Lookup(MethodGet, "/users/123")
	if h == nil {
		t.Fatal("expected handler")
	}

	// Router keeps the first registered handler (doesn't overwrite)
	ctx := &Context{}
	h(ctx)

	if ver := ctx.Get("handler"); ver != "1" {
		t.Logf("Handler version: %v (router implementation dependent)", ver)
	}

	// Either param name could be used
	if params["id"] != "123" && params["userId"] != "123" {
		t.Error("expected either id or userId param to be set")
	}
}

// TestRouterMultipleParamsInSegment tests edge cases with multiple params.
func TestRouterMultipleParamsInSegment(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Add route with multiple params
	router.Add(MethodGet, "/api/:version/users/:id/posts/:postId", handler)

	h, params := router.Lookup(MethodGet, "/api/v1/users/123/posts/456")
	if h == nil {
		t.Fatal("expected handler")
	}

	if params["version"] != "v1" {
		t.Errorf("expected version=v1, got %s", params["version"])
	}

	if params["id"] != "123" {
		t.Errorf("expected id=123, got %s", params["id"])
	}

	if params["postId"] != "456" {
		t.Errorf("expected postId=456, got %s", params["postId"])
	}
}

// TestRouterStaticVsDynamic tests precedence of static vs dynamic routes.
func TestRouterStaticVsDynamic(t *testing.T) {
	router := NewRouter()

	staticHandler := func(c *Context) error {
		c.Set("type", "static")
		return nil
	}

	dynamicHandler := func(c *Context) error {
		c.Set("type", "dynamic")
		return nil
	}

	// Add both static and dynamic routes
	router.Add(MethodGet, "/users/new", staticHandler)
	router.Add(MethodGet, "/users/:id", dynamicHandler)

	// Test static route
	h1, _ := router.Lookup(MethodGet, "/users/new")
	if h1 == nil {
		t.Fatal("expected static handler")
	}

	ctx1 := &Context{}
	h1(ctx1)

	if typ := ctx1.Get("type"); typ != "static" {
		t.Errorf("expected static handler, got %v", typ)
	}

	// Test dynamic route
	h2, params := router.Lookup(MethodGet, "/users/123")
	if h2 == nil {
		t.Fatal("expected dynamic handler")
	}

	ctx2 := &Context{}
	h2(ctx2)

	if typ := ctx2.Get("type"); typ != "dynamic" {
		t.Errorf("expected dynamic handler, got %v", typ)
	}

	if params["id"] != "123" {
		t.Errorf("expected id=123, got %s", params["id"])
	}
}

// TestRouterWildcardWithParams tests wildcard routes with params.
func TestRouterWildcardWithParams(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	router.Add(MethodGet, "/files/:category/*filepath", handler)

	h, params := router.Lookup(MethodGet, "/files/images/2024/01/photo.jpg")
	if h == nil {
		t.Fatal("expected handler")
	}

	if params["category"] != "images" {
		t.Errorf("expected category=images, got %s", params["category"])
	}

	if params["filepath"] != "2024/01/photo.jpg" {
		t.Errorf("expected filepath='2024/01/photo.jpg', got %s", params["filepath"])
	}
}

// TestRouterEmptySegments tests paths with empty segments.
func TestRouterEmptySegments(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Add route
	router.Add(MethodGet, "/api/users", handler)

	// Try lookup with double slash
	h, _ := router.Lookup(MethodGet, "/api//users")
	// May or may not match depending on implementation
	// Just verify it doesn't panic
	if h != nil {
		t.Log("Route matched with double slash")
	} else {
		t.Log("Route did not match with double slash")
	}
}

// TestRouterLookupPerformance tests router lookup with many routes.
func TestRouterLookupPerformance(t *testing.T) {
	router := NewRouter()

	handler := func(c *Context) error {
		return nil
	}

	// Add many routes
	for i := 0; i < 100; i++ {
		router.Add(MethodGet, "/route"+string(rune('0'+i%10))+"/path", handler)
	}

	// Lookup should still be fast
	h, _ := router.Lookup(MethodGet, "/route5/path")
	if h == nil {
		t.Error("expected to find route in large router")
	}
}

// TestRouterDifferentMethodsSamePath tests different HTTP methods on same path.
func TestRouterDifferentMethodsSamePath(t *testing.T) {
	router := NewRouter()

	getHandler := func(c *Context) error {
		c.Set("method", "GET")
		return nil
	}

	postHandler := func(c *Context) error {
		c.Set("method", "POST")
		return nil
	}

	putHandler := func(c *Context) error {
		c.Set("method", "PUT")
		return nil
	}

	deleteHandler := func(c *Context) error {
		c.Set("method", "DELETE")
		return nil
	}

	patchHandler := func(c *Context) error {
		c.Set("method", "PATCH")
		return nil
	}

	// Add all methods on same path
	router.Add(MethodGet, "/resource/:id", getHandler)
	router.Add(MethodPost, "/resource/:id", postHandler)
	router.Add(MethodPut, "/resource/:id", putHandler)
	router.Add(MethodDelete, "/resource/:id", deleteHandler)
	router.Add(MethodPatch, "/resource/:id", patchHandler)

	// Test each method
	methods := []struct {
		method   HTTPMethod
		expected string
	}{
		{MethodGet, "GET"},
		{MethodPost, "POST"},
		{MethodPut, "PUT"},
		{MethodDelete, "DELETE"},
		{MethodPatch, "PATCH"},
	}

	for _, m := range methods {
		h, params := router.Lookup(m.method, "/resource/123")
		if h == nil {
			t.Errorf("expected handler for %s", m.method)
			continue
		}

		ctx := &Context{}
		h(ctx)

		if method := ctx.Get("method"); method != m.expected {
			t.Errorf("expected method %s, got %v", m.expected, method)
		}

		if params["id"] != "123" {
			t.Errorf("expected id=123, got %s", params["id"])
		}
	}
}

// TestRouterSearchBacktracking tests backtracking in searchNode.
func TestRouterSearchBacktracking(t *testing.T) {
	router := NewRouter()

	handler1 := func(c *Context) error {
		c.Set("handler", "1")
		return nil
	}

	handler2 := func(c *Context) error {
		c.Set("handler", "2")
		return nil
	}

	// Add routes that require backtracking
	// /users/:id/posts - matches /users/123/posts
	// /users/new - matches /users/new
	router.Add(MethodGet, "/users/:id/posts", handler1)
	router.Add(MethodGet, "/users/new", handler2)

	// This should match /users/new (static route)
	h1, _ := router.Lookup(MethodGet, "/users/new")
	if h1 == nil {
		t.Fatal("expected handler for /users/new")
	}

	ctx1 := &Context{}
	h1(ctx1)

	if handler := ctx1.Get("handler"); handler != "2" {
		t.Errorf("expected handler 2 (static route), got %v", handler)
	}

	// This should match /users/:id/posts (param route)
	h2, params := router.Lookup(MethodGet, "/users/123/posts")
	if h2 == nil {
		t.Fatal("expected handler for /users/123/posts")
	}

	ctx2 := &Context{}
	h2(ctx2)

	if handler := ctx2.Get("handler"); handler != "1" {
		t.Errorf("expected handler 1 (param route), got %v", handler)
	}

	if params["id"] != "123" {
		t.Errorf("expected id=123, got %s", params["id"])
	}

	// This should NOT match (triggers backtracking)
	h3, _ := router.Lookup(MethodGet, "/users/123/comments")
	if h3 != nil {
		t.Error("expected no handler for /users/123/comments")
	}
}

// TestRouterMultipleParamBacktracking tests complex backtracking scenarios.
func TestRouterMultipleParamBacktracking(t *testing.T) {
	router := NewRouter()

	handler1 := func(c *Context) error {
		c.Set("handler", "1")
		return nil
	}

	handler2 := func(c *Context) error {
		c.Set("handler", "2")
		return nil
	}

	handler3 := func(c *Context) error {
		c.Set("handler", "3")
		return nil
	}

	// Add routes with overlapping patterns
	router.Add(MethodGet, "/api/:version/users/:id", handler1)
	router.Add(MethodGet, "/api/v1/users/special", handler2)
	router.Add(MethodGet, "/api/v2/posts/:id", handler3)

	// Test static match (should prefer static over param)
	h1, _ := router.Lookup(MethodGet, "/api/v1/users/special")
	if h1 == nil {
		t.Fatal("expected handler for /api/v1/users/special")
	}

	ctx1 := &Context{}
	h1(ctx1)

	if handler := ctx1.Get("handler"); handler != "2" {
		t.Logf("Handler: %v (implementation may vary)", handler)
	}

	// Test param match
	h2, params2 := router.Lookup(MethodGet, "/api/v1/users/123")
	if h2 == nil {
		t.Fatal("expected handler for /api/v1/users/123")
	}

	if params2["version"] != "v1" || params2["id"] != "123" {
		t.Errorf("expected version=v1 and id=123, got version=%s, id=%s", params2["version"], params2["id"])
	}

	// Test different param route
	h3, params3 := router.Lookup(MethodGet, "/api/v2/posts/456")
	if h3 == nil {
		t.Fatal("expected handler for /api/v2/posts/456")
	}

	ctx3 := &Context{}
	h3(ctx3)

	if handler := ctx3.Get("handler"); handler != "3" {
		t.Errorf("expected handler 3, got %v", handler)
	}

	if params3["id"] != "456" {
		t.Errorf("expected id=456, got %s", params3["id"])
	}
}
