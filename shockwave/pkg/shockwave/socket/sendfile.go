//go:build !linux && !darwin
// +build !linux,!darwin

package socket

import (
	"io"
	"net"
	"os"
)

// SendFile falls back to io.Copy on platforms without sendfile support.
// This provides a consistent API across all platforms.
func SendFile(conn net.Conn, file *os.File, offset int64, count int64) (written int64, err error) {
	return io.Copy(conn, io.NewSectionReader(file, offset, count))
}

// SendFileAll sends an entire file.
func SendFileAll(conn net.Conn, file *os.File) (written int64, err error) {
	stat, err := file.Stat()
	if err != nil {
		return 0, err
	}

	return SendFile(conn, file, 0, stat.Size())
}

// SendFileRange sends a range of a file.
func SendFileRange(conn net.Conn, file *os.File, start, end int64) (written int64, err error) {
	if end < start {
		return 0, io.EOF
	}

	count := end - start + 1
	return SendFile(conn, file, start, count)
}

// CanUseSendFile returns false on this platform.
func CanUseSendFile(conn net.Conn) bool {
	return false
}
