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

func TestGoUseEffect_RunsPreviousCleanupOnDependencyChange(parseT *testing.T) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]any)}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseCleanupRuns := 0
	GoUseEffect(func() func() {
		return func() { parseCleanupRuns++ }
	}, "a")

	if len(parseFiber.effects) != 1 {
		parseT.Fatalf("expected one queued effect on initial render, got %d", len(parseFiber.effects))
	}

	parseCleanup := parseFiber.effects[0].Fn()
	parseFiber.hooks.cleanups[0] = parseCleanup
	resetHookRenderState(parseFiber)

	GoUseEffect(func() func() {
		return func() { parseCleanupRuns++ }
	}, "b")

	if parseCleanupRuns != 1 {
		parseT.Fatalf("expected previous cleanup to run before replacement effect, got %d", parseCleanupRuns)
	}
	if len(parseFiber.effects) != 1 {
		parseT.Fatalf("expected replacement effect to be queued, got %d", len(parseFiber.effects))
	}
}
