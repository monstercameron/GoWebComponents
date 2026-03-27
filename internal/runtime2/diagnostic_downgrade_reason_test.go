package runtime2

import (
	"strings"
	"testing"
)

// TestValidateDiagnosticDowngradeReasonAcceptsSharedMemoryReasons verifies shared-memory downgrade diagnostics accept known reason values.
func TestValidateDiagnosticDowngradeReasonAcceptsSharedMemoryReasons(parseT *testing.T) {
	parseDowngradeReason := DiagnosticDowngradeReason{
		Path:   DiagnosticDowngradePathSharedMemory,
		Reason: "invalid-shared-page",
	}
	if parseErr := ValidateDiagnosticDowngradeReason(parseDowngradeReason); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticDowngradeReason(shared-memory) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticDowngradeReasonAcceptsBinaryReasons verifies binary downgrade diagnostics accept known reason values.
func TestValidateDiagnosticDowngradeReasonAcceptsBinaryReasons(parseT *testing.T) {
	parseDowngradeReason := DiagnosticDowngradeReason{
		Path:   DiagnosticDowngradePathBinary,
		Reason: "decode-failed",
	}
	if parseErr := ValidateDiagnosticDowngradeReason(parseDowngradeReason); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticDowngradeReason(binary) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticDowngradeReasonRejectsUnsupportedPath verifies unknown downgrade paths fail validation.
func TestValidateDiagnosticDowngradeReasonRejectsUnsupportedPath(parseT *testing.T) {
	parseErr := ValidateDiagnosticDowngradeReason(DiagnosticDowngradeReason{
		Path:   DiagnosticDowngradePath("structured-clone"),
		Reason: "encode-failed",
	})
	if parseErr == nil {
		parseT.Fatal("expected unsupported downgrade path to fail")
	}
	if !strings.Contains(parseErr.Error(), "path") {
		parseT.Fatalf("expected path validation guidance, got %v", parseErr)
	}
}

// TestValidateDiagnosticDowngradeReasonRejectsUnsupportedReason verifies unknown reasons fail within one supported path.
func TestValidateDiagnosticDowngradeReasonRejectsUnsupportedReason(parseT *testing.T) {
	parseErr := ValidateDiagnosticDowngradeReason(DiagnosticDowngradeReason{
		Path:   DiagnosticDowngradePathBinary,
		Reason: "invalid-shared-page",
	})
	if parseErr == nil {
		parseT.Fatal("expected unsupported binary downgrade reason to fail")
	}
	if !strings.Contains(parseErr.Error(), "unsupported") {
		parseT.Fatalf("expected reason validation guidance, got %v", parseErr)
	}
}
