//go:build production

package runtime

import "time"

// Production already removes the timers that feed per-fiber/component traces.
// Avoid retaining the remaining map/timestamp bookkeeping with zero durations;
// explicit user-reported profiling events remain available.
const runtimeHotPathProfilingEnabled = false

// Production builds drop per-fiber commit timing: the no-op implementations
// make the timing branches dead code the compiler eliminates.

func commitTimingStart() time.Time {
	return time.Time{}
}

func commitTimingSinceNs(parseStart time.Time) int64 {
	return 0
}
