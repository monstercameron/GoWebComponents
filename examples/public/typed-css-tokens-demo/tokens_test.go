package typedcsstokensdemo

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
	"github.com/monstercameron/GoWebComponents/css/u"
)

// TestTypedTokensResolveAgainstTheme proves the gwc-css-gen'd constants resolve against the
// custom theme — u.BgC(ColorBrand500) wires to the brand-500 color — so a typo'd token is a
// compile error (the constant would not exist), not a silent transparent fallback.
func TestTypedTokensResolveAgainstTheme(t *testing.T) {
	css.UseTheme(BrandTheme)

	want, ok := css.ColorValue(string(ColorBrand500))
	if !ok {
		t.Fatalf("ColorBrand500 (%q) not registered in the theme", ColorBrand500)
	}
	if want != css.Color("#6366f1") {
		t.Fatalf("ColorBrand500 resolved to %q, want #6366f1", want)
	}

	// The generated constants carry the css/u token types, so they are accepted by the
	// typed utility accessors without conversion.
	_ = u.BgC(ColorBrand500)
	_ = u.TextC(ColorBrand50)
	_ = u.Rounded(RadiusPill)
	_ = u.Pad(Spacing7)
	if rules := PrimaryButton(); len(rules) != 4 {
		t.Fatalf("PrimaryButton() emitted %d rules, want 4", len(rules))
	}
}
