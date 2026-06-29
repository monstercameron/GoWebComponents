package runtime2

import "testing"

// TestParseRenderNodeTableUniqueNodeIDsPass verifies node IDs are unique within one region table.
func TestParseRenderNodeTableUniqueNodeIDsPass(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID: 1,
			Kind:   uint8(RenderNodeKindFragment),
		},
		{
			NodeID: 2,
			Kind:   uint8(RenderNodeKindHostElement),
		},
	}
	parseTable, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr != nil {
		parseTesting.Fatalf("ParseRenderNodeTable(unique IDs) error = %v", parseErr)
	}
	if len(parseTable.Records) != 2 {
		parseTesting.Fatalf("ParseRenderNodeTable(unique IDs) record count = %d, want 2", len(parseTable.Records))
	}
}

// TestParseRenderNodeTableDuplicateNodeIDsFail verifies duplicate IDs are rejected.
func TestParseRenderNodeTableDuplicateNodeIDsFail(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID: 7,
			Kind:   uint8(RenderNodeKindText),
		},
		{
			NodeID: 7,
			Kind:   uint8(RenderNodeKindHostElement),
		},
	}
	_, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeTable(duplicate IDs) error = nil, want error")
	}
}

// TestParseRenderNodeTableZeroNodeIDFails verifies disallowed zero IDs are rejected.
func TestParseRenderNodeTableZeroNodeIDFails(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID: 0,
			Kind:   uint8(RenderNodeKindText),
		},
	}
	_, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeTable(zero ID) error = nil, want error")
	}
}
