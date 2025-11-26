package shockwave

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// LoadTestConfig configures the load test
type LoadTestConfig struct {
	// Duration of test
	Duration time.Duration

	// Number of concurrent workers
	Workers int

	// Buffer sizes to test (mix of sizes)
	BufferSizes []int

	// Operations per buffer before returning
	OpsPerBuffer int
}

// LoadTestResult contains load test results
type LoadTestResult struct {
	Duration        time.Duration
	TotalOperations uint64
	OpsPerSecond    float64
	HitRate         float64
	Metrics         BufferPoolMetrics

	// Memory stats
	AllocsBefore uint64
	AllocsAfter  uint64
	TotalAllocs  uint64

	// GC stats
	GCPausesBefore uint64
	GCPausesAfter  uint64
	TotalGCPauses  uint64
}

// RunLoadTest runs a comprehensive load test on the buffer pool
func RunLoadTest(config LoadTestConfig) LoadTestResult {
	if config.Duration == 0 {
		config.Duration = 10 * time.Second
	}
	if config.Workers == 0 {
		config.Workers = runtime.NumCPU()
	}
	if len(config.BufferSizes) == 0 {
		config.BufferSizes = []int{
			BufferSize2KB, BufferSize4KB, BufferSize8KB,
			BufferSize16KB, BufferSize32KB, BufferSize64KB,
		}
	}
	if config.OpsPerBuffer == 0 {
		config.OpsPerBuffer = 100
	}

	// Reset metrics
	globalBufferPool.ResetMetrics()

	// Warmup
	globalBufferPool.Warmup(100)

	// Collect initial memory stats
	var memStatsBefore runtime.MemStats
	runtime.ReadMemStats(&memStatsBefore)

	// Start workers
	var totalOps atomic.Uint64
	var wg sync.WaitGroup
	stopCh := make(chan struct{})

	for i := 0; i < config.Workers; i++ {
		wg.Add(1)
		go func(workerID int) {
			defer wg.Done()

			// Each worker cycles through buffer sizes
			sizeIdx := workerID % len(config.BufferSizes)
			localOps := uint64(0)

			for {
				select {
				case <-stopCh:
					totalOps.Add(localOps)
					return
				default:
					size := config.BufferSizes[sizeIdx]

					// Get buffer
					buf := globalBufferPool.Get(size)

					// Simulate work
					for j := 0; j < config.OpsPerBuffer; j++ {
						buf[j%len(buf)] = byte(j)
					}

					// Put buffer
					globalBufferPool.Put(buf)

					localOps++

					// Rotate to next size
					sizeIdx = (sizeIdx + 1) % len(config.BufferSizes)
				}
			}
		}(i)
	}

	// Run for specified duration
	time.Sleep(config.Duration)
	close(stopCh)
	wg.Wait()

	// Collect final memory stats
	var memStatsAfter runtime.MemStats
	runtime.ReadMemStats(&memStatsAfter)

	// Get metrics
	metrics := globalBufferPool.GetMetrics()

	// Calculate results
	result := LoadTestResult{
		Duration:        config.Duration,
		TotalOperations: totalOps.Load(),
		OpsPerSecond:    float64(totalOps.Load()) / config.Duration.Seconds(),
		HitRate:         metrics.GlobalHitRate,
		Metrics:         metrics,

		AllocsBefore: memStatsBefore.Mallocs,
		AllocsAfter:  memStatsAfter.Mallocs,
		TotalAllocs:  memStatsAfter.Mallocs - memStatsBefore.Mallocs,

		GCPausesBefore: uint64(memStatsBefore.NumGC),
		GCPausesAfter:  uint64(memStatsAfter.NumGC),
		TotalGCPauses:  uint64(memStatsAfter.NumGC - memStatsBefore.NumGC),
	}

	return result
}

// PrintLoadTestResult prints a formatted load test result
func PrintLoadTestResult(result LoadTestResult) {
	fmt.Println("\n========== Buffer Pool Load Test Results ==========")
	fmt.Printf("Duration: %v\n", result.Duration)
	fmt.Printf("Total Operations: %d\n", result.TotalOperations)
	fmt.Printf("Operations/Second: %.2f M ops/s\n", result.OpsPerSecond/1_000_000)
	fmt.Printf("Hit Rate: %.2f%%\n", result.HitRate)
	fmt.Println()

	fmt.Println("Memory Allocations:")
	fmt.Printf("  Before: %d\n", result.AllocsBefore)
	fmt.Printf("  After: %d\n", result.AllocsAfter)
	fmt.Printf("  Total: %d\n", result.TotalAllocs)
	fmt.Printf("  Per Operation: %.2f\n", float64(result.TotalAllocs)/float64(result.TotalOperations))
	fmt.Println()

	fmt.Println("Garbage Collection:")
	fmt.Printf("  GC Cycles Before: %d\n", result.GCPausesBefore)
	fmt.Printf("  GC Cycles After: %d\n", result.GCPausesAfter)
	fmt.Printf("  Total GC Cycles: %d\n", result.TotalGCPauses)
	fmt.Printf("  GC per 1M ops: %.2f\n", float64(result.TotalGCPauses)/float64(result.TotalOperations)*1_000_000)
	fmt.Println()

	fmt.Println("Pool Metrics:")
	fmt.Printf("  Global Hit Rate: %.2f%%\n", result.Metrics.GlobalHitRate)
	fmt.Printf("  Total Gets: %d\n", result.Metrics.TotalGets)
	fmt.Printf("  Total Puts: %d\n", result.Metrics.TotalPuts)
	fmt.Printf("  Memory Allocated: %.2f MB\n", float64(result.Metrics.MemoryAllocated)/(1024*1024))
	fmt.Printf("  Memory Reused: %.2f MB\n", float64(result.Metrics.MemoryReused)/(1024*1024))
	fmt.Printf("  Reuse Efficiency: %.2f%%\n", result.Metrics.ReuseEfficiency)
	fmt.Println()

	fmt.Println("Per-Size Pool Performance:")
	printSizedResultMetrics("2KB", result.Metrics.Pool2KB)
	printSizedResultMetrics("4KB", result.Metrics.Pool4KB)
	printSizedResultMetrics("8KB", result.Metrics.Pool8KB)
	printSizedResultMetrics("16KB", result.Metrics.Pool16KB)
	printSizedResultMetrics("32KB", result.Metrics.Pool32KB)
	printSizedResultMetrics("64KB", result.Metrics.Pool64KB)

	fmt.Println("\n========== Test Complete ==========")
}

func printSizedResultMetrics(label string, m SizedPoolMetrics) {
	fmt.Printf("  %s: Gets=%d, Puts=%d, Hit Rate=%.2f%%, Misses=%d\n",
		label, m.Gets, m.Puts, m.HitRate, m.Misses)
}

// Example usage:
//
//	func main() {
//	    config := LoadTestConfig{
//	        Duration:     30 * time.Second,
//	        Workers:      runtime.NumCPU(),
//	        BufferSizes:  []int{BufferSize4KB, BufferSize8KB},
//	        OpsPerBuffer: 100,
//	    }
//
//	    result := RunLoadTest(config)
//	    PrintLoadTestResult(result)
//	}
