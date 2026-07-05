//go:build !production

package runtime

import (
	goruntime "runtime"
)

const hookThreadingGuardEnabled = true

// computeHookGoroutineID returns the current goroutine id for development-time
// hook ownership checks. Go does not expose this as public API; runtime.Stack's
// header is stable enough for diagnostics and is compiled out in production.
// The header is parsed from the raw bytes without allocating: runtime.Stack is
// already a full traceback (microseconds), so callers cache the result per
// render session (see beginHookOwnershipSession) instead of calling this per
// hook invocation.
func computeHookGoroutineID() uint64 {
	var parseBuffer [64]byte
	parseN := goruntime.Stack(parseBuffer[:], false)
	const parsePrefix = "goroutine "
	if parseN <= len(parsePrefix) {
		return 0
	}
	parseID := uint64(0)
	for _, parseByte := range parseBuffer[len(parsePrefix):parseN] {
		if parseByte < '0' || parseByte > '9' {
			break
		}
		parseID = parseID*10 + uint64(parseByte-'0')
	}
	return parseID
}
