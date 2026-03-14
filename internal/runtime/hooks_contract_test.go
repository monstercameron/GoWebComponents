package runtime

import "testing"

func resetHookRenderState(fiber *Fiber) {
	fiber.effects = fiber.effects[:0]
	fiber.hooks.index = 0
	fiber.hooks.stateIndex = 0
	fiber.hooks.depIndex = 0
	fiber.hooks.memoIndex = 0
	fiber.hooks.callbackIndex = 0
	fiber.hooks.refIndex = 0
	fiber.hooks.idIndex = 0
	fiber.hooks.fetchIndex = 0
	fiber.hooks.funcIndex = 0
	fiber.hooks.atomIndex = 0
	fiber.hooks.cleanupIndex = 0
}

func TestGoUseState_FunctionalUpdatesCompose(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler, currentRoot: &Fiber{}}
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(rt, 1)
	set(func(prev int) int { return prev + 1 })
	set(func(prev int) int { return prev * 3 })

	if got := get(); got != 6 {
		t.Fatalf("expected composed functional updates to produce 6, got %d", got)
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected one scheduled update, got %d", len(scheduler.timeouts))
	}
}

func TestGoUseState_NilableStateCanResetToNil(t *testing.T) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler, currentRoot: &Fiber{}}
	initial := 42
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, set := GoUseState(rt, &initial)
	set(nil)

	if got := get(); got != nil {
		t.Fatal("expected pointer state to reset to nil")
	}
	if len(scheduler.timeouts) != 1 {
		t.Fatalf("expected nil reset to schedule one update, got %d", len(scheduler.timeouts))
	}
}

func TestGoUseEffect_RunsPreviousCleanupOnDependencyChange(t *testing.T) {
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	cleanupRuns := 0
	GoUseEffect(func() func() {
		return func() { cleanupRuns++ }
	}, "a")

	if len(fiber.effects) != 1 {
		t.Fatalf("expected one queued effect on initial render, got %d", len(fiber.effects))
	}

	cleanup := fiber.effects[0].Fn()
	fiber.hooks.cleanups[0] = cleanup
	resetHookRenderState(fiber)

	GoUseEffect(func() func() {
		return func() { cleanupRuns++ }
	}, "b")

	if cleanupRuns != 1 {
		t.Fatalf("expected previous cleanup to run before replacement effect, got %d", cleanupRuns)
	}
	if len(fiber.effects) != 1 {
		t.Fatalf("expected replacement effect to be queued, got %d", len(fiber.effects))
	}
}
