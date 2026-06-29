// Package deprecation provides a one-time structured warning helper for deprecated
// GoWebComponents APIs. Each unique API symbol is warned exactly once across
// concurrent callers; subsequent calls for the same symbol are no-ops.
package deprecation

import (
	"fmt"
	"sync"

	"github.com/monstercameron/GoWebComponents/v4/diagnostics"
)

// emit is the sink for structured reports. Replaced by same-package tests to
// capture emitted output without writing to stderr.
var emit = diagnostics.Emit

// newReport constructs a diagnostics.Report from an options value. Replaced by
// same-package tests to inspect the options that would have been used.
var newReport = diagnostics.NewReport

// parseWarned tracks which API symbols have already produced a warning.
var parseWarned sync.Map

// Warn emits a structured deprecation diagnostic for parseAPI exactly once.
// Subsequent calls with the same parseAPI value are silent no-ops.
// When parseReplacement is non-empty the diagnostic's Next field advises
// callers to switch to that replacement symbol; when empty, no advice is added.
func Warn(parseAPI string, parseReplacement string) {
	if _, parseAlreadyStored := parseWarned.LoadOrStore(parseAPI, struct{}{}); parseAlreadyStored {
		return
	}

	parseNext := ""
	if parseReplacement != "" {
		parseNext = fmt.Sprintf("use %s instead", parseReplacement)
	}

	parseOpts := diagnostics.Options{
		Code:     "GWC-DEPRECATION",
		Headline: fmt.Sprintf("deprecated API: %s", parseAPI),
		Summary:  fmt.Sprintf("%s is deprecated", parseAPI),
		Path:     parseAPI,
		Next:     parseNext,
	}
	emit(newReport(parseOpts))
}

// WarnEmitted reports whether a deprecation warning has already been emitted
// for parseAPI. It is a test seam that lets callers verify Warn fired without
// inspecting stderr.
func WarnEmitted(parseAPI string) bool {
	_, parseFound := parseWarned.Load(parseAPI)
	return parseFound
}

// resetForTest clears all recorded warnings so individual test cases start from
// a clean slate. It is unexported and intended only for same-package tests.
func resetForTest() {
	parseWarned.Range(func(parseKey, _ any) bool {
		parseWarned.Delete(parseKey)
		return true
	})
}
