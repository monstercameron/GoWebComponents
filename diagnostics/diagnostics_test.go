package diagnostics

import (
	"net/http/httptest"
	"strings"
	"testing"
)

func TestPublicDiagnosticsWrappers(t *testing.T) {
	report := NewReport(Options{
		Summary:  "broken",
		Code:     "example_failure",
		Headline: "Exploded",
		Next:     "fix it",
	})
	if report.Code != "example_failure" || report.Summary != "broken" || report.Headline != "Exploded" {
		t.Fatalf("NewReport() returned unexpected report: %+v", report)
	}

	recorder := httptest.NewRecorder()
	WriteHTTPError(recorder, 422, report)
	if recorder.Code != 422 {
		t.Fatalf("WriteHTTPError() status = %d, want 422", recorder.Code)
	}
	if body := recorder.Body.String(); !strings.Contains(body, "example_failure") || !strings.Contains(body, "broken") || !strings.Contains(body, "fix it") {
		t.Fatalf("WriteHTTPError() body missing report details: %s", body)
	}

	Emit(report)
}
