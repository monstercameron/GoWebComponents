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

// TransitionProperty names a property for Transition. The All/Colors/Transform/
// Opacity presets cover the common cases; Prop wraps an arbitrary property name.
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
func Transition(property TransitionProperty, duration Duration, easing Easing) Rule {
	return decl("transition", string(property)+" "+string(duration)+" "+string(easing))
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

// Shadow is a typed box-shadow value. Use the preset tokens or RawShadow.
type ShadowToken string

const (
	ShadowNone ShadowToken = "none"
	ShadowSm   ShadowToken = "0 1px 2px 0 rgba(0,0,0,0.05)"
	ShadowMd   ShadowToken = "0 4px 6px -1px rgba(0,0,0,0.1)"
	ShadowLg   ShadowToken = "0 10px 15px -3px rgba(0,0,0,0.1)"
	ShadowXl   ShadowToken = "0 20px 25px -5px rgba(0,0,0,0.1)"
	Shadow2xl  ShadowToken = "0 25px 50px -12px rgba(0,0,0,0.25)"
)

// Shadow sets box-shadow from a typed token.
func Shadow(token ShadowToken) Rule { return decl("box-shadow", string(token)) }

// Outline builds a solid outline of the given width and color.
func Outline(width Length, c Color) Rule {
	return decl("outline", string(width)+" solid "+string(c))
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
