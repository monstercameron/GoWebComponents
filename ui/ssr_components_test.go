//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

func ssrRender(t *testing.T, parseEl *runtime.Element) string {
	t.Helper()
	parseOut, parseErr := ui.RenderToString(ui.Node(parseEl))
	if parseErr != nil {
		t.Fatalf("render error: %v", parseErr)
	}
	return parseOut
}

// Ported from React's ReactDOMServerIntegrationBasic / Elements: a function
// component renders to its returned tree under SSR, props flow in, components
// nest, and a component returning nil renders nothing.
func TestSSRRendersNestedComponents(t *testing.T) {
	parseInner := func(parseProps map[string]any) *runtime.Element {
		parseLabel, _ := parseProps["label"].(string)
		return runtime.CreateElement("span", map[string]any{}, parseLabel)
	}
	parseOuter := func() *runtime.Element {
		return runtime.CreateElement("div", map[string]any{"id": "app"},
			runtime.CreateElement(parseInner, map[string]any{"label": "hello"}),
			runtime.CreateElement(parseInner, map[string]any{"label": "world"}))
	}
	if parseGot := ssrRender(t, runtime.CreateElement(parseOuter, map[string]any{})); parseGot != `<div id="app"><span>hello</span><span>world</span></div>` {
		t.Errorf("nested components SSR = %q", parseGot)
	}
}

// A component returning nil renders nothing (no wrapper, no placeholder).
func TestSSRComponentReturningNilRendersNothing(t *testing.T) {
	parseHidden := func() *runtime.Element { return nil }
	parseTree := runtime.CreateElement("div", map[string]any{},
		runtime.CreateElement(parseHidden, map[string]any{}),
		runtime.CreateElement("span", map[string]any{}, "after"))
	if parseGot := ssrRender(t, parseTree); parseGot != `<div><span>after</span></div>` {
		t.Errorf("nil-component SSR = %q, want <div><span>after</span></div>", parseGot)
	}
}

// A component rendering a fragment hoists its children into the parent under SSR.
func TestSSRComponentRenderingFragment(t *testing.T) {
	parseListItems := func() *runtime.Element {
		return runtime.CreateElement("FRAGMENT", nil,
			runtime.CreateElement("li", map[string]any{}, "1"),
			runtime.CreateElement("li", map[string]any{}, "2"))
	}
	parseTree := runtime.CreateElement("ul", map[string]any{},
		runtime.CreateElement(parseListItems, map[string]any{}),
		runtime.CreateElement("li", map[string]any{}, "3"))
	if parseGot := ssrRender(t, parseTree); parseGot != `<ul><li>1</li><li>2</li><li>3</li></ul>` {
		t.Errorf("fragment-component SSR = %q", parseGot)
	}
}
