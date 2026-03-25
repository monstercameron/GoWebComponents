package runtime

import "testing"

func BenchmarkGetGlobalRuntimeLazy(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetGlobalRuntimeForTest()
		if GetGlobalRuntime() == nil {
			parseB.Fatal("expected global runtime")
		}
	}
}

func BenchmarkInitGlobalRuntimeAfterLazyGet(parseB *testing.B) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		resetGlobalRuntimeForTest()
		_ = GetGlobalRuntime()
		InitGlobalRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
	}
}

func BenchmarkRenderTo(parseB *testing.B) {
	parseAdapter := newQueryTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseContainer := parseAdapter.CreateElement("div")
	parseAdapter.selectorResults["#app"] = parseContainer
	parseElement := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})
		parseScheduler.timeouts = parseScheduler.timeouts[:0]
		parseRt.RenderTo("#app", parseElement)
	}
}
