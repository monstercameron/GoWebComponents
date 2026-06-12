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

	for parseI := 0; parseI < parseMaxSteps; parseI++ {
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

	for parseI := 0; parseI < parseMaxSteps; parseI++ {
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

	for parseI := 0; parseI < 500; parseI++ {
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
