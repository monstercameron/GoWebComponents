//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
	"github.com/monstercameron/GoWebComponents/css/u"
)

func TestTypedValueFormatting(t *testing.T) {
	cases := []struct{ got, want string }{
		{string(css.Ms(120)), "120ms"},
		{string(css.S(0.2)), "0.2s"},
		{string(css.Deg(45)), "45deg"},
		{string(css.Turn(0.5)), "0.5turn"},
		{string(css.Num(1.5)), "1.5"},
		{string(css.Num(1)), "1"},
		{string(css.Ems(0.18)), "0.18em"},
		{string(css.Vh(50)), "50vh"},
	}
	for _, c := range cases {
		if c.got != c.want {
			t.Errorf("value mismatch: got %q want %q", c.got, c.want)
		}
	}
}

func TestTypedPropertyConstructors(t *testing.T) {
	css.Reset()
	class := css.New(
		css.Cursor.Pointer,
		css.UserSelect.None,
		css.Transition(css.PropAll, css.Ms(120), css.Ease),
		css.Transform(css.Scale(0.94)),
		css.Shadow(css.ShadowLg),
		css.Tracking(css.Ems(0.18)),
		css.LineHeight(css.Num(1)),
		css.TextTransform.Uppercase,
		css.FontVariantNumeric.TabularNums,
		css.Outline(css.Px(2), css.Sky400),
		css.OutlineOffset(css.Px(2)),
	)
	got := css.Harvest()
	for _, want := range []string{
		"cursor:pointer", "user-select:none",
		"transition:all 120ms ease", "transform:scale(0.94)",
		"box-shadow:0 10px 15px -3px rgba(0,0,0,0.1)",
		"letter-spacing:0.18em", "line-height:1",
		"text-transform:uppercase", "font-variant-numeric:tabular-nums",
		"outline:2px solid #38bdf8", "outline-offset:2px",
	} {
		if !strings.Contains(got, want) {
			t.Errorf("missing %q in:\n%s", want, got)
		}
	}
	_ = class
}

func TestTransformComposesMultipleFns(t *testing.T) {
	css.Reset()
	css.New(css.Transform(css.Scale(2), css.Rotate(css.Deg(45)), css.TranslateX(css.Px(8))))
	got := css.Harvest()
	if !strings.Contains(got, "transform:scale(2) rotate(45deg) translateX(8px)") {
		t.Fatalf("transform composition wrong:\n%s", got)
	}
}

func TestTypedScaleConstantsResolve(t *testing.T) {
	css.Reset()
	class := css.New(
		u.Rounded(u.RadiusLg), // theme radii lg -> 0.5rem
		u.TextSize(u.TextXl),  // theme font-sizes xl -> 1.25rem
		u.Pad(u.Spacing4),     // spacing 4 -> 1rem
	)
	got := css.Harvest()
	for _, want := range []string{"border-radius:0.5rem", "font-size:1.25rem", "padding:1rem"} {
		if !strings.Contains(got, want) {
			t.Errorf("typed scale constant did not resolve: missing %q in\n%s", want, got)
		}
	}
	_ = class
}

func TestRawEscapeHatch(t *testing.T) {
	css.Reset()
	class := css.New(css.Raw("backdrop-filter", "blur(4px)"))
	if !strings.Contains(css.Harvest(), "."+string(class)+"{backdrop-filter:blur(4px);}") {
		t.Fatalf("Raw escape hatch did not emit: %q", css.Harvest())
	}
}

func TestCubicBezierEasing(t *testing.T) {
	css.Reset()
	css.New(css.Transition(css.PropOpacity, css.S(0.3), css.CubicBezier(0.4, 0, 0.2, 1)))
	if !strings.Contains(css.Harvest(), "transition:opacity 0.3s cubic-bezier(0.4,0,0.2,1)") {
		t.Fatalf("cubic-bezier wrong:\n%s", css.Harvest())
	}
}
