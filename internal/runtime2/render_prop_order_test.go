package runtime2

import (
	"reflect"
	"testing"
)

// TestParseRenderPropRecordsEquivalentPropMapsEmitSameOrder verifies canonical prop-key ordering.
func TestParseRenderPropRecordsEquivalentPropMapsEmitSameOrder(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"a", "b", "one", "two"})
	getARef, _ := buildStringTable.GetRenderStringRef("a")
	getBRef, _ := buildStringTable.GetRenderStringRef("b")
	getOneRef, _ := buildStringTable.GetRenderStringRef("one")
	getTwoRef, _ := buildStringTable.GetRenderStringRef("two")
	parseRawA := []RenderPropRecordRaw{
		{Kind: uint8(RenderPropKindData), KeyRef: getBRef, ValueRef: getTwoRef},
		{Kind: uint8(RenderPropKindData), KeyRef: getARef, ValueRef: getOneRef},
	}
	parseRawB := []RenderPropRecordRaw{
		{Kind: uint8(RenderPropKindData), KeyRef: getARef, ValueRef: getOneRef},
		{Kind: uint8(RenderPropKindData), KeyRef: getBRef, ValueRef: getTwoRef},
	}
	parseRecordsA, parseErrA := ParseRenderPropRecords(parseRawA, buildStringTable)
	if parseErrA != nil {
		parseTesting.Fatalf("ParseRenderPropRecords(rawA) error = %v", parseErrA)
	}
	parseRecordsB, parseErrB := ParseRenderPropRecords(parseRawB, buildStringTable)
	if parseErrB != nil {
		parseTesting.Fatalf("ParseRenderPropRecords(rawB) error = %v", parseErrB)
	}
	if !reflect.DeepEqual(parseRecordsA, parseRecordsB) {
		parseTesting.Fatalf("ParseRenderPropRecords canonical ordering mismatch, A=%v B=%v", parseRecordsA, parseRecordsB)
	}
}

// TestParseRenderPropRecordsDuplicatePropKeysFail verifies duplicate prop keys are rejected.
func TestParseRenderPropRecordsDuplicatePropKeysFail(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"class", "v1", "v2"})
	getClassRef, _ := buildStringTable.GetRenderStringRef("class")
	getV1Ref, _ := buildStringTable.GetRenderStringRef("v1")
	getV2Ref, _ := buildStringTable.GetRenderStringRef("v2")
	parseRawRecords := []RenderPropRecordRaw{
		{Kind: uint8(RenderPropKindClass), KeyRef: getClassRef, ValueRef: getV1Ref},
		{Kind: uint8(RenderPropKindClass), KeyRef: getClassRef, ValueRef: getV2Ref},
	}
	_, parseErr := ParseRenderPropRecords(parseRawRecords, buildStringTable)
	if parseErr == nil {
		parseTesting.Fatal("ParseRenderPropRecords(duplicate keys) error = nil, want error")
	}
}

// TestParseRenderPropRecordsEmptyPropSetRemainsValid verifies empty prop lists decode consistently.
func TestParseRenderPropRecordsEmptyPropSetRemainsValid(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable(nil)
	parseRecords, parseErr := ParseRenderPropRecords(nil, buildStringTable)
	if parseErr != nil {
		parseTesting.Fatalf("ParseRenderPropRecords(nil) error = %v", parseErr)
	}
	if len(parseRecords) != 0 {
		parseTesting.Fatalf("ParseRenderPropRecords(nil) length = %d, want 0", len(parseRecords))
	}
}
