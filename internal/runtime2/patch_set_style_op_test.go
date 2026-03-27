package runtime2

import "testing"

// TestParsePatchSetStyleOpValidSetStyleOpDecodes verifies valid set-style payloads decode successfully.
func TestParsePatchSetStyleOpValidSetStyleOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		5: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"color:blue"})
	getStyleRef, _ := buildStringTable.GetRenderStringRef("color:blue")
	parseRawOp := PatchSetStyleOpRaw{
		TargetNodeID: 5,
		StyleRef:     getStyleRef,
	}
	parseOp, parseErr := ParsePatchSetStyleOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchSetStyleOp(valid) error = %v", parseErr)
	}
	if parseOp.StyleValue != "color:blue" {
		parseTesting.Fatalf("ParsePatchSetStyleOp(valid) style = %q, want %q", parseOp.StyleValue, "color:blue")
	}
}

// TestParsePatchSetStyleOpMissingTargetNodeFails verifies unknown style targets are rejected.
func TestParsePatchSetStyleOpMissingTargetNodeFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		5: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"color:blue"})
	getStyleRef, _ := buildStringTable.GetRenderStringRef("color:blue")
	parseRawOp := PatchSetStyleOpRaw{
		TargetNodeID: 99,
		StyleRef:     getStyleRef,
	}
	_, parseErr := ParsePatchSetStyleOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchSetStyleOp(missing target) error = nil, want error")
	}
}

// TestParsePatchSetStyleOpInvalidStyleReferenceFails verifies invalid style references are rejected.
func TestParsePatchSetStyleOpInvalidStyleReferenceFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		5: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"color:blue"})
	parseRawOp := PatchSetStyleOpRaw{
		TargetNodeID: 5,
		StyleRef:     99,
	}
	_, parseErr := ParsePatchSetStyleOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchSetStyleOp(invalid style ref) error = nil, want error")
	}
}
