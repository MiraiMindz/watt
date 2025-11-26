// Test application server for GoX framework
// Supports SSR, CSR, and Hydration modes
package main

import (
	"embed"
	"encoding/json"
	"fmt"
	"html/template"
	"io"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

//go:embed static/*
var staticFiles embed.FS

// ServerConfig holds the server configuration
type ServerConfig struct {
	Port        string
	Mode        string // "ssr", "csr", or "hybrid"
	BuildDir    string
	ComponentsDir string
}

// TestAppServer serves the GoX test application
type TestAppServer struct {
	config   *ServerConfig
	template *template.Template
}

// NewTestAppServer creates a new test app server
func NewTestAppServer(config *ServerConfig) *TestAppServer {
	return &TestAppServer{
		config: config,
	}
}

// Start starts the server
func (s *TestAppServer) Start() error {
	mux := http.NewServeMux()

	// Routes
	mux.HandleFunc("/", s.handleHome)
	mux.HandleFunc("/counter", s.handleCounter)
	mux.HandleFunc("/todo", s.handleTodo)
	mux.HandleFunc("/timer", s.handleTimer)
	mux.HandleFunc("/dashboard", s.handleDashboard)

	// API routes for CSR
	mux.HandleFunc("/api/components", s.handleAPIComponents)
	mux.HandleFunc("/api/state/", s.handleAPIState)

	// Static files
	mux.HandleFunc("/static/", s.handleStatic)
	mux.HandleFunc("/wasm/", s.handleWASM)
	mux.HandleFunc("/wasm_exec.js", s.handleWASMExec)

	// Health check
	mux.HandleFunc("/health", s.handleHealth)

	fmt.Printf(`
╔══════════════════════════════════════════════════════╗
║                  GoX Test Application                 ║
╠══════════════════════════════════════════════════════╣
║                                                        ║
║  🚀 Server running at: http://localhost:%s         ║
║  📦 Mode: %-44s ║
║  📁 Build Dir: %-39s ║
║                                                        ║
╠══════════════════════════════════════════════════════╣
║                     Available Routes                   ║
╠══════════════════════════════════════════════════════╣
║  Home        → http://localhost:%s/                ║
║  Counter     → http://localhost:%s/counter         ║
║  Todo List   → http://localhost:%s/todo            ║
║  Timer       → http://localhost:%s/timer           ║
║  Dashboard   → http://localhost:%s/dashboard       ║
║                                                        ║
╠══════════════════════════════════════════════════════╣
║                     API Endpoints                      ║
╠══════════════════════════════════════════════════════╣
║  Components  → /api/components                        ║
║  State       → /api/state/{component}                 ║
║  Health      → /health                                ║
║                                                        ║
╠══════════════════════════════════════════════════════╣
║  Press Ctrl+C to stop the server                      ║
╚══════════════════════════════════════════════════════╝
`, s.config.Port, strings.ToUpper(s.config.Mode), s.config.BuildDir,
		s.config.Port, s.config.Port, s.config.Port, s.config.Port, s.config.Port)

	return http.ListenAndServe(":"+s.config.Port, mux)
}

// handleHome serves the main page
func (s *TestAppServer) handleHome(w http.ResponseWriter, r *http.Request) {
	html := s.generateHomePage()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// generateHomePage generates the home page HTML
func (s *TestAppServer) generateHomePage() string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoX Test Application</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }

        body {
            font-family: -apple-system, BlinkMacSystemFont, 'Segoe UI', Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            min-height: 100vh;
            padding: 2rem;
        }

        .container {
            max-width: 1200px;
            margin: 0 auto;
        }

        header {
            text-align: center;
            color: white;
            margin-bottom: 3rem;
        }

        h1 {
            font-size: 3rem;
            margin-bottom: 1rem;
        }

        .subtitle {
            font-size: 1.2rem;
            opacity: 0.9;
        }

        .mode-badge {
            display: inline-block;
            background: rgba(255, 255, 255, 0.2);
            padding: 0.5rem 1rem;
            border-radius: 2rem;
            margin-top: 1rem;
            font-weight: bold;
        }

        .component-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(300px, 1fr));
            gap: 2rem;
            margin-bottom: 3rem;
        }

        .component-card {
            background: white;
            border-radius: 1rem;
            padding: 2rem;
            box-shadow: 0 10px 40px rgba(0, 0, 0, 0.1);
            transition: transform 0.3s, box-shadow 0.3s;
        }

        .component-card:hover {
            transform: translateY(-5px);
            box-shadow: 0 20px 60px rgba(0, 0, 0, 0.15);
        }

        .component-card h3 {
            color: #333;
            margin-bottom: 1rem;
            font-size: 1.5rem;
        }

        .component-card p {
            color: #666;
            margin-bottom: 1.5rem;
            line-height: 1.6;
        }

        .component-card a {
            display: inline-block;
            padding: 0.75rem 1.5rem;
            background: linear-gradient(135deg, #667eea 0%%, #764ba2 100%%);
            color: white;
            text-decoration: none;
            border-radius: 0.5rem;
            font-weight: 600;
            transition: transform 0.2s;
        }

        .component-card a:hover {
            transform: translateY(-2px);
        }

        .features {
            list-style: none;
            margin-top: 1rem;
        }

        .features li {
            color: #666;
            padding: 0.25rem 0;
        }

        .features li:before {
            content: "✓ ";
            color: #28a745;
            font-weight: bold;
            margin-right: 0.5rem;
        }

        .info-section {
            background: white;
            border-radius: 1rem;
            padding: 2rem;
            margin-bottom: 2rem;
        }

        .info-section h2 {
            color: #333;
            margin-bottom: 1rem;
        }

        .info-grid {
            display: grid;
            grid-template-columns: repeat(auto-fit, minmax(200px, 1fr));
            gap: 1rem;
            margin-top: 1rem;
        }

        .info-item {
            padding: 1rem;
            background: #f8f9fa;
            border-radius: 0.5rem;
        }

        .info-item strong {
            display: block;
            color: #333;
            margin-bottom: 0.5rem;
        }

        .info-item span {
            color: #666;
        }
    </style>
</head>
<body>
    <div class="container">
        <header>
            <h1>🚀 GoX Test Application</h1>
            <p class="subtitle">Testing Custom Components, SSR, CSR, and Hydration</p>
            <div class="mode-badge">Mode: %s</div>
        </header>

        <div class="info-section">
            <h2>System Information</h2>
            <div class="info-grid">
                <div class="info-item">
                    <strong>Rendering Mode</strong>
                    <span>%s</span>
                </div>
                <div class="info-item">
                    <strong>Server Port</strong>
                    <span>%s</span>
                </div>
                <div class="info-item">
                    <strong>Build Directory</strong>
                    <span>%s</span>
                </div>
                <div class="info-item">
                    <strong>Components</strong>
                    <span>4 Available</span>
                </div>
            </div>
        </div>

        <div class="component-grid">
            <div class="component-card">
                <h3>⚡ Counter Component</h3>
                <p>Interactive counter with state management, memos, and effects. Demonstrates basic hooks and reactivity.</p>
                <ul class="features">
                    <li>useState for state management</li>
                    <li>useMemo for computed values</li>
                    <li>useEffect for side effects</li>
                    <li>Custom step increments</li>
                </ul>
                <a href="/counter">View Counter →</a>
            </div>

            <div class="component-card">
                <h3>📝 Todo List Component</h3>
                <p>Full-featured todo list with filtering, completion tracking, and persistence. Shows complex state handling.</p>
                <ul class="features">
                    <li>Dynamic list management</li>
                    <li>Filter by status</li>
                    <li>Bulk operations</li>
                    <li>Keyboard shortcuts</li>
                </ul>
                <a href="/todo">View Todo List →</a>
            </div>

            <div class="component-card">
                <h3>⏱️ Timer Component</h3>
                <p>Stopwatch with lap times, demonstrating async operations, intervals, and cleanup functions.</p>
                <ul class="features">
                    <li>Start/Pause/Reset controls</li>
                    <li>Lap time recording</li>
                    <li>Effect cleanup</li>
                    <li>useRef for DOM refs</li>
                </ul>
                <a href="/timer">View Timer →</a>
            </div>

            <div class="component-card">
                <h3>📊 Dashboard Component</h3>
                <p>Combines all components in a single view. Tests component composition and performance.</p>
                <ul class="features">
                    <li>Multiple components</li>
                    <li>Shared state</li>
                    <li>Performance testing</li>
                    <li>Layout management</li>
                </ul>
                <a href="/dashboard">View Dashboard →</a>
            </div>
        </div>

        <div class="info-section">
            <h2>Testing Guide</h2>
            <div style="color: #666; line-height: 1.8;">
                <p><strong>SSR Mode:</strong> Components are rendered on the server. View page source to see the HTML.</p>
                <p><strong>CSR Mode:</strong> Components are rendered in the browser using WebAssembly.</p>
                <p><strong>Hybrid Mode:</strong> Server renders HTML, then hydrates with WASM for interactivity.</p>
                <br>
                <p>Open DevTools to see console logs, network requests, and performance metrics.</p>
            </div>
        </div>
    </div>
</body>
</html>`, strings.ToUpper(s.config.Mode), s.config.Mode, s.config.Port, s.config.BuildDir)
}

// handleCounter serves the counter component
func (s *TestAppServer) handleCounter(w http.ResponseWriter, r *http.Request) {
	s.serveComponent(w, r, "Counter", map[string]interface{}{
		"initialCount": 0,
		"step":         1,
	})
}

// handleTodo serves the todo component
func (s *TestAppServer) handleTodo(w http.ResponseWriter, r *http.Request) {
	s.serveComponent(w, r, "TodoList", map[string]interface{}{})
}

// handleTimer serves the timer component
func (s *TestAppServer) handleTimer(w http.ResponseWriter, r *http.Request) {
	s.serveComponent(w, r, "Timer", map[string]interface{}{})
}

// handleDashboard serves all components together
func (s *TestAppServer) handleDashboard(w http.ResponseWriter, r *http.Request) {
	// For dashboard, we'll render all components
	html := s.generateDashboardPage()
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// serveComponent serves a component based on the mode
func (s *TestAppServer) serveComponent(w http.ResponseWriter, r *http.Request, name string, props map[string]interface{}) {
	switch s.config.Mode {
	case "ssr":
		s.serveSSR(w, r, name, props)
	case "csr":
		s.serveCSR(w, r, name, props)
	case "hybrid":
		s.serveHybrid(w, r, name, props)
	default:
		http.Error(w, "Invalid mode", http.StatusInternalServerError)
	}
}

// serveSSR serves server-side rendered content
func (s *TestAppServer) serveSSR(w http.ResponseWriter, r *http.Request, name string, props map[string]interface{}) {
	// In real implementation, this would call the SSR renderer
	html := s.generateSSRPage(name, props)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// serveCSR serves client-side rendered content
func (s *TestAppServer) serveCSR(w http.ResponseWriter, r *http.Request, name string, props map[string]interface{}) {
	html := s.generateCSRPage(name, props)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// serveHybrid serves SSR with hydration
func (s *TestAppServer) serveHybrid(w http.ResponseWriter, r *http.Request, name string, props map[string]interface{}) {
	html := s.generateHybridPage(name, props)
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// generateDashboardPage generates the dashboard page
func (s *TestAppServer) generateDashboardPage() string {
	// Implementation would combine all components
	return "Dashboard page - combines all components"
}

// generateSSRPage generates an SSR page
func (s *TestAppServer) generateSSRPage(name string, props map[string]interface{}) string {
	// Simplified - in real implementation would use SSR renderer
	return fmt.Sprintf("<h1>SSR: %s Component</h1>", name)
}

// generateCSRPage generates a CSR page
func (s *TestAppServer) generateCSRPage(name string, props map[string]interface{}) string {
	propsJSON, _ := json.Marshal(props)
	return fmt.Sprintf(`<!DOCTYPE html>
<html>
<head>
    <title>%s - GoX CSR</title>
    <script src="/wasm_exec.js"></script>
</head>
<body>
    <div id="root">Loading %s...</div>
    <script>
        async function loadComponent() {
            const go = new Go();
            const result = await WebAssembly.instantiateStreaming(
                fetch('/wasm/%s.wasm'),
                go.importObject
            );
            await go.run(result.instance);

            if (window.%s) {
                const props = %s;
                const component = window.%s(props);
                component.Render(document.getElementById('root'));
            }
        }
        loadComponent();
    </script>
</body>
</html>`, name, name, strings.ToLower(name), name, string(propsJSON), name)
}

// generateHybridPage generates a hybrid SSR+Hydration page
func (s *TestAppServer) generateHybridPage(name string, props map[string]interface{}) string {
	// Simplified - would include SSR content + hydration script
	return fmt.Sprintf("<h1>Hybrid: %s Component</h1>", name)
}

// handleStatic serves static files
func (s *TestAppServer) handleStatic(w http.ResponseWriter, r *http.Request) {
	http.ServeFile(w, r, filepath.Join(s.config.BuildDir, r.URL.Path))
}

// handleWASM serves WASM files
func (s *TestAppServer) handleWASM(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/wasm")
	http.ServeFile(w, r, filepath.Join(s.config.BuildDir, r.URL.Path))
}

// handleWASMExec serves wasm_exec.js
func (s *TestAppServer) handleWASMExec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	http.ServeFile(w, r, filepath.Join(s.config.BuildDir, "wasm_exec.js"))
}

// handleHealth serves health check
func (s *TestAppServer) handleHealth(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"status": "healthy",
		"mode":   s.config.Mode,
		"port":   s.config.Port,
	})
}

// handleAPIComponents serves component list
func (s *TestAppServer) handleAPIComponents(w http.ResponseWriter, r *http.Request) {
	components := []string{"Counter", "TodoList", "Timer"}
	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(components)
}

// handleAPIState serves component state
func (s *TestAppServer) handleAPIState(w http.ResponseWriter, r *http.Request) {
	// Extract component name from path
	component := strings.TrimPrefix(r.URL.Path, "/api/state/")

	// Mock state data
	state := map[string]interface{}{
		"component": component,
		"state":     map[string]interface{}{},
		"timestamp": 0,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(state)
}

func main() {
	// Parse flags
	mode := os.Getenv("MODE")
	if mode == "" {
		mode = "csr"
	}

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	buildDir := os.Getenv("BUILD_DIR")
	if buildDir == "" {
		buildDir = "dist"
	}

	config := &ServerConfig{
		Port:     port,
		Mode:     mode,
		BuildDir: buildDir,
	}

	server := NewTestAppServer(config)
	log.Fatal(server.Start())
}