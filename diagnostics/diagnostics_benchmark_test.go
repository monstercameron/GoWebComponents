package diagnostics

import "testing"

func BenchmarkNewReport(parseB *testing.B) {
	parseB.ReportAllocs()
	parseOptions := Options{
		Summary:  "fetch failed with conflict",
		Code:     "fetch_conflict",
		Headline: "Replay failed",
		Path:     "fetch.Replay",
		Runtime:  "mutation replay aborted",
		Next:     "retry later",
		Docs:     "ACTIONABLE_ERRORS.md#fetch-conflict",
	}
	for parseB.Loop() {
		_ = NewReport(parseOptions)
	}
}
