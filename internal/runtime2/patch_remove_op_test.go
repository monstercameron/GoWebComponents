package runtime2

import "testing"

// TestParsePatchRemoveOpValidRemoveOpDecodes verifies valid remove payloads decode successfully.
func TestParsePatchRemoveOpValidRemoveOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		7: {},
	}
	parseRemovedNodeIDs := map[uint64]struct{}{}
	parseRawOp := PatchRemoveOpRaw{TargetNodeID: 7}
	parseOp, parseErr := ParsePatchRemoveOp(parseRawOp, parseKnownNodeIDs, parseRemovedNodeIDs)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchRemoveOp(valid) error = %v", parseErr)
	}
	if parseOp.TargetNodeID != 7 {
		parseTesting.Fatalf("ParsePatchRemoveOp(valid) target = %d, want %d", parseOp.TargetNodeID, 7)
	}
}

// TestParsePatchRemoveOpRemovingUnknownNodeFails verifies unknown remove targets are rejected.
func TestParsePatchRemoveOpRemovingUnknownNodeFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		7: {},
	}
	parseRemovedNodeIDs := map[uint64]struct{}{}
	parseRawOp := PatchRemoveOpRaw{TargetNodeID: 9}
	_, parseErr := ParsePatchRemoveOp(parseRawOp, parseKnownNodeIDs, parseRemovedNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveOp(unknown target) error = nil, want error")
	}
}

// TestParsePatchRemoveOpDuplicateRemoveFails verifies duplicate remove ops are rejected consistently.
func TestParsePatchRemoveOpDuplicateRemoveFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		7: {},
	}
	parseRemovedNodeIDs := map[uint64]struct{}{}
	parseRawOp := PatchRemoveOpRaw{TargetNodeID: 7}
	_, parseErr := ParsePatchRemoveOp(parseRawOp, parseKnownNodeIDs, parseRemovedNodeIDs)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchRemoveOp(first remove) error = %v", parseErr)
	}
	_, parseErr = ParsePatchRemoveOp(parseRawOp, parseKnownNodeIDs, parseRemovedNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveOp(duplicate remove) error = nil, want error")
	}
}
