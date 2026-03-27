package runtime2

import "testing"

// TestParsePatchReplaceSubtreeOpValidPayloadDecodes verifies valid replace-subtree payloads decode successfully.
func TestParsePatchReplaceSubtreeOpValidPayloadDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseSubtreeIR, parseSubtreeErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{"kind": "text", "text": "after"},
		},
	})
	if parseSubtreeErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR returned error: %v", parseSubtreeErr)
	}
	parseRawOp := PatchReplaceSubtreeOpRaw{
		TargetNodeID: 1,
		Subtree:      parseBuildReplaceSubtreePayload(parseSubtreeIR),
	}
	parseOp, parseErr := ParsePatchReplaceSubtreeOp(parseRawOp, parseKnownNodeIDs)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchReplaceSubtreeOp(valid) error = %v", parseErr)
	}
	if parseOp.TargetNodeID != 1 {
		parseTesting.Fatalf("ParsePatchReplaceSubtreeOp(valid) target = %d, want %d", parseOp.TargetNodeID, 1)
	}
}

// TestParsePatchReplaceSubtreeOpUnknownTargetFails verifies unknown replace-subtree targets are rejected.
func TestParsePatchReplaceSubtreeOpUnknownTargetFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseSubtreeIR, parseSubtreeErr := BuildCanonicalRenderIR(map[string]any{"kind": "text", "text": "after"})
	if parseSubtreeErr != nil {
		parseTesting.Fatalf("BuildCanonicalRenderIR returned error: %v", parseSubtreeErr)
	}
	parseRawOp := PatchReplaceSubtreeOpRaw{
		TargetNodeID: 99,
		Subtree:      parseBuildReplaceSubtreePayload(parseSubtreeIR),
	}
	_, parseErr := ParsePatchReplaceSubtreeOp(parseRawOp, parseKnownNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchReplaceSubtreeOp(unknown target) error = nil, want error")
	}
}

// TestParsePatchReplaceSubtreeOpInvalidSubtreeFails verifies malformed replace-subtree payloads are rejected.
func TestParsePatchReplaceSubtreeOpInvalidSubtreeFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		1: {},
	}
	parseRawOp := PatchReplaceSubtreeOpRaw{
		TargetNodeID: 1,
		Subtree: PatchReplaceSubtreePayloadRaw{
			RootNodeID:  1,
			StringTable: []string{"x"},
			NodeRecords: []RenderNodeRecordRaw{},
		},
	}
	_, parseErr := ParsePatchReplaceSubtreeOp(parseRawOp, parseKnownNodeIDs)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchReplaceSubtreeOp(invalid subtree) error = nil, want error")
	}
}
