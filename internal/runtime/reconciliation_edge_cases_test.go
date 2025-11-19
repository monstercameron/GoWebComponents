package runtime

import (
	"testing"
)

// ============================================================================
// Reconciliation Edge Cases and Safety Tests - 50 tests
// ============================================================================

func TestReconcileChildren_NilElements(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	rt.reconcileChildren(wipFiber, nil)

	if wipFiber.child != nil {
		t.Error("Expected child to be nil with nil elements")
	}
}

func TestReconcileChildren_EmptyElements(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	rt.reconcileChildren(wipFiber, []interface{}{})

	if wipFiber.child != nil {
		t.Error("Expected child to be nil with empty elements")
	}
}

func TestReconcileChildren_SingleElement(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child to be created")
	}
	if wipFiber.child.effectTag != "PLACEMENT" {
		t.Error("Expected PLACEMENT for new element")
	}
}

func TestReconcileChildren_SingleNilElement(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	rt.reconcileChildren(wipFiber, []interface{}{nil})

	// Nil elements should be skipped
	if wipFiber.child != nil {
		t.Error("Expected nil elements to be skipped")
	}
}

func TestReconcileChildren_MixedNilAndValid(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{
		nil,
		&Element{Type: "span"},
		nil,
		&Element{Type: "p"},
		nil,
	}

	rt.reconcileChildren(wipFiber, elements)

	// Count children - nils should be skipped
	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 2 {
		t.Errorf("Expected 2 children (skipping nils), got %d", count)
	}
}

func TestReconcileChildren_OldFiberWithoutSibling(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldFiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("span"),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span", Props: map[string]interface{}{"id": "new"}}})

	if wipFiber.child == nil {
		t.Fatal("Expected child to be created")
	}
	if wipFiber.child.effectTag != "UPDATE" {
		t.Error("Expected UPDATE effectTag for matching type")
	}
}

func TestReconcileChildren_OldFiberNilDOM(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldFiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    nil, // Nil DOM
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child to be created even with nil old DOM")
	}
}

func TestReconcileChildren_TypeChange(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("div"),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	// Change from div to span
	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child to be created")
	}
	if wipFiber.child.effectTag != "PLACEMENT" {
		t.Error("Expected PLACEMENT for type change")
	}
	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}
}

func TestReconcileChildren_SameTypeUpdate(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"id": "old"},
		dom:    adapter.CreateElement("div"),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "div", Props: map[string]interface{}{"id": "new"}}})

	if wipFiber.child == nil {
		t.Fatal("Expected child to be created")
	}
	if wipFiber.child.effectTag != "UPDATE" {
		t.Error("Expected UPDATE for same type")
	}
	if wipFiber.child.props["id"] != "new" {
		t.Error("Expected new props")
	}
}

func TestReconcileChildren_KeyedElementsReorder(t *testing.T) {
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

	// New: [c, a, b]
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
	}

	rt.reconcileChildren(wipFiber, elements)

	if wipFiber.child == nil {
		t.Fatal("Expected children to be created")
	}
}

func TestReconcileChildren_KeyedElementRemoval(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldB := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "b"}, dom: adapter.CreateElement("div")}
	oldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: adapter.CreateElement("div"), sibling: oldB}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldA},
	}

	// Only keep 'a', remove 'b'
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
	}

	rt.reconcileChildren(wipFiber, elements)

	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}
}

func TestReconcileChildren_KeyedElementAddition(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: adapter.CreateElement("div")}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldA},
	}

	// Add 'b' after 'a'
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
	}

	rt.reconcileChildren(wipFiber, elements)

	if wipFiber.child == nil {
		t.Fatal("Expected children")
	}
	if wipFiber.child.sibling == nil {
		t.Fatal("Expected second child")
	}
	if wipFiber.child.sibling.props["key"] != "b" {
		t.Error("Expected second child key 'b'")
	}
	if wipFiber.child.sibling.effectTag != "PLACEMENT" {
		t.Error("Expected PLACEMENT for new element")
	}
}

func TestReconcileChildren_AllElementsRemoved(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldB := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div")}
	oldA := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div"), sibling: oldB}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldA},
	}

	rt.reconcileChildren(wipFiber, []interface{}{})

	if wipFiber.child != nil {
		t.Error("Expected no children")
	}
	if len(rt.deletions) != 2 {
		t.Errorf("Expected 2 deletions, got %d", len(rt.deletions))
	}
}

func TestReconcileChildren_NoOldChildrenAddNew(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: nil}, // No old children
	}

	elements := []interface{}{
		&Element{Type: "span"},
		&Element{Type: "p"},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		if child.effectTag != "PLACEMENT" {
			t.Error("Expected PLACEMENT for all new elements")
		}
		child = child.sibling
	}

	if count != 2 {
		t.Errorf("Expected 2 children, got %d", count)
	}
}

func TestReconcileChildren_FunctionComponent(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	component := func(props map[string]interface{}) *Element {
		return &Element{Type: "div"}
	}

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{
		&Element{Type: component},
	}

	rt.reconcileChildren(wipFiber, elements)

	if wipFiber.child == nil {
		t.Fatal("Expected child for function component")
	}
}

func TestReconcileChildren_DeepNesting(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Create deeply nested structure
	var createElement func(depth int) *Element
	createElement = func(depth int) *Element {
		if depth == 0 {
			return &Element{Type: "span"}
		}
		child := createElement(depth - 1)
		return &Element{Type: "div", Props: map[string]interface{}{"children": []interface{}{child}}}
	}

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{createElement(10)}

	rt.reconcileChildren(wipFiber, elements)

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
}

func TestReconcileChildren_ManyChildren(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := make([]interface{}, 100)
	for i := 0; i < 100; i++ {
		elements[i] = &Element{Type: "span"}
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 100 {
		t.Errorf("Expected 100 children, got %d", count)
	}
}

func TestReconcileChildren_FragmentFlattening(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	fragment := &Element{
		Type: "FRAGMENT",
		Props: map[string]interface{}{
			"children": []interface{}{
				&Element{Type: "span"},
				&Element{Type: "p"},
			},
		},
	}

	rt.reconcileChildren(wipFiber, []interface{}{fragment})

	// Fragments should be flattened
	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 2 {
		t.Errorf("Expected 2 children from flattened fragment, got %d", count)
	}
}

func TestReconcileChildren_OldFiberChainLongerThanNew(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Old has 5 children
	old5 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div")}
	old4 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div"), sibling: old5}
	old3 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div"), sibling: old4}
	old2 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div"), sibling: old3}
	old1 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div"), sibling: old2}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: old1},
	}

	// New has only 2 children
	elements := []interface{}{
		&Element{Type: "div"},
		&Element{Type: "div"},
	}

	rt.reconcileChildren(wipFiber, elements)

	// Should delete 3 fibers
	if len(rt.deletions) != 3 {
		t.Errorf("Expected 3 deletions, got %d", len(rt.deletions))
	}

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 2 {
		t.Errorf("Expected 2 children, got %d", count)
	}
}

func TestReconcileChildren_NewLongerThanOld(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Old has 2 children
	old2 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div")}
	old1 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div"), sibling: old2}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: old1},
	}

	// New has 5 children
	elements := []interface{}{
		&Element{Type: "div"},
		&Element{Type: "div"},
		&Element{Type: "div"},
		&Element{Type: "div"},
		&Element{Type: "div"},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	placementCount := 0
	child := wipFiber.child
	for child != nil {
		count++
		if child.effectTag == "PLACEMENT" {
			placementCount++
		}
		child = child.sibling
	}

	if count != 5 {
		t.Errorf("Expected 5 children, got %d", count)
	}
	if placementCount != 3 {
		t.Errorf("Expected 3 PLACEMENT tags for new elements, got %d", placementCount)
	}
}

func TestReconcileChildren_ElementWithNilProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span", Props: nil}})

	if wipFiber.child == nil {
		t.Fatal("Expected child to be created")
	}
}

func TestReconcileChildren_ElementWithEmptyProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span", Props: map[string]interface{}{}}})

	if wipFiber.child == nil {
		t.Fatal("Expected child to be created")
	}
}

func TestReconcileChildren_SameFunctionComponent(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	component := func(props map[string]interface{}) *Element {
		return &Element{Type: "div"}
	}

	oldFiber := &Fiber{
		typeOf: component,
		props:  make(map[string]interface{}),
		dom:    nil, // Function components don't have DOM
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: component, Props: map[string]interface{}{"id": "new"}}})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.effectTag != "UPDATE" {
		t.Error("Expected UPDATE for same function component")
	}
}

func TestReconcileChildren_DifferentFunctionComponents(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	component1 := func(props map[string]interface{}) *Element {
		return &Element{Type: "div"}
	}
	component2 := func(props map[string]interface{}) *Element {
		return &Element{Type: "span"}
	}

	oldFiber := &Fiber{
		typeOf: component1,
		props:  make(map[string]interface{}),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: component2}})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	// Different functions should cause replacement
	if wipFiber.child.effectTag != "PLACEMENT" {
		t.Error("Expected PLACEMENT for different function component")
	}
}

func TestReconcileChildren_ParentReference(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "span"},
		&Element{Type: "p"},
	})

	// Verify all children have parent reference
	child := wipFiber.child
	for child != nil {
		if child.parent != wipFiber {
			t.Error("Expected child to have parent reference")
		}
		child = child.sibling
	}
}

func TestReconcileChildren_AlternateReference(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("div"),
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "div"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.alternate != oldFiber {
		t.Error("Expected child to have alternate reference to old fiber")
	}
}

func TestReconcileChildren_DOMReuseOnUpdate(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldDOM := adapter.CreateElement("div")
	oldFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"id": "old"},
		dom:    oldDOM,
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"id": "new"}},
	})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.dom != oldDOM {
		t.Error("Expected DOM to be reused on UPDATE")
	}
}

func TestReconcileChildren_NoAlternateReference(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: nil, // No alternate
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.alternate != nil {
		t.Error("Expected no alternate when parent has no alternate")
	}
}

func TestReconcileChildren_DirtyFlagOnNew(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	// Child should be marked dirty on creation
	if !wipFiber.child.dirty {
		t.Error("Expected new child to be marked dirty")
	}
}

func TestReconcileChildren_PropsDifferTriggerUpdate(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"className": "old"},
		dom:    adapter.CreateElement("div"),
		dirty:  false,
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"className": "new"}},
	})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if !wipFiber.child.dirty {
		t.Error("Expected dirty flag when props differ")
	}
}

func TestReconcileChildren_EventHandlerAlwaysDirty(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	handler := func() {}
	oldFiber := &Fiber{
		typeOf: "button",
		props:  map[string]interface{}{"onclick": handler},
		dom:    adapter.CreateElement("button"),
		dirty:  false,
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{
		&Element{Type: "button", Props: map[string]interface{}{"onclick": handler, "id": "new"}},
	})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	// Event handlers should always trigger dirty flag
	if !wipFiber.child.dirty {
		t.Error("Expected dirty flag for event handlers")
	}
}

func TestReconcileChildren_MultipleFragments(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": []interface{}{
					&Element{Type: "span"},
					&Element{Type: "p"},
				},
			},
		},
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": []interface{}{
					&Element{Type: "div"},
				},
			},
		},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 3 {
		t.Errorf("Expected 3 children from flattened fragments, got %d", count)
	}
}

func TestReconcileChildren_NestedFragments(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": []interface{}{
					&Element{
						Type: "FRAGMENT",
						Props: map[string]interface{}{
							"children": []interface{}{
								&Element{Type: "span"},
							},
						},
					},
					&Element{Type: "p"},
				},
			},
		},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 2 {
		t.Errorf("Expected 2 children from nested flattened fragments, got %d", count)
	}
}

func TestReconcileChildren_EmptyFragment(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{
		&Element{Type: "FRAGMENT", Props: map[string]interface{}{}},
		&Element{Type: "span"},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 1 {
		t.Errorf("Expected 1 child (empty fragment should add nothing), got %d", count)
	}
}

func TestReconcileChildren_FragmentWithNilChildren(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{
		&Element{Type: "FRAGMENT", Props: map[string]interface{}{"children": nil}},
		&Element{Type: "span"},
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 1 {
		t.Errorf("Expected 1 child (fragment with nil children), got %d", count)
	}
}

func TestReconcileChildren_AlternateWithDirtyFlag(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	oldFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("div"),
		dirty:  true, // Already dirty
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldFiber},
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "div"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	// Should propagate dirty flag
	if !wipFiber.child.dirty {
		t.Error("Expected dirty flag to be propagated from old fiber")
	}
}

func TestReconcileChildren_SiblingChainIntegrity(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	elements := []interface{}{
		&Element{Type: "span"},
		&Element{Type: "p"},
		&Element{Type: "div"},
		&Element{Type: "section"},
	}

	rt.reconcileChildren(wipFiber, elements)

	// Verify sibling chain
	siblings := make([]*Fiber, 0)
	child := wipFiber.child
	for child != nil {
		siblings = append(siblings, child)
		child = child.sibling
	}

	if len(siblings) != 4 {
		t.Errorf("Expected 4 siblings in chain, got %d", len(siblings))
	}

	// Verify last sibling has nil sibling
	if siblings[3].sibling != nil {
		t.Error("Expected last sibling to have nil sibling")
	}
}

func TestReconcileChildren_NoAlternateNoOldFiber(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: nil,
	}

	rt.reconcileChildren(wipFiber, []interface{}{&Element{Type: "span"}})

	if wipFiber.child == nil {
		t.Fatal("Expected child")
	}
	if wipFiber.child.effectTag != "PLACEMENT" {
		t.Error("Expected PLACEMENT when no old fiber exists")
	}
}

func TestReconcileChildren_ComplexReordering(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Old: [a, b, c, d, e]
	oldE := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "e"}, dom: adapter.CreateElement("div")}
	oldD := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "d"}, dom: adapter.CreateElement("div"), sibling: oldE}
	oldC := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "c"}, dom: adapter.CreateElement("div"), sibling: oldD}
	oldB := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "b"}, dom: adapter.CreateElement("div"), sibling: oldC}
	oldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: adapter.CreateElement("div"), sibling: oldB}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: oldA},
	}

	// New: [e, c, a, f, b] (removed d, added f, reordered)
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "e"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "f"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
	}

	rt.reconcileChildren(wipFiber, elements)

	// Count children
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

func TestReconcileChildren_ZeroChildrenToMany(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: nil},
	}

	elements := make([]interface{}, 50)
	for i := 0; i < 50; i++ {
		elements[i] = &Element{Type: "span"}
	}

	rt.reconcileChildren(wipFiber, elements)

	count := 0
	child := wipFiber.child
	for child != nil {
		count++
		child = child.sibling
	}

	if count != 50 {
		t.Errorf("Expected 50 children, got %d", count)
	}
}

func TestReconcileChildren_ManyChildrenToZero(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	// Create 50 old children
	var firstChild *Fiber
	var prevChild *Fiber
	for i := 0; i < 50; i++ {
		child := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div")}
		if i == 0 {
			firstChild = child
		} else {
			prevChild.sibling = child
		}
		prevChild = child
	}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: firstChild},
	}

	rt.reconcileChildren(wipFiber, []interface{}{})

	if wipFiber.child != nil {
		t.Error("Expected no children")
	}
	if len(rt.deletions) != 50 {
		t.Errorf("Expected 50 deletions, got %d", len(rt.deletions))
	}
}

func TestReconcileChildren_AlternatingUpdatesAndPlacements(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	old2 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div")}
	old1 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: adapter.CreateElement("div"), sibling: old2}

	wipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: old1},
	}

	// Keep first two, add new ones
	elements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"id": "new1"}},
		&Element{Type: "div", Props: map[string]interface{}{"id": "new2"}},
		&Element{Type: "span"},
		&Element{Type: "p"},
	}

	rt.reconcileChildren(wipFiber, elements)

	updateCount := 0
	placementCount := 0
	child := wipFiber.child
	for child != nil {
		switch child.effectTag {
		case "UPDATE":
			updateCount++
		case "PLACEMENT":
			placementCount++
		}
		child = child.sibling
	}

	if updateCount != 2 {
		t.Errorf("Expected 2 UPDATE tags, got %d", updateCount)
	}
	if placementCount != 2 {
		t.Errorf("Expected 2 PLACEMENT tags, got %d", placementCount)
	}
}
