#!/bin/bash

# GoX Test Application Build Script
# Builds all components for SSR, CSR, and Hybrid modes

set -e

# Colors for output
RED='\033[0;31m'
GREEN='\033[0;32m'
BLUE='\033[0;34m'
YELLOW='\033[1;33m'
NC='\033[0m' # No Color

# Configuration
GOXC="../cmd/goxc/main.go"
COMPONENTS_DIR="components"
BUILD_DIR="dist"
WASM_DIR="$BUILD_DIR/wasm"

# Print header
echo -e "${BLUE}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${BLUE}║          GoX Test Application Build Script            ║${NC}"
echo -e "${BLUE}╚══════════════════════════════════════════════════════╝${NC}"
echo ""

# Check if goxc exists
if [ ! -f "$GOXC" ]; then
    echo -e "${RED}Error: goxc compiler not found at $GOXC${NC}"
    echo "Please build goxc first: go build cmd/goxc/main.go"
    exit 1
fi

# Create build directories
echo -e "${YELLOW}Creating build directories...${NC}"
mkdir -p "$BUILD_DIR"
mkdir -p "$WASM_DIR"
mkdir -p "$BUILD_DIR/ssr"
mkdir -p "$BUILD_DIR/csr"

# Function to build a component
build_component() {
    local component=$1
    local mode=$2
    local output_dir=$3

    echo -e "${BLUE}Building $component in $mode mode...${NC}"

    if [ "$mode" = "ssr" ]; then
        go run $GOXC build -mode=ssr -o="$output_dir" "$COMPONENTS_DIR/$component.gox"
    else
        go run $GOXC build -mode=csr -o="$output_dir" "$COMPONENTS_DIR/$component.gox"
    fi

    if [ $? -eq 0 ]; then
        echo -e "${GREEN}✓ $component built successfully${NC}"
    else
        echo -e "${RED}✗ Failed to build $component${NC}"
        exit 1
    fi
}

# Build all components
echo ""
echo -e "${YELLOW}Building components...${NC}"
echo ""

# List of components to build
COMPONENTS=("Counter" "TodoList" "Timer")

# Build SSR versions
echo -e "${BLUE}═══ Building SSR Components ═══${NC}"
for comp in "${COMPONENTS[@]}"; do
    build_component "$comp" "ssr" "$BUILD_DIR/ssr"
done

# Build CSR versions
echo ""
echo -e "${BLUE}═══ Building CSR/WASM Components ═══${NC}"
for comp in "${COMPONENTS[@]}"; do
    build_component "$comp" "csr" "$BUILD_DIR/csr"
done

# Compile to WASM
echo ""
echo -e "${YELLOW}Compiling to WebAssembly...${NC}"

# Copy wasm_exec.js
if [ -f "/usr/lib/go/lib/wasm/wasm_exec.js" ]; then
    cp /usr/lib/go/lib/wasm/wasm_exec.js "$BUILD_DIR/"
    echo -e "${GREEN}✓ Copied wasm_exec.js${NC}"
else
    echo -e "${YELLOW}Warning: wasm_exec.js not found in standard location${NC}"
    echo "You may need to copy it manually from: $(go env GOROOT)/lib/wasm/wasm_exec.js"
fi

# Build WASM binaries
cd "$BUILD_DIR/csr"
for comp in "${COMPONENTS[@]}"; do
    go_file="${comp}_wasm.go"
    wasm_file="../wasm/${comp,,}.wasm"  # Convert to lowercase

    if [ -f "$go_file" ]; then
        echo -e "${BLUE}Compiling $comp to WASM...${NC}"
        GOOS=js GOARCH=wasm go build -o "$wasm_file" "$go_file"

        if [ $? -eq 0 ]; then
            echo -e "${GREEN}✓ Created $wasm_file${NC}"
        else
            echo -e "${RED}✗ Failed to compile $comp to WASM${NC}"
        fi
    fi
done
cd ../..

# Build the server
echo ""
echo -e "${YELLOW}Building server...${NC}"
go build -o "$BUILD_DIR/server" server/main.go

if [ $? -eq 0 ]; then
    echo -e "${GREEN}✓ Server built successfully${NC}"
else
    echo -e "${RED}✗ Failed to build server${NC}"
    exit 1
fi

# Create run scripts
echo ""
echo -e "${YELLOW}Creating run scripts...${NC}"

# SSR mode script
cat > "$BUILD_DIR/run-ssr.sh" << 'EOF'
#!/bin/bash
export MODE=ssr
export PORT=8080
export BUILD_DIR=.
echo "Starting server in SSR mode on port 8080..."
./server
EOF
chmod +x "$BUILD_DIR/run-ssr.sh"

# CSR mode script
cat > "$BUILD_DIR/run-csr.sh" << 'EOF'
#!/bin/bash
export MODE=csr
export PORT=8081
export BUILD_DIR=.
echo "Starting server in CSR mode on port 8081..."
./server
EOF
chmod +x "$BUILD_DIR/run-csr.sh"

# Hybrid mode script
cat > "$BUILD_DIR/run-hybrid.sh" << 'EOF'
#!/bin/bash
export MODE=hybrid
export PORT=8082
export BUILD_DIR=.
echo "Starting server in Hybrid mode on port 8082..."
./server
EOF
chmod +x "$BUILD_DIR/run-hybrid.sh"

echo -e "${GREEN}✓ Created run scripts${NC}"

# Print summary
echo ""
echo -e "${GREEN}╔══════════════════════════════════════════════════════╗${NC}"
echo -e "${GREEN}║               Build Complete!                         ║${NC}"
echo -e "${GREEN}╚══════════════════════════════════════════════════════╝${NC}"
echo ""
echo -e "${BLUE}Build Summary:${NC}"
echo -e "  • Components built: ${#COMPONENTS[@]}"
echo -e "  • SSR components in: $BUILD_DIR/ssr/"
echo -e "  • CSR components in: $BUILD_DIR/csr/"
echo -e "  • WASM files in: $WASM_DIR/"
echo -e "  • Server binary: $BUILD_DIR/server"
echo ""
echo -e "${BLUE}To run the test application:${NC}"
echo -e "  ${YELLOW}SSR Mode:${NC}    cd $BUILD_DIR && ./run-ssr.sh"
echo -e "  ${YELLOW}CSR Mode:${NC}    cd $BUILD_DIR && ./run-csr.sh"
echo -e "  ${YELLOW}Hybrid Mode:${NC} cd $BUILD_DIR && ./run-hybrid.sh"
echo ""
echo -e "${BLUE}Or run all modes simultaneously:${NC}"
echo -e "  ./run-all.sh"
echo ""