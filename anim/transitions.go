package anim

import "sort"

// ---------------------------------------------------------------------------
// Keyed-list FLIP orchestration
// ---------------------------------------------------------------------------

// KeyedRect pairs a stable list key with the rectangle it occupies. Measure the list
// before a reorder/insert/remove (the "first" layout) and again after (the "last"
// layout), then feed both to DiffKeyedRects to drive a FLIP animation.
type KeyedRect struct {
	Key  string
	Rect Rect
}

// ListTransition is the classification of a keyed layout change for a FLIP animation:
// which items entered, which left, and the invert transform for each item that survived.
type ListTransition struct {
	// Entering holds keys present in the new layout but not the old, in new-layout order.
	Entering []string
	// Exiting holds keys present in the old layout but not the new, in old-layout order.
	Exiting []string
	// Moving maps each surviving key to the FLIP invert transform that places it back at
	// its old position; animating that transform to identity plays the move. A key that
	// did not move maps to the identity transform.
	Moving map[string]FLIPTransform
}

// MovedKeys returns the surviving keys whose invert transform is not the identity (the
// items that actually need a move animation), sorted for deterministic iteration.
func (parseT ListTransition) MovedKeys() []string {
	var parseKeys []string
	for parseKey, parseTransform := range parseT.Moving {
		if parseTransform != (FLIPTransform{ScaleX: 1, ScaleY: 1, TranslateX: 0, TranslateY: 0}) {
			parseKeys = append(parseKeys, parseKey)
		}
	}
	sort.Strings(parseKeys)
	return parseKeys
}

// DiffKeyedRects classifies a keyed layout change and computes the FLIP invert transform
// for every surviving key. It is pure and order-stable: Entering follows new-layout order,
// Exiting follows old-layout order.
func DiffKeyedRects(parsePrev, parseNext []KeyedRect) ListTransition {
	parsePrevByKey := make(map[string]Rect, len(parsePrev))
	for _, parseItem := range parsePrev {
		parsePrevByKey[parseItem.Key] = parseItem.Rect
	}
	parseNextKeys := make(map[string]struct{}, len(parseNext))
	for _, parseItem := range parseNext {
		parseNextKeys[parseItem.Key] = struct{}{}
	}

	parseResult := ListTransition{Moving: map[string]FLIPTransform{}}
	for _, parseItem := range parseNext {
		parseFirst, parseSurvived := parsePrevByKey[parseItem.Key]
		if !parseSurvived {
			parseResult.Entering = append(parseResult.Entering, parseItem.Key)
			continue
		}
		parseResult.Moving[parseItem.Key] = ComputeFLIP(parseFirst, parseItem.Rect)
	}
	for _, parseItem := range parsePrev {
		if _, parseStillPresent := parseNextKeys[parseItem.Key]; !parseStillPresent {
			parseResult.Exiting = append(parseResult.Exiting, parseItem.Key)
		}
	}
	return parseResult
}

// ---------------------------------------------------------------------------
// Accessibility: prefers-reduced-motion
// ---------------------------------------------------------------------------

// MotionPreference is the user's motion preference, sourced on the client from the
// `prefers-reduced-motion` media query. Keeping it an explicit value (rather than reading
// the media query deep inside the animation code) keeps this package pure and testable,
// and forces every animated surface to make an accessible decision at the call site.
type MotionPreference int

const (
	// MotionFull plays animations normally (prefers-reduced-motion: no-preference).
	MotionFull MotionPreference = iota
	// MotionReduced collapses non-essential motion: transitions snap and FLIP moves are
	// skipped (prefers-reduced-motion: reduce).
	MotionReduced
)

// Animates reports whether motion should play for this preference. Use it to skip a FLIP
// move animation entirely under reduced motion.
func (parseP MotionPreference) Animates() bool {
	return parseP != MotionReduced
}

// EffectiveDuration returns the duration to use for this preference: the requested
// duration under MotionFull, or 0 under MotionReduced so the transition snaps instantly to
// its end state instead of moving.
func (parseP MotionPreference) EffectiveDuration(parseDuration float64) float64 {
	if parseP == MotionReduced {
		return 0
	}
	return parseDuration
}

// ---------------------------------------------------------------------------
// Enter / exit transition state machine
// ---------------------------------------------------------------------------

// Phase is the lifecycle stage of an enter/exit transition.
type Phase int

const (
	// PhaseEntering means the element is mounting and animating in.
	PhaseEntering Phase = iota
	// PhaseEntered means the enter animation has completed; the element is at rest.
	PhaseEntered
	// PhaseExiting means the element is animating out before removal.
	PhaseExiting
	// PhaseExited means the exit animation has completed; the element may be unmounted.
	PhaseExited
)

// String renders the phase for logs and diagnostics.
func (parseP Phase) String() string {
	switch parseP {
	case PhaseEntering:
		return "entering"
	case PhaseEntered:
		return "entered"
	case PhaseExiting:
		return "exiting"
	case PhaseExited:
		return "exited"
	default:
		return "unknown"
	}
}

// Transition is a pure, time-driven enter/exit state machine. It owns no DOM and no clock:
// the caller advances it by elapsed seconds, making it deterministic to unit-test and
// trivial to drive from a render loop or a fake clock.
type Transition struct {
	Phase    Phase
	Elapsed  float64 // seconds spent in the current animated phase
	Duration float64 // seconds the enter/exit animation runs
}

// NewTransition starts an element in PhaseEntering with the given enter/exit duration.
func NewTransition(parseDuration float64) Transition {
	return Transition{Phase: PhaseEntering, Duration: parseDuration}
}

// NewTransitionPref starts a transition whose duration honors the user's motion
// preference: under MotionReduced the duration collapses to 0 so the element snaps in/out
// with no animation, satisfying prefers-reduced-motion without branching at the call site.
func NewTransitionPref(parseDuration float64, parsePref MotionPreference) Transition {
	return NewTransition(parsePref.EffectiveDuration(parseDuration))
}

// Advance progresses the transition by parseDelta seconds, auto-promoting an entering
// element to PhaseEntered and an exiting element to PhaseExited once Elapsed reaches
// Duration. Settled phases (Entered/Exited) are unaffected. Negative deltas are ignored.
func (parseT Transition) Advance(parseDelta float64) Transition {
	if parseDelta <= 0 || (parseT.Phase != PhaseEntering && parseT.Phase != PhaseExiting) {
		return parseT
	}
	parseT.Elapsed += parseDelta
	if parseT.Elapsed >= parseT.Duration {
		parseT.Elapsed = parseT.Duration
		if parseT.Phase == PhaseEntering {
			parseT.Phase = PhaseEntered
		} else {
			parseT.Phase = PhaseExited
		}
	}
	return parseT
}

// BeginExit moves an entering or entered element into PhaseExiting, resetting the clock so
// the exit animation plays from the start. An already-exiting/exited element is unchanged.
func (parseT Transition) BeginExit() Transition {
	if parseT.Phase == PhaseExiting || parseT.Phase == PhaseExited {
		return parseT
	}
	parseT.Phase = PhaseExiting
	parseT.Elapsed = 0
	return parseT
}

// Progress returns the eased-input fraction 0..1 through the current animated phase. A
// settled Entered phase returns 1; a settled Exited phase returns 0. A zero duration
// snaps to the end of the phase.
func (parseT Transition) Progress() float64 {
	switch parseT.Phase {
	case PhaseEntered:
		return 1
	case PhaseExited:
		return 0
	}
	if parseT.Duration <= 0 {
		return 1
	}
	parseFraction := parseT.Elapsed / parseT.Duration
	if parseFraction < 0 {
		return 0
	}
	if parseFraction > 1 {
		return 1
	}
	return parseFraction
}

// IsAnimating reports whether the transition is mid enter or exit (a render loop should
// keep ticking while this is true).
func (parseT Transition) IsAnimating() bool {
	return parseT.Phase == PhaseEntering || parseT.Phase == PhaseExiting
}

// IsRemovable reports whether the exit animation has finished and the element can be
// unmounted from the DOM.
func (parseT Transition) IsRemovable() bool {
	return parseT.Phase == PhaseExited
}

// StaggerDelay returns the start delay, in seconds, for the item at parseIndex given a
// per-item parseStep — the staggered-cascade timing used for list enter animations.
// A negative index or step yields zero.
func StaggerDelay(parseIndex int, parseStep float64) float64 {
	if parseIndex <= 0 || parseStep <= 0 {
		return 0
	}
	return float64(parseIndex) * parseStep
}
