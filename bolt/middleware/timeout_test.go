package middleware

import (
	"testing"
	"time"

	"github.com/yourusername/bolt/core"
)

// TestTimeout tests basic timeout functionality.
func TestTimeout(t *testing.T) {
	middleware := Timeout(100 * time.Millisecond)

	handler := middleware(func(c *core.Context) error {
		// Sleep longer than timeout
		time.Sleep(200 * time.Millisecond)
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/slow")

	_ = handler(ctx)

	// Check status code - should be 408 (timeout)
	// In test mode, JSON() returns nil, so we check status instead
	if ctx.StatusCode() != 408 {
		t.Errorf("expected status 408 (timeout), got %d", ctx.StatusCode())
	}

	if !ctx.Written() {
		t.Error("expected response to be written after timeout")
	}
}

// TestTimeoutNoTimeout tests that fast requests don't timeout.
func TestTimeoutNoTimeout(t *testing.T) {
	middleware := Timeout(200 * time.Millisecond)

	called := false
	handler := middleware(func(c *core.Context) error {
		// Fast request (no sleep)
		called = true
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/fast")

	err := handler(ctx)

	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !called {
		t.Error("handler was not called")
	}
}

// TestTimeoutSkipPaths tests path skipping.
func TestTimeoutSkipPaths(t *testing.T) {
	config := TimeoutConfig{
		Timeout:   50 * time.Millisecond,
		SkipPaths: []string{"/upload", "/download"},
	}

	middleware := TimeoutWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		// Sleep longer than timeout
		time.Sleep(100 * time.Millisecond)
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Test skipped path - should NOT timeout
	ctx := &core.Context{}
	ctx.SetMethod("POST")
	ctx.SetPath("/upload")

	err := handler(ctx)

	if err != nil {
		t.Errorf("expected no timeout for skipped path, got %v", err)
	}

	// Test non-skipped path - should timeout
	ctx = &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")

	_ = handler(ctx)

	// Should timeout and set 408 status
	if ctx.StatusCode() != 408 {
		t.Errorf("expected timeout (408) for non-skipped path, got status %d", ctx.StatusCode())
	}
}

// TestTimeoutWithCustomHandler tests custom timeout handler.
func TestTimeoutWithCustomHandler(t *testing.T) {
	customHandlerCalled := false

	config := TimeoutConfig{
		Timeout: 50 * time.Millisecond,
		Handler: func(c *core.Context) error {
			customHandlerCalled = true
			return c.JSON(408, map[string]string{"error": "custom timeout"})
		},
	}

	middleware := TimeoutWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		time.Sleep(100 * time.Millisecond)
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/slow")

	_ = handler(ctx)

	if !customHandlerCalled {
		t.Error("expected custom timeout handler to be called")
	}
}

// TestTimeoutHandlerError tests timeout when handler returns error.
func TestTimeoutHandlerError(t *testing.T) {
	middleware := Timeout(200 * time.Millisecond)

	handler := middleware(func(c *core.Context) error {
		// Fast request that returns error
		return core.ErrBadRequest
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/error")

	err := handler(ctx)

	// Should return handler error (not timeout)
	if err != core.ErrBadRequest {
		t.Errorf("expected handler error, got %v", err)
	}
}

// TestDefaultTimeoutConfig tests default configuration.
func TestDefaultTimeoutConfig(t *testing.T) {
	config := DefaultTimeoutConfig()

	if config.Timeout != 30*time.Second {
		t.Errorf("expected default timeout 30s, got %v", config.Timeout)
	}

	if len(config.SkipPaths) != 0 {
		t.Error("expected empty SkipPaths by default")
	}

	if config.Handler != nil {
		t.Error("expected Handler to be nil by default")
	}
}

// TestTimeoutZeroConfig tests zero timeout (should use default).
func TestTimeoutZeroConfig(t *testing.T) {
	config := TimeoutConfig{
		Timeout: 0, // Should use default 30s
	}

	middleware := TimeoutWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		// Very fast request
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/fast")

	err := handler(ctx)

	// Should not timeout (default is 30s)
	if err != nil {
		t.Errorf("unexpected timeout with default config: %v", err)
	}
}

// TestTimeoutConcurrent tests timeout with concurrent requests.
func TestTimeoutConcurrent(t *testing.T) {
	middleware := Timeout(100 * time.Millisecond)

	handler := middleware(func(c *core.Context) error {
		time.Sleep(50 * time.Millisecond) // Within timeout
		return nil
	})

	// Run multiple concurrent requests
	done := make(chan bool, 10)

	for i := 0; i < 10; i++ {
		go func() {
			ctx := &core.Context{}
			ctx.SetMethod("GET")
			ctx.SetPath("/test")

			err := handler(ctx)

			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			done <- true
		}()
	}

	// Wait for all requests
	for i := 0; i < 10; i++ {
		<-done
	}
}

// BenchmarkTimeout benchmarks timeout middleware overhead.
func BenchmarkTimeout(b *testing.B) {
	middleware := Timeout(5 * time.Second)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/test")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// BenchmarkTimeoutSkipPath benchmarks skipped path performance.
func BenchmarkTimeoutSkipPath(b *testing.B) {
	config := TimeoutConfig{
		Timeout:   5 * time.Second,
		SkipPaths: []string{"/health"},
	}

	middleware := TimeoutWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/health") // Skipped path

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}
