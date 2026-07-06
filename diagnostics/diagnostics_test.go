//go:build !js

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

// TestFormattedPublicExcludesStackAndPaths pins that the client-safe rendering
// carries only the app-authored fields — never the stack frames, file paths, or
// runtime detail from debug.Stack() (which embed absolute build paths that
// disclose the OS username and server filesystem layout). This is the building
// block for the #60 gated-verbose hardening: WriteHTTPError can switch to it on a
// public error path without leaking internals, while Formatted() stays for logs.
func TestFormattedPublicExcludesStackAndPaths(parseT *testing.T) {
	parseReport := NewReport(Options{
		Summary:  "request failed",
		Code:     "req_failure",
		Headline: "Request Failed",
		Next:     "retry shortly",
		Docs:     "ERRORS.md#req",
	})

	// Sanity: the full rendering DOES capture stack frames (with the framework
	// module path), so the exclusions below are meaningful, not vacuous.
	if parseFull := parseReport.Formatted(); !strings.Contains(parseFull, "github.com/monstercameron/GoWebComponents") {
		parseT.Fatalf("expected the full report to contain stack frames, got: %s", parseFull)
	}

	parsePublic := parseReport.FormattedPublic()

	// Client-safe fields must be present.
	for _, parseWant := range []string{"req_failure", "request failed", "retry shortly", "ERRORS.md#req"} {
		if !strings.Contains(parsePublic, parseWant) {
			parseT.Fatalf("public rendering missing safe field %q: %s", parseWant, parsePublic)
		}
	}
	// Disclosure vectors must be absent.
	for _, parseLeak := range []string{
		"github.com/monstercameron/GoWebComponents", // stack frame package paths
		"stack:", "where:", "path:", "runtime:", ".go:", // Formatted()-only labels/locations
	} {
		if strings.Contains(parsePublic, parseLeak) {
			parseT.Fatalf("public rendering leaked %q: %s", parseLeak, parsePublic)
		}
	}
}
