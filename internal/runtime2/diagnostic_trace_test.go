package runtime2

import (
	"strings"
	"testing"
)

// TestValidateDiagnosticTraceMetadataAcceptsDebugTrace verifies debug trace metadata passes when all cross-attempt IDs are present.
func TestValidateDiagnosticTraceMetadataAcceptsDebugTrace(parseT *testing.T) {
	parseMetadata := DiagnosticTraceMetadata{
		IsDebug:            true,
		TraceID:            "trace-123",
		SchedulerAttemptID: 11,
		WorkerAttemptID:    14,
		CommitAttemptID:    19,
	}
	if parseErr := ValidateDiagnosticTraceMetadata(parseMetadata); parseErr != nil {
		parseT.Fatalf("ValidateDiagnosticTraceMetadata(valid) returned error: %v", parseErr)
	}
}

// TestValidateDiagnosticTraceMetadataRejectsNonDebugPayload verifies trace metadata enforces explicit debug-only opt-in.
func TestValidateDiagnosticTraceMetadataRejectsNonDebugPayload(parseT *testing.T) {
	parseMetadata := DiagnosticTraceMetadata{
		TraceID:            "trace-123",
		SchedulerAttemptID: 11,
		WorkerAttemptID:    14,
		CommitAttemptID:    19,
	}
	parseErr := ValidateDiagnosticTraceMetadata(parseMetadata)
	if parseErr == nil {
		parseT.Fatal("expected non-debug trace metadata to fail")
	}
	if !strings.Contains(parseErr.Error(), "debug-only") {
		parseT.Fatalf("expected debug-only validation guidance, got %v", parseErr)
	}
}

// TestValidateDiagnosticTraceMetadataRejectsMissingAttemptIDs verifies trace metadata requires scheduler, worker, and commit attempt IDs.
func TestValidateDiagnosticTraceMetadataRejectsMissingAttemptIDs(parseT *testing.T) {
	parseMetadata := DiagnosticTraceMetadata{
		IsDebug: true,
		TraceID: "trace-123",
	}
	parseErr := ValidateDiagnosticTraceMetadata(parseMetadata)
	if parseErr == nil {
		parseT.Fatal("expected missing attempt IDs to fail")
	}
	if !strings.Contains(parseErr.Error(), "attempt ID is required") {
		parseT.Fatalf("expected missing-attempt validation guidance, got %v", parseErr)
	}
}

// TestValidateDiagnosticTraceMetadataRejectsWhitespaceAndSpecificMissingIDs verifies trace metadata rejects surrounding whitespace and each remaining required attempt identifier.
func TestValidateDiagnosticTraceMetadataRejectsWhitespaceAndSpecificMissingIDs(parseT *testing.T) {
	if parseErr := ValidateDiagnosticTraceMetadata(DiagnosticTraceMetadata{
		IsDebug:            true,
		TraceID:            " trace-123 ",
		SchedulerAttemptID: 11,
		WorkerAttemptID:    14,
		CommitAttemptID:    19,
	}); parseErr == nil {
		parseT.Fatal("expected whitespace-padded trace ID to fail")
	}

	parseCases := []DiagnosticTraceMetadata{
		{
			IsDebug:            true,
			TraceID:            "trace-123",
			SchedulerAttemptID: 11,
			CommitAttemptID:    19,
		},
		{
			IsDebug:            true,
			TraceID:            "trace-123",
			SchedulerAttemptID: 11,
			WorkerAttemptID:    14,
		},
	}
	for _, parseCase := range parseCases {
		if parseErr := ValidateDiagnosticTraceMetadata(parseCase); parseErr == nil {
			parseT.Fatalf("expected incomplete trace metadata to fail for %+v", parseCase)
		}
	}
}
