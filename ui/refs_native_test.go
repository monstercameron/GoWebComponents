//go:build !js || !wasm

package ui_test

import (
	"testing"

	"github.com/monstercameron/GoWebComponents/v4/internal/platform/mockdom"
	"github.com/monstercameron/GoWebComponents/v4/internal/runtime"
)

// recordingSink is a DOMRefSink that records each (de)attachment as a bool:
// true when it receives a live node, false when cleared to nil.
type recordingSink struct {
	node    runtime.DOMNode
	history []bool
}

func (parseS *recordingSink) SetDOMNode(parseNode runtime.DOMNode) {
	parseS.node = parseNode
	parseS.history = append(parseS.history, parseNode != nil && !runtime.IsDOMNodeNull(parseNode))
}

func (parseS *recordingSink) attached() bool {
	return parseS.node != nil && !runtime.IsDOMNodeNull(parseS.node)
}

// Ported from React's ReactFiberRefs / ref attach-detach tests, adapted to GWC's
// sink-based DOM refs (a DOMRefSink carried under DOMRefKey receives the live node
// on placement and nil on deletion).
func TestDOMRefAttachesOnMountAndClearsOnUnmount(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")
	parseSink := &recordingSink{}

	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{},
		runtime.CreateElement("div", map[string]any{runtime.DOMRefKey: parseSink}, "x")))
	if !parseSink.attached() {
		t.Fatalf("after mount: ref not attached (history=%v)", parseSink.history)
	}
	// The attached node must be the actual host element.
	if parseNode, parseOk := parseSink.node.(*mockdom.MockDOMNode); !parseOk || parseNode.Tag != "div" {
		t.Errorf("attached node is not the <div> host element: %+v", parseSink.node)
	}

	rt.RenderInto(root, runtime.CreateElement("section", map[string]any{}))
	if parseSink.attached() {
		t.Errorf("after unmount: ref still attached (history=%v)", parseSink.history)
	}
	if len(parseSink.history) < 2 || parseSink.history[0] != true || parseSink.history[len(parseSink.history)-1] != false {
		t.Errorf("ref history = %v, want attach...detach", parseSink.history)
	}
}

// Detaching before attaching across a key-change remount: the runtime processes
// deletions before placements, so a ref moved to a freshly-keyed element observes
// the old element detach (nil) before the new element attaches.
func TestDOMRefDetachesBeforeAttachOnKeyChange(t *testing.T) {
	adapter := mockdom.NewMockDOMAdapter()
	rt := runtime.NewRuntime(runtime.Config{DOMAdapter: adapter, Reset: true})
	root := adapter.CreateElement("div")
	parseSink := &recordingSink{}

	render := func(parseKey string) {
		rt.RenderInto(root, runtime.CreateElement("section", map[string]any{},
			runtime.CreateElement("div", map[string]any{"key": parseKey, runtime.DOMRefKey: parseSink}, parseKey)))
	}

	render("a")
	if !parseSink.attached() {
		t.Fatalf("mount key=a: not attached (history=%v)", parseSink.history)
	}
	parseFirstNode := parseSink.node

	render("b") // different key -> old <div> deleted, new <div> placed
	if !parseSink.attached() {
		t.Errorf("after remount key=b: ref should be attached to the new element (history=%v)", parseSink.history)
	}
	if parseSink.node == parseFirstNode {
		t.Errorf("ref should point at the new element after key change, still points at old")
	}
	// History must contain a detach (false) between the two attaches.
	parseSawDetach := false
	for parseI := 1; parseI < len(parseSink.history); parseI++ {
		if !parseSink.history[parseI] {
			parseSawDetach = true
		}
	}
	if !parseSawDetach {
		t.Errorf("expected a detach (nil) across the key-change remount, history=%v", parseSink.history)
	}
}
