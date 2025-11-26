// Package wasm provides WebAssembly runtime support for GoX components
package wasm

import (
	"syscall/js"
	"fmt"
)

// Component represents a base component for WASM
type Component struct {
	hooks       *HookManager
	mounted     bool
	updateQueue []func()
	props       map[string]interface{}
	refs        map[string]js.Value
	hydrator    *Hydrator
	isHydrated  bool
}

// VNode represents a virtual DOM node
type VNode struct {
	Tag      string
	Attrs    map[string]string
	Events   map[string]js.Func
	Children []*VNode
	Text     string
	Key      string
}

// NewComponent creates a new WASM component
func NewComponent() *Component {
	return &Component{
		hooks:       NewHookManager(),
		props:       make(map[string]interface{}),
		refs:        make(map[string]js.Value),
		updateQueue: []func(){},
	}
}

// Mount mounts the component to a DOM element
func (c *Component) Mount(container js.Value) {
	c.mounted = true
	c.runUpdateQueue()
}

// Unmount cleans up the component
func (c *Component) Unmount() {
	c.mounted = false
	// Clean up event listeners
	for _, ref := range c.refs {
		if !ref.IsUndefined() {
			// Remove event listeners if needed
		}
	}
}

// RequestUpdate queues an update
func (c *Component) RequestUpdate(update func()) {
	c.updateQueue = append(c.updateQueue, update)
	if c.mounted {
		js.Global().Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			c.runUpdateQueue()
			return nil
		}))
	}
}

// runUpdateQueue processes all queued updates
func (c *Component) runUpdateQueue() {
	queue := c.updateQueue
	c.updateQueue = []func(){}

	for _, update := range queue {
		if update != nil {
			update()
		}
	}
}

// SetProp sets a component property
func (c *Component) SetProp(name string, value interface{}) {
	c.props[name] = value
}

// GetProp gets a component property
func (c *Component) GetProp(name string) interface{} {
	return c.props[name]
}

// CreateElement creates a DOM element from a VNode
func CreateElement(vnode *VNode) js.Value {
	if vnode == nil {
		return js.Undefined()
	}

	document := js.Global().Get("document")

	// Text node
	if vnode.Tag == "" && vnode.Text != "" {
		return document.Call("createTextNode", vnode.Text)
	}

	// Element node
	elem := document.Call("createElement", vnode.Tag)

	// Set attributes
	for name, value := range vnode.Attrs {
		elem.Call("setAttribute", name, value)
	}

	// Set event listeners
	for event, handler := range vnode.Events {
		elem.Call("addEventListener", event, handler)
	}

	// Add children
	for _, child := range vnode.Children {
		childElem := CreateElement(child)
		if !childElem.IsUndefined() {
			elem.Call("appendChild", childElem)
		}
	}

	return elem
}

// Diff performs virtual DOM diffing
func Diff(oldVNode, newVNode *VNode, container js.Value) {
	if oldVNode == nil && newVNode == nil {
		return
	}

	// Node added
	if oldVNode == nil {
		elem := CreateElement(newVNode)
		container.Call("appendChild", elem)
		return
	}

	// Node removed
	if newVNode == nil {
		container.Set("innerHTML", "")
		return
	}

	// Node type changed
	if oldVNode.Tag != newVNode.Tag {
		elem := CreateElement(newVNode)
		container.Set("innerHTML", "")
		container.Call("appendChild", elem)
		return
	}

	// Update attributes
	elem := container.Get("firstChild")
	if !elem.IsUndefined() {
		// Remove old attributes
		for name := range oldVNode.Attrs {
			if _, exists := newVNode.Attrs[name]; !exists {
				elem.Call("removeAttribute", name)
			}
		}

		// Set new attributes
		for name, value := range newVNode.Attrs {
			if oldValue, exists := oldVNode.Attrs[name]; !exists || oldValue != value {
				elem.Call("setAttribute", name, value)
			}
		}

		// Update children
		diffChildren(oldVNode.Children, newVNode.Children, elem)
	}
}

// diffChildren diffs child nodes
func diffChildren(oldChildren, newChildren []*VNode, container js.Value) {
	maxLen := len(oldChildren)
	if len(newChildren) > maxLen {
		maxLen = len(newChildren)
	}

	childNodes := container.Get("childNodes")

	for i := 0; i < maxLen; i++ {
		var oldChild, newChild *VNode
		var childContainer js.Value

		if i < len(oldChildren) {
			oldChild = oldChildren[i]
		}
		if i < len(newChildren) {
			newChild = newChildren[i]
		}
		if i < childNodes.Length() {
			childContainer = childNodes.Index(i)
		}

		if oldChild == nil && newChild != nil {
			// Add new child
			elem := CreateElement(newChild)
			container.Call("appendChild", elem)
		} else if oldChild != nil && newChild == nil {
			// Remove old child
			if !childContainer.IsUndefined() {
				container.Call("removeChild", childContainer)
			}
		} else if oldChild != nil && newChild != nil {
			// Update existing child
			if !childContainer.IsUndefined() {
				if oldChild.Tag != newChild.Tag {
					// Replace if tag changed
					elem := CreateElement(newChild)
					container.Call("replaceChild", elem, childContainer)
				} else if oldChild.Text != newChild.Text {
					// Update text content
					childContainer.Set("textContent", newChild.Text)
				} else {
					// Recursively diff element children
					Diff(oldChild, newChild, childContainer)
				}
			}
		}
	}
}

// Render renders a component to a container
func Render(component interface{}, container js.Value) {
	// Type assertion to get render method
	if renderer, ok := component.(interface{ Render(js.Value) }); ok {
		renderer.Render(container)
	} else {
		fmt.Println("Component does not implement Render method")
	}
}

// Hydrate hydrates server-rendered HTML with client-side interactivity
func (c *Component) Hydrate(container js.Value, vnode *VNode) error {
	if c.isHydrated {
		return fmt.Errorf("component already hydrated")
	}

	// Create hydrator
	c.hydrator = NewHydrator(c, container)

	// Perform hydration
	if err := c.hydrator.Hydrate(vnode); err != nil {
		return fmt.Errorf("hydration failed: %w", err)
	}

	// Extract initial state from hydration data
	if data := c.hydrator.GetHydrationData(); data != nil {
		// Apply hydrated state
		for key, value := range data.State {
			c.props[key] = value
		}
	}

	c.mounted = true
	c.isHydrated = true
	return nil
}

// SetCurrentComponent sets the current component for hooks
var currentComponent *Component

func SetCurrentComponent(c *Component) {
	currentComponent = c
}

// GetCurrentComponent gets the current component
func GetCurrentComponent() *Component {
	return currentComponent
}