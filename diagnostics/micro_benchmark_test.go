package diagnostics

import "testing"

func BenchmarkNewReportMicro(b *testing.B) {
	options := Options{
		Summary:  "failed to hydrate route payload",
		Code:     "GWC-EXAMPLE-FAILURE",
		Headline: "render failure",
		Next:     "refresh route payload and retry",
	}

	b.ReportAllocs()
	for i := 0; i < b.N; i++ {
		_ = NewReport(options)
	}
}
