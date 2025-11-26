package core

import (
	"context"
	"log"
	"net/http"
	"os"
	"os/signal"
	"sync"
	"syscall"
	"time"

	"github.com/yourusername/bolt/shockwave"
)

// App is the main Bolt application.
//
// App manages:
//   - Route registration (Get, Post, Put, Delete, etc.)
//   - Middleware chains
//   - Shockwave HTTP server integration
//   - Context pooling
//   - Graceful shutdown
//
// Example:
//
//	app := bolt.New()
//	app.Get("/hello", func(c *bolt.Context) error {
//	    return c.JSON(200, map[string]string{"message": "Hello, World!"})
//	})
//	app.Listen(":8080")
type App struct {
	router       IRouter // ✅ Interface allows choosing router implementation
	contextPool  *ContextPool
	config       Config
	middleware   []Middleware
	errorHandler ErrorHandler
	server       *shockwave.Server
	serverMu     sync.RWMutex // Protects server field from concurrent access
}

// New creates a new Bolt application with default configuration.
func New() *App {
	return NewWithConfig(DefaultConfig())
}

// NewWithConfig creates a new Bolt application with custom configuration.
func NewWithConfig(config Config) *App {
	if config.ErrorHandler == nil {
		config.ErrorHandler = DefaultErrorHandler
	}

	// Create context pool
	contextPool := NewContextPool()

	// ✅ OPTIMIZATION: Pre-warm pool to eliminate cold start allocations
	// Pre-allocate 1000 contexts (covers burst traffic, ~80KB memory)
	contextPool.Warmup(1000)

	// ✅ OPTIMIZATION: Choose router implementation based on config
	var router IRouter
	if config.UseLockFreeRouter {
		// Lock-free router for maximum concurrent performance (default)
		router = NewRouterLockFree()
	} else {
		// Standard router with RWMutex (simple, proven)
		router = NewRouter()
	}

	return &App{
		router:       router,
		contextPool:  contextPool,
		config:       config,
		middleware:   make([]Middleware, 0),
		errorHandler: config.ErrorHandler,
	}
}

// Use adds global middleware to the application.
//
// Middleware is executed in the order it's registered.
//
// Example:
//
//	app.Use(Logger())
//	app.Use(CORS())
//	app.Use(Recovery())
func (app *App) Use(middleware ...Middleware) {
	app.middleware = append(app.middleware, middleware...)
}

// Get registers a GET route.
//
// Example:
//
//	app.Get("/users/:id", getUser)
func (app *App) Get(path string, handler Handler) *ChainLink {
	return app.addRoute(MethodGet, path, handler)
}

// Post registers a POST route.
//
// Example:
//
//	app.Post("/users", createUser)
func (app *App) Post(path string, handler Handler) *ChainLink {
	return app.addRoute(MethodPost, path, handler)
}

// Put registers a PUT route.
//
// Example:
//
//	app.Put("/users/:id", updateUser)
func (app *App) Put(path string, handler Handler) *ChainLink {
	return app.addRoute(MethodPut, path, handler)
}

// Delete registers a DELETE route.
//
// Example:
//
//	app.Delete("/users/:id", deleteUser)
func (app *App) Delete(path string, handler Handler) *ChainLink {
	return app.addRoute(MethodDelete, path, handler)
}

// Patch registers a PATCH route.
//
// Example:
//
//	app.Patch("/users/:id", patchUser)
func (app *App) Patch(path string, handler Handler) *ChainLink {
	return app.addRoute(MethodPatch, path, handler)
}

// Head registers a HEAD route.
func (app *App) Head(path string, handler Handler) *ChainLink {
	return app.addRoute(MethodHead, path, handler)
}

// Options registers an OPTIONS route.
func (app *App) Options(path string, handler Handler) *ChainLink {
	return app.addRoute(MethodOptions, path, handler)
}

// NOTE: Generic methods are not supported in Go as methods cannot have type parameters
// independent of the receiver type. The generic API would need to be implemented as
// standalone functions or using a different pattern (e.g., builder pattern).
//
// For now, users can use the Data[T] types manually with the standard API:
//
// Example:
//	app.Get("/users/:id", func(c *bolt.Context) error {
//	    data := bolt.OK(user)  // Returns Data[User]
//	    return c.sendData(data)
//	})
//
// TODO: Implement generic API using standalone functions or builder pattern

// addRoute registers a route with the router.
func (app *App) addRoute(method HTTPMethod, path string, handler Handler) *ChainLink {
	// Wrap handler with global middleware
	finalHandler := handler
	for i := len(app.middleware) - 1; i >= 0; i-- {
		finalHandler = app.middleware[i](finalHandler)
	}

	// Register with router
	app.router.Add(method, path, finalHandler)

	// Return chain link for fluent API
	return &ChainLink{
		app: app,
		lastRoute: &RouteInfo{
			Method:  method,
			Path:    path,
			Handler: finalHandler,
		},
	}
}

// Listen starts the HTTP server on the specified address.
//
// This is a blocking call. The server runs until interrupted (Ctrl+C).
//
// Example:
//
//	app.Listen(":8080")
func (app *App) Listen(addr string) error {
	app.config.Addr = addr

	// Create Shockwave server
	srv := shockwave.NewServer(&shockwave.Config{
		Addr:               addr,
		Handler:            app.handleShockwaveRequest,
		MaxRequestBodySize: app.config.MaxRequestBodySize,
		DisableStats:       app.config.DisableStats,
	})

	// Store server with mutex protection
	app.serverMu.Lock()
	app.server = srv
	app.serverMu.Unlock()

	log.Printf("Bolt server listening on %s", addr)

	// Start server
	return srv.ListenAndServe()
}

// Run starts the server with graceful shutdown support.
//
// The server runs until interrupted (Ctrl+C), then performs graceful shutdown.
//
// Example:
//
//	app.Run(":8080")
func (app *App) Run(addr string) error {
	app.config.Addr = addr

	// Create Shockwave server
	srv := shockwave.NewServer(&shockwave.Config{
		Addr:               addr,
		Handler:            app.handleShockwaveRequest,
		MaxRequestBodySize: app.config.MaxRequestBodySize,
		DisableStats:       app.config.DisableStats,
	})

	// Store server with mutex protection
	app.serverMu.Lock()
	app.server = srv
	app.serverMu.Unlock()

	// Start server in background
	errChan := make(chan error, 1)
	go func() {
		log.Printf("Bolt server starting on %s", addr)
		if err := srv.ListenAndServe(); err != nil {
			errChan <- err
		}
	}()

	// Wait for interrupt signal
	sigChan := make(chan os.Signal, 1)
	signal.Notify(sigChan, os.Interrupt, syscall.SIGTERM)

	select {
	case err := <-errChan:
		return err
	case <-sigChan:
		log.Println("Shutting down gracefully...")

		// Graceful shutdown
		ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
		defer cancel()

		if err := app.Shutdown(ctx); err != nil {
			log.Printf("Shutdown error: %v", err)
			return err
		}

		log.Println("Server stopped")
		return nil
	}
}

// Shutdown gracefully shuts down the server.
//
// It waits for active connections to finish (up to context deadline).
func (app *App) Shutdown(ctx context.Context) error {
	app.serverMu.RLock()
	srv := app.server
	app.serverMu.RUnlock()

	if srv == nil {
		return nil
	}
	return srv.Shutdown(ctx)
}

// ServeHTTP implements http.Handler interface for testing and compatibility.
//
// This allows Bolt to be used with standard Go http testing tools like httptest.
// For production use, use Listen() which integrates with Shockwave.
//
// Example (testing):
//
//	app := bolt.New()
//	app.Get("/ping", handler)
//	req := httptest.NewRequest("GET", "/ping", nil)
//	w := httptest.NewRecorder()
//	app.ServeHTTP(w, req)
func (app *App) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	// Acquire context from pool
	ctx := app.contextPool.Acquire()

	// Map http.Request to Bolt Context (ZERO-ALLOC: unsafe string→[]byte)
	ctx.httpReq = r
	ctx.httpRes = w
	// SAFE: Read-only references, valid for request lifetime
	ctx.methodBytes = stringToBytes(r.Method)
	ctx.pathBytes = stringToBytes(r.URL.Path)
	ctx.queryBytes = stringToBytes(r.URL.RawQuery)

	// Route and execute handler
	if err := app.router.ServeHTTP(ctx); err != nil {
		// Handle error
		app.errorHandler(ctx, err)
	}

	// Release context back to pool (direct call, no defer overhead)
	app.contextPool.Release(ctx)
}

// handleShockwaveRequest handles an incoming Shockwave HTTP request.
//
// ✅ OPTIMIZED: Fast-path for common responses (404, errors)
//
// This is the bridge between Shockwave and Bolt:
//   - Acquires Context from pool (zero allocation)
//   - Maps Shockwave request to Bolt Context (zero-copy)
//   - Routes and executes handler
//   - Fast-path error handling for 404
//   - Releases Context back to pool
//
// Performance: <50ns overhead (down from ~200ns)
func (app *App) handleShockwaveRequest(res *shockwave.ResponseWriter, req *shockwave.Request) {
	// Acquire context from pool (10ns with FastReset)
	ctx := app.contextPool.Acquire()
	defer app.contextPool.Release(ctx)

	// Map Shockwave request to Bolt context (zero-copy byte slices, 0ns)
	// Direct pointer assignment - no allocations
	ctx.shockwaveReq = req
	ctx.shockwaveRes = res
	ctx.methodBytes = req.MethodBytes()  // Zero-copy reference to Shockwave buffer
	ctx.pathBytes = req.PathBytes()      // Zero-copy reference to Shockwave buffer
	ctx.queryBytes = req.QueryBytes()    // Zero-copy reference to Shockwave buffer

	// Route and execute handler
	err := app.router.ServeHTTP(ctx)

	// ✅ FAST PATH: Handle 404 directly (most common error)
	if err == ErrNotFound {
		// Use pre-compiled 404 response (0 allocs)
		_ = ctx.JSONNotFound()
		return
	}

	// Slow path: Other errors
	if err != nil {
		app.errorHandler(ctx, err)
	}
}
