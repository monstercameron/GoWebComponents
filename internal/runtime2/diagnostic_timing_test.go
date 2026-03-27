package runtime2

import (
	"strings"
	"testing"
)

// TestValidateDiagnosticTimingMetricsAcceptsNonZeroStage verifies structured timing diagnostics pass when any stage duration is present.
func TestValidateDiagnosticTimingMetricsAcceptsNonZeroStage(parseT *testing.T) {
	parseMetrics := DiagnosticTimingMetrics{
		RenderNanos: 1200,
	}
	if parseErr := ValidateDiagnosticTimingMetrics(parseMetrics); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticTimingMetrics(valid) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticTimingMetricsRejectsAllZeroStages verifies structured timing diagnostics reject empty stage payloads.
func TestValidateDiagnosticTimingMetricsRejectsAllZeroStages(parseT *testing.T) {
	parseErr := ValidateDiagnosticTimingMetrics(DiagnosticTimingMetrics{})
	if parseErr == nil {
		parseT.Fatal("expected all-zero timing metrics to fail")
	}
	if !strings.Contains(parseErr.Error(), "at least one non-zero stage duration") {
		parseT.Fatalf("expected non-zero-stage guidance, got %v", parseErr)
	}
}
