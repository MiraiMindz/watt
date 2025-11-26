//go:build !linux && !darwin
// +build !linux,!darwin

package socket

// applyPlatformOptions is a no-op on platforms without specific optimizations.
func applyPlatformOptions(fd int, cfg *Config) {
	// No platform-specific options available
}

// applyListenerOptions is a no-op on platforms without specific optimizations.
func applyListenerOptions(fd int, cfg *Config) error {
	// No platform-specific options available
	return nil
}

// SetQuickAck is a no-op on platforms without TCP_QUICKACK.
func SetQuickAck(fd int) error {
	return nil
}

// SocketInfo is empty on platforms without detailed TCP info.
type SocketInfo struct{}

// GetTCPInfo returns empty info on platforms without TCP_INFO.
func GetTCPInfo(fd int) (*SocketInfo, error) {
	return &SocketInfo{}, nil
}
