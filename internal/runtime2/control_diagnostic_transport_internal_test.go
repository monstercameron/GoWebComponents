package runtime2

import "testing"

// TestValidateControlEnvelopeAcceptsDiagnosticTransportTier verifies diagnostics can report a valid transport tier.
func TestValidateControlEnvelopeAcceptsDiagnosticTransportTier(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindPatchReady),
		TransportTier:    TransportTierBinary,
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(valid diagnostic transport tier) returned error: %v", parseErr)
	}
}

// TestValidateControlEnvelopeRejectsDiagnosticTransportTier verifies diagnostics reject unsupported transport tier values.
func TestValidateControlEnvelopeRejectsDiagnosticTransportTier(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindPatchReady),
		TransportTier:    TransportTier("tape-drive"),
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected unsupported diagnostic transport tier to fail")
	}
}
