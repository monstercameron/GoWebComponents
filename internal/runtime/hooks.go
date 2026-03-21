package runtime

import (
	"fmt"
	"reflect"
	"sync"
	"time"
	"unsafe"
)

type emptyInterfaceHeader struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

func sameFunctionIdentity(a, b interface{}) bool {
	if a == nil || b == nil {
		return a == b
	}
	va := reflect.ValueOf(a)
	vb := reflect.ValueOf(b)
	if !va.IsValid() || !vb.IsValid() || va.Kind() != reflect.Func || vb.Kind() != reflect.Func {
		return false
	}
	if va.Type() != vb.Type() {
		return false
	}
	aHeader := (*emptyInterfaceHeader)(unsafe.Pointer(&a))
	bHeader := (*emptyInterfaceHeader)(unsafe.Pointer(&b))
	return va.Pointer() == vb.Pointer() && aHeader.data == bHeader.data
}

var nilableTypeCache sync.Map

func isNilableType[T any]() bool {
	t := reflect.TypeOf((*T)(nil)).Elem()
	if cached, ok := nilableTypeCache.Load(t); ok {
		return cached.(bool)
	}

	var nilable bool
	switch t.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		nilable = true
	default:
		nilable = false
	}

	nilableTypeCache.Store(t, nilable)
	return nilable
}

// GoUseState provides state management for components
func GoUseState[T any](rt *Runtime, initialValue T) (func() T, func(interface{})) {
	fiber := GetCurrentFiber()
	if fiber == nil {
		ReportDiagnostic("runtime", DiagnosticError, "GoUseState called outside component context")
		panic(actionableHookUsagePanic("GoUseState"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	recordHookSignature(fiber.hooks, "state")
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
		restoredValue, ok := fiber.hooks.restoreStateValue(stateIdx)
		if !ok {
			restoredValue = initialValue
		} else if coerced, ok := coerceHotReloadValue(restoredValue, reflect.TypeOf(initialValue)); ok {
			restoredValue = coerced
		}
		fiber.hooks.states[stateIdx*2] = restoredValue
		fiber.hooks.states[stateIdx*2+1] = restoredValue
	}

	hooks := fiber.hooks
	// Capture indices for closure
	sIdx := stateIdx * 2
	pIdx := stateIdx*2 + 1
	nilableState := isNilableType[T]()

	getter := func() T {
		// Bounds check removed for performance - slice is grown before closure creation
		// Type assertion should always succeed if state was initialized correctly
		value, _ := hooks.states[sIdx].(T)
		return value
	}

	setter := func(newValueOrUpdater interface{}) {
		apply := func() {
			if pIdx >= len(hooks.states) {
				needed := pIdx + 1
				if needed > cap(hooks.states) {
					newStates := make([]interface{}, needed, needed*2)
					copy(newStates, hooks.states)
					hooks.states = newStates
				} else if needed > len(hooks.states) {
					hooks.states = hooks.states[:needed]
				}
			}

			var currentValue T
			if cv, ok := hooks.states[sIdx].(T); ok {
				currentValue = cv
			}

			newValue, ok := resolveStateUpdateValue(currentValue, newValueOrUpdater, nilableState)
			if !ok {
				return
			}

			if fastEqual(currentValue, newValue) {
				return
			}

			hooks.states[pIdx] = newValue
			hooks.states[sIdx] = newValue
			targetFiber := hooks.owner
			if targetFiber == nil {
				targetFiber = fiber
			}
			rt.ScheduleUpdateForFiber(targetFiber)
		}

		if rt != nil && rt.ShouldDeferStateUpdates() {
			rt.ScheduleTransition(apply)
			return
		}

		apply()
	}

	return getter, setter
}

// GoUseEffect runs side effects and supports cleanup
// The effect function can return a cleanup function that will be called before the next effect runs or on unmount
func GoUseEffect(effect func() func(), deps ...interface{}) {
	fiber := GetCurrentFiber()
	if fiber == nil {
		ReportDiagnostic("runtime", DiagnosticError, "GoUseEffect called outside component context")
		panic(actionableHookUsagePanic("GoUseEffect"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	hooks := fiber.hooks
	recordHookSignature(hooks, "effect")
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
	if len(hooks.effectEpochs) <= cleanupIdx {
		needed := cleanupIdx + 1
		if needed <= cap(hooks.effectEpochs) {
			hooks.effectEpochs = hooks.effectEpochs[:needed]
		} else {
			newEpochs := make([]int, needed, needed*2)
			copy(newEpochs, hooks.effectEpochs)
			hooks.effectEpochs = newEpochs
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
		if len(deps) == 0 || !areDepsEqual(prevDeps, deps) || hooks.effectEpochs[cleanupIdx] != hooks.effectEpoch {
			hooks.deps[depIdx] = deps
			shouldRun = true
		}
	}

	if shouldRun {
		// Run cleanup from previous effect if it exists
		if hooks.cleanups[cleanupIdx] != nil {
			start := time.Now()
			hooks.cleanups[cleanupIdx]()
			durationNs := time.Since(start).Nanoseconds()
			fiber.cleanupDurationNs += durationNs
			recordSlowOperationDiagnostic("cleanup", fiber, durationNs)
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
		hooks.effectEpochs[cleanupIdx] = hooks.effectEpoch
	}
}

// GoUseMemo memoizes expensive computations
func GoUseMemo(compute func() interface{}, deps ...interface{}) interface{} {
	return goUseMemo(compute, nil, deps...)
}

func goUseMemo(compute func() interface{}, targetType reflect.Type, deps ...interface{}) interface{} {
	fiber := GetCurrentFiber()
	if fiber == nil {
		ReportDiagnostic("runtime", DiagnosticError, "GoUseMemo called outside component context")
		panic(actionableHookUsagePanic("GoUseMemo"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	hooks := fiber.hooks
	recordHookSignature(hooks, "memo")
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

	if memo.deps == nil {
		if restoredValue, restoredDeps, ok := hooks.restoreMemoValue(memoIdx); ok {
			if coerced, ok := coerceHotReloadValue(restoredValue, targetType); ok {
				memo.value = coerced
				memo.deps = restoredDeps
			}
		}
	}

	// Check if we need to recompute: only if deps changed (or this is the first render and memo.deps is uninitialized)
	// First render: memo.deps will be nil, so we need to initialize it
	// Subsequent renders: check if deps changed
	if memo.deps == nil || !areDepsEqual(memo.deps, deps) {
		memo.value = compute()
		memo.deps = deps
	}

	return memo.value
}

// GoUseMemoTyped memoizes expensive computations and coerces restored hot reload
// values to the caller's expected type when possible.
func GoUseMemoTyped(compute func() interface{}, targetType reflect.Type, deps ...interface{}) interface{} {
	return goUseMemo(compute, targetType, deps...)
}

// GoUseCallback memoizes a callback function with dependency tracking
func GoUseCallback(fn interface{}, deps ...interface{}) interface{} {
	fiber := GetCurrentFiber()
	if fiber == nil {
		ReportDiagnostic("runtime", DiagnosticError, "GoUseCallback called outside component context")
		panic(actionableHookUsagePanic("GoUseCallback"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	hooks := fiber.hooks
	recordHookSignature(hooks, "callback")
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
		ReportDiagnostic("runtime", DiagnosticError, "GoUseRef called outside component context")
		panic(actionableHookUsagePanic("GoUseRef"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	hooks := fiber.hooks
	recordHookSignature(hooks, "ref")
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
		restoredValue, ok := hooks.restoreRefValue(refIdx)
		if !ok {
			restoredValue = initialValue
		}
		hooks.refs[refIdx] = &RefValue{Current: restoredValue}
	}

	return hooks.refs[refIdx]
}

// GoUseId generates a unique, stable identifier for accessibility attributes
// The ID is generated once and persists across renders without changing
// This is useful for associating labels with form inputs and other accessibility needs
func GoUseId() string {
	fiber := GetCurrentFiber()
	if fiber == nil {
		ReportDiagnostic("runtime", DiagnosticError, "GoUseId called outside component context")
		panic(actionableHookUsagePanic("GoUseId"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	hooks := fiber.hooks
	position := hooks.index
	recordHookSignature(hooks, "id")
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

		if restoredID, ok := hooks.restoreIDValue(idIdx); ok {
			hooks.ids[idIdx] = restoredID
		} else {
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
	}

	return hooks.ids[idIdx]
}

// GoUseFunc validates and stores a function for event handling
// The actual wrapping to js.Value happens in the WASM shim layer
func GoUseFunc(fn interface{}) interface{} {
	fiber := GetCurrentFiber()
	if fiber == nil {
		ReportDiagnostic("runtime", DiagnosticError, "GoUseFunc called outside component context")
		panic(actionableHookUsagePanic("GoUseFunc"))
	}

	if fiber.hooks == nil {
		fiber.hooks = &Hooks{owner: fiber}
	} else if fiber.hooks.owner == nil {
		fiber.hooks.owner = fiber
	}

	hooks := fiber.hooks
	recordHookSignature(hooks, "func")
	hooks.index++

	funcIdx := hooks.funcIndex
	hooks.funcIndex++

	// Validate that the input is a function.
	if !isValidHookFunction(fn) {
		panic("GoUseFunc requires a function")
	}

	// Always create new wrapper to ensure latest closure is captured
	// The old optimization of reusing wrappers caused stale closure bugs
	rt := GetGlobalRuntime()
	if rt.domAdapter == nil {
		panic("GoUseFunc domAdapter is nil")
	}

	wrapper := rt.domAdapter.WrapFunction(rt.wrapEventHandler(fiber, fn))

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

	return wrapper
}

// areDepsEqual compares dependency arrays
func areDepsEqual(prevDeps, newDeps []interface{}) bool {
	pLen := len(prevDeps)
	nLen := len(newDeps)

	if pLen != nLen {
		return false
	}

	if pLen == 0 {
		return true
	}

	// Unroll for common sizes (most hooks have 1-3 deps)
	switch pLen {
	case 1:
		return fastEqual(prevDeps[0], newDeps[0])
	case 2:
		return fastEqual(prevDeps[0], newDeps[0]) && fastEqual(prevDeps[1], newDeps[1])
	case 3:
		return fastEqual(prevDeps[0], newDeps[0]) && fastEqual(prevDeps[1], newDeps[1]) && fastEqual(prevDeps[2], newDeps[2])
	case 4:
		return fastEqual(prevDeps[0], newDeps[0]) && fastEqual(prevDeps[1], newDeps[1]) &&
			fastEqual(prevDeps[2], newDeps[2]) && fastEqual(prevDeps[3], newDeps[3])
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
	// Fast path: nil checks
	if a == nil && b == nil {
		return true
	}
	if a == nil || b == nil {
		return false
	}

	// Fast path for common primitives before reflection.
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
	case int8:
		if vb, ok := b.(int8); ok {
			return va == vb
		}
	case int16:
		if vb, ok := b.(int16); ok {
			return va == vb
		}
	case uint16:
		if vb, ok := b.(uint16); ok {
			return va == vb
		}
	case uintptr:
		if vb, ok := b.(uintptr); ok {
			return va == vb
		}
	}

	// Get types only for non-primitive fallbacks.
	ta := reflect.TypeOf(a)
	tb := reflect.TypeOf(b)

	if ta != tb {
		return false
	}

	if ta.Kind() == reflect.Func {
		return sameFunctionIdentity(a, b)
	}

	if ta.Comparable() {
		return a == b
	}

	// Optimization: Use reference equality for Slices and Maps
	// This aligns with React's behavior and avoids expensive DeepEqual
	if ta.Kind() == reflect.Slice || ta.Kind() == reflect.Map {
		return reflect.ValueOf(a).Pointer() == reflect.ValueOf(b).Pointer()
	}

	// Use reflect for complex types (structs with non-comparable fields)
	return reflect.DeepEqual(a, b)
}
