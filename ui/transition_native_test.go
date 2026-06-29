//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Ported from React's ReactStartTransition: a state update wrapped in
// StartTransition is applied (it commits to the host DOM); a same-value
// transition update does not re-render; multiple updates inside one transition
// all take effect.
func TestStartTransitionAppliesUpdates(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseSetA, parseSetB func(any)
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGetA, parseSA := runtime.GoUseState(rt, 0)
		parseGetB, parseSB := runtime.GoUseState(rt, 0)
		parseSetA = parseSA
		parseSetB = parseSB
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprintf("%v-%v", parseGetA(), parseGetB()))
	}, map[string]any{}))

	read := func() string {
		parseNode, _ := adapter.GetChildren(root)[0].(*mockdom.MockDOMNode)
		return parseNode.TextContent
	}

	if read() != "0-0" {
		t.Fatalf("mount = %q, want 0-0", read())
	}

	// A single transition update commits.
	rt.StartTransition(func() { parseSetA(7) })
	if read() != "7-0" {
		t.Errorf("after StartTransition setA(7) = %q, want 7-0", read())
	}

	// Multiple updates inside one transition all take effect.
	rt.StartTransition(func() {
		parseSetA(1)
		parseSetB(2)
	})
	if read() != "1-2" {
		t.Errorf("after multi-update transition = %q, want 1-2", read())
	}

	// A same-value transition update is a no-op (final value unchanged).
	rt.StartTransition(func() { parseSetA(1) })
	if read() != "1-2" {
		t.Errorf("after same-value transition = %q, want 1-2", read())
	}
}
