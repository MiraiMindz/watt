// Package shockwave provides high-performance HTTP server components.
package shockwave

import (
	"fmt"
	"sync"
	"sync/atomic"
)

// Buffer size classes for optimal memory usage
// Sizes are powers of 2 for efficient allocation
const (
	BufferSize2KB  = 2 * 1024   // 2KB - small requests/responses
	BufferSize4KB  = 4 * 1024   // 4KB - typical HTTP requests
	BufferSize8KB  = 8 * 1024   // 8KB - medium payloads
	BufferSize16KB = 16 * 1024  // 16KB - large headers
	BufferSize32KB = 32 * 1024  // 32KB - file chunks
	BufferSize64KB = 64 * 1024  // 64KB - large payloads
)

// BufferPool provides size-specific buffer pooling with metrics tracking.
//
// Design:
// - Multiple size classes (2KB, 4KB, 8KB, 16KB, 32KB, 64KB)
// - Automatic size selection based on requested size
// - Comprehensive metrics (hit rate, allocation count, reuse count)
// - Zero allocations on pool hit
// - Thread-safe with sync.Pool
//
// Performance characteristics:
// - Pool hit: 0 allocs/op, ~5-10 ns/op
// - Pool miss: 1 alloc/op, ~50-100 ns/op
// - Target hit rate: >95%
type BufferPool struct {
	// Size-specific pools
	pool2KB  *sizedBufferPool
	pool4KB  *sizedBufferPool
	pool8KB  *sizedBufferPool
	pool16KB *sizedBufferPool
	pool32KB *sizedBufferPool
	pool64KB *sizedBufferPool

	// Global metrics
	totalGets atomic.Uint64
	totalPuts atomic.Uint64
}

// sizedBufferPool manages a single size class of buffers
type sizedBufferPool struct {
	size int
	pool sync.Pool

	// Metrics
	gets      atomic.Uint64 // Total Get() calls
	puts      atomic.Uint64 // Total Put() calls
	hits      atomic.Uint64 // Pool hits (reused buffer)
	misses    atomic.Uint64 // Pool misses (new allocation)
	discards  atomic.Uint64 // Buffers discarded (wrong size)
	allocated atomic.Uint64 // Total bytes allocated
	reused    atomic.Uint64 // Total bytes reused
}

// newSizedBufferPool creates a new sized buffer pool
func newSizedBufferPool(size int) *sizedBufferPool {
	sbp := &sizedBufferPool{
		size: size,
	}
	sbp.pool.New = func() interface{} {
		// Track allocation
		sbp.misses.Add(1)
		sbp.allocated.Add(uint64(size))

		// Allocate buffer
		buf := make([]byte, size)
		return &buf
	}
	return sbp
}

// Get retrieves a buffer from the pool
// Returns a buffer of at least the requested size
// Allocation behavior: 0 allocs/op on hit, 1 alloc/op on miss
func (sbp *sizedBufferPool) Get() []byte {
	sbp.gets.Add(1)

	// Get from pool - either reuses existing or calls New()
	bufPtr := sbp.pool.Get().(*[]byte)
	buf := *bufPtr

	// Note: hits are computed as (gets - misses) in metrics
	// New() increments misses, so we can derive hits

	// Reset length to full capacity for consistent behavior
	return buf[:sbp.size]
}

// Put returns a buffer to the pool
// Buffers must be of the correct size, otherwise they're discarded
// Allocation behavior: 0 allocs/op
func (sbp *sizedBufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	sbp.puts.Add(1)

	// Verify buffer size
	if cap(buf) < sbp.size {
		sbp.discards.Add(1)
		return
	}

	// Reset to full capacity
	buf = buf[:sbp.size]

	// Return to pool
	sbp.pool.Put(&buf)
}

// Reset clears the buffer (zeroes it out)
// This is optional but recommended for security-sensitive data
func (sbp *sizedBufferPool) Reset(buf []byte) {
	for i := range buf {
		buf[i] = 0
	}
}

// NewBufferPool creates a new buffer pool with size-specific pools
func NewBufferPool() *BufferPool {
	return &BufferPool{
		pool2KB:  newSizedBufferPool(BufferSize2KB),
		pool4KB:  newSizedBufferPool(BufferSize4KB),
		pool8KB:  newSizedBufferPool(BufferSize8KB),
		pool16KB: newSizedBufferPool(BufferSize16KB),
		pool32KB: newSizedBufferPool(BufferSize32KB),
		pool64KB: newSizedBufferPool(BufferSize64KB),
	}
}

// Get retrieves a buffer of at least the requested size.
// Returns the smallest buffer that satisfies the size requirement.
//
// Example:
//   buf := pool.Get(3000)  // Returns 4KB buffer
//   defer pool.Put(buf)
//
// Allocation behavior: 0 allocs/op on hit, 1 alloc/op on miss
func (bp *BufferPool) Get(size int) []byte {
	bp.totalGets.Add(1)

	// Select appropriate pool based on size
	switch {
	case size <= BufferSize2KB:
		return bp.pool2KB.Get()
	case size <= BufferSize4KB:
		return bp.pool4KB.Get()
	case size <= BufferSize8KB:
		return bp.pool8KB.Get()
	case size <= BufferSize16KB:
		return bp.pool16KB.Get()
	case size <= BufferSize32KB:
		return bp.pool32KB.Get()
	case size <= BufferSize64KB:
		return bp.pool64KB.Get()
	default:
		// Size too large for pooling, allocate directly
		// This ensures we don't pool huge buffers
		return make([]byte, size)
	}
}

// Put returns a buffer to the appropriate pool.
// The buffer size determines which pool it goes to.
// Buffers larger than 64KB are not pooled.
//
// After calling Put, you MUST NOT use the buffer anymore.
//
// Allocation behavior: 0 allocs/op
func (bp *BufferPool) Put(buf []byte) {
	if buf == nil {
		return
	}

	bp.totalPuts.Add(1)

	size := cap(buf)

	// Route to appropriate pool based on capacity
	switch {
	case size >= BufferSize2KB && size < BufferSize4KB:
		bp.pool2KB.Put(buf)
	case size >= BufferSize4KB && size < BufferSize8KB:
		bp.pool4KB.Put(buf)
	case size >= BufferSize8KB && size < BufferSize16KB:
		bp.pool8KB.Put(buf)
	case size >= BufferSize16KB && size < BufferSize32KB:
		bp.pool16KB.Put(buf)
	case size >= BufferSize32KB && size < BufferSize64KB:
		bp.pool32KB.Put(buf)
	case size >= BufferSize64KB:
		bp.pool64KB.Put(buf)
	default:
		// Buffer too small or wrong size, discard
	}
}

// PutWithReset returns a buffer to the pool after zeroing it out.
// Use this for security-sensitive data (passwords, tokens, etc.)
//
// Allocation behavior: 0 allocs/op
func (bp *BufferPool) PutWithReset(buf []byte) {
	if buf == nil {
		return
	}

	// Zero out the buffer
	for i := range buf {
		buf[i] = 0
	}

	bp.Put(buf)
}

// BufferPoolMetrics contains comprehensive pool statistics
type BufferPoolMetrics struct {
	// Per-size metrics
	Pool2KB  SizedPoolMetrics
	Pool4KB  SizedPoolMetrics
	Pool8KB  SizedPoolMetrics
	Pool16KB SizedPoolMetrics
	Pool32KB SizedPoolMetrics
	Pool64KB SizedPoolMetrics

	// Global metrics
	TotalGets uint64
	TotalPuts uint64

	// Computed metrics
	GlobalHitRate    float64 // Overall hit rate across all pools
	MemoryAllocated  uint64  // Total bytes allocated
	MemoryReused     uint64  // Total bytes reused from pool
	ReuseEfficiency  float64 // Percentage of memory reused vs allocated
}

// SizedPoolMetrics contains metrics for a single size class
type SizedPoolMetrics struct {
	Size      int     // Buffer size for this pool
	Gets      uint64  // Total Get() calls
	Puts      uint64  // Total Put() calls
	Hits      uint64  // Pool hits (reused buffer)
	Misses    uint64  // Pool misses (new allocation)
	Discards  uint64  // Buffers discarded (wrong size)
	HitRate   float64 // Hit rate percentage
	Allocated uint64  // Total bytes allocated
	Reused    uint64  // Total bytes reused
}

// GetMetrics returns comprehensive pool metrics
func (bp *BufferPool) GetMetrics() BufferPoolMetrics {
	metrics := BufferPoolMetrics{
		Pool2KB:  bp.getSizedMetrics(bp.pool2KB),
		Pool4KB:  bp.getSizedMetrics(bp.pool4KB),
		Pool8KB:  bp.getSizedMetrics(bp.pool8KB),
		Pool16KB: bp.getSizedMetrics(bp.pool16KB),
		Pool32KB: bp.getSizedMetrics(bp.pool32KB),
		Pool64KB: bp.getSizedMetrics(bp.pool64KB),
		TotalGets: bp.totalGets.Load(),
		TotalPuts: bp.totalPuts.Load(),
	}

	// Compute global metrics
	totalHits := metrics.Pool2KB.Hits + metrics.Pool4KB.Hits +
	            metrics.Pool8KB.Hits + metrics.Pool16KB.Hits +
	            metrics.Pool32KB.Hits + metrics.Pool64KB.Hits

	totalMisses := metrics.Pool2KB.Misses + metrics.Pool4KB.Misses +
	              metrics.Pool8KB.Misses + metrics.Pool16KB.Misses +
	              metrics.Pool32KB.Misses + metrics.Pool64KB.Misses

	total := totalHits + totalMisses
	if total > 0 {
		metrics.GlobalHitRate = float64(totalHits) / float64(total) * 100.0
	}

	metrics.MemoryAllocated = metrics.Pool2KB.Allocated + metrics.Pool4KB.Allocated +
	                         metrics.Pool8KB.Allocated + metrics.Pool16KB.Allocated +
	                         metrics.Pool32KB.Allocated + metrics.Pool64KB.Allocated

	metrics.MemoryReused = metrics.Pool2KB.Reused + metrics.Pool4KB.Reused +
	                      metrics.Pool8KB.Reused + metrics.Pool16KB.Reused +
	                      metrics.Pool32KB.Reused + metrics.Pool64KB.Reused

	if metrics.MemoryAllocated > 0 {
		metrics.ReuseEfficiency = float64(metrics.MemoryReused) / float64(metrics.MemoryAllocated) * 100.0
	}

	return metrics
}

// getSizedMetrics extracts metrics from a sized pool
func (bp *BufferPool) getSizedMetrics(sbp *sizedBufferPool) SizedPoolMetrics {
	gets := sbp.gets.Load()
	puts := sbp.puts.Load()
	misses := sbp.misses.Load()
	discards := sbp.discards.Load()
	allocated := sbp.allocated.Load()
	reused := sbp.reused.Load()

	// Compute hits as (gets - misses)
	// Since New() increments misses, hits = successful reuses from pool
	var hits uint64
	if gets >= misses {
		hits = gets - misses
	}

	var hitRate float64
	if gets > 0 {
		hitRate = float64(hits) / float64(gets) * 100.0
	}

	return SizedPoolMetrics{
		Size:      sbp.size,
		Gets:      gets,
		Puts:      puts,
		Hits:      hits,
		Misses:    misses,
		Discards:  discards,
		HitRate:   hitRate,
		Allocated: allocated,
		Reused:    reused,
	}
}

// PrintMetrics prints pool metrics in a human-readable format
func (bp *BufferPool) PrintMetrics() {
	metrics := bp.GetMetrics()

	fmt.Println("\n========== Buffer Pool Metrics ==========")
	fmt.Printf("Global Hit Rate: %.2f%%\n", metrics.GlobalHitRate)
	fmt.Printf("Total Gets: %d\n", metrics.TotalGets)
	fmt.Printf("Total Puts: %d\n", metrics.TotalPuts)
	fmt.Printf("Memory Allocated: %.2f MB\n", float64(metrics.MemoryAllocated)/(1024*1024))
	fmt.Printf("Memory Reused: %.2f MB\n", float64(metrics.MemoryReused)/(1024*1024))
	fmt.Printf("Reuse Efficiency: %.2f%%\n\n", metrics.ReuseEfficiency)

	fmt.Println("Per-Size Pool Metrics:")
	bp.printSizedMetrics("2KB", metrics.Pool2KB)
	bp.printSizedMetrics("4KB", metrics.Pool4KB)
	bp.printSizedMetrics("8KB", metrics.Pool8KB)
	bp.printSizedMetrics("16KB", metrics.Pool16KB)
	bp.printSizedMetrics("32KB", metrics.Pool32KB)
	bp.printSizedMetrics("64KB", metrics.Pool64KB)
	fmt.Println("==========================================")
}

func (bp *BufferPool) printSizedMetrics(label string, m SizedPoolMetrics) {
	fmt.Printf("  %s: Gets=%d, Puts=%d, Hits=%d, Misses=%d, Hit Rate=%.2f%%, Discards=%d\n",
		label, m.Gets, m.Puts, m.Hits, m.Misses, m.HitRate, m.Discards)
}

// ResetMetrics resets all metrics to zero
// Useful for benchmarking and testing
func (bp *BufferPool) ResetMetrics() {
	bp.totalGets.Store(0)
	bp.totalPuts.Store(0)

	bp.resetSizedMetrics(bp.pool2KB)
	bp.resetSizedMetrics(bp.pool4KB)
	bp.resetSizedMetrics(bp.pool8KB)
	bp.resetSizedMetrics(bp.pool16KB)
	bp.resetSizedMetrics(bp.pool32KB)
	bp.resetSizedMetrics(bp.pool64KB)
}

func (bp *BufferPool) resetSizedMetrics(sbp *sizedBufferPool) {
	sbp.gets.Store(0)
	sbp.puts.Store(0)
	sbp.hits.Store(0)
	sbp.misses.Store(0)
	sbp.discards.Store(0)
	sbp.allocated.Store(0)
	sbp.reused.Store(0)
}

// Warmup pre-allocates buffers in all pools
// This is useful for avoiding cold-start allocations
// Recommended: 10-100 buffers per pool for most workloads
func (bp *BufferPool) Warmup(count int) {
	pools := []*sizedBufferPool{
		bp.pool2KB, bp.pool4KB, bp.pool8KB,
		bp.pool16KB, bp.pool32KB, bp.pool64KB,
	}

	for _, pool := range pools {
		for i := 0; i < count; i++ {
			buf := pool.Get()
			pool.Put(buf)
		}
	}
}

// Global buffer pool instance
var globalBufferPool = NewBufferPool()

// GetBuffer retrieves a buffer from the global pool
// Allocation behavior: 0 allocs/op on hit
func GetBuffer(size int) []byte {
	return globalBufferPool.Get(size)
}

// PutBuffer returns a buffer to the global pool
// Allocation behavior: 0 allocs/op
func PutBuffer(buf []byte) {
	globalBufferPool.Put(buf)
}

// PutBufferWithReset returns a buffer to the global pool after zeroing it
// Allocation behavior: 0 allocs/op
func PutBufferWithReset(buf []byte) {
	globalBufferPool.PutWithReset(buf)
}

// GetBufferPoolMetrics returns metrics from the global pool
func GetBufferPoolMetrics() BufferPoolMetrics {
	return globalBufferPool.GetMetrics()
}

// PrintBufferPoolMetrics prints metrics from the global pool
func PrintBufferPoolMetrics() {
	globalBufferPool.PrintMetrics()
}

// WarmupBufferPool pre-allocates buffers in the global pool
func WarmupBufferPool(count int) {
	globalBufferPool.Warmup(count)
}

// ResetBufferPoolMetrics resets metrics in the global pool
func ResetBufferPoolMetrics() {
	globalBufferPool.ResetMetrics()
}
