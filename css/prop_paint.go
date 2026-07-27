package css

import "strings"

// Gradients, background images, and the background sizing/positioning longhands.
//
// This is the surface that signature visual devices are made of — a perforated tear
// edge, a ruled-paper backdrop, a scrim over a hero — and it had no typed form at
// all, so every one of them was a block of css.Raw. Nothing here emits the
// `background` SHORTHAND: see the note on BgImage.

// --- gradient stops -------------------------------------------------------------

// ColorStop is one entry in a gradient's stop list. Build one with Stop, StopAt,
// StopSpan, or ColorHint.
type ColorStop string

func (s ColorStop) String() string { return string(s) }

// Stop is a color with no explicit position — the gradient distributes it evenly.
func Stop(c Color) ColorStop { return ColorStop(c) }

// StopAt pins a color to a position along the gradient line: StopAt(White, Percent(40)).
func StopAt(c Color, pos Length) ColorStop {
	return ColorStop(string(c) + " " + string(pos))
}

// StopSpan gives a color a start AND end position, so it holds flat between them and
// then changes abruptly. This is how hard-edged devices (stripes, dot grids,
// perforations) are drawn without an image asset:
//
//	StopSpan(paper, Zero, RawLength("3.5px"))  -> "<paper> 0 3.5px"
func StopSpan(c Color, from, to Length) ColorStop {
	return ColorStop(string(c) + " " + string(from) + " " + string(to))
}

// ColorHint is a bare position between two stops that shifts the midpoint of the
// blend without introducing a third color.
func ColorHint(pos Length) ColorStop { return ColorStop(pos) }

// joinStops joins a gradient stop list with COMMAS. Unlike a grid track list (spaces)
// and unlike the transition shorthand (where the comma separates whole transitions),
// a gradient's argument list is genuinely comma-separated and nothing is appended
// after it, so a plain comma join is the correct grammar here.
func joinStops(prefix string, stops []ColorStop) string {
	parts := make([]string, 0, len(stops)+1)
	if prefix != "" {
		parts = append(parts, prefix)
	}
	for _, s := range stops {
		if s == "" {
			continue
		}
		parts = append(parts, string(s))
	}
	return strings.Join(parts, ",")
}

// --- gradient images ------------------------------------------------------------

// Image is a CSS <image> value: a gradient, a url(), or the none keyword. Build one
// with the gradient constructors or URLImage.
type Image string

func (i Image) String() string { return string(i) }

// NoImage is the `none` keyword — the typed way to clear an inherited
// background-image.
const NoImage Image = "none"

// GradientDirection is a typed `to <side>` gradient direction.
type GradientDirection string

const (
	ToTop         GradientDirection = "to top"
	ToBottom      GradientDirection = "to bottom"
	ToLeft        GradientDirection = "to left"
	ToRight       GradientDirection = "to right"
	ToTopLeft     GradientDirection = "to top left"
	ToTopRight    GradientDirection = "to top right"
	ToBottomLeft  GradientDirection = "to bottom left"
	ToBottomRight GradientDirection = "to bottom right"
)

// LinearGradient builds linear-gradient(<angle>, stops…). The angle is the direction
// the gradient travels TOWARD, measured clockwise from "up": Deg(180) points down.
func LinearGradient(a Angle, stops ...ColorStop) Image {
	return Image("linear-gradient(" + joinStops(string(a), stops) + ")")
}

// LinearGradientTo builds linear-gradient(to <side>, stops…) — the keyword form,
// which unlike an angle stays correct when the box is not square.
func LinearGradientTo(dir GradientDirection, stops ...ColorStop) Image {
	return Image("linear-gradient(" + joinStops(string(dir), stops) + ")")
}

// RepeatingLinearGradient tiles a linear gradient's stop list — stripes, rules,
// ruled-paper backdrops.
func RepeatingLinearGradient(dir GradientDirection, stops ...ColorStop) Image {
	return Image("repeating-linear-gradient(" + joinStops(string(dir), stops) + ")")
}

// GradientShape is the typed shape/position prefix of a radial gradient.
type GradientShape string

const (
	Circle  GradientShape = "circle"
	Ellipse GradientShape = "ellipse"
)

// CircleAt / EllipseAt position the gradient's center:
// CircleAt(Percent(50), Percent(100)) -> "circle at 50% 100%".
func CircleAt(x, y Length) GradientShape {
	return GradientShape("circle at " + string(x) + " " + string(y))
}
func EllipseAt(x, y Length) GradientShape {
	return GradientShape("ellipse at " + string(x) + " " + string(y))
}

// CircleSizedAt gives the circle an explicit radius as well as a center.
func CircleSizedAt(radius Length, x, y Length) GradientShape {
	return GradientShape("circle " + string(radius) + " at " + string(x) + " " + string(y))
}

// RadialGradient builds radial-gradient(<shape>, stops…).
func RadialGradient(shape GradientShape, stops ...ColorStop) Image {
	return Image("radial-gradient(" + joinStops(string(shape), stops) + ")")
}

// RepeatingRadialGradient tiles a radial gradient — dot grids, and the
// half-circle perforation of a torn stub:
//
//	RepeatingRadialGradient(CircleAt(Percent(50), Percent(100)),
//	    StopSpan(paper, Zero, RawLength("3.5px")),
//	    StopAt(Transparent, RawLength("3.5px")))
func RepeatingRadialGradient(shape GradientShape, stops ...ColorStop) Image {
	return Image("repeating-radial-gradient(" + joinStops(string(shape), stops) + ")")
}

// ConicGradient builds conic-gradient(from <angle> at <x> <y>, stops…) — pie/donut
// meters without SVG.
func ConicGradient(from Angle, x, y Length, stops ...ColorStop) Image {
	prefix := "from " + string(from) + " at " + string(x) + " " + string(y)
	return Image("conic-gradient(" + joinStops(prefix, stops) + ")")
}

// URLImage builds url("…") from a path. The path is emitted inside a quoted CSS
// string with '"' and '\' escaped, so it cannot close the url() and inject
// declarations; it is NOT scheme-checked, so treat the path as author-trusted.
func URLImage(parsePath string) Image {
	return Image("url(\"" + cssStringEscape(parsePath) + "\")")
}

// RawImage is the escape hatch for an <image> value the constructors do not cover
// (image-set(), cross-fade(), a paint() worklet).
func RawImage(parseValue string) Image { return Image(parseValue) }

// --- background properties -------------------------------------------------------

// BgImage sets background-image from one or more layers. background-image IS a
// comma-separated list of layers, first layer on top, and nothing is appended after
// it — so joining with commas is the correct grammar (contrast Transition, where the
// comma separates whole transitions and a bare property list is silently wrong).
//
// There is deliberately NO Background() shorthand constructor in this package. The
// `background` shorthand is comma-separated per LAYER and each layer packs
// color/image/position/size/repeat/attachment/origin/clip in one slot; building it by
// concatenating a caller's image value with more tokens is exactly the defect class
// that made the old Transition emit dead CSS. Set background-color with Bg and the
// image/size/repeat/position longhands from here instead — as a bonus, a later rule
// can then override just the size without restating the image.
func BgImage(images ...Image) Rule {
	parts := make([]string, 0, len(images))
	for _, img := range images {
		if img == "" {
			continue
		}
		parts = append(parts, string(img))
	}
	if len(parts) == 0 {
		return decl("background-image", string(NoImage))
	}
	return decl("background-image", strings.Join(parts, ","))
}

// BgSizeValue is one background-size layer value.
type BgSizeValue string

func (v BgSizeValue) String() string { return string(v) }

const (
	// BgCover scales the image to cover the box, cropping the overflow;
	// BgContain fits it entirely, leaving gaps.
	BgCover   BgSizeValue = "cover"
	BgContain BgSizeValue = "contain"
	BgAuto    BgSizeValue = "auto"
)

// BgSizeXY is an explicit width/height tile size: BgSizeXY(Px(11), Px(7)) -> "11px 7px".
func BgSizeXY(w, h Length) BgSizeValue {
	return BgSizeValue(string(w) + " " + string(h))
}

// BgSize sets background-size, one value per BgImage layer (comma-separated list,
// same grammar note as BgImage).
func BgSize(sizes ...BgSizeValue) Rule {
	parts := make([]string, 0, len(sizes))
	for _, s := range sizes {
		if s == "" {
			continue
		}
		parts = append(parts, string(s))
	}
	if len(parts) == 0 {
		return decl("background-size", string(BgAuto))
	}
	return decl("background-size", strings.Join(parts, ","))
}

type bgRepeatProp struct {
	Repeat   Rule
	NoRepeat Rule
	RepeatX  Rule
	RepeatY  Rule
	// Round rescales the tile so a whole number fits; Space keeps the tile size and
	// distributes the remainder as gaps.
	Round Rule
	Space Rule
}

// BgRepeat is the typed namespace for background-repeat: css.BgRepeat.RepeatX.
var BgRepeat = bgRepeatProp{
	Repeat:   decl("background-repeat", "repeat"),
	NoRepeat: decl("background-repeat", "no-repeat"),
	RepeatX:  decl("background-repeat", "repeat-x"),
	RepeatY:  decl("background-repeat", "repeat-y"),
	Round:    decl("background-repeat", "round"),
	Space:    decl("background-repeat", "space"),
}

// BgPosition sets background-position from typed offsets.
func BgPosition(x, y Length) Rule {
	return decl("background-position", string(x)+" "+string(y))
}

type bgClipProp struct {
	BorderBox  Rule
	PaddingBox Rule
	ContentBox Rule
	// Text clips the background to the glyphs, the gradient-text device. Emitted
	// with the -webkit- longhand as well, which is still required in WebKit/Blink.
	Text Rule
}

// BgClip is the typed namespace for background-clip.
var BgClip = bgClipProp{
	BorderBox:  decl("background-clip", "border-box"),
	PaddingBox: decl("background-clip", "padding-box"),
	ContentBox: decl("background-clip", "content-box"),
	Text: Rule{decls: []declaration{
		{"-webkit-background-clip", "text"},
		{"background-clip", "text"},
	}},
}

// BgOrigin positions the image relative to a box edge.
type bgOriginProp struct {
	BorderBox  Rule
	PaddingBox Rule
	ContentBox Rule
}

var BgOrigin = bgOriginProp{
	BorderBox:  decl("background-origin", "border-box"),
	PaddingBox: decl("background-origin", "padding-box"),
	ContentBox: decl("background-origin", "content-box"),
}

// --- masks ----------------------------------------------------------------------

// MaskImage sets mask-image from the same typed Image values the gradients produce —
// the other half of the signature-device toolkit, where a gradient decides not what
// is painted but what survives. A fade-out edge on a scroll container, a torn paper
// edge, an icon tinted by currentColor: all of them are "paint a solid, mask it with a
// gradient", which needs no image asset and inverts with the theme for free.
//
//	MaskImage(LinearGradientTo(ToBottom, Stop(Black), Stop(Transparent)))
//
// Both the -webkit- longhand and the standard property are emitted: Safari still
// requires the prefix, and canonicalize sorts "-webkit-mask-image" before
// "mask-image", so the standard property always wins where it is supported.
func MaskImage(images ...Image) Rule {
	parts := make([]string, 0, len(images))
	for _, img := range images {
		if img == "" {
			continue
		}
		parts = append(parts, string(img))
	}
	value := string(NoImage)
	if len(parts) > 0 {
		value = strings.Join(parts, ",")
	}
	return Rule{decls: []declaration{
		{"-webkit-mask-image", value},
		{"mask-image", value},
	}}
}

// MaskSize / MaskRepeat mirror BgSize / BgRepeat for the mask layer.
func MaskSize(sizes ...BgSizeValue) Rule {
	parts := make([]string, 0, len(sizes))
	for _, s := range sizes {
		if s == "" {
			continue
		}
		parts = append(parts, string(s))
	}
	value := string(BgAuto)
	if len(parts) > 0 {
		value = strings.Join(parts, ",")
	}
	return Rule{decls: []declaration{
		{"-webkit-mask-size", value},
		{"mask-size", value},
	}}
}

type maskRepeatProp struct {
	Repeat   Rule
	NoRepeat Rule
	RepeatX  Rule
	RepeatY  Rule
}

func maskRepeatRule(parseValue string) Rule {
	return Rule{decls: []declaration{
		{"-webkit-mask-repeat", parseValue},
		{"mask-repeat", parseValue},
	}}
}

// MaskRepeat is the typed namespace for mask-repeat.
var MaskRepeat = maskRepeatProp{
	Repeat:   maskRepeatRule("repeat"),
	NoRepeat: maskRepeatRule("no-repeat"),
	RepeatX:  maskRepeatRule("repeat-x"),
	RepeatY:  maskRepeatRule("repeat-y"),
}
