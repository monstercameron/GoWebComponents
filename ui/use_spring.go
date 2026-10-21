package ui

import (
	"math"

	"github.com/monstercameron/GoWebComponents/v7/anim"
	"github.com/monstercameron/GoWebComponents/v7/interop"
)

// settledEpsilon is the positional/velocity threshold below which the spring is
// considered at rest and the animation loop stops.
const settledEpsilon = 0.1

// maxSpringDt is the maximum dt in seconds accepted per frame step.
const maxSpringDt = 0.064

// UseSpring animates a float64 value toward parseTarget using a damped harmonic
// spring. It returns the current animated position, which updates each animation
// frame via requestAnimationFrame until the spring settles.
// It honors prefers-reduced-motion by default: when the user requests reduced motion the
// value snaps to the target instead of animating, with no requestAnimationFrame loop.
func UseSpring(parseTarget float64, parseConfig anim.SpringConfig) float64 {
	parseReduced := UsePrefersReducedMotion()
	parsePos := UseState(parseTarget)
	parseSpringRef := UseRef[*anim.Spring](nil)
	if parseSpringRef.Get() == nil {
		parseSpringRef.Set(anim.NewSpring(parseConfig, parseTarget))
	}

	parseTargetRef := UseRef(parseTarget)
	if parseTargetRef.Get() != parseTarget {
		parseTargetRef.Set(parseTarget)
		if parseS := parseSpringRef.Get(); parseS != nil {
			parseS.SetTarget(parseTarget)
		}
	}

	UseEffect(func() func() {
		if springReducedSnap(parseReduced, parseTarget, parsePos.Set) {
			return func() {}
		}
		parseS := parseSpringRef.Get()
		if parseS == nil {
			return func() {}
		}
		return startSpringAnimationTo(parseS, parseTarget, parsePos.Set, interop.RequestAnimationFrame)
	}, parseTarget)

	return parsePos.Get()
}

// springReducedSnap honors prefers-reduced-motion: when reduced is true it snaps the value
// straight to the target and reports true so UseSpring skips the animation loop entirely.
func springReducedSnap(parseReduced bool, parseTarget float64, parseSetPosition func(float64)) bool {
	if !parseReduced {
		return false
	}
	if parseSetPosition != nil {
		parseSetPosition(parseTarget)
	}
	return true
}

type springFrameScheduler func(func(float64)) func()

// startSpringAnimation runs parseS to rest without knowing its destination, so it cannot snap the final
// position; UseSpring uses startSpringAnimationTo instead.
func startSpringAnimation(parseS *anim.Spring, parseSetPosition func(float64), parseRequestFrame springFrameScheduler) func() {
	return startSpringAnimationTo(parseS, math.NaN(), parseSetPosition, parseRequestFrame)
}

// startSpringAnimationTo animates parseS toward parseTarget (NaN when unknown). With a known target the
// rest threshold is capped at 1% of the travel, so a short move such as scale 1 -> 1.05 still animates
// instead of counting as settled on the first frame, and the final frame publishes the exact target
// rather than a position up to the threshold away from it.
func startSpringAnimationTo(parseS *anim.Spring, parseTarget float64, parseSetPosition func(float64), parseRequestFrame springFrameScheduler) func() {
	if parseS == nil || parseRequestFrame == nil {
		return func() {}
	}
	parseEpsilon := settledEpsilon
	isTargetKnown := !math.IsNaN(parseTarget)
	if isTargetKnown {
		if parseTravel := math.Abs(parseTarget - parseS.Position()); parseTravel*0.01 < parseEpsilon && parseTravel > 0 {
			parseEpsilon = parseTravel * 0.01
		}
	}
	if parseSetPosition == nil {
		parseSetPosition = func(float64) {}
	}

	var parseCancelCurrent func()
	parseActive := true
	var parsePrevTS float64

	var parseSchedule func()
	parseSchedule = func() {
		if !parseActive {
			return
		}
		parseCancelCurrent = parseRequestFrame(func(parseTimestampMillis float64) {
			if !parseActive {
				return
			}
			var parseDt float64
			if parsePrevTS > 0 {
				parseDt = (parseTimestampMillis - parsePrevTS) / 1000.0
				if parseDt > maxSpringDt {
					parseDt = maxSpringDt
				}
				if parseDt < 0 {
					parseDt = 0
				}
			}
			parsePrevTS = parseTimestampMillis

			parseNewPos := parseS.Step(parseDt)
			if parseS.IsSettled(parseEpsilon) {
				if isTargetKnown {
					parseNewPos = parseTarget
				}
				parseSetPosition(parseNewPos)
				return
			}
			parseSetPosition(parseNewPos)
			parseSchedule()
		})
	}

	parseSchedule()
	return func() {
		parseActive = false
		if parseCancelCurrent != nil {
			parseCancelCurrent()
		}
	}
}
