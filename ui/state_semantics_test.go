//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v5/internal/runtime"
)

// stateHarness mounts a single stateful component and exposes its first state
// hook for driving. The component re-renders synchronously (nil scheduler).
type stateHarness struct {
	adapter *mockdom.MockDOMAdapter
	rt      *runtime.Runtime
	root    runtime.DOMNode
}

func newStateHarness() *stateHarness {
	adapter := mockdom.NewMockDOMAdapter()
	return &stateHarness{
		adapter: adapter,
		rt:      runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true}),
		root:    adapter.CreateElement("div"),
	}
}

func (parseH *stateHarness) text() string {
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

// TestStateFunctionalUpdatesCompose: chained functional updaters compose, and a
// plain value can follow an updater.
func TestStateFunctionalUpdatesCompose(t *testing.T) {
	parseH := newStateHarness()
	var parseSet func(any)
	parseH.rt.RenderInto(parseH.root, runtime.CreateElement(func() *runtime.Element {
		parseGet, parseSetter := runtime.GoUseState(parseH.rt, 0)
		parseSet = parseSetter
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	parseSet(func(parseP int) int { return parseP + 1 })
	parseSet(func(parseP int) int { return parseP + 1 })
	if parseH.text() != "2" {
		t.Errorf("2x +1 = %q, want 2", parseH.text())
	}
	parseSet(10)
	parseSet(func(parseP int) int { return parseP + 5 })
	if parseH.text() != "15" {
		t.Errorf("set(10);+5 = %q, want 15", parseH.text())
	}
}

// TestStateGetterIsLiveRead documents GWC's design choice (a deliberate
// difference from React's per-render snapshot): the getter returned by GoUseState
// reads the current state live, so a getter captured at an earlier render returns
// the latest value rather than a stale snapshot. This avoids React's stale-closure
// footgun and is why functional updates / render-phase convergence read fresh
// state.
func TestStateGetterIsLiveRead(t *testing.T) {
	parseH := newStateHarness()
	var parseCaptured func() int
	var parseSet func(any)
	parseRenders := 0
	parseH.rt.RenderInto(parseH.root, runtime.CreateElement(func() *runtime.Element {
		parseRenders++
		parseGet, parseSetter := runtime.GoUseState(parseH.rt, 0)
		parseSet = parseSetter
		if parseRenders == 1 {
			parseCaptured = parseGet
		}
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	if parseCaptured() != 0 {
		t.Fatalf("initial captured getter = %d, want 0", parseCaptured())
	}
	parseSet(42)
	if parseCaptured() != 42 {
		t.Errorf("captured getter after set = %d; GWC getter is a live read, want 42", parseCaptured())
	}
}

// TestStateReferenceVsValueDedup: slice/map state (non-comparable, reference-like)
// re-renders on a new equal-content value (like React's Object.is on a new
// object), while a comparable struct value dedups on equality (Go value
// semantics).
func TestStateReferenceVsValueDedup(t *testing.T) {
	// Slice: new equal-content slice re-renders.
	parseSliceH := newStateHarness()
	var parseSetSlice func(any)
	parseSliceRenders := 0
	parseSliceH.rt.RenderInto(parseSliceH.root, runtime.CreateElement(func() *runtime.Element {
		parseSliceRenders++
		parseGet, parseSetter := runtime.GoUseState(parseSliceH.rt, []int{1, 2, 3})
		parseSetSlice = parseSetter
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(len(parseGet())))
	}, map[string]any{}))
	parseSetSlice([]int{1, 2, 3})
	if parseSliceRenders != 2 {
		t.Errorf("slice: new equal-content value should re-render (renders=%d, want 2)", parseSliceRenders)
	}

	// Struct: equal value dedups (no re-render).
	type parsePoint struct{ X, Y int }
	parseStructH := newStateHarness()
	var parseSetPt func(any)
	parseStructRenders := 0
	parseStructH.rt.RenderInto(parseStructH.root, runtime.CreateElement(func() *runtime.Element {
		parseStructRenders++
		parseGet, parseSetter := runtime.GoUseState(parseStructH.rt, parsePoint{1, 2})
		parseSetPt = parseSetter
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))
	parseSetPt(parsePoint{1, 2})
	if parseStructRenders != 1 {
		t.Errorf("struct: equal value should dedup (renders=%d, want 1)", parseStructRenders)
	}
	parseSetPt(parsePoint{3, 4})
	if parseStructRenders != 2 {
		t.Errorf("struct: changed value should re-render (renders=%d, want 2)", parseStructRenders)
	}
}

// TestStateFunctionValueStorable: a function can be stored as state — the setter
// distinguishes a functional updater (func(T) T) from a plain function value.
func TestStateFunctionValueStorable(t *testing.T) {
	parseH := newStateHarness()
	var parseSet func(any)
	parseH.rt.RenderInto(parseH.root, runtime.CreateElement(func() *runtime.Element {
		parseGet, parseSetter := runtime.GoUseState[func() string](parseH.rt, func() string { return "initial" })
		parseSet = parseSetter
		parseFn := parseGet()
		parseLabel := "<nil>"
		if parseFn != nil {
			parseLabel = parseFn()
		}
		return runtime.CreateElement("span", map[string]any{}, parseLabel)
	}, map[string]any{}))

	if parseH.text() != "initial" {
		t.Fatalf("mount = %q, want initial", parseH.text())
	}
	parseSet(func() string { return "stored" })
	if parseH.text() != "stored" {
		t.Errorf("after storing a function value = %q, want stored", parseH.text())
	}
}
