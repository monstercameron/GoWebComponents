package timetravel_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/timetravel"
)

// TestRecordAndCurrent proves recording advances the current state.
func TestRecordAndCurrent(parseT *testing.T) {
	parseHistory := timetravel.New(0, 0)
	parseHistory.Record("inc", 1)
	parseHistory.Record("inc", 2)
	if parseHistory.Current() != 2 {
		parseT.Fatalf("expected current 2, got %d", parseHistory.Current())
	}
	if parseHistory.Len() != 3 {
		parseT.Fatalf("expected 3 snapshots (initial + 2), got %d", parseHistory.Len())
	}
}

// TestUndoRedoStepThroughHistory proves stepping back and forward returns the recorded
// states and respects the boundaries.
func TestUndoRedoStepThroughHistory(parseT *testing.T) {
	parseHistory := timetravel.New(0, "a")
	parseHistory.Record("to-b", "b")
	parseHistory.Record("to-c", "c")

	if parseState, parseOk := parseHistory.Undo(); !parseOk || parseState != "b" {
		parseT.Fatalf("undo should return b, got %q ok=%v", parseState, parseOk)
	}
	if parseState, parseOk := parseHistory.Undo(); !parseOk || parseState != "a" {
		parseT.Fatalf("undo should return a, got %q ok=%v", parseState, parseOk)
	}
	if _, parseOk := parseHistory.Undo(); parseOk {
		parseT.Fatal("undo past the start should report ok=false")
	}
	if parseState, parseOk := parseHistory.Redo(); !parseOk || parseState != "b" {
		parseT.Fatalf("redo should return b, got %q ok=%v", parseState, parseOk)
	}
}

// TestRecordAfterUndoTruncatesRedoBranch proves recording a new state after undoing
// discards the previously-future snapshots (standard undo/redo semantics).
func TestRecordAfterUndoTruncatesRedoBranch(parseT *testing.T) {
	parseHistory := timetravel.New(0, "a")
	parseHistory.Record("to-b", "b")
	parseHistory.Record("to-c", "c")

	parseHistory.Undo() // back to b
	parseHistory.Record("to-d", "d")

	if parseHistory.Current() != "d" {
		parseT.Fatalf("expected current d, got %q", parseHistory.Current())
	}
	if parseHistory.CanRedo() {
		parseT.Fatal("recording after undo should have truncated the redo branch")
	}
	if parseLabels := strings.Join(parseHistory.Labels(), ","); parseLabels != "initial,to-b,to-d" {
		parseT.Fatalf("expected timeline initial,to-b,to-d, got %q", parseLabels)
	}
}

// TestCapacityEvictsOldest proves the ring bound drops the oldest snapshots while keeping
// the cursor and current state valid.
func TestCapacityEvictsOldest(parseT *testing.T) {
	parseHistory := timetravel.New(3, 0)
	for parseI := 1; parseI <= 5; parseI++ {
		parseHistory.Record("step", parseI)
	}
	if parseHistory.Len() != 3 {
		parseT.Fatalf("expected capacity to cap at 3, got %d", parseHistory.Len())
	}
	if parseHistory.Current() != 5 {
		parseT.Fatalf("expected current to remain 5 after eviction, got %d", parseHistory.Current())
	}
	// Oldest retained should be state 3 (1 and 2 evicted).
	if parseState, parseOk := parseHistory.ScrubTo(0); !parseOk || parseState != 3 {
		parseT.Fatalf("expected oldest retained state 3, got %d ok=%v", parseState, parseOk)
	}
}

// TestScrubToArbitraryIndex proves jumping to any valid index, and rejection of bad ones.
func TestScrubToArbitraryIndex(parseT *testing.T) {
	parseHistory := timetravel.New(0, 10)
	parseHistory.Record("a", 20)
	parseHistory.Record("b", 30)

	if parseState, parseOk := parseHistory.ScrubTo(1); !parseOk || parseState != 20 {
		parseT.Fatalf("scrub to 1 should give 20, got %d ok=%v", parseState, parseOk)
	}
	if parseHistory.Cursor() != 1 {
		parseT.Fatalf("cursor should be 1 after scrub, got %d", parseHistory.Cursor())
	}
	if _, parseOk := parseHistory.ScrubTo(99); parseOk {
		parseT.Fatal("scrub out of range should report ok=false")
	}
}

// TestSnapshotsCopyAndCursorImmutable proves Snapshots() returns the whole timeline as a COPY that
// does not move the cursor and cannot corrupt internal state when mutated.
func TestSnapshotsCopyAndCursorImmutable(parseT *testing.T) {
	parseHist := timetravel.New(0, "a")
	parseHist.Record("b", "b")
	parseHist.Record("c", "c")
	parseCursorBefore := parseHist.Cursor()

	parseSnaps := parseHist.Snapshots()
	if len(parseSnaps) != 3 {
		parseT.Fatalf("Snapshots len = %d, want 3 (initial + 2)", len(parseSnaps))
	}
	if parseSnaps[0].Label != "initial" || parseSnaps[2].State != "c" {
		parseT.Fatalf("unexpected snapshots: %+v", parseSnaps)
	}

	// Mutating the returned slice must not corrupt the history.
	parseSnaps[0].Label = "HACKED"
	if parseHist.Snapshots()[0].Label != "initial" {
		parseT.Fatal("Snapshots() must return a copy; mutation leaked into internal state")
	}
	// Reading the timeline must not move the cursor.
	if parseHist.Cursor() != parseCursorBefore {
		parseT.Fatalf("Snapshots() moved the cursor: %d != %d", parseHist.Cursor(), parseCursorBefore)
	}
}
