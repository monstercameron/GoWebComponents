package runtime

import "testing"

func TestRuntime_InitializationWithoutAtomRegistry(t *testing.T) {
	scheduler := newTestScheduler()
	adapter := newTestDOMAdapter()

	rt := &Runtime{
		domAdapter:   adapter,
		scheduler:    scheduler,
		deletions:    make([]*Fiber, 0),
		atomRegistry: nil, // No atom registry
	}

	if rt.domAdapter == nil {
		t.Error("Expected DOM adapter to be set")
	}
}

func TestFiber_NilAlternate(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	fiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: nil,
		dirty:     true,
	}

	rt.performUnitOfWork(fiber)

	if fiber.dom == nil {
		t.Error("Expected DOM to be created even without alternate")
	}
}

func TestReconcileChildren_EmptyChildren(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	// Empty children array
	rt.reconcileChildren(parent, []interface{}{})

	if parent.child != nil {
		t.Error("Expected no children when reconciling empty array")
	}
}

func TestReconcileChildren_OnlyDeletions(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	oldChild := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: oldChild,
		},
	}

	// No new children - all old should be deleted
	rt.reconcileChildren(parent, []interface{}{})

	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}
}

func TestPerformUnitOfWork_RootWithoutChildren(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	fiber := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]interface{}), // No children property
		dirty:  true,
	}

	next := rt.performUnitOfWork(fiber)

	// Should complete without error
	if fiber.child != nil {
		t.Error("Expected no children for root without children prop")
	}

	if next != nil {
		t.Error("Expected no next unit of work")
	}
}

func TestPerformUnitOfWork_HostComponentNoChildren(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	fiber := &Fiber{
		typeOf: "input",
		props:  map[string]interface{}{"type": "text"}, // No children
		dirty:  true,
	}

	rt.performUnitOfWork(fiber)

	if fiber.dom == nil {
		t.Error("Expected DOM to be created")
	}

	if fiber.child != nil {
		t.Error("Expected no children for self-closing element")
	}
}

func TestPerformUnitOfWork_FunctionComponentReturnsNil(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	componentFn := func(props map[string]interface{}) *Element {
		return nil // Component returns nothing
	}

	fiber := &Fiber{
		typeOf: componentFn,
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	rt.performUnitOfWork(fiber)

	// Should not crash with nil element
	if fiber.child != nil {
		t.Error("Expected no child when component returns nil")
	}
}

func TestCommitWork_NilParentDOM(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	// Function component (no DOM)
	parent := &Fiber{
		typeOf: func(p map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
	}

	child := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		dom:       adapter.CreateElement("div"),
		parent:    parent,
		effectTag: "PLACEMENT",
	}

	// Should handle case where parent has no DOM (function component)
	rt.commitWork(child)
}

func TestCommitWork_NilFiber(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	// Should not panic with nil
	rt.commitWork(nil)
}

func TestCommitDeletion_NilChild(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	parentDOM := adapter.CreateElement("div")

	fiber := &Fiber{
		typeOf: func(p map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
		// No dom, no child
	}

	// Should not crash
	rt.commitDeletion(fiber, parentDOM)
}

func TestUpdateDomProperties_ChildrenProperty(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	dom := adapter.CreateElement("div")

	oldProps := map[string]interface{}{
		"children": []interface{}{"old"},
	}
	newProps := map[string]interface{}{
		"children": []interface{}{"new"},
		"id":       "test",
	}

	rt.updateDomProperties(dom, oldProps, newProps)

	// children property should be skipped, not applied to DOM
	node := dom.(*testDOMNode)
	if node.attributes["children"] != "" {
		t.Error("Expected children property to be skipped")
	}

	if node.attributes["id"] != "test" {
		t.Errorf("Expected id='test', got %v", node.attributes["id"])
	}
}

func TestUpdateDomProperties_ClassVsClassName(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	dom := adapter.CreateElement("div")

	// Test className
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"className": "test-class",
	})

	node := dom.(*testDOMNode)
	if node.attributes["class"] != "test-class" {
		t.Errorf("Expected class='test-class' from className, got %v", node.attributes["class"])
	}

	// Test class
	dom2 := adapter.CreateElement("div")
	rt.updateDomProperties(dom2, map[string]interface{}{}, map[string]interface{}{
		"class": "test-class-2",
	})

	node2 := dom2.(*testDOMNode)
	if node2.attributes["class"] != "test-class-2" {
		t.Errorf("Expected class='test-class-2', got %v", node2.attributes["class"])
	}
}

func TestUpdateDomProperties_NonStringValue(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	dom := adapter.CreateElement("div")

	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"tabIndex": 5,
		"disabled": true,
	})

	node := dom.(*testDOMNode)
	if node.properties["tabIndex"] != 5 {
		t.Errorf("Expected tabIndex=5, got %v", node.properties["tabIndex"])
	}

	if node.properties["disabled"] != true {
		t.Errorf("Expected disabled=true, got %v", node.properties["disabled"])
	}
}

func TestCommitRoot_WithEffects(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	effectRan := false

	child := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("div"),
		effects: []func(){
			func() { effectRan = true },
		},
		effectTag: "PLACEMENT",
	}

	root := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("root"),
		child:  child,
	}

	child.parent = root
	rt.wipRoot = root

	rt.commitRoot()

	if !effectRan {
		t.Error("Expected effects to run during commit")
	}

	if rt.currentRoot != root {
		t.Error("Expected wipRoot to become currentRoot")
	}

	if rt.wipRoot != nil {
		t.Error("Expected wipRoot to be nil after commit")
	}
}

func TestPerformUnitOfWork_FunctionComponentAttrsType(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	componentFn := func(attrs Attrs) *Element {
		if attrs["testProp"] != "testValue" {
			return CreateElement("div", map[string]interface{}{"id": "error"})
		}
		return CreateElement("div", map[string]interface{}{"id": "success"})
	}

	fiber := &Fiber{
		typeOf: componentFn,
		props:  map[string]interface{}{"testProp": "testValue"},
		dirty:  true,
	}

	rt.performUnitOfWork(fiber)

	if fiber.child == nil {
		t.Fatal("Expected child element")
	}

	if fiber.child.props["id"] != "success" {
		t.Error("Expected component with Attrs to receive props correctly")
	}
}

func TestIsSameType_DifferentTypes(t *testing.T) {
	if isSameType(123, "string") {
		t.Error("Expected int and string to not match")
	}

	if isSameType(123, 456) {
		t.Error("Expected different ints to not match via DeepEqual")
	}
}

func TestReconcileChildren_NonElementInArray(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: adapter,
		Scheduler:  scheduler,
	})

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	// Mix of valid elements and non-elements
	rt.reconcileChildren(parent, []interface{}{
		CreateElement("span", nil),
		"not an element", // Should be skipped
		123,              // Should be skipped
	})

	if parent.child == nil {
		t.Fatal("Expected at least one child")
	}

	if parent.child.typeOf != "span" {
		t.Errorf("Expected first valid element to become child, got %v", parent.child.typeOf)
	}
}
