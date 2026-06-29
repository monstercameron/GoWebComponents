// Package anim provides deterministic animation primitives for the
// GoWebComponents framework: a semi-implicit Euler spring solver, a standard
// set of easing functions, FLIP (First-Last-Invert-Play) deltas, and pure
// pointer gesture helpers. All exported symbols are pure functions or plain structs with no
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
// frame via Step. Create one with NewSpring; the zero value is NOT valid —
// its Mass is 0, so Step divides by zero and silently produces NaN/Inf
// positions rather than panicking.
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
//
// Note: this is a func type and is unrelated to css.Easing (a string timing-
// function token). When a file imports both packages, EasingFunc is the
// drop-in alias to use for this type to avoid the bare-name collision.
type Easing func(parseT float64) float64

// EasingFunc is an alias for Easing, provided to disambiguate from css.Easing
// (a string type) when both packages are imported in the same file.
type EasingFunc = Easing

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

// ---------------------------------------------------------------------------
// Gesture helpers
// ---------------------------------------------------------------------------

// Point is a two-dimensional coordinate in client or layout space.
type Point struct {
	X float64
	Y float64
}

// GestureSample is one timestamped pointer position. Time is expressed in
// seconds and is caller-owned, so tests and browser integrations can use a
// deterministic clock.
type GestureSample struct {
	ID   string
	X    float64
	Y    float64
	Time float64
}

// PanGesture describes the state of a one-pointer drag/pan interaction.
type PanGesture struct {
	Start     Point
	Current   Point
	Delta     Point
	Velocity  Point
	StartTime float64
	Time      float64
	Active    bool
}

// StartPan begins a one-pointer pan gesture from parseSample.
func StartPan(parseSample GestureSample) PanGesture {
	parsePoint := Point{X: parseSample.X, Y: parseSample.Y}
	return PanGesture{
		Start:     parsePoint,
		Current:   parsePoint,
		StartTime: parseSample.Time,
		Time:      parseSample.Time,
		Active:    true,
	}
}

// Move advances a pan gesture to parseSample, updating total delta and
// instantaneous velocity. Non-positive elapsed time produces zero velocity.
func (parseG PanGesture) Move(parseSample GestureSample) PanGesture {
	parseNext := parseG
	parsePrevious := parseG.Current
	parsePreviousTime := parseG.Time
	parseNext.Current = Point{X: parseSample.X, Y: parseSample.Y}
	parseNext.Time = parseSample.Time
	parseNext.Delta = Point{
		X: parseNext.Current.X - parseNext.Start.X,
		Y: parseNext.Current.Y - parseNext.Start.Y,
	}
	parseDt := parseSample.Time - parsePreviousTime
	if parseDt > 0 {
		parseNext.Velocity = Point{
			X: (parseNext.Current.X - parsePrevious.X) / parseDt,
			Y: (parseNext.Current.Y - parsePrevious.Y) / parseDt,
		}
	} else {
		parseNext.Velocity = Point{}
	}
	parseNext.Active = true
	return parseNext
}

// End marks a pan gesture inactive while preserving its final deltas.
func (parseG PanGesture) End() PanGesture {
	parseG.Active = false
	parseG.Velocity = Point{}
	return parseG
}

// PinchGesture describes the scale/center produced by a two-pointer pinch.
type PinchGesture struct {
	StartA        Point
	StartB        Point
	CurrentA      Point
	CurrentB      Point
	StartDistance float64
	Distance      float64
	Scale         float64
	Center        Point
	Active        bool
}

// ComputePinch calculates the current two-pointer pinch scale and center. A
// zero start distance is treated as identity scale to avoid divide-by-zero.
func ComputePinch(parseStartA, parseStartB, parseCurrentA, parseCurrentB Point) PinchGesture {
	parseStartDistance := pointDistance(parseStartA, parseStartB)
	parseDistance := pointDistance(parseCurrentA, parseCurrentB)
	parseScale := 1.0
	if parseStartDistance > 0 {
		parseScale = parseDistance / parseStartDistance
	}
	return PinchGesture{
		StartA:        parseStartA,
		StartB:        parseStartB,
		CurrentA:      parseCurrentA,
		CurrentB:      parseCurrentB,
		StartDistance: parseStartDistance,
		Distance:      parseDistance,
		Scale:         parseScale,
		Center: Point{
			X: (parseCurrentA.X + parseCurrentB.X) / 2,
			Y: (parseCurrentA.Y + parseCurrentB.Y) / 2,
		},
		Active: true,
	}
}

func pointDistance(parseA, parseB Point) float64 {
	return math.Hypot(parseA.X-parseB.X, parseA.Y-parseB.Y)
}
