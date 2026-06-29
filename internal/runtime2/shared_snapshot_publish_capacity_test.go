package runtime2_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleSharedSnapshotPublishPayloadRejectsOverflow verifies publish fails when payload exceeds page capacity.
func TestHandleSharedSnapshotPublishPayloadRejectsOverflow(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(64)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parsePayload := strings.Repeat("x", 100)
	if _, parseErr := parsePage.HandleSharedSnapshotPublishPayload([]byte(parsePayload)); parseErr == nil {
		parseT.Fatal("expected publish overflow to fail")
	}
}
