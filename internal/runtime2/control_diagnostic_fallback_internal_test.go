package runtime2

import "testing"

// TestValidateControlEnvelopeAcceptsStructuredFallbackDiagnostic verifies diagnostic envelopes accept structured fallback reasons.
func TestValidateControlEnvelopeAcceptsStructuredFallbackDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindFallback),
		DiagnosticFallback: &DiagnosticFallbackReason{
			Domain: "transport",
			Reason: string(TransportFailureKindMalformedPatchPayload),
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(valid fallback diagnostic) returned error: %v", parseErr)
	}
}

// TestValidateControlEnvelopeRejectsUnsupportedFallbackDiagnostic verifies unsupported fallback reasons fail validation.
func TestValidateControlEnvelopeRejectsUnsupportedFallbackDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindFallback),
		DiagnosticFallback: &DiagnosticFallbackReason{
			Domain: "dom",
			Reason: string(TransportFailureKindMalformedPatchPayload),
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected unsupported fallback diagnostic reason to fail")
	}
}
