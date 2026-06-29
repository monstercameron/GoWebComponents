//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

// Ported from React's ReactHooksWithNoopRenderer render-phase-update tests
// ("keeps restarting until there are no more new updates"). When a component
// updates its own state during render, GWC re-runs the component to convergence
// so the committed output reflects the final state — not a half-rendered
// intermediate. Regression test for the v3.5.3 fix (previously the DOM showed a
// value the component never actually rendered).
func TestRenderPhaseUpdateConverges(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseSeen []int
	parseRenders := 0
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseRenders++
		parseGet, parseSet := runtime.GoUseState(rt, 0)
		parseSeen = append(parseSeen, parseGet())
		if parseGet() < 3 {
			parseSet(parseGet() + 1)
		}
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	parseNode, _ := adapter.GetChildren(root)[0].(*mockdom.MockDOMNode)
	if parseNode == nil || parseNode.TextContent != "3" {
		t.Errorf("render-phase update did not converge: domText = %q, want 3 (seen %v, renders %d)",
			func() string {
				if parseNode == nil {
					return "<nil>"
				}
				return parseNode.TextContent
			}(), parseSeen, parseRenders)
	}
	// Each render observed the incrementing value; the final committed output
	// matches the final render.
	if len(parseSeen) == 0 || parseSeen[len(parseSeen)-1] != 3 {
		t.Errorf("expected the last render to observe state 3, seen = %v", parseSeen)
	}
}

// A conditional render-phase update (derived state) is the common, supported
// pattern: set only when a prop/state differs, converging in one extra pass.
func TestRenderPhaseDerivedStateOneStep(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	parseRenders := 0
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseRenders++
		parseGet, parseSet := runtime.GoUseState(rt, "init")
		if parseGet() == "init" {
			parseSet("derived")
		}
		return runtime.CreateElement("span", map[string]any{}, parseGet())
	}, map[string]any{}))

	parseNode, _ := adapter.GetChildren(root)[0].(*mockdom.MockDOMNode)
	if parseNode == nil || parseNode.TextContent != "derived" {
		t.Errorf("derived-state render = %v, want derived", parseNode)
	}
	if parseRenders != 2 {
		t.Errorf("expected 2 renders (initial + one convergence), got %d", parseRenders)
	}
}

// An unconditional render-phase update must not hang — the convergence loop is
// bounded and gives up after the cap.
func TestRenderPhaseUnconditionalDoesNotHang(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	// Must return (not hang); we don't assert the value, only that it terminates.
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGet, parseSet := runtime.GoUseState(rt, 0)
		parseSet(parseGet() + 1)
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))
	if len(adapter.GetChildren(root)) == 0 {
		t.Fatal("nothing rendered")
	}
}
