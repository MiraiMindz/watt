package main

import (
	"context"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/http/httptest"
	"testing"
	"time"

	"github.com/valyala/fasthttp"
	"github.com/yourusername/shockwave/pkg/shockwave/client"
	"github.com/yourusername/shockwave/pkg/shockwave/http11"
	"github.com/yourusername/shockwave/pkg/shockwave/server"
)

// ==============================================================================
// CLIENT BENCHMARKS
// ==============================================================================

// BenchmarkClients_SimpleGET compares all clients for simple GET requests
func BenchmarkClients_SimpleGET(b *testing.B) {
	// Setup test server
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer srv.Close()

	b.Run("Shockwave", func(b *testing.B) {
		c := client.NewClient()
		c.Warmup(10)
		defer c.Close()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			resp, err := c.Get(srv.URL)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body())
			resp.Close()
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		c := &fasthttp.Client{}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := fasthttp.AcquireRequest()
			resp := fasthttp.AcquireResponse()

			req.SetRequestURI(srv.URL)
			err := c.Do(req, resp)
			if err != nil {
				b.Fatal(err)
			}

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}
	})

	b.Run("net/http", func(b *testing.B) {
		c := &http.Client{}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			resp, err := c.Get(srv.URL)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// BenchmarkClients_Concurrent compares clients under concurrent load
func BenchmarkClients_Concurrent(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Write([]byte("OK"))
	}))
	defer srv.Close()

	b.Run("Shockwave", func(b *testing.B) {
		c := client.NewClient()
		c.Warmup(10)
		defer c.Close()

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := c.Get(srv.URL)
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body())
				resp.Close()
			}
		})
	})

	b.Run("fasthttp", func(b *testing.B) {
		c := &fasthttp.Client{}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				req := fasthttp.AcquireRequest()
				resp := fasthttp.AcquireResponse()

				req.SetRequestURI(srv.URL)
				err := c.Do(req, resp)
				if err != nil {
					b.Fatal(err)
				}

				fasthttp.ReleaseRequest(req)
				fasthttp.ReleaseResponse(resp)
			}
		})
	})

	b.Run("net/http", func(b *testing.B) {
		c := &http.Client{}

		b.ReportAllocs()
		b.ResetTimer()

		b.RunParallel(func(pb *testing.PB) {
			for pb.Next() {
				resp, err := c.Get(srv.URL)
				if err != nil {
					b.Fatal(err)
				}
				io.Copy(io.Discard, resp.Body)
				resp.Body.Close()
			}
		})
	})
}

// BenchmarkClients_WithHeaders compares clients with multiple headers
func BenchmarkClients_WithHeaders(b *testing.B) {
	srv := httptest.NewServer(http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("X-Custom-Header", "value")
		w.Header().Set("X-Request-ID", "12345")
		w.Header().Set("X-Rate-Limit", "100")
		w.Write([]byte("OK"))
	}))
	defer srv.Close()

	b.Run("Shockwave", func(b *testing.B) {
		c := client.NewClient()
		c.Warmup(10)
		defer c.Close()

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			resp, err := c.Get(srv.URL)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body())
			resp.Close()
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		c := &fasthttp.Client{}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			req := fasthttp.AcquireRequest()
			resp := fasthttp.AcquireResponse()

			req.SetRequestURI(srv.URL)
			err := c.Do(req, resp)
			if err != nil {
				b.Fatal(err)
			}

			fasthttp.ReleaseRequest(req)
			fasthttp.ReleaseResponse(resp)
		}
	})

	b.Run("net/http", func(b *testing.B) {
		c := &http.Client{}

		b.ReportAllocs()
		b.ResetTimer()

		for i := 0; i < b.N; i++ {
			resp, err := c.Get(srv.URL)
			if err != nil {
				b.Fatal(err)
			}
			io.Copy(io.Discard, resp.Body)
			resp.Body.Close()
		}
	})
}

// ==============================================================================
// SERVER BENCHMARKS
// ==============================================================================

// BenchmarkServers_SimpleGET compares all servers for simple GET requests
func BenchmarkServers_SimpleGET(b *testing.B) {
	b.Run("Shockwave", func(b *testing.B) {
		config := server.DefaultConfig()
		config.Addr = "127.0.0.1:0"
		config.LegacyHandler = server.LegacyHandlerFunc(func(w server.ResponseWriter, r server.Request) {
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		})

		srv := server.NewServer(config)
		ln, err := net.Listen("tcp", config.Addr)
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go srv.Serve(ln)
		defer srv.Close()

		time.Sleep(50 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ReportAllocs()
		b.ResetTimer()

		buf := make([]byte, 1024)
		for i := 0; i < b.N; i++ {
			fmt.Fprintf(conn, "GET / HTTP/1.1\r\nHost: localhost\r\n\r\n")
			_, err = conn.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		handler := func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(200)
			ctx.SetBody([]byte("OK"))
		}

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go fasthttp.Serve(ln, handler)

		time.Sleep(50 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ReportAllocs()
		b.ResetTimer()

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

		srv := &http.Server{Handler: mux}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go srv.Serve(ln)
		defer srv.Shutdown(context.Background())

		time.Sleep(50 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ReportAllocs()
		b.ResetTimer()

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

// BenchmarkServers_Concurrent compares servers under concurrent load
func BenchmarkServers_Concurrent(b *testing.B) {
	b.Run("Shockwave", func(b *testing.B) {
		config := server.DefaultConfig()
		config.Addr = "127.0.0.1:0"
		config.Handler = func(w *http11.ResponseWriter, r *http11.Request) {
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		}

		srv := server.NewServer(config)
		ln, err := net.Listen("tcp", config.Addr)
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go srv.Serve(ln)
		defer srv.Close()

		time.Sleep(50 * time.Millisecond)

		b.ReportAllocs()
		b.ResetTimer()

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
	})

	b.Run("fasthttp", func(b *testing.B) {
		handler := func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(200)
			ctx.SetBody([]byte("OK"))
		}

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go fasthttp.Serve(ln, handler)

		time.Sleep(50 * time.Millisecond)

		b.ReportAllocs()
		b.ResetTimer()

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
	})

	b.Run("net/http", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(200)
			w.Write([]byte("OK"))
		})

		srv := &http.Server{Handler: mux}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go srv.Serve(ln)
		defer srv.Shutdown(context.Background())

		time.Sleep(50 * time.Millisecond)

		b.ReportAllocs()
		b.ResetTimer()

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
	})
}

// BenchmarkServers_JSON compares servers for JSON responses
func BenchmarkServers_JSON(b *testing.B) {
	jsonData := []byte(`{"status":"ok","message":"success","data":{"id":123,"name":"test"}}`)

	b.Run("Shockwave", func(b *testing.B) {
		config := server.DefaultConfig()
		config.Addr = "127.0.0.1:0"
		config.LegacyHandler = server.LegacyHandlerFunc(func(w server.ResponseWriter, r server.Request) {
			w.WriteJSON(200, jsonData)
		})

		srv := server.NewServer(config)
		ln, err := net.Listen("tcp", config.Addr)
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go srv.Serve(ln)
		defer srv.Close()

		time.Sleep(50 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ReportAllocs()
		b.ResetTimer()

		buf := make([]byte, 1024)
		for i := 0; i < b.N; i++ {
			fmt.Fprintf(conn, "GET /api HTTP/1.1\r\nHost: localhost\r\n\r\n")
			_, err = conn.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		handler := func(ctx *fasthttp.RequestCtx) {
			ctx.SetContentType("application/json")
			ctx.SetStatusCode(200)
			ctx.SetBody(jsonData)
		}

		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go fasthttp.Serve(ln, handler)

		time.Sleep(50 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ReportAllocs()
		b.ResetTimer()

		buf := make([]byte, 1024)
		for i := 0; i < b.N; i++ {
			fmt.Fprintf(conn, "GET /api HTTP/1.1\r\nHost: localhost\r\n\r\n")
			_, err = conn.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})

	b.Run("net/http", func(b *testing.B) {
		mux := http.NewServeMux()
		mux.HandleFunc("/api", func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Content-Type", "application/json")
			w.WriteHeader(200)
			w.Write(jsonData)
		})

		srv := &http.Server{Handler: mux}
		ln, err := net.Listen("tcp", "127.0.0.1:0")
		if err != nil {
			b.Fatal(err)
		}
		defer ln.Close()

		addr := ln.Addr().String()
		go srv.Serve(ln)
		defer srv.Shutdown(context.Background())

		time.Sleep(50 * time.Millisecond)

		conn, err := net.Dial("tcp", addr)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()

		b.ReportAllocs()
		b.ResetTimer()

		buf := make([]byte, 1024)
		for i := 0; i < b.N; i++ {
			fmt.Fprintf(conn, "GET /api HTTP/1.1\r\nHost: localhost\r\n\r\n")
			_, err = conn.Read(buf)
			if err != nil {
				b.Fatal(err)
			}
		}
	})
}
