package runtime2

import "testing"

// TestParseRenderNodeRecordValidRecordDecodes verifies a valid render-node record decodes successfully.
func TestParseRenderNodeRecordValidRecordDecodes(parseTesting *testing.T) {
	parseRecordRaw := RenderNodeRecordRaw{
		NodeID:     17,
		Kind:       uint8(RenderNodeKindHostElement),
		ChildStart: 4,
		ChildCount: 2,
		PropStart:  3,
		PropCount:  1,
		TextRef:    0,
		Flags:      9,
	}
	parseRecord, parseErr := ParseRenderNodeRecord(parseRecordRaw)
	if parseErr != nil {
		parseTesting.Fatalf("ParseRenderNodeRecord(valid) error = %v", parseErr)
	}
	if parseRecord.NodeID != parseRecordRaw.NodeID {
		parseTesting.Fatalf("ParseRenderNodeRecord(valid) NodeID = %d, want %d", parseRecord.NodeID, parseRecordRaw.NodeID)
	}
	if parseRecord.Kind != RenderNodeKindHostElement {
		parseTesting.Fatalf("ParseRenderNodeRecord(valid) Kind = %v, want %v", parseRecord.Kind, RenderNodeKindHostElement)
	}
}

// TestParseRenderNodeRecordInvalidChildSpanFails verifies overflow child spans are rejected.
func TestParseRenderNodeRecordInvalidChildSpanFails(parseTesting *testing.T) {
	parseRecordRaw := RenderNodeRecordRaw{
		NodeID:     1,
		Kind:       uint8(RenderNodeKindText),
		ChildStart: ^uint32(0),
		ChildCount: 2,
		PropStart:  0,
		PropCount:  0,
	}
	_, parseErr := ParseRenderNodeRecord(parseRecordRaw)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeRecord(invalid child span) error = nil, want error")
	}
}

// TestParseRenderNodeRecordInvalidPropSpanFails verifies overflow prop spans are rejected.
func TestParseRenderNodeRecordInvalidPropSpanFails(parseTesting *testing.T) {
	parseRecordRaw := RenderNodeRecordRaw{
		NodeID:     1,
		Kind:       uint8(RenderNodeKindText),
		ChildStart: 0,
		ChildCount: 0,
		PropStart:  ^uint32(0),
		PropCount:  2,
	}
	_, parseErr := ParseRenderNodeRecord(parseRecordRaw)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeRecord(invalid prop span) error = nil, want error")
	}
}
