package http11

import (
	"net"
	"strings"
	"sync"
	"time"
)

// mockConn implements net.Conn for testing
type mockConn struct {
	readData  *strings.Reader
	writeData *strings.Builder
	closed    bool
	deadline  time.Time
	mu        sync.Mutex
}

func newMockConn(data string) *mockConn {
	return &mockConn{
		readData:  strings.NewReader(data),
		writeData: &strings.Builder{},
	}
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.readData.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writeData.Write(b)
}

func (m *mockConn) Close() error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.closed = true
	return nil
}

func (m *mockConn) LocalAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 8080}
}

func (m *mockConn) RemoteAddr() net.Addr {
	return &net.TCPAddr{IP: net.ParseIP("127.0.0.1"), Port: 12345}
}

func (m *mockConn) SetDeadline(t time.Time) error {
	m.mu.Lock()
	defer m.mu.Unlock()
	m.deadline = t
	return nil
}

func (m *mockConn) SetReadDeadline(t time.Time) error {
	return m.SetDeadline(t)
}

func (m *mockConn) SetWriteDeadline(t time.Time) error {
	return m.SetDeadline(t)
}

func (m *mockConn) IsClosed() bool {
	m.mu.Lock()
	defer m.mu.Unlock()
	return m.closed
}

func (m *mockConn) GetWritten() string {
	return m.writeData.String()
}
