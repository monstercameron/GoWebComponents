package timetravel_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/timetravel"
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
