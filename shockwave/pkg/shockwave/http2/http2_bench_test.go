package http2

import (
	"sync"
	"testing"
)

// BenchmarkStreamCreation benchmarks stream creation performance
func BenchmarkStreamCreation(b *testing.B) {
	conn := NewConnection(true)
	conn.remoteSettings.MaxConcurrentStreams = 100

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		stream, err := conn.CreateStream()
		if err != nil {
			b.Fatal(err)
		}
		// Close stream immediately to avoid hitting limit
		conn.CloseStream(stream.ID())
	}
}

// BenchmarkConcurrentStreamCreation benchmarks concurrent stream creation
func BenchmarkConcurrentStreamCreation(b *testing.B) {
	conn := NewConnection(true)
	conn.remoteSettings.MaxConcurrentStreams = 10000 // High limit

	b.ResetTimer()

	b.RunParallel(func(pb *testing.PB) {
		for pb.Next() {
			_, err := conn.CreateStream()
			if err != nil {
				// Expected when hitting limit
				continue
			}
		}
	})
}

// BenchmarkStreamStateTransitions benchmarks state transitions
func BenchmarkStreamStateTransitions(b *testing.B) {
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		stream := NewStream(uint32(i), 65535)
		stream.Open()
		stream.CloseLocal()
		stream.CloseRemote()
	}
}

// BenchmarkFlowControlSend benchmarks sending with flow control
func BenchmarkFlowControlSend(b *testing.B) {
	fc := NewFlowController()
	stream := NewStream(1, 65535)
	data := make([]byte, 1024) // 1KB chunks

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		// Reset windows periodically
		if i%64 == 0 {
			fc.IncrementConnectionSendWindow(64 * 1024)
			stream.IncrementSendWindow(64 * 1024)
		}

		_, err := fc.SendData(stream, data)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkFlowControlReceive benchmarks receiving with flow control
func BenchmarkFlowControlReceive(b *testing.B) {
	fc := NewFlowController()
	stream := NewStream(1, 65535)

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		// Reset windows periodically
		if i%64 == 0 {
			fc.IncrementConnectionRecvWindow(64 * 1024)
			stream.IncrementRecvWindow(64 * 1024)
		}

		err := fc.ReceiveData(stream, 1024)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStreamIO benchmarks stream read/write operations
func BenchmarkStreamIO(b *testing.B) {
	conn := NewConnection(true)
	stream, _ := conn.CreateStream()
	data := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024)

	for i := 0; i < b.N; i++ {
		// Simulate receiving
		stream.ReceiveData(data)

		// Read from input buffer
		buf := make([]byte, 1024)
		_, err := stream.Read(buf)
		if err != nil {
			b.Fatal(err)
		}

		// Write to output buffer
		_, err = stream.Write(data)
		if err != nil {
			b.Fatal(err)
		}

		// Drain output buffer (simulate sending)
		stream.sendBufMu.Lock()
		stream.sendBuf = stream.sendBuf[:0]
		stream.sendBufMu.Unlock()
	}
}

// BenchmarkConcurrentStreamIO benchmarks concurrent I/O on multiple streams
func BenchmarkConcurrentStreamIO(b *testing.B) {
	conn := NewConnection(true)
	conn.remoteSettings.MaxConcurrentStreams = 100

	// Create streams
	streams := make([]*Stream, 100)
	for i := 0; i < 100; i++ {
		s, err := conn.CreateStream()
		if err != nil {
			b.Fatal(err)
		}
		streams[i] = s
	}

	data := make([]byte, 1024)

	b.ResetTimer()
	b.SetBytes(1024 * 100) // 100 streams * 1KB

	for i := 0; i < b.N; i++ {
		var wg sync.WaitGroup
		for j := 0; j < 100; j++ {
			wg.Add(1)
			go func(s *Stream) {
				defer wg.Done()
				s.Write(data)
			}(streams[j])
		}
		wg.Wait()
	}
}

// BenchmarkWindowUpdate benchmarks window update calculations
func BenchmarkWindowUpdate(b *testing.B) {
	fc := NewFlowController()
	currentWindow := int32(32768)
	initialWindow := int32(65535)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		if fc.ShouldSendWindowUpdate(currentWindow, initialWindow) {
			fc.CalculateWindowUpdate(currentWindow, initialWindow)
		}
	}
}

// BenchmarkHPACKEncoding benchmarks HPACK encoding in connection
func BenchmarkHPACKEncoding(b *testing.B) {
	conn := NewConnection(true)

	headers := []HeaderField{
		{":method", "GET"},
		{":path", "/"},
		{":scheme", "https"},
		{":authority", "example.com"},
		{"user-agent", "benchmark"},
		{"accept", "*/*"},
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		conn.EncodeHeaders(headers)
	}
}

// BenchmarkHPACKDecoding benchmarks HPACK decoding in connection
func BenchmarkHPACKDecoding(b *testing.B) {
	conn := NewConnection(true)

	headers := []HeaderField{
		{":method", "GET"},
		{":path", "/"},
		{":scheme", "https"},
		{":authority", "example.com"},
	}

	encoded := conn.EncodeHeaders(headers)

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, err := conn.DecodeHeaders(encoded)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkPriorityTree benchmarks priority tree operations
func BenchmarkPriorityTree(b *testing.B) {
	pt := NewPriorityTree()

	// Pre-populate with some streams
	for i := uint32(1); i <= 100; i += 2 {
		pt.AddStream(i, 0, 16, false)
	}

	b.ResetTimer()

	b.Run("AddStream", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			streamID := uint32(1000 + i*2)
			pt.AddStream(streamID, 0, 16, false)
		}
	})

	b.Run("UpdatePriority", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			streamID := uint32(1 + (i%50)*2)
			pt.UpdatePriority(streamID, 3, 20, false) // Ignore error in benchmark
		}
	})

	b.Run("CalculateWeight", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			streamID := uint32(1 + (i%50)*2)
			pt.CalculateWeight(streamID)
		}
	})

	b.Run("RemoveStream", func(b *testing.B) {
		for i := 0; i < b.N; i++ {
			streamID := uint32(1 + (i%50)*2)
			pt.RemoveStream(streamID)
			// Re-add for next iteration
			pt.AddStream(streamID, 0, 16, false)
		}
	})
}

// BenchmarkConnectionSettings benchmarks settings updates
func BenchmarkConnectionSettings(b *testing.B) {
	conn := NewConnection(true)

	settings := Settings{
		HeaderTableSize:      8192,
		EnablePush:           false,
		MaxConcurrentStreams: 200,
		InitialWindowSize:    32768,
		MaxFrameSize:         32768,
		MaxHeaderListSize:    8192,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		err := conn.UpdateSettings(settings)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkStreamLifecycle benchmarks full stream lifecycle
func BenchmarkStreamLifecycle(b *testing.B) {
	conn := NewConnection(true)
	conn.remoteSettings.MaxConcurrentStreams = 10000

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		stream, err := conn.CreateStream()
		if err != nil {
			continue
		}

		stream.Open()
		stream.Write([]byte("test"))
		stream.CloseLocal()
		conn.CloseStream(stream.ID())
	}
}

// BenchmarkChunkData benchmarks data chunking
func BenchmarkChunkData(b *testing.B) {
	fc := NewFlowController()
	fc.SetMaxFrameSize(16384)
	stream := NewStream(1, 65535)

	data := make([]byte, 100000) // 100KB

	b.ResetTimer()
	b.SetBytes(100000)

	for i := 0; i < b.N; i++ {
		chunks := fc.ChunkData(data, stream)
		_ = chunks
	}
}
