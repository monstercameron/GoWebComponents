package runtime2

import "testing"

// TestStructuredClonePatchPayloadEndToEndCommit verifies host commit is driven by decoded structured-clone patch payload bytes.
func TestStructuredClonePatchPayloadEndToEndCommit(parseT *testing.T) {
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
	parseDispatchResult, parseDispatchErr := buildHostRegionAdapter.HandleHostRegionUpdateDispatchWithTransport(
		parseSpec,
		2,
		CapabilityReport{
			HasWorkerSupport:          true,
			HasStructuredCloneSupport: true,
		},
		nil,
	)
	if parseDispatchErr != nil {
		parseT.Fatalf("HandleHostRegionUpdateDispatchWithTransport returned error: %v", parseDispatchErr)
	}
	if !parseDispatchResult.HasSnapshotTransport {
		parseT.Fatalf("expected snapshot transport for scheduled dispatch, got %+v", parseDispatchResult)
	}
	parseSnapshotEnvelope, parseSnapshotErr := ParseStructuredCloneSnapshotEnvelopeJSON(parseDispatchResult.GetSnapshotTransportResult.GetMessagePayload)
	if parseSnapshotErr != nil {
		parseT.Fatalf("ParseStructuredCloneSnapshotEnvelopeJSON returned error: %v", parseSnapshotErr)
	}
	parseWorkerUpdateResult, parseWorkerUpdateErr := buildWorkerRegionRuntime.HandleWorkerRegionUpdateWithPatchTransport(
		WorkerRegionUpdateSpec{
			RegionID:     "region-1",
			RendererID:   "dashboard.hot-panel",
			Epoch:        parseSnapshotEnvelope.Epoch,
			InputVersion: parseSnapshotEnvelope.InputVersion,
			Snapshot:     parseSnapshotEnvelope,
		},
		CapabilityReport{
			HasWorkerSupport:          true,
			HasStructuredCloneSupport: true,
		},
	)
	if parseWorkerUpdateErr != nil {
		parseT.Fatalf("HandleWorkerRegionUpdateWithPatchTransport returned error: %v", parseWorkerUpdateErr)
	}
	if !parseWorkerUpdateResult.HasPatchPayload {
		parseT.Fatalf("expected patch payload bytes from worker update, got %+v", parseWorkerUpdateResult)
	}
	parseConsumeResult, parseConsumeErr := buildHostRegionAdapter.HandleHostRegionPatchConsume(
		parseWorkerUpdateResult.GetTransportTier,
		parseWorkerUpdateResult.GetPatchPayload,
		nil,
		nil,
	)
	if parseConsumeErr != nil {
		parseT.Fatalf("HandleHostRegionPatchConsume returned error: %v", parseConsumeErr)
	}
	if !parseConsumeResult.GetCommitResult.HasCommitted {
		parseT.Fatalf("expected committed patch consume result, got %+v", parseConsumeResult)
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
