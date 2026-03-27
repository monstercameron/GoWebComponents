package runtime2

import "testing"

// TestParsePatchRemoveAttrOpValidRemoveAttrOpDecodes verifies valid remove-attr payloads decode successfully.
func TestParsePatchRemoveAttrOpValidRemoveAttrOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRemovedAttrKeys := map[string]struct{}{}
	buildStringTable := BuildRenderStringTable([]string{"class"})
	getClassRef, _ := buildStringTable.GetRenderStringRef("class")
	parseRawOp := PatchRemoveAttrOpRaw{
		TargetNodeID: 8,
		KeyRef:       getClassRef,
	}
	parseOp, parseErr := ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchRemoveAttrOp(valid) error = %v", parseErr)
	}
	if parseOp.Key != "class" {
		parseTesting.Fatalf("ParsePatchRemoveAttrOp(valid) key = %q, want %q", parseOp.Key, "class")
	}
}

// TestParsePatchRemoveAttrOpUnknownAttrReferenceFails verifies invalid attr references are rejected.
func TestParsePatchRemoveAttrOpUnknownAttrReferenceFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRemovedAttrKeys := map[string]struct{}{}
	buildStringTable := BuildRenderStringTable([]string{"class"})
	parseRawOp := PatchRemoveAttrOpRaw{
		TargetNodeID: 8,
		KeyRef:       77,
	}
	_, parseErr := ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveAttrOp(invalid key ref) error = nil, want error")
	}
}

// TestParsePatchRemoveAttrOpDuplicateRemoveBehavesConsistently verifies duplicate attr removes are rejected consistently.
func TestParsePatchRemoveAttrOpDuplicateRemoveBehavesConsistently(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		8: {},
	}
	parseRemovedAttrKeys := map[string]struct{}{}
	buildStringTable := BuildRenderStringTable([]string{"class"})
	getClassRef, _ := buildStringTable.GetRenderStringRef("class")
	parseRawOp := PatchRemoveAttrOpRaw{
		TargetNodeID: 8,
		KeyRef:       getClassRef,
	}
	_, parseErr := ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchRemoveAttrOp(first remove) error = %v", parseErr)
	}
	_, parseErr = ParsePatchRemoveAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable, parseRemovedAttrKeys)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchRemoveAttrOp(duplicate remove) error = nil, want error")
	}
}
