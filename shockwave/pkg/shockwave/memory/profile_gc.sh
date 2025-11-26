#!/bin/bash
# GC Profiling Script for Memory Management Modes
# Compares GC overhead across Standard, Green Tea, and Arena modes

set -e

RESULTS_DIR="gc_profile_results"
mkdir -p "$RESULTS_DIR"

echo "====== Shockwave Memory Management GC Profiling ======"
echo "Testing Date: $(date)"
echo ""

# Function to run benchmark with GC tracing
run_gc_profile() {
    local MODE=$1
    local BUILD_TAGS=$2
    local OUTPUT_FILE=$3

    echo "Running GC profile for $MODE mode..."

    if [ -n "$BUILD_TAGS" ]; then
        GODEBUG=gctrace=1 go test -tags="$BUILD_TAGS" \
            -bench=BenchmarkGCPressure \
            -benchtime=5s \
            -benchmem \
            2>&1 | tee "$OUTPUT_FILE"
    else
        GODEBUG=gctrace=1 go test \
            -bench=BenchmarkGCPressure \
            -benchtime=5s \
            -benchmem \
            2>&1 | tee "$OUTPUT_FILE"
    fi

    echo "Results saved to $OUTPUT_FILE"
    echo ""
}

# 1. Standard Pool Mode
echo "=== Standard Pool Mode (sync.Pool) ==="
run_gc_profile "Standard" "" "$RESULTS_DIR/standard_gc.txt"

# 2. Green Tea GC Mode
echo "=== Green Tea GC Mode ==="
run_gc_profile "Green Tea" "greenteagc" "$RESULTS_DIR/greentea_gc.txt"

# 3. Arena Mode (if arenas are available)
echo "=== Arena Mode (GOEXPERIMENT=arenas) ==="
if GOEXPERIMENT=arenas go version &>/dev/null; then
    GOEXPERIMENT=arenas run_gc_profile "Arena" "arenas" "$RESULTS_DIR/arena_gc.txt"
else
    echo "Arena support not available (requires GOEXPERIMENT=arenas)"
    echo "Skipping arena mode profiling"
fi

echo ""
echo "====== GC Profiling Complete ======"
echo "Results directory: $RESULTS_DIR"
echo ""
echo "Analyzing GC overhead..."

# Parse GC stats from output
parse_gc_stats() {
    local FILE=$1
    echo "--- $FILE ---"

    # Count GC cycles
    GC_COUNT=$(grep "^gc " "$FILE" | wc -l)
    echo "Total GC cycles: $GC_COUNT"

    # Parse pause times
    if [ $GC_COUNT -gt 0 ]; then
        AVG_PAUSE=$(grep "^gc " "$FILE" | awk '{sum+=$8} END {if (NR>0) print sum/NR; else print 0}')
        MAX_PAUSE=$(grep "^gc " "$FILE" | awk 'BEGIN{max=0} {if($8>max) max=$8} END{print max}')
        echo "Average GC pause: ${AVG_PAUSE}ms"
        echo "Maximum GC pause: ${MAX_PAUSE}ms"
    fi

    # Extract benchmark results
    BENCH_RESULT=$(grep "BenchmarkGCPressure" "$FILE" | tail -1)
    echo "Benchmark: $BENCH_RESULT"
    echo ""
}

echo ""
echo "=== Summary ==="
for file in "$RESULTS_DIR"/*.txt; do
    if [ -f "$file" ]; then
        parse_gc_stats "$file"
    fi
done

echo "====== Profile Complete ======"
echo ""
echo "To visualize GC traces:"
echo "  cat $RESULTS_DIR/standard_gc.txt | grep '^gc '"
echo "  cat $RESULTS_DIR/greentea_gc.txt | grep '^gc '"
echo ""
echo "Key metrics to compare:"
echo "  - Total GC cycles"
echo "  - Average pause time"
echo "  - Maximum pause time"
echo "  - Allocation rate (B/op)"
echo "  - Throughput (ops/sec)"
