//go:build !production

package runtime

import "time"

// Per-fiber commit/cleanup duration capture. The timings feed devtools
// inspection and slow-operation diagnostics; production-tagged builds swap in
// no-op implementations so the compiler removes the two time.Now calls per
// committed fiber entirely.

func commitTimingStart() time.Time {
	return time.Now()
}

func commitTimingSinceNs(parseStart time.Time) int64 {
	return time.Since(parseStart).Nanoseconds()
}
