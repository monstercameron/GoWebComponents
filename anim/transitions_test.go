package anim

import (
	"strings"
	"testing"
)

// TestDiffKeyedRectsClassifiesEnterExitMove proves a keyed layout diff separates entering,
// exiting, and surviving keys, and computes a non-identity FLIP transform for moved items.
func TestDiffKeyedRectsClassifiesEnterExitMove(parseT *testing.T) {
	parsePrev := []KeyedRect{
		{Key: "a", Rect: Rect{X: 0, Y: 0, Width: 100, Height: 20}},
		{Key: "b", Rect: Rect{X: 0, Y: 20, Width: 100, Height: 20}},
		{Key: "gone", Rect: Rect{X: 0, Y: 40, Width: 100, Height: 20}},
	}
	parseNext := []KeyedRect{
		{Key: "b", Rect: Rect{X: 0, Y: 0, Width: 100, Height: 20}},  // moved up from y=20
		{Key: "a", Rect: Rect{X: 0, Y: 20, Width: 100, Height: 20}}, // moved down from y=0
		{Key: "new", Rect: Rect{X: 0, Y: 40, Width: 100, Height: 20}},
	}

	parseResult := DiffKeyedRects(parsePrev, parseNext)

	if strings.Join(parseResult.Entering, ",") != "new" {
		parseT.Fatalf("expected Entering [new], got %v", parseResult.Entering)
	}
	if strings.Join(parseResult.Exiting, ",") != "gone" {
		parseT.Fatalf("expected Exiting [gone], got %v", parseResult.Exiting)
	}
	// b moved from y=20 to y=0: invert translateY should place it back at +20.
	parseB, parseOk := parseResult.Moving["b"]
	if !parseOk || parseB.TranslateY != 20 {
		parseT.Fatalf("expected b invert TranslateY=20, got %+v", parseB)
	}
	if strings.Join(parseResult.MovedKeys(), ",") != "a,b" {
		parseT.Fatalf("expected MovedKeys [a b], got %v", parseResult.MovedKeys())
	}
}

// TestDiffKeyedRectsUnmovedKeyIsIdentity proves a surviving key that did not move yields
// the identity transform and is excluded from MovedKeys.
func TestDiffKeyedRectsUnmovedKeyIsIdentity(parseT *testing.T) {
	parseRect := Rect{X: 5, Y: 5, Width: 50, Height: 50}
	parseResult := DiffKeyedRects(
		[]KeyedRect{{Key: "still", Rect: parseRect}},
		[]KeyedRect{{Key: "still", Rect: parseRect}},
	)
	parseTransform := parseResult.Moving["still"]
	if parseTransform != (FLIPTransform{ScaleX: 1, ScaleY: 1}) {
		parseT.Fatalf("expected identity transform for an unmoved key, got %+v", parseTransform)
	}
	if len(parseResult.MovedKeys()) != 0 {
		parseT.Fatalf("expected no moved keys, got %v", parseResult.MovedKeys())
	}
}

// TestTransitionEnterAutoPromotes proves an entering transition advances to Entered once
// its duration elapses and reports progress along the way.
func TestTransitionEnterAutoPromotes(parseT *testing.T) {
	parseTransition := NewTransition(1.0)
	if parseTransition.Phase != PhaseEntering || parseTransition.Progress() != 0 {
		parseT.Fatalf("expected fresh entering at progress 0, got %s/%v", parseTransition.Phase, parseTransition.Progress())
	}

	parseTransition = parseTransition.Advance(0.5)
	if parseTransition.Phase != PhaseEntering || parseTransition.Progress() != 0.5 {
		parseT.Fatalf("expected entering at 0.5, got %s/%v", parseTransition.Phase, parseTransition.Progress())
	}
	if !parseTransition.IsAnimating() {
		parseT.Fatal("expected IsAnimating mid-enter")
	}

	parseTransition = parseTransition.Advance(0.6) // past duration
	if parseTransition.Phase != PhaseEntered || parseTransition.Progress() != 1 {
		parseT.Fatalf("expected promotion to entered at progress 1, got %s/%v", parseTransition.Phase, parseTransition.Progress())
	}
	if parseTransition.IsAnimating() || parseTransition.IsRemovable() {
		parseT.Fatal("entered transition should be neither animating nor removable")
	}
}

// TestTransitionBeginExitThenRemovable proves BeginExit restarts the clock and the exit
// completes into a removable state.
func TestTransitionBeginExitThenRemovable(parseT *testing.T) {
	parseTransition := NewTransition(2.0).Advance(2.0) // entered
	if parseTransition.Phase != PhaseEntered {
		parseT.Fatalf("setup: expected entered, got %s", parseTransition.Phase)
	}

	parseTransition = parseTransition.BeginExit()
	if parseTransition.Phase != PhaseExiting || parseTransition.Elapsed != 0 {
		parseT.Fatalf("expected exiting with reset clock, got %s elapsed=%v", parseTransition.Phase, parseTransition.Elapsed)
	}

	parseTransition = parseTransition.Advance(1.0)
	if parseTransition.Phase != PhaseExiting || parseTransition.Progress() != 0.5 {
		parseT.Fatalf("expected exiting at 0.5, got %s/%v", parseTransition.Phase, parseTransition.Progress())
	}

	parseTransition = parseTransition.Advance(1.0)
	if !parseTransition.IsRemovable() || parseTransition.Progress() != 0 {
		parseT.Fatalf("expected removable exited at progress 0, got %s/%v removable=%v", parseTransition.Phase, parseTransition.Progress(), parseTransition.IsRemovable())
	}
}

// TestTransitionZeroDurationSnaps proves a zero-duration transition snaps to the end of
// its animated phase on the first advance.
func TestTransitionZeroDurationSnaps(parseT *testing.T) {
	parseTransition := NewTransition(0).Advance(0.016)
	if parseTransition.Phase != PhaseEntered {
		parseT.Fatalf("expected immediate entered for zero duration, got %s", parseTransition.Phase)
	}
}

// TestStaggerDelay proves the cascade timing: index 0 has no delay, later items scale by
// the step, and negative inputs are clamped to zero.
func TestStaggerDelay(parseT *testing.T) {
	if parseGot := StaggerDelay(0, 0.05); parseGot != 0 {
		parseT.Fatalf("index 0 should have no delay, got %v", parseGot)
	}
	if parseGot := StaggerDelay(3, 0.05); parseGot < 0.149 || parseGot > 0.151 {
		parseT.Fatalf("index 3 step 0.05 should be ~0.15, got %v", parseGot)
	}
	if parseGot := StaggerDelay(-2, 0.05); parseGot != 0 {
		parseT.Fatalf("negative index should clamp to 0, got %v", parseGot)
	}
}
