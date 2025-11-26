// Package ssr provides server-side rendering with hydration support
package ssr

import (
	"bytes"
	"encoding/json"
	"fmt"
	"html"
	"strings"
)

// HydratableRenderer extends the Renderer with hydration support
type HydratableRenderer struct {
	*Renderer
	hydrationData map[string]interface{}
	componentID   string
}

// NewHydratableRenderer creates a new renderer with hydration support
func NewHydratableRenderer() *HydratableRenderer {
	return &HydratableRenderer{
		Renderer:      NewRenderer(),
		hydrationData: make(map[string]interface{}),
		componentID:   generateComponentID(),
	}
}

// RenderWithHydration renders a component with hydration support
func (r *HydratableRenderer) RenderWithHydration(component Component, props map[string]interface{}) string {
	var buf bytes.Buffer

	// Render the component HTML
	componentHTML := r.Render(component, props)

	// Add hydration markers
	hydratedHTML := r.addHydrationMarkers(componentHTML)

	// Write the hydrated HTML
	buf.WriteString(hydratedHTML)

	// Add hydration script with initial state
	buf.WriteString(r.generateHydrationScript(component, props))

	return buf.String()
}

// addHydrationMarkers adds data attributes for hydration
func (r *HydratableRenderer) addHydrationMarkers(html string) string {
	// Add component ID to root element
	if strings.HasPrefix(html, "<") {
		insertPos := strings.Index(html, ">")
		if insertPos > 0 {
			marker := fmt.Sprintf(` data-gox-component="%s"`, r.componentID)
			return html[:insertPos] + marker + html[insertPos:]
		}
	}
	return html
}

// generateHydrationScript generates the hydration data script
func (r *HydratableRenderer) generateHydrationScript(component Component, props map[string]interface{}) string {
	hydrationData := map[string]interface{}{
		"componentID": r.componentID,
		"props":       props,
		"state":       r.extractComponentState(component),
		"config": map[string]interface{}{
			"hydrate": true,
		},
	}

	jsonData, err := json.Marshal(hydrationData)
	if err != nil {
		return ""
	}

	return fmt.Sprintf(
		`<script type="application/gox-hydration" data-component-id="%s">%s</script>`,
		r.componentID,
		string(jsonData),
	)
}

// extractComponentState extracts the current state from a component
func (r *HydratableRenderer) extractComponentState(component Component) map[string]interface{} {
	state := make(map[string]interface{})

	// Extract state based on component type
	if stateful, ok := component.(interface {
		GetState() map[string]interface{}
	}); ok {
		return stateful.GetState()
	}

	return state
}

// generateComponentID generates a unique component ID
func generateComponentID() string {
	// Simple ID generation - in production, use a better method
	return fmt.Sprintf("gox-%d", generateTimestamp())
}

// generateTimestamp generates a timestamp for IDs
func generateTimestamp() int64 {
	// Simplified - in production use proper time handling
	return 0 // Placeholder
}

// HydrationHTMLTemplate generates a complete HTML page with hydration support
func HydrationHTMLTemplate(title, componentHTML, wasmPath string) string {
	return fmt.Sprintf(`<!DOCTYPE html>
<html lang="en">
<head>
    <meta charset="UTF-8">
    <meta name="viewport" content="width=device-width, initial-scale=1.0">
    <title>%s</title>
    <script src="/wasm_exec.js"></script>
    <style>
        /* Component styles */
        .loading { display: none; }
        .error { color: red; padding: 1rem; }
    </style>
</head>
<body>
    <div id="root">
        %s
    </div>
    <script>
        // Progressive enhancement - enhance when WASM loads
        (async function() {
            const root = document.getElementById('root');

            try {
                // Check if WebAssembly is supported
                if (!window.WebAssembly) {
                    console.log('WebAssembly not supported, using server-rendered content');
                    return;
                }

                // Load WASM component
                const go = new Go();
                const result = await WebAssembly.instantiateStreaming(
                    fetch('%s'),
                    go.importObject
                );

                // Start Go runtime
                await go.run(result.instance);

                // Find hydration data
                const hydrationScript = document.querySelector('script[type="application/gox-hydration"]');
                if (hydrationScript) {
                    const hydrationData = JSON.parse(hydrationScript.textContent);

                    // Get component constructor
                    const componentName = hydrationData.componentName || 'Counter';
                    const ComponentConstructor = window[componentName];

                    if (ComponentConstructor) {
                        // Create component instance
                        const component = ComponentConstructor(hydrationData.props);

                        // Hydrate instead of render
                        if (component.Hydrate) {
                            component.Hydrate(root);
                            console.log('Component hydrated successfully');
                        } else {
                            // Fallback to client-side render
                            component.Render(root);
                            console.log('Component rendered (no hydration)');
                        }
                    }
                }
            } catch (error) {
                console.error('Failed to hydrate component:', error);
                // Server-rendered content remains functional
            }
        })();
    </script>
</body>
</html>`, html.EscapeString(title), componentHTML, wasmPath)
}