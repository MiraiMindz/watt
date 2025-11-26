package server

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"sync"
	"testing"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave/http11"
)

// BenchmarkServer_SimpleGET benchmarks a simple GET request
func BenchmarkServer_SimpleGET(b *testing.B) {
	// Create server
	config := DefaultConfig()
	config.Addr = "127.0.0.1:0" // Random port
	config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}

	srv := NewServer(config)

	// Start server
	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go srv.Serve(ln)
	defer srv.Close()

	// Wait for server to start
	time.Sleep(100 * time.Millisecond)

	// Benchmark
	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}

		// Send request
		fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")

		// Read response
		buf := make([]byte, 1024)
		_, err = conn.Read(buf)
		if err != nil {
			b.Fatal(err)
		}

		conn.Close()
	}
}

// BenchmarkServer_KeepAlive benchmarks keep-alive connections
func BenchmarkServer_KeepAlive(b *testing.B) {
	// Create server
	config := DefaultConfig()
	config.Addr = "127.0.0.1:0"
	config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}

	srv := NewServer(config)

	// Start server
	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go srv.Serve(ln)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	// Create persistent connection
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	buf := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		// Send request
		fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")

		// Read response
		_, err = conn.Read(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkServer_JSON benchmarks JSON response
func BenchmarkServer_JSON(b *testing.B) {
	jsonData := []byte(`{"status":"ok","message":"success"}`)

	// Create server
	config := DefaultConfig()
	config.Addr = "127.0.0.1:0"
	config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
		w.WriteJSON(200, jsonData)
	}

	srv := NewServer(config)

	// Start server
	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go srv.Serve(ln)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	// Create persistent connection
	conn, err := net.Dial("tcp", addr)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	buf := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		fmt.Fprintf(conn, "GET /api HTTP/1.1\r\nHost: localhost\r\n\r\n")
		_, err = conn.Read(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkServer_LargeResponse benchmarks large response handling
func BenchmarkServer_LargeResponse(b *testing.B) {
	// 100KB response
	largeData := make([]byte, 100*1024)
	for i := range largeData {
		largeData[i] = byte(i % 256)
	}

	config := DefaultConfig()
	config.Addr = "127.0.0.1:0"
	config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
		w.WriteHeader(200)
		w.Write(largeData)
	}

	srv := NewServer(config)

	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go srv.Serve(ln)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(largeData)))

	for i := 0; i < b.N; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}

		fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")

		// Read all response
		io.ReadAll(conn)

		conn.Close()
	}
}

// BenchmarkServer_Concurrent benchmarks concurrent connections
func BenchmarkServer_Concurrent(b *testing.B) {
	config := DefaultConfig()
	config.Addr = "127.0.0.1:0"
	config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}

	srv := NewServer(config)

	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go srv.Serve(ln)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	b.ResetTimer()
	b.ReportAllocs()

	// Run with parallelism
	b.RunParallel(func(pb *testing.PB) {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		buf := make([]byte, 1024)
		for pb.Next() {
			fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
			_, err = conn.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkServer_vs_NetHTTP compares against net/http
func BenchmarkServer_vs_NetHTTP(b *testing.B) {
	b.Run("Shockwave", func(b *testing.B) {
		config := DefaultConfig()
		config.Addr = "127.0.0.1:0"
		config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		}

		srv := NewServer(config)

		ln, err := net.Listen("tcp", config.Addr)
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()

		go srv.Serve(ln)
		defer srv.Close()

		time.Sleep(100 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ResetTimer()
		b.ReportAllocs()

		buf := make([]byte, 1024)
		for i := 0; i < b.N; i++ {
			fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
			_, err = conn.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("net/http", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		})

		srv := &http.Server{
			Handler: mux,
		}

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()

		go srv.Serve(ln)
		defer srv.Shutdown(context.Background())

		time.Sleep(100 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ResetTimer()
		b.ReportAllocs()

		buf := make([]byte, 1024)
		for i := 0; i < b.N; i++ {
			fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
			_, err = conn.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}

// BenchmarkServer_Throughput measures requests per second
func BenchmarkServer_Throughput(b *testing.B) {
	config := DefaultConfig()
	config.Addr = "127.0.0.1:0"
	config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}

	srv := NewServer(config)

	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go srv.Serve(ln)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	// Create connection pool
	numConns := 10
	conns := make([]net.Conn, numConns)
	for i := 0; i < numConns; i++ {
		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		conns[i] = conn
		defer conn.Close()
	}

	b.ResetTimer()
	b.ReportAllocs()

	// Measure throughput
	start := time.Now()

	var wg sync.WaitGroup
	requestsPerConn := b.N / numConns

	for _, conn := range conns {
		wg.Add(1)
		go func(c net.Conn) {
			defer wg.Done()

			buf := make([]byte, 1024)
			for i := 0; i < requestsPerConn; i++ {
				fmt.Fprintf(c, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
				_, err := c.Read(buf)
				if err != nil {
					return
				}
			}
		}(conn)
	}

	wg.Wait()

	duration := time.Since(start)
	rps := float64(b.N) / duration.Seconds()

	b.ReportMetric(rps, "req/s")
}

// BenchmarkServer_Stats benchmarks server statistics tracking
func BenchmarkServer_Stats(b *testing.B) {
	config := DefaultConfig()
	config.Addr = "127.0.0.1:0"
	config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
		w.WriteHeader(200)
		w.Write([]byte("OK"))
	}

	srv := NewServer(config)

	ln, err := net.Listen("tcp", config.Addr)
	if err != nil {
		b.Fatal(err)
	}
	defer ln.Close()

	addr := ln.Addr().String()

	go srv.Serve(ln)
	defer srv.Close()

	time.Sleep(100 * time.Millisecond)

	conn, err := net.Dial("tcp", addr)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	b.ResetTimer()
	b.ReportAllocs()

	buf := make([]byte, 1024)
	for i := 0; i < b.N; i++ {
		fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
		_, err = conn.Read(buf)
		if err != nil {
			b.Fatal(err)
		}

		// Check stats periodically
		if i%1000 == 0 {
			stats := srv.Stats()
			_ = stats.RequestsPerSecond()
		}
	}
}
