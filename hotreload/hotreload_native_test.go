//go:build !(js && wasm)

package hotreload

import "testing"

// TestSnapshotWedgeShouldClear pins the #57 restore-wedge discriminator: a
// well-formed JSON snapshot that failed to restore is un-restorable (schema/
// migration mismatch) and must be cleared so the restore path is not wedged
// forever; a malformed/truncated payload (possibly a mid-write) is kept.
func TestSnapshotWedgeShouldClear(parseT *testing.T) {
	parseCases := []struct {
		parseName    string
		parsePayload string
		parseWant    bool
	}{
		{"valid object -> clear", `{"version":2,"state":{"n":1}}`, true},
		{"valid array -> clear", `[1,2,3]`, true},
		{"valid string -> clear", `"snapshot"`, true},
		{"truncated mid-write -> keep", `{"version":2,"state":{"n":`, false},
		{"garbage -> keep", `not json at all`, false},
		{"empty -> keep", ``, false},
	}
	for _, parseCase := range parseCases {
		if parseGot := snapshotWedgeShouldClear(parseCase.parsePayload); parseGot != parseCase.parseWant {
			parseT.Fatalf("%s: snapshotWedgeShouldClear(%q) = %v, want %v",
				parseCase.parseName, parseCase.parsePayload, parseGot, parseCase.parseWant)
		}
	}
}
