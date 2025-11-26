package core

import (
	"context"
	"errors"
	"testing"
	"time"
)

// TestNewWithConfigDefaults tests NewWithConfig fills in ErrorHandler default.
func TestNewWithConfigDefaults(t *testing.T) {
	config := Config{
		Addr: ":9000",
		// Leave ErrorHandler nil - should be filled with default
	}

	app := NewWithConfig(config)

	if app == nil {
		t.Fatal("expected app, got nil")
	}

	// ErrorHandler should default to DefaultErrorHandler
	if app.errorHandler == nil {
		t.Error("expected default error handler to be set")
	}

	// Note: NewWithConfig does not set other defaults
	// Only ErrorHandler is defaulted
}

// TestNewWithConfigAllFields tests NewWithConfig with all fields set.
func TestNewWithConfigAllFields(t *testing.T) {
	customErrorHandler := func(c *Context, err error) {
		_ = c.JSON(500, map[string]string{"error": "custom"})
	}

	config := Config{
		Addr:               ":3000",
		ErrorHandler:       customErrorHandler,
		MaxRequestBodySize: 1024,
		EnableLogging:      true,
		DisableStats:       false,
	}

	app := NewWithConfig(config)

	if app.config.Addr != ":3000" {
		t.Errorf("expected addr :3000, got %s", app.config.Addr)
	}

	if app.config.MaxRequestBodySize != 1024 {
		t.Errorf("expected body size 1024, got %d", app.config.MaxRequestBodySize)
	}

	if !app.config.EnableLogging {
		t.Error("expected EnableLogging to be true")
	}

	if app.config.DisableStats {
		t.Error("expected DisableStats to be false")
	}
}

// TestShutdownWithContext tests Shutdown with context.
func TestShutdownWithContext(t *testing.T) {
	app := New()

	// Set up a mock server
	app.Get("/test", func(c *Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Create context with timeout
	ctx, cancel := context.WithTimeout(context.Background(), 100*time.Millisecond)
	defer cancel()

	// Shutdown should not panic even without running server
	err := app.Shutdown(ctx)
	if err != nil {
		// Expected - server not running
		t.Logf("shutdown returned: %v (expected, no server running)", err)
	}
}

// TestShutdownWithCancelledContext tests Shutdown with already-cancelled context.
func TestShutdownWithCancelledContext(t *testing.T) {
	app := New()

	ctx, cancel := context.WithCancel(context.Background())
	cancel() // Cancel immediately

	err := app.Shutdown(ctx)
	if err != nil {
		t.Logf("shutdown with cancelled context: %v", err)
	}
}

// TestHandleShockwaveRequestFlow tests handleShockwaveRequest lifecycle.
func TestHandleShockwaveRequestFlow(t *testing.T) {
	app := New()

	handlerCalled := false
	app.Get("/api/test", func(c *Context) error {
		handlerCalled = true
		return c.JSON(200, map[string]string{"message": "success"})
	})

	// We can't easily test with real Shockwave, but we can test the route setup
	handler, _ := app.router.Lookup(MethodGet, "/api/test")
	if handler == nil {
		t.Fatal("expected handler to be registered")
	}

	ctx := &Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api/test")

	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Error("expected handler to be called")
	}
}

// TestAppWithMiddlewareChain tests complex middleware chains.
func TestAppWithMiddlewareChain(t *testing.T) {
	app := New()

	var executionLog []string

	// Global middleware 1
	app.Use(func(next Handler) Handler {
		return func(c *Context) error {
			executionLog = append(executionLog, "global1-before")
			err := next(c)
			executionLog = append(executionLog, "global1-after")
			return err
		}
	})

	// Global middleware 2
	app.Use(func(next Handler) Handler {
		return func(c *Context) error {
			executionLog = append(executionLog, "global2-before")
			err := next(c)
			executionLog = append(executionLog, "global2-after")
			return err
		}
	})

	// Route-specific middleware
	routeMw := func(next Handler) Handler {
		return func(c *Context) error {
			executionLog = append(executionLog, "route-before")
			err := next(c)
			executionLog = append(executionLog, "route-after")
			return err
		}
	}

	app.Get("/test", func(c *Context) error {
		executionLog = append(executionLog, "handler")
		return nil
	}).Use(routeMw)

	// Execute
	handler, _ := app.router.Lookup(MethodGet, "/test")
	ctx := &Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	_ = handler(ctx)

	// Verify execution order
	// Route middleware wraps global middleware, so route executes first
	expected := []string{
		"route-before",
		"global1-before",
		"global2-before",
		"handler",
		"global2-after",
		"global1-after",
		"route-after",
	}

	if len(executionLog) != len(expected) {
		t.Fatalf("expected %d steps, got %d: %v", len(expected), len(executionLog), executionLog)
	}

	for i, step := range expected {
		if executionLog[i] != step {
			t.Errorf("step %d: expected %s, got %s", i, step, executionLog[i])
		}
	}
}

// TestAppErrorHandlerFlow tests error handler invocation.
func TestAppErrorHandlerFlow(t *testing.T) {
	errorHandlerCalled := false
	customErrorHandler := func(c *Context, err error) {
		errorHandlerCalled = true
		_ = c.JSON(500, map[string]string{"error": err.Error()})
	}

	app := NewWithConfig(Config{
		ErrorHandler: customErrorHandler,
	})

	testErr := errors.New("test error")
	app.Get("/error", func(c *Context) error {
		return testErr
	})

	handler, _ := app.router.Lookup(MethodGet, "/error")
	ctx := &Context{}

	// Execute handler which returns error
	err := handler(ctx)
	if err != testErr {
		t.Errorf("expected test error, got %v", err)
	}

	// In real scenario, router.ServeHTTP would call errorHandler
	// Let's simulate that
	app.errorHandler(ctx, err)

	if !errorHandlerCalled {
		t.Error("expected error handler to be called")
	}
}

// TestAppMultipleRoutesPerPath tests registering multiple HTTP methods for same path.
func TestAppMultipleRoutesPerPath(t *testing.T) {
	app := New()

	getCalled := false
	postCalled := false
	putCalled := false

	app.Get("/resource", func(c *Context) error {
		getCalled = true
		return nil
	})

	app.Post("/resource", func(c *Context) error {
		postCalled = true
		return nil
	})

	app.Put("/resource", func(c *Context) error {
		putCalled = true
		return nil
	})

	// Test GET
	h1, _ := app.router.Lookup(MethodGet, "/resource")
	ctx1 := &Context{}
	_ = h1(ctx1)
	if !getCalled {
		t.Error("GET handler not called")
	}

	// Test POST
	h2, _ := app.router.Lookup(MethodPost, "/resource")
	ctx2 := &Context{}
	_ = h2(ctx2)
	if !postCalled {
		t.Error("POST handler not called")
	}

	// Test PUT
	h3, _ := app.router.Lookup(MethodPut, "/resource")
	ctx3 := &Context{}
	_ = h3(ctx3)
	if !putCalled {
		t.Error("PUT handler not called")
	}
}

// TestAppRESTfulAPI tests a complete RESTful API setup.
func TestAppRESTfulAPI(t *testing.T) {
	app := New()

	// Standard RESTful routes
	app.Get("/users", func(c *Context) error {
		return c.JSON(200, []string{"user1", "user2"})
	})

	app.Get("/users/:id", func(c *Context) error {
		return c.JSON(200, map[string]string{"id": c.Param("id")})
	})

	app.Post("/users", func(c *Context) error {
		return c.JSON(201, map[string]string{"created": "true"})
	})

	app.Put("/users/:id", func(c *Context) error {
		return c.JSON(200, map[string]string{"updated": c.Param("id")})
	})

	app.Delete("/users/:id", func(c *Context) error {
		return c.NoContent()
	})

	// Test all routes
	tests := []struct {
		method HTTPMethod
		path   string
		status int
	}{
		{MethodGet, "/users", 200},
		{MethodGet, "/users/123", 200},
		{MethodPost, "/users", 201},
		{MethodPut, "/users/456", 200},
		{MethodDelete, "/users/789", 204},
	}

	for _, tt := range tests {
		handler, _ := app.router.Lookup(tt.method, tt.path)
		if handler == nil {
			t.Errorf("%s %s: handler not found", tt.method, tt.path)
			continue
		}

		ctx := &Context{}
		ctx.SetMethod(string(tt.method))
		ctx.SetPath(tt.path)

		err := handler(ctx)
		if err != nil {
			t.Errorf("%s %s: unexpected error: %v", tt.method, tt.path, err)
		}

		if ctx.StatusCode() != tt.status {
			t.Errorf("%s %s: expected status %d, got %d", tt.method, tt.path, tt.status, ctx.StatusCode())
		}
	}
}

// TestAppNestedRoutes tests nested route groups.
func TestAppNestedRoutes(t *testing.T) {
	app := New()

	// API v1
	app.Get("/api/v1/users", func(c *Context) error {
		return c.JSON(200, map[string]string{"version": "v1"})
	})

	// API v2
	app.Get("/api/v2/users", func(c *Context) error {
		return c.JSON(200, map[string]string{"version": "v2"})
	})

	// Test v1
	h1, _ := app.router.Lookup(MethodGet, "/api/v1/users")
	ctx1 := &Context{}
	_ = h1(ctx1)

	// Test v2
	h2, _ := app.router.Lookup(MethodGet, "/api/v2/users")
	ctx2 := &Context{}
	_ = h2(ctx2)

	// Both should work independently
	if h1 == nil || h2 == nil {
		t.Error("expected both versioned handlers to exist")
	}
}

// TestAppWithParameterValidation tests parameter extraction and validation.
func TestAppWithParameterValidation(t *testing.T) {
	app := New()

	app.Get("/users/:id/posts/:postId", func(c *Context) error {
		userID := c.Param("id")
		postID := c.Param("postId")

		if userID == "" || postID == "" {
			return c.JSON(400, map[string]string{"error": "missing params"})
		}

		return c.JSON(200, map[string]string{
			"userId": userID,
			"postId": postID,
		})
	})

	// Use ServeHTTP to properly set params
	ctx := &Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/users/user123/posts/post456")

	err := app.router.ServeHTTP(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}

	// Verify params were extracted
	if ctx.Param("id") != "user123" {
		t.Errorf("expected id='user123', got '%s'", ctx.Param("id"))
	}

	if ctx.Param("postId") != "post456" {
		t.Errorf("expected postId='post456', got '%s'", ctx.Param("postId"))
	}
}

// TestListenInitialization tests Listen setup.
func TestListenInitialization(t *testing.T) {
	app := New()

	app.Get("/health", func(c *Context) error {
		return c.JSON(200, map[string]string{"status": "healthy"})
	})

	// We can't actually Listen without blocking or having a real server
	// But we can verify the route is registered
	handler, _ := app.router.Lookup(MethodGet, "/health")
	if handler == nil {
		t.Error("expected health check handler to be registered")
	}
}

// TestConfigShockwaveSettings tests config passed to Shockwave.
func TestConfigShockwaveSettings(t *testing.T) {
	config := Config{
		Addr:               ":8080",
		MaxRequestBodySize: 5 * 1024 * 1024, // 5MB
		DisableStats:       true,
	}

	app := NewWithConfig(config)

	if app.config.MaxRequestBodySize != 5*1024*1024 {
		t.Error("expected max request body size to be set")
	}

	if !app.config.DisableStats {
		t.Error("expected stats to be disabled")
	}
}
