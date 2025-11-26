//go:build !goexperiment.arenas && !greenteagc

package server

import (
	"fmt"
	"io"
	"net"
	"sync"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave/http11"
)

// Adapter pools for zero-allocation request handling
// Note: These pools are kept for backward compatibility but the server now uses
// embedded adapters in adapterPair for true zero-allocation operation.
var (
	requestAdapterPool = sync.Pool{
		New: func() interface{} {
			return &requestAdapter{}
		},
	}

	responseWriterAdapterPool = sync.Pool{
		New: func() interface{} {
			return &responseWriterAdapter{}
		},
	}

	headerAdapterPool = sync.Pool{
		New: func() interface{} {
			return &headerAdapter{}
		},
	}

	// adapterPairPool provides pre-allocated adapter pairs for fallback
	adapterPairPool = sync.Pool{
		New: func() interface{} {
			return &adapterPair{}
		},
	}
)

// ShockwaveServer is the main HTTP/1.1 server implementation using standard pooling
type ShockwaveServer struct {
	*BaseServer
	// Shared handler for all connections (created once at server init)
	sharedHandler http11.Handler
}

// NewServer creates a new Shockwave HTTP server with standard pooling
func NewServer(config Config) Server {
	base := NewBaseServer(config)
	srv := &ShockwaveServer{
		BaseServer: base,
	}

	// Create shared handler once for all connections (zero per-connection allocation)
	if config.Handler != nil {
		srv.sharedHandler = func(req *http11.Request, rw *http11.ResponseWriter) error {
			// Update stats (counter is zero-allocation, time tracking allocates)
			srv.stats.TotalRequests.Add(1)

			// Only track time if stats are enabled (allocation-free when disabled)
			if srv.config.EnableStats {
				srv.stats.LastRequestTime.Store(time.Now())
			}

			// Call handler directly (zero allocations)
			srv.config.Handler(rw, req)

			// Check if connection should close
			if req.Close {
				return fmt.Errorf("connection close requested")
			}

			return nil
		}
	} else {
		// Legacy handler path - still creates adapters per connection
		// (This will be used in handleConnection as before)
		srv.sharedHandler = nil
	}

	return srv
}

// ListenAndServe listens on the configured address and serves requests
func (s *ShockwaveServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.Serve(ln)
}

// ListenAndServeTLS listens on the configured address with TLS
func (s *ShockwaveServer) ListenAndServeTLS(certFile, keyFile string) error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.ServeTLS(ln, certFile, keyFile)
}

// Serve accepts incoming connections on the Listener
func (s *ShockwaveServer) Serve(l net.Listener) error {
	s.listener = l
	defer l.Close()

	for {
		// Check if shutting down
		if s.shutdown.Load() {
			return nil
		}

		// Acquire connection slot if limit is set
		if s.connSem != nil {
			select {
			case s.connSem <- struct{}{}:
			case <-s.done:
				return nil
			}
		}

		// Accept connection
		conn, err := l.Accept()
		if err != nil {
			if s.shutdown.Load() {
				return nil
			}
			s.stats.ConnectionErrors.Add(1)

			// Release connection slot
			if s.connSem != nil {
				<-s.connSem
			}
			continue
		}

		s.stats.TotalConnections.Add(1)

		// Handle connection in goroutine
		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// ServeTLS accepts incoming connections on the Listener with TLS
func (s *ShockwaveServer) ServeTLS(l net.Listener, certFile, keyFile string) error {
	// TODO: Implement TLS support
	return fmt.Errorf("TLS not yet implemented")
}

// handleConnection handles a single connection with keep-alive support
func (s *ShockwaveServer) handleConnection(netConn net.Conn) {
	defer s.wg.Done()
	defer netConn.Close()

	// Release connection slot when done
	if s.connSem != nil {
		defer func() { <-s.connSem }()
	}

	// Track connection
	s.trackConnection(netConn)
	defer s.untrackConnection(netConn)

	// Create HTTP/1.1 connection with keep-alive support
	connConfig := http11.ConnectionConfig{
		KeepAliveTimeout: s.config.IdleTimeout,
		MaxRequests:      s.config.MaxKeepAliveRequests,
		ReadBufferSize:   s.config.ReadBufferSize,
		WriteBufferSize:  s.config.WriteBufferSize,
	}

	if s.config.DisableKeepalive {
		connConfig.MaxRequests = 1 // Only one request per connection
	}

	// Set connection-level timeouts (applies to all requests on this connection)
	if s.config.ReadTimeout > 0 {
		netConn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
	}
	if s.config.WriteTimeout > 0 {
		netConn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
	}

	// Use shared handler if available (created once at server init, zero per-connection allocation)
	var handler http11.Handler
	if s.sharedHandler != nil {
		handler = s.sharedHandler
	} else {
		// LegacyHandler path - create per-connection handler with adapters
		var adapters adapterPair

		handler = func(req *http11.Request, rw *http11.ResponseWriter) error {
			// Update stats
			s.stats.TotalRequests.Add(1)

			if s.config.EnableStats {
				s.stats.LastRequestTime.Store(time.Now())
			}

			// Setup embedded adapters (zero allocations)
			adapters.Setup(req, rw)

			// Call user handler (1 allocation for interface conversion)
			s.config.LegacyHandler.ServeHTTP(&adapters.rwAdapter, &adapters.reqAdapter)

			// Reset adapters for next request
			adapters.Reset()

			// Check if connection should close
			if req.Close {
				return fmt.Errorf("connection close requested")
			}

			return nil
		}
	}

	// Create connection with handler
	conn := http11.NewConnection(netConn, connConfig, handler)
	defer conn.Close()

	// Serve requests on this connection (handles keep-alive internally)
	err := conn.Serve()

	// Log error if not EOF (clean close)
	if err != nil {
		s.stats.RequestErrors.Add(1)
	}
}

// requestAdapter adapts http11.Request to server.Request interface
type requestAdapter struct {
	req *http11.Request
}

func (r *requestAdapter) Method() string {
	return r.req.Method()
}

func (r *requestAdapter) Path() string {
	return r.req.Path()
}

func (r *requestAdapter) Proto() string {
	return r.req.Proto
}

func (r *requestAdapter) Header() Header {
	// Return header adapter (allocates only if Header() is called)
	h := headerAdapterPool.Get().(*headerAdapter)
	h.h = &r.req.Header
	return h
}

func (r *requestAdapter) Body() io.Reader {
	return r.req.Body
}

func (r *requestAdapter) Close() bool {
	return r.req.Close
}
