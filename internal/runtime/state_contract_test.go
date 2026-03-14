package runtime

import "testing"

func TestGoUseAtom_FunctionalUpdatesCompose(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	fiber := newTestFiber("test")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(rt, "counter", 1)
	set(func(prev int) int { return prev + 1 })
	set(func(prev int) int { return prev * 3 })

	if got := get(); got != 6 {
		t.Fatalf("expected composed atom updates to produce 6, got %d", got)
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one scheduled update, got %d", len(scheduler.timeouts))
	}
}

func TestGoUseAtom_NilableStateCanResetToNil(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	initial := 42
	fiber := newTestFiber("test")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(rt, "ptr", &initial)
	set(nil)

	if got := get(); got != nil {
		t.Fatal("expected pointer atom state to reset to nil")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected nil atom reset to schedule one update, got %d", len(scheduler.timeouts))
	}
}
