//go:build goexperiment.arenas && !greenteagc

package server

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave/http11"
	"github.com/yourusername/shockwave/pkg/shockwave/memory"
)

// ArenaServer is an HTTP/1.1 server implementation using arena allocation
// This provides near-zero GC pressure by allocating all request data in arenas
type ArenaServer struct {
	*BaseServer
	arenaPool *memory.ArenaPool
}

// NewServer creates a new Shockwave HTTP server with arena allocation
func NewServer(config Config) Server {
	base := NewBaseServer(config)
	return &ArenaServer{
		BaseServer: base,
		arenaPool:  memory.NewArenaPool(),
	}
}

// ListenAndServe listens on the configured address and serves requests
func (s *ArenaServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.Serve(ln)
}

// ListenAndServeTLS listens on the configured address with TLS
func (s *ArenaServer) ListenAndServeTLS(certFile, keyFile string) error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.ServeTLS(ln, certFile, keyFile)
}

// Serve accepts incoming connections on the Listener
func (s *ArenaServer) Serve(l net.Listener) error {
	s.listener = l
	defer l.Close()

	for {
		if s.shutdown.Load() {
			return nil
		}

		if s.connSem != nil {
			select {
			case s.connSem <- struct{}{}:
			case <-s.done:
				return nil
			}
		}

		conn, err := l.Accept()
		if err != nil {
			if s.shutdown.Load() {
				return nil
			}
			s.stats.ConnectionErrors.Add(1)

			if s.connSem != nil {
				<-s.connSem
			}
			continue
		}

		s.stats.TotalConnections.Add(1)

		s.wg.Add(1)
		go s.handleConnection(conn)
	}
}

// ServeTLS accepts incoming connections on the Listener with TLS
func (s *ArenaServer) ServeTLS(l net.Listener, certFile, keyFile string) error {
	return fmt.Errorf("TLS not yet implemented")
}

// handleConnection handles a single connection with arena allocation
func (s *ArenaServer) handleConnection(netConn net.Conn) {
	defer s.wg.Done()
	defer netConn.Close()

	if s.connSem != nil {
		defer func() { <-s.connSem }()
	}

	s.trackConnection(netConn)
	defer s.untrackConnection(netConn)

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

	// Serve with arena allocation
	err := conn.Serve(func(req *http11.Request, rw *http11.ResponseWriter) error {
		// Get arena from pool for this request
		arena := s.arenaPool.Get()
		defer s.arenaPool.Put(arena)

		s.stats.TotalRequests.Add(1)
		s.stats.LastRequestTime.Store(time.Now())

		if s.config.ReadTimeout > 0 {
			netConn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
		}
		if s.config.WriteTimeout > 0 {
			netConn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
		}

		// Allocate request strings in arena
		arenaReq := &arenaRequest{
			arena:  arena,
			method: arena.MakeString(req.Method()),
			path:   arena.MakeString(req.Path()),
			proto:  req.Proto,
			header: &arenaHeader{
				arena:  arena,
				source: &req.Header,
			},
			body:  req.Body,
			close: req.Close,
		}

		rwAdapter := &responseWriterAdapter{rw: rw}

		// Call user handler with arena-allocated request
		s.config.Handler.ServeHTTP(rwAdapter, arenaReq)

		if req.Close {
			return fmt.Errorf("connection close requested")
		}

		// Arena will be automatically freed when deferred Put() is called
		return nil
	})

	if err != nil {
		s.stats.RequestErrors.Add(1)
	}
}

// arenaRequest wraps a request with arena-allocated data
type arenaRequest struct {
	arena  *memory.Arena
	method string
	path   string
	proto  string
	header Header
	body   io.Reader
	close  bool
}

func (r *arenaRequest) Method() string {
	return r.method
}

func (r *arenaRequest) Path() string {
	return r.path
}

func (r *arenaRequest) Proto() string {
	return r.proto
}

func (r *arenaRequest) Header() Header {
	return r.header
}

func (r *arenaRequest) Body() io.Reader {
	return r.body
}

func (r *arenaRequest) Close() bool {
	return r.close
}

// arenaHeader wraps headers with arena allocation
type arenaHeader struct {
	arena  *memory.Arena
	source *http11.Header
	cache  map[string]string // Cache for arena-allocated strings
}

func (h *arenaHeader) Get(key string) string {
	if h.cache == nil {
		h.cache = make(map[string]string)
	}

	// Check cache first
	if val, ok := h.cache[key]; ok {
		return val
	}

	// Get from source and cache in arena
	valBytes := h.source.Get([]byte(key))
	if len(valBytes) > 0 {
		val := h.arena.MakeString(string(valBytes))
		h.cache[key] = val
		return val
	}

	return ""
}

func (h *arenaHeader) Set(key, value string) {
	h.source.Set([]byte(key), []byte(value))
	if h.cache != nil {
		delete(h.cache, key)
	}
}

func (h *arenaHeader) Add(key, value string) {
	h.source.Add([]byte(key), []byte(value))
	if h.cache != nil {
		delete(h.cache, key)
	}
}

func (h *arenaHeader) Del(key string) {
	h.source.Del([]byte(key))
	if h.cache != nil {
		delete(h.cache, key)
	}
}

func (h *arenaHeader) Clone() Header {
	cloned := &http11.Header{}
	h.source.VisitAll(func(name, value []byte) bool {
		cloned.Set(name, value)
		return true
	})
	return &headerAdapter{h: cloned}
}
