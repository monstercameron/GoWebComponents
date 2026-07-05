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
