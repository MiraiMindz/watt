package quic

import (
	"crypto/rand"
	"encoding/binary"
	"errors"
	"fmt"
	"io"
)

// QUIC Packet Format (RFC 9000 Section 17)
//
// Long Header Packets (used during handshake):
//   Initial, 0-RTT, Handshake, Retry
//
// Short Header Packets (used after handshake):
//   1-RTT protected packets

// PacketType represents the type of QUIC packet
type PacketType uint8

const (
	PacketTypeInitial     PacketType = 0x00 // Initial packet
	PacketType0RTT        PacketType = 0x01 // 0-RTT packet
	PacketTypeHandshake   PacketType = 0x02 // Handshake packet
	PacketTypeRetry       PacketType = 0x03 // Retry packet
	PacketType1RTT        PacketType = 0x04 // 1-RTT (short header)
	PacketTypeVersionNeg  PacketType = 0xFF // Version Negotiation
)

const (
	// QUIC version 1 (RFC 9000)
	Version1 = 0x00000001

	// Header flags
	HeaderFormLong  = 0x80 // Long header (bit 7 = 1)
	HeaderFormShort = 0x00 // Short header (bit 7 = 0)
	FixedBit        = 0x40 // Fixed bit (bit 6 = 1, must be set)

	// Long header type bits (bits 5-4 when long header)
	LongHeaderTypeInitial   = 0x00
	LongHeaderType0RTT      = 0x10
	LongHeaderTypeHandshake = 0x20
	LongHeaderTypeRetry     = 0x30

	// Packet number length encoding (bits 1-0)
	PacketNumberLen1 = 0x00
	PacketNumberLen2 = 0x01
	PacketNumberLen3 = 0x02
	PacketNumberLen4 = 0x03

	// Maximum packet sizes
	MaxPacketSize     = 1452 // Typical MTU - IPv6 - UDP
	MinInitialPacket  = 1200 // RFC 9000 Section 14.1
	MaxConnectionIDLen = 20
)

var (
	ErrInvalidPacket     = errors.New("quic: invalid packet")
	ErrUnsupportedVersion = errors.New("quic: unsupported version")
	ErrPacketTooSmall    = errors.New("quic: packet too small")
)

// PacketHeader represents a QUIC packet header (Long or Short)
type PacketHeader struct {
	IsLongHeader bool
	Version      uint32
	Type         PacketType

	// Connection IDs
	DestConnID   ConnectionID
	SrcConnID    ConnectionID

	// Packet number
	PacketNumber     uint64
	PacketNumberLen  int // 1-4 bytes

	// Long header specific
	Token            []byte // For Initial packets
	Length           uint64 // Payload + packet number length

	// Retry specific
	RetryToken       []byte
	RetryIntegrity   [16]byte // Retry integrity tag
}

// Packet represents a QUIC packet
type Packet struct {
	Header  PacketHeader
	Payload []byte
}

// ParsePacket parses a QUIC packet from raw bytes.
// Returns (*Packet, bytesConsumed, error).
func ParsePacket(data []byte) (*Packet, int, error) {
	if len(data) == 0 {
		return nil, 0, ErrPacketTooSmall
	}

	firstByte := data[0]

	// Check if this is a long header packet
	if firstByte&HeaderFormLong != 0 {
		return parseLongHeaderPacket(data)
	}

	return parseShortHeaderPacket(data)
}

// parseLongHeaderPacket parses a long header packet.
func parseLongHeaderPacket(data []byte) (*Packet, int, error) {
	if len(data) < 5 { // Minimum: 1 byte flags + 4 bytes version
		return nil, 0, ErrPacketTooSmall
	}

	offset := 0
	firstByte := data[offset]
	offset++

	// Check fixed bit
	if firstByte&FixedBit == 0 {
		return nil, 0, ErrInvalidPacket
	}

	// Parse version
	version := binary.BigEndian.Uint32(data[offset:])
	offset += 4

	// Check for Version Negotiation packet (version == 0)
	if version == 0 {
		return parseVersionNegotiationPacket(data)
	}

	// Verify version
	if version != Version1 {
		return nil, 0, ErrUnsupportedVersion
	}

	// Determine packet type
	typeField := firstByte & 0x30
	var packetType PacketType
	switch typeField {
	case LongHeaderTypeInitial:
		packetType = PacketTypeInitial
	case LongHeaderType0RTT:
		packetType = PacketType0RTT
	case LongHeaderTypeHandshake:
		packetType = PacketTypeHandshake
	case LongHeaderTypeRetry:
		packetType = PacketTypeRetry
	default:
		return nil, 0, ErrInvalidPacket
	}

	// Parse Destination Connection ID
	destConnID, n, err := parseConnectionID(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("quic: parse dest conn id: %w", err)
	}
	offset += n

	// Parse Source Connection ID
	srcConnID, n, err := parseConnectionID(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("quic: parse src conn id: %w", err)
	}
	offset += n

	header := PacketHeader{
		IsLongHeader: true,
		Version:      version,
		Type:         packetType,
		DestConnID:   destConnID,
		SrcConnID:    srcConnID,
	}

	// Handle Retry packet (no packet number, has integrity tag)
	if packetType == PacketTypeRetry {
		if len(data) < offset+16 {
			return nil, 0, ErrPacketTooSmall
		}
		tokenLen := len(data) - offset - 16
		header.RetryToken = make([]byte, tokenLen)
		copy(header.RetryToken, data[offset:offset+tokenLen])
		copy(header.RetryIntegrity[:], data[offset+tokenLen:])

		return &Packet{Header: header}, len(data), nil
	}

	// Parse token (only for Initial packets)
	if packetType == PacketTypeInitial {
		tokenLen, n, err := parseVarint(data[offset:])
		if err != nil {
			return nil, 0, fmt.Errorf("quic: parse token length: %w", err)
		}
		offset += n

		if tokenLen > 0 {
			if uint64(len(data)) < uint64(offset)+tokenLen {
				return nil, 0, ErrPacketTooSmall
			}
			header.Token = make([]byte, tokenLen)
			copy(header.Token, data[offset:offset+int(tokenLen)])
			offset += int(tokenLen)
		}
	}

	// Parse length
	length, n, err := parseVarint(data[offset:])
	if err != nil {
		return nil, 0, fmt.Errorf("quic: parse length: %w", err)
	}
	offset += n
	header.Length = length

	// Check if we have enough data
	if uint64(len(data)) < uint64(offset)+length {
		return nil, 0, ErrPacketTooSmall
	}

	// Parse packet number length (from first byte, bits 1-0)
	pnLenBits := firstByte & 0x03
	pnLen := int(pnLenBits) + 1
	header.PacketNumberLen = pnLen

	if len(data) < offset+pnLen {
		return nil, 0, ErrPacketTooSmall
	}

	// Parse packet number (truncated, will be reconstructed later)
	pn := uint64(0)
	for i := 0; i < pnLen; i++ {
		pn = (pn << 8) | uint64(data[offset+i])
	}
	header.PacketNumber = pn
	offset += pnLen

	// The rest is encrypted payload
	payloadLen := int(length) - pnLen
	payload := make([]byte, payloadLen)
	copy(payload, data[offset:offset+payloadLen])
	offset += payloadLen

	return &Packet{
		Header:  header,
		Payload: payload,
	}, offset, nil
}

// parseShortHeaderPacket parses a short header (1-RTT) packet.
func parseShortHeaderPacket(data []byte) (*Packet, int, error) {
	if len(data) < 2 { // Minimum: 1 byte flags + at least 1 byte dest conn ID
		return nil, 0, ErrPacketTooSmall
	}

	offset := 0
	firstByte := data[offset]
	offset++

	// Check fixed bit
	if firstByte&FixedBit == 0 {
		return nil, 0, ErrInvalidPacket
	}

	// Parse Destination Connection ID
	// For short header, we need to know the connection ID length from context
	// For now, we'll assume a typical length of 8 bytes
	// In a real implementation, this would come from the connection state
	destConnIDLen := 8
	if len(data) < offset+destConnIDLen {
		return nil, 0, ErrPacketTooSmall
	}

	destConnID := make([]byte, destConnIDLen)
	copy(destConnID, data[offset:offset+destConnIDLen])
	offset += destConnIDLen

	// Parse packet number length (bits 1-0)
	pnLenBits := firstByte & 0x03
	pnLen := int(pnLenBits) + 1

	if len(data) < offset+pnLen {
		return nil, 0, ErrPacketTooSmall
	}

	// Parse packet number (truncated)
	pn := uint64(0)
	for i := 0; i < pnLen; i++ {
		pn = (pn << 8) | uint64(data[offset+i])
	}
	offset += pnLen

	header := PacketHeader{
		IsLongHeader:    false,
		Type:            PacketType1RTT,
		DestConnID:      destConnID,
		PacketNumber:    pn,
		PacketNumberLen: pnLen,
	}

	// The rest is encrypted payload
	payload := make([]byte, len(data)-offset)
	copy(payload, data[offset:])

	return &Packet{
		Header:  header,
		Payload: payload,
	}, len(data), nil
}

// parseVersionNegotiationPacket parses a Version Negotiation packet.
func parseVersionNegotiationPacket(data []byte) (*Packet, int, error) {
	offset := 1 // Skip first byte
	offset += 4 // Skip version field (already checked to be 0)

	// Parse Destination Connection ID
	destConnID, n, err := parseConnectionID(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	offset += n

	// Parse Source Connection ID
	srcConnID, n, err := parseConnectionID(data[offset:])
	if err != nil {
		return nil, 0, err
	}
	offset += n

	// The rest is a list of supported versions (4 bytes each)
	versions := []uint32{}
	for offset+4 <= len(data) {
		ver := binary.BigEndian.Uint32(data[offset:])
		versions = append(versions, ver)
		offset += 4
	}

	header := PacketHeader{
		IsLongHeader: true,
		Type:         PacketTypeVersionNeg,
		Version:      0,
		DestConnID:   destConnID,
		SrcConnID:    srcConnID,
	}

	// Store versions in payload as encoded bytes
	payload := make([]byte, len(versions)*4)
	for i, ver := range versions {
		binary.BigEndian.PutUint32(payload[i*4:], ver)
	}

	return &Packet{
		Header:  header,
		Payload: payload,
	}, offset, nil
}

// WriteTo writes the packet to w.
func (p *Packet) WriteTo(w io.Writer) (int64, error) {
	buf := make([]byte, 0, MaxPacketSize)
	buf = p.AppendTo(buf)
	n, err := w.Write(buf)
	return int64(n), err
}

// AppendTo appends the packet to buf and returns the updated buffer.
func (p *Packet) AppendTo(buf []byte) []byte {
	if p.Header.IsLongHeader {
		return p.appendLongHeader(buf)
	}
	return p.appendShortHeader(buf)
}

// appendLongHeader appends a long header packet to buf.
func (p *Packet) appendLongHeader(buf []byte) []byte {
	// First byte: form (1) + fixed (1) + type (2) + reserved (2) + pn_len (2)
	firstByte := byte(HeaderFormLong | FixedBit)

	switch p.Header.Type {
	case PacketTypeInitial:
		firstByte |= LongHeaderTypeInitial
	case PacketType0RTT:
		firstByte |= LongHeaderType0RTT
	case PacketTypeHandshake:
		firstByte |= LongHeaderTypeHandshake
	case PacketTypeRetry:
		firstByte |= LongHeaderTypeRetry
	}

	// Add packet number length (bits 1-0)
	if p.Header.Type != PacketTypeRetry {
		pnLenBits := byte(p.Header.PacketNumberLen - 1)
		firstByte |= pnLenBits
	}

	buf = append(buf, firstByte)

	// Version
	var verBuf [4]byte
	binary.BigEndian.PutUint32(verBuf[:], p.Header.Version)
	buf = append(buf, verBuf[:]...)

	// Destination Connection ID
	buf = appendConnectionID(buf, p.Header.DestConnID)

	// Source Connection ID
	buf = appendConnectionID(buf, p.Header.SrcConnID)

	// Handle Retry packet
	if p.Header.Type == PacketTypeRetry {
		buf = append(buf, p.Header.RetryToken...)
		buf = append(buf, p.Header.RetryIntegrity[:]...)
		return buf
	}

	// Token (for Initial packets)
	if p.Header.Type == PacketTypeInitial {
		var err error
		buf, err = appendVarint(buf, uint64(len(p.Header.Token)))
		if err != nil {
			return buf
		}
		buf = append(buf, p.Header.Token...)
	}

	// Length (packet number + payload)
	payloadLen := uint64(p.Header.PacketNumberLen + len(p.Payload))
	buf, _ = appendVarint(buf, payloadLen)

	// Packet number (truncated)
	for i := p.Header.PacketNumberLen - 1; i >= 0; i-- {
		buf = append(buf, byte(p.Header.PacketNumber>>(i*8)))
	}

	// Payload
	buf = append(buf, p.Payload...)

	return buf
}

// appendShortHeader appends a short header packet to buf.
func (p *Packet) appendShortHeader(buf []byte) []byte {
	// First byte: form (0) + fixed (1) + spin (1) + reserved (2) + key_phase (1) + pn_len (2)
	pnLenBits := byte(p.Header.PacketNumberLen - 1)
	firstByte := FixedBit | pnLenBits
	buf = append(buf, firstByte)

	// Destination Connection ID
	buf = append(buf, p.Header.DestConnID...)

	// Packet number (truncated)
	for i := p.Header.PacketNumberLen - 1; i >= 0; i-- {
		buf = append(buf, byte(p.Header.PacketNumber>>(i*8)))
	}

	// Payload
	buf = append(buf, p.Payload...)

	return buf
}

// GenerateConnectionID generates a random connection ID of the specified length.
func GenerateConnectionID(length int) (ConnectionID, error) {
	if length < 0 || length > MaxConnectionIDLen {
		return nil, fmt.Errorf("quic: invalid connection ID length %d", length)
	}
	if length == 0 {
		return ConnectionID{}, nil
	}

	cid := make([]byte, length)
	if _, err := rand.Read(cid); err != nil {
		return nil, err
	}
	return ConnectionID(cid), nil
}

// PacketNumberLen returns the minimum number of bytes needed to encode pn.
func PacketNumberLen(pn uint64, largestAcked uint64) int {
	// RFC 9000 Section 17.1: packet number should be encoded with enough
	// bytes to prevent the receiver from being unable to decode it
	delta := pn - largestAcked

	if delta < (1 << 7) {
		return 1
	}
	if delta < (1 << 15) {
		return 2
	}
	if delta < (1 << 23) {
		return 3
	}
	return 4
}

// DecodePacketNumber reconstructs the full packet number from a truncated value.
// RFC 9000 Appendix A.3
func DecodePacketNumber(largest uint64, truncated uint64, nbits int) uint64 {
	expected := largest + 1
	win := uint64(1) << (nbits * 8)
	hwin := win / 2
	mask := win - 1

	// Align expected to the next possible packet number
	candidate := (expected &^ mask) | truncated

	if candidate+hwin <= expected {
		return candidate + win
	}
	if candidate > expected+hwin && candidate > win {
		return candidate - win
	}
	return candidate
}
