package runtime

import (
	"strings"
	"testing"
)

func aliasWarningFired() bool {
	for _, parseDiag := range GetDiagnostics() {
		if strings.Contains(parseDiag.Message, "without keys") && strings.Contains(parseDiag.Message, "share one identity") {
			return true
		}
	}
	return false
}

// TestReportUnkeyedComponentAliasing pins the #78/#38 diagnostic: two or more
// UNKEYED component siblings that share one identity (the inline-closure-per-list-
// item footgun) warn once; keyed siblings, host children, distinct components, and
// single children do not.
func TestReportUnkeyedComponentAliasing(parseT *testing.T) {
	if !hookThreadingGuardEnabled {
		parseT.Skip("diagnostic is dev-build only")
	}
	parseParent := &Fiber{}
	parseComp := func(any, map[string]any) *Element { return nil }
	parseOther := func(any, map[string]any) *Element { return nil }

	parseReset := func() {
		ClearDiagnostics()
		unkeyedComponentAliasWarned.Store(false)
	}
	defer parseReset()

	// Positive: two unkeyed siblings, same identity → warn.
	parseReset()
	reportUnkeyedComponentAliasing(parseParent, []any{&Element{Type: parseComp}, &Element{Type: parseComp}})
	if !aliasWarningFired() {
		parseT.Fatal("expected a warning for two unkeyed same-identity component siblings")
	}

	// Negative: keyed siblings → no warn.
	parseReset()
	reportUnkeyedComponentAliasing(parseParent, []any{&Element{Type: parseComp, Key: "a"}, &Element{Type: parseComp, Key: "b"}})
	if aliasWarningFired() {
		parseT.Fatal("keyed component siblings must not warn")
	}

	// Negative: distinct component identities → no warn.
	parseReset()
	reportUnkeyedComponentAliasing(parseParent, []any{&Element{Type: parseComp}, &Element{Type: parseOther}})
	if aliasWarningFired() {
		parseT.Fatal("distinct component identities must not warn")
	}

	// Negative: host (string-typed) siblings carry no hook state → no warn.
	parseReset()
	reportUnkeyedComponentAliasing(parseParent, []any{&Element{Type: "div"}, &Element{Type: "div"}})
	if aliasWarningFired() {
		parseT.Fatal("unkeyed host siblings must not warn")
	}

	// Negative: a single component child → no warn.
	parseReset()
	reportUnkeyedComponentAliasing(parseParent, []any{&Element{Type: parseComp}})
	if aliasWarningFired() {
		parseT.Fatal("a single component child must not warn")
	}

	// Warn-once: after firing, a second aliasing list does not add another warning.
	parseReset()
	reportUnkeyedComponentAliasing(parseParent, []any{&Element{Type: parseComp}, &Element{Type: parseComp}})
	ClearDiagnostics()
	reportUnkeyedComponentAliasing(parseParent, []any{&Element{Type: parseComp}, &Element{Type: parseComp}})
	if aliasWarningFired() {
		parseT.Fatal("the aliasing warning must fire at most once per process")
	}
}
