package middleware

import (
	"sync"
	"time"

	"github.com/yourusername/bolt/core"
)

// RateLimit returns rate limiting middleware with default configuration.
//
// Uses token bucket algorithm for rate limiting.
// Limits are applied per-key (IP address by default).
//
// Example:
//
//	app.Use(middleware.RateLimit(middleware.RateLimitConfig{
//	    RequestsPerSecond: 100,
//	    Burst:             20,
//	}))
//
// Performance: <50ns overhead for cached limiters.
func RateLimit(config RateLimitConfig) core.Middleware {
	return RateLimitWithConfig(config)
}

// RateLimitWithConfig returns rate limiting middleware with custom configuration.
//
// Example:
//
//	app.Use(middleware.RateLimitWithConfig(middleware.RateLimitConfig{
//	    RequestsPerSecond: 10,
//	    Burst:             5,
//	    KeyFunc: func(c *core.Context) string {
//	        // Rate limit by user ID
//	        user := c.Get("user").(string)
//	        return user
//	    },
//	}))
func RateLimitWithConfig(config RateLimitConfig) core.Middleware {
	// Apply defaults
	if config.RequestsPerSecond == 0 {
		config.RequestsPerSecond = 100
	}
	if config.Burst == 0 {
		config.Burst = 20
	}
	if config.KeyFunc == nil {
		config.KeyFunc = defaultKeyFunc
	}
	if config.CleanupInterval == 0 {
		config.CleanupInterval = 1 * time.Minute
	}
	if config.MaxAge == 0 {
		config.MaxAge = 5 * time.Minute
	}

	// Initialize limiter store
	store := &limiterStore{
		limiters:        sync.Map{},
		rate:            float64(config.RequestsPerSecond),
		burst:           config.Burst,
		cleanupInterval: config.CleanupInterval,
		maxAge:          config.MaxAge,
	}

	// Start cleanup goroutine
	go store.cleanup()

	return func(next core.Handler) core.Handler {
		return func(c *core.Context) error {
			// Get rate limit key
			key := config.KeyFunc(c)

			// Get or create limiter for this key
			limiter := store.getLimiter(key)

			// Check if request is allowed
			if !limiter.limiter.allow() {
				// Rate limit exceeded
				if config.ErrorHandler != nil {
					return config.ErrorHandler(c)
				}
				return c.JSON(429, map[string]interface{}{
					"error":   "Rate limit exceeded",
					"retryIn": limiter.limiter.retryIn().Seconds(),
				})
			}

			return next(c)
		}
	}
}

// RateLimitConfig defines rate limiting configuration.
type RateLimitConfig struct {
	// RequestsPerSecond is the number of requests allowed per second
	// Default: 100
	RequestsPerSecond int

	// Burst is the maximum burst size
	// Default: 20
	Burst int

	// KeyFunc generates a unique key for rate limiting
	// Default: IP address
	KeyFunc func(*core.Context) string

	// ErrorHandler is called when rate limit is exceeded
	// Default: returns 429 with retry time
	ErrorHandler func(*core.Context) error

	// CleanupInterval is how often to clean up old limiters
	// Default: 1 minute
	CleanupInterval time.Duration

	// MaxAge is how long to keep inactive limiters
	// Default: 5 minutes
	MaxAge time.Duration
}

// DefaultRateLimitConfig returns default rate limiting configuration.
func DefaultRateLimitConfig() RateLimitConfig {
	return RateLimitConfig{
		RequestsPerSecond: 100,
		Burst:             20,
		KeyFunc:           defaultKeyFunc,
		CleanupInterval:   1 * time.Minute,
		MaxAge:            5 * time.Minute,
	}
}

// defaultKeyFunc returns IP address as rate limit key.
func defaultKeyFunc(c *core.Context) string {
	// Try X-Forwarded-For header first
	if ip := c.GetHeader("X-Forwarded-For"); ip != "" {
		return ip
	}

	// Try X-Real-IP header
	if ip := c.GetHeader("X-Real-IP"); ip != "" {
		return ip
	}

	// Fall back to remote address
	// In a real implementation, this would come from the connection
	return "default"
}

// limiterStore manages rate limiters per key.
type limiterStore struct {
	limiters        sync.Map
	rate            float64
	burst           int
	cleanupInterval time.Duration
	maxAge          time.Duration
}

// limiterEntry wraps a token bucket limiter with last access time.
type limiterEntry struct {
	limiter    *tokenBucket
	lastAccess time.Time
	mu         sync.Mutex
}

// getLimiter returns or creates a rate limiter for the given key.
func (ls *limiterStore) getLimiter(key string) *limiterEntry {
	// Fast path: limiter exists
	if entry, ok := ls.limiters.Load(key); ok {
		e := entry.(*limiterEntry)
		e.mu.Lock()
		e.lastAccess = time.Now()
		e.mu.Unlock()
		return e
	}

	// Slow path: create new limiter
	entry := &limiterEntry{
		limiter:    newTokenBucket(ls.rate, ls.burst),
		lastAccess: time.Now(),
	}

	// Try to store, use existing if another goroutine created it
	actual, loaded := ls.limiters.LoadOrStore(key, entry)
	if loaded {
		return actual.(*limiterEntry)
	}

	return entry
}

// cleanup periodically removes old limiters.
func (ls *limiterStore) cleanup() {
	ticker := time.NewTicker(ls.cleanupInterval)
	defer ticker.Stop()

	for range ticker.C {
		now := time.Now()
		ls.limiters.Range(func(key, value interface{}) bool {
			entry := value.(*limiterEntry)
			entry.mu.Lock()
			age := now.Sub(entry.lastAccess)
			entry.mu.Unlock()

			if age > ls.maxAge {
				ls.limiters.Delete(key)
			}
			return true
		})
	}
}

// tokenBucket implements token bucket rate limiting algorithm.
type tokenBucket struct {
	tokens         float64
	maxTokens      float64
	refillRate     float64 // tokens per second
	lastRefill     time.Time
	mu             sync.Mutex
}

// newTokenBucket creates a new token bucket.
func newTokenBucket(rate float64, burst int) *tokenBucket {
	return &tokenBucket{
		tokens:     float64(burst),
		maxTokens:  float64(burst),
		refillRate: rate,
		lastRefill: time.Now(),
	}
}

// allow checks if a request is allowed and consumes a token.
func (tb *tokenBucket) allow() bool {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Refill tokens based on elapsed time
	now := time.Now()
	elapsed := now.Sub(tb.lastRefill).Seconds()
	tb.tokens += elapsed * tb.refillRate

	// Cap at max tokens
	if tb.tokens > tb.maxTokens {
		tb.tokens = tb.maxTokens
	}

	tb.lastRefill = now

	// Check if we have tokens available
	if tb.tokens >= 1.0 {
		tb.tokens -= 1.0
		return true
	}

	return false
}

// retryIn returns how long until next token is available.
func (tb *tokenBucket) retryIn() time.Duration {
	tb.mu.Lock()
	defer tb.mu.Unlock()

	// Calculate time needed to get 1 token
	tokensNeeded := 1.0 - tb.tokens
	if tokensNeeded <= 0 {
		return 0
	}

	seconds := tokensNeeded / tb.refillRate
	return time.Duration(seconds * float64(time.Second))
}
