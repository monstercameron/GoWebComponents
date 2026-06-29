package runtime2

import "testing"

// TestParseControlEnvelopeJSONRejectsMalformedJSON verifies malformed control payload JSON fails decode.
func TestParseControlEnvelopeJSONRejectsMalformedJSON(parseT *testing.T) {
	parseValue := []byte(`{"protocol_version":"gwc.parallel.v1","kind":"ready"`)
	if _, parseErr := ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected malformed control envelope JSON to fail")
	}
}

// TestParseControlEnvelopeJSONRejectsNonObjectPayload verifies non-object control payloads fail decode.
func TestParseControlEnvelopeJSONRejectsNonObjectPayload(parseT *testing.T) {
	parseValue := []byte(`[{"protocol_version":"gwc.parallel.v1","kind":"ready"}]`)
	if _, parseErr := ParseControlEnvelopeJSON(parseValue); parseErr == nil {
		parseT.Fatal("expected non-object control envelope payload to fail")
	}
}
