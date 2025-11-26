package core

// IRouter defines the interface that all router implementations must satisfy.
//
// This allows switching between different router implementations:
//   - Router: Standard router with RWMutex (simple, proven)
//   - RouterLockFree: Lock-free router with atomic.Value (faster, concurrent)
type IRouter interface {
	// Add registers a route with the given method, path, and handler
	Add(method HTTPMethod, path string, handler Handler)

	// Lookup finds a handler for the given method and path
	Lookup(method HTTPMethod, path string) (Handler, map[string]string)

	// LookupBytes finds a handler using byte slices (zero-allocation)
	LookupBytes(method HTTPMethod, pathBytes []byte) (Handler, [8]ParamPair, int)

	// ServeHTTP handles an HTTP request using the router
	ServeHTTP(c *Context) error
}

// Verify that both implementations satisfy the interface
var (
	_ IRouter = (*Router)(nil)
	_ IRouter = (*RouterLockFree)(nil)
)
