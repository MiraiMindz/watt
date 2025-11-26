package shockwave

import (
	"runtime"
	"testing"
	"time"
)

// TestLoadTest_Short runs a short load test for validation
func TestLoadTest_Short(t *testing.T) {
	config := LoadTestConfig{
		Duration:     2 * time.Second,
		Workers:      runtime.NumCPU(),
		BufferSizes:  []int{BufferSize4KB, BufferSize8KB},
		OpsPerBuffer: 10,
	}

	result := RunLoadTest(config)

	// Validate results
	if result.TotalOperations == 0 {
		t.Error("Expected operations > 0")
	}

	if result.OpsPerSecond == 0 {
		t.Error("Expected ops/sec > 0")
	}

	// Hit rate should be very high (>95%)
	if result.HitRate < 95.0 {
		t.Errorf("Expected hit rate >95%%, got %.2f%%", result.HitRate)
	}

	t.Logf("Total operations: %d", result.TotalOperations)
	t.Logf("Ops/sec: %.2f M", result.OpsPerSecond/1_000_000)
	t.Logf("Hit rate: %.2f%%", result.HitRate)
	t.Logf("Total allocs: %d", result.TotalAllocs)
	t.Logf("Allocs per op: %.4f", float64(result.TotalAllocs)/float64(result.TotalOperations))
}

// TestLoadTest_HighConcurrency tests with many workers
func TestLoadTest_HighConcurrency(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping high concurrency test in short mode")
	}

	config := LoadTestConfig{
		Duration:     5 * time.Second,
		Workers:      runtime.NumCPU() * 4, // 4x oversubscription
		BufferSizes:  []int{BufferSize2KB, BufferSize4KB, BufferSize8KB, BufferSize16KB},
		OpsPerBuffer: 50,
	}

	result := RunLoadTest(config)

	// Should still have high hit rate under contention
	if result.HitRate < 90.0 {
		t.Errorf("Expected hit rate >90%% even under high concurrency, got %.2f%%", result.HitRate)
	}

	t.Logf("Workers: %d", config.Workers)
	t.Logf("Total operations: %d", result.TotalOperations)
	t.Logf("Ops/sec: %.2f M", result.OpsPerSecond/1_000_000)
	t.Logf("Hit rate: %.2f%%", result.HitRate)
	t.Logf("GC cycles: %d", result.TotalGCPauses)
}

// TestLoadTest_AllSizes tests all buffer sizes
func TestLoadTest_AllSizes(t *testing.T) {
	if testing.Short() {
		t.Skip("Skipping all sizes test in short mode")
	}

	config := LoadTestConfig{
		Duration: 5 * time.Second,
		Workers:  runtime.NumCPU(),
		BufferSizes: []int{
			BufferSize2KB, BufferSize4KB, BufferSize8KB,
			BufferSize16KB, BufferSize32KB, BufferSize64KB,
		},
		OpsPerBuffer: 100,
	}

	result := RunLoadTest(config)

	// All pools should have some activity
	if result.Metrics.Pool2KB.Gets == 0 {
		t.Error("Expected Pool2KB to have activity")
	}
	if result.Metrics.Pool4KB.Gets == 0 {
		t.Error("Expected Pool4KB to have activity")
	}
	if result.Metrics.Pool8KB.Gets == 0 {
		t.Error("Expected Pool8KB to have activity")
	}
	if result.Metrics.Pool16KB.Gets == 0 {
		t.Error("Expected Pool16KB to have activity")
	}
	if result.Metrics.Pool32KB.Gets == 0 {
		t.Error("Expected Pool32KB to have activity")
	}
	if result.Metrics.Pool64KB.Gets == 0 {
		t.Error("Expected Pool64KB to have activity")
	}

	// Global hit rate should still be high
	if result.HitRate < 95.0 {
		t.Errorf("Expected hit rate >95%%, got %.2f%%", result.HitRate)
	}

	t.Logf("Total operations: %d", result.TotalOperations)
	t.Logf("Global hit rate: %.2f%%", result.HitRate)
	t.Logf("GC cycles: %d (%.2f per 1M ops)",
		result.TotalGCPauses,
		float64(result.TotalGCPauses)/float64(result.TotalOperations)*1_000_000)
}

// BenchmarkLoadTest benchmarks the load test itself
func BenchmarkLoadTest(b *testing.B) {
	config := LoadTestConfig{
		Duration:     1 * time.Second,
		Workers:      runtime.NumCPU(),
		BufferSizes:  []int{BufferSize4KB},
		OpsPerBuffer: 10,
	}

	b.ResetTimer()

	for i := 0; i < b.N; i++ {
		result := RunLoadTest(config)
		if result.HitRate < 95.0 {
			b.Errorf("Hit rate too low: %.2f%%", result.HitRate)
		}
	}
}

// Example test that demonstrates usage
func ExampleRunLoadTest() {
	config := LoadTestConfig{
		Duration:     5 * time.Second,
		Workers:      4,
		BufferSizes:  []int{BufferSize4KB, BufferSize8KB},
		OpsPerBuffer: 100,
	}

	result := RunLoadTest(config)
	PrintLoadTestResult(result)
}
