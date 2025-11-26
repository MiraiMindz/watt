package quic

import (
	"math"
	"sync"
	"time"
)

// QUIC Congestion Control - NewReno (RFC 9002)
// Implements NewReno congestion control algorithm

const (
	// Constants from RFC 9002
	kInitialWindow       = 10 * kMaxDatagramSize
	kMinimumWindow       = 2 * kMaxDatagramSize
	kMaxDatagramSize     = 1200 // Conservative MTU
	kLossReductionFactor = 0.5
	kPersistentCongestionThreshold = 3 // PTOs
)

// CongestionState represents the congestion control state
type CongestionState int

const (
	CongestionStateSlowStart CongestionState = iota
	CongestionStateCongestionAvoidance
	CongestionStateRecovery
)

// CongestionController implements NewReno congestion control
type CongestionController struct {
	// Congestion window
	congestionWindow uint64
	bytesInFlight    uint64
	cwndMu           sync.RWMutex

	// State
	state              CongestionState
	slowStartThreshold uint64

	// Recovery tracking
	recoveryStart time.Time
	inRecovery    bool

	// Statistics
	packetsSent     uint64
	packetsLost     uint64
	packetsAcked    uint64
	bytesAcked      uint64
	minRTT          time.Duration
	smoothedRTT     time.Duration
	rttVar          time.Duration

	// Pacing
	pacingRate  uint64 // bytes per second
	enablePacing bool

	statsMu sync.RWMutex
}

// NewCongestionController creates a new congestion controller
func NewCongestionController() *CongestionController {
	return &CongestionController{
		congestionWindow:   kInitialWindow,
		slowStartThreshold: math.MaxUint64, // Start in slow start
		state:             CongestionStateSlowStart,
		minRTT:            time.Second, // Start with 1s, will be updated
		smoothedRTT:       333 * time.Millisecond, // RFC 9002 initial value
		rttVar:            166 * time.Millisecond, // Half of smoothedRTT
		enablePacing:      true,
	}
}

// CanSend checks if we can send n bytes given congestion window
func (cc *CongestionController) CanSend(n uint64) bool {
	cc.cwndMu.RLock()
	defer cc.cwndMu.RUnlock()

	return cc.bytesInFlight+n <= cc.congestionWindow
}

// OnPacketSent records a packet being sent
func (cc *CongestionController) OnPacketSent(packetSize uint64, now time.Time) {
	cc.cwndMu.Lock()
	cc.bytesInFlight += packetSize
	cc.cwndMu.Unlock()

	cc.statsMu.Lock()
	cc.packetsSent++
	cc.statsMu.Unlock()
}

// OnPacketAcked handles packet acknowledgment
func (cc *CongestionController) OnPacketAcked(packetSize uint64, rtt time.Duration, now time.Time) {
	cc.cwndMu.Lock()
	defer cc.cwndMu.Unlock()

	// Update bytes in flight
	if cc.bytesInFlight >= packetSize {
		cc.bytesInFlight -= packetSize
	} else {
		cc.bytesInFlight = 0
	}

	// Update RTT estimates
	cc.updateRTT(rtt)

	// Update statistics
	cc.statsMu.Lock()
	cc.packetsAcked++
	cc.bytesAcked += packetSize
	cc.statsMu.Unlock()

	// Don't increase cwnd if in recovery and packet was sent before recovery
	if cc.inRecovery && now.Before(cc.recoveryStart) {
		return
	}

	// Update congestion window based on state
	switch cc.state {
	case CongestionStateSlowStart:
		// Slow start: increase cwnd by packet size
		cc.congestionWindow += packetSize

		// Check if we should exit slow start
		if cc.congestionWindow >= cc.slowStartThreshold {
			cc.state = CongestionStateCongestionAvoidance
		}

	case CongestionStateCongestionAvoidance:
		// Congestion avoidance: increase cwnd by (MSS * MSS / cwnd)
		increase := (kMaxDatagramSize * packetSize) / cc.congestionWindow
		if increase == 0 {
			increase = 1 // Minimum increase
		}
		cc.congestionWindow += increase

	case CongestionStateRecovery:
		// In recovery, don't increase window
		// Exit recovery when all packets sent before recovery are acked
		if now.After(cc.recoveryStart) {
			cc.inRecovery = false
			cc.state = CongestionStateCongestionAvoidance
		}
	}

	// Update pacing rate
	cc.updatePacingRate()
}

// OnPacketLost handles packet loss
func (cc *CongestionController) OnPacketLost(packetSize uint64, now time.Time) {
	cc.cwndMu.Lock()
	defer cc.cwndMu.Unlock()

	// Update bytes in flight
	if cc.bytesInFlight >= packetSize {
		cc.bytesInFlight -= packetSize
	} else {
		cc.bytesInFlight = 0
	}

	// Update statistics
	cc.statsMu.Lock()
	cc.packetsLost++
	cc.statsMu.Unlock()

	// If already in recovery, don't reduce window again
	if cc.inRecovery {
		return
	}

	// Enter recovery
	cc.inRecovery = true
	cc.recoveryStart = now

	// NewReno: reduce cwnd by half, set ssthresh
	cc.slowStartThreshold = cc.congestionWindow / 2
	if cc.slowStartThreshold < kMinimumWindow {
		cc.slowStartThreshold = kMinimumWindow
	}

	cc.congestionWindow = cc.slowStartThreshold
	cc.state = CongestionStateRecovery

	// Update pacing rate
	cc.updatePacingRate()
}

// OnCongestionEvent handles multiple losses (persistent congestion)
func (cc *CongestionController) OnCongestionEvent(lostPackets uint64, now time.Time) {
	if lostPackets == 0 {
		return
	}

	cc.cwndMu.Lock()
	defer cc.cwndMu.Unlock()

	// Check for persistent congestion
	// If we've lost multiple PTOs worth of packets, reset to initial window
	if lostPackets >= kPersistentCongestionThreshold {
		cc.congestionWindow = kMinimumWindow
		cc.slowStartThreshold = kMinimumWindow
		cc.state = CongestionStateSlowStart
		cc.inRecovery = false
	}
}

// updateRTT updates RTT estimates using EWMA
func (cc *CongestionController) updateRTT(rtt time.Duration) {
	// Update min RTT
	if rtt < cc.minRTT {
		cc.minRTT = rtt
	}

	// First RTT sample
	if cc.smoothedRTT == 0 {
		cc.smoothedRTT = rtt
		cc.rttVar = rtt / 2
		return
	}

	// Exponentially weighted moving average (RFC 6298)
	// rttvar = 3/4 * rttvar + 1/4 * |smoothed_rtt - rtt|
	diff := cc.smoothedRTT - rtt
	if diff < 0 {
		diff = -diff
	}
	cc.rttVar = (3*cc.rttVar + diff) / 4

	// smoothed_rtt = 7/8 * smoothed_rtt + 1/8 * rtt
	cc.smoothedRTT = (7*cc.smoothedRTT + rtt) / 8
}

// updatePacingRate calculates pacing rate based on cwnd and RTT
func (cc *CongestionController) updatePacingRate() {
	if !cc.enablePacing {
		return
	}

	// Pacing rate = cwnd / smoothed_rtt
	if cc.smoothedRTT > 0 {
		// Convert to bytes per second
		rttSeconds := float64(cc.smoothedRTT) / float64(time.Second)
		cc.pacingRate = uint64(float64(cc.congestionWindow) / rttSeconds)
	} else {
		// Default to unlimited if no RTT estimate
		cc.pacingRate = math.MaxUint64
	}
}

// GetCongestionWindow returns the current congestion window
func (cc *CongestionController) GetCongestionWindow() uint64 {
	cc.cwndMu.RLock()
	defer cc.cwndMu.RUnlock()
	return cc.congestionWindow
}

// GetBytesInFlight returns current bytes in flight
func (cc *CongestionController) GetBytesInFlight() uint64 {
	cc.cwndMu.RLock()
	defer cc.cwndMu.RUnlock()
	return cc.bytesInFlight
}

// GetSlowStartThreshold returns the slow start threshold
func (cc *CongestionController) GetSlowStartThreshold() uint64 {
	cc.cwndMu.RLock()
	defer cc.cwndMu.RUnlock()
	return cc.slowStartThreshold
}

// GetState returns the current congestion control state
func (cc *CongestionController) GetState() CongestionState {
	cc.cwndMu.RLock()
	defer cc.cwndMu.RUnlock()
	return cc.state
}

// GetRTT returns RTT estimates
func (cc *CongestionController) GetRTT() (min, smoothed, variance time.Duration) {
	cc.cwndMu.RLock()
	defer cc.cwndMu.RUnlock()
	return cc.minRTT, cc.smoothedRTT, cc.rttVar
}

// GetPacingRate returns the current pacing rate in bytes/second
func (cc *CongestionController) GetPacingRate() uint64 {
	cc.cwndMu.RLock()
	defer cc.cwndMu.RUnlock()
	return cc.pacingRate
}

// GetStatistics returns congestion control statistics
func (cc *CongestionController) GetStatistics() (sent, acked, lost uint64, lossRate float64) {
	cc.statsMu.RLock()
	defer cc.statsMu.RUnlock()

	sent = cc.packetsSent
	acked = cc.packetsAcked
	lost = cc.packetsLost

	if sent > 0 {
		lossRate = float64(lost) / float64(sent)
	}

	return
}

// EnablePacing enables or disables pacing
func (cc *CongestionController) EnablePacing(enable bool) {
	cc.cwndMu.Lock()
	defer cc.cwndMu.Unlock()
	cc.enablePacing = enable
}

// SetCongestionWindow manually sets the congestion window (for testing)
func (cc *CongestionController) SetCongestionWindow(cwnd uint64) {
	cc.cwndMu.Lock()
	defer cc.cwndMu.Unlock()

	if cwnd < kMinimumWindow {
		cwnd = kMinimumWindow
	}
	cc.congestionWindow = cwnd
	cc.updatePacingRate()
}

// SetSlowStartThreshold manually sets ssthresh (for testing)
func (cc *CongestionController) SetSlowStartThreshold(ssthresh uint64) {
	cc.cwndMu.Lock()
	defer cc.cwndMu.Unlock()

	cc.slowStartThreshold = ssthresh
}

// Reset resets the congestion controller to initial state
func (cc *CongestionController) Reset() {
	cc.cwndMu.Lock()
	defer cc.cwndMu.Unlock()

	cc.congestionWindow = kInitialWindow
	cc.bytesInFlight = 0
	cc.slowStartThreshold = math.MaxUint64
	cc.state = CongestionStateSlowStart
	cc.inRecovery = false

	cc.statsMu.Lock()
	cc.packetsSent = 0
	cc.packetsLost = 0
	cc.packetsAcked = 0
	cc.bytesAcked = 0
	cc.statsMu.Unlock()

	cc.updatePacingRate()
}
