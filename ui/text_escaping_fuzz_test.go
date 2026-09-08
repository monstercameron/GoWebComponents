//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// FuzzTextNodeNoMarkup proves the text-content XSS invariant: rendering an
// arbitrary string as a bare Text node never emits a raw '<' or '>' — every
// angle bracket is entity-encoded, so no text payload can open a markup tag.
// The renderer must also never panic or error on a pure text node.
func FuzzTextNodeNoMarkup(parseF *testing.F) {
	for _, parseSeed := range []string{
		"", "plain", "<script>", "a & b", "x > y < z",
		"</div><img src=x onerror=alert(1)>", "\x00<\x00", "&lt;already&gt;",
		"unicode    \U0001F600", "\"'`",
	} {
		parseF.Add(parseSeed)
	}
	parseF.Fuzz(func(parseT *testing.T, parseIn string) {
		out, err := ui.RenderToString(ui.Text(parseIn))
		if err != nil {
			parseT.Fatalf("text render errored for %q: %v", parseIn, err)
		}
		if strings.ContainsAny(out, "<>") {
			parseT.Fatalf("raw angle bracket in rendered text: in=%q out=%q", parseIn, out)
		}
	})
}
