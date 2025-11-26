package quic

import (
	"math"
	"sync"
	"time"
)

// QUIC Loss Detection and Congestion Control (RFC 9002)
// Implements packet loss detection, retransmission, and PTO (Probe Timeout)

const (
	// RFC 9002 constants
	kPacketThreshold       = 3               // Packets to trigger fast retransmit
	kTimeThreshold         = 9.0 / 8.0       // Time threshold factor (1.125)
	kGranularity           = 1 * time.Millisecond
	kInitialRTT            = 333 * time.Millisecond
	kMaxAckDelay           = 25 * time.Millisecond
	kLossDelayMultiplier   = 9.0 / 8.0
	kProbeTimeoutMultiplier = 3.0
	kMinProbeTimeout       = 1 * time.Second
	kMaxProbeTimeoutCount  = 10
)

// PacketNumberSpace represents the different packet number spaces
type PacketNumberSpace int

const (
	PacketSpaceInitial PacketNumberSpace = iota
	PacketSpaceHandshake
	PacketSpaceApplication
)

// SentPacketInfo tracks information about a sent packet
type SentPacketInfo struct {
	PacketNumber  uint64
	TimesSent     uint64 // Number of times this packet has been sent
	TimeSent      time.Time
	TimeAcked     time.Time
	PacketSize    uint64
	IsAckEliciting bool   // Requires acknowledgment
	InFlight      bool   // Counted towards bytes in flight
	Frames        []FrameType // Frames contained in packet
}

// LossDetector implements QUIC loss detection and recovery
type LossDetector struct {
	// Sent packets tracking
	sentPackets     map[uint64]*SentPacketInfo
	largestAcked    uint64
	largestSent     uint64
	sentPacketsMu   sync.RWMutex

	// Loss detection state
	lossTime        time.Time
	lastAckEliciting time.Time
	timeSinceLastAckEliciting time.Duration

	// PTO state
	ptoCount        uint64
	ptoBase         time.Duration

	// RTT tracking
	latestRTT       time.Duration
	smoothedRTT     time.Duration
	rttVar          time.Duration
	minRTT          time.Duration
	maxAckDelay     time.Duration
	firstRTTSample  bool

	// Statistics
	packetsLost     uint64
	packetsRetransmitted uint64
	ptoEvents       uint64
	statsMu         sync.RWMutex

	// Callbacks
	onPacketLost    func(pkt *SentPacketInfo)
	onPacketAcked   func(pkt *SentPacketInfo)
}

// NewLossDetector creates a new loss detector
func NewLossDetector() *LossDetector {
	return &LossDetector{
		sentPackets:  make(map[uint64]*SentPacketInfo),
		smoothedRTT:  kInitialRTT,
		rttVar:       kInitialRTT / 2,
		minRTT:       time.Hour, // Will be updated
		maxAckDelay:  kMaxAckDelay,
		ptoBase:      kInitialRTT,
	}
}

// SetCallbacks sets loss and ack callbacks
func (ld *LossDetector) SetCallbacks(onLoss, onAck func(pkt *SentPacketInfo)) {
	ld.onPacketLost = onLoss
	ld.onPacketAcked = onAck
}

// OnPacketSent records a sent packet
func (ld *LossDetector) OnPacketSent(pkt *SentPacketInfo) {
	ld.sentPacketsMu.Lock()
	defer ld.sentPacketsMu.Unlock()

	pkt.TimeSent = time.Now()
	ld.sentPackets[pkt.PacketNumber] = pkt

	if pkt.PacketNumber > ld.largestSent {
		ld.largestSent = pkt.PacketNumber
	}

	if pkt.IsAckEliciting {
		ld.lastAckEliciting = pkt.TimeSent
	}

	// Set loss detection timer
	ld.setLossDetectionTimer()
}

// OnAckReceived processes an ACK frame
func (ld *LossDetector) OnAckReceived(largestAcked uint64, ackDelay time.Duration, ackedPackets []uint64, now time.Time) {
	ld.sentPacketsMu.Lock()
	defer ld.sentPacketsMu.Unlock()

	// Ignore ACK for packets we haven't sent
	if largestAcked > ld.largestSent {
		return
	}

	// Update largest acked
	if largestAcked > ld.largestAcked {
		ld.largestAcked = largestAcked
	}

	// Process newly acked packets
	newlyAcked := make([]*SentPacketInfo, 0, len(ackedPackets))
	for _, pn := range ackedPackets {
		if pkt, ok := ld.sentPackets[pn]; ok {
			newlyAcked = append(newlyAcked, pkt)
			pkt.TimeAcked = now
		}
	}

	// If no new packets acked, nothing to do
	if len(newlyAcked) == 0 {
		return
	}

	// Find the largest newly acked packet
	var largestNewlyAcked *SentPacketInfo
	for _, pkt := range newlyAcked {
		if largestNewlyAcked == nil || pkt.PacketNumber > largestNewlyAcked.PacketNumber {
			largestNewlyAcked = pkt
		}
	}

	// Update RTT using largest newly acked packet
	if largestNewlyAcked != nil {
		ld.updateRTT(largestNewlyAcked, ackDelay, now)
	}

	// Process acked packets
	for _, pkt := range newlyAcked {
		// Call callback
		if ld.onPacketAcked != nil {
			ld.onPacketAcked(pkt)
		}

		// Remove from sent packets
		delete(ld.sentPackets, pkt.PacketNumber)
	}

	// Detect and handle losses
	ld.detectLostPackets(now)

	// Reset PTO count on progress
	ld.ptoCount = 0

	// Update loss detection timer
	ld.setLossDetectionTimer()
}

// updateRTT updates RTT estimates
func (ld *LossDetector) updateRTT(pkt *SentPacketInfo, ackDelay time.Duration, now time.Time) {
	// Calculate latest RTT
	ld.latestRTT = now.Sub(pkt.TimeSent)

	// Adjust for ack delay
	if ld.latestRTT > ld.minRTT+ackDelay {
		ld.latestRTT -= ackDelay
	}

	// Update min RTT
	if ld.latestRTT < ld.minRTT {
		ld.minRTT = ld.latestRTT
	}

	// First RTT sample
	if !ld.firstRTTSample {
		ld.smoothedRTT = ld.latestRTT
		ld.rttVar = ld.latestRTT / 2
		ld.firstRTTSample = true
		return
	}

	// EWMA (RFC 6298)
	rttDiff := ld.latestRTT
	if ld.smoothedRTT > rttDiff {
		rttDiff = ld.smoothedRTT - rttDiff
	} else {
		rttDiff = rttDiff - ld.smoothedRTT
	}

	ld.rttVar = (3*ld.rttVar + rttDiff) / 4
	ld.smoothedRTT = (7*ld.smoothedRTT + ld.latestRTT) / 8
}

// detectLostPackets detects lost packets based on packet and time thresholds
func (ld *LossDetector) detectLostPackets(now time.Time) {
	// Packet threshold: packets more than kPacketThreshold behind largest acked
	packetThreshold := ld.largestAcked
	if packetThreshold > kPacketThreshold {
		packetThreshold -= kPacketThreshold
	} else {
		packetThreshold = 0
	}

	// Time threshold: packets sent more than loss delay ago
	lossDelay := time.Duration(float64(ld.smoothedRTT+ld.rttVar) * kLossDelayMultiplier)
	if lossDelay < kGranularity {
		lossDelay = kGranularity
	}
	timeThreshold := now.Add(-lossDelay)

	lostPackets := make([]*SentPacketInfo, 0)

	// Check all unacked packets
	for pn, pkt := range ld.sentPackets {
		// Skip if already acked
		if !pkt.TimeAcked.IsZero() {
			continue
		}

		// Packet threshold: lost if packet number is far enough behind
		if pn < packetThreshold {
			lostPackets = append(lostPackets, pkt)
			continue
		}

		// Time threshold: lost if sent long enough ago
		if pkt.TimeSent.Before(timeThreshold) {
			lostPackets = append(lostPackets, pkt)
			continue
		}
	}

	// Process lost packets
	for _, pkt := range lostPackets {
		ld.onPacketLost_locked(pkt)
	}
}

// onPacketLost_locked processes a lost packet (must hold sentPacketsMu)
func (ld *LossDetector) onPacketLost_locked(pkt *SentPacketInfo) {
	ld.statsMu.Lock()
	ld.packetsLost++
	ld.statsMu.Unlock()

	// Call callback
	if ld.onPacketLost != nil {
		ld.onPacketLost(pkt)
	}

	// Remove from sent packets
	delete(ld.sentPackets, pkt.PacketNumber)
}

// GetLossDetectionTimer returns when the next loss detection event should occur
func (ld *LossDetector) GetLossDetectionTimer() time.Time {
	ld.sentPacketsMu.RLock()
	defer ld.sentPacketsMu.RUnlock()

	// If we have a known loss time, use that
	if !ld.lossTime.IsZero() {
		return ld.lossTime
	}

	// If no ack-eliciting packets in flight, no timer
	if ld.lastAckEliciting.IsZero() {
		return time.Time{}
	}

	// Calculate PTO
	pto := ld.calculatePTO()
	return ld.lastAckEliciting.Add(pto)
}

// calculatePTO calculates the Probe Timeout
func (ld *LossDetector) calculatePTO() time.Duration {
	pto := ld.smoothedRTT + max(4*ld.rttVar, kGranularity) + ld.maxAckDelay

	// Exponential backoff for repeated PTOs
	if ld.ptoCount > 0 {
		multiplier := math.Pow(2, float64(ld.ptoCount))
		pto = time.Duration(float64(pto) * multiplier)
	}

	// Apply minimum
	if pto < kMinProbeTimeout {
		pto = kMinProbeTimeout
	}

	return pto
}

// OnLossDetectionTimeout handles loss detection timer expiry
func (ld *LossDetector) OnLossDetectionTimeout(now time.Time) {
	ld.sentPacketsMu.Lock()
	defer ld.sentPacketsMu.Unlock()

	// Check for time-based loss
	earliestLossTime := time.Time{}
	for _, pkt := range ld.sentPackets {
		if pkt.TimeAcked.IsZero() && pkt.IsAckEliciting {
			lossDelay := time.Duration(float64(ld.smoothedRTT+ld.rttVar) * kLossDelayMultiplier)
			lossTime := pkt.TimeSent.Add(lossDelay)
			if earliestLossTime.IsZero() || lossTime.Before(earliestLossTime) {
				earliestLossTime = lossTime
			}
		}
	}

	if !earliestLossTime.IsZero() && !earliestLossTime.After(now) {
		// Time-based loss detection
		ld.detectLostPackets(now)
		ld.setLossDetectionTimer()
		return
	}

	// PTO: send probe packets
	ld.ptoCount++
	ld.statsMu.Lock()
	ld.ptoEvents++
	ld.statsMu.Unlock()

	// Cap PTO count
	if ld.ptoCount > kMaxProbeTimeoutCount {
		// Connection should be considered dead
		return
	}

	// Reset loss detection timer
	ld.setLossDetectionTimer()
}

// setLossDetectionTimer updates the loss detection timer
func (ld *LossDetector) setLossDetectionTimer() {
	// Find earliest loss time
	earliestLossTime := time.Time{}
	for _, pkt := range ld.sentPackets {
		if pkt.TimeAcked.IsZero() && pkt.IsAckEliciting {
			lossDelay := time.Duration(float64(ld.smoothedRTT+ld.rttVar) * kLossDelayMultiplier)
			lossTime := pkt.TimeSent.Add(lossDelay)
			if earliestLossTime.IsZero() || lossTime.Before(earliestLossTime) {
				earliestLossTime = lossTime
			}
		}
	}

	ld.lossTime = earliestLossTime
}

// GetRTT returns RTT estimates
func (ld *LossDetector) GetRTT() (latest, smoothed, min, variance time.Duration) {
	ld.sentPacketsMu.RLock()
	defer ld.sentPacketsMu.RUnlock()
	return ld.latestRTT, ld.smoothedRTT, ld.minRTT, ld.rttVar
}

// GetInFlightPackets returns the number of packets in flight
func (ld *LossDetector) GetInFlightPackets() int {
	ld.sentPacketsMu.RLock()
	defer ld.sentPacketsMu.RUnlock()

	count := 0
	for _, pkt := range ld.sentPackets {
		if pkt.InFlight && pkt.TimeAcked.IsZero() {
			count++
		}
	}
	return count
}

// GetStatistics returns loss detection statistics
func (ld *LossDetector) GetStatistics() (lost, retransmitted, ptoEvents uint64, ptoCount uint64) {
	ld.statsMu.RLock()
	defer ld.statsMu.RUnlock()
	return ld.packetsLost, ld.packetsRetransmitted, ld.ptoEvents, ld.ptoCount
}

// GetSentPacket returns information about a sent packet
func (ld *LossDetector) GetSentPacket(pn uint64) *SentPacketInfo {
	ld.sentPacketsMu.RLock()
	defer ld.sentPacketsMu.RUnlock()
	return ld.sentPackets[pn]
}

// RemoveSentPacket removes a packet from tracking (for retransmission)
func (ld *LossDetector) RemoveSentPacket(pn uint64) {
	ld.sentPacketsMu.Lock()
	defer ld.sentPacketsMu.Unlock()
	delete(ld.sentPackets, pn)
}

// Reset resets the loss detector
func (ld *LossDetector) Reset() {
	ld.sentPacketsMu.Lock()
	defer ld.sentPacketsMu.Unlock()

	ld.sentPackets = make(map[uint64]*SentPacketInfo)
	ld.largestAcked = 0
	ld.largestSent = 0
	ld.lossTime = time.Time{}
	ld.lastAckEliciting = time.Time{}
	ld.ptoCount = 0
	ld.smoothedRTT = kInitialRTT
	ld.rttVar = kInitialRTT / 2
	ld.minRTT = time.Hour
	ld.firstRTTSample = false

	ld.statsMu.Lock()
	ld.packetsLost = 0
	ld.packetsRetransmitted = 0
	ld.ptoEvents = 0
	ld.statsMu.Unlock()
}

// Helper function
func max(a, b time.Duration) time.Duration {
	if a > b {
		return a
	}
	return b
}
