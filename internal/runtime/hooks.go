package runtime

import (
	"fmt"
	"reflect"
	"strconv"
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
func safeComparableEqual(parseA, parseB any) (bool, bool) {
	// A statically comparable struct may contain an interface field whose
	// dynamic value is a slice, map, or function. Comparing two such interface
	// values panics. This used to discover that case by actually comparing under
	// recover, which made panic construction/unwinding a production render hot
	// path for ordinary typed component props. Value.Comparable performs the
	// same recursive eligibility check without throwing; the caller retains its
	// DeepEqual fallback when either payload cannot be compared.
	parseValueA := reflect.ValueOf(parseA)
	parseValueB := reflect.ValueOf(parseB)
	if !parseValueA.IsValid() || !parseValueB.IsValid() ||
		!parseValueA.Comparable() || !parseValueB.Comparable() {
		return false, false
	}
	return parseA == parseB, true
}

// isNilableType is a core package helper.
func isNilableType[T any]() bool {
	parseT := reflect.TypeFor[T]()
	switch parseT.Kind() {
	case reflect.Chan, reflect.Func, reflect.Interface, reflect.Map, reflect.Pointer, reflect.Slice:
		return true
	default:
		return false
	}
}

// stateAccessor caches one state slot's getter/setter closure pair across
// renders. The closures depend only on the Runtime, the Hooks store, and the
// slot indices — all stable for the store's lifetime — so rebuilding them per
// render (2 allocations per UseState per render) is pure churn. The getter is
// stored type-erased; a failed func() T assertion (state type changed, e.g.
// hot reload) rebuilds the pair.
type stateAccessor struct {
	getter any
	setter func(any)
}

const initialHookSlotCapacity = 8

func hookBackingCapacity(parseNeeded, parseMinimum int) int {
	// Hook-heavy components commonly cross the initial eight-slot block. Jump
	// straight to 32 there: the old needed*2 policy allocated 8, 18, then 38
	// slots while building a 20-hook component, copying pointer-rich stores on
	// every step. The plateau removes one allocation/copy and retains fewer
	// total slots across the construction sequence.
	if parseMinimum <= initialHookSlotCapacity && parseNeeded > initialHookSlotCapacity && parseNeeded <= 32 {
		return 32
	}
	parseCapacity := parseNeeded * 2
	if parseCapacity < parseMinimum {
		return parseMinimum
	}
	return parseCapacity
}

func appendFiberEffect(parseFiber *Fiber, parseEffect Effect) {
	if parseFiber.effects == nil {
		parseFiber.effects = make([]Effect, 0, initialHookSlotCapacity)
	} else if len(parseFiber.effects) == cap(parseFiber.effects) {
		parseCapacity := hookBackingCapacity(len(parseFiber.effects)+1, initialHookSlotCapacity)
		parseEffects := make([]Effect, len(parseFiber.effects), parseCapacity)
		copy(parseEffects, parseFiber.effects)
		parseFiber.effects = parseEffects
	}
	parseFiber.effects = append(parseFiber.effects, parseEffect)
}

// StateSlot is a value-type handle to one state hook slot. It avoids allocating
// getter and setter closures for UI callers that can retain the slot directly.
// The low-level GoUseState API still exposes its historical closure pair.
type StateSlot[T any] struct {
	runtime      *Runtime
	hooks        *Hooks
	fiber        *Fiber
	stateIndex   int
	pendingIndex int
}

// Valid reports whether the slot was claimed during a component render.
func (parseSlot StateSlot[T]) Valid() bool {
	return parseSlot.hooks != nil && parseSlot.stateIndex >= 0 && parseSlot.stateIndex < len(parseSlot.hooks.states)
}

// Get returns the slot's current value.
func (parseSlot StateSlot[T]) Get() T {
	if parseSlot.hooks == nil || parseSlot.stateIndex < 0 || parseSlot.stateIndex >= len(parseSlot.hooks.states) {
		var parseZero T
		return parseZero
	}
	parseValue, _ := parseSlot.hooks.states[parseSlot.stateIndex].(T)
	return parseValue
}

// Set replaces the slot value or applies a functional updater, then schedules
// the owning fiber through the same lane/async rules as GoUseState.
func (parseSlot StateSlot[T]) Set(parseNewValueOrUpdater any) {
	parseHooks := parseSlot.hooks
	parseRt := parseSlot.runtime
	if parseHooks == nil || parseRt == nil {
		return
	}
	if parseRt.shouldPostAsyncStateUpdate() {
		parseRt.PostAsync(func() {
			if parseRt.ShouldDeferStateUpdates() {
				parseRt.ScheduleTransition(func() { parseSlot.applyStateUpdate(parseNewValueOrUpdater, "transition") })
				return
			}
			parseSlot.applyStateUpdate(parseNewValueOrUpdater, "async")
		})
		return
	}
	if parseRt.ShouldDeferStateUpdates() {
		parseRt.ScheduleTransition(func() { parseSlot.applyStateUpdate(parseNewValueOrUpdater, "transition") })
		return
	}
	parseSlot.applyStateUpdate(parseNewValueOrUpdater, "local-state")
}

// applyStateUpdate performs the synchronous half of StateSlot.Set. Keeping it
// as a method lets the overwhelmingly common direct event path avoid creating
// the escaping closure needed only by async and transition scheduling.
func (parseSlot StateSlot[T]) applyStateUpdate(parseNewValueOrUpdater any, parseUpdateOrigin string) {
	parseHooks := parseSlot.hooks
	parseRt := parseSlot.runtime
	if parseHooks == nil || parseRt == nil {
		return
	}
	parseTargetFiber := parseHooks.owner
	if parseTargetFiber == nil {
		parseTargetFiber = parseSlot.fiber
	}
	parseRt.reportStrictSetStateDuringRender(parseTargetFiber, "GoUseState")
	if parseSlot.pendingIndex >= len(parseHooks.states) {
		parseNeeded := parseSlot.pendingIndex + 1
		if parseNeeded > cap(parseHooks.states) {
			parseNewStates := make([]any, parseNeeded, parseNeeded*2)
			copy(parseNewStates, parseHooks.states)
			parseHooks.states = parseNewStates
		} else if parseNeeded > len(parseHooks.states) {
			parseHooks.states = parseHooks.states[:parseNeeded]
		}
	}

	var parseCurrentValue T
	if parseValue, parseOk := parseHooks.states[parseSlot.stateIndex].(T); parseOk {
		parseCurrentValue = parseValue
	}
	// Nilability matters only when interpreting a setter payload. Computing it
	// while claiming the hook made every render inspect T even if Set was never
	// called.
	parseNewValue, parseOk := resolveStateUpdateValue(parseCurrentValue, parseNewValueOrUpdater)
	if !parseOk || fastEqual(parseCurrentValue, parseNewValue) {
		return
	}

	parseHooks.states[parseSlot.pendingIndex] = parseNewValue
	parseHooks.states[parseSlot.stateIndex] = parseNewValue
	if parseRt.activeRenderFiber == parseTargetFiber {
		parseTargetFiber.renderPhaseUpdate = true
		return
	}
	// One event commonly updates several state slots owned by the same
	// component. The first setter has already resolved the live owner, marked
	// it and its ancestors, and scheduled the pass; every later setter only has
	// to publish its value before that pass starts. Avoid repeating the live-tree
	// resolution walk and scheduler entry for the already-covered lane.
	parseLane := laneForUpdateOrigin(parseUpdateOrigin)
	// Once traversal starts, even an already-dirty owner may have been visited.
	// Keep the scheduler's in-flight replay accounting on that path.
	if parseRt.updateScheduled && parseRt.wipRoot != nil && parseRt.nextUnitOfWork == parseRt.wipRoot &&
		parseTargetFiber.dirty && parseTargetFiber.needsUpdate &&
		parseTargetFiber.updateLane == parseLane {
		return
	}
	parseRt.ScheduleOwnedFiberUpdateWithOrigin(parseTargetFiber, parseUpdateOrigin)
}

// GoUseStateSlot claims one state hook without constructing accessor closures.
func GoUseStateSlot[T any](parseRt *Runtime, parseInitialValue T) StateSlot[T] {
	parseFiber := requireCurrentHookFiber("GoUseState")
	if parseFiber.hooks == nil {
		parseFiber.hooks = &Hooks{owner: parseFiber}
	} else if parseFiber.hooks.owner == nil {
		parseFiber.hooks.owner = parseFiber
	}

	parseHooks := parseFiber.hooks
	recordHookSignature(parseHooks, "state")
	parseHooks.index++
	parseStateIdx := parseHooks.stateIndex
	parseHooks.stateIndex++

	parseNeededLen := (parseStateIdx + 1) * 2
	if len(parseHooks.states) < parseNeededLen {
		if parseNeededLen <= cap(parseHooks.states) {
			parseHooks.states = parseHooks.states[:parseNeededLen]
		} else {
			parseNewStates := make([]any, parseNeededLen, hookBackingCapacity(parseNeededLen, initialHookSlotCapacity*2))
			copy(parseNewStates, parseHooks.states)
			parseHooks.states = parseNewStates
		}

		parseRestoredValue, parseOk := parseHooks.restoreStateValue(parseStateIdx)
		if !parseOk {
			parseRestoredValue = parseInitialValue
		} else if parseCoerced, parseOk2 := coerceHotReloadValue(parseRestoredValue, reflect.TypeOf(parseInitialValue)); parseOk2 {
			parseRestoredValue = parseCoerced
		} else {
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
		parseHooks.states[parseStateIdx*2] = parseRestoredValue
		parseHooks.states[parseStateIdx*2+1] = parseRestoredValue
	}

	return StateSlot[T]{
		runtime:      parseRt,
		hooks:        parseHooks,
		fiber:        parseFiber,
		stateIndex:   parseStateIdx * 2,
		pendingIndex: parseStateIdx*2 + 1,
	}
}

// GoUseState provides state management for components
func GoUseState[T any](parseRt *Runtime, parseInitialValue T) (func() T, func(any)) {
	parseSlot := GoUseStateSlot(parseRt, parseInitialValue)
	parseHooks := parseSlot.hooks
	parseStateIdx := parseHooks.stateIndex - 1

	// Cached accessor fast path: steady-state re-renders reuse the slot's
	// closure pair instead of allocating a fresh getter+setter per call.
	if parseStateIdx < len(parseHooks.stateAccessors) {
		parseCached := &parseHooks.stateAccessors[parseStateIdx]
		if parseCachedGetter, parseOk := parseCached.getter.(func() T); parseOk && parseCached.setter != nil {
			return parseCachedGetter, parseCached.setter
		}
	}

	parseGetter := func() T { return parseSlot.Get() }
	parseSetter := func(parseNewValueOrUpdater any) { parseSlot.Set(parseNewValueOrUpdater) }

	if parseNeededAcc := parseStateIdx + 1; len(parseHooks.stateAccessors) < parseNeededAcc {
		if parseNeededAcc <= cap(parseHooks.stateAccessors) {
			parseHooks.stateAccessors = parseHooks.stateAccessors[:parseNeededAcc]
		} else {
			parseNewAccessors := make([]stateAccessor, parseNeededAcc, parseNeededAcc*2)
			copy(parseNewAccessors, parseHooks.stateAccessors)
			parseHooks.stateAccessors = parseNewAccessors
		}
	}
	parseHooks.stateAccessors[parseStateIdx] = stateAccessor{getter: parseGetter, setter: parseSetter}

	return parseGetter, parseSetter
}

// GoUseEffect runs side effects and supports cleanup
// The effect function can return a cleanup function that will be called before the next effect runs or on unmount
// GoUseEffect registers a passive effect that runs after the commit's DOM
// mutation. See goUseEffectImpl.
func GoUseEffect(parseEffect func() func(), parseDeps ...any) {
	goUseEffectImpl(parseEffect, false, parseDeps...)
}

// GoUseLayoutEffect registers a layout effect (G36): it runs synchronously after
// DOM mutation, before the browser paints, and before this fiber's passive
// effects. Use it for post-render DOM reads/writes that must be observed before
// paint — focusing a freshly mounted element, measuring layout, scrolling — so
// they need no setTimeout/rAF guess. Same (effect, deps...) contract as
// GoUseEffect.
func GoUseLayoutEffect(parseEffect func() func(), parseDeps ...any) {
	goUseEffectImpl(parseEffect, true, parseDeps...)
}

func goUseEffectImpl(parseEffect func() func(), parseLayout bool, parseDeps ...any) {
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
		// The previous cleanup is deliberately NOT run here.
		//
		// This is the render phase, and a render is not a commitment: it can be
		// interrupted by a higher lane, restarted, or discarded by an error
		// boundary. A cleanup run here has already closed a socket, cancelled a
		// timer, or dropped a subscription for a setup that may never be queued
		// to replace it, leaving the component with neither.
		//
		// It stays in parseHooks.cleanups so runOneEffect can run it at COMMIT,
		// immediately before its replacement — and so an unmount before that
		// commit still finds it. See runOneEffect.

		// Queue the new effect
		// Capture position for cleanup storage
		appendFiberEffect(parseFiber, Effect{
			Fn:           parseEffect,
			CleanupIndex: parseCleanupIdx,
			Layout:       parseLayout,
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
			parseNewMemos := make([]memoizedValue, parseNeeded, hookBackingCapacity(parseNeeded, initialHookSlotCapacity))
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
				parseMemo.singleDep = nil
				parseMemo.hasSingleDep = false
			}
		}
	}

	// Check if we need to recompute: only if deps changed (or this is the first render and memo.deps is uninitialized)
	// First render: memo.deps will be nil, so we need to initialize it
	// Subsequent renders: check if deps changed
	if parseMemo.deps == nil || !areDepsEqual(parseMemo.deps, parseDeps) {
		parseMemo.value = parseCompute()
		parseMemo.deps = parseDeps
		parseMemo.singleDep = nil
		parseMemo.hasSingleDep = false
	}

	return parseMemo.value
}

// ensureMemoSlot runs the shared memo-hook prologue (hooks store init,
// signature, index bookkeeping, slot growth) and returns the slot.
func ensureMemoSlot(parseName string) *memoizedValue {
	parseFiber := requireCurrentHookFiber(parseName)

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

	if len(parseHooks.memos) <= parseMemoIdx {
		parseNeeded := parseMemoIdx + 1
		if parseNeeded <= cap(parseHooks.memos) {
			parseHooks.memos = parseHooks.memos[:parseNeeded]
		} else {
			parseNewMemos := make([]memoizedValue, parseNeeded, hookBackingCapacity(parseNeeded, initialHookSlotCapacity))
			copy(parseNewMemos, parseHooks.memos)
			parseHooks.memos = parseNewMemos
		}
	}
	return &parseHooks.memos[parseMemoIdx]
}

// hookRestoreForMemoSlot exposes the hot-reload restore pair for the memo
// slot the prologue just claimed (index was advanced, so -1).
func hookRestoreForMemoSlot() (any, []any, bool) {
	parseFiber := GetCurrentFiber()
	if parseFiber == nil || parseFiber.hooks == nil {
		return nil, nil, false
	}
	return parseFiber.hooks.restoreMemoValue(parseFiber.hooks.memoIndex - 1)
}

// GoUseMemoFor memoizes one typed computation without the func()-any adapter
// closure the untyped entry point forces on generic callers: the compute
// function passes through unwrapped, and the hot-reload type coercion target
// comes from the type parameter (reflect only runs on the restore path).
func GoUseMemoFor[T any](parseCompute func() T, parseDeps ...any) T {
	parseMemo := ensureMemoSlot("GoUseMemo")

	if parseMemo.deps == nil {
		if parseRestoredValue, parseRestoredDeps, parseOk := hookRestoreForMemoSlot(); parseOk {
			if parseCoerced, parseOk2 := coerceHotReloadValue(parseRestoredValue, reflect.TypeFor[T]()); parseOk2 {
				parseMemo.value = parseCoerced
				parseMemo.deps = parseRestoredDeps
				parseMemo.singleDep = nil
				parseMemo.hasSingleDep = false
			}
		}
	}

	if parseMemo.deps == nil || !areDepsEqual(parseMemo.deps, parseDeps) {
		parseValue := parseCompute()
		parseMemo.value = parseValue
		parseMemo.deps = parseDeps
		parseMemo.singleDep = nil
		parseMemo.hasSingleDep = false
		return parseValue
	}
	parseValue, _ := parseMemo.value.(T)
	return parseValue
}

// GoUseMemoOf memoizes one computation keyed by a single comparable
// dependency with zero steady-state allocations: the compute function
// receives the dependency (so it can be a static, non-capturing func), the
// unchanged-dep check type-asserts the stored box instead of boxing the new
// value, and the cached result is returned by assertion. It shares the memo
// slot machinery (and hook signature) with GoUseMemo, so the two may not be
// swapped for one another between renders of one component.
func GoUseMemoOf[T any, D comparable](parseCompute func(D) T, parseDep D) T {
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

	if len(parseHooks.memos) <= parseMemoIdx {
		parseNeeded := parseMemoIdx + 1
		if parseNeeded <= cap(parseHooks.memos) {
			parseHooks.memos = parseHooks.memos[:parseNeeded]
		} else {
			parseNewMemos := make([]memoizedValue, parseNeeded, hookBackingCapacity(parseNeeded, initialHookSlotCapacity))
			copy(parseNewMemos, parseHooks.memos)
			parseHooks.memos = parseNewMemos
		}
	}

	parseMemo := &parseHooks.memos[parseMemoIdx]
	if parseMemo.hasSingleDep {
		if parsePrev, parseOk := parseMemo.singleDep.(D); parseOk && parsePrev == parseDep {
			if parseValue, parseOk2 := parseMemo.value.(T); parseOk2 {
				return parseValue
			}
		}
	} else if len(parseMemo.deps) == 1 {
		if parsePrev, parseOk := parseMemo.deps[0].(D); parseOk && parsePrev == parseDep {
			if parseValue, parseOk2 := parseMemo.value.(T); parseOk2 {
				return parseValue
			}
		}
	}

	parseValue := parseCompute(parseDep)
	parseMemo.value = parseValue
	parseMemo.deps = nil
	parseMemo.singleDep = parseDep
	parseMemo.hasSingleDep = true
	return parseValue
}

// GoUseEffectOf registers a passive effect keyed by a single comparable
// dependency without the variadic []any deps allocation. Same slot machinery
// as GoUseEffect ("effect" signature); not swappable with it between renders.
func GoUseEffectOf[D comparable](parseEffect func() func(), parseDep D) {
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

	growTypedEffectSlots(parseHooks, parseCleanupIdx)
	if len(parseHooks.effectSingleDeps) <= parseDepIdx {
		parseNeeded := parseDepIdx + 1
		if parseNeeded <= cap(parseHooks.effectSingleDeps) {
			parseHooks.effectSingleDeps = parseHooks.effectSingleDeps[:parseNeeded]
		} else {
			parseNewDeps := make([]singleEffectDependency, parseNeeded, hookBackingCapacity(parseNeeded, initialHookSlotCapacity))
			copy(parseNewDeps, parseHooks.effectSingleDeps)
			parseHooks.effectSingleDeps = parseNewDeps
		}
	}

	shouldRun := false
	parseSingleDep := &parseHooks.effectSingleDeps[parseDepIdx]
	if !parseSingleDep.valid {
		parseSingleDep.value = parseDep
		parseSingleDep.valid = true
		shouldRun = true
	} else {
		parsePrev, parseOk := parseSingleDep.value.(D)
		if !parseOk || parsePrev != parseDep || parseHooks.effectEpochs[parseCleanupIdx] != parseHooks.effectEpoch {
			shouldRun = true
		}
		parseSingleDep.value = parseDep
		parseSingleDep.valid = true
	}

	if shouldRun {
		if parseHooks.cleanups[parseCleanupIdx] != nil {
			parseStart := time.Now()
			parseHooks.cleanups[parseCleanupIdx]()
			parseDurationNs := time.Since(parseStart).Nanoseconds()
			parseFiber.cleanupDurationNs += parseDurationNs
			recordSlowOperationDiagnostic("cleanup", parseFiber, parseDurationNs)
			parseHooks.cleanups[parseCleanupIdx] = nil
		}
		appendFiberEffect(parseFiber, Effect{
			Fn:           parseEffect,
			CleanupIndex: parseCleanupIdx,
		})
		parseHooks.effectEpochs[parseCleanupIdx] = parseHooks.effectEpoch
	}
}

// growTypedEffectSlots omits the generic dependency slice. UseEffectOf's API
// contract forbids swapping it with UseEffect at the same hook position, and
// its single comparable dependency already has dedicated storage.
func growTypedEffectSlots(parseHooks *Hooks, parseCleanupIdx int) {
	if len(parseHooks.cleanups) <= parseCleanupIdx {
		parseNeeded := parseCleanupIdx + 1
		if parseNeeded <= cap(parseHooks.cleanups) {
			parseHooks.cleanups = parseHooks.cleanups[:parseNeeded]
		} else {
			parseNewCleanups := make([]func(), parseNeeded, hookBackingCapacity(parseNeeded, initialHookSlotCapacity))
			copy(parseNewCleanups, parseHooks.cleanups)
			parseHooks.cleanups = parseNewCleanups
		}
	}
	if len(parseHooks.effectEpochs) <= parseCleanupIdx {
		parseNeeded := parseCleanupIdx + 1
		if parseNeeded <= cap(parseHooks.effectEpochs) {
			parseHooks.effectEpochs = parseHooks.effectEpochs[:parseNeeded]
		} else {
			parseNewEpochs := make([]int, parseNeeded, hookBackingCapacity(parseNeeded, initialHookSlotCapacity))
			copy(parseNewEpochs, parseHooks.effectEpochs)
			parseHooks.effectEpochs = parseNewEpochs
		}
	}
}

// growEffectSlots extends the deps/cleanups/effectEpochs stores to cover one
// effect slot; shared by the variadic and typed effect entry points.
func growEffectSlots(parseHooks *Hooks, parseDepIdx, parseCleanupIdx int) {
	if len(parseHooks.deps) <= parseDepIdx {
		parseNeeded := parseDepIdx + 1
		if parseNeeded <= cap(parseHooks.deps) {
			parseHooks.deps = parseHooks.deps[:parseNeeded]
		} else {
			parseNewDeps := make([][]any, parseNeeded, hookBackingCapacity(parseNeeded, initialHookSlotCapacity))
			copy(parseNewDeps, parseHooks.deps)
			parseHooks.deps = parseNewDeps
		}
	}
	if len(parseHooks.cleanups) <= parseCleanupIdx {
		parseNeeded2 := parseCleanupIdx + 1
		if parseNeeded2 <= cap(parseHooks.cleanups) {
			parseHooks.cleanups = parseHooks.cleanups[:parseNeeded2]
		} else {
			parseNewCleanups := make([]func(), parseNeeded2, hookBackingCapacity(parseNeeded2, initialHookSlotCapacity))
			copy(parseNewCleanups, parseHooks.cleanups)
			parseHooks.cleanups = parseNewCleanups
		}
	}
	if len(parseHooks.effectEpochs) <= parseCleanupIdx {
		parseNeeded3 := parseCleanupIdx + 1
		if parseNeeded3 <= cap(parseHooks.effectEpochs) {
			parseHooks.effectEpochs = parseHooks.effectEpochs[:parseNeeded3]
		} else {
			parseNewEpochs := make([]int, parseNeeded3, hookBackingCapacity(parseNeeded3, initialHookSlotCapacity))
			copy(parseNewEpochs, parseHooks.effectEpochs)
			parseHooks.effectEpochs = parseNewEpochs
		}
	}
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
			parseRt := runtimeForFiber(parseFiber)
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
//
// The ID is a CSS-safe identifier of the form "gwc-<global-id>-<hook-position>".
// It deliberately uses a hyphen separator (NOT a colon): an element id is most
// commonly fed straight into a CSS selector (`querySelector("#"+id)`), and a
// colon is a pseudo-class separator there — `#gwc:3:1` throws a SyntaxError and,
// in a wasm callback, panics the page. A hyphen is a valid CSS identifier
// character, so the generated id needs no escaping. (Historical note: this used
// to emit "gwc:N:N"; see G29.)
func formatHookID(parseID int, parsePosition int) string {
	parseBuffer := make([]byte, 0, 8+20+1+20)
	parseBuffer = append(parseBuffer, "gwc-"...)
	parseBuffer = strconv.AppendInt(parseBuffer, int64(parseID), 10)
	parseBuffer = append(parseBuffer, '-')
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
	parseRt := runtimeForFiber(parseFiber)
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

	// NOTE(perf A/B 2026-07-05): an interface-header identity fast path here
	// (same type descriptor + same data pointer => equal) measured FLAT in
	// the browser refresh/update scenarios and was removed: granular dirty
	// marking means clean shell components never re-render, so stable
	// handler/map references rarely reach this compare on hot paths. Re-try
	// only with evidence that DeepEqual fallbacks are hot.

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
			// Object.is semantics: treat NaN as equal to NaN, else a NaN dep makes
			// areDepsEqual report "changed" every render and re-runs forever.
			if parseVa != parseVa && parseVb4 != parseVb4 {
				return true
			}
			return parseVa == parseVb4
		}
	case int64:
		if parseVb5, parseOk5 := parseB.(int64); parseOk5 {
			return parseVa == parseVb5
		}
	case float32:
		if parseVb6, parseOk6 := parseB.(float32); parseOk6 {
			if parseVa != parseVa && parseVb6 != parseVb6 {
				return true
			}
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
	case map[string]string:
		// Same identity semantics as the generic slice/map fallback below, but
		// without paying two reflect.TypeOf calls to get there.
		if parseVb21, parseOk21 := parseB.(map[string]string); parseOk21 {
			if parseVa == nil || parseVb21 == nil {
				return parseVa == nil && parseVb21 == nil
			}
			return reflect.ValueOf(parseVa).Pointer() == reflect.ValueOf(parseVb21).Pointer()
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
