//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// TestConditionalNilRenderTogglesMount: a component that returns nil renders
// nothing, and toggling the condition unmounts then re-mounts its subtree —
// the common `if !show { return nil }` pattern, verified natively.
func TestConditionalNilRenderTogglesMount(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseSetShow func(any)
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseShow, parseSetter := runtime.GoUseState(rt, true)
		parseSetShow = parseSetter
		if !parseShow() {
			return nil
		}
		return runtime.CreateElement("span", map[string]any{}, "visible")
	}, map[string]any{}))

	childCount := func() int { return len(adapter.GetChildren(root)) }

	if parseGot := childCount(); parseGot != 1 {
		t.Fatalf("show=true: want 1 child, got %d", parseGot)
	}
	parseSetShow(false)
	if parseGot := childCount(); parseGot != 0 {
		t.Errorf("nil render: want 0 children, got %d", parseGot)
	}
	parseSetShow(true)
	if parseGot := childCount(); parseGot != 1 {
		t.Errorf("re-show: want 1 child remounted, got %d", parseGot)
	}
}
