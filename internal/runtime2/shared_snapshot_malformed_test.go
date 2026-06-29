package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestParseSharedSnapshotEnvelopeRejectsMalformedHeaderBeforeDecode verifies malformed page headers are rejected before payload decode.
func TestParseSharedSnapshotEnvelopeRejectsMalformedHeaderBeforeDecode(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.ParseSharedSnapshotEnvelope(); parseErr == nil {
		parseT.Fatal("expected malformed header to fail before payload decode")
	}
}

// TestParseSharedSnapshotEnvelopeRejectsCorruptPayloadLengthBeforeDecode verifies corrupt payload lengths fail before JSON decode.
func TestParseSharedSnapshotEnvelopeRejectsCorruptPayloadLengthBeforeDecode(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeaderBegin, parseErr := parsePage.HandleSharedSnapshotPublishBegin(4096)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotPublishComplete(parseHeaderBegin.Generation); parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishComplete returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.ParseSharedSnapshotEnvelope(); parseErr == nil {
		parseT.Fatal("expected corrupt payload length to fail before payload decode")
	}
}
