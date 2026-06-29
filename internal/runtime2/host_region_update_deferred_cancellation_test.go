package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// TestHandleHostRegionUpdateDispatchWithPriorityUrgentCancelsQueuedDeferred verifies newer urgent updates cancel queued deferred updates.
func TestHandleHostRegionUpdateDispatchWithPriorityUrgentCancelsQueuedDeferred(parseT *testing.T) {
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
		parseT.Fatalf("expected deferred input version 2 before urgent cancellation, got %d", buildHostRegionAdapter.GetHostRegionDeferredInputVersion())
	}
	getUrgentDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders-B"},
			SourceIDs:        []string{"count"},
		},
		3,
		runtime2.HostRegionDispatchPriorityUrgent,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithPriority(urgent) returned error: %v", parseErr)
	}
	if !getUrgentDispatchResult.HasDeferredCanceled {
		parseT.Fatal("expected newer urgent update to cancel queued deferred update")
	}
	if buildHostRegionAdapter.GetHostRegionDeferredInputVersion() != 0 {
		parseT.Fatalf("expected deferred queue to clear after urgent cancellation, got version %d", buildHostRegionAdapter.GetHostRegionDeferredInputVersion())
	}
}
