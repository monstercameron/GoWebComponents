// Package u is the Tailwind-shaped typed utility engine layered over the css
// Layer-1 foundation. Every utility is a typed Go symbol (no class-string parser),
// so the whole surface is autocompletable and compile-checked. Each utility
// resolves its argument against the active css.Theme and returns Layer-1
// css.Rule(s); emission, dedup, and SSR reuse the Layer-1 registry/sink unchanged.
//
// This is the curated core (spacing/color/type/flex/border/effects + the
// responsive/state variants). The design's full-parity surface is generated from
// a spec table over the theme; the core lands first and the long tail is
// incremental table entries.
package u

import "github.com/monstercameron/GoWebComponents/v4/css"

// --- display & layout (static utilities) --------------------------------------

var (
	Block       = css.Display.Block
	InlineBlock = css.Display.InlineBlock
	Inline      = css.Display.Inline
	Flex        = css.Display.Flex
	InlineFlex  = css.Display.InlineFlex
	Grid        = css.Display.Grid
	Hide        = css.Display.None

	FlexRow = css.FlexDir.Row
	FlexCol = css.FlexDir.Col

	ItemsStart   = css.Items.Start
	ItemsCenter  = css.Items.Center
	ItemsEnd     = css.Items.End
	ItemsStretch = css.Items.Stretch

	JustifyStart   = css.Justify.Start
	JustifyCenter  = css.Justify.Center
	JustifyEnd     = css.Justify.End
	JustifyBetween = css.Justify.Between
	JustifyAround  = css.Justify.Around

	Relative = css.Position.Relative
	Absolute = css.Position.Absolute
	Fixed    = css.Position.Fixed
	Sticky   = css.Position.Sticky

	FontNormal   = css.FontWeight.Normal
	FontMedium   = css.FontWeight.Medium
	FontSemibold = css.FontWeight.Semibold
	FontBold     = css.FontWeight.Bold
)

// --- spacing (theme spacing scale) --------------------------------------------

// P/Px/Py and M/Mx/My take a typed spacing-scale index resolved against the
// theme: u.P(u.Spacing3) -> padding: 0.75rem. Negative margins use the *N helpers.

func Pad(index Spacing) css.Rule  { return css.Padding(css.SpacingValue(int(index))) }
func PadX(index Spacing) css.Rule { return css.PaddingX(css.SpacingValue(int(index))) }
func PadY(index Spacing) css.Rule { return css.PaddingY(css.SpacingValue(int(index))) }

func M(index Spacing) css.Rule  { return css.Margin(css.SpacingValue(int(index))) }
func Mx(index Spacing) css.Rule { return css.MarginX(css.SpacingValue(int(index))) }
func My(index Spacing) css.Rule { return css.MarginY(css.SpacingValue(int(index))) }

// MtN is a negative top margin (-mt-n): MtN(u.Spacing4) -> margin-top: -1rem.
func MtN(index Spacing) css.Rule {
	return css.Raw("margin-top", "-"+string(css.SpacingValue(int(index))))
}

// Gap takes a typed spacing-scale index; GapV takes an arbitrary Layer-1 length.
func Gap(index Spacing) css.Rule { return css.Gap(css.SpacingValue(int(index))) }
func GapV(v css.Length) css.Rule { return css.Gap(v) }

// W/H take a typed spacing-scale index; WFull/HFull are 100%.
func W(index Spacing) css.Rule { return css.W(css.SpacingValue(int(index))) }
func H(index Spacing) css.Rule { return css.H(css.SpacingValue(int(index))) }

var (
	WFull = css.W(css.Full)
	HFull = css.H(css.Full)
	WAuto = css.W(css.Auto)
)

// --- color (theme color scale) ------------------------------------------------

// Color tokens re-exported for ergonomic u.Slate900 style call sites.
const (
	White    = css.White
	Black    = css.Black
	Slate100 = css.Slate100
	Slate500 = css.Slate500
	Slate800 = css.Slate800
	Slate900 = css.Slate900
	Sky500   = css.Sky500
	Red500   = css.Red500
)

// Bg / Text set background/text color from a Layer-1 color token.
func Bg(c css.Color) css.Rule { return css.Bg(c) }
func Fg(c css.Color) css.Rule { return css.TextColor(c) }

// BgC / TextC / BorderC set background / text / border color from a typed theme
// color token: u.BgC(u.ColorSlate900). A typo is a compile error (the constant
// does not exist) and the token set autocompletes — the canonical, type-safe path.
// For a custom theme, `gwc css gen` generates the matching u.ColorToken constants.
func BgC(c ColorToken) css.Rule     { return css.Bg(resolveColor(c)) }
func TextC(c ColorToken) css.Rule   { return css.TextColor(resolveColor(c)) }
func BorderC(c ColorToken) css.Rule { return css.Border(css.Px(1), resolveColor(c)) }

// BgToken / TextToken resolve a theme color token name ("slate-900"); an unknown
// token resolves to transparent so a typo fails visibly rather than silently
// inheriting.
//
// Deprecated: prefer the typed u.BgC(u.ColorSlate900) / u.TextC(...), where a typo
// is a compile error instead of a runtime transparent fallback. These string forms
// remain for arbitrary/runtime-computed token names.
func BgToken(name string) css.Rule {
	c, ok := css.ColorValue(name)
	if !ok {
		c = css.Transparent
	}
	return css.Bg(c)
}

// Deprecated: prefer the typed u.TextC(u.ColorSlate900). See BgToken.
func TextToken(name string) css.Rule {
	c, ok := css.ColorValue(name)
	if !ok {
		c = css.Transparent
	}
	return css.TextColor(c)
}

// --- type scale ---------------------------------------------------------------

// TextSize resolves a typed type-scale key: u.TextSize(u.TextLg).
func TextSize(scale TextScale) css.Rule { return css.FontSize(resolveText(scale)) }

// --- border & effects ---------------------------------------------------------

// Rounded resolves a typed radius key: u.Rounded(u.RadiusLg).
func Rounded(r Radius) css.Rule { return css.Rounded(resolveRadius(r)) }

// Border sets a 1px solid border in the given color.
func Border(c css.Color) css.Rule { return css.Border(css.Px(1), c) }

// Opacity sets opacity from a 0..100 percentage index (Tailwind's opacity-50).
func Opacity(pct int) css.Rule { return css.OpacityNum(css.Num(float64(pct) / 100)) }
