package runtime2

import "testing"

// FuzzParseControlEnvelopeJSON fuzzes control-plane envelope decoding and validation stability.
func FuzzParseControlEnvelopeJSON(parseF *testing.F) {
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"ready"}`))
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"capabilities","capabilities":{"HasWorkerSupport":true,"HasStructuredCloneSupport":true}}`))
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"mount","region_instance_id":"region-1","renderer_id":"dashboard.hot-panel","snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":1}}`))
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"update","region_instance_id":"region-1","input_version":2,"snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":2}}`))
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"patch-ready","region_instance_id":"region-1","patch_version":2,"input_version":2,"transport_tier":"structured-clone"}`))
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"diagnostic","region_instance_id":"region-1","diagnostic_type":"update","diagnostic_text":"ok"}`))
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"restart","region_instance_id":"region-1","epoch":2}`))
	parseF.Add([]byte(`{"protocol_version":"gwc.parallel.v1","kind":"restart","region_instance_id":"region-1"`))
	parseF.Fuzz(func(parseT *testing.T, parsePayload []byte) {
		parseEnvelope, parseErr := ParseControlEnvelopeJSON(parsePayload)
		if parseErr != nil {
			return
		}
		if parseValidateErr := ValidateControlEnvelope(parseEnvelope); parseValidateErr != nil {
			parseT.Fatalf("ValidateControlEnvelope(fuzz decoded envelope) returned error: %v", parseValidateErr)
		}
		if _, parseBuildErr := BuildControlEnvelopeJSON(parseEnvelope); parseBuildErr != nil {
			parseT.Fatalf("BuildControlEnvelopeJSON(fuzz decoded envelope) returned error: %v", parseBuildErr)
		}
	})
}
