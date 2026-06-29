package render

import "testing"

func BenchmarkNewResourceControllerMicro(parseB *testing.B) {
	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		_ = NewResourceController[int]()
	}
}
