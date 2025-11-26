package main

import (
	"fmt"
)

// This file demonstrates what goxc would generate from Counter.gox
// in SSR mode

// Counter component struct
type Counter struct {
	// Props
	initialValue int

	// State
	count int
}

// NewCounter creates a new Counter component
func NewCounter(initialValue int) *Counter {
	c := &Counter{
		initialValue: initialValue,
		count:        initialValue,
	}
	return c
}

// Render generates HTML for the component
func (c *Counter) Render() string {
	return fmt.Sprintf(`<div class="counter-container" data-gox-counter>
		<div class="counter-display">
			<h1>Counter</h1>
			<div class="count-value">%d</div>
		</div>

		<div class="counter-controls">
			<button class="btn btn-primary">
				+ Increment
			</button>
			<button class="btn btn-secondary">
				- Decrement
			</button>
			<button class="btn btn-danger">
				Reset
			</button>
		</div>

		<div class="counter-info">
			<p>Initial value: %d</p>
			<p>Current value: %d</p>
			<p>Difference: %d</p>
		</div>
	</div>`,
		c.count,
		c.initialValue,
		c.count,
		c.count-c.initialValue,
	)
}

// setCount updates the count state
func (c *Counter) setCount(value int) {
	c.count = value
	// In SSR mode, this would trigger a re-render if needed
	// For static SSR, state changes don't apply
}

// increment increments the counter
func (c *Counter) increment() {
	c.setCount(c.count + 1)
}

// decrement decrements the counter
func (c *Counter) decrement() {
	c.setCount(c.count - 1)
}

// reset resets the counter to initial value
func (c *Counter) reset() {
	c.setCount(c.initialValue)
}
