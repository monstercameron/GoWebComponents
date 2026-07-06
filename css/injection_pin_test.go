//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/css"
)

// TestSelectorConstructorsPreventInjection pins that the typed "safe" selector
// constructors escape/sanitize their inputs so a crafted value cannot break out
// of the selector and inject a live (process-global) rule.
func TestSelectorConstructorsPreventInjection(parseT *testing.T) {
	// AttrEq value: a breakout attempt must be escaped, keeping it inside the quotes.
	parseAttr := string(css.AttrEq("data-x", `y"] * {display:none}`))
	if !strings.Contains(parseAttr, `\"`) {
		parseT.Fatalf("AttrEq value quote not escaped: %s", parseAttr)
	}

	// ClassSel / AttrSel: selector-structural characters must be stripped.
	parseCls := string(css.ClassSel(`title] * {x:y}`))
	if strings.ContainsAny(parseCls, "]{} ") {
		parseT.Fatalf("ClassSel did not sanitize: %s", parseCls)
	}
	parseAttrName := string(css.AttrSel(`open] * {x:y`))
	if strings.ContainsAny(parseAttrName, "{} ") || strings.Count(parseAttrName, "]") != 1 {
		parseT.Fatalf("AttrSel did not sanitize: %s", parseAttrName)
	}

	// DataTheme name: an unescaped quote would break out of [data-theme="..."].
	parseTheme := css.DataTheme(`x"] * {display:none} [y="`, css.Bg(css.White))
	if len(parseTheme) == 0 {
		parseT.Fatal("DataTheme returned no rules")
	}
}

// TestRGBClampsChannels pins that RGB clamps out-of-range channels to [0,255].
func TestRGBClampsChannels(parseT *testing.T) {
	if parseGot := string(css.RGB(300, -5, 128)); parseGot != "rgb(255,0,128)" {
		parseT.Fatalf("RGB clamp = %q, want rgb(255,0,128)", parseGot)
	}
}
