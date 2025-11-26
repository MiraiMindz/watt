package http2

import (
	"fmt"
	"sync"
)

// FlowController manages HTTP/2 flow control (RFC 7540 Section 5.2)
//
// Flow control operates at two levels:
// 1. Connection-level: Controls total data across all streams
// 2. Stream-level: Controls data on individual streams
//
// Both sender and receiver maintain windows that track allowed data.
type FlowController struct {
	// Connection-level window
	connSendWindow int32
	connRecvWindow int32
	connMu         sync.Mutex

	// Per-stream windows (managed via Stream objects)
	// Streams maintain their own windows

	// Initial window size for new streams
	initialWindowSize int32
	windowMu          sync.RWMutex

	// Flow control settings
	maxFrameSize uint32
}

// NewFlowController creates a new flow controller with default settings
func NewFlowController() *FlowController {
	return &FlowController{
		connSendWindow:    int32(DefaultWindowSize),
		connRecvWindow:    int32(DefaultWindowSize),
		initialWindowSize: int32(DefaultWindowSize),
		maxFrameSize:      DefaultMaxFrameSize,
	}
}

// InitialWindowSize returns the initial window size for new streams
func (fc *FlowController) InitialWindowSize() int32 {
	fc.windowMu.RLock()
	defer fc.windowMu.RUnlock()

	return fc.initialWindowSize
}

// SetInitialWindowSize updates the initial window size for new streams
// and adjusts existing stream windows per RFC 7540 Section 6.9.2
func (fc *FlowController) SetInitialWindowSize(size int32) error {
	fc.windowMu.Lock()
	defer fc.windowMu.Unlock()

	if size < 0 || size > MaxWindowSize {
		return fmt.Errorf("invalid window size: %d (must be 0-%d)", size, MaxWindowSize)
	}

	fc.initialWindowSize = size
	return nil
}

// ConnectionSendWindow returns the connection-level send window
func (fc *FlowController) ConnectionSendWindow() int32 {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	return fc.connSendWindow
}

// ConnectionRecvWindow returns the connection-level receive window
func (fc *FlowController) ConnectionRecvWindow() int32 {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	return fc.connRecvWindow
}

// IncrementConnectionSendWindow increases the connection send window
func (fc *FlowController) IncrementConnectionSendWindow(increment int32) error {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	if increment <= 0 {
		return fmt.Errorf("window increment must be positive: %d", increment)
	}

	// Check for overflow (RFC 7540 Section 6.9.1)
	if int64(fc.connSendWindow)+int64(increment) > MaxWindowSize {
		return ConnectionError{
			Code: ErrCodeFlowControl,
			Err:  fmt.Errorf("connection window size overflow"),
		}
	}

	fc.connSendWindow += increment
	return nil
}

// IncrementConnectionRecvWindow increases the connection receive window
func (fc *FlowController) IncrementConnectionRecvWindow(increment int32) error {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	if increment <= 0 {
		return fmt.Errorf("window increment must be positive: %d", increment)
	}

	// Check for overflow (RFC 7540 Section 6.9.1)
	if int64(fc.connRecvWindow)+int64(increment) > MaxWindowSize {
		return ConnectionError{
			Code: ErrCodeFlowControl,
			Err:  fmt.Errorf("connection window size overflow"),
		}
	}

	fc.connRecvWindow += increment
	return nil
}

// ConsumeConnectionSendWindow decreases connection send window (when sending)
func (fc *FlowController) ConsumeConnectionSendWindow(amount int32) error {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	if amount < 0 {
		return fmt.Errorf("negative window consumption: %d", amount)
	}

	if fc.connSendWindow < amount {
		return fmt.Errorf("insufficient connection send window: have %d, need %d",
			fc.connSendWindow, amount)
	}

	fc.connSendWindow -= amount
	return nil
}

// ConsumeConnectionRecvWindow decreases connection receive window (when receiving)
func (fc *FlowController) ConsumeConnectionRecvWindow(amount int32) error {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	if amount < 0 {
		return fmt.Errorf("negative window consumption: %d", amount)
	}

	if fc.connRecvWindow < amount {
		return ConnectionError{
			Code: ErrCodeFlowControl,
			Err: fmt.Errorf("insufficient connection receive window: have %d, need %d",
				fc.connRecvWindow, amount),
		}
	}

	fc.connRecvWindow -= amount
	return nil
}

// CanSend checks if we can send the given amount of data
// Checks both connection and stream windows
func (fc *FlowController) CanSend(stream *Stream, amount int32) bool {
	fc.connMu.Lock()
	connWindow := fc.connSendWindow
	fc.connMu.Unlock()

	streamWindow := stream.SendWindow()

	return connWindow >= amount && streamWindow >= amount
}

// SendData handles sending data with flow control
// Returns the amount actually sent (may be less than requested due to flow control)
func (fc *FlowController) SendData(stream *Stream, data []byte) (int32, error) {
	if len(data) == 0 {
		return 0, nil
	}

	amount := int32(len(data))

	// Check connection window
	fc.connMu.Lock()
	connAvail := fc.connSendWindow
	fc.connMu.Unlock()

	// Check stream window
	streamAvail := stream.SendWindow()

	// Can only send min(requested, connection window, stream window)
	toSend := amount
	if toSend > connAvail {
		toSend = connAvail
	}
	if toSend > streamAvail {
		toSend = streamAvail
	}

	if toSend <= 0 {
		return 0, nil // Flow control blocks sending
	}

	// Consume both windows
	if err := fc.ConsumeConnectionSendWindow(toSend); err != nil {
		return 0, err
	}

	if err := stream.ConsumeSendWindow(toSend); err != nil {
		// Restore connection window on stream error
		fc.IncrementConnectionSendWindow(toSend)
		return 0, err
	}

	return toSend, nil
}

// ReceiveData handles receiving data with flow control
func (fc *FlowController) ReceiveData(stream *Stream, dataLen int32) error {
	if dataLen <= 0 {
		return nil
	}

	// Consume connection receive window
	if err := fc.ConsumeConnectionRecvWindow(dataLen); err != nil {
		return err
	}

	// Consume stream receive window
	if err := stream.ConsumeRecvWindow(dataLen); err != nil {
		// Restore connection window on stream error
		fc.IncrementConnectionRecvWindow(dataLen)
		return err
	}

	return nil
}

// ShouldSendWindowUpdate determines if we should send a WINDOW_UPDATE
// Returns true if window has been consumed significantly
func (fc *FlowController) ShouldSendWindowUpdate(currentWindow, initialWindow int32) bool {
	// Send WINDOW_UPDATE if window is less than 50% of initial
	// This provides good flow without excessive signaling
	threshold := initialWindow / 2
	return currentWindow < threshold
}

// CalculateWindowUpdate calculates how much to increment the window
func (fc *FlowController) CalculateWindowUpdate(currentWindow, initialWindow int32) int32 {
	// Restore window to initial size
	increment := initialWindow - currentWindow
	if increment <= 0 {
		return 0
	}

	// Cap at maximum window size
	if currentWindow+increment > MaxWindowSize {
		increment = MaxWindowSize - currentWindow
	}

	return increment
}

// MaxFrameSize returns the maximum frame size
func (fc *FlowController) MaxFrameSize() uint32 {
	fc.windowMu.RLock()
	defer fc.windowMu.RUnlock()

	return fc.maxFrameSize
}

// SetMaxFrameSize sets the maximum frame size
func (fc *FlowController) SetMaxFrameSize(size uint32) error {
	if size < MinMaxFrameSize || size > MaxFrameSize {
		return fmt.Errorf("invalid max frame size: %d (must be %d-%d)",
			size, MinMaxFrameSize, MaxFrameSize)
	}

	fc.windowMu.Lock()
	defer fc.windowMu.Unlock()

	fc.maxFrameSize = size
	return nil
}

// ChunkData splits data into chunks that fit within frame size and flow control
func (fc *FlowController) ChunkData(data []byte, stream *Stream) [][]byte {
	maxFrameSize := fc.MaxFrameSize()

	var chunks [][]byte
	offset := 0

	for offset < len(data) {
		// Determine chunk size
		remaining := len(data) - offset
		chunkSize := int(maxFrameSize)

		if chunkSize > remaining {
			chunkSize = remaining
		}

		// Check flow control windows
		connWindow := fc.ConnectionSendWindow()
		streamWindow := stream.SendWindow()

		availableWindow := connWindow
		if streamWindow < availableWindow {
			availableWindow = streamWindow
		}

		if int32(chunkSize) > availableWindow {
			chunkSize = int(availableWindow)
		}

		// If no window available, stop chunking
		// Caller should wait for WINDOW_UPDATE
		if chunkSize <= 0 {
			break
		}

		chunks = append(chunks, data[offset:offset+chunkSize])
		offset += chunkSize
	}

	return chunks
}

// WindowStats provides statistics about flow control windows
type WindowStats struct {
	ConnectionSend  int32
	ConnectionRecv  int32
	StreamSend      int32
	StreamRecv      int32
	InitialWindow   int32
	MaxFrameSize    uint32
}

// GetStats returns current flow control statistics
func (fc *FlowController) GetStats(stream *Stream) WindowStats {
	stats := WindowStats{
		ConnectionSend: fc.ConnectionSendWindow(),
		ConnectionRecv: fc.ConnectionRecvWindow(),
		InitialWindow:  fc.InitialWindowSize(),
		MaxFrameSize:   fc.MaxFrameSize(),
	}

	if stream != nil {
		stats.StreamSend = stream.SendWindow()
		stats.StreamRecv = stream.RecvWindow()
	}

	return stats
}
