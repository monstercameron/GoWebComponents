//go:build !js || !wasm

package ui_test

import (
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v6/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v6/internal/runtime"
)

// Ported from React's ReactEffectOrdering-test.js ("unmounts on deletion are
// fired in parent -> child order"). GWC has a single GoUseEffect (no separate
// layout-effect tier at the runtime level), so we assert the cleanup ordering
// that React guarantees for deletion: a parent's cleanup runs before its
// child's. (Architectural note: GWC runs effect *setup* top-down parent->child,
// where React runs setup child->parent; this test pins the cleanup order, which
// matches React.)
func TestEffectCleanupParentBeforeChildOnDeletion(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseLog []string
	parseChild := func() *runtime.Element {
		runtime.GoUseEffect(func() func() {
			return func() { parseLog = append(parseLog, "cleanup child") }
		})
		return runtime.CreateElement("span", map[string]any{}, "child")
	}
	parseParent := func() *runtime.Element {
		runtime.GoUseEffect(func() func() {
			return func() { parseLog = append(parseLog, "cleanup parent") }
		})
		return runtime.CreateElement("div", map[string]any{}, runtime.CreateElement(parseChild, map[string]any{}))
	}

	// Mount the parent/child tree under a host wrapper.
	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{},
		runtime.CreateElement(parseParent, map[string]any{})))
	parseLog = nil

	// Delete the whole subtree by re-rendering the wrapper empty.
	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{}))

	parseGot := strings.Join(parseLog, ",")
	if parseGot != "cleanup parent,cleanup child" {
		t.Errorf("deletion cleanup order = %q, want parent before child", parseGot)
	}
}

// TestEffectCleanupOrderAcrossSiblings: sibling components each clean up exactly
// once on deletion, in document order.
func TestEffectCleanupOrderAcrossSiblings(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")

	var parseLog []string
	makeLeaf := func(parseName string) func() *runtime.Element {
		return func() *runtime.Element {
			runtime.GoUseEffect(func() func() {
				return func() { parseLog = append(parseLog, "cleanup "+parseName) }
			})
			return runtime.CreateElement("span", map[string]any{}, parseName)
		}
	}

	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{},
		runtime.CreateElement(makeLeaf("a"), map[string]any{}),
		runtime.CreateElement(makeLeaf("b"), map[string]any{}),
		runtime.CreateElement(makeLeaf("c"), map[string]any{})))
	parseLog = nil

	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{}))

	parseGot := strings.Join(parseLog, ",")
	if parseGot != "cleanup a,cleanup b,cleanup c" {
		t.Errorf("sibling cleanup order = %q, want a,b,c each once", parseGot)
	}
}
