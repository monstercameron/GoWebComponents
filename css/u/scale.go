package u

import "github.com/monstercameron/GoWebComponents/css"

// Typed scale keys replace the former string theme keys, so a typo is a compile
// error (and autocompletes) instead of a silent runtime fallback. Each constant's
// underlying value is the theme scale key it resolves against.

// Radius is a typed border-radius scale key.
type Radius string

const (
	RadiusNone Radius = "none"
	RadiusSm   Radius = "sm"
	RadiusMd   Radius = "md"
	RadiusLg   Radius = "lg"
	RadiusXl   Radius = "xl"
	RadiusFull Radius = "full"
)

// TextScale is a typed font-size scale key.
type TextScale string

const (
	TextXs   TextScale = "xs"
	TextSm   TextScale = "sm"
	TextBase TextScale = "base"
	TextLg   TextScale = "lg"
	TextXl   TextScale = "xl"
	Text2xl  TextScale = "2xl"
	Text3xl  TextScale = "3xl"
	Text4xl  TextScale = "4xl"
)

// Spacing is a typed spacing-scale index. Built-in steps are provided as
// constants; any int is still valid (Tailwind's n*0.25rem formula), so arbitrary
// indices need no escape hatch.
type Spacing int

const (
	Spacing0  Spacing = 0
	Spacing1  Spacing = 1
	Spacing2  Spacing = 2
	Spacing3  Spacing = 3
	Spacing4  Spacing = 4
	Spacing5  Spacing = 5
	Spacing6  Spacing = 6
	Spacing8  Spacing = 8
	Spacing10 Spacing = 10
	Spacing12 Spacing = 12
	Spacing16 Spacing = 16
	Spacing20 Spacing = 20
	Spacing24 Spacing = 24
)

// resolveRadius / resolveText map a typed key to its theme value. Built-in
// constants always hit; a key fabricated via conversion falls back to the scale's
// sensible default.
func resolveRadius(r Radius) css.Length {
	if v, ok := css.RadiusValue(string(r)); ok {
		return v
	}
	v, _ := css.RadiusValue("md")
	return v
}

func resolveText(t TextScale) css.Length {
	if v, ok := css.FontSizeValue(string(t)); ok {
		return v
	}
	return css.Rem(1)
}
