package diagnostics

import (
	"net/http/httptest"
	"testing"
)

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

func BenchmarkWriteHTTPError(parseB *testing.B) {
	parseB.ReportAllocs()
	parseReport := NewReport(Options{
		Summary:  "router rejected navigation",
		Code:     "route_blocked",
		Headline: "Navigation blocked",
		Next:     "inspect route guards",
	})
	for parseB.Loop() {
		parseRecorder := httptest.NewRecorder()
		WriteHTTPError(parseRecorder, 422, parseReport)
	}
}
