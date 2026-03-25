package diagnostics

import (
	"net/http/httptest"
	"testing"
)

func BenchmarkNewReport(b *testing.B) {
	b.ReportAllocs()
	options := Options{
		Summary:  "fetch failed with conflict",
		Code:     "fetch_conflict",
		Headline: "Replay failed",
		Path:     "fetch.Replay",
		Runtime:  "mutation replay aborted",
		Next:     "retry later",
		Docs:     "ACTIONABLE_ERRORS.md#fetch-conflict",
	}
	for b.Loop() {
		_ = NewReport(options)
	}
}

func BenchmarkWriteHTTPError(b *testing.B) {
	b.ReportAllocs()
	report := NewReport(Options{
		Summary:  "router rejected navigation",
		Code:     "route_blocked",
		Headline: "Navigation blocked",
		Next:     "inspect route guards",
	})
	for b.Loop() {
		recorder := httptest.NewRecorder()
		WriteHTTPError(recorder, 422, report)
	}
}
