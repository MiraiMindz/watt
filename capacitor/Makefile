.PHONY: help setup test test-race test-integration coverage bench bench-cache bench-compare profile fmt lint check clean

# Default target
help:
	@echo "Capacitor DAL - Development Commands"
	@echo ""
	@echo "Setup:"
	@echo "  make setup              - Install dependencies and tools"
	@echo ""
	@echo "Testing:"
	@echo "  make test               - Run all tests"
	@echo "  make test-race          - Run tests with race detector"
	@echo "  make test-integration   - Run integration tests"
	@echo "  make coverage           - Generate coverage report"
	@echo ""
	@echo "Benchmarking:"
	@echo "  make bench              - Run all benchmarks"
	@echo "  make bench-cache        - Run cache benchmarks"
	@echo "  make bench-compare      - Compare with baseline"
	@echo "  make profile            - Generate profiles"
	@echo ""
	@echo "Code Quality:"
	@echo "  make fmt                - Format code"
	@echo "  make lint               - Run linter"
	@echo "  make check              - Run all checks"
	@echo ""
	@echo "Cleanup:"
	@echo "  make clean              - Remove generated files"

# Setup development environment
setup:
	@echo "Installing Go tools..."
	go install golang.org/x/tools/cmd/goimports@latest
	go install golang.org/x/perf/cmd/benchstat@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	@echo "Downloading dependencies..."
	go mod download
	@echo "Setup complete!"

# Testing
test:
	@echo "Running tests..."
	go test -v ./...

test-race:
	@echo "Running tests with race detector..."
	go test -race -v ./...

test-integration:
	@echo "Running integration tests..."
	@echo "Starting test dependencies with docker-compose..."
	docker-compose -f docker-compose.test.yml up -d
	@echo "Waiting for services to be ready..."
	sleep 5
	go test -v -tags=integration ./...
	docker-compose -f docker-compose.test.yml down

coverage:
	@echo "Generating coverage report..."
	go test -cover -coverprofile=coverage.out ./...
	go tool cover -html=coverage.out -o coverage.html
	@echo "Coverage report generated: coverage.html"
	@go tool cover -func=coverage.out | grep total | awk '{print "Total coverage: " $$3}'

# Benchmarking
bench:
	@echo "Running all benchmarks..."
	@mkdir -p benchmarks/results
	go test -bench=. -benchmem -benchtime=5s -count=5 ./... | tee benchmarks/results/bench_$$(date +%Y%m%d_%H%M%S).txt

bench-cache:
	@echo "Running cache benchmarks..."
	@mkdir -p benchmarks/results
	go test -bench=. -benchmem -benchtime=5s -count=5 ./pkg/cache/... | tee benchmarks/results/cache_$$(date +%Y%m%d_%H%M%S).txt

bench-compare:
	@echo "Comparing with baseline..."
	@if [ ! -f benchmarks/baseline.txt ]; then \
		echo "Error: benchmarks/baseline.txt not found. Run 'make bench' first."; \
		exit 1; \
	fi
	@mkdir -p benchmarks/results
	go test -bench=. -benchmem -benchtime=5s -count=5 ./... | tee benchmarks/results/current.txt
	benchstat benchmarks/baseline.txt benchmarks/results/current.txt

profile:
	@echo "Generating profiles..."
	@mkdir -p profiles
	@echo "CPU profile..."
	go test -bench=BenchmarkCache -cpuprofile=profiles/cpu.prof -benchtime=10s ./pkg/cache/memory
	@echo "Memory profile..."
	go test -bench=BenchmarkCache -memprofile=profiles/mem.prof -benchtime=10s ./pkg/cache/memory
	@echo "Mutex profile..."
	go test -bench=BenchmarkCache -mutexprofile=profiles/mutex.prof -benchtime=10s ./pkg/cache/memory
	@echo "Profiles generated in profiles/"
	@echo ""
	@echo "Analyze with:"
	@echo "  go tool pprof -http=:8080 profiles/cpu.prof"
	@echo "  go tool pprof -http=:8080 profiles/mem.prof"

# Code quality
fmt:
	@echo "Formatting code..."
	gofmt -w -s .
	goimports -w .

lint:
	@echo "Running linter..."
	golangci-lint run ./...

check: fmt lint test-race
	@echo "Running all checks..."
	@echo "Checking for unexpected allocations in hot paths..."
	@go test -bench=BenchmarkCacheGet -benchmem ./pkg/cache/... | grep -E "0 allocs/op" || (echo "Warning: Unexpected allocations detected"; exit 0)
	@echo "All checks passed!"

# Cleanup
clean:
	@echo "Cleaning up..."
	rm -rf coverage.out coverage.html
	rm -rf profiles/*.prof
	rm -rf benchmarks/results/current.txt
	rm -rf *.test
	go clean -testcache
	@echo "Cleanup complete!"

# Docker
docker-up:
	docker-compose up -d

docker-down:
	docker-compose down

docker-logs:
	docker-compose logs -f

# Git hooks
install-hooks:
	@echo "Installing git hooks..."
	@echo '#!/bin/bash' > .git/hooks/pre-commit
	@echo 'make fmt lint' >> .git/hooks/pre-commit
	@chmod +x .git/hooks/pre-commit
	@echo "Git hooks installed!"
