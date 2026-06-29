//go:build !(js && wasm)

package ui

import "time"

// defaultScheduleTimer is a no-op on native/SSR (no event loop to schedule on).
func defaultScheduleTimer(parseFn func(), parseDelay time.Duration, parseRepeat bool) func() {
	return nil
}
