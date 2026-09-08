//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v6/ui"
)

// Ported from React's ReactDOMServerIntegrationHooks: hooks run during server
// rendering. ui.RenderToString installs a transient hook fiber per component, so
// GoUseState returns its initial value, GoUseRef/GoUseMemo compute, and
// GoUseContextValue resolves to the nearest provider value (with effects queued
// but never run on the server).

func ssrString(t *testing.T, parseEl *runtime.Element) string {
	t.Helper()
	parseOut, parseErr := ui.RenderToString(ui.Node(parseEl))
	if parseErr != nil {
		t.Fatalf("render error: %v", parseErr)
	}
	return parseOut
}

func TestSSRUseStateRendersInitialValue(t *testing.T) {
	parseComponent := func() *runtime.Element {
		parseGet, _ := runtime.GoUseStateGlobal(7)
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}
	if parseGot := ssrString(t, runtime.CreateElement(parseComponent, map[string]any{})); parseGot != "<span>7</span>" {
		t.Errorf("useState SSR = %q, want <span>7</span>", parseGot)
	}
}

func TestSSRUseRefAndUseMemoCompute(t *testing.T) {
	parseComponent := func() *runtime.Element {
		parseRef := runtime.GoUseRef(99)
		parseMemo := runtime.GoUseMemo(func() any { return "memo" }, 1)
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprintf("%v-%v", parseRef.Current, parseMemo))
	}
	if parseGot := ssrString(t, runtime.CreateElement(parseComponent, map[string]any{})); parseGot != "<span>99-memo</span>" {
		t.Errorf("useRef+useMemo SSR = %q, want <span>99-memo</span>", parseGot)
	}
}

func TestSSRUseEffectIsNotRunOnServer(t *testing.T) {
	parseRan := false
	parseComponent := func() *runtime.Element {
		runtime.GoUseEffect(func() func() { parseRan = true; return nil })
		return runtime.CreateElement("span", map[string]any{}, "x")
	}
	if parseGot := ssrString(t, runtime.CreateElement(parseComponent, map[string]any{})); parseGot != "<span>x</span>" {
		t.Errorf("effect-component SSR = %q", parseGot)
	}
	if parseRan {
		t.Error("useEffect ran during server render; effects must be skipped on the server")
	}
}

func TestSSRUseContextResolvesProviderValue(t *testing.T) {
	parseDesc := runtime.NewContextDescriptor("DEFAULT")
	parseProvider := runtime.NewContextProviderType(parseDesc)
	parseConsumer := func() *runtime.Element {
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(runtime.GoUseContextValue(parseDesc)))
	}

	// Provided value reaches the consumer.
	parseProvided := runtime.CreateElementOwned(parseProvider, map[string]any{"value": "PROVIDED"},
		runtime.CreateElement(parseConsumer, map[string]any{}))
	if parseGot := ssrString(t, parseProvided); parseGot != "<span>PROVIDED</span>" {
		t.Errorf("useContext(provided) SSR = %q", parseGot)
	}

	// Default when no provider.
	if parseGot := ssrString(t, runtime.CreateElement(parseConsumer, map[string]any{})); parseGot != "<span>DEFAULT</span>" {
		t.Errorf("useContext(default) SSR = %q", parseGot)
	}

	// Nested provider overrides for a consumer nested under host elements.
	parseNested := runtime.CreateElementOwned(parseProvider, map[string]any{"value": "OUTER"},
		runtime.CreateElement("div", map[string]any{},
			runtime.CreateElementOwned(parseProvider, map[string]any{"value": "INNER"},
				runtime.CreateElement("section", map[string]any{},
					runtime.CreateElement(parseConsumer, map[string]any{})))))
	if parseGot := ssrString(t, parseNested); parseGot != "<div><section><span>INNER</span></section></div>" {
		t.Errorf("nested provider SSR = %q", parseGot)
	}
}

// The reconciler (client/native) path also renders hook components — kept to pin
// both render paths agree on initial output.
func TestHookComponentRendersViaReconciler(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")
	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseGet, _ := runtime.GoUseState(rt, 42)
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))
	parseSpan, _ := adapter.GetChildren(root)[0].(*mockdom.MockDOMNode)
	if parseSpan == nil || parseSpan.TextContent != "42" {
		t.Errorf("reconciler render = %+v, want text 42", parseSpan)
	}
}
