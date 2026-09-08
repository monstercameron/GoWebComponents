//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// FuzzRenderBootstrapScriptBreakout fuzzes the SSR inline-bootstrap renderer for
// the script-breakout invariant: no matter what attacker-controlled strings flow
// through the script id, correlation id, route path, or atom keys/values, the
// emitted markup must contain exactly one opening <script and one closing
// </script> — i.e. embedded data can never terminate the script element early
// (which would enable arbitrary HTML/JS injection on resume). Also asserts the
// renderer never panics.
func FuzzRenderBootstrapScriptBreakout(parseF *testing.F) {
	parseF.Add("boot", "cid", "/home", "k", "v")
	parseF.Add("</script>", "</script>", "</script>", "</script>", "</script>")
	parseF.Add("x\"><script>alert(1)</script>", "a", "b", "c", "d")
	parseF.Add("", "", "", "", "")
	parseF.Add("a", "b", "/p", "</SCRIPT\t>", "<!--<script>")

	parseF.Fuzz(func(parseT *testing.T, parseScriptID, parseCID, parsePath, parseAtomKey, parseAtomVal string) {
		parsePayload := ui.SSRBootstrap{
			CorrelationID: parseCID,
			Route:         ui.SSRRouteBootstrap{Path: parsePath},
			Atoms:         map[string]any{parseAtomKey: parseAtomVal},
		}

		parseOut, parseErr := ui.RenderBootstrapScript(parsePayload, parseScriptID)
		if parseErr != nil {
			return // a normalization error is acceptable; only a breakout/panic is a bug
		}

		parseLower := strings.ToLower(parseOut)
		if parseOpen := strings.Count(parseLower, "<script"); parseOpen != 1 {
			parseT.Fatalf("expected exactly 1 <script, got %d: %q", parseOpen, parseOut)
		}
		if parseClose := strings.Count(parseLower, "</script"); parseClose != 1 {
			parseT.Fatalf("expected exactly 1 </script, got %d: %q", parseClose, parseOut)
		}
	})
}
