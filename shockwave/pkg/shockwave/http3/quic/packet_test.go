package quic

import (
	"bytes"
	"testing"
)

func TestPacketNumberLen(t *testing.T) {
	tests := []struct {
		pn           uint64
		largestAcked uint64
		want         int
	}{
		{100, 99, 1},   // delta = 1 < 128
		{200, 100, 1},  // delta = 100 < 128
		{300, 100, 2},  // delta = 200 >= 128
		{1000, 100, 2}, // delta = 900
		{40000, 100, 3},
		{10000000, 100, 4},
	}

	for _, tt := range tests {
		got := PacketNumberLen(tt.pn, tt.largestAcked)
		if got != tt.want {
			t.Errorf("PacketNumberLen(%d, %d) = %d, want %d",
				tt.pn, tt.largestAcked, got, tt.want)
		}
	}
}

func TestDecodePacketNumber(t *testing.T) {
	tests := []struct {
		name      string
		largest   uint64
		truncated uint64
		nbits     int
		want      uint64
	}{
		{"simple", 100, 102, 1, 102},
		{"wrap-around", 255, 0, 1, 256},
		{"2-byte", 1000, 1001, 2, 1001},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			got := DecodePacketNumber(tt.largest, tt.truncated, tt.nbits)
			if got != tt.want {
				t.Errorf("DecodePacketNumber(%d, %d, %d) = %d, want %d",
					tt.largest, tt.truncated, tt.nbits, got, tt.want)
			}
		})
	}
}

func TestGenerateConnectionID(t *testing.T) {
	tests := []int{0, 1, 8, 16, 20}

	for _, length := range tests {
		cid, err := GenerateConnectionID(length)
		if err != nil {
			t.Fatalf("GenerateConnectionID(%d) error = %v", length, err)
		}
		if len(cid) != length {
			t.Errorf("GenerateConnectionID(%d) length = %d", length, len(cid))
		}
	}

	// Test invalid lengths
	if _, err := GenerateConnectionID(-1); err == nil {
		t.Error("GenerateConnectionID(-1) should fail")
	}
	if _, err := GenerateConnectionID(21); err == nil {
		t.Error("GenerateConnectionID(21) should fail")
	}
}

func TestInitialPacketRoundTrip(t *testing.T) {
	destCID, _ := GenerateConnectionID(8)
	srcCID, _ := GenerateConnectionID(8)

	packet := &Packet{
		Header: PacketHeader{
			IsLongHeader:    true,
			Version:         Version1,
			Type:            PacketTypeInitial,
			DestConnID:      destCID,
			SrcConnID:       srcCID,
			Token:           []byte("test-token"),
			PacketNumber:    42,
			PacketNumberLen: 2,
		},
		Payload: []byte("test payload data"),
	}

	// Update length field
	packet.Header.Length = uint64(packet.Header.PacketNumberLen + len(packet.Payload))

	// Encode
	buf := packet.AppendTo(nil)

	// Decode
	parsed, n, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket() error = %v", err)
	}

	if n != len(buf) {
		t.Errorf("ParsePacket() consumed %d bytes, want %d", n, len(buf))
	}

	// Verify header
	if !parsed.Header.IsLongHeader {
		t.Error("Expected long header")
	}
	if parsed.Header.Type != PacketTypeInitial {
		t.Errorf("Type = %v, want %v", parsed.Header.Type, PacketTypeInitial)
	}
	if parsed.Header.Version != Version1 {
		t.Errorf("Version = %x, want %x", parsed.Header.Version, Version1)
	}
	if !parsed.Header.DestConnID.Equal(destCID) {
		t.Errorf("DestConnID = %x, want %x", parsed.Header.DestConnID, destCID)
	}
	if !parsed.Header.SrcConnID.Equal(srcCID) {
		t.Errorf("SrcConnID = %x, want %x", parsed.Header.SrcConnID, srcCID)
	}
	if !bytes.Equal(parsed.Header.Token, packet.Header.Token) {
		t.Errorf("Token = %x, want %x", parsed.Header.Token, packet.Header.Token)
	}
	if parsed.Header.PacketNumber != packet.Header.PacketNumber {
		t.Errorf("PacketNumber = %d, want %d", parsed.Header.PacketNumber, packet.Header.PacketNumber)
	}
	if !bytes.Equal(parsed.Payload, packet.Payload) {
		t.Errorf("Payload = %x, want %x", parsed.Payload, packet.Payload)
	}
}

func TestHandshakePacketRoundTrip(t *testing.T) {
	destCID, _ := GenerateConnectionID(8)
	srcCID, _ := GenerateConnectionID(8)

	packet := &Packet{
		Header: PacketHeader{
			IsLongHeader:    true,
			Version:         Version1,
			Type:            PacketTypeHandshake,
			DestConnID:      destCID,
			SrcConnID:       srcCID,
			PacketNumber:    100,
			PacketNumberLen: 2,
		},
		Payload: []byte("handshake data"),
	}

	packet.Header.Length = uint64(packet.Header.PacketNumberLen + len(packet.Payload))

	// Encode
	buf := packet.AppendTo(nil)

	// Decode
	parsed, n, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket() error = %v", err)
	}

	if n != len(buf) {
		t.Errorf("ParsePacket() consumed %d bytes, want %d", n, len(buf))
	}

	if parsed.Header.Type != PacketTypeHandshake {
		t.Errorf("Type = %v, want %v", parsed.Header.Type, PacketTypeHandshake)
	}
	if !bytes.Equal(parsed.Payload, packet.Payload) {
		t.Errorf("Payload = %x, want %x", parsed.Payload, packet.Payload)
	}
}

func TestShortHeaderPacketRoundTrip(t *testing.T) {
	destCID, _ := GenerateConnectionID(8)

	packet := &Packet{
		Header: PacketHeader{
			IsLongHeader:    false,
			Type:            PacketType1RTT,
			DestConnID:      destCID,
			PacketNumber:    500,
			PacketNumberLen: 2,
		},
		Payload: []byte("application data"),
	}

	// Encode
	buf := packet.AppendTo(nil)

	// Decode (requires knowing dest conn ID length)
	parsed, n, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket() error = %v", err)
	}

	if n != len(buf) {
		t.Errorf("ParsePacket() consumed %d bytes, want %d", n, len(buf))
	}

	if parsed.Header.IsLongHeader {
		t.Error("Expected short header")
	}
	if parsed.Header.Type != PacketType1RTT {
		t.Errorf("Type = %v, want %v", parsed.Header.Type, PacketType1RTT)
	}
	if !bytes.Equal(parsed.Payload, packet.Payload) {
		t.Errorf("Payload = %x, want %x", parsed.Payload, packet.Payload)
	}
}

func TestRetryPacket(t *testing.T) {
	destCID, _ := GenerateConnectionID(8)
	srcCID, _ := GenerateConnectionID(8)

	packet := &Packet{
		Header: PacketHeader{
			IsLongHeader: true,
			Version:      Version1,
			Type:         PacketTypeRetry,
			DestConnID:   destCID,
			SrcConnID:    srcCID,
			RetryToken:   []byte("retry-token-data"),
			RetryIntegrity: [16]byte{
				0x01, 0x02, 0x03, 0x04, 0x05, 0x06, 0x07, 0x08,
				0x09, 0x0A, 0x0B, 0x0C, 0x0D, 0x0E, 0x0F, 0x10,
			},
		},
	}

	// Encode
	buf := packet.AppendTo(nil)

	// Decode
	parsed, n, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket() error = %v", err)
	}

	if n != len(buf) {
		t.Errorf("ParsePacket() consumed %d bytes, want %d", n, len(buf))
	}

	if parsed.Header.Type != PacketTypeRetry {
		t.Errorf("Type = %v, want %v", parsed.Header.Type, PacketTypeRetry)
	}
	if !bytes.Equal(parsed.Header.RetryToken, packet.Header.RetryToken) {
		t.Errorf("RetryToken = %x, want %x", parsed.Header.RetryToken, packet.Header.RetryToken)
	}
	if parsed.Header.RetryIntegrity != packet.Header.RetryIntegrity {
		t.Errorf("RetryIntegrity = %x, want %x", parsed.Header.RetryIntegrity, packet.Header.RetryIntegrity)
	}
}

func TestVersionNegotiationPacket(t *testing.T) {
	destCID, _ := GenerateConnectionID(8)
	srcCID, _ := GenerateConnectionID(8)

	// Build Version Negotiation packet manually
	buf := make([]byte, 0, 128)
	buf = append(buf, HeaderFormLong|FixedBit) // First byte
	buf = append(buf, 0x00, 0x00, 0x00, 0x00)  // Version = 0
	buf = appendConnectionID(buf, destCID)
	buf = appendConnectionID(buf, srcCID)

	// Add supported versions
	buf = append(buf, 0x00, 0x00, 0x00, 0x01) // Version 1

	// Parse
	parsed, n, err := ParsePacket(buf)
	if err != nil {
		t.Fatalf("ParsePacket() error = %v", err)
	}

	if n != len(buf) {
		t.Errorf("ParsePacket() consumed %d bytes, want %d", n, len(buf))
	}

	if parsed.Header.Type != PacketTypeVersionNeg {
		t.Errorf("Type = %v, want %v", parsed.Header.Type, PacketTypeVersionNeg)
	}
	if parsed.Header.Version != 0 {
		t.Errorf("Version = %x, want 0", parsed.Header.Version)
	}
	if !parsed.Header.DestConnID.Equal(destCID) {
		t.Errorf("DestConnID = %x, want %x", parsed.Header.DestConnID, destCID)
	}
	if !parsed.Header.SrcConnID.Equal(srcCID) {
		t.Errorf("SrcConnID = %x, want %x", parsed.Header.SrcConnID, srcCID)
	}
}

func TestInvalidPackets(t *testing.T) {
	tests := []struct {
		name string
		data []byte
	}{
		{"empty", []byte{}},
		{"too small", []byte{0x80}},
		{"missing fixed bit", []byte{0x80, 0x00, 0x00, 0x00, 0x01, 0x00, 0x00}},
		{"unsupported version", []byte{0xC0, 0x00, 0x00, 0x00, 0x02, 0x00, 0x00}},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			_, _, err := ParsePacket(tt.data)
			if err == nil {
				t.Error("ParsePacket() should fail for invalid packet")
			}
		})
	}
}

func BenchmarkPacketEncode(b *testing.B) {
	destCID, _ := GenerateConnectionID(8)
	srcCID, _ := GenerateConnectionID(8)

	packet := &Packet{
		Header: PacketHeader{
			IsLongHeader:    true,
			Version:         Version1,
			Type:            PacketTypeInitial,
			DestConnID:      destCID,
			SrcConnID:       srcCID,
			Token:           []byte("test-token"),
			PacketNumber:    42,
			PacketNumberLen: 2,
		},
		Payload: make([]byte, 1200),
	}
	packet.Header.Length = uint64(packet.Header.PacketNumberLen + len(packet.Payload))

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		buf := packet.AppendTo(nil)
		_ = buf
	}
}

func BenchmarkPacketDecode(b *testing.B) {
	destCID, _ := GenerateConnectionID(8)
	srcCID, _ := GenerateConnectionID(8)

	packet := &Packet{
		Header: PacketHeader{
			IsLongHeader:    true,
			Version:         Version1,
			Type:            PacketTypeInitial,
			DestConnID:      destCID,
			SrcConnID:       srcCID,
			Token:           []byte("test-token"),
			PacketNumber:    42,
			PacketNumberLen: 2,
		},
		Payload: make([]byte, 1200),
	}
	packet.Header.Length = uint64(packet.Header.PacketNumberLen + len(packet.Payload))

	buf := packet.AppendTo(nil)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_, _, err := ParsePacket(buf)
		if err != nil {
			b.Fatal(err)
		}
	}
}
