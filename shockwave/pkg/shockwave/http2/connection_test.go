package http2

import (
	"sync"
	"testing"
	"time"
)

func TestNewConnection(t *testing.T) {
	conn := NewConnection(true)

	if conn.isClient != true {
		t.Error("expected client connection")
	}

	if conn.IsClosed() {
		t.Error("new connection should not be closed")
	}

	if conn.nextStreamID != 1 {
		t.Errorf("next stream ID = %d, want 1 (client)", conn.nextStreamID)
	}

	// Test server connection
	serverConn := NewConnection(false)
	if serverConn.nextStreamID != 2 {
		t.Errorf("next stream ID = %d, want 2 (server)", serverConn.nextStreamID)
	}
}

func TestConnectionCreateStream(t *testing.T) {
	conn := NewConnection(true)

	stream, err := conn.CreateStream()
	if err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}

	if stream.ID() != 1 {
		t.Errorf("stream ID = %d, want 1", stream.ID())
	}

	// Create another stream
	stream2, err := conn.CreateStream()
	if err != nil {
		t.Fatalf("CreateStream() error: %v", err)
	}

	if stream2.ID() != 3 {
		t.Errorf("stream ID = %d, want 3 (next odd)", stream2.ID())
	}

	// Verify stream is in map
	retrieved, exists := conn.GetStream(1)
	if !exists {
		t.Error("stream not found in connection map")
	}

	if retrieved != stream {
		t.Error("retrieved stream doesn't match created stream")
	}
}

func TestConnectionConcurrentStreamCreation(t *testing.T) {
	conn := NewConnection(true)

	var wg sync.WaitGroup
	streamCount := 50

	for i := 0; i < streamCount; i++ {
		wg.Add(1)
		go func() {
			defer wg.Done()
			_, err := conn.CreateStream()
			if err != nil {
				t.Errorf("CreateStream() error: %v", err)
			}
		}()
	}

	wg.Wait()

	if conn.ActiveStreams() != uint32(streamCount) {
		t.Errorf("active streams = %d, want %d", conn.ActiveStreams(), streamCount)
	}
}

func TestConnectionStreamLimit(t *testing.T) {
	conn := NewConnection(true)

	// Set low concurrent stream limit
	settings := conn.localSettings
	settings.MaxConcurrentStreams = 5
	conn.UpdateSettings(settings)

	conn.remoteSettings.MaxConcurrentStreams = 5

	// Create up to limit
	for i := 0; i < 5; i++ {
		_, err := conn.CreateStream()
		if err != nil {
			t.Fatalf("CreateStream() error at %d: %v", i, err)
		}
	}

	// Next should fail
	_, err := conn.CreateStream()
	if err == nil {
		t.Error("expected error exceeding max concurrent streams, got nil")
	}
}

func TestConnectionCloseStream(t *testing.T) {
	conn := NewConnection(true)

	stream, _ := conn.CreateStream()
	streamID := stream.ID()

	if err := conn.CloseStream(streamID); err != nil {
		t.Fatalf("CloseStream() error: %v", err)
	}

	_, exists := conn.GetStream(streamID)
	if exists {
		t.Error("stream still exists after close")
	}

	stats := conn.Stats()
	if stats.StreamsCreated != 1 {
		t.Errorf("streams created = %d, want 1", stats.StreamsCreated)
	}

	if stats.StreamsClosed != 1 {
		t.Errorf("streams closed = %d, want 1", stats.StreamsClosed)
	}
}

func TestConnectionSettings(t *testing.T) {
	conn := NewConnection(true)

	newSettings := Settings{
		HeaderTableSize:      8192,
		EnablePush:           false,
		MaxConcurrentStreams: 200,
		InitialWindowSize:    32768,
		MaxFrameSize:         32768,
		MaxHeaderListSize:    8192,
	}

	if err := conn.UpdateSettings(newSettings); err != nil {
		t.Fatalf("UpdateSettings() error: %v", err)
	}

	if conn.localSettings.HeaderTableSize != 8192 {
		t.Errorf("header table size = %d, want 8192", conn.localSettings.HeaderTableSize)
	}

	if conn.localSettings.InitialWindowSize != 32768 {
		t.Errorf("initial window size = %d, want 32768", conn.localSettings.InitialWindowSize)
	}
}

func TestConnectionGoAway(t *testing.T) {
	conn := NewConnection(true)

	// Create some streams
	for i := 0; i < 3; i++ {
		conn.CreateStream()
	}

	if err := conn.GoAway(1, ErrCodeNo); err != nil {
		t.Fatalf("GoAway() error: %v", err)
	}

	conn.stateMu.RLock()
	state := conn.state
	conn.stateMu.RUnlock()

	if state != ConnectionStateGoingAway {
		t.Errorf("state = %v, want going away", state)
	}

	// Context should be cancelled
	select {
	case <-conn.Context().Done():
		// Expected
	case <-time.After(100 * time.Millisecond):
		t.Error("context not cancelled after GoAway")
	}
}

func TestConnectionClose(t *testing.T) {
	conn := NewConnection(true)

	// Create streams
	for i := 0; i < 5; i++ {
		conn.CreateStream()
	}

	if err := conn.Close(); err != nil {
		t.Fatalf("Close() error: %v", err)
	}

	if !conn.IsClosed() {
		t.Error("connection not marked as closed")
	}

	// All streams should be removed
	streamCount := conn.streams.Len()

	if streamCount != 0 {
		t.Errorf("stream count = %d, want 0 after close", streamCount)
	}
}

func TestConnectionHPACK(t *testing.T) {
	conn := NewConnection(true)

	headers := []HeaderField{
		{":method", "GET"},
		{":path", "/"},
		{":scheme", "https"},
	}

	// Encode
	encoded := conn.EncodeHeaders(headers)
	if len(encoded) == 0 {
		t.Fatal("encoded headers are empty")
	}

	// Decode
	decoded, err := conn.DecodeHeaders(encoded)
	if err != nil {
		t.Fatalf("DecodeHeaders() error: %v", err)
	}

	if len(decoded) != len(headers) {
		t.Errorf("decoded %d headers, want %d", len(decoded), len(headers))
	}

	for i, h := range decoded {
		if h.Name != headers[i].Name || h.Value != headers[i].Value {
			t.Errorf("header %d = %+v, want %+v", i, h, headers[i])
		}
	}
}

func TestPriorityTree(t *testing.T) {
	pt := NewPriorityTree()

	// Add streams
	pt.AddStream(1, 0, 16, false)
	pt.AddStream(3, 1, 16, false)
	pt.AddStream(5, 1, 16, false)

	// Calculate weight
	weight := pt.CalculateWeight(1)
	if weight != 17 { // weight + 1
		t.Errorf("weight = %d, want 17", weight)
	}

	// Update priority
	err := pt.UpdatePriority(3, 5, 20, true)
	if err != nil {
		t.Fatalf("UpdatePriority() error: %v", err)
	}

	// Test self-dependency rejection
	err = pt.UpdatePriority(3, 3, 16, false)
	if err != ErrStreamSelfDependency {
		t.Errorf("expected ErrStreamSelfDependency, got: %v", err)
	}

	// Test cycle detection: create a cycle 1 -> 3 -> 5 -> 1
	// First: 3 depends on 5 (already done above)
	// Second: 5 depends on 1
	err = pt.UpdatePriority(5, 1, 16, false)
	if err != nil {
		t.Fatalf("UpdatePriority(5, 1) error: %v", err)
	}
	// Third: Try to make 1 depend on 3 (should break the cycle)
	err = pt.UpdatePriority(1, 3, 16, false)
	// This should succeed but break the cycle by depending on root
	if err != nil {
		t.Fatalf("UpdatePriority(1, 3) error: %v", err)
	}

	// Remove stream
	pt.RemoveStream(1)

	// Check stream 3 was reparented
	pt.mu.RLock()
	node := pt.streams[3]
	pt.mu.RUnlock()

	if node == nil {
		t.Fatal("stream 3 not found after reparenting")
	}
}

func TestConnectionStats(t *testing.T) {
	conn := NewConnection(true)

	// Create and close streams
	for i := 0; i < 10; i++ {
		stream, _ := conn.CreateStream()
		conn.CloseStream(stream.ID())
	}

	stats := conn.Stats()

	if stats.StreamsCreated != 10 {
		t.Errorf("streams created = %d, want 10", stats.StreamsCreated)
	}

	if stats.StreamsClosed != 10 {
		t.Errorf("streams closed = %d, want 10", stats.StreamsClosed)
	}
}
