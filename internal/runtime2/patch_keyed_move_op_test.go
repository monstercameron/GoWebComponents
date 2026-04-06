package runtime2

import "testing"

// TestParsePatchKeyedMoveOpValidKeyedMoveDecodes verifies valid keyed-move payloads decode successfully.
func TestParsePatchKeyedMoveOpValidKeyedMoveDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
		3: {},
	}
	parseSiblingCountByParent := map[uint64]uint32{
		1: 4,
	}
	parseRawOp := PatchKeyedMoveOpRaw{
		ParentNodeID:     1,
		SourceNodeID:     3,
		DestinationIndex: 2,
	}
	parseOp, parseErr := ParsePatchKeyedMoveOp(parseRawOp, parseKnownNodeIDs, parseSiblingCountByParent)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchKeyedMoveOp(valid) error = %v", parseErr)
	}
	if parseOp.DestinationIndex != 2 {
		parseTesting.Fatalf("ParsePatchKeyedMoveOp(valid) destination = %d, want %d", parseOp.DestinationIndex, 2)
	}
}

// TestParsePatchKeyedMoveOpMissingSourceNodeFails verifies unknown source nodes are rejected.
func TestParsePatchKeyedMoveOpMissingSourceNodeFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseSiblingCountByParent := map[uint64]uint32{
		1: 4,
	}
	parseRawOp := PatchKeyedMoveOpRaw{
		ParentNodeID:     1,
		SourceNodeID:     7,
		DestinationIndex: 2,
	}
	_, parseErr := ParsePatchKeyedMoveOp(parseRawOp, parseKnownNodeIDs, parseSiblingCountByParent)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchKeyedMoveOp(missing source) error = nil, want error")
	}
}

// TestParsePatchKeyedMoveOpInvalidDestinationPositionFails verifies out-of-range destination indices are rejected.
func TestParsePatchKeyedMoveOpInvalidDestinationPositionFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
		3: {},
	}
	parseSiblingCountByParent := map[uint64]uint32{
		1: 2,
	}
	parseRawOp := PatchKeyedMoveOpRaw{
		ParentNodeID:     1,
		SourceNodeID:     3,
		DestinationIndex: 9,
	}
	_, parseErr := ParsePatchKeyedMoveOp(parseRawOp, parseKnownNodeIDs, parseSiblingCountByParent)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchKeyedMoveOp(invalid destination) error = nil, want error")
	}
}

// TestParsePatchKeyedMoveOpRejectsMissingParentSourceAndSiblingBounds verifies keyed-move parsing rejects the remaining required identity and bounds branches.
func TestParsePatchKeyedMoveOpRejectsMissingParentSourceAndSiblingBounds(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
		3: {},
	}

	if _, parseErr := ParsePatchKeyedMoveOp(PatchKeyedMoveOpRaw{
		SourceNodeID:     3,
		DestinationIndex: 1,
	}, parseKnownNodeIDs, map[uint64]uint32{1: 2}); parseErr == nil {
		parseTesting.Fatal("expected missing parent node ID to fail")
	}

	if _, parseErr := ParsePatchKeyedMoveOp(PatchKeyedMoveOpRaw{
		ParentNodeID:     7,
		SourceNodeID:     3,
		DestinationIndex: 1,
	}, parseKnownNodeIDs, map[uint64]uint32{1: 2}); parseErr == nil {
		parseTesting.Fatal("expected unknown parent node ID to fail")
	}

	if _, parseErr := ParsePatchKeyedMoveOp(PatchKeyedMoveOpRaw{
		ParentNodeID:     1,
		DestinationIndex: 1,
	}, parseKnownNodeIDs, map[uint64]uint32{1: 2}); parseErr == nil {
		parseTesting.Fatal("expected missing source node ID to fail")
	}

	if _, parseErr := ParsePatchKeyedMoveOp(PatchKeyedMoveOpRaw{
		ParentNodeID:     1,
		SourceNodeID:     3,
		DestinationIndex: 1,
	}, parseKnownNodeIDs, map[uint64]uint32{}); parseErr == nil {
		parseTesting.Fatal("expected missing sibling bounds to fail")
	}
}
