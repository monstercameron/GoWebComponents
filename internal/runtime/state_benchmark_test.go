package runtime

import "testing"

func BenchmarkAtomRegistryInitAtomExisting(parseB *testing.B) {
	parseRegistry := NewAtomRegistry()
	parseRegistry.InitAtom("counter", 0)

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRegistry.InitAtom("counter", parseI)
	}
}

func BenchmarkAtomRegistryGetAtom(parseB *testing.B) {
	parseRegistry := NewAtomRegistry()
	parseRegistry.InitAtom("counter", 42)

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseValue, parseOk := parseRegistry.GetAtom("counter")
		if !parseOk || parseValue.(int) != 42 {
			parseB.Fatal("expected atom value")
		}
	}
}

func BenchmarkAtomRegistrySubscribeUnsubscribe(parseB *testing.B) {
	parseRegistry := NewAtomRegistry()
	parseFiber := newTestFiber("bench")

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRegistry.Subscribe("counter", parseFiber)
		parseRegistry.Unsubscribe("counter", parseFiber)
	}
}

func BenchmarkAtomRegistrySetAtom32Subscribers(parseB *testing.B) {
	parseRegistry := NewAtomRegistry()
	for parseI := 0; parseI < 32; parseI++ {
		parseRegistry.Subscribe("counter", &Fiber{typeOf: "sub"})
	}

	parseB.ReportAllocs()
	for parseI2 := 0; parseI2 < parseB.N; parseI2++ {
		parseSubs := parseRegistry.SetAtom("counter", parseI2)
		if len(parseSubs) != 32 {
			parseB.Fatalf("expected 32 subscribers, got %d", len(parseSubs))
		}
	}
}

func BenchmarkGoUseAtomIntUpdate(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	parseFiber := newTestFiber("bench")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseAtom(parseRt, "counter", 0)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.updateScheduled = false
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		set(parseI)
	}
}

func BenchmarkGoUseAtomPointerNilReset(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	parseInitial := 1
	parseFiber := newTestFiber("bench")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseAtom(parseRt, "ptr", &parseInitial)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.updateScheduled = false
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		if parseI%2 == 0 {
			set(nil)
		} else {
			parseValue := parseI
			set(&parseValue)
		}
	}
}

func BenchmarkGoUseAtomGetter(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	parseFiber := newTestFiber("bench")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	get, _ := GoUseAtom(parseRt, "counter", 42)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if parseValue := get(); parseValue != 42 {
			parseB.Fatal("expected atom value")
		}
	}
}

func BenchmarkGoUseAtomStableRerender(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{Scheduler: parseScheduler})
	parseRt.currentRoot = &Fiber{typeOf: "ROOT"}
	parseFiber := newTestFiber("bench")
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	_, _ = GoUseAtom(parseRt, "counter", 42)
	resetHookRenderState(parseFiber)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetHookRenderState(parseFiber)
		get, _ := GoUseAtom(parseRt, "counter", 42)
		if parseValue := get(); parseValue != 42 {
			parseB.Fatal("expected atom value")
		}
	}
}

func BenchmarkCleanupAtomSubscriptions8(parseB *testing.B) {
	parseRt := NewRuntime(Config{Scheduler: newTestScheduler()})
	parseFiber := newTestFiber("bench")
	parseFiber.hooks.atoms = []string{"a", "b", "c", "d", "e", "f", "g", "h"}
	for _, parseAtomID := range parseFiber.hooks.atoms {
		parseRt.atomRegistry.Subscribe(parseAtomID, parseFiber)
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		for _, parseAtomID2 := range parseFiber.hooks.atoms {
			parseRt.atomRegistry.Subscribe(parseAtomID2, parseFiber)
		}
		parseRt.CleanupAtomSubscriptions(parseFiber)
	}
}

func BenchmarkAtomRegistryUnsubscribeMany8(parseB *testing.B) {
	parseRegistry := NewAtomRegistry()
	parseFiber := newTestFiber("bench")
	parseAtomIDs := []string{"a", "b", "c", "d", "e", "f", "g", "h"}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		for _, parseAtomID := range parseAtomIDs {
			parseRegistry.Subscribe(parseAtomID, parseFiber)
		}
		parseRegistry.UnsubscribeMany(parseAtomIDs, parseFiber)
	}
}
