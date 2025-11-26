package main

import (
	"fmt"
	"github.com/user/gox/runtime"
)

// Test is a GoX component
type Test struct {
	*gox.Component
}

// NewTest creates a new Test component
func NewTest() *Test {
	c := &Test{
		Component: gox.NewComponent(),
	}

	return c
}

// Render generates the HTML for the component
func (c *Test) Render() string {
	gox.SetCurrentComponent(c.Component)
	defer func() { gox.SetCurrentComponent(nil) }()

	c.Component.hooks.Reset()

	return `<div>Hello</div>`
}
