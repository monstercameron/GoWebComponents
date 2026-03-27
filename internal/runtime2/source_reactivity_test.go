package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestSetRegionDeclaredSourcesDeclaredSourceChangeEnqueuesUpdate verifies declared source changes enqueue region updates.
func TestSetRegionDeclaredSourcesDeclaredSourceChangeEnqueuesUpdate(parseT *testing.T) {
	buildSourceReactivity := runtime2.BuildSourceReactivity()
	if parseErr := buildSourceReactivity.SetRegionDeclaredSources(runtime2.RegionInstanceID("region-1"), []string{"status", "count"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources returned error: %v", parseErr)
	}
	hasSourceChangeQueued := buildSourceReactivity.HandleSourceChange("status")
	if !hasSourceChangeQueued {
		parseT.Fatal("expected declared source change to enqueue region update")
	}
	getRegionUpdates := buildSourceReactivity.GetRegionUpdateQueue()
	if len(getRegionUpdates) != 1 || getRegionUpdates[0] != runtime2.RegionInstanceID("region-1") {
		parseT.Fatalf("expected queued update for region-1, got %+v", getRegionUpdates)
	}
}

// TestSetRegionDeclaredSourcesUnrelatedSourceChangeDoesNotEnqueue verifies unrelated source changes do not enqueue region updates.
func TestSetRegionDeclaredSourcesUnrelatedSourceChangeDoesNotEnqueue(parseT *testing.T) {
	buildSourceReactivity := runtime2.BuildSourceReactivity()
	if parseErr := buildSourceReactivity.SetRegionDeclaredSources(runtime2.RegionInstanceID("region-1"), []string{"status", "count"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources returned error: %v", parseErr)
	}
	hasSourceChangeQueued := buildSourceReactivity.HandleSourceChange("other")
	if hasSourceChangeQueued {
		parseT.Fatal("expected unrelated source change to skip region enqueue")
	}
	getRegionUpdates := buildSourceReactivity.GetRegionUpdateQueue()
	if len(getRegionUpdates) != 0 {
		parseT.Fatalf("expected no queued updates for unrelated source change, got %+v", getRegionUpdates)
	}
}

// TestSetRegionDeclaredSourcesMultipleDeclaredChangesCoalesce verifies multiple declared source changes coalesce consistently.
func TestSetRegionDeclaredSourcesMultipleDeclaredChangesCoalesce(parseT *testing.T) {
	buildSourceReactivity := runtime2.BuildSourceReactivity()
	if parseErr := buildSourceReactivity.SetRegionDeclaredSources(runtime2.RegionInstanceID("region-1"), []string{"status", "count"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources returned error: %v", parseErr)
	}
	if parseErr := buildSourceReactivity.SetRegionDeclaredSources(runtime2.RegionInstanceID("region-2"), []string{"other"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources region-2 returned error: %v", parseErr)
	}
	buildSourceReactivity.HandleSourceChange("status")
	buildSourceReactivity.HandleSourceChange("count")
	buildSourceReactivity.HandleSourceChange("other")
	getRegionUpdates := buildSourceReactivity.GetRegionUpdateQueue()
	if len(getRegionUpdates) != 2 {
		parseT.Fatalf("expected two coalesced queued updates, got %+v", getRegionUpdates)
	}
	if getRegionUpdates[0] != runtime2.RegionInstanceID("region-1") || getRegionUpdates[1] != runtime2.RegionInstanceID("region-2") {
		parseT.Fatalf("expected stable coalesced ordering [region-1 region-2], got %+v", getRegionUpdates)
	}
}
