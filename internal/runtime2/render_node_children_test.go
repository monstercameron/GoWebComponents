package runtime2

import (
	"reflect"
	"testing"
)

// TestGetRenderNodeChildOrderSiblingOrderRoundTrips verifies sibling ordering is stable through parse and lookup.
func TestGetRenderNodeChildOrderSiblingOrderRoundTrips(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID:     1,
			Kind:       uint8(RenderNodeKindFragment),
			ChildStart: 1,
			ChildCount: 2,
		},
		{
			NodeID: 2,
			Kind:   uint8(RenderNodeKindText),
		},
		{
			NodeID: 3,
			Kind:   uint8(RenderNodeKindHostElement),
		},
	}
	parseTable, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr != nil {
		parseTesting.Fatalf("ParseRenderNodeTable(round-trip order) error = %v", parseErr)
	}
	getChildOrder, getChildOrderErr := parseTable.GetRenderNodeChildOrder(1)
	if getChildOrderErr != nil {
		parseTesting.Fatalf("GetRenderNodeChildOrder(1) error = %v", getChildOrderErr)
	}
	parseWantChildOrder := []uint64{2, 3}
	if !reflect.DeepEqual(getChildOrder, parseWantChildOrder) {
		parseTesting.Fatalf("GetRenderNodeChildOrder(1) = %v, want %v", getChildOrder, parseWantChildOrder)
	}
}

// TestParseRenderNodeTableMissingChildReferencedBySpanFails verifies invalid child references are rejected.
func TestParseRenderNodeTableMissingChildReferencedBySpanFails(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID:     1,
			Kind:       uint8(RenderNodeKindFragment),
			ChildStart: 1,
			ChildCount: 2,
		},
		{
			NodeID: 2,
			Kind:   uint8(RenderNodeKindText),
		},
	}
	_, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeTable(missing child) error = nil, want error")
	}
}

// TestParseRenderNodeTableOverlappingChildSpansFail verifies overlapping child spans are rejected.
func TestParseRenderNodeTableOverlappingChildSpansFail(parseTesting *testing.T) {
	parseRawRecords := []RenderNodeRecordRaw{
		{
			NodeID:     1,
			Kind:       uint8(RenderNodeKindFragment),
			ChildStart: 2,
			ChildCount: 2,
		},
		{
			NodeID:     2,
			Kind:       uint8(RenderNodeKindFragment),
			ChildStart: 3,
			ChildCount: 1,
		},
		{
			NodeID: 3,
			Kind:   uint8(RenderNodeKindText),
		},
		{
			NodeID: 4,
			Kind:   uint8(RenderNodeKindHostElement),
		},
	}
	_, parseErr := ParseRenderNodeTable(parseRawRecords)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderNodeTable(overlapping spans) error = nil, want error")
	}
}
