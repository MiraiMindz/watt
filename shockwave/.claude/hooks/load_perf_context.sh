#!/bin/bash
# Load current performance context for Claude

RESULTS_DIR="results"

echo "📊 Current Performance Context:"
echo ""

# Check if we have recent benchmark results
if [ -d "$RESULTS_DIR" ]; then
    LATEST_RESULT=$(ls -t "$RESULTS_DIR"/full_*.txt 2>/dev/null | head -1)

    if [ -n "$LATEST_RESULT" ]; then
        echo "Latest benchmark run: $(basename "$LATEST_RESULT")"
        echo ""

        # Show summary of key benchmarks
        echo "Key Benchmark Results:"
        grep -E "BenchmarkParseRequest|BenchmarkKeepAlive|BenchmarkWriteStatus" "$LATEST_RESULT" | head -5 || echo "No critical benchmark data found"
    else
        echo "No recent benchmark results found. Run /bench to establish baseline."
    fi
else
    echo "No results directory. Run /bench to create baseline."
fi

echo ""

# Show current git status
if git rev-parse --git-dir > /dev/null 2>&1; then
    CURRENT_BRANCH=$(git branch --show-current)
    LAST_COMMIT=$(git log -1 --oneline)
    echo "Git context:"
    echo "  Branch: $CURRENT_BRANCH"
    echo "  Latest: $LAST_COMMIT"
fi

exit 0
