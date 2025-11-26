package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

// This example shows how to serve actual compiled GoX components
// Build your components first:
//   goxc build examples/counter-ssr/Counter.gox
// Then run this server:
//   go run examples/serve-compiled/main.go

// HTML template with proper GoX integration
const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}} - GoX App</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, "Helvetica Neue", Arial, sans-serif;
            background: #f5f5f5;
            line-height: 1.6;
            color: #333;
        }
        .container {
            max-width: 1200px;
            margin: 0 auto;
            padding: 2rem;
        }
        /* Counter specific styles */
        .counter-container {
            max-width: 400px;
            margin: 50px auto;
            padding: 30px;
            background: #fff;
            border-radius: 12px;
            box-shadow: 0 4px 6px rgba(0, 0, 0, 0.1);
        }
        .counter-display {
            text-align: center;
            margin-bottom: 30px;
        }
        .counter-display h1 {
            color: #333;
            margin: 0 0 20px 0;
            font-size: 28px;
        }
        .count-value {
            font-size: 72px;
            font-weight: bold;
            color: #007bff;
            margin: 20px 0;
        }
        .counter-controls {
            display: flex;
            flex-direction: column;
            gap: 10px;
            margin-bottom: 20px;
        }
        .btn {
            padding: 12px 24px;
            font-size: 16px;
            border: none;
            border-radius: 6px;
            cursor: pointer;
            font-weight: 600;
            transition: all 0.2s;
        }
        .btn:hover {
            transform: translateY(-2px);
            box-shadow: 0 2px 4px rgba(0, 0, 0, 0.2);
        }
        .btn:active {
            transform: translateY(0);
        }
        .btn-primary {
            background: #007bff;
            color: white;
        }
        .btn-primary:hover {
            background: #0056b3;
        }
        .btn-secondary {
            background: #6c757d;
            color: white;
        }
        .btn-secondary:hover {
            background: #545b62;
        }
        .btn-danger {
            background: #dc3545;
            color: white;
        }
        .btn-danger:hover {
            background: #c82333;
        }
        .counter-info {
            background: #f8f9fa;
            padding: 15px;
            border-radius: 6px;
            font-size: 14px;
            color: #666;
        }
        .counter-info p {
            margin: 5px 0;
        }
        {{.CustomStyles}}
    </style>
</head>
<body>
    <div class="container">
        <div id="root">{{.Content}}</div>
    </div>

    <!-- GoX Runtime for interactivity -->
    <script>
        // This would be replaced with actual GoX runtime for hydration
        console.log('GoX Component Rendered (SSR Mode)');

        // Add basic interactivity for demo
        document.addEventListener('DOMContentLoaded', function() {
            // Convert data-onclick to onclick for demo
            const buttons = document.querySelectorAll('[data-onclick]');
            buttons.forEach(btn => {
                btn.style.cursor = 'pointer';
                btn.addEventListener('click', function() {
                    console.log('Click event:', this.getAttribute('data-onclick'));
                    // In production, this would trigger GoX event handlers
                });
            });
        });
    </script>
</body>
</html>`

// processSSRContent processes the SSR-rendered content
func processSSRContent(html string) string {
	// Fix className to class for HTML
	html = strings.ReplaceAll(html, "className=", "class=")

	// Convert onClick to data-onclick for SSR
	html = strings.ReplaceAll(html, "onClick=", "data-onclick=")

	// Process template variables (simplified)
	// In production, this would properly evaluate expressions
	html = strings.ReplaceAll(html, "${c.", "")
	html = strings.ReplaceAll(html, "${", "")
	html = strings.ReplaceAll(html, "}", "")

	return html
}

// handleComponent handles component rendering
func handleComponent(w http.ResponseWriter, r *http.Request) {
	// For demo, we'll use mock HTML
	// In production, you'd import and use the actual compiled component
	componentHTML := `
		<div class="counter-container">
			<div class="counter-display">
				<h1>Counter</h1>
				<div class="count-value">0</div>
			</div>
			<div class="counter-controls">
				<button class="btn btn-primary" data-onclick="increment">
					+ Increment
				</button>
				<button class="btn btn-secondary" data-onclick="decrement">
					- Decrement
				</button>
				<button class="btn btn-danger" data-onclick="reset">
					Reset
				</button>
			</div>
			<div class="counter-info">
				<p>Initial value: 0</p>
				<p>Current value: 0</p>
				<p>Difference: 0</p>
			</div>
		</div>
	`

	// Process the component HTML
	componentHTML = processSSRContent(componentHTML)

	// Parse and execute template
	tmpl, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	data := struct {
		Title        string
		Content      template.HTML
		CustomStyles template.CSS
	}{
		Title:        "Counter Component",
		Content:      template.HTML(componentHTML),
		CustomStyles: "",
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// handleHome shows instructions
func handleHome(w http.ResponseWriter, r *http.Request) {
	html := `
	<!DOCTYPE html>
	<html>
	<head>
		<title>GoX Server</title>
		<style>
			body {
				font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
				max-width: 800px;
				margin: 50px auto;
				padding: 20px;
				background: #f5f5f5;
			}
			.container {
				background: white;
				padding: 30px;
				border-radius: 10px;
				box-shadow: 0 2px 10px rgba(0,0,0,0.1);
			}
			h1 { color: #333; }
			h2 { color: #666; margin-top: 30px; }
			pre {
				background: #f8f9fa;
				padding: 15px;
				border-radius: 5px;
				overflow-x: auto;
			}
			code {
				background: #f8f9fa;
				padding: 2px 5px;
				border-radius: 3px;
			}
			.links {
				margin-top: 30px;
			}
			.links a {
				display: inline-block;
				padding: 10px 20px;
				background: #007bff;
				color: white;
				text-decoration: none;
				border-radius: 5px;
				margin-right: 10px;
			}
			.links a:hover {
				background: #0056b3;
			}
		</style>
	</head>
	<body>
		<div class="container">
			<h1>🚀 GoX Server Running</h1>
			<p>Welcome to the GoX component server!</p>

			<h2>How to Use</h2>
			<ol>
				<li>Build your GoX components:
					<pre><code>goxc build examples/counter-ssr/Counter.gox</code></pre>
				</li>
				<li>Import the compiled component in your server</li>
				<li>Register routes for your components</li>
			</ol>

			<h2>Example Routes</h2>
			<div class="links">
				<a href="/counter">View Counter Component</a>
				<a href="/api/health">Health Check</a>
			</div>

			<h2>Quick Start Code</h2>
			<pre><code>// Import compiled component
import counter "path/to/dist"

// Create route
http.HandleFunc("/counter", func(w http.ResponseWriter, r *http.Request) {
    component := counter.NewCounter(0)
    html := component.Render()
    // Serve the HTML...
})</code></pre>
		</div>
	</body>
	</html>
	`

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	fmt.Fprint(w, html)
}

// handleAPI provides a simple API endpoint
func handleAPI(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	fmt.Fprint(w, `{"status":"ok","server":"GoX","version":"1.0.0"}`)
}

func main() {
	// Setup routes
	http.HandleFunc("/", handleHome)
	http.HandleFunc("/counter", handleComponent)
	http.HandleFunc("/api/health", handleAPI)

	// Serve static files
	staticDir := "./static"
	if _, err := os.Stat(staticDir); err == nil {
		http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir(staticDir))))
	}

	// Get port from environment or use default
	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Print startup message
	fmt.Println("╔══════════════════════════════════════╗")
	fmt.Println("║       GoX Component Server           ║")
	fmt.Println("╚══════════════════════════════════════╝")
	fmt.Printf("\n🚀 Server running on http://localhost:%s\n", port)
	fmt.Println("\nAvailable routes:")
	fmt.Printf("  • http://localhost:%s/          - Instructions\n", port)
	fmt.Printf("  • http://localhost:%s/counter   - Counter Component\n", port)
	fmt.Printf("  • http://localhost:%s/api/health - API Health Check\n", port)
	fmt.Println("\nPress Ctrl+C to stop the server")
	fmt.Println("")

	// Get executable path for better error messages
	exe, _ := os.Executable()
	exeDir := filepath.Dir(exe)
	fmt.Printf("Working directory: %s\n\n", exeDir)

	// Start server
	log.Fatal(http.ListenAndServe(":"+port, nil))
}