//go:build !js || !wasm

package ui

import "testing"

// TestSpringReducedSnapHonorsPreference proves UseSpring's reduced-motion gate: when reduced
// is true the value snaps straight to the target and the caller skips the animation loop;
// when false it animates as normal.
func TestSpringReducedSnapHonorsPreference(parseT *testing.T) {
	var parseSnappedTo float64
	parseCalled := false
	parseSet := func(parseValue float64) {
		parseSnappedTo = parseValue
		parseCalled = true
	}

	if !springReducedSnap(true, 42, parseSet) {
		parseT.Fatal("reduced motion should report handled=true (skip the loop)")
	}
	if !parseCalled || parseSnappedTo != 42 {
		parseT.Fatalf("reduced motion should snap to the target 42, got %v (called=%v)", parseSnappedTo, parseCalled)
	}

	parseCalled = false
	if springReducedSnap(false, 99, parseSet) {
		parseT.Fatal("full motion should report handled=false (animate)")
	}
	if parseCalled {
		parseT.Fatal("full motion must not snap the position")
	}
}
