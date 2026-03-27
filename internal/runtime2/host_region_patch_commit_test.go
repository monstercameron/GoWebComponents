package runtime2

import (
	"strings"
	"testing"
)

func parseBuildCanonicalIRForHostPatchTest(parseT testing.TB, parseRenderOutput any) CanonicalRenderIR {
	parseT.Helper()
	parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(parseRenderOutput)
	if parseCanonicalErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	return parseCanonicalIR
}

func parseSeedRegionDOMIndexFromCanonical(parseT testing.TB, parseDOMIndex *RegionDOMIndex, parseRegionID string, parseCanonicalIR CanonicalRenderIR) {
	parseT.Helper()
	parseTree, parseTreeErr := ParseCanonicalRenderTree(parseCanonicalIR)
	if parseTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree returned error: %v", parseTreeErr)
	}
	for getNodeID, getNode := range parseTree.getNodeByID {
		buildDOMNode := &RegionDOMNode{
			GetNodeID:       getNodeID,
			GetTag:          getNode.getTag,
			GetText:         getNode.getText,
			GetAttrByKey:    map[string]string{},
			GetChildNodeIDs: append([]uint64(nil), getNode.getChildNodeIDs...),
			GetParentNodeID: getNode.getParentNodeID,
			GetNodeKey:      getNode.getKey,
		}
		for getAttrKey, getAttrRecord := range getNode.getPropByKey {
			buildDOMNode.GetAttrByKey[getAttrKey] = getAttrRecord.Value
		}
		if parseSetErr := parseDOMIndex.SetRegionDOMNode(parseRegionID, getNodeID, buildDOMNode); parseSetErr != nil {
			parseT.Fatalf("SetRegionDOMNode(%d) returned error: %v", getNodeID, parseSetErr)
		}
	}
}

func parseBuildPatchStreamForHostPatchTest(
	parseT testing.TB,
	parseRegionID string,
	parseEpoch uint64,
	parseInputVersion uint64,
	parsePatchVersion uint64,
	parsePreviousOutput any,
	parseNextOutput any,
) PatchStreamRaw {
	parseT.Helper()
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseT, parsePreviousOutput)
	parseNextIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseNextOutput)
	parsePatchStream, hasNoOp, parsePatchErr := BuildCanonicalPatchStream(
		parseRegionID,
		parseEpoch,
		parseInputVersion,
		parsePatchVersion,
		parsePreviousIR,
		parseNextIR,
	)
	if parsePatchErr != nil {
		parseT.Fatalf("BuildCanonicalPatchStream returned error: %v", parsePatchErr)
	}
	if hasNoOp {
		parseT.Fatal("BuildCanonicalPatchStream returned no-op for differing payloads")
	}
	return parsePatchStream
}

// TestHandleHostRegionPatchCommitCommitsTransaction verifies host patch commit parses and commits one typed patch stream.
func TestHandleHostRegionPatchCommitCommitsTransaction(parseT *testing.T) {
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
	parseResult, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
	}
	if !parseResult.HasCommitted {
		parseT.Fatalf("HandleHostRegionPatchCommit expected committed result, got %+v", parseResult)
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseT, parseNextOutput))
	if parseTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	parseRootNode, hasRootNode := parseNextTree.getNodeByID[parseNextTree.getRootNodeID]
	if !hasRootNode {
		parseT.Fatal("expected root node in decoded canonical tree")
	}
	parseDOMNode, parseLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseRootNode.getNodeID)
	if parseLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseLookupErr)
	}
	if parseDOMNode.GetText != "after" {
		parseT.Fatalf("GetRegionDOMNode(root) text = %q, want %q", parseDOMNode.GetText, "after")
	}
}

// TestHandleHostRegionPatchCommitStalePatchVersionIgnored verifies stale patch versions are suppressed.
func TestHandleHostRegionPatchCommitStalePatchVersionIgnored(parseT *testing.T) {
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
	if _, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil); parseCommitErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit(first) returned error: %v", parseCommitErr)
	}
	parseStalePatchStream := parseBuildPatchStreamForHostPatchTest(parseT, "region-1", 1, 3, 1, parseNextOutput, map[string]any{"kind": "text", "text": "ignored"})
	parseResult, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parseStalePatchStream, nil)
	if parseCommitErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit(stale) returned error: %v", parseCommitErr)
	}
	if !parseResult.HasIgnored {
		parseT.Fatalf("HandleHostRegionPatchCommit(stale) expected ignored result, got %+v", parseResult)
	}
}

// TestHandleHostRegionPatchCommitRejectsRegionMismatch verifies region mismatch is rejected in host orchestration.
func TestHandleHostRegionPatchCommitRejectsRegionMismatch(parseT *testing.T) {
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
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(
		parseT,
		"region-other",
		1,
		2,
		2,
		map[string]any{"kind": "text", "text": "a"},
		map[string]any{"kind": "text", "text": "b"},
	)
	_, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr == nil {
		parseT.Fatal("expected region mismatch to fail host patch commit")
	}
	if !strings.Contains(parseCommitErr.Error(), "does not match expected") {
		parseT.Fatalf("expected region mismatch details, got %v", parseCommitErr)
	}
}

// TestHandleHostRegionPatchCommitRejectsEpochMismatch verifies epoch mismatch is rejected in host orchestration.
func TestHandleHostRegionPatchCommitRejectsEpochMismatch(parseT *testing.T) {
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
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(
		parseT,
		"region-1",
		9,
		2,
		2,
		map[string]any{"kind": "text", "text": "a"},
		map[string]any{"kind": "text", "text": "b"},
	)
	_, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr == nil {
		parseT.Fatal("expected epoch mismatch to fail host patch commit")
	}
	if !strings.Contains(parseCommitErr.Error(), "does not match expected epoch") {
		parseT.Fatalf("expected epoch mismatch details, got %v", parseCommitErr)
	}
}

// TestHandleHostRegionPatchCommitCommitsSetStyle verifies set-style patch ops commit normalized style updates.
func TestHandleHostRegionPatchCommitCommitsSetStyle(parseT *testing.T) {
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
	parsePreviousOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"style": map[string]any{
				"color": "red",
			},
		},
	}
	parseNextOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"style": map[string]any{
				"color": "blue",
			},
		},
	}
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseT, parsePreviousOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(parseT, "region-1", 1, 2, 2, parsePreviousOutput, parseNextOutput)
	parseResult, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
	}
	if !parseResult.HasCommitted {
		parseT.Fatalf("HandleHostRegionPatchCommit expected committed result, got %+v", parseResult)
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseT, parseNextOutput))
	if parseTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	parseRootNode, hasRootNode := parseNextTree.getNodeByID[parseNextTree.getRootNodeID]
	if !hasRootNode {
		parseT.Fatal("expected root node in decoded canonical tree")
	}
	parseDOMNode, parseLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseRootNode.getNodeID)
	if parseLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseLookupErr)
	}
	if parseDOMNode.GetAttrByKey["style"] != "color:blue" {
		parseT.Fatalf("GetRegionDOMNode(root) style = %q, want %q", parseDOMNode.GetAttrByKey["style"], "color:blue")
	}
}

// TestHandleHostRegionPatchCommitCommitsRemoveStyle verifies remove-style patch ops clear style attributes.
func TestHandleHostRegionPatchCommitCommitsRemoveStyle(parseT *testing.T) {
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
	parsePreviousOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": "card",
			"style": map[string]any{
				"color": "red",
			},
		},
	}
	parseNextOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"props": map[string]any{
			"class": "card",
		},
	}
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseT, parsePreviousOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(parseT, "region-1", 1, 2, 2, parsePreviousOutput, parseNextOutput)
	parseResult, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
	}
	if !parseResult.HasCommitted {
		parseT.Fatalf("HandleHostRegionPatchCommit expected committed result, got %+v", parseResult)
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseT, parseNextOutput))
	if parseTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	parseRootNode, hasRootNode := parseNextTree.getNodeByID[parseNextTree.getRootNodeID]
	if !hasRootNode {
		parseT.Fatal("expected root node in decoded canonical tree")
	}
	parseDOMNode, parseLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseRootNode.getNodeID)
	if parseLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseLookupErr)
	}
	if _, hasStyle := parseDOMNode.GetAttrByKey["style"]; hasStyle {
		parseT.Fatalf("GetRegionDOMNode(root) style should be removed, got %q", parseDOMNode.GetAttrByKey["style"])
	}
}

// TestHandleHostRegionPatchCommitCommitsReplaceSubtree verifies replace-subtree commits rebuild region DOM index state atomically.
func TestHandleHostRegionPatchCommitCommitsReplaceSubtree(parseT *testing.T) {
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
	parsePreviousOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{"kind": "text", "text": "before"},
		},
	}
	parseNextOutput := map[string]any{
		"kind": "text",
		"text": "after",
	}
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseT, parsePreviousOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(parseT, "region-1", 1, 2, 2, parsePreviousOutput, parseNextOutput)
	parseResult, parseCommitErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
	}
	if !parseResult.HasCommitted {
		parseT.Fatalf("HandleHostRegionPatchCommit expected committed result, got %+v", parseResult)
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseT, parseNextOutput))
	if parseTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	parseRegionNodeByID, hasRegionNodeByID := buildHostRegionAdapter.GetHostRegionDOMIndex().storeRegionDOMNodeByRegionID["region-1"]
	if !hasRegionNodeByID {
		parseT.Fatal("expected region DOM index entry after replace-subtree commit")
	}
	if len(parseRegionNodeByID) != len(parseNextTree.getNodeByID) {
		parseT.Fatalf("region DOM index node count = %d, want %d", len(parseRegionNodeByID), len(parseNextTree.getNodeByID))
	}
	for getNodeID, getExpectedNode := range parseNextTree.getNodeByID {
		getDOMNode, hasDOMNode := parseRegionNodeByID[getNodeID]
		if !hasDOMNode {
			parseT.Fatalf("missing node id %d after replace-subtree commit", getNodeID)
		}
		if getDOMNode.GetTag != getExpectedNode.getTag {
			parseT.Fatalf("node id %d tag = %q, want %q", getNodeID, getDOMNode.GetTag, getExpectedNode.getTag)
		}
		if getDOMNode.GetText != getExpectedNode.getText {
			parseT.Fatalf("node id %d text = %q, want %q", getNodeID, getDOMNode.GetText, getExpectedNode.getText)
		}
		if getDOMNode.GetParentNodeID != getExpectedNode.getParentNodeID {
			parseT.Fatalf("node id %d parent = %d, want %d", getNodeID, getDOMNode.GetParentNodeID, getExpectedNode.getParentNodeID)
		}
	}
}

// TestHandleHostRegionPatchCommitAcceptsGapPatchVersionAndRejectsStalePatchVersion verifies no-op input-version gaps still accept the next patch version while stale patch versions are ignored before DOM commit.
func TestHandleHostRegionPatchCommitAcceptsGapPatchVersionAndRejectsStalePatchVersion(parseT *testing.T) {
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
	parseBaselineOutput := map[string]any{"kind": "text", "text": "baseline"}
	parseCommittedOutput := map[string]any{"kind": "text", "text": "committed"}
	parseStaleOutput := map[string]any{"kind": "text", "text": "stale"}
	parseBaselineIR := parseBuildCanonicalIRForHostPatchTest(parseT, parseBaselineOutput)
	parseSeedRegionDOMIndexFromCanonical(parseT, buildHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parseBaselineIR)

	parseGapPatch := parseBuildPatchStreamForHostPatchTest(
		parseT,
		"region-1",
		1,
		4,
		2,
		parseBaselineOutput,
		parseCommittedOutput,
	)
	parseGapResult, parseGapErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parseGapPatch, nil)
	if parseGapErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit(gap patch) returned error: %v", parseGapErr)
	}
	if !parseGapResult.HasCommitted {
		parseT.Fatalf("expected gap patch to commit, got %+v", parseGapResult)
	}

	parseCommittedTree, parseCommittedTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseT, parseCommittedOutput))
	if parseCommittedTreeErr != nil {
		parseT.Fatalf("ParseCanonicalRenderTree(committed) returned error: %v", parseCommittedTreeErr)
	}
	parseCommittedRoot, hasCommittedRoot := parseCommittedTree.getNodeByID[parseCommittedTree.getRootNodeID]
	if !hasCommittedRoot {
		parseT.Fatal("expected committed root node in canonical tree")
	}
	parseCommittedDOMNode, parseCommittedLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseCommittedRoot.getNodeID)
	if parseCommittedLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(committed root) returned error: %v", parseCommittedLookupErr)
	}
	if parseCommittedDOMNode.GetText != "committed" {
		parseT.Fatalf("GetRegionDOMNode(committed root) text = %q, want %q", parseCommittedDOMNode.GetText, "committed")
	}

	parseStalePatch := parseBuildPatchStreamForHostPatchTest(
		parseT,
		"region-1",
		1,
		5,
		2,
		parseCommittedOutput,
		parseStaleOutput,
	)
	parseStaleResult, parseStaleErr := buildHostRegionAdapter.HandleHostRegionPatchCommit(parseStalePatch, nil)
	if parseStaleErr != nil {
		parseT.Fatalf("HandleHostRegionPatchCommit(stale patch version) returned error: %v", parseStaleErr)
	}
	if !parseStaleResult.HasIgnored {
		parseT.Fatalf("expected stale patch version to be ignored before commit, got %+v", parseStaleResult)
	}
	parsePostStaleDOMNode, parsePostStaleLookupErr := buildHostRegionAdapter.GetHostRegionDOMIndex().GetRegionDOMNode("region-1", parseCommittedRoot.getNodeID)
	if parsePostStaleLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(after stale patch) returned error: %v", parsePostStaleLookupErr)
	}
	if parsePostStaleDOMNode.GetText != "committed" {
		parseT.Fatalf("expected stale patch to leave committed DOM text unchanged, got %q", parsePostStaleDOMNode.GetText)
	}
}
