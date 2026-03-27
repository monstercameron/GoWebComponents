package runtime2

import (
	"fmt"
	"strconv"
	"testing"
)

var storeHostWorkerOrchestrationBenchmarkSink HostWorkerRegionUpdateOrchestrationResult

// buildHostWorkerOrchestrationBenchFixture stores one reusable host-worker orchestration benchmark fixture.
type buildHostWorkerOrchestrationBenchFixture struct {
	getHostRegionAdapter   *HostRegionAdapter
	getWorkerRegionRuntime *WorkerRegionRuntime
	getCapabilityReport    CapabilityReport
}

// buildHostWorkerOrchestrationBenchmarkFixture builds one mounted host+worker fixture used by orchestration compare benchmarks.
func buildHostWorkerOrchestrationBenchmarkFixture(parseB *testing.B) buildHostWorkerOrchestrationBenchFixture {
	parseB.Helper()
	buildHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("bench-region"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseB.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	getMountSpec := ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("bench-region"),
		Props:            map[string]any{"title": "seed"},
	}
	if _, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(getMountSpec, 1); parseMountErr != nil {
		parseB.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	buildWorkerRegionRuntime := BuildWorkerRegionRuntime()
	if parseRegisterErr := buildWorkerRegionRuntime.RegisterWorkerRegionRenderer(
		"dashboard.hot-panel",
		func(parseMount WorkerRegionMountSpec) (any, error) {
			parseProps, _ := parseMount.Snapshot.Props.(map[string]any)
			return map[string]any{
				"kind": "text",
				"text": parseProps["title"],
			}, nil
		},
	); parseRegisterErr != nil {
		parseB.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	buildInitialSnapshot, parseSnapshotErr := BuildSnapshotEnvelope(
		"bench-region",
		1,
		1,
		map[string]any{"title": "seed"},
		nil,
		nil,
		nil,
	)
	if parseSnapshotErr != nil {
		parseB.Fatalf("BuildSnapshotEnvelope returned error: %v", parseSnapshotErr)
	}
	buildInitialIR, parseInitialIRErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "seed"})
	if parseInitialIRErr != nil {
		parseB.Fatalf("BuildCanonicalRenderIR(initial) returned error: %v", parseInitialIRErr)
	}
	parseSeedRegionDOMIndexFromCanonical(parseB, buildHostRegionAdapter.GetHostRegionDOMIndex(), "bench-region", buildInitialIR)
	if _, parseWorkerMountErr := buildWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "bench-region",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     buildInitialSnapshot,
	}); parseWorkerMountErr != nil {
		parseB.Fatalf("HandleWorkerRegionMount returned error: %v", parseWorkerMountErr)
	}
	return buildHostWorkerOrchestrationBenchFixture{
		getHostRegionAdapter:   buildHostRegionAdapter,
		getWorkerRegionRuntime: buildWorkerRegionRuntime,
		getCapabilityReport: CapabilityReport{
			HasWorkerSupport:          true,
			HasStructuredCloneSupport: true,
		},
	}
}

// handleHostWorkerRegionUpdateOrchestrationLegacy preserves the previous orchestration path that always decoded snapshot transport before worker update.
func handleHostWorkerRegionUpdateOrchestrationLegacy(
	parseHostRegionAdapter *HostRegionAdapter,
	parseWorkerRegionRuntime *WorkerRegionRuntime,
	parseSpec ParallelRegionSpec,
	parseInputVersion uint64,
	parseCapabilityReport CapabilityReport,
	parseSharedSnapshotPage *SharedSnapshotPage,
	parseSharedPatchPage *SharedPatchPage,
	parseDOMCommitter *DOMCommitter,
) (HostWorkerRegionUpdateOrchestrationResult, error) {
	if parseHostRegionAdapter == nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, fmt.Errorf("runtime2: host region adapter is nil")
	}
	if parseWorkerRegionRuntime == nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, fmt.Errorf("runtime2: worker region runtime is nil")
	}
	parseDispatchResult, parseDispatchErr := parseHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		parseSpec,
		parseInputVersion,
		parseCapabilityReport,
		parseSharedSnapshotPage,
	)
	if parseDispatchErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parseDispatchErr
	}
	parseResult := HostWorkerRegionUpdateOrchestrationResult{
		GetDispatchResult: parseDispatchResult,
	}
	if !parseDispatchResult.GetDispatchResult.HasScheduled {
		return parseResult, nil
	}
	if !parseDispatchResult.HasSnapshotTransport {
		return HostWorkerRegionUpdateOrchestrationResult{}, fmt.Errorf("runtime2: snapshot transport result is required for scheduled dispatch")
	}
	parseSnapshotEnvelope, parseSnapshotErr := parseDecodeSnapshotForOrchestration(
		parseDispatchResult.GetSnapshotTransportResult,
		parseSharedSnapshotPage,
	)
	if parseSnapshotErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parseSnapshotErr
	}
	parseWorkerUpdateResult, parseWorkerUpdateErr := parseWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		WorkerRegionUpdateSpec{
			RegionID:     string(parseSpec.RegionInstanceID),
			RendererID:   string(parseSpec.RendererID),
			Epoch:        parseSnapshotEnvelope.Epoch,
			InputVersion: parseSnapshotEnvelope.InputVersion,
			Snapshot:     parseSnapshotEnvelope,
		},
		parseCapabilityReport,
	)
	if parseWorkerUpdateErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parseWorkerUpdateErr
	}
	parseResult.GetWorkerUpdateResult = parseWorkerUpdateResult
	if !parseWorkerUpdateResult.HasPatchPayload {
		return parseResult, nil
	}
	parsePatchConsumeResult, parsePatchConsumeErr := parseHostRegionAdapter.HandleHostRegionPatchConsume(
		parseWorkerUpdateResult.GetTransportTier,
		parseWorkerUpdateResult.GetPatchPayload,
		parseSharedPatchPage,
		parseDOMCommitter,
	)
	if parsePatchConsumeErr != nil {
		return HostWorkerRegionUpdateOrchestrationResult{}, parsePatchConsumeErr
	}
	parseResult.GetPatchConsumeResult = parsePatchConsumeResult
	parseResult.HasPatchCommitted = parsePatchConsumeResult.GetCommitResult.HasCommitted
	return parseResult, nil
}

// BenchmarkHandleHostWorkerRegionUpdateOrchestrationCurrentVsLegacy compares current in-process snapshot-envelope reuse against the previous always-decode orchestration path.
func BenchmarkHandleHostWorkerRegionUpdateOrchestrationCurrentVsLegacy(parseB *testing.B) {
	parseB.Run("legacy_decode_every_update", func(parseB *testing.B) {
		getFixture := buildHostWorkerOrchestrationBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getSpec := ParallelRegionSpec{
				RendererID:       RendererID("dashboard.hot-panel"),
				RegionInstanceID: RegionInstanceID("bench-region"),
				Props: map[string]any{
					"title": "value-" + strconv.Itoa(parseIndex),
				},
			}
			getResult, parseErr := handleHostWorkerRegionUpdateOrchestrationLegacy(
				getFixture.getHostRegionAdapter,
				getFixture.getWorkerRegionRuntime,
				getSpec,
				uint64(parseIndex+2),
				getFixture.getCapabilityReport,
				nil,
				nil,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("handleHostWorkerRegionUpdateOrchestrationLegacy returned error: %v", parseErr)
			}
			storeHostWorkerOrchestrationBenchmarkSink = getResult
		}
	})
	parseB.Run("current_reuse_dispatch_snapshot", func(parseB *testing.B) {
		getFixture := buildHostWorkerOrchestrationBenchmarkFixture(parseB)
		parseB.ReportAllocs()
		parseB.ResetTimer()
		for parseIndex := 0; parseIndex < parseB.N; parseIndex++ {
			getSpec := ParallelRegionSpec{
				RendererID:       RendererID("dashboard.hot-panel"),
				RegionInstanceID: RegionInstanceID("bench-region"),
				Props: map[string]any{
					"title": "value-" + strconv.Itoa(parseIndex),
				},
			}
			getResult, parseErr := HandleHostWorkerRegionUpdateOrchestration(
				getFixture.getHostRegionAdapter,
				getFixture.getWorkerRegionRuntime,
				getSpec,
				uint64(parseIndex+2),
				getFixture.getCapabilityReport,
				nil,
				nil,
				nil,
			)
			if parseErr != nil {
				parseB.Fatalf("HandleHostWorkerRegionUpdateOrchestration returned error: %v", parseErr)
			}
			storeHostWorkerOrchestrationBenchmarkSink = getResult
		}
	})
}
