package runtime2

import "testing"

// TestParsePatchSetAttrOpValidSetAttrOpDecodes verifies valid set-attr payloads decode successfully.
func TestParsePatchSetAttrOpValidSetAttrOpDecodes(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		3: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"class", "active"})
	getClassRef, _ := buildStringTable.GetRenderStringRef("class")
	getActiveRef, _ := buildStringTable.GetRenderStringRef("active")
	parseRawOp := PatchSetAttrOpRaw{
		TargetNodeID: 3,
		Attr: RenderPropRecordRaw{
			Kind:     uint8(RenderPropKindClass),
			KeyRef:   getClassRef,
			ValueRef: getActiveRef,
		},
	}
	parseOp, parseErr := ParsePatchSetAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr != nil {
		parseTesting.Fatalf("ParsePatchSetAttrOp(valid) error = %v", parseErr)
	}
	if parseOp.Attr.Key != "class" || parseOp.Attr.Value != "active" {
		parseTesting.Fatalf("ParsePatchSetAttrOp(valid) attr = %+v, want key=class value=active", parseOp.Attr)
	}
}

// TestParsePatchSetAttrOpUnsupportedAttrKindFails verifies unsupported attr kinds are rejected.
func TestParsePatchSetAttrOpUnsupportedAttrKindFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		3: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"text", "active"})
	getTextRef, _ := buildStringTable.GetRenderStringRef("text")
	getActiveRef, _ := buildStringTable.GetRenderStringRef("active")
	parseRawOp := PatchSetAttrOpRaw{
		TargetNodeID: 3,
		Attr: RenderPropRecordRaw{
			Kind:     uint8(RenderPropKindTextAdjacent),
			KeyRef:   getTextRef,
			ValueRef: getActiveRef,
		},
	}
	_, parseErr := ParsePatchSetAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchSetAttrOp(unsupported kind) error = nil, want error")
	}
}

// TestParsePatchSetAttrOpMissingAttrKeyReferenceFails verifies missing attr key refs are rejected.
func TestParsePatchSetAttrOpMissingAttrKeyReferenceFails(parseTesting *testing.T) {
	parseKnownNodeIDs := map[uint64]struct{}{
		3: {},
	}
	buildStringTable := BuildRenderStringTable([]string{"active"})
	getActiveRef, _ := buildStringTable.GetRenderStringRef("active")
	parseRawOp := PatchSetAttrOpRaw{
		TargetNodeID: 3,
		Attr: RenderPropRecordRaw{
			Kind:     uint8(RenderPropKindClass),
			KeyRef:   99,
			ValueRef: getActiveRef,
		},
	}
	_, parseErr := ParsePatchSetAttrOp(parseRawOp, parseKnownNodeIDs, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParsePatchSetAttrOp(missing key ref) error = nil, want error")
	}
}

// TestParsePatchSetAttrOpRejectsMissingOrUnknownTargets verifies set-attr payloads require a known target node ID before prop decoding.
func TestParsePatchSetAttrOpRejectsMissingOrUnknownTargets(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"class", "active"})
	getClassRef, _ := buildStringTable.GetRenderStringRef("class")
	getActiveRef, _ := buildStringTable.GetRenderStringRef("active")
	testCases := []struct {
		name              string
		parseKnownNodeIDs map[uint64]struct{}
		parseTargetNodeID uint64
	}{
		{
			name:              "missing target",
			parseKnownNodeIDs: map[uint64]struct{}{3: {}},
			parseTargetNodeID: 0,
		},
		{
			name:              "unknown target",
			parseKnownNodeIDs: map[uint64]struct{}{3: {}},
			parseTargetNodeID: 4,
		},
	}
	for _, parseTestCase := range testCases {
		parseTesting.Run(parseTestCase.name, func(parseSubTest *testing.T) {
			_, parseErr := ParsePatchSetAttrOp(PatchSetAttrOpRaw{
				TargetNodeID: parseTestCase.parseTargetNodeID,
				Attr: RenderPropRecordRaw{
					Kind:     uint8(RenderPropKindClass),
					KeyRef:   getClassRef,
					ValueRef: getActiveRef,
				},
			}, parseTestCase.parseKnownNodeIDs, buildStringTable)
			if parseErr == nil {
				parseSubTest.Fatalf("ParsePatchSetAttrOp(%s) error = nil, want error", parseTestCase.name)
			}
		})
	}
}
