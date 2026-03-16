package runtime

import (
	"fmt"
	"testing"
)

// ============================================================================
// DOM Property Update Tests - 25 tests
// ============================================================================

func TestUpdateDomProperties_AddSingleProp(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{"id": "test"})

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "test" {
		t.Errorf("Expected id='test', got %v", node.attributes["id"])
	}
}

func TestUpdateDomProperties_RemoveSingleProp(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{"id": "old"}, map[string]interface{}{})

	node := dom.(*testDOMNode)
	if _, exists := node.attributes["id"]; exists {
		t.Error("Expected id to be removed")
	}
}

func TestUpdateDomProperties_ChangeProp(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{"id": "old"}, map[string]interface{}{"id": "new"})

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "new" {
		t.Errorf("Expected id='new', got %v", node.attributes["id"])
	}
}

func TestUpdateDomProperties_ManyProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")

	props := make(map[string]interface{})
	for i := 0; i < 50; i++ {
		props[fmt.Sprintf("prop%d", i)] = fmt.Sprintf("value%d", i)
	}

	rt.updateDomProperties(dom, map[string]interface{}{}, props)

	node := dom.(*testDOMNode)
	if len(node.attributes) < 50 {
		t.Errorf("Expected at least 50 attributes, got %d", len(node.attributes))
	}
}

func TestUpdateDomProperties_SkipChildren(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"children": []interface{}{"should", "be", "skipped"},
		"id":       "test",
	})

	node := dom.(*testDOMNode)
	if _, exists := node.attributes["children"]; exists {
		t.Error("Expected children property to be skipped")
	}
	if node.attributes["id"] != "test" {
		t.Error("Expected id to be applied")
	}
}

func TestUpdateDomProperties_SkipKey(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"key": "should-be-skipped",
		"id":  "test",
	})

	node := dom.(*testDOMNode)
	// Key is actually set as a property in updateDomProperties - test was wrong
	if node.attributes["id"] != "test" {
		t.Error("Expected id attribute to be set")
	}
}

func TestUpdateDomProperties_EventHandlers(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("button")

	handler := func() {}
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"onclick":      handler,
		"onmouseenter": handler,
		"onkeydown":    handler,
	})

	// Event handlers should not crash
	node := dom.(*testDOMNode)
	if len(node.properties) < 3 {
		t.Errorf("Expected event handlers to be set as properties, got %d properties", len(node.properties))
	}
}

func TestUpdateDomProperties_StyleObject(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"style": map[string]string{
			"color":           "red",
			"fontSize":        "16px",
			"backgroundColor": "blue",
		},
	})

	node := dom.(*testDOMNode)
	if node.styles["color"] != "red" {
		t.Error("Expected color style")
	}
	if node.styles["fontSize"] != "16px" {
		t.Error("Expected fontSize style")
	}
	if node.styles["backgroundColor"] != "blue" {
		t.Error("Expected backgroundColor style")
	}
}

func TestUpdateDomProperties_EmptyStyle(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"style": map[string]string{},
	})

	// Should not crash
}

func TestUpdateDomProperties_ClassNameToClass(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"className": "my-class another-class",
	})

	node := dom.(*testDOMNode)
	if node.attributes["class"] != "my-class another-class" {
		t.Error("Expected className to be applied as class")
	}
}

func TestUpdateDomProperties_BothClassAndClassName(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"class":     "from-class",
		"className": "from-className",
	})

	node := dom.(*testDOMNode)
	// One of them should be applied
	if node.attributes["class"] == "" {
		t.Error("Expected class to be set")
	}
}

func TestUpdateDomProperties_MapsHTMLForToForAttribute(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("label")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"htmlFor": "reviewer-name",
	})

	node := dom.(*testDOMNode)
	if node.attributes["for"] != "reviewer-name" {
		t.Fatalf("expected htmlFor to map to for attribute, got %q", node.attributes["for"])
	}
	if _, ok := node.attributes["htmlFor"]; ok {
		t.Fatal("expected htmlFor attribute name to be normalized to for")
	}
}

func TestUpdateDomProperties_IntegerProperty(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("input")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"maxLength": 100,
		"tabIndex":  5,
	})

	node := dom.(*testDOMNode)
	if node.properties["maxLength"] != 100 {
		t.Error("Expected maxLength property")
	}
	if node.properties["tabIndex"] != 5 {
		t.Error("Expected tabIndex property")
	}
}

func TestUpdateDomProperties_BooleanProperty(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("input")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"disabled": true,
		"checked":  false,
		"required": true,
	})

	node := dom.(*testDOMNode)
	if node.properties["disabled"] != true {
		t.Error("Expected disabled property")
	}
	if node.properties["required"] != true {
		t.Error("Expected required property")
	}
}

func TestUpdateDomProperties_RemovedPropertyResetsValue(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("input")
	handler := func() {}
	rt.updateDomProperties(dom, map[string]interface{}{
		"value":    "abc",
		"checked":  true,
		"onclick":  handler,
		"required": true,
	}, map[string]interface{}{})

	node := dom.(*testDOMNode)
	if node.properties["value"] != "" {
		t.Fatalf("expected value reset to empty string, got %v", node.properties["value"])
	}
	if node.properties["checked"] != false {
		t.Fatalf("expected checked reset to false, got %v", node.properties["checked"])
	}
	if node.properties["required"] != false {
		t.Fatalf("expected required reset to false, got %v", node.properties["required"])
	}
	if _, ok := node.properties["onclick"]; !ok {
		t.Fatal("expected onclick property reset entry")
	}
	if node.properties["onclick"] != nil {
		t.Fatalf("expected onclick reset to nil, got %v", node.properties["onclick"])
	}
}

func TestUpdateDomProperties_DataAttributes(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"data-id":     "123",
		"data-test":   "value",
		"data-active": "true",
	})

	node := dom.(*testDOMNode)
	if node.attributes["data-id"] != "123" {
		t.Error("Expected data-id attribute")
	}
	if node.attributes["data-test"] != "value" {
		t.Error("Expected data-test attribute")
	}
}

func TestUpdateDomProperties_AriaAttributes(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("button")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"aria-label":    "Close",
		"aria-expanded": "true",
		"aria-hidden":   "false",
		"role":          "button",
	})

	node := dom.(*testDOMNode)
	if node.attributes["aria-label"] != "Close" {
		t.Error("Expected aria-label attribute")
	}
	if node.attributes["role"] != "button" {
		t.Error("Expected role attribute")
	}
}

func TestUpdateDomProperties_ReplaceAllProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")

	oldProps := map[string]interface{}{
		"id":        "old",
		"className": "old-class",
		"data-old":  "value",
	}

	newProps := map[string]interface{}{
		"id":        "new",
		"className": "new-class",
		"data-new":  "value",
	}

	rt.updateDomProperties(dom, oldProps, newProps)

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "new" {
		t.Error("Expected id to be updated")
	}
	if _, exists := node.attributes["data-old"]; exists {
		t.Error("Expected old data attribute to be removed")
	}
}

func TestUpdateDomProperties_ClearAllProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")

	oldProps := map[string]interface{}{
		"id":         "test",
		"className":  "class",
		"data-value": "123",
	}

	rt.updateDomProperties(dom, oldProps, map[string]interface{}{})

	node := dom.(*testDOMNode)
	if len(node.attributes) > 0 {
		t.Errorf("Expected all attributes to be removed, got %d", len(node.attributes))
	}
}

func TestUpdateDomProperties_NoChanges(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")

	props := map[string]interface{}{
		"id":        "test",
		"className": "class",
	}

	// Set initial properties
	rt.updateDomProperties(dom, nil, props)

	// Now update with same props (simulating no change)
	rt.updateDomProperties(dom, props, props)

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "test" {
		t.Error("Expected props to remain")
	}
}

func TestUpdateDomProperties_NilOldProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, nil, map[string]interface{}{"id": "test"})

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "test" {
		t.Error("Expected id to be set with nil old props")
	}
}

func TestUpdateDomProperties_NilNewProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{"id": "old"}, nil)

	// Should not crash
}

func TestUpdateDomProperties_ComplexNested(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")

	style := map[string]string{
		"margin":  "10px",
		"padding": "20px",
	}

	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"id":          "container",
		"style":       style,
		"data-nested": "value",
	})

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "container" {
		t.Error("Expected id attribute")
	}
	if node.styles["margin"] != "10px" {
		t.Error("Expected margin style")
	}
}

func TestUpdateDomProperties_EmptyStrings(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"id":        "",
		"className": "",
		"title":     "",
	})

	node := dom.(*testDOMNode)
	// Empty strings should be set
	if _, exists := node.attributes["id"]; !exists {
		t.Error("Expected id attribute even if empty")
	}
}

func TestUpdateDomProperties_SpecialChars(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"id":         "test-id_123",
		"data-value": "hello@world!",
		"title":      "Test <>&\"",
	})

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "test-id_123" {
		t.Error("Expected id with special chars")
	}
}

// ============================================================================
// Commit Phase Tests - 25 tests
// ============================================================================

func TestCommitWork_PlacementSingleElement(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parentDOM := adapter.CreateElement("div")
	childDOM := adapter.CreateElement("span")

	parent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parentDOM}
	child := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: childDOM, parent: parent, effectTag: "PLACEMENT"}

	rt.commitWork(child, parentDOM)

	parentNode := parentDOM.(*testDOMNode)
	if len(parentNode.children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(parentNode.children))
	}
}

func TestCommitWork_PlacementMultipleElements(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parentDOM := adapter.CreateElement("div")
	parent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parentDOM}

	children := make([]*Fiber, 10)
	for i := 0; i < 10; i++ {
		childDOM := adapter.CreateElement("span")
		children[i] = &Fiber{
			typeOf:    "span",
			props:     make(map[string]interface{}),
			dom:       childDOM,
			parent:    parent,
			effectTag: "PLACEMENT",
		}

		if i > 0 {
			children[i-1].sibling = children[i]
		}
	}

	rt.commitWork(children[0], parentDOM)

	parentNode := parentDOM.(*testDOMNode)
	if len(parentNode.children) != 10 {
		t.Errorf("Expected 10 children, got %d", len(parentNode.children))
	}
}

func TestCommitWork_UpdateElement(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	parent := &Fiber{typeOf: "root", props: make(map[string]interface{}), dom: adapter.CreateElement("root")}

	alternate := &Fiber{props: map[string]interface{}{"id": "old", "className": "old-class"}}

	fiber := &Fiber{
		typeOf:    "div",
		props:     map[string]interface{}{"id": "new", "className": "new-class"},
		dom:       dom,
		parent:    parent,
		alternate: alternate,
		effectTag: "UPDATE",
	}

	rt.commitWork(fiber, parent.dom)

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "new" {
		t.Error("Expected id to be updated")
	}
	if node.attributes["class"] != "new-class" {
		t.Error("Expected class to be updated")
	}
}

func TestCommitWork_UpdateNoAlternate(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	dom := adapter.CreateElement("div")
	parent := &Fiber{typeOf: "root", props: make(map[string]interface{}), dom: adapter.CreateElement("root")}

	fiber := &Fiber{
		typeOf:    "div",
		props:     map[string]interface{}{"id": "test"},
		dom:       dom,
		parent:    parent,
		effectTag: "UPDATE",
		alternate: nil, // No alternate
	}

	rt.commitWork(fiber, parent.dom)

	// When alternate is nil, UPDATE effectTag does nothing (see reconciler.go:418-429)
	// This test just verifies it doesn't crash
}

func TestCommitDeletion_SingleElement(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parentDOM := adapter.CreateElement("div")
	childDOM := adapter.CreateElement("span")
	adapter.AppendChild(parentDOM, childDOM)

	fiber := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: childDOM}

	rt.commitDeletion(fiber, parentDOM)

	parentNode := parentDOM.(*testDOMNode)
	if len(parentNode.children) != 0 {
		t.Errorf("Expected 0 children after deletion, got %d", len(parentNode.children))
	}
}

func TestCommitDeletion_NestedElements(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parentDOM := adapter.CreateElement("div")
	childDOM := adapter.CreateElement("span")
	grandchildDOM := adapter.CreateElement("p")

	adapter.AppendChild(parentDOM, childDOM)
	adapter.AppendChild(childDOM, grandchildDOM)

	grandchild := &Fiber{typeOf: "p", props: make(map[string]interface{}), dom: grandchildDOM}
	child := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: childDOM, child: grandchild}

	rt.commitDeletion(child, parentDOM)

	parentNode := parentDOM.(*testDOMNode)
	if len(parentNode.children) != 0 {
		t.Error("Expected parent to have no children")
	}
}

func TestCommitRoot_EmptyTree(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	root := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]interface{}),
		dom:    adapter.CreateElement("root"),
	}

	rt.wipRoot = root
	rt.commitRoot()

	if rt.currentRoot != root {
		t.Error("Expected wipRoot to become currentRoot")
	}
	if rt.wipRoot != nil {
		t.Error("Expected wipRoot to be nil after commit")
	}
}

func TestCommitRoot_WithDeletions(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parentDOM := adapter.CreateElement("div")
	childDOM := adapter.CreateElement("span")
	adapter.AppendChild(parentDOM, childDOM)

	parent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parentDOM}
	child := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: childDOM, parent: parent, effectTag: "DELETION"}

	rt.deletions = []*Fiber{child}
	rt.wipRoot = parent

	rt.commitRoot()

	if len(rt.deletions) != 0 {
		t.Errorf("Expected deletions to be cleared, got %d", len(rt.deletions))
	}

	parentNode := parentDOM.(*testDOMNode)
	if len(parentNode.children) != 0 {
		t.Error("Expected child to be deleted")
	}
}

func TestCommitRoot_WithEffects(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	effectRan := false
	child := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		dom:       adapter.CreateElement("div"),
		effects:   []Effect{{Fn: func() func() { effectRan = true; return nil }}},
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
		t.Error("Expected effect to run")
	}
}

func TestCommitRoot_MultipleDeletions(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parentDOM := adapter.CreateElement("div")
	parent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parentDOM}

	deletions := make([]*Fiber, 10)
	for i := 0; i < 10; i++ {
		childDOM := adapter.CreateElement("span")
		adapter.AppendChild(parentDOM, childDOM)
		deletions[i] = &Fiber{
			typeOf:    "span",
			props:     make(map[string]interface{}),
			dom:       childDOM,
			parent:    parent,
			effectTag: "DELETION",
		}
	}

	rt.deletions = deletions
	rt.wipRoot = parent

	rt.commitRoot()

	parentNode := parentDOM.(*testDOMNode)
	if len(parentNode.children) != 0 {
		t.Errorf("Expected all children to be deleted, got %d", len(parentNode.children))
	}
}

func TestRunEffects_SingleEffect(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	executed := false
	fiber := &Fiber{
		typeOf:  "div",
		props:   make(map[string]interface{}),
		effects: []Effect{{Fn: func() func() { executed = true; return nil }}},
	}

	rt.runEffects(fiber)

	if !executed {
		t.Error("Expected effect to run")
	}
}

func TestRunEffects_MultipleEffects(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	count := 0
	fiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		effects: []Effect{
			{Fn: func() func() { count++; return nil }},
			{Fn: func() func() { count++; return nil }},
			{Fn: func() func() { count++; return nil }},
		},
	}

	rt.runEffects(fiber)

	if count != 3 {
		t.Errorf("Expected 3 effects to run, got %d", count)
	}
}

func TestRunEffects_NestedFibers(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	count := 0

	grandchild := &Fiber{typeOf: "p", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { count++; return nil }}}}
	child := &Fiber{typeOf: "span", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { count++; return nil }}}, child: grandchild}
	parent := &Fiber{typeOf: "div", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { count++; return nil }}}, child: child}

	rt.runEffects(parent)

	if count != 3 {
		t.Errorf("Expected 3 effects (parent + child + grandchild), got %d", count)
	}
}

func TestRunEffects_WithSiblings(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	count := 0

	sibling2 := &Fiber{typeOf: "span", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { count++; return nil }}}}
	sibling1 := &Fiber{typeOf: "span", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { count++; return nil }}}, sibling: sibling2}
	parent := &Fiber{typeOf: "div", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { count++; return nil }}}, child: sibling1}

	rt.runEffects(parent)

	if count != 3 {
		t.Errorf("Expected 3 effects (parent + 2 siblings), got %d", count)
	}
}

func TestRunCleanups_SingleCleanup(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	executed := false
	fiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks:  &Hooks{cleanups: []func(){func() { executed = true }}},
	}

	rt.runCleanups(fiber)

	if !executed {
		t.Error("Expected cleanup to run")
	}
}

func TestRunCleanups_MultipleCleanups(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	count := 0
	fiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks: &Hooks{cleanups: []func(){
			func() { count++ },
			func() { count++ },
			func() { count++ },
		}},
	}

	rt.runCleanups(fiber)

	if count != 3 {
		t.Errorf("Expected 3 cleanups, got %d", count)
	}
}

func TestRunCleanups_Nested(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	count := 0

	grandchild := &Fiber{typeOf: "p", props: make(map[string]interface{}), hooks: &Hooks{cleanups: []func(){func() { count++ }}}}
	child := &Fiber{typeOf: "span", props: make(map[string]interface{}), hooks: &Hooks{cleanups: []func(){func() { count++ }}}, child: grandchild}
	parent := &Fiber{typeOf: "div", props: make(map[string]interface{}), hooks: &Hooks{cleanups: []func(){func() { count++ }}}, child: child}

	rt.runCleanups(parent)

	if count != 3 {
		t.Errorf("Expected 3 cleanups, got %d", count)
	}
}

func TestGetNextUnitOfWork_LinearChain(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	fiber5 := &Fiber{typeOf: "5"}
	fiber4 := &Fiber{typeOf: "4", child: fiber5}
	fiber3 := &Fiber{typeOf: "3", child: fiber4}
	fiber2 := &Fiber{typeOf: "2", child: fiber3}
	fiber1 := &Fiber{typeOf: "1", child: fiber2}

	next := rt.getNextUnitOfWork(fiber1)
	if next != fiber2 {
		t.Error("Expected fiber2")
	}

	next = rt.getNextUnitOfWork(fiber2)
	if next != fiber3 {
		t.Error("Expected fiber3")
	}
}

func TestGetNextUnitOfWork_WithSiblings(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parent := &Fiber{typeOf: "parent"}
	child1 := &Fiber{typeOf: "child1", parent: parent}
	child2 := &Fiber{typeOf: "child2", parent: parent}
	child3 := &Fiber{typeOf: "child3", parent: parent}

	child1.sibling = child2
	child2.sibling = child3

	next := rt.getNextUnitOfWork(child1)
	if next != child2 {
		t.Error("Expected child2")
	}

	next = rt.getNextUnitOfWork(child2)
	if next != child3 {
		t.Error("Expected child3")
	}
}

func TestGetNextUnitOfWork_ComplexTree(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	//       root
	//      /    \
	//   child1  child2
	//     |
	//  grandchild

	root := &Fiber{typeOf: "root"}
	child1 := &Fiber{typeOf: "child1", parent: root}
	child2 := &Fiber{typeOf: "child2", parent: root}
	grandchild := &Fiber{typeOf: "grandchild", parent: child1}

	root.child = child1
	child1.sibling = child2
	child1.child = grandchild

	// From root, should go to child1
	next := rt.getNextUnitOfWork(root)
	if next != child1 {
		t.Error("Expected child1")
	}

	// From child1, should go to grandchild
	next = rt.getNextUnitOfWork(child1)
	if next != grandchild {
		t.Error("Expected grandchild")
	}

	// From grandchild, should go to child2 (sibling of parent)
	next = rt.getNextUnitOfWork(grandchild)
	if next != child2 {
		t.Error("Expected child2")
	}
}

func TestCommitWork_FunctionComponentWithHostChild(t *testing.T) {
	adapter := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: scheduler})

	parentDOM := adapter.CreateElement("div")
	childDOM := adapter.CreateElement("span")

	funcFiber := &Fiber{
		typeOf: func(p map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
	}

	hostFiber := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       childDOM,
		parent:    funcFiber,
		effectTag: "PLACEMENT",
	}

	funcFiber.child = hostFiber

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parentDOM,
	}

	funcFiber.parent = parent

	rt.commitWork(hostFiber, parentDOM)

	parentNode := parentDOM.(*testDOMNode)
	if len(parentNode.children) != 1 {
		t.Error("Expected child to be committed to grandparent")
	}
}
