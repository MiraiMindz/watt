#!/bin/bash

# Run all GoX test application modes simultaneously
# Each mode runs on a different port

set -e

# Colors
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
RED='\033[0;31m'
NC='\033[0m'

# Check if server is built
if [ ! -f "dist/server" ]; then
    echo -e "${RED}Error: Server not found. Please run ./build.sh first${NC}"
    exit 1
fi

# Function to kill all servers on exit
cleanup() {
    echo ""
    echo -e "${YELLOW}Stopping all servers...${NC}"
    kill $(jobs -p) 2>/dev/null
    echo -e "${GREEN}All servers stopped${NC}"
}

# Register cleanup function
trap cleanup EXIT

# Print header
echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║      GoX Test Application - Multi-Mode Runner         ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Start SSR server
echo -e "${YELLOW}Starting SSR server on port 8080...${NC}"
cd dist
MODE=ssr PORT=8080 BUILD_DIR=. ./server > /dev/null 2>&1 &
SSR_PID=$!
cd ..

# Start CSR server
echo -e "${YELLOW}Starting CSR server on port 8081...${NC}"
cd dist
MODE=csr PORT=8081 BUILD_DIR=. ./server > /dev/null 2>&1 &
CSR_PID=$!
cd ..

# Start Hybrid server
echo -e "${YELLOW}Starting Hybrid server on port 8082...${NC}"
cd dist
MODE=hybrid PORT=8082 BUILD_DIR=. ./server > /dev/null 2>&1 &
HYBRID_PID=$!
cd ..

# Wait a moment for servers to start
sleep 2

# Check if servers are running
if kill -0 $SSR_PID 2>/dev/null && kill -0 $CSR_PID 2>/dev/null && kill -0 $HYBRID_PID 2>/dev/null; then
    echo ""
    echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
    echo -e "${GREEN}║           All Servers Running Successfully!           ║${NC}"
    echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
    echo ""
    echo -e "${BLUE}Access the applications at:${NC}"
    echo ""
    echo -e "  ${YELLOW}SSR Mode:${NC}    http://localhost:8080"
    echo -e "               View source to see server-rendered HTML"
    echo ""
    echo -e "  ${YELLOW}CSR Mode:${NC}    http://localhost:8081"
    echo -e "               Components rendered by WebAssembly"
    echo ""
    echo -e "  ${YELLOW}Hybrid Mode:${NC} http://localhost:8082"
    echo -e "               SSR with client-side hydration"
    echo ""
    echo -e "${BLUE}Test different features:${NC}"
    echo -e "  • Counter:   /counter"
    echo -e "  • Todo List: /todo"
    echo -e "  • Timer:     /timer"
    echo -e "  • Dashboard: /dashboard"
    echo ""
    echo -e "${BLUE}API Endpoints:${NC}"
    echo -e "  • Component List: /api/components"
    echo -e "  • Health Check:   /health"
    echo ""
    echo -e "${YELLOW}Press Ctrl+C to stop all servers${NC}"
    echo ""

    # Keep script running
    wait
else
    echo -e "${RED}Error: One or more servers failed to start${NC}"
    exit 1
fi