package diagnostics

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicDiagnosticsWrappers(parseT *testing.T) {
	parseReport := NewReport(Options{
		Summary:  "broken",
		Code:     "example_failure",
		Headline: "Exploded",
		Next:     "fix it",
	})
	if parseReport.Code != "example_failure" || parseReport.Summary != "broken" || parseReport.Headline != "Exploded" {
		parseT.Fatalf("NewReport() returned unexpected report: %+v", parseReport)
	}

	parseRecorder := httptest.NewRecorder()
	WriteHTTPError(parseRecorder, 422, parseReport)
	if parseRecorder.Code != 422 {
		parseT.Fatalf("WriteHTTPError() status = %d, want 422", parseRecorder.Code)
	}
	if parseBody := parseRecorder.Body.String(); !strings.Contains(parseBody, "example_failure") || !strings.Contains(parseBody, "broken") || !strings.Contains(parseBody, "fix it") {
		parseT.Fatalf("WriteHTTPError() body missing report details: %s", parseBody)
	}

	Emit(parseReport)
}
