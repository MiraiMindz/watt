package competitors

import (
	"bytes"
	"net/http"
	"net/http/httptest"
	"sync"
	"testing"
	"time"

	"github.com/gorilla/websocket"
)

// WebSocket upgrader
var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
}

// BenchmarkGorillaWebSocketEcho benchmarks echo server (single message round-trip)
func BenchmarkGorillaWebSocketEcho(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + server.URL[4:] // Convert http:// to ws://
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	message := []byte("Hello, WebSocket!")
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(message) * 2)) // Count both send and receive

	for i := 0; i < b.N; i++ {
		if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
			b.Fatal(err)
		}

		_, receivedMsg, err := conn.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
		_ = receivedMsg
	}
}

// BenchmarkGorillaWebSocketBroadcast benchmarks message broadcasting to multiple clients
func BenchmarkGorillaWebSocketBroadcast(b *testing.B) {
	// Hub for managing connections
	type Hub struct {
		clients    map[*websocket.Conn]bool
		broadcast  chan []byte
		register   chan *websocket.Conn
		unregister chan *websocket.Conn
		mu         sync.RWMutex
	}

	hub := &Hub{
		clients:    make(map[*websocket.Conn]bool),
		broadcast:  make(chan []byte, 256),
		register:   make(chan *websocket.Conn),
		unregister: make(chan *websocket.Conn),
	}

	// Run hub
	go func() {
		for {
			select {
			case client := <-hub.register:
				hub.mu.Lock()
				hub.clients[client] = true
				hub.mu.Unlock()

			case client := <-hub.unregister:
				hub.mu.Lock()
				if _, ok := hub.clients[client]; ok {
					delete(hub.clients, client)
					client.Close()
				}
				hub.mu.Unlock()

			case message := <-hub.broadcast:
				hub.mu.RLock()
				for client := range hub.clients {
					err := client.WriteMessage(websocket.TextMessage, message)
					if err != nil {
						client.Close()
					}
				}
				hub.mu.RUnlock()
			}
		}
	}()

	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		hub.register <- conn

		// Read pump
		go func() {
			defer func() {
				hub.unregister <- conn
			}()
			for {
				_, message, err := conn.ReadMessage()
				if err != nil {
					return
				}
				hub.broadcast <- message
			}
		}()
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]

	// Create 10 clients
	clientCount := 10
	clients := make([]*websocket.Conn, clientCount)
	for i := 0; i < clientCount; i++ {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Fatal(err)
		}
		defer conn.Close()
		clients[i] = conn
	}

	// Wait for connections to register
	time.Sleep(100 * time.Millisecond)

	message := []byte("Broadcast message!")
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(message) * clientCount)) // Message sent to all clients

	for i := 0; i < b.N; i++ {
		// Send message from first client
		if err := clients[0].WriteMessage(websocket.TextMessage, message); err != nil {
			b.Fatal(err)
		}

		// All clients should receive it
		for j := 0; j < clientCount; j++ {
			_, _, err := clients[j].ReadMessage()
			if err != nil {
				b.Fatal(err)
			}
		}
	}
}

// BenchmarkGorillaWebSocketThroughput benchmarks sustained throughput
func BenchmarkGorillaWebSocketThroughput(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Just receive messages, don't echo
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	message := generateBody(1024) // 1KB message
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(message)))

	for i := 0; i < b.N; i++ {
		if err := conn.WriteMessage(websocket.BinaryMessage, message); err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkGorillaWebSocketConcurrent benchmarks concurrent connections
func BenchmarkGorillaWebSocketConcurrent(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]

	message := []byte("Concurrent test message")

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(message) * 2))

	b.SetParallelism(100)
	b.RunParallel(func(pb *testing.PB) {
		conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
		if err != nil {
			b.Error(err)
			return
		}
		defer conn.Close()

		for pb.Next() {
			if err := conn.WriteMessage(websocket.TextMessage, message); err != nil {
				b.Error(err)
				return
			}

			_, _, err := conn.ReadMessage()
			if err != nil {
				b.Error(err)
				return
			}
		}
	})
}

// BenchmarkGorillaWebSocketLargeMessage benchmarks sending large messages (1MB)
func BenchmarkGorillaWebSocketLargeMessage(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Set larger message size limit
		conn.SetReadLimit(2 * 1024 * 1024) // 2MB

		for {
			messageType, message, err := conn.ReadMessage()
			if err != nil {
				return
			}
			if err := conn.WriteMessage(messageType, message); err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	largeMessage := generateBody(1024 * 1024) // 1MB
	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(largeMessage) * 2)) // Count both send and receive

	for i := 0; i < b.N; i++ {
		if err := conn.WriteMessage(websocket.BinaryMessage, largeMessage); err != nil {
			b.Fatal(err)
		}

		_, receivedMsg, err := conn.ReadMessage()
		if err != nil {
			b.Fatal(err)
		}
		_ = receivedMsg
	}
}

// BenchmarkGorillaWebSocketPing benchmarks ping/pong performance
func BenchmarkGorillaWebSocketPing(b *testing.B) {
	handler := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		conn, err := upgrader.Upgrade(w, r, nil)
		if err != nil {
			return
		}
		defer conn.Close()

		// Handle pong
		conn.SetPongHandler(func(string) error {
			return nil
		})

		// Read messages to keep connection alive
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	})

	server := httptest.NewServer(handler)
	defer server.Close()

	wsURL := "ws" + server.URL[4:]
	conn, _, err := websocket.DefaultDialer.Dial(wsURL, nil)
	if err != nil {
		b.Fatal(err)
	}
	defer conn.Close()

	// Set up pong handler
	pongReceived := make(chan struct{}, 1)
	conn.SetPongHandler(func(string) error {
		select {
		case pongReceived <- struct{}{}:
		default:
		}
		return nil
	})

	// Start read pump
	go func() {
		for {
			_, _, err := conn.ReadMessage()
			if err != nil {
				return
			}
		}
	}()

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		if err := conn.WriteControl(websocket.PingMessage, []byte("ping"), time.Now().Add(time.Second)); err != nil {
			b.Fatal(err)
		}

		select {
		case <-pongReceived:
			// Pong received
		case <-time.After(time.Second):
			b.Fatal("Pong timeout")
		}
	}
}

// BenchmarkGorillaWebSocketMessageParsing benchmarks WebSocket frame parsing
func BenchmarkGorillaWebSocketMessageParsing(b *testing.B) {
	// Create a simple WebSocket frame
	frame := []byte{
		0x81, 0x85, // FIN=1, opcode=1 (text), MASK=1, len=5
		0x37, 0xfa, 0x21, 0x3d, // Masking key
		0x7f, 0x9f, 0x4d, 0x51, 0x58, // Masked "Hello"
	}

	b.ResetTimer()
	b.ReportAllocs()
	b.SetBytes(int64(len(frame)))

	for i := 0; i < b.N; i++ {
		reader := bytes.NewReader(frame)
		// Simulate frame parsing (simplified)
		header := make([]byte, 2)
		reader.Read(header)

		masked := (header[1] & 0x80) != 0
		length := header[1] & 0x7f

		if masked {
			mask := make([]byte, 4)
			reader.Read(mask)
		}

		payload := make([]byte, length)
		reader.Read(payload)

		_ = payload
	}
}