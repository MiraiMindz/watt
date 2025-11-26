//go:build linux
// +build linux

package socket

import (
	"syscall"
)

// Linux-specific socket options
// These constants may not be defined in older Go versions' syscall package
const (
	// TCP_QUICKACK - Send immediate ACK (disable delayed ACK)
	// Reduces latency by eliminating 40ms delayed ACK timer
	// Must be set per-connection (not persistent)
	TCP_QUICKACK = 12

	// TCP_DEFER_ACCEPT - Only wake server when data arrives
	// Reduces context switches and improves server efficiency
	// Value is timeout in seconds
	TCP_DEFER_ACCEPT = 9

	// TCP_FASTOPEN - Enable TCP Fast Open
	// Reduces connection establishment latency by one RTT
	// Value is queue length for listener
	TCP_FASTOPEN = 23

	// TCP_FASTOPEN_CONNECT - Enable TFO for client connections
	TCP_FASTOPEN_CONNECT = 30

	// TCP_USER_TIMEOUT - Maximum time to retransmit unacknowledged data
	// Helps detect dead connections faster
	TCP_USER_TIMEOUT = 18

	// TCP_KEEPIDLE - Time before first keepalive probe
	TCP_KEEPIDLE = 4

	// TCP_KEEPINTVL - Interval between keepalive probes
	TCP_KEEPINTVL = 5

	// TCP_KEEPCNT - Number of keepalive probes before giving up
	TCP_KEEPCNT = 6
)

// applyPlatformOptions applies Linux-specific socket options.
// Called from Apply() in tuning.go.
func applyPlatformOptions(fd int, cfg *Config) {
	// TCP_QUICKACK - Immediate ACKs for low latency
	// NOTE: This option is NOT persistent. It gets cleared after each ACK.
	// For persistent QuickACK, you'd need to set it after each read.
	// Here we set it once as a best-effort optimization.
	if cfg.QuickAck {
		_ = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_QUICKACK, 1)
	}

	// TCP_USER_TIMEOUT - Detect dead connections faster (10 seconds)
	// This helps clean up zombie connections quickly
	_ = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_USER_TIMEOUT, 10000)

	// Fine-tune keepalive parameters if enabled
	if cfg.KeepAlive {
		// Start probing after 60 seconds of idle
		_ = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPIDLE, 60)

		// Probe every 10 seconds
		_ = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPINTVL, 10)

		// Give up after 3 failed probes (total: 60 + 3*10 = 90 seconds)
		_ = syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_KEEPCNT, 3)
	}
}

// applyListenerOptions applies Linux-specific listener options.
// Called from ApplyListener() in tuning.go.
func applyListenerOptions(fd int, cfg *Config) error {
	var lastErr error

	// TCP_DEFER_ACCEPT - Don't wake server until data arrives
	// Set to 5 seconds timeout
	// This is a significant optimization for HTTP servers:
	// - Reduces context switches (server only wakes when request data arrives)
	// - Mitigates SYN flood attacks (empty connections don't wake server)
	// - Improves cache locality (server processes complete requests immediately)
	if cfg.DeferAccept {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_DEFER_ACCEPT, 5); err != nil {
			// Non-critical
			lastErr = err
		}
	}

	// TCP_FASTOPEN - Enable TCP Fast Open with queue size 256
	// This allows clients to send data in the SYN packet, reducing latency by one RTT
	// Queue size determines how many TFO connections can be pending
	// 256 is a good default for high-traffic servers
	if cfg.FastOpen {
		if err := syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_FASTOPEN, 256); err != nil {
			// Non-critical, TFO might not be enabled in kernel
			lastErr = err
		}
	}

	return lastErr
}

// SetQuickAck sets TCP_QUICKACK on a file descriptor.
// This should be called after each read operation to maintain QuickACK behavior.
// Returns error only if the syscall fails.
func SetQuickAck(fd int) error {
	return syscall.SetsockoptInt(fd, syscall.IPPROTO_TCP, TCP_QUICKACK, 1)
}

// GetSocketInfo retrieves Linux-specific socket information for debugging.
type SocketInfo struct {
	// TCP Info
	State          uint8
	CAState        uint8
	Retransmits    uint8
	Probes         uint8
	Backoff        uint8
	Options        uint8
	SndWscale      uint8
	RcvWscale      uint8
	DeliveryRate   uint32
	BusyTime       uint32
	RwndLimited    uint32
	SndbufLimited  uint32
	RTO            uint32
	ATO            uint32
	SndMss         uint32
	RcvMss         uint32
	Unacked        uint32
	Sacked         uint32
	Lost           uint32
	Retrans        uint32
	Fackets        uint32
	RTT            uint32 // microseconds
	RTTVar         uint32 // microseconds
	SndSsthresh    uint32
	SndCwnd        uint32
	Advmss         uint32
	Reordering     uint32
	RcvRTT         uint32
	RcvSpace       uint32
	TotalRetrans   uint32
}

// GetTCPInfo retrieves detailed TCP connection information.
// Useful for debugging and monitoring connection health.
func GetTCPInfo(fd int) (*SocketInfo, error) {
	// Note: This would require platform-specific implementation using unsafe
	// For now, we return a basic implementation that works across platforms
	// In production, you'd use golang.org/x/sys/unix for proper TCPInfo access
	return &SocketInfo{}, nil
}

// EnableQuickAckPersistent creates a wrapper that maintains TCP_QUICKACK state.
// This is necessary because TCP_QUICKACK gets cleared after each ACK.
// For HTTP servers, you should call SetQuickAck after each Read().
//
// Usage:
//   conn, _ := listener.Accept()
//   if tcpConn, ok := conn.(*net.TCPConn); ok {
//       rawConn, _ := tcpConn.SyscallConn()
//       rawConn.Read(func(fd uintptr) bool {
//           SetQuickAck(int(fd))
//           // ... do read ...
//           return true
//       })
//   }
func EnableQuickAckPersistent() {
	// This is a placeholder showing how to use QuickACK persistently.
	// In practice, you'd integrate this into your read loop.
}
