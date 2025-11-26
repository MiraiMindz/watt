package http2

import (
	"encoding/binary"
	"testing"
)

// Benchmark frame header parsing (should be 0 allocs/op)
func BenchmarkParseFrameHeader(b *testing.B) {
	input := [9]byte{0x00, 0x00, 0x0A, 0x00, 0x01, 0x00, 0x00, 0x00, 0x01}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ParseFrameHeader(input)
	}
}

// Benchmark frame header writing (should be 0 allocs/op)
func BenchmarkWriteFrameHeader(b *testing.B) {
	fh := FrameHeader{
		Length:   10,
		Type:     FrameData,
		Flags:    FlagDataEndStream,
		StreamID: 1,
	}

	var buf [9]byte

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		WriteFrameHeader(buf[:], fh)
	}
}

// Benchmark frame header validation
func BenchmarkFrameHeaderValidation(b *testing.B) {
	fh := FrameHeader{
		Length:   100,
		Type:     FrameData,
		Flags:    0,
		StreamID: 1,
	}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = fh.Validate()
	}
}

// Benchmark DATA frame parsing
func BenchmarkParseDataFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   1024,
		Type:     FrameData,
		Flags:    0,
		StreamID: 1,
	}
	payload := make([]byte, 1024)

	b.ReportAllocs()
	b.SetBytes(1024)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseDataFrame(fh, payload)
	}
}

// Benchmark DATA frame with padding
func BenchmarkParseDataFramePadded(b *testing.B) {
	fh := FrameHeader{
		Length:   1024,
		Type:     FrameData,
		Flags:    FlagDataPadded,
		StreamID: 1,
	}
	payload := make([]byte, 1024)
	payload[0] = 10 // 10 bytes padding

	b.ReportAllocs()
	b.SetBytes(1024)
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseDataFrame(fh, payload)
	}
}

// Benchmark HEADERS frame parsing
func BenchmarkParseHeadersFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   100,
		Type:     FrameHeaders,
		Flags:    FlagHeadersEndHeaders,
		StreamID: 1,
	}
	payload := make([]byte, 100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseHeadersFrame(fh, payload)
	}
}

// Benchmark HEADERS frame with priority
func BenchmarkParseHeadersFrameWithPriority(b *testing.B) {
	fh := FrameHeader{
		Length:   105,
		Type:     FrameHeaders,
		Flags:    FlagHeadersPriority | FlagHeadersEndHeaders,
		StreamID: 1,
	}
	payload := make([]byte, 105)
	// Set priority fields
	binary.BigEndian.PutUint32(payload[0:4], 0x80000003) // Exclusive, depends on stream 3
	payload[4] = 16                                       // Weight

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseHeadersFrame(fh, payload)
	}
}

// Benchmark PRIORITY frame parsing (should be very fast, 0 allocs after header)
func BenchmarkParsePriorityFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   5,
		Type:     FramePriority,
		Flags:    0,
		StreamID: 1,
	}
	payload := make([]byte, 5)
	binary.BigEndian.PutUint32(payload[0:4], 0x80000003)
	payload[4] = 16

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParsePriorityFrame(fh, payload)
	}
}

// Benchmark RST_STREAM frame parsing (should be very fast, 0 allocs after header)
func BenchmarkParseRSTStreamFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   4,
		Type:     FrameRSTStream,
		Flags:    0,
		StreamID: 1,
	}
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, uint32(ErrCodeCancel))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseRSTStreamFrame(fh, payload)
	}
}

// Benchmark SETTINGS frame parsing
func BenchmarkParseSettingsFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   12,
		Type:     FrameSettings,
		Flags:    0,
		StreamID: 0,
	}
	payload := make([]byte, 12)
	// Setting 1: HEADER_TABLE_SIZE = 4096
	binary.BigEndian.PutUint16(payload[0:2], uint16(SettingHeaderTableSize))
	binary.BigEndian.PutUint32(payload[2:6], 4096)
	// Setting 2: MAX_CONCURRENT_STREAMS = 100
	binary.BigEndian.PutUint16(payload[6:8], uint16(SettingMaxConcurrentStreams))
	binary.BigEndian.PutUint32(payload[8:12], 100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseSettingsFrame(fh, payload)
	}
}

// Benchmark SETTINGS ACK (should be fastest, minimal work)
func BenchmarkParseSettingsFrameAck(b *testing.B) {
	fh := FrameHeader{
		Length:   0,
		Type:     FrameSettings,
		Flags:    FlagSettingsAck,
		StreamID: 0,
	}
	payload := []byte{}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseSettingsFrame(fh, payload)
	}
}

// Benchmark PING frame parsing (should be very fast, 0 allocs)
func BenchmarkParsePingFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   8,
		Type:     FramePing,
		Flags:    0,
		StreamID: 0,
	}
	payload := []byte{0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08}

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParsePingFrame(fh, payload)
	}
}

// Benchmark GOAWAY frame parsing
func BenchmarkParseGoAwayFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   8,
		Type:     FrameGoAway,
		Flags:    0,
		StreamID: 0,
	}
	payload := make([]byte, 8)
	binary.BigEndian.PutUint32(payload[0:4], 7)
	binary.BigEndian.PutUint32(payload[4:8], uint32(ErrCodeNo))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseGoAwayFrame(fh, payload)
	}
}

// Benchmark GOAWAY frame with debug data
func BenchmarkParseGoAwayFrameWithDebugData(b *testing.B) {
	fh := FrameHeader{
		Length:   28,
		Type:     FrameGoAway,
		Flags:    0,
		StreamID: 0,
	}
	payload := make([]byte, 28)
	binary.BigEndian.PutUint32(payload[0:4], 7)
	binary.BigEndian.PutUint32(payload[4:8], uint32(ErrCodeProtocol))
	copy(payload[8:], []byte("connection terminated"))

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseGoAwayFrame(fh, payload)
	}
}

// Benchmark WINDOW_UPDATE frame parsing (should be very fast, 0 allocs)
func BenchmarkParseWindowUpdateFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   4,
		Type:     FrameWindowUpdate,
		Flags:    0,
		StreamID: 1,
	}
	payload := make([]byte, 4)
	binary.BigEndian.PutUint32(payload, 1024)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseWindowUpdateFrame(fh, payload)
	}
}

// Benchmark CONTINUATION frame parsing
func BenchmarkParseContinuationFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   100,
		Type:     FrameContinuation,
		Flags:    FlagContinuationEndHeaders,
		StreamID: 1,
	}
	payload := make([]byte, 100)

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParseContinuationFrame(fh, payload)
	}
}

// Benchmark PUSH_PROMISE frame parsing
func BenchmarkParsePushPromiseFrame(b *testing.B) {
	fh := FrameHeader{
		Length:   104,
		Type:     FramePushPromise,
		Flags:    FlagPushPromiseEndHeaders,
		StreamID: 1,
	}
	payload := make([]byte, 104)
	binary.BigEndian.PutUint32(payload[0:4], 2) // Promised stream ID

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_, _ = ParsePushPromiseFrame(fh, payload)
	}
}

// Benchmark suite: all frame types
func BenchmarkFrameParsingAllTypes(b *testing.B) {
	b.Run("DATA", BenchmarkParseDataFrame)
	b.Run("HEADERS", BenchmarkParseHeadersFrame)
	b.Run("PRIORITY", BenchmarkParsePriorityFrame)
	b.Run("RST_STREAM", BenchmarkParseRSTStreamFrame)
	b.Run("SETTINGS", BenchmarkParseSettingsFrame)
	b.Run("PUSH_PROMISE", BenchmarkParsePushPromiseFrame)
	b.Run("PING", BenchmarkParsePingFrame)
	b.Run("GOAWAY", BenchmarkParseGoAwayFrame)
	b.Run("WINDOW_UPDATE", BenchmarkParseWindowUpdateFrame)
	b.Run("CONTINUATION", BenchmarkParseContinuationFrame)
}

// Benchmark frame type string conversion
func BenchmarkFrameTypeString(b *testing.B) {
	ft := FrameData

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ft.String()
	}
}

// Benchmark error code string conversion
func BenchmarkErrorCodeString(b *testing.B) {
	ec := ErrCodeProtocol

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = ec.String()
	}
}

// Benchmark flag checking
func BenchmarkFlagsHas(b *testing.B) {
	flags := FlagDataEndStream | FlagDataPadded

	b.ReportAllocs()
	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		_ = flags.Has(FlagDataEndStream)
	}
}
