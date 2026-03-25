package hotreload

import "testing"

func BenchmarkEnableDisableCycleMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		Enable()
		Disable()
		_ = IsEnabled()
	}
}
