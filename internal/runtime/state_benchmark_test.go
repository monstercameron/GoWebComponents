package runtime

import "testing"

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
