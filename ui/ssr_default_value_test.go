//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/html"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Ported from React's ReactDOMServerIntegrationInput defaultValue / defaultChecked
// behavior: an uncontrolled form field's defaultValue is serialized as the
// element's controlled value (and defaultChecked as checked) so it renders with
// its initial value server-side instead of an inert, browser-ignored
// defaultValue/defaultChecked attribute. Regression test for the v3.5.2 fix.
func TestSSRDefaultValueMapsToControlledValue(t *testing.T) {
	cases := []struct {
		name string
		node ui.Node
		want string
	}{
		{
			"input-defaultValue",
			html.Input(html.Props{Type: "text", Raw: map[string]any{"defaultValue": "dv"}}),
			`<input type="text" value="dv">`,
		},
		{
			"value-overrides-defaultValue",
			html.Input(html.Props{Type: "text", Value: "v", Raw: map[string]any{"defaultValue": "dv"}}),
			`<input type="text" value="v">`,
		},
		{
			"checkbox-defaultChecked",
			html.Input(html.Props{Type: "checkbox", Raw: map[string]any{"defaultChecked": true}}),
			`<input checked type="checkbox">`,
		},
		{
			"checkbox-defaultChecked-false",
			html.Input(html.Props{Type: "checkbox", Raw: map[string]any{"defaultChecked": false}}),
			`<input type="checkbox">`,
		},
		{
			"textarea-defaultValue-as-content",
			html.Textarea(html.Props{Raw: map[string]any{"defaultValue": "body"}}),
			`<textarea>body</textarea>`,
		},
		{
			"select-defaultValue-selects-option",
			html.Select(html.Props{Raw: map[string]any{"defaultValue": "b"}},
				html.Option(html.Props{Value: "a"}, ui.Text("A")),
				html.Option(html.Props{Value: "b"}, ui.Text("B"))),
			`<select><option value="a">A</option><option selected value="b">B</option></select>`,
		},
		{
			"no-default-unchanged",
			html.Input(html.Props{Type: "text", Value: "v"}),
			`<input type="text" value="v">`,
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
