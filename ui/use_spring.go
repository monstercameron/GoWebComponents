package ui

import (
	"github.com/monstercameron/GoWebComponents/anim"
	"github.com/monstercameron/GoWebComponents/interop"
)

// settledEpsilon is the positional/velocity threshold below which the spring is
// considered at rest and the animation loop stops.
const settledEpsilon = 0.1

// maxSpringDt is the maximum dt (in seconds) accepted per frame step. Frames
// longer than this (e.g. the very first frame, or after a tab backgrounding)
// are clamped so the semi-implicit Euler integrator stays numerically stable.
const maxSpringDt = 0.064

// UseSpring animates a float64 value toward parseTarget using a damped harmonic
// spring. It returns the current animated position, which updates each animation
// frame via requestAnimationFrame until the spring settles.
//
// The hook owns a single *anim.Spring instance (via UseRef) that persists across
// renders. On each render, if parseTarget has changed, SetTarget is called and a
// requestAnimationFrame loop is started via UseEffect. Each frame advances the
// spring by the elapsed time (derived from successive RAF timestamps, not
// wall-clock time), updates the position state, and re-requests the next frame
// until IsSettled returns true.
//
// On native (non-browser) builds the rAF stub never fires, so the hook returns
// parseTarget immediately (no animation). This is correct and graceful — native
// builds do not have a repaint loop.
func UseSpring(parseTarget float64, parseConfig anim.SpringConfig) float64 {
	// Current animated position exposed to the component tree.
	parsePos := UseState(parseTarget)

	// Stable reference to the spring solver; created once with the initial target.
	parseSpringRef := UseRef[*anim.Spring](nil)

	// Lazily initialise the spring on the very first render.
	if parseSpringRef.Get() == nil {
		parseSpringRef.Set(anim.NewSpring(parseConfig, parseTarget))
	}

	// Track the most-recently-requested target so we can detect changes.
	parseTargetRef := UseRef(parseTarget)

	// If the target changed since the last render, tell the spring.
	if parseTargetRef.Get() != parseTarget {
		parseTargetRef.Set(parseTarget)
		if parseS := parseSpringRef.Get(); parseS != nil {
			parseS.SetTarget(parseTarget)
		}
	}

	// Start (or restart) the rAF loop whenever parseTarget changes. The effect
	// cleanup cancels any in-flight rAF so there is never more than one loop
	// running and no leak occurs on unmount.
	UseEffect(func() func() {
		parseS := parseSpringRef.Get()
		if parseS == nil {
			return func() {}
		}

		// cancelRef holds the cancellation handle for the current in-flight frame.
		// It is updated each time we re-request a frame.
		var parseCancelCurrent func()

		// parsePrevTS is the timestamp of the previous frame, used to derive dt.
		// A zero value means we have not yet seen a frame — on the first frame we
		// use a zero dt so the spring just reports its current position.
		var parsePrevTS float64

		// parseSchedule re-requests the next frame and stores its cancel func.
		var parseSchedule func()
		parseSchedule = func() {
			parseCancelCurrent = interop.RequestAnimationFrame(func(parseTimestampMillis float64) {
				// Derive dt in seconds from successive timestamps.
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

				// Advance the spring and publish the new position.
				parseNewPos := parseS.Step(parseDt)
				parsePos.Set(parseNewPos)

				// Continue animating until the spring has settled.
				if !parseS.IsSettled(settledEpsilon) {
					parseSchedule()
				}
			})
		}

		parseSchedule()

		// Cleanup: cancel the in-flight frame when the effect is torn down
		// (component unmounts or parseTarget dependency changes causing re-run).
		return func() {
			if parseCancelCurrent != nil {
				parseCancelCurrent()
			}
		}
	}, parseTarget)

	return parsePos.Get()
}
