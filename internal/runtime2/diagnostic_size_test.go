package runtime2

import (
	"strings"
	"testing"
)

// TestValidateDiagnosticSizeMetricsAcceptsNonZeroMetric verifies structured size diagnostics pass when any metric is non-zero.
func TestValidateDiagnosticSizeMetricsAcceptsNonZeroMetric(parseT *testing.T) {
	parseMetrics := DiagnosticSizeMetrics{
		PatchBytes: 512,
	}
	if parseErr := ValidateDiagnosticSizeMetrics(parseMetrics); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticSizeMetrics(valid) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticSizeMetricsRejectsAllZeroMetrics verifies structured size diagnostics reject empty payloads.
func TestValidateDiagnosticSizeMetricsRejectsAllZeroMetrics(parseT *testing.T) {
	parseErr := ValidateDiagnosticSizeMetrics(DiagnosticSizeMetrics{})
	if parseErr == nil {
		parseT.Fatal("expected all-zero size metrics to fail")
	}
	if !strings.Contains(parseErr.Error(), "at least one non-zero metric") {
		parseT.Fatalf("expected non-zero-metric guidance, got %v", parseErr)
	}
}
