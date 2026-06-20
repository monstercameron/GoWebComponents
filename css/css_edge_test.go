//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/css"
	"github.com/monstercameron/GoWebComponents/css/u"
)

func TestUnknownTypedScaleFallsBack(t *testing.T) {
	css.Reset()
	// A key fabricated via conversion (not a built-in const) falls back sensibly
	// rather than emitting an empty value.
	css.New(u.Rounded(u.Radius("bogus")))
	if !strings.Contains(css.Harvest(), "border-radius:0.375rem") { // md
		t.Fatalf("unknown radius did not fall back to md: %q", css.Harvest())
	}
	css.Reset()
	css.New(u.TextSize(u.TextScale("nope")))
	if !strings.Contains(css.Harvest(), "font-size:1rem") { // base fallback
		t.Fatalf("unknown text scale did not fall back: %q", css.Harvest())
	}
}

func TestArbitrarySpacingIndexUsesFormula(t *testing.T) {
	css.Reset()
	// Index 7 isn't in the curated map; Tailwind's n*0.25rem formula applies.
	css.New(u.Pad(u.Spacing(7)))
	if !strings.Contains(css.Harvest(), "padding:1.75rem") {
		t.Fatalf("arbitrary spacing index wrong: %q", css.Harvest())
	}
}

func TestEmptyCombinatorEmitsNothing(t *testing.T) {
	css.Reset()
	// A combinator with no declarations contributes no block.
	class := css.New(css.Child(css.El("a"))...)
	if class != "" || css.Harvest() != "" {
		t.Fatalf("empty combinator should emit nothing, got class=%q css=%q", class, css.Harvest())
	}
}

func TestDeepVariantNesting(t *testing.T) {
	css.Reset()
	// Media > Not > Hover > Child — four layers fold into one selector.
	class := css.New(css.Media(css.MinW(640),
		css.Not(css.ClassSel("ghost"),
			css.Hover(css.Child(css.El("a"), css.TextColor(css.Sky500))...)...,
		)...,
	)...)
	got := css.Harvest()
	want := "@media (min-width:640px){." + string(class) + ":not(.ghost):hover > a{color:#0ea5e9;}}"
	if got != want {
		t.Fatalf("deep nesting wrong\nwant: %s\ngot:  %s", want, got)
	}
}

func TestIdentityCacheSingleEmit(t *testing.T) {
	css.Reset()
	rules := func() []css.Rule { return css.Rules(css.Display.Flex, css.Gap(css.Px(8))) }
	first := css.New(rules()...)
	for i := 0; i < 100; i++ {
		if css.New(rules()...) != first {
			t.Fatal("cache returned a different class")
		}
	}
	if n := strings.Count(css.Harvest(), "{"); n != 1 {
		t.Fatalf("identity cache failed: expected 1 block, got %d:\n%s", n, css.Harvest())
	}
}

func TestResetClearsCache(t *testing.T) {
	css.Reset()
	css.New(css.Display.Flex)
	if css.Harvest() == "" {
		t.Fatal("expected emission before reset")
	}
	css.Reset()
	// After reset the cache + registry are empty, so the same rule re-emits.
	css.New(css.Display.Flex)
	if css.Harvest() == "" {
		t.Fatal("expected re-emission after reset (cache not cleared)")
	}
}

func TestHasNotAcceptRefTargets(t *testing.T) {
	css.Reset()
	badge := css.New(css.Display.Inline)
	// :has(.badge) targeting another generated class — fully typed.
	class := css.New(css.Has(css.Ref(badge), css.Padding(css.Px(4)))...)
	want := "." + string(class) + ":has(." + string(badge) + "){padding:4px;}"
	if !strings.Contains(css.Harvest(), want) {
		t.Fatalf("Has(Ref) missing %q in:\n%s", want, css.Harvest())
	}
}

func TestStyleBreakoutIsNeutralized(t *testing.T) {
	css.Reset()
	// Even through the Raw escape hatch, no input may terminate the <style>
	// element or a CSS comment in emitted output — the hardening boundary.
	css.New(css.Raw("content", `"</style><script>alert(1)</script>"`))
	css.New(css.Raw("x", "y/**/"))  // comment-close attempt
	css.New(css.Raw("z", "a\x00b")) // NUL
	body := css.Harvest()
	for _, bad := range []string{"</style", "<script", "*/", "\x00"} {
		if strings.Contains(body, bad) {
			t.Fatalf("breakout sequence %q survived hardening:\n%s", bad, body)
		}
	}
	// The full SSR block's only </style> is its own closing tag.
	block := css.StyleBlock()
	if n := strings.Count(block, "</style>"); n != 1 {
		t.Fatalf("StyleBlock should contain exactly one (closing) </style>, got %d:\n%s", n, block)
	}
	// Typed constructors are safe-by-construction too.
	css.Reset()
	css.New(css.Bg(css.Hex(`</style><script>`)), css.TextColor(css.Var("--x) }*{")))
	tb := css.Harvest()
	for _, bad := range []string{"</style", "<script", "}*{"} {
		if strings.Contains(tb, bad) {
			t.Fatalf("typed constructor leaked %q:\n%s", bad, tb)
		}
	}
}

func TestHoistedVarFoldsOnceAcrossClasses(t *testing.T) {
	css.Reset()
	// Two New(...) calls share a base bundle; the shared declarations must dedup
	// into a single emitted block per resulting class (no duplicate base block).
	base := css.Rules(css.Display.Flex, css.Padding(css.Px(8)))
	a := css.New(css.Rules(base, css.Bg(css.Sky500))...)
	b := css.New(css.Rules(base, css.Bg(css.Slate700))...)
	if a == b {
		t.Fatal("distinct backgrounds should produce distinct classes")
	}
	// Each class emits exactly one block (base merged with its own bg).
	if n := strings.Count(css.Harvest(), "{"); n != 2 {
		t.Fatalf("expected 2 blocks total, got %d:\n%s", n, css.Harvest())
	}
}
