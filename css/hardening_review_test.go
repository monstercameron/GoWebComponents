//go:build !(js && wasm)

package css_test

import (
	"math"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/css"
)

// TestNegativeZeroFoldsToZero pins that -0 and 0 produce identical value text, so
// they cannot mint two classes for the same style.
func TestNegativeZeroFoldsToZero(parseT *testing.T) {
	parseNegZero := math.Copysign(0, -1)
	if css.Rem(parseNegZero) != css.Rem(0) || css.Num(parseNegZero) != css.Num(0) || css.Percent(parseNegZero) != css.Percent(0) {
		parseT.Fatalf("-0 formatted differently from 0: %q vs %q", css.Rem(parseNegZero), css.Rem(0))
	}
	css.Reset()
	if css.New(css.Gap(css.Rem(parseNegZero))) != css.New(css.Gap(css.Rem(0))) {
		parseT.Fatalf("-0 and 0 gap folded to different classes")
	}
}

// TestInvalidNumericArgumentsStayLiveCSS pins that out-of-range numeric arguments
// are clamped rather than emitted as declarations the browser would drop.
func TestInvalidNumericArgumentsStayLiveCSS(parseT *testing.T) {
	if parseGot := string(css.CubicBezier(-0.5, 2, 1.5, 3)); parseGot != "cubic-bezier(0,2,1,3)" {
		parseT.Fatalf("CubicBezier x not clamped: %q", parseGot)
	}
	if parseGot := string(css.Repeat(0, css.Fr(1))); parseGot != "repeat(1,1fr)" {
		parseT.Fatalf("Repeat(0) = %q", parseGot)
	}
	if parseGot := string(css.Repeat(-3, css.Fr(1))); parseGot != "repeat(1,1fr)" {
		parseT.Fatalf("Repeat(-3) = %q", parseGot)
	}
	if parseGot := css.GridSpan(0).String(); parseGot != "span 1" {
		parseT.Fatalf("GridSpan(0) = %q", parseGot)
	}
	if parseGot := css.GridSpan(-2).String(); parseGot != "span 1" {
		parseT.Fatalf("GridSpan(-2) = %q", parseGot)
	}
}

// TestDynamicAndElementNamesCannotInjectCSS pins that hostile names passed to
// DynamicVar and El cannot close var()/the selector and add declarations or rules.
func TestDynamicAndElementNamesCannotInjectCSS(parseT *testing.T) {
	css.Reset()
	parseDynamic := css.DynamicVar("x); } body{display:none} .y{z:", "gap", "1px")
	css.New(parseDynamic.Rule())
	css.New(css.Child(css.El("a{} body{display:none}"), css.Display.Flex)...)
	parseOut := css.Harvest()
	if strings.Contains(parseOut, "body{display:none}") || strings.Contains(parseOut, "} body") {
		parseT.Fatalf("hostile name injected CSS: %q", parseOut)
	}
}
