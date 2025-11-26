#!/bin/bash
# Pre-commit hook to verify benchmarks still pass

set -e

echo "🔍 Running quick benchmark validation before commit..." >&2

# Run quick benchmarks (1 second each)
if ! go test -bench=. -benchtime=1s -timeout=30s ./pkg/shockwave/... > /tmp/precommit_bench.txt 2>&1; then
    echo "❌ Benchmark tests failed. Please fix before committing." >&2
    cat /tmp/precommit_bench.txt >&2
    exit 1
fi

# Check for allocation regressions in critical benchmarks
CRITICAL_BENCHMARKS=(
    "BenchmarkParseRequest"
    "BenchmarkKeepAlive"
    "BenchmarkWriteStatus"
)

for bench in "${CRITICAL_BENCHMARKS[@]}"; do
    if grep "$bench" /tmp/precommit_bench.txt | grep -v "0 allocs/op" > /dev/null 2>&1; then
        echo "❌ Critical benchmark $bench has allocations! Zero allocations required." >&2
        grep "$bench" /tmp/precommit_bench.txt >&2
        exit 1
    fi
done

echo "✅ Benchmark validation passed" >&2
exit 0
