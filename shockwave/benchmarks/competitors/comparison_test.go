package competitors

import (
	"bufio"
	"bytes"
	"fmt"
	"net"
	"net/http"
	"net/http/httptest"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"github.com/valyala/fasthttp"
	"github.com/valyala/fasthttp/fasthttputil"
)

// Direct comparison benchmarks for easy analysis

// BenchmarkComparisonSimpleGET compares simple GET request performance
func BenchmarkComparisonSimpleGET(b *testing.B) {
	b.Run("net/http", func(b *testing.B) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		client := &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 100,
				DisableCompression:  true,
			},
		}

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		handler := func(ctx *fasthttp.RequestCtx) {
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.WriteString("OK")
		}

		server := &fasthttp.Server{Handler: handler}
		ln := fasthttputil.NewInmemoryListener()
		defer ln.Close()
		go server.Serve(ln)

		client := &fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) {
				return ln.Dial()
			},
		}

		var req fasthttp.Request
		var resp fasthttp.Response
		req.SetRequestURI("http://localhost/")

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			client.Do(&req, &resp)
			resp.Reset()
		}
	})
}

// BenchmarkComparisonRequestParsing compares HTTP request parsing
func BenchmarkComparisonRequestParsing(b *testing.B) {
	reqStr := "GET /path HTTP/1.1\r\n" +
		"Host: example.com\r\n" +
		"User-Agent: benchmark\r\n" +
		"Accept: */*\r\n" +
		"Connection: keep-alive\r\n" +
		"Content-Length: 0\r\n" +
		"\r\n"
	reqBytes := []byte(reqStr)

	b.Run("net/http", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(reqStr)))

		for i := 0; i < b.N; i++ {
			req, _ := http.ReadRequest(bufio.NewReader(strings.NewReader(reqStr)))
			_ = req
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(reqBytes)))

		var req fasthttp.Request
		for i := 0; i < b.N; i++ {
			req.Reset()
			br := bufio.NewReader(bytes.NewReader(reqBytes))
			req.Read(br)
		}
	})
}

// BenchmarkComparisonResponseWriting compares HTTP response writing
func BenchmarkComparisonResponseWriting(b *testing.B) {
	b.Run("net/http", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			rec := httptest.NewRecorder()
			rec.WriteHeader(http.StatusOK)
			rec.Write([]byte("Hello, World!"))
			_ = rec.Result()
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		b.ReportAllocs()

		var resp fasthttp.Response
		var buf bytes.Buffer

		for i := 0; i < b.N; i++ {
			resp.Reset()
			buf.Reset()
			resp.SetStatusCode(fasthttp.StatusOK)
			resp.SetBody([]byte("Hello, World!"))
			resp.WriteTo(&buf)
		}
	})
}

// BenchmarkComparisonHeaderProcessing compares header-heavy request handling
func BenchmarkComparisonHeaderProcessing(b *testing.B) {
	// Build request with many headers
	var reqBuilder strings.Builder
	reqBuilder.WriteString("GET /path HTTP/1.1\r\n")
	reqBuilder.WriteString("Host: example.com\r\n")
	for i := 0; i < 30; i++ {
		reqBuilder.WriteString(fmt.Sprintf("X-Custom-Header-%d: value-%d\r\n", i, i))
	}
	reqBuilder.WriteString("\r\n")
	reqStr := reqBuilder.String()
	reqBytes := []byte(reqStr)

	b.Run("net/http", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(reqStr)))

		for i := 0; i < b.N; i++ {
			req, _ := http.ReadRequest(bufio.NewReader(strings.NewReader(reqStr)))
			_ = req.Header.Get("X-Custom-Header-15")
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		b.ReportAllocs()
		b.SetBytes(int64(len(reqBytes)))

		var req fasthttp.Request
		for i := 0; i < b.N; i++ {
			req.Reset()
			br := bufio.NewReader(bytes.NewReader(reqBytes))
			req.Read(br)
			_ = req.Header.Peek("X-Custom-Header-15")
		}
	})
}

// BenchmarkComparisonWebSocketEcho compares WebSocket echo performance
func BenchmarkComparisonWebSocketEcho(b *testing.B) {
	message := []byte("Hello, WebSocket!")

	b.Run("gorilla/websocket", func(b *testing.B) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			conn, err := upgrader.Upgrade(w, r, nil)
			if err != nil {
				return
			}
			defer conn.Close()

			for {
				messageType, msg, err := conn.ReadMessage()
				if err != nil {
					return
				}
				if err := conn.WriteMessage(messageType, msg); err != nil {
					return
				}
			}
		})

		server := httptest.NewServer(handler)
		defer server.Close()

		wsURL := "ws" + server.URL[4:]
		conn, _, _ := websocket.DefaultDialer.Dial(wsURL, nil)
		defer conn.Close()

		b.ResetTimer()
		b.ReportAllocs()
		b.SetBytes(int64(len(message) * 2))

		for i := 0; i < b.N; i++ {
			conn.WriteMessage(websocket.TextMessage, message)
			_, _, _ = conn.ReadMessage()
		}
	})

	// Note: fasthttp doesn't have built-in WebSocket support
	// This is a key differentiator for Shockwave which will have integrated WS
}

// BenchmarkComparisonKeepAlive compares keep-alive connection handling
func BenchmarkComparisonKeepAlive(b *testing.B) {
	b.Run("net/http", func(b *testing.B) {
		handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			w.Header().Set("Connection", "keep-alive")
			w.WriteHeader(http.StatusOK)
			w.Write([]byte("OK"))
		})
		server := httptest.NewServer(handler)
		defer server.Close()

		client := &http.Client{
			Transport: &http.Transport{
				MaxIdleConnsPerHost: 100,
				DisableCompression:  true,
				DisableKeepAlives:   false,
			},
		}

		// Warm up
		resp, err := client.Get(server.URL)
		if err != nil {
			b.Fatal(err)
		}
		resp.Body.Close()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			resp, err := client.Get(server.URL)
			if err != nil {
				b.Fatal(err)
			}
			resp.Body.Close()
		}
	})

	b.Run("fasthttp", func(b *testing.B) {
		handler := func(ctx *fasthttp.RequestCtx) {
			ctx.Response.Header.Set("Connection", "keep-alive")
			ctx.SetStatusCode(fasthttp.StatusOK)
			ctx.WriteString("OK")
		}

		server := &fasthttp.Server{Handler: handler}
		ln := fasthttputil.NewInmemoryListener()
		defer ln.Close()

		go server.Serve(ln)

		client := &fasthttp.Client{
			Dial: func(addr string) (net.Conn, error) {
				return ln.Dial()
			},
			MaxConnsPerHost:     100,
			MaxIdleConnDuration: 90 * time.Second,
		}

		var req fasthttp.Request
		var resp fasthttp.Response
		req.SetRequestURI("http://localhost/")

		// Warm up
		client.Do(&req, &resp)
		resp.Reset()

		b.ResetTimer()
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			if err := client.Do(&req, &resp); err != nil {
				b.Fatal(err)
			}
			resp.Reset()
		}
	})
}

// BenchmarkComparisonMemoryUsage provides a high-level memory comparison
func BenchmarkComparisonMemoryUsage(b *testing.B) {
	b.Run("net/http-server-alloc", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			server := &http.Server{
				Addr: ":0",
				Handler: http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
					w.Write([]byte("OK"))
				}),
			}
			_ = server
		}
	})

	b.Run("fasthttp-server-alloc", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			server := &fasthttp.Server{
				Handler: func(ctx *fasthttp.RequestCtx) {
					ctx.WriteString("OK")
				},
			}
			_ = server
		}
	})

	b.Run("net/http-request-alloc", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			req, _ := http.NewRequest("GET", "http://example.com", nil)
			req.Header.Set("User-Agent", "benchmark")
			_ = req
		}
	})

	b.Run("fasthttp-request-alloc", func(b *testing.B) {
		b.ReportAllocs()

		for i := 0; i < b.N; i++ {
			var req fasthttp.Request
			req.SetRequestURI("http://example.com")
			req.Header.Set("User-Agent", "benchmark")
			_ = req
		}
	})
}

// BenchmarkComparisonScalability tests performance under load
func BenchmarkComparisonScalability(b *testing.B) {
	concurrencies := []int{1, 10, 50, 100}

	for _, concurrency := range concurrencies {
		b.Run(fmt.Sprintf("net/http-c%d", concurrency), func(b *testing.B) {
			handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
				w.Write([]byte("OK"))
			})
			server := httptest.NewServer(handler)
			defer server.Close()

			b.SetParallelism(concurrency)
			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				client := &http.Client{
					Transport: &http.Transport{
						MaxIdleConnsPerHost: 10,
					},
				}
				for pb.Next() {
					resp, err := client.Get(server.URL)
					if err != nil {
						b.Fatal(err)
					}
					resp.Body.Close()
				}
			})
		})

		b.Run(fmt.Sprintf("fasthttp-c%d", concurrency), func(b *testing.B) {
			handler := func(ctx *fasthttp.RequestCtx) {
				ctx.WriteString("OK")
			}
			server := &fasthttp.Server{
				Handler:     handler,
				Concurrency: concurrency * 100, // High limit for benchmark load
			}
			ln := fasthttputil.NewInmemoryListener()
			defer ln.Close()
			go server.Serve(ln)

			b.SetParallelism(concurrency)
			b.ResetTimer()
			b.ReportAllocs()

			b.RunParallel(func(pb *testing.PB) {
				client := &fasthttp.Client{
					Dial: func(addr string) (net.Conn, error) {
						return ln.Dial()
					},
					MaxConnsPerHost: 10,
				}
				var req fasthttp.Request
				var resp fasthttp.Response
				req.SetRequestURI("http://localhost/")

				for pb.Next() {
					client.Do(&req, &resp)
					resp.Reset()
				}
			})
		})
	}
}