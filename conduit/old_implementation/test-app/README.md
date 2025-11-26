# GoX Test Application

A comprehensive test application demonstrating all features of the GoX framework including custom components, Server-Side Rendering (SSR), Client-Side Rendering (CSR), and hydration.

## 📋 Table of Contents

- [Overview](#overview)
- [Components](#components)
- [Quick Start](#quick-start)
- [Testing Guide](#testing-guide)
- [Architecture](#architecture)
- [Performance Testing](#performance-testing)
- [Troubleshooting](#troubleshooting)

## Overview

This test application showcases:

- ✅ **Custom GoX Components** - Counter, TodoList, Timer
- ✅ **SSR (Server-Side Rendering)** - HTML generated on server
- ✅ **CSR (Client-Side Rendering)** - WebAssembly in browser
- ✅ **Hybrid Mode** - SSR with client-side hydration
- ✅ **React-like Hooks** - useState, useEffect, useMemo, useRef
- ✅ **Component Composition** - Dashboard combining multiple components
- ✅ **Performance Monitoring** - Built-in metrics and logging

## Components

### 1. **Counter Component** (`components/Counter.gox`)
- State management with `useState`
- Computed values with `useMemo`
- Side effects with `useEffect`
- Custom step increments
- Props: `initialCount`, `step`

### 2. **Todo List Component** (`components/TodoList.gox`)
- Dynamic list management
- Filter by status (All/Active/Completed)
- Bulk operations (Clear completed)
- Keyboard shortcuts (Enter to add)
- Complex state handling

### 3. **Timer Component** (`components/Timer.gox`)
- Start/Pause/Reset functionality
- Lap time recording
- Async operations with intervals
- Effect cleanup on unmount
- `useRef` for persistent values

### 4. **Dashboard** (Combined view)
- All components in one page
- Tests performance with multiple components
- Shared state demonstration

## Quick Start

### Prerequisites

- Go 1.21 or later
- Bash shell
- Modern web browser

### Build Everything

```bash
# Make scripts executable
chmod +x build.sh run-all.sh

# Build all components and server
./build.sh
```

This will:
1. Build all components in SSR mode
2. Build all components in CSR mode
3. Compile WASM binaries
4. Build the test server
5. Create run scripts

### Run Test Application

#### Option 1: Run All Modes Simultaneously

```bash
./run-all.sh
```

This starts:
- SSR server on http://localhost:8080
- CSR server on http://localhost:8081
- Hybrid server on http://localhost:8082

#### Option 2: Run Individual Modes

```bash
# Server-Side Rendering
cd dist && ./run-ssr.sh

# Client-Side Rendering (WASM)
cd dist && ./run-csr.sh

# Hybrid (SSR + Hydration)
cd dist && ./run-hybrid.sh
```

## Testing Guide

### 1. Testing SSR Mode

Visit http://localhost:8080

**What to test:**
- View page source - HTML should be fully rendered
- Disable JavaScript - content should still be visible
- Check initial load time - should be fast
- Test SEO - content should be crawlable

**Expected behavior:**
- Immediate content display
- No loading spinners
- Full HTML in page source
- Works without JavaScript (view only)

### 2. Testing CSR Mode

Visit http://localhost:8081

**What to test:**
- Open DevTools Network tab
- Watch WASM file loading
- Check console for component logs
- Test interactivity

**Expected behavior:**
- Loading message initially
- WASM file downloaded (~2MB)
- Component renders client-side
- Full interactivity

### 3. Testing Hybrid Mode

Visit http://localhost:8082

**What to test:**
- Initial content display (SSR)
- Hydration process in console
- Interactivity after hydration
- No content flicker

**Expected behavior:**
- Immediate content (SSR)
- Background WASM loading
- Smooth hydration
- No layout shifts

### 4. Component-Specific Tests

#### Counter Component (`/counter`)
```javascript
// Test in console:
// 1. Click increment - count should increase
// 2. Click decrement - count should decrease
// 3. Click reset - returns to initial value
// 4. Check console for effect logs
```

#### Todo List (`/todo`)
```javascript
// Test scenarios:
// 1. Add new todo (type + Enter)
// 2. Toggle completion (checkbox)
// 3. Filter by status (All/Active/Completed)
// 4. Delete individual todos
// 5. Clear all completed
```

#### Timer (`/timer`)
```javascript
// Test scenarios:
// 1. Start timer - seconds increment
// 2. Pause - stops counting
// 3. Resume - continues from paused time
// 4. Lap - records current time
// 5. Reset - returns to 00:00:00
```

### 5. Performance Testing

#### Metrics to Monitor

Open DevTools Performance tab and record:

1. **First Contentful Paint (FCP)**
   - SSR: < 500ms
   - CSR: 1-2s (includes WASM load)
   - Hybrid: < 500ms

2. **Time to Interactive (TTI)**
   - SSR: N/A (no interactivity)
   - CSR: 2-3s
   - Hybrid: 2-3s (progressive)

3. **Bundle Sizes**
   ```bash
   ls -lh dist/wasm/
   # Each component ~2MB (WASM overhead)
   ```

4. **Memory Usage**
   - Check Chrome Task Manager
   - Monitor during interactions

## Architecture

### Directory Structure
```
test-app/
├── components/          # GoX components
│   ├── Counter.gox
│   ├── TodoList.gox
│   └── Timer.gox
├── server/             # Test server
│   └── main.go
├── dist/               # Build output
│   ├── ssr/           # SSR compiled Go files
│   ├── csr/           # CSR compiled Go files
│   ├── wasm/          # WASM binaries
│   ├── server         # Server binary
│   └── wasm_exec.js   # WASM runtime
├── build.sh           # Build script
├── run-all.sh        # Multi-mode runner
└── README.md         # This file
```

### Request Flow

#### SSR Mode
```
Browser → Server → GoX SSR Renderer → HTML → Browser
```

#### CSR Mode
```
Browser → Server → HTML Shell → Load WASM → Render in Browser
```

#### Hybrid Mode
```
Browser → Server → SSR HTML → Browser Display → Load WASM → Hydrate DOM
```

## API Endpoints

### Component List
```bash
curl http://localhost:8080/api/components
# Returns: ["Counter", "TodoList", "Timer"]
```

### Health Check
```bash
curl http://localhost:8080/health
# Returns: {"status":"healthy","mode":"ssr","port":"8080"}
```

### Component State (Mock)
```bash
curl http://localhost:8080/api/state/Counter
# Returns component state information
```

## Advanced Testing

### Browser Compatibility

Test in multiple browsers:
- Chrome/Edge (Chromium)
- Firefox
- Safari
- Mobile browsers

### Network Conditions

Use Chrome DevTools Network throttling:
- Fast 3G
- Slow 3G
- Offline (SSR should show content)

### Error Scenarios

1. **WASM Load Failure**
   - Block .wasm files in DevTools
   - SSR/Hybrid should still show content

2. **JavaScript Disabled**
   - Disable JS in browser
   - SSR content should be visible

3. **Component Errors**
   - Introduce errors in components
   - Test error boundaries

## Performance Optimization

### Reduce WASM Size

```bash
# Build with optimization
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o app.wasm

# Compress with gzip
gzip -9 app.wasm
```

### Enable Caching

```go
// In server/main.go
w.Header().Set("Cache-Control", "public, max-age=3600")
```

### Lazy Loading

Only load components when needed:

```javascript
// Lazy load component
if (route === '/counter') {
    loadComponent('counter');
}
```

## Troubleshooting

### Common Issues

#### "WASM not loading"
- Check browser console for errors
- Verify MIME type is `application/wasm`
- Ensure wasm_exec.js is loaded

#### "Component not rendering"
- Check if component is exported to window
- Verify props are passed correctly
- Look for JavaScript errors

#### "Hydration mismatch"
- Server and client render differently
- Check for non-deterministic content
- Ensure same props on both sides

#### "Build failures"
- Ensure goxc is built: `go build cmd/goxc/main.go`
- Check Go version: `go version` (need 1.21+)
- Verify component syntax

### Debug Mode

Enable detailed logging:

```bash
# Set debug environment variable
DEBUG=true ./run-all.sh
```

Check browser console for:
- Component lifecycle logs
- State changes
- Render timings
- Hydration status

## Contributing

To add new test components:

1. Create component in `components/`
2. Update `build.sh` COMPONENTS array
3. Add route in `server/main.go`
4. Update this README

## Next Steps

After testing the basic functionality:

1. **Performance Profiling** - Use Chrome DevTools
2. **Memory Analysis** - Check for leaks
3. **Accessibility Testing** - Screen readers, keyboard nav
4. **Cross-browser Testing** - Safari, Firefox, Edge
5. **Mobile Testing** - Responsive design, touch events
6. **Production Build** - Minification, compression
7. **Deployment** - Docker, cloud platforms

## Resources

- [GoX Documentation](../README.md)
- [WASM Guide](../docs/wasm-guide.md)
- [SSR Documentation](../docs/serving-components.md)
- [WebAssembly MDN](https://developer.mozilla.org/en-US/docs/WebAssembly)

## License

MIT - See main project LICENSE