//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/internal/runtime"
)

func mockFirstChild(parseA *mockdom.MockDOMAdapter, parseRoot runtime.DOMNode) *mockdom.MockDOMNode {
	parseKids := parseA.GetChildren(parseRoot)
	if len(parseKids) == 0 {
		return nil
	}
	parseNode, _ := parseKids[0].(*mockdom.MockDOMNode)
	return parseNode
}

// TestReconcileTextUpdateInPlace: re-rendering an element with changed text must
// reuse the host node and update its content rather than recreate it.
func TestReconcileTextUpdateInPlace(t *testing.T) {
	parseA := mockdom.NewMockDOMAdapter()
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
	parseRoot := parseA.CreateElement("div")

	parseRT.RenderInto(parseRoot, runtime.CreateElement("p", map[string]any{}, "hello"))
	parseFirst := mockFirstChild(parseA, parseRoot)
	if parseFirst == nil || parseFirst.TextContent != "hello" {
		t.Fatalf("mount: expected <p>hello, got %+v", parseFirst)
	}
	parseFirstID := parseFirst.ID

	parseRT.RenderInto(parseRoot, runtime.CreateElement("p", map[string]any{}, "world"))
	parseSecond := mockFirstChild(parseA, parseRoot)
	if parseSecond == nil {
		t.Fatal("update: <p> disappeared")
	}
	if parseSecond.ID != parseFirstID {
		t.Errorf("text update recreated the node (id %d -> %d)", parseFirstID, parseSecond.ID)
	}
	if parseSecond.TextContent != "world" {
		t.Errorf("text not updated in place: got %q", parseSecond.TextContent)
	}
}

// TestReconcileConditionalUnmountAndMount: removing a child unmounts exactly that
// node while reusing the parent; adding it back re-mounts under the same parent.
func TestReconcileConditionalUnmountAndMount(t *testing.T) {
	parseA := mockdom.NewMockDOMAdapter()
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
	parseRoot := parseA.CreateElement("div")

	withChild := func() *runtime.Element {
		return runtime.CreateElement("div", map[string]any{},
			runtime.CreateElement("span", map[string]any{}, "child"))
	}
	withoutChild := func() *runtime.Element {
		return runtime.CreateElement("div", map[string]any{})
	}

	parseRT.RenderInto(parseRoot, withChild())
	parseOuter := mockFirstChild(parseA, parseRoot)
	if parseOuter == nil || len(parseA.GetChildren(parseOuter)) != 1 {
		t.Fatalf("mount: expected outer with 1 child, got %+v", parseOuter)
	}
	parseOuterID := parseOuter.ID

	parseRT.RenderInto(parseRoot, withoutChild())
	parseOuter2 := mockFirstChild(parseA, parseRoot)
	if parseOuter2.ID != parseOuterID {
		t.Errorf("outer recreated on unmount (id %d -> %d)", parseOuterID, parseOuter2.ID)
	}
	if parseN := len(parseA.GetChildren(parseOuter2)); parseN != 0 {
		t.Errorf("child not unmounted: outer still has %d children", parseN)
	}

	parseRT.RenderInto(parseRoot, withChild())
	parseOuter3 := mockFirstChild(parseA, parseRoot)
	if parseN := len(parseA.GetChildren(parseOuter3)); parseN != 1 {
		t.Errorf("child not re-mounted: outer has %d children", parseN)
	}
}

// TestReconcileTagChangeReplacesNode: changing the element type at a position
// replaces the host node (new identity) rather than mutating the tag in place.
func TestReconcileTagChangeReplacesNode(t *testing.T) {
	parseA := mockdom.NewMockDOMAdapter()
	parseRT := runtime.NewRuntime(runtime.Config{DOMAdapter: parseA, Reset: true})
	parseRoot := parseA.CreateElement("div")

	parseRT.RenderInto(parseRoot, runtime.CreateElement("span", map[string]any{}, "x"))
	parseBefore := mockFirstChild(parseA, parseRoot)
	if parseBefore == nil || parseBefore.Tag != "span" {
		t.Fatalf("mount: expected <span>, got %+v", parseBefore)
	}

	parseRT.RenderInto(parseRoot, runtime.CreateElement("section", map[string]any{}, "x"))
	parseAfter := mockFirstChild(parseA, parseRoot)
	if parseAfter == nil || parseAfter.Tag != "section" {
		t.Fatalf("update: expected <section>, got %+v", parseAfter)
	}
	if parseAfter.ID == parseBefore.ID {
		t.Errorf("tag change reused the node (id %d) instead of replacing it", parseAfter.ID)
	}
}
