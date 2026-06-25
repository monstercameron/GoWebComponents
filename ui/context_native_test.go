//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

type ctxHarness struct {
	adapter *mockdom.MockDOMAdapter
	rt      *runtime.Runtime
	root    runtime.DOMNode
}

func newCtxHarness() *ctxHarness {
	adapter := mockdom.NewMockDOMAdapter()
	return &ctxHarness{
		adapter: adapter,
		rt:      runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true}),
		root:    adapter.CreateElement("div"),
	}
}

// allText concatenates the text content of every node in the mounted tree, so a
// test can assert on what consumers rendered regardless of nesting depth.
func (parseH *ctxHarness) allText() string {
	var parseWalk func(parseNode *mockdom.MockDOMNode) string
	parseWalk = func(parseNode *mockdom.MockDOMNode) string {
		if parseNode == nil {
			return ""
		}
		parseOut := parseNode.TextContent
		for _, parseChild := range parseNode.Children {
			parseOut += parseWalk(parseChild)
		}
		return parseOut
	}
	parseOut := ""
	for _, parseKid := range parseH.adapter.GetChildren(parseH.root) {
		if parseNode, parseOk := parseKid.(*mockdom.MockDOMNode); parseOk {
			parseOut += parseWalk(parseNode)
		}
	}
	return parseOut
}

// TestContextProvidesAndDefaults: a consumer reads the nearest provider's value,
// or the descriptor default when no provider is present.
func TestContextProvidesAndDefaults(t *testing.T) {
	parseDesc := runtime.NewContextDescriptor("DEFAULT")
	parseProvider := runtime.NewContextProviderType(parseDesc)
	parseConsumer := func() *runtime.Element {
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(runtime.GoUseContextValue(parseDesc)))
	}

	parseH := newCtxHarness()
	parseH.rt.RenderInto(parseH.root, runtime.CreateElementOwned(parseProvider, map[string]any{"value": "HELLO"},
		runtime.CreateElement(parseConsumer, map[string]any{})))
	if parseH.allText() != "HELLO" {
		t.Errorf("inside provider: got %q want HELLO", parseH.allText())
	}

	parseH2 := newCtxHarness()
	parseH2.rt.RenderInto(parseH2.root, runtime.CreateElement(parseConsumer, map[string]any{}))
	if parseH2.allText() != "DEFAULT" {
		t.Errorf("no provider: got %q want DEFAULT", parseH2.allText())
	}
}

// TestContextNestedProviderOverrides: a nested provider shadows the outer value
// for consumers below it (the classic context-precedence invariant).
func TestContextNestedProviderOverrides(t *testing.T) {
	parseDesc := runtime.NewContextDescriptor("DEFAULT")
	parseProvider := runtime.NewContextProviderType(parseDesc)
	parseConsumer := func() *runtime.Element {
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(runtime.GoUseContextValue(parseDesc)))
	}

	parseH := newCtxHarness()
	parseH.rt.RenderInto(parseH.root,
		runtime.CreateElementOwned(parseProvider, map[string]any{"value": "OUTER"},
			runtime.CreateElement(parseConsumer, map[string]any{}),
			runtime.CreateElementOwned(parseProvider, map[string]any{"value": "INNER"},
				runtime.CreateElement(parseConsumer, map[string]any{}))))

	// One consumer sees OUTER, the nested one sees INNER.
	if got := parseH.allText(); got != "OUTERINNER" {
		t.Errorf("nested providers: got %q want OUTERINNER", got)
	}
}

// TestContextValueUpdatePropagates: changing the provider value (via a parent
// state update) re-renders the consumer with the new value.
func TestContextValueUpdatePropagates(t *testing.T) {
	parseDesc := runtime.NewContextDescriptor("v0")
	parseProvider := runtime.NewContextProviderType(parseDesc)
	parseConsumer := func() *runtime.Element {
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(runtime.GoUseContextValue(parseDesc)))
	}

	parseH := newCtxHarness()
	var parseSet func(any)
	parseApp := func() *runtime.Element {
		parseGet, parseSetter := runtime.GoUseState(parseH.rt, "v0")
		parseSet = parseSetter
		return runtime.CreateElementOwned(parseProvider, map[string]any{"value": parseGet()},
			runtime.CreateElement(parseConsumer, map[string]any{}))
	}

	parseH.rt.RenderInto(parseH.root, runtime.CreateElement(parseApp, map[string]any{}))
	if parseH.allText() != "v0" {
		t.Fatalf("mount: got %q want v0", parseH.allText())
	}
	parseSet("v1")
	if parseH.allText() != "v1" {
		t.Errorf("after update: got %q want v1 (context did not propagate)", parseH.allText())
	}
}
