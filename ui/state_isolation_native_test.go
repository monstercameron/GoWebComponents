//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// childTextByMarker finds the text of the node carrying data-r="child".
func childTextByMarker(parseA *mockdom.MockDOMAdapter, parseRoot runtime.DOMNode) string {
	var parseWalk func(parseN *mockdom.MockDOMNode) string
	parseWalk = func(parseN *mockdom.MockDOMNode) string {
		if parseN == nil {
			return ""
		}
		if parseN.Attrs["data-r"] == "child" {
			return parseN.TextContent
		}
		for _, parseC := range parseN.Children {
			if parseR := parseWalk(parseC); parseR != "" {
				return parseR
			}
		}
		return ""
	}
	for _, parseK := range parseA.GetChildren(parseRoot) {
		if parseN, parseOk := parseK.(*mockdom.MockDOMNode); parseOk {
			if parseR := parseWalk(parseN); parseR != "" {
				return parseR
			}
		}
	}
	return ""
}

// Ported from React's ReactIncremental / state-preservation and bailout tests:
// a child component keeps its own state across a parent re-render, and a parent
// re-render that does not change the child's props bails out of re-rendering the
// child (automatic memoization). Changing the child's props re-renders it.
func TestChildStateIsolationAndBailout(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseSetParent, parseSetChild func(any)
	parseChildRenders := 0

	parseChild := func(parseProps map[string]any) *runtime.Element {
		parseChildRenders++
		parseLabel, _ := parseProps["label"].(string)
		parseGet, parseSet := runtime.GoUseState(rt, 100)
		parseSetChild = parseSet
		return runtime.CreateElement("span", map[string]any{"data-r": "child"},
			fmt.Sprintf("%s:%v", parseLabel, parseGet()))
	}
	parseParent := func() *runtime.Element {
		parseGet, parseSet := runtime.GoUseState(rt, 0)
		parseSetParent = parseSet
		return runtime.CreateElement("div", map[string]any{},
			runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet())),
			runtime.CreateElement(parseChild, map[string]any{"key": "c", "label": "L"}))
	}

	rt.RenderInto(root, runtime.CreateElement(parseParent, map[string]any{}))
	if parseChildTextByMarker := childTextByMarker(adapter, root); parseChildTextByMarker != "L:100" {
		t.Fatalf("mount child = %q, want L:100", parseChildTextByMarker)
	}
	parseRendersAfterMount := parseChildRenders

	// Child updates its own state.
	parseSetChild(200)
	if parseGot := childTextByMarker(adapter, root); parseGot != "L:200" {
		t.Errorf("after setChild = %q, want L:200", parseGot)
	}

	// Parent re-renders without changing the child's props: the child keeps its
	// state AND is not re-rendered (bailout).
	parseRendersBeforeParent := parseChildRenders
	parseSetParent(5)
	if parseGot := childTextByMarker(adapter, root); parseGot != "L:200" {
		t.Errorf("after setParent the child state was lost: %q, want L:200", parseGot)
	}
	if parseChildRenders != parseRendersBeforeParent {
		t.Errorf("child re-rendered on an unrelated parent update (no bailout): %d -> %d",
			parseRendersBeforeParent, parseChildRenders)
	}

	_ = parseRendersAfterMount
}
