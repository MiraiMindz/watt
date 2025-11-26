package http2

import "errors"

// Error codes (RFC 7540 §7)
const (
	ErrCodeNo                 ErrorCode = 0x0
	ErrCodeProtocol           ErrorCode = 0x1
	ErrCodeInternal           ErrorCode = 0x2
	ErrCodeFlowControl        ErrorCode = 0x3
	ErrCodeSettingsTimeout    ErrorCode = 0x4
	ErrCodeStreamClosed       ErrorCode = 0x5
	ErrCodeFrameSize          ErrorCode = 0x6
	ErrCodeRefusedStream      ErrorCode = 0x7
	ErrCodeCancel             ErrorCode = 0x8
	ErrCodeCompression        ErrorCode = 0x9
	ErrCodeConnect            ErrorCode = 0xa
	ErrCodeEnhanceYourCalm    ErrorCode = 0xb
	ErrCodeInadequateSecurity ErrorCode = 0xc
	ErrCodeHTTP11Required     ErrorCode = 0xd
)

// ErrorCode represents an HTTP/2 error code
type ErrorCode uint32

// String returns the string representation of the error code
func (e ErrorCode) String() string {
	switch e {
	case ErrCodeNo:
		return "NO_ERROR"
	case ErrCodeProtocol:
		return "PROTOCOL_ERROR"
	case ErrCodeInternal:
		return "INTERNAL_ERROR"
	case ErrCodeFlowControl:
		return "FLOW_CONTROL_ERROR"
	case ErrCodeSettingsTimeout:
		return "SETTINGS_TIMEOUT"
	case ErrCodeStreamClosed:
		return "STREAM_CLOSED"
	case ErrCodeFrameSize:
		return "FRAME_SIZE_ERROR"
	case ErrCodeRefusedStream:
		return "REFUSED_STREAM"
	case ErrCodeCancel:
		return "CANCEL"
	case ErrCodeCompression:
		return "COMPRESSION_ERROR"
	case ErrCodeConnect:
		return "CONNECT_ERROR"
	case ErrCodeEnhanceYourCalm:
		return "ENHANCE_YOUR_CALM"
	case ErrCodeInadequateSecurity:
		return "INADEQUATE_SECURITY"
	case ErrCodeHTTP11Required:
		return "HTTP_1_1_REQUIRED"
	default:
		return "UNKNOWN_ERROR"
	}
}

// Protocol errors
var (
	ErrInvalidPreface        = errors.New("http2: invalid connection preface")
	ErrFrameTooLarge         = errors.New("http2: frame size exceeds maximum")
	ErrInvalidFrameType      = errors.New("http2: invalid frame type")
	ErrInvalidStreamID       = errors.New("http2: invalid stream ID")
	ErrInvalidPadding        = errors.New("http2: invalid padding")
	ErrInvalidFrameLength    = errors.New("http2: invalid frame length")
	ErrInvalidWindowUpdate   = errors.New("http2: invalid window update")
	ErrInvalidSettings       = errors.New("http2: invalid settings")
	ErrInvalidPriority       = errors.New("http2: invalid priority")
	ErrStreamClosed           = errors.New("http2: stream closed")
	ErrStreamReset            = errors.New("http2: stream reset")
	ErrFlowControlViolation   = errors.New("http2: flow control violation")
	ErrFlowControlOverflow    = errors.New("http2: flow control window overflow")
	ErrSettingsAckWithLength  = errors.New("http2: SETTINGS ACK must have zero length")
	ErrPingAckWithData        = errors.New("http2: PING ACK must echo original data")
	ErrGoAwayStreamID         = errors.New("http2: GOAWAY stream ID must be zero")
	ErrStreamSelfDependency   = errors.New("http2: stream cannot depend on itself")
	ErrPriorityCycleDetected  = errors.New("http2: priority dependency cycle detected")
	ErrBufferSizeExceeded     = errors.New("http2: buffer size limit exceeded")
	ErrRateLimitExceeded      = errors.New("http2: rate limit exceeded")
	ErrIdleTimeout            = errors.New("http2: idle timeout exceeded")
	ErrWindowUnderflow        = errors.New("http2: flow control window underflow")
)

// ConnectionError represents a connection-level error
type ConnectionError struct {
	Code ErrorCode
	Err  error
}

func (e ConnectionError) Error() string {
	if e.Err != nil {
		return "http2: " + e.Code.String() + ": " + e.Err.Error()
	}
	return "http2: " + e.Code.String()
}

// StreamError represents a stream-level error
type StreamError struct {
	StreamID uint32
	Code     ErrorCode
	Err      error
}

func (e StreamError) Error() string {
	if e.Err != nil {
		return "http2: stream " + string(rune(e.StreamID)) + ": " + e.Code.String() + ": " + e.Err.Error()
	}
	return "http2: stream " + string(rune(e.StreamID)) + ": " + e.Code.String()
}
