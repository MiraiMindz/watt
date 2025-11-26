package main

import (
	"fmt"
	"github.com/user/gox/runtime"
)

// SimpleTest is a GoX component
type SimpleTest struct {
	*gox.Component
	count int
}

// NewSimpleTest creates a new SimpleTest component
func NewSimpleTest() *SimpleTest {
	c := &SimpleTest{
		Component: gox.NewComponent(),
	}

	c.count = 0
	return c
}

// Render generates the HTML for the component
func (c *SimpleTest) Render() string {
	gox.SetCurrentComponent(c.Component)
	defer func() { gox.SetCurrentComponent(nil) }()

	c.Component.hooks.Reset()

	return ""
}

// setCount updates the count state
func (c *SimpleTest) setCount(value int) {
	c.count = value
	c.RequestUpdate()
}
