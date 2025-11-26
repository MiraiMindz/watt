# ⚡ Watt

**Maximum power, minimal overhead.**

Watt is a high-performance web framework built on electrical architecture principles—where each component channels energy efficiently to deliver blazing-fast web applications.

## 🔌 Architecture

Watt's electrical architecture connects specialized components into a unified, powerful system:

```
⚡ Bolt (Core Framework)
   └─ Speed and power: The central framework delivering high-performance HTTP handling

🌊 Shockwave (HTTP Engine)
   └─ Initial impact: Custom HTTP/1.1 and HTTP/2 implementation handling request ingress

💾 Capacitor (Data Layer)
   └─ Energy storage: High-performance data structures storing routes, metadata, and framework state

🔌 Conduit (Templating)
   └─ Safe channel: Type-safe templating engine channeling data to views

⚡ Jolt (Partial Updates)
   └─ Sharp bursts: Targeted HTML updates with minimal payload

�electron Electron (Shared Internals)
   └─ Fundamental particles: Core utilities and primitives shared across all components
```

## 🔋 How It Works

1. **Shockwave** receives and parses HTTP requests with zero-copy parsing
2. **Capacitor** provides ultra-fast route lookup and metadata retrieval
3. **Bolt** orchestrates request handling through composable middleware chains
4. **Conduit** renders server-side templates with compile-time type safety
5. **Jolt** delivers surgical HTML updates for dynamic UX without JavaScript complexity
6. **Electron** powers everything with shared primitives and utilities

## 🚀 Why Watt?

- **⚡ Blazing Fast**: Custom HTTP engine and zero-overhead abstractions
- **🔒 Type-Safe**: End-to-end type safety from routes to templates
- **🎯 Focused**: Server-side rendering with progressive enhancement
- **🪶 Lightweight**: Minimal dependencies, maximum performance
- **🔧 Composable**: Mix and match components based on your needs

## 📦 Repository Structure

```
watt/
├── bolt/           # Core framework and request orchestration with performant and ergonomic API
├── shockwave/      # custom HTTP/1, HTTP/1.1, HTTP/2 and HTTP/3 (QUIC) engine
├── capacitor/      # High-performance data access layer
├── conduit/        # Type-safe templating engine with react-like Go superset syntax
├── jolt/           # Partial update system via AJAX calls
└── electron/       # Shared internals and utilities
```

## 🛠️ Getting Started

> **Note**: Watt is under active development. API and structure are subject to change.

```bash
# Clone the repository
git clone https://github.com/yourusername/watt.git
cd watt

# Install dependencies
go mod download

# Build all components
go build ./...

# Run examples
go run examples/hello-world/main.go
```

## 🧪 Development Status

Watt is in **early development**. Current focus areas:

- [ ] Core HTTP engine (Shockwave)
- [ ] Routing and middleware (Bolt)
- [ ] Template system (Conduit)
- [ ] Partial updates (Jolt)
- [ ] Performance benchmarks
- [ ] Documentation and examples

## 📖 Documentation

Detailed documentation for each component:

- [Bolt](./bolt/README.md) - Core framework
- [Shockwave](./shockwave/README.md) - HTTP engine
- [Capacitor](./capacitor/README.md) - Data structures
- [Conduit](./conduit/README.md) - Templating
- [Jolt](./jolt/README.md) - Partial updates
- [Electron](./electron/README.md) - Shared internals

## 🤝 Contributing

Contributions are welcome! Please read our [Contributing Guide](CONTRIBUTING.md) for details on our development process and how to submit pull requests.

## 📄 License

[MIT License](LICENSE) - see the LICENSE file for details.

## ⚡ Philosophy

Watt is built on the principle that **web frameworks should amplify your power, not drain it**. Every architectural decision prioritizes:

- **Performance**: Zero-cost abstractions and minimal overhead
- **Clarity**: Explicit over implicit, simple over clever
- **Reliability**: Type safety and compile-time guarantees
- **Composability**: Use what you need, nothing more

---

**Built with ⚡ by the Watt team**