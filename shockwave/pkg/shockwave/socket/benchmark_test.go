package socket

import (
	"io"
	"net"
	"os"
	"testing"
	"time"
)

// BenchmarkApply benchmarks applying socket options
func BenchmarkApply(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	cfg := DefaultConfig()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Connect
		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			b.Fatal(err)
		}

		// Accept
		serverConn, err := listener.Accept()
		if err != nil {
			b.Fatal(err)
		}

		// Apply options
		Apply(serverConn, cfg)

		serverConn.Close()
		conn.Close()
	}
}

// BenchmarkLatency_Baseline measures baseline latency without optimizations
func BenchmarkLatency_Baseline(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Server: echo back
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					c.Write(buf[:n])
				}
			}(conn)
		}
	}()

	// Client: measure round-trip time
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	msg := []byte("ping")
	buf := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Send
		if _, err := conn.Write(msg); err != nil {
			b.Fatal(err)
		}

		// Receive
		if _, err := conn.Read(buf); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkLatency_Optimized measures latency with socket optimizations
func BenchmarkLatency_Optimized(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Apply listener optimizations
	ApplyListener(listener, LowLatencyConfig())

	// Server: echo back with optimizations
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			Apply(conn, LowLatencyConfig())

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 1024)
				for {
					n, err := c.Read(buf)
					if err != nil {
						return
					}
					c.Write(buf[:n])
				}
			}(conn)
		}
	}()

	// Client
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	Apply(conn, LowLatencyConfig())

	msg := []byte("ping")
	buf := make([]byte, 1024)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn.Write(msg)
		conn.Read(buf)
	}
}

// BenchmarkThroughput_Baseline measures throughput without optimizations
func BenchmarkThroughput_Baseline(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Server: discard all data
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go io.Copy(io.Discard, conn)
		}
	}()

	// Client: send data
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	data := make([]byte, 64*1024) // 64KB chunks
	for i := range data {
		data[i] = byte(i % 256)
	}

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if _, err := conn.Write(data); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkThroughput_Optimized measures throughput with optimizations
func BenchmarkThroughput_Optimized(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ApplyListener(listener, HighThroughputConfig())

	// Server: discard with optimizations
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			Apply(conn, HighThroughputConfig())
			go io.Copy(io.Discard, conn)
		}
	}()

	// Client
	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	Apply(conn, HighThroughputConfig())

	data := make([]byte, 64*1024) // 64KB chunks

	b.SetBytes(int64(len(data)))
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn.Write(data)
	}
}

// BenchmarkSendFile_Small benchmarks sendfile with small files
func BenchmarkSendFile_Small(b *testing.B) {
	// Create 10KB test file
	tmpfile, err := os.CreateTemp("", "bench-small-*.bin")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	data := make([]byte, 10*1024)
	tmpfile.Write(data)
	tmpfile.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Receiver
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			io.Copy(io.Discard, conn)
			conn.Close()
		}
	}()

	b.SetBytes(10 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		file, _ := os.Open(tmpfile.Name())

		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			b.Fatal(err)
		}

		SendFileAll(conn, file)

		conn.Close()
		file.Close()
	}
}

// BenchmarkSendFile_Large benchmarks sendfile with large files
func BenchmarkSendFile_Large(b *testing.B) {
	// Create 10MB test file
	tmpfile, err := os.CreateTemp("", "bench-large-*.bin")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	data := make([]byte, 10*1024*1024)
	tmpfile.Write(data)
	tmpfile.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Receiver
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			io.Copy(io.Discard, conn)
			conn.Close()
		}
	}()

	b.SetBytes(10 * 1024 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		file, _ := os.Open(tmpfile.Name())

		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			b.Fatal(err)
		}

		SendFileAll(conn, file)

		conn.Close()
		file.Close()
	}
}

// BenchmarkIOCopy_Large benchmarks io.Copy for comparison
func BenchmarkIOCopy_Large(b *testing.B) {
	// Create 10MB test file
	tmpfile, err := os.CreateTemp("", "bench-iocopy-*.bin")
	if err != nil {
		b.Fatalf("Failed to create temp file: %v", err)
	}
	defer os.Remove(tmpfile.Name())

	data := make([]byte, 10*1024*1024)
	tmpfile.Write(data)
	tmpfile.Close()

	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Receiver
	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			io.Copy(io.Discard, conn)
			conn.Close()
		}
	}()

	b.SetBytes(10 * 1024 * 1024)
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		file, _ := os.Open(tmpfile.Name())

		conn, err := net.Dial("tcp", listener.Addr().String())
		if err != nil {
			b.Fatal(err)
		}

		io.Copy(conn, file)

		conn.Close()
		file.Close()
	}
}

// BenchmarkConnectionEstablishment_Baseline measures connection setup time
func BenchmarkConnectionEstablishment_Baseline(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			conn.Close()
		}
	}()

	addr := listener.Addr().String()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		conn.Close()

		// Small delay to allow proper cleanup
		time.Sleep(time.Microsecond)
	}
}

// BenchmarkConnectionEstablishment_Optimized measures connection setup with TFO
func BenchmarkConnectionEstablishment_Optimized(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Apply TFO and other optimizations
	ApplyListener(listener, DefaultConfig())

	go func() {
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			Apply(conn, DefaultConfig())
			conn.Close()
		}
	}()

	addr := listener.Addr().String()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		Apply(conn, DefaultConfig())
		conn.Close()

		time.Sleep(time.Microsecond)
	}
}

// BenchmarkSmallRequests_Baseline benchmarks many small HTTP-like requests
func BenchmarkSmallRequests_Baseline(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	// Simple HTTP-like server
	go func() {
		response := []byte("HTTP/1.1 200 OK\r\nContent-Length: 13\r\n\r\nHello, World!")
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					_, err := c.Read(buf)
					if err != nil {
						return
					}
					c.Write(response)
				}
			}(conn)
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	request := []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	response := make([]byte, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn.Write(request)
		conn.Read(response)
	}
}

// BenchmarkSmallRequests_Optimized benchmarks with all optimizations
func BenchmarkSmallRequests_Optimized(b *testing.B) {
	listener, err := net.Listen("tcp", "127.0.0.1:0")
	if err != nil {
		b.Fatalf("Failed to create listener: %v", err)
	}
	defer listener.Close()

	ApplyListener(listener, LowLatencyConfig())

	// Optimized server
	go func() {
		response := []byte("HTTP/1.1 200 OK\r\nContent-Length: 13\r\n\r\nHello, World!")
		for {
			conn, err := listener.Accept()
			if err != nil {
				return
			}
			Apply(conn, LowLatencyConfig())

			go func(c net.Conn) {
				defer c.Close()
				buf := make([]byte, 4096)
				for {
					_, err := c.Read(buf)
					if err != nil {
						return
					}
					c.Write(response)
				}
			}(conn)
		}
	}()

	conn, err := net.Dial("tcp", listener.Addr().String())
	if err != nil {
		b.Fatalf("Failed to dial: %v", err)
	}
	defer conn.Close()

	Apply(conn, LowLatencyConfig())

	request := []byte("GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
	response := make([]byte, 4096)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn.Write(request)
		conn.Read(response)
	}
}
