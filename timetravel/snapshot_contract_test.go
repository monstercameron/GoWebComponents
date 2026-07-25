package timetravel_test

import (
	"maps"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/timetravel"
)

type ttCounter struct {
	N int
}

// TestValueTypeSnapshotsAreIndependent pins the documented value-semantics
// contract: because History stores State by value, mutating the caller's copy
// after Record must NOT alter the retained snapshot. This guards against a
// refactor that accidentally stores a pointer/shared reference (which would make
// undo/redo return the same mutated object and silently break time-travel).
func TestValueTypeSnapshotsAreIndependent(parseT *testing.T) {
	parseHistory := timetravel.New(0, ttCounter{N: 0})

	parseState := ttCounter{N: 1}
	parseHistory.Record("one", parseState)
	// Mutate the caller's copy after recording — must not touch the snapshot.
	parseState.N = 999

	if parseGot := parseHistory.Current().N; parseGot != 1 {
		parseT.Fatalf("value snapshot aliased the caller's copy: Current().N = %d, want 1", parseGot)
	}

	parseHistory.Record("two", ttCounter{N: 2})
	if parsePrev, parseOk := parseHistory.Undo(); !parseOk || parsePrev.N != 1 {
		parseT.Fatalf("undo returned %+v ok=%t, want {N:1} true", parsePrev, parseOk)
	}
}

// TestReferenceTypeSnapshotsAliasByContract documents (locks) the intentional
// reference-type behavior: when T is a pointer, all snapshots share the referent,
// so mutating it is visible through every snapshot. This is the documented
// contract ("use value types or copy on Record"); the test exists so the behavior
// is explicit and any future opt-in clone hook can be validated against it.
func TestReferenceTypeSnapshotsAliasByContract(parseT *testing.T) {
	parseShared := &ttCounter{N: 1}
	parseHistory := timetravel.New(0, parseShared)
	parseHistory.Record("still-shared", parseShared)

	// Mutating the shared referent is visible through the snapshot — by contract.
	parseShared.N = 42
	if parseGot := parseHistory.Current().N; parseGot != 42 {
		parseT.Fatalf("expected pointer snapshots to alias by contract, got N=%d", parseGot)
	}
}

// TestMapStateAliasesByContract extends the reference-type contract to the most
// common React-style state shape (a map): History stores the map header by value,
// so the snapshot shares the backing data and a post-Record mutation is visible
// through it. Pins the documented aliasing so a caller knows to copy on Record.
func TestMapStateAliasesByContract(parseT *testing.T) {
	parseState := map[string]int{"n": 1}
	parseHistory := timetravel.New(0, parseState)
	parseHistory.Record("snap", parseState)

	parseState["n"] = 42 // mutate the shared backing map
	if parseGot := parseHistory.Current()["n"]; parseGot != 42 {
		parseT.Fatalf("expected map snapshot to alias by contract, got n=%d", parseGot)
	}
}

// TestSliceElementMutationAliasesByContract pins that mutating a slice element
// after Record is visible through the snapshot (shared backing array) — the same
// reference-type contract for the other common state shape.
func TestSliceElementMutationAliasesByContract(parseT *testing.T) {
	parseState := []int{1, 2, 3}
	parseHistory := timetravel.New(0, parseState)
	parseHistory.Record("snap", parseState)

	parseState[0] = 99 // mutate the shared backing array
	if parseGot := parseHistory.Current()[0]; parseGot != 99 {
		parseT.Fatalf("expected slice snapshot to alias by contract, got [0]=%d", parseGot)
	}
}

// TestCopyOnRecordIsolatesReferenceState pins the documented CORRECT usage: when a
// caller copies reference-typed state before Record ("use value types or copy on
// Record"), a later mutation of the caller's live map does NOT corrupt the
// historical snapshot — which is what makes undo/redo trustworthy.
func TestCopyOnRecordIsolatesReferenceState(parseT *testing.T) {
	parseLive := map[string]int{"n": 1}
	parseHistory := timetravel.New(0, map[string]int{"n": 0})

	// Record a copy, per the contract.
	parseSnap := maps.Clone(parseLive)
	parseHistory.Record("one", parseSnap)

	parseLive["n"] = 999 // mutating the live map must not touch the recorded copy
	if parseGot := parseHistory.Current()["n"]; parseGot != 1 {
		parseT.Fatalf("copy-on-Record failed to isolate snapshot: got n=%d, want 1", parseGot)
	}
}
