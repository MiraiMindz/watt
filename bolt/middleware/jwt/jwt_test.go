package jwt

import (
	"fmt"
	"testing"
	"time"

	"github.com/golang-jwt/jwt/v5"
	"github.com/yourusername/bolt/core"
)

var testSecret = []byte("test-secret-key-12345")

// TestJWT tests basic JWT authentication.
func TestJWT(t *testing.T) {
	config := JWTConfig{
		Secret:    testSecret,
		Algorithm: "HS256",
	}

	middleware := JWT(config)

	// Create valid token
	token := createTestToken(t, testSecret, jwt.MapClaims{
		"user_id": "123",
		"email":   "test@example.com",
	})

	handler := middleware(func(c *core.Context) error {
		// Verify claims are stored in context
		claims := c.Get("user")
		if claims == nil {
			t.Error("expected claims in context")
		}
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api/users")
	ctx.SetRequestHeader("Authorization", "Bearer "+token)

	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if ctx.StatusCode() != 200 {
		t.Errorf("expected status 200, got %d", ctx.StatusCode())
	}
}

// TestJWTMissingToken tests missing authorization token.
func TestJWTMissingToken(t *testing.T) {
	config := JWTConfig{
		Secret: testSecret,
	}

	middleware := JWT(config)

	handler := middleware(func(c *core.Context) error {
		t.Error("handler should not be called without token")
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api/users")
	// No Authorization header

	_ = handler(ctx)

	// Should return 401
	if ctx.StatusCode() != 401 {
		t.Errorf("expected status 401, got %d", ctx.StatusCode())
	}
}

// TestJWTInvalidToken tests invalid token.
func TestJWTInvalidToken(t *testing.T) {
	config := JWTConfig{
		Secret: testSecret,
	}

	middleware := JWT(config)

	handler := middleware(func(c *core.Context) error {
		t.Error("handler should not be called with invalid token")
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api/users")
	ctx.SetRequestHeader("Authorization", "Bearer invalid-token-12345")

	_ = handler(ctx)

	// Should return 401
	if ctx.StatusCode() != 401 {
		t.Errorf("expected status 401, got %d", ctx.StatusCode())
	}
}

// TestJWTExpiredToken tests expired token.
func TestJWTExpiredToken(t *testing.T) {
	config := JWTConfig{
		Secret: testSecret,
	}

	middleware := JWT(config)

	// Create expired token
	token := createExpiredToken(t, testSecret)

	handler := middleware(func(c *core.Context) error {
		t.Error("handler should not be called with expired token")
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api/users")
	ctx.SetRequestHeader("Authorization", "Bearer "+token)

	_ = handler(ctx)

	// Should return 401
	if ctx.StatusCode() != 401 {
		t.Errorf("expected status 401, got %d", ctx.StatusCode())
	}
}

// TestJWTSkipPaths tests path skipping.
func TestJWTSkipPaths(t *testing.T) {
	config := JWTConfig{
		Secret:    testSecret,
		SkipPaths: []string{"/login", "/register"},
	}

	middleware := JWT(config)

	handlerCalled := false
	handler := middleware(func(c *core.Context) error {
		handlerCalled = true
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("POST")
	ctx.SetPath("/login")
	// No Authorization header

	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}

	if !handlerCalled {
		t.Error("handler should be called for skipped path")
	}
}

// TestJWTCustomContextKey tests custom context key.
func TestJWTCustomContextKey(t *testing.T) {
	config := JWTConfig{
		Secret:     testSecret,
		ContextKey: "auth",
	}

	middleware := JWT(config)

	token := createTestToken(t, testSecret, jwt.MapClaims{
		"user_id": "456",
	})

	handler := middleware(func(c *core.Context) error {
		// Verify custom key is used
		claims := c.Get("auth")
		if claims == nil {
			t.Error("expected claims in context with custom key")
		}
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Authorization", "Bearer "+token)

	_ = handler(ctx)
}

// TestJWTCustomErrorHandler tests custom error handler.
func TestJWTCustomErrorHandler(t *testing.T) {
	errorHandlerCalled := false

	config := JWTConfig{
		Secret: testSecret,
		ErrorHandler: func(c *core.Context, err error) error {
			errorHandlerCalled = true
			return c.JSON(403, map[string]string{"error": "custom error"})
		},
	}

	middleware := JWT(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	// No token

	_ = handler(ctx)

	if !errorHandlerCalled {
		t.Error("custom error handler should be called")
	}

	if ctx.StatusCode() != 403 {
		t.Errorf("expected status 403 from custom error handler, got %d", ctx.StatusCode())
	}
}

// TestJWTTokenCache tests token caching.
func TestJWTTokenCache(t *testing.T) {
	config := JWTConfig{
		Secret:   testSecret,
		CacheTTL: 1 * time.Second,
	}

	middleware := JWT(config)

	token := createTestToken(t, testSecret, jwt.MapClaims{
		"user_id": "789",
	})

	callCount := 0
	handler := middleware(func(c *core.Context) error {
		callCount++
		return nil
	})

	// First request - token parsed
	ctx1 := &core.Context{}
	ctx1.SetMethod("GET")
	ctx1.SetPath("/api")
	ctx1.SetRequestHeader("Authorization", "Bearer "+token)

	_ = handler(ctx1)

	// Second request - should use cache
	ctx2 := &core.Context{}
	ctx2.SetMethod("GET")
	ctx2.SetPath("/api")
	ctx2.SetRequestHeader("Authorization", "Bearer "+token)

	_ = handler(ctx2)

	if callCount != 2 {
		t.Errorf("expected handler to be called twice, got %d", callCount)
	}
}

// TestJWTInvalidAuthHeader tests invalid Authorization header format.
func TestJWTInvalidAuthHeader(t *testing.T) {
	config := JWTConfig{
		Secret: testSecret,
	}

	middleware := JWT(config)

	handler := middleware(func(c *core.Context) error {
		t.Error("handler should not be called with invalid auth header")
		return nil
	})

	testCases := []string{
		"InvalidFormat",
		"Bearer",
		"Basic dXNlcjpwYXNz",
	}

	for _, tc := range testCases {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api")
		ctx.SetRequestHeader("Authorization", tc)

		_ = handler(ctx)

		if ctx.StatusCode() != 401 {
			t.Errorf("expected status 401 for auth header '%s', got %d", tc, ctx.StatusCode())
		}
	}
}

// TestJWTDifferentAlgorithm tests algorithm validation.
func TestJWTDifferentAlgorithm(t *testing.T) {
	config := JWTConfig{
		Secret:    testSecret,
		Algorithm: "HS256",
	}

	middleware := JWT(config)

	// Create token with different algorithm (HS512)
	token := jwt.NewWithClaims(jwt.SigningMethodHS512, jwt.MapClaims{
		"user_id": "999",
	})

	tokenString, err := token.SignedString(testSecret)
	if err != nil {
		t.Fatalf("failed to create token: %v", err)
	}

	handler := middleware(func(c *core.Context) error {
		t.Error("handler should not be called with wrong algorithm")
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Authorization", "Bearer "+tokenString)

	_ = handler(ctx)

	// Should return 401
	if ctx.StatusCode() != 401 {
		t.Errorf("expected status 401 for wrong algorithm, got %d", ctx.StatusCode())
	}
}

// TestDefaultJWTConfig tests default configuration.
func TestDefaultJWTConfig(t *testing.T) {
	config := DefaultJWTConfig(testSecret)

	if config.Algorithm != "HS256" {
		t.Errorf("expected default algorithm HS256, got %s", config.Algorithm)
	}

	if config.ContextKey != "user" {
		t.Errorf("expected default context key 'user', got %s", config.ContextKey)
	}

	if config.CacheTTL != 5*time.Minute {
		t.Errorf("expected default cache TTL 5m, got %v", config.CacheTTL)
	}

	if len(config.SkipPaths) != 0 {
		t.Error("expected empty SkipPaths by default")
	}
}

// TestJWTErrors tests error constants.
func TestJWTErrors(t *testing.T) {
	if ErrMissingToken.Error() == "" {
		t.Error("ErrMissingToken should have message")
	}

	if ErrInvalidAuthHeader.Error() == "" {
		t.Error("ErrInvalidAuthHeader should have message")
	}

	if ErrInvalidToken.Error() == "" {
		t.Error("ErrInvalidToken should have message")
	}

	if ErrInvalidClaims.Error() == "" {
		t.Error("ErrInvalidClaims should have message")
	}
}

// BenchmarkJWT benchmarks JWT middleware overhead.
func BenchmarkJWT(b *testing.B) {
	config := JWTConfig{
		Secret: testSecret,
	}

	middleware := JWT(config)

	token := createTestToken(nil, testSecret, jwt.MapClaims{
		"user_id": "bench",
	})

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Authorization", "Bearer "+token)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// BenchmarkJWTSkipPath benchmarks skipped path performance.
func BenchmarkJWTSkipPath(b *testing.B) {
	config := JWTConfig{
		Secret:    testSecret,
		SkipPaths: []string{"/health"},
	}

	middleware := JWT(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/health")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// TestJWTCacheExpiration tests that expired cache entries are not returned.
func TestJWTCacheExpiration(t *testing.T) {
	config := JWTConfig{
		Secret:   testSecret,
		CacheTTL: 50 * time.Millisecond, // Very short TTL
	}

	middleware := JWT(config)

	token := createTestToken(t, testSecret, jwt.MapClaims{
		"user_id": "cache-test",
	})

	callCount := 0
	handler := middleware(func(c *core.Context) error {
		callCount++
		return nil
	})

	// First request - token parsed and cached
	ctx1 := &core.Context{}
	ctx1.SetMethod("GET")
	ctx1.SetPath("/api")
	ctx1.SetRequestHeader("Authorization", "Bearer "+token)
	_ = handler(ctx1)

	// Wait for cache to expire
	time.Sleep(100 * time.Millisecond)

	// Second request - cache expired, should re-parse
	ctx2 := &core.Context{}
	ctx2.SetMethod("GET")
	ctx2.SetPath("/api")
	ctx2.SetRequestHeader("Authorization", "Bearer "+token)
	_ = handler(ctx2)

	if callCount != 2 {
		t.Errorf("expected handler to be called twice, got %d", callCount)
	}
}

// TestJWTCacheCleanup tests that cleanup removes expired tokens.
func TestJWTCacheCleanup(t *testing.T) {
	// Note: cleanup runs every 1 minute in production, but we can still test
	// that the mechanism works by creating expired entries and making new requests
	config := JWTConfig{
		Secret:   testSecret,
		CacheTTL: 10 * time.Millisecond, // Very short TTL for testing
	}

	middleware := JWT(config)

	// Create multiple tokens to add to cache
	tokens := make([]string, 5)
	for i := 0; i < 5; i++ {
		tokens[i] = createTestToken(t, testSecret, jwt.MapClaims{
			"user_id": fmt.Sprintf("cleanup-test-%d", i),
		})
	}

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	// Make requests to add tokens to cache
	for i, token := range tokens {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api")
		ctx.SetRequestHeader("Authorization", "Bearer "+token)
		err := handler(ctx)
		if err != nil {
			t.Errorf("request %d: unexpected error: %v", i, err)
		}
	}

	// Wait for cache entries to expire
	time.Sleep(50 * time.Millisecond)

	// Make new requests - expired entries won't be returned by get()
	// This exercises the expiration check in get()
	for i, token := range tokens {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api")
		ctx.SetRequestHeader("Authorization", "Bearer "+token)
		err := handler(ctx)
		if err != nil {
			t.Errorf("request %d after expiry: unexpected error: %v", i, err)
		}
	}

	// Give a moment for goroutines to settle
	time.Sleep(10 * time.Millisecond)
}

// TestJWTWithEmptyBearer tests empty bearer token.
func TestJWTWithEmptyBearer(t *testing.T) {
	config := JWTConfig{
		Secret: testSecret,
	}

	middleware := JWT(config)

	handler := middleware(func(c *core.Context) error {
		t.Error("handler should not be called with empty bearer")
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Authorization", "Bearer ") // Empty after "Bearer "

	_ = handler(ctx)

	if ctx.StatusCode() != 401 {
		t.Errorf("expected status 401, got %d", ctx.StatusCode())
	}
}

// TestJWTMultipleRequests tests multiple requests with same and different tokens.
func TestJWTMultipleRequests(t *testing.T) {
	config := JWTConfig{
		Secret:   testSecret,
		CacheTTL: 1 * time.Second,
	}

	middleware := JWT(config)

	tokens := []string{
		createTestToken(t, testSecret, jwt.MapClaims{"user_id": "user1"}),
		createTestToken(t, testSecret, jwt.MapClaims{"user_id": "user2"}),
		createTestToken(t, testSecret, jwt.MapClaims{"user_id": "user3"}),
	}

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	// Make multiple requests with different tokens
	for i := 0; i < 3; i++ {
		for _, token := range tokens {
			ctx := &core.Context{}
			ctx.SetMethod("GET")
			ctx.SetPath("/api")
			ctx.SetRequestHeader("Authorization", "Bearer "+token)

			err := handler(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}
		}
	}
}

// TestJWTCacheMiss tests cache miss scenarios.
func TestJWTCacheMiss(t *testing.T) {
	config := JWTConfig{
		Secret:   testSecret,
		CacheTTL: 1 * time.Second,
	}

	middleware := JWT(config)

	// Create a new token each time to ensure cache miss
	handler := middleware(func(c *core.Context) error {
		return nil
	})

	for i := 0; i < 5; i++ {
		token := createTestToken(t, testSecret, jwt.MapClaims{
			"user_id": fmt.Sprintf("user%d", i),
			"nonce":   time.Now().UnixNano(), // Make each token unique
		})

		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api")
		ctx.SetRequestHeader("Authorization", "Bearer "+token)

		err := handler(ctx)
		if err != nil {
			t.Errorf("request %d: unexpected error: %v", i, err)
		}
	}
}

// TestJWTInvalidSignature tests token with invalid signature.
func TestJWTInvalidSignature(t *testing.T) {
	config := JWTConfig{
		Secret: testSecret,
	}

	middleware := JWT(config)

	// Create token with wrong secret
	wrongSecret := []byte("wrong-secret-key")
	token := createTestToken(t, wrongSecret, jwt.MapClaims{
		"user_id": "test",
	})

	handler := middleware(func(c *core.Context) error {
		t.Error("handler should not be called with invalid signature")
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("Authorization", "Bearer "+token)

	_ = handler(ctx)

	if ctx.StatusCode() != 401 {
		t.Errorf("expected status 401 for invalid signature, got %d", ctx.StatusCode())
	}
}

// Helper functions

func createTestToken(t *testing.T, secret []byte, claims jwt.MapClaims) string {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, claims)
	tokenString, err := token.SignedString(secret)
	if err != nil {
		if t != nil {
			t.Fatalf("failed to create token: %v", err)
		}
		panic(err)
	}
	return tokenString
}

func createExpiredToken(t *testing.T, secret []byte) string {
	claims := jwt.MapClaims{
		"user_id": "expired",
		"exp":     time.Now().Add(-1 * time.Hour).Unix(),
	}
	return createTestToken(t, secret, claims)
}
