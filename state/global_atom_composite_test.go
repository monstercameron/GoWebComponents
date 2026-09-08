//go:build !(js && wasm)

package state_test

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/state"
)

type gaPrefs struct {
	Name string
	N    int
}

// TestGlobalAtomCompositeValueTypes proves GlobalAtom round-trips non-primitive
// value types (slice/struct/map) through Get/Set/Update and falls back on a type
// mismatch — the generic atom is not limited to scalars.
func TestGlobalAtomCompositeValueTypes(parseT *testing.T) {
	parseSlice := state.NewGlobalAtom("test:ga:slice", []string{"a"})
	parseSlice.Set([]string{"x", "y"})
	parseSlice.Update(func(parsePrev []string) []string { return append(parsePrev, "z") })
	if parseGot := parseSlice.Get(); !reflect.DeepEqual(parseGot, []string{"x", "y", "z"}) {
		parseT.Fatalf("slice atom = %v, want [x y z]", parseGot)
	}

	parseStruct := state.NewGlobalAtom("test:ga:struct", gaPrefs{})
	parseStruct.Set(gaPrefs{Name: "cam", N: 7})
	if parseGot := parseStruct.Get(); parseGot != (gaPrefs{Name: "cam", N: 7}) {
		parseT.Fatalf("struct atom = %+v, want {cam 7}", parseGot)
	}

	parseMap := state.NewGlobalAtom("test:ga:map", map[string]int{})
	parseMap.Set(map[string]int{"k": 1})
	if parseGot := parseMap.Get(); !reflect.DeepEqual(parseGot, map[string]int{"k": 1}) {
		parseT.Fatalf("map atom = %v, want map[k:1]", parseGot)
	}

	// A handle of a different type for the struct id falls back to its default.
	parseMismatch := state.NewGlobalAtom("test:ga:struct", "default")
	if parseGot := parseMismatch.Get(); parseGot != "default" {
		parseT.Fatalf("type-mismatch atom = %q, want default", parseGot)
	}
}
