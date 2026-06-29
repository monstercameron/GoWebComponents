package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleSharedSnapshotReadHeaderAtGenerationAcceptsMatchingGeneration verifies matching generation reads succeed.
func TestHandleSharedSnapshotReadHeaderAtGenerationAcceptsMatchingGeneration(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeaderBegin, parseErr := parsePage.HandleSharedSnapshotPublishBegin(32)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotPublishComplete(parseHeaderBegin.Generation); parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishComplete returned error: %v", parseErr)
	}
	parseHeaderRead, parseErr := parsePage.HandleSharedSnapshotReadHeaderAtGeneration(parseHeaderBegin.Generation)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotReadHeaderAtGeneration returned error: %v", parseErr)
	}
	if parseHeaderRead.Generation != parseHeaderBegin.Generation {
		parseT.Fatalf("expected generation %d, got %d", parseHeaderBegin.Generation, parseHeaderRead.Generation)
	}
}

// TestHandleSharedSnapshotReadHeaderAtGenerationRejectsStaleGeneration verifies mismatched expected generations are rejected.
func TestHandleSharedSnapshotReadHeaderAtGenerationRejectsStaleGeneration(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeaderBegin, parseErr := parsePage.HandleSharedSnapshotPublishBegin(32)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotPublishComplete(parseHeaderBegin.Generation); parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishComplete returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotReadHeaderAtGeneration(parseHeaderBegin.Generation + 1); parseErr == nil {
		parseT.Fatal("expected stale generation read to fail")
	}
}
