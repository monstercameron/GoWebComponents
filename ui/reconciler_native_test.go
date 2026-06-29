//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// These tests exercise GoWebComponents' full client reconciler natively by
// driving runtime.RenderInto against the mock DOM adapter (the noop-renderer
// pattern React uses in react-noop-renderer). They assert the reconciliation
// invariants that matter for correctness — keyed identity preservation, removal,
// and insertion — which were previously only covered under the wasm browser
// lifecycle.

// reconHarness mounts a runtime backed by the mock DOM and exposes the root.
type reconHarness struct {
	adapter *mockdom.MockDOMAdapter
	rt      *runtime.Runtime
	root    runtime.DOMNode
}

func newReconHarness() *reconHarness {
	adapter := mockdom.NewMockDOMAdapter()
	return &reconHarness{
		adapter: adapter,
		rt:      runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true}),
		root:    adapter.CreateElement("div"),
	}
}

func keyedUL(parseKeys ...string) *runtime.Element {
	parseKids := make([]any, len(parseKeys))
	for parseI, parseK := range parseKeys {
		parseKids[parseI] = runtime.CreateElement("li", map[string]any{"key": parseK, "data-k": parseK}, parseK)
	}
	return runtime.CreateElement("ul", map[string]any{}, parseKids...)
}

// render commits one tree into the shared root and returns the ordered list of
// (data-k, node-ID) pairs for the <ul>'s children.
func (parseH *reconHarness) renderKeyed(parseKeys ...string) ([]string, map[string]int) {
	parseH.rt.RenderInto(parseH.root, keyedUL(parseKeys...))
	parseChildren := parseH.adapter.GetChildren(parseH.root)
	if len(parseChildren) == 0 {
		return nil, nil
	}
	parseUL := parseChildren[0]
	var parseOrder []string
	parseIDs := map[string]int{}
	for _, parseLI := range parseH.adapter.GetChildren(parseUL) {
		if parseNode, parseOk := parseLI.(*mockdom.MockDOMNode); parseOk {
			parseKey := parseNode.Attrs["data-k"]
			parseOrder = append(parseOrder, parseKey)
			parseIDs[parseKey] = parseNode.ID
		}
	}
	return parseOrder, parseIDs
}

func equalOrder(parseGot, parseWant []string) bool {
	if len(parseGot) != len(parseWant) {
		return false
	}
	for parseI := range parseGot {
		if parseGot[parseI] != parseWant[parseI] {
			return false
		}
	}
	return true
}

// TestReconcileKeyedReorderPreservesIdentity: reordering a keyed list must move
// the existing nodes, not recreate them (stable node identity across reorder).
func TestReconcileKeyedReorderPreservesIdentity(t *testing.T) {
	parseH := newReconHarness()
	_, parseBefore := parseH.renderKeyed("a", "b", "c")
	parseOrder, parseAfter := parseH.renderKeyed("c", "a", "b")

	if !equalOrder(parseOrder, []string{"c", "a", "b"}) {
		t.Fatalf("reorder produced wrong child order: %v", parseOrder)
	}
	for _, parseK := range []string{"a", "b", "c"} {
		if parseBefore[parseK] != parseAfter[parseK] {
			t.Errorf("key %q node recreated on reorder (id %d -> %d)", parseK, parseBefore[parseK], parseAfter[parseK])
		}
	}
}

// TestReconcileKeyedRemovalKeepsSurvivors: removing a middle key drops only that
// node; the surviving keys keep their identity.
func TestReconcileKeyedRemovalKeepsSurvivors(t *testing.T) {
	parseH := newReconHarness()
	_, parseBefore := parseH.renderKeyed("a", "b", "c")
	parseOrder, parseAfter := parseH.renderKeyed("a", "c")

	if !equalOrder(parseOrder, []string{"a", "c"}) {
		t.Fatalf("removal produced wrong child order: %v", parseOrder)
	}
	if _, parseStillThere := parseAfter["b"]; parseStillThere {
		t.Errorf("removed key b is still present: %v", parseOrder)
	}
	for _, parseK := range []string{"a", "c"} {
		if parseBefore[parseK] != parseAfter[parseK] {
			t.Errorf("survivor key %q recreated on removal (id %d -> %d)", parseK, parseBefore[parseK], parseAfter[parseK])
		}
	}
}

// TestReconcileKeyedInsertionKeepsExisting: inserting a new key in the middle
// adds exactly one node and preserves the existing nodes' identity.
func TestReconcileKeyedInsertionKeepsExisting(t *testing.T) {
	parseH := newReconHarness()
	_, parseBefore := parseH.renderKeyed("a", "c")
	parseOrder, parseAfter := parseH.renderKeyed("a", "b", "c")

	if !equalOrder(parseOrder, []string{"a", "b", "c"}) {
		t.Fatalf("insertion produced wrong child order: %v", parseOrder)
	}
	for _, parseK := range []string{"a", "c"} {
		if parseBefore[parseK] != parseAfter[parseK] {
			t.Errorf("existing key %q recreated on insertion (id %d -> %d)", parseK, parseBefore[parseK], parseAfter[parseK])
		}
	}
	if _, parseInserted := parseAfter["b"]; !parseInserted {
		t.Errorf("inserted key b is missing: %v", parseOrder)
	}
}

// TestReconcileMountStructure: a freshly mounted tree produces the expected
// parent/child structure in the host DOM.
func TestReconcileMountStructure(t *testing.T) {
	parseH := newReconHarness()
	parseOrder, _ := parseH.renderKeyed("x", "y")
	if !equalOrder(parseOrder, []string{"x", "y"}) {
		t.Fatalf("mount structure wrong: %v", parseOrder)
	}
}
