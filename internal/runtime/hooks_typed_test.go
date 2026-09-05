package runtime

import "testing"

// resetTypedHookIndices rewinds the per-render hook cursors the way
// renderFunctionComponent does between renders of one component.
func resetTypedHookIndices(parseFiber *Fiber) {
	if parseFiber.hooks == nil {
		return
	}
	parseFiber.hooks.index = 0
	parseFiber.hooks.memoIndex = 0
	parseFiber.hooks.depIndex = 0
	parseFiber.hooks.cleanupIndex = 0
	parseFiber.hooks.signature = parseFiber.hooks.signature[:0]
}

func TestGoUseStateSlotUpdatesWithoutAccessorClosures(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler, currentRoot: &Fiber{}}
	parseFiber := &Fiber{typeOf: "test"}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseSlot := GoUseStateSlot(parseRt, 5)
	if !parseSlot.Valid() || parseSlot.Get() != 5 {
		parseT.Fatalf("initial slot = (%t, %d), want (true, 5)", parseSlot.Valid(), parseSlot.Get())
	}
	if len(parseFiber.hooks.stateAccessors) != 0 {
		parseT.Fatalf("slot path created %d accessor closures", len(parseFiber.hooks.stateAccessors))
	}
	parseSlot.Set(func(parseValue int) int { return parseValue + 3 })
	if parseSlot.Get() != 8 {
		parseT.Fatalf("updated slot = %d, want 8", parseSlot.Get())
	}
}

func TestGoUseMemoOf_CachesAndRecomputesOnDepChange(parseT *testing.T) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]any)}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseComputeCount := 0
	parseCompute := func(parseDep int) int {
		parseComputeCount++
		return parseDep * 3
	}

	if parseGot := GoUseMemoOf(parseCompute, 2); parseGot != 6 {
		parseT.Fatalf("first compute: got %d, want 6", parseGot)
	}
	resetTypedHookIndices(parseFiber)
	if parseGot := GoUseMemoOf(parseCompute, 2); parseGot != 6 {
		parseT.Fatalf("cached read: got %d, want 6", parseGot)
	}
	if parseComputeCount != 1 {
		parseT.Fatalf("same dep recomputed: %d computes, want 1", parseComputeCount)
	}
	resetTypedHookIndices(parseFiber)
	if parseGot := GoUseMemoOf(parseCompute, 5); parseGot != 15 {
		parseT.Fatalf("dep change: got %d, want 15", parseGot)
	}
	if parseComputeCount != 2 {
		parseT.Fatalf("dep change computes = %d, want 2", parseComputeCount)
	}
}

func TestGoUseMemoOf_StoresSingleDependencyWithoutSlice(parseT *testing.T) {
	parseFiber := &Fiber{typeOf: "test"}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	if parseGot := GoUseMemoOf(func(parseDep int) int { return parseDep * 2 }, 7); parseGot != 14 {
		parseT.Fatalf("expected computed value 14, got %d", parseGot)
	}
	parseMemo := parseFiber.hooks.memos[0]
	if !parseMemo.hasSingleDep || parseMemo.singleDep != 7 {
		parseT.Fatalf("expected specialized dependency 7, got %#v", parseMemo)
	}
	if parseMemo.deps != nil {
		parseT.Fatalf("expected no dependency slice on typed single-dependency path, got %#v", parseMemo.deps)
	}
	if parseDeps := memoizedDependencies(parseMemo); len(parseDeps) != 1 || parseDeps[0] != 7 {
		parseT.Fatalf("expected cold dependency view [7], got %#v", parseDeps)
	}
}

func TestGoUseEffectOf_RunsOnceAndRerunsOnDepChange(parseT *testing.T) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]any)}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseRunCount := 0
	parseCleanupCount := 0
	parseEffect := func() func() {
		parseRunCount++
		return func() { parseCleanupCount++ }
	}

	runQueuedTypedEffects := func() {
		for _, parseQueued := range parseFiber.effects {
			if parseCleanup := parseQueued.Fn(); parseCleanup != nil {
				parseFiber.hooks.cleanups[parseQueued.CleanupIndex] = parseCleanup
			}
		}
		parseFiber.effects = parseFiber.effects[:0]
	}

	GoUseEffectOf(parseEffect, "a")
	if len(parseFiber.hooks.deps) != 0 || !parseFiber.hooks.effectSingleDeps[0].valid || parseFiber.hooks.effectSingleDeps[0].value != "a" {
		parseT.Fatalf("typed effect allocated generic deps instead of storing one specialized dependency: deps=%#v specialized=%#v", parseFiber.hooks.deps[0], parseFiber.hooks.effectSingleDeps[0])
	}
	runQueuedTypedEffects()
	if parseRunCount != 1 {
		parseT.Fatalf("first render runs = %d, want 1", parseRunCount)
	}

	resetTypedHookIndices(parseFiber)
	GoUseEffectOf(parseEffect, "a")
	runQueuedTypedEffects()
	if parseRunCount != 1 || parseCleanupCount != 0 {
		parseT.Fatalf("same dep re-ran effect (runs=%d cleanups=%d)", parseRunCount, parseCleanupCount)
	}

	resetTypedHookIndices(parseFiber)
	GoUseEffectOf(parseEffect, "b")
	runQueuedTypedEffects()
	if parseRunCount != 2 || parseCleanupCount != 1 {
		parseT.Fatalf("dep change: runs=%d cleanups=%d, want 2/1", parseRunCount, parseCleanupCount)
	}
}
