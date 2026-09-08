//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

func dupKeyUL(parseKeys ...string) *runtime.Element {
	parseKids := make([]any, len(parseKeys))
	for parseI, parseK := range parseKeys {
		parseKids[parseI] = runtime.CreateElement("li", map[string]any{"key": parseK}, parseK)
	}
	return runtime.CreateElement("ul", map[string]any{}, parseKids...)
}

func ulChildCount(parseA *mockdom.MockDOMAdapter, parseRoot runtime.DOMNode) int {
	parseKids := parseA.GetChildren(parseRoot)
	if len(parseKids) == 0 {
		return -1
	}
	return len(parseA.GetChildren(parseKids[0]))
}

// TestReconcileDuplicateKeysDoNotLeak is a regression test for a reconciler bug
// where duplicate keys in a list orphaned a stale host node: the keyed old-fiber
// map holds one fiber per key, so a second fiber with the same key overwrote the
// first, and the overwritten fiber was never tagged for deletion. The list's host
// child count then grew past the element count and stale nodes bled into later
// (even structurally different) renders. Duplicate-keyed old fibers are now routed
// to the positionally-matched fallback list so every old fiber is cleaned up.
func TestReconcileDuplicateKeysDoNotLeak(t *testing.T) {
	parseA := mockdom.NewMockDOMAdapter()
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
	parseRoot := parseA.CreateElement("div")

	steps := []struct {
		keys []string
		want int
	}{
		{[]string{"a", "a", "b"}, 3},      // mount with a duplicate
		{[]string{"a", "b"}, 2},           // shrink: the extra "a" must be removed
		{[]string{"b", "a", "a"}, 3},      // grow back with the duplicate reordered
		{[]string{"a"}, 1},                // collapse to a single node
		{[]string{"x", "y", "z"}, 3},      // structurally different list: no stale bleed
		{[]string{"a", "a", "a", "a"}, 4}, // many duplicates
		{[]string{}, 0},                   // empty
	}
	for _, parseStep := range steps {
		parseRT.RenderInto(parseRoot, dupKeyUL(parseStep.keys...))
		if parseGot := ulChildCount(parseA, parseRoot); parseGot != parseStep.want {
			t.Errorf("keys=%v: host child count = %d, want %d (node leak/corruption)", parseStep.keys, parseGot, parseStep.want)
		}
	}
}

// TestReconcileUniqueKeysUnaffected guards that the duplicate-key fix did not
// change correct behavior for the normal unique-key case.
func TestReconcileUniqueKeysUnaffected(t *testing.T) {
	parseA := mockdom.NewMockDOMAdapter()
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
	parseRoot := parseA.CreateElement("div")

	parseRT.RenderInto(parseRoot, dupKeyUL("a", "b", "c"))
	if parseGot := ulChildCount(parseA, parseRoot); parseGot != 3 {
		t.Fatalf("mount: got %d want 3", parseGot)
	}
	parseRT.RenderInto(parseRoot, dupKeyUL("c", "a", "b"))
	if parseGot := ulChildCount(parseA, parseRoot); parseGot != 3 {
		t.Errorf("reorder: got %d want 3", parseGot)
	}
	parseRT.RenderInto(parseRoot, dupKeyUL("a", "c"))
	if parseGot := ulChildCount(parseA, parseRoot); parseGot != 2 {
		t.Errorf("removal: got %d want 2", parseGot)
	}
}
