package runtime

import "testing"

func TestGoUseAtom_FunctionalUpdatesCompose(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	parseFiber := newTestFiber("test")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(parseRt, "counter", 1)
	set(func(parsePrev int) int { return parsePrev + 1 })
	set(func(parsePrev2 int) int { return parsePrev2 * 3 })

	if parseGot := get(); parseGot != 6 {
		parseT.Fatalf("expected composed atom updates to produce 6, got %d", parseGot)
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected one scheduled update, got %d", len(parseScheduler.timeouts))
	}
}

func TestGoUseAtom_NilableStateCanResetToNil(parseT *testing.T) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	parseInitial := 42
	parseFiber := newTestFiber("test")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(parseRt, "ptr", &parseInitial)
	set(nil)

	if parseGot := get(); parseGot != nil {
		parseT.Fatal("expected pointer atom state to reset to nil")
	}
	if len(parseScheduler.timeouts) != 1 {
		parseT.Fatalf("expected nil atom reset to schedule one update, got %d", len(parseScheduler.timeouts))
	}
}
