//go:build production

package runtime

import "time"

// Production builds drop per-fiber commit timing: the constant false makes
// the timing branches dead code the compiler eliminates.
const commitTimingEnabled = false

func commitTimingStart() time.Time {
	return time.Time{}
}

func commitTimingSinceNs(parseStart time.Time) int64 {
	return 0
}
