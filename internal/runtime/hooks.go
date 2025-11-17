package runtime

import (
	"fmt"
	"reflect"
)

// GoUseState manages state in a component
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
				fmt.Printf("[GETSTATE] Reading state[%d]=%v\n", idx, value)
				return value
			}
		}
		var zero T
		fmt.Printf("[GETSTATE] Reading state[%d]=zero (not found)\n", idx)
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
			fmt.Printf("[SETSTATE] Reading PENDING state[%d]=%v\n", idx, currentValue)
		}
	} else {
		fmt.Printf("[SETSTATE] No pending state at [%d], using zero\n", idx)
	}		// Determine the new value
		var newValue T
		// Try to treat as functional update (func(T) T)
		if fn, ok := newValueOrUpdater.(func(T) T); ok {
			newValue = fn(currentValue)
			fmt.Printf("[SETSTATE] Functional update state[%d]: %v -> %v\n", idx, currentValue, newValue)
		} else if directValue, ok := newValueOrUpdater.(T); ok {
			// Direct value
			newValue = directValue
		} else {
			fmt.Printf("[SETSTATE] Type mismatch, expected %T or func(%T) %T, got %T\n", currentValue, currentValue, currentValue, newValueOrUpdater)
			return
		}

		// Check if newValue is different from current PENDING value
		if !fastEqual(currentValue, newValue) {
			fmt.Printf("[SETSTATE] Updating state[%d]: %v -> %v\n", idx, currentValue, newValue)
			// Update pending state immediately (visible to next setState in same render cycle)
			hooks.pendingState[idx] = newValue
			// Also update committed state (this is what triggers re-render)
			hooks.state[idx] = newValue
			rt.ScheduleUpdateForFiber(fiber)
		} else {
			fmt.Printf("[SETSTATE] Skipping update, value unchanged: %v\n", newValue)
		}
	}

	return getter, setter
}

// GoUseEffect runs side effects
func GoUseEffect(effect func(), deps ...interface{}) {
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
			callOrder:    make([]HookCall, 0),
			prevOrder:    make([]HookCall, 0),
		}
	}

	position := fiber.hooks.index
	fiber.hooks.index++

	// Validate hook order
	validateHookOrder(fiber.hooks, HookTypeEffect, position)

	// Grow deps if needed
	if len(fiber.hooks.deps) <= position {
		newDeps := make([][]interface{}, position+1, (position+1)*2)
		copy(newDeps, fiber.hooks.deps)
		fiber.hooks.deps = newDeps
	}

	if fiber.hooks.deps[position] == nil {
		// First run
		fiber.hooks.deps[position] = deps
		if fiber.effects == nil {
			fiber.effects = make([]func(), 0)
		}
		fiber.effects = append(fiber.effects, effect)
	} else {
		// Check if deps changed
		prevDeps := fiber.hooks.deps[position]
		if len(deps) == 0 || !areDepsEqual(prevDeps, deps) {
			fiber.hooks.deps[position] = deps
			if fiber.effects == nil {
				fiber.effects = make([]func(), 0)
			}
			fiber.effects = append(fiber.effects, effect)
		}
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

	if memo.value == nil || !areDepsEqual(memo.deps, deps) {
		memo.value = compute()
		memo.deps = deps
	}

	return memo.value
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

	// Try direct comparison first
	if a == b {
		return true
	}

	// Use reflect for complex types
	return reflect.DeepEqual(a, b)
}
