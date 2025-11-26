package middleware

import (
	"testing"

	"github.com/yourusername/bolt/core"
)

// TestCORS tests basic CORS functionality.
func TestCORS(t *testing.T) {
	middleware := CORS()

	handler := middleware(func(c *core.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api/users")
	ctx.SetRequestHeader("Origin", "https://example.com")

	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check CORS headers (response headers)
	allowOrigin := ctx.GetResponseHeader("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected Access-Control-Allow-Origin=*, got %s", allowOrigin)
	}
}

// TestCORSPreflight tests preflight OPTIONS request handling.
func TestCORSPreflight(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:  []string{"https://example.com"},
		AllowMethods:  []string{"GET", "POST", "PUT"},
		AllowHeaders:  []string{"Content-Type", "Authorization"},
		MaxAge:        3600,
	}

	middleware := CORSWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		// Should not reach here for OPTIONS
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	ctx := &core.Context{}
	ctx.SetMethod("OPTIONS")
	ctx.SetPath("/api/users")
	ctx.SetRequestHeader("Origin", "https://example.com")

	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	// Check preflight headers
	allowOrigin := ctx.GetResponseHeader("Access-Control-Allow-Origin")
	if allowOrigin != "https://example.com" {
		t.Errorf("expected origin https://example.com, got %s", allowOrigin)
	}

	allowMethods := ctx.GetResponseHeader("Access-Control-Allow-Methods")
	if allowMethods == "" {
		t.Error("expected Access-Control-Allow-Methods header")
	}

	allowHeaders := ctx.GetResponseHeader("Access-Control-Allow-Headers")
	if allowHeaders == "" {
		t.Error("expected Access-Control-Allow-Headers header")
	}

	maxAge := ctx.GetResponseHeader("Access-Control-Max-Age")
	if maxAge != "3600" {
		t.Errorf("expected Max-Age 3600, got %s", maxAge)
	}

	// Check status is 204 for preflight
	if ctx.StatusCode() != 204 {
		t.Errorf("expected status 204 for preflight, got %d", ctx.StatusCode())
	}
}

// TestCORSSpecificOrigin tests specific origin allowlist.
func TestCORSSpecificOrigin(t *testing.T) {
	config := CORSConfig{
		AllowOrigins: []string{"https://app.example.com", "https://admin.example.com"},
	}

	middleware := CORSWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	// Test allowed origin
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Origin", "https://app.example.com")

	_ = handler(ctx)

	allowOrigin := ctx.GetResponseHeader("Access-Control-Allow-Origin")
	if allowOrigin != "https://app.example.com" {
		t.Errorf("expected origin https://app.example.com, got %s", allowOrigin)
	}

	// Test disallowed origin
	ctx = &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Origin", "https://evil.com")

	_ = handler(ctx)

	allowOrigin = ctx.GetResponseHeader("Access-Control-Allow-Origin")
	if allowOrigin != "" {
		t.Errorf("expected no CORS header for disallowed origin, got %s", allowOrigin)
	}
}

// TestCORSCredentials tests credentials handling.
func TestCORSCredentials(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:     []string{"https://example.com"},
		AllowCredentials: true,
	}

	middleware := CORSWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Origin", "https://example.com")

	_ = handler(ctx)

	allowCredentials := ctx.GetResponseHeader("Access-Control-Allow-Credentials")
	if allowCredentials != "true" {
		t.Errorf("expected Allow-Credentials=true, got %s", allowCredentials)
	}
}

// TestCORSExposeHeaders tests exposed headers.
func TestCORSExposeHeaders(t *testing.T) {
	config := CORSConfig{
		AllowOrigins:  []string{"*"},
		ExposeHeaders: []string{"X-Request-ID", "X-RateLimit-Remaining"},
	}

	middleware := CORSWithConfig(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Origin", "https://example.com")

	_ = handler(ctx)

	exposeHeaders := ctx.GetResponseHeader("Access-Control-Expose-Headers")
	if exposeHeaders == "" {
		t.Error("expected Expose-Headers to be set")
	}

	if exposeHeaders != "X-Request-ID, X-RateLimit-Remaining" {
		t.Errorf("expected specific expose headers, got %s", exposeHeaders)
	}
}

// TestCORSNoOrigin tests request without Origin header.
func TestCORSNoOrigin(t *testing.T) {
	middleware := CORS()

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	// No Origin header

	_ = handler(ctx)

	// Should still set CORS headers for wildcard
	allowOrigin := ctx.GetResponseHeader("Access-Control-Allow-Origin")
	if allowOrigin != "*" {
		t.Errorf("expected wildcard origin, got %s", allowOrigin)
	}
}

// TestDefaultCORSConfig tests default configuration.
func TestDefaultCORSConfig(t *testing.T) {
	config := DefaultCORSConfig()

	if len(config.AllowOrigins) != 1 || config.AllowOrigins[0] != "*" {
		t.Error("expected default AllowOrigins to be [*]")
	}

	if len(config.AllowMethods) == 0 {
		t.Error("expected default AllowMethods to be set")
	}

	if len(config.AllowHeaders) != 1 || config.AllowHeaders[0] != "*" {
		t.Error("expected default AllowHeaders to be [*]")
	}

	if config.AllowCredentials {
		t.Error("expected AllowCredentials to be false by default")
	}

	if config.MaxAge != 86400 {
		t.Errorf("expected MaxAge 86400, got %d", config.MaxAge)
	}
}

// TestCORSNonPreflightRequest tests regular (non-OPTIONS) requests.
func TestCORSNonPreflightRequest(t *testing.T) {
	middleware := CORS()

	called := false
	handler := middleware(func(c *core.Context) error {
		called = true
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("POST") // Non-OPTIONS
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Origin", "https://example.com")

	_ = handler(ctx)

	// Handler should be called for non-OPTIONS
	if !called {
		t.Error("expected handler to be called for non-OPTIONS request")
	}

	// CORS headers should still be set
	allowOrigin := ctx.GetResponseHeader("Access-Control-Allow-Origin")
	if allowOrigin == "" {
		t.Error("expected CORS headers to be set for non-OPTIONS request")
	}
}

// BenchmarkCORS benchmarks CORS middleware overhead.
func BenchmarkCORS(b *testing.B) {
	middleware := CORS()

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Origin", "https://example.com")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// BenchmarkCORSPreflight benchmarks preflight handling.
func BenchmarkCORSPreflight(b *testing.B) {
	middleware := CORS()

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("OPTIONS")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Origin", "https://example.com")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}
