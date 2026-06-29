package runtime2

import "testing"

// TestBinaryPatchPayloadEndToEndCommit verifies binary patch payload decode drives host patch commit end to end.
func TestBinaryPatchPayloadEndToEndCommit(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	parseSpec := ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
		Props:            map[string]any{"title": "after"},
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(parseSpec, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	buildWorkerRegionRuntime := BuildWorkerRegionRuntime()
	parseRegisterErr := buildWorkerRegionRuntime.RegisterWorkerRegionRenderer("dashboard.hot-panel", func(parseMount WorkerRegionMountSpec) (any, error) {
		parseProps, _ := parseMount.Snapshot.Props.(map[string]any)
		return map[string]any{
			"kind": "text",
			"text": parseProps["title"],
		}, nil
	})
	if parseRegisterErr != nil {
		parseT.Fatalf("RegisterWorkerRegionRenderer returned error: %v", parseRegisterErr)
	}
	parseInitialSnapshot, parseInitialSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		1,
		map[string]any{"title": "before"},
		nil,
		nil,
		nil,
	)
	if parseInitialSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope(initial) returned error: %v", parseInitialSnapshotErr)
	}
	parsePreviousIR, parsePreviousIRErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "before"})
	if parsePreviousIRErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(previous) returned error: %v", parsePreviousIRErr)
	}
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	_, parseWorkerMountErr := buildWorkerRegionRuntime.HandleWorkerRegionMount(WorkerRegionMountSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 1,
		Snapshot:     parseInitialSnapshot,
	})
	if parseWorkerMountErr != nil {
		parseT.Fatalf("HandleWorkerRegionMount returned error: %v", parseWorkerMountErr)
	}
	parseSnapshotEnvelope, parseSnapshotErr := BuildSnapshotEnvelope(
		"region-1",
		1,
		2,
		map[string]any{"title": "after"},
		nil,
		nil,
		nil,
	)
	if parseSnapshotErr != nil {
		parseT.Fatalf("BuildSnapshotEnvelope(update) returned error: %v", parseSnapshotErr)
	}
	parseWorkerUpdateResult, parseWorkerUpdateErr := buildWorkerRegionRuntime.HandleWorkerRegionUpdate(WorkerRegionUpdateSpec{
		RegionID:     "region-1",
		RendererID:   "dashboard.hot-panel",
		Epoch:        1,
		InputVersion: 2,
		Snapshot:     parseSnapshotEnvelope,
	})
	if parseWorkerUpdateErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdate returned error: %v", parseWorkerUpdateErr)
	}
	if !parseWorkerUpdateResult.HasPatchReady {
		parseT.Fatalf("expected patch-ready worker update, got %+v", parseWorkerUpdateResult)
	}
	parseBinaryPatchPayload, parseBinaryPatchErr := BuildBinaryPatchPayload(parseWorkerUpdateResult.PatchIR)
	if parseBinaryPatchErr != nil {
		parseT.Fatalf("BuildBinaryPatchPayload returned error: %v", parseBinaryPatchErr)
	}
	parseConsumeResult, parseConsumeErr := buildHostRegionAdapter.HandleHostRegionPatchConsume(
		TransportTierBinary,
		parseBinaryPatchPayload,
		nil,
		nil,
	)
	if parseConsumeErr != nil {
		parseT.Fatalf("HandleHostRegionPatchConsume returned error: %v", parseConsumeErr)
	}
	if !parseConsumeResult.GetCommitResult.HasCommitted {
		parseT.Fatalf("expected committed binary patch consume result, got %+v", parseConsumeResult)
	}
	parseNextIR, parseNextIRErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "after"})
	if parseNextIRErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR(next) returned error: %v", parseNextIRErr)
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseNextIR)
	if parseTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	parseRootNode := parseNextTree.getNodeByID[parseNextTree.getRootNodeID]
	parseDOMNode, parseLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseRootNode.getNodeID)
	if parseLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseLookupErr)
	}
	if parseDOMNode.GetText != "after" {
		parseT.Fatalf("GetRegionDOMNode(root) text = %q, want %q", parseDOMNode.GetText, "after")
	}
}
