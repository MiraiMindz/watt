package http2

import (
	"sync"
	"time"
)

// ConnectionConfig holds configuration for HTTP/2 connections
type ConnectionConfig struct {
	// Buffer limits
	MaxStreamBufferSize   int64 // Maximum buffer size per stream (default: 1MB)
	MaxConnectionBuffer   int64 // Maximum total buffer size for connection (default: 10MB)

	// Rate limiting
	MaxPriorityUpdatesPerSecond int   // Maximum PRIORITY frames per second (default: 100)
	PriorityRateLimitWindow     time.Duration // Rate limit window (default: 1s)

	// Timeouts
	StreamIdleTimeout     time.Duration // Stream idle timeout (default: 5min)
	ConnectionIdleTimeout time.Duration // Connection idle timeout (default: 10min)
	PingTimeout           time.Duration // PING response timeout (default: 30s)

	// Flow control
	EnableBackpressure    bool  // Enable connection-level backpressure (default: true)
	BackpressureThreshold int32 // Backpressure threshold (default: 16KB)
}

// DefaultConnectionConfig returns the default configuration
func DefaultConnectionConfig() *ConnectionConfig {
	return &ConnectionConfig{
		MaxStreamBufferSize:         1024 * 1024,       // 1MB
		MaxConnectionBuffer:         10 * 1024 * 1024,  // 10MB
		MaxPriorityUpdatesPerSecond: 100,
		PriorityRateLimitWindow:     time.Second,
		StreamIdleTimeout:           5 * time.Minute,
		ConnectionIdleTimeout:       10 * time.Minute,
		PingTimeout:                 30 * time.Second,
		EnableBackpressure:          true,
		BackpressureThreshold:       16384, // 16KB
	}
}

// Validate validates the configuration
func (c *ConnectionConfig) Validate() error {
	if c.MaxStreamBufferSize <= 0 {
		return ErrInvalidSettings
	}
	if c.MaxConnectionBuffer <= 0 {
		return ErrInvalidSettings
	}
	if c.MaxPriorityUpdatesPerSecond < 0 {
		return ErrInvalidSettings
	}
	if c.PriorityRateLimitWindow <= 0 {
		c.PriorityRateLimitWindow = time.Second
	}
	if c.StreamIdleTimeout <= 0 {
		c.StreamIdleTimeout = 5 * time.Minute
	}
	if c.ConnectionIdleTimeout <= 0 {
		c.ConnectionIdleTimeout = 10 * time.Minute
	}
	if c.PingTimeout <= 0 {
		c.PingTimeout = 30 * time.Second
	}
	if c.BackpressureThreshold <= 0 {
		c.BackpressureThreshold = 16384
	}
	return nil
}

// rateLimiter tracks rate-limited operations
type rateLimiter struct {
	count        int
	window       time.Duration
	lastReset    time.Time
	maxPerWindow int
	mu           sync.Mutex // Protects count and lastReset
}

// newRateLimiter creates a new rate limiter
func newRateLimiter(maxPerWindow int, window time.Duration) *rateLimiter {
	return &rateLimiter{
		count:        0,
		window:       window,
		lastReset:    time.Now(),
		maxPerWindow: maxPerWindow,
	}
}

// allow checks if the operation is allowed
func (rl *rateLimiter) allow() bool {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	now := time.Now()

	// Reset window if expired
	if now.Sub(rl.lastReset) >= rl.window {
		rl.count = 0
		rl.lastReset = now
	}

	// Check limit
	if rl.count >= rl.maxPerWindow {
		return false
	}

	rl.count++
	return true
}

// reset resets the rate limiter
func (rl *rateLimiter) reset() {
	rl.mu.Lock()
	defer rl.mu.Unlock()

	rl.count = 0
	rl.lastReset = time.Now()
}
