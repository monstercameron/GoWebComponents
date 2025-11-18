package runtime

import (
	"testing"
)

// TestReconcileChildren_RerenderNoChanges verifies that re-rendering with identical structure
// doesn't create duplicate DOM nodes (the bug where Statistics appeared twice)
func TestReconcileChildren_RerenderNoChanges(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	rt := &Runtime{domAdapter: mockDOM, deletions: make([]*Fiber, 0)}

	// Create initial structure: div with two children (div and span)
	parentFiber := &Fiber{
		typeOf: "ROOT",
		dom:    &testDOMNode{tag: "div", children: make([]DOMNode, 0)},
		props:  make(map[string]interface{}),
	}

	// First render
	elements1 := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"class": "first"}},
		&Element{Type: "span", Props: map[string]interface{}{"class": "second"}},
	}

	rt.reconcileChildren(parentFiber, elements1)

	// Check that children fibers were created
	if parentFiber.child == nil {
		t.Fatal("Expected child fiber to be created")
	}
	if parentFiber.child.dom != nil {
		t.Error("Initial render shouldn't have DOM yet - only fibers created")
	}

	// Simulate commit - connect fibers to actual DOM
	child1 := parentFiber.child
	child1.dom = mockDOM.CreateElement("div")
	child2 := child1.sibling
	child2.dom = mockDOM.CreateElement("span")

	// Manually "commit" by adding to parent
	parentFiber.dom.(*testDOMNode).children = append(parentFiber.dom.(*testDOMNode).children, child1.dom)
	parentFiber.dom.(*testDOMNode).children = append(parentFiber.dom.(*testDOMNode).children, child2.dom)

	// Verify we have 2 children
	if len(parentFiber.dom.(*testDOMNode).children) != 2 {
		t.Fatalf("Expected 2 children after first render, got %d", len(parentFiber.dom.(*testDOMNode).children))
	}

	// Second render with identical structure
	parentFiber2 := &Fiber{
		typeOf:    "ROOT",
		dom:       parentFiber.dom, // Same DOM node
		props:     make(map[string]interface{}),
		alternate: parentFiber, // Link to previous fiber tree
	}

	elements2 := []interface{}{
		&Element{Type: "div", Props: map[string]interface{}{"class": "first"}},
		&Element{Type: "span", Props: map[string]interface{}{"class": "second"}},
	}

	rt.reconcileChildren(parentFiber2, elements2)

	// Traverse the new fiber tree to count how many were marked as PLACEMENT
	placementCount := 0
	updateCount := 0
	fiber := parentFiber2.child
	for fiber != nil {
		if fiber.effectTag == "PLACEMENT" {
			placementCount++
		} else if fiber.effectTag == "UPDATE" {
			updateCount++
		}
		fiber = fiber.sibling
	}

	// Should have 0 PLACEMENT (no new nodes) and 2 UPDATE (existing nodes updated)
	if placementCount != 0 {
		t.Errorf("Expected 0 PLACEMENT fibers, got %d - this causes duplicate DOM nodes!", placementCount)
	}
	if updateCount != 2 {
		t.Errorf("Expected 2 UPDATE fibers, got %d", updateCount)
	}
}

// TestReconcileChildren_RerenderSameComponentMultipleTimes verifies that rendering
// the same component multiple times doesn't append duplicates
func TestReconcileChildren_RerenderSameComponentTwice(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	rt := &Runtime{domAdapter: mockDOM, deletions: make([]*Fiber, 0)}

	// Component that returns a div
	component := func(props map[string]interface{}) *Element {
		return &Element{Type: "div", Props: map[string]interface{}{"class": "stats"}}
	}

	parentFiber := &Fiber{
		typeOf: "ROOT",
		dom:    &testDOMNode{tag: "div", children: make([]DOMNode, 0)},
		props:  make(map[string]interface{}),
	}

	// First render - component called
	elements1 := []interface{}{component(nil)}
	rt.reconcileChildren(parentFiber, elements1)

	// Verify first child was created as PLACEMENT
	if parentFiber.child == nil || parentFiber.child.effectTag != "PLACEMENT" {
		t.Error("First render should create PLACEMENT fiber")
	}

	// Setup for second render - simulate what happens after commit
	oldFiber := parentFiber.child
	oldFiber.dom = mockDOM.CreateElement("div")
	parentFiber.dom.(*testDOMNode).children = append(parentFiber.dom.(*testDOMNode).children, oldFiber.dom)

	// Second render - same component structure
	parentFiber2 := &Fiber{
		typeOf:    "ROOT",
		dom:       parentFiber.dom,
		props:     make(map[string]interface{}),
		alternate: parentFiber,
	}

	elements2 := []interface{}{component(nil)}
	rt.reconcileChildren(parentFiber2, elements2)

	// Should be UPDATE, not PLACEMENT
	if parentFiber2.child.effectTag != "UPDATE" {
		t.Errorf("Second render should UPDATE existing component, got %s", parentFiber2.child.effectTag)
	}

	// Verify parent DOM still has only 1 child (not duplicated)
	if len(parentFiber2.dom.(*testDOMNode).children) != 1 {
		t.Errorf("Parent should still have 1 child after re-render, got %d", len(parentFiber2.dom.(*testDOMNode).children))
	}
}
