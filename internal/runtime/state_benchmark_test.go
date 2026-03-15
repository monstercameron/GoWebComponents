package runtime

import "testing"

func BenchmarkAtomRegistryInitAtomExisting(b *testing.B) {
	registry := NewAtomRegistry()
	registry.InitAtom("counter", 0)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		registry.InitAtom("counter", i)
	}
}

func BenchmarkAtomRegistryGetAtom(b *testing.B) {
	registry := NewAtomRegistry()
	registry.InitAtom("counter", 42)

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		value, ok := registry.GetAtom("counter")
		if !ok || value.(int) != 42 {
			b.Fatal("expected atom value")
		}
	}
}

func BenchmarkAtomRegistrySubscribeUnsubscribe(b *testing.B) {
	registry := NewAtomRegistry()
	fiber := newTestFiber("bench")

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		registry.Subscribe("counter", fiber)
		registry.Unsubscribe("counter", fiber)
	}
}

func BenchmarkAtomRegistrySetAtom32Subscribers(b *testing.B) {
	registry := NewAtomRegistry()
	for i := 0; i < 32; i++ {
		registry.Subscribe("counter", &Fiber{typeOf: "sub"})
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		subs := registry.SetAtom("counter", i)
		if len(subs) != 32 {
			b.Fatalf("expected 32 subscribers, got %d", len(subs))
		}
	}
}

func BenchmarkGoUseAtomIntUpdate(b *testing.B) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	fiber := newTestFiber("bench")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseAtom(rt, "counter", 0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.updateScheduled = false
		scheduler.timeouts = scheduler.timeouts[:0]
		set(i)
	}
}

func BenchmarkGoUseAtomPointerNilReset(b *testing.B) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	initial := 1
	fiber := newTestFiber("bench")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseAtom(rt, "ptr", &initial)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.updateScheduled = false
		scheduler.timeouts = scheduler.timeouts[:0]
		if i%2 == 0 {
			set(nil)
		} else {
			value := i
			set(&value)
		}
	}
}

func BenchmarkGoUseAtomGetter(b *testing.B) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	fiber := newTestFiber("bench")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	get, _ := GoUseAtom(rt, "counter", 42)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		if value := get(); value != 42 {
			b.Fatal("expected atom value")
		}
	}
}

func BenchmarkGoUseAtomStableRerender(b *testing.B) {
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{Scheduler: scheduler})
	rt.currentRoot = &Fiber{typeOf: "ROOT"}
	fiber := newTestFiber("bench")
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	_, _ = GoUseAtom(rt, "counter", 42)
	resetHookRenderState(fiber)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetHookRenderState(fiber)
		get, _ := GoUseAtom(rt, "counter", 42)
		if value := get(); value != 42 {
			b.Fatal("expected atom value")
		}
	}
}

func BenchmarkCleanupAtomSubscriptions8(b *testing.B) {
	rt := NewRuntime(Config{Scheduler: newTestScheduler()})
	fiber := newTestFiber("bench")
	fiber.hooks.atoms = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for _, atomID := range fiber.hooks.atoms {
		rt.atomRegistry.Subscribe(atomID, fiber)
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, atomID := range fiber.hooks.atoms {
			rt.atomRegistry.Subscribe(atomID, fiber)
		}
		rt.CleanupAtomSubscriptions(fiber)
	}
}

func BenchmarkAtomRegistryUnsubscribeMany8(b *testing.B) {
	registry := NewAtomRegistry()
	fiber := newTestFiber("bench")
	atomIDs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		for _, atomID := range atomIDs {
			registry.Subscribe(atomID, fiber)
		}
		registry.UnsubscribeMany(atomIDs, fiber)
	}
}
