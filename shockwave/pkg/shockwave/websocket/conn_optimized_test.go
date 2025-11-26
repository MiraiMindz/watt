package websocket

import (
	"bytes"
	"io"
	"testing"
)

// TestReadMessageInto tests the zero-allocation ReadMessageInto API
func TestReadMessageInto(t *testing.T) {
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	// Write a simple masked message
	payload := []byte("Hello, WebSocket!")
	payloadCopy := make([]byte, len(payload))
	copy(payloadCopy, payload)
	fw.WriteFrame(OpcodeText, true, payloadCopy, &maskKey)

	// Create connection
	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	// Read into pre-allocated buffer
	readBuf := make([]byte, 1024)
	msgType, n, err := conn.ReadMessageInto(readBuf)
	if err != nil {
		t.Fatalf("ReadMessageInto failed: %v", err)
	}

	if msgType != TextMessage {
		t.Errorf("Expected TextMessage, got %v", msgType)
	}

	if n != len(payload) {
		t.Errorf("Expected %d bytes, got %d", len(payload), n)
	}

	if string(readBuf[:n]) != string(payload) {
		t.Errorf("Expected %q, got %q", payload, readBuf[:n])
	}
}

// TestReadMessageIntoFragmented tests fragmented message with ReadMessageInto
func TestReadMessageIntoFragmented(t *testing.T) {
	var buf bytes.Buffer
	fw := NewFrameWriter(&buf)
	maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

	// Write fragmented message
	payload1 := make([]byte, len("Hello, "))
	copy(payload1, "Hello, ")
	fw.WriteFrame(OpcodeText, false, payload1, &maskKey)

	payload2 := make([]byte, len("WebSocket!"))
	copy(payload2, "WebSocket!")
	fw.WriteFrame(OpcodeContinuation, true, payload2, &maskKey)

	conn := newConn(&mockConn{reader: &buf, writer: io.Discard}, true, 4096, 4096, "")

	readBuf := make([]byte, 1024)
	msgType, n, err := conn.ReadMessageInto(readBuf)
	if err != nil {
		t.Fatalf("ReadMessageInto failed: %v", err)
	}

	if msgType != TextMessage {
		t.Errorf("Expected TextMessage, got %v", msgType)
	}

	expected := "Hello, WebSocket!"
	if string(readBuf[:n]) != expected {
		t.Errorf("Expected %q, got %q", expected, readBuf[:n])
	}
}

// BenchmarkReadMessageInto benchmarks the zero-allocation ReadMessageInto API
func BenchmarkReadMessageInto(b *testing.B) {
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

	// Pre-allocate read buffer (simulating buffer pool usage)
	readBuf := make([]byte, 4096)

	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := conn.ReadMessageInto(readBuf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkReadMessageVsReadMessageInto compares allocations
func BenchmarkReadMessageVsReadMessageInto(b *testing.B) {
	prepareData := func(n int) *bytes.Buffer {
		var buf bytes.Buffer
		fw := NewFrameWriter(&buf)
		data := make([]byte, 1024)
		maskKey := [4]byte{0x12, 0x34, 0x56, 0x78}

		for i := 0; i < n; i++ {
			payload := make([]byte, len(data))
			copy(payload, data)
			fw.WriteFrame(OpcodeBinary, true, payload, &maskKey)
		}
		return &buf
	}

	b.Run("ReadMessage", func(b *testing.B) {
		buf := prepareData(b.N)
		conn := newConn(&mockConn{reader: buf, writer: io.Discard}, true, 4096, 4096, "")

		b.SetBytes(1024)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _, err := conn.ReadMessage()
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ReadMessageInto", func(b *testing.B) {
		buf := prepareData(b.N)
		conn := newConn(&mockConn{reader: buf, writer: io.Discard}, true, 4096, 4096, "")
		readBuf := make([]byte, 4096)

		b.SetBytes(1024)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			_, _, err := conn.ReadMessageInto(readBuf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("ReadMessageInto+Pool", func(b *testing.B) {
		buf := prepareData(b.N)
		conn := newConn(&mockConn{reader: buf, writer: io.Discard}, true, 4096, 4096, "")
		pool := &BufferPool{}

		b.SetBytes(1024)
		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			readBuf := pool.Get(4096)
			_, _, err := conn.ReadMessageInto(readBuf)
			if err != nil {
				b.Fatal(err)
			}
			pool.Put(readBuf)
		}
	})
}

// BenchmarkBufferPooledReading shows the performance with buffer pooling
func BenchmarkBufferPooledReading(b *testing.B) {
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
	pool := DefaultBufferPool

	b.SetBytes(1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Get buffer from pool
		readBuf := pool.Get(4096)

		// Read message
		_, _, err := conn.ReadMessageInto(readBuf)
		if err != nil {
			b.Fatal(err)
		}

		// Return buffer to pool
		pool.Put(readBuf)
	}
}
