package http2

import (
	"io"
	"testing"
	"time"
)

// Test stream creation
func TestNewStream(t *testing.T) {
	stream := NewStream(1, 65535)

	if stream.ID() != 1 {
		t.Errorf("stream ID = %d, want 1", stream.ID())
	}

	if stream.State() != StateIdle {
		t.Errorf("initial state = %s, want idle", stream.State())
	}

	if stream.SendWindow() != 65535 {
		t.Errorf("send window = %d, want 65535", stream.SendWindow())
	}

	if stream.RecvWindow() != 65535 {
		t.Errorf("recv window = %d, want 65535", stream.RecvWindow())
	}
}

// Test state transitions
func TestStreamStateTransitions(t *testing.T) {
	tests := []struct {
		name      string
		initial   StreamState
		next      StreamState
		shouldErr bool
	}{
		// Valid transitions
		{"idle to open", StateIdle, StateOpen, false},
		{"idle to half-closed(local)", StateIdle, StateHalfClosedLocal, false},
		{"idle to half-closed(remote)", StateIdle, StateHalfClosedRemote, false},
		{"open to half-closed(local)", StateOpen, StateHalfClosedLocal, false},
		{"open to half-closed(remote)", StateOpen, StateHalfClosedRemote, false},
		{"half-closed(local) to closed", StateHalfClosedLocal, StateClosed, false},
		{"half-closed(remote) to closed", StateHalfClosedRemote, StateClosed, false},

		// Invalid transitions
		{"closed to open", StateClosed, StateOpen, true},
		{"half-closed(local) to open", StateHalfClosedLocal, StateOpen, true},
		{"half-closed(remote) to open", StateHalfClosedRemote, StateOpen, true},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			stream := NewStream(1, 65535)
			stream.state = tt.initial

			err := stream.setState(tt.next)

			if tt.shouldErr && err == nil {
				t.Error("expected error, got nil")
			}

			if !tt.shouldErr && err != nil {
				t.Errorf("unexpected error: %v", err)
			}

			if !tt.shouldErr && stream.State() != tt.next {
				t.Errorf("state = %s, want %s", stream.State(), tt.next)
			}
		})
	}
}

// Test stream open
func TestStreamOpen(t *testing.T) {
	stream := NewStream(1, 65535)

	if err := stream.Open(); err != nil {
		t.Fatalf("Open() error: %v", err)
	}

	if stream.State() != StateOpen {
		t.Errorf("state = %s, want open", stream.State())
	}
}

// Test stream close local
func TestStreamCloseLocal(t *testing.T) {
	stream := NewStream(1, 65535)
	stream.Open()

	if err := stream.CloseLocal(); err != nil {
		t.Fatalf("CloseLocal() error: %v", err)
	}

	if stream.State() != StateHalfClosedLocal {
		t.Errorf("state = %s, want half-closed(local)", stream.State())
	}

	stream.sendBufMu.Lock()
	closed := stream.sendClosed
	stream.sendBufMu.Unlock()

	if !closed {
		t.Error("send side not marked as closed")
	}
}

// Test stream close remote
func TestStreamCloseRemote(t *testing.T) {
	stream := NewStream(1, 65535)
	stream.Open()

	if err := stream.CloseRemote(); err != nil {
		t.Fatalf("CloseRemote() error: %v", err)
	}

	if stream.State() != StateHalfClosedRemote {
		t.Errorf("state = %s, want half-closed(remote)", stream.State())
	}

	stream.recvBufMu.Lock()
	closed := stream.recvClosed
	stream.recvBufMu.Unlock()

	if !closed {
		t.Error("receive side not marked as closed")
	}
}

// Test stream full close
func TestStreamFullClose(t *testing.T) {
	stream := NewStream(1, 65535)
	stream.Open()

	stream.CloseLocal()
	stream.CloseRemote()

	if stream.State() != StateClosed {
		t.Errorf("state = %s, want closed", stream.State())
	}

	if !stream.IsClosed() {
		t.Error("IsClosed() = false, want true")
	}
}

// Test stream reset
func TestStreamReset(t *testing.T) {
	stream := NewStream(1, 65535)
	stream.Open()

	if err := stream.Reset(ErrCodeCancel); err != nil {
		t.Fatalf("Reset() error: %v", err)
	}

	if stream.State() != StateClosed {
		t.Errorf("state = %s, want closed", stream.State())
	}

	if stream.resetCode != ErrCodeCancel {
		t.Errorf("reset code = %v, want CANCEL", stream.resetCode)
	}
}

// Test window operations
func TestStreamWindowOperations(t *testing.T) {
	stream := NewStream(1, 65535)

	// Test increment
	if err := stream.IncrementSendWindow(1000); err != nil {
		t.Fatalf("IncrementSendWindow() error: %v", err)
	}

	if stream.SendWindow() != 66535 {
		t.Errorf("send window = %d, want 66535", stream.SendWindow())
	}

	// Test consume
	if err := stream.ConsumeSendWindow(1000); err != nil {
		t.Fatalf("ConsumeSendWindow() error: %v", err)
	}

	if stream.SendWindow() != 65535 {
		t.Errorf("send window = %d, want 65535", stream.SendWindow())
	}

	// Test consume more than available
	err := stream.ConsumeSendWindow(70000)
	if err == nil {
		t.Error("expected error consuming more than window, got nil")
	}

	// Test overflow protection
	stream.sendWindow = MaxWindowSize
	err = stream.IncrementSendWindow(1)
	if err == nil {
		t.Error("expected overflow error, got nil")
	}
}

// Test priority operations
func TestStreamPriority(t *testing.T) {
	stream := NewStream(1, 65535)

	err := stream.SetPriority(20, 3, true)
	if err != nil {
		t.Fatalf("SetPriority() error: %v", err)
	}

	weight, dep, exclusive := stream.Priority()

	if weight != 20 {
		t.Errorf("weight = %d, want 20", weight)
	}

	if dep != 3 {
		t.Errorf("dependency = %d, want 3", dep)
	}

	if !exclusive {
		t.Error("exclusive = false, want true")
	}

	// Test self-dependency rejection (RFC 7540 Section 5.3.1)
	err = stream.SetPriority(20, 1, false) // stream 1 depends on itself
	if err != ErrStreamSelfDependency {
		t.Errorf("expected ErrStreamSelfDependency, got: %v", err)
	}
}

// Test stream I/O operations
func TestStreamIO(t *testing.T) {
	stream := NewStream(1, 65535)

	// Write data
	data := []byte("hello world")
	n, err := stream.Write(data)
	if err != nil {
		t.Fatalf("Write() error: %v", err)
	}

	if n != len(data) {
		t.Errorf("wrote %d bytes, want %d", n, len(data))
	}

	// Receive data (simulate incoming data)
	incoming := []byte("test data")
	if err := stream.ReceiveData(incoming); err != nil {
		t.Fatalf("ReceiveData() error: %v", err)
	}

	// Read data (non-blocking read with timeout)
	done := make(chan bool)
	var readData []byte
	var readErr error

	go func() {
		buf := make([]byte, 100)
		n, err := stream.Read(buf)
		readData = buf[:n]
		readErr = err
		done <- true
	}()

	select {
	case <-done:
		if readErr != nil {
			t.Errorf("Read() error: %v", readErr)
		}

		if string(readData) != string(incoming) {
			t.Errorf("read data = %q, want %q", readData, incoming)
		}
	case <-time.After(100 * time.Millisecond):
		t.Fatal("Read() timeout")
	}
}

// Test stream read after close
func TestStreamReadAfterClose(t *testing.T) {
	stream := NewStream(1, 65535)

	// Add some data
	data := []byte("test")
	stream.ReceiveData(data)

	// Close receive side
	stream.CloseRemote()

	// Should still be able to read buffered data
	buf := make([]byte, 100)
	n, err := stream.Read(buf)
	if err != nil {
		t.Fatalf("Read() error: %v", err)
	}

	if string(buf[:n]) != string(data) {
		t.Errorf("read data = %q, want %q", buf[:n], data)
	}

	// Next read should return EOF
	n, err = stream.Read(buf)
	if err != io.EOF {
		t.Errorf("expected EOF, got: %v", err)
	}

	if n != 0 {
		t.Errorf("read %d bytes after EOF, want 0", n)
	}
}

// Test stream write after close
func TestStreamWriteAfterClose(t *testing.T) {
	stream := NewStream(1, 65535)

	stream.Open()
	stream.CloseLocal()

	// Writing after local close should fail
	_, err := stream.Write([]byte("test"))
	if err != io.EOF {
		t.Errorf("expected EOF, got: %v", err)
	}
}

// Test stream headers
func TestStreamHeaders(t *testing.T) {
	stream := NewStream(1, 65535)

	reqHeaders := []HeaderField{
		{":method", "GET"},
		{":path", "/"},
	}

	stream.SetRequestHeaders(reqHeaders)

	got := stream.RequestHeaders()
	if len(got) != len(reqHeaders) {
		t.Fatalf("got %d request headers, want %d", len(got), len(reqHeaders))
	}

	for i, h := range got {
		if h.Name != reqHeaders[i].Name || h.Value != reqHeaders[i].Value {
			t.Errorf("header %d = %+v, want %+v", i, h, reqHeaders[i])
		}
	}

	// Test response headers
	respHeaders := []HeaderField{
		{":status", "200"},
		{"content-type", "text/html"},
	}

	stream.SetResponseHeaders(respHeaders)

	got = stream.ResponseHeaders()
	if len(got) != len(respHeaders) {
		t.Fatalf("got %d response headers, want %d", len(got), len(respHeaders))
	}
}

// Test stream activity tracking
func TestStreamActivity(t *testing.T) {
	stream := NewStream(1, 65535)

	time1 := stream.LastActivity()
	time.Sleep(10 * time.Millisecond)

	err := stream.SetPriority(10, 0, false)
	if err != nil {
		t.Fatalf("SetPriority() error: %v", err)
	}

	time2 := stream.LastActivity()

	if !time2.After(time1) {
		t.Error("activity not updated after SetPriority")
	}

	if stream.Age() < 10*time.Millisecond {
		t.Error("stream age too small")
	}
}

// Test stream context
func TestStreamContext(t *testing.T) {
	stream := NewStream(1, 65535)

	ctx := stream.Context()
	if ctx == nil {
		t.Fatal("context is nil")
	}

	select {
	case <-ctx.Done():
		t.Error("context done prematurely")
	default:
	}

	// Cancel by resetting stream
	stream.Reset(ErrCodeCancel)

	select {
	case <-ctx.Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("context not cancelled after reset")
	}
}

// Test concurrent access to stream
func TestStreamConcurrentAccess(t *testing.T) {
	stream := NewStream(1, 65535)

	done := make(chan bool)

	// Goroutine 1: Write data
	go func() {
		for i := 0; i < 100; i++ {
			stream.Write([]byte("test"))
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 2: Update window
	go func() {
		for i := 0; i < 100; i++ {
			stream.IncrementSendWindow(100)
			stream.ConsumeSendWindow(50)
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Goroutine 3: Set priority
	go func() {
		for i := 0; i < 100; i++ {
			stream.SetPriority(uint8(i%256), 0, false) // Ignore errors in concurrent test
			time.Sleep(time.Microsecond)
		}
		done <- true
	}()

	// Wait for all goroutines
	for i := 0; i < 3; i++ {
		<-done
	}
}
