package runtime

import (
	"testing"
)

// TestReconcileChildren_RerenderNoChanges verifies that re-rendering with identical structure
// doesn't create duplicate DOM nodes (the bug where Statistics appeared twice)
func TestReconcileChildren_RerenderNoChanges(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseRt := &Runtime{domAdapter: parseMockDOM, deletions: make([]*Fiber, 0)}

	// Create initial structure: div with two children (div and span)
	parseParentFiber := &Fiber{
		typeOf: "ROOT",
		dom:    &testDOMNode{tag: "div", children: make([]DOMNode, 0)},
		props:  make(map[string]interface{}),
	}

	// First render
	parseElements1 := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"class": "first"}},
		&Element{Type: "span", Props: map[string]interface{}{"class": "second"}},
	}

	parseRt.reconcileChildren(parseParentFiber, parseElements1)

	// Check that children fibers were created
	if parseParentFiber.child == nil {
		parseT.Fatal("Expected child fiber to be created")
	}
	if parseParentFiber.child.dom != nil {
		parseT.Error("Initial render shouldn't have DOM yet - only fibers created")
	}

	// Simulate commit - connect fibers to actual DOM
	parseChild1 := parseParentFiber.child
	parseChild1.dom = parseMockDOM.CreateElement("div")
	parseChild2 := parseChild1.sibling
	parseChild2.dom = parseMockDOM.CreateElement("span")

	// Manually "commit" by adding to parent
	parseParentFiber.dom.(*testDOMNode).children = append(parseParentFiber.dom.(*testDOMNode).children, parseChild1.dom)
	parseParentFiber.dom.(*testDOMNode).children = append(parseParentFiber.dom.(*testDOMNode).children, parseChild2.dom)

	// Verify we have 2 children
	if len(parseParentFiber.dom.(*testDOMNode).children) != 2 {
		parseT.Fatalf("Expected 2 children after first render, got %d", len(parseParentFiber.dom.(*testDOMNode).children))
	}

	// Second render with identical structure
	parseParentFiber2 := &Fiber{
		typeOf:    "ROOT",
		dom:       parseParentFiber.dom, // Same DOM node
		props:     make(map[string]interface{}),
		alternate: parseParentFiber, // Link to previous fiber tree
	}

	parseElements2 := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"class": "first"}},
		&Element{Type: "span", Props: map[string]interface{}{"class": "second"}},
	}

	parseRt.reconcileChildren(parseParentFiber2, parseElements2)

	// Traverse the new fiber tree to count how many were marked as PLACEMENT
	parsePlacementCount := 0
	parseUpdateCount := 0
	parseFiber := parseParentFiber2.child
	for parseFiber != nil {
		switch parseFiber.effectTag {
		case effectTagPlacement:
			parsePlacementCount++
		case effectTagUpdate:
			parseUpdateCount++
		}
		parseFiber = parseFiber.sibling
	}

	// Should have 0 PLACEMENT (no new nodes) and 2 UPDATE (existing nodes updated)
	if parsePlacementCount != 0 {
		parseT.Errorf("Expected 0 PLACEMENT fibers, got %d - this causes duplicate DOM nodes!", parsePlacementCount)
	}
	if parseUpdateCount != 2 {
		parseT.Errorf("Expected 2 UPDATE fibers, got %d", parseUpdateCount)
	}
}

// TestReconcileChildren_RerenderSameComponentTwice verifies that rendering
// the same component multiple times doesn't append duplicates
func TestReconcileChildren_RerenderSameComponentTwice(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseRt := &Runtime{domAdapter: parseMockDOM, deletions: make([]*Fiber, 0)}

	// Component that returns a div
	parseComponent := func(_ map[string]interface{}) *Element {
		return &Element{Type: "div", Props: map[string]interface{}{"class": "stats"}}
	}

	parseParentFiber := &Fiber{
		typeOf: "ROOT",
		dom:    &testDOMNode{tag: "div", children: make([]DOMNode, 0)},
		props:  make(map[string]interface{}),
	}

	// First render - component called
	parseElements1 := []interface{}{parseComponent(nil)}
	parseRt.reconcileChildren(parseParentFiber, parseElements1)

	// Verify first child was created as PLACEMENT
	if parseParentFiber.child == nil || parseParentFiber.child.effectTag != effectTagPlacement {
		parseT.Error("First render should create PLACEMENT fiber")
	}

	// Setup for second render - simulate what happens after commit
	parseOldFiber := parseParentFiber.child
	parseOldFiber.dom = parseMockDOM.CreateElement("div")
	parseParentFiber.dom.(*testDOMNode).children = append(parseParentFiber.dom.(*testDOMNode).children, parseOldFiber.dom)

	// Second render - same component structure
	parseParentFiber2 := &Fiber{
		typeOf:    "ROOT",
		dom:       parseParentFiber.dom,
		props:     make(map[string]interface{}),
		alternate: parseParentFiber,
	}

	parseElements2 := []interface{}{parseComponent(nil)}
	parseRt.reconcileChildren(parseParentFiber2, parseElements2)

	// Should be UPDATE, not PLACEMENT
	if parseParentFiber2.child.effectTag != effectTagUpdate {
		parseT.Errorf("Second render should UPDATE existing component, got %s", parseParentFiber2.child.effectTag)
	}

	// Verify parent DOM still has only 1 child (not duplicated)
	if len(parseParentFiber2.dom.(*testDOMNode).children) != 1 {
		parseT.Errorf("Parent should still have 1 child after re-render, got %d", len(parseParentFiber2.dom.(*testDOMNode).children))
	}
}
