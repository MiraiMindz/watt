// Package shockwave provides integration between Bolt and Shockwave HTTP server.
//
// Shockwave is a high-performance HTTP implementation achieving 40-60% better
// performance than net/http through:
//   - Zero-copy request parsing
//   - Built-in connection pooling
//   - Optimized response writing
//
// This package adapts Shockwave types to Bolt's API.
package shockwave

import (
	"context"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave/http11"
	"github.com/yourusername/shockwave/pkg/shockwave/server"
)

// Config holds Shockwave server configuration for Bolt.
type Config struct {
	// Server address (e.g., ":8080")
	Addr string

	// Request handler (concrete types for zero allocations)
	// Uses server.Handler type directly for compatibility
	Handler server.Handler

	// Timeouts
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
	IdleTimeout  time.Duration

	// Limits
	MaxHeaderBytes     int
	MaxRequestBodySize int

	// Performance
	DisableStats bool // Set to true for zero-allocation mode
}

// DefaultConfig returns default Shockwave configuration for Bolt.
func DefaultConfig() *Config {
	return &Config{
		Addr:               ":8080",
		ReadTimeout:        5 * time.Second,
		WriteTimeout:       10 * time.Second,
		IdleTimeout:        120 * time.Second,
		MaxHeaderBytes:     1 << 20,  // 1 MB
		MaxRequestBodySize: 10 << 20, // 10 MB
		DisableStats:       true,     // Zero-allocation mode by default
	}
}

// Server wraps the Shockwave HTTP server for Bolt.
type Server struct {
	config *Config
	srv    server.Server
}

// NewServer creates a new Shockwave-backed HTTP server for Bolt.
func NewServer(config *Config) *Server {
	if config == nil {
		config = DefaultConfig()
	}

	s := &Server{
		config: config,
	}

	// Create Shockwave config
	shockwaveConfig := server.DefaultConfig()
	shockwaveConfig.Addr = config.Addr
	shockwaveConfig.EnableStats = !config.DisableStats
	shockwaveConfig.ReadTimeout = config.ReadTimeout
	shockwaveConfig.WriteTimeout = config.WriteTimeout
	shockwaveConfig.IdleTimeout = config.IdleTimeout
	shockwaveConfig.MaxHeaderBytes = config.MaxHeaderBytes
	shockwaveConfig.MaxRequestBodySize = config.MaxRequestBodySize

	// Set handler (direct pass-through, no wrapping needed)
	shockwaveConfig.Handler = config.Handler

	// Create server
	s.srv = server.NewServer(shockwaveConfig)
	return s
}

// ListenAndServe starts the HTTP server.
func (s *Server) ListenAndServe() error {
	return s.srv.ListenAndServe()
}

// ListenAndServeTLS starts the HTTPS server.
func (s *Server) ListenAndServeTLS(certFile, keyFile string) error {
	return s.srv.ListenAndServeTLS(certFile, keyFile)
}

// Shutdown gracefully shuts down the server.
func (s *Server) Shutdown(ctx context.Context) error {
	return s.srv.Shutdown(ctx)
}

// Request is just an alias to http11.Request for convenience.
// No wrapping needed - Bolt works directly with Shockwave types.
type Request = http11.Request

// ResponseWriter is just an alias to http11.ResponseWriter for convenience.
// No wrapping needed - Bolt works directly with Shockwave types.
type ResponseWriter = http11.ResponseWriter
