//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Ported from React's ReactDOMServerIntegrationTextarea: a controlled textarea's
// value is serialized as its (escaped) text content, NOT as a value attribute —
// HTML ignores a value attribute on <textarea>, so the attribute form renders an
// empty field server-side and mismatches on hydration. Regression test for the
// v3.4.9 SSR fix.
func TestSSRTextareaValueRendersAsContent(t *testing.T) {
	cases := []struct {
		name string
		node ui.Node
		want string
	}{
		{"value-as-content", html.Textarea(html.Props{Value: "body"}), "<textarea>body</textarea>"},
		{"value-escaped", html.Textarea(html.Props{Value: `a & <b>`}), "<textarea>a &amp; &lt;b&gt;</textarea>"},
		{"value-with-other-attrs", html.Textarea(html.Props{ID: "t", Value: "v"}), `<textarea id="t">v</textarea>`},
		{"children-when-no-value", html.Textarea(html.Props{}, ui.Text("child")), "<textarea>child</textarea>"},
		{"empty", html.Textarea(html.Props{}), "<textarea></textarea>"},
	}
	for _, parseCase := range cases {
		out, err := ui.RenderToString(parseCase.node)
		if err != nil {
			t.Fatalf("%s: render error: %v", parseCase.name, err)
		}
		if out != parseCase.want {
			t.Errorf("%s: got %q, want %q", parseCase.name, out, parseCase.want)
		}
	}
}

// Ported from ReactDOMServerIntegrationInput / Checkbox: an input's value is a
// real attribute (unlike textarea), and a boolean `checked` renders as a bare
// attribute when true and is omitted when false.
func TestSSRInputAndCheckbox(t *testing.T) {
	cases := []struct {
		name string
		node ui.Node
		want string
	}{
		{"input-value", html.Input(html.Props{Type: "text", Value: "hello"}), `<input type="text" value="hello">`},
		{"checkbox-checked", html.Input(html.Props{Type: "checkbox", Raw: map[string]any{"checked": true}}), `<input checked type="checkbox">`},
		{"checkbox-unchecked", html.Input(html.Props{Type: "checkbox", Raw: map[string]any{"checked": false}}), `<input type="checkbox">`},
	}
	for _, parseCase := range cases {
		out, err := ui.RenderToString(parseCase.node)
		if err != nil {
			t.Fatalf("%s: render error: %v", parseCase.name, err)
		}
		if out != parseCase.want {
			t.Errorf("%s: got %q, want %q", parseCase.name, out, parseCase.want)
		}
	}
}

// TestSSROptionSelected: an <option> with selected=true renders the bare selected
// attribute (the controlled form GWC supports for SSR select state).
func TestSSROptionSelected(t *testing.T) {
	out, err := ui.RenderToString(html.Option(html.Props{Value: "a", Raw: map[string]any{"selected": true}}, ui.Text("A")))
	if err != nil {
		t.Fatalf("render error: %v", err)
	}
	if out != `<option selected value="a">A</option>` {
		t.Errorf("got %q, want <option selected value=\"a\">A</option>", out)
	}
}
