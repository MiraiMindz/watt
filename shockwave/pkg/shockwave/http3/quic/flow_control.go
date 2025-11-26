package quic

import (
	"errors"
	"sync"
	"time"
)

// QUIC Flow Control (RFC 9000 Section 4)
// Implements connection-level and stream-level flow control

var (
	ErrFlowControlViolation = errors.New("quic: flow control violation")
	ErrStreamBlocked        = errors.New("quic: stream blocked by flow control")
	ErrConnectionBlocked    = errors.New("quic: connection blocked by flow control")
)

// FlowController manages flow control for connection and streams
type FlowController struct {
	// Connection-level flow control
	connMaxData      uint64
	connDataSent     uint64
	connDataReceived uint64
	peerConnMaxData  uint64
	connMu           sync.RWMutex

	// Auto-tuning
	lastUpdateTime time.Time
	windowUpdateThreshold float64 // Fraction of window before sending update

	// Blocked tracking
	blockedAt time.Time
	isBlocked bool
}

// NewFlowController creates a new flow controller
func NewFlowController(initialMaxData uint64, peerMaxData uint64) *FlowController {
	return &FlowController{
		connMaxData:           initialMaxData,
		peerConnMaxData:       peerMaxData,
		lastUpdateTime:        time.Now(),
		windowUpdateThreshold: 0.5, // Update when 50% consumed
	}
}

// CanSendData checks if we can send n bytes at connection level
func (fc *FlowController) CanSendData(n uint64) bool {
	fc.connMu.RLock()
	defer fc.connMu.RUnlock()

	return fc.connDataSent+n <= fc.peerConnMaxData
}

// RecordDataSent records bytes sent at connection level
func (fc *FlowController) RecordDataSent(n uint64) error {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	newTotal := fc.connDataSent + n
	if newTotal > fc.peerConnMaxData {
		return ErrFlowControlViolation
	}

	fc.connDataSent = newTotal

	// Check if we're getting close to the limit
	if float64(fc.connDataSent) >= float64(fc.peerConnMaxData)*0.9 {
		fc.isBlocked = true
		fc.blockedAt = time.Now()
	}

	return nil
}

// RecordDataReceived records bytes received at connection level
func (fc *FlowController) RecordDataReceived(n uint64) error {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	newTotal := fc.connDataReceived + n
	if newTotal > fc.connMaxData {
		return ErrFlowControlViolation
	}

	fc.connDataReceived = newTotal
	return nil
}

// ShouldSendMaxData checks if we should send MAX_DATA frame
func (fc *FlowController) ShouldSendMaxData() bool {
	fc.connMu.RLock()
	defer fc.connMu.RUnlock()

	consumed := float64(fc.connDataReceived) / float64(fc.connMaxData)
	return consumed >= fc.windowUpdateThreshold
}

// UpdateMaxData updates our connection-level MAX_DATA
func (fc *FlowController) UpdateMaxData(increment uint64) uint64 {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	fc.connMaxData += increment
	fc.lastUpdateTime = time.Now()
	return fc.connMaxData
}

// UpdatePeerMaxData updates peer's MAX_DATA limit
func (fc *FlowController) UpdatePeerMaxData(maxData uint64) {
	fc.connMu.Lock()
	defer fc.connMu.Unlock()

	if maxData > fc.peerConnMaxData {
		fc.peerConnMaxData = maxData
		// Unblock if we were blocked
		if fc.isBlocked {
			fc.isBlocked = false
		}
	}
}

// GetConnectionStats returns current connection flow control stats
func (fc *FlowController) GetConnectionStats() (sent, received, maxData, peerMaxData uint64) {
	fc.connMu.RLock()
	defer fc.connMu.RUnlock()

	return fc.connDataSent, fc.connDataReceived, fc.connMaxData, fc.peerConnMaxData
}

// IsBlocked returns whether the connection is blocked by flow control
func (fc *FlowController) IsBlocked() bool {
	fc.connMu.RLock()
	defer fc.connMu.RUnlock()
	return fc.isBlocked
}

// GetBlockedDuration returns how long we've been blocked
func (fc *FlowController) GetBlockedDuration() time.Duration {
	fc.connMu.RLock()
	defer fc.connMu.RUnlock()

	if !fc.isBlocked {
		return 0
	}
	return time.Since(fc.blockedAt)
}

// StreamFlowController manages flow control for a single stream
type StreamFlowController struct {
	streamID uint64

	// Send flow control (data we send to peer)
	sendMaxData    uint64 // Peer's MAX_STREAM_DATA limit
	sendOffset     uint64 // How much we've sent
	sendMu         sync.RWMutex

	// Receive flow control (data we receive from peer)
	recvMaxData    uint64 // Our MAX_STREAM_DATA limit
	recvOffset     uint64 // How much we've received
	recvMu         sync.RWMutex

	// Auto-tuning
	windowUpdateThreshold float64
	autoTuneEnabled       bool
	initialWindow         uint64
}

// NewStreamFlowController creates a flow controller for a stream
func NewStreamFlowController(streamID uint64, initialMaxData uint64, peerMaxData uint64) *StreamFlowController {
	return &StreamFlowController{
		streamID:              streamID,
		sendMaxData:           peerMaxData,
		recvMaxData:           initialMaxData,
		windowUpdateThreshold: 0.5,
		autoTuneEnabled:       true,
		initialWindow:         initialMaxData,
	}
}

// CanSend checks if we can send n bytes on this stream
func (sfc *StreamFlowController) CanSend(n uint64) bool {
	sfc.sendMu.RLock()
	defer sfc.sendMu.RUnlock()

	return sfc.sendOffset+n <= sfc.sendMaxData
}

// RecordSent records bytes sent on this stream
func (sfc *StreamFlowController) RecordSent(n uint64) error {
	sfc.sendMu.Lock()
	defer sfc.sendMu.Unlock()

	newOffset := sfc.sendOffset + n
	if newOffset > sfc.sendMaxData {
		return ErrFlowControlViolation
	}

	sfc.sendOffset = newOffset
	return nil
}

// RecordReceived records bytes received on this stream
func (sfc *StreamFlowController) RecordReceived(n uint64) error {
	sfc.recvMu.Lock()
	defer sfc.recvMu.Unlock()

	newOffset := sfc.recvOffset + n
	if newOffset > sfc.recvMaxData {
		return ErrFlowControlViolation
	}

	sfc.recvOffset = newOffset
	return nil
}

// ShouldSendMaxStreamData checks if we should send MAX_STREAM_DATA
func (sfc *StreamFlowController) ShouldSendMaxStreamData() bool {
	sfc.recvMu.RLock()
	defer sfc.recvMu.RUnlock()

	if sfc.recvMaxData == 0 {
		return false
	}

	consumed := float64(sfc.recvOffset) / float64(sfc.recvMaxData)
	return consumed >= sfc.windowUpdateThreshold
}

// UpdateMaxStreamData updates our MAX_STREAM_DATA limit
func (sfc *StreamFlowController) UpdateMaxStreamData(increment uint64) uint64 {
	sfc.recvMu.Lock()
	defer sfc.recvMu.Unlock()

	sfc.recvMaxData += increment
	return sfc.recvMaxData
}

// UpdatePeerMaxStreamData updates peer's MAX_STREAM_DATA limit
func (sfc *StreamFlowController) UpdatePeerMaxStreamData(maxData uint64) {
	sfc.sendMu.Lock()
	defer sfc.sendMu.Unlock()

	if maxData > sfc.sendMaxData {
		sfc.sendMaxData = maxData
	}
}

// GetSendStats returns send flow control stats
func (sfc *StreamFlowController) GetSendStats() (offset, maxData uint64) {
	sfc.sendMu.RLock()
	defer sfc.sendMu.RUnlock()
	return sfc.sendOffset, sfc.sendMaxData
}

// GetReceiveStats returns receive flow control stats
func (sfc *StreamFlowController) GetReceiveStats() (offset, maxData uint64) {
	sfc.recvMu.RLock()
	defer sfc.recvMu.RUnlock()
	return sfc.recvOffset, sfc.recvMaxData
}

// BytesAvailableToSend returns how many bytes can be sent
func (sfc *StreamFlowController) BytesAvailableToSend() uint64 {
	sfc.sendMu.RLock()
	defer sfc.sendMu.RUnlock()

	if sfc.sendOffset >= sfc.sendMaxData {
		return 0
	}
	return sfc.sendMaxData - sfc.sendOffset
}

// BytesAvailableToReceive returns how many bytes can be received
func (sfc *StreamFlowController) BytesAvailableToReceive() uint64 {
	sfc.recvMu.RLock()
	defer sfc.recvMu.RUnlock()

	if sfc.recvOffset >= sfc.recvMaxData {
		return 0
	}
	return sfc.recvMaxData - sfc.recvOffset
}

// AutoTuneWindow adjusts the receive window based on usage patterns
func (sfc *StreamFlowController) AutoTuneWindow() uint64 {
	if !sfc.autoTuneEnabled {
		return sfc.recvMaxData
	}

	sfc.recvMu.Lock()
	defer sfc.recvMu.Unlock()

	// If we're consuming the window quickly, increase it
	consumed := float64(sfc.recvOffset) / float64(sfc.recvMaxData)
	if consumed > 0.75 {
		// Double the window, up to a maximum
		newWindow := sfc.recvMaxData * 2
		maxWindow := sfc.initialWindow * 16 // Max 16x initial
		if newWindow > maxWindow {
			newWindow = maxWindow
		}
		sfc.recvMaxData = newWindow
	}

	return sfc.recvMaxData
}

// EnableAutoTune enables/disables automatic window tuning
func (sfc *StreamFlowController) EnableAutoTune(enable bool) {
	sfc.recvMu.Lock()
	defer sfc.recvMu.Unlock()
	sfc.autoTuneEnabled = enable
}

// SetWindowUpdateThreshold sets the threshold for sending updates
func (sfc *StreamFlowController) SetWindowUpdateThreshold(threshold float64) {
	if threshold < 0.1 || threshold > 0.9 {
		return // Ignore invalid thresholds
	}

	sfc.recvMu.Lock()
	defer sfc.recvMu.Unlock()
	sfc.windowUpdateThreshold = threshold
}
