// +build prometheus

package shockwave

import (
	"github.com/prometheus/client_golang/prometheus"
	"github.com/prometheus/client_golang/prometheus/promauto"
)

// Prometheus metrics for buffer pool
var (
	// Buffer pool operations
	bufferPoolGets = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "gets_total",
			Help:      "Total number of buffer Get operations",
		},
		[]string{"size"},
	)

	bufferPoolPuts = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "puts_total",
			Help:      "Total number of buffer Put operations",
		},
		[]string{"size"},
	)

	bufferPoolHits = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "hits_total",
			Help:      "Total number of buffer pool hits (reuse)",
		},
		[]string{"size"},
	)

	bufferPoolMisses = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "misses_total",
			Help:      "Total number of buffer pool misses (new allocation)",
		},
		[]string{"size"},
	)

	bufferPoolDiscards = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "discards_total",
			Help:      "Total number of buffers discarded (wrong size)",
		},
		[]string{"size"},
	)

	// Buffer pool gauge metrics
	bufferPoolHitRate = promauto.NewGaugeVec(
		prometheus.GaugeOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "hit_rate",
			Help:      "Current buffer pool hit rate (0-100%)",
		},
		[]string{"size"},
	)

	bufferPoolBytesAllocated = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "bytes_allocated_total",
			Help:      "Total bytes allocated",
		},
		[]string{"size"},
	)

	bufferPoolBytesReused = promauto.NewCounterVec(
		prometheus.CounterOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "bytes_reused_total",
			Help:      "Total bytes reused from pool",
		},
		[]string{"size"},
	)

	// Global metrics
	bufferPoolGlobalHitRate = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "global_hit_rate",
			Help:      "Global buffer pool hit rate across all sizes (0-100%)",
		},
	)

	bufferPoolMemoryAllocated = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "memory_allocated_bytes",
			Help:      "Total memory allocated across all pools",
		},
	)

	bufferPoolMemoryReused = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "memory_reused_bytes",
			Help:      "Total memory reused across all pools",
		},
	)

	bufferPoolReuseEfficiency = promauto.NewGauge(
		prometheus.GaugeOpts{
			Namespace: "shockwave",
			Subsystem: "buffer_pool",
			Name:      "reuse_efficiency",
			Help:      "Memory reuse efficiency (0-100%)",
		},
	)
)

// UpdatePrometheusMetrics updates all Prometheus metrics from current pool state
// Call this periodically (e.g., every 10 seconds) from a background goroutine
func UpdatePrometheusMetrics() {
	metrics := GetBufferPoolMetrics()

	// Update per-size metrics
	updateSizedPrometheusMetrics("2kb", metrics.Pool2KB)
	updateSizedPrometheusMetrics("4kb", metrics.Pool4KB)
	updateSizedPrometheusMetrics("8kb", metrics.Pool8KB)
	updateSizedPrometheusMetrics("16kb", metrics.Pool16KB)
	updateSizedPrometheusMetrics("32kb", metrics.Pool32KB)
	updateSizedPrometheusMetrics("64kb", metrics.Pool64KB)

	// Update global metrics
	bufferPoolGlobalHitRate.Set(metrics.GlobalHitRate)
	bufferPoolMemoryAllocated.Set(float64(metrics.MemoryAllocated))
	bufferPoolMemoryReused.Set(float64(metrics.MemoryReused))
	bufferPoolReuseEfficiency.Set(metrics.ReuseEfficiency)
}

func updateSizedPrometheusMetrics(label string, m SizedPoolMetrics) {
	// Counters (add delta since last update)
	bufferPoolGets.WithLabelValues(label).Add(float64(m.Gets))
	bufferPoolPuts.WithLabelValues(label).Add(float64(m.Puts))
	bufferPoolHits.WithLabelValues(label).Add(float64(m.Hits))
	bufferPoolMisses.WithLabelValues(label).Add(float64(m.Misses))
	bufferPoolDiscards.WithLabelValues(label).Add(float64(m.Discards))
	bufferPoolBytesAllocated.WithLabelValues(label).Add(float64(m.Allocated))
	bufferPoolBytesReused.WithLabelValues(label).Add(float64(m.Reused))

	// Gauges (set current value)
	bufferPoolHitRate.WithLabelValues(label).Set(m.HitRate)
}

// PrometheusCollector implements prometheus.Collector for custom collection
type PrometheusCollector struct {
	pool *BufferPool
}

// NewPrometheusCollector creates a new Prometheus collector for a buffer pool
func NewPrometheusCollector(pool *BufferPool) *PrometheusCollector {
	return &PrometheusCollector{pool: pool}
}

// Describe implements prometheus.Collector
func (pc *PrometheusCollector) Describe(ch chan<- *prometheus.Desc) {
	// Metrics are already registered via promauto
	// This is a no-op for compatibility
}

// Collect implements prometheus.Collector
func (pc *PrometheusCollector) Collect(ch chan<- prometheus.Metric) {
	// Update metrics on each scrape
	UpdatePrometheusMetrics()
}

// Example usage:
//
//	import (
//	    "net/http"
//	    "time"
//	    "github.com/prometheus/client_golang/prometheus/promhttp"
//	)
//
//	func main() {
//	    // Register custom collector
//	    prometheus.MustRegister(NewPrometheusCollector(globalBufferPool))
//
//	    // Start periodic updates (optional if using custom collector)
//	    go func() {
//	        ticker := time.NewTicker(10 * time.Second)
//	        defer ticker.Stop()
//	        for range ticker.C {
//	            UpdatePrometheusMetrics()
//	        }
//	    }()
//
//	    // Expose metrics endpoint
//	    http.Handle("/metrics", promhttp.Handler())
//	    http.ListenAndServe(":9090", nil)
//	}
