package runtime

import "testing"

func BenchmarkIsNilableTypeCachedPointer(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if !isNilableType[*int]() {
			parseB.Fatal("expected pointer type to be nilable")
		}
	}
}

func BenchmarkFastEqualInt(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if !fastEqual(42, 42) {
			parseB.Fatal("expected equal ints")
		}
	}
}

func BenchmarkFastEqualString(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if !fastEqual("alpha", "alpha") {
			parseB.Fatal("expected equal strings")
		}
	}
}

func BenchmarkAreDepsEqual3Primitives(parseB *testing.B) {
	parsePrev := []interface{}{1, "two", true}
	parseNext := []interface{}{1, "two", true}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if !areDepsEqual(parsePrev, parseNext) {
			parseB.Fatal("expected equal deps")
		}
	}
}

func BenchmarkGoUseStateIntDirectUpdate(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler, currentRoot: &Fiber{typeOf: "ROOT"}}
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseState(parseRt, 0)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt.updateScheduled = false
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		set(parseI)
	}
}

func BenchmarkGoUseStatePointerNilReset(parseB *testing.B) {
	parseScheduler := newTestScheduler()
	parseRt := &Runtime{scheduler: parseScheduler, currentRoot: &Fiber{typeOf: "ROOT"}}
	parseInitial := 1
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	_, set := GoUseState(parseRt, &parseInitial)

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

func BenchmarkGoUseMemoSameDeps(parseB *testing.B) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	_ = GoUseMemo(func() interface{} { return 42 }, "dep")
	resetHookRenderState(parseFiber)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetHookRenderState(parseFiber)
		if parseValue := GoUseMemo(func() interface{} { return 42 }, "dep"); parseValue != 42 {
			parseB.Fatal("expected memoized value")
		}
	}
}

func BenchmarkGoUseCallbackSameDeps(parseB *testing.B) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseHandler := func() {}
	_ = GoUseCallback(parseHandler, "dep")
	resetHookRenderState(parseFiber)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetHookRenderState(parseFiber)
		if parseValue := GoUseCallback(parseHandler, "dep"); parseValue == nil {
			parseB.Fatal("expected callback")
		}
	}
}

func BenchmarkGoUseRefStable(parseB *testing.B) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseRef := GoUseRef("payload")
	resetHookRenderState(parseFiber)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetHookRenderState(parseFiber)
		if parseCurrent := GoUseRef(nil); parseCurrent != parseRef {
			parseB.Fatal("expected stable ref")
		}
	}
}

func BenchmarkGoUseIdStable(parseB *testing.B) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseExpected := GoUseId()
	resetHookRenderState(parseFiber)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetHookRenderState(parseFiber)
		if parseId := GoUseId(); parseId != parseExpected {
			parseB.Fatal("expected stable id")
		}
	}
}

func BenchmarkGoUseFuncWrap(parseB *testing.B) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()

	parseReleased := 0
	parseAdapter := &funcWrapTestAdapter{
		testDOMAdapter: newTestDOMAdapter(),
		releasedCount:  &parseReleased,
	}
	InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseHandler := func() {}
	_ = GoUseFunc(parseHandler)
	resetHookRenderState(parseFiber)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetHookRenderState(parseFiber)
		if parseWrapped := GoUseFunc(parseHandler); parseWrapped == nil {
			parseB.Fatal("expected wrapped handler")
		}
	}
}

func BenchmarkGoUseEffectSameDeps(parseB *testing.B) {
	parseFiber := &Fiber{typeOf: "test", props: make(map[string]interface{})}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	GoUseEffect(func() func() { return nil }, "dep")
	resetHookRenderState(parseFiber)

	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetHookRenderState(parseFiber)
		GoUseEffect(func() func() { return nil }, "dep")
	}
}
