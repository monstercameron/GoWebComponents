package runtime

import (
	"strings"
	"testing"
)

// getParityDocs maps the pre-registry docs reference to the registry's. The old
// anchors targeted a file that never existed; the registry points at the
// generated reference page instead, anchored on the lowercase code.
func getParityDocs(parseOldDocs string, parseCode string) string {
	const parsePage = "docs/REFERENCE_MANUAL/error-codes.md"
	if parseOldDocs == "" {
		return ""
	}
	if parseAnchor, parseOk := strings.CutPrefix(parseOldDocs, "ACTIONABLE_ERRORS.md#"); parseOk {
		return parsePage + "#" + parseAnchor
	}
	// The unanchored and TROUBLESHOOTING.md references became anchored too.
	return parsePage + "#" + strings.ToLower(parseCode)
}

// TestDiagnosticMetadataMatchesPreRegistryTable proves the registry-backed
// lookup returns exactly what the old substring switch returned, for every
// message fragment across every severity and classification.
func TestDiagnosticMetadataMatchesPreRegistryTable(parseT *testing.T) {
	if len(goldenDiagnosticRows) < 300 {
		parseT.Fatalf("golden table shrank to %d rows", len(goldenDiagnosticRows))
	}
	for _, parseRow := range goldenDiagnosticRows {
		parseGot := diagnosticMetadata("runtime", parseRow.Severity, parseRow.Class, parseRow.Message)
		if parseGot.Code != parseRow.Code ||
			parseGot.Remediation != parseRow.Remediation ||
			parseGot.Recoverable != parseRow.Recoverable ||
			parseGot.Docs != getParityDocs(parseRow.Docs, parseRow.Code) {
			parseT.Fatalf("message %q (%s/%s): got %+v, want code=%q docs=%q rem=%q recoverable=%t",
				parseRow.Message, parseRow.Severity, parseRow.Class, parseGot,
				parseRow.Code, getParityDocs(parseRow.Docs, parseRow.Code), parseRow.Remediation, parseRow.Recoverable)
		}
	}
}

// TestPanicPhaseDiagnosticsMatchPreRegistryTable does the same for the
// phase-keyed helpers, including the unknown-phase fallback.
func TestPanicPhaseDiagnosticsMatchPreRegistryTable(parseT *testing.T) {
	for _, parseRow := range goldenPhaseRows {
		if parseGot := panicDiagnosticCode(parseRow.Phase); parseGot != parseRow.Code {
			parseT.Fatalf("phase %q code = %q, want %q", parseRow.Phase, parseGot, parseRow.Code)
		}
		if parseGot, parseWant := panicDiagnosticDocs(parseRow.Phase), getParityDocs(parseRow.Docs, parseRow.Code); parseGot != parseWant {
			parseT.Fatalf("phase %q docs = %q, want %q", parseRow.Phase, parseGot, parseWant)
		}
		if parseGot := panicDiagnosticRemediation(parseRow.Phase); parseGot != parseRow.Remediation {
			parseT.Fatalf("phase %q remediation = %q, want %q", parseRow.Phase, parseGot, parseRow.Remediation)
		}
	}
}

// TestActionablePanicPrefersExplicitCode checks the direct-code path and the
// fallbacks around it.
func TestActionablePanicPrefersExplicitCode(parseT *testing.T) {
	parseOut := ActionableFrameworkPanic(ActionablePanicOptions{Source: "runtime", Subject: "X", Code: "GWC-ROUTER-REDIRECT-LOOP"})
	if !strings.Contains(parseOut, "GWC-ROUTER-REDIRECT-LOOP") || !strings.Contains(parseOut, "Route redirect loop") {
		parseT.Fatalf("explicit code not honoured: %q", parseOut)
	}
	parseOut = ActionableFrameworkPanic(ActionablePanicOptions{Source: "runtime", Subject: "X", Message: "route loader failed: x", Code: "GWC-NOT-REGISTERED"})
	if !strings.Contains(parseOut, "GWC-ROUTER-LOADER-FAILED") {
		parseT.Fatalf("unknown code should fall back to the substring matcher: %q", parseOut)
	}
	parseOut = ActionableFrameworkPanic(ActionablePanicOptions{Source: "runtime", Subject: "X", Message: "no match here"})
	if !strings.Contains(parseOut, "GWC-FRAMEWORK-USAGE") {
		parseT.Fatalf("unmatched message should use GWC-FRAMEWORK-USAGE: %q", parseOut)
	}
}
