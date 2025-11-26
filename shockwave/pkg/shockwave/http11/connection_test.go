package http11

import (
	"fmt"
	"io"
	"strings"
	"sync"
	"testing"
	"time"
)

// Tests

func TestConnectionStateString(t *testing.T) {
	tests := []struct {
		state    ConnectionState
		expected string
	}{
		{StateNew, "new"},
		{StateActive, "active"},
		{StateIdle, "idle"},
		{StateClosed, "closed"},
		{ConnectionState(999), "unknown"},
	}

	for _, tt := range tests {
		t.Run(tt.expected, func(t *testing.T) {
			result := tt.state.String()
			if result != tt.expected {
				t.Errorf("State.String() = %s, want %s", result, tt.expected)
			}
		})
	}
}

func TestDefaultConnectionConfig(t *testing.T) {
	config := DefaultConnectionConfig()

	if config.KeepAliveTimeout != 60*time.Second {
		t.Errorf("KeepAliveTimeout = %v, want 60s", config.KeepAliveTimeout)
	}

	if config.MaxRequests != 0 {
		t.Errorf("MaxRequests = %d, want 0", config.MaxRequests)
	}

	if config.ReadBufferSize != 4096 {
		t.Errorf("ReadBufferSize = %d, want 4096", config.ReadBufferSize)
	}

	if config.WriteBufferSize != 4096 {
		t.Errorf("WriteBufferSize = %d, want 4096", config.WriteBufferSize)
	}
}

func TestNewConnection(t *testing.T) {
	mockConn := newMockConn("")
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	if conn == nil {
		t.Fatal("NewConnection returned nil")
	}

	if conn.State() != StateNew {
		t.Errorf("Initial state = %v, want StateNew", conn.State())
	}

	if conn.RequestCount() != 0 {
		t.Errorf("Initial RequestCount = %d, want 0", conn.RequestCount())
	}

	if conn.RemoteAddr() == nil {
		t.Error("RemoteAddr is nil")
	}

	if conn.LocalAddr() == nil {
		t.Error("LocalAddr is nil")
	}
}

func TestConnectionSingleRequest(t *testing.T) {
	requestData := "GET /test HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()
	config.KeepAliveTimeout = 1 * time.Second

	handlerCalled := false
	handler := func(req *Request, rw *ResponseWriter) error {
		handlerCalled = true

		// Verify request
		if req.MethodID != MethodGET {
			t.Errorf("Method = %d, want GET", req.MethodID)
		}

		if req.Path() != "/test" {
			t.Errorf("Path = %s, want /test", req.Path())
		}

		// Write response
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))

		// Close connection after this request
		return io.EOF
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != io.EOF {
		t.Errorf("Serve error = %v, want EOF", err)
	}

	if !handlerCalled {
		t.Error("Handler was not called")
	}

	if conn.RequestCount() != 1 {
		t.Errorf("RequestCount = %d, want 1", conn.RequestCount())
	}

	// Check response was written
	written := mockConn.GetWritten()
	if !strings.Contains(written, "HTTP/1.1 200 OK") {
		t.Error("Response missing status line")
	}

	if !strings.Contains(written, "OK") {
		t.Error("Response missing body")
	}
}

func TestConnectionKeepAlive(t *testing.T) {
	// HTTP pipelining now supported - parser properly tracks buffer boundaries
	// Two complete requests on same connection
	requestData := "GET /first HTTP/1.1\r\nHost: example.com\r\n\r\n" +
		"GET /second HTTP/1.1\r\nHost: example.com\r\n\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	requestCount := 0
	handler := func(req *Request, rw *ResponseWriter) error {
		requestCount++

		if requestCount == 1 && req.Path() != "/first" {
			t.Errorf("First request path = %s, want /first", req.Path())
		}

		if requestCount == 2 && req.Path() != "/second" {
			t.Errorf("Second request path = %s, want /second", req.Path())
		}

		// Write response
		rw.WriteHeader(200)
		rw.Write([]byte(fmt.Sprintf("Response %d", requestCount)))

		// Close after second request
		if requestCount >= 2 {
			return io.EOF
		}

		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != io.EOF {
		t.Errorf("Serve error = %v, want EOF", err)
	}

	if requestCount != 2 {
		t.Errorf("Handler called %d times, want 2", requestCount)
	}

	if conn.RequestCount() != 2 {
		t.Errorf("RequestCount = %d, want 2", conn.RequestCount())
	}
}

func TestConnectionCloseHeader(t *testing.T) {
	requestData := "GET /test HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"Connection: close\r\n" +
		"\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		// Verify request has Close set
		if !req.Close {
			t.Error("Request.Close should be true")
		}

		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != nil && err != io.EOF {
		t.Errorf("Serve error = %v", err)
	}

	// Should handle only 1 request due to Connection: close
	if conn.RequestCount() != 1 {
		t.Errorf("RequestCount = %d, want 1", conn.RequestCount())
	}
}

func TestConnectionMaxRequests(t *testing.T) {
	// HTTP pipelining now supported - parser properly handles multiple requests
	// Three requests, but max is 2
	requestData := "GET /first HTTP/1.1\r\nHost: example.com\r\n\r\n" +
		"GET /second HTTP/1.1\r\nHost: example.com\r\n\r\n" +
		"GET /third HTTP/1.1\r\nHost: example.com\r\n\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()
	config.MaxRequests = 2 // Limit to 2 requests

	requestCount := 0
	handler := func(req *Request, rw *ResponseWriter) error {
		requestCount++
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	conn.Serve()

	// Should handle only 2 requests due to MaxRequests
	if requestCount != 2 {
		t.Errorf("Handler called %d times, want 2", requestCount)
	}
}

func TestConnectionClose(t *testing.T) {
	mockConn := newMockConn("")
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		return nil
	}

	conn := NewConnection(mockConn, config, handler)

	// Close connection
	err := conn.Close()
	if err != nil {
		t.Errorf("Close() error = %v", err)
	}

	// State should be closed
	if conn.State() != StateClosed {
		t.Errorf("State after Close = %v, want StateClosed", conn.State())
	}

	// Underlying conn should be closed
	if !mockConn.IsClosed() {
		t.Error("Underlying connection not closed")
	}

	// Double close should not error
	err = conn.Close()
	if err != nil {
		t.Errorf("Double Close() error = %v", err)
	}
}

func TestConnectionIdleTime(t *testing.T) {
	mockConn := newMockConn("")
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	// Initially should be recent (small idle time)
	idleTime := conn.IdleTime()
	if idleTime < 0 || idleTime > 100*time.Millisecond {
		t.Errorf("Initial IdleTime = %v, want ~0", idleTime)
	}

	// Set to active state
	conn.setState(StateActive)

	// Active connections should have 0 idle time
	idleTime = conn.IdleTime()
	if idleTime != 0 {
		t.Errorf("IdleTime during active = %v, want 0", idleTime)
	}
}

func TestConnectionHandlerError(t *testing.T) {
	requestData := "GET /test HTTP/1.1\r\n\r\n"

	mockConn := newMockConn(requestData)
	config := DefaultConnectionConfig()

	handlerError := fmt.Errorf("handler error")
	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(500)
		return handlerError
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != handlerError {
		t.Errorf("Serve() error = %v, want %v", err, handlerError)
	}
}

func TestConnectionEOF(t *testing.T) {
	// Empty connection (immediate EOF)
	mockConn := newMockConn("")
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		t.Error("Handler should not be called for EOF")
		return nil
	}

	conn := NewConnection(mockConn, config, handler)
	defer conn.Close()

	err := conn.Serve()
	if err != nil {
		t.Errorf("Serve() with EOF should return nil, got %v", err)
	}
}

func TestConnectionConcurrentClose(t *testing.T) {
	// Test concurrent Close() calls
	mockConn := newMockConn("GET /test HTTP/1.1\r\n\r\n")
	config := DefaultConnectionConfig()

	handler := func(req *Request, rw *ResponseWriter) error {
		return nil
	}

	conn := NewConnection(mockConn, config, handler)

	var wg sync.WaitGroup
	for i := 0; i < 10; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			conn.Close()
		}()
	}

	wg.Wait()

	// Connection should be closed
	if conn.State() != StateClosed {
		t.Error("Connection should be closed")
	}
}

// Benchmarks

func BenchmarkConnectionServe(b *testing.B) {
	requestData := strings.Repeat("GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n", 1000)

	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		mockConn := newMockConn(requestData)
		config := DefaultConnectionConfig()
		config.MaxRequests = 10 // Limit requests

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}

func BenchmarkConnectionKeepAlive(b *testing.B) {
	handler := func(req *Request, rw *ResponseWriter) error {
		rw.WriteHeader(200)
		rw.Write([]byte("OK"))
		return nil
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		requestData := "GET /test HTTP/1.1\r\nHost: example.com\r\n\r\n"
		mockConn := newMockConn(strings.Repeat(requestData, 10))
		config := DefaultConnectionConfig()
		config.MaxRequests = 10

		conn := NewConnection(mockConn, config, handler)
		conn.Serve()
		conn.Close()
	}
}
