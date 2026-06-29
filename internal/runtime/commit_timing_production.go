//go:build production

package runtime

import "time"

// Production builds drop per-fiber commit timing: the no-op implementations
// make the timing branches dead code the compiler eliminates.

func commitTimingStart() time.Time {
	return time.Time{}
}

func commitTimingSinceNs(parseStart time.Time) int64 {
	return 0
}
