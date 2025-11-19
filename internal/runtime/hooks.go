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
		fiber.hooks = &Hooks{}
	}

	fiber.hooks.index++

	stateIdx := fiber.hooks.stateIndex
	fiber.hooks.stateIndex++

	// Initialize state if needed
	// We need 2 slots per state hook: [state, pending]
	neededLen := (stateIdx + 1) * 2
	if len(fiber.hooks.states) < neededLen {
		if neededLen <= cap(fiber.hooks.states) {
			// Extend slice within capacity
			fiber.hooks.states = fiber.hooks.states[:neededLen]
		} else {
			// Grow slice
			newStates := make([]interface{}, neededLen, neededLen*2)
			copy(newStates, fiber.hooks.states)
			fiber.hooks.states = newStates
		}
		
		// Initialize new slots
		fiber.hooks.states[stateIdx*2] = initialValue
		fiber.hooks.states[stateIdx*2+1] = initialValue
	}

	hooks := fiber.hooks
	// Capture indices for closure
	sIdx := stateIdx * 2
	pIdx := stateIdx * 2 + 1

	getter := func() T {
		// Bounds check removed for performance - slice is grown before closure creation
		if value, ok := hooks.states[sIdx].(T); ok {
			return value
		}
		var zero T
		return zero
	}

	setter := func(newValueOrUpdater interface{}) {
		// Bounds check removed for performance
		if pIdx >= len(hooks.states) {
			// This should rarely happen if logic is correct, but handle just in case of race/weirdness
			// Actually, if we are here, hooks.states might have been reallocated but we have the pointer to hooks
			// So len(hooks.states) should be valid.
			// But if we are adding state dynamically (conditional hooks - bad practice), we might need to grow.
			// But React rules say no conditional hooks.
			// Let's keep growth logic but optimized.
			needed := pIdx + 1
			if needed > cap(hooks.states) {
				newStates := make([]interface{}, needed, needed*2)
				copy(newStates, hooks.states)
				hooks.states = newStates
			} else if needed > len(hooks.states) {
				hooks.states = hooks.states[:needed]
			}
		}

		// Get PENDING value (most recent update, may not be committed yet)
		var currentValue T
		// We can trust the type if the slot was initialized with T
		if cv, ok := hooks.states[pIdx].(T); ok {
			currentValue = cv
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
		areEqual := fastEqual(currentValue, newValue)
		if !areEqual {
			// Update pending state immediately (visible to next setState in same render cycle)
			hooks.states[pIdx] = newValue
			// Also update committed state (this is what triggers re-render)
			hooks.states[sIdx] = newValue
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
		fiber.hooks = &Hooks{}
	}

	hooks := fiber.hooks
	hooks.index++

	depIdx := hooks.depIndex
	hooks.depIndex++

	cleanupIdx := hooks.cleanupIndex
	hooks.cleanupIndex++

	// Grow deps and cleanups if needed
	if len(hooks.deps) <= depIdx {
		needed := depIdx + 1
		if needed <= cap(hooks.deps) {
			hooks.deps = hooks.deps[:needed]
		} else {
			newDeps := make([][]interface{}, needed, needed*2)
			copy(newDeps, hooks.deps)
			hooks.deps = newDeps
		}
	}
	if len(hooks.cleanups) <= cleanupIdx {
		needed := cleanupIdx + 1
		if needed <= cap(hooks.cleanups) {
			hooks.cleanups = hooks.cleanups[:needed]
		} else {
			newCleanups := make([]func(), needed, needed*2)
			copy(newCleanups, hooks.cleanups)
			hooks.cleanups = newCleanups
		}
	}

	shouldRun := false
	if hooks.deps[depIdx] == nil {
		// First run
		hooks.deps[depIdx] = deps
		shouldRun = true
	} else {
		// Check if deps changed
		prevDeps := hooks.deps[depIdx]
		if len(deps) == 0 || !areDepsEqual(prevDeps, deps) {
			hooks.deps[depIdx] = deps
			shouldRun = true
		}
	}

	if shouldRun {
		// Run cleanup from previous effect if it exists
		if hooks.cleanups[cleanupIdx] != nil {
			hooks.cleanups[cleanupIdx]()
			hooks.cleanups[cleanupIdx] = nil
		}

		// Queue the new effect
		if fiber.effects == nil {
			fiber.effects = make([]Effect, 0)
		}

		// Capture position for cleanup storage
		fiber.effects = append(fiber.effects, Effect{
			Fn:           effect,
			CleanupIndex: cleanupIdx,
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
		fiber.hooks = &Hooks{}
	}

	hooks := fiber.hooks
	hooks.index++

	memoIdx := hooks.memoIndex
	hooks.memoIndex++

	// Grow memos if needed
	if len(hooks.memos) <= memoIdx {
		needed := memoIdx + 1
		if needed <= cap(hooks.memos) {
			hooks.memos = hooks.memos[:needed]
		} else {
			newMemos := make([]memoizedValue, needed, needed*2)
			copy(newMemos, hooks.memos)
			hooks.memos = newMemos
		}
	}

	memo := &hooks.memos[memoIdx]

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
		fiber.hooks = &Hooks{}
	}

	hooks := fiber.hooks
	hooks.index++

	callbackIdx := hooks.callbackIndex
	hooks.callbackIndex++

	// Grow callbacks if needed
	if len(hooks.callbacks) <= callbackIdx {
		needed := callbackIdx + 1
		if needed <= cap(hooks.callbacks) {
			hooks.callbacks = hooks.callbacks[:needed]
		} else {
			newCallbacks := make([]callbackValue, needed, needed*2)
			copy(newCallbacks, hooks.callbacks)
			hooks.callbacks = newCallbacks
		}
	}

	callback := &hooks.callbacks[callbackIdx]

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
		fiber.hooks = &Hooks{}
	}

	hooks := fiber.hooks
	hooks.index++

	refIdx := hooks.refIndex
	hooks.refIndex++

	// Grow refs if needed
	if len(hooks.refs) <= refIdx {
		needed := refIdx + 1
		if needed <= cap(hooks.refs) {
			hooks.refs = hooks.refs[:needed]
		} else {
			newRefs := make([]*RefValue, needed, needed*2)
			copy(newRefs, hooks.refs)
			hooks.refs = newRefs
		}
	}

	// If ref is nil (first render), create it with initial value
	if hooks.refs[refIdx] == nil {
		hooks.refs[refIdx] = &RefValue{Current: initialValue}
	}

	return hooks.refs[refIdx]
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
		fiber.hooks = &Hooks{}
	}

	hooks := fiber.hooks
	position := hooks.index
	hooks.index++

	idIdx := hooks.idIndex
	hooks.idIndex++

	// Initialize IDs array if needed
	if len(hooks.ids) <= idIdx {
		needed := idIdx + 1
		if needed <= cap(hooks.ids) {
			hooks.ids = hooks.ids[:needed]
		} else {
			newIds := make([]string, needed, needed*2)
			copy(newIds, hooks.ids)
			hooks.ids = newIds
		}

		// Generate a new unique ID
		// On first render, create a new ID using fiber's component ID counter
		rt := GetGlobalRuntime()
		rt.idCounterMu.Lock()
		rt.idCounter++
		id := rt.idCounter
		rt.idCounterMu.Unlock()

		// Create ID in format: "gwc:<global-id>:<hook-position>"
		// This ensures uniqueness and stability across renders
		hooks.ids[idIdx] = fmt.Sprintf("gwc:%d:%d", id, position)
	}

	// Return the persistent ID
	if idIdx < len(hooks.ids) {
		return hooks.ids[idIdx]
	}

	return ""
}

// GoUseFunc validates and stores a function for event handling
// The actual wrapping to js.Value happens in the WASM shim layer
func GoUseFunc(fn interface{}) interface{} {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseFunc called outside component context")
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{}
	}

	hooks := fiber.hooks
	hooks.index++

	funcIdx := hooks.funcIndex
	hooks.funcIndex++

	// Validate that the input is a function
	fnType := reflect.TypeOf(fn)
	if fnType == nil || fnType.Kind() != reflect.Func {
		panic("GoUseFunc requires a function")
	}

	// Check if we have a stored handler and if it matches
	var wrapper interface{}
	shouldCreate := true

	if funcIdx < len(hooks.funcs) {
		stored := hooks.funcs[funcIdx]
		// Compare function pointers to see if the function reference is stable
		// This works well with UseCallback which returns stable function references
		if stored.fn != nil && reflect.ValueOf(fn).Pointer() == reflect.ValueOf(stored.fn).Pointer() {
			if stored.wrapper != nil {
				wrapper = stored.wrapper
				shouldCreate = false
			}
		}
	}

	if shouldCreate {
		// Create new wrapper
		rt := GetGlobalRuntime()
		if rt == nil {
			panic("GoUseFunc rt is nil")
		}
		if rt.domAdapter == nil {
			panic("GoUseFunc domAdapter is nil")
		}

		wrapper = rt.domAdapter.WrapFunction(fn)

		// Store it
		handlerVal := funcHandlerValue{
			fn:      fn,
			wrapper: wrapper,
		}

		if len(hooks.funcs) <= funcIdx {
			needed := funcIdx + 1
			if needed <= cap(hooks.funcs) {
				hooks.funcs = hooks.funcs[:needed]
			} else {
				newFuncs := make([]funcHandlerValue, needed, needed*2)
				copy(newFuncs, hooks.funcs)
				hooks.funcs = newFuncs
			}
			hooks.funcs[funcIdx] = handlerVal
		} else {
			// Release old wrapper if it exists to prevent memory leaks
			oldHandler := hooks.funcs[funcIdx]
			if oldHandler.wrapper != nil {
				if releasable, ok := oldHandler.wrapper.(interface{ Release() }); ok {
					releasable.Release()
				}
			}
			hooks.funcs[funcIdx] = handlerVal
		}
	}

	return wrapper
}



// areDepsEqual compares dependency arrays
func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	if len(prevDeps) != len(newDeps) {
		return false
	}

	if len(prevDeps) == 0 {
		return true
	}

	// Unroll for common sizes
	if len(prevDeps) == 1 {
		return fastEqual(prevDeps[0], newDeps[0])
	}
	if len(prevDeps) == 2 {
		return fastEqual(prevDeps[0], newDeps[0]) && fastEqual(prevDeps[1], newDeps[1])
	}
	if len(prevDeps) == 3 {
		return fastEqual(prevDeps[0], newDeps[0]) && fastEqual(prevDeps[1], newDeps[1]) && fastEqual(prevDeps[2], newDeps[2])
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

	// Fast path for common types
	switch va := a.(type) {
	case int:
		if vb, ok := b.(int); ok {
			return va == vb
		}
	case string:
		if vb, ok := b.(string); ok {
			return va == vb
		}
	case bool:
		if vb, ok := b.(bool); ok {
			return va == vb
		}
	case float64:
		if vb, ok := b.(float64); ok {
			return va == vb
		}
	case int64:
		if vb, ok := b.(int64); ok {
			return va == vb
		}
	case float32:
		if vb, ok := b.(float32); ok {
			return va == vb
		}
	case int32:
		if vb, ok := b.(int32); ok {
			return va == vb
		}
	case uint:
		if vb, ok := b.(uint); ok {
			return va == vb
		}
	case uint64:
		if vb, ok := b.(uint64); ok {
			return va == vb
		}
	case uint32:
		if vb, ok := b.(uint32); ok {
			return va == vb
		}
	case uint8:
		if vb, ok := b.(uint8); ok {
			return va == vb
		}
	}

	// Try direct comparison first, but only for comparable types
	ta := reflect.TypeOf(a)
	tb := reflect.TypeOf(b)
	
	if ta != tb {
		return false
	}

	if ta.Comparable() {
		if a == b {
			return true
		}
	}

	// Handle functions (compare pointers)
	if ta.Kind() == reflect.Func {
		return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
	}

	// Optimization: Use reference equality for Slices and Maps
	// This aligns with React's behavior and avoids expensive DeepEqual
	if ta.Kind() == reflect.Slice || ta.Kind() == reflect.Map {
		return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
	}

	// Use reflect for complex types (structs with non-comparable fields)
	return reflect.DeepEqual(a, b)
}
