//go:build !js

package diagnostics

import (
	"net/http/httptest"
	"testing"
)

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
