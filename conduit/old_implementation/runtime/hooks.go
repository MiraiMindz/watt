package gox

import (
	"reflect"
	"sync"
)

// HookState manages the state for all hooks in a component
type HookState struct {
	states    []interface{}
	effects   []Effect
	memos     []Memo
	callbacks []Callback
	refs      []interface{}
	contexts  map[string]interface{}
	index     int
	mu        sync.RWMutex
}

// NewHookState creates a new HookState instance
func NewHookState() *HookState {
	return &HookState{
		states:    []interface{}{},
		effects:   []Effect{},
		memos:     []Memo{},
		callbacks: []Callback{},
		refs:      []interface{}{},
		contexts:  make(map[string]interface{}),
		index:     0,
	}
}

// Reset resets the hook index for a new render
func (h *HookState) Reset() {
	h.mu.Lock()
	defer h.mu.Unlock()
	h.index = 0
}

// Effect represents a side effect hook
type Effect struct {
	Setup   func() func()
	Cleanup func()
	Deps    []interface{}
}

// Memo represents a memoized value
type Memo struct {
	Value interface{}
	Deps  []interface{}
}

// Callback represents a memoized callback function
type Callback struct {
	Func interface{}
	Deps []interface{}
}

// currentComponent is used to track which component is currently rendering
// This is a simplified implementation - in production, we'd use context or other mechanism
var (
	currentComponent *Component
	componentMu      sync.RWMutex
)

// SetCurrentComponent sets the currently rendering component
func SetCurrentComponent(c *Component) {
	componentMu.Lock()
	defer componentMu.Unlock()
	currentComponent = c
}

// GetCurrentComponent gets the currently rendering component
func GetCurrentComponent() *Component {
	componentMu.RLock()
	defer componentMu.RUnlock()
	return currentComponent
}

// UseState creates a state variable with a setter function
func UseState[T any](initial T) (T, func(T)) {
	comp := GetCurrentComponent()
	if comp == nil {
		panic("useState called outside of component render")
	}

	comp.hooks.mu.Lock()
	defer comp.hooks.mu.Unlock()

	idx := comp.hooks.index
	comp.hooks.index++

	// Initialize on first render
	if idx >= len(comp.hooks.states) {
		comp.hooks.states = append(comp.hooks.states, initial)
	}

	// Get current state value
	state, ok := comp.hooks.states[idx].(T)
	if !ok {
		// Type mismatch - use initial value
		state = initial
		comp.hooks.states[idx] = initial
	}

	// Create setter function
	setter := func(newValue T) {
		comp.hooks.mu.Lock()
		comp.hooks.states[idx] = newValue
		comp.hooks.mu.Unlock()
		comp.RequestUpdate()
	}

	return state, setter
}

// UseEffect runs a side effect after render
func UseEffect(setup func() func(), deps []interface{}) {
	comp := GetCurrentComponent()
	if comp == nil {
		panic("useEffect called outside of component render")
	}

	comp.hooks.mu.Lock()
	defer comp.hooks.mu.Unlock()

	idx := comp.hooks.index
	comp.hooks.index++

	// Check if deps changed
	if idx >= len(comp.hooks.effects) {
		// First run
		cleanup := setup()
		comp.hooks.effects = append(comp.hooks.effects, Effect{
			Setup:   setup,
			Cleanup: cleanup,
			Deps:    deps,
		})
	} else {
		effect := comp.hooks.effects[idx]
		if depsChanged(effect.Deps, deps) {
			// Run cleanup
			if effect.Cleanup != nil {
				effect.Cleanup()
			}
			// Run setup
			cleanup := setup()
			comp.hooks.effects[idx] = Effect{
				Setup:   setup,
				Cleanup: cleanup,
				Deps:    deps,
			}
		}
	}
}

// UseMemo memoizes a computed value
func UseMemo[T any](compute func() T, deps []interface{}) T {
	comp := GetCurrentComponent()
	if comp == nil {
		panic("useMemo called outside of component render")
	}

	comp.hooks.mu.Lock()
	defer comp.hooks.mu.Unlock()

	idx := comp.hooks.index
	comp.hooks.index++

	if idx >= len(comp.hooks.memos) {
		// First run
		value := compute()
		comp.hooks.memos = append(comp.hooks.memos, Memo{
			Value: value,
			Deps:  deps,
		})
		return value
	}

	memo := comp.hooks.memos[idx]
	if depsChanged(memo.Deps, deps) {
		value := compute()
		comp.hooks.memos[idx] = Memo{
			Value: value,
			Deps:  deps,
		}
		return value
	}

	result, ok := memo.Value.(T)
	if !ok {
		// Type mismatch - recompute
		value := compute()
		comp.hooks.memos[idx] = Memo{
			Value: value,
			Deps:  deps,
		}
		return value
	}

	return result
}

// UseCallback memoizes a callback function
func UseCallback[T any](callback T, deps []interface{}) T {
	comp := GetCurrentComponent()
	if comp == nil {
		panic("useCallback called outside of component render")
	}

	comp.hooks.mu.Lock()
	defer comp.hooks.mu.Unlock()

	idx := comp.hooks.index
	comp.hooks.index++

	if idx >= len(comp.hooks.callbacks) {
		comp.hooks.callbacks = append(comp.hooks.callbacks, Callback{
			Func: callback,
			Deps: deps,
		})
		return callback
	}

	cb := comp.hooks.callbacks[idx]
	if depsChanged(cb.Deps, deps) {
		comp.hooks.callbacks[idx] = Callback{
			Func: callback,
			Deps: deps,
		}
		return callback
	}

	result, ok := cb.Func.(T)
	if !ok {
		// Type mismatch - use new callback
		comp.hooks.callbacks[idx] = Callback{
			Func: callback,
			Deps: deps,
		}
		return callback
	}

	return result
}

// UseRef creates a mutable reference that persists across renders
func UseRef[T any](initial T) *Ref[T] {
	comp := GetCurrentComponent()
	if comp == nil {
		panic("useRef called outside of component render")
	}

	comp.hooks.mu.Lock()
	defer comp.hooks.mu.Unlock()

	idx := comp.hooks.index
	comp.hooks.index++

	if idx >= len(comp.hooks.refs) {
		ref := NewRef(initial)
		comp.hooks.refs = append(comp.hooks.refs, ref)
		return ref
	}

	ref, ok := comp.hooks.refs[idx].(*Ref[T])
	if !ok {
		// Type mismatch - create new ref
		ref = NewRef(initial)
		comp.hooks.refs[idx] = ref
	}

	return ref
}

// UseContext retrieves a value from context
func UseContext[T any](ctx *Context[T]) T {
	comp := GetCurrentComponent()
	if comp == nil {
		panic("useContext called outside of component render")
	}

	comp.hooks.mu.RLock()
	defer comp.hooks.mu.RUnlock()

	// Look up context value from component tree
	if provider, ok := comp.hooks.contexts[ctx.key]; ok {
		if p, ok := provider.(*ContextProvider[T]); ok {
			return p.Value
		}
	}

	// Return default value if no provider found
	return ctx.defaultValue
}

// UseReducer provides a reducer-based state management hook
func UseReducer[S any, A any](reducer func(S, A) S, initialState S) (S, func(A)) {
	state, setState := UseState(initialState)

	dispatch := func(action A) {
		newState := reducer(state, action)
		setState(newState)
	}

	return state, dispatch
}

// UseLayoutEffect is similar to UseEffect but runs synchronously after DOM mutations
// For SSR, it behaves the same as UseEffect
func UseLayoutEffect(setup func() func(), deps []interface{}) {
	UseEffect(setup, deps)
}

// UseId generates a stable unique ID
func UseId() string {
	comp := GetCurrentComponent()
	if comp == nil {
		panic("useId called outside of component render")
	}

	// Use a ref to store the ID so it persists across renders
	idRef := UseRef[string]("")

	if idRef.Current == "" {
		idRef.Current = generateID()
	}

	return idRef.Current
}

// UseDeferredValue returns a deferred version of the value
// For SSR, it returns the value immediately
func UseDeferredValue[T any](value T) T {
	// In SSR mode, we don't defer values
	// In CSR mode with concurrent features, this would defer updates
	return value
}

// UseTransition provides a way to mark updates as non-urgent
// For SSR, it returns a simple implementation
func UseTransition() (bool, func(func())) {
	isPending, setIsPending := UseState(false)

	startTransition := func(callback func()) {
		setIsPending(true)
		// In a real implementation, this would schedule the callback
		// For now, we execute it immediately
		callback()
		setIsPending(false)
	}

	return isPending, startTransition
}

// Helper function to check if dependencies have changed
func depsChanged(oldDeps, newDeps []interface{}) bool {
	if len(oldDeps) != len(newDeps) {
		return true
	}

	for i := range oldDeps {
		if !reflect.DeepEqual(oldDeps[i], newDeps[i]) {
			return true
		}
	}

	return false
}