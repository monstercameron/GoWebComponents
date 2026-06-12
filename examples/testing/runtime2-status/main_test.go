//go:build js && wasm

package main

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/ui"
)

// TestBuildRuntime2StatusNoticeTextsReportsRenderOnlyCaveats verifies the example warns when it only has a bridge shard label and no hydration path.
func TestBuildRuntime2StatusNoticeTextsReportsRenderOnlyCaveats(parseT *testing.T) {
	getNoticeTexts := buildRuntime2StatusNoticeTexts(true, ui.ParallelRegionStatus{
		GetAssignedWorkerShard:   "ui-parallel-region",
		GetIsHydrationComplete:   false,
		HasHydratedShellAnchor:   false,
		HasPostHydrationAttached: false,
	})
	if len(getNoticeTexts) != 2 {
		parseT.Fatalf("buildRuntime2StatusNoticeTexts() len = %d, want 2", len(getNoticeTexts))
	}
}

// TestBuildRuntime2StatusNoticeTextsSkipsWorkerAttachedHydratedState verifies the example stays quiet once the runtime status no longer matches the reviewed caveats.
func TestBuildRuntime2StatusNoticeTextsSkipsWorkerAttachedHydratedState(parseT *testing.T) {
	getNoticeTexts := buildRuntime2StatusNoticeTexts(true, ui.ParallelRegionStatus{
		GetAssignedWorkerShard:   "shard-a",
		GetIsHydrationComplete:   true,
		HasHydratedShellAnchor:   true,
		HasPostHydrationAttached: true,
	})
	if len(getNoticeTexts) != 0 {
		parseT.Fatalf("buildRuntime2StatusNoticeTexts() len = %d, want 0", len(getNoticeTexts))
	}
}
