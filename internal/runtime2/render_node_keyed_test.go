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

// TestParseRenderNodeTableDuplicateKeysBeyondPairwiseLimitFail verifies duplicate keyed siblings are rejected when sibling count exceeds pairwise threshold.
func TestParseRenderNodeTableDuplicateKeysBeyondPairwiseLimitFail(parseTesting *testing.T) {
	parseChildCount := getRenderNodeSiblingKeyPairwiseLimit + 4
	parseRawRecords := make([]RenderNodeRecordRaw, 0, parseChildCount+1)
	parseRawRecords = append(parseRawRecords, RenderNodeRecordRaw{
		NodeID:     1,
		Kind:       uint8(RenderNodeKindFragment),
		ChildStart: 1,
		ChildCount: uint32(parseChildCount),
	})
	for parseIndex := 0; parseIndex < parseChildCount; parseIndex++ {
		parseRawRecords = append(parseRawRecords, RenderNodeRecordRaw{
			NodeID:  uint64(parseIndex + 2),
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: uint64(1000 + parseIndex),
			KeyText: "dup-window",
		})
	}
	parseRawRecords[len(parseRawRecords)-1].KeyHash = parseRawRecords[1].KeyHash
	_, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeTable(duplicate sibling keys beyond pairwise limit) error = nil, want error")
	}
}

// TestParseRenderNodeTableHashCollisionDifferentKeyTextPass verifies siblings with identical key hashes but different key text do not collide as duplicates.
func TestParseRenderNodeTableHashCollisionDifferentKeyTextPass(parseTesting *testing.T) {
	parseChildCount := getRenderNodeSiblingKeyPairwiseLimit + 2
	parseRawRecords := make([]RenderNodeRecordRaw, 0, parseChildCount+1)
	parseRawRecords = append(parseRawRecords, RenderNodeRecordRaw{
		NodeID:     1,
		Kind:       uint8(RenderNodeKindFragment),
		ChildStart: 1,
		ChildCount: uint32(parseChildCount),
	})
	for parseIndex := 0; parseIndex < parseChildCount; parseIndex++ {
		parseRawRecords = append(parseRawRecords, RenderNodeRecordRaw{
			NodeID:  uint64(parseIndex + 2),
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: uint64(2000 + parseIndex),
			KeyText: "key-stable",
		})
	}
	parseRawRecords[1].KeyHash = 777
	parseRawRecords[1].KeyText = "key-a"
	parseRawRecords[len(parseRawRecords)-1].KeyHash = 777
	parseRawRecords[len(parseRawRecords)-1].KeyText = "key-b"
	if _, parseErr := ParseRenderNodeTable(parseRawRecords); parseErr != nil {
		parseTesting.Fatalf("ParseRenderNodeTable(hash collision with different key text) returned error: %v", parseErr)
	}
}

// TestParseRenderNodeTableDenseHashCollisionDifferentKeyTextPass verifies the dense keyed-sibling path preserves hash-collision safety for small sibling sets.
func TestParseRenderNodeTableDenseHashCollisionDifferentKeyTextPass(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID:     1,
			Kind:       uint8(RenderNodeKindFragment),
			ChildStart: 1,
			ChildCount: 4,
		},
		{
			NodeID:  2,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 1,
			KeyText: "row-a",
		},
		{
			NodeID:  3,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 65,
			KeyText: "row-b",
		},
		{
			NodeID:  4,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 129,
			KeyText: "row-c",
		},
		{
			NodeID:  5,
			Kind:    uint8(RenderNodeKindHostElement),
			KeyHash: 193,
			KeyText: "row-d",
		},
	}
	if _, parseErr := ParseRenderNodeTable(parseRawRecords); parseErr != nil {
		parseTesting.Fatalf("ParseRenderNodeTable(dense hash collision with different key text) returned error: %v", parseErr)
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
