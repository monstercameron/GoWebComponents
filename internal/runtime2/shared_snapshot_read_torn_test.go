package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleSharedSnapshotReadHeaderRejectsTornWrite verifies reads reject pages still marked as writing.
func TestHandleSharedSnapshotReadHeaderRejectsTornWrite(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotPublishBegin(32); parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin returned error: %v", parseErr)
	}
	if _, parseErr := parsePage.HandleSharedSnapshotReadHeader(); parseErr == nil {
		parseT.Fatal("expected read during writing status to fail as torn write")
	}
}

// TestHandleSharedSnapshotReadHeaderAcceptsCompletedPublish verifies reads accept pages marked complete.
func TestHandleSharedSnapshotReadHeaderAcceptsCompletedPublish(parseT *testing.T) {
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
	parseHeaderRead, parseErr := parsePage.HandleSharedSnapshotReadHeader()
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotReadHeader returned error: %v", parseErr)
	}
	if parseHeaderRead.Status != runtime2.SharedSnapshotPageStatusComplete {
		parseT.Fatalf("expected complete status on read, got %d", parseHeaderRead.Status)
	}
}
