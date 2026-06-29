package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildStructuredCloneSnapshotEnvelopeJSONRoundTrips verifies valid snapshot envelopes round-trip through structured-clone encoding.
func TestBuildStructuredCloneSnapshotEnvelopeJSONRoundTrips(parseT *testing.T) {
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		SourceVersion:    7,
		Props:            map[string]any{"title": "Orders"},
		Sources:          map[string]any{"status": "healthy"},
	}
	parsePayload, parseErr := runtime2.BuildStructuredCloneSnapshotEnvelopeJSON(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
	}
	if parseDecoded.RegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("expected region instance ID %q, got %q", parseEnvelope.RegionInstanceID, parseDecoded.RegionInstanceID)
	}
	if parseDecoded.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected input version %d, got %d", parseEnvelope.InputVersion, parseDecoded.InputVersion)
	}
}

// TestParseStructuredCloneSnapshotEnvelopeJSONFailsMalformedEnvelope verifies malformed payloads fail decode.
func TestParseStructuredCloneSnapshotEnvelopeJSONFailsMalformedEnvelope(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","epoch":"bad","input_version":2}`)
	if _, parseErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected malformed snapshot payload to fail decode")
	}
}

// TestParseStructuredCloneSnapshotEnvelopeJSONDefaultsOptionalFields verifies omitted optional fields decode to stable zero values.
func TestParseStructuredCloneSnapshotEnvelopeJSONDefaultsOptionalFields(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","epoch":1,"input_version":2}`)
	parseDecoded, parseErr := runtime2.ParseStructuredCloneSnapshotEnvelopeJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseErr)
	}
	if parseDecoded.SourceVersion != 0 {
		parseT.Fatalf("expected source version default 0, got %d", parseDecoded.SourceVersion)
	}
	if parseDecoded.Props != nil {
		parseT.Fatalf("expected props default nil, got %#v", parseDecoded.Props)
	}
	if parseDecoded.Sources != nil {
		parseT.Fatalf("expected sources default nil, got %#v", parseDecoded.Sources)
	}
}
