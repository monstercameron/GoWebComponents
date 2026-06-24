//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
)

// FuzzNewRawHardening fuzzes the hashed-class emit path (New + Raw, distinct from
// Inject) to prove it too cannot be driven to emit a <style>/comment breakout
// through an arbitrary property value.
func FuzzNewRawHardening(parseF *testing.F) {
	for _, parseSeed := range []string{
		"", "red", "</style>", "url(x)*/", "a\x00/", "<\x00/style>",
		"\"</style>\"", "}.evil{display:none", "1px;color:red",
	} {
		parseF.Add(parseSeed)
	}
	parseF.Fuzz(func(parseT *testing.T, parseVal string) {
		css.Reset()
		css.New(css.Raw("content", parseVal))
		parseOut := css.Harvest()
		parseLower := strings.ToLower(parseOut)
		if strings.Contains(parseLower, "</style") || strings.Contains(parseLower, "</script") {
			parseT.Fatalf("New+Raw emit leaked a style/script close for value %q -> %q", parseVal, parseOut)
		}
		if strings.Contains(parseOut, "*/") {
			parseT.Fatalf("New+Raw emit leaked */ comment-close for value %q -> %q", parseVal, parseOut)
		}
	})
}
