//go:build js && wasm
// +build js,wasm

package fiber

import (
	"fmt"
	"sync/atomic"
)

// GoUseState manages state in a component with optimized equality checking
// This is the Go-branded version of useState for GoWebComponents
func GoUseState[T any](initialValue T) (func() T, func(T)) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeState, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] GoUseState: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	if len(currentFiber.hooks.state) > position {
		// Existing state
	} else {
		// Initial state - grow slice efficiently
		if cap(currentFiber.hooks.state) <= position {
			// Double capacity when needed
			newCap := max(8, len(currentFiber.hooks.state)*2)
			newState := make([]interface{}, len(currentFiber.hooks.state), newCap)
			copy(newState, currentFiber.hooks.state)
			currentFiber.hooks.state = newState
		}
		currentFiber.hooks.state = append(currentFiber.hooks.state, initialValue)
	}

	// Capture hooks and position
	hooks := currentFiber.hooks
	idx := position

	getter := func() T {
		// Bounds check to prevent index out of range panic
		if idx >= len(hooks.state) {
			var zero T
			return zero
		}
		if value, ok := hooks.state[idx].(T); ok {
			return value
		}
		// Return zero value if type assertion fails
		var zero T
		return zero
	}

	setter := func(newValue T) {
		apply := func() {
			if idx >= len(hooks.state) {
				for len(hooks.state) <= idx {
					hooks.state = append(hooks.state, nil)
				}
			}
			if hooks.state[idx] == nil || !fastEqual(hooks.state[idx], newValue) {
				hooks.state[idx] = newValue
				scheduleUpdateAtRoot()
			}
		}
		if atomic.LoadInt32(&schedulerActive) == 1 {
			apply()
		} else {
			enqueueUI(apply)
		}
	}

	return getter, setter
}

// useEffect runs side effects in a component with dependency tracking
func useEffect(effect func(), deps ...interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeEffect, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] useEffect: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow deps slice efficiently
	for len(currentFiber.hooks.deps) <= position {
		currentFiber.hooks.deps = append(currentFiber.hooks.deps, nil)
	}

	if currentFiber.hooks.deps[position] == nil {
		// First time this effect is used
		currentFiber.hooks.deps[position] = deps
		// Grow effects slice efficiently
		if cap(currentFiber.effects) <= len(currentFiber.effects) {
			newCap := max(4, cap(currentFiber.effects)*2)
			newEffects := make([]func(), len(currentFiber.effects), newCap)
			copy(newEffects, currentFiber.effects)
			currentFiber.effects = newEffects
		}
		currentFiber.effects = append(currentFiber.effects, effect)
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		shouldRun := len(deps) == 0 || !areDepsEqual(prevDeps, deps)
		if shouldRun {
			// Dependencies have changed or no dependencies provided
			currentFiber.hooks.deps[position] = deps
			currentFiber.effects = append(currentFiber.effects, effect)
		}
	}
}

// GoUseEffect runs side effects in a component with dependency tracking
// This is the Go-branded version of useEffect for GoWebComponents
func GoUseEffect(effect func(), deps ...interface{}) {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeEffect, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] GoUseEffect: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow deps slice efficiently
	for len(currentFiber.hooks.deps) <= position {
		currentFiber.hooks.deps = append(currentFiber.hooks.deps, nil)
	}

	if currentFiber.hooks.deps[position] == nil {
		// First time this effect is used
		currentFiber.hooks.deps[position] = deps
		// Grow effects slice efficiently
		if cap(currentFiber.effects) <= len(currentFiber.effects) {
			newCap := max(4, cap(currentFiber.effects)*2)
			newEffects := make([]func(), len(currentFiber.effects), newCap)
			copy(newEffects, currentFiber.effects)
			currentFiber.effects = newEffects
		}
		currentFiber.effects = append(currentFiber.effects, effect)
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		shouldRun := len(deps) == 0 || !areDepsEqual(prevDeps, deps)
		if shouldRun {
			// Dependencies have changed or no dependencies provided
			currentFiber.hooks.deps[position] = deps
			currentFiber.effects = append(currentFiber.effects, effect)
		}
	}
}

// useMemo memoizes expensive computations with dependency tracking
func useMemo(compute func() interface{}, deps ...interface{}) interface{} {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeMemo, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] useMemo: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow memos slice efficiently
	for len(currentFiber.hooks.memos) <= position {
		currentFiber.hooks.memos = append(currentFiber.hooks.memos, memoizedValue{})
	}

	memo := &currentFiber.hooks.memos[position]

	if memo.value == nil {
		// First time this memo is used
		value := compute() // Remove goroutine overhead for simple computations
		memo.value = value
		memo.deps = deps
		return value
	}

	shouldCompute := len(deps) == 0 || !areDepsEqual(memo.deps, deps)
	if shouldCompute {
		value := compute() // Direct call, no goroutine
		memo.value = value
		memo.deps = deps
		return value
	}

	// Dependencies haven't changed, return the memoized value
	return memo.value
}

// GoUseMemo memoizes expensive computations with dependency tracking
// This is the Go-branded version of useMemo for GoWebComponents
func GoUseMemo(compute func() interface{}, deps ...interface{}) interface{} {
	currentFiber := getCurrentFiber()
	if currentFiber.hooks == nil {
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeMemo, position); err != nil {
		fmt.Printf("🚨 [HOOK_ORDER_ERROR] GoUseMemo: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Grow memos slice efficiently
	for len(currentFiber.hooks.memos) <= position {
		currentFiber.hooks.memos = append(currentFiber.hooks.memos, memoizedValue{})
	}

	memo := &currentFiber.hooks.memos[position]

	if memo.value == nil {
		// First time this memo is used
		value := compute() // Remove goroutine overhead for simple computations
		memo.value = value
		memo.deps = deps
		return value
	}

	shouldCompute := len(deps) == 0 || !areDepsEqual(memo.deps, deps)
	if shouldCompute {
		value := compute() // Direct call, no goroutine
		memo.value = value
		memo.deps = deps
		return value
	}

	// Dependencies haven't changed, return the memoized value
	return memo.value
}

// validateHookOrder checks that hooks are called in the same order as previous render
func validateHookOrder(hooks *Hooks, hookType HookType, position int) error {
	// Debug: Log that validation is being called with detailed info
	debugf("🔍 [HOOK_DEBUG] validateHookOrder called: type=%d, position=%d, prevOrder=%d, callOrder=%d\n",
		hookType, position, len(hooks.prevOrder), len(hooks.callOrder))

	// Record this hook call
	hookCall := HookCall{Type: hookType, Position: position}
	hooks.callOrder = append(hooks.callOrder, hookCall)

	// Skip validation on first render (no previous order to compare)
	if len(hooks.prevOrder) == 0 {
		debugf("🔍 [HOOK_DEBUG] Skipping validation - first render (no prevOrder)\n")
		return nil
	}

	// Debug: Show what we're comparing
	debugf("🔍 [HOOK_DEBUG] Validating: current hook type=%d pos=%d vs previous hook type=%d pos=%d\n",
		hookType, position,
		func() int {
			if position < len(hooks.prevOrder) {
				return int(hooks.prevOrder[position].Type)
			} else {
				return -1
			}
		}(),
		func() int {
			if position < len(hooks.prevOrder) {
				return hooks.prevOrder[position].Position
			} else {
				return -1
			}
		}())

	// Check if we have a corresponding hook call from previous render
	if position >= len(hooks.prevOrder) {
		return fmt.Errorf("hook order violation: more hooks called than previous render (position %d >= prevOrder length %d)", position, len(hooks.prevOrder))
	}

	prevCall := hooks.prevOrder[position]
	if prevCall.Type != hookType {
		return fmt.Errorf("hook order violation: hook type mismatch at position %d (expected %d, got %d)",
			position, prevCall.Type, hookType)
	}

	if prevCall.Position != position {
		return fmt.Errorf("hook order violation: position mismatch at index %d (expected %d, got %d)",
			position, prevCall.Position, position)
	}

	debugf("✅ [HOOK_DEBUG] Hook order validation passed for position %d\n", position)
	return nil
}

// finalizeHookOrder completes hook order validation after all hooks have been called
func finalizeHookOrder(hooks *Hooks) error {
	// Debug: Log finalization
	debugf("🔍 [HOOK_DEBUG] finalizeHookOrder called: callOrder=%d, prevOrder=%d\n",
		len(hooks.callOrder), len(hooks.prevOrder))

	// Check if fewer hooks were called than previous render
	if len(hooks.prevOrder) > 0 && len(hooks.callOrder) < len(hooks.prevOrder) {
		return fmt.Errorf("hook order violation: fewer hooks called than previous render (%d vs %d)",
			len(hooks.callOrder), len(hooks.prevOrder))
	}

	// Move current order to previous order for next render
	hooks.prevOrder = make([]HookCall, len(hooks.callOrder))
	copy(hooks.prevOrder, hooks.callOrder)

	// Debug: Log the transfer
	debugf("🔍 [HOOK_DEBUG] Transferred %d hook calls to prevOrder for next render\n", len(hooks.prevOrder))

	// Cap prevOrder slice to prevent unbounded growth
	const maxPrevOrder = 256
	if len(hooks.prevOrder) > maxPrevOrder {
		hooks.prevOrder = hooks.prevOrder[:maxPrevOrder]
	}

	// Reset current order for next render
	hooks.callOrder = hooks.callOrder[:0]
	hooks.orderChecked = true

	return nil
}

// areDepsEqual performs optimized dependency comparison
func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	if len(prevDeps) != len(newDeps) {
		return false
	}

	// Fast path for empty deps
	if len(prevDeps) == 0 {
		return true
	}

	// Use fast equality check
	for i := range prevDeps {
		if !fastEqual(prevDeps[i], newDeps[i]) {
			return false
		}
	}
	return true
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
