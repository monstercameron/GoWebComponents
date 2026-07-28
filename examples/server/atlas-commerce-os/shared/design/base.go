package design

import (
	"sort"

	"github.com/monstercameron/GoWebComponents/v5/css"
)

// This file is the global layer: the only place the design system emits rules that
// are not scoped to a generated class. Everything here goes through css.Global,
// which emits the selector verbatim instead of hashing it.
//
// Keeping the global layer this small is the point. Global rules are the ones you
// cannot opt out of, so each one has to earn its place by being something that
// must be true of EVERY element (the reset, the token block, the focus ring) rather
// than something a component wants.

// Install emits Atlas's global foundation. Call it exactly once at startup, in both
// lanes (server bootstrap and wasm main), before the first render.
//
// It is idempotent: css.Global dedupes on (selector + rules) content, so a second
// call emits nothing. It is also cheap to call again after css.Reset() in a test,
// which is why the token block is NOT wrapped in a sync.Once — a Once would leave a
// test with primitives referencing tokens that no longer exist.
//
// Order matters and is the conventional cascade order:
//
//  1. css.Preflight — the framework's reset. Reused rather than reimplemented:
//     box-sizing, margin zeroing, `font: inherit` on form controls, and
//     `font-size/font-weight: inherit` on headings. That last one is what lets
//     Display(StepSubhead) on an <h2> actually mean something instead of fighting
//     the UA stylesheet.
//  2. The :root token block, then the dark-mode override of the same names.
//  3. The element baseline (html/body/table/form defaults).
//  4. The accessibility floor (focus ring, selection).
//  5. The print layer (see print.go) — LAST, because it is a third :root token block
//     that has to beat both the light and the dark ones. A dark-mode user printing a
//     receipt still wants ink on paper, and both media queries can match at once, so
//     the tie is broken by emission order.
//
// CONVERSION NOTE: Install assumes it owns the base layer. While Atlas still loads
// examples/static/css/tailwind.css and example-shell.css, calling Install will emit
// a second, competing reset. Remove those two stylesheets in the same change that
// wires this in.
func Install() {
	css.Preflight()
	installTokens()
	installElementBaseline()
	installAccessibilityFloor()
	installPrint()
}

// installTokens emits the light palette into :root and the dark palette into a
// prefers-color-scheme override of the SAME names.
//
// This is the whole theming mechanism, and it is worth understanding why it is
// enough: no class is regenerated, no primitive is duplicated, and no component
// branches on a theme. A primitive says "paint with ink"; the media query decides
// what ink is. Adding a third theme means adding a third :root block.
func installTokens() {
	css.Global(":root", declarations(lightTokenValues())...)

	// css.Dark is the typed (prefers-color-scheme:dark) query — preferred over
	// css.RawMedia("(prefers-color-scheme: dark)") because a typo in a raw media
	// string is a silently dead block, not a compile error.
	css.Global(":root", css.Media(css.Dark, declarations(darkTokenValues())...)...)
}

// declarations turns a token map into custom-property rules in a deterministic
// order. Sorting is belt-and-braces: css.canonicalize already sorts declarations by
// property before hashing, so emission is deterministic regardless — but a sorted
// input makes that independence obvious to a reader instead of load-bearing on a
// detail of another package.
//
// Custom properties go through css.Raw because there is no typed constructor for
// them; css.Root's own doc comment names Raw as the intended path.
func declarations(parseValues map[string]string) []css.Rule {
	parseNames := make([]string, 0, len(parseValues))
	for parseName := range parseValues {
		parseNames = append(parseNames, parseName)
	}
	sort.Strings(parseNames)

	parseRules := make([]css.Rule, 0, len(parseNames))
	for _, parseName := range parseNames {
		parseRules = append(parseRules, css.Raw(parseName, parseValues[parseName]))
	}
	return parseRules
}

// installElementBaseline sets the defaults an unstyled element should already have,
// so markup that forgets a class still lands inside the design system rather than
// in the UA stylesheet.
func installElementBaseline() {
	// `color-scheme: light dark` declares that BOTH schemes are supported, so the
	// UA renders form controls, scrollbars and the canvas to match the user's
	// preference. This is the correct form of the declaration that
	// example-shell.css got wrong: it hard-set `color-scheme: dark`, which locks
	// the UA to dark regardless of preference — one half of why the app rendered
	// near-black while reporting THEME: light.
	css.Global("html",
		css.Raw("color-scheme", "light dark"),
		// Anchor scrolling under the sticky storefront header / sticky table head.
		css.Raw("scroll-padding-top", "5rem"),
	)

	// This `body` block and Preflight's both target `body` at equal specificity, so the
	// later emission wins — which is why Install calls Preflight FIRST. Reordering the
	// two calls would silently hand Atlas Preflight's line-height instead of the
	// design system's.
	css.Global("body",
		css.Bg(Paper()),
		css.TextColor(Ink()),
		fontFamily(TokenFontBody),
		css.FontSize(stepMetrics(StepBase).size),
		css.LineHeight(stepMetrics(StepBase).proseLineHeight),
		css.MinHeight(css.Vh(100)),
		// Opt into the text figures Atlas actually wants everywhere: lining,
		// tabular. A price inside a sentence should still align with the price in
		// the table above it.
		css.FontVariantNumeric.TabularNums,
		css.Raw("text-underline-offset", "0.15em"),
	)

	// Tables collapse their borders by default, because every table in Atlas is a
	// hairline manifest and a doubled border is always a bug here.
	css.Global("table", css.Raw("border-collapse", "collapse"))

	// Form controls inherit the paper/ink pair rather than the UA's white/black, so
	// a control that has not been given Input() yet is still legible in dark mode.
	css.Global("input,select,textarea,button",
		css.Bg(css.Transparent),
		css.TextColor(css.CurrentCo),
	)
}

// installAccessibilityFloor emits the guarantees that must hold for every element,
// including ones this design system has never seen.
//
// A focus ring declared per-primitive is a promise each primitive has to keep; a
// focus ring declared globally is a property of the app. Both are here: the global
// rule is the floor, and individual primitives add an inset/offset variant where
// their geometry needs one. If a new interactive element ships without a design
// class, it is still keyboard-visible.
func installAccessibilityFloor() {
	// :focus-visible rather than :focus, so a mouse click on a button does not draw
	// a ring while a Tab to it does. Text inputs match :focus-visible on click too,
	// per spec, so this covers them.
	css.Global(":focus-visible",
		css.Outline(ManifestRuleWidth, Lane()),
		css.OutlineOffset(css.Px(2)),
	)

	// Selection uses lane blue rather than the UA default, which on a manila paper
	// background can land on a near-invisible pale blue.
	css.Global("::selection", css.Bg(Lane()), css.TextColor(Paper()))
}

// fontFamily is the typed-ish wrapper over the one property the css package has no
// constructor for. It exists so `font-family` is written exactly once per role and
// always as a token reference, never as an inline stack.
//
// GAP NOTE: css/global.go's own doc comment advertises css.Font(css.SansStack),
// which does not exist in the package. font-family, and the whole font shorthand,
// are missing from the typed layer.
func fontFamily(parseToken string) css.Rule {
	return css.Raw("font-family", "var("+parseToken+")")
}

// hairlineRule / manifestRule build side-specific borders.
//
// GAP NOTE: css.Border(width, color) only emits the four-sided `border` shorthand.
// Every hairline in this design system is one-sided (a rule under a row, a rule
// beside the rail), so all of them go through css.Raw. BorderTop/Right/Bottom/Left
// constructors would remove most of this package's Raw usage on their own.
func hairlineRule() string { return string(HairlineWidth) + " solid var(" + TokenHairline + ")" }
func manifestRule() string { return string(ManifestRuleWidth) + " solid var(" + TokenInk + ")" }

// motionReduce is the reduced-motion query.
const motionReduce css.MediaQuery = "(prefers-reduced-motion: reduce)"

// easeManifest is the single easing curve in the system. One curve, because a
// design system with four easings has four different opinions about how fast the
// UI is, and nobody can tell them apart anyway.
var easeManifest = css.CubicBezier(0.2, 0, 0.2, 1)

// withMotion pairs a transition with its own reduced-motion neutralization, as one
// bundle, so motion cannot ship without the opt-out.
//
// The alternative — one global `* { transition-duration: 0.01ms !important }` under
// the media query — is the pattern most codebases use, and it is exactly the
// !important sledgehammer that made example-shell.css unfightable. Scoping the
// override into the primitive needs no !important at all: within a folded class,
// blocks are emitted sorted by at-rule, and "" sorts before "@media …", so the
// media block lands last and wins on order at equal specificity.
//
// Every transition in this package goes through here, and
// TestEveryTransitionCarriesAReducedMotionOverride proves it by counting emitted
// transition-duration declarations against reduced-motion blocks.
//
// GAP NOTE — and this one is a real bug in the typed layer, found by reading the
// emitted CSS rather than by reading the API:
//
//	css.Transition(css.PropColors, css.Ms(120), ease)
//
// emits `transition: color, background-color, border-color, fill, stroke 120ms …`.
// That is NOT five 120ms transitions. The transition shorthand is comma-separated
// per-property, so the browser parses four transitions with default 0s timing plus
// one (`stroke`) at 120ms — i.e. the multi-property PropColors preset silently
// animates nothing. Any TransitionProperty holding a comma is unusable through the
// shorthand. The longhands below are correct for a property LIST, which is why this
// helper does not use css.Transition at all.
//
// The 0.01ms (rather than 0ms) reduced-motion duration is deliberate: it still fires
// transitionend, so any code waiting on that event does not hang for users who asked
// for reduced motion.
func withMotion(parseProperty css.TransitionProperty, parseDuration css.Duration) []css.Rule {
	return css.Rules(
		css.Raw("transition-property", string(parseProperty)),
		css.Raw("transition-duration", string(parseDuration)),
		css.Raw("transition-timing-function", string(easeManifest)),
		css.Media(motionReduce, css.Raw("transition-duration", "0.01ms")),
	)
}
