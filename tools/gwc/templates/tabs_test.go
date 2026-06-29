package components

import "testing"

// TestTabsNextIndexArrowKeys proves the WAI-ARIA tabs keyboard pattern: arrows move with
// wrap-around, Home/End jump to the ends, and other keys leave selection unchanged.
func TestTabsNextIndexArrowKeys(parseT *testing.T) {
	const parseCount = 3
	parseCases := []struct {
		key     string
		current int
		want    int
	}{
		{"ArrowRight", 0, 1},
		{"ArrowRight", 2, 0}, // wrap past the end
		{"ArrowDown", 1, 2},
		{"ArrowLeft", 1, 0},
		{"ArrowLeft", 0, 2}, // wrap past the start
		{"ArrowUp", 2, 1},
		{"Home", 2, 0},
		{"End", 0, 2},
		{"Enter", 1, 1}, // unrelated key → unchanged
		{"a", 0, 0},     // typeahead not handled here → unchanged
	}
	for _, parseCase := range parseCases {
		if parseGot := tabsNextIndex(parseCase.key, parseCase.current, parseCount); parseGot != parseCase.want {
			parseT.Fatalf("tabsNextIndex(%q, %d, %d) = %d, want %d", parseCase.key, parseCase.current, parseCount, parseGot, parseCase.want)
		}
	}
	// Degenerate: no tabs never panics or divides by zero.
	if parseGot := tabsNextIndex("ArrowRight", 0, 0); parseGot != 0 {
		parseT.Fatalf("empty tabs should stay at 0, got %d", parseGot)
	}
}
