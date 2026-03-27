package runtime2

import "testing"

// TestValidateControlEnvelopeAcceptsDebugTraceDiagnostic verifies diagnostic envelopes accept debug-only cross-attempt trace metadata.
func TestValidateControlEnvelopeAcceptsDebugTraceDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindPatchReady),
		DiagnosticTrace: &DiagnosticTraceMetadata{
			IsDebug:            true,
			TraceID:            "trace-123",
			SchedulerAttemptID: 5,
			WorkerAttemptID:    7,
			CommitAttemptID:    9,
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr != nil {
		parseT.Fatalf("ValidateControlEnvelope(valid trace diagnostic) returned error: %v", parseErr)
	}
}

// TestValidateControlEnvelopeRejectsNonDebugTraceDiagnostic verifies trace metadata without explicit debug opt-in is rejected.
func TestValidateControlEnvelopeRejectsNonDebugTraceDiagnostic(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: RegionInstanceID("region-1"),
		DiagnosticType:   string(DiagnosticEventKindPatchReady),
		DiagnosticTrace: &DiagnosticTraceMetadata{
			TraceID:            "trace-123",
			SchedulerAttemptID: 5,
			WorkerAttemptID:    7,
			CommitAttemptID:    9,
		},
	}
	if parseErr := ValidateControlEnvelope(parseEnvelope); parseErr == nil {
		parseT.Fatal("expected non-debug trace diagnostic metadata to fail")
	}
}
