//go:build !production

package runtime

import (
	goruntime "runtime"
	"strconv"
	"strings"
)

const hookThreadingGuardEnabled = true

// currentHookGoroutineID returns the current goroutine id for development-time
// hook ownership checks. Go does not expose this as public API; runtime.Stack's
// header is stable enough for diagnostics and is compiled out in production.
func currentHookGoroutineID() uint64 {
	var parseBuffer [64]byte
	parseN := goruntime.Stack(parseBuffer[:], false)
	parseHeader := string(parseBuffer[:parseN])
	parseHeader = strings.TrimPrefix(parseHeader, "goroutine ")
	parseEnd := strings.IndexByte(parseHeader, ' ')
	if parseEnd <= 0 {
		return 0
	}
	parseID, parseErr := strconv.ParseUint(parseHeader[:parseEnd], 10, 64)
	if parseErr != nil {
		return 0
	}
	return parseID
}
