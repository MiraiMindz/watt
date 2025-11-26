package wasm

import (
	"syscall/js"
	"reflect"
)

// HookManager manages hooks for a WASM component
type HookManager struct {
	hookIndex   int
	states      []interface{}
	effects     []effectHook
	memos       []memoHook
	callbacks   []callbackHook
	refs        []refHook
	contexts    map[string]interface{}
	cleanups    []func()
}

type effectHook struct {
	effect func()
	deps   []interface{}
}

type memoHook struct {
	compute func() interface{}
	value   interface{}
	deps    []interface{}
}

type callbackHook struct {
	callback interface{}
	deps     []interface{}
}

type refHook struct {
	current interface{}
}

// NewHookManager creates a new hook manager
func NewHookManager() *HookManager {
	return &HookManager{
		states:    []interface{}{},
		effects:   []effectHook{},
		memos:     []memoHook{},
		callbacks: []callbackHook{},
		refs:      []refHook{},
		contexts:  make(map[string]interface{}),
		cleanups:  []func(){},
	}
}

// Reset resets the hook index for a new render
func (h *HookManager) Reset() {
	h.hookIndex = 0
}

// UseState implements the useState hook for WASM
func UseState[T any](initial T) (T, func(T)) {
	comp := GetCurrentComponent()
	if comp == nil || comp.hooks == nil {
		var zero T
		return zero, func(T) {}
	}

	h := comp.hooks
	currentIndex := h.hookIndex
	h.hookIndex++

	// Initialize state if first render
	if currentIndex >= len(h.states) {
		h.states = append(h.states, initial)
	}

	// Get current value
	value := h.states[currentIndex].(T)

	// Create setter function
	setter := func(newValue T) {
		h.states[currentIndex] = newValue
		// Trigger re-render
		if comp.mounted {
			js.Global().Call("requestAnimationFrame", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
				// Re-render component
				comp.RequestUpdate(nil)
				return nil
			}))
		}
	}

	return value, setter
}

// UseEffect implements the useEffect hook for WASM
func UseEffect(effect func(), deps []interface{}) {
	comp := GetCurrentComponent()
	if comp == nil || comp.hooks == nil {
		return
	}

	h := comp.hooks
	currentIndex := h.hookIndex
	h.hookIndex++

	// Check if dependencies changed
	shouldRun := false
	if currentIndex >= len(h.effects) {
		// First render
		h.effects = append(h.effects, effectHook{effect: effect, deps: deps})
		shouldRun = true
	} else {
		// Check if deps changed
		oldDeps := h.effects[currentIndex].deps
		if !depsEqual(oldDeps, deps) {
			h.effects[currentIndex] = effectHook{effect: effect, deps: deps}
			shouldRun = true
		}
	}

	if shouldRun {
		// Schedule effect to run after render
		js.Global().Call("setTimeout", js.FuncOf(func(this js.Value, args []js.Value) interface{} {
			// Run cleanup if exists
			if currentIndex < len(h.cleanups) && h.cleanups[currentIndex] != nil {
				h.cleanups[currentIndex]()
			}

			// Run effect
			effect()
			return nil
		}), 0)
	}
}

// UseMemo implements the useMemo hook for WASM
func UseMemo[T any](compute func() T, deps []interface{}) T {
	comp := GetCurrentComponent()
	if comp == nil || comp.hooks == nil {
		return compute()
	}

	h := comp.hooks
	currentIndex := h.hookIndex
	h.hookIndex++

	// Check if we need to recompute
	if currentIndex >= len(h.memos) {
		// First render
		value := compute()
		h.memos = append(h.memos, memoHook{
			compute: func() interface{} { return compute() },
			value:   value,
			deps:    deps,
		})
		return value
	}

	// Check if deps changed
	oldDeps := h.memos[currentIndex].deps
	if !depsEqual(oldDeps, deps) {
		// Recompute
		value := compute()
		h.memos[currentIndex] = memoHook{
			compute: func() interface{} { return compute() },
			value:   value,
			deps:    deps,
		}
		return value
	}

	// Return memoized value
	return h.memos[currentIndex].value.(T)
}

// UseCallback implements the useCallback hook for WASM
func UseCallback[T any](callback T, deps []interface{}) T {
	comp := GetCurrentComponent()
	if comp == nil || comp.hooks == nil {
		return callback
	}

	h := comp.hooks
	currentIndex := h.hookIndex
	h.hookIndex++

	// Check if we need to update callback
	if currentIndex >= len(h.callbacks) {
		// First render
		h.callbacks = append(h.callbacks, callbackHook{
			callback: callback,
			deps:     deps,
		})
		return callback
	}

	// Check if deps changed
	oldDeps := h.callbacks[currentIndex].deps
	if !depsEqual(oldDeps, deps) {
		// Update callback
		h.callbacks[currentIndex] = callbackHook{
			callback: callback,
			deps:     deps,
		}
		return callback
	}

	// Return memoized callback
	return h.callbacks[currentIndex].callback.(T)
}

// UseRef implements the useRef hook for WASM
func UseRef[T any](initial T) *RefObject[T] {
	comp := GetCurrentComponent()
	if comp == nil || comp.hooks == nil {
		return &RefObject[T]{Current: initial}
	}

	h := comp.hooks
	currentIndex := h.hookIndex
	h.hookIndex++

	// Initialize ref if first render
	if currentIndex >= len(h.refs) {
		h.refs = append(h.refs, refHook{current: initial})
	}

	// Create ref object
	return &RefObject[T]{
		Current: h.refs[currentIndex].current.(T),
		index:   currentIndex,
		hooks:   h,
	}
}

// RefObject holds a mutable reference
type RefObject[T any] struct {
	Current T
	index   int
	hooks   *HookManager
}

// Set updates the ref value
func (r *RefObject[T]) Set(value T) {
	r.Current = value
	if r.hooks != nil && r.index < len(r.hooks.refs) {
		r.hooks.refs[r.index].current = value
	}
}

// UseContext implements the useContext hook for WASM
func UseContext[T any](key string) T {
	comp := GetCurrentComponent()
	if comp == nil || comp.hooks == nil {
		var zero T
		return zero
	}

	if value, exists := comp.hooks.contexts[key]; exists {
		return value.(T)
	}

	var zero T
	return zero
}

// CreateContext creates a new context
func CreateContext[T any](key string, defaultValue T) {
	comp := GetCurrentComponent()
	if comp != nil && comp.hooks != nil {
		comp.hooks.contexts[key] = defaultValue
	}
}

// depsEqual checks if two dependency arrays are equal
func depsEqual(a, b []interface{}) bool {
	if len(a) != len(b) {
		return false
	}
	for i := range a {
		if !reflect.DeepEqual(a[i], b[i]) {
			return false
		}
	}
	return true
}

// Cleanup registers a cleanup function for the current effect
func Cleanup(cleanup func()) {
	comp := GetCurrentComponent()
	if comp == nil || comp.hooks == nil {
		return
	}

	h := comp.hooks
	currentIndex := h.hookIndex - 1 // Use previous index (the effect that called this)

	// Ensure cleanups array is large enough
	for len(h.cleanups) <= currentIndex {
		h.cleanups = append(h.cleanups, nil)
	}

	h.cleanups[currentIndex] = cleanup
}