// Fuzzing is unsupported on js/wasm, and the wasm-under-node runner cannot
// read the seed corpus directory on Windows (O_DIRECTORY unsupported) — the
// seeds execute on the native build.
//go:build !(js && wasm)

package ui

import "testing"

// FuzzCSSEscape checks CSSEscape never panics and upholds its core invariants for
// any input: non-empty input yields non-empty output, SelectorID == "#"+escape,
// and a colon never survives unescaped (the original G29 footgun).
func FuzzCSSEscape(parseF *testing.F) {
	for _, parseSeed := range []string{"", "a", "gwc:3:1", "a b", "1x", "-", "--x", "\x00", "\t", "cafe", "a}b", "id\U0001F4A5"} {
		parseF.Add(parseSeed)
	}
	parseF.Fuzz(func(parseT *testing.T, parseIn string) {
		parseOut := CSSEscape(parseIn)
		if parseIn != "" && parseOut == "" {
			parseT.Fatalf("non-empty input %q produced empty output", parseIn)
		}
		if SelectorID(parseIn) != "#"+parseOut {
			parseT.Fatalf("SelectorID mismatch for %q", parseIn)
		}
		// Every colon in the output must be backslash-escaped; a bare ':' would
		// reintroduce the G29 querySelector trap.
		for parseI := 0; parseI < len(parseOut); parseI++ {
			if parseOut[parseI] == ':' && (parseI == 0 || parseOut[parseI-1] != '\\') {
				parseT.Fatalf("unescaped colon at %d in %q (input %q)", parseI, parseOut, parseIn)
			}
		}
	})
}
