package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestHandleHostRegionUpdateDispatchDefaultsToUrgentPriority verifies runtime2 update dispatch defaults to urgent classification.
func TestHandleHostRegionUpdateDispatchDefaultsToUrgentPriority(parseT *testing.T) {
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
	getDispatchResult, parseErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatch(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
			Props:            map[string]any{"title": "Orders"},
			SourceIDs:        []string{"count"},
		},
		2,
	)
	if parseErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatch returned error: %v", parseErr)
	}
	if getDispatchResult.GetDispatchPriority != runtime2.HostRegionDispatchPriorityUrgent {
		parseT.Fatalf("expected urgent dispatch priority, got %q", getDispatchResult.GetDispatchPriority)
	}
}
