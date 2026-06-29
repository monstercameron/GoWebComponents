package anim

import (
	"math"
	"testing"
)

// ---------------------------------------------------------------------------
// Spring tests
// ---------------------------------------------------------------------------

// TestSpringConverges verifies that a gentle spring starting at 0 reaches
// the neighbourhood of target 1.0 within a realistic number of 16 ms frames.
func TestSpringConverges(t *testing.T) {
	parseSpring := NewSpring(GentleSpring(), 0)
	parseSpring.SetTarget(1.0)

	const parseDt = 0.016
	const parseMaxSteps = 500

	for range parseMaxSteps {
		parseSpring.Step(parseDt)
		if parseSpring.IsSettled(0.001) {
			break
		}
	}

	if !parseSpring.IsSettled(0.001) {
		t.Errorf("spring did not settle after %d steps: position=%.6f velocity=%.6f",
			parseMaxSteps, parseSpring.Position(), parseSpring.Velocity())
	}

	parsePos := parseSpring.Position()
	if math.Abs(parsePos-1.0) > 0.01 {
		t.Errorf("settled position %.6f is not close enough to target 1.0", parsePos)
	}
}

// TestSpringNoWildOvershoot confirms that a gentle spring never strays more
// than 2× the target value during its travel — ruling out numerical explosion.
func TestSpringNoWildOvershoot(t *testing.T) {
	parseSpring := NewSpring(GentleSpring(), 0)
	parseSpring.SetTarget(1.0)

	const parseDt = 0.016
	const parseMaxSteps = 500

	for parseI := range parseMaxSteps {
		parsePos := parseSpring.Step(parseDt)
		if parsePos > 2.0 {
			t.Errorf("position %.6f exceeded 2× target at step %d", parsePos, parseI)
			return
		}
	}
}

// TestSpringHugeDtStable ensures that a very large dt is clamped and produces
// a finite, non-NaN result rather than a numerical runaway.
func TestSpringHugeDtStable(t *testing.T) {
	parseSpring := NewSpring(GentleSpring(), 0)
	parseSpring.SetTarget(1.0)

	parsePos := parseSpring.Step(10.0)

	if math.IsNaN(parsePos) {
		t.Error("Step with dt=10 produced NaN")
	}
	if math.IsInf(parsePos, 0) {
		t.Error("Step with dt=10 produced Inf")
	}
	if math.IsNaN(parseSpring.Velocity()) {
		t.Error("Step with dt=10 left velocity as NaN")
	}
	if math.IsInf(parseSpring.Velocity(), 0) {
		t.Error("Step with dt=10 left velocity as Inf")
	}
}

// TestSpringZeroDtNoop checks that stepping with dt=0 leaves the spring
// unchanged.
func TestSpringZeroDtNoop(t *testing.T) {
	parseSpring := NewSpring(GentleSpring(), 0.5)
	parseSpring.SetTarget(1.0)
	parsePos := parseSpring.Step(0)
	if parsePos != 0.5 {
		t.Errorf("expected position 0.5 after zero dt, got %.6f", parsePos)
	}
}

// TestSpringDefaultConfig verifies that a zero-value SpringConfig is treated
// as the gentle preset and still converges.
func TestSpringDefaultConfig(t *testing.T) {
	parseSpring := NewSpring(SpringConfig{}, 0)
	parseSpring.SetTarget(1.0)

	for range 500 {
		parseSpring.Step(0.016)
		if parseSpring.IsSettled(0.001) {
			return
		}
	}
	t.Error("spring with zero-value config did not settle")
}

// ---------------------------------------------------------------------------
// Easing tests
// ---------------------------------------------------------------------------

// TestLinearBoundary checks that Linear returns exactly 0 at t=0, 1 at t=1,
// and 0.5 at t=0.5.
func TestLinearBoundary(t *testing.T) {
	parseCases := []struct {
		parseIn  float64
		parseOut float64
	}{
		{0, 0},
		{1, 1},
		{0.5, 0.5},
	}
	for _, parseC := range parseCases {
		parseGot := Linear(parseC.parseIn)
		if math.Abs(parseGot-parseC.parseOut) > 1e-9 {
			t.Errorf("Linear(%v) = %v, want %v", parseC.parseIn, parseGot, parseC.parseOut)
		}
	}
}

// TestEaseInOutQuadSymmetry checks boundary and midpoint behaviour for
// EaseInOutQuad, the most commonly used symmetric easing.
func TestEaseInOutQuadSymmetry(t *testing.T) {
	parseCases := []struct {
		parseIn  float64
		parseOut float64
	}{
		{0, 0},
		{1, 1},
		{0.5, 0.5},
	}
	for _, parseC := range parseCases {
		parseGot := EaseInOutQuad(parseC.parseIn)
		if math.Abs(parseGot-parseC.parseOut) > 1e-9 {
			t.Errorf("EaseInOutQuad(%v) = %v, want %v", parseC.parseIn, parseGot, parseC.parseOut)
		}
	}
}

func TestCubicEasingInteriorValues(t *testing.T) {
	parseCases := []struct {
		parseName string
		parseEase Easing
		parseIn   float64
		parseWant float64
	}{
		{"EaseInCubic", EaseInCubic, 0.5, 0.125},
		{"EaseOutCubic", EaseOutCubic, 0.5, 0.875},
		{"EaseInOutCubic", EaseInOutCubic, 0.25, 0.0625},
	}
	for _, parseCase := range parseCases {
		parseGot := parseCase.parseEase(parseCase.parseIn)
		if math.Abs(parseGot-parseCase.parseWant) > 1e-12 {
			t.Fatalf("%s(%v) = %.12f, want %.12f", parseCase.parseName, parseCase.parseIn, parseGot, parseCase.parseWant)
		}
	}
}

// TestEasingClampBelowZero confirms that all easings clamp negative inputs to
// 0 rather than producing out-of-range or unspecified values.
func TestEasingClampBelowZero(t *testing.T) {
	parseEasings := []struct {
		parseName string
		parseFunc Easing
	}{
		{"Linear", Linear},
		{"EaseInQuad", EaseInQuad},
		{"EaseOutQuad", EaseOutQuad},
		{"EaseInOutQuad", EaseInOutQuad},
		{"EaseInCubic", EaseInCubic},
		{"EaseOutCubic", EaseOutCubic},
		{"EaseInOutCubic", EaseInOutCubic},
	}
	for _, parseE := range parseEasings {
		parseGot := parseE.parseFunc(-1)
		if math.Abs(parseGot-0) > 1e-9 {
			t.Errorf("%s(-1) = %.6f, want 0", parseE.parseName, parseGot)
		}
	}
}

// TestEasingClampAboveOne confirms that all easings clamp inputs above 1 to
// an output of 1.
func TestEasingClampAboveOne(t *testing.T) {
	parseEasings := []struct {
		parseName string
		parseFunc Easing
	}{
		{"Linear", Linear},
		{"EaseInQuad", EaseInQuad},
		{"EaseOutQuad", EaseOutQuad},
		{"EaseInOutQuad", EaseInOutQuad},
		{"EaseInCubic", EaseInCubic},
		{"EaseOutCubic", EaseOutCubic},
		{"EaseInOutCubic", EaseInOutCubic},
	}
	for _, parseE := range parseEasings {
		parseGot := parseE.parseFunc(2)
		if math.Abs(parseGot-1) > 1e-9 {
			t.Errorf("%s(2) = %.6f, want 1", parseE.parseName, parseGot)
		}
	}
}

// TestEaseOutQuadClampNegative is an explicit named test matching the spec's
// example: EaseOutQuad(-1) should equal 0.
func TestEaseOutQuadClampNegative(t *testing.T) {
	parseGot := EaseOutQuad(-1)
	if math.Abs(parseGot-0) > 1e-9 {
		t.Errorf("EaseOutQuad(-1) = %.6f, want 0", parseGot)
	}
}

// TestInterpolateLinear verifies that Interpolate with Linear blends halfway
// between two values correctly.
func TestInterpolateLinear(t *testing.T) {
	parseGot := Interpolate(0, 10, 0.5, Linear)
	if math.Abs(parseGot-5) > 1e-9 {
		t.Errorf("Interpolate(0,10,0.5,Linear) = %.6f, want 5", parseGot)
	}
}

func TestInterpolateAppliesEasingFunction(t *testing.T) {
	parseGot := Interpolate(0, 10, 0.5, EaseInQuad)
	if math.Abs(parseGot-2.5) > 1e-12 {
		t.Fatalf("Interpolate(0,10,0.5,EaseInQuad) = %.12f, want 2.5", parseGot)
	}
}

// TestInterpolateBoundaries checks that Interpolate returns the endpoints
// exactly at t=0 and t=1.
func TestInterpolateBoundaries(t *testing.T) {
	if parseGot := Interpolate(3, 7, 0, Linear); math.Abs(parseGot-3) > 1e-9 {
		t.Errorf("Interpolate at t=0: got %.6f, want 3", parseGot)
	}
	if parseGot := Interpolate(3, 7, 1, Linear); math.Abs(parseGot-7) > 1e-9 {
		t.Errorf("Interpolate at t=1: got %.6f, want 7", parseGot)
	}
}

func TestSpringIsSettledBoundarySemantics(t *testing.T) {
	parseSpring := NewSpring(GentleSpring(), 1)
	parseSpring.SetTarget(1)
	if !parseSpring.IsSettled(0.001) {
		t.Fatal("expected exact target with zero velocity to be settled")
	}

	parseNearlySettled := NewSpring(GentleSpring(), 1.0005)
	parseNearlySettled.SetTarget(1)
	if !parseNearlySettled.IsSettled(0.001) {
		t.Fatal("expected position inside tolerance to be settled")
	}

	parseOutside := NewSpring(GentleSpring(), 1.002)
	parseOutside.SetTarget(1)
	if parseOutside.IsSettled(0.001) {
		t.Fatal("expected position outside tolerance to remain unsettled")
	}
}

// ---------------------------------------------------------------------------
// FLIP tests
// ---------------------------------------------------------------------------

// TestComputeFLIPKnownValues checks a concrete first/last pair and verifies
// the expected translate and scale values.
func TestComputeFLIPKnownValues(t *testing.T) {
	parseFirst := Rect{X: 10, Y: 20, Width: 100, Height: 50}
	parseLast := Rect{X: 30, Y: 60, Width: 200, Height: 100}

	parseFLIP := ComputeFLIP(parseFirst, parseLast)

	const parseTol = 1e-9

	if math.Abs(parseFLIP.TranslateX-(-20)) > parseTol {
		t.Errorf("TranslateX = %.6f, want -20", parseFLIP.TranslateX)
	}
	if math.Abs(parseFLIP.TranslateY-(-40)) > parseTol {
		t.Errorf("TranslateY = %.6f, want -40", parseFLIP.TranslateY)
	}
	if math.Abs(parseFLIP.ScaleX-0.5) > parseTol {
		t.Errorf("ScaleX = %.6f, want 0.5", parseFLIP.ScaleX)
	}
	if math.Abs(parseFLIP.ScaleY-0.5) > parseTol {
		t.Errorf("ScaleY = %.6f, want 0.5", parseFLIP.ScaleY)
	}
}

// TestComputeFLIPZeroLastWidth verifies that a zero Width in parseLast yields
// ScaleX = 1 instead of a divide-by-zero or NaN.
func TestComputeFLIPZeroLastWidth(t *testing.T) {
	parseFirst := Rect{X: 0, Y: 0, Width: 100, Height: 50}
	parseLast := Rect{X: 0, Y: 0, Width: 0, Height: 50}

	parseFLIP := ComputeFLIP(parseFirst, parseLast)

	if math.IsNaN(parseFLIP.ScaleX) {
		t.Error("ScaleX is NaN when last.Width is 0")
	}
	if math.IsInf(parseFLIP.ScaleX, 0) {
		t.Error("ScaleX is Inf when last.Width is 0")
	}
	if math.Abs(parseFLIP.ScaleX-1) > 1e-9 {
		t.Errorf("ScaleX = %.6f, want 1 when last.Width is 0", parseFLIP.ScaleX)
	}
}

// TestComputeFLIPZeroLastHeight verifies that a zero Height in parseLast
// yields ScaleY = 1 instead of a divide-by-zero or NaN.
func TestComputeFLIPZeroLastHeight(t *testing.T) {
	parseFirst := Rect{X: 0, Y: 0, Width: 50, Height: 100}
	parseLast := Rect{X: 0, Y: 0, Width: 50, Height: 0}

	parseFLIP := ComputeFLIP(parseFirst, parseLast)

	if math.IsNaN(parseFLIP.ScaleY) {
		t.Error("ScaleY is NaN when last.Height is 0")
	}
	if math.IsInf(parseFLIP.ScaleY, 0) {
		t.Error("ScaleY is Inf when last.Height is 0")
	}
	if math.Abs(parseFLIP.ScaleY-1) > 1e-9 {
		t.Errorf("ScaleY = %.6f, want 1 when last.Height is 0", parseFLIP.ScaleY)
	}
}

// TestComputeFLIPIdentity checks that equal first and last rects produce a
// zero-translate, unit-scale transform — animating nothing.
func TestComputeFLIPIdentity(t *testing.T) {
	parseR := Rect{X: 5, Y: 10, Width: 80, Height: 40}
	parseFLIP := ComputeFLIP(parseR, parseR)

	const parseTol = 1e-9
	if math.Abs(parseFLIP.TranslateX) > parseTol || math.Abs(parseFLIP.TranslateY) > parseTol {
		t.Errorf("identity FLIP has non-zero translation: (%v, %v)",
			parseFLIP.TranslateX, parseFLIP.TranslateY)
	}
	if math.Abs(parseFLIP.ScaleX-1) > parseTol || math.Abs(parseFLIP.ScaleY-1) > parseTol {
		t.Errorf("identity FLIP has non-unit scale: (%v, %v)",
			parseFLIP.ScaleX, parseFLIP.ScaleY)
	}
}

func TestPanGestureTracksDeltaVelocityAndEnd(t *testing.T) {
	parseGesture := StartPan(GestureSample{ID: "p1", X: 10, Y: 20, Time: 1})
	if !parseGesture.Active || parseGesture.Start != (Point{X: 10, Y: 20}) {
		t.Fatalf("unexpected started pan gesture: %+v", parseGesture)
	}

	parseGesture = parseGesture.Move(GestureSample{ID: "p1", X: 18, Y: 14, Time: 1.2})
	if parseGesture.Delta != (Point{X: 8, Y: -6}) {
		t.Fatalf("unexpected pan delta: %+v", parseGesture.Delta)
	}
	if math.Abs(parseGesture.Velocity.X-40) > 1e-9 || math.Abs(parseGesture.Velocity.Y-(-30)) > 1e-9 {
		t.Fatalf("unexpected pan velocity: %+v", parseGesture.Velocity)
	}

	parseGesture = parseGesture.Move(GestureSample{ID: "p1", X: 19, Y: 13, Time: 1.2})
	if parseGesture.Velocity != (Point{}) {
		t.Fatalf("expected zero velocity for non-positive dt, got %+v", parseGesture.Velocity)
	}

	parseEnded := parseGesture.End()
	if parseEnded.Active || parseEnded.Velocity != (Point{}) || parseEnded.Delta != parseGesture.Delta {
		t.Fatalf("unexpected ended pan gesture: %+v", parseEnded)
	}
}

func TestComputePinchScaleCenterAndZeroDistance(t *testing.T) {
	parsePinch := ComputePinch(
		Point{X: 0, Y: 0},
		Point{X: 10, Y: 0},
		Point{X: -5, Y: 0},
		Point{X: 15, Y: 0},
	)
	if !parsePinch.Active {
		t.Fatal("expected pinch to be active")
	}
	if math.Abs(parsePinch.StartDistance-10) > 1e-9 || math.Abs(parsePinch.Distance-20) > 1e-9 {
		t.Fatalf("unexpected pinch distances: %+v", parsePinch)
	}
	if math.Abs(parsePinch.Scale-2) > 1e-9 {
		t.Fatalf("unexpected pinch scale: %v", parsePinch.Scale)
	}
	if parsePinch.Center != (Point{X: 5, Y: 0}) {
		t.Fatalf("unexpected pinch center: %+v", parsePinch.Center)
	}

	parseDegenerate := ComputePinch(
		Point{X: 2, Y: 2},
		Point{X: 2, Y: 2},
		Point{X: 2, Y: 2},
		Point{X: 5, Y: 6},
	)
	if math.Abs(parseDegenerate.Scale-1) > 1e-9 {
		t.Fatalf("expected zero-distance pinch to keep identity scale, got %v", parseDegenerate.Scale)
	}
}
