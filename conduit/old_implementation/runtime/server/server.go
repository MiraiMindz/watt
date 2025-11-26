// Package server provides HTTP server functionality for GoX SSR components
package server

import (
	"fmt"
	"html/template"
	"net/http"
	"strings"
)

// Component interface that all GoX components must implement
type Component interface {
	Render() string
}

// Server represents a GoX SSR server
type Server struct {
	routes   map[string]func() Component
	port     string
	template *template.Template
}

// New creates a new GoX server
func New(port string) *Server {
	if port == "" {
		port = "8080"
	}

	// Default HTML template
	htmlTemplate := `<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>{{.Title}}</title>
    <style>
        * {
            margin: 0;
            padding: 0;
            box-sizing: border-box;
        }
        body {
            font-family: -apple-system, BlinkMacSystemFont, "Segoe UI", Roboto, sans-serif;
            line-height: 1.6;
            color: #333;
        }
        {{.Styles}}
    </style>
</head>
<body>
    <div id="root">{{.Content}}</div>
    {{.Scripts}}
</body>
</html>`

	tmpl, _ := template.New("page").Parse(htmlTemplate)

	return &Server{
		routes:   make(map[string]func() Component),
		port:     port,
		template: tmpl,
	}
}

// Route registers a component for a specific path
func (s *Server) Route(path string, componentFactory func() Component) {
	s.routes[path] = componentFactory
}

// Static serves static files from a directory
func (s *Server) Static(path, dir string) {
	http.Handle(path, http.StripPrefix(path, http.FileServer(http.Dir(dir))))
}

// Start starts the HTTP server
func (s *Server) Start() error {
	// Register routes
	for path, factory := range s.routes {
		// Capture variables in closure
		p := path
		f := factory
		http.HandleFunc(p, func(w http.ResponseWriter, r *http.Request) {
			s.handleRequest(w, r, f)
		})
	}

	fmt.Printf("🚀 GoX Server running on http://localhost:%s\n", s.port)
	return http.ListenAndServe(":"+s.port, nil)
}

// handleRequest handles HTTP requests and renders components
func (s *Server) handleRequest(w http.ResponseWriter, r *http.Request, factory func() Component) {
	// Create component instance
	component := factory()

	// Render component
	html := component.Render()

	// Process the rendered HTML to extract styles if embedded
	content, styles := s.extractStyles(html)

	// Prepare template data
	data := struct {
		Title   string
		Content template.HTML
		Styles  template.CSS
		Scripts template.JS
	}{
		Title:   "GoX App",
		Content: template.HTML(s.processHTML(content)),
		Styles:  template.CSS(styles),
		Scripts: "",
	}

	// Set content type
	w.Header().Set("Content-Type", "text/html; charset=utf-8")

	// Execute template
	if err := s.template.Execute(w, data); err != nil {
		http.Error(w, err.Error(), http.StatusInternalServerError)
	}
}

// extractStyles extracts embedded styles from HTML
func (s *Server) extractStyles(html string) (content, styles string) {
	// For now, just return the HTML as-is
	// In a real implementation, we'd parse and extract <style> tags
	return html, ""
}

// processHTML processes the component HTML for SSR
func (s *Server) processHTML(html string) string {
	// Replace ${} expressions with actual values
	// This is a simplified version - real implementation would need proper parsing
	html = strings.ReplaceAll(html, "${", "")
	html = strings.ReplaceAll(html, "}", "")

	// Fix className to class for HTML
	html = strings.ReplaceAll(html, "className=", "class=")

	// Remove onClick and other event handlers for SSR
	// In a real implementation, these would be hydrated on the client
	html = strings.ReplaceAll(html, "onClick=", "data-onclick=")

	return html
}

// SetTemplate allows setting a custom HTML template
func (s *Server) SetTemplate(tmpl *template.Template) {
	s.template = tmpl
}