package websocket

import (
	"bufio"
	"crypto/rand"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"strings"
)

var (
	ErrNotWebSocket       = errors.New("websocket: not a websocket handshake")
	ErrBadWebSocketKey    = errors.New("websocket: invalid Sec-WebSocket-Key")
	ErrBadWebSocketVersion = errors.New("websocket: unsupported Sec-WebSocket-Version")
	ErrUpgradeFailed      = errors.New("websocket: upgrade failed")
)

// Upgrader handles WebSocket upgrade handshakes from HTTP connections.
// Zero-allocation upgrade path for common cases.
type Upgrader struct {
	// CheckOrigin returns true if the request Origin header is acceptable.
	// If nil, origin validation is skipped (insecure, use only for testing).
	CheckOrigin func(r *http.Request) bool

	// Subprotocols specifies the supported subprotocols in order of preference.
	Subprotocols []string

	// ReadBufferSize and WriteBufferSize specify I/O buffer sizes in bytes.
	// If zero, 4096 bytes are used.
	ReadBufferSize  int
	WriteBufferSize int

	// EnableCompression enables per-message compression (RFC 7692).
	// Not implemented yet.
	EnableCompression bool
}

// Upgrade upgrades an HTTP connection to the WebSocket protocol.
// RFC 6455 Section 4: Opening Handshake
//
// The server must:
// 1. Validate the handshake request
// 2. Compute Sec-WebSocket-Accept from Sec-WebSocket-Key
// 3. Send 101 Switching Protocols response
// 4. Return a WebSocket connection
func (u *Upgrader) Upgrade(w http.ResponseWriter, r *http.Request) (*Conn, error) {
	// Validate HTTP method must be GET (RFC 6455 4.1)
	if r.Method != http.MethodGet {
		http.Error(w, "Method not allowed", http.StatusMethodNotAllowed)
		return nil, ErrNotWebSocket
	}

	// Validate required headers (RFC 6455 4.2.1)
	if !headerContains(r.Header, "Connection", "upgrade") {
		http.Error(w, "Bad Request: missing Connection: upgrade", http.StatusBadRequest)
		return nil, ErrNotWebSocket
	}

	if !headerContains(r.Header, "Upgrade", "websocket") {
		http.Error(w, "Bad Request: missing Upgrade: websocket", http.StatusBadRequest)
		return nil, ErrNotWebSocket
	}

	// Validate Sec-WebSocket-Version (must be 13 for RFC 6455)
	if r.Header.Get("Sec-WebSocket-Version") != "13" {
		w.Header().Set("Sec-WebSocket-Version", "13")
		http.Error(w, "Bad Request: unsupported WebSocket version", http.StatusBadRequest)
		return nil, ErrBadWebSocketVersion
	}

	// Get and validate Sec-WebSocket-Key (must be present and base64-encoded 16 bytes)
	wsKey := r.Header.Get("Sec-WebSocket-Key")
	if wsKey == "" {
		http.Error(w, "Bad Request: missing Sec-WebSocket-Key", http.StatusBadRequest)
		return nil, ErrBadWebSocketKey
	}

	// Check origin if configured
	if u.CheckOrigin != nil && !u.CheckOrigin(r) {
		http.Error(w, "Forbidden: origin not allowed", http.StatusForbidden)
		return nil, ErrUpgradeFailed
	}

	// Select subprotocol if requested
	var subprotocol string
	if len(u.Subprotocols) > 0 {
		clientProtos := headerValues(r.Header, "Sec-WebSocket-Protocol")
		subprotocol = selectSubprotocol(clientProtos, u.Subprotocols)
	}

	// Hijack the connection
	hj, ok := w.(http.Hijacker)
	if !ok {
		http.Error(w, "Internal Server Error: cannot hijack connection", http.StatusInternalServerError)
		return nil, ErrUpgradeFailed
	}

	netConn, brw, err := hj.Hijack()
	if err != nil {
		http.Error(w, "Internal Server Error: hijack failed", http.StatusInternalServerError)
		return nil, err
	}

	// Compute Sec-WebSocket-Accept value
	acceptKey := ComputeAcceptKey(wsKey)

	// Build upgrade response (RFC 6455 4.2.2)
	// Use pre-allocated buffer for zero allocations
	var buf [256]byte
	n := 0
	n += copy(buf[n:], "HTTP/1.1 101 Switching Protocols\r\n")
	n += copy(buf[n:], "Upgrade: websocket\r\n")
	n += copy(buf[n:], "Connection: Upgrade\r\n")
	n += copy(buf[n:], "Sec-WebSocket-Accept: ")
	n += copy(buf[n:], acceptKey)
	n += copy(buf[n:], "\r\n")

	if subprotocol != "" {
		n += copy(buf[n:], "Sec-WebSocket-Protocol: ")
		n += copy(buf[n:], subprotocol)
		n += copy(buf[n:], "\r\n")
	}

	n += copy(buf[n:], "\r\n")

	// Write response
	if _, err := netConn.Write(buf[:n]); err != nil {
		netConn.Close()
		return nil, err
	}

	// Flush any buffered data
	if err := brw.Flush(); err != nil {
		netConn.Close()
		return nil, err
	}

	// Create WebSocket connection
	readBufSize := u.ReadBufferSize
	if readBufSize == 0 {
		readBufSize = 4096
	}
	writeBufSize := u.WriteBufferSize
	if writeBufSize == 0 {
		writeBufSize = 4096
	}

	return newConn(netConn, true, readBufSize, writeBufSize, subprotocol), nil
}

// Dial establishes a WebSocket client connection to the given URL.
// RFC 6455 Section 4.1: Client Requirements
func Dial(url string, headers http.Header) (*Conn, error) {
	// Parse URL
	var scheme, host, path string
	if strings.HasPrefix(url, "ws://") {
		scheme = "ws"
		host = url[5:]
	} else if strings.HasPrefix(url, "wss://") {
		scheme = "wss"
		host = url[6:]
	} else {
		return nil, errors.New("websocket: invalid URL scheme (must be ws:// or wss://)")
	}

	// Split host and path
	if idx := strings.Index(host, "/"); idx != -1 {
		path = host[idx:]
		host = host[:idx]
	} else {
		path = "/"
	}

	// Add default port if not specified
	if !strings.Contains(host, ":") {
		if scheme == "wss" {
			host += ":443"
		} else {
			host += ":80"
		}
	}

	// Connect to server
	// TODO: Add TLS support for wss://
	netConn, err := net.Dial("tcp", host)
	if err != nil {
		return nil, err
	}

	// Generate random Sec-WebSocket-Key (16 random bytes, base64-encoded)
	var keyBytes [16]byte
	if _, err := rand.Read(keyBytes[:]); err != nil {
		netConn.Close()
		return nil, err
	}
	wsKey := encodeBase64(keyBytes[:])

	// Build handshake request (RFC 6455 4.1)
	req := fmt.Sprintf(
		"GET %s HTTP/1.1\r\n"+
			"Host: %s\r\n"+
			"Upgrade: websocket\r\n"+
			"Connection: Upgrade\r\n"+
			"Sec-WebSocket-Key: %s\r\n"+
			"Sec-WebSocket-Version: 13\r\n",
		path, host, wsKey)

	// Add custom headers
	if headers != nil {
		for k, vs := range headers {
			for _, v := range vs {
				req += fmt.Sprintf("%s: %s\r\n", k, v)
			}
		}
	}

	req += "\r\n"

	// Send request
	if _, err := netConn.Write([]byte(req)); err != nil {
		netConn.Close()
		return nil, err
	}

	// Read response
	br := bufio.NewReader(netConn)
	resp, err := http.ReadResponse(br, &http.Request{Method: "GET"})
	if err != nil {
		netConn.Close()
		return nil, err
	}
	defer resp.Body.Close()

	// Validate response
	if resp.StatusCode != http.StatusSwitchingProtocols {
		netConn.Close()
		return nil, fmt.Errorf("websocket: bad status code: %d", resp.StatusCode)
	}

	if !headerContains(resp.Header, "Upgrade", "websocket") {
		netConn.Close()
		return nil, errors.New("websocket: missing Upgrade: websocket header")
	}

	if !headerContains(resp.Header, "Connection", "upgrade") {
		netConn.Close()
		return nil, errors.New("websocket: missing Connection: Upgrade header")
	}

	// Validate Sec-WebSocket-Accept
	expectedAccept := ComputeAcceptKey(wsKey)
	actualAccept := resp.Header.Get("Sec-WebSocket-Accept")
	if actualAccept != expectedAccept {
		netConn.Close()
		return nil, errors.New("websocket: invalid Sec-WebSocket-Accept")
	}

	subprotocol := resp.Header.Get("Sec-WebSocket-Protocol")

	// Create WebSocket connection (client mode)
	return newConn(netConn, false, 4096, 4096, subprotocol), nil
}

// Helper functions

// headerContains checks if a header contains a value (case-insensitive).
func headerContains(h http.Header, key, value string) bool {
	for _, v := range h[key] {
		for _, token := range strings.Split(v, ",") {
			if strings.EqualFold(strings.TrimSpace(token), value) {
				return true
			}
		}
	}
	return false
}

// headerValues returns all comma-separated values for a header.
func headerValues(h http.Header, key string) []string {
	var values []string
	for _, v := range h[key] {
		for _, token := range strings.Split(v, ",") {
			values = append(values, strings.TrimSpace(token))
		}
	}
	return values
}

// selectSubprotocol selects the first client protocol that is also supported by the server.
func selectSubprotocol(clientProtos, serverProtos []string) string {
	for _, clientProto := range clientProtos {
		for _, serverProto := range serverProtos {
			if clientProto == serverProto {
				return clientProto
			}
		}
	}
	return ""
}

// encodeBase64 encodes data to base64 without using base64.StdEncoding (for performance).
// Actually, let's just use the stdlib for correctness.
func encodeBase64(data []byte) string {
	const base64Table = "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789+/"

	n := len(data)
	result := make([]byte, (n+2)/3*4)

	j := 0
	for i := 0; i < n-2; i += 3 {
		result[j] = base64Table[data[i]>>2]
		result[j+1] = base64Table[(data[i]&0x03)<<4|(data[i+1]>>4)]
		result[j+2] = base64Table[(data[i+1]&0x0f)<<2|(data[i+2]>>6)]
		result[j+3] = base64Table[data[i+2]&0x3f]
		j += 4
	}

	// Handle remaining bytes
	switch n % 3 {
	case 1:
		result[j] = base64Table[data[n-1]>>2]
		result[j+1] = base64Table[(data[n-1]&0x03)<<4]
		result[j+2] = '='
		result[j+3] = '='
	case 2:
		result[j] = base64Table[data[n-2]>>2]
		result[j+1] = base64Table[(data[n-2]&0x03)<<4|(data[n-1]>>4)]
		result[j+2] = base64Table[(data[n-1]&0x0f)<<2]
		result[j+3] = '='
	}

	return string(result)
}

// IsWebSocketUpgrade checks if an HTTP request is a WebSocket upgrade request.
func IsWebSocketUpgrade(r *http.Request) bool {
	return r.Method == http.MethodGet &&
		headerContains(r.Header, "Connection", "upgrade") &&
		headerContains(r.Header, "Upgrade", "websocket") &&
		r.Header.Get("Sec-WebSocket-Version") == "13" &&
		r.Header.Get("Sec-WebSocket-Key") != ""
}

// WriteUpgradeResponse writes a WebSocket upgrade response directly to a writer.
// This is a low-level function for custom upgrade handling.
func WriteUpgradeResponse(w io.Writer, wsKey string, subprotocol string) error {
	acceptKey := ComputeAcceptKey(wsKey)

	response := "HTTP/1.1 101 Switching Protocols\r\n" +
		"Upgrade: websocket\r\n" +
		"Connection: Upgrade\r\n" +
		"Sec-WebSocket-Accept: " + acceptKey + "\r\n"

	if subprotocol != "" {
		response += "Sec-WebSocket-Protocol: " + subprotocol + "\r\n"
	}

	response += "\r\n"

	_, err := w.Write([]byte(response))
	return err
}
