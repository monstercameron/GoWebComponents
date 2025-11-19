package runtime

import (
	"testing"
)

// ============================================================================
// Additional Advanced Reconciliation Tests - 8 tests
// ============================================================================

func TestReconcileChildren_TextNodeCreation(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	// TEXT_ELEMENT type for text nodes
	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "TEXT_ELEMENT", Props: map[string]interface{}{"nodeValue": "Hello"}},
	})

	if wipFiber.child == nil {
		t.Fatal("Expected text node child")
	}
	if wipFiber.child.typeOf != "TEXT_ELEMENT" {
		t.Error("Expected TEXT_ELEMENT type")
	}
	if wipFiber.child.props["nodeValue"] != "Hello" {
		t.Error("Expected nodeValue Hello")
	}
}

func TestReconcileChildren_TextNodeUpdate(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldText := &Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]interface{}{"nodeValue": "Old"},
		dom:    adapter.CreateTextNode("Old"),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldText},
	}

	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "TEXT_ELEMENT", Props: map[string]interface{}{"nodeValue": "New"}},
	})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.effectTag != "UPDATE" {
		t.Error("Expected UPDATE for text node")
	}
	if wipFiber.child.props["nodeValue"] != "New" {
		t.Error("Expected new nodeValue")
	}
}

func TestReconcileChildren_ElementToTextNodeChange(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldElement := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("span"),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldElement},
	}

	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "TEXT_ELEMENT", Props: map[string]interface{}{"nodeValue": "Text"}},
	})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.effectTag != "PLACEMENT" {
		t.Error("Expected PLACEMENT for type change from element to text")
	}
	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}
}

func TestReconcileChildren_TextNodeToElementChange(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldText := &Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]interface{}{"nodeValue": "Text"},
		dom:    adapter.CreateTextNode("Text"),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldText},
	}

	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "span"},
	})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.effectTag != "PLACEMENT" {
		t.Error("Expected PLACEMENT for type change from text to element")
	}
	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}
}

func TestReconcileChildren_KeyedReorderBackward(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Old: [1, 2, 3, 4, 5]
	old5 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "5"}, dom: adapter.CreateElement("div")}
	old4 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "4"}, dom: adapter.CreateElement("div"), sibling: old5}
	old3 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "3"}, dom: adapter.CreateElement("div"), sibling: old4}
	old2 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "2"}, dom: adapter.CreateElement("div"), sibling: old3}
	old1 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "1"}, dom: adapter.CreateElement("div"), sibling: old2}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: old1},
	}

	// New: [5, 4, 3, 2, 1] (reverse order)
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "5"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "4"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "3"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "2"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "1"}},
	}

	rt.reconcileChildren(wipFiber, elements)

	// Verify children exist
	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 5 {
		t.Errorf("Expected 5 children, got %d", count)
	}
}

func TestReconcileChildren_KeyedInsertMiddle(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Old: [a, c]
	oldC := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "c"}, dom: adapter.CreateElement("div")}
	oldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: adapter.CreateElement("div"), sibling: oldC}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldA},
	}

	// New: [a, b, c] (insert b in middle)
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		if count == 2 && child.props["key"] != "b" {
			t.Error("Expected second child to be 'b'")
		}
		child = child.sibling
	}

	if count != 3 {
		t.Errorf("Expected 3 children, got %d", count)
	}
}

func TestReconcileChildren_KeyedRemoveMiddle(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Old: [a, b, c]
	oldC := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "c"}, dom: adapter.CreateElement("div")}
	oldB := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "b"}, dom: adapter.CreateElement("div"), sibling: oldC}
	oldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: adapter.CreateElement("div"), sibling: oldB}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldA},
	}

	// New: [a, c] (remove b from middle)
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 2 {
		t.Errorf("Expected 2 children, got %d", count)
	}
	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}
}

func TestReconcileChildren_LongChainNoKeys(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Create old chain of 20 elements
	var firstOld *Fiber
	var prevOld *Fiber
	for i := 0; i < 20; i++ {
		old := &Fiber{
			typeOf: "div",
			props:  make(map[string]interface{}),
			dom:    adapter.CreateElement("div"),
		}
		if i == 0 {
			firstOld = old
		} else {
			prevOld.sibling = old
		}
		prevOld = old
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: firstOld},
	}

	// Create new elements (same count, no keys)
	elements := make([]interface{}, 20)
	for i := 0; i < 20; i++ {
		// Add a prop to force update
		elements[i] = &Element{Type: "div", Props: map[string]interface{}{"id": "new"}}
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		if child.effectTag != "UPDATE" {
			t.Errorf("Expected UPDATE for position %d, got %s", count, child.effectTag)
		}
		child = child.sibling
	}

	if count != 20 {
		t.Errorf("Expected 20 children, got %d", count)
	}
}
