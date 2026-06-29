package localfirst_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/localfirst"
)

// TestTextBasicEditing proves local insert/delete behave like an ordinary mutable string.
func TestTextBasicEditing(parseT *testing.T) {
	parseDoc := localfirst.NewText("A")
	parseDoc.Insert(0, "hello")
	parseDoc.Insert(5, " world")
	if parseDoc.Value() != "hello world" {
		parseT.Fatalf("expected 'hello world', got %q", parseDoc.Value())
	}
	parseDoc.Insert(5, ",")
	if parseDoc.Value() != "hello, world" {
		parseT.Fatalf("expected mid-insert 'hello, world', got %q", parseDoc.Value())
	}
	parseDoc.Delete(0, 1) // drop 'h'
	if parseDoc.Value() != "ello, world" || parseDoc.Len() != 11 {
		parseT.Fatalf("expected 'ello, world' (len 11), got %q (len %d)", parseDoc.Value(), parseDoc.Len())
	}
}

// TestTextConcurrentEditsConverge is the core CRDT property the audit demanded: two replicas
// make CONCURRENT edits to the same document — different insertions AND a deletion — then merge
// each other's state in opposite orders. They must converge to the SAME string, every concurrent
// insertion must survive (no last-write-wins), and the deletion must apply on both sides.
func TestTextConcurrentEditsConverge(parseT *testing.T) {
	// Shared starting point: "hello".
	parseSeed := localfirst.NewText("A")
	parseSeed.Insert(0, "hello")
	parseSeedState := parseSeed.Export()

	parseA := localfirst.RestoreText(parseSeedState, "A")
	parseB := localfirst.RestoreText(parseSeedState, "B")

	// Concurrent edits, neither having seen the other:
	parseA.Insert(5, " world") // A appends " world"
	parseB.Insert(5, "!")      // B appends "!"
	parseB.Delete(0, 1)        // B also deletes the leading 'h'

	// Merge in OPPOSITE orders.
	parseA.Merge(parseB)
	parseB.Merge(parseA)

	if parseA.Value() != parseB.Value() {
		parseT.Fatalf("replicas diverged: A=%q B=%q", parseA.Value(), parseB.Value())
	}
	parseConverged := parseA.Value()
	// Every concurrent edit survived (no silent LWW): A's " world", B's "!", and B's delete.
	if !strings.Contains(parseConverged, "world") {
		parseT.Fatalf("A's concurrent ' world' insert was lost: %q", parseConverged)
	}
	if !strings.Contains(parseConverged, "!") {
		parseT.Fatalf("B's concurrent '!' insert was lost: %q", parseConverged)
	}
	if strings.HasPrefix(parseConverged, "h") {
		parseT.Fatalf("B's concurrent delete of 'h' was lost: %q", parseConverged)
	}
}

// TestTextSamePositionConcurrentInsertsBothSurvive proves two replicas inserting at the SAME
// position concurrently both keep their text (the hardest LWW case), converging deterministically.
func TestTextSamePositionConcurrentInsertsBothSurvive(parseT *testing.T) {
	parseSeed := localfirst.NewText("A")
	parseSeed.Insert(0, "XY")
	parseState := parseSeed.Export()

	parseA := localfirst.RestoreText(parseState, "A")
	parseB := localfirst.RestoreText(parseState, "B")
	parseA.Insert(1, "aaa") // between X and Y
	parseB.Insert(1, "bbb") // also between X and Y, concurrently

	parseA.Merge(parseB)
	parseB.Merge(parseA)

	if parseA.Value() != parseB.Value() {
		parseT.Fatalf("same-position inserts diverged: A=%q B=%q", parseA.Value(), parseB.Value())
	}
	parseGot := parseA.Value()
	if !strings.Contains(parseGot, "aaa") || !strings.Contains(parseGot, "bbb") {
		parseT.Fatalf("both concurrent inserts must survive, got %q", parseGot)
	}
	if !strings.HasPrefix(parseGot, "X") || !strings.HasSuffix(parseGot, "Y") {
		parseT.Fatalf("anchors X..Y must bracket the result, got %q", parseGot)
	}
}

// TestTextMergeIdempotentAndCommutative proves merging the same peer twice, or in either order,
// yields the same state — the CvRDT requirement.
func TestTextMergeIdempotentAndCommutative(parseT *testing.T) {
	parseA := localfirst.NewText("A")
	parseA.Insert(0, "abc")
	parseB := localfirst.NewText("B")
	parseB.Insert(0, "xyz")

	parseAB := localfirst.RestoreText(parseA.Export(), "A")
	parseAB.Merge(parseB)
	parseAB.Merge(parseB) // twice — idempotent

	parseBA := localfirst.RestoreText(parseB.Export(), "B")
	parseBA.Merge(parseA)

	if parseAB.Value() != parseBA.Value() {
		parseT.Fatalf("merge not commutative: AB=%q BA=%q", parseAB.Value(), parseBA.Value())
	}
}

// TestTextExportRestoreRoundTrip proves a document survives serialization unchanged, including
// tombstones.
func TestTextExportRestoreRoundTrip(parseT *testing.T) {
	parseDoc := localfirst.NewText("A")
	parseDoc.Insert(0, "draft")
	parseDoc.Delete(0, 1) // tombstone 'd'
	parseRestored := localfirst.RestoreText(parseDoc.Export(), "A")
	if parseRestored.Value() != parseDoc.Value() {
		parseT.Fatalf("round-trip changed value: %q vs %q", parseRestored.Value(), parseDoc.Value())
	}
	if parseRestored.Value() != "raft" {
		parseT.Fatalf("expected 'raft' after tombstoning 'd', got %q", parseRestored.Value())
	}
}
