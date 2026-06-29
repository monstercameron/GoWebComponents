package runtime2

import "testing"

// TestValidateControlEnvelopeAcceptsDiagnosticShardID verifies diagnostic envelopes accept normalized shard IDs.
func TestValidateControlEnvelopeAcceptsDiagnosticShardID(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:   ProtocolVersionParallelV1,
		Kind:              ControlKindDiagnostic,
		RegionInstanceID:  RegionInstanceID("region-1"),
		DiagnosticType:    string(DiagnosticEventKindUpdate),
		DiagnosticShardID: SchedulerShardID("shard-2"),
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(valid shard diagnostic) returned error: %v", parseErr)
	}
}

// TestValidateControlEnvelopeRejectsInvalidDiagnosticShardID verifies diagnostic envelopes reject malformed shard IDs.
func TestValidateControlEnvelopeRejectsInvalidDiagnosticShardID(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:   ProtocolVersionParallelV1,
		Kind:              ControlKindDiagnostic,
		RegionInstanceID:  RegionInstanceID("region-1"),
		DiagnosticType:    string(DiagnosticEventKindUpdate),
		DiagnosticShardID: SchedulerShardID(" shard-2 "),
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected malformed diagnostic shard ID to fail")
	}
}
