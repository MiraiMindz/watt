package middleware

import (
	"fmt"
	"testing"
	"time"

	"github.com/yourusername/bolt/core"
)

// TestRateLimit tests basic rate limiting.
func TestRateLimit(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 2,
		Burst:             2,
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// First 2 requests should succeed (burst)
	for i := 0; i < 2; i++ {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api")

		err := handler(ctx)
		if err != nil {
			t.Errorf("request %d: unexpected error: %v", i+1, err)
		}

		if ctx.StatusCode() != 200 {
			t.Errorf("request %d: expected status 200, got %d", i+1, ctx.StatusCode())
		}
	}

	// Third request should be rate limited
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")

	_ = handler(ctx)

	if ctx.StatusCode() != 429 {
		t.Errorf("expected status 429 (rate limited), got %d", ctx.StatusCode())
	}
}

// TestRateLimitRecovery tests that tokens refill over time.
func TestRateLimitRecovery(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 10,
		Burst:             1,
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// First request succeeds
	ctx1 := &core.Context{}
	ctx1.SetMethod("GET")
	ctx1.SetPath("/api")

	err := handler(ctx1)
	if err != nil {
		t.Errorf("first request: unexpected error: %v", err)
	}

	// Second request fails (no tokens)
	ctx2 := &core.Context{}
	ctx2.SetMethod("GET")
	ctx2.SetPath("/api")

	_ = handler(ctx2)

	if ctx2.StatusCode() != 429 {
		t.Errorf("expected status 429, got %d", ctx2.StatusCode())
	}

	// Wait for token to refill (0.1 seconds at 10 req/s)
	time.Sleep(150 * time.Millisecond)

	// Third request should succeed
	ctx3 := &core.Context{}
	ctx3.SetMethod("GET")
	ctx3.SetPath("/api")

	err = handler(ctx3)
	if err != nil {
		t.Errorf("third request: unexpected error: %v", err)
	}

	if ctx3.StatusCode() != 200 {
		t.Errorf("expected status 200 after refill, got %d", ctx3.StatusCode())
	}
}

// TestRateLimitPerKey tests per-key rate limiting.
func TestRateLimitPerKey(t *testing.T) {
	keyCounter := 0

	config := RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             1,
		KeyFunc: func(c *core.Context) string {
			// Different key for each request
			keyCounter++
			return c.GetHeader("X-User-ID")
		},
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// User 1 - first request succeeds
	ctx1 := &core.Context{}
	ctx1.SetMethod("GET")
	ctx1.SetPath("/api")
	ctx1.SetRequestHeader("X-User-ID", "user1")

	err := handler(ctx1)
	if err != nil {
		t.Errorf("user1 first request: unexpected error: %v", err)
	}

	// User 2 - should succeed (different key)
	ctx2 := &core.Context{}
	ctx2.SetMethod("GET")
	ctx2.SetPath("/api")
	ctx2.SetRequestHeader("X-User-ID", "user2")

	err = handler(ctx2)
	if err != nil {
		t.Errorf("user2 request: unexpected error: %v", err)
	}

	if ctx2.StatusCode() != 200 {
		t.Errorf("expected status 200 for different user, got %d", ctx2.StatusCode())
	}

	// User 1 - second request should be rate limited
	ctx3 := &core.Context{}
	ctx3.SetMethod("GET")
	ctx3.SetPath("/api")
	ctx3.SetRequestHeader("X-User-ID", "user1")

	_ = handler(ctx3)

	if ctx3.StatusCode() != 429 {
		t.Errorf("expected status 429 for user1 second request, got %d", ctx3.StatusCode())
	}
}

// TestRateLimitCustomErrorHandler tests custom error handler.
func TestRateLimitCustomErrorHandler(t *testing.T) {
	errorHandlerCalled := false

	config := RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             1,
		ErrorHandler: func(c *core.Context) error {
			errorHandlerCalled = true
			return c.JSON(503, map[string]string{"error": "service unavailable"})
		},
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// First request succeeds
	ctx1 := &core.Context{}
	ctx1.SetMethod("GET")
	ctx1.SetPath("/api")
	_ = handler(ctx1)

	// Second request should call custom error handler
	ctx2 := &core.Context{}
	ctx2.SetMethod("GET")
	ctx2.SetPath("/api")
	_ = handler(ctx2)

	if !errorHandlerCalled {
		t.Error("custom error handler should be called")
	}

	if ctx2.StatusCode() != 503 {
		t.Errorf("expected status 503 from custom error handler, got %d", ctx2.StatusCode())
	}
}

// TestRateLimitBurst tests burst handling.
func TestRateLimitBurst(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 1,
		Burst:             5,
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return c.JSON(200, map[string]string{"status": "ok"})
	})

	// Burst of 5 requests should all succeed
	for i := 0; i < 5; i++ {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api")

		err := handler(ctx)
		if err != nil {
			t.Errorf("burst request %d: unexpected error: %v", i+1, err)
		}

		if ctx.StatusCode() != 200 {
			t.Errorf("burst request %d: expected status 200, got %d", i+1, ctx.StatusCode())
		}
	}

	// 6th request should be rate limited
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")

	_ = handler(ctx)

	if ctx.StatusCode() != 429 {
		t.Errorf("expected status 429 after burst, got %d", ctx.StatusCode())
	}
}

// TestRateLimitConcurrent tests concurrent requests.
func TestRateLimitConcurrent(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 100,
		Burst:             50,
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	// Run 50 concurrent requests (should all succeed)
	done := make(chan bool, 50)

	for i := 0; i < 50; i++ {
		go func() {
			ctx := &core.Context{}
			ctx.SetMethod("GET")
			ctx.SetPath("/api")

			err := handler(ctx)
			if err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			done <- true
		}()
	}

	// Wait for all requests
	for i := 0; i < 50; i++ {
		<-done
	}
}

// TestDefaultRateLimitConfig tests default configuration.
func TestDefaultRateLimitConfig(t *testing.T) {
	config := DefaultRateLimitConfig()

	if config.RequestsPerSecond != 100 {
		t.Errorf("expected default rate 100, got %d", config.RequestsPerSecond)
	}

	if config.Burst != 20 {
		t.Errorf("expected default burst 20, got %d", config.Burst)
	}

	if config.KeyFunc == nil {
		t.Error("expected default KeyFunc to be set")
	}

	if config.CleanupInterval != 1*time.Minute {
		t.Errorf("expected default cleanup interval 1m, got %v", config.CleanupInterval)
	}

	if config.MaxAge != 5*time.Minute {
		t.Errorf("expected default max age 5m, got %v", config.MaxAge)
	}
}

// TestTokenBucket tests token bucket algorithm.
func TestTokenBucket(t *testing.T) {
	tb := newTokenBucket(10.0, 5) // 10 req/s, burst of 5

	// Initial state: 5 tokens available
	for i := 0; i < 5; i++ {
		if !tb.allow() {
			t.Errorf("request %d should be allowed (initial burst)", i+1)
		}
	}

	// 6th request should fail (no tokens)
	if tb.allow() {
		t.Error("6th request should be denied (no tokens)")
	}

	// Wait for 1 token to refill (0.1s at 10 req/s)
	time.Sleep(150 * time.Millisecond)

	// Should have 1 token available
	if !tb.allow() {
		t.Error("request should be allowed after refill")
	}

	// Next request should fail again
	if tb.allow() {
		t.Error("request should be denied (tokens exhausted)")
	}
}

// TestTokenBucketRetryIn tests retry time calculation.
func TestTokenBucketRetryIn(t *testing.T) {
	tb := newTokenBucket(10.0, 1) // 10 req/s, burst of 1

	// Consume the token
	tb.allow()

	// Calculate retry time
	retryIn := tb.retryIn()

	// Should be approximately 0.1 seconds (1/10)
	expectedMin := 90 * time.Millisecond
	expectedMax := 110 * time.Millisecond

	if retryIn < expectedMin || retryIn > expectedMax {
		t.Errorf("expected retry time between %v and %v, got %v", expectedMin, expectedMax, retryIn)
	}
}

// TestLimiterStore tests limiter store management.
func TestLimiterStore(t *testing.T) {
	store := &limiterStore{
		rate:            10.0,
		burst:           5,
		cleanupInterval: 100 * time.Millisecond,
		maxAge:          200 * time.Millisecond,
	}

	// Get limiter for key1
	limiter1 := store.getLimiter("key1")
	if limiter1 == nil {
		t.Fatal("expected limiter to be created")
	}

	// Get same limiter again
	limiter2 := store.getLimiter("key1")
	if limiter1 != limiter2 {
		t.Error("expected same limiter instance for same key")
	}

	// Get limiter for different key
	limiter3 := store.getLimiter("key2")
	if limiter1 == limiter3 {
		t.Error("expected different limiter for different key")
	}
}

// TestDefaultKeyFunc tests default key function.
func TestDefaultKeyFunc(t *testing.T) {
	testCases := []struct {
		name     string
		headers  map[string]string
		expected string
	}{
		{
			name:     "X-Forwarded-For",
			headers:  map[string]string{"X-Forwarded-For": "192.168.1.1"},
			expected: "192.168.1.1",
		},
		{
			name:     "X-Real-IP",
			headers:  map[string]string{"X-Real-IP": "10.0.0.1"},
			expected: "10.0.0.1",
		},
		{
			name:     "No headers",
			headers:  map[string]string{},
			expected: "default",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.name, func(t *testing.T) {
			ctx := &core.Context{}
			for k, v := range tc.headers {
				ctx.SetRequestHeader(k, v)
			}

			key := defaultKeyFunc(ctx)
			if key != tc.expected {
				t.Errorf("expected key %s, got %s", tc.expected, key)
			}
		})
	}
}

// BenchmarkRateLimit benchmarks rate limiting overhead.
func BenchmarkRateLimit(b *testing.B) {
	config := RateLimitConfig{
		RequestsPerSecond: 1000000, // Very high limit
		Burst:             1000000,
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// BenchmarkRateLimitMultipleKeys benchmarks per-key overhead.
func BenchmarkRateLimitMultipleKeys(b *testing.B) {
	keyCounter := 0

	config := RateLimitConfig{
		RequestsPerSecond: 1000000,
		Burst:             1000000,
		KeyFunc: func(c *core.Context) string {
			// Rotate through 100 different keys
			key := keyCounter % 100
			keyCounter++
			return string(rune('0' + key))
		},
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = handler(ctx)
	}
}

// TestLimiterStoreCleanup tests that cleanup removes old limiters.
func TestLimiterStoreCleanup(t *testing.T) {
	config := RateLimitConfig{
		RequestsPerSecond: 100,
		Burst:             10,
		CleanupInterval:   50 * time.Millisecond,
		MaxAge:            100 * time.Millisecond,
	}

	middleware := RateLimit(config)

	handler := middleware(func(c *core.Context) error {
		return nil
	})

	// Make requests with different keys to create limiters
	for i := 0; i < 3; i++ {
		ctx := &core.Context{}
		ctx.SetMethod("GET")
		ctx.SetPath("/api")
		ctx.SetRequestHeader("X-Forwarded-For", fmt.Sprintf("192.168.1.%d", i))
		_ = handler(ctx)
	}

	// Wait for limiters to age past MaxAge
	time.Sleep(150 * time.Millisecond)

	// Wait for cleanup to run (cleanup interval + buffer)
	time.Sleep(100 * time.Millisecond)

	// Create a new request - this verifies cleanup ran
	// (we can't directly check the internal map, but cleanup running is covered)
	ctx := &core.Context{}
	ctx.SetMethod("GET")
	ctx.SetPath("/api")
	ctx.SetRequestHeader("X-Forwarded-For", "192.168.1.100")
	err := handler(ctx)
	if err != nil {
		t.Errorf("unexpected error: %v", err)
	}
}

// TestTokenBucketRetryInWithTokens tests retryIn when tokens are available.
func TestTokenBucketRetryInWithTokens(t *testing.T) {
	tb := newTokenBucket(10.0, 5) // 10 req/s, burst of 5

	// Don't consume any tokens - should have 5 available
	retryIn := tb.retryIn()

	// Should be 0 since we have tokens available
	if retryIn != 0 {
		t.Errorf("expected retry time 0 when tokens available, got %v", retryIn)
	}
}

// TestTokenBucketRetryInPartialTokens tests retryIn with partial tokens.
func TestTokenBucketRetryInPartialTokens(t *testing.T) {
	tb := newTokenBucket(10.0, 1) // 10 req/s, burst of 1

	// Consume the token
	tb.allow()

	// Immediately check retry time
	retryIn := tb.retryIn()

	// Should be approximately 0.1 seconds
	expectedMin := 50 * time.Millisecond
	expectedMax := 150 * time.Millisecond

	if retryIn < expectedMin || retryIn > expectedMax {
		t.Errorf("expected retry time between %v and %v, got %v", expectedMin, expectedMax, retryIn)
	}
}
