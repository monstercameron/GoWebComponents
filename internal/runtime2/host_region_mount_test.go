package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionMountMountsValidatedParallelRegion verifies host-side mount accepts a valid parallel region spec and stores mounted coordinator state.
func TestHandleHostRegionMountMountsValidatedParallelRegion(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	getMountResult, parseErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
		},
		1,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseErr)
	}
	if getMountResult.GetSchedulerJob.GetSchedulerJobKind != runtime2.SchedulerJobKindMount {
		parseT.Fatalf("expected mount scheduler job, got %q", getMountResult.GetSchedulerJob.GetSchedulerJobKind)
	}
	if getMountResult.GetCoordinatorEntry.CurrentState != runtime2.CoordinatorStateMounted {
		parseT.Fatalf("expected mounted coordinator state, got %q", getMountResult.GetCoordinatorEntry.CurrentState)
	}
	if getMountResult.GetCoordinatorEntry.AssignedWorkerShard == "" {
		parseT.Fatal("expected assigned worker shard on mount")
	}
}

// TestHandleHostRegionMountRejectsInvalidParallelRegionSpec verifies host-side mount rejects invalid specs before scheduling.
func TestHandleHostRegionMountRejectsInvalidParallelRegionSpec(parseT *testing.T) {
	buildHostRegionAdapter, parseErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       "",
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		1,
	)
	if parseErr == nil {
		parseT.Fatal("expected invalid parallel region spec to fail mount")
	}
	if buildHostRegionAdapter.GetHostRegionScheduler().GetSchedulerQueueDepth() != 0 {
		parseT.Fatal("expected invalid spec to skip scheduler mount")
	}
}

// TestHandleHostRegionMountRejectsMismatchedAdapterRegion verifies host-side mount enforces one adapter per region instance.
func TestHandleHostRegionMountRejectsMismatchedAdapterRegion(parseT *testing.T) {
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
			RegionInstanceID: runtime2.RegionInstanceID("region-2"),
		},
		1,
	)
	if parseErr == nil {
		parseT.Fatal("expected mismatched adapter region to fail mount")
	}
}
