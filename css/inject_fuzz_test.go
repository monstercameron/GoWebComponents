//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/css"
)

// FuzzInjectHardening fuzzes the arbitrary-CSS-string injection surface to prove
// the <style>-breakout hardening holds for ANY input: the emitted text can never
// contain a literal </style> or </script> (which would terminate the style raw-
// text element) or a */ (which would close a CSS comment).
func FuzzInjectHardening(parseF *testing.F) {
	for _, parseSeed := range []string{
		"", "a{}", "</style>", "</STYLE>", "<script>x</script>", "a{}*/b{}",
		"\x00", "</\nstyle>", "<!--", "@font-face{}</style>",
	} {
		parseF.Add(parseSeed)
	}
	parseF.Fuzz(func(parseT *testing.T, parseIn string) {
		css.Reset()
		css.Inject("fuzz", parseIn)
		parseOut := css.Harvest()
		parseLower := strings.ToLower(parseOut)
		if strings.Contains(parseLower, "</style") {
			parseT.Fatalf("breakout: </style survived for input %q -> %q", parseIn, parseOut)
		}
		if strings.Contains(parseLower, "</script") {
			parseT.Fatalf("breakout: </script survived for input %q -> %q", parseIn, parseOut)
		}
		if strings.Contains(parseOut, "*/") {
			parseT.Fatalf("breakout: */ comment-close survived for input %q -> %q", parseIn, parseOut)
		}
	})
}
