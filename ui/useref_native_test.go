//go:build !js || !wasm

package ui_test

import (
	"fmt"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// Ported from React's useRef contract tests: a ref is a stable mutable container
// across renders; mutating .Current does not trigger a re-render, and its value
// persists across re-renders driven by state.
func TestUseRefStableMutableAndDoesNotRerender(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseRefSeen *runtime.RefValue
	var parseSet func(any)
	parseRenders := 0

	rt.RenderInto(root, runtime.CreateElement(func() *runtime.Element {
		parseRenders++
		parseRef := runtime.GoUseRef(7)
		parseGet, parseSetter := runtime.GoUseState(rt, 0)
		parseSet = parseSetter
		parseRefSeen = parseRef
		return runtime.CreateElement("span", map[string]any{}, fmt.Sprint(parseGet()))
	}, map[string]any{}))

	if parseRenders != 1 {
		t.Fatalf("mount renders = %d, want 1", parseRenders)
	}
	if parseRefSeen == nil || parseRefSeen.Current != 7 {
		t.Fatalf("initial ref = %+v, want Current=7", parseRefSeen)
	}
	parseFirstRef := parseRefSeen

	// Mutating the ref must NOT cause a re-render.
	parseRefSeen.Current = 99
	if parseRenders != 1 {
		t.Errorf("mutating ref triggered a re-render: renders = %d", parseRenders)
	}

	// A state update re-renders; the ref must be the same object with its value
	// preserved (not reset to the initial 7).
	parseSet(1)
	if parseRenders != 2 {
		t.Fatalf("after state update renders = %d, want 2", parseRenders)
	}
	if parseRefSeen != parseFirstRef {
		t.Errorf("ref identity changed across re-render")
	}
	if parseRefSeen.Current != 99 {
		t.Errorf("ref value not preserved across re-render: Current = %v, want 99", parseRefSeen.Current)
	}
}
