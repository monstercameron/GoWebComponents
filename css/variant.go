package css

import (
	"strconv"
	"strings"
)

// Variants wrap inner rules in a selector and/or at-rule scope. They compose by
// nesting — css.Media(css.MinW(768), css.Hover(css.Display.Flex)) produces a
// :hover rule inside a min-width media query.

// Hover scopes rules to the :hover state.
func Hover(rules ...Rule) []Rule { return applyVariant("&:hover", "", rules) }

// Focus scopes rules to the :focus state.
func Focus(rules ...Rule) []Rule { return applyVariant("&:focus", "", rules) }

// FocusVisible scopes rules to :focus-visible.
func FocusVisible(rules ...Rule) []Rule { return applyVariant("&:focus-visible", "", rules) }

// Active scopes rules to :active.
func Active(rules ...Rule) []Rule { return applyVariant("&:active", "", rules) }

// Disabled scopes rules to the :disabled pseudo-class. It uses the bare pseudo-class
// name like the other variants (Hover, Focus, Active, …).
func Disabled(rules ...Rule) []Rule { return applyVariant("&:disabled", "", rules) }

// WhenDisabled scopes rules to :disabled.
//
// Deprecated: use Disabled, which matches the bare pseudo-class naming of the other variants.
func WhenDisabled(rules ...Rule) []Rule { return Disabled(rules...) }

// FirstChild scopes rules to the :first-child structural pseudo-class.
func FirstChild(rules ...Rule) []Rule { return applyVariant("&:first-child", "", rules) }

// LastChild scopes rules to the :last-child structural pseudo-class.
func LastChild(rules ...Rule) []Rule { return applyVariant("&:last-child", "", rules) }

// Before / After scope to the ::before / ::after pseudo-elements. A content
// declaration defaults to "" when absent so the pseudo-element renders.
func Before(rules ...Rule) []Rule {
	return applyVariant("&::before", "", withDefaultContent(rules))
}
func After(rules ...Rule) []Rule {
	return applyVariant("&::after", "", withDefaultContent(rules))
}

func withDefaultContent(rules []Rule) []Rule {
	hasContent := false
	for _, r := range rules {
		for _, d := range r.decls {
			if d.property == "content" {
				hasContent = true
			}
		}
	}
	if hasContent {
		return rules
	}
	return append([]Rule{decl("content", "\"\"")}, rules...)
}

// Media is a media-query at-rule wrapper. Build the query with MinW/MaxW or pass
// a raw query string via RawMedia.
type MediaQuery string

// MinW is a min-width media query.
func MinW(px int) MediaQuery { return MediaQuery("(min-width:" + strconv.Itoa(px) + "px)") }

// MaxW is a max-width media query.
func MaxW(px int) MediaQuery { return MediaQuery("(max-width:" + strconv.Itoa(px) + "px)") }

// Dark / Light are the prefers-color-scheme media queries.
const (
	Dark  MediaQuery = "(prefers-color-scheme:dark)"
	Light MediaQuery = "(prefers-color-scheme:light)"
)

// The accessibility preference queries. These are typed constants rather than
// RawMedia strings for a specific reason: a media query is not validated by anything.
// A typo in RawMedia("(prefers-reduced-motion: reduse)") does not fail to compile, does
// not fail to emit, and does not warn — it produces a syntactically valid @media block
// that simply never matches, so the accessibility accommodation silently does not
// exist. These three (plus MotionOK) are the a11y floor, so they are the ones that must
// be unmisspellable.
const (
	// ReducedMotion matches when the user has asked the OS to reduce animation
	// (Windows "Show animations", macOS "Reduce motion", GNOME/Android equivalents).
	// Motion under this query should be neutralized, not merely shortened.
	ReducedMotion MediaQuery = "(prefers-reduced-motion:reduce)"
	// MotionOK is the inverse — the explicit "no preference" state. Prefer gating
	// motion ON with MotionOK over gating it OFF with ReducedMotion when the
	// animation is decorative: the default then costs nothing for a user whose
	// preference is unknown.
	MotionOK MediaQuery = "(prefers-reduced-motion:no-preference)"

	// ContrastMore / ContrastLess match prefers-contrast. Under ContrastMore, raise
	// border and text contrast rather than adding decoration.
	ContrastMore MediaQuery = "(prefers-contrast:more)"
	ContrastLess MediaQuery = "(prefers-contrast:less)"

	// ForcedColors matches when the platform has substituted its own palette
	// (Windows High Contrast / forced-colors mode). Inside it, colors are overridden
	// by the UA, so the useful work is restoring structure the forced palette
	// erases — borders on elements that were distinguished only by background color,
	// and forced-color-adjust:none on the few places a brand color must survive.
	ForcedColors MediaQuery = "(forced-colors:active)"

	// ReducedTransparency and ReducedData round out the preference set.
	ReducedTransparency MediaQuery = "(prefers-reduced-transparency:reduce)"
	ReducedData         MediaQuery = "(prefers-reduced-data:reduce)"
)

// Print matches paged output.
const Print MediaQuery = "print"

// Hover / NoHover match the hover media feature — the correct test for "does this
// pointer have a hover state", as opposed to inferring it from viewport width.
// (Named HoverCapable/NoHover to avoid colliding with the Hover pseudo-class variant.)
const (
	HoverCapable MediaQuery = "(hover:hover)"
	NoHover      MediaQuery = "(hover:none)"
)

// MediaAll combines features with `and`, so every one must match:
//
//	MediaAll(MinW(768), Dark) -> "(min-width:768px) and (prefers-color-scheme:dark)"
//
// Note this is `and`, not the comma that @media uses for `or`. A comma-joined query
// list would match if ANY feature matched, which is almost never what a caller
// nesting two conditions means — hence only the `and` form is provided typed.
func MediaAll(queries ...MediaQuery) MediaQuery {
	parts := make([]string, 0, len(queries))
	for _, q := range queries {
		if q == "" {
			continue
		}
		parts = append(parts, string(q))
	}
	return MediaQuery(strings.Join(parts, " and "))
}

// RawMedia is the escape hatch for an arbitrary media feature query.
func RawMedia(parseQuery string) MediaQuery { return MediaQuery(parseQuery) }

// Media scopes rules inside an @media at-rule.
func Media(query MediaQuery, rules ...Rule) []Rule {
	return applyVariant("", "@media "+string(query), rules)
}

// DefineVariant returns a reusable variant function from a selector template
// containing a single "&" placeholder (e.g. ".group:hover &" for group-hover, or
// "&[data-open]" for a data-attribute state). It is the open extension point for
// new pseudo/combinator variants beyond the built-ins.
func DefineVariant(selectorTemplate string) func(...Rule) []Rule {
	return func(rules ...Rule) []Rule {
		return applyVariant(selectorTemplate, "", rules)
	}
}

// --- keyframes ----------------------------------------------------------------

// Frame is a single keyframe step: an offset (e.g. "0%", "from", "100%") and the
// rules applied at that offset.
type Frame struct {
	Offset string
	Rules  []Rule
}

// At builds a keyframe Frame at the given offset.
func At(offset string, rules ...Rule) Frame { return Frame{Offset: offset, Rules: rules} }

// Keyframes registers an @keyframes animation built from frames and returns a
// Rule that both carries the (deduped, content-named) @keyframes block and sets
// animation-name to it, so applying the rule animates the element. duration and
// timing are applied via the returned rule's shorthand.
func Keyframes(parseName string, frames ...Frame) Rule {
	body := keyframesBody(frames)
	name := parseName + "-" + shortHash(body)
	block := "@keyframes " + name + "{" + body + "}"
	return Rule{
		decls: []declaration{{"animation-name", name}},
		raw:   []string{block},
	}
}

func keyframesBody(frames []Frame) string {
	var b []byte
	for _, f := range frames {
		b = append(b, f.Offset...)
		b = append(b, '{')
		// reuse the canonical declaration serialization for a stable body.
		_, groups, _ := canonicalize(f.Rules)
		for _, g := range groups {
			b = append(b, g.body...)
		}
		b = append(b, '}')
	}
	return string(b)
}

// Animation sets the animation shorthand timing for a Keyframes rule. Compose it
// alongside the Keyframes rule in the same New(...) call. It takes a typed Duration and Easing
// (matching Transition), so a unit-less or mistyped value is a compile error: Animation(Ms(200), Linear).
func Animation(duration Duration, timing Easing) Rule {
	return Rule{decls: []declaration{
		{"animation-duration", string(duration)},
		{"animation-timing-function", string(timing)},
	}}
}
