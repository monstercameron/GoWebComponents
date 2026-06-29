//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// readDeepText returns the deepest first-child text under root (helper local to
// this file to avoid clashing with the hook harness in hooks_state_native_test).
func readDeepText(parseA *mockdom.MockDOMAdapter, parseRoot runtime.DOMNode) string {
	parseKids := parseA.GetChildren(parseRoot)
	if len(parseKids) == 0 {
		return ""
	}
	parseNode, _ := parseKids[0].(*mockdom.MockDOMNode)
	for parseNode != nil && parseNode.TextContent == "" && len(parseNode.Children) > 0 {
		parseNode = parseNode.Children[0]
	}
	if parseNode == nil {
		return ""
	}
	return parseNode.TextContent
}

// Ported from React's ReactHooksWithNoopRenderer ("returns the same updater
// function every time" + functional updates): the state setter has stable
// identity across renders and accepts a functional updater (prev -> next).
func TestHookFunctionalUpdateAndStableSetter(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseSet func(any)
	var parseSetterIDs []string
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGet, parseSetter := runtime.GoUseState(rt, 10)
		parseSet = parseSetter
		parseSetterIDs = append(parseSetterIDs, fmt.Sprintf("%p", parseSetter))
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	if parseGot := readDeepText(adapter, root); parseGot != "10" {
		t.Fatalf("mount text = %q, want 10", parseGot)
	}

	// Functional updater: prev -> prev+1.
	parseSet(func(parsePrev int) int { return parsePrev + 1 })
	if parseGot := readDeepText(adapter, root); parseGot != "11" {
		t.Errorf("after functional update text = %q, want 11", parseGot)
	}

	// Plain value still works.
	parseSet(42)
	if parseGot := readDeepText(adapter, root); parseGot != "42" {
		t.Errorf("after plain set text = %q, want 42", parseGot)
	}

	// The setter identity must be identical across all renders so far.
	for parseI := 1; parseI < len(parseSetterIDs); parseI++ {
		if parseSetterIDs[parseI] != parseSetterIDs[0] {
			t.Errorf("setter identity changed across renders: %v", parseSetterIDs)
			break
		}
	}
}

// Ported from React's "does not warn on set after unmount": calling a state
// setter after its component has unmounted is a safe no-op (no panic, no effect).
func TestHookSetAfterUnmountIsSafe(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseSet func(any)
	parseChild := func() *runtime.Element {
		_, parseSetter := runtime.GoUseState(rt, 0)
		parseSet = parseSetter
		return runtime.CreateElement("span", map[string]any{}, "child")
	}
	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{},
		runtime.CreateElement(parseChild, map[string]any{})))

	// Unmount the child.
	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{}))
	parseOuter := adapter.GetChildren(root)[0]
	if parseN := len(adapter.GetChildren(parseOuter)); parseN != 0 {
		t.Fatalf("child not unmounted: %d children remain", parseN)
	}

	// This must not panic.
	parseSet(5)
	parseSet(func(parsePrev int) int { return parsePrev + 1 })
}
