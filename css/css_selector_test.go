//go:build !(js && wasm)

package css_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/css"
)

func TestCombinatorsEmitCorrectSelectors(t *testing.T) {
	cases := []struct {
		name   string
		rules  []css.Rule
		suffix string // selector suffix after the class
	}{
		{"child", css.Child(css.El("a"), css.Display.Block), " > a"},
		{"descendant", css.Descendant(css.El("span"), css.Display.Block), " span"},
		{"adjacent", css.Adjacent(css.El("p"), css.Display.Block), " + p"},
		{"sibling", css.Sibling(css.El("p"), css.Display.Block), " ~ p"},
		{"attr", css.Descendant(css.AttrSel("data-open"), css.Display.Block), " [data-open]"},
		{"attreq", css.Descendant(css.AttrEq("type", "submit"), css.Display.Block), ` [type="submit"]`},
		{"classsel", css.Descendant(css.ClassSel("title"), css.Display.Block), " .title"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			css.Reset()
			class := css.New(c.rules...)
			want := "." + string(class) + c.suffix + "{display:block;}"
			if got := css.Harvest(); got != want {
				t.Fatalf("want %q got %q", want, got)
			}
		})
	}
}

func TestRefComposesGeneratedClasses(t *testing.T) {
	css.Reset()
	title := css.New(css.FontWeight.Bold)
	// Parent styles its descendant .title (another generated class), type-safe.
	parent := css.New(css.Descendant(css.SheetRef(title), css.TextColor(css.Sky500))...)
	got := css.Harvest()
	want := "." + string(parent) + " ." + string(title) + "{color:#0ea5e9;}"
	if !strings.Contains(got, want) {
		t.Fatalf("Ref composition missing %q in:\n%s", want, got)
	}
}

func TestSubstitutionDirectionMatchesNestingOrder(t *testing.T) {
	// Hover(Descendant(...)) -> .c:hover h3   (parent hovered, style descendant)
	css.Reset()
	a := css.New(css.Hover(css.Descendant(css.El("h3"), css.TextColor(css.White))...)...)
	if got := css.Harvest(); !strings.Contains(got, "."+string(a)+":hover h3{color:#ffffff;}") {
		t.Fatalf("Hover(Descendant) wrong:\n%s", got)
	}

	// Descendant(Hover(...)) -> .c h3:hover   (the descendant itself hovered)
	css.Reset()
	b := css.New(css.Descendant(css.El("h3"), css.Hover(css.TextColor(css.White))...)...)
	if got := css.Harvest(); !strings.Contains(got, "."+string(b)+" h3:hover{color:#ffffff;}") {
		t.Fatalf("Descendant(Hover) wrong:\n%s", got)
	}
}

func TestFunctionalPseudoClasses(t *testing.T) {
	cases := []struct {
		name   string
		rules  []css.Rule
		needle string
	}{
		{"not", css.Not(css.ClassSel("ghost"), css.Bg(css.Slate900)), ":not(.ghost){background-color:#0f172a;}"},
		{"has", css.Has(css.El("img"), css.Padding(css.Px(0))), ":has(img){padding:0px;}"},
		{"is", css.Is([]css.Selector{css.El("h1"), css.El("h2")}, css.FontWeight.Bold), ":is(h1, h2){font-weight:700;}"},
		{"nth-odd", css.NthChild(css.Odd, css.Bg(css.Slate100)), ":nth-child(odd){background-color:#f1f5f9;}"},
		{"nth-anb", css.NthChild(css.AnB(3, 1), css.Display.Block), ":nth-child(3n+1){display:block;}"},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			css.Reset()
			class := css.New(c.rules...)
			if got := css.Harvest(); !strings.Contains(got, "."+string(class)+c.needle) {
				t.Fatalf("want suffix %q for class %q in:\n%s", c.needle, class, got)
			}
		})
	}
}

func TestAnBNegative(t *testing.T) {
	if got := string(css.AnB(2, -1)); got != "2n-1" {
		t.Fatalf("AnB(2,-1) = %q want 2n-1", got)
	}
	if got := string(css.AnB(3, 1)); got != "3n+1" {
		t.Fatalf("AnB(3,1) = %q want 3n+1", got)
	}
}

func TestCombinatorPlusVariantStacking(t *testing.T) {
	css.Reset()
	// Child(a) then Hover then Media — three layers compose into one selector.
	class := css.New(css.Media(css.MinW(768),
		css.Hover(css.Child(css.El("a"), css.TextColor(css.Sky500))...)...,
	)...)
	got := css.Harvest()
	want := "@media (min-width:768px){." + string(class) + ":hover > a{color:#0ea5e9;}}"
	if got != want {
		t.Fatalf("three-layer composition wrong\nwant: %s\ngot:  %s", want, got)
	}
}
