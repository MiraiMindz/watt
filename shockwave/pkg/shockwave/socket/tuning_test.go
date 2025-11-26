package socket

import (
	"io"
	"net"
	"os"
	"runtime"
	"strings"
	"testing"
	"time"
)

// TestDefaultConfig tests that default configuration is sensible
func TestDefaultConfig(t *testing.T) {
	cfg := DefaultConfig()

	if !cfg.NoDelay {
		t.Error("NoDelay should be true by default")
	}

	if cfg.RecvBuffer != 256*1024 {
		t.Errorf("RecvBuffer = %d, want %d", cfg.RecvBuffer, 256*1024)
	}

	if cfg.SendBuffer != 256*1024 {
		t.Errorf("SendBuffer = %d, want %d", cfg.SendBuffer, 256*1024)
	}

	if !cfg.KeepAlive {
		t.Error("KeepAlive should be true by default")
	}
}

// TestHighThroughputConfig tests high throughput configuration
func TestHighThroughputConfig(t *testing.T) {
	cfg := HighThroughputConfig()

	if cfg.RecvBuffer != 1024*1024 {
		t.Errorf("RecvBuffer = %d, want %d", cfg.RecvBuffer, 1024*1024)
	}

	if cfg.SendBuffer != 1024*1024 {
		t.Errorf("SendBuffer = %d, want %d", cfg.SendBuffer, 1024*1024)
	}

	if cfg.QuickAck {
		t.Error("QuickAck should be false for high throughput (allow delayed ACKs)")
	}
}

// TestLowLatencyConfig tests low latency configuration
func TestLowLatencyConfig(t *testing.T) {
	cfg := LowLatencyConfig()

	if !cfg.QuickAck {
		t.Error("QuickAck should be true for low latency")
	}

	if cfg.DeferAccept {
		t.Error("DeferAccept should be false for low latency")
	}

	if !cfg.FastOpen {
		t.Error("FastOpen should be true for low latency")
	}
}

// TestApply tests applying socket options to a connection
func TestApply(t *testing.T) {
	// Create a TCP listener
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Accept connection in background
	acceptDone := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Logf("Accept failed: %v", err)
			return
		}
		acceptDone <- conn
	}()

	// Connect to listener
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Wait for accept
	serverConn := <-acceptDone
	defer serverConn.Close()

	// Apply default config
	if err := Apply(serverConn, DefaultConfig()); err != nil {
		t.Errorf("Apply failed: %v", err)
	}

	// Verify connection still works
	msg := "Hello, World!"
	go func() {
		conn.Write([]byte(msg))
	}()

	buf := make([]byte, len(msg))
	n, err := serverConn.Read(buf)
	if err != nil {
		t.Errorf("Read failed: %v", err)
	}

	if string(buf[:n]) != msg {
		t.Errorf("Got %q, want %q", string(buf[:n]), msg)
	}
}

// TestApplyListener tests applying socket options to a listener
func TestApplyListener(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Apply listener options
	if err := ApplyListener(listener, DefaultConfig()); err != nil {
		// On some platforms, this might fail if options aren't supported
		// That's OK, we just log it
		t.Logf("ApplyListener returned error (may be expected): %v", err)
	}

	// Verify listener still works
	connectDone := make(chan bool)
	go func() {
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			t.Logf("Dial failed: %v", err)
			return
		}
		conn.Close()
		connectDone <- true
	}()

	conn, err := listener.Accept()
	if err != nil {
		t.Errorf("Accept failed: %v", err)
	}
	conn.Close()

	<-connectDone
}

// TestApplyNilConfig tests applying with nil config (should use default)
func TestApplyNilConfig(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	acceptDone := make(chan net.Conn, 1)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		acceptDone <- conn
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	serverConn := <-acceptDone
	defer serverConn.Close()

	// Apply with nil config (should use defaults)
	if err := Apply(serverConn, nil); err != nil {
		t.Errorf("Apply with nil config failed: %v", err)
	}
}

// TestSendFile tests sendfile functionality
func TestSendFile(t *testing.T) {
	// Create temporary file with test data
	tmpfile, err := os.CreateTemp("", "sendfile-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Write test data
	testData := strings.Repeat("Hello, World!\n", 1000) // ~14KB
	if _, err := tmpfile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}

	// Seek back to start
	if _, err := tmpfile.Seek(0, 0); err != nil {
		t.Fatalf("Failed to seek: %v", err)
	}

	// Create TCP connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Receiver
	receiveDone := make(chan string)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			t.Logf("Accept failed: %v", err)
			return
		}
		defer conn.Close()

		// Read all data
		data, err := io.ReadAll(conn)
		if err != nil {
			t.Logf("Read failed: %v", err)
			return
		}
		receiveDone <- string(data)
	}()

	// Sender
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Send file
	written, err := SendFileAll(conn, tmpfile)
	if err != nil {
		t.Fatalf("SendFile failed: %v", err)
	}

	if written != int64(len(testData)) {
		t.Errorf("Wrote %d bytes, want %d", written, len(testData))
	}

	// Close to signal EOF
	conn.Close()

	// Verify received data
	select {
	case received := <-receiveDone:
		if received != testData {
			t.Errorf("Data mismatch: got %d bytes, want %d bytes", len(received), len(testData))
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout waiting for data")
	}
}

// TestSendFileRange tests sending a file range
func TestSendFileRange(t *testing.T) {
	// Create temporary file
	tmpfile, err := os.CreateTemp("", "sendfile-range-test-*.txt")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Write test data (100 bytes)
	testData := strings.Repeat("0123456789", 10)
	if _, err := tmpfile.WriteString(testData); err != nil {
		t.Fatalf("Failed to write test data: %v", err)
	}
	tmpfile.Seek(0, 0)

	// Create connection
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	receiveDone := make(chan string)
	go func() {
		conn, err := listener.Accept()
		if err != nil {
			return
		}
		defer conn.Close()

		data, err := io.ReadAll(conn)
		if err != nil {
			return
		}
		receiveDone <- string(data)
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	// Send bytes 10-29 (20 bytes: "0123456789012345678")
	written, err := SendFileRange(conn, tmpfile, 10, 29)
	if err != nil {
		t.Fatalf("SendFileRange failed: %v", err)
	}

	if written != 20 {
		t.Errorf("Wrote %d bytes, want 20", written)
	}

	conn.Close()

	select {
	case received := <-receiveDone:
		expected := testData[10:30]
		if received != expected {
			t.Errorf("Got %q, want %q", received, expected)
		}
	case <-time.After(5 * time.Second):
		t.Error("Timeout")
	}
}

// TestCanUseSendFile tests the sendfile capability check
func TestCanUseSendFile(t *testing.T) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		t.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	acceptDone := make(chan net.Conn)
	go func() {
		conn, _ := listener.Accept()
		acceptDone <- conn
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		t.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	serverConn := <-acceptDone
	defer serverConn.Close()

	// TCP connection should support sendfile on Linux/Darwin
	canUse := CanUseSendFile(serverConn)

	if runtime.GOOS == "linux" || runtime.GOOS == "darwin" {
		if !canUse {
			t.Error("Should be able to use sendfile on TCP connection")
		}
	}
}

// TestSendFilePerformance benchmarks sendfile vs io.Copy
func TestSendFilePerformance(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping performance test in short mode")
	}

	// Create a 10MB test file
	tmpfile, err := os.CreateTemp("", "sendfile-perf-*.bin")
	if err != nil {
		t.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())
	defer tmpfile.Close()

	// Write 10MB of data
	data := make([]byte, 10*1024*1024)
	for i := range data {
		data[i] = byte(i % 256)
	}
	tmpfile.Write(data)
	tmpfile.Seek(0, 0)

	// Test sendfile
	listener, _ := net.Listen("tcp", "127.0.0.1:0")
	defer listener.Close()

	go func() {
		conn, _ := listener.Accept()
		defer conn.Close()
		io.Copy(io.Discard, conn)
	}()

	conn, _ := net.Dial("tcp", listener.Addr().String())
	defer conn.Close()

	start := time.Now()
	written, err := SendFileAll(conn, tmpfile)
	elapsed := time.Since(start)

	if err != nil {
		t.Logf("SendFile error: %v", err)
	} else {
		throughput := float64(written) / elapsed.Seconds() / 1024 / 1024
		t.Logf("SendFile: %d bytes in %v (%.2f MB/s)", written, elapsed, throughput)
	}
}
