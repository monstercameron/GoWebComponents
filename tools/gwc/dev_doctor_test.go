package main

import (
	"strings"
	"testing"
)

// TestFormatEnvironmentDiagnosisHealthyIsSilent confirms a passing doctor report
// produces no diagnosis, so a dev failure unrelated to the environment does not
// spam the developer.
func TestFormatEnvironmentDiagnosisHealthyIsSilent(parseT *testing.T) {
	parseReport := doctorReport{
		OK: true,
		Checks: []doctorCheck{
			{Name: "Go toolchain", Status: "pass", Summary: "go1.25 found"},
		},
	}
	if parseSummary, parseShow := formatEnvironmentDiagnosis(parseReport); parseShow || parseSummary != "" {
		parseT.Fatalf("expected silence on a healthy report, got show=%v summary=%q", parseShow, parseSummary)
	}
}

// TestFormatEnvironmentDiagnosisListsFailingChecks confirms blocking checks are
// surfaced with their fix hint, while non-blocking warnings are not listed.
func TestFormatEnvironmentDiagnosisListsFailingChecks(parseT *testing.T) {
	parseReport := doctorReport{
		OK: false,
		Checks: []doctorCheck{
			{Name: "Go toolchain", Status: "fail", Summary: "go1.21 is too old", Hint: "install Go 1.25+"},
			{Name: "Port 8080", Status: "warn", Summary: "in use"},
			{Name: "wasm_exec.js", Status: "fail", Summary: "missing", Remediation: "reinstall the Go toolchain"},
		},
	}
	parseSummary, parseShow := formatEnvironmentDiagnosis(parseReport)
	if !parseShow {
		parseT.Fatal("expected a diagnosis to be shown for a failing report")
	}
	for _, parseWant := range []string{"Go toolchain", "install Go 1.25+", "wasm_exec.js", "reinstall the Go toolchain"} {
		if !strings.Contains(parseSummary, parseWant) {
			parseT.Fatalf("diagnosis missing %q:\n%s", parseWant, parseSummary)
		}
	}
	if strings.Contains(parseSummary, "Port 8080") {
		parseT.Fatalf("non-blocking warn check should not be listed:\n%s", parseSummary)
	}
}

// TestFormatEnvironmentDiagnosisWarnOnlyIsSilent confirms that a not-OK report
// whose only non-passing checks are warnings produces no blocking diagnosis.
func TestFormatEnvironmentDiagnosisWarnOnlyIsSilent(parseT *testing.T) {
	parseReport := doctorReport{
		OK: false,
		Checks: []doctorCheck{
			{Name: "Port 8080", Status: "warn", Summary: "in use"},
		},
	}
	if _, parseShow := formatEnvironmentDiagnosis(parseReport); parseShow {
		parseT.Fatal("expected silence when only warnings are present")
	}
}
