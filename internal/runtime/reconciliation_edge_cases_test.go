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
		props:  make(map[string]any),
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
		props:  make(map[string]any),
	}

	parseRt.reconcileChildren(parseWipFiber, []any{})

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
		props:  make(map[string]any),
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span"}})

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
		props:  make(map[string]any),
	}

	parseRt.reconcileChildren(parseWipFiber, []any{nil})

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
		props:  make(map[string]any),
	}

	parseElements := []any{
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
		props:  make(map[string]any),
		dom:    parseAdapter.CreateElement("span"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span", Props: map[string]any{"id": "new"}}})

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
		props:  make(map[string]any),
		dom:    nil, // Nil DOM
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span"}})

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
		props:  make(map[string]any),
		dom:    parseAdapter.CreateElement("div"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	// Change from div to span
	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span"}})

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
		props:  map[string]any{"id": "old"},
		dom:    parseAdapter.CreateElement("div"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "div", Props: map[string]any{"id": "new"}}})

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
	parseOldC := &Fiber{typeOf: "div", props: map[string]any{"key": "c"}, dom: parseAdapter.CreateElement("div")}
	parseOldB := &Fiber{typeOf: "div", props: map[string]any{"key": "b"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldC}
	parseOldA := &Fiber{typeOf: "div", props: map[string]any{"key": "a"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldA},
	}

	// New: [c, a, b]
	parseElements := []any{
		&Element{Type: "div", Props: map[string]any{"key": "c"}},
		&Element{Type: "div", Props: map[string]any{"key": "a"}},
		&Element{Type: "div", Props: map[string]any{"key": "b"}},
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

	parseOldB := &Fiber{typeOf: "div", props: map[string]any{"key": "b"}, dom: parseAdapter.CreateElement("div")}
	parseOldA := &Fiber{typeOf: "div", props: map[string]any{"key": "a"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldA},
	}

	// Only keep 'a', remove 'b'
	parseElements := []any{
		&Element{Type: "div", Props: map[string]any{"key": "a"}},
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

	parseOldA := &Fiber{typeOf: "div", props: map[string]any{"key": "a"}, dom: parseAdapter.CreateElement("div")}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldA},
	}

	// Add 'b' after 'a'
	parseElements := []any{
		&Element{Type: "div", Props: map[string]any{"key": "a"}},
		&Element{Type: "div", Props: map[string]any{"key": "b"}},
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

	parseOldB := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div")}
	parseOldA := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldA},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{})

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
		props:     make(map[string]any),
		alternate: &Fiber{child: nil}, // No old children
	}

	parseElements := []any{
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

	parseComponent := func(parseProps map[string]any) *Element {
		return &Element{Type: "div"}
	}

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]any),
	}

	parseElements := []any{
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
		return &Element{Type: "div", Props: map[string]any{"children": []any{parseChild}}}
	}

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]any),
	}

	parseElements := []any{parseCreateElement(10)}

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
		props:  make(map[string]any),
	}

	parseElements := make([]any, 100)
	for parseI := range 100 {
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
		props:  make(map[string]any),
	}

	parseFragment := &Element{
		Type: "FRAGMENT",
		Props: map[string]any{
			"children": []any{
				&Element{Type: "span"},
				&Element{Type: "p"},
			},
		},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{parseFragment})

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
	parseOld5 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div")}
	parseOld4 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div"), sibling: parseOld5}
	parseOld3 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div"), sibling: parseOld4}
	parseOld2 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div"), sibling: parseOld3}
	parseOld1 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div"), sibling: parseOld2}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOld1},
	}

	// New has only 2 children
	parseElements := []any{
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
	parseOld2 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div")}
	parseOld1 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div"), sibling: parseOld2}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOld1},
	}

	// New has 5 children
	parseElements := []any{
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
		props:  make(map[string]any),
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span", Props: nil}})

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
		props:  make(map[string]any),
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span", Props: map[string]any{}}})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child to be created")
	}
}

func TestReconcileChildren_SameFunctionComponent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseComponent := func(parseProps map[string]any) *Element {
		return &Element{Type: "div"}
	}

	parseOldFiber := &Fiber{
		typeOf: parseComponent,
		props:  make(map[string]any),
		dom:    nil, // Function components don't have DOM
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: parseComponent, Props: map[string]any{"id": "new"}}})

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

	parseComponent1 := func(parseProps map[string]any) *Element {
		return &Element{Type: "div"}
	}
	parseComponent2 := func(parseProps2 map[string]any) *Element {
		return &Element{Type: "span"}
	}

	parseOldFiber := &Fiber{
		typeOf: parseComponent1,
		props:  make(map[string]any),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: parseComponent2}})

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
		props:  make(map[string]any),
	}

	parseRt.reconcileChildren(parseWipFiber, []any{
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
		props:  make(map[string]any),
		dom:    parseAdapter.CreateElement("div"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "div"}})

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
		props:  map[string]any{"id": "old"},
		dom:    parseOldDOM,
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{
		&Element{Type: "div", Props: map[string]any{"id": "new"}},
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
		props:     make(map[string]any),
		alternate: nil, // No alternate
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span"}})

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
		props:  make(map[string]any),
		dirty:  true,
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span"}})

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
		props:  map[string]any{"className": "old"},
		dom:    parseAdapter.CreateElement("div"),
		dirty:  false,
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{
		&Element{Type: "div", Props: map[string]any{"className": "new"}},
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
		props:  map[string]any{"onclick": parseHandler},
		dom:    parseAdapter.CreateElement("button"),
		dirty:  false,
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{
		&Element{Type: "button", Props: map[string]any{"onclick": parseHandler, "id": "new"}},
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
		props:  make(map[string]any),
	}

	parseElements := []any{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]any{
				"children": []any{
					&Element{Type: "span"},
					&Element{Type: "p"},
				},
			},
		},
		&Element{
			Type: "FRAGMENT",
			Props: map[string]any{
				"children": []any{
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
		props:  make(map[string]any),
	}

	parseElements := []any{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]any{
				"children": []any{
					&Element{
						Type: "FRAGMENT",
						Props: map[string]any{
							"children": []any{
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
		props:  make(map[string]any),
	}

	parseElements := []any{
		&Element{Type: "FRAGMENT", Props: map[string]any{}},
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
		props:  make(map[string]any),
	}

	parseElements := []any{
		&Element{Type: "FRAGMENT", Props: map[string]any{"children": nil}},
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
		props:  make(map[string]any),
		dom:    parseAdapter.CreateElement("div"),
		dirty:  true, // Already dirty
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldFiber},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "div"}})

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
		props:  make(map[string]any),
	}

	parseElements := []any{
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
		props:     make(map[string]any),
		alternate: nil,
	}

	parseRt.reconcileChildren(parseWipFiber, []any{&Element{Type: "span"}})

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
	parseOldE := &Fiber{typeOf: "div", props: map[string]any{"key": "e"}, dom: parseAdapter.CreateElement("div")}
	parseOldD := &Fiber{typeOf: "div", props: map[string]any{"key": "d"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldE}
	parseOldC := &Fiber{typeOf: "div", props: map[string]any{"key": "c"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldD}
	parseOldB := &Fiber{typeOf: "div", props: map[string]any{"key": "b"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldC}
	parseOldA := &Fiber{typeOf: "div", props: map[string]any{"key": "a"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldB}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOldA},
	}

	// New: [e, c, a, f, b] (removed d, added f, reordered)
	parseElements := []any{
		&Element{Type: "div", Props: map[string]any{"key": "e"}},
		&Element{Type: "div", Props: map[string]any{"key": "c"}},
		&Element{Type: "div", Props: map[string]any{"key": "a"}},
		&Element{Type: "div", Props: map[string]any{"key": "f"}},
		&Element{Type: "div", Props: map[string]any{"key": "b"}},
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
		props:     make(map[string]any),
		alternate: &Fiber{child: nil},
	}

	parseElements := make([]any, 50)
	for parseI := range 50 {
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
	for parseI := range 50 {
		parseChild := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div")}
		if parseI == 0 {
			parseFirstChild = parseChild
		} else {
			parsePrevChild.sibling = parseChild
		}
		parsePrevChild = parseChild
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseFirstChild},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{})

	if parseWipFiber.child != nil {
		parseT.Error("Expected no children")
	}
	if len(parseRt.deletions) != 50 {
		parseT.Errorf("Expected 50 deletions, got %d", len(parseRt.deletions))
	}
}

func TestReconcileKeyedChildrenAppendsRemainingDeletionsInOldOrder(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	var parseFirstChild *Fiber
	var parsePrevChild *Fiber
	for parseI := 1; parseI <= 6; parseI++ {
		parseChild := &Fiber{
			typeOf: "div",
			props:  map[string]any{"key": parseI},
			dom:    parseAdapter.CreateElement("div"),
		}
		if parseFirstChild == nil {
			parseFirstChild = parseChild
		} else {
			parsePrevChild.sibling = parseChild
		}
		parsePrevChild = parseChild
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseFirstChild},
	}

	parseRt.reconcileChildren(parseWipFiber, []any{
		&Element{Type: "div", Props: map[string]any{"key": 1}},
		&Element{Type: "div", Props: map[string]any{"key": 3}},
		&Element{Type: "div", Props: map[string]any{"key": 6}},
	})

	if len(parseRt.deletions) != 3 {
		parseT.Fatalf("expected 3 keyed deletions, got %d", len(parseRt.deletions))
	}
	getDeletedKeys := []any{
		parseRt.deletions[0].props["key"],
		parseRt.deletions[1].props["key"],
		parseRt.deletions[2].props["key"],
	}
	getExpectedKeys := []any{2, 4, 5}
	for parseIndex := range getExpectedKeys {
		if getDeletedKeys[parseIndex] != getExpectedKeys[parseIndex] {
			parseT.Fatalf("expected keyed deletions in old sibling order %v, got %v", getExpectedKeys, getDeletedKeys)
		}
		if parseRt.deletions[parseIndex].effectTag != effectTagDeletion {
			parseT.Fatalf("expected deletion effect tag for key %v", getDeletedKeys[parseIndex])
		}
	}
}

func TestReconcileKeyedChildrenTrailingRemovalUsesOrderedFastPath(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseOld := make([]*Fiber, 6)
	for parseIndex := range parseOld {
		parseOld[parseIndex] = &Fiber{
			typeOf: "div",
			props:  map[string]any{"key": parseIndex + 1},
			dom:    parseAdapter.CreateElement("div"),
		}
		if parseIndex > 0 {
			parseOld[parseIndex-1].sibling = parseOld[parseIndex]
		}
	}
	parseWipFiber := &Fiber{typeOf: "div", alternate: &Fiber{child: parseOld[0]}}
	parseRt.reconcileChildren(parseWipFiber, []any{
		&Element{Type: "div", Props: map[string]any{"key": 1}},
		&Element{Type: "div", Props: map[string]any{"key": 2}},
	})

	if parseWipFiber.child == nil || parseWipFiber.child.dom != parseOld[0].dom || parseWipFiber.child.sibling == nil || parseWipFiber.child.sibling.dom != parseOld[1].dom {
		parseT.Fatal("expected the retained keyed prefix to preserve DOM identity")
	}
	if parseWipFiber.child.sibling.sibling != nil {
		parseT.Fatal("expected the retained keyed chain to end after key 2")
	}
	if len(parseRt.deletions) != 4 {
		parseT.Fatalf("expected 4 trailing deletions, got %d", len(parseRt.deletions))
	}
	for parseIndex, parseDeleted := range parseRt.deletions {
		if parseDeleted.props["key"] != parseIndex+3 || parseDeleted.effectTag != effectTagDeletion {
			parseT.Fatalf("deletion %d = key %#v tag %q, want key %d deletion", parseIndex, parseDeleted.props["key"], parseDeleted.effectTag, parseIndex+3)
		}
	}
}

func TestReconcileChildren_AlternatingUpdatesAndPlacements(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOld2 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div")}
	parseOld1 := &Fiber{typeOf: "div", props: make(map[string]any), dom: parseAdapter.CreateElement("div"), sibling: parseOld2}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]any),
		alternate: &Fiber{child: parseOld1},
	}

	// Keep first two, add new ones
	parseElements := []any{
		&Element{Type: "div", Props: map[string]any{"id": "new1"}},
		&Element{Type: "div", Props: map[string]any{"id": "new2"}},
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
