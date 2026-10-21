package u_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v7/css"
	"github.com/monstercameron/GoWebComponents/v7/css/u"
)

// TestMissingBreakpointIsNoOp pins that a responsive variant whose breakpoint is
// absent from the active theme contributes no rules instead of applying always.
func TestMissingBreakpointIsNoOp(parseT *testing.T) {
	parsePrev := css.ActiveTheme()
	defer css.UseTheme(parsePrev)
	parseTheme := css.DefaultTheme()
	parseTheme.Breakpoints = map[string]css.Length{"md": "768px"}
	css.UseTheme(parseTheme)

	for parseName, parseFn := range map[string]func(...css.Rule) []css.Rule{"sm": u.Sm, "lg": u.Lg, "xl": u.Xl, "2xl": u.Xl2} {
		if parseGot := parseFn(u.Opacity(60)); len(parseGot) != 0 {
			parseT.Errorf("%s: missing breakpoint must yield no rules, got %d", parseName, len(parseGot))
		}
	}
	if parseGot := u.Md(u.Opacity(60)); len(parseGot) == 0 {
		parseT.Errorf("defined breakpoint md must still produce a media rule")
	}
}