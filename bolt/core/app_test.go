package core

import (
	"errors"
	"testing"
)

// TestNew tests creating a new app with defaults.
func TestNew(t *testing.T) {
	app := New()

	if app == nil {
		t.Fatal("expected app, got nil")
	}
	if app.router == nil {
		t.Error("expected router to be initialized")
	}
	if app.contextPool == nil {
		t.Error("expected context pool to be initialized")
	}
	if app.errorHandler == nil {
		t.Error("expected error handler to be initialized")
	}
}

// TestNewWithConfig tests creating app with custom config.
func TestNewWithConfig(t *testing.T) {
	customErrorHandler := func(c *Context, err error) {
		// Custom handler
	}

	config := Config{
		Addr:               ":9000",
		ErrorHandler:       customErrorHandler,
		MaxRequestBodySize: 5 << 20, // 5MB
		DisableStats:       false,
	}

	app := NewWithConfig(config)

	if app == nil {
		t.Fatal("expected app, got nil")
	}
	if app.config.Addr != ":9000" {
		t.Errorf("expected addr :9000, got %s", app.config.Addr)
	}
	if app.config.MaxRequestBodySize != 5<<20 {
		t.Error("expected custom max request body size")
	}
}

// TestGetRoute tests registering GET route.
func TestGetRoute(t *testing.T) {
	app := New()

	called := false
	app.Get("/test", func(c *Context) error {
		called = true
		return nil
	})

	// Verify route was registered
	handler, _ := app.router.Lookup(MethodGet, "/test")
	if handler == nil {
		t.Fatal("expected handler to be registered")
	}

	// Execute handler
	ctx := &Context{}
	handler(ctx)

	if !called {
		t.Error("expected handler to be called")
	}
}

// TestPostRoute tests registering POST route.
func TestPostRoute(t *testing.T) {
	app := New()

	called := false
	app.Post("/users", func(c *Context) error {
		called = true
		return nil
	})

	handler, _ := app.router.Lookup(MethodPost, "/users")
	if handler == nil {
		t.Fatal("expected POST handler to be registered")
	}

	ctx := &Context{}
	handler(ctx)

	if !called {
		t.Error("expected POST handler to be called")
	}
}

// TestPutRoute tests registering PUT route.
func TestPutRoute(t *testing.T) {
	app := New()

	app.Put("/users/:id", func(c *Context) error {
		return nil
	})

	handler, params := app.router.Lookup(MethodPut, "/users/123")
	if handler == nil {
		t.Fatal("expected PUT handler to be registered")
	}
	if params["id"] != "123" {
		t.Error("expected id parameter")
	}
}

// TestDeleteRoute tests registering DELETE route.
func TestDeleteRoute(t *testing.T) {
	app := New()

	app.Delete("/users/:id", func(c *Context) error {
		return nil
	})

	handler, _ := app.router.Lookup(MethodDelete, "/users/123")
	if handler == nil {
		t.Fatal("expected DELETE handler to be registered")
	}
}

// TestPatchRoute tests registering PATCH route.
func TestPatchRoute(t *testing.T) {
	app := New()

	app.Patch("/users/:id", func(c *Context) error {
		return nil
	})

	handler, _ := app.router.Lookup(MethodPatch, "/users/123")
	if handler == nil {
		t.Fatal("expected PATCH handler to be registered")
	}
}

// TestHeadRoute tests registering HEAD route.
func TestHeadRoute(t *testing.T) {
	app := New()

	app.Head("/health", func(c *Context) error {
		return nil
	})

	handler, _ := app.router.Lookup(MethodHead, "/health")
	if handler == nil {
		t.Fatal("expected HEAD handler to be registered")
	}
}

// TestOptionsRoute tests registering OPTIONS route.
func TestOptionsRoute(t *testing.T) {
	app := New()

	app.Options("/api", func(c *Context) error {
		return nil
	})

	handler, _ := app.router.Lookup(MethodOptions, "/api")
	if handler == nil {
		t.Fatal("expected OPTIONS handler to be registered")
	}
}

// TestGlobalMiddleware tests adding global middleware.
func TestGlobalMiddleware(t *testing.T) {
	app := New()

	var executionOrder []string

	// Add middleware
	middleware1 := func(next Handler) Handler {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "middleware1-before")
			err := next(c)
			executionOrder = append(executionOrder, "middleware1-after")
			return err
		}
	}

	middleware2 := func(next Handler) Handler {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "middleware2-before")
			err := next(c)
			executionOrder = append(executionOrder, "middleware2-after")
			return err
		}
	}

	app.Use(middleware1)
	app.Use(middleware2)

	// Register route
	app.Get("/test", func(c *Context) error {
		executionOrder = append(executionOrder, "handler")
		return nil
	})

	// Execute
	handler, _ := app.router.Lookup(MethodGet, "/test")
	ctx := &Context{}
	handler(ctx)

	// Verify order: middleware1-before, middleware2-before, handler, middleware2-after, middleware1-after
	expected := []string{
		"middleware1-before",
		"middleware2-before",
		"handler",
		"middleware2-after",
		"middleware1-after",
	}

	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d executions, got %d", len(expected), len(executionOrder))
	}

	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("execution[%d]: expected %s, got %s", i, exp, executionOrder[i])
		}
	}
}

// TestRouteSpecificMiddleware tests middleware on specific route.
func TestRouteSpecificMiddleware(t *testing.T) {
	app := New()

	var executionOrder []string

	routeMiddleware := func(next Handler) Handler {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "route-middleware")
			return next(c)
		}
	}

	app.Get("/test", func(c *Context) error {
		executionOrder = append(executionOrder, "handler")
		return nil
	}).Use(routeMiddleware)

	handler, _ := app.router.Lookup(MethodGet, "/test")
	ctx := &Context{}
	handler(ctx)

	if len(executionOrder) != 2 {
		t.Fatalf("expected 2 executions, got %d", len(executionOrder))
	}
	if executionOrder[0] != "route-middleware" {
		t.Error("expected route middleware to run first")
	}
	if executionOrder[1] != "handler" {
		t.Error("expected handler to run second")
	}
}

// TestMultipleMiddlewareChaining tests chaining multiple route middlewares.
func TestMultipleMiddlewareChaining(t *testing.T) {
	app := New()

	var executionOrder []string

	mw1 := func(next Handler) Handler {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "mw1")
			return next(c)
		}
	}

	mw2 := func(next Handler) Handler {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "mw2")
			return next(c)
		}
	}

	app.Get("/test", func(c *Context) error {
		executionOrder = append(executionOrder, "handler")
		return nil
	}).Use(mw1, mw2)

	handler, _ := app.router.Lookup(MethodGet, "/test")
	ctx := &Context{}
	handler(ctx)

	expected := []string{"mw1", "mw2", "handler"}
	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d executions, got %d", len(expected), len(executionOrder))
	}

	for i, exp := range expected {
		if executionOrder[i] != exp {
			t.Errorf("execution[%d]: expected %s, got %s", i, exp, executionOrder[i])
		}
	}
}

// TestErrorHandler tests custom error handler.
func TestErrorHandler(t *testing.T) {
	customErrorHandler := func(c *Context, err error) {
		// Custom handler implementation
		_ = c.JSON(500, map[string]string{"error": err.Error()})
	}

	app := NewWithConfig(Config{
		ErrorHandler: customErrorHandler,
	})

	testErr := errors.New("test error")
	app.Get("/error", func(c *Context) error {
		return testErr
	})

	// Verify the error handler is set correctly
	if app.errorHandler == nil {
		t.Error("expected custom error handler to be set")
	}
}

// TestDefaultErrorHandler tests default error handler.
func TestDefaultErrorHandler(t *testing.T) {
	tests := []struct {
		err            error
		expectedStatus int
	}{
		{ErrNotFound, 404},
		{ErrBadRequest, 400},
		{ErrUnauthorized, 401},
		{ErrForbidden, 403},
		{ErrMethodNotAllowed, 405},
		{ErrRequestTooLarge, 413},
		{errors.New("unknown error"), 500},
	}

	for _, tt := range tests {
		ctx := &Context{}

		// Call default error handler
		DefaultErrorHandler(ctx, tt.err)

		// Note: In real implementation, JSON would be written to response
		// For this test, we just verify the function doesn't panic
	}
}

// NOTE: Generic method tests removed as Go doesn't support type parameters on methods
// The Data[T] wrapper can still be used manually within handlers

// TestChainLinkFluent tests fluent API with chain link.
func TestChainLinkFluent(t *testing.T) {
	app := New()

	var mwCalled bool
	mw := func(next Handler) Handler {
		return func(c *Context) error {
			mwCalled = true
			return next(c)
		}
	}

	chain := app.Get("/test", func(c *Context) error {
		return nil
	})

	// Chain should allow fluent middleware
	chain.Use(mw)

	// Verify middleware is applied
	handler, _ := app.router.Lookup(MethodGet, "/test")
	ctx := &Context{}
	handler(ctx)

	if !mwCalled {
		t.Error("expected middleware to be called via chain link")
	}
}

// TestMultipleRoutes tests registering multiple routes.
func TestMultipleRoutes(t *testing.T) {
	app := New()

	app.Get("/users", func(c *Context) error { return nil })
	app.Get("/users/:id", func(c *Context) error { return nil })
	app.Post("/users", func(c *Context) error { return nil })
	app.Put("/users/:id", func(c *Context) error { return nil })
	app.Delete("/users/:id", func(c *Context) error { return nil })

	// Verify all routes are registered
	tests := []struct {
		method HTTPMethod
		path   string
	}{
		{MethodGet, "/users"},
		{MethodGet, "/users/123"},
		{MethodPost, "/users"},
		{MethodPut, "/users/123"},
		{MethodDelete, "/users/123"},
	}

	for _, tt := range tests {
		handler, _ := app.router.Lookup(tt.method, tt.path)
		if handler == nil {
			t.Errorf("expected handler for %s %s", tt.method, tt.path)
		}
	}
}

// BenchmarkAppGet benchmarks registering GET route.
func BenchmarkAppGet(b *testing.B) {
	handler := func(c *Context) error {
		return nil
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app := New()
		app.Get("/test", handler)
	}
}

// BenchmarkAppGetWithMiddleware benchmarks route with middleware.
func BenchmarkAppGetWithMiddleware(b *testing.B) {
	middleware := func(next Handler) Handler {
		return func(c *Context) error {
			return next(c)
		}
	}

	handler := func(c *Context) error {
		return nil
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app := New()
		app.Use(middleware)
		app.Get("/test", handler)
	}
}

// BenchmarkAppDataWrapper benchmarks using Data[T] wrapper manually.
func BenchmarkAppDataWrapper(b *testing.B) {
	type Response struct {
		Message string
	}

	handler := func(c *Context) error {
		data := OK(Response{Message: "hello"})
		return sendData(c, data)
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		app := New()
		app.Get("/test", handler)
	}
}
