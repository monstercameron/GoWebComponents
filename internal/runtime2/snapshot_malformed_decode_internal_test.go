package runtime2

import "testing"

// TestParseStructuredCloneSnapshotEnvelopeJSONRejectsMalformedJSON verifies malformed snapshot payload JSON fails decode.
func TestParseStructuredCloneSnapshotEnvelopeJSONRejectsMalformedJSON(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","epoch":1,"input_version":2`)
	if _, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected malformed snapshot envelope JSON to fail")
	}
}

// TestParseStructuredCloneSnapshotEnvelopeJSONRejectsNonObjectPayload verifies non-object snapshot payloads fail decode.
func TestParseStructuredCloneSnapshotEnvelopeJSONRejectsNonObjectPayload(parseT *testing.T) {
	parsePayload := []byte(`[{"region_instance_id":"region-1","epoch":1,"input_version":2}]`)
	if _, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected non-object snapshot envelope payload to fail")
	}
}
