//go:build js && wasm

package runtime

import "testing"

func BenchmarkTextShim(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = Text("payload")
	}
}

func BenchmarkGoUseStateGlobalInit(parseB *testing.B) {
	resetGlobalRuntimeForTest()
	InitGlobalRuntime(Config{
		DOMAdapter: newTestDOMAdapter(),
		Scheduler:  newTestScheduler(),
	})
	defer resetGlobalRuntimeForTest()

	parseB.ReportAllocs()
	for parseB.Loop() {
		SetCurrentFiber(&Fiber{typeOf: "bench", props: map[string]interface{}{}})
		_, _ = GoUseStateGlobal(1)
	}
	SetCurrentFiber(nil)
}

func BenchmarkGoUseAtomGlobalInit(parseB *testing.B) {
	resetGlobalRuntimeForTest()
	InitGlobalRuntime(Config{
		DOMAdapter: newTestDOMAdapter(),
		Scheduler:  newTestScheduler(),
	})
	defer resetGlobalRuntimeForTest()

	parseB.ReportAllocs()
	for parseB.Loop() {
		SetCurrentFiber(&Fiber{typeOf: "bench", props: map[string]interface{}{}})
		_, _ = GoUseAtomGlobal("bench-atom", 1)
	}
	SetCurrentFiber(nil)
}
