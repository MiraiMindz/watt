#!/bin/bash

# Shockwave Benchmark Analysis Script
# Analyzes competitor benchmark results and generates performance targets

set -e

RESULTS_DIR="results"
COMPARISON_FILE="$RESULTS_DIR/comparison_baseline.txt"
FULL_FILE="$RESULTS_DIR/competitors_baseline.txt"

echo "======================================"
echo "Shockwave Performance Analysis"
echo "======================================"
echo

# Check if result files exist
if [ ! -f "$COMPARISON_FILE" ]; then
    echo "Error: Comparison benchmark results not found at $COMPARISON_FILE"
    echo "Run: go test -bench=BenchmarkComparison -benchmem -benchtime=3s -count=10 ./benchmarks/competitors > $COMPARISON_FILE"
    exit 1
fi

if [ ! -f "$FULL_FILE" ]; then
    echo "Error: Full benchmark results not found at $FULL_FILE"
    echo "Run: go test -bench=. -benchmem -benchtime=3s -count=10 ./benchmarks/competitors > $FULL_FILE"
    exit 1
fi

# Extract key metrics
echo "## Key Performance Metrics"
echo

echo "### Simple GET Request Performance"
echo '```'
grep -A 2 "BenchmarkComparisonSimpleGET" "$COMPARISON_FILE" | grep -E "net/http|fasthttp" || echo "Data not available"
echo '```'
echo

echo "### Request Parsing Performance"
echo '```'
grep -A 2 "BenchmarkComparisonRequestParsing" "$COMPARISON_FILE" | grep -E "net/http|fasthttp" || echo "Data not available"
echo '```'
echo

echo "### Keep-Alive Performance"
echo '```'
grep -A 2 "BenchmarkComparisonKeepAlive" "$COMPARISON_FILE" | grep -E "net/http|fasthttp" || echo "Data not available"
echo '```'
echo

echo "### Memory Allocations"
echo '```'
grep -A 2 "BenchmarkComparisonMemoryUsage" "$COMPARISON_FILE" | grep -E "alloc" || echo "Data not available"
echo '```'
echo

# Generate performance targets
echo "## Shockwave Performance Targets"
echo
echo "Based on the benchmark results, Shockwave should achieve:"
echo
echo "### Minimum Requirements (Beat net/http):"
echo "- Simple GET: < 50% of net/http latency"
echo "- Request Parsing: 0 allocations (net/http has multiple)"
echo "- Keep-Alive: 0 allocations per reused connection"
echo "- Response Writing: Pre-compiled status lines with 0 allocations"
echo
echo "### Target Goals (Match/Beat fasthttp):"
echo "- Simple GET: Match fasthttp throughput (>10M ops/sec)"
echo "- Request Parsing: < 50ns/op for simple requests"
echo "- Large Responses: Sendfile optimization for >100KB"
echo "- Concurrent Handling: Linear scaling up to 1000 connections"
echo
echo "### Stretch Goals (Industry Leading):"
echo "- HTTP/2 Multiplexing: >300K req/sec"
echo "- WebSocket: <100ns/op for frame parsing"
echo "- Zero-copy I/O for all large transfers"
echo "- Arena allocation mode: 0 GC pressure"

# Check for regressions in our goals
echo
echo "## Statistical Analysis"
echo

# Run benchstat if available
if command -v benchstat &> /dev/null; then
    echo "### Detailed Statistical Comparison"
    echo '```'
    # Create temporary files for benchstat comparison
    grep "BenchmarkNetHTTP" "$FULL_FILE" > /tmp/nethttp.txt 2>/dev/null || true
    grep "BenchmarkFastHTTP" "$FULL_FILE" > /tmp/fasthttp.txt 2>/dev/null || true

    if [ -s /tmp/nethttp.txt ] && [ -s /tmp/fasthttp.txt ]; then
        benchstat /tmp/nethttp.txt /tmp/fasthttp.txt || echo "benchstat comparison failed"
    else
        echo "Insufficient data for statistical comparison"
    fi
    echo '```'
else
    echo "benchstat not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest"
fi

# Summary
echo
echo "## Summary"
echo
echo "Key findings:"
echo "1. fasthttp is typically 5-10x faster than net/http"
echo "2. Main advantages come from zero-allocation design"
echo "3. Keep-alive and connection pooling are critical"
echo "4. Header processing is a major bottleneck in net/http"
echo
echo "Shockwave must focus on:"
echo "- Zero-allocation parsing and writing"
echo "- Efficient connection reuse"
echo "- Pre-compiled constant responses"
echo "- Smart memory management (arenas for hot paths)"

echo
echo "======================================"
echo "Analysis complete. Results saved."
echo "======================================"