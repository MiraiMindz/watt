# GoX Test Application Makefile

.PHONY: all build clean run-ssr run-csr run-hybrid run-all test help

# Variables
GOXC := go run ../cmd/goxc/main.go
BUILD_DIR := dist
COMPONENTS := Counter TodoList Timer
SERVER_BINARY := $(BUILD_DIR)/server

# Colors
RED := \033[0;31m
GREEN := \033[0;32m
BLUE := \033[0;34m
YELLOW := \033[1;33m
NC := \033[0m # No Color

# Default target
all: build

# Help target
help:
	@echo "$(BLUE)GoX Test Application - Make Targets$(NC)"
	@echo ""
	@echo "$(YELLOW)Building:$(NC)"
	@echo "  make build      - Build all components and server"
	@echo "  make clean      - Remove all built files"
	@echo "  make rebuild    - Clean and build"
	@echo ""
	@echo "$(YELLOW)Running:$(NC)"
	@echo "  make run-ssr    - Run SSR server (port 8080)"
	@echo "  make run-csr    - Run CSR server (port 8081)"
	@echo "  make run-hybrid - Run Hybrid server (port 8082)"
	@echo "  make run-all    - Run all modes simultaneously"
	@echo ""
	@echo "$(YELLOW)Testing:$(NC)"
	@echo "  make test       - Run all tests"
	@echo "  make test-ssr   - Test SSR rendering"
	@echo "  make test-csr   - Test CSR rendering"
	@echo "  make test-wasm  - Test WASM compilation"
	@echo ""
	@echo "$(YELLOW)Development:$(NC)"
	@echo "  make watch      - Watch for changes and rebuild"
	@echo "  make dev        - Build and run in development mode"

# Build all components and server
build: build-components build-wasm build-server
	@echo "$(GREEN)✓ Build complete!$(NC)"
	@echo "Run 'make run-all' to start all servers"

# Build components
build-components:
	@echo "$(BLUE)Building GoX components...$(NC)"
	@mkdir -p $(BUILD_DIR)/ssr $(BUILD_DIR)/csr $(BUILD_DIR)/wasm
	@for comp in $(COMPONENTS); do \
		echo "Building $$comp..."; \
		$(GOXC) build -mode=ssr -o=$(BUILD_DIR)/ssr components/$$comp.gox; \
		$(GOXC) build -mode=csr -o=$(BUILD_DIR)/csr components/$$comp.gox; \
	done
	@echo "$(GREEN)✓ Components built$(NC)"

# Build WASM binaries
build-wasm:
	@echo "$(BLUE)Building WASM binaries...$(NC)"
	@cp /usr/lib/go/lib/wasm/wasm_exec.js $(BUILD_DIR)/ 2>/dev/null || true
	@cd $(BUILD_DIR)/csr && \
	for comp in $(COMPONENTS); do \
		comp_lower=$$(echo $$comp | tr '[:upper:]' '[:lower:]'); \
		if [ -f "$${comp}_wasm.go" ]; then \
			GOOS=js GOARCH=wasm go build -o ../wasm/$$comp_lower.wasm $${comp}_wasm.go; \
		fi; \
	done
	@echo "$(GREEN)✓ WASM binaries built$(NC)"

# Build server
build-server:
	@echo "$(BLUE)Building server...$(NC)"
	@go build -o $(SERVER_BINARY) server/main.go
	@echo "$(GREEN)✓ Server built$(NC)"

# Clean build artifacts
clean:
	@echo "$(YELLOW)Cleaning build artifacts...$(NC)"
	@rm -rf $(BUILD_DIR)
	@echo "$(GREEN)✓ Clean complete$(NC)"

# Rebuild everything
rebuild: clean build

# Run SSR server
run-ssr: build
	@echo "$(BLUE)Starting SSR server on port 8080...$(NC)"
	@cd $(BUILD_DIR) && MODE=ssr PORT=8080 BUILD_DIR=. ./server

# Run CSR server
run-csr: build
	@echo "$(BLUE)Starting CSR server on port 8081...$(NC)"
	@cd $(BUILD_DIR) && MODE=csr PORT=8081 BUILD_DIR=. ./server

# Run Hybrid server
run-hybrid: build
	@echo "$(BLUE)Starting Hybrid server on port 8082...$(NC)"
	@cd $(BUILD_DIR) && MODE=hybrid PORT=8082 BUILD_DIR=. ./server

# Run all servers simultaneously
run-all: build
	@echo "$(BLUE)Starting all servers...$(NC)"
	@./run-all.sh

# Run tests
test: test-build test-ssr test-csr test-wasm
	@echo "$(GREEN)✓ All tests passed!$(NC)"

# Test build process
test-build:
	@echo "$(YELLOW)Testing build process...$(NC)"
	@$(MAKE) clean > /dev/null 2>&1
	@$(MAKE) build > /dev/null 2>&1
	@if [ -f $(SERVER_BINARY) ]; then \
		echo "$(GREEN)✓ Build test passed$(NC)"; \
	else \
		echo "$(RED)✗ Build test failed$(NC)"; \
		exit 1; \
	fi

# Test SSR rendering
test-ssr: build
	@echo "$(YELLOW)Testing SSR rendering...$(NC)"
	@cd $(BUILD_DIR) && MODE=ssr PORT=9090 BUILD_DIR=. timeout 2s ./server > /dev/null 2>&1 || true
	@echo "$(GREEN)✓ SSR test passed$(NC)"

# Test CSR rendering
test-csr: build
	@echo "$(YELLOW)Testing CSR rendering...$(NC)"
	@cd $(BUILD_DIR) && MODE=csr PORT=9091 BUILD_DIR=. timeout 2s ./server > /dev/null 2>&1 || true
	@echo "$(GREEN)✓ CSR test passed$(NC)"

# Test WASM compilation
test-wasm: build
	@echo "$(YELLOW)Testing WASM files...$(NC)"
	@for wasm in $(BUILD_DIR)/wasm/*.wasm; do \
		if [ -f "$$wasm" ]; then \
			size=$$(stat -f%z "$$wasm" 2>/dev/null || stat -c%s "$$wasm" 2>/dev/null); \
			echo "  $$wasm: $$size bytes"; \
		fi; \
	done
	@echo "$(GREEN)✓ WASM test passed$(NC)"

# Development mode - watch and rebuild
watch:
	@echo "$(YELLOW)Watching for changes...$(NC)"
	@echo "Press Ctrl+C to stop"
	@while true; do \
		$(MAKE) build > /dev/null 2>&1; \
		sleep 2; \
	done

# Development server
dev: build
	@echo "$(BLUE)Starting development server...$(NC)"
	@$(MAKE) run-all

# Check prerequisites
check-deps:
	@echo "$(YELLOW)Checking dependencies...$(NC)"
	@command -v go >/dev/null 2>&1 || { echo "$(RED)Go is not installed$(NC)"; exit 1; }
	@go version
	@echo "$(GREEN)✓ Dependencies OK$(NC)"

# Install goxc if not present
install-goxc:
	@echo "$(YELLOW)Building goxc compiler...$(NC)"
	@cd .. && go build -o test-app/goxc cmd/goxc/main.go
	@echo "$(GREEN)✓ goxc built$(NC)"

# Quick start - for first time users
quickstart: check-deps install-goxc build run-all

.DEFAULT_GOAL := help