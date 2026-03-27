package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildStructuredCloneMountEnvelopeJSONRoundTrips verifies valid mount envelopes round-trip through structured-clone transport.
func TestBuildStructuredCloneMountEnvelopeJSONRoundTrips(parseT *testing.T) {
	parseEnvelope := runtime2.StructuredCloneMountEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Snapshot: runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     1,
			Props:            map[string]any{"title": "Orders"},
		},
	}
	parsePayload, parseErr := runtime2.BuildStructuredCloneMountEnvelopeJSON(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneMountEnvelopeJSON returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneMountEnvelopeJSON returned error: %v", parseErr)
	}
	if parseDecoded.RendererID != parseEnvelope.RendererID {
		parseT.Fatalf("expected renderer ID %q, got %q", parseEnvelope.RendererID, parseDecoded.RendererID)
	}
}

// TestParseStructuredCloneMountEnvelopeJSONFailsMissingRendererID verifies mount envelopes require renderer IDs.
func TestParseStructuredCloneMountEnvelopeJSONFailsMissingRendererID(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":1}}`)
	if _, parseErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected missing renderer ID to fail")
	}
}

// TestParseStructuredCloneMountEnvelopeJSONAllowsMissingProps verifies snapshot props remain optional for mount transport.
func TestParseStructuredCloneMountEnvelopeJSONAllowsMissingProps(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","renderer_id":"dashboard.hot-panel","snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":1}}`)
	parseDecoded, parseErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneMountEnvelopeJSON returned error: %v", parseErr)
	}
	if parseDecoded.Snapshot.Props != nil {
		parseT.Fatalf("expected optional props to decode as nil, got %#v", parseDecoded.Snapshot.Props)
	}
}
