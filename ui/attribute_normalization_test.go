//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/html"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

func rawDiv(parseRaw map[string]any) string {
	parseOut, _ := ui.RenderToString(html.Tag("div", html.Props{Raw: parseRaw}, ui.Text("x")))
	return parseOut
}

// Deepens the ReactDOMServerIntegrationAttributes port toward equivalency: the
// React-prop -> HTML-attribute name normalizations and the props that must never
// be emitted as attributes.
func TestAttributeNameNormalization(t *testing.T) {
	if parseGot := rawDiv(map[string]any{"className": "c1"}); parseGot != `<div class="c1">x</div>` {
		t.Errorf("className->class: %q", parseGot)
	}
	parseLabel, _ := ui.RenderToString(html.Tag("label", html.Props{Raw: map[string]any{"htmlFor": "id1"}}, ui.Text("x")))
	if parseLabel != `<label for="id1">x</label>` {
		t.Errorf("htmlFor->for: %q", parseLabel)
	}
}

// Internal/framework props are never serialized as attributes (React: ref, key,
// children, dangerouslySetInnerHTML, suppress*Warning; GWC: key, children, event
// handlers, the DOM-ref sink).
func TestNonAttributePropsAreNotEmitted(t *testing.T) {
	cases := []struct {
		name string
		raw  map[string]any
	}{
		{"key", map[string]any{"key": "k", "id": "real"}},
		{"children", map[string]any{"children": "nope", "id": "real"}},
		{"onClick", map[string]any{"onClick": func() {}, "id": "real"}},
		{"onclick-lower", map[string]any{"onclick": func() {}, "id": "real"}},
	}
	for _, parseCase := range cases {
		if parseGot := rawDiv(parseCase.raw); parseGot != `<div id="real">x</div>` {
			t.Errorf("%s should be omitted: %q", parseCase.name, parseGot)
		}
	}
}

// Style maps serialize to a deterministic, sorted "k:v;k:v" string (React sorts
// for stable SSR output).
func TestStyleMapSerialization(t *testing.T) {
	parseOut, _ := ui.RenderToString(html.Div(html.Props{
		Style: map[string]string{"margin": "0", "color": "red", "z-index": "2"},
	}, ui.Text("x")))
	if parseOut != `<div style="color:red;margin:0;z-index:2">x</div>` {
		t.Errorf("style map = %q, want sorted color;margin;z-index", parseOut)
	}
}

// Numeric and boolean attribute values (via Raw): a number stringifies, a true
// boolean is a bare attribute, a false boolean is omitted.
func TestNumericAndBooleanAttributeValues(t *testing.T) {
	if parseGot := rawDiv(map[string]any{"tabindex": 0}); parseGot != `<div tabindex="0">x</div>` {
		t.Errorf("numeric zero: %q", parseGot)
	}
	if parseGot := rawDiv(map[string]any{"data-n": 42}); parseGot != `<div data-n="42">x</div>` {
		t.Errorf("numeric: %q", parseGot)
	}
	if parseGot := rawDiv(map[string]any{"hidden": true}); parseGot != `<div hidden>x</div>` {
		t.Errorf("bool true: %q", parseGot)
	}
	if parseGot := rawDiv(map[string]any{"hidden": false}); parseGot != `<div>x</div>` {
		t.Errorf("bool false should be omitted: %q", parseGot)
	}
}
