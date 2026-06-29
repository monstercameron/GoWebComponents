package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleSharedSnapshotPublishCompleteAcceptsCompletePublish verifies completion marks one begun publish as complete.
func TestHandleSharedSnapshotPublishCompleteAcceptsCompletePublish(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeaderBegin, parseErr := parsePage.HandleSharedSnapshotPublishBegin(32)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin returned error: %v", parseErr)
	}
	parseHeaderComplete, parseErr := parsePage.HandleSharedSnapshotPublishComplete(parseHeaderBegin.Generation)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishComplete returned error: %v", parseErr)
	}
	if parseHeaderComplete.Status != runtime2.SharedSnapshotPageStatusComplete {
		parseT.Fatalf("expected complete status, got %d", parseHeaderComplete.Status)
	}
	if parseHeaderComplete.Generation != parseHeaderBegin.Generation {
		parseT.Fatalf("expected generation %d, got %d", parseHeaderBegin.Generation, parseHeaderComplete.Generation)
	}
}

// TestHandleSharedSnapshotPublishCompleteRejectsIncompletePublish verifies completion is rejected when no begin state exists.
func TestHandleSharedSnapshotPublishCompleteRejectsIncompletePublish(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotPublishComplete(1); parseErr == nil {
		parseT.Fatal("expected completion without begin to fail")
	}
}

// TestHandleSharedSnapshotPublishCompleteRejectsGenerationMismatch verifies completion rejects mismatched generations.
func TestHandleSharedSnapshotPublishCompleteRejectsGenerationMismatch(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeaderBegin, parseErr := parsePage.HandleSharedSnapshotPublishBegin(32)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotPublishComplete(parseHeaderBegin.Generation + 1); parseErr == nil {
		parseT.Fatal("expected generation mismatch to fail")
	}
}
