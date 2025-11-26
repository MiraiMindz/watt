// Package socket provides cross-platform socket tuning and optimizations.
//
// Performance-critical socket options are applied to minimize latency and
// maximize throughput for HTTP workloads. Platform-specific optimizations
// are in tuning_linux.go and tuning_darwin.go.
package socket

import (
	"net"
	"syscall"
)

// Config represents socket tuning configuration.
// Zero values mean "use system defaults".
type Config struct {
	// TCP_NODELAY - Disable Nagle's algorithm for low latency
	// Default: true (recommended for HTTP/1.1 and HTTP/2)
	NoDelay bool

	// SO_RCVBUF - Receive buffer size in bytes
	// Default: 0 (use system default, typically 128KB-256KB)
	// Recommended: 256KB-1MB for high-throughput workloads
	RecvBuffer int

	// SO_SNDBUF - Send buffer size in bytes
	// Default: 0 (use system default, typically 128KB-256KB)
	// Recommended: 256KB-1MB for high-throughput workloads
	SendBuffer int

	// TCP_QUICKACK - Send immediate ACKs (Linux only)
	// Default: false
	// Reduces latency by 40ms (one delayed ACK timeout)
	QuickAck bool

	// TCP_DEFER_ACCEPT - Don't wake server until data arrives (Linux only)
	// Default: false
	// Reduces context switches and improves efficiency
	DeferAccept bool

	// TCP_FASTOPEN - Enable TCP Fast Open (Linux 3.7+, Darwin 10.11+)
	// Default: false
	// Reduces connection establishment latency by one RTT
	FastOpen bool

	// SO_KEEPALIVE - Enable TCP keepalive
	// Default: true (recommended for long-lived connections)
	KeepAlive bool
}

// DefaultConfig returns the recommended configuration for HTTP workloads.
// This provides optimal latency and throughput for typical web servers.
func DefaultConfig() *Config {
	return &Config{
		NoDelay:      true,  // Disable Nagle for low latency
		RecvBuffer:   256 * 1024, // 256KB receive buffer
		SendBuffer:   256 * 1024, // 256KB send buffer
		QuickAck:     true,  // Immediate ACKs (Linux only)
		DeferAccept:  true,  // Don't wake until data (Linux only)
		FastOpen:     true,  // Enable TFO (Linux/Darwin)
		KeepAlive:    true,  // Enable keepalive
	}
}

// HighThroughputConfig returns configuration optimized for maximum throughput.
// Use this for bulk data transfer or high-bandwidth workloads.
func HighThroughputConfig() *Config {
	return &Config{
		NoDelay:      true,  // Still disable Nagle
		RecvBuffer:   1024 * 1024, // 1MB receive buffer
		SendBuffer:   1024 * 1024, // 1MB send buffer
		QuickAck:     false, // Allow delayed ACKs for throughput
		DeferAccept:  true,
		FastOpen:     true,
		KeepAlive:    true,
	}
}

// LowLatencyConfig returns configuration optimized for minimum latency.
// Use this for real-time applications or API servers.
func LowLatencyConfig() *Config {
	return &Config{
		NoDelay:      true,
		RecvBuffer:   128 * 1024, // Smaller buffers for lower latency
		SendBuffer:   128 * 1024,
		QuickAck:     true,  // Immediate ACKs critical for latency
		DeferAccept:  false, // Don't delay connection acceptance
		FastOpen:     true,  // Reduce handshake latency
		KeepAlive:    true,
	}
}

// Apply applies socket tuning options to a connection.
// Returns error if any critical option fails (TCP_NODELAY).
// Non-critical options (platform-specific) log warnings but don't fail.
//
// This should be called immediately after accepting a connection.
func Apply(conn net.Conn, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	// Get raw socket file descriptor
	tcpConn, ok := conn.(*net.TCPConn)
	if !ok {
		// Not a TCP connection, can't tune
		return nil
	}

	rawConn, err := tcpConn.SyscallConn()
	if err != nil {
		return err
	}

	var lastErr error

	// Apply cross-platform options first
	err = rawConn.Control(func(fd uintptr) {
		// TCP_NODELAY - Critical for HTTP performance
		if cfg.NoDelay {
			if err := syscall.SetsockoptInt(int(fd), syscall.IPPROTO_TCP, syscall.TCP_NODELAY, 1); err != nil {
				lastErr = err
				return
			}
		}

		// SO_RCVBUF - Receive buffer size
		if cfg.RecvBuffer > 0 {
			if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_RCVBUF, cfg.RecvBuffer); err != nil {
				// Non-critical, continue
				_ = err
			}
		}

		// SO_SNDBUF - Send buffer size
		if cfg.SendBuffer > 0 {
			if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_SNDBUF, cfg.SendBuffer); err != nil {
				// Non-critical, continue
				_ = err
			}
		}

		// SO_KEEPALIVE
		if cfg.KeepAlive {
			if err := syscall.SetsockoptInt(int(fd), syscall.SOL_SOCKET, syscall.SO_KEEPALIVE, 1); err != nil {
				// Non-critical, continue
				_ = err
			}
		}

		// Apply platform-specific options
		applyPlatformOptions(int(fd), cfg)
	})

	if err != nil {
		return err
	}

	return lastErr
}

// ApplyListener applies socket tuning options to a listening socket.
// This sets options like TCP_DEFER_ACCEPT and TCP_FASTOPEN that must be
// set on the listener before accepting connections.
func ApplyListener(listener net.Listener, cfg *Config) error {
	if cfg == nil {
		cfg = DefaultConfig()
	}

	tcpListener, ok := listener.(*net.TCPListener)
	if !ok {
		return nil
	}

	// Get raw file descriptor
	file, err := tcpListener.File()
	if err != nil {
		return err
	}
	defer file.Close()

	fd := int(file.Fd())

	// Apply platform-specific listener options
	return applyListenerOptions(fd, cfg)
}
