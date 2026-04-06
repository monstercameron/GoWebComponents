package runtime2

import "testing"

// TestParsePatchRemoveStyleOpValidRemoveStyleOpDecodes verifies valid remove-style payloads decode successfully.
func TestParsePatchRemoveStyleOpValidRemoveStyleOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRawOp := PatchRemoveStyleOpRaw{
		TargetNodeID: 8,
	}
	parseOp, parseErr := ParsePatchRemoveStyleOp(parseRawOp, parseKnownNodeIDs)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchRemoveStyleOp(valid) error = %v", parseErr)
	}
	if parseOp.TargetNodeID != 8 {
		parseTesting.Fatalf("ParsePatchRemoveStyleOp(valid) target = %d, want %d", parseOp.TargetNodeID, 8)
	}
}

// TestParsePatchRemoveStyleOpUnknownTargetFails verifies unknown remove-style targets are rejected.
func TestParsePatchRemoveStyleOpUnknownTargetFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRawOp := PatchRemoveStyleOpRaw{
		TargetNodeID: 99,
	}
	_, parseErr := ParsePatchRemoveStyleOp(parseRawOp, parseKnownNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveStyleOp(unknown target) error = nil, want error")
	}
}

// TestParsePatchRemoveStyleOpMissingTargetFails verifies remove-style payloads require a target node ID.
func TestParsePatchRemoveStyleOpMissingTargetFails(parseTesting *testing.T) {
	if _, parseErr := ParsePatchRemoveStyleOp(PatchRemoveStyleOpRaw{}, map[uint64]struct{}{8: {}}); parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveStyleOp(missing target) error = nil, want error")
	}
}
