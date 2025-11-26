package quic

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// QUIC Frame Types (RFC 9000 Section 19)
// Frames are the building blocks of QUIC packets

type FrameType uint64

const (
	// Frame types
	FrameTypePadding         FrameType = 0x00
	FrameTypePing            FrameType = 0x01
	FrameTypeAck             FrameType = 0x02 // ACK without ECN
	FrameTypeAckECN          FrameType = 0x03 // ACK with ECN
	FrameTypeResetStream     FrameType = 0x04
	FrameTypeStopSending     FrameType = 0x05
	FrameTypeCrypto          FrameType = 0x06
	FrameTypeNewToken        FrameType = 0x07
	FrameTypeStream          FrameType = 0x08 // Base type, actual range 0x08-0x0F
	FrameTypeMaxData         FrameType = 0x10
	FrameTypeMaxStreamData   FrameType = 0x11
	FrameTypeMaxStreams      FrameType = 0x12 // Bidirectional
	FrameTypeMaxStreamsBidi  FrameType = 0x12
	FrameTypeMaxStreamsUni   FrameType = 0x13 // Unidirectional
	FrameTypeDataBlocked     FrameType = 0x14
	FrameTypeStreamDataBlocked FrameType = 0x15
	FrameTypeStreamsBlocked  FrameType = 0x16 // Bidirectional
	FrameTypeStreamsBlockedBidi FrameType = 0x16
	FrameTypeStreamsBlockedUni  FrameType = 0x17 // Unidirectional
	FrameTypeNewConnectionID FrameType = 0x18
	FrameTypeRetireConnectionID FrameType = 0x19
	FrameTypePathChallenge   FrameType = 0x1A
	FrameTypePathResponse    FrameType = 0x1B
	FrameTypeConnectionClose FrameType = 0x1C // QUIC error
	FrameTypeConnectionCloseApp FrameType = 0x1D // Application error
	FrameTypeHandshakeDone   FrameType = 0x1E

	// Extension frames (RFC 9221 - Datagrams)
	FrameTypeDatagram        FrameType = 0x30
	FrameTypeDatagramLen     FrameType = 0x31
)

// Stream frame flags (bits 0-2 of frame type)
const (
	StreamFrameFlagFIN = 0x01 // Last frame of stream
	StreamFrameFlagLEN = 0x02 // Length field present
	StreamFrameFlagOFF = 0x04 // Offset field present
)

var (
	ErrInvalidFrame = errors.New("quic: invalid frame")
	ErrFrameTooLarge = errors.New("quic: frame too large")
)

// Frame represents a QUIC frame
type Frame interface {
	Type() FrameType
	AppendTo(buf []byte) ([]byte, error)
}

// PaddingFrame is a PADDING frame (0x00)
type PaddingFrame struct {
	Length int
}

func (f *PaddingFrame) Type() FrameType { return FrameTypePadding }

func (f *PaddingFrame) AppendTo(buf []byte) ([]byte, error) {
	for i := 0; i < f.Length; i++ {
		buf = append(buf, 0x00)
	}
	return buf, nil
}

// PingFrame is a PING frame (0x01)
type PingFrame struct{}

func (f *PingFrame) Type() FrameType { return FrameTypePing }

func (f *PingFrame) AppendTo(buf []byte) ([]byte, error) {
	return append(buf, byte(FrameTypePing)), nil
}

// AckRange represents a range of acknowledged packets
type AckRange struct {
	Gap   uint64 // Gap from previous range
	Length uint64 // Number of acknowledged packets
}

// AckFrame is an ACK frame (0x02 or 0x03)
type AckFrame struct {
	LargestAcked uint64
	AckDelay     uint64 // In microseconds, encoded/decoded with ack_delay_exponent
	Ranges       []AckRange
	ECN          *ECNCounts // nil if not present
}

// ECNCounts holds ECN counters
type ECNCounts struct {
	ECT0 uint64
	ECT1 uint64
	CE   uint64
}

func (f *AckFrame) Type() FrameType {
	if f.ECN != nil {
		return FrameTypeAckECN
	}
	return FrameTypeAck
}

func (f *AckFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(f.Type()))

	var err error
	buf, err = appendVarint(buf, f.LargestAcked)
	if err != nil {
		return buf, err
	}

	buf, err = appendVarint(buf, f.AckDelay)
	if err != nil {
		return buf, err
	}

	// ACK Range Count
	buf, err = appendVarint(buf, uint64(len(f.Ranges)-1))
	if err != nil {
		return buf, err
	}

	// First ACK Range
	if len(f.Ranges) > 0 {
		buf, err = appendVarint(buf, f.Ranges[0].Length)
		if err != nil {
			return buf, err
		}
	}

	// Additional ACK Ranges
	for i := 1; i < len(f.Ranges); i++ {
		buf, err = appendVarint(buf, f.Ranges[i].Gap)
		if err != nil {
			return buf, err
		}
		buf, err = appendVarint(buf, f.Ranges[i].Length)
		if err != nil {
			return buf, err
		}
	}

	// ECN Counts (if present)
	if f.ECN != nil {
		buf, err = appendVarint(buf, f.ECN.ECT0)
		if err != nil {
			return buf, err
		}
		buf, err = appendVarint(buf, f.ECN.ECT1)
		if err != nil {
			return buf, err
		}
		buf, err = appendVarint(buf, f.ECN.CE)
		if err != nil {
			return buf, err
		}
	}

	return buf, nil
}

// ResetStreamFrame is a RESET_STREAM frame (0x04)
type ResetStreamFrame struct {
	StreamID  uint64
	ErrorCode uint64
	FinalSize uint64
}

func (f *ResetStreamFrame) Type() FrameType { return FrameTypeResetStream }

func (f *ResetStreamFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(FrameTypeResetStream))

	var err error
	buf, err = appendVarint(buf, f.StreamID)
	if err != nil {
		return buf, err
	}
	buf, err = appendVarint(buf, f.ErrorCode)
	if err != nil {
		return buf, err
	}
	buf, err = appendVarint(buf, f.FinalSize)
	if err != nil {
		return buf, err
	}

	return buf, nil
}

// StopSendingFrame is a STOP_SENDING frame (0x05)
type StopSendingFrame struct {
	StreamID  uint64
	ErrorCode uint64
}

func (f *StopSendingFrame) Type() FrameType { return FrameTypeStopSending }

func (f *StopSendingFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(FrameTypeStopSending))

	var err error
	buf, err = appendVarint(buf, f.StreamID)
	if err != nil {
		return buf, err
	}
	buf, err = appendVarint(buf, f.ErrorCode)
	if err != nil {
		return buf, err
	}

	return buf, nil
}

// CryptoFrame is a CRYPTO frame (0x06)
type CryptoFrame struct {
	Offset uint64
	Data   []byte
}

func (f *CryptoFrame) Type() FrameType { return FrameTypeCrypto }

func (f *CryptoFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(FrameTypeCrypto))

	var err error
	buf, err = appendVarint(buf, f.Offset)
	if err != nil {
		return buf, err
	}
	buf, err = appendVarint(buf, uint64(len(f.Data)))
	if err != nil {
		return buf, err
	}
	buf = append(buf, f.Data...)

	return buf, nil
}

// NewTokenFrame is a NEW_TOKEN frame (0x07)
type NewTokenFrame struct {
	Token []byte
}

func (f *NewTokenFrame) Type() FrameType { return FrameTypeNewToken }

func (f *NewTokenFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(FrameTypeNewToken))

	var err error
	buf, err = appendVarint(buf, uint64(len(f.Token)))
	if err != nil {
		return buf, err
	}
	buf = append(buf, f.Token...)

	return buf, nil
}

// StreamFrame is a STREAM frame (0x08-0x0F)
type StreamFrame struct {
	StreamID uint64
	Offset   uint64
	Data     []byte
	Fin      bool
}

func (f *StreamFrame) Type() FrameType {
	typ := uint8(FrameTypeStream)
	if f.Fin {
		typ |= StreamFrameFlagFIN
	}
	typ |= StreamFrameFlagLEN // Always include length
	if f.Offset > 0 {
		typ |= StreamFrameFlagOFF
	}
	return FrameType(typ)
}

func (f *StreamFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(f.Type()))

	var err error
	buf, err = appendVarint(buf, f.StreamID)
	if err != nil {
		return buf, err
	}

	if f.Offset > 0 {
		buf, err = appendVarint(buf, f.Offset)
		if err != nil {
			return buf, err
		}
	}

	buf, err = appendVarint(buf, uint64(len(f.Data)))
	if err != nil {
		return buf, err
	}
	buf = append(buf, f.Data...)

	return buf, nil
}

// MaxDataFrame is a MAX_DATA frame (0x10)
type MaxDataFrame struct {
	MaximumData uint64
}

func (f *MaxDataFrame) Type() FrameType { return FrameTypeMaxData }

func (f *MaxDataFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(FrameTypeMaxData))
	return appendVarint(buf, f.MaximumData)
}

// MaxStreamDataFrame is a MAX_STREAM_DATA frame (0x11)
type MaxStreamDataFrame struct {
	StreamID    uint64
	MaximumData uint64
}

func (f *MaxStreamDataFrame) Type() FrameType { return FrameTypeMaxStreamData }

func (f *MaxStreamDataFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(FrameTypeMaxStreamData))

	var err error
	buf, err = appendVarint(buf, f.StreamID)
	if err != nil {
		return buf, err
	}
	return appendVarint(buf, f.MaximumData)
}

// MaxStreamsFrame is a MAX_STREAMS frame (0x12 or 0x13)
type MaxStreamsFrame struct {
	MaximumStreams uint64
	Bidirectional  bool
}

func (f *MaxStreamsFrame) Type() FrameType {
	if f.Bidirectional {
		return FrameTypeMaxStreamsBidi
	}
	return FrameTypeMaxStreamsUni
}

func (f *MaxStreamsFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(f.Type()))
	return appendVarint(buf, f.MaximumStreams)
}

// ConnectionCloseFrame is a CONNECTION_CLOSE frame (0x1C or 0x1D)
type ConnectionCloseFrame struct {
	ErrorCode    uint64
	FrameType    uint64 // Only for QUIC errors (0x1C)
	ReasonPhrase []byte
	IsAppError   bool   // true for 0x1D, false for 0x1C
}

func (f *ConnectionCloseFrame) Type() FrameType {
	if f.IsAppError {
		return FrameTypeConnectionCloseApp
	}
	return FrameTypeConnectionClose
}

func (f *ConnectionCloseFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(f.Type()))

	var err error
	buf, err = appendVarint(buf, f.ErrorCode)
	if err != nil {
		return buf, err
	}

	// Frame Type (only for QUIC errors)
	if !f.IsAppError {
		buf, err = appendVarint(buf, f.FrameType)
		if err != nil {
			return buf, err
		}
	}

	// Reason Phrase
	buf, err = appendVarint(buf, uint64(len(f.ReasonPhrase)))
	if err != nil {
		return buf, err
	}
	buf = append(buf, f.ReasonPhrase...)

	return buf, nil
}

// DatagramFrame is a DATAGRAM frame (0x30 or 0x31) - RFC 9221
type DatagramFrame struct {
	Data []byte
}

func (f *DatagramFrame) Type() FrameType {
	return FrameTypeDatagramLen // Always use length-prefixed variant
}

func (f *DatagramFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = append(buf, byte(FrameTypeDatagramLen))

	var err error
	buf, err = appendVarint(buf, uint64(len(f.Data)))
	if err != nil {
		return buf, err
	}
	buf = append(buf, f.Data...)

	return buf, nil
}

// HandshakeDoneFrame is a HANDSHAKE_DONE frame (0x1E)
type HandshakeDoneFrame struct{}

func (f *HandshakeDoneFrame) Type() FrameType { return FrameTypeHandshakeDone }

func (f *HandshakeDoneFrame) AppendTo(buf []byte) ([]byte, error) {
	return append(buf, byte(FrameTypeHandshakeDone)), nil
}

// ParseFrame parses a single frame from data.
// Returns (frame, bytesConsumed, error).
func ParseFrame(data []byte) (Frame, int, error) {
	if len(data) == 0 {
		return nil, 0, io.ErrUnexpectedEOF
	}

	r := bytes.NewReader(data)

	// Read frame type
	frameType, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset := n

	var frame Frame

	switch FrameType(frameType) {
	case FrameTypePadding:
		// Count consecutive padding bytes
		count := 1
		for offset < len(data) && data[offset] == 0x00 {
			count++
			offset++
		}
		frame = &PaddingFrame{Length: count}

	case FrameTypePing:
		frame = &PingFrame{}

	case FrameTypeAck, FrameTypeAckECN:
		ack, n, err := parseAckFrame(data[offset:], frameType == uint64(FrameTypeAckECN))
		if err != nil {
			return nil, 0, err
		}
		offset += n
		frame = ack

	case FrameTypeCrypto:
		crypto, n, err := parseCryptoFrame(data[offset:])
		if err != nil {
			return nil, 0, err
		}
		offset += n
		frame = crypto

	case FrameTypeConnectionClose, FrameTypeConnectionCloseApp:
		cc, n, err := parseConnectionCloseFrame(data[offset:], frameType == uint64(FrameTypeConnectionCloseApp))
		if err != nil {
			return nil, 0, err
		}
		offset += n
		frame = cc

	case FrameTypeHandshakeDone:
		frame = &HandshakeDoneFrame{}

	default:
		// Check if it's a STREAM frame (0x08-0x0F)
		if frameType >= 0x08 && frameType <= 0x0F {
			stream, n, err := parseStreamFrame(data[offset:], uint8(frameType))
			if err != nil {
				return nil, 0, err
			}
			offset += n
			frame = stream
		} else {
			return nil, 0, fmt.Errorf("quic: unsupported frame type 0x%02x", frameType)
		}
	}

	return frame, offset, nil
}

// Helper parsers for complex frames

func parseAckFrame(data []byte, hasECN bool) (*AckFrame, int, error) {
	r := bytes.NewReader(data)
	offset := 0

	largestAcked, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	ackDelay, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	rangeCount, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	firstRange, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	ranges := []AckRange{{Gap: 0, Length: firstRange}}

	for i := uint64(0); i < rangeCount; i++ {
		gap, n, err := readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n

		length, n, err := readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n

		ranges = append(ranges, AckRange{Gap: gap, Length: length})
	}

	ack := &AckFrame{
		LargestAcked: largestAcked,
		AckDelay:     ackDelay,
		Ranges:       ranges,
	}

	if hasECN {
		ect0, n, err := readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n

		ect1, n, err := readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n

		ce, n, err := readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n

		ack.ECN = &ECNCounts{ECT0: ect0, ECT1: ect1, CE: ce}
	}

	return ack, offset, nil
}

func parseCryptoFrame(data []byte) (*CryptoFrame, int, error) {
	r := bytes.NewReader(data)
	offset := 0

	cryptoOffset, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	length, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	if uint64(len(data)) < uint64(offset)+length {
		return nil, 0, io.ErrUnexpectedEOF
	}

	cryptoData := make([]byte, length)
	copy(cryptoData, data[offset:offset+int(length)])
	offset += int(length)

	return &CryptoFrame{
		Offset: cryptoOffset,
		Data:   cryptoData,
	}, offset, nil
}

func parseStreamFrame(data []byte, frameTypeByte uint8) (*StreamFrame, int, error) {
	r := bytes.NewReader(data)
	offset := 0

	fin := (frameTypeByte & StreamFrameFlagFIN) != 0
	hasLen := (frameTypeByte & StreamFrameFlagLEN) != 0
	hasOff := (frameTypeByte & StreamFrameFlagOFF) != 0

	streamID, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	streamOffset := uint64(0)
	if hasOff {
		streamOffset, n, err = readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n
	}

	var streamData []byte
	if hasLen {
		length, n, err := readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n

		if uint64(len(data)) < uint64(offset)+length {
			return nil, 0, io.ErrUnexpectedEOF
		}

		streamData = make([]byte, length)
		copy(streamData, data[offset:offset+int(length)])
		offset += int(length)
	} else {
		// No length, data extends to end of packet
		streamData = make([]byte, len(data)-offset)
		copy(streamData, data[offset:])
		offset = len(data)
	}

	return &StreamFrame{
		StreamID: streamID,
		Offset:   streamOffset,
		Data:     streamData,
		Fin:      fin,
	}, offset, nil
}

func parseConnectionCloseFrame(data []byte, isAppError bool) (*ConnectionCloseFrame, int, error) {
	r := bytes.NewReader(data)
	offset := 0

	errorCode, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	frameType := uint64(0)
	if !isAppError {
		frameType, n, err = readVarint(r)
		if err != nil {
			return nil, 0, err
		}
		offset += n
	}

	reasonLen, n, err := readVarint(r)
	if err != nil {
		return nil, 0, err
	}
	offset += n

	if uint64(len(data)) < uint64(offset)+reasonLen {
		return nil, 0, io.ErrUnexpectedEOF
	}

	reason := make([]byte, reasonLen)
	copy(reason, data[offset:offset+int(reasonLen)])
	offset += int(reasonLen)

	return &ConnectionCloseFrame{
		ErrorCode:    errorCode,
		FrameType:    frameType,
		ReasonPhrase: reason,
		IsAppError:   isAppError,
	}, offset, nil
}
