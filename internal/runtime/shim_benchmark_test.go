//go:build js && wasm

package runtime

import "testing"

func BenchmarkTextShim(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
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
	for parseI := 0; parseI < parseB.N; parseI++ {
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
	for parseI := 0; parseI < parseB.N; parseI++ {
		SetCurrentFiber(&Fiber{typeOf: "bench", props: map[string]interface{}{}})
		_, _ = GoUseAtomGlobal("bench-atom", 1)
	}
	SetCurrentFiber(nil)
}
