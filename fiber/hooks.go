//go:build js && wasm
// +build js,wasm

package fiber

import (
	"fmt"
	"runtime"
	"sync"
	"sync/atomic"
	"time"
)

// GoUseState manages state in a component with optimized equality checking
// This is the Go-branded version of useState for GoWebComponents
func GoUseState[T any](initialValue T) (func() T, func(T)) {
	debugf("HOOKS", "🎯 GoUseState called with type %T, value: %+v\n", initialValue, initialValue)

	currentFiber := getCurrentFiber()
	if currentFiber == nil {
		debugf("HOOKS", "🚨 GoUseState: currentFiber is nil!\n")
		panic("GoUseState called outside component context")
	}

	debugf("HOOKS", "📊 GoUseState: fiber %p, hooks %p\n", currentFiber, currentFiber.hooks)

	if currentFiber.hooks == nil {
		debugf("HOOKS", "🔧 GoUseState: initializing hooks container\n")
		currentFiber.hooks = getHooksFromPool()
		debugf("HOOKS", "✅ GoUseState: hooks container initialized %p\n", currentFiber.hooks)
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	debugf("HOOKS", "📍 GoUseState: hook position %d, total hooks so far: %d\n", position, currentFiber.hooks.index)

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeState, position); err != nil {
		debugf("HOOKS", "🚨 GoUseState: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Memory diagnostics - optimized with sampling to reduce overhead
	// Only collect memory stats occasionally to avoid performance impact
	if shouldCollectMemStats() {
		var memStats runtime.MemStats
		runtime.ReadMemStats(&memStats)
		debugf("HOOKS", "💾 GoUseState: heap objects: %d, allocs: %d, total alloc: %d KB\n",
			memStats.HeapObjects, memStats.Mallocs, memStats.TotalAlloc/1024)
	}

	if len(currentFiber.hooks.state) > position {
		// Existing state
		existingValue := currentFiber.hooks.state[position]
		debugf("HOOKS", "🔄 GoUseState: reusing existing state at position %d, value: %+v\n", position, existingValue)
	} else {
		// Initial state - grow slice efficiently with smart capacity estimation
		debugf("HOOKS", "🆕 GoUseState: creating initial state at position %d\n", position)
		requiredCap := position + 1
		if cap(currentFiber.hooks.state) < requiredCap {
			// Smart capacity estimation: ensure we have room for this position plus some growth
			oldCap := cap(currentFiber.hooks.state)
			newCap := max(8, max(requiredCap, oldCap*2))
			// For components with many hooks, be more aggressive with initial allocation
			if position >= 4 {
				newCap = max(newCap, position*2) // Anticipate more hooks
			}
			debugf("HOOKS", "📈 GoUseState: growing state slice from cap %d to %d (position %d)\n", oldCap, newCap, position)
			newState := make([]interface{}, len(currentFiber.hooks.state), newCap)
			copy(newState, currentFiber.hooks.state)
			currentFiber.hooks.state = newState
		}
		currentFiber.hooks.state = append(currentFiber.hooks.state, initialValue)
		debugf("HOOKS", "✅ GoUseState: initial state set, slice len: %d, cap: %d\n",
			len(currentFiber.hooks.state), cap(currentFiber.hooks.state))
	}

	// Capture hooks and position
	hooks := currentFiber.hooks
	idx := position

	getter := func() T {
		debugf("HOOKS", "🔍 GoUseState getter called for position %d\n", idx)
		// Optimized bounds checking using slice pattern for better performance
		// This approach is faster than manual bounds checking
		if idx < len(hooks.state) {
			// Safe slice access - compiler optimizes this pattern
			if value, ok := hooks.state[idx : idx+1][0].(T); ok {
				debugf("HOOKS", "✅ GoUseState getter: returning value %+v from position %d\n", value, idx)
				return value
			}
			// Return zero value if type assertion fails
			debugf("HOOKS", "🚨 GoUseState getter: type assertion failed for position %d, expected %T, got %T\n",
				idx, *new(T), hooks.state[idx])
		} else {
			debugf("HOOKS", "🚨 GoUseState getter: index %d out of bounds (len: %d)\n", idx, len(hooks.state))
		}
		var zero T
		return zero
	}

	setter := func(newValue T) {
		debugf("HOOKS", "✏️ GoUseState setter called for position %d with value: %+v\n", idx, newValue)
		startTime := time.Now()

		apply := func() {
			// Optimized slice expansion with safe bounds checking
			if idx >= len(hooks.state) {
				debugf("HOOKS", "📈 GoUseState setter: expanding state slice to fit index %d\n", idx)
				// Optimize: grow slice to required size in one operation
				requiredLen := idx + 1
				if cap(hooks.state) < requiredLen {
					// Pre-allocate with reasonable capacity
					newCap := max(8, max(requiredLen, cap(hooks.state)*2))
					newState := make([]interface{}, len(hooks.state), newCap)
					copy(newState, hooks.state)
					hooks.state = newState
				}
				// Extend to required length using optimized slice growth
				// This is more efficient than a loop
				if requiredLen > len(hooks.state) {
					// Extend slice in one operation
					extension := make([]interface{}, requiredLen-len(hooks.state))
					hooks.state = append(hooks.state, extension...)
				}
			}

			// Safe slice access using optimized pattern
			oldValue := hooks.state[idx : idx+1][0]
			debugf("HOOKS", "🔄 GoUseState setter: comparing old (%+v) vs new (%+v)\n", oldValue, newValue)

			if oldValue == nil || !fastEqual(oldValue, newValue) {
				debugf("HOOKS", "💾 GoUseState setter: state changed, updating and scheduling re-render\n")
				// Safe slice assignment using optimized pattern
				hooks.state[idx : idx+1][0] = newValue
				scheduleUpdateAtRoot()
				debugf("HOOKS", "⚡ GoUseState setter: update scheduled, took %v\n", time.Since(startTime))
			} else {
				debugf("HOOKS", "⏭️ GoUseState setter: no change detected, skipping update\n")
			}
		}

		if atomic.LoadInt32(&schedulerActive) == 1 {
			debugf("HOOKS", "🎯 GoUseState setter: applying immediately (scheduler active)\n")
			apply()
		} else {
			debugf("HOOKS", "📋 GoUseState setter: enqueueing for UI thread\n")
			enqueueUI(apply)
		}
	}

	debugf("HOOKS", "✅ GoUseState: setup complete for position %d\n", position)
	return getter, setter
}

// GoUseEffect runs side effects in a component with dependency tracking
// This is the Go-branded version of useEffect for GoWebComponents
func GoUseEffect(effect func(), deps ...interface{}) {
	debugf("HOOKS", "🎯 GoUseEffect called with %d dependencies: %+v\n", len(deps), deps)

	currentFiber := getCurrentFiber()
	if currentFiber == nil {
		debugf("HOOKS", "🚨 GoUseEffect: currentFiber is nil!\n")
		panic("GoUseEffect called outside component context")
	}

	if currentFiber.hooks == nil {
		debugf("HOOKS", "🔧 GoUseEffect: initializing hooks container\n")
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	debugf("HOOKS", "📍 GoUseEffect: hook position %d\n", position)

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeEffect, position); err != nil {
		debugf("HOOKS", "🚨 GoUseEffect: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Optimized deps slice growth - avoid multiple append operations
	if len(currentFiber.hooks.deps) <= position {
		// Calculate required capacity and grow in one operation
		requiredLen := position + 1
		if cap(currentFiber.hooks.deps) < requiredLen {
			// Pre-allocate with reasonable capacity to avoid future reallocations
			newCap := max(8, max(requiredLen, cap(currentFiber.hooks.deps)*2))
			newDeps := make([][]interface{}, len(currentFiber.hooks.deps), newCap)
			copy(newDeps, currentFiber.hooks.deps)
			currentFiber.hooks.deps = newDeps
		}
		// Optimized slice extension - single append operation
		if requiredLen > len(currentFiber.hooks.deps) {
			extension := make([][]interface{}, requiredLen-len(currentFiber.hooks.deps))
			currentFiber.hooks.deps = append(currentFiber.hooks.deps, extension...)
		}
	}

	if currentFiber.hooks.deps[position] == nil {
		// First time this effect is used
		debugf("HOOKS", "🆕 GoUseEffect: first run, adding effect\n")
		currentFiber.hooks.deps[position] = deps
		// Grow effects slice efficiently
		if cap(currentFiber.effects) <= len(currentFiber.effects) {
			oldCap := cap(currentFiber.effects)
			newCap := max(4, cap(currentFiber.effects)*2)
			debugf("HOOKS", "📈 GoUseEffect: growing effects slice from cap %d to %d\n", oldCap, newCap)
			newEffects := make([]func(), len(currentFiber.effects), newCap)
			copy(newEffects, currentFiber.effects)
			currentFiber.effects = newEffects
		}
		currentFiber.effects = append(currentFiber.effects, effect)
		debugf("HOOKS", "✅ GoUseEffect: effect queued, total effects: %d\n", len(currentFiber.effects))
	} else {
		prevDeps := currentFiber.hooks.deps[position]
		debugf("HOOKS", "🔄 GoUseEffect: comparing deps - prev: %+v, current: %+v\n", prevDeps, deps)

		shouldRun := len(deps) == 0 || !areDepsEqual(prevDeps, deps)
		debugf("HOOKS", "🤔 GoUseEffect: shouldRun = %v (no deps: %v, deps changed: %v)\n",
			shouldRun, len(deps) == 0, !areDepsEqual(prevDeps, deps))

		if shouldRun {
			// Dependencies have changed or no dependencies provided
			debugf("HOOKS", "🔄 GoUseEffect: dependencies changed, re-running effect\n")
			currentFiber.hooks.deps[position] = deps
			currentFiber.effects = append(currentFiber.effects, effect)
			debugf("HOOKS", "✅ GoUseEffect: effect re-queued, total effects: %d\n", len(currentFiber.effects))
		} else {
			debugf("HOOKS", "⏭️ GoUseEffect: dependencies unchanged, skipping effect\n")
		}
	}
}

// GoUseMemo memoizes expensive computations with dependency tracking
// This is the Go-branded version of useMemo for GoWebComponents
func GoUseMemo(compute func() interface{}, deps ...interface{}) interface{} {
	debugf("HOOKS", "🎯 GoUseMemo called with %d dependencies: %+v\n", len(deps), deps)
	startTime := time.Now()

	currentFiber := getCurrentFiber()
	if currentFiber == nil {
		debugf("HOOKS", "🚨 GoUseMemo: currentFiber is nil!\n")
		panic("GoUseMemo called outside component context")
	}

	if currentFiber.hooks == nil {
		debugf("HOOKS", "🔧 GoUseMemo: initializing hooks container\n")
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	debugf("HOOKS", "📍 GoUseMemo: hook position %d\n", position)

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeMemo, position); err != nil {
		debugf("HOOKS", "🚨 GoUseMemo: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	// Optimized memos slice growth - avoid multiple append operations
	if len(currentFiber.hooks.memos) <= position {
		// Calculate required capacity and grow in one operation
		requiredLen := position + 1
		if cap(currentFiber.hooks.memos) < requiredLen {
			// Pre-allocate with reasonable capacity to avoid future reallocations
			newCap := max(4, max(requiredLen, cap(currentFiber.hooks.memos)*2))
			newMemos := make([]memoizedValue, len(currentFiber.hooks.memos), newCap)
			copy(newMemos, currentFiber.hooks.memos)
			currentFiber.hooks.memos = newMemos
		}
		// Optimized slice extension - single append operation
		if requiredLen > len(currentFiber.hooks.memos) {
			extension := make([]memoizedValue, requiredLen-len(currentFiber.hooks.memos))
			currentFiber.hooks.memos = append(currentFiber.hooks.memos, extension...)
		}
	}

	memo := &currentFiber.hooks.memos[position]

	if memo.value == nil || !areDepsEqual(memo.deps, deps) {
		// Need to recompute
		debugf("HOOKS", "🔄 GoUseMemo: recomputing memo (first time or deps changed)\n")
		memo.value = compute()
		memo.deps = deps
		debugf("HOOKS", "⚡ GoUseMemo: computation completed in %v\n", time.Since(startTime))
	} else {
		debugf("HOOKS", "⏭️ GoUseMemo: returning cached value (deps unchanged)\n")
	}

	return memo.value
}

// Global atom storage and subscription tracking
var atomRegistry = make(map[string]interface{})
var atomSubscriptions = make(map[string]map[*Fiber]bool)
var atomMutex = &sync.RWMutex{}

// GoUseAtom provides SolidJS-style fine-grained reactivity with global atoms
// Atoms are accessible from anywhere in the component tree by ID
// This is the Go-branded version of atom-based state management for GoWebComponents
func GoUseAtom[T any](id string, initialValue T) (func() T, func(T)) {
	debugf("HOOKS", "🎯 GoUseAtom called with id '%s', type %T, value: %+v\n", id, initialValue, initialValue)

	currentFiber := getCurrentFiber()
	if currentFiber == nil {
		debugf("HOOKS", "🚨 GoUseAtom: currentFiber is nil!\n")
		panic("GoUseAtom called outside component context")
	}

	debugf("HOOKS", "📊 GoUseAtom: fiber %p for atom '%s'\n", currentFiber, id)

	// Initialize atom if it doesn't exist (first one wins)
	atomMutex.Lock()
	if _, exists := atomRegistry[id]; !exists {
		atomRegistry[id] = initialValue
		atomSubscriptions[id] = make(map[*Fiber]bool)
		debugf("HOOKS", "🆕 GoUseAtom: initialized new atom '%s' with value: %+v\n", id, initialValue)
	}

	// Subscribe this fiber to the atom
	atomSubscriptions[id][currentFiber] = true
	debugf("HOOKS", "📝 GoUseAtom: subscribed fiber %p to atom '%s' (total subscribers: %d)\n",
		currentFiber, id, len(atomSubscriptions[id]))
	atomMutex.Unlock()

	// This hook position is needed for cleanup when component unmounts
	if currentFiber.hooks == nil {
		debugf("HOOKS", "🔧 GoUseAtom: initializing hooks container\n")
		currentFiber.hooks = getHooksFromPool()
	}

	position := currentFiber.hooks.index
	currentFiber.hooks.index++

	// Validate hook order
	if err := validateHookOrder(currentFiber.hooks, HookTypeAtom, position); err != nil {
		debugf("HOOKS", "🚨 GoUseAtom: %v\n", err)
		// Continue execution but log the error - don't panic in production
	}

	getter := func() T {
		debugf("HOOKS", "🔍 GoUseAtom getter called for atom '%s'\n", id)
		atomMutex.RLock()
		value, exists := atomRegistry[id]
		atomMutex.RUnlock()

		if !exists {
			debugf("HOOKS", "🚨 GoUseAtom getter: atom '%s' not found\n", id)
			var zero T
			return zero
		}

		if typedValue, ok := value.(T); ok {
			debugf("HOOKS", "✅ GoUseAtom getter: returning value %+v for atom '%s'\n", typedValue, id)
			return typedValue
		}

		// Type assertion failed - runtime panic with clear message
		debugf("HOOKS", "🚨 GoUseAtom getter: type assertion failed for atom '%s', expected %T, got %T\n",
			id, *new(T), value)
		panic(fmt.Sprintf("GoUseAtom: atom '%s' type mismatch - expected '%T', got '%T'", id, *new(T), value))
	}

	setter := func(newValue T) {
		debugf("HOOKS", "✏️ GoUseAtom setter called for atom '%s' with value: %+v\n", id, newValue)
		startTime := time.Now()

		apply := func() {
			atomMutex.Lock()
			oldValue := atomRegistry[id]

			// Only update if value actually changed
			if oldValue == nil || !fastEqual(oldValue, newValue) {
				atomRegistry[id] = newValue
				subscribers := make([]*Fiber, 0, len(atomSubscriptions[id]))
				for fiber := range atomSubscriptions[id] {
					subscribers = append(subscribers, fiber)
				}
				atomMutex.Unlock()

				debugf("HOOKS", "💾 GoUseAtom setter: atom '%s' updated, notifying %d subscribers\n",
					id, len(subscribers))

				// Schedule re-render for all subscribed components
				scheduleUpdateAtRoot()

				debugf("HOOKS", "⚡ GoUseAtom setter: atom '%s' update completed in %v\n", id, time.Since(startTime))
			} else {
				atomMutex.Unlock()
				debugf("HOOKS", "⏭️ GoUseAtom setter: atom '%s' value unchanged, skipping update\n", id)
			}
		}

		if atomic.LoadInt32(&schedulerActive) == 1 {
			debugf("HOOKS", "🎯 GoUseAtom setter: applying immediately (scheduler active)\n")
			apply()
		} else {
			debugf("HOOKS", "📋 GoUseAtom setter: enqueueing for UI thread\n")
			enqueueUI(apply)
		}
	}

	debugf("HOOKS", "✅ GoUseAtom: setup complete for atom '%s'\n", id)
	return getter, setter
}

// validateHookOrder checks that hooks are called in the same order as previous render
func validateHookOrder(hooks *Hooks, hookType HookType, position int) error {
	// Debug: Log that validation is being called with detailed info
	debugf("HOOKS", "🔍 validateHookOrder called: type=%d, position=%d, prevOrder=%d, callOrder=%d\n",
		hookType, position, len(hooks.prevOrder), len(hooks.callOrder))

	// Record this hook call
	hookCall := HookCall{Type: hookType, Position: position}
	hooks.callOrder = append(hooks.callOrder, hookCall)

	debugf("HOOKS", "📝 validateHookOrder: recorded hook call %+v, total calls: %d\n",
		hookCall, len(hooks.callOrder))

	// Skip validation on first render (no previous order to compare)
	if len(hooks.prevOrder) == 0 {
		debugf("HOOKS", "🔍 Skipping validation - first render (no prevOrder)\n")
		return nil
	}

	// Debug: Show what we're comparing
	debugf("HOOKS", "🔍 Validating: current hook type=%d pos=%d vs previous hook type=%d pos=%d\n",
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

	// Optimized bounds checking for hook order validation
	if position < len(hooks.prevOrder) {
		// Safe slice access using optimized pattern
		prevCall := hooks.prevOrder[position : position+1][0]

		if prevCall.Type != hookType {
			err := fmt.Errorf("hook order violation: hook type mismatch at position %d (expected %d, got %d)",
				position, prevCall.Type, hookType)
			debugf("HOOKS", "🚨 validateHookOrder: %v\n", err)
			return err
		}

		if prevCall.Position != position {
			err := fmt.Errorf("hook order violation: position mismatch at index %d (expected %d, got %d)",
				position, prevCall.Position, position)
			debugf("HOOKS", "🚨 validateHookOrder: %v\n", err)
			return err
		}

		debugf("HOOKS", "✅ Hook order validation passed for position %d\n", position)
		return nil
	} else {
		// Handle out of bounds case
		err := fmt.Errorf("hook order violation: more hooks called than previous render (position %d >= prevOrder length %d)", position, len(hooks.prevOrder))
		debugf("HOOKS", "🚨 validateHookOrder: %v\n", err)
		return err
	}
}

// finalizeHookOrder completes hook order validation after all hooks have been called
func finalizeHookOrder(hooks *Hooks) error {
	// Debug: Log finalization
	debugf("HOOKS", "🔍 finalizeHookOrder called: callOrder=%d, prevOrder=%d\n",
		len(hooks.callOrder), len(hooks.prevOrder))

	// Check if fewer hooks were called than previous render
	if len(hooks.prevOrder) > 0 && len(hooks.callOrder) < len(hooks.prevOrder) {
		err := fmt.Errorf("hook order violation: fewer hooks called than previous render (%d vs %d)",
			len(hooks.callOrder), len(hooks.prevOrder))
		debugf("HOOKS", "🚨 finalizeHookOrder: %v\n", err)
		return err
	}

	// Move current order to previous order for next render
	hooks.prevOrder = make([]HookCall, len(hooks.callOrder))
	copy(hooks.prevOrder, hooks.callOrder)

	// Debug: Log the transfer with details
	debugf("HOOKS", "🔄 finalizeHookOrder: transferred %d hook calls to prevOrder for next render\n", len(hooks.prevOrder))
	for i, call := range hooks.prevOrder {
		debugf("HOOKS", "  📋 Hook %d: type=%d, position=%d\n", i, call.Type, call.Position)
	}

	// Cap prevOrder slice to prevent unbounded growth
	const maxPrevOrder = 256
	if len(hooks.prevOrder) > maxPrevOrder {
		debugf("HOOKS", "✂️ finalizeHookOrder: capping prevOrder from %d to %d\n", len(hooks.prevOrder), maxPrevOrder)
		hooks.prevOrder = hooks.prevOrder[:maxPrevOrder]
	}

	// Reset current order for next render
	hooks.callOrder = hooks.callOrder[:0]
	hooks.orderChecked = true

	debugf("HOOKS", "✅ finalizeHookOrder: validation complete, ready for next render\n")
	return nil
}

// areDepsEqual performs optimized dependency comparison
func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	debugf("HOOKS", "🔄 areDepsEqual: comparing %d prev deps vs %d new deps\n", len(prevDeps), len(newDeps))

	if len(prevDeps) != len(newDeps) {
		debugf("HOOKS", "❌ areDepsEqual: length mismatch (%d vs %d)\n", len(prevDeps), len(newDeps))
		return false
	}

	// Fast path for empty deps
	if len(prevDeps) == 0 {
		debugf("HOOKS", "✅ areDepsEqual: both empty, equal\n")
		return true
	}

	// Use fast equality check
	for i := range prevDeps {
		equal := fastEqual(prevDeps[i], newDeps[i])
		debugf("HOOKS", "🔍 areDepsEqual: dep %d - prev: %+v, new: %+v, equal: %v\n",
			i, prevDeps[i], newDeps[i], equal)
		if !equal {
			debugf("HOOKS", "❌ areDepsEqual: dependency %d changed\n", i)
			return false
		}
	}

	debugf("HOOKS", "✅ areDepsEqual: all dependencies equal\n")
	return true
}

// max returns the larger of two integers
func max(a, b int) int {
	if a > b {
		return a
	}
	return b
}
