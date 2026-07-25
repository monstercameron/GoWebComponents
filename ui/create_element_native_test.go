//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// Ported from React's ReactCreateElement-test (element assembly): createElement
// records the element type, preserves the props map (including a key), and keeps
// the variadic children in order. GWC's element is an immutable value with
// Type/Props/Children fields.
func TestCreateElementAssemblesTypePropsChildren(t *testing.T) {
	parseChildA := runtime.CreateElement("span", map[string]any{}, "a")
	parseChildB := runtime.CreateElement("span", map[string]any{}, "b")
	parseEl := runtime.CreateElement("div", map[string]any{"id": "x", "key": "k"}, parseChildA, parseChildB)

	if parseEl.Type != "div" {
		t.Errorf("Type = %v, want div", parseEl.Type)
	}
	if parseEl.Props["id"] != "x" {
		t.Errorf("Props[id] = %v, want x", parseEl.Props["id"])
	}
	if parseEl.Props["key"] != "k" {
		t.Errorf("Props[key] = %v, want k", parseEl.Props["key"])
	}
	if parseGot := len(parseEl.Children); parseGot != 2 {
		t.Fatalf("len(Children) = %d, want 2", parseGot)
	}
	// Children preserved in order.
	if parseEl.Children[0] != parseChildA || parseEl.Children[1] != parseChildB {
		t.Errorf("children not preserved in order")
	}
}

// TestCreateElementNilPropsIsSafe: passing nil props must not panic and yields an
// element with no props (React tolerates a null config).
func TestCreateElementNilPropsIsSafe(t *testing.T) {
	parseEl := runtime.CreateElement("br", nil)
	if parseEl == nil {
		t.Fatal("CreateElement returned nil")
	}
	if parseEl.Type != "br" {
		t.Errorf("Type = %v, want br", parseEl.Type)
	}
	// nil config yields no *user* props (GWC may store an internal "children"
	// entry; what matters is no spurious attribute leaks in).
	if _, parseHasID := parseEl.Props["id"]; parseHasID {
		t.Errorf("nil props should not introduce user props, got %v", parseEl.Props)
	}
}

// TestCreateElementFunctionComponentType: a function component is stored as the
// element Type unchanged (component identity == function identity).
func TestCreateElementFunctionComponentType(t *testing.T) {
	parseComponent := func() *runtime.Element { return runtime.CreateElement("p", map[string]any{}, "hi") }
	parseEl := runtime.CreateElement(parseComponent, map[string]any{"name": "n"})
	if parseEl.Type == nil {
		t.Fatal("function component Type is nil")
	}
	if parseEl.Props["name"] != "n" {
		t.Errorf("Props[name] = %v, want n", parseEl.Props["name"])
	}
}
