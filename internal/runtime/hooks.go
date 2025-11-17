package runtime

import (
	"fmt"
	"reflect"
)

// GoUseState manages state in a component
func GoUseState[T any](rt *Runtime, initialValue T) (func() T, func(T)) {
	fiber := GetCurrentFiber()
	if fiber == nil {
		panic("GoUseState called outside component context")
	}
	
	if fiber.hooks == nil {
		fiber.hooks = &Hooks{
			state:     make([]interface{}, 0),
			deps:      make([][]interface{}, 0),
			memos:     make([]memoizedValue, 0),
			callOrder: make([]HookCall, 0),
			prevOrder: make([]HookCall, 0),
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
	
	setter := func(newValue T) {
		if idx >= len(hooks.state) {
			// Expand state
			newState := make([]interface{}, idx+1)
			copy(newState, hooks.state)
			hooks.state = newState
		}
		
		oldValue := hooks.state[idx]
		if oldValue == nil || !fastEqual(oldValue, newValue) {
			hooks.state[idx] = newValue
			rt.ScheduleUpdateForFiber(fiber)
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
			state:     make([]interface{}, 0),
			deps:      make([][]interface{}, 0),
			memos:     make([]memoizedValue, 0),
			callOrder: make([]HookCall, 0),
			prevOrder: make([]HookCall, 0),
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
			state:     make([]interface{}, 0),
			deps:      make([][]interface{}, 0),
			memos:     make([]memoizedValue, 0),
			callOrder: make([]HookCall, 0),
			prevOrder: make([]HookCall, 0),
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
