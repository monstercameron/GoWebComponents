package runtime2

import "testing"

// TestParsePatchSetTextOpValidSetTextOpDecodes verifies valid set-text payloads decode successfully.
func TestParsePatchSetTextOpValidSetTextOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		5: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"next text"})
	getTextRef, _ := buildStringTable.GetRenderStringRef("next text")
	parseRawOp := PatchSetTextOpRaw{
		TargetNodeID: 5,
		TextRef:      getTextRef,
	}
	parseOp, parseErr := ParsePatchSetTextOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchSetTextOp(valid) error = %v", parseErr)
	}
	if parseOp.Text != "next text" {
		parseTesting.Fatalf("ParsePatchSetTextOp(valid) text = %q, want %q", parseOp.Text, "next text")
	}
}

// TestParsePatchSetTextOpMissingTargetNodeFails verifies unknown text targets are rejected.
func TestParsePatchSetTextOpMissingTargetNodeFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		5: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"next text"})
	getTextRef, _ := buildStringTable.GetRenderStringRef("next text")
	parseRawOp := PatchSetTextOpRaw{
		TargetNodeID: 7,
		TextRef:      getTextRef,
	}
	_, parseErr := ParsePatchSetTextOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchSetTextOp(missing target) error = nil, want error")
	}
}

// TestParsePatchSetTextOpInvalidTextReferenceFails verifies invalid text references are rejected.
func TestParsePatchSetTextOpInvalidTextReferenceFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		5: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"next text"})
	parseRawOp := PatchSetTextOpRaw{
		TargetNodeID: 5,
		TextRef:      99,
	}
	_, parseErr := ParsePatchSetTextOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchSetTextOp(invalid text ref) error = nil, want error")
	}
}
