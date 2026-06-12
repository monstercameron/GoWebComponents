// Package anim provides deterministic animation primitives for the
// GoWebComponents framework: a semi-implicit Euler spring solver, a standard
// set of easing functions, and a FLIP (First-Last-Invert-Play) delta
// calculator. All exported symbols are pure functions or plain structs with no
// side effects, no I/O, and no dependency on wall-clock time or randomness —
// a future requestAnimationFrame hook drives them by supplying elapsed seconds.
//
// The package compiles without modification for native Go and for
// GOOS=js GOARCH=wasm.
package anim

import "math"

// ---------------------------------------------------------------------------
// Spring solver
// ---------------------------------------------------------------------------

// SpringConfig carries the physical parameters that shape a spring's motion.
// Values <= 0 are replaced with the "gentle" defaults (Stiffness 170, Damping
// 26, Mass 1) at the point of use, so a zero-value SpringConfig is valid and
// produces gentle behaviour.
type SpringConfig struct {
	// Stiffness controls how strongly the spring pulls toward its target.
	// Higher values produce a faster, snappier response.
	Stiffness float64

	// Damping controls how quickly oscillations die out.
	// Higher values reduce overshoot.
	Damping float64

	// Mass acts as inertia: heavier springs accelerate more slowly.
	Mass float64
}

// resolvedConfig returns a copy of parseC with any non-positive field replaced
// by the gentle preset default.
func (parseC SpringConfig) resolvedConfig() SpringConfig {
	parseResolved := parseC
	if parseResolved.Stiffness <= 0 {
		parseResolved.Stiffness = 170
	}
	if parseResolved.Damping <= 0 {
		parseResolved.Damping = 26
	}
	if parseResolved.Mass <= 0 {
		parseResolved.Mass = 1
	}
	return parseResolved
}

// GentleSpring returns a SpringConfig tuned for soft, flowing motion — the
// react-spring "gentle" preset (stiffness 170, damping 26, mass 1).
func GentleSpring() SpringConfig {
	return SpringConfig{Stiffness: 170, Damping: 26, Mass: 1}
}

// WobblySpring returns a SpringConfig that produces visible ringing before
// settling — the react-spring "wobbly" preset (stiffness 180, damping 12,
// mass 1).
func WobblySpring() SpringConfig {
	return SpringConfig{Stiffness: 180, Damping: 12, Mass: 1}
}

// StiffSpring returns a SpringConfig for fast, snappy motion that settles
// quickly — the react-spring "stiff" preset (stiffness 210, damping 20,
// mass 1).
func StiffSpring() SpringConfig {
	return SpringConfig{Stiffness: 210, Damping: 20, Mass: 1}
}

// maxDt is the largest time step accepted by Step; frames longer than this
// are clamped to keep the Euler integrator numerically stable.
const maxDt = 0.064

// Spring is a damped harmonic oscillator whose position advances frame by
// frame via Step. Create one with NewSpring; zero value is not valid.
type Spring struct {
	parseConfig   SpringConfig
	parsePosition float64
	parseVelocity float64
	parseTarget   float64
}

// NewSpring returns a Spring initialised at parseInitial with zero velocity,
// targeting parseInitial, using parseConfig for its physical parameters.
func NewSpring(parseConfig SpringConfig, parseInitial float64) *Spring {
	return &Spring{
		parseConfig:   parseConfig.resolvedConfig(),
		parsePosition: parseInitial,
		parseVelocity: 0,
		parseTarget:   parseInitial,
	}
}

// SetTarget changes the destination the spring animates toward. The spring
// continues from its current position and velocity — there is no discontinuity.
func (parseS *Spring) SetTarget(parseTarget float64) {
	parseS.parseTarget = parseTarget
}

// Step advances the spring by parseDt seconds using semi-implicit Euler
// integration and returns the new position. Time steps larger than the
// internal stability ceiling are clamped automatically, so callers do not
// need to subdivide large gaps.
func (parseS *Spring) Step(parseDt float64) float64 {
	// Clamp to keep the integrator stable on long or initial frames.
	if parseDt > maxDt {
		parseDt = maxDt
	}
	if parseDt <= 0 {
		return parseS.parsePosition
	}

	parseCfg := parseS.parseConfig
	parseDisplacement := parseS.parsePosition - parseS.parseTarget
	parseForce := -parseCfg.Stiffness*parseDisplacement - parseCfg.Damping*parseS.parseVelocity
	parseAccel := parseForce / parseCfg.Mass

	// Semi-implicit Euler: update velocity first, then position with new velocity.
	parseS.parseVelocity += parseAccel * parseDt
	parseS.parsePosition += parseS.parseVelocity * parseDt

	return parseS.parsePosition
}

// Position returns the spring's current position without advancing time.
func (parseS *Spring) Position() float64 {
	return parseS.parsePosition
}

// Velocity returns the spring's current velocity without advancing time.
func (parseS *Spring) Velocity() float64 {
	return parseS.parseVelocity
}

// IsSettled reports whether the spring has come to rest within parseEpsilon of
// its target. Both positional error and residual velocity must fall below
// parseEpsilon for this to return true.
func (parseS *Spring) IsSettled(parseEpsilon float64) bool {
	return math.Abs(parseS.parsePosition-parseS.parseTarget) < parseEpsilon &&
		math.Abs(parseS.parseVelocity) < parseEpsilon
}

// ---------------------------------------------------------------------------
// Easing functions
// ---------------------------------------------------------------------------

// Easing is a function that maps a progress value in [0, 1] to a shaped
// output in [0, 1]. Inputs outside [0, 1] are clamped before the curve is
// applied, so all easings are well-behaved for any finite input.
type Easing func(parseT float64) float64

// clamp01 constrains parseT to the closed interval [0, 1].
func clamp01(parseT float64) float64 {
	if parseT < 0 {
		return 0
	}
	if parseT > 1 {
		return 1
	}
	return parseT
}

// Linear maps progress to itself — no acceleration or deceleration.
var Linear Easing = func(parseT float64) float64 {
	return clamp01(parseT)
}

// EaseInQuad accelerates from zero using a quadratic curve.
var EaseInQuad Easing = func(parseT float64) float64 {
	parseT = clamp01(parseT)
	return parseT * parseT
}

// EaseOutQuad decelerates to zero using a quadratic curve.
var EaseOutQuad Easing = func(parseT float64) float64 {
	parseT = clamp01(parseT)
	return parseT * (2 - parseT)
}

// EaseInOutQuad accelerates then decelerates symmetrically using a piecewise
// quadratic curve.
var EaseInOutQuad Easing = func(parseT float64) float64 {
	parseT = clamp01(parseT)
	if parseT < 0.5 {
		return 2 * parseT * parseT
	}
	return -1 + (4-2*parseT)*parseT
}

// EaseInCubic accelerates from zero using a cubic curve.
var EaseInCubic Easing = func(parseT float64) float64 {
	parseT = clamp01(parseT)
	return parseT * parseT * parseT
}

// EaseOutCubic decelerates to zero using a cubic curve.
var EaseOutCubic Easing = func(parseT float64) float64 {
	parseT = clamp01(parseT)
	parseT--
	return parseT*parseT*parseT + 1
}

// EaseInOutCubic accelerates then decelerates symmetrically using a piecewise
// cubic curve.
var EaseInOutCubic Easing = func(parseT float64) float64 {
	parseT = clamp01(parseT)
	if parseT < 0.5 {
		return 4 * parseT * parseT * parseT
	}
	parseT = 2*parseT - 2
	return (parseT*parseT*parseT)/2 + 1
}

// Interpolate linearly blends from parseFrom to parseTo using parseEasing
// applied to parseT. parseT = 0 returns parseFrom; parseT = 1 returns parseTo.
func Interpolate(parseFrom, parseTo, parseT float64, parseEasing Easing) float64 {
	parseEased := parseEasing(parseT)
	return parseFrom + (parseTo-parseFrom)*parseEased
}

// ---------------------------------------------------------------------------
// FLIP helper
// ---------------------------------------------------------------------------

// Rect describes an axis-aligned bounding rectangle in layout (or screen)
// coordinates. X and Y are the top-left corner; Width and Height are the
// extents.
type Rect struct {
	X      float64
	Y      float64
	Width  float64
	Height float64
}

// FLIPTransform is the invert step of a FLIP animation: the translation and
// scale that — when applied at the element's Last position — makes it appear
// to occupy its First position. Animating this transform back to the identity
// (zero translation, unit scale) produces the play step.
type FLIPTransform struct {
	TranslateX float64
	TranslateY float64
	ScaleX     float64
	ScaleY     float64
}

// ComputeFLIP derives the invert transform for a FLIP animation from
// parseFirst (measured before the DOM change) and parseLast (measured after).
// When parseLast has zero width or height the corresponding scale component
// defaults to 1, preventing a divide-by-zero.
func ComputeFLIP(parseFirst, parseLast Rect) FLIPTransform {
	parseScaleX := 1.0
	if parseLast.Width != 0 {
		parseScaleX = parseFirst.Width / parseLast.Width
	}

	parseScaleY := 1.0
	if parseLast.Height != 0 {
		parseScaleY = parseFirst.Height / parseLast.Height
	}

	return FLIPTransform{
		TranslateX: parseFirst.X - parseLast.X,
		TranslateY: parseFirst.Y - parseLast.Y,
		ScaleX:     parseScaleX,
		ScaleY:     parseScaleY,
	}
}
