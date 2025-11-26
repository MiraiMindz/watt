package quic

import (
	"testing"
	"time"
)

func TestFlowControllerBasic(t *testing.T) {
	fc := NewFlowController(10000, 10000)

	// Test initial state
	sent, received, maxData, peerMaxData := fc.GetConnectionStats()
	if sent != 0 || received != 0 {
		t.Errorf("Initial state: sent=%d, received=%d, want 0, 0", sent, received)
	}
	if maxData != 10000 || peerMaxData != 10000 {
		t.Errorf("Initial limits: maxData=%d, peerMaxData=%d, want 10000, 10000", maxData, peerMaxData)
	}
}

func TestFlowControllerCanSend(t *testing.T) {
	fc := NewFlowController(10000, 10000)

	// Should be able to send within limit
	if !fc.CanSendData(5000) {
		t.Error("CanSendData(5000) = false, want true")
	}

	// Record some data sent
	if err := fc.RecordDataSent(5000); err != nil {
		t.Fatalf("RecordDataSent(5000) error = %v", err)
	}

	// Should be able to send more
	if !fc.CanSendData(4000) {
		t.Error("CanSendData(4000) = false, want true")
	}

	// Should not be able to exceed limit
	if fc.CanSendData(6000) {
		t.Error("CanSendData(6000) = true, want false")
	}
}

func TestFlowControllerSendViolation(t *testing.T) {
	fc := NewFlowController(10000, 10000)

	// Send up to limit
	if err := fc.RecordDataSent(10000); err != nil {
		t.Fatalf("RecordDataSent(10000) error = %v", err)
	}

	// Try to exceed limit
	err := fc.RecordDataSent(1)
	if err != ErrFlowControlViolation {
		t.Errorf("RecordDataSent(1) after limit error = %v, want ErrFlowControlViolation", err)
	}
}

func TestFlowControllerReceive(t *testing.T) {
	fc := NewFlowController(10000, 10000)

	// Receive within limit
	if err := fc.RecordDataReceived(5000); err != nil {
		t.Fatalf("RecordDataReceived(5000) error = %v", err)
	}

	// Check state
	_, received, _, _ := fc.GetConnectionStats()
	if received != 5000 {
		t.Errorf("received = %d, want 5000", received)
	}

	// Receive more
	if err := fc.RecordDataReceived(4000); err != nil {
		t.Fatalf("RecordDataReceived(4000) error = %v", err)
	}

	// Try to exceed limit
	err := fc.RecordDataReceived(2000)
	if err != ErrFlowControlViolation {
		t.Errorf("RecordDataReceived(2000) after limit error = %v, want ErrFlowControlViolation", err)
	}
}

func TestFlowControllerMaxDataUpdate(t *testing.T) {
	fc := NewFlowController(10000, 10000)

	// Receive data
	fc.RecordDataReceived(6000)

	// Should recommend sending MAX_DATA
	if !fc.ShouldSendMaxData() {
		t.Error("ShouldSendMaxData() = false, want true after 60% consumption")
	}

	// Update MAX_DATA
	newMax := fc.UpdateMaxData(10000)
	if newMax != 20000 {
		t.Errorf("UpdateMaxData(10000) = %d, want 20000", newMax)
	}

	// Should not recommend after update
	if fc.ShouldSendMaxData() {
		t.Error("ShouldSendMaxData() = true, want false after update")
	}
}

func TestFlowControllerPeerMaxDataUpdate(t *testing.T) {
	fc := NewFlowController(10000, 5000) // Peer initially allows 5000

	// Can't send more than peer allows
	if fc.CanSendData(6000) {
		t.Error("CanSendData(6000) = true, want false with peerMaxData=5000")
	}

	// Send some data
	fc.RecordDataSent(4000)

	// Peer sends MAX_DATA update
	fc.UpdatePeerMaxData(15000)

	// Now we can send more
	if !fc.CanSendData(10000) {
		t.Error("CanSendData(10000) = false, want true after peer update")
	}
}

func TestFlowControllerBlocking(t *testing.T) {
	fc := NewFlowController(10000, 10000)

	// Not blocked initially
	if fc.IsBlocked() {
		t.Error("IsBlocked() = true, want false initially")
	}

	// Send 90% of limit (should trigger blocked state)
	fc.RecordDataSent(9000)

	// Should be blocked
	if !fc.IsBlocked() {
		t.Error("IsBlocked() = false, want true after 90% sent")
	}

	// Check blocked duration
	time.Sleep(10 * time.Millisecond)
	duration := fc.GetBlockedDuration()
	if duration < 10*time.Millisecond {
		t.Errorf("GetBlockedDuration() = %v, want >= 10ms", duration)
	}

	// Update peer's MAX_DATA should unblock
	fc.UpdatePeerMaxData(20000)
	if fc.IsBlocked() {
		t.Error("IsBlocked() = true, want false after peer update")
	}
}

func TestStreamFlowControllerBasic(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 10000)

	// Test initial state
	sendOffset, sendMax := sfc.GetSendStats()
	recvOffset, recvMax := sfc.GetReceiveStats()

	if sendOffset != 0 || sendMax != 10000 {
		t.Errorf("Initial send: offset=%d, max=%d, want 0, 10000", sendOffset, sendMax)
	}
	if recvOffset != 0 || recvMax != 10000 {
		t.Errorf("Initial recv: offset=%d, max=%d, want 0, 10000", recvOffset, recvMax)
	}
}

func TestStreamFlowControllerSend(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 10000)

	// Can send within limit
	if !sfc.CanSend(5000) {
		t.Error("CanSend(5000) = false, want true")
	}

	// Record sent
	if err := sfc.RecordSent(5000); err != nil {
		t.Fatalf("RecordSent(5000) error = %v", err)
	}

	// Check available
	available := sfc.BytesAvailableToSend()
	if available != 5000 {
		t.Errorf("BytesAvailableToSend() = %d, want 5000", available)
	}

	// Can't exceed limit
	if sfc.CanSend(6000) {
		t.Error("CanSend(6000) = true, want false")
	}
}

func TestStreamFlowControllerReceive(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 10000)

	// Receive within limit
	if err := sfc.RecordReceived(3000); err != nil {
		t.Fatalf("RecordReceived(3000) error = %v", err)
	}

	// Check available
	available := sfc.BytesAvailableToReceive()
	if available != 7000 {
		t.Errorf("BytesAvailableToReceive() = %d, want 7000", available)
	}

	// Should not recommend MAX_STREAM_DATA yet (only 30% consumed)
	if sfc.ShouldSendMaxStreamData() {
		t.Error("ShouldSendMaxStreamData() = true, want false at 30%")
	}

	// Receive more (total 60%)
	sfc.RecordReceived(3000)

	// Should recommend MAX_STREAM_DATA now
	if !sfc.ShouldSendMaxStreamData() {
		t.Error("ShouldSendMaxStreamData() = false, want true at 60%")
	}
}

func TestStreamFlowControllerUpdate(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 5000)

	// Initially can only send 5000
	if !sfc.CanSend(4000) {
		t.Error("CanSend(4000) = false, want true")
	}
	if sfc.CanSend(6000) {
		t.Error("CanSend(6000) = true, want false with limit 5000")
	}

	// Peer sends MAX_STREAM_DATA update
	sfc.UpdatePeerMaxStreamData(15000)

	// Now can send more
	if !sfc.CanSend(10000) {
		t.Error("CanSend(10000) = false, want true after update")
	}
}

func TestStreamFlowControllerAutoTune(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 10000)

	// Receive 80% of window
	sfc.RecordReceived(8000)

	// Auto-tune should increase window
	newWindow := sfc.AutoTuneWindow()
	if newWindow <= 10000 {
		t.Errorf("AutoTuneWindow() = %d, want > 10000", newWindow)
	}

	// Window should have doubled
	if newWindow != 20000 {
		t.Errorf("AutoTuneWindow() = %d, want 20000 (doubled)", newWindow)
	}
}

func TestStreamFlowControllerAutoTuneMax(t *testing.T) {
	sfc := NewStreamFlowController(4, 1000, 1000)

	// Repeatedly trigger auto-tune
	for i := 0; i < 10; i++ {
		_, recvMax := sfc.GetReceiveStats()
		// Consume 80% to trigger auto-tune
		sfc.RecordReceived(uint64(float64(recvMax) * 0.8))
		sfc.AutoTuneWindow()
	}

	// Should cap at 16x initial
	_, finalMax := sfc.GetReceiveStats()
	if finalMax > 16000 {
		t.Errorf("Final max = %d, want <= 16000 (16x initial)", finalMax)
	}
}

func TestStreamFlowControllerAutoTuneDisable(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 10000)

	// Disable auto-tune
	sfc.EnableAutoTune(false)

	// Receive most of window
	sfc.RecordReceived(9000)

	// Auto-tune should not increase window
	initialMax := sfc.recvMaxData
	newWindow := sfc.AutoTuneWindow()
	if newWindow != initialMax {
		t.Errorf("AutoTuneWindow() with disabled = %d, want %d (unchanged)", newWindow, initialMax)
	}
}

func TestStreamFlowControllerThreshold(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 10000)

	// Set threshold to 75%
	sfc.SetWindowUpdateThreshold(0.75)

	// Receive 60% (should not trigger)
	sfc.RecordReceived(6000)
	if sfc.ShouldSendMaxStreamData() {
		t.Error("ShouldSendMaxStreamData() at 60% = true, want false with 75% threshold")
	}

	// Receive to 80% (should trigger)
	sfc.RecordReceived(2000)
	if !sfc.ShouldSendMaxStreamData() {
		t.Error("ShouldSendMaxStreamData() at 80% = false, want true with 75% threshold")
	}
}

func TestStreamFlowControllerViolation(t *testing.T) {
	sfc := NewStreamFlowController(4, 10000, 10000)

	// Send up to limit
	sfc.RecordSent(10000)

	// Try to exceed
	err := sfc.RecordSent(1)
	if err != ErrFlowControlViolation {
		t.Errorf("RecordSent(1) after limit error = %v, want ErrFlowControlViolation", err)
	}

	// Same for receive
	sfc.RecordReceived(10000)
	err = sfc.RecordReceived(1)
	if err != ErrFlowControlViolation {
		t.Errorf("RecordReceived(1) after limit error = %v, want ErrFlowControlViolation", err)
	}
}

func BenchmarkFlowControllerCanSend(b *testing.B) {
	fc := NewFlowController(1000000, 1000000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = fc.CanSendData(1000)
	}
}

func BenchmarkFlowControllerRecordSent(b *testing.B) {
	fc := NewFlowController(uint64(b.N)*1000, uint64(b.N)*1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		fc.RecordDataSent(1000)
	}
}

func BenchmarkStreamFlowControllerCanSend(b *testing.B) {
	sfc := NewStreamFlowController(4, 1000000, 1000000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		_ = sfc.CanSend(1000)
	}
}

func BenchmarkStreamFlowControllerRecordSent(b *testing.B) {
	sfc := NewStreamFlowController(4, uint64(b.N)*1000, uint64(b.N)*1000)

	b.ResetTimer()
	b.ReportAllocs()

	for i := 0; i < b.N; i++ {
		sfc.RecordSent(1000)
	}
}
