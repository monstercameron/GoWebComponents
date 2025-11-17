package runtime

import (
	"fmt"
	"reflect"
)

// GoUseState provides state management for components
func GoUseState[T any](rt *Runtime, initialValue T) (func() T, func(interface{})) {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseState called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	// TODO: propagate validateHookOrder errors instead of ignoring them
	validateHookOrder(fiber.hooks, HookTypeState, position)

	// Initialize state if needed
	if len(fiber.hooks.state) <= position {
		// Grow slice
		newState := make([]interface{}, position+1, (position+1)*2)
		copy(newState, fiber.hooks.state)
		newState[position] = initialValue
		fiber.hooks.state = newState
	}

	// Initialize pending state if needed
	if len(fiber.hooks.pendingState) <= position {
		newPending := make([]interface{}, position+1, (position+1)*2)
		copy(newPending, fiber.hooks.pendingState)
		// Initially, pending equals committed
		newPending[position] = fiber.hooks.state[position]
		fiber.hooks.pendingState = newPending
	}

	hooks := fiber.hooks
	idx := position

	getter := func() T {
		if idx < len(hooks.state) {
			if value, ok := hooks.state[idx].(T); ok {
				return value
			}
		}
		var zero T
		return zero
	}

	setter := func(newValueOrUpdater interface{}) {
		if idx >= len(hooks.state) {
			// Expand state
			newState := make([]interface{}, idx+1)
			copy(newState, hooks.state)
			hooks.state = newState
		}
		if idx >= len(hooks.pendingState) {
			// Expand pending state
			newPending := make([]interface{}, idx+1)
			copy(newPending, hooks.pendingState)
			hooks.pendingState = newPending
		}

		// Get PENDING value (most recent update, may not be committed yet)
		var currentValue T
		if idx < len(hooks.pendingState) {
			if cv, ok := hooks.pendingState[idx].(T); ok {
				currentValue = cv
			}
		}

		// Determine the new value
		var newValue T
		// Try to treat as functional update (func(T) T)
		if fn, ok := newValueOrUpdater.(func(T) T); ok {
			newValue = fn(currentValue)
		} else if directValue, ok := newValueOrUpdater.(T); ok {
			// Direct value
			newValue = directValue
		} else {
			return
		}

		// Check if newValue is different from current PENDING value
		if !fastEqual(currentValue, newValue) {
			// Update pending state immediately (visible to next setState in same render cycle)
			hooks.pendingState[idx] = newValue
			// Also update committed state (this is what triggers re-render)
			hooks.state[idx] = newValue
			rt.ScheduleUpdateForFiber(fiber)
		}
	}

	return getter, setter
}

// GoUseEffect runs side effects and supports cleanup
// The effect function can return a cleanup function that will be called before the next effect runs or on unmount
func GoUseEffect(effect func() func(), deps ...interface{}) {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseEffect called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			callbacks:    make([]callbackValue, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeEffect, position)

	// Grow deps and cleanups if needed
	if len(fiber.hooks.deps) <= position {
		newDeps := make([][]interface{}, position+1, (position+1)*2)
		copy(newDeps, fiber.hooks.deps)
		fiber.hooks.deps = newDeps
	}
	if len(fiber.hooks.cleanups) <= position {
		newCleanups := make([]func(), position+1, (position+1)*2)
		copy(newCleanups, fiber.hooks.cleanups)
		fiber.hooks.cleanups = newCleanups
	}

	shouldRun := false
	if fiber.hooks.deps[position] == nil {
		// First run
		fiber.hooks.deps[position] = deps
		shouldRun = true
	} else {
		// Check if deps changed
		prevDeps := fiber.hooks.deps[position]
		if len(deps) == 0 || !areDepsEqual(prevDeps, deps) {
			fiber.hooks.deps[position] = deps
			shouldRun = true
		}
	}

	if shouldRun {
		// Run cleanup from previous effect if it exists
		if fiber.hooks.cleanups[position] != nil {
			fiber.hooks.cleanups[position]()
			fiber.hooks.cleanups[position] = nil
		}

		// Queue the new effect
		if fiber.effects == nil {
			fiber.effects = make([]func(), 0)
		}

		// Capture position for cleanup storage
		effectPosition := position
		fiber.effects = append(fiber.effects, func() {
			cleanup := effect()
			if cleanup != nil {
				fiber.hooks.cleanups[effectPosition] = cleanup
			}
		})
	}
}

// GoUseMemo memoizes expensive computations
func GoUseMemo(compute func() interface{}, deps ...interface{}) interface{} {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseMemo called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			callbacks:    make([]callbackValue, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeMemo, position)

	// Grow memos if needed
	if len(fiber.hooks.memos) <= position {
		newMemos := make([]memoizedValue, position+1, (position+1)*2)
		copy(newMemos, fiber.hooks.memos)
		fiber.hooks.memos = newMemos
	}

	memo := &fiber.hooks.memos[position]

	// Check if we need to recompute: only if deps changed (or this is the first render and memo.deps is uninitialized)
	// First render: memo.deps will be nil, so we need to initialize it
	// Subsequent renders: check if deps changed
	if memo.deps == nil || !areDepsEqual(memo.deps, deps) {
		memo.value = compute()
		memo.deps = deps
	}

	return memo.value
}

// GoUseCallback memoizes a callback function with dependency tracking
func GoUseCallback(fn interface{}, deps ...interface{}) interface{} {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseCallback called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			callbacks:    make([]callbackValue, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeCallback, position)

	// Grow callbacks if needed
	if len(fiber.hooks.callbacks) <= position {
		newCallbacks := make([]callbackValue, position+1, (position+1)*2)
		copy(newCallbacks, fiber.hooks.callbacks)
		fiber.hooks.callbacks = newCallbacks
	}

	callback := &fiber.hooks.callbacks[position]

	// Check if we need to update: only if deps changed or this is the first render
	if callback.deps == nil || !areDepsEqual(callback.deps, deps) {
		callback.fn = fn
		callback.deps = deps
	}

	return callback.fn
}

// GoUseRef creates a mutable reference that persists across renders
// It returns a RefValue object with a .Current field that can hold any value
// Unlike state, updating a ref does NOT trigger a re-render
func GoUseRef(initialValue interface{}) *RefValue {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseRef called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			callbacks:    make([]callbackValue, 0),
			refs:         make([]*RefValue, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeRef, position)

	// Grow refs if needed
	if len(fiber.hooks.refs) <= position {
		newRefs := make([]*RefValue, position+1, (position+1)*2)
		copy(newRefs, fiber.hooks.refs)
		fiber.hooks.refs = newRefs
	}

	// If ref is nil (first render), create it with initial value
	if fiber.hooks.refs[position] == nil {
		fiber.hooks.refs[position] = &RefValue{Current: initialValue}
	}

	return fiber.hooks.refs[position]
}

// GoUseId generates a unique, stable identifier for accessibility attributes
// The ID is generated once and persists across renders without changing
// This is useful for associating labels with form inputs and other accessibility needs
func GoUseId() string {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseId called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			callbacks:    make([]callbackValue, 0),
			refs:         make([]*RefValue, 0),
			ids:          make([]string, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeId, position)

	// Initialize IDs array if needed
	if len(fiber.hooks.ids) <= position {
		newIds := make([]string, position+1, (position+1)*2)
		copy(newIds, fiber.hooks.ids)

		// Generate a new unique ID
		// On first render, create a new ID using fiber's component ID counter
		rt := GetGlobalRuntime()
		rt.idCounterMu.Lock()
		rt.idCounter++
		id := rt.idCounter
		rt.idCounterMu.Unlock()

		// Create ID in format: "gwc:<global-id>:<hook-position>"
		// This ensures uniqueness and stability across renders
		newIds[position] = fmt.Sprintf("gwc:%d:%d", id, position)
		fiber.hooks.ids = newIds
	}

	// Return the persistent ID
	if position < len(fiber.hooks.ids) {
		return fiber.hooks.ids[position]
	}

	return ""
}

// GoUseFetch is a manual-trigger fetch hook that manages async data fetching
// It uses goroutines and channels internally to fetch data without blocking
// Unlike GoUseEffect-based auto-fetching, this requires explicit refetch() calls
//
// Returns:
//   - A getter function for the current FetchState (data, error, loading)
//   - A refetch function to manually trigger the fetch
//
// The FetchState contains:
//   - Data: the fetched response body (as string)
//   - Error: error message if fetch failed (empty string if successful)
//   - Loading: whether currently fetching
func GoUseFetch(url string, options ...interface{}) (func() FetchState, func()) {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseFetch called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			callbacks:    make([]callbackValue, 0),
			refs:         make([]*RefValue, 0),
			ids:          make([]string, 0),
			fetches:      make([]fetchValue, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeFetch, position)

	// Initialize fetch state if needed
	if len(fiber.hooks.fetches) <= position {
		newFetches := make([]fetchValue, position+1, (position+1)*2)
		copy(newFetches, fiber.hooks.fetches)
		newFetches[position] = fetchValue{
			state: FetchState{Data: nil, Error: "", Loading: false},
			url:   url,
		}
		fiber.hooks.fetches = newFetches
	}

	hooks := fiber.hooks
	idx := position

	// Getter returns current fetch state
	getter := func() FetchState {
		if idx < len(hooks.fetches) {
			return hooks.fetches[idx].state
		}
		return FetchState{Data: nil, Error: "", Loading: false}
	}

	// Refetch function to manually trigger a fetch
	refetch := func() {
		if idx >= len(hooks.fetches) {
			return
		}

		// Mark as loading
		hooks.fetches[idx].state = FetchState{Data: nil, Error: "", Loading: true}

		// Trigger component re-render
		rt := GetGlobalRuntime()
		if rt != nil {
			rt.ScheduleUpdateForFiber(fiber)
		}

		// Start fetch in a goroutine
		go func() {
			// Simple HTTP fetch using basic string concatenation for URL
			// In real implementation, would use proper HTTP client or JS fetch API
			// For now, this is a placeholder that shows the pattern

			// Simulate async fetch operation
			// In WASM context, this would call js.Global().Call("fetch", url)
			// and handle the promise chain

			// For demonstration:
			// 1. Make the fetch request
			// 2. When complete, update state
			// 3. Trigger re-render

			// Placeholder - actual implementation would use channels and async operations
			hooks.fetches[idx].state = FetchState{
				Data:    nil,
				Error:   "Fetch not fully implemented yet",
				Loading: false,
			}

			// Schedule re-render after fetch completes
			if rt != nil {
				rt.ScheduleUpdateForFiber(fiber)
			}
		}()
	}

	return getter, refetch
}

// GoUseFunc validates and stores a function for event handling
// The actual wrapping to js.Value happens in the WASM shim layer
func GoUseFunc(fn interface{}) interface{} {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseFunc called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:        make([]interface{}, 0),
			pendingState: make([]interface{}, 0),
			deps:         make([][]interface{}, 0),
			memos:        make([]memoizedValue, 0),
			callbacks:    make([]callbackValue, 0),
			refs:         make([]*RefValue, 0),
			ids:          make([]string, 0),
			fetches:      make([]fetchValue, 0),
			funcs:        make([]funcHandlerValue, 0),
			cleanups:     make([]func(), 0),
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeFunc, position)

	// Validate that the input is a function
	fnType := reflect.TypeOf(fn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		panic("GoUseFunc requires a function")
	}

	// Initialize handler if needed
	if len(fiber.hooks.funcs) <= position {
		newFuncs := make([]funcHandlerValue, position+1, (position+1)*2)
		copy(newFuncs, fiber.hooks.funcs)
		newFuncs[position] = funcHandlerValue{fn: fn}
		fiber.hooks.funcs = newFuncs
	}

	// Return the function - WASM shim will wrap it as js.Value
	return fn
}

// validateHookOrder checks that hooks are called in the same order
func validateHookOrder(hooks *Hooks, hookType HookType, position int) error {
	hookCall := HookCall{Type: hookType, Position: position}
	hooks.callOrder = append(hooks.callOrder, hookCall)

	// Skip validation on first render
	if len(hooks.prevOrder) == 0 {
		return nil
	}

	if position < len(hooks.prevOrder) {
		prevCall := hooks.prevOrder[position]
		if prevCall.Type != hookType {
			return fmt.Errorf("hook order violation: hook type mismatch at position %d", position)
		}
	} else {
		return fmt.Errorf("hook order violation: more hooks called than previous render")
	}

	return nil
}

// FinalizeHookOrder completes hook order validation
func FinalizeHookOrder(hooks *Hooks) error {
	if len(hooks.prevOrder) > 0 && len(hooks.callOrder) < len(hooks.prevOrder) {
		return fmt.Errorf("hook order violation: fewer hooks called than previous render")
	}

	hooks.prevOrder = make([]HookCall, len(hooks.callOrder))
	copy(hooks.prevOrder, hooks.callOrder)
	hooks.callOrder = make([]HookCall, 0)
	hooks.orderChecked = true

	return nil
}

// areDepsEqual compares dependency arrays
func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	if len(prevDeps) != len(newDeps) {
		return false
	}

	if len(prevDeps) == 0 {
		return true
	}

	for i := range prevDeps {
		if !fastEqual(prevDeps[i], newDeps[i]) {
			return false
		}
	}

	return true
}

// fastEqual performs optimized equality checking
func fastEqual(a, b interface{}) bool {
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Try direct comparison first, but only for comparable types
	ta := reflect.TypeOf(a)
	tb := reflect.TypeOf(b)
	if ta == tb && ta.Comparable() {
		if a == b {
			return true
		}
	}

	// Use reflect for complex types
	return reflect.DeepEqual(a, b)
}
