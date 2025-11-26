#!/bin/bash

# Comprehensive test runner for Electron project
# Runs unit tests, benchmarks, race detector, and generates coverage

set -e

echo "🧪 Electron Comprehensive Test Suite"
echo "===================================="
echo ""

# Colors for output
GREEN='\033[0;32m'
RED='\033[0;31m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Track results
FAILURES=0

# 1. Unit Tests
echo "📋 Running unit tests..."
if go test -v ./... > test_results.txt 2>&1; then
    echo -e "${GREEN}✅ Unit tests passed${NC}"
else
    echo -e "${RED}❌ Unit tests failed${NC}"
    cat test_results.txt
    FAILURES=$((FAILURES + 1))
fi
echo ""

# 2. Race Detector
echo "🏃 Running race detector..."
if go test -race ./... > race_results.txt 2>&1; then
    echo -e "${GREEN}✅ No data races detected${NC}"
else
    echo -e "${RED}❌ Data races detected${NC}"
    cat race_results.txt
    FAILURES=$((FAILURES + 1))
fi
echo ""

# 3. Benchmarks
echo "⚡ Running benchmarks..."
go test -bench=. -benchmem -run=^$ ./... > bench_results.txt 2>&1
echo -e "${GREEN}✅ Benchmarks completed${NC}"
echo "Results saved to bench_results.txt"
echo ""

# 4. Coverage
echo "📊 Generating coverage report..."
go test -coverprofile=coverage.out ./... > /dev/null 2>&1
COVERAGE=$(go tool cover -func=coverage.out | grep total | awk '{print $3}')
echo -e "${GREEN}✅ Coverage: $COVERAGE${NC}"
echo "Run 'go tool cover -html=coverage.out' to view detailed report"
echo ""

# 5. Vet
echo "🔍 Running go vet..."
if go vet ./... > vet_results.txt 2>&1; then
    echo -e "${GREEN}✅ go vet passed${NC}"
else
    echo -e "${YELLOW}⚠️  go vet warnings${NC}"
    cat vet_results.txt
fi
echo ""

# 6. Check for unsafe usage
echo "🔒 Checking unsafe package usage..."
UNSAFE_COUNT=$(grep -r "unsafe\." --include="*.go" | wc -l)
if [ "$UNSAFE_COUNT" -gt 0 ]; then
    echo -e "${YELLOW}⚠️  Found $UNSAFE_COUNT uses of unsafe package${NC}"
    echo "Run '/unsafe-review' command for detailed review"
else
    echo -e "${GREEN}✅ No unsafe code found${NC}"
fi
echo ""

# Summary
echo "===================================="
echo "📈 Test Summary"
echo "===================================="
echo "Coverage: $COVERAGE"
echo "Unsafe usage: $UNSAFE_COUNT locations"
echo ""

if [ $FAILURES -eq 0 ]; then
    echo -e "${GREEN}✅ All tests passed!${NC}"
    exit 0
else
    echo -e "${RED}❌ $FAILURES test suite(s) failed${NC}"
    exit 1
fi
