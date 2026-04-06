package runtime2

import "testing"

// TestRenderNodeTableChildOrderHandlesEmptyOutOfRangeAndMissingNodes verifies child-order lookup returns empty children, rejects invalid spans, and fails cleanly for missing node IDs.
func TestRenderNodeTableChildOrderHandlesEmptyOutOfRangeAndMissingNodes(parseT *testing.T) {
	parseTable := RenderNodeTable{
		Records: []RenderNodeRecord{
			{NodeID: 1},
			{NodeID: 2, ChildStart: 5, ChildCount: 1},
		},
	}

	parseChildOrder, parseErr := parseTable.GetRenderNodeChildOrder(1)
	if parseErr != nil {
		parseT.Fatalf("GetRenderNodeChildOrder(empty children) returned error: %v", parseErr)
	}
	if len(parseChildOrder) != 0 {
		parseT.Fatalf("GetRenderNodeChildOrder(empty children) length = %d, want 0", len(parseChildOrder))
	}

	if _, parseErr := parseTable.GetRenderNodeChildOrder(2); parseErr == nil {
		parseT.Fatal("expected out-of-range child span to fail")
	}
	if _, parseErr := parseTable.GetRenderNodeChildOrder(99); parseErr == nil {
		parseT.Fatal("expected missing node child-order lookup to fail")
	}
}

// TestRenderNodeTableKeyMetadataHandlesUnkeyedAndMissingNodes verifies key metadata lookup returns zero values for unkeyed nodes and fails clearly for missing node IDs.
func TestRenderNodeTableKeyMetadataHandlesUnkeyedAndMissingNodes(parseT *testing.T) {
	parseTable := RenderNodeTable{
		Records: []RenderNodeRecord{
			{NodeID: 1},
			{NodeID: 2, KeyHash: 17, KeyText: "primary"},
		},
	}

	parseMetadata, parseErr := parseTable.GetRenderNodeKeyMetadata(1)
	if parseErr != nil {
		parseT.Fatalf("GetRenderNodeKeyMetadata(unkeyed) returned error: %v", parseErr)
	}
	if parseMetadata.HasKey {
		parseT.Fatalf("GetRenderNodeKeyMetadata(unkeyed) = %+v, want no key", parseMetadata)
	}

	parseMetadata, parseErr = parseTable.GetRenderNodeKeyMetadata(2)
	if parseErr != nil {
		parseT.Fatalf("GetRenderNodeKeyMetadata(keyed) returned error: %v", parseErr)
	}
	if !parseMetadata.HasKey || parseMetadata.KeyHash != 17 || parseMetadata.KeyText != "primary" {
		parseT.Fatalf("GetRenderNodeKeyMetadata(keyed) = %+v, want keyed metadata", parseMetadata)
	}

	if _, parseErr := parseTable.GetRenderNodeKeyMetadata(99); parseErr == nil {
		parseT.Fatal("expected missing node key metadata lookup to fail")
	}
}
