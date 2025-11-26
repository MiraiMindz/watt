# GoX Quick Start Guide

Get up and running with GoX in 5 minutes!

## Installation

```bash
# Clone the repository
git clone https://github.com/user/gox
cd gox

# Install the GoX compiler
go install ./cmd/goxc
```

## Your First Component

### 1. Create a Component File

Create `Hello.gox`:

```gox
package main

import "gox"

component Hello(name string) {
    render {
        <div>
            <h1>Hello, {name}!</h1>
            <p>Welcome to GoX</p>
        </div>
    }
}
```

### 2. Compile the Component

```bash
goxc build Hello.gox
```

This generates `dist/Hello.go` with your compiled component.

### 3. Create a Server

Create `server.go`:

```go
package main

import (
    "fmt"
    "net/http"
)

func main() {
    http.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
        // For now, serve static HTML
        // In production, import and use your compiled component
        html := `
        <!DOCTYPE html>
        <html>
        <head>
            <title>GoX App</title>
        </head>
        <body>
            <div>
                <h1>Hello, World!</h1>
                <p>Welcome to GoX</p>
            </div>
        </body>
        </html>
        `
        w.Header().Set("Content-Type", "text/html")
        fmt.Fprint(w, html)
    })

    fmt.Println("Server running on http://localhost:8080")
    http.ListenAndServe(":8080", nil)
}
```

### 4. Run Your App

```bash
go run server.go
```

Visit http://localhost:8080 to see your component!

## Complete Example: Counter App

### 1. Build the Counter Component

```bash
# Build the example counter
goxc build examples/counter-ssr/Counter.gox

# Or build your own
goxc build test-counter-minimal.gox
```

### 2. Run the Example Server

```bash
# Run the provided example server
go run examples/serve-compiled/main.go
```

### 3. View Your App

Open http://localhost:8080/counter in your browser.

## Interactive Counter Example

Here's a complete working counter:

### Counter.gox

```gox
package main

import "gox"

component Counter() {
    count, setCount := gox.UseState[int](0)

    increment := func() {
        setCount(count + 1)
    }

    decrement := func() {
        setCount(count - 1)
    }

    render {
        <div className="counter">
            <h1>Count: {count}</h1>
            <button onClick={increment}>+</button>
            <button onClick={decrement}>-</button>
        </div>
    }
}
```

### Build and Run

```bash
# Build the component
goxc build Counter.gox

# Run the simple server
go run examples/simple-server/main.go
```

## Using Compiled Components

After building with `goxc`, your component becomes a Go package:

```go
// Import your compiled component
import counter "path/to/dist"

// Use it in your server
func handler(w http.ResponseWriter, r *http.Request) {
    // Create component instance
    comp := counter.NewCounter(0)

    // Render to HTML
    html := comp.Render()

    // Serve the HTML
    fmt.Fprint(w, html)
}
```

## Features

✅ **React-like Syntax** - Familiar component structure
✅ **Type Safety** - Full Go type checking
✅ **SSR Support** - Server-side rendering out of the box
✅ **Hooks** - useState, useEffect, useMemo, and more
✅ **JSX Support** - Write HTML naturally in Go
✅ **Hot Reload** - Fast development cycle

## Project Structure

```
my-app/
├── components/        # Your .gox files
│   ├── App.gox
│   ├── Counter.gox
│   └── TodoList.gox
├── dist/             # Compiled Go files
├── server/           # Your server code
│   └── main.go
└── static/           # CSS, JS, images
```

## Common Commands

```bash
# Build single file
goxc build Component.gox

# Build multiple files
goxc build components/*.gox

# Build to specific directory
goxc build -o ./build Component.gox

# Watch mode (auto-rebuild)
goxc build --watch components/*.gox

# Verbose output
goxc build -v Component.gox
```

## What's Next?

- 📖 Read the [full documentation](./docs/README.md)
- 🎯 Check out [example components](./examples/)
- 🚀 Learn about [serving components](./docs/serving-components.md)
- 💡 Explore [advanced features](./docs/advanced.md)

## Need Help?

- 🐛 [Report issues](https://github.com/user/gox/issues)
- 💬 [Join discussions](https://github.com/user/gox/discussions)
- 📚 [Read the guide](./docs/serving-components.md)

## Try It Now!

```bash
# Quick test - build and run a simple component
echo 'package main
import "gox"
component Test() {
    render { <h1>It works!</h1> }
}' > test.gox

goxc build test.gox
go run examples/simple-server/main.go
```

Open http://localhost:8080 and see your component!