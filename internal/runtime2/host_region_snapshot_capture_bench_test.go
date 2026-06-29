package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// BenchmarkHandleHostRegionUpdateSnapshot benchmarks host-side source snapshot capture for one mounted region.
func BenchmarkHandleHostRegionUpdateSnapshot(parseB *testing.B) {
	buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(
		runtime2.RegionInstanceID("region-1"),
		[]runtime2.SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
		runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		},
		3,
	)
	if parseMountErr != nil {
		parseB.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseSourceLookupErr := buildHostRegionAdapter.SetHostRegionSourceLookup(
		func(parseSourceIDs []string) (map[string]any, map[string]uint64, error) {
			return map[string]any{
					"count": 5,
				},
				map[string]uint64{
					"count": 9,
				},
				nil
		},
	)
	if parseSourceLookupErr != nil {
		parseB.Fatalf("SetHostRegionSourceLookup returned error: %v", parseSourceLookupErr)
	}
	getSpec := runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Props:            map[string]any{"title": "Orders"},
		SourceIDs:        []string{"count"},
	}
	parseB.ResetTimer()
	for getIteration := 0; getIteration < parseB.N; getIteration++ {
		_, parseSnapshotErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(getSpec, uint64(getIteration+1))
		if parseSnapshotErr != nil {
			parseB.Fatalf("HandleHostRegionUpdateSnapshot returned error: %v", parseSnapshotErr)
		}
	}
}
