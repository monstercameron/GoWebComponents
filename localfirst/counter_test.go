package localfirst

import (
	"encoding/json"
	"testing"
)

// TestCounterConcurrentIncrementsConverge is the FC6 guarantee LWW gets wrong: two replicas
// each increment the SAME counter offline; after merging both directions, every replica
// converges to the SUM (2), not 1 — no increment is lost.
func TestCounterConcurrentIncrementsConverge(parseT *testing.T) {
	parseA := NewCounter()
	parseB := NewCounter()
	parseA.Inc("A", 1)
	parseB.Inc("B", 1)

	parseA.Merge(parseB)
	parseB.Merge(parseA)

	if parseA.Value() != 2 || parseB.Value() != 2 {
		parseT.Fatalf("concurrent increments must sum to 2 (LWW would lose one): A=%d B=%d", parseA.Value(), parseB.Value())
	}
}

// TestCounterMergeIsIdempotentAndCommutative proves merging the same peer twice, and in
// either order, yields the same value.
func TestCounterMergeIsIdempotentAndCommutative(parseT *testing.T) {
	parseA := NewCounter()
	parseA.Inc("A", 5)
	parseA.Dec("A", 2)
	parseB := NewCounter()
	parseB.Inc("B", 3)

	parseFirst := NewCounter()
	parseFirst.Merge(parseA)
	parseFirst.Merge(parseB)
	parseFirst.Merge(parseB) // idempotent

	parseSecond := NewCounter()
	parseSecond.Merge(parseB)
	parseSecond.Merge(parseA) // commutative

	if parseFirst.Value() != parseSecond.Value() || parseFirst.Value() != 6 {
		parseT.Fatalf("expected both merge orders to converge to 6, got %d and %d", parseFirst.Value(), parseSecond.Value())
	}
}

// TestCounterExportRestoreRoundTrips proves the counter survives JSON persistence.
func TestCounterExportRestoreRoundTrips(parseT *testing.T) {
	parseCounter := NewCounter()
	parseCounter.Inc("A", 4)
	parseCounter.Dec("B", 1)

	parseBlob, parseErr := json.Marshal(parseCounter.Export())
	if parseErr != nil {
		parseT.Fatalf("marshal: %v", parseErr)
	}
	var parseState CounterState
	if parseErr := json.Unmarshal(parseBlob, &parseState); parseErr != nil {
		parseT.Fatalf("unmarshal: %v", parseErr)
	}
	if parseRestored := RestoreCounter(parseState); parseRestored.Value() != 3 {
		parseT.Fatalf("expected restored value 3, got %d", parseRestored.Value())
	}
}

// TestCursorSelectionGeometry proves the typed cursor's selection helpers are
// drag-direction-independent.
func TestCursorSelectionGeometry(parseT *testing.T) {
	parseForward := Cursor{Anchor: 3, Head: 7}
	if !parseForward.HasSelection() || parseForward.Start() != 3 || parseForward.End() != 7 {
		parseT.Fatalf("forward selection wrong: %+v", parseForward)
	}
	parseBackward := Cursor{Anchor: 9, Head: 4}
	if parseBackward.Start() != 4 || parseBackward.End() != 9 {
		parseT.Fatalf("backward selection should normalize: %+v", parseBackward)
	}
	parseCaret := Cursor{Anchor: 5, Head: 5}
	if parseCaret.HasSelection() {
		parseT.Fatal("a collapsed caret has no selection")
	}
}
