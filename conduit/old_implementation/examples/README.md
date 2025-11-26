# GoX Examples

This directory contains example applications demonstrating GoX features and capabilities.

---

## 📁 Examples

### 1. Counter (SSR)
**Location**: `counter-ssr/`

A simple counter component compiled to Server-Side Rendering (SSR) mode.

**Features:**
- ✅ useState hook
- ✅ Event handlers (non-interactive in SSR)
- ✅ Scoped styles
- ✅ Component props

**Run:**
```bash
cd counter-ssr
go run .
# Visit http://localhost:8080
```

### 2. Counter (WASM)
**Location**: `counter-wasm/`

The same counter component compiled to WebAssembly for client-side rendering.

**Features:**
- ✅ useState hook (reactive)
- ✅ useEffect hook
- ✅ Interactive event handlers
- ✅ Virtual DOM updates
- ✅ Browser console logging

**Build & Run:**
```bash
cd counter-wasm

# Copy wasm_exec.js from Go installation
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .

# Build WASM (once GoX is implemented)
# goxc build --mode=csr Counter.gox
# GOOS=js GOARCH=wasm go build -o counter.wasm .

# Serve
python3 -m http.server 8080
# Visit http://localhost:8080
```

---

## 🔄 SSR vs CSR Comparison

### Same Source Code

Both examples use nearly identical `.gox` source code. The only differences:

**SSR Version:**
```gox
component Counter(initialValue int) {
    count, setCount := gox.UseState[int](initialValue)

    // Non-interactive event handlers
    increment := func() {
        setCount(count + 1)
    }

    render { /* JSX */ }
    style { /* CSS */ }
}
```

**WASM Version:**
```gox
component Counter(initialValue int) {
    count, setCount := gox.UseState[int](initialValue)

    // Interactive event handlers with js.Value
    increment := func(e js.Value) {
        setCount(count + 1)
    }

    // Same render and style
    render { /* JSX */ }
    style { /* CSS */ }
}
```

### Different Output

**SSR Output** (`counter-ssr/counter_generated.go`):
```go
type Counter struct {
    initialValue int
    count int
}

func (c *Counter) Render() string {
    return fmt.Sprintf(`<div>...</div>`, c.count)
}
```

**WASM Output** (would be generated):
```go
func (c *Counter) Render() *gox.VNode {
    return gox.H("div", gox.Props{...},
        gox.H("button", gox.Props{
            "onClick": c.increment,
        }))
}
```

---

## 📚 What Each Example Teaches

### Counter SSR
- How to create a GoX component
- Using useState for state management
- Writing JSX markup
- Scoped CSS styling
- Compiling to SSR
- Serving with Go HTTP server

### Counter WASM
- Same component in CSR mode
- Interactive event handling
- useEffect for side effects
- Virtual DOM updates
- WASM compilation
- Browser integration

---

## 🎯 Future Examples

### Coming Soon

#### Todo App (SSR + CSR)
- Multiple components
- List rendering
- Form handling
- Local storage persistence

#### Blog (SSR)
- Multiple pages
- Routing
- Markdown rendering
- SEO optimization

#### Dashboard (WASM)
- Real-time data
- Charts and graphs
- WebSocket integration
- Complex state management

#### Full-Stack App (Hybrid)
- SSR for initial render
- WASM for interactivity
- API integration
- Authentication

---

## 🏗️ Project Structure

Each example follows this structure:

```
example-name/
├── Component.gox          # GoX source (what you write)
├── component_generated.go # Generated Go code (from goxc)
├── main.go               # Entry point
├── index.html            # HTML template (WASM only)
├── styles.css            # Generated styles
├── go.mod                # Go module
└── README.md             # Example-specific docs
```

---

## 🔨 Building Examples

### Prerequisites

```bash
# Install Go 1.21+
go version

# Install GoX compiler (once available)
go install github.com/yourusername/gox/cmd/goxc@latest
```

### SSR Mode

```bash
# Compile .gox to .go
goxc build --mode=ssr -o . Component.gox

# Run Go program
go run .
```

### CSR Mode (WASM)

```bash
# Compile .gox to .go with WASM support
goxc build --mode=csr -o . Component.gox

# Build WASM binary
GOOS=js GOARCH=wasm go build -o app.wasm .

# Copy wasm_exec.js
cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" .

# Serve with any static server
python3 -m http.server 8080
# or
goxc dev --port 8080
```

---

## 📖 Learning Path

Recommended order for learning GoX:

1. **Counter SSR** - Basics of components and SSR
2. **Counter WASM** - Understanding CSR and interactivity
3. **Todo App** - Multiple components and composition
4. **Blog** - Real-world SSR application
5. **Dashboard** - Complex WASM application
6. **Full-Stack** - Combining SSR and CSR

---

## 🐛 Troubleshooting

### SSR Examples

**"component keyword not recognized"**
- Make sure goxc is installed
- Check file has `.gox` extension
- Verify you're running `goxc build`

**Generated code doesn't compile**
- Report as bug - this shouldn't happen
- Check generated code for syntax errors
- Try cleaning and rebuilding

### WASM Examples

**"Failed to load WASM"**
- Ensure WASM file is built: `GOOS=js GOARCH=wasm go build`
- Check file exists: `ls -lh *.wasm`
- Verify wasm_exec.js is present
- Check browser console for errors

**"WASM loaded but nothing happens"**
- Check browser console for Go output
- Ensure #root element exists in HTML
- Verify WASM entry point calls Mount()

**Buttons don't work**
- This is expected if using placeholder code
- Full implementation needs reconciler
- Check that event handlers are attached

### Build Errors

**"cannot find package gox/runtime"**
- Install GoX runtime: `go get github.com/yourusername/gox/runtime`
- Verify go.mod has correct dependencies

**"undefined: gox.UseState"**
- Runtime not imported correctly
- Check import: `import "gox"`
- Verify runtime implementation exists

---

## 💡 Tips & Best Practices

### Performance

1. **SSR Mode**
   - Pre-render static content
   - Cache rendered HTML
   - Use template pooling

2. **WASM Mode**
   - Use TinyGo for smaller bundles
   - Lazy load components
   - Memoize expensive computations

### Development

1. **Use watch mode** for rapid iteration:
   ```bash
   goxc watch --mode=ssr .
   ```

2. **Enable debug mode** for verbose logging:
   ```bash
   GOX_DEBUG=1 goxc build ...
   ```

3. **Test both modes** to ensure portability:
   ```bash
   # Test SSR
   goxc build --mode=ssr . && go run .

   # Test CSR
   goxc build --mode=csr . && GOOS=js GOARCH=wasm go build -o app.wasm
   ```

### Code Organization

1. **One component per file**: `Button.gox`, `Card.gox`
2. **Shared types in separate files**: `types.go`
3. **Utilities in Go files**: `utils.go`, `helpers.go`
4. **Keep components small**: < 200 lines per component

---

## 🤝 Contributing Examples

We welcome new examples! Guidelines:

1. **Clear purpose**: Each example should teach something specific
2. **Self-contained**: Should work standalone
3. **Well-documented**: Include README and comments
4. **Both modes**: Provide SSR and WASM versions when applicable
5. **Best practices**: Follow GoX conventions

### Example Submission Checklist

- [ ] Example works in both SSR and CSR (if applicable)
- [ ] README.md explains what it demonstrates
- [ ] Code is well-commented
- [ ] Follows GoX style guidelines
- [ ] No external dependencies (if possible)
- [ ] Includes build instructions
- [ ] Screenshots/GIFs of running example

---

## 📝 Example Template

Use this template for new examples:

```
my-example/
├── README.md
├── Component.gox
├── main.go
├── go.mod
└── screenshots/
    └── demo.gif
```

**README.md template:**

```markdown
# Example Name

Brief description of what this example demonstrates.

## Features
- Feature 1
- Feature 2

## What You'll Learn
- Concept 1
- Concept 2

## Running
\`\`\`bash
# Build and run instructions
\`\`\`

## Code Walkthrough
Explain key parts of the code.

## Next Steps
What to learn next.
```

---

## 📞 Support

- **Questions**: Open a [GitHub Discussion](https://github.com/yourusername/gox/discussions)
- **Bug Reports**: File an [Issue](https://github.com/yourusername/gox/issues)
- **Discord**: Join our [community](https://discord.gg/gox)

---

## 📄 License

All examples are MIT licensed and free to use in your projects.
