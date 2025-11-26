# GoX WebAssembly (WASM) Guide

This guide explains how to compile and run GoX components in the browser using WebAssembly.

## Overview

GoX supports Client-Side Rendering (CSR) through WebAssembly, allowing you to run Go components directly in the browser with near-native performance.

## Prerequisites

- Go 1.21 or later
- Modern web browser with WebAssembly support (Chrome, Firefox, Safari, Edge)
- Python 3 (for local development server)

## Quick Start

### 1. Create a GoX Component

Create a `.gox` file with your component:

```gox
// examples/wasm/Counter.gox
package main

import "gox"

component Counter(initialCount int) {
    count, setCount := gox.UseState[int](initialCount)

    increment := func() {
        setCount(count + 1)
    }

    decrement := func() {
        setCount(count - 1)
    }

    reset := func() {
        setCount(initialCount)
    }

    render {
        <div className="counter-container">
            <h1>GoX Counter</h1>
            <div className="count-value">{count}</div>
            <div className="counter-controls">
                <button onClick={increment}>+ Increment</button>
                <button onClick={decrement}>- Decrement</button>
                <button onClick={reset}>Reset</button>
            </div>
        </div>
    }

    style {
        .counter-container {
            padding: 2rem;
            text-align: center;
        }

        .count-value {
            font-size: 3rem;
            font-weight: bold;
            margin: 2rem 0;
        }
    }
}
```

### 2. Build for WASM

Compile your component to WASM-compatible Go code:

```bash
# Compile .gox to WASM Go code
goxc build -mode=csr -o=dist examples/wasm/Counter.gox

# Navigate to output directory
cd dist

# Copy WASM runtime support
cp $(go env GOROOT)/lib/wasm/wasm_exec.js ./

# Build WASM binary
GOOS=js GOARCH=wasm go build -o counter.wasm Counter_wasm.go
```

### 3. Create HTML Loader

Create an HTML file to load your WASM component:

```html
<!DOCTYPE html>
<html>
<head>
    <title>GoX WASM Component</title>
    <script src="wasm_exec.js"></script>
</head>
<body>
    <div id="root">Loading...</div>
    <script>
        async function loadComponent() {
            const go = new Go();
            const result = await WebAssembly.instantiateStreaming(
                fetch('counter.wasm'),
                go.importObject
            );

            await go.run(result.instance);

            // Component is now available on window
            if (window.Counter) {
                const counter = window.Counter(0); // Initial count
                counter.Render(document.getElementById('root'));
            }
        }

        loadComponent();
    </script>
</body>
</html>
```

### 4. Serve the Files

Use the provided Python server to serve with correct MIME types:

```bash
# Using the provided server script
python3 serve.py

# Or using Python's built-in server (Python 3.8+)
python3 -m http.server 8080

# Or using Node.js
npx serve -s .
```

Open `http://localhost:8080` in your browser to see your component running!

## Architecture

### CSR Transpilation Process

1. **Parse**: The `.gox` file is parsed into an AST
2. **Analyze**: Components are analyzed and converted to IR
3. **Transpile**: The CSR transpiler generates WASM-compatible Go code
4. **Compile**: Go compiler builds the code to WebAssembly

### Generated Code Structure

The CSR transpiler generates:
- Component struct with state fields
- Virtual DOM creation methods
- Event handlers
- DOM rendering logic
- WASM exports for browser integration

### Runtime Architecture

```
Browser
  ├── wasm_exec.js (Go WASM runtime)
  ├── counter.wasm (Compiled component)
  └── HTML/JS (Loader and container)
```

## WASM Runtime API

### Component Lifecycle

```go
// Component creation
comp := NewCounter(initialValue)

// Rendering
comp.Render(container js.Value)

// Updates trigger re-render
comp.Update()
```

### Virtual DOM

The WASM runtime uses a lightweight Virtual DOM for efficient updates:

```go
type VNode struct {
    Tag      string
    Attrs    map[string]string
    Events   map[string]js.Func
    Children []*VNode
    Text     string
}
```

### Hooks

GoX hooks work in WASM with some limitations:

- ✅ `useState` - Fully supported
- ⚠️ `useEffect` - Limited (no cleanup yet)
- ⚠️ `useMemo` - Basic support
- ⚠️ `useCallback` - Basic support
- ❌ `useContext` - Not yet implemented
- ❌ `useRef` - Not yet implemented

## Advanced Usage

### Multiple Components

Export multiple components from a single WASM module:

```go
func main() {
    js.Global().Set("Counter", js.FuncOf(NewCounterFactory))
    js.Global().Set("TodoList", js.FuncOf(NewTodoListFactory))
    js.Global().Set("Timer", js.FuncOf(NewTimerFactory))

    select {} // Keep program running
}
```

### Passing Props from JavaScript

```javascript
// Pass props to component
const counter = window.Counter({
    initialValue: 10,
    step: 5,
    max: 100
});
```

### Component Communication

```javascript
// Components can expose methods
counter.setValue(42);
counter.increment();

// Or emit events
counter.on('change', (value) => {
    console.log('Counter changed:', value);
});
```

## Performance Optimization

### 1. Minimize DOM Operations

- Batch updates using Virtual DOM
- Use keys for list items
- Memoize expensive computations

### 2. Reduce WASM Size

```bash
# Build with optimization flags
GOOS=js GOARCH=wasm go build -ldflags="-s -w" -o app.wasm

# Further compress with gzip
gzip -9 app.wasm
```

### 3. Lazy Loading

```javascript
// Load WASM on demand
async function loadComponent(name) {
    const module = await import(`./components/${name}.wasm`);
    return module.default;
}
```

## Debugging

### Browser DevTools

1. Open Chrome DevTools
2. Go to Sources tab
3. Find your WASM module
4. Set breakpoints in the WAT view

### Console Logging

```go
import "syscall/js"

func log(args ...interface{}) {
    js.Global().Get("console").Call("log", args...)
}
```

### Error Handling

```go
defer func() {
    if r := recover(); r != nil {
        js.Global().Get("console").Call("error", "Panic:", r)
    }
}()
```

## Limitations

Current limitations of GoX WASM:

1. **File Size**: WASM binaries are larger than equivalent JavaScript
2. **Startup Time**: Initial load can be slower
3. **Browser APIs**: Limited access compared to JavaScript
4. **Debugging**: More difficult than JavaScript
5. **Hot Reload**: Not yet supported

## Best Practices

1. **Keep Components Small**: Smaller components = smaller WASM modules
2. **Use Virtual DOM**: Minimize direct DOM manipulation
3. **Handle Errors**: Always include error boundaries
4. **Progressive Enhancement**: Provide fallbacks for no-WASM support
5. **Cache WASM Files**: Use service workers for caching

## Examples

See the `examples/wasm/` directory for complete examples:

- `Counter` - Basic state management
- `TodoList` - List handling and events
- `Timer` - Async operations
- `DataGrid` - Complex UI with sorting/filtering

## Troubleshooting

### WASM not loading

- Check MIME type is `application/wasm`
- Ensure CORS headers are set correctly
- Verify file paths are correct

### Component not rendering

- Check browser console for errors
- Verify component is exported correctly
- Ensure DOM element exists

### Performance issues

- Use browser profiler
- Check for memory leaks
- Optimize Virtual DOM updates

## Future Enhancements

Planned improvements for GoX WASM:

- [ ] Hot module replacement (HMR)
- [ ] Source maps for better debugging
- [ ] Tree shaking for smaller bundles
- [ ] Web Workers support
- [ ] Shared memory between components
- [ ] Full hooks API support
- [ ] SSR to CSR hydration

## Resources

- [WebAssembly MDN](https://developer.mozilla.org/en-US/docs/WebAssembly)
- [Go WebAssembly Wiki](https://github.com/golang/go/wiki/WebAssembly)
- [GoX Documentation](https://github.com/user/gox)
- [WASM Performance Tips](https://webassembly.org/docs/performance/)

## Contributing

We welcome contributions to improve GoX WASM support! Please see our [Contributing Guide](../CONTRIBUTING.md) for details.