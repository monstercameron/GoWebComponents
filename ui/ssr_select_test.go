//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

// Ported from React's ReactDOMServerIntegrationSelect: a controlled
// <select value="x"> renders the matching <option> with the selected attribute
// and does NOT put a (browser-ignored) value attribute on the select itself.
// Regression test for the v3.4.10 SSR fix.
func TestSSRSelectMarksSelectedOption(t *testing.T) {
	opt := func(parseV, parseTxt string) ui.Node {
		return html.Option(html.Props{Value: parseV}, ui.Text(parseTxt))
	}

	cases := []struct {
		name string
		node ui.Node
		want string
	}{
		{
			"value-matches-option",
			html.Select(html.Props{Value: "b"}, opt("a", "A"), opt("b", "B"), opt("c", "C")),
			`<select><option value="a">A</option><option selected value="b">B</option><option value="c">C</option></select>`,
		},
		{
			"no-match-no-selection",
			html.Select(html.Props{Value: "zzz"}, opt("a", "A"), opt("b", "B")),
			`<select><option value="a">A</option><option value="b">B</option></select>`,
		},
		{
			"match-by-option-text-when-no-value",
			html.Select(html.Props{Value: "Two"}, html.Option(html.Props{}, ui.Text("One")), html.Option(html.Props{}, ui.Text("Two"))),
			`<select><option>One</option><option selected>Two</option></select>`,
		},
		{
			"no-controlled-value-unchanged",
			html.Select(html.Props{}, opt("a", "A")),
			`<select><option value="a">A</option></select>`,
		},
	}
	for _, parseCase := range cases {
		out, err := ui.RenderToString(parseCase.node)
		if err != nil {
			t.Fatalf("%s: render error: %v", parseCase.name, err)
		}
		if out != parseCase.want {
			t.Errorf("%s:\n got %q\nwant %q", parseCase.name, out, parseCase.want)
		}
	}
}

// TestSSRSelectMatchesOptionInsideOptgroup confirms an option nested in an
// <optgroup> is still matched by the controlled select value.
func TestSSRSelectMatchesOptionInsideOptgroup(t *testing.T) {
	parseGroup := html.Tag("optgroup", html.Props{Raw: map[string]any{"label": "G"}},
		html.Option(html.Props{Value: "x"}, ui.Text("X")),
		html.Option(html.Props{Value: "y"}, ui.Text("Y")))
	out, err := ui.RenderToString(html.Select(html.Props{Value: "y"}, parseGroup))
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	const want = `<select><optgroup label="G"><option value="x">X</option><option selected value="y">Y</option></optgroup></select>`
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}

// TestSSRSelectRespectsExplicitOptionSelected: if an option already declares
// selected, it is left as-is (not duplicated) even when the value also matches.
func TestSSRSelectRespectsExplicitOptionSelected(t *testing.T) {
	out, err := ui.RenderToString(html.Select(html.Props{Value: "a"},
		html.Option(html.Props{Value: "a", Raw: map[string]any{"selected": true}}, ui.Text("A"))))
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	const want = `<select><option selected value="a">A</option></select>`
	if out != want {
		t.Errorf("got %q, want %q", out, want)
	}
}
