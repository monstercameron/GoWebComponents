package runtime2

import "testing"

// TestGetRenderNodeKeyMetadataValidKeyedMetadataRoundTrips verifies keyed metadata survives parse and lookup.
func TestGetRenderNodeKeyMetadataValidKeyedMetadataRoundTrips(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID:     1,
			Kind:       uint8(RenderNodeKindFragment),
			ChildStart: 1,
			ChildCount: 2,
		},
		{
			NodeID:  2,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 11,
			KeyText: "row-a",
		},
		{
			NodeID:  3,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 12,
			KeyText: "row-b",
		},
	}
	parseTable, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr != nil {
		parseTesting.Fatalf("ParseRenderNodeTable(valid keyed metadata) error = %v", parseErr)
	}
	getKeyMetadata, getKeyMetadataErr := parseTable.GetRenderNodeKeyMetadata(2)
	if getKeyMetadataErr != nil {
		parseTesting.Fatalf("GetRenderNodeKeyMetadata(2) error = %v", getKeyMetadataErr)
	}
	if !getKeyMetadata.HasKey || getKeyMetadata.KeyHash != 11 || getKeyMetadata.KeyText != "row-a" {
		parseTesting.Fatalf("GetRenderNodeKeyMetadata(2) = %+v, want keyed metadata", getKeyMetadata)
	}
}

// TestParseRenderNodeTableDuplicateKeysInSiblingSetFail verifies duplicate keyed siblings are rejected.
func TestParseRenderNodeTableDuplicateKeysInSiblingSetFail(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID:     1,
			Kind:       uint8(RenderNodeKindFragment),
			ChildStart: 1,
			ChildCount: 2,
		},
		{
			NodeID:  2,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 99,
			KeyText: "dup-key",
		},
		{
			NodeID:  3,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 99,
			KeyText: "dup-key",
		},
	}
	_, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeTable(duplicate sibling keys) error = nil, want error")
	}
}

// TestParseRenderNodeRecordInvalidKeyHashOrMissingPayloadFails verifies invalid key fields are rejected.
func TestParseRenderNodeRecordInvalidKeyHashOrMissingPayloadFails(parseTesting *testing.T) {
	parseCases := []RenderNodeRecordRaw{
		{
			NodeID:  2,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 0,
			KeyText: "missing-hash",
		},
		{
			NodeID:  3,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 7,
			KeyText: "",
		},
	}
	for parseCaseIndex, parseCase := range parseCases {
		_, parseErr := ParseRenderNodeRecord(parseCase)
		if parseErr == nil {
			parseTesting.Fatalf("ParseRenderNodeRecord(case %d) error = nil, want error", parseCaseIndex)
		}
	}
}
