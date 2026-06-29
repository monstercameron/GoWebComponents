package runtime2

import "testing"

// TestValidateControlEnvelopeAcceptsStructuredTimingDiagnostic verifies diagnostic envelopes accept structured timing payloads.
func TestValidateControlEnvelopeAcceptsStructuredTimingDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindUpdate),
		DiagnosticTiming: &DiagnosticTimingMetrics{
			QueueNanos:     1200,
			RenderNanos:    3400,
			DiffNanos:      900,
			EncodeNanos:    700,
			TransportNanos: 1100,
			CommitNanos:    800,
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(valid timing diagnostic) returned error: %v", parseErr)
	}
}

// TestValidateControlEnvelopeRejectsZeroStructuredTimingDiagnostic verifies all-zero structured timing payloads are rejected.
func TestValidateControlEnvelopeRejectsZeroStructuredTimingDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindUpdate),
		DiagnosticTiming: &DiagnosticTimingMetrics{},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected all-zero structured timing diagnostic payload to fail")
	}
}
