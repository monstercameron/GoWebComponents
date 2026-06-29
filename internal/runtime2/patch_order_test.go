package runtime2

import "testing"

// TestValidatePatchOrderValidParentBeforeChildSequencePasses verifies valid insert ordering passes.
func TestValidatePatchOrderValidParentBeforeChildSequencePasses(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseEntries := []PatchOrderEntry{
		{
			OpCode:       PatchOpCodeInsertNode,
			NodeID:       10,
			ParentNodeID: 1,
		},
		{
			OpCode:       PatchOpCodeInsertNode,
			NodeID:       11,
			ParentNodeID: 10,
		},
	}
	parseErr := ValidatePatchOrder(parseEntries, parseKnownNodeIDs)
	if parseErr != nil {
		parseTesting.Fatalf("ValidatePatchOrder(valid parent-before-child) error = %v", parseErr)
	}
}

// TestValidatePatchOrderChildInsertBeforeParentFails verifies child inserts before parent are rejected.
func TestValidatePatchOrderChildInsertBeforeParentFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseEntries := []PatchOrderEntry{
		{
			OpCode:       PatchOpCodeInsertNode,
			NodeID:       11,
			ParentNodeID: 10,
		},
		{
			OpCode:       PatchOpCodeInsertNode,
			NodeID:       10,
			ParentNodeID: 1,
		},
	}
	parseErr := ValidatePatchOrder(parseEntries, parseKnownNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ValidatePatchOrder(child-before-parent) error = nil, want error")
	}
}

// TestValidatePatchOrderMoveAfterRemoveSameNodeFails verifies moving a removed node is rejected.
func TestValidatePatchOrderMoveAfterRemoveSameNodeFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
		7: {},
	}
	parseEntries := []PatchOrderEntry{
		{
			OpCode: PatchOpCodeRemoveNode,
			NodeID: 7,
		},
		{
			OpCode:       PatchOpCodeMoveKeyedChild,
			SourceNodeID: 7,
			ParentNodeID: 1,
		},
	}
	parseErr := ValidatePatchOrder(parseEntries, parseKnownNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ValidatePatchOrder(move-after-remove) error = nil, want error")
	}
}

// TestValidatePatchOrderRejectsMissingRequiredIDs verifies insert and move entries fail fast when required IDs are missing.
func TestValidatePatchOrderRejectsMissingRequiredIDs(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	testCases := []struct {
		name         string
		parseEntries []PatchOrderEntry
	}{
		{
			name: "insert parent node id",
			parseEntries: []PatchOrderEntry{
				{
					OpCode: PatchOpCodeInsertNode,
					NodeID: 10,
				},
			},
		},
		{
			name: "insert node id",
			parseEntries: []PatchOrderEntry{
				{
					OpCode:       PatchOpCodeInsertNode,
					ParentNodeID: 1,
				},
			},
		},
		{
			name: "move source node id",
			parseEntries: []PatchOrderEntry{
				{
					OpCode:       PatchOpCodeMoveKeyedChild,
					ParentNodeID: 1,
				},
			},
		},
		{
			name: "remove node id",
			parseEntries: []PatchOrderEntry{
				{
					OpCode: PatchOpCodeRemoveNode,
				},
			},
		},
	}
	for _, parseTestCase := range testCases {
		parseTesting.Run(parseTestCase.name, func(parseSubTest *testing.T) {
			if parseErr := ValidatePatchOrder(parseTestCase.parseEntries, parseKnownNodeIDs); parseErr == nil {
				parseSubTest.Fatalf("ValidatePatchOrder(%s) error = nil, want error", parseTestCase.name)
			}
		})
	}
}
