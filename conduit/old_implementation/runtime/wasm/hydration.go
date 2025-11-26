// Package wasm provides hydration support for server-rendered GoX components
package wasm

import (
	"encoding/json"
	"fmt"
	"syscall/js"
)

// HydrationData contains the initial state and props for hydration
type HydrationData struct {
	Props  map[string]interface{} `json:"props"`
	State  map[string]interface{} `json:"state"`
	Config map[string]interface{} `json:"config"`
}

// Hydrator handles the hydration of server-rendered HTML
type Hydrator struct {
	component  *Component
	container  js.Value
	data       *HydrationData
	eventMap   map[string]js.Func
	isHydrated bool
}

// NewHydrator creates a new hydrator for a component
func NewHydrator(component *Component, container js.Value) *Hydrator {
	return &Hydrator{
		component: component,
		container: container,
		eventMap:  make(map[string]js.Func),
	}
}

// Hydrate enhances server-rendered HTML with client-side interactivity
func (h *Hydrator) Hydrate(vnode *VNode) error {
	if h.isHydrated {
		return fmt.Errorf("component already hydrated")
	}

	// Extract hydration data from DOM
	if err := h.extractHydrationData(); err != nil {
		return fmt.Errorf("failed to extract hydration data: %w", err)
	}

	// Walk the existing DOM and attach event listeners
	if err := h.hydrateNode(vnode, h.container); err != nil {
		return fmt.Errorf("failed to hydrate node: %w", err)
	}

	h.isHydrated = true
	return nil
}

// extractHydrationData extracts initial state from server-rendered HTML
func (h *Hydrator) extractHydrationData() error {
	// Look for hydration script tag
	document := js.Global().Get("document")
	scripts := document.Call("querySelectorAll", "script[type='application/gox-hydration']")

	if scripts.Length() == 0 {
		// No hydration data, use defaults
		h.data = &HydrationData{
			Props: make(map[string]interface{}),
			State: make(map[string]interface{}),
		}
		return nil
	}

	// Parse the first hydration script
	scriptContent := scripts.Index(0).Get("textContent").String()

	// Parse JSON data
	if err := json.Unmarshal([]byte(scriptContent), &h.data); err != nil {
		return fmt.Errorf("failed to parse hydration data: %w", err)
	}

	// Remove hydration script from DOM
	scripts.Index(0).Call("remove")

	return nil
}

// hydrateNode recursively hydrates DOM nodes with event listeners
func (h *Hydrator) hydrateNode(vnode *VNode, domNode js.Value) error {
	if vnode == nil || domNode.IsUndefined() || domNode.IsNull() {
		return nil
	}

	// Text nodes don't need hydration
	if vnode.Tag == "" {
		return nil
	}

	// Attach event listeners
	for eventName, handler := range vnode.Events {
		// Store handler for cleanup
		h.eventMap[eventName] = handler

		// Add event listener to existing DOM element
		domNode.Call("addEventListener", eventName, handler)
	}

	// Hydrate children
	if len(vnode.Children) > 0 {
		childNodes := domNode.Get("childNodes")
		vIndex := 0

		for i := 0; i < childNodes.Length() && vIndex < len(vnode.Children); i++ {
			childDOM := childNodes.Index(i)
			nodeType := childDOM.Get("nodeType").Int()

			// Skip non-element nodes (text, comments, etc.) if VNode is an element
			if nodeType == 3 { // Text node
				if vnode.Children[vIndex].Tag == "" {
					// Text VNode matches text DOM node
					vIndex++
				}
				// Skip text nodes when looking for elements
				continue
			}

			if nodeType == 1 { // Element node
				if vnode.Children[vIndex].Tag != "" {
					// Recursively hydrate child
					if err := h.hydrateNode(vnode.Children[vIndex], childDOM); err != nil {
						return err
					}
					vIndex++
				}
			}
		}
	}

	// Add hydration marker
	domNode.Call("setAttribute", "data-gox-hydrated", "true")

	return nil
}

// HydrateComponent hydrates a component with server-rendered HTML
func HydrateComponent(component interface{}, container js.Value) error {
	// Type assertion to get component with hydration support
	if hydratable, ok := component.(interface {
		Hydrate(js.Value) error
	}); ok {
		return hydratable.Hydrate(container)
	}

	// Fall back to regular render if hydration not supported
	if renderer, ok := component.(interface {
		Render(js.Value)
	}); ok {
		renderer.Render(container)
		return nil
	}

	return fmt.Errorf("component does not support hydration or rendering")
}

// ExtractStateFromDOM extracts component state from DOM attributes
func ExtractStateFromDOM(element js.Value) map[string]interface{} {
	state := make(map[string]interface{})

	// Check for data-gox-state attribute
	if element.Get("dataset").Get("goxState").Truthy() {
		stateJSON := element.Get("dataset").Get("goxState").String()
		json.Unmarshal([]byte(stateJSON), &state)
	}

	return state
}

// Cleanup removes all event listeners added during hydration
func (h *Hydrator) Cleanup() {
	// Remove all event listeners
	for eventName, handler := range h.eventMap {
		h.container.Call("removeEventListener", eventName, handler)
		handler.Release() // Release the JS function
	}

	h.eventMap = make(map[string]js.Func)
	h.isHydrated = false
}

// IsHydrated returns whether the component has been hydrated
func (h *Hydrator) IsHydrated() bool {
	return h.isHydrated
}

// GetHydrationData returns the extracted hydration data
func (h *Hydrator) GetHydrationData() *HydrationData {
	return h.data
}

// HydrationScript generates a script tag with hydration data
func HydrationScript(props, state map[string]interface{}) string {
	data := HydrationData{
		Props: props,
		State: state,
	}

	jsonData, err := json.Marshal(data)
	if err != nil {
		return ""
	}

	return fmt.Sprintf(`<script type="application/gox-hydration">%s</script>`, string(jsonData))
}