package websocket

import (
	"bytes"
	"io"
	"net"
	"testing"
	"time"
)

// mockConn is a mock net.Conn for testing
type mockConn struct {
	reader io.Reader
	writer io.Writer
}

func (m *mockConn) Read(b []byte) (n int, err error) {
	return m.reader.Read(b)
}

func (m *mockConn) Write(b []byte) (n int, err error) {
	return m.writer.Write(b)
}

func (m *mockConn) Close() error                       { return nil }
func (m *mockConn) LocalAddr() net.Addr                { return nil }
func (m *mockConn) RemoteAddr() net.Addr               { return nil }
func (m *mockConn) SetDeadline(t time.Time) error      { return nil }
func (m *mockConn) SetReadDeadline(t time.Time) error  { return nil }
func (m *mockConn) SetWriteDeadline(t time.Time) error { return nil }

func TestConnReadMessageSimple(t *testing.T) {
	// Create a simple text frame (masked, as client would send)
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}
	payload := []byte("Hello, WebSocket!")
	payloadCopy := make([]byte, len(payload))
	copy(payloadCopy, payload)
	fw.WriteFrame(OpcodeText, true, payloadCopy, &maskKey)

	// Create connection (server expects masked frames)
	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	// Read message
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if msgType != TextMessage {
		t.Errorf("Expected TextMessage, got %v", msgType)
	}

	if string(data) != "Hello, WebSocket!" {
		t.Errorf("Expected 'Hello, WebSocket!', got %q", data)
	}
}

func TestConnReadMessageFragmented(t *testing.T) {
	// Create fragmented message: "Hello, " + "WebSocket!"
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	// First fragment (not final, masked)
	payload1 := make([]byte, len("Hello, "))
	copy(payload1, "Hello, ")
	fw.WriteFrame(OpcodeText, false, payload1, &maskKey)

	// Second fragment (final, continuation, masked)
	payload2 := make([]byte, len("WebSocket!"))
	copy(payload2, "WebSocket!")
	fw.WriteFrame(OpcodeContinuation, true, payload2, &maskKey)

	// Create connection (server expects masked frames)
	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	// Read message (should automatically assemble fragments)
	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if msgType != TextMessage {
		t.Errorf("Expected TextMessage, got %v", msgType)
	}

	if string(data) != "Hello, WebSocket!" {
		t.Errorf("Expected 'Hello, WebSocket!', got %q", data)
	}
}

func TestConnReadMessageMultiFragmented(t *testing.T) {
	// Create message with multiple fragments: "A" + "B" + "C" + "D"
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	payloads := [][]byte{[]byte("A"), []byte("B"), []byte("C"), []byte("D")}
	for i, p := range payloads {
		payload := make([]byte, len(p))
		copy(payload, p)
		opcode := byte(OpcodeContinuation)
		if i == 0 {
			opcode = OpcodeText
		}
		fin := i == len(payloads)-1
		fw.WriteFrame(opcode, fin, payload, &maskKey)
	}

	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	msgType, data, err := conn.ReadMessage()
	if err != nil {
		t.Fatalf("ReadMessage failed: %v", err)
	}

	if msgType != TextMessage {
		t.Errorf("Expected TextMessage, got %v", msgType)
	}

	if string(data) != "ABCD" {
		t.Errorf("Expected 'ABCD', got %q", data)
	}
}

func TestConnPingPong(t *testing.T) {
	// Server receives Ping, should send Pong automatically
	var readBuf bytes.Buffer
	var writeBuf bytes.Buffer
	fw := NewFrameWriter(&readBuf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	// Send masked Ping frame (client→server)
	payload := make([]byte, len("ping"))
	copy(payload, "ping")
	fw.WriteControlFrame(OpcodePing, payload, &maskKey)

	conn := newConn(&mockConn{reader: &readBuf, writer: &writeBuf}, true, 4096, 4096, "")

	// Try to read message - Ping should be handled automatically
	// Since there's no data frame after Ping, we should get io.EOF
	go func() {
		_, _, _ = conn.ReadMessage()
	}()

	// Give it a moment to process
	time.Sleep(10 * time.Millisecond)

	// Check that Pong was sent
	fr := NewFrameReader(&writeBuf)
	frame, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("Failed to read Pong frame: %v", err)
	}

	if frame.Opcode != OpcodePong {
		t.Errorf("Expected Pong frame, got opcode 0x%X", frame.Opcode)
	}

	if string(frame.Payload) != "ping" {
		t.Errorf("Expected Pong payload 'ping', got %q", frame.Payload)
	}
}

func TestConnWriteMessage(t *testing.T) {
	var buf bytes.Buffer
	conn := newConn(&mockConn{reader: bytes.NewReader(nil), writer: &buf}, true, 4096, 4096, "")

	// Write text message
	err := conn.WriteMessage(TextMessage, []byte("Hello, WebSocket!"))
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read back the frame
	fr := NewFrameReader(&buf)
	frame, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("Failed to read frame: %v", err)
	}

	if frame.Opcode != OpcodeText {
		t.Errorf("Expected OpcodeText, got 0x%X", frame.Opcode)
	}

	if !frame.Fin {
		t.Error("Expected FIN=true")
	}

	if string(frame.Payload) != "Hello, WebSocket!" {
		t.Errorf("Expected 'Hello, WebSocket!', got %q", frame.Payload)
	}
}

func TestConnClientMasking(t *testing.T) {
	// Client must mask frames
	var buf bytes.Buffer
	conn := newConn(&mockConn{reader: bytes.NewReader(nil), writer: &buf}, false, 4096, 4096, "")

	// Write text message
	err := conn.WriteMessage(TextMessage, []byte("Test"))
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read back the frame
	fr := NewFrameReader(&buf)
	frame, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("Failed to read frame: %v", err)
	}

	// Client frame must be masked
	if !frame.Masked {
		t.Error("Client frame must be masked")
	}

	if string(frame.Payload) != "Test" {
		t.Errorf("Expected 'Test', got %q", frame.Payload)
	}
}

func TestConnServerNoMasking(t *testing.T) {
	// Server must not mask frames
	var buf bytes.Buffer
	conn := newConn(&mockConn{reader: bytes.NewReader(nil), writer: &buf}, true, 4096, 4096, "")

	// Write text message
	err := conn.WriteMessage(TextMessage, []byte("Test"))
	if err != nil {
		t.Fatalf("WriteMessage failed: %v", err)
	}

	// Read back the frame
	fr := NewFrameReader(&buf)
	frame, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("Failed to read frame: %v", err)
	}

	// Server frame must not be masked
	if frame.Masked {
		t.Error("Server frame must not be masked")
	}

	if string(frame.Payload) != "Test" {
		t.Errorf("Expected 'Test', got %q", frame.Payload)
	}
}

func TestConnInvalidUTF8(t *testing.T) {
	// Create text frame with invalid UTF-8
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}
	invalidUTF8 := []byte{0xFF, 0xFE, 0xFD} // Invalid UTF-8
	payload := make([]byte, len(invalidUTF8))
	copy(payload, invalidUTF8)
	fw.WriteFrame(OpcodeText, true, payload, &maskKey)

	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	// Read message - should fail with UTF-8 error
	_, _, err := conn.ReadMessage()
	if err != ErrInvalidUTF8 {
		t.Errorf("Expected ErrInvalidUTF8, got %v", err)
	}
}

func TestConnProtocolViolationContinuationWithoutStart(t *testing.T) {
	// Send continuation frame without starting a fragmented message
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}
	payload := make([]byte, len("test"))
	copy(payload, "test")
	fw.WriteFrame(OpcodeContinuation, true, payload, &maskKey)

	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	_, _, err := conn.ReadMessage()
	if err != ErrProtocolViolation {
		t.Errorf("Expected ErrProtocolViolation, got %v", err)
	}
}

func TestConnProtocolViolationDataDuringFragmentation(t *testing.T) {
	// Start fragmented message, then send another data frame (not continuation)
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	payload1 := make([]byte, len("start"))
	copy(payload1, "start")
	fw.WriteFrame(OpcodeText, false, payload1, &maskKey)

	payload2 := make([]byte, len("invalid"))
	copy(payload2, "invalid")
	fw.WriteFrame(OpcodeText, true, payload2, &maskKey) // Should be continuation

	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	_, _, err := conn.ReadMessage()
	if err != ErrProtocolViolation {
		t.Errorf("Expected ErrProtocolViolation, got %v", err)
	}
}

func TestConnMessageTooLarge(t *testing.T) {
	// Set small max message size
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}
	largeData := make([]byte, 1000)
	fw.WriteFrame(OpcodeText, true, largeData, &maskKey)

	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")
	conn.SetMaxMessageSize(500) // Set limit to 500 bytes

	_, _, err := conn.ReadMessage()
	if err != ErrMessageTooLarge {
		t.Errorf("Expected ErrMessageTooLarge, got %v", err)
	}
}

func TestConnClose(t *testing.T) {
	var writeBuf bytes.Buffer
	conn := newConn(&mockConn{reader: bytes.NewReader(nil), writer: &writeBuf}, true, 4096, 4096, "")

	err := conn.Close()
	if err != nil {
		t.Fatalf("Close failed: %v", err)
	}

	// Check that close frame was sent
	fr := NewFrameReader(&writeBuf)
	frame, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("Failed to read Close frame: %v", err)
	}

	if frame.Opcode != OpcodeClose {
		t.Errorf("Expected Close frame, got opcode 0x%X", frame.Opcode)
	}

	// Check close code (1000 = normal closure)
	if len(frame.Payload) >= 2 {
		code := uint16(frame.Payload[0])<<8 | uint16(frame.Payload[1])
		if code != CloseNormalClosure {
			t.Errorf("Expected close code %d, got %d", CloseNormalClosure, code)
		}
	}
}

func TestConnCloseWithCode(t *testing.T) {
	var writeBuf bytes.Buffer
	conn := newConn(&mockConn{reader: bytes.NewReader(nil), writer: &writeBuf}, true, 4096, 4096, "")

	err := conn.CloseWithCode(CloseGoingAway, "goodbye")
	if err != nil {
		t.Fatalf("CloseWithCode failed: %v", err)
	}

	// Check that close frame was sent
	fr := NewFrameReader(&writeBuf)
	frame, err := fr.ReadFrame()
	if err != nil {
		t.Fatalf("Failed to read Close frame: %v", err)
	}

	if frame.Opcode != OpcodeClose {
		t.Errorf("Expected Close frame, got opcode 0x%X", frame.Opcode)
	}

	// Check close code
	if len(frame.Payload) >= 2 {
		code := uint16(frame.Payload[0])<<8 | uint16(frame.Payload[1])
		if code != CloseGoingAway {
			t.Errorf("Expected close code %d, got %d", CloseGoingAway, code)
		}

		reason := string(frame.Payload[2:])
		if reason != "goodbye" {
			t.Errorf("Expected close reason 'goodbye', got %q", reason)
		}
	}
}

// Benchmarks

func BenchmarkConnWriteMessage(b *testing.B) {
	conn := newConn(&mockConn{reader: bytes.NewReader(nil), writer: io.Discard}, true, 4096, 4096, "")
	data := make([]byte, 1024)

	b.SetBytes(1024)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := conn.WriteMessage(BinaryMessage, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConnReadMessage(b *testing.B) {
	// Prepare frames
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	data := make([]byte, 1024)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	for i := 0; i < b.N; i++ {
		payload := make([]byte, len(data))
		copy(payload, data)
		fw.WriteFrame(OpcodeBinary, true, payload, &maskKey)
	}

	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	b.SetBytes(1024)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _, err := conn.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
	}
}

func BenchmarkConnReadWriteRoundtrip(b *testing.B) {
	// Use pipe for bidirectional communication
	serverReader, clientWriter := io.Pipe()
	clientReader, serverWriter := io.Pipe()

	serverConn := newConn(&mockConn{reader: serverReader, writer: serverWriter}, true, 4096, 4096, "")
	clientConn := newConn(&mockConn{reader: clientReader, writer: clientWriter}, false, 4096, 4096, "")

	data := make([]byte, 1024)

	b.SetBytes(1024)
	b.ResetTimer()

	done := make(chan error, 1)

	go func() {
		for i := 0; i < b.N; i++ {
			_, _, err := serverConn.ReadMessage()
			if err != nil {
				done <- err
				return
			}
			err = serverConn.WriteMessage(BinaryMessage, data)
			if err != nil {
				done <- err
				return
			}
		}
		done <- nil
	}()

	for i := 0; i < b.N; i++ {
		err := clientConn.WriteMessage(BinaryMessage, data)
		if err != nil {
			b.Fatal(err)
		}
		_, _, err = clientConn.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
	}

	if err := <-done; err != nil {
		b.Fatal(err)
	}
}
