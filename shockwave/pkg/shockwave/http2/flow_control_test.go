package http2

import (
	"testing"
)

func TestNewFlowController(t *testing.T) {
	fc := NewFlowController()

	if fc.ConnectionSendWindow() != int32(DefaultWindowSize) {
		t.Errorf("connection send window = %d, want %d", fc.ConnectionSendWindow(), DefaultWindowSize)
	}

	if fc.InitialWindowSize() != int32(DefaultWindowSize) {
		t.Errorf("initial window size = %d, want %d", fc.InitialWindowSize(), DefaultWindowSize)
	}
}

func TestFlowControllerWindowOperations(t *testing.T) {
	fc := NewFlowController()

	// Test increment
	if err := fc.IncrementConnectionSendWindow(1000); err != nil {
		t.Fatalf("IncrementConnectionSendWindow() error: %v", err)
	}

	expected := int32(DefaultWindowSize) + 1000
	if fc.ConnectionSendWindow() != expected {
		t.Errorf("connection send window = %d, want %d", fc.ConnectionSendWindow(), expected)
	}

	// Test consume
	if err := fc.ConsumeConnectionSendWindow(1000); err != nil {
		t.Fatalf("ConsumeConnectionSendWindow() error: %v", err)
	}

	if fc.ConnectionSendWindow() != int32(DefaultWindowSize) {
		t.Errorf("connection send window = %d, want %d", fc.ConnectionSendWindow(), DefaultWindowSize)
	}

	// Test overflow protection
	fc.connSendWindow = MaxWindowSize
	err := fc.IncrementConnectionSendWindow(1)
	if err == nil {
		t.Error("expected overflow error, got nil")
	}
}

func TestFlowControllerSendData(t *testing.T) {
	fc := NewFlowController()
	stream := NewStream(1, 65535)

	data := make([]byte, 1000)
	sent, err := fc.SendData(stream, data)
	if err != nil {
		t.Fatalf("SendData() error: %v", err)
	}

	if sent != 1000 {
		t.Errorf("sent %d bytes, want 1000", sent)
	}

	// Check windows consumed
	if stream.SendWindow() != 65535-1000 {
		t.Errorf("stream send window = %d, want %d", stream.SendWindow(), 65535-1000)
	}

	if fc.ConnectionSendWindow() != int32(DefaultWindowSize)-1000 {
		t.Errorf("connection send window = %d, want %d", fc.ConnectionSendWindow(), int32(DefaultWindowSize)-1000)
	}
}

func TestFlowControllerReceiveData(t *testing.T) {
	fc := NewFlowController()
	stream := NewStream(1, 65535)

	if err := fc.ReceiveData(stream, 1000); err != nil {
		t.Fatalf("ReceiveData() error: %v", err)
	}

	// Check windows consumed
	if stream.RecvWindow() != 65535-1000 {
		t.Errorf("stream recv window = %d, want %d", stream.RecvWindow(), 65535-1000)
	}

	if fc.ConnectionRecvWindow() != int32(DefaultWindowSize)-1000 {
		t.Errorf("connection recv window = %d, want %d", fc.ConnectionRecvWindow(), int32(DefaultWindowSize)-1000)
	}
}

func TestFlowControllerWindowUpdate(t *testing.T) {
	fc := NewFlowController()

	initialWindow := int32(65535)
	currentWindow := int32(20000)

	// Should send WINDOW_UPDATE if below 50% threshold
	if !fc.ShouldSendWindowUpdate(currentWindow, initialWindow) {
		t.Error("ShouldSendWindowUpdate = false, want true")
	}

	// Calculate increment
	increment := fc.CalculateWindowUpdate(currentWindow, initialWindow)
	expected := initialWindow - currentWindow

	if increment != expected {
		t.Errorf("window update = %d, want %d", increment, expected)
	}

	// Should not send if above threshold
	if fc.ShouldSendWindowUpdate(initialWindow, initialWindow) {
		t.Error("ShouldSendWindowUpdate = true, want false (window full)")
	}
}

func TestFlowControllerChunkData(t *testing.T) {
	fc := NewFlowController()
	fc.SetMaxFrameSize(16384)

	stream := NewStream(1, 65535)

	// Data larger than max frame size
	data := make([]byte, 50000)

	chunks := fc.ChunkData(data, stream)

	if len(chunks) == 0 {
		t.Fatal("no chunks returned")
	}

	// First chunk should be max frame size
	if len(chunks[0]) != 16384 {
		t.Errorf("first chunk size = %d, want 16384", len(chunks[0]))
	}

	// Total should equal original (up to window limits)
	total := 0
	for _, chunk := range chunks {
		total += len(chunk)
	}

	// Should be limited by window size (65535)
	if total > 65535 {
		t.Errorf("total chunked = %d, exceeds window 65535", total)
	}
}

func TestFlowControllerSetInitialWindowSize(t *testing.T) {
	fc := NewFlowController()

	newSize := int32(32768)
	if err := fc.SetInitialWindowSize(newSize); err != nil {
		t.Fatalf("SetInitialWindowSize() error: %v", err)
	}

	if fc.InitialWindowSize() != newSize {
		t.Errorf("initial window size = %d, want %d", fc.InitialWindowSize(), newSize)
	}

	// Test invalid size
	err := fc.SetInitialWindowSize(-1)
	if err == nil {
		t.Error("expected error for negative window size, got nil")
	}

	err = fc.SetInitialWindowSize(MaxWindowSize)
	if err != nil {
		t.Errorf("unexpected error for max window size: %v", err)
	}
}

func TestFlowControllerStats(t *testing.T) {
	fc := NewFlowController()
	stream := NewStream(1, 65535)

	stats := fc.GetStats(stream)

	if stats.ConnectionSend != int32(DefaultWindowSize) {
		t.Errorf("ConnectionSend = %d, want %d", stats.ConnectionSend, DefaultWindowSize)
	}

	if stats.StreamSend != 65535 {
		t.Errorf("StreamSend = %d, want 65535", stats.StreamSend)
	}

	if stats.MaxFrameSize != DefaultMaxFrameSize {
		t.Errorf("MaxFrameSize = %d, want %d", stats.MaxFrameSize, DefaultMaxFrameSize)
	}
}
