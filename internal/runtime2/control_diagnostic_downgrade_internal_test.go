package runtime2

import "testing"

// TestValidateControlEnvelopeAcceptsDiagnosticDowngrade verifies diagnostics accept structured downgrade reporting.
func TestValidateControlEnvelopeAcceptsDiagnosticDowngrade(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindFallback),
		DiagnosticDowngrade: &DiagnosticDowngradeReason{
			Path:   DiagnosticDowngradePathBinary,
			Reason: "decode-failed",
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(valid diagnostic downgrade) returned error: %v", parseErr)
	}
}

// TestValidateControlEnvelopeRejectsUnsupportedDiagnosticDowngrade verifies diagnostics reject unsupported downgrade reason combinations.
func TestValidateControlEnvelopeRejectsUnsupportedDiagnosticDowngrade(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindFallback),
		DiagnosticDowngrade: &DiagnosticDowngradeReason{
			Path:   DiagnosticDowngradePathBinary,
			Reason: "invalid-shared-page",
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected unsupported diagnostic downgrade reason to fail")
	}
}
