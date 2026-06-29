package runtime2

import (
	"slices"
	"testing"
)

// TestParseDiagnosticEventKindAcceptsStableKinds verifies each stable lifecycle diagnostic kind parses successfully.
func TestParseDiagnosticEventKindAcceptsStableKinds(parseT *testing.T) {
	for _, parseKind := range GetDiagnosticEventKinds() {
		if _, parseErr := ParseDiagnosticEventKind(string(parseKind)); parseErr != nil {
			parseT.Fatalf("ParseDiagnosticEventKind(%q) returned error: %v", parseKind, parseErr)
		}
	}
}

// TestParseDiagnosticEventKindRejectsUnsupportedKind verifies unknown lifecycle diagnostic kinds fail validation.
func TestParseDiagnosticEventKindRejectsUnsupportedKind(parseT *testing.T) {
	if _, parseErr := ParseDiagnosticEventKind("timing"); parseErr == nil {
		parseT.Fatal("expected unsupported diagnostic event kind to fail")
	}
}

// TestGetDiagnosticEventKindsReturnsExpectedStableOrder verifies stable lifecycle diagnostic kind ordering remains deterministic.
func TestGetDiagnosticEventKindsReturnsExpectedStableOrder(parseT *testing.T) {
	parseKinds := GetDiagnosticEventKinds()
	parseExpectedKinds := []DiagnosticEventKind{
		DiagnosticEventKindMount,
		DiagnosticEventKindUpdate,
		DiagnosticEventKindCancel,
		DiagnosticEventKindDispose,
		DiagnosticEventKindRestart,
		DiagnosticEventKindPatchReady,
		DiagnosticEventKindFallback,
		DiagnosticEventKindRepair,
	}
	if !slices.Equal(parseKinds, parseExpectedKinds) {
		parseT.Fatalf("GetDiagnosticEventKinds() = %+v, want %+v", parseKinds, parseExpectedKinds)
	}
}
