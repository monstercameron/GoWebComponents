package agentbridge

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/state"
)

// seedAtom registers an atom id with a starting value in the global runtime so
// set-atom tests have a concrete current type to validate against.
func seedAtom(parseTB testing.TB, parseID string, parseValue any) {
	parseTB.Helper()
	if parseErr := state.ApplySnapshot(state.Snapshot{parseID: parseValue}); parseErr != nil {
		parseTB.Fatalf("seed atom %q: %v", parseID, parseErr)
	}
}

// currentAtom reads an atom's current value from the global runtime snapshot.
func currentAtom(parseTB testing.TB, parseID string) any {
	parseTB.Helper()
	parseValue, parseOk := runtime.GetGlobalRuntime().SnapshotAtoms()[parseID]
	if !parseOk {
		parseTB.Fatalf("atom %q not present after seeding", parseID)
	}
	return parseValue
}

// TestWriteSetAtomTypeMismatchFailsClosed pins the todo guarantee: a value
// whose type family differs from the atom's current type is rejected with
// bad-payload AND the atom keeps its prior value (no silent corruption that
// would only surface later at the typed UseAtom[T]().Get() site).
func TestWriteSetAtomTypeMismatchFailsClosed(t *testing.T) {
	writeActivateAgentMode(t)
	seedAtom(t, "agenttest.theme", "light")

	_, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id":    "agenttest.theme",
		"value": map[string]any{"nested": true},
	})
	if parseErr == nil {
		t.Fatalf("expected a type-mismatch rejection, got success")
	}
	if parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("expected %q, got %q (%s)", ErrorCodeBadPayload, parseErr.Code, parseErr.Message)
	}
	if parseGot := currentAtom(t, "agenttest.theme"); parseGot != "light" {
		t.Fatalf("atom was mutated by a rejected write: got %#v, want \"light\"", parseGot)
	}
}

// TestWriteSetAtomSameTypeSucceeds pins that a compatible same-family write
// actually changes the value (the guard does not over-reject).
func TestWriteSetAtomSameTypeSucceeds(t *testing.T) {
	writeActivateAgentMode(t)
	seedAtom(t, "agenttest.theme2", "light")

	_, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id":    "agenttest.theme2",
		"value": "dark",
	})
	if parseErr != nil {
		t.Fatalf("compatible string write rejected: %s", parseErr.Message)
	}
	if parseGot := currentAtom(t, "agenttest.theme2"); parseGot != "dark" {
		t.Fatalf("atom not updated: got %#v, want \"dark\"", parseGot)
	}
}

// TestWriteSetAtomNumberFamilyAccepted pins that JSON's float64 number decoding
// does not cause a spurious mismatch against a numeric atom.
func TestWriteSetAtomNumberFamilyAccepted(t *testing.T) {
	writeActivateAgentMode(t)
	seedAtom(t, "agenttest.count", 1)

	_, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id":    "agenttest.count",
		"value": 7,
	})
	if parseErr != nil {
		t.Fatalf("numeric write rejected by the type guard: %s", parseErr.Message)
	}
}

// TestSetAtomRejectsConcreteTypedComposite pins the fix for the silent
// wrong-type write: a concrete-typed slice/map atom (e.g. UseAtom[[]string])
// cannot be set from a JSON array/object (which decodes to []any/map[string]any
// and would fail the typed Get() assertion). It must fail closed, not report a
// false success — while a generic []any atom remains settable.
func TestSetAtomRejectsConcreteTypedComposite(t *testing.T) {
	writeActivateAgentMode(t)
	seedAtom(t, "agenttest.typedlist", []string{"a", "b"})

	_, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id": "agenttest.typedlist", "value": []any{"c"},
	})
	if parseErr == nil || parseErr.Code != ErrorCodeBadPayload {
		t.Fatalf("expected bad-payload for a concrete-typed slice atom, got %+v", parseErr)
	}
	if parseGot, _ := currentAtom(t, "agenttest.typedlist").([]string); len(parseGot) != 2 {
		t.Fatalf("rejected write still mutated the atom: %#v", currentAtom(t, "agenttest.typedlist"))
	}

	// A generic []any atom is still settable.
	seedAtom(t, "agenttest.genlist", []any{"x"})
	if _, parseErr := writeCallHandler(writeHandleSetAtom, map[string]any{
		"id": "agenttest.genlist", "value": []any{"y", "z"},
	}); parseErr != nil {
		t.Fatalf("generic []any atom should be settable: %s", parseErr.Message)
	}
}

// TestWriteAtomValueCompatible unit-tests the type-family guard directly.
func TestWriteAtomValueCompatible(t *testing.T) {
	parseCases := []struct {
		name    string
		current any
		next    any
		want    bool
	}{
		{"string to string", "a", "b", true},
		{"string to object", "a", map[string]any{}, false},
		{"int to float (json number)", 3, 4.5, true},
		{"bool to bool", true, false, true},
		{"bool to string", true, "x", false},
		{"nil current accepts anything", nil, map[string]any{}, true},
		{"any to nil reset", "a", nil, true},
		{"slice to slice", []any{1}, []any{2, 3}, true},
		{"slice to map", []any{1}, map[string]any{}, false},
	}
	for _, parseCase := range parseCases {
		t.Run(parseCase.name, func(t *testing.T) {
			if parseGot := writeAtomValueCompatible(parseCase.current, parseCase.next); parseGot != parseCase.want {
				t.Fatalf("compatible(%#v, %#v) = %v, want %v", parseCase.current, parseCase.next, parseGot, parseCase.want)
			}
		})
	}
}
