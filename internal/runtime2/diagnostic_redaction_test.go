package runtime2

import (
	"strings"
	"testing"
)

// TestRedactDiagnosticTextRedactsJSONSourceSnapshotAndSecrets verifies JSON diagnostic payloads redact source snapshots and secret fields.
func TestRedactDiagnosticTextRedactsJSONSourceSnapshotAndSecrets(parseT *testing.T) {
	parseDiagnosticText := `{"message":"worker update failed","snapshot":{"props":{"name":"alice","token":"token-123"},"sources":{"cart":"value-42"}},"api_key":"api-xyz","other":"keep"}`
	getRedactedText := RedactDiagnosticText(parseDiagnosticText)
	if strings.Contains(getRedactedText, "alice") ||
		strings.Contains(getRedactedText, "token-123") ||
		strings.Contains(getRedactedText, "value-42") ||
		strings.Contains(getRedactedText, "api-xyz") {
		parseT.Fatalf("expected JSON diagnostic text to redact sensitive values, got %q", getRedactedText)
	}
	if !strings.Contains(getRedactedText, "\"other\":\"keep\"") {
		parseT.Fatalf("expected non-sensitive JSON fields to remain visible, got %q", getRedactedText)
	}
	if !strings.Contains(getRedactedText, "[redacted]") {
		parseT.Fatalf("expected redaction marker in diagnostic text, got %q", getRedactedText)
	}
}

// TestRedactDiagnosticTextRedactsKeyValueSecrets verifies key-value diagnostics redact snapshot-like and secret tokens.
func TestRedactDiagnosticTextRedactsKeyValueSecrets(parseT *testing.T) {
	parseDiagnosticText := `update failed source_values={"theme":"dark"} token=token-123 Authorization=Bearer api-secret`
	getRedactedText := RedactDiagnosticText(parseDiagnosticText)
	if strings.Contains(getRedactedText, "dark") ||
		strings.Contains(getRedactedText, "token-123") ||
		strings.Contains(getRedactedText, "api-secret") {
		parseT.Fatalf("expected key-value diagnostic text to redact sensitive values, got %q", getRedactedText)
	}
	if !strings.Contains(getRedactedText, "source_values=[redacted]") {
		parseT.Fatalf("expected source_values redaction marker, got %q", getRedactedText)
	}
	if !strings.Contains(getRedactedText, "token=[redacted]") {
		parseT.Fatalf("expected token redaction marker, got %q", getRedactedText)
	}
	if !strings.Contains(getRedactedText, "Authorization=[redacted]") {
		parseT.Fatalf("expected authorization redaction marker, got %q", getRedactedText)
	}
}

// TestBuildControlDiagnosticEnvelopeRedactsDiagnosticText verifies diagnostic envelope builders redact source snapshot values.
func TestBuildControlDiagnosticEnvelopeRedactsDiagnosticText(parseT *testing.T) {
	getEnvelope, parseErr := BuildControlDiagnosticEnvelope("region-1", ControlDiagnosticEnvelopeSpec{
		DiagnosticType: DiagnosticEventKindUpdate,
		DiagnosticText: `{"source_values":{"session":"abc-123"},"token":"token-xyz","message":"update failed"}`,
	})
	if parseErr != nil {
		parseT.Fatalf("BuildControlDiagnosticEnvelope returned error: %v", parseErr)
	}
	if strings.Contains(getEnvelope.DiagnosticText, "abc-123") || strings.Contains(getEnvelope.DiagnosticText, "token-xyz") {
		parseT.Fatalf("expected built envelope diagnostic text to redact sensitive values, got %q", getEnvelope.DiagnosticText)
	}
	if !strings.Contains(getEnvelope.DiagnosticText, "[redacted]") {
		parseT.Fatalf("expected built envelope diagnostic text to include redaction marker, got %q", getEnvelope.DiagnosticText)
	}
}

// TestParseControlEnvelopeJSONRedactsDiagnosticText verifies parsed diagnostic envelopes redact source snapshots and secret tokens.
func TestParseControlEnvelopeJSONRedactsDiagnosticText(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"diagnostic","region_instance_id":"region-1","diagnostic_type":"update","diagnostic_text":"source_values={\"theme\":\"dark\"} token=tok-1"}`)
	getEnvelope, parseErr := ParseControlEnvelopeJSON(parseValue)
	if parseErr != nil {
		parseT.Fatalf("ParseControlEnvelopeJSON returned error: %v", parseErr)
	}
	if strings.Contains(getEnvelope.DiagnosticText, "dark") || strings.Contains(getEnvelope.DiagnosticText, "tok-1") {
		parseT.Fatalf("expected parsed envelope diagnostic text to redact sensitive values, got %q", getEnvelope.DiagnosticText)
	}
	if !strings.Contains(getEnvelope.DiagnosticText, "source_values=[redacted]") {
		parseT.Fatalf("expected parsed envelope source_values redaction marker, got %q", getEnvelope.DiagnosticText)
	}
	if !strings.Contains(getEnvelope.DiagnosticText, "token=[redacted]") {
		parseT.Fatalf("expected parsed envelope token redaction marker, got %q", getEnvelope.DiagnosticText)
	}
}

// TestBuildControlEnvelopeJSONRedactsDiagnosticText verifies control-envelope JSON builders redact diagnostic source values.
func TestBuildControlEnvelopeJSONRedactsDiagnosticText(parseT *testing.T) {
	parseEnvelope := ControlEnvelope{
		ProtocolVersion:  ProtocolVersionParallelV1,
		Kind:             ControlKindDiagnostic,
		RegionInstanceID: "region-1",
		DiagnosticType:   string(DiagnosticEventKindUpdate),
		DiagnosticText:   `token=abc-123`,
	}
	getJSONPayload, parseErr := BuildControlEnvelopeJSON(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildControlEnvelopeJSON returned error: %v", parseErr)
	}
	if strings.Contains(string(getJSONPayload), "abc-123") {
		parseT.Fatalf("expected built control JSON to redact token value, got %s", string(getJSONPayload))
	}
	if !strings.Contains(string(getJSONPayload), "token=[redacted]") {
		parseT.Fatalf("expected built control JSON to include token redaction marker, got %s", string(getJSONPayload))
	}
}
