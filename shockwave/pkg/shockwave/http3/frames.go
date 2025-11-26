package http3

import (
	"bytes"
	"errors"
	"fmt"
	"io"
)

// HTTP/3 Frame Types (RFC 9114 Section 7.2)

type FrameType uint64

const (
	FrameTypeData         FrameType = 0x00
	FrameTypeHeaders      FrameType = 0x01
	FrameTypeCancelPush   FrameType = 0x03
	FrameTypeSettings     FrameType = 0x04
	FrameTypePushPromise  FrameType = 0x05
	FrameTypeGoAway       FrameType = 0x07
	FrameTypeMaxPushID    FrameType = 0x0D
)

// Settings identifiers (RFC 9114 Section 7.2.4.1)
const (
	SettingQPackMaxTableCapacity    uint64 = 0x01
	SettingMaxFieldSectionSize      uint64 = 0x06
	SettingQPackBlockedStreams      uint64 = 0x07
	SettingEnableConnectProtocol    uint64 = 0x08
	SettingH3Datagram               uint64 = 0x33
)

var (
	ErrInvalidFrame     = errors.New("http3: invalid frame")
	ErrFrameTooLarge    = errors.New("http3: frame too large")
	ErrUnknownFrameType = errors.New("http3: unknown frame type")
)

// Frame represents an HTTP/3 frame
type Frame interface {
	Type() FrameType
	Length() uint64
	AppendTo(buf []byte) ([]byte, error)
}

// DataFrame represents a DATA frame (0x00)
type DataFrame struct {
	Data []byte
}

func (f *DataFrame) Type() FrameType { return FrameTypeData }
func (f *DataFrame) Length() uint64  { return uint64(len(f.Data)) }

func (f *DataFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = appendVarInt(buf, uint64(f.Type()))
	buf = appendVarInt(buf, f.Length())
	buf = append(buf, f.Data...)
	return buf, nil
}

// HeadersFrame represents a HEADERS frame (0x01)
type HeadersFrame struct {
	HeaderBlock []byte // QPACK-encoded headers
}

func (f *HeadersFrame) Type() FrameType { return FrameTypeHeaders }
func (f *HeadersFrame) Length() uint64  { return uint64(len(f.HeaderBlock)) }

func (f *HeadersFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = appendVarInt(buf, uint64(f.Type()))
	buf = appendVarInt(buf, f.Length())
	buf = append(buf, f.HeaderBlock...)
	return buf, nil
}

// CancelPushFrame represents a CANCEL_PUSH frame (0x03)
type CancelPushFrame struct {
	PushID uint64
}

func (f *CancelPushFrame) Type() FrameType { return FrameTypeCancelPush }
func (f *CancelPushFrame) Length() uint64  { return varIntLen(f.PushID) }

func (f *CancelPushFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = appendVarInt(buf, uint64(f.Type()))
	buf = appendVarInt(buf, f.Length())
	buf = appendVarInt(buf, f.PushID)
	return buf, nil
}

// Setting represents a single HTTP/3 setting
type Setting struct {
	ID    uint64
	Value uint64
}

// SettingsFrame represents a SETTINGS frame (0x04)
type SettingsFrame struct {
	Settings []Setting
}

func (f *SettingsFrame) Type() FrameType { return FrameTypeSettings }

func (f *SettingsFrame) Length() uint64 {
	length := uint64(0)
	for _, s := range f.Settings {
		length += varIntLen(s.ID) + varIntLen(s.Value)
	}
	return length
}

func (f *SettingsFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = appendVarInt(buf, uint64(f.Type()))
	buf = appendVarInt(buf, f.Length())

	for _, s := range f.Settings {
		buf = appendVarInt(buf, s.ID)
		buf = appendVarInt(buf, s.Value)
	}

	return buf, nil
}

// GetSetting returns the value of a setting by ID
func (f *SettingsFrame) GetSetting(id uint64) (uint64, bool) {
	for _, s := range f.Settings {
		if s.ID == id {
			return s.Value, true
		}
	}
	return 0, false
}

// PushPromiseFrame represents a PUSH_PROMISE frame (0x05)
type PushPromiseFrame struct {
	PushID      uint64
	HeaderBlock []byte // QPACK-encoded headers
}

func (f *PushPromiseFrame) Type() FrameType { return FrameTypePushPromise }

func (f *PushPromiseFrame) Length() uint64 {
	return varIntLen(f.PushID) + uint64(len(f.HeaderBlock))
}

func (f *PushPromiseFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = appendVarInt(buf, uint64(f.Type()))
	buf = appendVarInt(buf, f.Length())
	buf = appendVarInt(buf, f.PushID)
	buf = append(buf, f.HeaderBlock...)
	return buf, nil
}

// GoAwayFrame represents a GOAWAY frame (0x07)
type GoAwayFrame struct {
	StreamID uint64 // Last processed stream ID
}

func (f *GoAwayFrame) Type() FrameType { return FrameTypeGoAway }
func (f *GoAwayFrame) Length() uint64  { return varIntLen(f.StreamID) }

func (f *GoAwayFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = appendVarInt(buf, uint64(f.Type()))
	buf = appendVarInt(buf, f.Length())
	buf = appendVarInt(buf, f.StreamID)
	return buf, nil
}

// MaxPushIDFrame represents a MAX_PUSH_ID frame (0x0D)
type MaxPushIDFrame struct {
	PushID uint64
}

func (f *MaxPushIDFrame) Type() FrameType { return FrameTypeMaxPushID }
func (f *MaxPushIDFrame) Length() uint64  { return varIntLen(f.PushID) }

func (f *MaxPushIDFrame) AppendTo(buf []byte) ([]byte, error) {
	buf = appendVarInt(buf, uint64(f.Type()))
	buf = appendVarInt(buf, f.Length())
	buf = appendVarInt(buf, f.PushID)
	return buf, nil
}

// ParseFrame parses an HTTP/3 frame from data
func ParseFrame(r io.Reader) (Frame, error) {
	// Read frame type
	frameType, err := readVarInt(r)
	if err != nil {
		return nil, err
	}

	// Read frame length
	length, err := readVarInt(r)
	if err != nil {
		return nil, err
	}

	// Read frame payload
	payload := make([]byte, length)
	if _, err := io.ReadFull(r, payload); err != nil {
		return nil, err
	}

	// Parse based on type
	switch FrameType(frameType) {
	case FrameTypeData:
		return parseDataFrame(payload)
	case FrameTypeHeaders:
		return parseHeadersFrame(payload)
	case FrameTypeCancelPush:
		return parseCancelPushFrame(payload)
	case FrameTypeSettings:
		return parseSettingsFrame(payload)
	case FrameTypePushPromise:
		return parsePushPromiseFrame(payload)
	case FrameTypeGoAway:
		return parseGoAwayFrame(payload)
	case FrameTypeMaxPushID:
		return parseMaxPushIDFrame(payload)
	default:
		// Unknown frame type - skip it (RFC 9114 Section 9)
		return nil, fmt.Errorf("%w: 0x%x", ErrUnknownFrameType, frameType)
	}
}

// Frame parsers

func parseDataFrame(payload []byte) (*DataFrame, error) {
	return &DataFrame{Data: payload}, nil
}

func parseHeadersFrame(payload []byte) (*HeadersFrame, error) {
	return &HeadersFrame{HeaderBlock: payload}, nil
}

func parseCancelPushFrame(payload []byte) (*CancelPushFrame, error) {
	r := bytes.NewReader(payload)
	pushID, err := readVarInt(r)
	if err != nil {
		return nil, err
	}
	return &CancelPushFrame{PushID: pushID}, nil
}

func parseSettingsFrame(payload []byte) (*SettingsFrame, error) {
	r := bytes.NewReader(payload)
	var settings []Setting

	for {
		id, err := readVarInt(r)
		if err == io.EOF {
			break
		}
		if err != nil {
			return nil, err
		}

		value, err := readVarInt(r)
		if err != nil {
			return nil, err
		}

		settings = append(settings, Setting{ID: id, Value: value})
	}

	return &SettingsFrame{Settings: settings}, nil
}

func parsePushPromiseFrame(payload []byte) (*PushPromiseFrame, error) {
	r := bytes.NewReader(payload)

	pushID, err := readVarInt(r)
	if err != nil {
		return nil, err
	}

	// Rest is header block
	headerBlock := make([]byte, r.Len())
	if _, err := r.Read(headerBlock); err != nil && err != io.EOF {
		return nil, err
	}

	return &PushPromiseFrame{
		PushID:      pushID,
		HeaderBlock: headerBlock,
	}, nil
}

func parseGoAwayFrame(payload []byte) (*GoAwayFrame, error) {
	r := bytes.NewReader(payload)
	streamID, err := readVarInt(r)
	if err != nil {
		return nil, err
	}
	return &GoAwayFrame{StreamID: streamID}, nil
}

func parseMaxPushIDFrame(payload []byte) (*MaxPushIDFrame, error) {
	r := bytes.NewReader(payload)
	pushID, err := readVarInt(r)
	if err != nil {
		return nil, err
	}
	return &MaxPushIDFrame{PushID: pushID}, nil
}

// Variable-length integer helpers (RFC 9000 style)

func appendVarInt(buf []byte, v uint64) []byte {
	switch {
	case v <= 63:
		return append(buf, byte(v))
	case v <= 16383:
		return append(buf, byte(v>>8)|0x40, byte(v))
	case v <= 1073741823:
		return append(buf,
			byte(v>>24)|0x80,
			byte(v>>16),
			byte(v>>8),
			byte(v),
		)
	default:
		return append(buf,
			byte(v>>56)|0xC0,
			byte(v>>48),
			byte(v>>40),
			byte(v>>32),
			byte(v>>24),
			byte(v>>16),
			byte(v>>8),
			byte(v),
		)
	}
}

func readVarInt(r io.Reader) (uint64, error) {
	var firstByte [1]byte
	if _, err := io.ReadFull(r, firstByte[:]); err != nil {
		return 0, err
	}

	prefix := firstByte[0] >> 6

	switch prefix {
	case 0: // 1-byte
		return uint64(firstByte[0] & 0x3F), nil

	case 1: // 2-byte
		var buf [1]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		v := uint64(firstByte[0]&0x3F)<<8 | uint64(buf[0])
		return v, nil

	case 2: // 4-byte
		var buf [3]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		v := uint64(firstByte[0]&0x3F)<<24 |
			uint64(buf[0])<<16 |
			uint64(buf[1])<<8 |
			uint64(buf[2])
		return v, nil

	case 3: // 8-byte
		var buf [7]byte
		if _, err := io.ReadFull(r, buf[:]); err != nil {
			return 0, err
		}
		v := uint64(firstByte[0]&0x3F)<<56 |
			uint64(buf[0])<<48 |
			uint64(buf[1])<<40 |
			uint64(buf[2])<<32 |
			uint64(buf[3])<<24 |
			uint64(buf[4])<<16 |
			uint64(buf[5])<<8 |
			uint64(buf[6])
		return v, nil
	}

	return 0, errors.New("http3: invalid varint")
}

func varIntLen(v uint64) uint64 {
	switch {
	case v <= 63:
		return 1
	case v <= 16383:
		return 2
	case v <= 1073741823:
		return 4
	default:
		return 8
	}
}

// DefaultSettings returns default HTTP/3 settings
func DefaultSettings() *SettingsFrame {
	return &SettingsFrame{
		Settings: []Setting{
			{ID: SettingQPackMaxTableCapacity, Value: 4096},
			{ID: SettingMaxFieldSectionSize, Value: 16384},
			{ID: SettingQPackBlockedStreams, Value: 100},
			{ID: SettingH3Datagram, Value: 1}, // Enable datagrams
		},
	}
}
