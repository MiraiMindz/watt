package server

import (
	"context"
	"crypto/tls"
	"io"
	"net"
	"sync"
	"sync/atomic"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave/http11"
)

// Handler is the primary request handler function using concrete types.
// This avoids interface conversion allocations for zero-allocation operation.
// It receives concrete http11 types directly for maximum performance.
type Handler func(w *http11.ResponseWriter, r *http11.Request)

// LegacyHandler handles HTTP requests using interfaces (for backward compatibility).
// It receives a Request and ResponseWriter interface and should write the response.
// Use Handler (FastHandler) for better performance.
type LegacyHandler interface {
	ServeHTTP(w ResponseWriter, r Request)
}

// LegacyHandlerFunc is an adapter to allow the use of ordinary functions as HTTP handlers.
type LegacyHandlerFunc func(ResponseWriter, Request)

// ServeHTTP calls f(w, r).
func (f LegacyHandlerFunc) ServeHTTP(w ResponseWriter, r Request) {
	f(w, r)
}

// Request represents an HTTP request (interface to http11.Request)
type Request interface {
	Method() string
	Path() string
	Proto() string
	Header() Header
	Body() io.Reader
	Close() bool
}

// ResponseWriter represents an HTTP response writer (interface to http11.ResponseWriter)
type ResponseWriter interface {
	Header() Header
	WriteHeader(statusCode int)
	Write(data []byte) (int, error)
	WriteString(s string) (int, error)
	WriteJSON(statusCode int, data []byte) error
	Flush() error
}

// Header represents HTTP headers (interface to http11.Header)
type Header interface {
	Get(key string) string
	Set(key, value string)
	Add(key, value string)
	Del(key string)
	Clone() Header
}

// Server represents an HTTP server
type Server interface {
	// ListenAndServe listens on the configured address and serves requests
	ListenAndServe() error

	// ListenAndServeTLS listens on the configured address with TLS
	ListenAndServeTLS(certFile, keyFile string) error

	// Serve accepts incoming connections on the Listener
	Serve(l net.Listener) error

	// ServeTLS accepts incoming connections on the Listener with TLS
	ServeTLS(l net.Listener, certFile, keyFile string) error

	// Shutdown gracefully shuts down the server
	Shutdown(ctx context.Context) error

	// Close immediately closes all active connections
	Close() error

	// Stats returns server statistics
	Stats() *Stats
}

// Config holds server configuration
type Config struct {
	// Addr is the TCP address to listen on (e.g., ":8080")
	// Default: ":8080"
	Addr string

	// Handler is the primary request handler (uses concrete types for zero allocations)
	// This is the recommended handler type for maximum performance.
	// Either Handler or LegacyHandler must be set.
	// Example: Handler: func(w *http11.ResponseWriter, r *http11.Request) { w.WriteHeader(200) }
	Handler Handler

	// LegacyHandler is for backward compatibility (uses interface types)
	// This incurs 1 allocation per request due to interface conversions.
	// Use Handler instead for zero-allocation operation.
	LegacyHandler LegacyHandler

	// ReadTimeout is the maximum duration for reading the entire request
	// Default: 60 seconds
	ReadTimeout time.Duration

	// WriteTimeout is the maximum duration before timing out writes of the response
	// Default: 60 seconds
	WriteTimeout time.Duration

	// IdleTimeout is the maximum amount of time to wait for the next request
	// when keep-alive is enabled
	// Default: 120 seconds
	IdleTimeout time.Duration

	// MaxHeaderBytes controls the maximum number of bytes the server will
	// read parsing the request header's keys and values
	// Default: 1 MB
	MaxHeaderBytes int

	// MaxRequestBodySize is the maximum size of a request body
	// Default: 10 MB
	MaxRequestBodySize int

	// MaxKeepAliveRequests is the maximum number of requests per connection
	// 0 means unlimited
	// Default: 0 (unlimited)
	MaxKeepAliveRequests int

	// TLSConfig optionally provides a TLS configuration
	TLSConfig *tls.Config

	// ReadBufferSize is the size of the read buffer per connection
	// Default: 4096 bytes
	ReadBufferSize int

	// WriteBufferSize is the size of the write buffer per connection
	// Default: 4096 bytes
	WriteBufferSize int

	// MaxConcurrentConnections is the maximum number of concurrent connections
	// 0 means unlimited
	// Default: 0 (unlimited)
	MaxConcurrentConnections int

	// DisableKeepalive disables keep-alive connections
	// Default: false (keep-alive enabled)
	DisableKeepalive bool

	// EnableStats enables request time tracking (causes 1 allocation per request)
	// Set to false for zero-allocation operation (like fasthttp)
	// Default: false (stats disabled for zero allocations)
	EnableStats bool

	// AllocationMode specifies the memory allocation strategy
	// Options: "standard" (default), "arena", "greentea"
	// Default: "standard"
	AllocationMode string
}

// DefaultConfig returns the default server configuration
func DefaultConfig() Config {
	return Config{
		Addr:                     ":8080",
		ReadTimeout:              60 * time.Second,
		WriteTimeout:             60 * time.Second,
		IdleTimeout:              120 * time.Second,
		MaxHeaderBytes:           1 << 20, // 1 MB
		MaxRequestBodySize:       10 << 20, // 10 MB
		MaxKeepAliveRequests:     0, // Unlimited
		ReadBufferSize:           4096,
		WriteBufferSize:          4096,
		MaxConcurrentConnections: 0, // Unlimited
		DisableKeepalive:         false,
		AllocationMode:           "standard",
	}
}

// Stats represents server statistics
type Stats struct {
	// Total number of connections accepted
	TotalConnections atomic.Uint64

	// Current number of active connections
	ActiveConnections atomic.Int64

	// Total number of requests handled
	TotalRequests atomic.Uint64

	// Total number of bytes read
	BytesRead atomic.Uint64

	// Total number of bytes written
	BytesWritten atomic.Uint64

	// Number of connection errors
	ConnectionErrors atomic.Uint64

	// Number of request errors
	RequestErrors atomic.Uint64

	// Server start time
	StartTime time.Time

	// Last request time
	LastRequestTime atomic.Value // time.Time
}

// Duration returns the time since the server started
func (s *Stats) Duration() time.Duration {
	return time.Since(s.StartTime)
}

// RequestsPerSecond returns the average requests per second
func (s *Stats) RequestsPerSecond() float64 {
	duration := s.Duration().Seconds()
	if duration == 0 {
		return 0
	}
	return float64(s.TotalRequests.Load()) / duration
}

// ConnectionsPerSecond returns the average connections per second
func (s *Stats) ConnectionsPerSecond() float64 {
	duration := s.Duration().Seconds()
	if duration == 0 {
		return 0
	}
	return float64(s.TotalConnections.Load()) / duration
}

// BaseServer provides common server functionality
type BaseServer struct {
	config   Config
	listener net.Listener
	stats    Stats

	// Shutdown coordination
	mu       sync.RWMutex
	shutdown atomic.Bool
	done     chan struct{}
	wg       sync.WaitGroup

	// Connection tracking
	conns   map[net.Conn]struct{}
	connsMu sync.Mutex

	// Connection semaphore (for limiting concurrent connections)
	connSem chan struct{}
}

// NewBaseServer creates a new base server
func NewBaseServer(config Config) *BaseServer {
	if config.Handler == nil && config.LegacyHandler == nil {
		panic("server: either Handler or LegacyHandler is required")
	}

	// Apply defaults
	if config.Addr == "" {
		config.Addr = ":8080"
	}
	if config.ReadTimeout == 0 {
		config.ReadTimeout = 60 * time.Second
	}
	if config.WriteTimeout == 0 {
		config.WriteTimeout = 60 * time.Second
	}
	if config.IdleTimeout == 0 {
		config.IdleTimeout = 120 * time.Second
	}
	if config.MaxHeaderBytes == 0 {
		config.MaxHeaderBytes = 1 << 20 // 1 MB
	}
	if config.MaxRequestBodySize == 0 {
		config.MaxRequestBodySize = 10 << 20 // 10 MB
	}
	if config.ReadBufferSize == 0 {
		config.ReadBufferSize = 4096
	}
	if config.WriteBufferSize == 0 {
		config.WriteBufferSize = 4096
	}

	s := &BaseServer{
		config: config,
		done:   make(chan struct{}),
		conns:  make(map[net.Conn]struct{}),
	}

	s.stats.StartTime = time.Now()
	s.stats.LastRequestTime.Store(time.Now())

	// Create connection semaphore if limit is set
	if config.MaxConcurrentConnections > 0 {
		s.connSem = make(chan struct{}, config.MaxConcurrentConnections)
	}

	return s
}

// Stats returns server statistics
func (s *BaseServer) Stats() *Stats {
	return &s.stats
}

// trackConnection adds a connection to tracking
func (s *BaseServer) trackConnection(conn net.Conn) {
	s.connsMu.Lock()
	s.conns[conn] = struct{}{}
	s.connsMu.Unlock()

	s.stats.ActiveConnections.Add(1)
}

// untrackConnection removes a connection from tracking
func (s *BaseServer) untrackConnection(conn net.Conn) {
	s.connsMu.Lock()
	delete(s.conns, conn)
	s.connsMu.Unlock()

	s.stats.ActiveConnections.Add(-1)
}

// closeAllConnections closes all tracked connections
func (s *BaseServer) closeAllConnections() {
	s.connsMu.Lock()
	conns := make([]net.Conn, 0, len(s.conns))
	for conn := range s.conns {
		conns = append(conns, conn)
	}
	s.connsMu.Unlock()

	for _, conn := range conns {
		conn.Close()
	}
}

// Shutdown gracefully shuts down the server
func (s *BaseServer) Shutdown(ctx context.Context) error {
	if !s.shutdown.CompareAndSwap(false, true) {
		return nil // Already shutting down
	}

	// Close listener to stop accepting new connections
	if s.listener != nil {
		s.listener.Close()
	}

	// Signal shutdown
	close(s.done)

	// Wait for connections to close or context to expire
	shutdownComplete := make(chan struct{})
	go func() {
		s.wg.Wait()
		close(shutdownComplete)
	}()

	select {
	case <-shutdownComplete:
		return nil
	case <-ctx.Done():
		// Context expired, force close all connections
		s.closeAllConnections()
		return ctx.Err()
	}
}

// Close immediately closes the server and all active connections
func (s *BaseServer) Close() error {
	if !s.shutdown.CompareAndSwap(false, true) {
		return nil // Already closed
	}

	// Close listener
	if s.listener != nil {
		s.listener.Close()
	}

	// Signal shutdown
	close(s.done)

	// Force close all connections
	s.closeAllConnections()

	// Wait for goroutines to finish
	s.wg.Wait()

	return nil
}
