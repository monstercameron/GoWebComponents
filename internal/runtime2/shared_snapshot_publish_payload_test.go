package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleSharedSnapshotPublishPayloadWritesReadablePage verifies one publish writes a readable complete page payload.
func TestHandleSharedSnapshotPublishPayloadWritesReadablePage(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(512)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parsePayload := []byte(`{"region_instance_id":"region-1","epoch":1,"input_version":1}`)
	parseHeader, parseErr := parsePage.HandleSharedSnapshotPublishPayload(parsePayload)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishPayload returned error: %v", parseErr)
	}
	if parseHeader.Status != runtime2.SharedSnapshotPageStatusComplete {
		parseT.Fatalf("expected complete status, got %d", parseHeader.Status)
	}
	parseReadPayload, parseErr := parsePage.GetSharedSnapshotReadPayload()
	if parseErr != nil {
		parseT.Fatalf("GetSharedSnapshotReadPayload returned error: %v", parseErr)
	}
	if string(parseReadPayload) != string(parsePayload) {
		parseT.Fatalf("expected read payload %q, got %q", string(parsePayload), string(parseReadPayload))
	}
}
