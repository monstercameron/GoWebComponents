package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionUpdateMountedRegionDispatchesUpdate verifies host-side update dispatches scheduler and coordinator state for one mounted region.
func TestHandleHostRegionUpdateMountedRegionDispatchesUpdate(parseT *testing.T) {
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
	getUpdateResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdate(2)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate returned error: %v", parseErr)
	}
	if getUpdateResult.GetSchedulerJob.GetSchedulerJobKind != runtime2.SchedulerJobKindUpdate {
		parseT.Fatalf("expected update scheduler job, got %q", getUpdateResult.GetSchedulerJob.GetSchedulerJobKind)
	}
	parseEntry, parseHasEntry := buildHostRegionAdapter.GetHostRegionCoordinator().GetEntry(runtime2.RegionInstanceID("region-1"))
	if !parseHasEntry {
		parseT.Fatal("expected mounted coordinator entry")
	}
	if parseEntry.LastDispatchedVersion != 2 {
		parseT.Fatalf("expected dispatched version 2, got %d", parseEntry.LastDispatchedVersion)
	}
}

// TestHandleHostRegionUpdateRejectsNotMountedRegion verifies host-side update fails when the adapter region has not mounted.
func TestHandleHostRegionUpdateRejectsNotMountedRegion(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdate(1)
	if parseErr == nil {
		parseT.Fatal("expected host update without mount to fail")
	}
}

// TestHandleHostRegionUpdateRejectsStaleVersion verifies host-side update rejects stale input versions.
func TestHandleHostRegionUpdateRejectsStaleVersion(parseT *testing.T) {
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
	if _, parseErr = buildHostRegionAdapter.HandleHostRegionUpdate(2); parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate(2) returned error: %v", parseErr)
	}
	if _, parseErr = buildHostRegionAdapter.HandleHostRegionUpdate(1); parseErr == nil {
		parseT.Fatal("expected stale host update version to fail")
	}
}
