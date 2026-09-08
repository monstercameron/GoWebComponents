//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// fragmentEl builds a GWC fragment (the "FRAGMENT" marker the reconciler flattens).
func fragmentEl(parseKids ...any) *runtime.Element {
	return runtime.CreateElement("FRAGMENT", nil, parseKids...)
}

// Ported from React's ReactFragment-test.js (render single/zero/multiple/nested
// children). A fragment has no host node of its own — its children hoist into the
// nearest host parent.
func TestFragmentHoistsChildren(t *testing.T) {
	parseSpan := func(parseS string) *runtime.Element { return runtime.CreateElement("span", map[string]any{}, parseS) }

	mountRootChildCount := func(parseTree *runtime.Element) int {
		parseA := mockdom.NewMockDOMAdapter()
		parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
		parseRoot := parseA.CreateElement("div")
		parseRT.RenderInto(parseRoot, parseTree)
		return len(parseA.GetChildren(parseRoot))
	}

	if parseGot := mountRootChildCount(fragmentEl(parseSpan("a"), parseSpan("b"))); parseGot != 2 {
		t.Errorf("fragment(2 children): root host children = %d, want 2", parseGot)
	}
	if parseGot := mountRootChildCount(fragmentEl()); parseGot != 0 {
		t.Errorf("empty fragment: root host children = %d, want 0", parseGot)
	}
	if parseGot := mountRootChildCount(fragmentEl(fragmentEl(parseSpan("a")), parseSpan("b"))); parseGot != 2 {
		t.Errorf("nested fragment: root host children = %d, want 2 (flattened)", parseGot)
	}

	// Fragment nested inside a host element hoists into that host.
	parseA := mockdom.NewMockDOMAdapter()
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
	parseRoot := parseA.CreateElement("div")
	parseRT.RenderInto(parseRoot, runtime.CreateElement("ul", map[string]any{},
		fragmentEl(parseSpan("x"), parseSpan("y")), parseSpan("z")))
	parseUL := parseA.GetChildren(parseRoot)[0]
	if parseGot := len(parseA.GetChildren(parseUL)); parseGot != 3 {
		t.Errorf("fragment inside <ul> + sibling: ul children = %d, want 3", parseGot)
	}
}

// Ported from React's "should preserve state with reordering" fragment tests:
// keyed children inside a fragment keep their component state (and host-node
// identity) when reordered.
func TestFragmentKeyedChildrenPreserveStateOnReorder(t *testing.T) {
	parseA := mockdom.NewMockDOMAdapter()
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
	parseRoot := parseA.CreateElement("div")

	parseSetters := map[string]func(any){}
	// A single, stable component function (component identity == func identity in
	// GWC, as in React). It reads its name from props and is rendered with a key,
	// so the reconciler can preserve its fiber/state across reorder.
	parseCounter := func(parseProps map[string]any) *runtime.Element {
		parseName, _ := parseProps["name"].(string)
		parseGet, parseSet := runtime.GoUseState(parseRT, 0)
		parseSetters[parseName] = parseSet
		return runtime.CreateElement("span", map[string]any{"data-n": parseName}, fmt.Sprint(parseGet()))
	}
	makeCounter := func(parseName string) *runtime.Element {
		return runtime.CreateElement(parseCounter, map[string]any{"key": parseName, "name": parseName})
	}

	// Read each span's text by its data-n marker.
	readState := func() map[string]string {
		parseOut := map[string]string{}
		for _, parseKid := range parseA.GetChildren(parseRoot) {
			if parseNode, parseOk := parseKid.(*mockdom.MockDOMNode); parseOk {
				if parseName := parseNode.Attrs["data-n"]; parseName != "" {
					parseOut[parseName] = parseNode.TextContent
				}
			}
		}
		return parseOut
	}

	parseRT.RenderInto(parseRoot, fragmentEl(makeCounter("a"), makeCounter("b")))
	parseSetters["a"](5) // bump a's state
	if parseState := readState(); parseState["a"] != "5" || parseState["b"] != "0" {
		t.Fatalf("after bump: state = %v, want a=5 b=0", parseState)
	}

	// Reorder the keyed fragment children; a's state must survive the move.
	parseRT.RenderInto(parseRoot, fragmentEl(makeCounter("b"), makeCounter("a")))
	if parseState := readState(); parseState["a"] != "5" || parseState["b"] != "0" {
		t.Errorf("after reorder: state = %v, want a=5 (preserved) b=0", parseState)
	}
}
