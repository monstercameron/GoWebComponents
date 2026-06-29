package css

import "github.com/monstercameron/GoWebComponents/deprecation"

// This file defines the typed property surface for Layer 1: typed namespaces of
// typed values (the css.Display.Flex shape) and typed property constructors
// (css.Gap(css.Px(8))). Every constructor returns a Layer-1 Rule.

// --- display ------------------------------------------------------------------

type displayProp struct {
	Block       Rule
	InlineBlock Rule
	Inline      Rule
	Flex        Rule
	InlineFlex  Rule
	Grid        Rule
	None        Rule
}

// Display is the typed namespace for the CSS display property:
// css.Display.Flex, css.Display.Grid, css.Display.None, …
var Display = displayProp{
	Block:       decl("display", "block"),
	InlineBlock: decl("display", "inline-block"),
	Inline:      decl("display", "inline"),
	Flex:        decl("display", "flex"),
	InlineFlex:  decl("display", "inline-flex"),
	Grid:        decl("display", "grid"),
	None:        decl("display", "none"),
}

// --- position -----------------------------------------------------------------

type positionProp struct {
	Static   Rule
	Relative Rule
	Absolute Rule
	Fixed    Rule
	Sticky   Rule
}

// Position is the typed namespace for the CSS position property.
var Position = positionProp{
	Static:   decl("position", "static"),
	Relative: decl("position", "relative"),
	Absolute: decl("position", "absolute"),
	Fixed:    decl("position", "fixed"),
	Sticky:   decl("position", "sticky"),
}

// --- flex layout --------------------------------------------------------------

type flexDirProp struct {
	Row    Rule
	RowRev Rule
	Col    Rule
	ColRev Rule
}

// FlexDir is the typed namespace for flex-direction.
var FlexDir = flexDirProp{
	Row:    decl("flex-direction", "row"),
	RowRev: decl("flex-direction", "row-reverse"),
	Col:    decl("flex-direction", "column"),
	ColRev: decl("flex-direction", "column-reverse"),
}

type alignProp struct {
	Start   Rule
	Center  Rule
	End     Rule
	Stretch Rule
}

// Items is the typed namespace for align-items.
var Items = alignProp{
	Start:   decl("align-items", "flex-start"),
	Center:  decl("align-items", "center"),
	End:     decl("align-items", "flex-end"),
	Stretch: decl("align-items", "stretch"),
}

type justifyProp struct {
	Start   Rule
	Center  Rule
	End     Rule
	Between Rule
	Around  Rule
}

// Justify is the typed namespace for justify-content.
var Justify = justifyProp{
	Start:   decl("justify-content", "flex-start"),
	Center:  decl("justify-content", "center"),
	End:     decl("justify-content", "flex-end"),
	Between: decl("justify-content", "space-between"),
	Around:  decl("justify-content", "space-around"),
}

// --- spacing & sizing ---------------------------------------------------------

// Gap sets the flex/grid gap.
func Gap(v Length) Rule { return decl("gap", string(v)) }

// Padding sets uniform padding; PaddingX/Y set the axis pairs.
func Padding(v Length) Rule { return decl("padding", string(v)) }
func PaddingX(v Length) Rule {
	return Rule{decls: []declaration{{"padding-left", string(v)}, {"padding-right", string(v)}}}
}
func PaddingY(v Length) Rule {
	return Rule{decls: []declaration{{"padding-top", string(v)}, {"padding-bottom", string(v)}}}
}

// Margin sets uniform margin; MarginX/Y set the axis pairs.
func Margin(v Length) Rule { return decl("margin", string(v)) }
func MarginX(v Length) Rule {
	return Rule{decls: []declaration{{"margin-left", string(v)}, {"margin-right", string(v)}}}
}
func MarginY(v Length) Rule {
	return Rule{decls: []declaration{{"margin-top", string(v)}, {"margin-bottom", string(v)}}}
}

// W / H / MinWidth / MaxWidth size a box (W=width, H=height).
func W(v Length) Rule         { return decl("width", string(v)) }
func H(v Length) Rule         { return decl("height", string(v)) }
func MinWidth(v Length) Rule  { return decl("min-width", string(v)) }
func MaxWidth(v Length) Rule  { return decl("max-width", string(v)) }
func MinHeight(v Length) Rule { return decl("min-height", string(v)) }
func MaxHeight(v Length) Rule { return decl("max-height", string(v)) }

// --- color & type -------------------------------------------------------------

// Bg sets background-color.
func Bg(c Color) Rule { return decl("background-color", string(c)) }

// TextColor sets the text color.
func TextColor(c Color) Rule { return decl("color", string(c)) }

// FontSize sets font-size.
func FontSize(v Length) Rule { return decl("font-size", string(v)) }

type fontWeightProp struct {
	Normal   Rule
	Medium   Rule
	Semibold Rule
	Bold     Rule
}

// FontWeight is the typed namespace for font-weight.
var FontWeight = fontWeightProp{
	Normal:   decl("font-weight", "400"),
	Medium:   decl("font-weight", "500"),
	Semibold: decl("font-weight", "600"),
	Bold:     decl("font-weight", "700"),
}

// --- border & effects ---------------------------------------------------------

// Rounded sets border-radius.
func Rounded(v Length) Rule { return decl("border-radius", string(v)) }

// Border sets a solid border of the given width and color.
func Border(width Length, c Color) Rule {
	return decl("border", string(width)+" solid "+string(c))
}

// Opacity sets the opacity (0..1).
//
// Deprecated: prefer the typed OpacityNum(Num(n)) — it matches the typed-value convention used by
// the rest of the prop constructors. Opacity(float64) remains for convenience.
func Opacity(n float64) Rule { return decl("opacity", trimFloat(n)) }

// Property is the legacy string escape hatch.
//
// Deprecated: use Raw instead — it is the single, intentionally-named declaration
// escape hatch so raw usage stays greppable / lint-gateable.
func Property(parseName, parseValue string) Rule {
	deprecation.Warn("css.Property", "css.Raw")
	return decl(parseName, parseValue)
}
