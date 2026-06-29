package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildStructuredCloneUpdateEnvelopeJSONRoundTrips verifies valid update envelopes round-trip through structured-clone transport.
func TestBuildStructuredCloneUpdateEnvelopeJSONRoundTrips(parseT *testing.T) {
	parseEnvelope := runtime2.StructuredCloneUpdateEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		InputVersion:     2,
		Snapshot: runtime2.SnapshotEnvelope{
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Epoch:            1,
			InputVersion:     2,
			Props:            map[string]any{"title": "Orders"},
		},
	}
	parsePayload, parseErr := runtime2.BuildStructuredCloneUpdateEnvelopeJSON(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("BuildStructuredCloneUpdateEnvelopeJSON returned error: %v", parseErr)
	}
	parseDecoded, parseErr := runtime2.ParseStructuredCloneUpdateEnvelopeJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneUpdateEnvelopeJSON returned error: %v", parseErr)
	}
	if parseDecoded.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected input version %d, got %d", parseEnvelope.InputVersion, parseDecoded.InputVersion)
	}
}

// TestParseStructuredCloneUpdateEnvelopeJSONFailsCorruptedPayload verifies corrupted update payloads fail decode.
func TestParseStructuredCloneUpdateEnvelopeJSONFailsCorruptedPayload(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","input_version":2,"snapshot":"bad"}`)
	if _, parseErr := runtime2.ParseStructuredCloneUpdateEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected corrupted update payload to fail decode")
	}
}

// TestParseStructuredCloneUpdateEnvelopeJSONDoesNotApplyStaleOrderingRules verifies stale-version ordering is validated separately from decode.
func TestParseStructuredCloneUpdateEnvelopeJSONDoesNotApplyStaleOrderingRules(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","input_version":1,"snapshot":{"region_instance_id":"region-1","epoch":1,"input_version":1}}`)
	parseDecoded, parseErr := runtime2.ParseStructuredCloneUpdateEnvelopeJSON(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("ParseStructuredCloneUpdateEnvelopeJSON returned error: %v", parseErr)
	}
	if parseDecoded.InputVersion != 1 {
		parseT.Fatalf("expected decoded input version 1, got %d", parseDecoded.InputVersion)
	}
}
