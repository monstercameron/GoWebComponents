//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
	"github.com/monstercameron/GoWebComponents/css/u"
	"github.com/monstercameron/GoWebComponents/html/shorthand"
	"github.com/monstercameron/GoWebComponents/ui"
)

// These integration tests exercise the real crossing into html/shorthand + the
// ui SSR renderer: css.Class must flow through the existing PropOption path, and
// the buffer sink must harvest into an SSR-ready <style> block.

func TestClassFlowsThroughShorthandRender(t *testing.T) {
	css.Reset()
	node := shorthand.Div(
		css.Class(css.Display.Flex, css.Gap(css.Px(8))),
		shorthand.Span(css.Class(css.TextColor(css.Slate900)), "hi"),
	)
	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The folded class must appear on the element's class attribute.
	flexClass := css.New(css.Display.Flex, css.Gap(css.Px(8)))
	if !strings.Contains(markup, `class="`+string(flexClass)+`"`) {
		t.Fatalf("rendered markup missing css class %q:\n%s", flexClass, markup)
	}
}

func TestSheetFlowsThroughClassNames(t *testing.T) {
	css.Reset()
	sheet := css.New(css.Display.Grid)
	// A Sheet is a fmt.Stringer, so it composes in html.ClassNames alongside
	// literal strings.
	node := shorthand.Div(shorthand.FromProps(shorthand.Props{Class: "static " + string(sheet)}), "x")
	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	if !strings.Contains(markup, "static "+string(sheet)) {
		t.Fatalf("sheet did not compose with literal class:\n%s", markup)
	}
}

func TestStyleBlockIsSSRReady(t *testing.T) {
	css.Reset()
	class := css.New(css.Display.Flex)
	block := css.StyleBlock()
	if !strings.HasPrefix(block, `<style data-gwc-css="`) {
		t.Fatalf("style block missing data attr: %q", block)
	}
	if !strings.Contains(block, string(class)) {
		t.Fatalf("style block missing class %q: %q", class, block)
	}
	if !strings.Contains(block, "display:flex") {
		t.Fatalf("style block missing rule body: %q", block)
	}
	if !strings.HasSuffix(block, "</style>") {
		t.Fatalf("style block not closed: %q", block)
	}
}

func TestStyleBlockEmptyWhenNothingEmitted(t *testing.T) {
	css.Reset()
	if got := css.StyleBlock(); got != "" {
		t.Fatalf("expected empty style block, got %q", got)
	}
}

func TestUtilityLayerResolvesAgainstTheme(t *testing.T) {
	css.Reset()
	// u.Gap(u.Spacing3) -> theme spacing index 3 -> 0.75rem.
	class := css.New(u.Flex, u.Gap(u.Spacing3))
	got := css.Harvest()
	want := "." + string(class) + "{display:flex;gap:0.75rem;}"
	if got != want {
		t.Fatalf("utility resolution mismatch\nwant: %s\ngot:  %s", want, got)
	}
}

func TestUtilityVariantsCompose(t *testing.T) {
	css.Reset()
	// u.Md(u.Hover(u.Flex)) == md:hover:flex.
	class := css.New(u.Md(u.Hover(u.Flex)...)...)
	got := css.Harvest()
	want := "@media (min-width:768px){." + string(class) + ":hover{display:flex;}}"
	if got != want {
		t.Fatalf("utility variant composition mismatch\nwant: %s\ngot:  %s", want, got)
	}
}

func TestUseThemeSwapsScales(t *testing.T) {
	css.Reset()
	base := css.DefaultTheme()
	base.Spacing = map[int]css.Length{3: css.Px(99)}
	css.UseTheme(base)
	defer css.UseTheme(css.DefaultTheme())

	class := css.New(u.Gap(u.Spacing3))
	if !strings.Contains(css.Harvest(), "."+string(class)+"{gap:99px;}") {
		t.Fatalf("theme swap not honored: %q", css.Harvest())
	}
}

func TestDefineUtilityComposes(t *testing.T) {
	css.Reset()
	card := css.DefineUtility("card", css.Padding(css.Px(16)), css.Rounded(css.Px(8)))
	class := css.New(card...)
	got := css.Harvest()
	if !strings.Contains(got, "padding:16px") || !strings.Contains(got, "border-radius:8px") {
		t.Fatalf("DefineUtility bundle not emitted: %q", got)
	}
	if again, ok := css.Utility("card"); !ok || len(again) != 2 {
		t.Fatalf("Utility lookup failed: %v ok=%v", again, ok)
	}
	_ = class
}

func TestSelectorCompositionThroughRender(t *testing.T) {
	css.Reset()
	// A card that styles its own child <span> via typed selector composition.
	card := css.Class(
		css.Display.Flex,
		css.Child(css.El("span"), css.TextColor(css.Sky500), css.FontWeight.Bold),
	)
	node := shorthand.Div(card, shorthand.Span("label"))
	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	cardClass := css.New(css.Rules(
		css.Display.Flex,
		css.Child(css.El("span"), css.TextColor(css.Sky500), css.FontWeight.Bold),
	)...)
	if !strings.Contains(markup, `class="`+string(cardClass)+`"`) {
		t.Fatalf("card class not on element:\n%s", markup)
	}
	// The child rule must be in the harvested sheet as a "> span" selector.
	if !strings.Contains(css.Harvest(), "."+string(cardClass)+" > span{color:#0ea5e9;font-weight:700;}") {
		t.Fatalf("child-combinator rule missing from sheet:\n%s", css.Harvest())
	}
}

func TestRefCrossClassThroughStyleBlock(t *testing.T) {
	css.Reset()
	title := css.New(css.FontWeight.Bold)
	css.New(css.Descendant(css.SheetRef(title), css.TextColor(css.Sky500))...)
	block := css.StyleBlock()
	if !strings.Contains(block, " ."+string(title)+"{color:#0ea5e9;}") {
		t.Fatalf("Ref cross-class rule missing from SSR style block:\n%s", block)
	}
	// Both classes are listed in the data attribute for hydration seeding.
	if !strings.Contains(block, string(title)) {
		t.Fatalf("style block data attr missing title class:\n%s", block)
	}
}

func TestDynamicPairsClassAndStyleVar(t *testing.T) {
	css.Reset()
	d := css.DynamicLength("--gap", "gap", css.Px(13))
	node := shorthand.Div(d.Class(), d.Style(), "x")
	markup, err := ui.RenderToString(node)
	if err != nil {
		t.Fatalf("render: %v", err)
	}
	// The class references the var; the inline style carries the live value.
	if !strings.Contains(css.Harvest(), "gap:var(--gap)") {
		t.Fatalf("dynamic class missing var reference: %q", css.Harvest())
	}
	if !strings.Contains(markup, "--gap:13px") {
		t.Fatalf("dynamic inline style missing live value:\n%s", markup)
	}
}
