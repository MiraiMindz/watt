//go:build !goexperiment.arenas && greenteagc

package server

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave"
	"github.com/yourusername/shockwave/pkg/shockwave/http11"
)

// GreenTeaServer is an HTTP/1.1 server implementation using Green Tea GC
// This provides improved cache locality and reduced GC overhead through generational pooling
type GreenTeaServer struct {
	*BaseServer
	pool *shockwave.GreenTeaPool
}

// NewServer creates a new Shockwave HTTP server with Green Tea GC
func NewServer(config Config) Server {
	base := NewBaseServer(config)
	return &GreenTeaServer{
		BaseServer: base,
		pool:       shockwave.GlobalGreenTeaPool,
	}
}

// ListenAndServe listens on the configured address and serves requests
func (s *GreenTeaServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.Serve(ln)
}

// ListenAndServeTLS listens on the configured address with TLS
func (s *GreenTeaServer) ListenAndServeTLS(certFile, keyFile string) error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.ServeTLS(ln, certFile, keyFile)
}

// Serve accepts incoming connections on the Listener
func (s *GreenTeaServer) Serve(l net.Listener) error {
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
func (s *GreenTeaServer) ServeTLS(l net.Listener, certFile, keyFile string) error {
	// TODO: Implement TLS support
	return fmt.Errorf("TLS not yet implemented")
}

// handleConnection handles a single connection with Green Tea GC
func (s *GreenTeaServer) handleConnection(netConn net.Conn) {
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
		connConfig.MaxRequests = 1
	}

	conn := http11.NewConnection(netConn, connConfig)
	defer conn.Close()

	// Serve requests on this connection with Green Tea GC
	err := conn.Serve(func(req *http11.Request, rw *http11.ResponseWriter) error {
		// Get request generation from pool
		reqGen := s.pool.GetRequestGeneration()
		defer s.pool.PutRequestGeneration(reqGen)

		// Update stats
		s.stats.TotalRequests.Add(1)
		s.stats.LastRequestTime.Store(time.Now())

		// Set timeouts
		if s.config.ReadTimeout > 0 {
			netConn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
		}
		if s.config.WriteTimeout > 0 {
			netConn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
		}

		// Copy request data into generation for cache locality
		greenTeaReq := &greenTeaRequest{
			gen:    reqGen,
			method: req.Method(),
			path:   req.Path(),
			proto:  req.Proto,
			header: &greenTeaHeader{
				gen:    reqGen,
				source: &req.Header,
			},
			body:  req.Body,
			close: req.Close,
		}

		// Wrap response writer
		rwAdapter := &responseWriterAdapter{rw: rw}

		// Call user handler with generation-pooled request
		s.config.Handler.ServeHTTP(rwAdapter, greenTeaReq)

		// Check if connection should close
		if req.Close {
			return fmt.Errorf("connection close requested")
		}

		// Generation will be automatically returned to pool
		return nil
	})

	// Log error if not EOF (clean close)
	if err != nil {
		s.stats.RequestErrors.Add(1)
	}
}

// greenTeaRequest wraps a request with Green Tea GC generation
type greenTeaRequest struct {
	gen    *shockwave.RequestGeneration
	method string
	path   string
	proto  string
	header Header
	body   io.Reader
	close  bool
}

func (r *greenTeaRequest) Method() string {
	return r.method
}

func (r *greenTeaRequest) Path() string {
	return r.path
}

func (r *greenTeaRequest) Proto() string {
	return r.proto
}

func (r *greenTeaRequest) Header() Header {
	return r.header
}

func (r *greenTeaRequest) Body() io.Reader {
	return r.body
}

func (r *greenTeaRequest) Close() bool {
	return r.close
}

// greenTeaHeader wraps headers with generation pooling
type greenTeaHeader struct {
	gen    *shockwave.RequestGeneration
	source *http11.Header
}

func (h *greenTeaHeader) Get(key string) string {
	return h.source.GetString([]byte(key))
}

func (h *greenTeaHeader) Set(key, value string) {
	h.source.Set([]byte(key), []byte(value))
}

func (h *greenTeaHeader) Add(key, value string) {
	h.source.Add([]byte(key), []byte(value))
}

func (h *greenTeaHeader) Del(key string) {
	h.source.Del([]byte(key))
}

func (h *greenTeaHeader) Clone() Header {
	cloned := &http11.Header{}
	h.source.VisitAll(func(name, value []byte) bool {
		cloned.Set(name, value)
		return true
	})
	return &headerAdapter{h: cloned}
}
