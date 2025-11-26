# GoX

**A React-like Frontend Framework for Go**

GoX is a superset of Go that brings React-style component-based UI development to the Go ecosystem. Write declarative UIs with JSX-like syntax, hooks for state management, and compile to either Server-Side Rendering (SSR) or Client-Side Rendering (CSR/WASM).

---

## ✨ Features

- 🎯 **React-like API**: Familiar hooks (useState, useEffect, useMemo, etc.)
- 📝 **JSX Syntax**: Write markup alongside your Go code
- 🎨 **Scoped CSS**: Component-scoped styling out of the box
- 🚀 **Dual Compilation**: SSR for servers, WASM for browsers
- 💪 **Type Safety**: Full Go type system with generics
- ⚡ **Performance**: Native Go speed for SSR, optimized WASM for CSR
- 🔧 **Go Native**: Seamlessly integrates with existing Go code

---

## 🚀 Quick Start

### Installation

```bash
go install github.com/yourusername/gox/cmd/goxc@latest
```

### Your First Component

Create `Counter.gox`:

```gox
package main

import "gox"

component Counter() {
    count, setCount := gox.UseState[int](0)

    render {
        <div className="counter">
            <h1>Count: {count}</h1>
            <button onClick={func() { setCount(count + 1) }}>
                Increment
            </button>
        </div>
    }

    style {
        .counter {
            text-align: center;
            padding: 20px;
        }
        .counter button {
            background: #007bff;
            color: white;
            padding: 10px 20px;
            border: none;
            cursor: pointer;
        }
    }
}
```

### Build & Run (SSR)

```bash
# Compile .gox to .go
goxc build --mode=ssr Counter.gox

# Run your Go app
go run .
```

### Build & Run (WASM)

```bash
# Compile to WASM
goxc build --mode=csr Counter.gox
GOOS=js GOARCH=wasm go build -o app.wasm .

# Serve
goxc dev --port 3000
```

Visit http://localhost:3000

---

## 📚 Documentation

- **[Quick Start Guide](QUICK_START.md)** - Get up and running in 5 minutes
- **[Syntax Specification](SYNTAX_SPEC.md)** - Complete language reference
- **[Implementation Guide](IMPLEMENTATION_GUIDE.md)** - Deep dive into internals
- **[Roadmap](ROADMAP.md)** - Development timeline and milestones

---

## 🎯 Why GoX?

### For Go Developers
- Build web UIs without leaving Go
- Leverage your existing Go knowledge
- Type-safe frontend development
- Share code between backend and frontend

### For React Developers
- Familiar component model and hooks
- JSX-like syntax you already know
- Similar patterns and best practices
- Better performance with Go

### For Everyone
- Single language for full-stack apps
- Strong typing catches bugs early
- Great IDE support (once implemented)
- Simplified deployment (single binary)

---

## 🏗️ Architecture

```
┌─────────────┐
│  .gox files │  ← Your component source code
└──────┬──────┘
       │
       ▼
┌─────────────────┐
│  GoX Compiler   │  ← Lexer, Parser, Analyzer
└────────┬────────┘
         │
         ▼
    ┌────────┐
    │  AST   │  ← Intermediate representation
    └───┬────┘
        │
   ┌────┴─────┐
   │          │
   ▼          ▼
┌─────┐   ┌──────┐
│ SSR │   │ CSR  │  ← Different code generation
└──┬──┘   └───┬──┘
   │          │
   ▼          ▼
┌────────┐ ┌─────────┐
│.go file│ │.go+WASM │  ← Compiled output
│+ HTML  │ │+ VDom   │
└────────┘ └─────────┘
```

### SSR Mode
- Compiles components to Go structs
- `Render()` methods return HTML strings
- Fast server-side rendering
- Perfect for static sites, blogs, docs

### CSR Mode (WASM)
- Compiles components to Go structs
- `Render()` methods return Virtual DOM
- Interactive in-browser apps
- Full client-side reactivity

---

## 🎨 Examples

### Todo App

```gox
component TodoApp() {
    todos, setTodos := gox.UseState[[]Todo]([]Todo{})
    input, setInput := gox.UseState[string]("")

    addTodo := func() {
        if input != "" {
            setTodos(append(todos, Todo{Text: input, Done: false}))
            setInput("")
        }
    }

    render {
        <div className="todo-app">
            <h1>My Todos</h1>
            <div className="add-todo">
                <input
                    value={input}
                    onInput={func(e js.Value) {
                        setInput(e.Get("target").Get("value").String())
                    }}
                    placeholder="What needs to be done?"
                />
                <button onClick={addTodo}>Add</button>
            </div>
            <ul>
                {todos.map(func(todo Todo, i int) *gox.VNode {
                    return <TodoItem key={i} todo={todo} />
                })}
            </ul>
        </div>
    }

    style {
        .todo-app {
            max-width: 600px;
            margin: 50px auto;
            padding: 20px;
        }
        .add-todo {
            display: flex;
            margin-bottom: 20px;
        }
        .add-todo input {
            flex: 1;
            padding: 10px;
            border: 1px solid #ddd;
        }
    }
}
```

### Data Fetching

```gox
component UserProfile(userID string) {
    user, setUser := gox.UseState[*User](nil)
    loading, setLoading := gox.UseState[bool](true)

    gox.UseEffect(func() func() {
        go func() {
            data, err := fetchUser(userID)
            if err == nil {
                setUser(data)
            }
            setLoading(false)
        }()
        return nil
    }, []interface{}{userID})

    render {
        <div className="profile">
            {loading ? (
                <div>Loading...</div>
            ) : (
                <div>
                    <h1>{user.Name}</h1>
                    <p>{user.Email}</p>
                    <p>{user.Bio}</p>
                </div>
            )}
        </div>
    }
}
```

More examples in the [examples/](examples/) directory.

---

## 🔧 Development Status

⚠️ **GoX is currently in design/planning phase.**

This repository contains the architecture and implementation plan. Active development has not yet started.

### Current Phase
- [x] Architecture design
- [x] Syntax specification
- [x] Implementation guide
- [x] Roadmap planning
- [ ] **Phase 1: Lexer & Parser** ← Next up!

See [ROADMAP.md](ROADMAP.md) for detailed development timeline.

---

## 🤝 Contributing

We welcome contributions! Here's how you can help:

1. **Star this repo** - Show your interest
2. **Join discussions** - Share ideas and feedback
3. **Report issues** - Found a bug? Let us know
4. **Submit PRs** - Code contributions welcome
5. **Write docs** - Help improve documentation
6. **Create examples** - Build cool apps with GoX

### Development Setup

```bash
# Clone the repository
git clone https://github.com/yourusername/gox.git
cd gox

# Install dependencies
go mod download

# Run tests
go test ./...

# Build the compiler
go build -o goxc cmd/goxc/main.go
```

See [CONTRIBUTING.md](CONTRIBUTING.md) for more details.

---

## 📖 Learn More

### Core Concepts

**Components**: Reusable UI building blocks
```gox
component Button(text string, onClick func()) {
    render { <button onClick={onClick}>{text}</button> }
}
```

**Hooks**: State and lifecycle management
```gox
count, setCount := gox.UseState[int](0)
gox.UseEffect(func() func() { /* side effect */ }, []interface{}{})
```

**JSX**: Declarative UI syntax
```gox
render {
    <div className="container">
        <h1>{title}</h1>
        {items.map(func(item string, i int) {
            return <li key={i}>{item}</li>
        })}
    </div>
}
```

**Scoped Styles**: Component-level CSS
```gox
style {
    .button { background: blue; }
    .button:hover { background: darkblue; }
}
```

---

## 🎯 Roadmap Highlights

### Phase 1-5: Foundation (Weeks 1-12)
- Lexer & Parser
- useState hook
- JSX parsing
- SSR code generation
- CSS processing

### Phase 6-10: CSR Support (Weeks 13-24)
- useEffect hook
- WASM foundation
- Virtual DOM
- CSR code generation
- Additional hooks

### Phase 11-15: Polish (Weeks 25-34)
- Build tooling
- Developer experience
- Performance optimization
- Testing & examples
- v1.0 Release

See [ROADMAP.md](ROADMAP.md) for complete timeline.

---

## 🔬 Technical Details

### Compiler Architecture
- **Lexer**: Tokenizes .gox files with context-aware modes (Go/JSX/CSS)
- **Parser**: Builds AST from tokens, extends go/ast
- **Analyzer**: Semantic analysis, creates IR
- **Transpiler**: Generates Go code (SSR or CSR)
- **Runtime**: Provides hooks, VDOM, and component base

### SSR Output
```go
type Counter struct {
    *gox.Component
    count int
}

func (c *Counter) Render() string {
    return fmt.Sprintf(`<div class="counter">
        <h1>Count: %d</h1>
        <button>Increment</button>
    </div>`, c.count)
}
```

### CSR Output
```go
func (c *Counter) Render() *gox.VNode {
    return gox.H("div", gox.Props{"className": "counter"},
        gox.H("h1", nil, gox.Text(fmt.Sprintf("Count: %d", c.count))),
        gox.H("button", gox.Props{"onClick": c.handleClick},
            gox.Text("Increment")),
    )
}
```

---

## 🌟 Inspiration

GoX is inspired by:
- **React** - Component model and hooks
- **Svelte** - Single-file components with scoped styles
- **SolidJS** - Fine-grained reactivity
- **Templ** - Go-based templating
- **Go** - Simplicity and pragmatism

---

## 📄 License

MIT License - see [LICENSE](LICENSE) file for details

---

## 💬 Community

- **Discord**: [Join our server](https://discord.gg/gox) (coming soon)
- **Twitter**: [@gox_lang](https://twitter.com/gox_lang) (coming soon)
- **GitHub Discussions**: [Start a discussion](https://github.com/yourusername/gox/discussions)
- **Blog**: [gox.dev/blog](https://gox.dev/blog) (coming soon)

---

## 🙏 Acknowledgments

Special thanks to:
- The Go team for creating an amazing language
- The React team for pioneering modern UI patterns
- The WebAssembly community for making this possible
- All contributors and early adopters

---

## 🚧 Status & Disclaimer

**Current Status**: 🔴 Pre-Alpha / Design Phase

GoX is currently in the design and planning stage. The API and syntax may change significantly before the first release. Use at your own risk in production.

**Not recommended for**:
- ❌ Production applications (yet)
- ❌ Critical systems
- ❌ Stable API requirements

**Great for**:
- ✅ Experimentation
- ✅ Contributing to development
- ✅ Providing feedback
- ✅ Learning about compilers

---

## 📊 Project Goals

### Primary Goals
- ✅ Provide React-like DX for Go developers
- ✅ Support both SSR and CSR from same source
- ✅ Maintain Go's simplicity and pragmatism
- ✅ Achieve great performance
- ✅ Enable full-stack Go applications

### Non-Goals
- ❌ Feature parity with React (we're inspired, not copying)
- ❌ Support for every web framework pattern
- ❌ Backwards compatibility with Go templates
- ❌ Browser support for legacy browsers

---

## 🎓 Learning Resources

- [Quick Start Guide](QUICK_START.md) - 5-minute intro
- [Syntax Specification](SYNTAX_SPEC.md) - Language reference
- [Implementation Guide](IMPLEMENTATION_GUIDE.md) - Build your own!
- [Examples](examples/) - Real-world apps
- [Blog Posts](https://gox.dev/blog) - Tutorials and deep dives (coming soon)

---

## ❓ FAQ

**Q: Why not just use gopherjs or GopherJS?**
A: GoX provides a React-like component model with hooks, JSX syntax, and dual compilation (SSR/CSR). It's a higher-level abstraction.

**Q: Will this work with existing Go web frameworks?**
A: Yes! SSR mode generates Go code that integrates with any Go web framework.

**Q: What about SEO?**
A: SSR mode is perfect for SEO. CSR mode supports pre-rendering.

**Q: Is WASM performance good enough?**
A: WASM is fast and getting faster. For most UIs, it's more than sufficient. We also support TinyGo for smaller bundles.

**Q: Can I use existing Go libraries?**
A: Absolutely! GoX is a superset of Go - all Go code works.

**Q: What about mobile?**
A: Not a primary goal, but WASM works in mobile browsers.

---

<div align="center">

**[Getting Started](QUICK_START.md)** • **[Documentation](SYNTAX_SPEC.md)** • **[Roadmap](ROADMAP.md)** • **[Contributing](CONTRIBUTING.md)**

Made with ❤️ by the GoX community

</div>
