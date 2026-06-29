package u

import (
	"reflect"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/css"
)

// TestColorTokenResolvesAgainstTheme proves a built-in typed color token resolves to the
// theme color and that u.BgC(u.ColorSlate900) wires to css.Bg of that color.
func TestColorTokenResolvesAgainstTheme(parseT *testing.T) {
	css.UseTheme(css.DefaultTheme())
	parseWant, _ := css.ColorValue("slate-900")
	if parseGot := resolveColor(ColorSlate900); parseGot != parseWant {
		parseT.Fatalf("ColorSlate900 resolved to %q, want %q", parseGot, parseWant)
	}
	if parseGot, parseWantRule := BgC(ColorSlate900), css.Bg(parseWant); !reflect.DeepEqual(parseGot, parseWantRule) {
		parseT.Fatalf("BgC(ColorSlate900) = %+v, want css.Bg(slate-900) = %+v", parseGot, parseWantRule)
	}
}

// TestColorTokenUnknownFallsBackTransparent proves a token fabricated via conversion (the
// only way to dodge the typed constants) degrades to transparent rather than panicking.
func TestColorTokenUnknownFallsBackTransparent(parseT *testing.T) {
	css.UseTheme(css.DefaultTheme())
	if parseGot := resolveColor(ColorToken("does-not-exist")); parseGot != css.Transparent {
		parseT.Fatalf("unknown token resolved to %q, want transparent", parseGot)
	}
}
