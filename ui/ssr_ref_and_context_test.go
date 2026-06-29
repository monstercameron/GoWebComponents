//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
	"github.com/monstercameron/GoWebComponents/v4/ui"
)

type countingSink struct{ calls int }

func (parseS *countingSink) SetDOMNode(parseNode runtime.DOMNode) { parseS.calls++ }

// Ported from React's ReactDOMServerIntegrationRefs: a ref does not attach during
// server rendering (there is no live DOM), and the ref sink must never leak into
// the serialized markup as an attribute.
func TestSSRRefIsInertDuringServerRender(t *testing.T) {
	parseSink := &countingSink{}
	parseNode := runtime.CreateElement("div", map[string]any{runtime.DOMRefKey: parseSink}, "x")
	parseOut, parseErr := ui.RenderToString(ui.Node(parseNode))
	if parseErr != nil {
		t.Fatalf("render error: %v", parseErr)
	}
	if parseOut != "<div>x</div>" {
		t.Errorf("ref leaked into SSR markup: %q", parseOut)
	}
	if parseSink.calls != 0 {
		t.Errorf("ref attached during SSR: sink called %d times, want 0", parseSink.calls)
	}
}

// Ported from React's ReactNewContext (multiple consumers): every consumer under
// a provider reads its value, and updating the provider value (via a parent state
// change) updates all consumers in one render pass.
func TestContextMultipleConsumersAllUpdate(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	parseDesc := runtime.NewContextDescriptor("v0")
	parseProvider := runtime.NewContextProviderType(parseDesc)
	parseConsumer := func(parseProps map[string]any) *runtime.Element {
		parseTag, _ := parseProps["data-c"].(string)
		return runtime.CreateElement("span", map[string]any{"data-c": parseTag},
			fmt.Sprint(runtime.GoUseContextValue(parseDesc)))
	}
	makeConsumer := func(parseTag string) *runtime.Element {
		return runtime.CreateElement(parseConsumer, map[string]any{"key": parseTag, "data-c": parseTag})
	}

	// Collect all consumer texts by their data-c marker (walk the tree).
	collect := func() map[string]string {
		parseOut := map[string]string{}
		var parseWalk func(parseNode *mockdom.MockDOMNode)
		parseWalk = func(parseNode *mockdom.MockDOMNode) {
			if parseNode == nil {
				return
			}
			if parseC := parseNode.Attrs["data-c"]; parseC != "" {
				parseOut[parseC] = parseNode.TextContent
			}
			for _, parseChild := range parseNode.Children {
				parseWalk(parseChild)
			}
		}
		for _, parseKid := range adapter.GetChildren(root) {
			if parseNode, parseOk := parseKid.(*mockdom.MockDOMNode); parseOk {
				parseWalk(parseNode)
			}
		}
		return parseOut
	}

	var parseSet func(any)
	parseApp := func() *runtime.Element {
		parseGet, parseSetter := runtime.GoUseState(rt, "v0")
		parseSet = parseSetter
		return runtime.CreateElementOwned(parseProvider, map[string]any{"value": parseGet()},
			makeConsumer("a"), makeConsumer("b"), makeConsumer("c"))
	}

	rt.RenderInto(root, runtime.CreateElement(parseApp, map[string]any{}))
	if parseState := collect(); parseState["a"] != "v0" || parseState["b"] != "v0" || parseState["c"] != "v0" {
		t.Fatalf("mount: consumers = %v, want all v0", parseState)
	}

	parseSet("v1")
	if parseState := collect(); parseState["a"] != "v1" || parseState["b"] != "v1" || parseState["c"] != "v1" {
		t.Errorf("after update: consumers = %v, want all v1", parseState)
	}
}
