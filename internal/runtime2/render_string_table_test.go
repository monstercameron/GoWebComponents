package runtime2

import (
	"fmt"
	"testing"
)

// TestBuildRenderStringTableDuplicateStringsShareOneEntry verifies duplicate strings map to one table entry.
func TestBuildRenderStringTableDuplicateStringsShareOneEntry(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"div", "div", "span"})
	if len(buildStringTable.Entries) != 2 {
		parseTesting.Fatalf("BuildRenderStringTable duplicate entry count = %d, want 2", len(buildStringTable.Entries))
	}
	getDivRefOne, hasDivRefOne := buildStringTable.GetRenderStringRef("div")
	if !hasDivRefOne {
		parseTesting.Fatal("GetRenderStringRef(div) did not find entry")
	}
	getDivRefTwo, hasDivRefTwo := buildStringTable.GetRenderStringRef("div")
	if !hasDivRefTwo {
		parseTesting.Fatal("GetRenderStringRef(div) second lookup did not find entry")
	}
	if getDivRefOne != getDivRefTwo {
		parseTesting.Fatalf("GetRenderStringRef(div) refs differ, first=%d second=%d", getDivRefOne, getDivRefTwo)
	}
}

// TestBuildRenderStringTableEmptyStringIsRepresentedConsistently verifies empty-string encoding is stable.
func TestBuildRenderStringTableEmptyStringIsRepresentedConsistently(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"", "", "value"})
	getEmptyRef, hasEmptyRef := buildStringTable.GetRenderStringRef("")
	if !hasEmptyRef {
		parseTesting.Fatal("GetRenderStringRef(\"\") did not find empty-string entry")
	}
	getEmptyValue, getEmptyValueErr := buildStringTable.GetRenderStringByRef(getEmptyRef)
	if getEmptyValueErr != nil {
		parseTesting.Fatalf("GetRenderStringByRef(empty) error = %v", getEmptyValueErr)
	}
	if getEmptyValue != "" {
		parseTesting.Fatalf("GetRenderStringByRef(empty) = %q, want empty string", getEmptyValue)
	}
}

// TestGetRenderStringByRefInvalidReferenceFails verifies out-of-range string refs are rejected.
func TestGetRenderStringByRefInvalidReferenceFails(parseTesting *testing.T) {
	buildStringTable := BuildRenderStringTable([]string{"a"})
	_, getStringErr := buildStringTable.GetRenderStringByRef(7)
	if getStringErr == nil {
		parseTesting.Fatal("GetRenderStringByRef(invalid) error = nil, want error")
	}
}

// TestBuildRenderStringTableLargeLookupUsesReverseMap verifies large string tables build the reverse lookup map and preserve canonical refs.
func TestBuildRenderStringTableLargeLookupUsesReverseMap(parseTesting *testing.T) {
	parseValues := make([]string, 0, 40)
	for parseIndex := 39; parseIndex >= 0; parseIndex-- {
		parseValues = append(parseValues, fmt.Sprintf("value-%02d", parseIndex))
	}
	buildStringTable := BuildRenderStringTable(parseValues)
	if len(buildStringTable.Entries) != 40 {
		parseTesting.Fatalf("BuildRenderStringTable large entry count = %d, want 40", len(buildStringTable.Entries))
	}
	getRenderStringRef, hasRenderStringRef := buildStringTable.GetRenderStringRef("value-17")
	if !hasRenderStringRef {
		parseTesting.Fatal("GetRenderStringRef(value-17) did not find entry")
	}
	getRenderStringByRef, parseErr := buildStringTable.GetRenderStringByRef(getRenderStringRef)
	if parseErr != nil {
		parseTesting.Fatalf("GetRenderStringByRef(value-17 ref) error = %v", parseErr)
	}
	if getRenderStringByRef != "value-17" {
		parseTesting.Fatalf("GetRenderStringByRef(value-17 ref) = %q, want %q", getRenderStringByRef, "value-17")
	}
	if _, hasMissingRenderStringRef := buildStringTable.GetRenderStringRef("value-99"); hasMissingRenderStringRef {
		parseTesting.Fatal("GetRenderStringRef(value-99) found unexpected entry")
	}
}
