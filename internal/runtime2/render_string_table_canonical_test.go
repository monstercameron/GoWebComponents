package runtime2

import (
	"reflect"
	"testing"
)

// TestBuildRenderStringTableSameLogicalContentYieldsSameOrdering verifies canonical output is stable.
func TestBuildRenderStringTableSameLogicalContentYieldsSameOrdering(parseTesting *testing.T) {
	buildTableA := BuildRenderStringTable([]string{"beta", "alpha", "gamma", "alpha"})
	buildTableB := BuildRenderStringTable([]string{"gamma", "beta", "alpha"})
	if !reflect.DeepEqual(buildTableA.Entries, buildTableB.Entries) {
		parseTesting.Fatalf("canonical ordering mismatch, tableA=%v tableB=%v", buildTableA.Entries, buildTableB.Entries)
	}
}

// TestBuildRenderStringTableInsertionOrderDoesNotChangeCanonicalOutput verifies insertion-order differences do not affect output.
func TestBuildRenderStringTableInsertionOrderDoesNotChangeCanonicalOutput(parseTesting *testing.T) {
	buildTableA := BuildRenderStringTable([]string{"z", "a", "m"})
	buildTableB := BuildRenderStringTable([]string{"m", "z", "a"})
	if !reflect.DeepEqual(buildTableA.Entries, buildTableB.Entries) {
		parseTesting.Fatalf("insertion order changed canonical output, tableA=%v tableB=%v", buildTableA.Entries, buildTableB.Entries)
	}
}

// TestBuildRenderStringTableEmptyTableRemainsValid verifies an empty string table is valid.
func TestBuildRenderStringTableEmptyTableRemainsValid(parseTesting *testing.T) {
	buildTable := BuildRenderStringTable(nil)
	if len(buildTable.Entries) != 0 {
		parseTesting.Fatalf("BuildRenderStringTable(nil) entry count = %d, want 0", len(buildTable.Entries))
	}
	_, hasRef := buildTable.GetRenderStringRef("anything")
	if hasRef {
		parseTesting.Fatal("GetRenderStringRef on empty table reported an entry")
	}
}
