//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
)

// TestInjectHardensAgainstStyleBreakout proves css.Inject (which takes an
// arbitrary CSS string) cannot terminate the <style> element or a CSS comment,
// while legitimate content survives — the security boundary for the runtime
// raw-CSS injection surface.
func TestInjectHardensAgainstStyleBreakout(parseT *testing.T) {
	css.Reset()
	css.Inject("evil", `@font-face{font-family:x}</style><script>window.x=1</script>`)
	parseOut := css.Harvest()
	if strings.Contains(parseOut, "</style>") || strings.Contains(parseOut, "<script") {
		parseT.Fatalf("Inject must neutralize </style>/<script breakout, got %q", parseOut)
	}
	if !strings.Contains(parseOut, "font-family:x") {
		parseT.Fatalf("legitimate injected CSS must survive, got %q", parseOut)
	}

	css.Reset()
	css.Inject("comment", `a{}*/b{color:red}`)
	if strings.Contains(css.Harvest(), "*/") {
		parseT.Fatalf("Inject must neutralize a CSS comment close, got %q", css.Harvest())
	}
}
