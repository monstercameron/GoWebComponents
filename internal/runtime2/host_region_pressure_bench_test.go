package runtime2_test

import (
	"fmt"
	"math"
	"slices"
	"testing"
	"time"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime2"
)

// buildHostRegionPressureLatencySampleSlice builds one reusable latency sample slice for percentile reporting.
func buildHostRegionPressureLatencySampleSlice(parseCapacity int) []int64 {
	if parseCapacity <= 0 {
		return nil
	}
	return make([]int64, 0, parseCapacity)
}

// storeHostRegionPressureLatencySample appends one latency sample to parseSamples and returns the updated slice.
func storeHostRegionPressureLatencySample(parseSamples []int64, parseDurationNS int64) []int64 {
	return append(parseSamples, parseDurationNS)
}

// getHostRegionPressureLatencyPercentile returns one latency percentile from sorted nanosecond samples.
func getHostRegionPressureLatencyPercentile(parseSortedSamples []int64, parsePercentile float64) int64 {
	if len(parseSortedSamples) == 0 {
		return 0
	}
	if parsePercentile <= 0 {
		return parseSortedSamples[0]
	}
	if parsePercentile >= 1 {
		return parseSortedSamples[len(parseSortedSamples)-1]
	}
	getIndex := max(int(math.Ceil(parsePercentile*float64(len(parseSortedSamples)))-1), 0)
	if getIndex >= len(parseSortedSamples) {
		getIndex = len(parseSortedSamples) - 1
	}
	return parseSortedSamples[getIndex]
}

// reportHostRegionPressureLatencyPercentiles reports p50/p95/p99 latency metrics from collected dispatch-batch samples.
func reportHostRegionPressureLatencyPercentiles(parseB *testing.B, parseSamples []int64) {
	if len(parseSamples) == 0 {
		return
	}
	slices.Sort(parseSamples)
	parseB.ReportMetric(float64(getHostRegionPressureLatencyPercentile(parseSamples, 0.50)), "dispatch-batch-p50-ns")
	parseB.ReportMetric(float64(getHostRegionPressureLatencyPercentile(parseSamples, 0.95)), "dispatch-batch-p95-ns")
	parseB.ReportMetric(float64(getHostRegionPressureLatencyPercentile(parseSamples, 0.99)), "dispatch-batch-p99-ns")
	parseB.ReportMetric(float64(len(parseSamples)), "dispatch-batch-samples")
}

// BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers benchmarks many hot regions sharing a bounded worker shard set.
func BenchmarkHandleHostRegionManyHotRegionsBoundedWorkers(parseB *testing.B) {
	getRegionCount := 48
	getShardIDs := []runtime2.SchedulerShardID{"shard-a", "shard-b", "shard-c", "shard-d"}
	getAdapters := make([]*runtime2.HostRegionAdapter, 0, getRegionCount)
	getRegionIDs := make([]runtime2.RegionInstanceID, 0, getRegionCount)
	getPropsValues := make([]map[string]any, 0, getRegionCount)
	getSpecs := make([]runtime2.ParallelRegionSpec, 0, getRegionCount)
	getVersions := make([]uint64, 0, getRegionCount)
	for getRegionIndex := range getRegionCount {
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
	getLatencySamples := buildHostRegionPressureLatencySampleSlice(parseB.N)
	getLatencySpanSize := 64
	getLatencySpanStart := time.Now()
	getLatencySpanCount := 0
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
		getLatencySpanCount++
		if getLatencySpanCount == getLatencySpanSize {
			getLatencySamples = storeHostRegionPressureLatencySample(
				getLatencySamples,
				time.Since(getLatencySpanStart).Nanoseconds()/int64(getLatencySpanCount),
			)
			getLatencySpanStart = time.Now()
			getLatencySpanCount = 0
		}
	}
	if getLatencySpanCount > 0 {
		getLatencySamples = storeHostRegionPressureLatencySample(
			getLatencySamples,
			time.Since(getLatencySpanStart).Nanoseconds()/int64(getLatencySpanCount),
		)
	}
	parseB.StopTimer()
	reportHostRegionPressureLatencyPercentiles(parseB, getLatencySamples)
}
