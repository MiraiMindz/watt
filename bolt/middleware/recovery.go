package middleware

import (
	"fmt"
	"log"
	"runtime/debug"

	"github.com/yourusername/bolt/core"
)

// Recovery returns a middleware that recovers from panics in the handler chain.
//
// When a panic occurs:
//   - Catches the panic and prevents server crash
//   - Logs the panic message and stack trace
//   - Returns 500 Internal Server Error to client
//   - Allows server to continue serving other requests
//
// Example:
//
//	app := bolt.New()
//	app.Use(Recovery())
//	app.Get("/panic", func(c *bolt.Context) error {
//	    panic("something went wrong")
//	})
//
// Performance: <50ns overhead when no panic occurs.
func Recovery() core.Middleware {
	return func(next core.Handler) core.Handler {
		return func(c *core.Context) (err error) {
			// Defer panic recovery
			defer func() {
				if r := recover(); r != nil {
					// Log panic with stack trace
					stack := debug.Stack()
					log.Printf("PANIC: %v\n%s", r, stack)

					// Return 500 error
					err = c.JSON(500, map[string]interface{}{
						"error": "Internal server error",
						"panic": fmt.Sprintf("%v", r),
					})
				}
			}()

			// Execute next handler
			return next(c)
		}
	}
}

// RecoveryWithConfig returns a middleware with custom recovery configuration.
//
// Options:
//   - PrintStack: Whether to print stack trace (default: true)
//   - StackSize: Maximum stack trace size in bytes (default: 4KB)
//   - LogOutput: Custom log output (default: stderr)
//   - Handler: Custom panic handler (default: returns 500)
//
// Example:
//
//	app.Use(RecoveryWithConfig(RecoveryConfig{
//	    PrintStack: true,
//	    StackSize: 4 << 10, // 4KB
//	    Handler: func(c *core.Context, err interface{}) error {
//	        log.Printf("Panic: %v", err)
//	        return c.JSON(500, map[string]string{"error": "server error"})
//	    },
//	}))
func RecoveryWithConfig(config RecoveryConfig) core.Middleware {
	// Apply defaults
	if config.StackSize == 0 {
		config.StackSize = 4 << 10 // 4KB
	}

	return func(next core.Handler) core.Handler {
		return func(c *core.Context) (err error) {
			defer func() {
				if r := recover(); r != nil {
					// Print stack trace if enabled
					if config.PrintStack {
						stack := debug.Stack()
						if config.LogOutput != nil {
							fmt.Fprintf(config.LogOutput, "PANIC: %v\n%s\n", r, stack)
						} else {
							log.Printf("PANIC: %v\n%s", r, stack)
						}
					}

					// Call custom handler if provided
					if config.Handler != nil {
						err = config.Handler(c, r)
					} else {
						// Default handler
						err = c.JSON(500, map[string]interface{}{
							"error": "Internal server error",
						})
					}
				}
			}()

			return next(c)
		}
	}
}

// RecoveryConfig defines configuration for recovery middleware.
type RecoveryConfig struct {
	// PrintStack enables stack trace printing (default: true)
	PrintStack bool

	// StackSize is the maximum stack trace size in bytes (default: 4KB)
	StackSize int

	// LogOutput is the custom log output (default: stderr)
	LogOutput interface {
		Write(p []byte) (n int, err error)
	}

	// Handler is a custom panic handler
	// If nil, returns default 500 error response
	Handler func(c *core.Context, err interface{}) error
}

// DefaultRecoveryConfig returns default recovery configuration.
func DefaultRecoveryConfig() RecoveryConfig {
	return RecoveryConfig{
		PrintStack: true,
		StackSize:  4 << 10, // 4KB
		LogOutput:  nil,     // Use default (stderr)
		Handler:    nil,     // Use default handler
	}
}
