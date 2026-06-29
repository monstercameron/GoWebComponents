package runtime2_test

import (
	"encoding/json"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestParseSharedSnapshotEnvelopeDecodesPublishedSnapshot verifies shared-page payloads decode back into SnapshotEnvelope.
func TestParseSharedSnapshotEnvelopeDecodesPublishedSnapshot(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(1024)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseEnvelope := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     2,
		Props:            map[string]any{"title": "Orders"},
	}
	parsePayload, parseErr := json.Marshal(parseEnvelope)
	if parseErr != nil {
		parseT.Fatalf("json.Marshal(snapshot) returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotPublishPayload(parsePayload); parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishPayload returned error: %v", parseErr)
	}
	parseDecoded, parseErr := parsePage.ParseSharedSnapshotEnvelope()
	if parseErr != nil {
		parseT.Fatalf("ParseSharedSnapshotEnvelope returned error: %v", parseErr)
	}
	if parseDecoded.RegionInstanceID != parseEnvelope.RegionInstanceID {
		parseT.Fatalf("expected region %q, got %q", parseEnvelope.RegionInstanceID, parseDecoded.RegionInstanceID)
	}
	if parseDecoded.InputVersion != parseEnvelope.InputVersion {
		parseT.Fatalf("expected input version %d, got %d", parseEnvelope.InputVersion, parseDecoded.InputVersion)
	}
}
