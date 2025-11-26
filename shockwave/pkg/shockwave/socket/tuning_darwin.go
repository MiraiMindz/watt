//go:build darwin
// +build darwin

package socket

import (
	"syscall"
)

// Darwin (macOS) specific socket options
const (
	// TCP_FASTOPEN - Enable TCP Fast Open on macOS (10.11+)
	// Value is 1 for client, other values for server
	TCP_FASTOPEN = 0x105

	// TCP_KEEPALIVE - macOS equivalent of Linux TCP_KEEPIDLE
	TCP_KEEPALIVE = 0x10

	// SO_NOSIGPIPE - Don't send SIGPIPE on broken pipe (macOS specific)
	SO_NOSIGPIPE = 0x1022
)

// applyPlatformOptions applies Darwin-specific socket options.
// Called from Apply() in tuning.go.
func applyPlatformOptions(fd int, cfg *Config) {
	// SO_NOSIGPIPE - Prevent SIGPIPE on write to closed socket
	// This is macOS-specific; Linux uses MSG_NOSIGNAL on send()
	_ = syscall.SetsockoptInt(fd, syscall.SOL_SOCKET, SO_NOSIGPIPE, 1)

	// Fine-tune keepalive parameters if enabled
	if cfg.KeepAlive {
		// TCP_KEEPALIVE on macOS sets the idle time in seconds
		// Start probing after 60 seconds of idle
		_ = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPALIVE, 60)
	}
}

// applyListenerOptions applies Darwin-specific listener options.
// Called from ApplyListener() in tuning.go.
func applyListenerOptions(fd int, cfg *Config) error {
	var lastErr error

	// TCP_FASTOPEN on macOS
	// For server sockets, we need to enable it before listen()
	// The value represents the maximum number of pending TFO connections
	if cfg.FastOpen {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_FASTOPEN, 256); err != nil {
			// Non-critical, TFO might not be enabled in kernel
			lastErr = err
		}
	}

	return lastErr
}

// SetQuickAck is a no-op on Darwin (no TCP_QUICKACK equivalent).
// macOS doesn't have a direct equivalent to Linux's TCP_QUICKACK.
// This function exists for API compatibility.
func SetQuickAck(fd int) error {
	// No-op on Darwin
	return nil
}

// GetSocketInfo retrieves Darwin-specific socket information for debugging.
type SocketInfo struct {
	// Basic TCP info available on macOS
	State       uint8
	RTT         uint32 // microseconds
	RTTVar      uint32 // microseconds
	SndCwnd     uint32
	SndSsthresh uint32
	RcvSpace    uint32
}

// GetTCPInfo retrieves TCP connection information on Darwin.
// Note: macOS has limited TCP_INFO compared to Linux.
func GetTCPInfo(fd int) (*SocketInfo, error) {
	// macOS doesn't expose TCP_INFO in the same way as Linux
	// We could use getsockopt with TCP_CONNECTION_INFO on macOS 10.10+
	// For now, return limited info
	return &SocketInfo{}, nil
}

// Note: macOS doesn't have TCP_DEFER_ACCEPT equivalent
// The closest alternative is to set a very short accept timeout
// and only process connections that have data ready immediately.
// However, this requires application-level handling.
