package runtime2

import "testing"

// TestApplyRegionDOMCanonicalSnapshotReplacesRegionState verifies canonical snapshot apply clears stale nodes and installs the decoded tree.
func TestApplyRegionDOMCanonicalSnapshotReplacesRegionState(parseT *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	if parseSetErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 99, &RegionDOMNode{
		GetNodeID: 99,
		GetTag:    "stale",
	}); parseSetErr != nil {
		parseT.Fatalf("SetRegionDOMNode(stale) returned error: %v", parseSetErr)
	}
	parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "div",
		"children": []any{
			map[string]any{"kind": "text", "text": "hello"},
		},
	})
	if parseCanonicalErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}
	parseResult, parseApplyErr := ApplyRegionDOMCanonicalSnapshot(parseRegionDOMIndex, "region-42", parseCanonicalIR)
	if parseApplyErr != nil {
		parseT.Fatalf("ApplyRegionDOMCanonicalSnapshot returned error: %v", parseApplyErr)
	}
	if parseResult.GetNodeCount != 2 {
		parseT.Fatalf("ApplyRegionDOMCanonicalSnapshot node count = %d, want 2", parseResult.GetNodeCount)
	}
	if _, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 99); parseLookupErr == nil {
		parseT.Fatal("expected stale node to be cleared")
	}
	parseRootNode, parseRootLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", parseCanonicalIR.GetRootNodeID)
	if parseRootLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(root) returned error: %v", parseRootLookupErr)
	}
	if parseRootNode.GetTag != "div" {
		parseT.Fatalf("root tag = %q, want %q", parseRootNode.GetTag, "div")
	}
	if len(parseRootNode.GetChildNodeIDs) != 1 {
		parseT.Fatalf("root child count = %d, want 1", len(parseRootNode.GetChildNodeIDs))
	}
	parseTextNode, parseTextLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", parseRootNode.GetChildNodeIDs[0])
	if parseTextLookupErr != nil {
		parseT.Fatalf("GetRegionDOMNode(text) returned error: %v", parseTextLookupErr)
	}
	if parseTextNode.GetText != "hello" {
		parseT.Fatalf("text node text = %q, want %q", parseTextNode.GetText, "hello")
	}
}

// TestApplyRegionDOMCanonicalSnapshotRejectsInvalidInputs verifies canonical snapshot apply fails fast on missing index, missing region ID, and malformed IR.
func TestApplyRegionDOMCanonicalSnapshotRejectsInvalidInputs(parseT *testing.T) {
	parseCanonicalIR, parseCanonicalErr := BuildCanonicalRenderIR(map[string]any{
		"kind": "host-element",
		"tag":  "div",
	})
	if parseCanonicalErr != nil {
		parseT.Fatalf("BuildCanonicalRenderIR returned error: %v", parseCanonicalErr)
	}

	if _, parseApplyErr := ApplyRegionDOMCanonicalSnapshot(nil, "region-42", parseCanonicalIR); parseApplyErr == nil {
		parseT.Fatal("expected nil region DOM index to fail")
	}
	if _, parseApplyErr := ApplyRegionDOMCanonicalSnapshot(BuildRegionDOMIndex(), "", parseCanonicalIR); parseApplyErr == nil {
		parseT.Fatal("expected empty region ID to fail")
	}
	if _, parseApplyErr := ApplyRegionDOMCanonicalSnapshot(BuildRegionDOMIndex(), "region-42", CanonicalRenderIR{}); parseApplyErr == nil {
		parseT.Fatal("expected malformed canonical render IR to fail")
	}
}
