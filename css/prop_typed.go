package css

import "strings"

// This file adds typed constructors for the common properties that previously
// forced callers into the css.Property(string,string) escape hatch. With these,
// the typed path is the easy path and Raw/Sel stay rare.

// --- cursor -------------------------------------------------------------------

type cursorProp struct {
	Auto       Rule
	Pointer    Rule
	Default    Rule
	Text       Rule
	Move       Rule
	NotAllowed Rule
	Wait       Rule
	Grab       Rule
	Grabbing   Rule
}

// Cursor is the typed namespace for the cursor property: css.Cursor.Pointer.
var Cursor = cursorProp{
	Auto:       decl("cursor", "auto"),
	Pointer:    decl("cursor", "pointer"),
	Default:    decl("cursor", "default"),
	Text:       decl("cursor", "text"),
	Move:       decl("cursor", "move"),
	NotAllowed: decl("cursor", "not-allowed"),
	Wait:       decl("cursor", "wait"),
	Grab:       decl("cursor", "grab"),
	Grabbing:   decl("cursor", "grabbing"),
}

// --- user-select --------------------------------------------------------------

type selectProp struct {
	None Rule
	Text Rule
	All  Rule
	Auto Rule
}

// Select is the typed namespace for user-select: css.UserSelect.None.
var UserSelect = selectProp{
	None: decl("user-select", "none"),
	Text: decl("user-select", "text"),
	All:  decl("user-select", "all"),
	Auto: decl("user-select", "auto"),
}

// --- text -----------------------------------------------------------

type textTransformProp struct {
	None       Rule
	Uppercase  Rule
	Lowercase  Rule
	Capitalize Rule
}

// TextTransform is the typed namespace for text-transform.
var TextTransform = textTransformProp{
	None:       decl("text-transform", "none"),
	Uppercase:  decl("text-transform", "uppercase"),
	Lowercase:  decl("text-transform", "lowercase"),
	Capitalize: decl("text-transform", "capitalize"),
}

// Tracking sets letter-spacing from a typed Length: css.Tracking(css.Ems(0.18)).
func Tracking(v Length) Rule { return decl("letter-spacing", string(v)) }

// LineHeight sets line-height from a typed Number (unitless) or Length.
func LineHeight(n Number) Rule { return decl("line-height", string(n)) }

// LineHeightLen sets line-height from a typed Length.
func LineHeightLen(v Length) Rule { return decl("line-height", string(v)) }

// FontVariantNumeric exposes the common tabular-nums toggle as a typed value.
type fontVariantNumericProp struct {
	Normal      Rule
	TabularNums Rule
}

var FontVariantNumeric = fontVariantNumericProp{
	Normal:      decl("font-variant-numeric", "normal"),
	TabularNums: decl("font-variant-numeric", "tabular-nums"),
}

// --- transition ---------------------------------------------------------------

// TransitionProperty names one property — or a comma-separated LIST of properties
// (see PropColors) — for Transition. The All/Colors/Transform/Opacity presets cover
// the common cases; Prop wraps an arbitrary property name and TransitionProps
// composes several into a list.
type TransitionProperty string

const (
	PropAll       TransitionProperty = "all"
	PropColors    TransitionProperty = "color, background-color, border-color, fill, stroke"
	PropOpacity   TransitionProperty = "opacity"
	PropTransform TransitionProperty = "transform"
	PropShadow    TransitionProperty = "box-shadow"
)

// Prop names an arbitrary transition property (typed wrapper, not a free arg).
func Prop(parseName string) TransitionProperty { return TransitionProperty(parseName) }

// TransitionProps composes several properties into one comma-separated
// TransitionProperty list, the same shape as the PropColors preset:
//
//	css.Transition(css.TransitionProps(css.PropOpacity, css.PropTransform), css.Ms(120), css.Ease)
//
// Transition expands the list correctly (one transition per property) — see the
// grammar note on Transition.
func TransitionProps(parseProps ...TransitionProperty) TransitionProperty {
	parts := make([]string, 0, len(parseProps))
	for _, p := range parseProps {
		for _, name := range splitCommaList(string(p)) {
			parts = append(parts, name)
		}
	}
	return TransitionProperty(strings.Join(parts, ", "))
}

// splitCommaList splits a top-level comma-separated CSS value list, ignoring
// commas nested inside parentheses (so "cubic-bezier(0.4,0,0.2,1)" and
// "rgba(0,0,0,0.5)" stay intact) and trimming surrounding whitespace. Empty
// segments are dropped so a trailing comma cannot produce a dead entry.
func splitCommaList(parseValue string) []string {
	var out []string
	depth := 0
	start := 0
	flush := func(end int) {
		if segment := strings.TrimSpace(parseValue[start:end]); segment != "" {
			out = append(out, segment)
		}
	}
	for i := 0; i < len(parseValue); i++ {
		switch parseValue[i] {
		case '(':
			depth++
		case ')':
			if depth > 0 {
				depth--
			}
		case ',':
			if depth == 0 {
				flush(i)
				start = i + 1
			}
		}
	}
	flush(len(parseValue))
	return out
}

// Easing is a typed timing-function. Use the presets or CubicBezier.
type Easing string

const (
	Linear    Easing = "linear"
	Ease      Easing = "ease"
	EaseIn    Easing = "ease-in"
	EaseOut   Easing = "ease-out"
	EaseInOut Easing = "ease-in-out"
)

// CubicBezier builds a typed cubic-bezier easing.
func CubicBezier(p1, p2, p3, p4 float64) Easing {
	return Easing("cubic-bezier(" + trimFloat(p1) + "," + trimFloat(p2) + "," + trimFloat(p3) + "," + trimFloat(p4) + ")")
}

// Transition builds a typed transition shorthand: property, duration, easing.
//
// GRAMMAR — DO NOT "simplify" the expansion below back into a concatenation.
// The `transition` shorthand is a comma-separated list of *whole transitions*,
// not a property list followed by shared timing:
//
//	transition: <prop> <dur> <ease>, <prop> <dur> <ease>, …
//
// So the naive `string(property)+" "+dur+" "+ease` is silently wrong the moment
// property holds more than one name. With PropColors it emitted
//
//	transition: color, background-color, border-color, fill, stroke 120ms ease
//
// which the browser parses as FOUR transitions with the default 0s duration
// (`color`, `background-color`, `border-color`, `fill`) plus one 120ms transition
// on `stroke` — i.e. the preset animated nothing anybody could see. The comma
// belongs *between* transitions, so a property list has to be distributed over the
// duration/easing rather than pasted in front of them.
//
// Emission therefore expands one transition per property:
//
//	Transition(PropOpacity, Ms(120), Ease) -> transition:opacity 120ms ease
//	Transition(PropColors, Ms(120), Ease)  -> transition:color 120ms ease,
//	                                          background-color 120ms ease, … (5 total)
//
// The single-property case is byte-identical to the old output, so class hashes for
// single-property transitions are unchanged. (The longhand alternative —
// transition-property/-duration/-timing-function — is equally correct CSS but would
// have changed every single-property class hash, so the shorthand expansion wins.)
// Commas nested inside functional values (cubic-bezier(...), rgba(...)) are not
// list separators and are preserved; see splitCommaList.
func Transition(property TransitionProperty, duration Duration, easing Easing) Rule {
	timing := " " + string(duration) + " " + string(easing)
	names := splitCommaList(string(property))
	if len(names) <= 1 {
		// Includes the degenerate empty case, which keeps the previous behavior
		// (a leading space rather than a panic) for an empty TransitionProperty.
		return decl("transition", string(property)+timing)
	}
	var b strings.Builder
	b.Grow(len(property) + len(names)*(len(timing)+1))
	for i, name := range names {
		if i > 0 {
			b.WriteByte(',')
		}
		b.WriteString(name)
		b.WriteString(timing)
	}
	return decl("transition", b.String())
}

// TransitionLonghands is the longhand form of Transition:
// transition-property / -duration / -timing-function as three declarations.
//
// It exists for the case where a later rule (a reduced-motion override, say) wants
// to replace ONLY the duration: canonicalize folds declarations by property name
// within a scope, so an override of `transition-duration` beats the longhand but
// cannot reach inside the `transition` shorthand's single value. Prefer Transition
// unless you specifically need that.
func TransitionLonghands(property TransitionProperty, duration Duration, easing Easing) Rule {
	return Rule{decls: []declaration{
		{"transition-property", string(property)},
		{"transition-duration", string(duration)},
		{"transition-timing-function", string(easing)},
	}}
}

// TransitionDuration sets transition-duration on its own — the typed path for a
// reduced-motion override that neutralizes motion without restating the property
// list (pair it with TransitionLonghands).
func TransitionDuration(duration Duration) Rule {
	return decl("transition-duration", string(duration))
}

// --- transform ----------------------------------------------------------------

// TransformFn is a single typed transform function (Scale, Rotate, …). Compose
// several in one Transform call.
type TransformFn string

// Scale builds a uniform scale(): css.Scale(0.94).
func Scale(n float64) TransformFn { return TransformFn("scale(" + trimFloat(n) + ")") }

// ScaleXY builds scale(x, y).
func ScaleXY(x, y float64) TransformFn {
	return TransformFn("scale(" + trimFloat(x) + "," + trimFloat(y) + ")")
}

// Rotate builds rotate() from a typed Angle.
func Rotate(a Angle) TransformFn { return TransformFn("rotate(" + string(a) + ")") }

// TranslateX / TranslateY build translate functions from typed Lengths.
func TranslateX(v Length) TransformFn { return TransformFn("translateX(" + string(v) + ")") }
func TranslateY(v Length) TransformFn { return TransformFn("translateY(" + string(v) + ")") }

// Transform builds the transform property from one or more typed transform fns.
func Transform(fns ...TransformFn) Rule {
	parts := make([]string, 0, len(fns))
	for _, f := range fns {
		parts = append(parts, string(f))
	}
	return decl("transform", strings.Join(parts, " "))
}

// --- box-shadow & outline -----------------------------------------------------

// ShadowToken is a typed box-shadow value. Use the preset tokens or RawShadow.
type ShadowToken string

// String returns the box-shadow value, matching the other typed CSS value types.
func (parseToken ShadowToken) String() string { return string(parseToken) }

const (
	ShadowNone ShadowToken = "none"
	ShadowSm   ShadowToken = "0 1px 2px 0 rgba(0,0,0,0.05)"
	ShadowMd   ShadowToken = "0 4px 6px -1px rgba(0,0,0,0.1)"
	ShadowLg   ShadowToken = "0 10px 15px -3px rgba(0,0,0,0.1)"
	ShadowXl   ShadowToken = "0 20px 25px -5px rgba(0,0,0,0.1)"
	Shadow2xl  ShadowToken = "0 25px 50px -12px rgba(0,0,0,0.25)"
)

// ShadowOf builds one shadow from typed parts: offset, blur, spread, color.
//
//	ShadowOf(Zero, Px(1), Px(2), Zero, RGBA(0,0,0,0.05)) -> "0 1px 2px 0 rgba(0,0,0,0.05)"
func ShadowOf(x, y, blur, spread Length, c Color) ShadowToken {
	return ShadowToken(string(x) + " " + string(y) + " " + string(blur) + " " + string(spread) + " " + string(c))
}

// ShadowInset is ShadowOf with the `inset` keyword (an inner shadow).
func ShadowInset(x, y, blur, spread Length, c Color) ShadowToken {
	return "inset " + ShadowOf(x, y, blur, spread, c)
}

// RawShadow is the escape hatch for a box-shadow value no preset token or
// ShadowOf call covers (a `drop-shadow`-style stack copied from a design tool,
// a currentColor ring, …): RawShadow("0 0 0 3px currentColor").
//
// Named like RawLength/RawMedia so raw usage stays greppable. The value is emitted
// verbatim, so it is author-trusted CSS — see hardenCSS for what is (and is not)
// guaranteed about untrusted input.
func RawShadow(parseValue string) ShadowToken { return ShadowToken(parseValue) }

// Shadow sets box-shadow from a typed token.
func Shadow(token ShadowToken) Rule { return decl("box-shadow", string(token)) }

// Shadows layers several shadows into one box-shadow value.
//
// Unlike `transition`, box-shadow's comma-separated list IS a list of complete
// values with nothing appended after it, so joining tokens with a comma is the
// whole grammar — there is no per-item timing to distribute (contrast Transition).
// ShadowNone is not a list item; a "none" among several shadows makes the entire
// declaration invalid, so it is dropped here rather than emitted into the list.
func Shadows(tokens ...ShadowToken) ShadowToken {
	parts := make([]string, 0, len(tokens))
	for _, t := range tokens {
		if t == "" || t == ShadowNone {
			continue
		}
		parts = append(parts, string(t))
	}
	if len(parts) == 0 {
		return ShadowNone
	}
	return ShadowToken(strings.Join(parts, ","))
}

// Outline builds a solid outline of the given width and color.
func Outline(width Length, c Color) Rule {
	return decl("outline", solidLine(width, c))
}

// OutlineOffset sets outline-offset from a typed Length.
func OutlineOffset(v Length) Rule { return decl("outline-offset", string(v)) }

// --- numeric props ------------------------------------------------------------

// OpacityNum sets opacity from a typed Number.
func OpacityNum(n Number) Rule { return decl("opacity", string(n)) }

// --- escape hatches (the ONLY string-accepting authoring fns) -----------------

// Raw is the explicit declaration escape hatch for any property/value not covered
// by a typed constructor: css.Raw("backdrop-filter", "blur(4px)"). Intentionally
// named so it is greppable / lint-gateable; prefer a typed constructor.
func Raw(parseProperty, parseValue string) Rule { return decl(parseProperty, parseValue) }
