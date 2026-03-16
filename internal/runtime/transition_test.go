package runtime

import "testing"

func TestGoUseStateTransitionDefersUpdateUntilTimeout(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	fiber := &Fiber{typeOf: "test", props: map[string]interface{}{}}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(rt, 1)
	rt.StartTransition(func() {
		set(5)
	})

	if got := get(); got != 1 {
		t.Fatalf("expected transition update to stay deferred before timeout, got %d", got)
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one deferred transition timeout, got %d", len(scheduler.timeouts))
	}
	if pending, _ := rt.GetAtomValue(transitionPendingAtomID); pending != true {
		t.Fatalf("expected transition pending atom true before flush, got %#v", pending)
	}

	scheduler.timeouts[0]()

	if got := get(); got != 5 {
		t.Fatalf("expected deferred state update after timeout, got %d", got)
	}
	if pending, _ := rt.GetAtomValue(transitionPendingAtomID); pending != false {
		t.Fatalf("expected transition pending atom false after flush, got %#v", pending)
	}
}

func TestGoUseAtomTransitionDefersSharedUpdateUntilTimeout(t *testing.T) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	fiber := &Fiber{typeOf: "test", props: map[string]interface{}{}}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseAtom(rt, "shared-transition", 2)
	rt.StartTransition(func() {
		set(8)
	})

	if got := get(); got != 2 {
		t.Fatalf("expected atom update to stay deferred before timeout, got %d", got)
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one deferred atom timeout, got %d", len(scheduler.timeouts))
	}

	scheduler.timeouts[0]()

	if got := get(); got != 8 {
		t.Fatalf("expected deferred atom update after timeout, got %d", got)
	}
}

func TestScheduleTransitionWithoutSchedulerRunsImmediately(t *testing.T) {
	rt := NewRuntime(Config{})
	executed := false
	rt.ScheduleTransition(func() {
		executed = true
	})
	if !executed {
		t.Fatal("expected transition callback to run immediately without scheduler")
	}
}