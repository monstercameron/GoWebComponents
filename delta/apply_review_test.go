package delta

import (
	"reflect"
	"testing"
)

// TestApplyFailureLeavesEngineUnchanged pins atomicity: when a later change in the batch fails,
// the earlier changes must not stay applied to the engine. Previously Apply returned (nil, err)
// but kept the partial mutations, so the engine moved on while the consumer never received the
// matching ops and every later diff was computed against a state the consumer does not have.
func TestApplyFailureLeavesEngineUnchanged(parseT *testing.T) {
	for _, parseBad := range []Change{
		{Kind: ChangeUpsert, Row: Row{Key: "z", Version: 1}, AfterKey: "missing"},
		{Kind: ChangeKind(99), Row: Row{Key: "z", Version: 1}},
		{Kind: ChangeUpsert, Row: Row{Key: "", Version: 1}},
	} {
		parseEngine := New()
		if _, parseErr := parseEngine.Publish([]Row{{Key: "a", Version: 1}, {Key: "b", Version: 1}, {Key: "c", Version: 1}}); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		parseBefore := parseEngine.ExportState()

		parseOps, parseErr := parseEngine.Apply([]Change{
			{Kind: ChangeRemove, Row: Row{Key: "b"}},
			{Kind: ChangeUpsert, Row: Row{Key: "a", Version: 2}},
			{Kind: ChangeUpsert, Row: Row{Key: "n", Version: 1}, AfterKey: "a"},
			{Kind: ChangeRemove, Row: Row{Key: "c"}},
			parseBad,
		})
		if parseErr == nil {
			parseT.Fatalf("change %#v: expected an error", parseBad)
		}
		if parseOps != nil {
			parseT.Fatalf("change %#v: ops returned alongside an error: %v", parseBad, parseOps)
		}
		if parseAfter := parseEngine.ExportState(); !reflect.DeepEqual(parseBefore, parseAfter) {
			parseT.Fatalf("change %#v: engine mutated by a failed batch\nbefore %#v\nafter  %#v", parseBad, parseBefore, parseAfter)
		}
	}
}
