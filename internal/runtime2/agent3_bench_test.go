package runtime2

import (
	"math"
	"slices"
	"testing"
	"time"
)

// buildAgent3UpdateDispatchLatencySampleSlice builds one reusable latency sample slice for percentile reporting.
func buildAgent3UpdateDispatchLatencySampleSlice(parseCapacity int) []int64 {
	if parseCapacity <= 0 {
		return nil
	}
	return make([]int64, 0, parseCapacity)
}

// storeAgent3UpdateDispatchLatencySample appends one update-dispatch latency sample to parseSamples.
func storeAgent3UpdateDispatchLatencySample(parseSamples []int64, parseDurationNS int64) []int64 {
	return append(parseSamples, parseDurationNS)
}

// getAgent3UpdateDispatchLatencyPercentile returns one latency percentile from sorted nanosecond samples.
func getAgent3UpdateDispatchLatencyPercentile(parseSortedSamples []int64, parsePercentile float64) int64 {
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

// reportAgent3UpdateDispatchLatencyPercentiles reports p50/p95/p99 latency metrics from update-dispatch samples.
func reportAgent3UpdateDispatchLatencyPercentiles(parseB *testing.B, parseSamples []int64) {
	if len(parseSamples) == 0 {
		return
	}
	slices.Sort(parseSamples)
	parseB.ReportMetric(float64(getAgent3UpdateDispatchLatencyPercentile(parseSamples, 0.50)), "dispatch-loop-p50-ns")
	parseB.ReportMetric(float64(getAgent3UpdateDispatchLatencyPercentile(parseSamples, 0.95)), "dispatch-loop-p95-ns")
	parseB.ReportMetric(float64(getAgent3UpdateDispatchLatencyPercentile(parseSamples, 0.99)), "dispatch-loop-p99-ns")
	parseB.ReportMetric(float64(len(parseSamples)), "dispatch-loop-samples")
}

func parseBuildAgent3BenchRenderOutput(parseText string, parseClass string) map[string]any {
	return map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": parseClass,
		},
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "ul",
				"children": []any{
					map[string]any{"kind": "host-element", "tag": "li", "key": "a"},
					map[string]any{"kind": "host-element", "tag": "li", "key": "b"},
					map[string]any{"kind": "text", "text": parseText},
				},
			},
		},
	}
}

// BenchmarkBuildCanonicalRenderIR benchmarks worker-side canonical IR build.
func BenchmarkBuildCanonicalRenderIR(parseB *testing.B) {
	parseRenderOutput := parseBuildAgent3BenchRenderOutput("hello", "active")
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, parseErr := BuildCanonicalRenderIR(parseRenderOutput); parseErr != nil {
			parseB.Fatalf("BuildCanonicalRenderIR returned error: %v", parseErr)
		}
	}
}

// BenchmarkBuildCanonicalPatchStream benchmarks worker-side canonical diff and patch generation.
func BenchmarkBuildCanonicalPatchStream(parseB *testing.B) {
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, _, parsePatchErr := BuildCanonicalPatchStream("bench-region", 1, uint64(parseIndex+2), uint64(parseIndex+2), parsePreviousIR, parseNextIR); parsePatchErr != nil {
			parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
		}
	}
}

// BenchmarkParsePatchStreamTransaction benchmarks patch-stream decode and validation.
func BenchmarkParsePatchStreamTransaction(parseB *testing.B) {
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream("bench-region", 1, 2, 2, parsePreviousIR, parseNextIR)
	if parsePatchErr != nil {
		parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseB.Fatal("BuildCanonicalPatchStream returned no-op for differing payloads")
	}
	parsePreviousTree, parseTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
	if parseTreeErr != nil {
		parseB.Fatalf("ParseCanonicalRenderTree returned error: %v", parseTreeErr)
	}
	parseKnownNodeIDs := make(map[uint64]struct{}, len(parsePreviousTree.getNodeByID))
	for getNodeID := range parsePreviousTree.getNodeByID {
		parseKnownNodeIDs[getNodeID] = struct{}{}
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		if _, _, parseParseErr := ParsePatchStreamTransaction(parsePatchStream, "bench-region", 1, parseKnownNodeIDs, map[uint64]uint32{}, nil); parseParseErr != nil {
			parseB.Fatalf("ParsePatchStreamTransaction returned error: %v", parseParseErr)
		}
	}
}

// BenchmarkCommitRegionPatchTransaction benchmarks patch transaction commit.
func BenchmarkCommitRegionPatchTransaction(parseB *testing.B) {
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream("bench-region", 1, 2, 2, parsePreviousIR, parseNextIR)
	if parsePatchErr != nil {
		parseB.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseB.Fatal("BuildCanonicalPatchStream returned no-op for differing payloads")
	}
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		parseDOMIndex := BuildRegionDOMIndex()
		parseSeedRegionDOMIndexFromCanonical(parseB, parseDOMIndex, "bench-region", parsePreviousIR)
		parseKnownNodeIDs, parseSiblingCountByParent := BuildRegionDOMPatchLookupMaps(parseDOMIndex, "bench-region")
		parseParseResult, hasApply, parseParseErr := ParsePatchStreamTransaction(parsePatchStream, "bench-region", 1, parseKnownNodeIDs, parseSiblingCountByParent, BuildPatchIdempotencyTracker())
		if parseParseErr != nil {
			parseB.Fatalf("ParsePatchStreamTransaction returned error: %v", parseParseErr)
		}
		if !hasApply {
			parseB.Fatal("ParsePatchStreamTransaction unexpectedly ignored first patch")
		}
		parseDOMCommitter := BuildDOMCommitter(parseDOMIndex)
		if _, parseCommitErr := parseDOMCommitter.CommitRegionPatchTransaction(parseParseResult.GetTransaction); parseCommitErr != nil {
			parseB.Fatalf("CommitRegionPatchTransaction returned error: %v", parseCommitErr)
		}
	}
}

// BenchmarkCompareLocalVsWorkerBackedRendering benchmarks local render-only conversion versus worker-backed diff production.
func BenchmarkCompareLocalVsWorkerBackedRendering(parseB *testing.B) {
	parseLocalRenderOutput := parseBuildAgent3BenchRenderOutput("local", "active")
	parsePreviousIR, parsePreviousErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("before", "active"))
	if parsePreviousErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := BuildCanonicalRenderIR(parseBuildAgent3BenchRenderOutput("after", "idle"))
	if parseNextErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parseB.Run("local", func(parseB *testing.B) {
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, parseErr := BuildCanonicalRenderIR(parseLocalRenderOutput); parseErr != nil {
				parseB.Fatalf("BuildCanonicalRenderIR(local) returned error: %v", parseErr)
			}
		}
	})
	parseB.Run("worker-backed", func(parseB *testing.B) {
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			if _, _, parseErr := BuildCanonicalPatchStream("bench-region", 1, uint64(parseIndex+2), uint64(parseIndex+2), parsePreviousIR, parseNextIR); parseErr != nil {
				parseB.Fatalf("BuildCanonicalPatchStream(worker-backed) returned error: %v", parseErr)
			}
		}
	})
}

// BenchmarkHandleHostRegionUpdateDispatchLoop benchmarks one host update-dispatch loop and reports latency percentiles.
func BenchmarkHandleHostRegionUpdateDispatchLoop(parseB *testing.B) {
	getHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(
		RegionInstanceID("bench-region"),
		[]SchedulerShardID{"shard-a"},
	)
	if parseBuildErr != nil {
		parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	_, parseMountErr := getHostRegionAdapter.HandleHostRegionMount(
		ParallelRegionSpec{
			RendererID:       RendererID("dashboard.hot-panel"),
			RegionInstanceID: RegionInstanceID("bench-region"),
			Props: map[string]any{
				"tick": 0,
			},
		},
		1,
	)
	if parseMountErr != nil {
		parseB.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	getProps := map[string]any{
		"tick": 0,
	}
	getSpec := ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("bench-region"),
		Props:            getProps,
	}
	getLatencySamples := buildAgent3UpdateDispatchLatencySampleSlice(parseB.N)
	getLatencySpanSize := 4096
	getLatencySpanStart := time.Now()
	getLatencySpanCount := 0
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
		getProps["tick"] = parseIndex
		_, parseDispatchErr := getHostRegionAdapter.HandleHostRegionUpdateDispatch(getSpec, uint64(parseIndex+2))
		if parseDispatchErr != nil {
			parseB.Fatalf("HandleHostRegionUpdateDispatch returned error: %v", parseDispatchErr)
		}
		getLatencySpanCount++
		if getLatencySpanCount == getLatencySpanSize {
			getLatencySamples = storeAgent3UpdateDispatchLatencySample(
				getLatencySamples,
				time.Since(getLatencySpanStart).Nanoseconds()/int64(getLatencySpanCount),
			)
			getLatencySpanStart = time.Now()
			getLatencySpanCount = 0
		}
	}
	if getLatencySpanCount > 0 {
		getLatencySamples = storeAgent3UpdateDispatchLatencySample(
			getLatencySamples,
			time.Since(getLatencySpanStart).Nanoseconds()/int64(getLatencySpanCount),
		)
	}
	parseB.StopTimer()
	reportAgent3UpdateDispatchLatencyPercentiles(parseB, getLatencySamples)
}
