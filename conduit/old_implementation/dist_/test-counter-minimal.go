package main

import (
	"fmt"
	"github.com/user/gox/runtime"
)

// Counter is a GoX component
type Counter struct {
	*gox.Component
	InitialValue int
	count        int
}

// NewCounter creates a new Counter component
func NewCounter(initialValue int) *Counter {
	c := &Counter{
		Component:    gox.NewComponent(),
		InitialValue: initialValue,
	}

	c.count = initialValue
	return c
}

// Render generates the HTML for the component
func (c *Counter) Render() string {
	gox.SetCurrentComponent(c.Component)
	defer func() { gox.SetCurrentComponent(nil) }()

	c.Component.hooks.Reset()

	return `<div className="counter-container"><h1>Counter</h1><div>${c.count}</div><button>+ Increment</button><button>- Decrement</button><button>Reset</button><p>Initial value: ${c.initialValue}</p><p>Current value: ${c.count}</p><p>Difference: ${c.count}</p></div>`
}

// setCount updates the count state
func (c *Counter) setCount(value int) {
	c.count = value
	c.RequestUpdate()
}
