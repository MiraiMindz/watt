package main

import (
	"fmt"
	"html/template"
	"log"
	"net/http"
	"strings"
)

// Simplified component interface
type Component interface {
	Render() string
}

// Import your compiled component here
// For demo, we'll create a mock component
type Counter struct {
	count int
}

func (c *Counter) Render() string {
	// This would be the actual output from your compiled .go file
	return fmt.Sprintf(`
		<div class="counter-container">
			<h1>Counter Demo</h1>
			<div class="count-value">%d</div>
			<div class="counter-controls">
				<button onclick="increment()">+ Increment</button>
				<button onclick="decrement()">- Decrement</button>
				<button onclick="reset()">Reset</button>
			</div>
		</div>
	`, c.count)
}

func NewCounter(initial int) *Counter {
	return &Counter{count: initial}
}

// HTML template for the page
const pageTemplate = `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>GoX Component Server</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            min-height: 100vh;
            display: flex;
            justify-content: center;
            align-items: center;
        }
        .counter-container {
            background: white;
            padding: 3rem;
            border-radius: 1rem;
            box-shadow: 0 20px 40px rgba(0,0,0,0.1);
            text-align: center;
            min-width: 400px;
        }
        h1 {
            color: #333;
            margin-bottom: 2rem;
            font-size: 2rem;
        }
        .count-value {
            font-size: 5rem;
            font-weight: bold;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            -webkit-background-clip: text;
            -webkit-text-fill-color: transparent;
            margin: 2rem 0;
        }
        .counter-controls {
            display: flex;
            flex-direction: column;
            gap: 1rem;
        }
        button {
            padding: 1rem 2rem;
            font-size: 1.1rem;
            border: none;
            border-radius: 0.5rem;
            cursor: pointer;
            font-weight: 600;
            transition: all 0.3s ease;
            background: linear-gradient(135deg, #667eea 0%, #764ba2 100%);
            color: white;
        }
        button:hover {
            transform: translateY(-2px);
            box-shadow: 0 10px 20px rgba(0,0,0,0.2);
        }
        button:active {
            transform: translateY(0);
        }
    </style>
</head>
<body>
    <div id="root">{{.Content}}</div>
    <script>
        // For demo purposes - in production this would be hydrated
        let count = {{.InitialCount}};

        function updateDisplay() {
            document.querySelector('.count-value').textContent = count;
        }

        function increment() {
            count++;
            updateDisplay();
        }

        function decrement() {
            count--;
            updateDisplay();
        }

        function reset() {
            count = {{.InitialCount}};
            updateDisplay();
        }
    </script>
</body>
</html>`

// Server handler
func componentHandler(w http.ResponseWriter, r *http.Request) {
	// Create component instance
	counter := NewCounter(0)

	// Render component
	componentHTML := counter.Render()

	// Process HTML (convert className to class, etc.)
	componentHTML = strings.ReplaceAll(componentHTML, "className=", "class=")

	// Parse template
	tmpl, err := template.New("page").Parse(pageTemplate)
	if err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
		return
	}

	// Execute template with component HTML
	data := struct {
		Content      template.HTML
		InitialCount int
	}{
		Content:      template.HTML(componentHTML),
		InitialCount: 0,
	}

	w.Header().Set("Content-Type", "text/html; charset=utf-8")
	if err := tmpl.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

func main() {
	// Register routes
	http.HandleFunc("/", componentHandler)

	// Serve static files
	http.Handle("/static/", http.StripPrefix("/static/", http.FileServer(http.Dir("./static"))))

	// Start server
	port := "8080"
	fmt.Printf("🚀 GoX Server running on http://localhost:%s\n", port)
	fmt.Printf("   Visit http://localhost:%s to see your component\n", port)
	fmt.Printf("   Press Ctrl+C to stop\n\n")

	log.Fatal(http.ListenAndServe(":"+port, nil))
}