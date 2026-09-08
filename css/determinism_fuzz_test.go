//go:build !(js && wasm)

package css_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/css"
)

// FuzzNewOrderIndependence fuzzes the documented determinism invariant: folding
// the same set of distinct-property rules in any order must yield the same
// content-hashed class (and the same emitted CSS). Order-dependence here would
// break dedup and build reproducibility. (Same-property rules are last-write-wins
// by design, so the property is checked only for distinct properties.)
func FuzzNewOrderIndependence(parseF *testing.F) {
	parseF.Add("color", "red", "margin", "0")
	parseF.Add("a", "1", "b", "2")
	parseF.Add("display", "flex", "display", "grid")
	parseF.Fuzz(func(parseT *testing.T, parseP1, parseV1, parseP2, parseV2 string) {
		if parseP1 == parseP2 {
			return // same property: last-write-wins, order is meant to matter
		}
		css.Reset()
		parseForward := css.New(css.Raw(parseP1, parseV1), css.Raw(parseP2, parseV2))
		parseForwardCSS := css.Harvest()

		css.Reset()
		parseReverse := css.New(css.Raw(parseP2, parseV2), css.Raw(parseP1, parseV1))
		parseReverseCSS := css.Harvest()

		if parseForward != parseReverse {
			parseT.Fatalf("order-dependent class: %q vs %q for {%q:%q,%q:%q}",
				parseForward, parseReverse, parseP1, parseV1, parseP2, parseV2)
		}
		if parseForwardCSS != parseReverseCSS {
			parseT.Fatalf("order-dependent CSS:\n%q\nvs\n%q", parseForwardCSS, parseReverseCSS)
		}
	})
}
