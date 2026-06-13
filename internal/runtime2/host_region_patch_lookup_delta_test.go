package runtime2

import "testing"

func assertUint64Set(t *testing.T, parseGot []uint64, parseWant ...uint64) {
	t.Helper()
	parseSeen := map[uint64]int{}
	for _, parseValue := range parseGot {
		parseSeen[parseValue]++
	}
	if len(parseSeen) != len(parseWant) {
		t.Fatalf("set = %#v, want %#v", parseGot, parseWant)
	}
	for _, parseValue := range parseWant {
		if parseSeen[parseValue] != 1 {
			t.Fatalf("set = %#v, want value %d once", parseGot, parseValue)
		}
	}
}

func buildHostRegionPatchLookupTestNodes() map[uint64]*RegionDOMNode {
	return map[uint64]*RegionDOMNode{
		1: {GetNodeID: 1, GetTag: "ul", GetChildNodeIDs: []uint64{2, 3}},
		2: {GetNodeID: 2, GetTag: "li", GetParentNodeID: 1, GetChildNodeIDs: []uint64{4}},
		3: {GetNodeID: 3, GetTag: "li", GetParentNodeID: 1},
		4: {GetNodeID: 4, GetText: "nested", GetParentNodeID: 2},
	}
}

func buildHostRegionPatchLookupTestAdapter() *HostRegionAdapter {
	parseIndex := BuildRegionDOMIndex()
	parseIndex.storeRegionDOMNodeByRegionID["region-a"] = buildHostRegionPatchLookupTestNodes()
	parseIndex.storeRegionDOMVersionByRegionID["region-a"] = 7
	return &HostRegionAdapter{
		storeRegionInstanceID:                    RegionInstanceID("region-a"),
		storeRegionDOMIndexHandle:                parseIndex,
		storeHostRegionPatchKnownNodeIDs:         map[uint64]struct{}{1: {}, 2: {}, 3: {}, 4: {}},
		storeHostRegionPatchSiblingCountByParent: map[uint64]uint32{1: 2, 2: 1},
		hasHostRegionPatchLookupCache:            true,
		hasHostRegionPatchSiblingCountCache:      true,
	}
}

func TestHostRegionPatchLookupPureHelpers(t *testing.T) {
	parseNodes := buildHostRegionPatchLookupTestNodes()
	assertUint64Set(t, buildRegionDOMSubtreeNodeIDs(parseNodes, 2), 2, 4)
	if parseGot := buildRegionDOMSubtreeNodeIDs(nil, 1); parseGot != nil {
		t.Fatalf("nil map subtree = %#v, want nil", parseGot)
	}
	if parseGot := buildRegionDOMSubtreeNodeIDs(parseNodes, 0); parseGot != nil {
		t.Fatalf("zero root subtree = %#v, want nil", parseGot)
	}

	if !hasHostRegionPatchLookupRemovedAncestor(parseNodes, map[uint64]struct{}{2: {}}, 4) {
		t.Fatal("expected node 4 to report removed ancestor 2")
	}
	if hasHostRegionPatchLookupRemovedAncestor(parseNodes, map[uint64]struct{}{3: {}}, 4) {
		t.Fatal("did not expect unrelated removed ancestor match")
	}
	if hasHostRegionPatchLookupRemovedAncestor(parseNodes, map[uint64]struct{}{2: {}}, 99) {
		t.Fatal("missing node should not report removed ancestor")
	}

	if hasHostRegionPatchLookupRemoveOnlyOps(nil) {
		t.Fatal("empty ops should not be remove-only")
	}
	if !hasHostRegionPatchLookupRemoveOnlyOps([]RegionPatchOp{{GetKind: RegionPatchOpKindRemoveNode}}) {
		t.Fatal("single remove op should be remove-only")
	}
	if hasHostRegionPatchLookupRemoveOnlyOps([]RegionPatchOp{{GetKind: RegionPatchOpKindRemoveNode}, {GetKind: RegionPatchOpKindSetText}}) {
		t.Fatal("mixed ops should not be remove-only")
	}
}

func TestBuildHostRegionPatchLookupRemoveOnlyDelta(t *testing.T) {
	parseAdapter := buildHostRegionPatchLookupTestAdapter()
	parseDelta, parseOK := parseAdapter.buildHostRegionPatchLookupRemoveOnlyDelta(RegionPatchTransaction{
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 2},
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 4},
		},
	}, buildHostRegionPatchLookupTestNodes())
	if !parseOK {
		t.Fatal("remove-only delta was not built")
	}
	assertUint64Set(t, parseDelta.getRemovedNodeIDs, 2, 4)
	assertUint64Set(t, parseDelta.getClearedSiblingParentIDs, 2)
	if parseDelta.getSiblingCountByParent[1] != 1 {
		t.Fatalf("parent 1 sibling count = %d, want 1", parseDelta.getSiblingCountByParent[1])
	}

	if _, parseOK := parseAdapter.buildHostRegionPatchLookupRemoveOnlyDelta(RegionPatchTransaction{GetOps: []RegionPatchOp{{GetKind: RegionPatchOpKindRemoveNode}}}, buildHostRegionPatchLookupTestNodes()); parseOK {
		t.Fatal("zero node ID remove op should reject delta")
	}
	if _, parseOK := (*HostRegionAdapter)(nil).buildHostRegionPatchLookupRemoveOnlyDelta(RegionPatchTransaction{GetOps: []RegionPatchOp{{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 2}}}, buildHostRegionPatchLookupTestNodes()); parseOK {
		t.Fatal("nil adapter should reject remove-only delta")
	}
}

func TestBuildAndApplyHostRegionPatchLookupDelta(t *testing.T) {
	parseAdapter := buildHostRegionPatchLookupTestAdapter()
	parseDelta, parseOK := parseAdapter.buildHostRegionPatchLookupDelta(RegionPatchTransaction{
		GetRegionID: "region-a",
		GetOps: []RegionPatchOp{
			{GetKind: RegionPatchOpKindSetText, GetNodeID: 3, GetText: "unchanged structurally"},
			{GetKind: RegionPatchOpKindInsertNode, GetParentNodeID: 1, GetInsertNode: &RegionDOMNode{GetNodeID: 5, GetTag: "li", GetParentNodeID: 1, GetChildNodeIDs: []uint64{6}}},
			{GetKind: RegionPatchOpKindRemoveNode, GetNodeID: 3},
		},
	})
	if !parseOK {
		t.Fatal("generic structural delta was not built")
	}
	assertUint64Set(t, parseDelta.getAddedNodeIDs, 5)
	assertUint64Set(t, parseDelta.getRemovedNodeIDs, 3)
	if parseDelta.getSiblingCountByParent[1] != 2 {
		t.Fatalf("parent 1 sibling count = %d, want 2 after insert+remove", parseDelta.getSiblingCountByParent[1])
	}
	if parseDelta.getSiblingCountByParent[5] != 1 {
		t.Fatalf("inserted parent 5 child count = %d, want 1", parseDelta.getSiblingCountByParent[5])
	}

	if !parseAdapter.applyHostRegionPatchLookupDelta(parseDelta) {
		t.Fatal("applyHostRegionPatchLookupDelta returned false")
	}
	if _, parseExists := parseAdapter.storeHostRegionPatchKnownNodeIDs[3]; parseExists {
		t.Fatal("removed node 3 still present in known-node cache")
	}
	if _, parseExists := parseAdapter.storeHostRegionPatchKnownNodeIDs[5]; !parseExists {
		t.Fatal("added node 5 missing from known-node cache")
	}
	if parseAdapter.storeHostRegionPatchSiblingCountByParent[1] != 2 || parseAdapter.storeHostRegionPatchSiblingCountByParent[5] != 1 {
		t.Fatalf("sibling cache = %#v, want updated parent counts", parseAdapter.storeHostRegionPatchSiblingCountByParent)
	}
	if parseAdapter.storeHostRegionPatchLookupVersion != 7 {
		t.Fatalf("lookup version = %d, want region mutation version 7", parseAdapter.storeHostRegionPatchLookupVersion)
	}

	if _, parseOK := parseAdapter.buildHostRegionPatchLookupDelta(RegionPatchTransaction{GetOps: []RegionPatchOp{{GetKind: RegionPatchOpKindSetAttr, GetNodeID: 1}}}); parseOK {
		t.Fatal("non-structural transaction should not build a delta")
	}
	if _, parseOK := parseAdapter.buildHostRegionPatchLookupDelta(RegionPatchTransaction{GetOps: []RegionPatchOp{{GetKind: regionPatchOpKindInvalid}}}); parseOK {
		t.Fatal("invalid op should reject delta")
	}
	if (*HostRegionAdapter)(nil).applyHostRegionPatchLookupDelta(parseDelta) {
		t.Fatal("nil adapter should not apply delta")
	}
}
