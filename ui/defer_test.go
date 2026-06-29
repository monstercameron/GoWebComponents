//go:build !js || !wasm

package ui

import "testing"

// TestDeferLatchedStaysShown proves the latch rule: once shown, it never reverts even if the
// trigger later goes false — so a deferred view that has mounted does not flicker back to
// its placeholder.
func TestDeferLatchedStaysShown(parseT *testing.T) {
	if deferLatched(false, false) {
		parseT.Fatal("not previously shown and not triggered should stay hidden")
	}
	if !deferLatched(false, true) {
		parseT.Fatal("a fresh trigger should show the view")
	}
	if !deferLatched(true, false) {
		parseT.Fatal("once shown, the view must stay shown even when the trigger goes false")
	}
	if !deferLatched(true, true) {
		parseT.Fatal("shown and triggered should remain shown")
	}
}
