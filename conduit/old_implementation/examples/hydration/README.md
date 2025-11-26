# GoX SSR to CSR Hydration Example

This example demonstrates how to use GoX's hydration system to enhance server-rendered HTML with client-side interactivity.

## What is Hydration?

Hydration is the process of:
1. Rendering HTML on the server (SSR)
2. Sending that HTML to the browser immediately (fast initial load)
3. Loading the JavaScript/WASM component
4. "Hydrating" the existing HTML with event listeners and state

## Benefits

- ✅ **Fast Initial Load**: Users see content immediately
- ✅ **SEO Friendly**: Search engines can crawl the content
- ✅ **Progressive Enhancement**: Works without JavaScript
- ✅ **Interactive**: Full interactivity once hydrated

## Architecture

```
Server (Go)                 Browser
    │                           │
    ├─ Render HTML ────────────>│ Display immediately
    │                           │
    ├─ Send WASM ──────────────>│ Load in background
    │                           │
    └─ Hydration Data ─────────>│ Hydrate DOM
                                │
                                └─ Fully Interactive
```

## Running the Example

### 1. Build the WASM Component

```bash
# Build the Counter component for WASM
cd examples/hydration
goxc build -mode=csr -o=dist ../wasm/Counter.gox

# Copy WASM runtime
cp $(go env GOROOT)/lib/wasm/wasm_exec.js dist/

# Build WASM binary
cd dist
GOOS=js GOARCH=wasm go build -o counter.wasm Counter_wasm.go
cd ..
```

### 2. Start the SSR Server

```bash
# Run the hydration server
go run server.go
```

### 3. Test the Hydration

1. Open http://localhost:8080
2. View page source - you'll see the server-rendered HTML
3. Open DevTools Console - you'll see hydration messages
4. Click buttons - they work after hydration

### 4. Test Progressive Enhancement

1. Disable JavaScript in your browser
2. Reload the page
3. The content is still visible (server-rendered)
4. Buttons won't work without JavaScript (expected)

## How It Works

### Server Side (server.go)

```go
// 1. Create component with initial state
component := &CounterComponent{
    Count: 42,
    Title: "SSR Counter",
}

// 2. Render with hydration data
html := renderer.RenderWithHydration(component, props)

// 3. Serve HTML with embedded hydration script
```

### Client Side (Counter_wasm.go)

```go
// 1. Component supports hydration
func (c *Counter) Hydrate(container js.Value) error {
    // Extract server state
    // Attach event listeners
    // Don't re-render content
}

// 2. Regular render for CSR
func (c *Counter) Render(container js.Value) {
    // Full client-side render
}
```

### Hydration Script (in HTML)

```html
<!-- Server-rendered content -->
<div id="root" data-gox-component="gox-123">
    <div class="counter">
        <h1>Counter</h1>
        <div>Count: 42</div>
        <button>Increment</button>
    </div>
</div>

<!-- Hydration data -->
<script type="application/gox-hydration">
{
    "componentID": "gox-123",
    "props": {"initialCount": 42},
    "state": {"count": 42}
}
</script>

<!-- Hydration logic -->
<script>
// Load WASM component
// Extract hydration data
// Call component.Hydrate()
// Attach event listeners
</script>
```

## Hydration Flow

1. **Server Render**
   - Component renders to HTML string
   - State is serialized to JSON
   - HTML includes hydration markers

2. **Initial Load**
   - Browser displays HTML immediately
   - Page is readable but not interactive

3. **WASM Load**
   - WebAssembly module loads in background
   - Component constructor registers on window

4. **Hydration**
   - Extract state from hydration script
   - Create component with server state
   - Walk DOM and attach event listeners
   - Mark nodes as hydrated

5. **Interactive**
   - Component is fully interactive
   - Updates use Virtual DOM diffing
   - State changes trigger re-renders

## Best Practices

### DO ✅

- Render the same content on server and client
- Include fallback behavior for no-JS users
- Use semantic HTML for accessibility
- Cache rendered HTML when possible
- Minimize hydration data size

### DON'T ❌

- Change content during hydration (causes flicker)
- Rely on client-only features during SSR
- Include sensitive data in hydration script
- Block rendering waiting for hydration

## Debugging

### Check Hydration Status

```javascript
// In browser console
document.querySelectorAll('[data-gox-hydrated]')
// Should show all hydrated components
```

### Hydration Mismatches

If server and client render differently:
1. Check initial props match
2. Verify state serialization
3. Ensure deterministic rendering
4. Check for client-only code

### Performance

```javascript
// Measure hydration time
performance.mark('hydration-start');
await component.Hydrate(container);
performance.mark('hydration-end');
performance.measure('hydration', 'hydration-start', 'hydration-end');
```

## Advanced Topics

### Partial Hydration

Only hydrate interactive components:

```go
if component.NeedsHydration() {
    component.Hydrate(container)
} else {
    // Keep as static HTML
}
```

### Lazy Hydration

Hydrate on user interaction:

```javascript
container.addEventListener('mouseenter', () => {
    if (!container.hydrated) {
        component.Hydrate(container);
    }
}, { once: true });
```

### Streaming SSR

Send HTML as it's generated:

```go
w.Header().Set("Transfer-Encoding", "chunked")
w.Write([]byte(headerHTML))
w.Flush()

w.Write([]byte(componentHTML))
w.Flush()

w.Write([]byte(footerHTML))
```

## Troubleshooting

### "Hydration failed"
- Check browser console for specific errors
- Verify WASM file is loading
- Ensure hydration data is valid JSON

### Event handlers not working
- Verify hydration completed
- Check for JavaScript errors
- Ensure event names match

### Content flickers on hydration
- Server and client render differently
- Check for non-deterministic content
- Verify props and state match

## Next Steps

- Implement streaming SSR
- Add error boundaries
- Cache rendered components
- Optimize hydration performance
- Support partial hydration