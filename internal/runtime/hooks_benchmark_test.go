package runtime

import "testing"

func BenchmarkFastEqualInt(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !fastEqual(42, 42) {
			b.Fatal("expected equal ints")
		}
	}
}

func BenchmarkFastEqualString(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !fastEqual("alpha", "alpha") {
			b.Fatal("expected equal strings")
		}
	}
}

func BenchmarkAreDepsEqual3Primitives(b *testing.B) {
	prev := []interface{}{1, "two", true}
	next := []interface{}{1, "two", true}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		if !areDepsEqual(prev, next) {
			b.Fatal("expected equal deps")
		}
	}
}

func BenchmarkGoUseStateIntDirectUpdate(b *testing.B) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler, currentRoot: &Fiber{typeOf: "ROOT"}}
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseState(rt, 0)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		rt.updateScheduled = false
		scheduler.timeouts = scheduler.timeouts[:0]
		set(i)
	}
}

func BenchmarkGoUseStatePointerNilReset(b *testing.B) {
	scheduler := newTestScheduler()
	rt := &Runtime{scheduler: scheduler, currentRoot: &Fiber{typeOf: "ROOT"}}
	initial := 1
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseState(rt, &initial)

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

func BenchmarkGoUseMemoSameDeps(b *testing.B) {
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	_ = GoUseMemo(func() interface{} { return 42 }, "dep")
	resetHookRenderState(fiber)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetHookRenderState(fiber)
		if value := GoUseMemo(func() interface{} { return 42 }, "dep"); value != 42 {
			b.Fatal("expected memoized value")
		}
	}
}

func BenchmarkGoUseEffectSameDeps(b *testing.B) {
	fiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(fiber)
	defer SetCurrentFiber(nil)

	GoUseEffect(func() func() { return nil }, "dep")
	resetHookRenderState(fiber)

	b.ReportAllocs()
	b.ResetTimer()
	for i := 0; i < b.N; i++ {
		resetHookRenderState(fiber)
		GoUseEffect(func() func() { return nil }, "dep")
	}
}
