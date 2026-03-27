package runtime2

import "testing"

// FuzzParseStructuredCloneSnapshotEnvelopeJSON fuzzes structured-clone snapshot decode to ensure malformed payloads fail safely and successful decodes remain valid.
func FuzzParseStructuredCloneSnapshotEnvelopeJSON(parseF *testing.F) {
	parseF.Add([]byte(`{"region_instance_id":"region-1","epoch":1,"input_version":1}`))
	parseF.Add([]byte(`{"region_instance_id":"region-2","epoch":3,"input_version":9,"props":{"title":"Orders"},"sources":{"status":"ok"}}`))
	parseF.Add([]byte(`{"region_instance_id":"region-1","epoch":"bad","input_version":2}`))
	parseF.Add([]byte(`{"region_instance_id":"region-1","epoch":1,"input_version":2`))
	parseF.Fuzz(func(parseT *testing.T, parsePayload []byte) {
		parseEnvelope, parseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload)
		if parseErr != nil {
			return
		}
		if parseValidateErr := ValidateSnapshotEnvelope(parseEnvelope); parseValidateErr != nil {
			parseT.Fatalf("ValidateSnapshotEnvelope(fuzz decoded envelope) returned error: %v", parseValidateErr)
		}
		parseRoundTripPayload, parseRoundTripBuildErr := BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
		if parseRoundTripBuildErr != nil {
			parseT.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON(fuzz decoded envelope) returned error: %v", parseRoundTripBuildErr)
		}
		if _, parseRoundTripParseErr := ParseStructuredCloneSnapshotEnvelopeJSON(parseRoundTripPayload); parseRoundTripParseErr != nil {
			parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON(round-trip) returned error: %v", parseRoundTripParseErr)
		}
	})
}
