//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

func keyedListWithData(parseKeys ...string) *runtime.Element {
	parseKids := make([]any, len(parseKeys))
	for parseI, parseK := range parseKeys {
		parseKids[parseI] = runtime.CreateElement("li", map[string]any{"key": parseK, "data-k": parseK}, parseK)
	}
	return runtime.CreateElement("ul", map[string]any{}, parseKids...)
}

// Deepens keyed-reconciliation coverage with the harder move patterns React's
// reconciler tests exercise: full reverse (identity preserved), simultaneous
// insert+remove, and collapse — each producing the correct order and child count.
func TestKeyedComplexMoves(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	order := func() []string {
		parseUL := adapter.GetChildren(root)[0]
		var parseOut []string
		for _, parseLI := range adapter.GetChildren(parseUL) {
			if parseN, parseOk := parseLI.(*mockdom.MockDOMNode); parseOk {
				parseOut = append(parseOut, parseN.Attrs["data-k"])
			}
		}
		return parseOut
	}
	idOfKey := func(parseKey string) int {
		parseUL := adapter.GetChildren(root)[0]
		for _, parseLI := range adapter.GetChildren(parseUL) {
			if parseN, parseOk := parseLI.(*mockdom.MockDOMNode); parseOk && parseN.Attrs["data-k"] == parseKey {
				return parseN.ID
			}
		}
		return -1
	}
	equal := func(parseGot, parseWant []string) bool {
		if len(parseGot) != len(parseWant) {
			return false
		}
		for parseI := range parseGot {
			if parseGot[parseI] != parseWant[parseI] {
				return false
			}
		}
		return true
	}

	rt.RenderInto(root, keyedListWithData("a", "b", "c", "d", "e"))
	parseIDA, parseIDE := idOfKey("a"), idOfKey("e")

	// Full reverse: order flips, every node is moved (not recreated).
	rt.RenderInto(root, keyedListWithData("e", "d", "c", "b", "a"))
	if !equal(order(), []string{"e", "d", "c", "b", "a"}) {
		t.Fatalf("reverse order = %v", order())
	}
	if idOfKey("a") != parseIDA || idOfKey("e") != parseIDE {
		t.Errorf("reverse recreated nodes (a: %d->%d, e: %d->%d)", parseIDA, idOfKey("a"), parseIDE, idOfKey("e"))
	}

	// Simultaneous insert (x, y) and remove (b, d).
	rt.RenderInto(root, keyedListWithData("e", "x", "c", "y", "a"))
	if !equal(order(), []string{"e", "x", "c", "y", "a"}) {
		t.Errorf("insert+remove order = %v, want [e x c y a]", order())
	}

	// Collapse to a single survivor.
	rt.RenderInto(root, keyedListWithData("c"))
	if !equal(order(), []string{"c"}) {
		t.Errorf("collapse order = %v, want [c]", order())
	}

	// Grow back from one.
	rt.RenderInto(root, keyedListWithData("c", "z", "w"))
	if !equal(order(), []string{"c", "z", "w"}) {
		t.Errorf("grow order = %v, want [c z w]", order())
	}
}
