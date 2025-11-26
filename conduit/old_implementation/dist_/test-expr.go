package main

import (
	"fmt"
	"github.com/user/gox/runtime"
)

// Test is a GoX component
type Test struct {
	*gox.Component
	A int
	B int
}

// NewTest creates a new Test component
func NewTest(a int, b int) *Test {
	c := &Test{
		Component: gox.NewComponent(),
		A:         a,
		B:         b,
	}

	return c
}

// Render generates the HTML for the component
func (c *Test) Render() string {
	gox.SetCurrentComponent(c.Component)
	defer func() { gox.SetCurrentComponent(nil) }()

	c.Component.hooks.Reset()

	return `<div><p>${c.a}</p></div>`
}
