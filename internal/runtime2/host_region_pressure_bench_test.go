package runtime2_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers benchmarks many hot regions sharing a bounded worker shard set.
func BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers(parseB *testing.B) {
	getRegionCount := 48
	getShardIDs := []runtime2.SchedulerShardID{"shard-a", "shard-b", "shard-c", "shard-d"}
	getAdapters := make([]*runtime2.HostRegionAdapter, 0, getRegionCount)
	getRegionIDs := make([]runtime2.RegionInstanceID, 0, getRegionCount)
	getPropsValues := make([]map[string]any, 0, getRegionCount)
	getSpecs := make([]runtime2.ParallelRegionSpec, 0, getRegionCount)
	getVersions := make([]uint64, 0, getRegionCount)
	for getRegionIndex := 0; getRegionIndex < getRegionCount; getRegionIndex++ {
		getRegionID := runtime2.RegionInstanceID(fmt.Sprintf("region-%d", getRegionIndex))
		buildHostRegionAdapter, parseBuildErr := runtime2.BuildHostRegionAdapter(getRegionID, getShardIDs)
		if parseBuildErr != nil {
			parseB.Fatalf("BuildHostRegionAdapter(%s) returned error: %v", getRegionID, parseBuildErr)
		}
		_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(
			runtime2.ParallelRegionSpec{
				RendererID:       runtime2.RendererID("dashboard.hot-panel"),
				RegionInstanceID: getRegionID,
			},
			1,
		)
		if parseMountErr != nil {
			parseB.Fatalf("HandleHostRegionMount(%s) returned error: %v", getRegionID, parseMountErr)
		}
		getRegionIDs = append(getRegionIDs, getRegionID)
		getPropsValue := map[string]any{
			"tick": 0,
		}
		getPropsValues = append(getPropsValues, getPropsValue)
		getSpecs = append(getSpecs, runtime2.ParallelRegionSpec{
			RendererID:       runtime2.RendererID("dashboard.hot-panel"),
			RegionInstanceID: getRegionID,
			Props:            getPropsValue,
		})
		getAdapters = append(getAdapters, buildHostRegionAdapter)
		getVersions = append(getVersions, 1)
	}
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for getIteration := 0; getIteration < parseB.N; getIteration++ {
		for getRegionIndex, getHostRegionAdapter := range getAdapters {
			getVersions[getRegionIndex]++
			getRegionID := getRegionIDs[getRegionIndex]
			getPropsValues[getRegionIndex]["tick"] = getIteration
			getSpec := getSpecs[getRegionIndex]
			_, parseDispatchErr := getHostRegionAdapter.HandleHostRegionUpdateDispatch(
				getSpec,
				getVersions[getRegionIndex],
			)
			if parseDispatchErr != nil {
				parseB.Fatalf("HandleHostRegionUpdateDispatch(%s) returned error: %v", getRegionID, parseDispatchErr)
			}
		}
	}
}
