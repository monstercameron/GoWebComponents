package runtime2

import "testing"

// TestGetHostRegionPatchLookupStateRebuildsAfterRegionMutation verifies cached known-node lookup state refreshes after the region DOM version changes.
func TestGetHostRegionPatchLookupStateRebuildsAfterRegionMutation(parseTesting *testing.T) {
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseTesting.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseTesting.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2},
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) returned error: %v", parseSetErr)
	}
	parseKnownNodeIDs, _ := parseHostRegionAdapter.getHostRegionPatchLookupState(false)
	if len(parseKnownNodeIDs) != 2 {
		parseTesting.Fatalf("getHostRegionPatchLookupState(initial) known count = %d, want 2", len(parseKnownNodeIDs))
	}
	if !parseHostRegionAdapter.hasHostRegionPatchLookupCache {
		parseTesting.Fatal("expected patch lookup cache after initial build")
	}
	parseInitialLookupVersion := parseHostRegionAdapter.storeHostRegionPatchLookupVersion
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-b",
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(new child) returned error: %v", parseSetErr)
	}
	parseKnownNodeIDs, _ = parseHostRegionAdapter.getHostRegionPatchLookupState(false)
	if _, hasKnownNodeID := parseKnownNodeIDs[3]; !hasKnownNodeID {
		parseTesting.Fatal("expected rebuilt patch lookup cache to include externally inserted node")
	}
	if parseHostRegionAdapter.storeHostRegionPatchLookupVersion <= parseInitialLookupVersion {
		parseTesting.Fatalf(
			"patch lookup version = %d, want greater than %d after rebuild",
			parseHostRegionAdapter.storeHostRegionPatchLookupVersion,
			parseInitialLookupVersion,
		)
	}
}

// TestApplyHostRegionPatchLookupTransactionUpdatesInsertAndInvalidatesRemove verifies incremental cache updates keep insert traffic hot and invalidate remove traffic.
func TestApplyHostRegionPatchLookupTransactionUpdatesInsertAndInvalidatesRemove(parseTesting *testing.T) {
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseTesting.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseTesting.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2},
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetNodeKey:      "row-a",
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) returned error: %v", parseSetErr)
	}
	parseKnownNodeIDs, parseSiblingCountByParent := parseHostRegionAdapter.getHostRegionPatchLookupState(true)
	if len(parseKnownNodeIDs) != 2 || parseSiblingCountByParent[1] != 1 {
		parseTesting.Fatalf(
			"getHostRegionPatchLookupState(seed) = known:%d siblings:%d, want known:2 siblings:1",
			len(parseKnownNodeIDs),
			parseSiblingCountByParent[1],
		)
	}
	parseHostRegionAdapter.GetHostRegionDOMIndex().storeRegionDOMMutationVersion("region-1")
	if !parseHostRegionAdapter.applyHostRegionPatchLookupTransaction(RegionPatchTransaction{
		GetRegionID: "region-1",
		GetOps: []RegionPatchOp{
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode: &RegionDOMNode{
					GetNodeID: 3,
					GetTag:    "li",
				},
			},
		},
	}) {
		parseTesting.Fatal("expected insert-only cache update to stay incremental")
	}
	if _, hasKnownNodeID := parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[3]; !hasKnownNodeID {
		parseTesting.Fatal("expected insert-only cache update to include inserted node ID")
	}
	if parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[1] != 2 {
		parseTesting.Fatalf(
			"insert-only sibling count = %d, want 2",
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[1],
		)
	}
	if parseHostRegionAdapter.applyHostRegionPatchLookupTransaction(RegionPatchTransaction{
		GetRegionID: "region-1",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindRemoveNode,
				GetNodeID: 2,
			},
		},
	}) {
		parseTesting.Fatal("expected remove-node cache update to invalidate incremental state")
	}
}

// TestCanHostRegionPatchLookupApplyTransactionSupportsInsertOnlyTraffic verifies the host patch cache fast path skips structural delta precomputation for insert-only traffic and still rejects destructive transactions.
func TestCanHostRegionPatchLookupApplyTransactionSupportsInsertOnlyTraffic(parseTesting *testing.T) {
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseTesting.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseTesting.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2},
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) returned error: %v", parseSetErr)
	}
	parseHostRegionAdapter.getHostRegionPatchLookupState(true)
	if !parseHostRegionAdapter.canHostRegionPatchLookupApplyTransaction(RegionPatchTransaction{
		GetRegionID: "region-1",
		GetOps: []RegionPatchOp{
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode:   &RegionDOMNode{GetNodeID: 3, GetTag: "li"},
			},
			{
				GetKind:    RegionPatchOpKindSetAttr,
				GetNodeID:  2,
				GetAttrKey: "class",
			},
		},
	}) {
		parseTesting.Fatal("expected insert-only transaction to stay on the incremental lookup-cache path")
	}
	if parseHostRegionAdapter.canHostRegionPatchLookupApplyTransaction(RegionPatchTransaction{
		GetRegionID: "region-1",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindRemoveNode,
				GetNodeID: 2,
			},
		},
	}) {
		parseTesting.Fatal("expected remove-node transaction to require structural lookup-delta handling")
	}
}

// TestApplyHostRegionPatchLookupTransactionUpdatesNestedInsertSiblingCounts verifies incremental cache updates keep sibling counts correct when one inserted parent receives children later in the same transaction.
func TestApplyHostRegionPatchLookupTransactionUpdatesNestedInsertSiblingCounts(parseTesting *testing.T) {
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseTesting.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseTesting.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2},
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(child) returned error: %v", parseSetErr)
	}
	parseHostRegionAdapter.getHostRegionPatchLookupState(true)
	parseHostRegionAdapter.GetHostRegionDOMIndex().storeRegionDOMMutationVersion("region-1")
	if !parseHostRegionAdapter.applyHostRegionPatchLookupTransaction(RegionPatchTransaction{
		GetRegionID: "region-1",
		GetOps: []RegionPatchOp{
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 1,
				GetInsertNode:   &RegionDOMNode{GetNodeID: 3, GetTag: "li"},
			},
			{
				GetKind:         RegionPatchOpKindInsertNode,
				GetParentNodeID: 3,
				GetInsertNode:   &RegionDOMNode{GetNodeID: 4, GetTag: "span"},
			},
		},
	}) {
		parseTesting.Fatal("expected nested insert transaction to stay on the incremental lookup-cache path")
	}
	for _, getKnownNodeID := range []uint64{3, 4} {
		if _, hasKnownNodeID := parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[getKnownNodeID]; !hasKnownNodeID {
			parseTesting.Fatalf("expected inserted node ID %d in incremental lookup cache", getKnownNodeID)
		}
	}
	if parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[1] != 2 {
		parseTesting.Fatalf(
			"root sibling count = %d, want 2 after nested insert transaction",
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[1],
		)
	}
	if parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[3] != 1 {
		parseTesting.Fatalf(
			"inserted parent sibling count = %d, want 1 after nested insert transaction",
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[3],
		)
	}
}

// TestHandleHostRegionPatchCommitKeepsLookupCacheAcrossRemoveNode verifies remove-node commits keep the lookup cache hot by applying one structural delta instead of invalidating the whole region cache.
func TestHandleHostRegionPatchCommitKeepsLookupCacheAcrossRemoveNode(parseTesting *testing.T) {
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseTesting.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseTesting.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	parsePreviousOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "ul",
				"children": []any{
					map[string]any{"kind": "host-element", "tag": "li", "key": "a", "children": []any{map[string]any{"kind": "text", "text": "a"}}},
					map[string]any{"kind": "host-element", "tag": "li", "key": "b", "children": []any{map[string]any{"kind": "text", "text": "b"}}},
					map[string]any{"kind": "host-element", "tag": "li", "key": "c", "children": []any{map[string]any{"kind": "text", "text": "c"}}},
				},
			},
		},
	}
	parseNextOutput := map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{
				"kind": "host-element",
				"tag":  "ul",
				"children": []any{
					map[string]any{"kind": "host-element", "tag": "li", "key": "a", "children": []any{map[string]any{"kind": "text", "text": "a"}}},
					map[string]any{"kind": "host-element", "tag": "li", "key": "c", "children": []any{map[string]any{"kind": "text", "text": "c"}}},
				},
			},
		},
	}
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseTesting, parsePreviousOutput)
	parseSeedRegionDOMIndexFromCanonical(parseTesting, parseHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	parseSeedKnownNodeIDs, parseSeedSiblingCountByParent := parseHostRegionAdapter.getHostRegionPatchLookupState(true)
	if len(parseSeedKnownNodeIDs) == 0 || len(parseSeedSiblingCountByParent) == 0 {
		parseTesting.Fatal("expected seeded patch lookup cache before remove-node commit")
	}
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(parseTesting, "region-1", 1, 2, 2, parsePreviousOutput, parseNextOutput)
	parseCommitResult, parseCommitErr := parseHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr != nil {
		parseTesting.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
	}
	if !parseCommitResult.HasCommitted {
		parseTesting.Fatalf("HandleHostRegionPatchCommit expected committed result, got %+v", parseCommitResult)
	}
	if !parseHostRegionAdapter.hasHostRegionPatchLookupCache {
		parseTesting.Fatal("expected lookup cache to remain available after remove-node commit")
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseTesting, parseNextOutput))
	if parseTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	for getNodeID := range parseNextTree.getNodeByID {
		if _, hasKnownNodeID := parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[getNodeID]; !hasKnownNodeID {
			parseTesting.Fatalf("expected known-node lookup cache to include next node ID %d", getNodeID)
		}
	}
	parsePreviousTree, parsePreviousTreeErr := ParseCanonicalRenderTree(parsePreviousIR)
	if parsePreviousTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(previous) returned error: %v", parsePreviousTreeErr)
	}
	for getNodeID := range parsePreviousTree.getNodeByID {
		if _, hasNextNodeID := parseNextTree.getNodeByID[getNodeID]; hasNextNodeID {
			continue
		}
		if _, hasKnownNodeID := parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[getNodeID]; hasKnownNodeID {
			parseTesting.Fatalf("expected removed node ID %d to be absent from lookup cache after commit", getNodeID)
		}
	}
	var getListNodeID uint64
	for getNodeID, getNode := range parseNextTree.getNodeByID {
		if getNode.getTag == "ul" {
			getListNodeID = getNodeID
			break
		}
	}
	if getListNodeID == 0 {
		parseTesting.Fatal("expected next canonical tree to contain one list node")
	}
	if parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[getListNodeID] != 2 {
		parseTesting.Fatalf(
			"list sibling count = %d, want 2 after remove-node commit",
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[getListNodeID],
		)
	}
}

// TestBuildHostRegionPatchLookupDeltaRemoveOnlyClearsNestedSiblings verifies the remove-only lookup-delta fast path keeps cache state correct when nested targets collapse one parent subtree and empty the remaining root sibling list.
func TestBuildHostRegionPatchLookupDeltaRemoveOnlyClearsNestedSiblings(parseTesting *testing.T) {
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseTesting.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseTesting.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 1, &RegionDOMNode{
		GetNodeID:       1,
		GetTag:          "ul",
		GetChildNodeIDs: []uint64{2, 4},
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(root) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 2, &RegionDOMNode{
		GetNodeID:       2,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetChildNodeIDs: []uint64{3},
		GetNodeKey:      "row-a",
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(parent child) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 3, &RegionDOMNode{
		GetNodeID:       3,
		GetTag:          "span",
		GetParentNodeID: 2,
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(descendant) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 4, &RegionDOMNode{
		GetNodeID:       4,
		GetTag:          "li",
		GetParentNodeID: 1,
		GetChildNodeIDs: []uint64{5},
		GetNodeKey:      "row-b",
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(sibling child) returned error: %v", parseSetErr)
	}
	if parseSetErr := parseHostRegionAdapter.GetHostRegionDOMIndex().SetRegionDOMNode("region-1", 5, &RegionDOMNode{
		GetNodeID:       5,
		GetTag:          "span",
		GetParentNodeID: 4,
	}); parseSetErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(sibling descendant) returned error: %v", parseSetErr)
	}
	parseKnownNodeIDs, parseSiblingCountByParent := parseHostRegionAdapter.getHostRegionPatchLookupState(true)
	if len(parseKnownNodeIDs) != 5 || parseSiblingCountByParent[1] != 2 || parseSiblingCountByParent[2] != 1 || parseSiblingCountByParent[4] != 1 {
		parseTesting.Fatalf(
			"getHostRegionPatchLookupState(seed) = known:%d siblings:%v, want known:5 root:2 parent:1 sibling:1",
			len(parseKnownNodeIDs),
			parseSiblingCountByParent,
		)
	}
	parseTransaction := RegionPatchTransaction{
		GetRegionID: "region-1",
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 3},
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 2},
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 4},
		},
	}
	parseLookupDelta, hasLookupDelta := parseHostRegionAdapter.buildHostRegionPatchLookupDelta(parseTransaction)
	if !hasLookupDelta {
		parseTesting.Fatal("expected remove-only lookup delta fast path")
	}
	parseDOMCommitter := BuildDOMCommitter(parseHostRegionAdapter.GetHostRegionDOMIndex())
	if _, parseCommitErr := parseDOMCommitter.CommitRegionPatchTransaction(parseTransaction); parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(remove-only) returned error: %v", parseCommitErr)
	}
	if !parseHostRegionAdapter.applyHostRegionPatchLookupDelta(parseLookupDelta) {
		parseTesting.Fatal("expected remove-only lookup delta apply to succeed")
	}
	if !parseHostRegionAdapter.hasHostRegionPatchLookupCache {
		parseTesting.Fatal("expected lookup cache to remain hot after remove-only delta apply")
	}
	if len(parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs) != 1 {
		parseTesting.Fatalf(
			"known-node lookup size = %d, want 1 after nested remove-only delta apply",
			len(parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs),
		)
	}
	if _, hasKnownNodeID := parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[1]; !hasKnownNodeID {
		parseTesting.Fatal("expected root node ID to remain in lookup cache after remove-only delta apply")
	}
	for _, getRemovedNodeID := range []uint64{2, 3, 4, 5} {
		if _, hasKnownNodeID := parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[getRemovedNodeID]; hasKnownNodeID {
			parseTesting.Fatalf("expected removed node ID %d to be absent from lookup cache after delta apply", getRemovedNodeID)
		}
	}
	if _, hasSiblingCount := parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[1]; hasSiblingCount {
		parseTesting.Fatalf(
			"root sibling count = %d, want cleared entry after removing every child",
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[1],
		)
	}
	if _, hasSiblingCount := parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[2]; hasSiblingCount {
		parseTesting.Fatalf(
			"removed parent sibling count = %d, want cleared entry after nested remove-only delta apply",
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[2],
		)
	}
	if _, hasSiblingCount := parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[4]; hasSiblingCount {
		parseTesting.Fatalf(
			"removed sibling parent count = %d, want cleared entry after remove-only delta apply",
			parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent[4],
		)
	}
}

// TestHandleHostRegionPatchCommitKeepsLookupCacheAcrossReplaceSubtree verifies replace-subtree commits retain a valid lookup cache for the replacement tree.
func TestHandleHostRegionPatchCommitKeepsLookupCacheAcrossReplaceSubtree(parseTesting *testing.T) {
	parseHostRegionAdapter, parseBuildErr := BuildHostRegionAdapter(RegionInstanceID("region-1"), []SchedulerShardID{"shard-a"})
	if parseBuildErr != nil {
		parseTesting.Fatalf("BuildHostRegionAdapter returned error: %v", parseBuildErr)
	}
	if _, parseMountErr := parseHostRegionAdapter.HandleHostRegionMount(ParallelRegionSpec{
		RendererID:       RendererID("dashboard.hot-panel"),
		RegionInstanceID: RegionInstanceID("region-1"),
	}, 1); parseMountErr != nil {
		parseTesting.Fatalf("HandleHostRegionMount returned error: %v", parseMountErr)
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
	parsePreviousIR := parseBuildCanonicalIRForHostPatchTest(parseTesting, parsePreviousOutput)
	parseSeedRegionDOMIndexFromCanonical(parseTesting, parseHostRegionAdapter.GetHostRegionDOMIndex(), "region-1", parsePreviousIR)
	parseSeedKnownNodeIDs, parseSeedSiblingCountByParent := parseHostRegionAdapter.getHostRegionPatchLookupState(true)
	if len(parseSeedKnownNodeIDs) == 0 || len(parseSeedSiblingCountByParent) == 0 {
		parseTesting.Fatal("expected seeded patch lookup cache before replace-subtree commit")
	}
	parsePatchStream := parseBuildPatchStreamForHostPatchTest(parseTesting, "region-1", 1, 2, 2, parsePreviousOutput, parseNextOutput)
	parseCommitResult, parseCommitErr := parseHostRegionAdapter.HandleHostRegionPatchCommit(parsePatchStream, nil)
	if parseCommitErr != nil {
		parseTesting.Fatalf("HandleHostRegionPatchCommit returned error: %v", parseCommitErr)
	}
	if !parseCommitResult.HasCommitted {
		parseTesting.Fatalf("HandleHostRegionPatchCommit expected committed result, got %+v", parseCommitResult)
	}
	if !parseHostRegionAdapter.hasHostRegionPatchLookupCache {
		parseTesting.Fatal("expected lookup cache to remain available after replace-subtree commit")
	}
	parseNextTree, parseTreeErr := ParseCanonicalRenderTree(parseBuildCanonicalIRForHostPatchTest(parseTesting, parseNextOutput))
	if parseTreeErr != nil {
		parseTesting.Fatalf("ParseCanonicalRenderTree(next) returned error: %v", parseTreeErr)
	}
	if len(parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs) != len(parseNextTree.getNodeByID) {
		parseTesting.Fatalf(
			"known-node lookup size = %d, want %d after replace-subtree commit",
			len(parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs),
			len(parseNextTree.getNodeByID),
		)
	}
	for getNodeID := range parseNextTree.getNodeByID {
		if _, hasKnownNodeID := parseHostRegionAdapter.storeHostRegionPatchKnownNodeIDs[getNodeID]; !hasKnownNodeID {
			parseTesting.Fatalf("expected replacement node ID %d in lookup cache", getNodeID)
		}
	}
	if len(parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent) != 0 {
		parseTesting.Fatalf(
			"replacement sibling-count cache size = %d, want 0 for text-only replacement",
			len(parseHostRegionAdapter.storeHostRegionPatchSiblingCountByParent),
		)
	}
}
