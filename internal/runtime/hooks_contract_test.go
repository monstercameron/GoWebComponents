package runtime

import "testing"

func resetHookRenderState(parseFiber *Fiber) {
	parseFiber.effects = parseFiber.effects[:0]
	parseFiber.hooks.index = 0
	parseFiber.hooks.stateIndex = 0
	parseFiber.hooks.depIndex = 0
	parseFiber.hooks.memoIndex = 0
	parseFiber.hooks.callbackIndex = 0
	parseFiber.hooks.refIndex = 0
	parseFiber.hooks.idIndex = 0
	parseFiber.hooks.fetchIndex = 0
	parseFiber.hooks.funcIndex = 0
	parseFiber.hooks.atomIndex = 0
	parseFiber.hooks.cleanupIndex = 0
}

func TestGoUseState_FunctionalUpdatesCompose(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler, currentRoot: &Fiber{}}
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]any)}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(parseRt, 1)
	set(func(parsePrev int) int { return parsePrev + 1 })
	set(func(parsePrev2 int) int { return parsePrev2 * 3 })

	if parseGot := get(); parseGot != 6 {
		parseT.Fatalf("expected composed functional updates to produce 6, got %d", parseGot)
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one scheduled update, got %d", len(parseScheduler.timeouts))
	}
}

func TestGoUseState_NilableStateCanResetToNil(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler, currentRoot: &Fiber{}}
	parseInitial := 42
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]any)}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(parseRt, &parseInitial)
	set(nil)

	if parseGot := get(); parseGot != nil {
		parseT.Fatal("expected pointer state to reset to nil")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected nil reset to schedule one update, got %d", len(parseScheduler.timeouts))
	}
}

// TestGoUseEffect_RunsPreviousCleanupOnDependencyChange pins WHEN the previous
// cleanup runs, not merely that it does.
//
// It previously asserted the cleanup ran inside GoUseEffect, during render. That
// was the contract and it was the bug: a render is not a commitment — it can be
// interrupted by a higher lane, restarted, or discarded by an error boundary —
// and a cleanup run during one that never commits has closed a socket or dropped
// a subscription for a setup that will never be queued to replace it.
//
// The contract now is React's: the cleanup is left pending through render and
// runs at COMMIT, immediately before the setup that supersedes it. Both halves
// are asserted, because "runs eventually" and "does not run during render" are
// different guarantees and only the pair is useful.
func TestGoUseEffect_RunsPreviousCleanupOnDependencyChange(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Reset: true})
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]any)}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseCleanupRuns := 0
	parseSetupRuns := 0
	GoUseEffect(func() func() {
		parseSetupRuns++
		return func() { parseCleanupRuns++ }
	}, "a")

	if len(parseFiber.effects) != 1 {
		parseT.Fatalf("expected one queued effect on initial render, got %d", len(parseFiber.effects))
	}

	// Commit the first effect the way the runtime does, so its cleanup is
	// registered by the same path production uses.
	parseRt.runOneEffect(parseFiber, &parseFiber.effects[0])
	if parseSetupRuns != 1 || parseCleanupRuns != 0 {
		parseT.Fatalf("after the first commit: setups=%d cleanups=%d, want 1 and 0", parseSetupRuns, parseCleanupRuns)
	}

	resetHookRenderState(parseFiber)
	GoUseEffect(func() func() {
		parseSetupRuns++
		return func() { parseCleanupRuns++ }
	}, "b")

	// RENDER is done. Nothing may have been torn down yet.
	if parseCleanupRuns != 0 {
		parseT.Fatalf("the previous cleanup ran during render (%d times); a render that never commits would tear down state with nothing queued to replace it",
			parseCleanupRuns)
	}
	if len(parseFiber.effects) != 1 {
		parseT.Fatalf("expected replacement effect to be queued, got %d", len(parseFiber.effects))
	}
	if parseFiber.hooks.cleanups[0] == nil {
		parseT.Fatal("the pending cleanup was discarded during render; an unmount before the next commit would never run it")
	}

	// COMMIT. Cleanup first, then the replacement setup.
	parseRt.runOneEffect(parseFiber, &parseFiber.effects[0])
	if parseCleanupRuns != 1 {
		parseT.Fatalf("expected the previous cleanup to run at commit, got %d", parseCleanupRuns)
	}
	if parseSetupRuns != 2 {
		parseT.Fatalf("expected the replacement setup to run at commit, got %d setups", parseSetupRuns)
	}
}
