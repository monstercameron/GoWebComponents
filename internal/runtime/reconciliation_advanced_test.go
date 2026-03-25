package runtime

import (
	"testing"
)

// ============================================================================
// Additional Advanced Reconciliation Tests - 8 tests
// ============================================================================

func TestReconcileChildren_TextNodeCreation(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseWipFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	// TEXT_ELEMENT type for text nodes
	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "TEXT_ELEMENT", Props: map[string]interface{}{"nodeValue": "Hello"}},
	})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected text node child")
	}
	if parseWipFiber.child.typeOf != "TEXT_ELEMENT" {
		parseT.Error("Expected TEXT_ELEMENT type")
	}
	if parseWipFiber.child.props["nodeValue"] != "Hello" {
		parseT.Error("Expected nodeValue Hello")
	}
}

func TestReconcileChildren_TextNodeUpdate(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldText := &Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]interface{}{"nodeValue": "Old"},
		dom:    parseAdapter.CreateTextNode("Old"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldText},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "TEXT_ELEMENT", Props: map[string]interface{}{"nodeValue": "New"}},
	})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.effectTag != "UPDATE" {
		parseT.Error("Expected UPDATE for text node")
	}
	if parseWipFiber.child.props["nodeValue"] != "New" {
		parseT.Error("Expected new nodeValue")
	}
}

func TestReconcileChildren_ElementToTextNodeChange(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldElement := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    parseAdapter.CreateElement("span"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldElement},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "TEXT_ELEMENT", Props: map[string]interface{}{"nodeValue": "Text"}},
	})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.effectTag != "PLACEMENT" {
		parseT.Error("Expected PLACEMENT for type change from element to text")
	}
	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}
}

func TestReconcileChildren_TextNodeToElementChange(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseOldText := &Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]interface{}{"nodeValue": "Text"},
		dom:    parseAdapter.CreateTextNode("Text"),
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldText},
	}

	parseRt.reconcileChildren(parseWipFiber, []interface{}{
		&Element{Type: "span"},
	})

	if parseWipFiber.child == nil {
		parseT.Fatal("Expected child")
	}
	if parseWipFiber.child.effectTag != "PLACEMENT" {
		parseT.Error("Expected PLACEMENT for type change from text to element")
	}
	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}
}

func TestReconcileChildren_KeyedReorderBackward(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Old: [1, 2, 3, 4, 5]
	parseOld5 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "5"}, dom: parseAdapter.CreateElement("div")}
	parseOld4 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "4"}, dom: parseAdapter.CreateElement("div"), sibling: parseOld5}
	parseOld3 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "3"}, dom: parseAdapter.CreateElement("div"), sibling: parseOld4}
	parseOld2 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "2"}, dom: parseAdapter.CreateElement("div"), sibling: parseOld3}
	parseOld1 := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "1"}, dom: parseAdapter.CreateElement("div"), sibling: parseOld2}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOld1},
	}

	// New: [5, 4, 3, 2, 1] (reverse order)
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "5"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "4"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "3"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "2"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "1"}},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	// Verify children exist
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

func TestReconcileChildren_KeyedInsertMiddle(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Old: [a, c]
	parseOldC := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "c"}, dom: parseAdapter.CreateElement("div")}
	parseOldA := &Fiber{typeOf: "div", props: map[string]interface{}{"key": "a"}, dom: parseAdapter.CreateElement("div"), sibling: parseOldC}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseOldA},
	}

	// New: [a, b, c] (insert b in middle)
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "b"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		if parseCount == 2 && parseChild.props["key"] != "b" {
			parseT.Error("Expected second child to be 'b'")
		}
		parseChild = parseChild.sibling
	}

	if parseCount != 3 {
		parseT.Errorf("Expected 3 children, got %d", parseCount)
	}
}

func TestReconcileChildren_KeyedRemoveMiddle(parseT *testing.T) {
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

	// New: [a, c] (remove b from middle)
	parseElements := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"key": "a"}},
		&Element{Type: "div", Props: map[string]interface{}{"key": "c"}},
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		parseChild = parseChild.sibling
	}

	if parseCount != 2 {
		parseT.Errorf("Expected 2 children, got %d", parseCount)
	}
	if parseWipFiber.child == nil || parseWipFiber.child.props["key"] != "a" {
		parseT.Fatal("expected first keyed child to remain 'a'")
	}
	if parseWipFiber.child.sibling == nil || parseWipFiber.child.sibling.props["key"] != "c" {
		parseT.Fatal("expected second keyed child to remain 'c'")
	}
	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}
	if parseRt.deletions[0].props["key"] != "b" {
		parseT.Fatalf("expected deleted keyed child to be 'b', got %#v", parseRt.deletions[0].props["key"])
	}
}

func TestReconcileChildren_LongChainNoKeys(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	// Create old chain of 20 elements
	var parseFirstOld *Fiber
	var parsePrevOld *Fiber
	for parseI := 0; parseI < 20; parseI++ {
		parseOld := &Fiber{
			typeOf: "div",
			props:  make(map[string]interface{}),
			dom:    parseAdapter.CreateElement("div"),
		}
		if parseI == 0 {
			parseFirstOld = parseOld
		} else {
			parsePrevOld.sibling = parseOld
		}
		parsePrevOld = parseOld
	}

	parseWipFiber := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		alternate: &Fiber{child: parseFirstOld},
	}

	// Create new elements (same count, no keys)
	parseElements := make([]interface{}, 20)
	for parseI2 := 0; parseI2 < 20; parseI2++ {
		// Add a prop to force update
		parseElements[parseI2] = &Element{Type: "div", Props: map[string]interface{}{"id": "new"}}
	}

	parseRt.reconcileChildren(parseWipFiber, parseElements)

	parseCount := 0
	parseChild := parseWipFiber.child
	for parseChild != nil {
		parseCount++
		if parseChild.effectTag != "UPDATE" {
			parseT.Errorf("Expected UPDATE for position %d, got %s", parseCount, parseChild.effectTag)
		}
		parseChild = parseChild.sibling
	}

	if parseCount != 20 {
		parseT.Errorf("Expected 20 children, got %d", parseCount)
	}
}
