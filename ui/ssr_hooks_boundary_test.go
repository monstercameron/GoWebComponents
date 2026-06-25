//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
	"github.com/monstercameron/GoWebComponents/ui"
)

// This documents an architectural difference from React's
// ReactDOMServerIntegrationHooks: GWC's string serializer (ui.RenderToString) is
// intentionally a lightweight, hook-less render path — it resolves components but
// does not establish the hook/fiber context. A component that calls a hook
// therefore renders to a string only through the reconciler (the client/native
// render path), which DOES run hooks. The two assertions below pin both halves of
// that boundary so a future change (e.g. adding SSR hook support) is noticed.

// TestHookComponentRendersViaReconciler: the supported path — a hook-using
// component renders its initial state when driven through the reconciler.
func TestHookComponentRendersViaReconciler(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGet, _ := runtime.GoUseState(rt, 42)
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	parseSpan := adapter.GetChildren(root)
	if len(parseSpan) == 0 {
		t.Fatal("nothing rendered")
	}
	parseNode, _ := parseSpan[0].(*mockdom.MockDOMNode)
	if parseNode == nil || parseNode.TextContent != "42" {
		t.Errorf("reconciler render of hook component = %+v, want text 42", parseNode)
	}
}

// TestRenderToStringRejectsHookComponent: the documented boundary — string SSR of
// a hook-using component returns the "outside component context" error rather than
// silently dropping state. (If GWC gains SSR hook support, update this test.)
func TestRenderToStringRejectsHookComponent(t *testing.T) {
	parseComponent := func() *runtime.Element {
		parseGet, _ := runtime.GoUseStateGlobal(1)
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}
	_, parseErr := ui.RenderToString(ui.Node(runtime.CreateElement(parseComponent, map[string]any{})))
	if parseErr == nil {
		t.Fatal("expected an error rendering a hook component via RenderToString (SSR is hook-less)")
	}
	if !strings.Contains(parseErr.Error(), "outside component context") {
		t.Errorf("unexpected error: %v", parseErr)
	}
}
