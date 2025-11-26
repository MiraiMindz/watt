package http2

import (
	"encoding/binary"
	"fmt"
)

// FrameType represents an HTTP/2 frame type (RFC 7540 §4.1)
type FrameType uint8

const (
	FrameData         FrameType = 0x0
	FrameHeaders      FrameType = 0x1
	FramePriority     FrameType = 0x2
	FrameRSTStream    FrameType = 0x3
	FrameSettings     FrameType = 0x4
	FramePushPromise  FrameType = 0x5
	FramePing         FrameType = 0x6
	FrameGoAway       FrameType = 0x7
	FrameWindowUpdate FrameType = 0x8
	FrameContinuation FrameType = 0x9
)

// String returns the string representation of the frame type
func (t FrameType) String() string {
	switch t {
	case FrameData:
		return "DATA"
	case FrameHeaders:
		return "HEADERS"
	case FramePriority:
		return "PRIORITY"
	case FrameRSTStream:
		return "RST_STREAM"
	case FrameSettings:
		return "SETTINGS"
	case FramePushPromise:
		return "PUSH_PROMISE"
	case FramePing:
		return "PING"
	case FrameGoAway:
		return "GOAWAY"
	case FrameWindowUpdate:
		return "WINDOW_UPDATE"
	case FrameContinuation:
		return "CONTINUATION"
	default:
		return fmt.Sprintf("UNKNOWN(%d)", uint8(t))
	}
}

// Flags represents frame flags (RFC 7540 §4.1)
type Flags uint8

const (
	// Flags for DATA frames
	FlagDataEndStream Flags = 0x1
	FlagDataPadded    Flags = 0x8

	// Flags for HEADERS frames
	FlagHeadersEndStream  Flags = 0x1
	FlagHeadersEndHeaders Flags = 0x4
	FlagHeadersPadded     Flags = 0x8
	FlagHeadersPriority   Flags = 0x20

	// Flags for SETTINGS frames
	FlagSettingsAck Flags = 0x1

	// Flags for PING frames
	FlagPingAck Flags = 0x1

	// Flags for CONTINUATION frames
	FlagContinuationEndHeaders Flags = 0x4

	// Flags for PUSH_PROMISE frames
	FlagPushPromiseEndHeaders Flags = 0x4
	FlagPushPromisePadded     Flags = 0x8
)

// Has checks if a specific flag is set
func (f Flags) Has(flag Flags) bool {
	return f&flag != 0
}

// FrameHeader represents an HTTP/2 frame header (9 bytes)
// RFC 7540 §4.1:
// +-----------------------------------------------+
// |                 Length (24)                   |
// +---------------+---------------+---------------+
// |   Type (8)    |   Flags (8)   |
// +-+-------------+---------------+-------------------------------+
// |R|                 Stream Identifier (31)                      |
// +=+=============================================================+
type FrameHeader struct {
	Length   uint32    // 24-bit payload length
	Type     FrameType // Frame type
	Flags    Flags     // Frame flags
	StreamID uint32    // 31-bit stream identifier
}

// ParseFrameHeader parses a 9-byte frame header
// This function performs zero allocations - the FrameHeader is returned on the stack
func ParseFrameHeader(b [9]byte) FrameHeader {
	return FrameHeader{
		Length:   uint32(b[0])<<16 | uint32(b[1])<<8 | uint32(b[2]),
		Type:     FrameType(b[3]),
		Flags:    Flags(b[4]),
		StreamID: binary.BigEndian.Uint32(b[5:9]) & 0x7fffffff, // Clear reserved bit
	}
}

// WriteFrameHeader writes a frame header to a 9-byte buffer
// Returns the number of bytes written (always 9)
func WriteFrameHeader(b []byte, fh FrameHeader) int {
	if len(b) < 9 {
		panic("buffer too small for frame header")
	}

	// Write 24-bit length
	b[0] = byte(fh.Length >> 16)
	b[1] = byte(fh.Length >> 8)
	b[2] = byte(fh.Length)

	// Write type and flags
	b[3] = byte(fh.Type)
	b[4] = byte(fh.Flags)

	// Write 31-bit stream ID (clear reserved bit)
	binary.BigEndian.PutUint32(b[5:9], fh.StreamID&0x7fffffff)

	return 9
}

// Validate checks if the frame header is valid according to RFC 7540
func (fh *FrameHeader) Validate() error {
	// Check frame size (RFC 7540 §4.2)
	if fh.Length > MaxFrameSize {
		return ErrFrameTooLarge
	}

	// Validate based on frame type
	switch fh.Type {
	case FrameData:
		return fh.validateData()
	case FrameHeaders:
		return fh.validateHeaders()
	case FramePriority:
		return fh.validatePriority()
	case FrameRSTStream:
		return fh.validateRSTStream()
	case FrameSettings:
		return fh.validateSettings()
	case FramePushPromise:
		return fh.validatePushPromise()
	case FramePing:
		return fh.validatePing()
	case FrameGoAway:
		return fh.validateGoAway()
	case FrameWindowUpdate:
		return fh.validateWindowUpdate()
	case FrameContinuation:
		return fh.validateContinuation()
	default:
		// Unknown frame types are ignored (RFC 7540 §4.1)
		return nil
	}
}

// validateData validates DATA frame header (RFC 7540 §6.1)
func (fh *FrameHeader) validateData() error {
	// DATA frames MUST be associated with a stream
	if fh.StreamID == 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	return nil
}

// validateHeaders validates HEADERS frame header (RFC 7540 §6.2)
func (fh *FrameHeader) validateHeaders() error {
	// HEADERS frames MUST be associated with a stream
	if fh.StreamID == 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	return nil
}

// validatePriority validates PRIORITY frame header (RFC 7540 §6.3)
func (fh *FrameHeader) validatePriority() error {
	// PRIORITY frames MUST be associated with a stream
	if fh.StreamID == 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	// PRIORITY frames are always 5 bytes
	if fh.Length != 5 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}
	return nil
}

// validateRSTStream validates RST_STREAM frame header (RFC 7540 §6.4)
func (fh *FrameHeader) validateRSTStream() error {
	// RST_STREAM frames MUST be associated with a stream
	if fh.StreamID == 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	// RST_STREAM frames are always 4 bytes
	if fh.Length != 4 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}
	return nil
}

// validateSettings validates SETTINGS frame header (RFC 7540 §6.5)
func (fh *FrameHeader) validateSettings() error {
	// SETTINGS frames always apply to a connection, not a stream
	if fh.StreamID != 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	// SETTINGS frame length must be a multiple of 6
	if fh.Length%6 != 0 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}
	// ACK'd SETTINGS frames must have zero length
	if fh.Flags.Has(FlagSettingsAck) && fh.Length != 0 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrSettingsAckWithLength}
	}
	return nil
}

// validatePushPromise validates PUSH_PROMISE frame header (RFC 7540 §6.6)
func (fh *FrameHeader) validatePushPromise() error {
	// PUSH_PROMISE frames MUST be associated with a stream
	if fh.StreamID == 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	// PUSH_PROMISE must have at least 4 bytes (promised stream ID)
	if fh.Length < 4 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}
	return nil
}

// validatePing validates PING frame header (RFC 7540 §6.7)
func (fh *FrameHeader) validatePing() error {
	// PING frames are not associated with any individual stream
	if fh.StreamID != 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	// PING frames must be exactly 8 bytes
	if fh.Length != 8 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}
	return nil
}

// validateGoAway validates GOAWAY frame header (RFC 7540 §6.8)
func (fh *FrameHeader) validateGoAway() error {
	// GOAWAY frames are not associated with any individual stream
	if fh.StreamID != 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	// GOAWAY must have at least 8 bytes (last stream ID + error code)
	if fh.Length < 8 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}
	return nil
}

// validateWindowUpdate validates WINDOW_UPDATE frame header (RFC 7540 §6.9)
func (fh *FrameHeader) validateWindowUpdate() error {
	// WINDOW_UPDATE can be for a connection (stream 0) or a stream
	// No stream ID validation needed

	// WINDOW_UPDATE frames are always 4 bytes
	if fh.Length != 4 {
		return ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}
	return nil
}

// validateContinuation validates CONTINUATION frame header (RFC 7540 §6.10)
func (fh *FrameHeader) validateContinuation() error {
	// CONTINUATION frames MUST be associated with a stream
	if fh.StreamID == 0 {
		return ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidStreamID}
	}
	return nil
}

// Frame is the interface implemented by all frame types
type Frame interface {
	// Header returns the frame header
	Header() FrameHeader

	// Type returns the frame type
	Type() FrameType

	// StreamID returns the stream identifier
	StreamID() uint32
}

// DataFrame represents an HTTP/2 DATA frame (RFC 7540 §6.1)
type DataFrame struct {
	FrameHeader
	Data      []byte // Frame payload data
	PadLength uint8  // Padding length (if PADDED flag set)
}

// Header returns the frame header
func (f *DataFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *DataFrame) Type() FrameType { return FrameData }

// EndStream returns true if END_STREAM flag is set
func (f *DataFrame) EndStream() bool {
	return f.Flags.Has(FlagDataEndStream)
}

// Padded returns true if PADDED flag is set
func (f *DataFrame) Padded() bool {
	return f.Flags.Has(FlagDataPadded)
}

// ParseDataFrame parses a DATA frame from payload
func ParseDataFrame(fh FrameHeader, payload []byte) (*DataFrame, error) {
	df := &DataFrame{
		FrameHeader: fh,
	}

	offset := 0

	// Parse padding length if PADDED flag is set
	if fh.Flags.Has(FlagDataPadded) {
		if len(payload) < 1 {
			return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidPadding}
		}
		df.PadLength = payload[0]
		offset = 1
	}

	// Calculate data length
	dataLen := len(payload) - offset - int(df.PadLength)
	if dataLen < 0 {
		return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidPadding}
	}

	// Zero-copy reference to data
	df.Data = payload[offset : offset+dataLen]

	return df, nil
}

// HeadersFrame represents an HTTP/2 HEADERS frame (RFC 7540 §6.2)
type HeadersFrame struct {
	FrameHeader
	PadLength        uint8  // Padding length (if PADDED flag set)
	StreamDependency uint32 // Stream dependency (if PRIORITY flag set)
	Weight           uint8  // Priority weight (if PRIORITY flag set)
	Exclusive        bool   // Exclusive flag (if PRIORITY flag set)
	HeaderBlock      []byte // Compressed header block
}

// Header returns the frame header
func (f *HeadersFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *HeadersFrame) Type() FrameType { return FrameHeaders }

// EndStream returns true if END_STREAM flag is set
func (f *HeadersFrame) EndStream() bool {
	return f.Flags.Has(FlagHeadersEndStream)
}

// EndHeaders returns true if END_HEADERS flag is set
func (f *HeadersFrame) EndHeaders() bool {
	return f.Flags.Has(FlagHeadersEndHeaders)
}

// HasPriority returns true if PRIORITY flag is set
func (f *HeadersFrame) HasPriority() bool {
	return f.Flags.Has(FlagHeadersPriority)
}

// ParseHeadersFrame parses a HEADERS frame from payload
func ParseHeadersFrame(fh FrameHeader, payload []byte) (*HeadersFrame, error) {
	hf := &HeadersFrame{
		FrameHeader: fh,
	}

	offset := 0

	// Parse padding length if PADDED flag is set
	if fh.Flags.Has(FlagHeadersPadded) {
		if len(payload) < 1 {
			return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidPadding}
		}
		hf.PadLength = payload[0]
		offset = 1
	}

	// Parse priority if PRIORITY flag is set
	if fh.Flags.Has(FlagHeadersPriority) {
		if len(payload) < offset+5 {
			return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidPriority}
		}

		// Read 32-bit stream dependency (E bit + 31-bit stream ID)
		streamDep := binary.BigEndian.Uint32(payload[offset : offset+4])
		hf.Exclusive = (streamDep >> 31) == 1
		hf.StreamDependency = streamDep & 0x7fffffff

		// Read weight
		hf.Weight = payload[offset+4]

		offset += 5
	}

	// Calculate header block length
	headerBlockLen := len(payload) - offset - int(hf.PadLength)
	if headerBlockLen < 0 {
		return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidPadding}
	}

	// Zero-copy reference to header block
	hf.HeaderBlock = payload[offset : offset+headerBlockLen]

	return hf, nil
}

// PriorityFrame represents an HTTP/2 PRIORITY frame (RFC 7540 §6.3)
type PriorityFrame struct {
	FrameHeader
	StreamDependency uint32 // Stream dependency
	Weight           uint8  // Priority weight (1-256, stored as 0-255)
	Exclusive        bool   // Exclusive flag
}

// Header returns the frame header
func (f *PriorityFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *PriorityFrame) Type() FrameType { return FramePriority }

// ParsePriorityFrame parses a PRIORITY frame from payload
func ParsePriorityFrame(fh FrameHeader, payload []byte) (*PriorityFrame, error) {
	if len(payload) != 5 {
		return nil, ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}

	pf := &PriorityFrame{
		FrameHeader: fh,
	}

	// Read 32-bit stream dependency (E bit + 31-bit stream ID)
	streamDep := binary.BigEndian.Uint32(payload[0:4])
	pf.Exclusive = (streamDep >> 31) == 1
	pf.StreamDependency = streamDep & 0x7fffffff

	// Read weight (0-255 represents 1-256)
	pf.Weight = payload[4]

	return pf, nil
}

// RSTStreamFrame represents an HTTP/2 RST_STREAM frame (RFC 7540 §6.4)
type RSTStreamFrame struct {
	FrameHeader
	ErrorCode ErrorCode // Error code
}

// Header returns the frame header
func (f *RSTStreamFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *RSTStreamFrame) Type() FrameType { return FrameRSTStream }

// ParseRSTStreamFrame parses a RST_STREAM frame from payload
func ParseRSTStreamFrame(fh FrameHeader, payload []byte) (*RSTStreamFrame, error) {
	if len(payload) != 4 {
		return nil, ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}

	rf := &RSTStreamFrame{
		FrameHeader: fh,
		ErrorCode:   ErrorCode(binary.BigEndian.Uint32(payload[0:4])),
	}

	return rf, nil
}

// Setting represents a single SETTINGS parameter
type Setting struct {
	ID    SettingID
	Value uint32
}

// SettingsFrame represents an HTTP/2 SETTINGS frame (RFC 7540 §6.5)
type SettingsFrame struct {
	FrameHeader
	Settings []Setting // Settings parameters
}

// Header returns the frame header
func (f *SettingsFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *SettingsFrame) Type() FrameType { return FrameSettings }

// IsAck returns true if ACK flag is set
func (f *SettingsFrame) IsAck() bool {
	return f.Flags.Has(FlagSettingsAck)
}

// ParseSettingsFrame parses a SETTINGS frame from payload
func ParseSettingsFrame(fh FrameHeader, payload []byte) (*SettingsFrame, error) {
	sf := &SettingsFrame{
		FrameHeader: fh,
	}

	// ACK'd SETTINGS must have zero length (already validated in header)
	if fh.Flags.Has(FlagSettingsAck) {
		return sf, nil
	}

	// Parse settings (6 bytes each)
	numSettings := len(payload) / 6
	if numSettings > 0 {
		sf.Settings = make([]Setting, numSettings)
		for i := 0; i < numSettings; i++ {
			offset := i * 6
			sf.Settings[i] = Setting{
				ID:    SettingID(binary.BigEndian.Uint16(payload[offset : offset+2])),
				Value: binary.BigEndian.Uint32(payload[offset+2 : offset+6]),
			}
		}
	}

	return sf, nil
}

// PushPromiseFrame represents an HTTP/2 PUSH_PROMISE frame (RFC 7540 §6.6)
type PushPromiseFrame struct {
	FrameHeader
	PadLength        uint8  // Padding length (if PADDED flag set)
	PromisedStreamID uint32 // Promised stream ID
	HeaderBlock      []byte // Compressed header block
}

// Header returns the frame header
func (f *PushPromiseFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *PushPromiseFrame) Type() FrameType { return FramePushPromise }

// EndHeaders returns true if END_HEADERS flag is set
func (f *PushPromiseFrame) EndHeaders() bool {
	return f.Flags.Has(FlagPushPromiseEndHeaders)
}

// ParsePushPromiseFrame parses a PUSH_PROMISE frame from payload
func ParsePushPromiseFrame(fh FrameHeader, payload []byte) (*PushPromiseFrame, error) {
	ppf := &PushPromiseFrame{
		FrameHeader: fh,
	}

	offset := 0

	// Parse padding length if PADDED flag is set
	if fh.Flags.Has(FlagPushPromisePadded) {
		if len(payload) < 1 {
			return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidPadding}
		}
		ppf.PadLength = payload[0]
		offset = 1
	}

	// Parse promised stream ID (must have at least 4 bytes)
	if len(payload) < offset+4 {
		return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidFrameLength}
	}

	ppf.PromisedStreamID = binary.BigEndian.Uint32(payload[offset:offset+4]) & 0x7fffffff
	offset += 4

	// Calculate header block length
	headerBlockLen := len(payload) - offset - int(ppf.PadLength)
	if headerBlockLen < 0 {
		return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidPadding}
	}

	// Zero-copy reference to header block
	ppf.HeaderBlock = payload[offset : offset+headerBlockLen]

	return ppf, nil
}

// PingFrame represents an HTTP/2 PING frame (RFC 7540 §6.7)
type PingFrame struct {
	FrameHeader
	Data [8]byte // Opaque data
}

// Header returns the frame header
func (f *PingFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *PingFrame) Type() FrameType { return FramePing }

// IsAck returns true if ACK flag is set
func (f *PingFrame) IsAck() bool {
	return f.Flags.Has(FlagPingAck)
}

// ParsePingFrame parses a PING frame from payload
func ParsePingFrame(fh FrameHeader, payload []byte) (*PingFrame, error) {
	if len(payload) != 8 {
		return nil, ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}

	pf := &PingFrame{
		FrameHeader: fh,
	}

	// Copy 8 bytes of data
	copy(pf.Data[:], payload)

	return pf, nil
}

// GoAwayFrame represents an HTTP/2 GOAWAY frame (RFC 7540 §6.8)
type GoAwayFrame struct {
	FrameHeader
	LastStreamID uint32    // Last stream ID
	ErrorCode    ErrorCode // Error code
	DebugData    []byte    // Optional debug data
}

// Header returns the frame header
func (f *GoAwayFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *GoAwayFrame) Type() FrameType { return FrameGoAway }

// ParseGoAwayFrame parses a GOAWAY frame from payload
func ParseGoAwayFrame(fh FrameHeader, payload []byte) (*GoAwayFrame, error) {
	if len(payload) < 8 {
		return nil, ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}

	gaf := &GoAwayFrame{
		FrameHeader:  fh,
		LastStreamID: binary.BigEndian.Uint32(payload[0:4]) & 0x7fffffff,
		ErrorCode:    ErrorCode(binary.BigEndian.Uint32(payload[4:8])),
	}

	// Zero-copy reference to debug data (if any)
	if len(payload) > 8 {
		gaf.DebugData = payload[8:]
	}

	return gaf, nil
}

// WindowUpdateFrame represents an HTTP/2 WINDOW_UPDATE frame (RFC 7540 §6.9)
type WindowUpdateFrame struct {
	FrameHeader
	WindowSizeIncrement uint32 // Window size increment
}

// Header returns the frame header
func (f *WindowUpdateFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *WindowUpdateFrame) Type() FrameType { return FrameWindowUpdate }

// ParseWindowUpdateFrame parses a WINDOW_UPDATE frame from payload
func ParseWindowUpdateFrame(fh FrameHeader, payload []byte) (*WindowUpdateFrame, error) {
	if len(payload) != 4 {
		return nil, ConnectionError{Code: ErrCodeFrameSize, Err: ErrInvalidFrameLength}
	}

	wuf := &WindowUpdateFrame{
		FrameHeader:         fh,
		WindowSizeIncrement: binary.BigEndian.Uint32(payload[0:4]) & 0x7fffffff,
	}

	// Window size increment must not be zero (RFC 7540 §6.9)
	if wuf.WindowSizeIncrement == 0 {
		if fh.StreamID == 0 {
			return nil, ConnectionError{Code: ErrCodeProtocol, Err: ErrInvalidWindowUpdate}
		}
		return nil, StreamError{StreamID: fh.StreamID, Code: ErrCodeProtocol, Err: ErrInvalidWindowUpdate}
	}

	return wuf, nil
}

// ContinuationFrame represents an HTTP/2 CONTINUATION frame (RFC 7540 §6.10)
type ContinuationFrame struct {
	FrameHeader
	HeaderBlock []byte // Compressed header block fragment
}

// Header returns the frame header
func (f *ContinuationFrame) Header() FrameHeader { return f.FrameHeader }

// Type returns the frame type
func (f *ContinuationFrame) Type() FrameType { return FrameContinuation }

// EndHeaders returns true if END_HEADERS flag is set
func (f *ContinuationFrame) EndHeaders() bool {
	return f.Flags.Has(FlagContinuationEndHeaders)
}

// ParseContinuationFrame parses a CONTINUATION frame from payload
func ParseContinuationFrame(fh FrameHeader, payload []byte) (*ContinuationFrame, error) {
	cf := &ContinuationFrame{
		FrameHeader: fh,
		HeaderBlock: payload, // Zero-copy reference
	}

	return cf, nil
}
