package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionOwnerRemoveDisposesMountedRegion verifies owner removal disposes mounted region state.
func TestHandleHostRegionOwnerRemoveDisposesMountedRegion(parseT *testing.T) {
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
	getOwnerRemoveResult, parseErr := buildHostRegionAdapter.HandleHostRegionOwnerRemove()
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionOwnerRemove returned error: %v", parseErr)
	}
	if !getOwnerRemoveResult.HasDisposed {
		parseT.Fatal("expected owner removal to dispose mounted region")
	}
	if _, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1")); parseHasEntry {
		parseT.Fatal("expected owner-removed region to be absent from coordinator")
	}
}

// TestHandleHostRegionOwnerRemoveSuppressesLateWorkerOutput verifies owner removal suppresses late worker output commits.
func TestHandleHostRegionOwnerRemoveSuppressesLateWorkerOutput(parseT *testing.T) {
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
	if _, parseErr = buildHostRegionAdapter.HandleHostRegionOwnerRemove(); parseErr != nil {
		parseT.Fatalf("HandleHostRegionOwnerRemove returned error: %v", parseErr)
	}
	getWorkerOutputResult, parseErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(2)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseErr)
	}
	if !getWorkerOutputResult.HasIgnored {
		parseT.Fatal("expected late worker output to be suppressed after owner removal")
	}
	if getWorkerOutputResult.HasCommitted {
		parseT.Fatal("expected suppressed late worker output not to commit")
	}
}
