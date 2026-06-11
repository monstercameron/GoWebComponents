//go:build !production

package runtime

import "time"

// commitTimingEnabled controls per-fiber commit/cleanup duration capture.
// The timings feed devtools inspection and slow-operation diagnostics; in
// production-tagged builds the constant false lets the compiler remove the
// two time.Now calls per committed fiber entirely.
const commitTimingEnabled = true

func commitTimingStart() time.Time {
	return time.Now()
}

func commitTimingSinceNs(parseStart time.Time) int64 {
	return time.Since(parseStart).Nanoseconds()
}
