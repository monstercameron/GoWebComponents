package virtualization

import "testing"

func BenchmarkComputeViewportStateMicro(parseB *testing.B) {
	parseConfig := ViewportConfig{
		TotalItems: 20000,
		RowHeight:  32,
		Overscan:   4,
	}

	parseB.ReportAllocs()
	for parseI := 0; parseI < parseB.N; parseI++ {
		parseScrollTop := float64((parseI % 6000) * 12)
		_, parseErr := ComputeViewportState(parseConfig, parseScrollTop, 640)
		if parseErr != nil {
			parseB.Fatal(parseErr)
		}
	}
}
