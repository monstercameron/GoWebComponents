package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestParseStructuredCloneMountEnvelopeJSONRejectsNumberWhereStringRequired verifies type guards reject numeric IDs.
func TestParseStructuredCloneMountEnvelopeJSONRejectsNumberWhereStringRequired(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":123,"renderer_id":"dashboard.hot-panel","snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":1}}`)
	if _, parseErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected numeric region_instance_id to fail")
	}
}

// TestParseStructuredCloneMountEnvelopeJSONRejectsMapWhereListRequired verifies type guards reject map values for list fields.
func TestParseStructuredCloneMountEnvelopeJSONRejectsMapWhereListRequired(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","renderer_id":"dashboard.hot-panel","source_ids":{"id":"a"},"snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":1}}`)
	if _, parseErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected source_ids map value to fail")
	}
}

// TestParseStructuredCloneMountEnvelopeJSONRejectsMissingNestedObject verifies required nested snapshot objects are enforced.
func TestParseStructuredCloneMountEnvelopeJSONRejectsMissingNestedObject(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","renderer_id":"dashboard.hot-panel"}`)
	if _, parseErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected missing snapshot object to fail")
	}
}
