package http2

// Frame size limits (RFC 7540 §4.2)
const (
	// MaxFrameSize is the maximum allowed frame payload size (16,777,215 bytes)
	MaxFrameSize = 1<<24 - 1 // 16,777,215 bytes

	// DefaultMaxFrameSize is the default maximum frame size (16 KB)
	DefaultMaxFrameSize = 16384 // 16 KB

	// MinMaxFrameSize is the minimum allowed max frame size
	MinMaxFrameSize = 16384 // 16 KB

	// FrameHeaderLen is the length of the HTTP/2 frame header
	FrameHeaderLen = 9
)

// Window size limits (RFC 7540 §6.9.1)
const (
	// MaxWindowSize is the maximum window size (2^31-1)
	MaxWindowSize = 1<<31 - 1 // 2,147,483,647 bytes

	// DefaultWindowSize is the default initial window size
	DefaultWindowSize = 65535 // 64 KB - 1

	// ConnectionStreamID is the stream ID for connection-level frames
	ConnectionStreamID = 0
)

// Settings IDs (RFC 7540 §6.5.2)
const (
	SettingHeaderTableSize      SettingID = 0x1
	SettingEnablePush           SettingID = 0x2
	SettingMaxConcurrentStreams SettingID = 0x3
	SettingInitialWindowSize    SettingID = 0x4
	SettingMaxFrameSize         SettingID = 0x5
	SettingMaxHeaderListSize    SettingID = 0x6
)

// SettingID represents a setting identifier
type SettingID uint16

// Default setting values
const (
	DefaultHeaderTableSize      = 4096
	DefaultEnablePush           = 1
	DefaultMaxConcurrentStreams = 100
)

// Pre-compiled connection preface (RFC 7540 §3.5)
var (
	// ClientPreface is the magic string sent by clients: "PRI * HTTP/2.0\r\n\r\nSM\r\n\r\n"
	ClientPreface = []byte{
		0x50, 0x52, 0x49, 0x20, 0x2a, 0x20, 0x48, 0x54,
		0x54, 0x50, 0x2f, 0x32, 0x2e, 0x30, 0x0d, 0x0a,
		0x0d, 0x0a, 0x53, 0x4d, 0x0d, 0x0a, 0x0d, 0x0a,
	}
)

// Maximum values
const (
	MaxStreamID = 1<<31 - 1 // Maximum stream ID (2^31-1)
	MaxPadding  = 255       // Maximum padding length
)
