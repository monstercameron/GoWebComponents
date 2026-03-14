package runtime

import "testing"

func BenchmarkGetGlobalRuntimeLazy(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resetGlobalRuntimeForTest()
		if GetGlobalRuntime() == nil {
			b.Fatal("expected global runtime")
		}
	}
}

func BenchmarkInitGlobalRuntimeAfterLazyGet(b *testing.B) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		resetGlobalRuntimeForTest()
		_ = GetGlobalRuntime()
		InitGlobalRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
	}
}

func BenchmarkRenderTo(b *testing.B) {
	adapter := newQueryTestDOMAdapter()
	scheduler := newTestScheduler()
	container := adapter.CreateElement("div")
	adapter.selectorResults["#app"] = container
	element := &Element{Type: "div", Props: map[string]interface{}{"id": "app"}}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})
		scheduler.timeouts = scheduler.timeouts[:0]
		rt.RenderTo("#app", element)
	}
}
