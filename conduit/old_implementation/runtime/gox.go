// Package gox provides the runtime library for GoX components
package gox

import (
	"fmt"
	"sync"
)

// Component is the base type for all GoX components
type Component struct {
	id           string
	currentVNode *VNode
	hooks        *HookState
	context      map[string]interface{}
	updateQueue  chan struct{}
	mounted      bool
	mu           sync.RWMutex
}

// NewComponent creates a new Component instance
func NewComponent() *Component {
	return &Component{
		id:          generateID(),
		hooks:       NewHookState(),
		context:     make(map[string]interface{}),
		updateQueue: make(chan struct{}, 1),
	}
}

// RequestUpdate queues a re-render of the component
func (c *Component) RequestUpdate() {
	select {
	case c.updateQueue <- struct{}{}:
		// Update queued
	default:
		// Update already pending
	}
}

// Mount marks the component as mounted
func (c *Component) Mount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mounted = true
}

// Unmount marks the component as unmounted
func (c *Component) Unmount() {
	c.mu.Lock()
	defer c.mu.Unlock()
	c.mounted = false
	c.cleanupEffects()
}

// IsMounted returns whether the component is mounted
func (c *Component) IsMounted() bool {
	c.mu.RLock()
	defer c.mu.RUnlock()
	return c.mounted
}

// cleanupEffects runs cleanup functions for all effects
func (c *Component) cleanupEffects() {
	if c.hooks != nil {
		for _, effect := range c.hooks.effects {
			if effect.Cleanup != nil {
				effect.Cleanup()
			}
		}
	}
}

// VNode represents a virtual DOM node
type VNode struct {
	Type     VNodeType
	Tag      string
	Props    Props
	Children []*VNode
	Key      string
	Ref      interface{}
	Text     string // For text nodes
}

// VNodeType represents the type of virtual node
type VNodeType int

const (
	VNodeTypeElement VNodeType = iota
	VNodeTypeComponent
	VNodeTypeText
	VNodeTypeFragment
)

// Props represents properties for elements and components
type Props map[string]interface{}

// H creates a VNode for an HTML element
func H(tag string, props Props, children ...*VNode) *VNode {
	return &VNode{
		Type:     VNodeTypeElement,
		Tag:      tag,
		Props:    props,
		Children: children,
	}
}

// Text creates a VNode for text content
func Text(content string) *VNode {
	return &VNode{
		Type: VNodeTypeText,
		Text: content,
	}
}

// Fragment creates a VNode for a fragment
func Fragment(children ...*VNode) *VNode {
	return &VNode{
		Type:     VNodeTypeFragment,
		Children: children,
	}
}

// Ref represents a mutable reference
type Ref[T any] struct {
	Current T
	mu      sync.RWMutex
}

// NewRef creates a new Ref with an initial value
func NewRef[T any](initial T) *Ref[T] {
	return &Ref[T]{Current: initial}
}

// Get returns the current value of the ref
func (r *Ref[T]) Get() T {
	r.mu.RLock()
	defer r.mu.RUnlock()
	return r.Current
}

// Set updates the value of the ref
func (r *Ref[T]) Set(value T) {
	r.mu.Lock()
	defer r.mu.Unlock()
	r.Current = value
}

// Context represents a context for passing data through the component tree
type Context[T any] struct {
	defaultValue T
	key          string
}

// ContextProvider provides a context value to child components
type ContextProvider[T any] struct {
	Context *Context[T]
	Value   T
}

// CreateContext creates a new context with a default value
func CreateContext[T any](defaultValue T) *Context[T] {
	return &Context[T]{
		defaultValue: defaultValue,
		key:          generateID(),
	}
}

// Provide creates a provider for this context
func (ctx *Context[T]) Provide(value T) *ContextProvider[T] {
	return &ContextProvider[T]{
		Context: ctx,
		Value:   value,
	}
}

// GetDefault returns the default value of the context
func (ctx *Context[T]) GetDefault() T {
	return ctx.defaultValue
}

// generateID generates a unique ID for components and contexts
var idCounter uint64
var idMu sync.Mutex

func generateID() string {
	idMu.Lock()
	defer idMu.Unlock()
	idCounter++
	return fmt.Sprintf("gox-%d", idCounter)
}