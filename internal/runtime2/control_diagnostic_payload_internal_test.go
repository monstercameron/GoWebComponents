package runtime2

import "testing"

// TestParseControlEnvelopeJSONAcceptsDiagnosticEnvelopeWithStructuredPayloadFields verifies diagnostic payloads can carry timing, size, fallback, and trace fields together.
func TestParseControlEnvelopeJSONAcceptsDiagnosticEnvelopeWithStructuredPayloadFields(parseT *testing.T) {
	parseValue := []byte(`{
		"protocol_version":"gwc.parallel.v1",
		"kind":"diagnostic",
		"region_instance_id":"region-1",
		"diagnostic_type":"fallback",
		"diagnostic_text":"worker fallback",
		"diagnostic_timing":{"queue_ns":100,"render_ns":200},
		"diagnostic_size":{"snapshot_bytes":2048,"patch_bytes":256},
		"diagnostic_fallback":{"domain":"worker-death","reason":"worker-death-no-reassign"},
		"diagnostic_trace":{"is_debug":true,"trace_id":"trace-123","scheduler_attempt_id":1,"worker_attempt_id":2,"commit_attempt_id":3}
	}`)
	getEnvelope, parseErr := ParseControlEnvelopeJSON(parseValue)
	if parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON(valid diagnostic payload fields) returned error: %v", parseErr)
	}
	if getEnvelope.DiagnosticTiming == nil {
		parseT.Fatal("expected parsed diagnostic timing payload")
	}
	if getEnvelope.DiagnosticSize == nil {
		parseT.Fatal("expected parsed diagnostic size payload")
	}
	if getEnvelope.DiagnosticFallback == nil {
		parseT.Fatal("expected parsed diagnostic fallback payload")
	}
	if getEnvelope.DiagnosticTrace == nil {
		parseT.Fatal("expected parsed diagnostic trace payload")
	}
	if getEnvelope.DiagnosticTrace.TraceID != "trace-123" {
		parseT.Fatalf("expected parsed trace_id trace-123, got %q", getEnvelope.DiagnosticTrace.TraceID)
	}
	if getEnvelope.DiagnosticTrace.SchedulerAttemptID != 1 ||
		getEnvelope.DiagnosticTrace.WorkerAttemptID != 2 ||
		getEnvelope.DiagnosticTrace.CommitAttemptID != 3 {
		parseT.Fatalf(
			"expected parsed trace attempt IDs 1/2/3, got %d/%d/%d",
			getEnvelope.DiagnosticTrace.SchedulerAttemptID,
			getEnvelope.DiagnosticTrace.WorkerAttemptID,
			getEnvelope.DiagnosticTrace.CommitAttemptID,
		)
	}
}
