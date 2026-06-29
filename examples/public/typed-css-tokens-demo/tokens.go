// Package typedcsstokensdemo shows `gwc css gen` against a real theme: theme.json is the
// source of truth, `gwc css gen -theme theme.json` writes the typed token constants in
// css_tokens_gen.go, and a typo'd token reference is then a compile error instead of a
// silent transparent fallback. `gwc css check` (in CI) fails if css_tokens_gen.go drifts
// from theme.json.
//
// Built-in css.DefaultTheme tokens already ship typed as u.ColorSlate900 etc.; this demo
// covers a *custom* theme, where the generator produces the matching u.ColorToken set.
package typedcsstokensdemo

import (
	"github.com/monstercameron/GoWebComponents/v4/css"
	"github.com/monstercameron/GoWebComponents/v4/css/u"
)

// BrandTheme is the custom theme these tokens are generated from. Activate it before
// emitting utilities so the typed tokens resolve to these colors.
var BrandTheme = css.Theme{
	Colors: map[string]css.Color{
		"brand-50":  css.Color("#eef2ff"),
		"brand-500": css.Color("#6366f1"),
		"brand-900": css.Color("#312e81"),
		"ink":       css.Color("#0f172a"),
	},
	FontSizes: map[string]css.Length{"display": css.Rem(3)},
	Radii:     map[string]css.Length{"pill": css.Length("9999px")},
	Spacing:   map[int]css.Length{7: css.Rem(1.75), 9: css.Rem(2.25)},
}

// PrimaryButton composes background + radius from the generated typed tokens. A typo —
// say u.BgC(ColorBrand5000) — would not compile, because that constant does not exist.
func PrimaryButton() []css.Rule {
	return []css.Rule{
		u.BgC(ColorBrand500),
		u.TextC(ColorBrand50),
		u.Rounded(RadiusPill),
		u.Pad(Spacing7),
	}
}
