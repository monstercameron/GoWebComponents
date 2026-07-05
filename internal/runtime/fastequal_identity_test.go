package runtime

import "testing"

type fastEqualIncomparable struct {
	_  [0]func()
	ID uint64
}

// TestFastEqualIdentityFastPath pins the interface-header identity shortcut:
// the SAME boxed value must compare equal without reflection — including
// values whose types are not ==-comparable (the js.Func shape) — while
// distinct boxes keep their existing semantics.
func TestFastEqualIdentityFastPath(parseT *testing.T) {
	parseHandler := func() {}
	var parseBoxed any = parseHandler
	if !fastEqual(parseBoxed, parseBoxed) {
		parseT.Fatal("same boxed func must be equal")
	}

	parseValue := fastEqualIncomparable{ID: 7}
	var parseBoxedStruct any = parseValue
	if !fastEqual(parseBoxedStruct, parseBoxedStruct) {
		parseT.Fatal("same boxed incomparable struct must be equal")
	}
	var parseOtherBox any = fastEqualIncomparable{ID: 7}
	if !fastEqual(parseBoxedStruct, parseOtherBox) {
		parseT.Fatal("distinct boxes with DeepEqual-equal content must still be equal (fallback path)")
	}
	var parseDifferent any = fastEqualIncomparable{ID: 8}
	if fastEqual(parseBoxedStruct, parseDifferent) {
		parseT.Fatal("different content must not be equal")
	}

	parseMap := map[string]any{"a": 1}
	var parseBoxedMap any = parseMap
	if !fastEqual(parseBoxedMap, parseBoxedMap) {
		parseT.Fatal("same boxed map must be equal")
	}
	if fastEqual(any(map[string]any{"a": 1}), any(map[string]any{"a": 1})) {
		parseT.Fatal("distinct maps keep identity semantics (not equal)")
	}

	if !fastEqual(any(42), any(42)) || fastEqual(any(42), any(43)) {
		parseT.Fatal("primitive semantics unchanged")
	}
}
