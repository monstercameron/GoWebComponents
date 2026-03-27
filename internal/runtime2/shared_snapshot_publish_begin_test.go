package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleSharedSnapshotPublishBeginWritesWritingHeader verifies publish-begin writes a writing-status header.
func TestHandleSharedSnapshotPublishBeginWritesWritingHeader(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeader, parseErr := parsePage.HandleSharedSnapshotPublishBegin(32)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin returned error: %v", parseErr)
	}
	if parseHeader.Status != runtime2.SharedSnapshotPageStatusWriting {
		parseT.Fatalf("expected writing status, got %d", parseHeader.Status)
	}
	if parseHeader.Generation != 1 {
		parseT.Fatalf("expected generation 1, got %d", parseHeader.Generation)
	}
}

// TestHandleSharedSnapshotPublishBeginAdvancesGeneration verifies repeated publish-begin calls advance generation.
func TestHandleSharedSnapshotPublishBeginAdvancesGeneration(parseT *testing.T) {
	parsePage, parseErr := runtime2.BuildSharedSnapshotPage(256)
	if parseErr != nil {
		parseT.Fatalf("BuildSharedSnapshotPage returned error: %v", parseErr)
	}
	parseHeaderFirst, parseErr := parsePage.HandleSharedSnapshotPublishBegin(16)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin(first) returned error: %v", parseErr)
	}
	parseHeaderSecond, parseErr := parsePage.HandleSharedSnapshotPublishBegin(24)
	if parseErr != nil {
		parseT.Fatalf("HandleSharedSnapshotPublishBegin(second) returned error: %v", parseErr)
	}
	if parseHeaderSecond.Generation <= parseHeaderFirst.Generation {
		parseT.Fatalf("expected generation advance, first=%d second=%d", parseHeaderFirst.Generation, parseHeaderSecond.Generation)
	}
}
