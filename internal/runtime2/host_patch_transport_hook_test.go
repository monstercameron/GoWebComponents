package runtime2

import "testing"

// TestHandleHostRegionPatchConsumeStructuredCloneCommits verifies structured-clone patch payload decode feeds host patch commit.
func TestHandleHostRegionPatchConsumeStructuredCloneCommits(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parsePreviousOutput := map[string]any{"kind": "text", "text": "before"}
	parseNextOutput := map[string]any{"kind": "text", "text": "after"}
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseT, parsePreviousOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(parseT, "region-1", 1, 2, 2, parsePreviousOutput, parseNextOutput)
	parsePatchPayload, parsePatchPayloadErr := BuildStructuredClonePatchPayloadJSON(parsePatchStream)
	if parsePatchPayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchPayloadJSON returned error: %v", parsePatchPayloadErr)
	}
	parseConsumeResult, parseConsumeErr := buildHostRegionAdapter.HandleHostRegionPatchConsume(
		TransportTierStructuredClone,
		parsePatchPayload,
		nil,
		nil,
	)
	if parseConsumeErr != nil {
		parseT.Fatalf("HandleHostRegionPatchConsume returned error: %v", parseConsumeErr)
	}
	if !parseConsumeResult.GetCommitResult.HasCommitted {
		parseT.Fatalf("expected committed consume result, got %+v", parseConsumeResult)
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseT, parseNextOutput))
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

// TestHandleHostRegionPatchConsumeIgnoresStalePatchPayload verifies stale patch payload versions are ignored before DOM mutation.
func TestHandleHostRegionPatchConsumeIgnoresStalePatchPayload(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseBeforeOutput := map[string]any{"kind": "text", "text": "before"}
	parseAfterOutput := map[string]any{"kind": "text", "text": "after"}
	parseIgnoredOutput := map[string]any{"kind": "text", "text": "ignored"}
	parseBeforeIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseBeforeOutput)
	parseAfterIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseAfterOutput)
	parseIgnoredIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseIgnoredOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parseBeforeIR)
	parseFirstPatch, hasFirstNoOp, parseFirstPatchErr := BuildCanonicalPatchStream("region-1", 1, 2, 2, parseBeforeIR, parseAfterIR)
	if parseFirstPatchErr != nil {
		parseT.Fatalf("BuildCanonicalPatchStream(first) returned error: %v", parseFirstPatchErr)
	}
	if hasFirstNoOp {
		parseT.Fatal("expected first patch stream to emit ops")
	}
	parseFirstPayload, parseFirstPayloadErr := BuildStructuredClonePatchPayloadJSON(parseFirstPatch)
	if parseFirstPayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchPayloadJSON(first) returned error: %v", parseFirstPayloadErr)
	}
	parseFirstConsumeResult, parseFirstConsumeErr := buildHostRegionAdapter.HandleHostRegionPatchConsume(
		TransportTierStructuredClone,
		parseFirstPayload,
		nil,
		nil,
	)
	if parseFirstConsumeErr != nil {
		parseT.Fatalf("HandleHostRegionPatchConsume(first) returned error: %v", parseFirstConsumeErr)
	}
	if !parseFirstConsumeResult.GetCommitResult.HasCommitted {
		parseT.Fatalf("expected first patch payload to commit, got %+v", parseFirstConsumeResult)
	}
	parseStalePatch, hasStaleNoOp, parseStalePatchErr := BuildCanonicalPatchStream("region-1", 1, 3, 1, parseAfterIR, parseIgnoredIR)
	if parseStalePatchErr != nil {
		parseT.Fatalf("BuildCanonicalPatchStream(stale) returned error: %v", parseStalePatchErr)
	}
	if hasStaleNoOp {
		parseT.Fatal("expected stale patch stream to emit ops")
	}
	parseStalePayload, parseStalePayloadErr := BuildStructuredClonePatchPayloadJSON(parseStalePatch)
	if parseStalePayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchPayloadJSON(stale) returned error: %v", parseStalePayloadErr)
	}
	parseStaleConsumeResult, parseStaleConsumeErr := buildHostRegionAdapter.HandleHostRegionPatchConsume(
		TransportTierStructuredClone,
		parseStalePayload,
		nil,
		nil,
	)
	if parseStaleConsumeErr != nil {
		parseT.Fatalf("HandleHostRegionPatchConsume(stale) returned error: %v", parseStaleConsumeErr)
	}
	if !parseStaleConsumeResult.GetCommitResult.HasIgnored {
		parseT.Fatalf("expected stale patch payload to be ignored, got %+v", parseStaleConsumeResult)
	}
	parseAfterTree, parseAfterTreeErr := ParseCanonicalRenderTree(parseAfterIR)
	if parseAfterTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(after) returned error: %v", parseAfterTreeErr)
	}
	parseRootNode := parseAfterTree.getNodeByID[parseAfterTree.getRootNodeID]
	parseDOMNode, parseLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseRootNode.getNodeID)
	if parseLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseLookupErr)
	}
	if parseDOMNode.GetText != "after" {
		parseT.Fatalf("GetRegionDOMNode(root) text = %q, want %q", parseDOMNode.GetText, "after")
	}
}

// TestHandleHostRegionPatchConsumeRejectsWrongRegionPayload verifies wrong-region patch payloads are rejected before DOM mutation.
func TestHandleHostRegionPatchConsumeRejectsWrongRegionPayload(parseT *testing.T) {
	buildHostRegionAdapter, parseAdapterErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseAdapterErr != nil {
		parseT.Fatalf("BuildHostRegionAdapter returned error: %v", parseAdapterErr)
	}
	_, parseMountErr := buildHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1)
	if parseMountErr != nil {
		parseT.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parseBeforeOutput := map[string]any{"kind": "text", "text": "before"}
	parseAfterOutput := map[string]any{"kind": "text", "text": "after"}
	parseBeforeIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseBeforeOutput)
	parseAfterIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseAfterOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parseBeforeIR)
	parseWrongRegionPatch, hasWrongRegionNoOp, parseWrongRegionPatchErr := BuildCanonicalPatchStream("region-other", 1, 2, 2, parseBeforeIR, parseAfterIR)
	if parseWrongRegionPatchErr != nil {
		parseT.Fatalf("BuildCanonicalPatchStream(wrong region) returned error: %v", parseWrongRegionPatchErr)
	}
	if hasWrongRegionNoOp {
		parseT.Fatal("expected wrong-region patch stream to emit ops")
	}
	parseWrongRegionPayload, parseWrongRegionPayloadErr := BuildStructuredClonePatchPayloadJSON(parseWrongRegionPatch)
	if parseWrongRegionPayloadErr != nil {
		parseT.Fatalf("BuildStructuredClonePatchPayloadJSON(wrong region) returned error: %v", parseWrongRegionPayloadErr)
	}
	_, parseConsumeErr := buildHostRegionAdapter.HandleHostRegionPatchConsume(
		TransportTierStructuredClone,
		parseWrongRegionPayload,
		nil,
		nil,
	)
	if parseConsumeErr == nil {
		parseT.Fatal("expected wrong-region patch payload to fail host consume")
	}
	parseBeforeTree, parseBeforeTreeErr := ParseCanonicalRenderTree(parseBeforeIR)
	if parseBeforeTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(before) returned error: %v", parseBeforeTreeErr)
	}
	parseRootNode := parseBeforeTree.getNodeByID[parseBeforeTree.getRootNodeID]
	parseDOMNode, parseLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseRootNode.getNodeID)
	if parseLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseLookupErr)
	}
	if parseDOMNode.GetText != "before" {
		parseT.Fatalf("GetRegionDOMNode(root) text = %q, want %q", parseDOMNode.GetText, "before")
	}
}
