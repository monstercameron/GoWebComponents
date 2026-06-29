package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleSharedSnapshotPublishPayloadRepeatedPublishAdvancesGeneration verifies repeated publishes advance generation and update payload.
func TestHandleSharedSnapshotPublishPayloadRepeatedPublishAdvancesGeneration(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(512)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeaderFirst, parseErr := parsePage.HandleSharedSnapshotPublishPayload([]byte(`{"input_version":1}`))
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishPayload(first) returned error: %v", parseErr)
	}
	parseHeaderSecond, parseErr := parsePage.HandleSharedSnapshotPublishPayload([]byte(`{"input_version":2}`))
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishPayload(second) returned error: %v", parseErr)
	}
	if parseHeaderSecond.Generation <= parseHeaderFirst.Generation {
		parseT.Fatalf("expected generation advance, first=%d second=%d", parseHeaderFirst.Generation, parseHeaderSecond.Generation)
	}
	parseReadPayload, parseErr := parsePage.GetSharedSnapshotReadPayload()
	if parseErr != nil {
		parseT.Fatalf("GetSharedSnapshotReadPayload returned error: %v", parseErr)
	}
	if string(parseReadPayload) != `{"input_version":2}` {
		parseT.Fatalf("expected latest payload to be returned, got %q", string(parseReadPayload))
	}
}
