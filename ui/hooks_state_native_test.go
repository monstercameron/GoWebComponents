//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// hookHarness mounts a function component natively and reads back the rendered
// text. Because the harness configures no scheduler, set()/state updates flush
// synchronously (dispatchRuntimeWork runs work inline when scheduler == nil), so
// a re-render is observable immediately after the setter returns.
type hookHarness struct {
	adapter *mockdom.MockDOMAdapter
	rt      *runtime.Runtime
	root    runtime.DOMNode
}

func newHookHarness() *hookHarness {
	adapter := mockdom.NewMockDOMAdapter()
	return &hookHarness{
		adapter: adapter,
		rt:      runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true}),
		root:    adapter.CreateElement("div"),
	}
}

func (parseH *hookHarness) mount(parseComponent func() *runtime.Element) {
	parseH.rt.RenderInto(parseH.root, runtime.CreateElement(parseComponent, map[string]any{}))
}

// text returns the deepest first-child text content under the root.
func (parseH *hookHarness) text() string {
	parseKids := parseH.adapter.GetChildren(parseH.root)
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

// TestHookStateRerendersAndPersists: a state update re-renders the component and
// the new value is committed to the host DOM; a same-value set does not re-render.
func TestHookStateRerendersAndPersists(t *testing.T) {
	parseH := newHookHarness()
	var parseSet func(any)
	parseRenders := 0
	parseH.mount(func() *runtime.Element {
		parseRenders++
		parseGet, parseSetter := runtime.GoUseState(parseH.rt, 0)
		parseSet = parseSetter
		return runtime.CreateElement("p", map[string]any{}, fmt.Sprint(parseGet()))
	})

	if parseRenders != 1 || parseH.text() != "0" {
		t.Fatalf("mount: renders=%d text=%q", parseRenders, parseH.text())
	}
	parseSet(5)
	if parseRenders != 2 || parseH.text() != "5" {
		t.Fatalf("after set(5): renders=%d text=%q", parseRenders, parseH.text())
	}
	parseSet(5) // same value
	if parseRenders != 2 {
		t.Errorf("same-value set caused a re-render: renders=%d", parseRenders)
	}
	parseSet(7)
	if parseRenders != 3 || parseH.text() != "7" {
		t.Errorf("after set(7): renders=%d text=%q", parseRenders, parseH.text())
	}
}

// TestHookMultipleStatesIndependent: two states in one component update
// independently and keep their slots stable across re-renders (hook ordering).
func TestHookMultipleStatesIndependent(t *testing.T) {
	parseH := newHookHarness()
	var parseSetA, parseSetB func(any)
	parseH.mount(func() *runtime.Element {
		parseGetA, parseSetterA := runtime.GoUseState(parseH.rt, "a0")
		parseGetB, parseSetterB := runtime.GoUseState(parseH.rt, "b0")
		parseSetA = parseSetterA
		parseSetB = parseSetterB
		return runtime.CreateElement("p", map[string]any{}, parseGetA()+"|"+parseGetB())
	})

	if parseH.text() != "a0|b0" {
		t.Fatalf("mount: %q", parseH.text())
	}
	parseSetA("a1")
	if parseH.text() != "a1|b0" {
		t.Errorf("after setA: %q (state B leaked or hook order drifted)", parseH.text())
	}
	parseSetB("b1")
	if parseH.text() != "a1|b1" {
		t.Errorf("after setB: %q", parseH.text())
	}
}

// TestHookStateOrderStableAcrossRenders: many states keep correct values across
// repeated re-renders — a regression here would mean hook slots are mis-indexed.
func TestHookStateOrderStableAcrossRenders(t *testing.T) {
	parseH := newHookHarness()
	var parseSetters []func(any)
	parseH.mount(func() *runtime.Element {
		parseSetters = parseSetters[:0]
		parseParts := ""
		for parseI := range 5 {
			parseGet, parseSet := runtime.GoUseState(parseH.rt, parseI)
			parseSetters = append(parseSetters, parseSet)
			parseParts += fmt.Sprint(parseGet())
		}
		return runtime.CreateElement("p", map[string]any{}, parseParts)
	})

	if parseH.text() != "01234" {
		t.Fatalf("mount: %q", parseH.text())
	}
	parseSetters[2](9) // bump the middle slot
	if parseH.text() != "01934" {
		t.Errorf("after bumping slot 2: %q (hook indexing drifted)", parseH.text())
	}
}
