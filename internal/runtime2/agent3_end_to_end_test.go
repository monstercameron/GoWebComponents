package runtime2_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/runtime2"
)

// TestStructuredCloneEndToEndMountAndUpdate verifies one display-only region mounts and updates through structured-clone envelopes.
func TestStructuredCloneEndToEndMountAndUpdate(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := runtime2.BuildHostRegionAdapter(runtime2.RegionInstanceID("region-1"), []runtime2.SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	buildSnapshotEnvelope, parseSnapshotErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Props:            map[string]any{"title": "Orders"},
	}, 1)
	if parseSnapshotErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot returned error: %v", parseSnapshotErr)
	}
	buildMountPayload, parseBuildMountErr := runtime2.BuildStructuredCloneMountEnvelopeJSON(runtime2.StructuredCloneMountEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		Snapshot:         buildSnapshotEnvelope,
	})
	if parseBuildMountErr != nil {
		parseT.Fatalf("BuildStructuredCloneMountEnvelopeJSON returned error: %v", parseBuildMountErr)
	}
	parseMountEnvelope, parseMountEnvelopeErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(buildMountPayload)
	if parseMountEnvelopeErr != nil {
		parseT.Fatalf("ParseStructuredCloneMountEnvelopeJSON returned error: %v", parseMountEnvelopeErr)
	}
	buildWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	parseRegisterErr := buildWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
		return map[string]any{
			"kind": "host-element",
			"tag":  "div",
			"children": []any{
				map[string]any{"kind": "text", "text": parseMount.InputVersion},
			},
		}, nil
	})
	if parseRegisterErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	_, parseWorkerMountErr := buildWorkerRegionRuntime.HandleWorkerRegionMount(runtime2.WorkerRegionMountSpec{
		RegionID:     string(parseMountEnvelope.RegionInstanceID),
		RendererID:   string(parseMountEnvelope.RendererID),
		Epoch:        parseMountEnvelope.Snapshot.Epoch,
		InputVersion: parseMountEnvelope.Snapshot.InputVersion,
	})
	if parseWorkerMountErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseWorkerMountErr)
	}
	buildUpdateSnapshotEnvelope, parseUpdateSnapshotErr := buildHostRegionAdapter.HandleHostRegionUpdateSnapshot(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Props:            map[string]any{"title": "Orders"},
	}, 2)
	if parseUpdateSnapshotErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateSnapshot(second) returned error: %v", parseUpdateSnapshotErr)
	}
	buildUpdatePayload, parseBuildUpdateErr := runtime2.BuildStructuredCloneUpdateEnvelopeJSON(runtime2.StructuredCloneUpdateEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		InputVersion:     2,
		Snapshot:         buildUpdateSnapshotEnvelope,
	})
	if parseBuildUpdateErr != nil {
		parseT.Fatalf("BuildStructuredCloneUpdateEnvelopeJSON returned error: %v", parseBuildUpdateErr)
	}
	parseUpdateEnvelope, parseUpdateEnvelopeErr := runtime2.ParseStructuredCloneUpdateEnvelopeJSON(buildUpdatePayload)
	if parseUpdateEnvelopeErr != nil {
		parseT.Fatalf("ParseStructuredCloneUpdateEnvelopeJSON returned error: %v", parseUpdateEnvelopeErr)
	}
	parseUpdateResult, parseWorkerUpdateErr := buildWorkerRegionRuntime.HandleWorkerRegionUpdate(runtime2.WorkerRegionUpdateSpec{
		RegionID:     string(parseUpdateEnvelope.RegionInstanceID),
		Epoch:        parseUpdateEnvelope.Snapshot.Epoch,
		InputVersion: parseUpdateEnvelope.InputVersion,
	})
	if parseWorkerUpdateErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdate returned error: %v", parseWorkerUpdateErr)
	}
	if !parseUpdateResult.HasPatchReady {
		parseT.Fatalf("expected structured-clone update to produce patch-ready result, got %+v", parseUpdateResult)
	}
}

// TestStructuredCloneEndToEndNoOpUpdate verifies no-op updates produce explicit no-op outcomes.
func TestStructuredCloneEndToEndNoOpUpdate(parseT *testing.T) {
	buildWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	parseRegisterErr := buildWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount runtime2.WorkerRegionMountSpec) (any, error) {
		return map[string]any{"kind": "text", "text": "stable"}, nil
	})
	if parseRegisterErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	_, parseMountErr := buildWorkerRegionRuntime.HandleWorkerRegionMount(runtime2.WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
	})
	if parseMountErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseMountErr)
	}
	parseUpdateResult, parseUpdateErr := buildWorkerRegionRuntime.HandleWorkerRegionUpdate(runtime2.WorkerRegionUpdateSpec{
		RegionID:     "region-1",
		Epoch:        1,
		InputVersion: 2,
	})
	if parseUpdateErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdate returned error: %v", parseUpdateErr)
	}
	if !parseUpdateResult.IsNoOp || parseUpdateResult.HasPatchReady {
		parseT.Fatalf("expected no-op update outcome, got %+v", parseUpdateResult)
	}
}

// TestEndToEndCancelSuppressesOutdatedCommit verifies outdated worker output does not commit after newer owner precedence.
func TestEndToEndCancelSuppressesOutdatedCommit(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := runtime2.BuildHostRegionAdapter(runtime2.RegionInstanceID("region-1"), []runtime2.SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if parseOwnerErr := buildHostRegionAdapter.HandleHostRegionOwnerRerender(3); parseOwnerErr != nil {
		parseT.Fatalf("HandleHostRegionOwnerRerender returned error: %v", parseOwnerErr)
	}
	parseWorkerOutputResult, parseWorkerOutputErr := buildHostRegionAdapter.HandleHostRegionWorkerOutput(2)
	if parseWorkerOutputErr != nil {
		parseT.Fatalf("HandleHostRegionWorkerOutput returned error: %v", parseWorkerOutputErr)
	}
	if !parseWorkerOutputResult.HasIgnored {
		parseT.Fatalf("expected outdated worker output to be ignored, got %+v", parseWorkerOutputResult)
	}
}

// TestEndToEndDisposeCleansRegionStateAndDOMIndex verifies dispose clears coordinator and DOM index state.
func TestEndToEndDisposeCleansRegionStateAndDOMIndex(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := runtime2.BuildHostRegionAdapter(runtime2.RegionInstanceID("region-1"), []runtime2.SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if parseSetErr := buildHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 1, &runtime2.RegionDOMNode{GetNodeID: 1}); parseSetErr != nil {
		parseT.Fatalf("SetRegionDOMNode returned error: %v", parseSetErr)
	}
	parseDisposeResult, parseDisposeErr := buildHostRegionAdapter.HandleHostRegionDispose()
	if parseDisposeErr != nil {
		parseT.Fatalf("HandleHostRegionDispose returned error: %v", parseDisposeErr)
	}
	if !parseDisposeResult.HasCoordinatorDisposed || parseDisposeResult.GetClearedDOMNodeCount == 0 {
		parseT.Fatalf("expected dispose cleanup, got %+v", parseDisposeResult)
	}
}

// TestEndToEndStickyAffinityRepeatedUpdatesStayOnOneShard verifies repeated updates keep scheduler shard affinity.
func TestEndToEndStickyAffinityRepeatedUpdatesStayOnOneShard(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := runtime2.BuildHostRegionAdapter(runtime2.RegionInstanceID("region-1"), []runtime2.SchedulerShardID{"shard-a", "shard-b", "shard-c"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseFirstUpdateResult, parseFirstUpdateErr := buildHostRegionAdapter.HandleHostRegionUpdate(2)
	if parseFirstUpdateErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate(first) returned error: %v", parseFirstUpdateErr)
	}
	parseSecondUpdateResult, parseSecondUpdateErr := buildHostRegionAdapter.HandleHostRegionUpdate(3)
	if parseSecondUpdateErr != nil {
		parseT.Fatalf("HandleHostRegionUpdate(second) returned error: %v", parseSecondUpdateErr)
	}
	if parseFirstUpdateResult.GetSchedulerJob.GetSchedulerShardID != parseSecondUpdateResult.GetSchedulerJob.GetSchedulerShardID {
		parseT.Fatalf(
			"expected sticky shard affinity, got first=%q second=%q",
			parseFirstUpdateResult.GetSchedulerJob.GetSchedulerShardID,
			parseSecondUpdateResult.GetSchedulerJob.GetSchedulerShardID,
		)
	}
}

// TestDuplicateMountForActiveRegionFails verifies duplicate mount attempts fail while region is active.
func TestDuplicateMountForActiveRegionFails(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := runtime2.BuildHostRegionAdapter(runtime2.RegionInstanceID("region-1"), []runtime2.SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount(first) returned error: %v", parseMountErr)
	}
	_, parseSecondMountErr := buildHostRegionAdapter.HandleHostRegionMount(runtime2.ParallelRegionSpec{
		RendererID:       runtime2.RendererID("dashboard.hot-panel"),
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
	}, 1)
	if parseSecondMountErr == nil {
		parseT.Fatal("expected duplicate mount to fail")
	}
}

// TestUnknownRendererThroughRuntimePathFails verifies unknown renderer IDs fail through structured-clone mount to worker runtime.
func TestUnknownRendererThroughRuntimePathFails(parseT *testing.T) {
	buildSnapshot := runtime2.SnapshotEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		Epoch:            1,
		InputVersion:     1,
	}
	buildMountPayload, parseBuildErr := runtime2.BuildStructuredCloneMountEnvelopeJSON(runtime2.StructuredCloneMountEnvelope{
		RegionInstanceID: runtime2.RegionInstanceID("region-1"),
		RendererID:       runtime2.RendererID("unknown.renderer"),
		Snapshot:         buildSnapshot,
	})
	if parseBuildErr != nil {
		parseT.Fatalf("BuildStructuredCloneMountEnvelopeJSON returned error: %v", parseBuildErr)
	}
	parseMountEnvelope, parseParseErr := runtime2.ParseStructuredCloneMountEnvelopeJSON(buildMountPayload)
	if parseParseErr != nil {
		parseT.Fatalf("ParseStructuredCloneMountEnvelopeJSON returned error: %v", parseParseErr)
	}
	buildWorkerRegionRuntime := runtime2.BuildWorkerRegionRuntime()
	_, parseMountErr := buildWorkerRegionRuntime.HandleWorkerRegionMount(runtime2.WorkerRegionMountSpec{
		RegionID:     string(parseMountEnvelope.RegionInstanceID),
		RendererID:   string(parseMountEnvelope.RendererID),
		Epoch:        parseMountEnvelope.Snapshot.Epoch,
		InputVersion: parseMountEnvelope.Snapshot.InputVersion,
	})
	if parseMountErr == nil {
		parseT.Fatal("expected unknown renderer mount failure")
	}
}

// TestZeroChildRegionCanonicalIR verifies zero-child regions remain valid canonical trees.
func TestZeroChildRegionCanonicalIR(parseT *testing.T) {
	parseCanonicalIR, parseErr := runtime2.BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "div",
	})
	if parseErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR returned error: %v", parseErr)
	}
	if len(parseCanonicalIR.GetNodeRecords) != 1 {
		parseT.Fatalf("expected one node for zero-child region, got %d", len(parseCanonicalIR.GetNodeRecords))
	}
	if parseCanonicalIR.GetNodeRecords[0].ChildCount != 0 {
		parseT.Fatalf("expected zero child count, got %d", parseCanonicalIR.GetNodeRecords[0].ChildCount)
	}
}

// TestSingleTextNodeRegionCanonicalIR verifies single-text regions are encoded as text-root canonical trees.
func TestSingleTextNodeRegionCanonicalIR(parseT *testing.T) {
	parseCanonicalIR, parseErr := runtime2.BuildCanonicalRenderIR("hello")
	if parseErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR returned error: %v", parseErr)
	}
	if len(parseCanonicalIR.GetNodeRecords) != 1 {
		parseT.Fatalf("expected one node for single-text region, got %d", len(parseCanonicalIR.GetNodeRecords))
	}
	if runtime2.RenderNodeKind(parseCanonicalIR.GetNodeRecords[0].Kind) != runtime2.RenderNodeKindText {
		parseT.Fatalf("expected text root node kind, got %v", runtime2.RenderNodeKind(parseCanonicalIR.GetNodeRecords[0].Kind))
	}
}

// TestEmptyTextUpdateProducesPatch verifies empty-string text updates still produce valid patch streams.
func TestEmptyTextUpdateProducesPatch(parseT *testing.T) {
	parsePreviousIR, parsePreviousErr := runtime2.BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "before"})
	if parsePreviousErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousErr)
	}
	parseNextIR, parseNextErr := runtime2.BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": ""})
	if parseNextErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextErr)
	}
	parsePatchStream, hasNoOp, parsePatchErr := runtime2.BuildCanonicalPatchStream("region-1", 1, 2, 2, parsePreviousIR, parseNextIR)
	if parsePatchErr != nil {
		parseT.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseT.Fatal("expected empty-text update to produce non-no-op patch stream")
	}
	if len(parsePatchStream.GetOps) == 0 {
		parseT.Fatal("expected at least one patch op for empty-text update")
	}
}

// TestEmptyPropSetCanonicalExtraction verifies empty prop sets remain valid.
func TestEmptyPropSetCanonicalExtraction(parseT *testing.T) {
	parsePropRecords, _, parseErr := runtime2.BuildRenderPropRecordsFromRenderOutput(map[string]any{
		"kind":  "host-element",
		"tag":   "div",
		"props": map[string]any{},
	})
	if parseErr != nil {
		parseT.Fatalf("BuildRenderPropRecordsFromRenderOutput returned error: %v", parseErr)
	}
	if len(parsePropRecords) != 0 {
		parseT.Fatalf("expected empty prop records, got %d", len(parsePropRecords))
	}
}

// TestDuplicateKeySiblingCanonicalBuildFails verifies duplicate keyed siblings are rejected during canonical IR build.
func TestDuplicateKeySiblingCanonicalBuildFails(parseT *testing.T) {
	_, parseErr := runtime2.BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "ul",
		"children": []any{
			map[string]any{"kind": "host-element", "tag": "li", "key": "dup"},
			map[string]any{"kind": "host-element", "tag": "li", "key": "dup"},
		},
	})
	if parseErr == nil {
		parseT.Fatal("expected duplicate keyed siblings to fail canonical IR build")
	}
}
