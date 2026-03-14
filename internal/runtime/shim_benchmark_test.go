//go:build js && wasm
// +build js,wasm

package runtime

import "testing"

func BenchmarkTextShim(b *testing.B) {
	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = Text("payload")
	}
}

func BenchmarkGoUseStateGlobalInit(b *testing.B) {
	resetGlobalRuntimeForTest()
	InitGlobalRuntime(Config{
		DOMAdapter: newTestDOMAdapter(),
		Scheduler:  newTestScheduler(),
	})
	defer resetGlobalRuntimeForTest()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		SetCurrentFiber(&Fiber{typeOf: "bench", props: map[string]interface{}{}})
		_, _ = GoUseStateGlobal(1)
	}
	SetCurrentFiber(nil)
}

func BenchmarkGoUseAtomGlobalInit(b *testing.B) {
	resetGlobalRuntimeForTest()
	InitGlobalRuntime(Config{
		DOMAdapter: newTestDOMAdapter(),
		Scheduler:  newTestScheduler(),
	})
	defer resetGlobalRuntimeForTest()

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		SetCurrentFiber(&Fiber{typeOf: "bench", props: map[string]interface{}{}})
		_, _ = GoUseAtomGlobal("bench-atom", 1)
	}
	SetCurrentFiber(nil)
}
