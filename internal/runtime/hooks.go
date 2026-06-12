package runtime

import (
	"fmt"
	"reflect"
	"strconv"
	"sync"
	"time"
	"unsafe"
)

type emptyInterfaceHeader struct {
	typ  unsafe.Pointer
	data unsafe.Pointer
}

// sameFunctionIdentity is a core package helper.
func sameFunctionIdentity(parseA, parseB any) bool {
	if parseA == nil || parseB == nil {
		return parseA == parseB
	}
	parseVa := reflect.ValueOf(parseA)
	parseVb := reflect.ValueOf(parseB)
	if !parseVa.IsValid() || !parseVb.IsValid() || parseVa.Kind() != reflect.Func || parseVb.Kind() != reflect.Func {
		return false
	}
	if parseVa.Type() != parseVb.Type() {
		return false
	}
	parseAHeader := (*emptyInterfaceHeader)(unsafe.Pointer(&parseA))
	parseBHeader := (*emptyInterfaceHeader)(unsafe.Pointer(&parseB))
	return parseVa.Pointer() == parseVb.Pointer() && parseAHeader.data == parseBHeader.data
}

// safeComparableEqual is a core package helper.
func safeComparableEqual(parseA, parseB any) (isEqual bool, isOk bool) {
	defer func() {
		if recover() != nil {
			isOk = false
		}
	}()
	return parseA == parseB, true
}

var nilableTypeCache sync.Map

// isNilableType is a core package helper.
func isNilableType[T any]() bool {
	parseT := reflect.TypeFor[T]()
	if parseCached, parseOk := nilableTypeCache.Load(parseT); parseOk {
		return parseCached.(bool)
	}

	var isNilable bool
	switch parseT.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		isNilable = true
	default:
		isNilable = false
	}

	nilableTypeCache.Store(parseT, isNilable)
	return isNilable
}

// GoUseState provides state management for components
func GoUseState[T any](parseRt *Runtime, parseInitialValue T) (func() T, func(any)) {
	parseFiber := requireCurrentHookFiber("GoUseState")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	recordHookSignature(parseFiber.hooks, "state")
	parseFiber.hooks.index++

	parseStateIdx := parseFiber.hooks.stateIndex
	parseFiber.hooks.stateIndex++

	// Initialize state if needed
	// We need 2 slots per state hook: [state, pending]
	parseNeededLen := (parseStateIdx + 1) * 2
	if len(parseFiber.hooks.states) < parseNeededLen {
		if parseNeededLen <= cap(parseFiber.hooks.states) {
			// Extend slice within capacity
			parseFiber.hooks.states = parseFiber.hooks.states[:parseNeededLen]
		} else {
			// Grow slice
			parseNewStates := make([]any, parseNeededLen, parseNeededLen*2)
			copy(parseNewStates, parseFiber.hooks.states)
			parseFiber.hooks.states = parseNewStates
		}

		// Initialize new slots
		parseRestoredValue, parseOk := parseFiber.hooks.restoreStateValue(parseStateIdx)
		if !parseOk {
			parseRestoredValue = parseInitialValue
		} else if parseCoerced, parseOk2 := coerceHotReloadValue(parseRestoredValue, reflect.TypeOf(parseInitialValue)); parseOk2 {
			parseRestoredValue = parseCoerced
		} else {
			// Coercion failed: the snapshot value is incompatible with the
			// current state type.  Fall back to the initial value so we do not
			// silently store an untyped value into the wrong slot, and report a
			// diagnostic so the developer can see what happened.
			parseSnapshotType := fmt.Sprintf("%T", parseRestoredValue)
			parseRestoredValue = parseInitialValue
			ReportDiagnosticWithContext(
				"runtime",
				DiagnosticWarning,
				fmt.Sprintf("hot reload state restore dropped for hook slot %d: snapshot value %s is not coercible to %T; using initial value",
					parseStateIdx, parseSnapshotType, parseInitialValue),
				diagnosticPathForFiber(parseFiber),
				diagnosticComponentStack(parseFiber),
			)
		}
		parseFiber.hooks.states[parseStateIdx*2] = parseRestoredValue
		parseFiber.hooks.states[parseStateIdx*2+1] = parseRestoredValue
	}

	parseHooks := parseFiber.hooks
	// Capture indices for closure
	parseSIdx := parseStateIdx * 2
	parsePIdx := parseStateIdx*2 + 1
	parseNilableState := isNilableType[T]()

	parseGetter := func() T {
		// Bounds check removed for performance - slice is grown before closure creation
		// Type assertion should always succeed if state was initialized correctly
		parseValue, _ := parseHooks.states[parseSIdx].(T)
		return parseValue
	}

	parseSetter := func(parseNewValueOrUpdater any) {
		apply := func(parseUpdateOrigin string) {
			if parsePIdx >= len(parseHooks.states) {
				parseNeeded := parsePIdx + 1
				if parseNeeded > cap(parseHooks.states) {
					parseNewStates2 := make([]any, parseNeeded, parseNeeded*2)
					copy(parseNewStates2, parseHooks.states)
					parseHooks.states = parseNewStates2
				} else if parseNeeded > len(parseHooks.states) {
					parseHooks.states = parseHooks.states[:parseNeeded]
				}
			}

			var parseCurrentValue T
			if parseCv, parseOk3 := parseHooks.states[parseSIdx].(T); parseOk3 {
				parseCurrentValue = parseCv
			}

			parseNewValue, parseOk4 := resolveStateUpdateValue(parseCurrentValue, parseNewValueOrUpdater, parseNilableState)
			if !parseOk4 {
				return
			}

			if fastEqual(parseCurrentValue, parseNewValue) {
				return
			}

			parseHooks.states[parsePIdx] = parseNewValue
			parseHooks.states[parseSIdx] = parseNewValue
			parseTargetFiber := parseHooks.owner
			if parseTargetFiber == nil {
				parseTargetFiber = parseFiber
			}
			parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseTargetFiber, parseUpdateOrigin)
		}

		if parseRt != nil && parseRt.ShouldDeferStateUpdates() {
			parseRt.ScheduleTransition(func() {
				apply("transition")
			})
			return
		}

		apply("local-state")
	}

	return parseGetter, parseSetter
}

// GoUseEffect runs side effects and supports cleanup
// The effect function can return a cleanup function that will be called before the next effect runs or on unmount
func GoUseEffect(parseEffect func() func(), parseDeps ...any) {
	parseFiber := requireCurrentHookFiber("GoUseEffect")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	parseHooks := parseFiber.hooks
	recordHookSignature(parseHooks, "effect")
	parseHooks.index++

	parseDepIdx := parseHooks.depIndex
	parseHooks.depIndex++

	parseCleanupIdx := parseHooks.cleanupIndex
	parseHooks.cleanupIndex++

	// Grow deps and cleanups if needed
	if len(parseHooks.deps) <= parseDepIdx {
		parseNeeded := parseDepIdx + 1
		if parseNeeded <= cap(parseHooks.deps) {
			parseHooks.deps = parseHooks.deps[:parseNeeded]
		} else {
			parseNewDeps := make([][]any, parseNeeded, parseNeeded*2)
			copy(parseNewDeps, parseHooks.deps)
			parseHooks.deps = parseNewDeps
		}
	}
	if len(parseHooks.cleanups) <= parseCleanupIdx {
		parseNeeded2 := parseCleanupIdx + 1
		if parseNeeded2 <= cap(parseHooks.cleanups) {
			parseHooks.cleanups = parseHooks.cleanups[:parseNeeded2]
		} else {
			parseNewCleanups := make([]func(), parseNeeded2, parseNeeded2*2)
			copy(parseNewCleanups, parseHooks.cleanups)
			parseHooks.cleanups = parseNewCleanups
		}
	}
	if len(parseHooks.effectEpochs) <= parseCleanupIdx {
		parseNeeded3 := parseCleanupIdx + 1
		if parseNeeded3 <= cap(parseHooks.effectEpochs) {
			parseHooks.effectEpochs = parseHooks.effectEpochs[:parseNeeded3]
		} else {
			parseNewEpochs := make([]int, parseNeeded3, parseNeeded3*2)
			copy(parseNewEpochs, parseHooks.effectEpochs)
			parseHooks.effectEpochs = parseNewEpochs
		}
	}

	shouldRun := false
	if parseHooks.deps[parseDepIdx] == nil {
		// First run
		parseHooks.deps[parseDepIdx] = parseDeps
		shouldRun = true
	} else {
		// Check if deps changed
		parsePrevDeps := parseHooks.deps[parseDepIdx]
		if len(parseDeps) == 0 || !areDepsEqual(parsePrevDeps, parseDeps) || parseHooks.effectEpochs[parseCleanupIdx] != parseHooks.effectEpoch {
			parseHooks.deps[parseDepIdx] = parseDeps
			shouldRun = true
		}
	}

	if shouldRun {
		// Run cleanup from previous effect if it exists
		if parseHooks.cleanups[parseCleanupIdx] != nil {
			parseStart := time.Now()
			parseHooks.cleanups[parseCleanupIdx]()
			parseDurationNs := time.Since(parseStart).Nanoseconds()
			parseFiber.cleanupDurationNs += parseDurationNs
			recordSlowOperationDiagnostic("cleanup", parseFiber, parseDurationNs)
			parseHooks.cleanups[parseCleanupIdx] = nil
		}

		// Queue the new effect
		if parseFiber.effects == nil {
			parseFiber.effects = make([]Effect, 0)
		}

		// Capture position for cleanup storage
		parseFiber.effects = append(parseFiber.effects, Effect{
			Fn:           parseEffect,
			CleanupIndex: parseCleanupIdx,
		})
		parseHooks.effectEpochs[parseCleanupIdx] = parseHooks.effectEpoch
	}
}

// GoUseMemo memoizes expensive computations
func GoUseMemo(parseCompute func() any, parseDeps ...any) any {
	return goUseMemo(parseCompute, nil, parseDeps...)
}

// goUseMemo is a core package helper.
func goUseMemo(parseCompute func() any, parseTargetType reflect.Type, parseDeps ...any) any {
	parseFiber := requireCurrentHookFiber("GoUseMemo")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	parseHooks := parseFiber.hooks
	recordHookSignature(parseHooks, "memo")
	parseHooks.index++

	parseMemoIdx := parseHooks.memoIndex
	parseHooks.memoIndex++

	// Grow memos if needed
	if len(parseHooks.memos) <= parseMemoIdx {
		parseNeeded := parseMemoIdx + 1
		if parseNeeded <= cap(parseHooks.memos) {
			parseHooks.memos = parseHooks.memos[:parseNeeded]
		} else {
			parseNewMemos := make([]memoizedValue, parseNeeded, parseNeeded*2)
			copy(parseNewMemos, parseHooks.memos)
			parseHooks.memos = parseNewMemos
		}
	}

	parseMemo := &parseHooks.memos[parseMemoIdx]

	if parseMemo.deps == nil {
		if parseRestoredValue, parseRestoredDeps, parseOk := parseHooks.restoreMemoValue(parseMemoIdx); parseOk {
			if parseCoerced, parseOk2 := coerceHotReloadValue(parseRestoredValue, parseTargetType); parseOk2 {
				parseMemo.value = parseCoerced
				parseMemo.deps = parseRestoredDeps
			}
		}
	}

	// Check if we need to recompute: only if deps changed (or this is the first render and memo.deps is uninitialized)
	// First render: memo.deps will be nil, so we need to initialize it
	// Subsequent renders: check if deps changed
	if parseMemo.deps == nil || !areDepsEqual(parseMemo.deps, parseDeps) {
		parseMemo.value = parseCompute()
		parseMemo.deps = parseDeps
	}

	return parseMemo.value
}

// GoUseMemoTyped memoizes expensive computations and coerces restored hot reload
// values to the caller's expected type when possible.
func GoUseMemoTyped(parseCompute func() any, parseTargetType reflect.Type, parseDeps ...any) any {
	return goUseMemo(parseCompute, parseTargetType, parseDeps...)
}

// GoUseCallback memoizes a callback function with dependency tracking
func GoUseCallback(parseFn any, parseDeps ...any) any {
	parseFiber := requireCurrentHookFiber("GoUseCallback")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	parseHooks := parseFiber.hooks
	recordHookSignature(parseHooks, "callback")
	parseHooks.index++

	parseCallbackIdx := parseHooks.callbackIndex
	parseHooks.callbackIndex++

	// Grow callbacks if needed
	if len(parseHooks.callbacks) <= parseCallbackIdx {
		parseNeeded := parseCallbackIdx + 1
		if parseNeeded <= cap(parseHooks.callbacks) {
			parseHooks.callbacks = parseHooks.callbacks[:parseNeeded]
		} else {
			parseNewCallbacks := make([]callbackValue, parseNeeded, parseNeeded*2)
			copy(parseNewCallbacks, parseHooks.callbacks)
			parseHooks.callbacks = parseNewCallbacks
		}
	}

	parseCallback := &parseHooks.callbacks[parseCallbackIdx]

	// Check if we need to update: only if deps changed or this is the first render
	if parseCallback.deps == nil || !areDepsEqual(parseCallback.deps, parseDeps) {
		parseCallback.fn = parseFn
		parseCallback.deps = parseDeps
	}

	return parseCallback.fn
}

// GoUseRef creates a mutable reference that persists across renders
// It returns a RefValue object with a .Current field that can hold any value
// Unlike state, updating a ref does NOT trigger a re-render
func GoUseRef(parseInitialValue any) *RefValue {
	parseFiber := requireCurrentHookFiber("GoUseRef")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	parseHooks := parseFiber.hooks
	recordHookSignature(parseHooks, "ref")
	parseHooks.index++

	parseRefIdx := parseHooks.refIndex
	parseHooks.refIndex++

	// Grow refs if needed
	if len(parseHooks.refs) <= parseRefIdx {
		parseNeeded := parseRefIdx + 1
		if parseNeeded <= cap(parseHooks.refs) {
			parseHooks.refs = parseHooks.refs[:parseNeeded]
		} else {
			parseNewRefs := make([]*RefValue, parseNeeded, parseNeeded*2)
			copy(parseNewRefs, parseHooks.refs)
			parseHooks.refs = parseNewRefs
		}
	}

	// If ref is nil (first render), create it with initial value
	if parseHooks.refs[parseRefIdx] == nil {
		parseRestoredValue, parseOk := parseHooks.restoreRefValue(parseRefIdx)
		if !parseOk {
			parseRestoredValue = parseInitialValue
		}
		parseHooks.refs[parseRefIdx] = &RefValue{Current: parseRestoredValue}
	}

	return parseHooks.refs[parseRefIdx]
}

// GoUseId generates a unique, stable identifier for accessibility attributes
// The ID is generated once and persists across renders without changing
// This is useful for associating labels with form inputs and other accessibility needs
func GoUseId() string {
	parseFiber := requireCurrentHookFiber("GoUseId")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	parseHooks := parseFiber.hooks
	parsePosition := parseHooks.index
	recordHookSignature(parseHooks, "id")
	parseHooks.index++

	parseIdIdx := parseHooks.idIndex
	parseHooks.idIndex++

	// Initialize IDs array if needed
	if len(parseHooks.ids) <= parseIdIdx {
		parseNeeded := parseIdIdx + 1
		if parseNeeded <= cap(parseHooks.ids) {
			parseHooks.ids = parseHooks.ids[:parseNeeded]
		} else {
			parseNewIds := make([]string, parseNeeded, parseNeeded*2)
			copy(parseNewIds, parseHooks.ids)
			parseHooks.ids = parseNewIds
		}

		if parseRestoredID, parseOk := parseHooks.restoreIDValue(parseIdIdx); parseOk {
			parseHooks.ids[parseIdIdx] = parseRestoredID
		} else {
			// Generate a new unique ID
			// On first render, create a new ID using fiber's component ID counter
			parseRt := GetGlobalRuntime()
			parseRt.idCounterMu.Lock()
			parseRt.idCounter++
			parseId := parseRt.idCounter
			parseRt.idCounterMu.Unlock()

			// Create ID in format: "gwc:<global-id>:<hook-position>".
			// This stays stable across renders without the fmt.Sprintf allocation path.
			parseHooks.ids[parseIdIdx] = formatHookID(parseId, parsePosition)
		}
	}

	return parseHooks.ids[parseIdIdx]
}

// formatHookID builds one stable hook ID using the runtime hook-global counter and hook position.
func formatHookID(parseID int, parsePosition int) string {
	parseBuffer := make([]byte, 0, 8+20+1+20)
	parseBuffer = append(parseBuffer, "gwc:"...)
	parseBuffer = strconv.AppendInt(parseBuffer, int64(parseID), 10)
	parseBuffer = append(parseBuffer, ':')
	parseBuffer = strconv.AppendInt(parseBuffer, int64(parsePosition), 10)
	return string(parseBuffer)
}

// GoUseFunc validates and stores a function for event handling
// The actual wrapping to js.Value happens in the WASM shim layer
func GoUseFunc(parseFn any) any {
	parseFiber := requireCurrentHookFiber("GoUseFunc")

	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	parseHooks := parseFiber.hooks
	recordHookSignature(parseHooks, "func")
	parseHooks.index++

	parseFuncIdx := parseHooks.funcIndex
	parseHooks.funcIndex++

	// Validate that the input is a function.
	if !isValidHookFunction(parseFn) {
		panic(actionableGoUseFuncTypePanic())
	}

	// Always create new wrapper to ensure latest closure is captured
	// The old optimization of reusing wrappers caused stale closure bugs
	parseRt := GetGlobalRuntime()
	if parseRt.domAdapter == nil {
		panic(actionableGoUseFuncDOMAdapterPanic())
	}

	if len(parseHooks.funcs) <= parseFuncIdx {
		parseNeeded := parseFuncIdx + 1
		if parseNeeded <= cap(parseHooks.funcs) {
			parseHooks.funcs = parseHooks.funcs[:parseNeeded]
		} else {
			parseNewFuncs := make([]funcHandlerValue, parseNeeded, parseNeeded*2)
			copy(parseNewFuncs, parseHooks.funcs)
			parseHooks.funcs = parseNewFuncs
		}
	}

	parseHandlerVal := parseHooks.funcs[parseFuncIdx]
	parseFnType := reflect.TypeOf(parseFn)
	if parseHandlerVal.wrapper != nil && parseHandlerVal.cell != nil && parseHandlerVal.cell.fnType == parseFnType {
		parseHandlerVal.fn = parseFn
		parseHandlerVal.cell.fn = parseFn
		parseHandlerVal.cell.fnVal = reflect.ValueOf(parseFn)
		parseHandlerVal.cell.owner = parseFiber
		parseHooks.funcs[parseFuncIdx] = parseHandlerVal
		return parseHandlerVal.wrapper
	}

	releaseFuncHandlerWrapper(parseHandlerVal.wrapper)
	parseCell := &funcHandlerCell{
		owner:  parseFiber,
		fn:     parseFn,
		fnVal:  reflect.ValueOf(parseFn),
		fnType: parseFnType,
	}
	parseHandlerVal = funcHandlerValue{
		fn:      parseFn,
		wrapper: parseRt.domAdapter.WrapFunction(parseRt.wrapEventHandlerCell(parseCell)),
		cell:    parseCell,
	}
	parseHooks.funcs[parseFuncIdx] = parseHandlerVal

	return parseHandlerVal.wrapper
}

// releaseFuncHandlerWrapper releases one wrapped event handler when the adapter exposes cleanup.
func releaseFuncHandlerWrapper(parseWrapper any) {
	if parseWrapper == nil {
		return
	}
	if parseReleasable, parseOk := parseWrapper.(interface{ Release() }); parseOk {
		parseReleasable.Release()
	}
}

// areDepsEqual compares dependency arrays
func areDepsEqual(parsePrevDeps, parseNewDeps []any) bool {
	parsePLen := len(parsePrevDeps)
	parseNLen := len(parseNewDeps)

	if parsePLen != parseNLen {
		return false
	}

	if parsePLen == 0 {
		return true
	}

	// Unroll for common sizes (most hooks have 1-3 deps)
	switch parsePLen {
	case 1:
		return fastEqual(parsePrevDeps[0], parseNewDeps[0])
	case 2:
		return fastEqual(parsePrevDeps[0], parseNewDeps[0]) && fastEqual(parsePrevDeps[1], parseNewDeps[1])
	case 3:
		return fastEqual(parsePrevDeps[0], parseNewDeps[0]) && fastEqual(parsePrevDeps[1], parseNewDeps[1]) && fastEqual(parsePrevDeps[2], parseNewDeps[2])
	case 4:
		return fastEqual(parsePrevDeps[0], parseNewDeps[0]) && fastEqual(parsePrevDeps[1], parseNewDeps[1]) &&
			fastEqual(parsePrevDeps[2], parseNewDeps[2]) && fastEqual(parsePrevDeps[3], parseNewDeps[3])
	}

	for parseI := range parsePrevDeps {
		if !fastEqual(parsePrevDeps[parseI], parseNewDeps[parseI]) {
			return false
		}
	}

	return true
}

// fastEqual performs optimized equality checking
func fastEqual(parseA, parseB any) bool {
	// Fast path: nil checks
	if parseA == nil && parseB == nil {
		return true
	}
	if parseA == nil || parseB == nil {
		return false
	}

	// Fast path for common primitives before reflection.
	switch parseVa := parseA.(type) {
	case int:
		if parseVb, parseOk := parseB.(int); parseOk {
			return parseVa == parseVb
		}
	case string:
		if parseVb2, parseOk2 := parseB.(string); parseOk2 {
			return parseVa == parseVb2
		}
	case bool:
		if parseVb3, parseOk3 := parseB.(bool); parseOk3 {
			return parseVa == parseVb3
		}
	case float64:
		if parseVb4, parseOk4 := parseB.(float64); parseOk4 {
			return parseVa == parseVb4
		}
	case int64:
		if parseVb5, parseOk5 := parseB.(int64); parseOk5 {
			return parseVa == parseVb5
		}
	case float32:
		if parseVb6, parseOk6 := parseB.(float32); parseOk6 {
			return parseVa == parseVb6
		}
	case int32:
		if parseVb7, parseOk7 := parseB.(int32); parseOk7 {
			return parseVa == parseVb7
		}
	case uint:
		if parseVb8, parseOk8 := parseB.(uint); parseOk8 {
			return parseVa == parseVb8
		}
	case uint64:
		if parseVb9, parseOk9 := parseB.(uint64); parseOk9 {
			return parseVa == parseVb9
		}
	case uint32:
		if parseVb10, parseOk10 := parseB.(uint32); parseOk10 {
			return parseVa == parseVb10
		}
	case uint8:
		if parseVb11, parseOk11 := parseB.(uint8); parseOk11 {
			return parseVa == parseVb11
		}
	case int8:
		if parseVb12, parseOk12 := parseB.(int8); parseOk12 {
			return parseVa == parseVb12
		}
	case int16:
		if parseVb13, parseOk13 := parseB.(int16); parseOk13 {
			return parseVa == parseVb13
		}
	case uint16:
		if parseVb14, parseOk14 := parseB.(uint16); parseOk14 {
			return parseVa == parseVb14
		}
	case uintptr:
		if parseVb15, parseOk15 := parseB.(uintptr); parseOk15 {
			return parseVa == parseVb15
		}
	// Common collection types: use reference equality (same-pointer ⇒ equal, different pointer ⇒ not equal).
	// This matches React's shallow-identity semantics for dep arrays.
	case []string:
		if parseVb16, parseOk16 := parseB.([]string); parseOk16 {
			if len(parseVa) == 0 && len(parseVb16) == 0 {
				return true
			}
			return len(parseVa) > 0 && len(parseVb16) > 0 &&
				&parseVa[0] == &parseVb16[0] && len(parseVa) == len(parseVb16)
		}
		return false
	case []int:
		if parseVb17, parseOk17 := parseB.([]int); parseOk17 {
			if len(parseVa) == 0 && len(parseVb17) == 0 {
				return true
			}
			return len(parseVa) > 0 && len(parseVb17) > 0 &&
				&parseVa[0] == &parseVb17[0] && len(parseVa) == len(parseVb17)
		}
		return false
	case []any:
		if parseVb18, parseOk18 := parseB.([]any); parseOk18 {
			if len(parseVa) == 0 && len(parseVb18) == 0 {
				return true
			}
			return len(parseVa) > 0 && len(parseVb18) > 0 &&
				&parseVa[0] == &parseVb18[0] && len(parseVa) == len(parseVb18)
		}
		return false
	case map[string]any:
		// Map identity semantics: equal only when both are nil or both are the
		// same map object (one reflect pointer compare; the type switch already
		// avoided the more expensive TypeOf calls on both sides).
		if parseVb19, parseOk19 := parseB.(map[string]any); parseOk19 {
			if parseVa == nil || parseVb19 == nil {
				return parseVa == nil && parseVb19 == nil
			}
			return reflect.ValueOf(parseVa).Pointer() == reflect.ValueOf(parseVb19).Pointer()
		}
		return false
	case time.Time:
		if parseVb20, parseOk20 := parseB.(time.Time); parseOk20 {
			return parseVa.Equal(parseVb20)
		}
		return false
	}

	// Get types only for non-primitive fallbacks.
	parseTa := reflect.TypeOf(parseA)
	parseTb := reflect.TypeOf(parseB)

	if parseTa != parseTb {
		return false
	}

	if parseTa.Kind() == reflect.Func {
		return sameFunctionIdentity(parseA, parseB)
	}

	if parseTa.Comparable() {
		if parseEqual, parseOk16 := safeComparableEqual(parseA, parseB); parseOk16 {
			return parseEqual
		}
		// Comparable static types can still panic when interface-typed fields carry
		// slices, maps, funcs, or other uncomparable concrete values.
		return reflect.DeepEqual(parseA, parseB)
	}

	// Optimization: Use reference equality for Slices and Maps
	// This aligns with React's behavior and avoids expensive DeepEqual
	if parseTa.Kind() == reflect.Slice || parseTa.Kind() == reflect.Map {
		return reflect.ValueOf(parseA).Pointer() == reflect.ValueOf(parseB).Pointer()
	}

	// Use reflect for complex types (structs with non-comparable fields)
	return reflect.DeepEqual(parseA, parseB)
}
