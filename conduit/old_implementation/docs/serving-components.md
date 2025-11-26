# GoX Component Serving Guide

This guide explains how to compile, build, and serve GoX components in your applications.

## Table of Contents
- [Quick Start](#quick-start)
- [Building Components](#building-components)
- [Server-Side Rendering (SSR)](#server-side-rendering-ssr)
- [Client-Side Rendering (CSR)](#client-side-rendering-csr)
- [Development Workflow](#development-workflow)
- [Production Deployment](#production-deployment)

## Quick Start

### 1. Write Your Component

Create a `.gox` file with your component:

```gox
// components/Counter.gox
package main

import "gox"

component Counter(initialValue int) {
    count, setCount := gox.UseState[int](initialValue)

    increment := func() {
        setCount(count + 1)
    }

    render {
        <div className="counter">
            <h1>Count: {count}</h1>
            <button onClick={increment}>Increment</button>
        </div>
    }
}
```

### 2. Compile the Component

```bash
# Build for SSR (default)
goxc build components/Counter.gox

# Build with specific output directory
goxc build components/Counter.gox -o ./dist

# Build for CSR/WASM (coming soon)
goxc build components/Counter.gox --mode=csr
```

### 3. Create a Server

```go
// server/main.go
package main

import (
    "log"
    "github.com/user/gox/runtime/server"
    "path/to/your/compiled/components"
)

func main() {
    // Create server on port 8080
    srv := server.New("8080")

    // Register your component routes
    srv.Route("/", func() server.Component {
        return components.NewCounter(0)
    })

    // Start the server
    log.Fatal(srv.Start())
}
```

### 4. Run Your Application

```bash
# Run the server
go run server/main.go

# Visit http://localhost:8080
```

## Building Components

### Build Command Options

```bash
goxc build [options] <file.gox>

Options:
  -o <dir>         Output directory (default: ./dist)
  --mode <mode>    Build mode: ssr or csr (default: ssr)
  -v              Verbose output
  --watch         Watch for file changes and rebuild
```

### Batch Building

Build multiple components at once:

```bash
# Build all .gox files in a directory
goxc build components/*.gox

# Build with watch mode for development
goxc build --watch components/*.gox
```

## Server-Side Rendering (SSR)

### Basic SSR Setup

1. **Install the GoX runtime**:
```bash
go get github.com/user/gox/runtime
```

2. **Import compiled components**:
```go
import (
    counter "path/to/dist"
    "github.com/user/gox/runtime/server"
)
```

3. **Create routes for components**:
```go
srv := server.New("8080")

// Simple route
srv.Route("/", func() server.Component {
    return counter.NewCounter(0)
})

// Dynamic route with parameters
srv.Route("/counter/:id", func(params map[string]string) server.Component {
    id, _ := strconv.Atoi(params["id"])
    return counter.NewCounter(id)
})
```

### Advanced SSR Features

#### Custom HTML Template

```go
tmpl := template.Must(template.New("custom").Parse(`
<!DOCTYPE html>
<html>
<head>
    <title>{{.Title}}</title>
    <link rel="stylesheet" href="/static/styles.css">
</head>
<body>
    {{.Content}}
    <script src="/static/bundle.js"></script>
</body>
</html>
`))

srv.SetTemplate(tmpl)
```

#### Middleware Support

```go
// Add middleware for logging, auth, etc.
srv.Use(LoggingMiddleware)
srv.Use(AuthMiddleware)
```

#### Static File Serving

```go
// Serve CSS, JS, images
srv.Static("/static/", "./public")
```

### SSR with State Management

```go
// server/main.go
package main

import (
    "github.com/user/gox/runtime/server"
    "github.com/user/gox/runtime/store"
)

func main() {
    srv := server.New("8080")

    // Create global state store
    appStore := store.New()

    srv.Route("/app", func(r *http.Request) server.Component {
        // Pass store to component
        return components.NewApp(appStore)
    })

    srv.Start()
}
```

## Client-Side Rendering (CSR)

*Note: CSR/WASM support is coming soon*

### Building for CSR

```bash
# Build component for client-side rendering
goxc build --mode=csr components/Counter.gox

# Output: dist/counter.wasm and dist/counter.js
```

### Serving CSR Components

```html
<!DOCTYPE html>
<html>
<head>
    <script src="/gox-runtime.js"></script>
</head>
<body>
    <div id="root"></div>
    <script>
        GoX.load('/counter.wasm').then(module => {
            const app = module.NewCounter(0);
            GoX.render(app, document.getElementById('root'));
        });
    </script>
</body>
</html>
```

## Development Workflow

### Hot Reload Setup

```bash
# Install development server
go install github.com/user/gox/cmd/gox-dev-server

# Run with hot reload
gox-dev-server --watch components/
```

### Development Server Features

1. **Auto-compilation**: Automatically rebuilds on file changes
2. **Live reload**: Browser automatically refreshes
3. **Error overlay**: Shows compilation errors in browser
4. **Component inspector**: Debug component state and props

### Project Structure

Recommended project structure:

```
my-gox-app/
├── components/          # GoX components (.gox files)
│   ├── Counter.gox
│   ├── TodoList.gox
│   └── Layout.gox
├── dist/               # Compiled Go files
│   ├── Counter.go
│   ├── TodoList.go
│   └── Layout.go
├── server/             # Server application
│   └── main.go
├── static/             # Static assets
│   ├── styles.css
│   └── app.js
├── go.mod
└── gox.config.json     # GoX configuration
```

### Configuration File

Create `gox.config.json` for project settings:

```json
{
    "build": {
        "input": "./components",
        "output": "./dist",
        "mode": "ssr",
        "watch": true
    },
    "server": {
        "port": 8080,
        "static": "./static"
    },
    "optimize": {
        "minify": true,
        "bundleStyles": true
    }
}
```

## Production Deployment

### Building for Production

```bash
# Production build with optimizations
goxc build --production components/*.gox

# Features enabled in production mode:
# - Minification
# - Dead code elimination
# - Style optimization
# - Bundle splitting
```

### Docker Deployment

```dockerfile
# Dockerfile
FROM golang:1.21 AS builder

WORKDIR /app
COPY . .

# Build components
RUN goxc build --production components/*.gox

# Build server
RUN go build -o server ./server

FROM alpine:latest
RUN apk --no-cache add ca-certificates
WORKDIR /root/

COPY --from=builder /app/server .
COPY --from=builder /app/dist ./dist
COPY --from=builder /app/static ./static

EXPOSE 8080
CMD ["./server"]
```

### Deployment Options

#### Heroku
```bash
# Create Procfile
echo "web: ./server" > Procfile

# Deploy
git push heroku main
```

#### Google Cloud Run
```bash
# Build and push image
gcloud builds submit --tag gcr.io/PROJECT/gox-app

# Deploy
gcloud run deploy --image gcr.io/PROJECT/gox-app
```

#### AWS Lambda
```go
// lambda/main.go
package main

import (
    "github.com/aws/aws-lambda-go/lambda"
    "github.com/user/gox/runtime/server/lambda"
)

func main() {
    srv := server.NewLambdaServer()
    // Configure routes...
    lambda.Start(srv.Handler())
}
```

### Performance Optimization

1. **Enable HTTP/2**:
```go
http2.ConfigureServer(srv, &http2.Server{})
```

2. **Add caching headers**:
```go
srv.Use(func(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
        w.Header().Set("Cache-Control", "public, max-age=3600")
        next.ServeHTTP(w, r)
    })
})
```

3. **Enable compression**:
```go
import "github.com/gorilla/handlers"
srv.Use(handlers.CompressHandler)
```

## API Reference

### Server API

```go
// Create new server
srv := server.New(port string) *Server

// Register component route
srv.Route(path string, factory func() Component)

// Serve static files
srv.Static(path string, dir string)

// Add middleware
srv.Use(middleware func(http.Handler) http.Handler)

// Set custom template
srv.SetTemplate(tmpl *template.Template)

// Start server
srv.Start() error
```

### Component Interface

```go
type Component interface {
    Render() string
    GetProps() map[string]interface{}
    SetProps(props map[string]interface{})
}
```

## Examples

### Todo App Server

```go
package main

import (
    "github.com/user/gox/runtime/server"
    "myapp/components"
)

func main() {
    srv := server.New("3000")

    // Home page
    srv.Route("/", func() server.Component {
        return components.NewLayout(
            components.NewTodoList(),
        )
    })

    // API routes
    srv.Route("/api/todos", TodoAPIHandler)

    // Static files
    srv.Static("/static/", "./public")

    srv.Start()
}
```

### Multi-Page Application

```go
srv.Route("/", HomePage)
srv.Route("/about", AboutPage)
srv.Route("/blog", BlogList)
srv.Route("/blog/:slug", BlogPost)
srv.Route("/admin", AdminDashboard)
```

## Troubleshooting

### Common Issues

1. **Component not rendering**: Check that Render() returns valid HTML
2. **Styles not applying**: Ensure className is converted to class for SSR
3. **State not persisting**: Implement proper state management
4. **Build errors**: Check GoX syntax and imports

### Debug Mode

```bash
# Enable debug logging
GOXC_DEBUG=true goxc build components/*.gox

# Verbose server logs
GOX_SERVER_DEBUG=true go run server/main.go
```

## Next Steps

- Learn about [Component Lifecycle](./component-lifecycle.md)
- Explore [State Management](./state-management.md)
- Read about [Performance Optimization](./optimization.md)
- Check out [Example Projects](https://github.com/user/gox-examples)