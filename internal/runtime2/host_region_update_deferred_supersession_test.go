package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionUpdateDispatchWithPriorityDeferredSupersedesOlderDeferred verifies newer deferred updates supersede older deferred updates.
func TestHandleHostRegionUpdateDispatchWithPriorityDeferredSupersedesOlderDeferred(parseT *testing.T) {
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
	getFirstDeferredDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
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
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithPriority(first deferred) returned error: %v", parseErr)
	}
	if !getFirstDeferredDispatchResult.HasDeferredQueued {
		parseT.Fatal("expected first deferred dispatch to queue deferred work")
	}
	if getFirstDeferredDispatchResult.HasDeferredSuperseded {
		parseT.Fatal("expected first deferred dispatch not to supersede prior deferred work")
	}

	getSecondDeferredDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders-B"},
			SourceIDs:        []string{"count"},
		},
		3,
		runtime2.HostRegionDispatchPriorityDeferred,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithPriority(second deferred) returned error: %v", parseErr)
	}
	if !getSecondDeferredDispatchResult.HasDeferredQueued {
		parseT.Fatal("expected second deferred dispatch to queue deferred work")
	}
	if !getSecondDeferredDispatchResult.HasDeferredSuperseded {
		parseT.Fatal("expected second deferred dispatch to supersede prior deferred work")
	}
	if buildHostRegionAdapter.GetHostRegionDeferredInputVersion() != 3 {
		parseT.Fatalf("expected deferred input version 3 after supersession, got %d", buildHostRegionAdapter.GetHostRegionDeferredInputVersion())
	}
}
