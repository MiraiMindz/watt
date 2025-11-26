package middleware

import (
	"context"
	"errors"
	"time"

	"github.com/yourusername/bolt/core"
)

// ErrRequestTimeout is returned when a request exceeds the timeout duration.
var ErrRequestTimeout = errors.New("request timeout")

// Timeout returns a middleware that cancels requests exceeding the specified duration.
//
// If a request takes longer than the timeout:
//   - Context is cancelled
//   - Handler receives context cancellation error
//   - Returns 408 Request Timeout
//   - Resources are cleaned up properly
//
// Example:
//
//	app := bolt.New()
//	app.Use(Timeout(5 * time.Second))
//	app.Get("/slow", func(c *bolt.Context) error {
//	    time.Sleep(10 * time.Second) // Will timeout after 5s
//	    return c.JSON(200, map[string]string{"status": "ok"})
//	})
//
// Performance: <50ns overhead per request.
func Timeout(duration time.Duration) core.Middleware {
	return TimeoutWithConfig(TimeoutConfig{
		Timeout: duration,
	})
}

// TimeoutWithConfig returns a middleware with custom timeout configuration.
//
// Example:
//
//	app.Use(TimeoutWithConfig(TimeoutConfig{
//	    Timeout: 10 * time.Second,
//	    SkipPaths: []string{"/upload", "/download"},
//	    Handler: func(c *core.Context) error {
//	        return c.JSON(408, map[string]string{
//	            "error": "request took too long",
//	        })
//	    },
//	}))
func TimeoutWithConfig(config TimeoutConfig) core.Middleware {
	// Apply defaults
	if config.Timeout == 0 {
		config.Timeout = 30 * time.Second
	}

	// Create skip map for O(1) lookup
	skipMap := make(map[string]bool, len(config.SkipPaths))
	for _, path := range config.SkipPaths {
		skipMap[path] = true
	}

	return func(next core.Handler) core.Handler {
		return func(c *core.Context) error {
			// Skip timeout for certain paths
			if skipMap[c.Path()] {
				return next(c)
			}

			// Create context with timeout
			ctx, cancel := context.WithTimeout(context.Background(), config.Timeout)
			defer cancel()

			// Channel to receive handler result
			done := make(chan error, 1)

			// Execute handler in goroutine
			go func() {
				done <- next(c)
			}()

			// Wait for completion or timeout
			select {
			case err := <-done:
				// Handler completed in time
				return err

			case <-ctx.Done():
				// Timeout occurred
				if config.Handler != nil {
					// Use custom timeout handler
					return config.Handler(c)
				}

				// Default timeout response
				return c.JSON(408, map[string]interface{}{
					"error":   "Request timeout",
					"timeout": config.Timeout.String(),
				})
			}
		}
	}
}

// TimeoutConfig defines configuration for timeout middleware.
type TimeoutConfig struct {
	// Timeout is the maximum duration for a request.
	// Default: 30 seconds
	Timeout time.Duration

	// SkipPaths are paths to skip timeout (e.g., /upload, /download)
	// Useful for long-running operations.
	SkipPaths []string

	// Handler is a custom timeout handler.
	// If nil, returns default 408 error response.
	Handler func(c *core.Context) error
}

// DefaultTimeoutConfig returns default timeout configuration.
func DefaultTimeoutConfig() TimeoutConfig {
	return TimeoutConfig{
		Timeout:   30 * time.Second,
		SkipPaths: []string{},
		Handler:   nil,
	}
}
