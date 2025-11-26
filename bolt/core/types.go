package core

import (
	"context"
	"errors"
)

// HTTPMethod represents an HTTP method.
type HTTPMethod string

// HTTP methods supported by Bolt.
const (
	MethodGet     HTTPMethod = "GET"
	MethodPost    HTTPMethod = "POST"
	MethodPut     HTTPMethod = "PUT"
	MethodDelete  HTTPMethod = "DELETE"
	MethodPatch   HTTPMethod = "PATCH"
	MethodHead    HTTPMethod = "HEAD"
	MethodOptions HTTPMethod = "OPTIONS"
	MethodConnect HTTPMethod = "CONNECT"
	MethodTrace   HTTPMethod = "TRACE"
)

// Handler defines a standard request handler function.
//
// Handlers receive a Context and return an error. If an error is returned,
// the framework's error handler processes it.
//
// Example:
//
//	func getUser(c *bolt.Context) error {
//	    user, err := db.GetUser(c.Param("id"))
//	    if err != nil {
//	        return err
//	    }
//	    return c.JSON(200, user)
//	}
type Handler func(*Context) error

// Middleware wraps a Handler to provide cross-cutting functionality.
//
// Middleware can:
//   - Execute code before the handler (authentication, logging)
//   - Execute code after the handler (response modification, cleanup)
//   - Short-circuit the handler (return early)
//   - Modify the context (add values, set headers)
//
// Example:
//
//	func Logger() Middleware {
//	    return func(next Handler) Handler {
//	        return func(c *Context) error {
//	            start := time.Now()
//	            err := next(c)
//	            log.Printf("%s %s - %v", c.Method(), c.Path(), time.Since(start))
//	            return err
//	        }
//	    }
//	}
type Middleware func(Handler) Handler

// ErrorHandler handles errors returned by handlers.
//
// The default error handler sends a 500 Internal Server Error response.
// Custom error handlers can provide more sophisticated error handling.
//
// Example:
//
//	func customErrorHandler(c *Context, err error) {
//	    if errors.Is(err, ErrNotFound) {
//	        c.JSON(404, map[string]string{"error": "not found"})
//	        return
//	    }
//	    c.JSON(500, map[string]string{"error": "internal server error"})
//	}
type ErrorHandler func(*Context, error)

// Common errors returned by the framework.
var (
	// ErrNotFound is returned when a resource is not found.
	ErrNotFound = errors.New("not found")

	// ErrBadRequest is returned for malformed requests.
	ErrBadRequest = errors.New("bad request")

	// ErrUnauthorized is returned for authentication failures.
	ErrUnauthorized = errors.New("unauthorized")

	// ErrForbidden is returned for authorization failures.
	ErrForbidden = errors.New("forbidden")

	// ErrMethodNotAllowed is returned when HTTP method is not supported.
	ErrMethodNotAllowed = errors.New("method not allowed")

	// ErrRequestTooLarge is returned when request body exceeds limits.
	ErrRequestTooLarge = errors.New("request too large")

	// ErrInternalServerError is returned for internal errors.
	ErrInternalServerError = errors.New("internal server error")
)

// RouteInfo contains metadata about a registered route.
type RouteInfo struct {
	Method  HTTPMethod
	Path    string
	Handler Handler
}

// ChainLink allows fluent API for route configuration.
//
// Example:
//
//	app.Get("/users", listUsers).
//	    Use(AuthMiddleware()).
//	    Use(RateLimitMiddleware())
type ChainLink struct {
	app       *App
	lastRoute *RouteInfo
}

// Use adds middleware to the last registered route.
//
// Example:
//
//	app.Get("/admin", adminHandler).
//	    Use(AuthMiddleware()).
//	    Use(AdminMiddleware())
func (cl *ChainLink) Use(middleware ...Middleware) *ChainLink {
	if cl.lastRoute != nil && cl.app != nil {
		// Wrap the handler with middleware (in reverse order)
		handler := cl.lastRoute.Handler
		for i := len(middleware) - 1; i >= 0; i-- {
			handler = middleware[i](handler)
		}
		cl.lastRoute.Handler = handler

		// Re-register the route with the updated handler
		// This overwrites the previous registration
		cl.app.router.Add(cl.lastRoute.Method, cl.lastRoute.Path, handler)
	}
	return cl
}

// Config holds application configuration.
type Config struct {
	// Server address (default: ":8080")
	Addr string

	// Error handler (default: DefaultErrorHandler)
	ErrorHandler ErrorHandler

	// Context for graceful shutdown
	ShutdownContext context.Context

	// Maximum request body size (default: 10MB)
	// Uses int to match Shockwave's Config type
	MaxRequestBodySize int

	// Enable request logging (default: false)
	EnableLogging bool

	// Disable stats collection for zero-allocation mode
	DisableStats bool

	// ✅ OPTIMIZATION: Use lock-free router (optional, disabled by default)
	// Lock-free router uses atomic.Value for zero-contention reads
	// Phase 2 testing showed RWMutex router is faster for most workloads
	// Set to true for experimentation or high-concurrency edge cases
	// Recommended: false (default) for best performance
	UseLockFreeRouter bool
}

// DefaultConfig returns the default configuration.
func DefaultConfig() Config {
	return Config{
		Addr:               ":8080",
		ErrorHandler:       DefaultErrorHandler,
		MaxRequestBodySize: 10 << 20, // 10MB
		EnableLogging:      false,
		DisableStats:       true,  // Zero-allocation mode by default
		UseLockFreeRouter:  false, // ✅ RWMutex router (faster for most workloads)
	}
}

// DefaultErrorHandler is the default error handler.
//
// It sends a 500 Internal Server Error for all errors.
// Override with custom error handler for better error handling.
func DefaultErrorHandler(c *Context, err error) {
	// Map common errors to HTTP status codes
	status := 500
	message := "Internal Server Error"

	switch {
	case errors.Is(err, ErrNotFound):
		status = 404
		message = "Not Found"
	case errors.Is(err, ErrBadRequest):
		status = 400
		message = "Bad Request"
	case errors.Is(err, ErrUnauthorized):
		status = 401
		message = "Unauthorized"
	case errors.Is(err, ErrForbidden):
		status = 403
		message = "Forbidden"
	case errors.Is(err, ErrMethodNotAllowed):
		status = 405
		message = "Method Not Allowed"
	case errors.Is(err, ErrRequestTooLarge):
		status = 413
		message = "Request Too Large"
	}

	// Send JSON error response
	c.JSON(status, map[string]string{
		"error": message,
	})
}
