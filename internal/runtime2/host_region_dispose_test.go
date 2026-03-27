package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionDisposeMountedRegionClearsHostState verifies host-side dispose clears mounted coordinator, scheduler, and DOM index state.
func TestHandleHostRegionDisposeMountedRegionClearsHostState(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	parseErr = buildHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode(
		"region-1",
		1,
		&runtime2.RegionDOMNode{GetNodeID: 1, GetTag: "div"},
	)
	if parseErr != nil {
		parseT.Fatalf("SetRegionDOMNode returned error: %v", parseErr)
	}
	getDisposeResult, parseErr := buildHostRegionAdapter.HandleHostRegionDispose()
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionDispose returned error: %v", parseErr)
	}
	if !getDisposeResult.HasCoordinatorDisposed {
		parseT.Fatal("expected coordinator entry disposal")
	}
	if !getDisposeResult.HasSchedulerDisposed {
		parseT.Fatal("expected scheduler disposal")
	}
	if getDisposeResult.GetClearedDOMNodeCount != 1 {
		parseT.Fatalf("expected one cleared DOM node, got %d", getDisposeResult.GetClearedDOMNodeCount)
	}
	if _, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1")); parseHasEntry {
		parseT.Fatal("expected disposed coordinator entry to be removed")
	}
	if _, parseErr = buildHostRegionAdapter.GetHostRegionScheduler().HandleSchedulerUpdate("region-1"); parseErr == nil {
		parseT.Fatal("expected disposed scheduler region to reject updates")
	}
	if _, parseErr = buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", 1); parseErr == nil {
		parseT.Fatal("expected disposed DOM index entry to be removed")
	}
}

// TestHandleHostRegionDisposeRejectsNotMountedRegion verifies host-side dispose fails when the adapter region is not mounted.
func TestHandleHostRegionDisposeRejectsNotMountedRegion(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionDispose()
	if parseErr == nil {
		parseT.Fatal("expected host dispose without mount to fail")
	}
}
