package runtime2

import "testing"

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
