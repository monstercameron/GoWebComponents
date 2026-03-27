package runtime2_test

import (
	"bytes"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestBuildStructuredClonePatchEnvelopeJSONRoundTrips verifies structured-clone patch envelopes round-trip through JSON encode/decode.
func TestBuildStructuredClonePatchEnvelopeJSONRoundTrips(parseT *testing.T) {
	parseEnvelope := runtime2.StructuredClonePatchEnvelope{
		RegionInstanceID: "region-1",
		Epoch:            2,
		PatchVersion:     7,
		PatchPayload:     []byte("patch-bytes"),
	}
	parsePayload, parsePayloadErr := runtime2.BuildStructuredClonePatchEnvelopeJSON(parseEnvelope)
	if parsePayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchEnvelopeJSON returned error: %v", parsePayloadErr)
	}
	parseDecodedEnvelope, parseDecodedEnvelopeErr := runtime2.ParseStructuredClonePatchEnvelopeJSON(parsePayload)
	if parseDecodedEnvelopeErr != nil {
		parseT.Fatalf("ParseStructuredClonePatchEnvelopeJSON returned error: %v", parseDecodedEnvelopeErr)
	}
	if parseDecodedEnvelope.RegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("expected region instance ID %q, got %q", parseEnvelope.RegionInstanceID, parseDecodedEnvelope.RegionInstanceID)
	}
	if parseDecodedEnvelope.PatchVersion != parseEnvelope.PatchVersion {
		parseT.Fatalf("expected patch version %d, got %d", parseEnvelope.PatchVersion, parseDecodedEnvelope.PatchVersion)
	}
	if !bytes.Equal(parseDecodedEnvelope.PatchPayload, parseEnvelope.PatchPayload) {
		parseT.Fatalf("expected patch payload %q, got %q", string(parseEnvelope.PatchPayload), string(parseDecodedEnvelope.PatchPayload))
	}
}

// TestParseStructuredClonePatchEnvelopeJSONRejectsMalformedPayload verifies malformed structured-clone patch envelopes fail decode.
func TestParseStructuredClonePatchEnvelopeJSONRejectsMalformedPayload(parseT *testing.T) {
	parsePayload := []byte(`{"region_instance_id":"region-1","epoch":2,"patch_version":"bad","patch_payload":"cGF0Y2g="}`)
	if _, parseErr := runtime2.ParseStructuredClonePatchEnvelopeJSON(parsePayload); parseErr == nil {
		parseT.Fatal("expected malformed patch envelope payload to fail decode")
	}
}
