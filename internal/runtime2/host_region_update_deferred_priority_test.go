package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionUpdateDispatchWithPriorityDeferredClassification verifies runtime2 update dispatch can classify updates as deferred.
func TestHandleHostRegionUpdateDispatchWithPriorityDeferredClassification(parseT *testing.T) {
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
	getDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithPriority(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		2,
		runtime2.HostRegionDispatchPriorityDeferred,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithPriority returned error: %v", parseErr)
	}
	if getDispatchResult.GetDispatchPriority != runtime2.HostRegionDispatchPriorityDeferred {
		parseT.Fatalf("expected deferred dispatch priority, got %q", getDispatchResult.GetDispatchPriority)
	}
}
