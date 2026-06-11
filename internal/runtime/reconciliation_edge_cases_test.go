package runtime

import (
	"testing"
)

// ============================================================================
// Reconciliation Edge Cases and Safety Tests - 50 tests
// ============================================================================

func TestReconcileChildren_NilElements(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseRt.reconcileChildren(parseWipFiber, nil)

	if parseWipFiber.child != nil {
		parseT.Error("Expected child to be nil with nil elements")
	}
}

func TestReconcileChildren_EmptyElements(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{})

	if parseWipFiber.child != nil {
		parseT.Error("Expected child to be nil with empty elements")
	}
}

func TestReconcileChildren_SingleElement(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created")
	}
	if parseWipFiber.child.effectTag != effectTagPlacement {
		parseT.Error("Expected PLACEMENT for new element")
	}
}

func TestReconcileChildren_SingleNilElement(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{nil})

	// Nil elements should be skipped
	if parseWipFiber.child != nil {
		parseT.Error("Expected nil elements to be skipped")
	}
}

func TestReconcileChildren_MixedNilAndValid(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{
		nil,
		&Element{Type: "span"},
		nil,
		&Element{Type: "p"},
		nil,
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	// Count children - nils should be skipped
	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 2 {
		parseT.Errorf("Expected 2 children (skipping nils), got %d", parseCount)
	}
}

func TestReconcileChildren_OldFiberWithoutSibling(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldFiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    parseAdapter.CreateElement("span"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span", Props: map[string]interface{}{"id": "new"}}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created")
	}
	if parseWipFiber.child.effectTag != effectTagUpdate {
		parseT.Error("Expected UPDATE effectTag for matching type")
	}
}

func TestReconcileChildren_OldFiberNilDOM(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldFiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    nil, // Nil DOM
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created even with nil old DOM")
	}
}

func TestReconcileChildren_TypeChange(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseAdapter.CreateElement("div"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	// Change from div to span
	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created")
	}
	if parseWipFiber.child.effectTag != effectTagPlacement {
		parseT.Error("Expected PLACEMENT for type change")
	}
	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}
}

func TestReconcileChildren_SameTypeUpdate(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"id": "old"},
		dom:    parseAdapter.CreateElement("div"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "div", Props: map[string]interface{}{"id": "new"}}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created")
	}
	if parseWipFiber.child.effectTag != effectTagUpdate {
		parseT.Error("Expected UPDATE for same type")
	}
	if parseWipFiber.child.props["id"] != "new" {
		parseT.Error("Expected new props")
	}
}

func TestReconcileChildren_KeyedElementsReorder(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Old: [a, b, c]
	parseOldC := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "c"}, dom: parseAdapter.CreateElement("div")}
	parseOldB := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "b"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldC}
	parseOldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldA},
	}

	// New: [c, a, b]
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected children to be created")
	}
}

func TestReconcileChildren_KeyedElementRemoval(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldB := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "b"}, dom: parseAdapter.CreateElement("div")}
	parseOldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldA},
	}

	// Only keep 'a', remove 'b'
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}
}

func TestReconcileChildren_KeyedElementAddition(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: parseAdapter.CreateElement("div")}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldA},
	}

	// Add 'b' after 'a'
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected children")
	}
	if parseWipFiber.child.sibling == nil {
		parseT.Fatal("Expected second child")
	}
	if parseWipFiber.child.sibling.props["key"] != "b" {
		parseT.Error("Expected second child key 'b'")
	}
	if parseWipFiber.child.sibling.effectTag != effectTagPlacement {
		parseT.Error("Expected PLACEMENT for new element")
	}
}

func TestReconcileChildren_AllElementsRemoved(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldB := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div")}
	parseOldA := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldA},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{})

	if parseWipFiber.child != nil {
		parseT.Error("Expected no children")
	}
	if len(parseRt.deletions) != 2 {
		parseT.Errorf("Expected 2 deletions, got %d", len(parseRt.deletions))
	}
}

func TestReconcileChildren_NoOldChildrenAddNew(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: nil}, // No old children
	}

	parseElements := []interface{}{
		&Element{Type: "span"},
		&Element{Type: "p"},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		if parseChild.effectTag != effectTagPlacement {
			parseT.Error("Expected PLACEMENT for all new elements")
		}
		parseChild = parseChild.sibling
	}

	if parseCount != 2 {
		parseT.Errorf("Expected 2 children, got %d", parseCount)
	}
}

func TestReconcileChildren_FunctionComponent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseComponent := func(parseProps map[string]interface{}) *Element {
		return &Element{Type: "div"}
	}

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{
		&Element{Type: parseComponent},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child for function component")
	}
}

func TestReconcileChildren_DeepNesting(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Create deeply nested structure
	var parseCreateElement func(parseDepth int) *Element
	parseCreateElement = func(parseDepth2 int) *Element {
		if parseDepth2 == 0 {
			return &Element{Type: "span"}
		}
		parseChild := parseCreateElement(parseDepth2 - 1)
		return &Element{Type: "div", Props: map[string]interface{}{"children": []interface{}{parseChild}}}
	}

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{parseCreateElement(10)}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
}

func TestReconcileChildren_ManyChildren(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := make([]interface{}, 100)
	for parseI := 0; parseI < 100; parseI++ {
		parseElements[parseI] = &Element{Type: "span"}
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 100 {
		parseT.Errorf("Expected 100 children, got %d", parseCount)
	}
}

func TestReconcileChildren_FragmentFlattening(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseFragment := &Element{
		Type: "FRAGMENT",
		Props: map[string]interface{}{
			"children": []interface{}{
				&Element{Type: "span"},
				&Element{Type: "p"},
			},
		},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{parseFragment})

	// Fragments should be flattened
	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 2 {
		parseT.Errorf("Expected 2 children from flattened fragment, got %d", parseCount)
	}
}

func TestReconcileChildren_OldFiberChainLongerThanNew(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Old has 5 children
	parseOld5 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div")}
	parseOld4 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div"), sibling: parseOld5}
	parseOld3 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div"), sibling: parseOld4}
	parseOld2 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div"), sibling: parseOld3}
	parseOld1 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div"), sibling: parseOld2}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOld1},
	}

	// New has only 2 children
	parseElements := []interface{}{
		&Element{Type: "div"},
		&Element{Type: "div"},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	// Should delete 3 fibers
	if len(parseRt.deletions) != 3 {
		parseT.Errorf("Expected 3 deletions, got %d", len(parseRt.deletions))
	}

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 2 {
		parseT.Errorf("Expected 2 children, got %d", parseCount)
	}
}

func TestReconcileChildren_NewLongerThanOld(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Old has 2 children
	parseOld2 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div")}
	parseOld1 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div"), sibling: parseOld2}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOld1},
	}

	// New has 5 children
	parseElements := []interface{}{
		&Element{Type: "div"},
		&Element{Type: "div"},
		&Element{Type: "div"},
		&Element{Type: "div"},
		&Element{Type: "div"},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parsePlacementCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		if parseChild.effectTag == effectTagPlacement {
			parsePlacementCount++
		}
		parseChild = parseChild.sibling
	}

	if parseCount != 5 {
		parseT.Errorf("Expected 5 children, got %d", parseCount)
	}
	if parsePlacementCount != 3 {
		parseT.Errorf("Expected 3 PLACEMENT tags for new elements, got %d", parsePlacementCount)
	}
}

func TestReconcileChildren_ElementWithNilProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span", Props: nil}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created")
	}
}

func TestReconcileChildren_ElementWithEmptyProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span", Props: map[string]interface{}{}}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created")
	}
}

func TestReconcileChildren_SameFunctionComponent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseComponent := func(parseProps map[string]interface{}) *Element {
		return &Element{Type: "div"}
	}

	parseOldFiber := &Fiber{
		typeOf: parseComponent,
		props:  make(map[string]interface{}),
		dom:    nil, // Function components don't have DOM
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: parseComponent, Props: map[string]interface{}{"id": "new"}}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.effectTag != effectTagUpdate {
		parseT.Error("Expected UPDATE for same function component")
	}
}

func TestReconcileChildren_DifferentFunctionComponents(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseComponent1 := func(parseProps map[string]interface{}) *Element {
		return &Element{Type: "div"}
	}
	parseComponent2 := func(parseProps2 map[string]interface{}) *Element {
		return &Element{Type: "span"}
	}

	parseOldFiber := &Fiber{
		typeOf: parseComponent1,
		props:  make(map[string]interface{}),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: parseComponent2}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	// Different functions should cause replacement
	if parseWipFiber.child.effectTag != effectTagPlacement {
		parseT.Error("Expected PLACEMENT for different function component")
	}
}

func TestReconcileChildren_ParentReference(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "span"},
		&Element{Type: "p"},
	})

	// Verify all children have parent reference
	parseChild := parseWipFiber.child
	for parseChild != nil {
		if parseChild.parent != parseWipFiber {
			parseT.Error("Expected child to have parent reference")
		}
		parseChild = parseChild.sibling
	}
}

func TestReconcileChildren_AlternateReference(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseAdapter.CreateElement("div"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "div"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.alternate != parseOldFiber {
		parseT.Error("Expected child to have alternate reference to old fiber")
	}
}

func TestReconcileChildren_DOMReuseOnUpdate(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldDOM := parseAdapter.CreateElement("div")
	parseOldFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"id": "old"},
		dom:    parseOldDOM,
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"id": "new"}},
	})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.dom != parseOldDOM {
		parseT.Error("Expected DOM to be reused on UPDATE")
	}
}

func TestReconcileChildren_NoAlternateReference(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: nil, // No alternate
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.alternate != nil {
		parseT.Error("Expected no alternate when parent has no alternate")
	}
}

func TestReconcileChildren_DirtyFlagOnNew(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	// Child should be marked dirty on creation
	if !parseWipFiber.child.dirty {
		parseT.Error("Expected new child to be marked dirty")
	}
}

func TestReconcileChildren_PropsDifferTriggerUpdate(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"className": "old"},
		dom:    parseAdapter.CreateElement("div"),
		dirty:  false,
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"className": "new"}},
	})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if !parseWipFiber.child.dirty {
		parseT.Error("Expected dirty flag when props differ")
	}
}

func TestReconcileChildren_EventHandlerAlwaysDirty(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseHandler := func() {}
	parseOldFiber := &Fiber{
		typeOf: "button",
		props:  map[string]interface{}{"onclick": parseHandler},
		dom:    parseAdapter.CreateElement("button"),
		dirty:  false,
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "button", Props: map[string]interface{}{"onclick": parseHandler, "id": "new"}},
	})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	// Event handlers should always trigger dirty flag
	if !parseWipFiber.child.dirty {
		parseT.Error("Expected dirty flag for event handlers")
	}
}

func TestReconcileChildren_MultipleFragments(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{
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

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 3 {
		parseT.Errorf("Expected 3 children from flattened fragments, got %d", parseCount)
	}
}

func TestReconcileChildren_NestedFragments(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{
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

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 2 {
		parseT.Errorf("Expected 2 children from nested flattened fragments, got %d", parseCount)
	}
}

func TestReconcileChildren_EmptyFragment(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{
		&Element{Type: "FRAGMENT", Props: map[string]interface{}{}},
		&Element{Type: "span"},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 1 {
		parseT.Errorf("Expected 1 child (empty fragment should add nothing), got %d", parseCount)
	}
}

func TestReconcileChildren_FragmentWithNilChildren(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{
		&Element{Type: "FRAGMENT", Props: map[string]interface{}{"children": nil}},
		&Element{Type: "span"},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 1 {
		parseT.Errorf("Expected 1 child (fragment with nil children), got %d", parseCount)
	}
}

func TestReconcileChildren_AlternateWithDirtyFlag(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseAdapter.CreateElement("div"),
		dirty:  true, // Already dirty
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "div"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	// Should propagate dirty flag
	if !parseWipFiber.child.dirty {
		parseT.Error("Expected dirty flag to be propagated from old fiber")
	}
}

func TestReconcileChildren_SiblingChainIntegrity(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseElements := []interface{}{
		&Element{Type: "span"},
		&Element{Type: "p"},
		&Element{Type: "div"},
		&Element{Type: "section"},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	// Verify sibling chain
	parseSiblings := make([]*Fiber, 0)
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseSiblings = append(parseSiblings, parseChild)
		parseChild = parseChild.sibling
	}

	if len(parseSiblings) != 4 {
		parseT.Errorf("Expected 4 siblings in chain, got %d", len(parseSiblings))
	}

	// Verify last sibling has nil sibling
	if parseSiblings[3].sibling != nil {
		parseT.Error("Expected last sibling to have nil sibling")
	}
}

func TestReconcileChildren_NoAlternateNoOldFiber(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: nil,
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{&Element{Type: "span"}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.effectTag != effectTagPlacement {
		parseT.Error("Expected PLACEMENT when no old fiber exists")
	}
}

func TestReconcileChildren_ComplexReordering(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Old: [a, b, c, d, e]
	parseOldE := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "e"}, dom: parseAdapter.CreateElement("div")}
	parseOldD := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "d"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldE}
	parseOldC := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "c"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldD}
	parseOldB := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "b"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldC}
	parseOldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldA},
	}

	// New: [e, c, a, f, b] (removed d, added f, reordered)
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "e"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "f"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	// Count children
	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 5 {
		parseT.Errorf("Expected 5 children, got %d", parseCount)
	}
}

func TestReconcileChildren_ZeroChildrenToMany(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: nil},
	}

	parseElements := make([]interface{}, 50)
	for parseI := 0; parseI < 50; parseI++ {
		parseElements[parseI] = &Element{Type: "span"}
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 50 {
		parseT.Errorf("Expected 50 children, got %d", parseCount)
	}
}

func TestReconcileChildren_ManyChildrenToZero(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Create 50 old children
	var parseFirstChild *Fiber
	var parsePrevChild *Fiber
	for parseI := 0; parseI < 50; parseI++ {
		parseChild := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div")}
		if parseI == 0 {
			parseFirstChild = parseChild
		} else {
			parsePrevChild.sibling = parseChild
		}
		parsePrevChild = parseChild
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseFirstChild},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{})

	if parseWipFiber.child != nil {
		parseT.Error("Expected no children")
	}
	if len(parseRt.deletions) != 50 {
		parseT.Errorf("Expected 50 deletions, got %d", len(parseRt.deletions))
	}
}

func TestReconcileChildren_AlternatingUpdatesAndPlacements(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOld2 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div")}
	parseOld1 := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("div"), sibling: parseOld2}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOld1},
	}

	// Keep first two, add new ones
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"id": "new1"}},
		&Element{Type: "div", Props: map[string]interface{}{"id": "new2"}},
		&Element{Type: "span"},
		&Element{Type: "p"},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseUpdateCount := 0
	parsePlacementCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		switch parseChild.effectTag {
		case effectTagUpdate:
			parseUpdateCount++
		case effectTagPlacement:
			parsePlacementCount++
		}
		parseChild = parseChild.sibling
	}

	if parseUpdateCount != 2 {
		parseT.Errorf("Expected 2 UPDATE tags, got %d", parseUpdateCount)
	}
	if parsePlacementCount != 2 {
		parseT.Errorf("Expected 2 PLACEMENT tags, got %d", parsePlacementCount)
	}
}
