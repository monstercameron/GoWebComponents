package u

import (
	"strconv"
	"strings"

	"github.com/monstercameron/GoWebComponents/v6/css"
)

// Variants in the u layer mirror Tailwind's prefix variants but compose as
// function nesting: u.Md(u.Hover(u.Flex)) == md:hover:flex. State variants
// delegate straight to Layer 1; responsive variants resolve the breakpoint from
// the active theme.

// State variants.
func Hover(rules ...css.Rule) []css.Rule        { return css.Hover(rules...) }
func Focus(rules ...css.Rule) []css.Rule        { return css.Focus(rules...) }
func Active(rules ...css.Rule) []css.Rule       { return css.Active(rules...) }
func WhenDisabled(rules ...css.Rule) []css.Rule { return css.WhenDisabled(rules...) }

// Group/Peer variants via parent/sibling state selectors. Mark the parent with
// the "group"/"peer" class (plain html.Class("group")).
func GroupHover(rules ...css.Rule) []css.Rule {
	return css.DefineVariant(".group:hover &")(rules...)
}
func PeerFocus(rules ...css.Rule) []css.Rule {
	return css.DefineVariant(".peer:focus ~ &")(rules...)
}

// Dark / Light are the prefers-color-scheme variants.
func Dark(rules ...css.Rule) []css.Rule  { return css.Media(css.Dark, rules...) }
func Light(rules ...css.Rule) []css.Rule { return css.Media(css.Light, rules...) }

// The accessibility preference variants. They exist as named functions for the same
// reason the css.MediaQuery constants do: a misspelled RawMedia query compiles, emits,
// and never matches, so the accommodation silently does not exist. A misspelled
// function name does not compile.
//
//	css.New(u.Transition(u.PropColors, u.Ms(120), u.Ease),
//	        u.ReducedMotion(u.TransitionDuration(u.Ms(0)))...)
func ReducedMotion(rules ...css.Rule) []css.Rule {
	return css.Media(css.ReducedMotion, rules...)
}

// MotionOK is the inverse of ReducedMotion — gate decorative motion ON with this
// rather than gating it off with ReducedMotion, so the no-preference default is calm.
func MotionOK(rules ...css.Rule) []css.Rule { return css.Media(css.MotionOK, rules...) }

// ContrastMore / ContrastLess are the prefers-contrast variants.
func ContrastMore(rules ...css.Rule) []css.Rule { return css.Media(css.ContrastMore, rules...) }
func ContrastLess(rules ...css.Rule) []css.Rule { return css.Media(css.ContrastLess, rules...) }

// ForcedColors is the Windows-High-Contrast / forced-colors variant. Inside it the UA
// owns the palette, so the useful work is restoring structure (borders on boxes that
// were distinguished only by background color).
func ForcedColors(rules ...css.Rule) []css.Rule { return css.Media(css.ForcedColors, rules...) }

// Print scopes rules to paged output.
func Print(rules ...css.Rule) []css.Rule { return css.Media(css.Print, rules...) }

// Responsive variants resolve their breakpoint min-width from the active theme.
func Sm(rules ...css.Rule) []css.Rule  { return breakpoint("sm", rules) }
func Md(rules ...css.Rule) []css.Rule  { return breakpoint("md", rules) }
func Lg(rules ...css.Rule) []css.Rule  { return breakpoint("lg", rules) }
func Xl(rules ...css.Rule) []css.Rule  { return breakpoint("xl", rules) }
func Xl2(rules ...css.Rule) []css.Rule { return breakpoint("2xl", rules) }

func breakpoint(name string, rules []css.Rule) []css.Rule {
	width, ok := css.BreakpointValue(name)
	if !ok {
		return rules
	}
	return css.Media(css.RawMedia("(min-width:"+normalizePx(width)+")"), rules...)
}

// normalizePx returns the breakpoint length unchanged when it already has units;
// the theme stores breakpoints as css.Length so this is just a string passthrough.
func normalizePx(v css.Length) string {
	s := string(v)
	if s == "" {
		return "0px"
	}
	// Ensure a bare integer carries px (defensive; theme values already have units).
	if _, err := strconv.Atoi(strings.TrimSpace(s)); err == nil {
		return s + "px"
	}
	return s
}

// Important appends !important to every declaration in the given rules. Use
// sparingly — it is the typed analog of Tailwind's "!" modifier.
func Important(rules ...css.Rule) []css.Rule {
	out := make([]css.Rule, 0, len(rules))
	for _, r := range rules {
		out = append(out, css.MarkImportant(r))
	}
	return out
}
