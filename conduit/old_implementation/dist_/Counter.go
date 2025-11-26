package counter_ssr

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

	return `<div className="counter-container"><div className="counter-display"><h1>Counter</h1><div className="count-value">${c.count}</div></div><div className="counter-controls"><button className="btn btn-primary">+ Increment
				</button><button className="btn btn-secondary">- Decrement
				</button><button className="btn btn-danger">Reset
				</button></div><div className="counter-info"><p>Initial value: ${c.initialValue}</p><p>Current value: ${c.count}</p><p>Difference: ${c.count}</p></div></div>`
}

// setCount updates the count state
func (c *Counter) setCount(value int) {
	c.count = value
	c.RequestUpdate()
}
