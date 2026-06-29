package ui

import "time"

// Timer hooks (G19): UseTimeout/UseInterval own the js timer + js.Func lifetime
// via the effect cleanup — no manual setTimeout/clearTimeout/Release. On the
// native/SSR build they are safe no-ops.

// scheduleTimer is the platform seam: schedule fn after delay (repeating when
// repeat is true) and return a cancel func. Swappable for tests.
var scheduleTimer = defaultScheduleTimer

// UseTimeout runs fn once after delay while the component is mounted; it is
// cancelled (and its js.Func released) on unmount or when deps change.
func UseTimeout(parseFn func(), parseDelay time.Duration, parseDeps ...any) {
	useTimer(parseFn, parseDelay, false, parseDeps...)
}

// UseInterval runs fn every interval while the component is mounted; cancelled on
// unmount or when deps change.
func UseInterval(parseFn func(), parseInterval time.Duration, parseDeps ...any) {
	useTimer(parseFn, parseInterval, true, parseDeps...)
}

func useTimer(parseFn func(), parseDelay time.Duration, parseRepeat bool, parseDeps ...any) {
	parseEffectDeps := timerDeps(parseDelay, parseRepeat, parseDeps)
	UseEffect(func() func() {
		if parseFn == nil || parseDelay < 0 {
			return nil
		}
		return scheduleTimer(parseFn, parseDelay, parseRepeat)
	}, parseEffectDeps...)
}

// timerDeps yields a stable, non-empty dep set when none are given (so the timer
// is scheduled once on mount, not re-scheduled every render).
func timerDeps(parseDelay time.Duration, parseRepeat bool, parseDeps []any) []any {
	if len(parseDeps) == 0 {
		return []any{timerSentinel, int64(parseDelay), parseRepeat}
	}
	return parseDeps
}

const timerSentinel = "gwc-timer"
