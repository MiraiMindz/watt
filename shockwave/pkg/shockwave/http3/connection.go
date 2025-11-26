package http3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"net"
	"sync"

	"github.com/yourusername/shockwave/pkg/shockwave/http3/qpack"
	"github.com/yourusername/shockwave/pkg/shockwave/http3/quic"
)

// HTTP/3 Connection (RFC 9114)

var (
	ErrConnectionClosed = errors.New("http3: connection closed")
	ErrStreamClosed     = errors.New("http3: stream closed")
)

// Connection represents an HTTP/3 connection
type Connection struct {
	quicConn *quic.Connection
	isClient bool

	// Settings
	localSettings  *SettingsFrame
	remoteSettings *SettingsFrame
	settingsMu     sync.RWMutex

	// QPACK
	encoder *qpack.Encoder
	decoder *qpack.Decoder

	// Control streams
	controlStreamID       uint64
	qpackEncoderStreamID  uint64
	qpackDecoderStreamID  uint64

	// Stream management
	requestStreams   map[uint64]*RequestStream
	requestStreamsMu sync.RWMutex

	// Graceful shutdown
	goAwaySent     bool
	goAwayReceived bool
	goAwayMu       sync.Mutex

	// Context
	ctx    context.Context
	cancel context.CancelFunc
}

// NewConnection creates a new HTTP/3 connection
func NewConnection(quicConn *quic.Connection, isClient bool) *Connection {
	ctx, cancel := context.WithCancel(context.Background())

	maxTableCapacity := uint64(4096)
	if isClient {
		maxTableCapacity = 16384
	}

	conn := &Connection{
		quicConn:       quicConn,
		isClient:       isClient,
		localSettings:  DefaultSettings(),
		encoder:        qpack.NewEncoder(maxTableCapacity),
		decoder:        qpack.NewDecoder(maxTableCapacity),
		requestStreams: make(map[uint64]*RequestStream),
		ctx:            ctx,
		cancel:         cancel,
	}

	// Start control streams
	go conn.handleConnection()

	return conn
}

// handleConnection manages the HTTP/3 connection lifecycle
func (c *Connection) handleConnection() {
	// Create control stream (stream ID 0 for client, 1 for server)
	controlStream, err := c.quicConn.OpenUniStream()
	if err != nil {
		return
	}
	c.controlStreamID = controlStream.ID()

	// Send SETTINGS frame on control stream
	settingsData, _ := c.localSettings.AppendTo(nil)
	controlStream.Write(settingsData)

	// Create QPACK encoder stream (stream ID 2 for client, 3 for server)
	encoderStream, err := c.quicConn.OpenUniStream()
	if err != nil {
		return
	}
	c.qpackEncoderStreamID = encoderStream.ID()

	// Create QPACK decoder stream (stream ID 6 for client, 7 for server)
	decoderStream, err := c.quicConn.OpenUniStream()
	if err != nil {
		return
	}
	c.qpackDecoderStreamID = decoderStream.ID()

	// Handle incoming streams
	// In a real implementation, this would loop accepting streams
	<-c.ctx.Done()
}

// RoundTrip performs an HTTP/3 request (client-side)
func (c *Connection) RoundTrip(req *Request) (*Response, error) {
	if !c.isClient {
		return nil, errors.New("http3: RoundTrip only available for clients")
	}

	// Open bidirectional stream for request
	stream, err := c.quicConn.OpenStream()
	if err != nil {
		return nil, fmt.Errorf("http3: failed to open stream: %w", err)
	}

	reqStream := &RequestStream{
		stream:   stream,
		streamID: stream.ID(),
		conn:     c,
	}

	c.requestStreamsMu.Lock()
	c.requestStreams[stream.ID()] = reqStream
	c.requestStreamsMu.Unlock()

	// Send request
	if err := reqStream.SendRequest(req); err != nil {
		return nil, err
	}

	// Receive response
	resp, err := reqStream.ReceiveResponse()
	if err != nil {
		return nil, err
	}

	return resp, nil
}

// AcceptStream waits for and accepts an incoming request stream (server-side)
func (c *Connection) AcceptStream(ctx context.Context) (*RequestStream, error) {
	if c.isClient {
		return nil, errors.New("http3: AcceptStream only available for servers")
	}

	// Accept stream from QUIC connection
	stream, err := c.quicConn.AcceptStream()
	if err != nil {
		return nil, err
	}

	reqStream := &RequestStream{
		stream:   stream,
		streamID: stream.ID(),
		conn:     c,
	}

	c.requestStreamsMu.Lock()
	c.requestStreams[stream.ID()] = reqStream
	c.requestStreamsMu.Unlock()

	return reqStream, nil
}

// SendGoAway sends a GOAWAY frame
func (c *Connection) SendGoAway(lastStreamID uint64) error {
	c.goAwayMu.Lock()
	defer c.goAwayMu.Unlock()

	if c.goAwaySent {
		return nil
	}

	c.goAwaySent = true

	// Get control stream
	// In a real implementation, we'd track the control stream
	// For now, create GOAWAY frame
	frame := &GoAwayFrame{StreamID: lastStreamID}
	data, err := frame.AppendTo(nil)
	if err != nil {
		return err
	}

	// Send on control stream (would need to track the stream)
	_ = data

	return nil
}

// Close closes the HTTP/3 connection
func (c *Connection) Close() error {
	c.cancel()
	return c.quicConn.Close()
}

// RequestStream represents an HTTP/3 request stream
type RequestStream struct {
	stream   *quic.Stream
	streamID uint64
	conn     *Connection

	requestSent  bool
	responseSent bool

	mu sync.Mutex
}

// SendRequest sends an HTTP/3 request on the stream
func (rs *RequestStream) SendRequest(req *Request) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.requestSent {
		return errors.New("http3: request already sent")
	}

	// Encode headers with QPACK
	headers := []qpack.Header{
		{Name: ":method", Value: req.Method},
		{Name: ":scheme", Value: req.Scheme},
		{Name: ":authority", Value: req.Authority},
		{Name: ":path", Value: req.Path},
	}

	// Add regular headers
	for name, values := range req.Header {
		for _, value := range values {
			headers = append(headers, qpack.Header{Name: name, Value: value})
		}
	}

	// Encode headers
	encodedHeaders, _, err := rs.conn.encoder.EncodeHeaders(headers)
	if err != nil {
		return fmt.Errorf("http3: failed to encode headers: %w", err)
	}

	// Send HEADERS frame
	headersFrame := &HeadersFrame{HeaderBlock: encodedHeaders}
	frameData, err := headersFrame.AppendTo(nil)
	if err != nil {
		return err
	}

	if _, err := rs.stream.Write(frameData); err != nil {
		return fmt.Errorf("http3: failed to write headers: %w", err)
	}

	// Send body if present
	if req.Body != nil {
		buf := make([]byte, 16384) // 16KB chunks
		for {
			n, err := req.Body.Read(buf)
			if n > 0 {
				dataFrame := &DataFrame{Data: buf[:n]}
				frameData, err := dataFrame.AppendTo(nil)
				if err != nil {
					return err
				}

				if _, err := rs.stream.Write(frameData); err != nil {
					return fmt.Errorf("http3: failed to write data: %w", err)
				}
			}

			if err == io.EOF {
				break
			}
			if err != nil {
				return err
			}
		}
	}

	// Close stream for writing (sends FIN)
	rs.stream.Close()
	rs.requestSent = true

	return nil
}

// ReceiveResponse receives an HTTP/3 response from the stream
func (rs *RequestStream) ReceiveResponse() (*Response, error) {
	// Read frames from stream
	resp := &Response{
		Header: make(map[string][]string),
	}

	buf := make([]byte, 65536) // 64KB buffer
	var frameBuffer []byte

	for {
		n, err := rs.stream.Read(buf)
		if n > 0 {
			frameBuffer = append(frameBuffer, buf[:n]...)

			// Try to parse frames
			for len(frameBuffer) > 0 {
				frame, consumed, err := rs.tryParseFrame(frameBuffer)
				if err == io.ErrUnexpectedEOF {
					// Not enough data yet
					break
				}
				if err != nil {
					return nil, err
				}

				frameBuffer = frameBuffer[consumed:]

				// Handle frame
				switch f := frame.(type) {
				case *HeadersFrame:
					// Decode headers with QPACK decoder
					headers, err := rs.conn.decoder.DecodeHeaders(f.HeaderBlock)
					if err != nil {
						return nil, fmt.Errorf("http3: failed to decode headers: %w", err)
					}

					// Parse pseudo-headers and regular headers
					for _, h := range headers {
						if h.Name == ":status" {
							// Parse status code
							fmt.Sscanf(h.Value, "%d", &resp.StatusCode)
						} else if h.Name[0] != ':' {
							// Regular header
							resp.Header[h.Name] = append(resp.Header[h.Name], h.Value)
						}
					}

				case *DataFrame:
					// Accumulate body data
					resp.Body = append(resp.Body, f.Data...)
				}
			}
		}

		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}
	}

	return resp, nil
}

// tryParseFrame attempts to parse a frame from the buffer
func (rs *RequestStream) tryParseFrame(data []byte) (Frame, int, error) {
	if len(data) < 2 {
		return nil, 0, io.ErrUnexpectedEOF
	}

	r := &byteReader{data: data}
	frame, err := ParseFrame(r)
	if err != nil {
		return nil, 0, err
	}

	consumed := r.offset
	return frame, consumed, nil
}

// SendResponse sends an HTTP/3 response on the stream (server-side)
func (rs *RequestStream) SendResponse(resp *Response) error {
	rs.mu.Lock()
	defer rs.mu.Unlock()

	if rs.responseSent {
		return errors.New("http3: response already sent")
	}

	// Encode headers with QPACK
	headers := []qpack.Header{
		{Name: ":status", Value: fmt.Sprintf("%d", resp.StatusCode)},
	}

	// Add regular headers
	for name, values := range resp.Header {
		for _, value := range values {
			headers = append(headers, qpack.Header{Name: name, Value: value})
		}
	}

	// Encode headers
	encodedHeaders, _, err := rs.conn.encoder.EncodeHeaders(headers)
	if err != nil {
		return fmt.Errorf("http3: failed to encode headers: %w", err)
	}

	// Send HEADERS frame
	headersFrame := &HeadersFrame{HeaderBlock: encodedHeaders}
	frameData, err := headersFrame.AppendTo(nil)
	if err != nil {
		return err
	}

	if _, err := rs.stream.Write(frameData); err != nil {
		return fmt.Errorf("http3: failed to write headers: %w", err)
	}

	// Send body
	if len(resp.Body) > 0 {
		dataFrame := &DataFrame{Data: resp.Body}
		frameData, err := dataFrame.AppendTo(nil)
		if err != nil {
			return err
		}

		if _, err := rs.stream.Write(frameData); err != nil {
			return fmt.Errorf("http3: failed to write data: %w", err)
		}
	}

	// Close stream
	rs.stream.Close()
	rs.responseSent = true

	return nil
}

// Request represents an HTTP/3 request
type Request struct {
	Method    string
	Scheme    string
	Authority string
	Path      string
	Header    map[string][]string
	Body      io.Reader
}

// Response represents an HTTP/3 response
type Response struct {
	StatusCode  int
	Header      map[string][]string
	Body        []byte
	HeaderBlock []byte // Raw QPACK-encoded headers
}

// byteReader is a simple reader that tracks position
type byteReader struct {
	data   []byte
	offset int
}

func (r *byteReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}

// Helper to create a new client connection
func DialH3(addr string) (*Connection, error) {
	// Create UDP connection
	udpAddr, err := net.ResolveUDPAddr("udp", addr)
	if err != nil {
		return nil, err
	}

	udpConn, err := net.ListenUDP("udp", nil)
	if err != nil {
		return nil, err
	}

	// Create QUIC connection
	config := quic.DefaultConfig(true)
	quicConn, err := quic.NewConnection(udpConn, udpAddr, config, true)
	if err != nil {
		return nil, err
	}

	// Start QUIC handshake
	if err := quicConn.Start(); err != nil {
		return nil, err
	}

	// Create HTTP/3 connection
	h3Conn := NewConnection(quicConn, true)

	return h3Conn, nil
}
