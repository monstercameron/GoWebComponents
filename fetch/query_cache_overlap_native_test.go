//go:build !js || !wasm

package fetch

import "testing"

// TestFetchNativeOptimisticUpdatesOverlapRollBackToPreUpdateState pins that two
// overlapping optimistic updates that both roll back return the cache to the
// value from before the first update, whichever order they roll back in.
func TestFetchNativeOptimisticUpdatesOverlapRollBackToPreUpdateState(parseT *testing.T) {
	for _, parseOrder := range [][]int{{0, 1}, {1, 0}} {
		installFetchTestHookContext(parseT)
		setCachedValue("overlap", "x")

		parseUpdates := []OptimisticUpdate[string]{
			ApplyOptimisticUpdate("overlap", func(parsePrev string) string { return parsePrev + "+a" }),
			ApplyOptimisticUpdate("overlap", func(parsePrev string) string { return parsePrev + "+b" }),
		}
		if parseGot := currentCachedSnapshot("overlap").Value; parseGot != "x+a+b" {
			parseT.Fatalf("expected both optimistic updates applied, got %v", parseGot)
		}
		for _, parseIndex := range parseOrder {
			parseUpdates[parseIndex].Rollback()
		}
		if parseGot := currentCachedSnapshot("overlap").Value; parseGot != "x" {
			parseT.Fatalf("order %v: expected rollback to pre-update %q, got %v", parseOrder, "x", parseGot)
		}
	}
}

// TestFetchNativeOptimisticRollbackKeepsOverlappingUpdate pins that rolling back
// one update leaves the other still-active optimistic update applied.
func TestFetchNativeOptimisticRollbackKeepsOverlappingUpdate(parseT *testing.T) {
	installFetchTestHookContext(parseT)
	setCachedValue("overlap-keep", "x")

	parseA := ApplyOptimisticUpdate("overlap-keep", func(parsePrev string) string { return parsePrev + "+a" })
	parseB := ApplyOptimisticUpdate("overlap-keep", func(parsePrev string) string { return parsePrev + "+b" })
	parseA.Rollback()
	if parseGot := currentCachedSnapshot("overlap-keep").Value; parseGot != "x+b" {
		parseT.Fatalf("expected the still-active update to survive, got %v", parseGot)
	}
	parseB.Commit()
	if parseGot := currentCachedSnapshot("overlap-keep").Value; parseGot != "x+b" {
		parseT.Fatalf("expected committed value to remain, got %v", parseGot)
	}
}
