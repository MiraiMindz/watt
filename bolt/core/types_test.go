package core

import (
	"errors"
	"testing"
)

// TestCommonErrors tests predefined error constants.
func TestCommonErrors(t *testing.T) {
	errors := []error{
		ErrNotFound,
		ErrBadRequest,
		ErrUnauthorized,
		ErrForbidden,
		ErrMethodNotAllowed,
		ErrRequestTooLarge,
		ErrInternalServerError,
	}

	for _, err := range errors {
		if err == nil {
			t.Error("expected error to be defined")
		}
		if err.Error() == "" {
			t.Error("expected error to have message")
		}
	}
}

// TestChainLinkUse tests ChainLink middleware chaining.
func TestChainLinkUse(t *testing.T) {
	app := New()

	executionOrder := []string{}

	middleware1 := func(next Handler) Handler {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "m1-before")
			err := next(c)
			executionOrder = append(executionOrder, "m1-after")
			return err
		}
	}

	middleware2 := func(next Handler) Handler {
		return func(c *Context) error {
			executionOrder = append(executionOrder, "m2-before")
			err := next(c)
			executionOrder = append(executionOrder, "m2-after")
			return err
		}
	}

	handler := func(c *Context) error {
		executionOrder = append(executionOrder, "handler")
		return nil
	}

	// Chain middleware
	app.Get("/test", handler).Use(middleware1).Use(middleware2)

	// Execute handler
	ctx := &Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	err := app.router.ServeHTTP(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Verify execution order
	expected := []string{
		"m2-before",
		"m1-before",
		"handler",
		"m1-after",
		"m2-after",
	}

	if len(executionOrder) != len(expected) {
		t.Fatalf("expected %d execution steps, got %d", len(expected), len(executionOrder))
	}

	for i, step := range expected {
		if executionOrder[i] != step {
			t.Errorf("step %d: expected %s, got %s", i, step, executionOrder[i])
		}
	}
}

// TestChainLinkMultipleMiddleware tests multiple middleware on same route.
func TestChainLinkMultipleMiddleware(t *testing.T) {
	app := New()

	counter := 0

	incrementMiddleware := func(next Handler) Handler {
		return func(c *Context) error {
			counter++
			return next(c)
		}
	}

	handler := func(c *Context) error {
		return nil
	}

	// Add 3 middleware
	app.Post("/test", handler).
		Use(incrementMiddleware).
		Use(incrementMiddleware).
		Use(incrementMiddleware)

	ctx := &Context{}
	ctx.SetMethod("POST")
	ctx.SetPath("/test")

	app.router.ServeHTTP(ctx)

	if counter != 3 {
		t.Errorf("expected counter=3, got %d", counter)
	}
}

// TestDefaultConfig tests default configuration values.
func TestDefaultConfig(t *testing.T) {
	config := DefaultConfig()

	if config.Addr != ":8080" {
		t.Errorf("expected default addr ':8080', got %s", config.Addr)
	}

	if config.MaxRequestBodySize != 10*1024*1024 {
		t.Errorf("expected default body size 10MB, got %d", config.MaxRequestBodySize)
	}

	if config.ErrorHandler == nil {
		t.Error("expected default error handler to be set")
	}

	if config.EnableLogging {
		t.Error("expected EnableLogging to be false by default")
	}

	if !config.DisableStats {
		t.Error("expected DisableStats to be true by default")
	}
}

// TestConfigCustomValues tests custom configuration.
func TestConfigCustomValues(t *testing.T) {
	customErrorHandler := func(c *Context, err error) {
		_ = c.JSON(418, map[string]string{"error": "teapot"})
	}

	config := Config{
		Addr:               ":3000",
		ErrorHandler:       customErrorHandler,
		MaxRequestBodySize: 1024,
		EnableLogging:      true,
		DisableStats:       false,
	}

	if config.Addr != ":3000" {
		t.Errorf("expected addr ':3000', got %s", config.Addr)
	}

	if config.MaxRequestBodySize != 1024 {
		t.Errorf("expected body size 1024, got %d", config.MaxRequestBodySize)
	}

	if !config.EnableLogging {
		t.Error("expected EnableLogging to be true")
	}

	if config.DisableStats {
		t.Error("expected DisableStats to be false")
	}
}

// TestMethodConstants tests HTTP method constants.
func TestMethodConstants(t *testing.T) {
	methods := []HTTPMethod{
		MethodGet,
		MethodPost,
		MethodPut,
		MethodDelete,
		MethodPatch,
		MethodHead,
		MethodOptions,
	}

	expectedMethods := []string{
		"GET",
		"POST",
		"PUT",
		"DELETE",
		"PATCH",
		"HEAD",
		"OPTIONS",
	}

	for i, method := range methods {
		if string(method) != expectedMethods[i] {
			t.Errorf("expected method %s, got %s", expectedMethods[i], method)
		}
	}
}

// TestRouteInfo tests RouteInfo struct.
func TestRouteInfo(t *testing.T) {
	handler := func(c *Context) error {
		return nil
	}

	route := &RouteInfo{
		Method:  MethodGet,
		Path:    "/users",
		Handler: handler,
	}

	if route.Method != MethodGet {
		t.Errorf("expected method GET, got %s", route.Method)
	}

	if route.Path != "/users" {
		t.Errorf("expected path /users, got %s", route.Path)
	}

	if route.Handler == nil {
		t.Error("expected handler to be set")
	}
}

// TestCustomErrorHandler tests using custom error handler with errors.
func TestCustomErrorHandler(t *testing.T) {
	customCalled := false

	customHandler := func(c *Context, err error) {
		customCalled = true
		// Custom error handling logic
		if errors.Is(err, ErrNotFound) {
			_ = c.JSON(404, map[string]string{"error": "not found"})
		} else {
			_ = c.JSON(500, map[string]string{"error": "server error"})
		}
	}

	app := NewWithConfig(Config{
		ErrorHandler: customHandler,
	})

	if app.errorHandler == nil {
		t.Fatal("expected error handler to be set")
	}

	// Simulate error
	ctx := &Context{}
	app.errorHandler(ctx, ErrNotFound)

	if !customCalled {
		t.Error("expected custom error handler to be called")
	}
}
