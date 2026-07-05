package runtime

import "testing"

// BenchmarkGetGlobalRuntime gates the lock-free-global experiment: the wasm
// ui.UseState path takes this mutex once per hook call (800x/render in the
// hooks scenario).
func BenchmarkGetGlobalRuntime(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		if GetGlobalRuntime() == nil {
			parseB.Fatal("nil runtime")
		}
	}
}
