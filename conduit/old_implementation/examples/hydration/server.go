// Example server demonstrating SSR with hydration
package main

import (
	"fmt"
	"log"
	"net/http"
	"path/filepath"

	"github.com/user/gox/runtime/ssr"
)

// CounterComponent represents our Counter component for SSR
type CounterComponent struct {
	Count int
	Title string
}

// GetState returns the component's current state for hydration
func (c *CounterComponent) GetState() map[string]interface{} {
	return map[string]interface{}{
		"count": c.Count,
		"title": c.Title,
	}
}

// Render generates the HTML for the component
func (c *CounterComponent) Render() string {
	return fmt.Sprintf(`
		<div class="counter-container">
			<h1>%s</h1>
			<div class="count-value" data-count="%d">%d</div>
			<div class="counter-controls">
				<button id="increment" data-action="increment">+ Increment</button>
				<button id="decrement" class="btn-secondary" data-action="decrement">- Decrement</button>
				<button id="reset" class="btn-danger" data-action="reset">Reset</button>
			</div>
		</div>
	`, c.Title, c.Count, c.Count)
}

// HydrationServer serves SSR pages with hydration support
type HydrationServer struct {
	renderer *ssr.HydratableRenderer
}

// NewHydrationServer creates a new hydration server
func NewHydrationServer() *HydrationServer {
	return &HydrationServer{
		renderer: ssr.NewHydratableRenderer(),
	}
}

// ServeHTTP handles HTTP requests
func (s *HydrationServer) ServeHTTP(w http.ResponseWriter, r *http.Request) {
	switch r.URL.Path {
	case "/":
		s.serveHomePage(w, r)
	case "/wasm_exec.js":
		s.serveWASMExec(w, r)
	case "/counter.wasm":
		s.serveWASM(w, r)
	default:
		http.NotFound(w, r)
	}
}

// serveHomePage serves the SSR page with hydration
func (s *HydrationServer) serveHomePage(w http.ResponseWriter, r *http.Request) {
	// Create component with initial state
	component := &CounterComponent{
		Count: 42, // Initial count from server
		Title: "GoX Counter (SSR with Hydration)",
	}

	// Render with hydration support
	props := map[string]interface{}{
		"initialCount": component.Count,
		"title":        component.Title,
	}

	componentHTML := s.renderer.RenderWithHydration(component, props)

	// Generate complete HTML page
	html := ssr.HydrationHTMLTemplate(
		"GoX Hydration Demo",
		componentHTML,
		"/counter.wasm",
	)

	// Send response
	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	w.Write([]byte(html))
}

// serveWASMExec serves the wasm_exec.js file
func (s *HydrationServer) serveWASMExec(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/javascript")
	http.ServeFile(w, r, filepath.Join("dist", "wasm_exec.js"))
}

// serveWASM serves the compiled WASM file
func (s *HydrationServer) serveWASM(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/wasm")
	w.Header().Set("Cache-Control", "public, max-age=3600")
	http.ServeFile(w, r, filepath.Join("dist", "counter.wasm"))
}

func main() {
	server := NewHydrationServer()

	fmt.Println("🚀 GoX Hydration Server starting on http://localhost:8080")
	fmt.Println("📦 Serving SSR with hydration support")
	fmt.Println("")
	fmt.Println("Features:")
	fmt.Println("  ✅ Server-side rendering")
	fmt.Println("  ✅ Progressive enhancement")
	fmt.Println("  ✅ Client-side hydration")
	fmt.Println("  ✅ Works without JavaScript")
	fmt.Println("")
	fmt.Println("Press Ctrl+C to stop")

	log.Fatal(http.ListenAndServe(":8080", server))
}