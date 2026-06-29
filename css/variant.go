package css

import "strconv"

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

// Dark is the prefers-color-scheme:dark media query.
const Dark MediaQuery = "(prefers-color-scheme:dark)"

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
