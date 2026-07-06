//go:build !js || !wasm

package virtualization

import "testing"

// TestKeySignatureChangeDetection pins the correctness contract keySignature must
// hold as a UseEffect change-detection dependency: stable for identical keys,
// different when any key changes, on reorder, on a key-boundary shift, and on a
// length change. (The implementation is a cheap FNV-1a fold that replaced a
// per-render fmt.Sprintf("%q", keys) allocation.)
func TestKeySignatureChangeDetection(parseT *testing.T) {
	parseBase := keySignature([]string{"item-0", "item-1", "item-2"})
	if parseBase != keySignature([]string{"item-0", "item-1", "item-2"}) {
		parseT.Fatal("not stable for identical keys")
	}
	if parseBase == keySignature([]string{"item-0", "item-1", "item-X"}) {
		parseT.Fatal("did not change when a key changed")
	}
	if parseBase == keySignature([]string{"item-1", "item-0", "item-2"}) {
		parseT.Fatal("did not change on reorder")
	}
	if keySignature([]string{"ab", "c"}) == keySignature([]string{"a", "bc"}) {
		parseT.Fatal("collided across a key-boundary shift")
	}
	if keySignature([]string{"a"}) == keySignature([]string{"a", "a"}) {
		parseT.Fatal("did not distinguish length")
	}
}
