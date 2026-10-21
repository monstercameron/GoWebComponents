package diagnostics

import "testing"

func BenchmarkNewReportMicro(parseB *testing.B) {
	parseOptions := Options{
		Summary:  "failed to hydrate route payload",
		Code:     "GWC-EXAMPLE-FAILURE",
		Headline: "render failure",
		Next:     "refresh route payload and retry",
	}

	parseB.ReportAllocs()
	for parseB.Loop() {
		_ = NewReport(parseOptions)
	}
}
