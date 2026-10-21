package anim

import (
	"math"
	"testing"
)

// TestSpringNaNDtDoesNotPoison pins that a NaN frame time is ignored rather than
// permanently turning the spring's position and velocity into NaN.
func TestSpringNaNDtDoesNotPoison(parseT *testing.T) {
	parseS := NewSpring(GentleSpring(), 0)
	parseS.SetTarget(1)
	parseS.Step(math.NaN())
	parseS.Step(0.016)
	if math.IsNaN(parseS.Position()) || math.IsNaN(parseS.Velocity()) {
		parseT.Fatalf("spring poisoned by NaN dt: pos=%v vel=%v", parseS.Position(), parseS.Velocity())
	}
}

// TestSpringStaysFiniteForStiffLightConfig pins that a high-stiffness, low-mass
// spring (omega*dt far above the explicit-Euler stability limit) still converges
// instead of diverging to Inf/NaN.
func TestSpringStaysFiniteForStiffLightConfig(parseT *testing.T) {
	parseS := NewSpring(SpringConfig{Stiffness: 50000, Damping: 200, Mass: 0.1}, 0)
	parseS.SetTarget(1)
	for parseI := 0; parseI < 600; parseI++ {
		parseS.Step(0.016)
		if math.IsNaN(parseS.Position()) || math.IsInf(parseS.Position(), 0) || math.Abs(parseS.Position()) > 10 {
			parseT.Fatalf("diverged at frame %d: pos=%v", parseI, parseS.Position())
		}
	}
	if !parseS.IsSettled(0.001) {
		parseT.Fatalf("stiff spring did not settle: pos=%v vel=%v", parseS.Position(), parseS.Velocity())
	}
}

// TestZeroValueSpringIsInert pins that the zero-value Spring (mass 0) no longer
// yields NaN.
func TestZeroValueSpringIsInert(parseT *testing.T) {
	var parseS Spring
	parseS.SetTarget(1)
	parseS.Step(0.016)
	if math.IsNaN(parseS.Position()) || math.IsInf(parseS.Position(), 0) {
		parseT.Fatalf("zero-value spring produced %v", parseS.Position())
	}
}

// TestEasingAndInterpolateEdges pins NaN input handling, exact endpoints and a nil
// easing.
func TestEasingAndInterpolateEdges(parseT *testing.T) {
	for _, parseE := range []Easing{Linear, EaseInQuad, EaseOutQuad, EaseInOutQuad, EaseInCubic, EaseOutCubic, EaseInOutCubic} {
		if parseV := parseE(math.NaN()); math.IsNaN(parseV) {
			parseT.Fatalf("easing returned NaN for NaN input")
		}
	}
	if parseV := Interpolate(0.1, 0.3, 1, Linear); parseV != 0.3 {
		parseT.Fatalf("Interpolate end = %v, want exactly 0.3", parseV)
	}
	if parseV := Interpolate(0.1, 0.7, 1, Linear); parseV != 0.7 {
		parseT.Fatalf("Interpolate end = %v, want exactly 0.7", parseV)
	}
	if parseV := Interpolate(2, 4, 0.5, nil); parseV != 3 {
		parseT.Fatalf("Interpolate with nil easing = %v, want linear 3", parseV)
	}
}

// TestTransitionNaNAndZeroDuration pins that a NaN delta cannot wedge a transition
// and that a zero-duration (reduced motion) transition completes without needing a
// positive frame delta.
func TestTransitionNaNAndZeroDuration(parseT *testing.T) {
	parseTr := NewTransition(1).Advance(math.NaN())
	if math.IsNaN(parseTr.Elapsed) || math.IsNaN(parseTr.Progress()) {
		parseT.Fatalf("NaN delta poisoned transition: %+v", parseTr)
	}
	parseSnap := NewTransition(0).Advance(0)
	if parseSnap.Phase != PhaseEntered {
		parseT.Fatalf("zero-duration enter did not complete on first tick: %+v", parseSnap)
	}
	parseOut := NewTransition(0).Advance(0).BeginExit().Advance(0)
	if !parseOut.IsRemovable() {
		parseT.Fatalf("zero-duration exit not removable: %+v", parseOut)
	}
}

// TestZeroDurationTransitionIgnoresNaNDelta pins that a NaN delta still completes
// a zero-duration transition.
func TestZeroDurationTransitionIgnoresNaNDelta(parseT *testing.T) {
	if parseTr := NewTransition(0).Advance(math.NaN()); parseTr.Phase != PhaseEntered {
		parseT.Fatalf("zero-duration transition wedged on NaN delta: %+v", parseTr)
	}
}
