package main

import (
	"fmt"
	"github.com/user/gox/runtime/wasm"
	"syscall/js"
)

// Counter is a GoX component for WASM
type Counter struct {
	*wasm.Component
	InitialCount int
	count        int
	element      js.Value
	document     js.Value
}

// NewCounter creates a new Counter component for WASM
func NewCounter(initialCount int) *Counter {
	c := &Counter{
		Component:    wasm.NewComponent(),
		document:     js.Global().Get("document"),
		InitialCount: initialCount,
	}

	// Initialize hooks
	wasm.SetCurrentComponent(c.Component)
	defer wasm.SetCurrentComponent(nil)

	return c
}

// Render renders the component to the DOM
func (c *Counter) Render(container js.Value) {
	c.element = container

	// Clear existing content
	container.Set("innerHTML", "")

	// Create virtual DOM
	vdom := c.createVDOM()

	// Render to actual DOM
	c.renderVDOM(vdom, container)
}

// createVDOM creates the virtual DOM for the component
func (c *Counter) createVDOM() *wasm.VNode {
	vnode := &wasm.VNode{
		Tag:      "div",
		Attrs:    make(map[string]string),
		Events:   make(map[string]js.Func),
		Children: []*wasm.VNode{},
	}

	vnode.Attrs["class"] = "counter-container"

	// Add children
	vnode.Children = append(vnode.Children, func() *wasm.VNode {
		vnode := &wasm.VNode{
			Tag:      "h1",
			Attrs:    make(map[string]string),
			Events:   make(map[string]js.Func),
			Children: []*wasm.VNode{},
		}

		// Add children
		vnode.Children = append(vnode.Children, &wasm.VNode{
			Text: "GoX Counter (WASM)",
		})
		return vnode
	}())
	vnode.Children = append(vnode.Children, func() *wasm.VNode {
		vnode := &wasm.VNode{
			Tag:      "div",
			Attrs:    make(map[string]string),
			Events:   make(map[string]js.Func),
			Children: []*wasm.VNode{},
		}

		vnode.Attrs["class"] = "count-value"

		// Add children
		vnode.Children = append(vnode.Children, &wasm.VNode{
			Text: fmt.Sprint(c.count),
		})
		return vnode
	}())
	vnode.Children = append(vnode.Children, func() *wasm.VNode {
		vnode := &wasm.VNode{
			Tag:      "div",
			Attrs:    make(map[string]string),
			Events:   make(map[string]js.Func),
			Children: []*wasm.VNode{},
		}

		vnode.Attrs["class"] = "counter-controls"

		// Add children
		vnode.Children = append(vnode.Children, func() *wasm.VNode {
			vnode := &wasm.VNode{
				Tag:      "button",
				Attrs:    make(map[string]string),
				Events:   make(map[string]js.Func),
				Children: []*wasm.VNode{},
			}

			vnode.Attrs["id"] = "increment"
			vnode.Events["click"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				c.increment()
				return nil
			})

			// Add children
			vnode.Children = append(vnode.Children, &wasm.VNode{
				Text: "+ Increment",
			})
			return vnode
		}())
		vnode.Children = append(vnode.Children, func() *wasm.VNode {
			vnode := &wasm.VNode{
				Tag:      "button",
				Attrs:    make(map[string]string),
				Events:   make(map[string]js.Func),
				Children: []*wasm.VNode{},
			}

			vnode.Attrs["class"] = "btn-secondary"
			vnode.Attrs["id"] = "decrement"
			vnode.Events["click"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				c.decrement()
				return nil
			})

			// Add children
			vnode.Children = append(vnode.Children, &wasm.VNode{
				Text: "- Decrement",
			})
			return vnode
		}())
		vnode.Children = append(vnode.Children, func() *wasm.VNode {
			vnode := &wasm.VNode{
				Tag:      "button",
				Attrs:    make(map[string]string),
				Events:   make(map[string]js.Func),
				Children: []*wasm.VNode{},
			}

			vnode.Attrs["id"] = "reset"
			vnode.Attrs["class"] = "btn-danger"
			vnode.Events["click"] = js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				c.reset()
				return nil
			})

			// Add children
			vnode.Children = append(vnode.Children, &wasm.VNode{
				Text: "Reset",
			})
			return vnode
		}())
		return vnode
	}())
	return vnode
}

// renderVDOM renders virtual DOM to actual DOM
func (c *Counter) renderVDOM(vnode *wasm.VNode, container js.Value) {
	if vnode == nil {
		return
	}

	// Create DOM element
	elem := c.document.Call("createElement", vnode.Tag)

	// Set attributes
	for name, value := range vnode.Attrs {
		elem.Call("setAttribute", name, value)
	}

	// Set event listeners
	for event, handler := range vnode.Events {
		elem.Call("addEventListener", event, handler)
	}

	// Render children
	for _, child := range vnode.Children {
		if child.Tag == "" {
			// Text node
			text := c.document.Call("createTextNode", child.Text)
			elem.Call("appendChild", text)
		} else {
			// Element node
			c.renderVDOM(child, elem)
		}
	}

	// Append to container
	container.Call("appendChild", elem)
}

// Update re-renders the component
func (c *Counter) Update() {
	if c.element.IsUndefined() {
		return
	}
	c.Render(c.element)
}

// setCount updates the count state
func (c *Counter) setCount(value int) {
	c.count = value
	c.Update()
}

// Event handlers
func (c *Counter) increment() {
	c.setCount(c.count + 1)
}

func (c *Counter) decrement() {
	c.setCount(c.count - 1)
}

func (c *Counter) reset() {
	c.setCount(c.InitialCount)
}


// WASM exports
func main() {
	// Register component factory
	js.Global().Set("Counter", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
		// Parse props from arguments
		comp := NewCounter(args[0].Int())
		return comp
	}))

	// Keep the Go program running
	select {}
}
