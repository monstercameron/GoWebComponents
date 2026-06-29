package state

import "testing"

// TestAtomValuesEqualNonComparable proves equal non-comparable values (slices/maps) compare
// equal via the DeepEqual fallback, so setting an equal slice/map is a no-op (no needless
// re-notify) — and differing ones still compare unequal.
func TestAtomValuesEqualNonComparable(t *testing.T) {
	if !atomValuesEqual([]int{1, 2}, []int{1, 2}) {
		t.Fatal("equal slices should compare equal (no re-notify)")
	}
	if atomValuesEqual([]int{1, 2}, []int{1, 3}) {
		t.Fatal("differing slices should compare unequal")
	}
	if !atomValuesEqual(map[string]int{"a": 1}, map[string]int{"a": 1}) {
		t.Fatal("equal maps should compare equal")
	}
	// Comparable fast path still works.
	if !atomValuesEqual(5, 5) || atomValuesEqual(5, 6) {
		t.Fatal("comparable == fast path broken")
	}
}
