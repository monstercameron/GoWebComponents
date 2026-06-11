package runtime

import "testing"

func TestRuntime_InitializationWithoutAtomRegistry(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()

	parseRt := &Runtime{
		domAdapter: parseAdapter,
	}

	if parseRt.domAdapter == nil {
		parseT.Error("Expected DOM adapter to be set")
	}
}

func TestFiber_NilAlternate(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: nil,
		dirty:     true,
	}

	parseRt.performUnitOfWork(parseFiber)

	if parseFiber.dom == nil {
		parseT.Error("Expected DOM to be created even without alternate")
	}
}

func TestReconcileChildren_EmptyChildren(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	// Empty children array
	parseRt.reconcileChildren(parseParent, []interface{}{})

	if parseParent.child != nil {
		parseT.Error("Expected no children when reconciling empty array")
	}
}

func TestReconcileChildren_OnlyDeletions(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseOldChild := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseParent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: parseOldChild,
		},
	}

	// No new children - all old should be deleted
	parseRt.reconcileChildren(parseParent, []interface{}{})

	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}
}

func TestPerformUnitOfWork_RootWithoutChildren(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseFiber := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]interface{}), // No children property
		dirty:  true,
	}

	parseNext := parseRt.performUnitOfWork(parseFiber)

	// Should complete without error
	if parseFiber.child != nil {
		parseT.Error("Expected no children for root without children prop")
	}

	if parseNext != nil {
		parseT.Error("Expected no next unit of work")
	}
}

func TestPerformUnitOfWork_HostComponentNoChildren(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseFiber := &Fiber{
		typeOf: "input",
		props:  map[string]interface{}{"type": "text"}, // No children
		dirty:  true,
	}

	parseRt.performUnitOfWork(parseFiber)

	if parseFiber.dom == nil {
		parseT.Error("Expected DOM to be created")
	}

	if parseFiber.child != nil {
		parseT.Error("Expected no children for self-closing element")
	}
}

func TestPerformUnitOfWork_FunctionComponentReturnsNil(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseComponentFn := func(parseProps map[string]interface{}) *Element {
		return nil // Component returns nothing
	}

	parseFiber := &Fiber{
		typeOf: parseComponentFn,
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	parseRt.performUnitOfWork(parseFiber)

	// Should not crash with nil element
	if parseFiber.child != nil {
		parseT.Error("Expected no child when component returns nil")
	}
}

func TestCommitWork_NilParentDOM(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	// Function component (no DOM)
	parseParent := &Fiber{
		typeOf: func(parseP map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
	}

	parseChild := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		dom:       parseAdapter.CreateElement("div"),
		parent:    parseParent,
		effectTag: effectTagPlacement,
	}

	// Should handle case where parent has no DOM (function component)
	parseRt.commitWork(parseChild, nil)
}

func TestCommitWork_NilFiber(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	// Should not panic with nil
	parseRt.commitWork(nil, nil)
}

func TestCommitDeletion_NilChild(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseParentDOM := parseAdapter.CreateElement("div")

	parseFiber := &Fiber{
		typeOf: func(parseP map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
		// No dom, no child
	}

	// Should not crash
	parseRt.commitDeletion(parseFiber, parseParentDOM)
}

func TestUpdateDomProperties_ChildrenProperty(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseDom := parseAdapter.CreateElement("div")

	parseOldProps := map[string]interface{}{
		"children": []interface{}{"old"},
	}
	parseNewProps := map[string]interface{}{
		"children": []interface{}{"new"},
		"id":       "test",
	}

	parseRt.updateDomProperties(parseDom, parseOldProps, parseNewProps)

	// children property should be skipped, not applied to DOM
	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["children"] != "" {
		parseT.Error("Expected children property to be skipped")
	}

	if parseNode.attributes["id"] != "test" {
		parseT.Errorf("Expected id='test', got %v", parseNode.attributes["id"])
	}
}

func TestUpdateDomProperties_ClassVsClassName(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseDom := parseAdapter.CreateElement("div")

	// Test className
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"className": "test-class",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["class"] != "test-class" {
		parseT.Errorf("Expected class='test-class' from className, got %v", parseNode.attributes["class"])
	}

	// Test class
	parseDom2 := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom2, map[string]interface{}{}, map[string]interface{}{
		"class": "test-class-2",
	})

	parseNode2 := parseDom2.(*testDOMNode)
	if parseNode2.attributes["class"] != "test-class-2" {
		parseT.Errorf("Expected class='test-class-2', got %v", parseNode2.attributes["class"])
	}
}

func TestUpdateDomProperties_NonStringValue(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseDom := parseAdapter.CreateElement("div")

	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"tabIndex": 5,
		"disabled": true,
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.properties["tabIndex"] != 5 {
		parseT.Errorf("Expected tabIndex=5, got %v", parseNode.properties["tabIndex"])
	}

	if parseNode.properties["disabled"] != true {
		parseT.Errorf("Expected disabled=true, got %v", parseNode.properties["disabled"])
	}
}

func TestCommitRoot_WithEffects_EdgeCase(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	isParseEffectRan := false

	parseChild := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		dom:       parseAdapter.CreateElement("div"),
		effects:   []Effect{{Fn: func() func() { isParseEffectRan = true; return nil }}},
		effectTag: effectTagPlacement,
	}

	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]interface{}),
		dom:    parseAdapter.CreateElement("root"),
		child:  parseChild,
	}

	parseChild.parent = parseRoot
	parseRt.wipRoot = parseRoot

	parseRt.commitRoot()

	if !isParseEffectRan {
		parseT.Error("Expected effects to run during commit")
	}

	if parseRt.currentRoot != parseRoot {
		parseT.Error("Expected wipRoot to become currentRoot")
	}

	if parseRt.wipRoot != nil {
		parseT.Error("Expected wipRoot to be nil after commit")
	}
}

func TestPerformUnitOfWork_FunctionComponentAttrsType(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseComponentFn := func(parseAttrs Attrs) *Element {
		if parseAttrs["testProp"] != "testValue" {
			return CreateElement("div", map[string]interface{}{"id": "error"})
		}
		return CreateElement("div", map[string]interface{}{"id": "success"})
	}

	parseFiber := &Fiber{
		typeOf: parseComponentFn,
		props:  map[string]interface{}{"testProp": "testValue"},
		dirty:  true,
	}

	parseRt.performUnitOfWork(parseFiber)

	if parseFiber.child == nil {
		parseT.Fatal("Expected child element")
	}

	if parseFiber.child.props["id"] != "success" {
		parseT.Error("Expected component with Attrs to receive props correctly")
	}
}

func TestIsSameType_DifferentTypes(parseT *testing.T) {
	if isSameType(123, "string") {
		parseT.Error("Expected int and string to not match")
	}

	if isSameType(123, 456) {
		parseT.Error("Expected different ints to not match via DeepEqual")
	}
}

func TestReconcileChildren_NonElementInArray(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseAdapter,
		Scheduler:  parseScheduler,
	})

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	// Mix of valid elements and non-elements
	parseRt.reconcileChildren(parseParent, []interface{}{
		CreateElement("span", nil),
		"not an element", // Should be skipped
		123,              // Should be skipped
	})

	if parseParent.child == nil {
		parseT.Fatal("Expected at least one child")
	}

	if parseParent.child.typeOf != "span" {
		parseT.Errorf("Expected first valid element to become child, got %v", parseParent.child.typeOf)
	}
}
