package runtime

import "testing"

// BenchmarkGetHookRuntimeOwnedFiber measures the owner lookup used by the
// public state-slot wrapper while a component fiber is current.
func BenchmarkGetHookRuntimeOwnedFiber(parseB *testing.B) {
	resetGlobalRuntimeForTest()
	defer resetGlobalRuntimeForTest()
	parseRt := GetGlobalRuntime()
	parseFiber := &Fiber{typeOf: "hook-grid-cell", ownerRuntime: parseRt}
	SetCurrentFiber(parseFiber)
	defer SetCurrentFiber(nil)

	parseB.ReportAllocs()
	for parseB.Loop() {
		if getHookRuntime() != parseRt {
			parseB.Fatal("hook runtime did not match the owning fiber")
		}
	}
}
