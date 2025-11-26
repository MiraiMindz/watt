package http3

import (
	"bytes"
	"io"
	"testing"
	"time"

	"github.com/yourusername/shockwave/pkg/shockwave/http3/qpack"
	"github.com/yourusername/shockwave/pkg/shockwave/http3/quic"
)

// TestHTTP3RequestResponseRoundTrip tests a complete HTTP/3 request/response cycle
func TestHTTP3RequestResponseRoundTrip(t *testing.T) {
	// Create encoder and decoder
	encoder := qpack.NewEncoder(4096)
	decoder := qpack.NewDecoder(4096)

	// Test request
	req := &Request{
		Method:    "GET",
		Scheme:    "https",
		Authority: "example.com",
		Path:      "/test",
		Header: map[string][]string{
			"user-agent": {"shockwave/1.0"},
			"accept":     {"text/html"},
		},
	}

	// Encode request headers
	headers := []qpack.Header{
		{Name: ":method", Value: req.Method},
		{Name: ":scheme", Value: req.Scheme},
		{Name: ":authority", Value: req.Authority},
		{Name: ":path", Value: req.Path},
		{Name: "user-agent", Value: "shockwave/1.0"},
		{Name: "accept", Value: "text/html"},
	}

	encodedHeaders, _, err := encoder.EncodeHeaders(headers)
	if err != nil {
		t.Fatalf("Failed to encode headers: %v", err)
	}

	// Create HEADERS frame
	headersFrame := &HeadersFrame{HeaderBlock: encodedHeaders}
	frameData, err := headersFrame.AppendTo(nil)
	if err != nil {
		t.Fatalf("Failed to serialize HEADERS frame: %v", err)
	}

	// Parse frame back
	r := &connByteReader{data: frameData}
	parsedFrame, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("Failed to parse frame: %v", err)
	}

	parsedHeaders, ok := parsedFrame.(*HeadersFrame)
	if !ok {
		t.Fatalf("Expected HeadersFrame, got %T", parsedFrame)
	}

	// Decode headers
	decodedHeaders, err := decoder.DecodeHeaders(parsedHeaders.HeaderBlock)
	if err != nil {
		t.Fatalf("Failed to decode headers: %v", err)
	}

	// Verify decoded headers
	expectedHeaders := map[string]string{
		":method":    "GET",
		":scheme":    "https",
		":authority": "example.com",
		":path":      "/test",
		"user-agent": "shockwave/1.0",
		"accept":     "text/html",
	}

	if len(decodedHeaders) != len(expectedHeaders) {
		t.Errorf("Expected %d headers, got %d", len(expectedHeaders), len(decodedHeaders))
	}

	for _, h := range decodedHeaders {
		if expected, ok := expectedHeaders[h.Name]; ok {
			if h.Value != expected {
				t.Errorf("Header %s: expected %q, got %q", h.Name, expected, h.Value)
			}
		} else {
			t.Errorf("Unexpected header: %s = %q", h.Name, h.Value)
		}
	}

	t.Logf("✓ Request headers encoded and decoded successfully (%d bytes)", len(encodedHeaders))
}

// TestHTTP3MultipleStreams tests handling multiple concurrent request streams
func TestHTTP3MultipleStreams(t *testing.T) {
	encoder := qpack.NewEncoder(4096)
	decoder := qpack.NewDecoder(4096)

	// Simulate 3 concurrent streams
	numStreams := 3
	streamData := make([][]byte, numStreams)

	for i := 0; i < numStreams; i++ {
		headers := []qpack.Header{
			{Name: ":method", Value: "GET"},
			{Name: ":scheme", Value: "https"},
			{Name: ":authority", Value: "example.com"},
			{Name: ":path", Value: "/stream" + string(rune('0'+i))},
		}

		encodedHeaders, _, err := encoder.EncodeHeaders(headers)
		if err != nil {
			t.Fatalf("Stream %d: failed to encode headers: %v", i, err)
		}

		headersFrame := &HeadersFrame{HeaderBlock: encodedHeaders}
		frameData, err := headersFrame.AppendTo(nil)
		if err != nil {
			t.Fatalf("Stream %d: failed to serialize frame: %v", i, err)
		}

		streamData[i] = frameData
	}

	// Decode all streams
	for i := 0; i < numStreams; i++ {
		r := &connByteReader{data: streamData[i]}
		frame, err := ParseFrame(r)
		if err != nil {
			t.Fatalf("Stream %d: failed to parse frame: %v", i, err)
		}

		headersFrame := frame.(*HeadersFrame)
		decodedHeaders, err := decoder.DecodeHeaders(headersFrame.HeaderBlock)
		if err != nil {
			t.Fatalf("Stream %d: failed to decode headers: %v", i, err)
		}

		// Verify stream-specific path
		pathFound := false
		for _, h := range decodedHeaders {
			if h.Name == ":path" {
				expectedPath := "/stream" + string(rune('0'+i))
				if h.Value != expectedPath {
					t.Errorf("Stream %d: expected path %q, got %q", i, expectedPath, h.Value)
				}
				pathFound = true
			}
		}

		if !pathFound {
			t.Errorf("Stream %d: :path header not found", i)
		}
	}

	t.Logf("✓ Successfully processed %d concurrent streams", numStreams)
}

// TestHTTP3LargeHeaders tests handling of large header blocks
func TestHTTP3LargeHeaders(t *testing.T) {
	encoder := qpack.NewEncoder(16384) // Larger table for many headers
	decoder := qpack.NewDecoder(16384)

	// Create request with many headers
	headers := []qpack.Header{
		{Name: ":method", Value: "POST"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "example.com"},
		{Name: ":path", Value: "/api/upload"},
		{Name: "content-type", Value: "application/json"},
		{Name: "user-agent", Value: "shockwave/1.0"},
		{Name: "accept", Value: "application/json"},
		{Name: "accept-encoding", Value: "gzip, deflate, br"},
		{Name: "accept-language", Value: "en-US,en;q=0.9"},
		{Name: "cache-control", Value: "no-cache"},
		{Name: "connection", Value: "keep-alive"},
	}

	// Add custom headers
	for i := 0; i < 20; i++ {
		headers = append(headers, qpack.Header{
			Name:  "x-custom-header-" + string(rune('a'+i)),
			Value: "value-" + string(rune('a'+i)),
		})
	}

	encodedHeaders, _, err := encoder.EncodeHeaders(headers)
	if err != nil {
		t.Fatalf("Failed to encode large headers: %v", err)
	}

	// Decode
	decodedHeaders, err := decoder.DecodeHeaders(encodedHeaders)
	if err != nil {
		t.Fatalf("Failed to decode large headers: %v", err)
	}

	if len(decodedHeaders) != len(headers) {
		t.Errorf("Expected %d headers, got %d", len(headers), len(decodedHeaders))
	}

	t.Logf("✓ Successfully encoded/decoded %d headers (%d bytes compressed)", len(headers), len(encodedHeaders))
}

// TestHTTP3DataFrameStreaming tests streaming large response bodies
func TestHTTP3DataFrameStreaming(t *testing.T) {
	// Create large response body (1MB)
	bodySize := 1024 * 1024
	responseBody := make([]byte, bodySize)
	for i := range responseBody {
		responseBody[i] = byte(i % 256)
	}

	// Stream in chunks (16KB each)
	chunkSize := 16384
	var totalSent int
	var frames [][]byte

	for offset := 0; offset < bodySize; offset += chunkSize {
		end := offset + chunkSize
		if end > bodySize {
			end = bodySize
		}

		chunk := responseBody[offset:end]
		dataFrame := &DataFrame{Data: chunk}
		frameData, err := dataFrame.AppendTo(nil)
		if err != nil {
			t.Fatalf("Failed to create DATA frame: %v", err)
		}

		frames = append(frames, frameData)
		totalSent += len(chunk)
	}

	// Parse and reassemble
	var receivedBody []byte
	for i, frameData := range frames {
		r := &connByteReader{data: frameData}
		frame, err := ParseFrame(r)
		if err != nil {
			t.Fatalf("Frame %d: failed to parse: %v", i, err)
		}

		dataFrame, ok := frame.(*DataFrame)
		if !ok {
			t.Fatalf("Frame %d: expected DataFrame, got %T", i, frame)
		}

		receivedBody = append(receivedBody, dataFrame.Data...)
	}

	if len(receivedBody) != bodySize {
		t.Errorf("Expected %d bytes, got %d", bodySize, len(receivedBody))
	}

	if !bytes.Equal(receivedBody, responseBody) {
		t.Error("Received body does not match sent body")
	}

	t.Logf("✓ Successfully streamed %d bytes in %d chunks", bodySize, len(frames))
}

// TestHTTP3SettingsExchange tests SETTINGS frame exchange
func TestHTTP3SettingsExchange(t *testing.T) {
	// Client settings
	clientSettings := &SettingsFrame{
		Settings: []Setting{
			{ID: SettingMaxFieldSectionSize, Value: 16384},
			{ID: SettingQPackMaxTableCapacity, Value: 4096},
			{ID: SettingQPackBlockedStreams, Value: 100},
		},
	}

	// Serialize
	settingsData, err := clientSettings.AppendTo(nil)
	if err != nil {
		t.Fatalf("Failed to serialize SETTINGS: %v", err)
	}

	// Parse
	r := &connByteReader{data: settingsData}
	frame, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("Failed to parse SETTINGS: %v", err)
	}

	parsedSettings, ok := frame.(*SettingsFrame)
	if !ok {
		t.Fatalf("Expected SettingsFrame, got %T", frame)
	}

	// Verify settings
	expectedSettings := map[uint64]uint64{
		SettingMaxFieldSectionSize:   16384,
		SettingQPackMaxTableCapacity: 4096,
		SettingQPackBlockedStreams:   100,
	}

	for _, setting := range parsedSettings.Settings {
		if expectedValue, ok := expectedSettings[setting.ID]; ok {
			if setting.Value != expectedValue {
				t.Errorf("Setting %d: expected %d, got %d", setting.ID, expectedValue, setting.Value)
			}
		}
	}

	t.Logf("✓ SETTINGS frame exchange successful")
}

// TestHTTP3ErrorHandling tests error frame handling
func TestHTTP3ErrorHandling(t *testing.T) {
	// Test GOAWAY frame
	goaway := &GoAwayFrame{StreamID: 100}
	goawayData, err := goaway.AppendTo(nil)
	if err != nil {
		t.Fatalf("Failed to serialize GOAWAY: %v", err)
	}

	r := &connByteReader{data: goawayData}
	frame, err := ParseFrame(r)
	if err != nil {
		t.Fatalf("Failed to parse GOAWAY: %v", err)
	}

	parsedGoaway, ok := frame.(*GoAwayFrame)
	if !ok {
		t.Fatalf("Expected GoAwayFrame, got %T", frame)
	}

	if parsedGoaway.StreamID != 100 {
		t.Errorf("Expected StreamID 100, got %d", parsedGoaway.StreamID)
	}

	t.Logf("✓ GOAWAY frame handling successful")
}

// TestHTTP3QUICIntegration tests HTTP/3 with QUIC transport primitives
func TestHTTP3QUICIntegration(t *testing.T) {
	// Test flow control interaction
	fc := quic.NewFlowController(1024*1024, 256*1024) // 1MB connection, 256KB stream

	// Simulate sending HTTP/3 frames through flow control
	encoder := qpack.NewEncoder(4096)
	headers := []qpack.Header{
		{Name: ":method", Value: "POST"},
		{Name: ":path", Value: "/upload"},
		{Name: "content-length", Value: "102400"},
	}

	encodedHeaders, _, err := encoder.EncodeHeaders(headers)
	if err != nil {
		t.Fatalf("Failed to encode headers: %v", err)
	}

	headersFrame := &HeadersFrame{HeaderBlock: encodedHeaders}
	headersData, _ := headersFrame.AppendTo(nil)

	// Check if we can send headers
	if !fc.CanSendData(uint64(len(headersData))) {
		t.Error("Flow control blocking headers frame")
	}

	fc.RecordDataSent(uint64(len(headersData)))

	// Simulate sending 100KB body in 10KB chunks
	chunkSize := 10240
	numChunks := 10

	for i := 0; i < numChunks; i++ {
		chunk := make([]byte, chunkSize)
		dataFrame := &DataFrame{Data: chunk}
		frameData, _ := dataFrame.AppendTo(nil)

		if !fc.CanSendData(uint64(len(frameData))) {
			t.Errorf("Flow control blocking chunk %d", i)
		}

		fc.RecordDataSent(uint64(len(frameData)))
	}

	t.Logf("✓ HTTP/3 frames sent through flow control")
}

// BenchmarkHTTP3HeaderEncoding benchmarks header encoding performance
func BenchmarkHTTP3HeaderEncoding(b *testing.B) {
	encoder := qpack.NewEncoder(4096)

	headers := []qpack.Header{
		{Name: ":method", Value: "GET"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "example.com"},
		{Name: ":path", Value: "/api/v1/users"},
		{Name: "user-agent", Value: "shockwave/1.0"},
		{Name: "accept", Value: "application/json"},
	}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := encoder.EncodeHeaders(headers)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHTTP3HeaderDecoding benchmarks header decoding performance
func BenchmarkHTTP3HeaderDecoding(b *testing.B) {
	encoder := qpack.NewEncoder(4096)
	decoder := qpack.NewDecoder(4096)

	headers := []qpack.Header{
		{Name: ":method", Value: "GET"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "example.com"},
		{Name: ":path", Value: "/api/v1/users"},
		{Name: "user-agent", Value: "shockwave/1.0"},
		{Name: "accept", Value: "application/json"},
	}

	encodedHeaders, _, _ := encoder.EncodeHeaders(headers)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := decoder.DecodeHeaders(encodedHeaders)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHTTP3FrameSerialization benchmarks frame serialization
func BenchmarkHTTP3FrameSerialization(b *testing.B) {
	data := make([]byte, 1024)
	dataFrame := &DataFrame{Data: data}

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, err := dataFrame.AppendTo(nil)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHTTP3FrameParsing benchmarks frame parsing
func BenchmarkHTTP3FrameParsing(b *testing.B) {
	data := make([]byte, 1024)
	dataFrame := &DataFrame{Data: data}
	frameData, _ := dataFrame.AppendTo(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		r := &connByteReader{data: frameData}
		_, err := ParseFrame(r)
		if err != nil {
			b.Fatal(err)
		}
	}
}

// BenchmarkHTTP3FullRequestResponse benchmarks a complete request/response cycle
func BenchmarkHTTP3FullRequestResponse(b *testing.B) {
	encoder := qpack.NewEncoder(4096)
	decoder := qpack.NewDecoder(4096)

	requestHeaders := []qpack.Header{
		{Name: ":method", Value: "GET"},
		{Name: ":scheme", Value: "https"},
		{Name: ":authority", Value: "example.com"},
		{Name: ":path", Value: "/"},
	}

	responseHeaders := []qpack.Header{
		{Name: ":status", Value: "200"},
		{Name: "content-type", Value: "text/html"},
		{Name: "content-length", Value: "1234"},
	}

	responseBody := make([]byte, 1234)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		// Encode request
		reqEncoded, _, _ := encoder.EncodeHeaders(requestHeaders)
		reqFrame := &HeadersFrame{HeaderBlock: reqEncoded}
		reqData, _ := reqFrame.AppendTo(nil)

		// Parse request
		r := &connByteReader{data: reqData}
		parsedReqFrame, _ := ParseFrame(r)
		decoder.DecodeHeaders(parsedReqFrame.(*HeadersFrame).HeaderBlock)

		// Encode response
		respEncoded, _, _ := encoder.EncodeHeaders(responseHeaders)
		respHeadersFrame := &HeadersFrame{HeaderBlock: respEncoded}
		respHeadersData, _ := respHeadersFrame.AppendTo(nil)

		respDataFrame := &DataFrame{Data: responseBody}
		respBodyData, _ := respDataFrame.AppendTo(nil)

		// Parse response
		r2 := &connByteReader{data: respHeadersData}
		parsedRespFrame, _ := ParseFrame(r2)
		decoder.DecodeHeaders(parsedRespFrame.(*HeadersFrame).HeaderBlock)

		r3 := &connByteReader{data: respBodyData}
		ParseFrame(r3)
	}
}

// TestHTTP3ConcurrentConnections tests multiple connections
func TestHTTP3ConcurrentConnections(t *testing.T) {
	numConns := 10
	results := make(chan error, numConns)

	for i := 0; i < numConns; i++ {
		go func(connID int) {
			encoder := qpack.NewEncoder(4096)
			decoder := qpack.NewDecoder(4096)

			headers := []qpack.Header{
				{Name: ":method", Value: "GET"},
				{Name: ":scheme", Value: "https"},
				{Name: ":authority", Value: "example.com"},
				{Name: ":path", Value: "/conn" + string(rune('0'+connID))},
			}

			// Encode
			encodedHeaders, _, err := encoder.EncodeHeaders(headers)
			if err != nil {
				results <- err
				return
			}

			// Decode
			_, err = decoder.DecodeHeaders(encodedHeaders)
			results <- err
		}(i)
	}

	// Wait for all connections
	timeout := time.After(5 * time.Second)
	for i := 0; i < numConns; i++ {
		select {
		case err := <-results:
			if err != nil {
				t.Errorf("Connection failed: %v", err)
			}
		case <-timeout:
			t.Fatal("Test timed out")
		}
	}

	t.Logf("✓ Successfully processed %d concurrent connections", numConns)
}

// Helper function to create test request
func createTestRequest(method, path string) *Request {
	return &Request{
		Method:    method,
		Scheme:    "https",
		Authority: "example.com",
		Path:      path,
		Header:    make(map[string][]string),
		Body:      nil,
	}
}

// Helper function to create test response
func createTestResponse(statusCode int, body []byte) *Response {
	return &Response{
		StatusCode: statusCode,
		Header:     make(map[string][]string),
		Body:       body,
	}
}

// connByteReader helper for tests (named differently to avoid conflict with connection.go)
type connByteReader struct {
	data   []byte
	offset int
}

func (r *connByteReader) Read(p []byte) (int, error) {
	if r.offset >= len(r.data) {
		return 0, io.EOF
	}
	n := copy(p, r.data[r.offset:])
	r.offset += n
	return n, nil
}
