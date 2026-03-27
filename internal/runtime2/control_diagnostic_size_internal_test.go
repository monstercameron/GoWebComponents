package runtime2

import "testing"

// TestValidateControlEnvelopeAcceptsStructuredSizeDiagnostic verifies diagnostic envelopes accept structured size payloads.
func TestValidateControlEnvelopeAcceptsStructuredSizeDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindPatchReady),
		DiagnosticSize: &DiagnosticSizeMetrics{
			SnapshotBytes:   2048,
			IRBytes:         1536,
			PatchBytes:      640,
			SharedPageBytes: 4096,
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(valid size diagnostic) returned error: %v", parseErr)
	}
}

// TestValidateControlEnvelopeRejectsZeroStructuredSizeDiagnostic verifies all-zero structured size payloads are rejected.
func TestValidateControlEnvelopeRejectsZeroStructuredSizeDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindPatchReady),
		DiagnosticSize:   &DiagnosticSizeMetrics{},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected all-zero structured size diagnostic payload to fail")
	}
}
