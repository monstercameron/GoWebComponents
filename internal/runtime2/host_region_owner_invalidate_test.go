package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionOwnerInvalidateCleansQueuedState verifies owner-side invalidation clears queued scheduler and deferred update state.
func TestHandleHostRegionOwnerInvalidateCleansQueuedState(parseT *testing.T) {
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
	parseErr = buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{"count": 5}, map[string]uint64{"count": 3}, nil
		},
	)
	if parseErr != nil {
		parseT.Fatalf("SetHostRegionSourceLookup returned error: %v", parseErr)
	}
	_, parseErr = buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders-A"},
			SourceIDs:        []string{"count"},
		},
		2,
		runtime2.HostRegionDispatchPriorityDeferred,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithPriority(deferred) returned error: %v", parseErr)
	}
	if buildHostRegionAdapter.GetHostRegionDeferredInputVersion() != 2 {
		parseT.Fatalf("expected deferred input version 2 before owner invalidation, got %d", buildHostRegionAdapter.GetHostRegionDeferredInputVersion())
	}
	if buildHostRegionAdapter.GetHostRegionScheduler().GetSchedulerQueueDepth() == 0 {
		parseT.Fatal("expected queued scheduler jobs before owner invalidation")
	}
	getInvalidateResult, parseErr := buildHostRegionAdapter.HandleHostRegionOwnerInvalidate()
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionOwnerInvalidate returned error: %v", parseErr)
	}
	if !getInvalidateResult.HasSchedulerCanceled {
		parseT.Fatal("expected owner invalidation to cancel queued scheduler jobs")
	}
	if !getInvalidateResult.HasDeferredCleared {
		parseT.Fatal("expected owner invalidation to clear deferred queue state")
	}
	if buildHostRegionAdapter.GetHostRegionDeferredInputVersion() != 0 {
		parseT.Fatalf("expected deferred input version to clear after invalidation, got %d", buildHostRegionAdapter.GetHostRegionDeferredInputVersion())
	}
	if buildHostRegionAdapter.GetHostRegionScheduler().GetSchedulerQueueDepth() != 0 {
		parseT.Fatalf("expected scheduler queue depth 0 after invalidation, got %d", buildHostRegionAdapter.GetHostRegionScheduler().GetSchedulerQueueDepth())
	}
}
