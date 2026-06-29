package runtime2

import "testing"

// TestParsePatchInsertOpValidInsertOpDecodes verifies valid insert payloads decode successfully.
func TestParsePatchInsertOpValidInsertOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
		2: {},
	}
	parseRawOp := PatchInsertOpRaw{
		ParentNodeID: 1,
		AnchorNodeID: 2,
		Node: RenderNodeRecordRaw{
			NodeID: 11,
			Kind:   uint8(RenderNodeKindHostElement),
		},
	}
	parseOp, parseErr := ParsePatchInsertOp(parseRawOp, parseKnownNodeIDs)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchInsertOp(valid) error = %v", parseErr)
	}
	if parseOp.ParentNodeID != 1 || parseOp.Node.NodeID != 11 {
		parseTesting.Fatalf("ParsePatchInsertOp(valid) unexpected payload %+v", parseOp)
	}
}

// TestParsePatchInsertOpMissingParentReferenceFails verifies missing parent references are rejected.
func TestParsePatchInsertOpMissingParentReferenceFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseRawOp := PatchInsertOpRaw{
		ParentNodeID: 5,
		Node: RenderNodeRecordRaw{
			NodeID: 11,
			Kind:   uint8(RenderNodeKindHostElement),
		},
	}
	_, parseErr := ParsePatchInsertOp(parseRawOp, parseKnownNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchInsertOp(missing parent) error = nil, want error")
	}
}

// TestParsePatchInsertOpInvalidSiblingAnchorFails verifies invalid sibling anchors are rejected.
func TestParsePatchInsertOpInvalidSiblingAnchorFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseRawOp := PatchInsertOpRaw{
		ParentNodeID: 1,
		AnchorNodeID: 7,
		Node: RenderNodeRecordRaw{
			NodeID: 11,
			Kind:   uint8(RenderNodeKindHostElement),
		},
	}
	_, parseErr := ParsePatchInsertOp(parseRawOp, parseKnownNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchInsertOp(invalid anchor) error = nil, want error")
	}
}
