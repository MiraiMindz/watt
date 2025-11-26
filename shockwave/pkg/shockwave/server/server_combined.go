//go:build goexperiment.arenas && greenteagc

package server

import (
	"fmt"
	"io"
	"net"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave"
	"github.com/yourusername/shockwave/pkg/shockwave/http11"
	"github.com/yourusername/shockwave/pkg/shockwave/memory"
)

// CombinedServer uses both arena allocation AND Green Tea GC generation pooling
// This combines the benefits of near-zero GC pressure with improved cache locality
type CombinedServer struct {
	*BaseServer
	arenaPool *memory.ArenaPool
	greenPool *shockwave.GreenTeaPool
}

// NewServer creates a new Shockwave HTTP server with combined arena + Green Tea GC
func NewServer(config Config) Server {
	base := NewBaseServer(config)
	return &CombinedServer{
		BaseServer: base,
		arenaPool:  memory.NewArenaPool(),
		greenPool:  shockwave.GlobalGreenTeaPool,
	}
}

// ListenAndServe listens on the configured address and serves requests
func (s *CombinedServer) ListenAndServe() error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.Serve(ln)
}

// ListenAndServeTLS listens on the configured address with TLS
func (s *CombinedServer) ListenAndServeTLS(certFile, keyFile string) error {
	ln, err := net.Listen("tcp", s.config.Addr)
	if err != nil {
		return fmt.Errorf("failed to listen on %s: %w", s.config.Addr, err)
	}
	return s.ServeTLS(ln, certFile, keyFile)
}

// Serve accepts incoming connections on the Listener
func (s *CombinedServer) Serve(l net.Listener) error {
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
func (s *CombinedServer) ServeTLS(l net.Listener, certFile, keyFile string) error {
	return fmt.Errorf("TLS not yet implemented")
}

// handleConnection handles a single connection with combined arena + Green Tea GC
func (s *CombinedServer) handleConnection(netConn net.Conn) {
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

	// Serve with combined allocation strategy
	err := conn.Serve(func(req *http11.Request, rw *http11.ResponseWriter) error {
		// Get both arena and generation
		arena := s.arenaPool.Get()
		defer s.arenaPool.Put(arena)

		reqGen := s.greenPool.GetRequestGeneration()
		defer s.greenPool.PutRequestGeneration(reqGen)

		s.stats.TotalRequests.Add(1)
		s.stats.LastRequestTime.Store(time.Now())

		if s.config.ReadTimeout > 0 {
			netConn.SetReadDeadline(time.Now().Add(s.config.ReadTimeout))
		}
		if s.config.WriteTimeout > 0 {
			netConn.SetWriteDeadline(time.Now().Add(s.config.WriteTimeout))
		}

		// Allocate request data in arena with generation tracking
		combinedReq := &combinedRequest{
			arena:  arena,
			gen:    reqGen,
			method: arena.MakeString(req.Method()),
			path:   arena.MakeString(req.Path()),
			proto:  req.Proto,
			header: &combinedHeader{
				arena:  arena,
				gen:    reqGen,
				source: &req.Header,
			},
			body:  req.Body,
			close: req.Close,
		}

		rwAdapter := &responseWriterAdapter{rw: rw}

		s.config.Handler.ServeHTTP(rwAdapter, combinedReq)

		if req.Close {
			return fmt.Errorf("connection close requested")
		}

		return nil
	})

	if err != nil {
		s.stats.RequestErrors.Add(1)
	}
}

// combinedRequest uses both arena allocation and generation tracking
type combinedRequest struct {
	arena  *memory.Arena
	gen    *shockwave.RequestGeneration
	method string
	path   string
	proto  string
	header Header
	body   io.Reader
	close  bool
}

func (r *combinedRequest) Method() string {
	return r.method
}

func (r *combinedRequest) Path() string {
	return r.path
}

func (r *combinedRequest) Proto() string {
	return r.proto
}

func (r *combinedRequest) Header() Header {
	return r.header
}

func (r *combinedRequest) Body() io.Reader {
	return r.body
}

func (r *combinedRequest) Close() bool {
	return r.close
}

// combinedHeader uses both arena and generation
type combinedHeader struct {
	arena  *memory.Arena
	gen    *shockwave.RequestGeneration
	source *http11.Header
}

func (h *combinedHeader) Get(key string) string {
	return h.source.GetString([]byte(key))
}

func (h *combinedHeader) Set(key, value string) {
	h.source.Set([]byte(key), []byte(value))
}

func (h *combinedHeader) Add(key, value string) {
	h.source.Add([]byte(key), []byte(value))
}

func (h *combinedHeader) Del(key string) {
	h.source.Del([]byte(key))
}

func (h *combinedHeader) Clone() Header {
	cloned := &http11.Header{}
	h.source.VisitAll(func(name, value []byte) bool {
		cloned.Set(name, value)
		return true
	})
	return &headerAdapter{h: cloned}
}
