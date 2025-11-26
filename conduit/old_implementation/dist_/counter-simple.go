package examples

import (
	"fmt"
	"github.com/user/gox/runtime"
)

// Counter is a GoX component
type Counter struct {
	*gox.Component
	count int
}

// NewCounter creates a new Counter component
func NewCounter() *Counter {
	c := &Counter{
		Component: gox.NewComponent(),
	}

	c.count = 0
	return c
}

// Render generates the HTML for the component
func (c *Counter) Render() string {
	gox.SetCurrentComponent(c.Component)
	defer func() { gox.SetCurrentComponent(nil) }()

	c.Component.hooks.Reset()

	return `<div><h1>Counter: ${c.count}</h1></div>`
}

// setCount updates the count state
func (c *Counter) setCount(value int) {
	c.count = value
	c.RequestUpdate()
}
