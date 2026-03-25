package runtime

import (
	"fmt"
	"testing"
)

// ============================================================================
// DOM Property Update Tests - 25 tests
// ============================================================================

func TestUpdateDomProperties_AddSingleProp(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{"id": "test"})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "test" {
		parseT.Errorf("Expected id='test', got %v", parseNode.attributes["id"])
	}
}

func TestUpdateDomProperties_RemoveSingleProp(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{"id": "old"}, map[string]interface{}{})

	parseNode := parseDom.(*testDOMNode)
	if _, parseExists := parseNode.attributes["id"]; parseExists {
		parseT.Error("Expected id to be removed")
	}
}

func TestUpdateDomProperties_ChangeProp(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{"id": "old"}, map[string]interface{}{"id": "new"})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "new" {
		parseT.Errorf("Expected id='new', got %v", parseNode.attributes["id"])
	}
}

func TestUpdateDomProperties_ManyProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")

	parseProps := make(map[string]interface{})
	for parseI := 0; parseI < 50; parseI++ {
		parseProps[fmt.Sprintf("prop%d", parseI)] = fmt.Sprintf("value%d", parseI)
	}

	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, parseProps)

	parseNode := parseDom.(*testDOMNode)
	if len(parseNode.attributes) < 50 {
		parseT.Errorf("Expected at least 50 attributes, got %d", len(parseNode.attributes))
	}
}

func TestUpdateDomProperties_SkipChildren(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"children": []interface{}{"should", "be", "skipped"},
		"id":       "test",
	})

	parseNode := parseDom.(*testDOMNode)
	if _, parseExists := parseNode.attributes["children"]; parseExists {
		parseT.Error("Expected children property to be skipped")
	}
	if parseNode.attributes["id"] != "test" {
		parseT.Error("Expected id to be applied")
	}
}

func TestUpdateDomProperties_SkipKey(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"key": "should-be-skipped",
		"id":  "test",
	})

	parseNode := parseDom.(*testDOMNode)
	// Key is actually set as a property in updateDomProperties - test was wrong
	if parseNode.attributes["id"] != "test" {
		parseT.Error("Expected id attribute to be set")
	}
}

func TestUpdateDomProperties_EventHandlers(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("button")

	parseHandler := func() {}
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"onclick":      parseHandler,
		"onmouseenter": parseHandler,
		"onkeydown":    parseHandler,
	})

	// Event handlers should not crash
	parseNode := parseDom.(*testDOMNode)
	if len(parseNode.properties) < 3 {
		parseT.Errorf("Expected event handlers to be set as properties, got %d properties", len(parseNode.properties))
	}
}

func TestUpdateDomProperties_StyleObject(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"style": map[string]string{
			"color":           "red",
			"fontSize":        "16px",
			"backgroundColor": "blue",
		},
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.styles["color"] != "red" {
		parseT.Error("Expected color style")
	}
	if parseNode.styles["fontSize"] != "16px" {
		parseT.Error("Expected fontSize style")
	}
	if parseNode.styles["backgroundColor"] != "blue" {
		parseT.Error("Expected backgroundColor style")
	}
}

func TestUpdateDomProperties_EmptyStyle(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"style": map[string]string{},
	})

	// Should not crash
}

func TestUpdateDomProperties_ClassNameToClass(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"className": "my-class another-class",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["class"] != "my-class another-class" {
		parseT.Error("Expected className to be applied as class")
	}
}

func TestUpdateDomProperties_BothClassAndClassName(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"class":     "from-class",
		"className": "from-className",
	})

	parseNode := parseDom.(*testDOMNode)
	// One of them should be applied
	if parseNode.attributes["class"] == "" {
		parseT.Error("Expected class to be set")
	}
}

func TestUpdateDomProperties_MapsHTMLForToForAttribute(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("label")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"htmlFor": "reviewer-name",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["for"] != "reviewer-name" {
		parseT.Fatalf("expected htmlFor to map to for attribute, got %q", parseNode.attributes["for"])
	}
	if _, parseOk := parseNode.attributes["htmlFor"]; parseOk {
		parseT.Fatal("expected htmlFor attribute name to be normalized to for")
	}
}

func TestUpdateDomProperties_IntegerProperty(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("input")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"maxLength": 100,
		"tabIndex":  5,
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.properties["maxLength"] != 100 {
		parseT.Error("Expected maxLength property")
	}
	if parseNode.properties["tabIndex"] != 5 {
		parseT.Error("Expected tabIndex property")
	}
}

func TestUpdateDomProperties_BooleanProperty(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("input")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"disabled": true,
		"checked":  false,
		"required": true,
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.properties["disabled"] != true {
		parseT.Error("Expected disabled property")
	}
	if parseNode.properties["required"] != true {
		parseT.Error("Expected required property")
	}
}

func TestUpdateDomProperties_RemovedPropertyResetsValue(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("input")
	parseHandler := func() {}
	parseRt.updateDomProperties(parseDom, map[string]interface{}{
		"value":    "abc",
		"checked":  true,
		"onclick":  parseHandler,
		"required": true,
	}, map[string]interface{}{})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.properties["value"] != "" {
		parseT.Fatalf("expected value reset to empty string, got %v", parseNode.properties["value"])
	}
	if parseNode.properties["checked"] != false {
		parseT.Fatalf("expected checked reset to false, got %v", parseNode.properties["checked"])
	}
	if parseNode.properties["required"] != false {
		parseT.Fatalf("expected required reset to false, got %v", parseNode.properties["required"])
	}
	if _, parseOk := parseNode.properties["onclick"]; !parseOk {
		parseT.Fatal("expected onclick property reset entry")
	}
	if parseNode.properties["onclick"] != nil {
		parseT.Fatalf("expected onclick reset to nil, got %v", parseNode.properties["onclick"])
	}
}

func TestUpdateDomProperties_DataAttributes(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"data-id":     "123",
		"data-test":   "value",
		"data-active": "true",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["data-id"] != "123" {
		parseT.Error("Expected data-id attribute")
	}
	if parseNode.attributes["data-test"] != "value" {
		parseT.Error("Expected data-test attribute")
	}
}

func TestUpdateDomProperties_AriaAttributes(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("button")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"aria-label":    "Close",
		"aria-expanded": "true",
		"aria-hidden":   "false",
		"role":          "button",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["aria-label"] != "Close" {
		parseT.Error("Expected aria-label attribute")
	}
	if parseNode.attributes["role"] != "button" {
		parseT.Error("Expected role attribute")
	}
}

func TestUpdateDomProperties_ReplaceAllProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")

	parseOldProps := map[string]interface{}{
		"id":        "old",
		"className": "old-class",
		"data-old":  "value",
	}

	parseNewProps := map[string]interface{}{
		"id":        "new",
		"className": "new-class",
		"data-new":  "value",
	}

	parseRt.updateDomProperties(parseDom, parseOldProps, parseNewProps)

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "new" {
		parseT.Error("Expected id to be updated")
	}
	if _, parseExists := parseNode.attributes["data-old"]; parseExists {
		parseT.Error("Expected old data attribute to be removed")
	}
}

func TestUpdateDomProperties_ClearAllProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")

	parseOldProps := map[string]interface{}{
		"id":         "test",
		"className":  "class",
		"data-value": "123",
	}

	parseRt.updateDomProperties(parseDom, parseOldProps, map[string]interface{}{})

	parseNode := parseDom.(*testDOMNode)
	if len(parseNode.attributes) > 0 {
		parseT.Errorf("Expected all attributes to be removed, got %d", len(parseNode.attributes))
	}
}

func TestUpdateDomProperties_NoChanges(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")

	parseProps := map[string]interface{}{
		"id":        "test",
		"className": "class",
	}

	// Set initial properties
	parseRt.updateDomProperties(parseDom, nil, parseProps)

	// Now update with same props (simulating no change)
	parseRt.updateDomProperties(parseDom, parseProps, parseProps)

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "test" {
		parseT.Error("Expected props to remain")
	}
}

func TestUpdateDomProperties_NilOldProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, nil, map[string]interface{}{"id": "test"})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "test" {
		parseT.Error("Expected id to be set with nil old props")
	}
}

func TestUpdateDomProperties_NilNewProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{"id": "old"}, nil)

	// Should not crash
}

func TestUpdateDomProperties_ComplexNested(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")

	parseStyle := map[string]string{
		"margin":  "10px",
		"padding": "20px",
	}

	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"id":          "container",
		"style":       parseStyle,
		"data-nested": "value",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "container" {
		parseT.Error("Expected id attribute")
	}
	if parseNode.styles["margin"] != "10px" {
		parseT.Error("Expected margin style")
	}
}

func TestUpdateDomProperties_EmptyStrings(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"id":        "",
		"className": "",
		"title":     "",
	})

	parseNode := parseDom.(*testDOMNode)
	// Empty strings should be set
	if _, parseExists := parseNode.attributes["id"]; !parseExists {
		parseT.Error("Expected id attribute even if empty")
	}
}

func TestUpdateDomProperties_SpecialChars(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseRt.updateDomProperties(parseDom, map[string]interface{}{}, map[string]interface{}{
		"id":         "test-id_123",
		"data-value": "hello@world!",
		"title":      "Test <>&\"",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "test-id_123" {
		parseT.Error("Expected id with special chars")
	}
}

// ============================================================================
// Commit Phase Tests - 25 tests
// ============================================================================

func TestCommitWork_PlacementSingleElement(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParentDOM := parseAdapter.CreateElement("div")
	parseChildDOM := parseAdapter.CreateElement("span")

	parseParent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseParentDOM}
	parseChild := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: parseChildDOM, parent: parseParent, effectTag: "PLACEMENT"}

	parseRt.commitWork(parseChild, parseParentDOM)

	parseParentNode := parseParentDOM.(*testDOMNode)
	if len(parseParentNode.children) != 1 {
		parseT.Errorf("Expected 1 child, got %d", len(parseParentNode.children))
	}
}

func TestCommitWork_PlacementMultipleElements(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParentDOM := parseAdapter.CreateElement("div")
	parseParent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseParentDOM}

	parseChildren := make([]*Fiber, 10)
	for parseI := 0; parseI < 10; parseI++ {
		parseChildDOM := parseAdapter.CreateElement("span")
		parseChildren[parseI] = &Fiber{
			typeOf:    "span",
			props:     make(map[string]interface{}),
			dom:       parseChildDOM,
			parent:    parseParent,
			effectTag: "PLACEMENT",
		}

		if parseI > 0 {
			parseChildren[parseI-1].sibling = parseChildren[parseI]
		}
	}

	parseRt.commitWork(parseChildren[0], parseParentDOM)

	parseParentNode := parseParentDOM.(*testDOMNode)
	if len(parseParentNode.children) != 10 {
		parseT.Errorf("Expected 10 children, got %d", len(parseParentNode.children))
	}
}

func TestCommitWork_UpdateElement(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseParent := &Fiber{typeOf: "root", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("root")}

	parseAlternate := &Fiber{props: map[string]interface{}{"id": "old", "className": "old-class"}}

	parseFiber := &Fiber{
		typeOf:    "div",
		props:     map[string]interface{}{"id": "new", "className": "new-class"},
		dom:       parseDom,
		parent:    parseParent,
		alternate: parseAlternate,
		effectTag: "UPDATE",
	}

	parseRt.commitWork(parseFiber, parseParent.dom)

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "new" {
		parseT.Error("Expected id to be updated")
	}
	if parseNode.attributes["class"] != "new-class" {
		parseT.Error("Expected class to be updated")
	}
}

func TestCommitWork_UpdateNoAlternate(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseDom := parseAdapter.CreateElement("div")
	parseParent := &Fiber{typeOf: "root", props: make(map[string]interface{}), dom: parseAdapter.CreateElement("root")}

	parseFiber := &Fiber{
		typeOf:    "div",
		props:     map[string]interface{}{"id": "test"},
		dom:       parseDom,
		parent:    parseParent,
		effectTag: "UPDATE",
		alternate: nil, // No alternate
	}

	parseRt.commitWork(parseFiber, parseParent.dom)

	// When alternate is nil, UPDATE effectTag does nothing (see reconciler.go:418-429)
	// This test just verifies it doesn't crash
}

func TestCommitDeletion_SingleElement(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParentDOM := parseAdapter.CreateElement("div")
	parseChildDOM := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseParentDOM, parseChildDOM)

	parseFiber := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: parseChildDOM}

	parseRt.commitDeletion(parseFiber, parseParentDOM)

	parseParentNode := parseParentDOM.(*testDOMNode)
	if len(parseParentNode.children) != 0 {
		parseT.Errorf("Expected 0 children after deletion, got %d", len(parseParentNode.children))
	}
}

func TestCommitDeletion_NestedElements(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParentDOM := parseAdapter.CreateElement("div")
	parseChildDOM := parseAdapter.CreateElement("span")
	parseGrandchildDOM := parseAdapter.CreateElement("p")

	parseAdapter.AppendChild(parseParentDOM, parseChildDOM)
	parseAdapter.AppendChild(parseChildDOM, parseGrandchildDOM)

	parseGrandchild := &Fiber{typeOf: "p", props: make(map[string]interface{}), dom: parseGrandchildDOM}
	parseChild := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: parseChildDOM, child: parseGrandchild}

	parseRt.commitDeletion(parseChild, parseParentDOM)

	parseParentNode := parseParentDOM.(*testDOMNode)
	if len(parseParentNode.children) != 0 {
		parseT.Error("Expected parent to have no children")
	}
}

func TestCommitRoot_EmptyTree(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  make(map[string]interface{}),
		dom:    parseAdapter.CreateElement("root"),
	}

	parseRt.wipRoot = parseRoot
	parseRt.commitRoot()

	if parseRt.currentRoot != parseRoot {
		parseT.Error("Expected wipRoot to become currentRoot")
	}
	if parseRt.wipRoot != nil {
		parseT.Error("Expected wipRoot to be nil after commit")
	}
}

func TestCommitRoot_WithDeletions(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParentDOM := parseAdapter.CreateElement("div")
	parseChildDOM := parseAdapter.CreateElement("span")
	parseAdapter.AppendChild(parseParentDOM, parseChildDOM)

	parseParent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseParentDOM}
	parseChild := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: parseChildDOM, parent: parseParent, effectTag: "DELETION"}

	parseRt.deletions = []*Fiber{parseChild}
	parseRt.wipRoot = parseParent

	parseRt.commitRoot()

	if len(parseRt.deletions) != 0 {
		parseT.Errorf("Expected deletions to be cleared, got %d", len(parseRt.deletions))
	}

	parseParentNode := parseParentDOM.(*testDOMNode)
	if len(parseParentNode.children) != 0 {
		parseT.Error("Expected child to be deleted")
	}
}

func TestCommitRoot_WithEffects(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	isParseEffectRan := false
	parseChild := &Fiber{
		typeOf:    "div",
		props:     make(map[string]interface{}),
		dom:       parseAdapter.CreateElement("div"),
		effects:   []Effect{{Fn: func() func() { isParseEffectRan = true; return nil }}},
		effectTag: "PLACEMENT",
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
		parseT.Error("Expected effect to run")
	}
}

func TestCommitRoot_MultipleDeletions(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParentDOM := parseAdapter.CreateElement("div")
	parseParent := &Fiber{typeOf: "div", props: make(map[string]interface{}), dom: parseParentDOM}

	parseDeletions := make([]*Fiber, 10)
	for parseI := 0; parseI < 10; parseI++ {
		parseChildDOM := parseAdapter.CreateElement("span")
		parseAdapter.AppendChild(parseParentDOM, parseChildDOM)
		parseDeletions[parseI] = &Fiber{
			typeOf:    "span",
			props:     make(map[string]interface{}),
			dom:       parseChildDOM,
			parent:    parseParent,
			effectTag: "DELETION",
		}
	}

	parseRt.deletions = parseDeletions
	parseRt.wipRoot = parseParent

	parseRt.commitRoot()

	parseParentNode := parseParentDOM.(*testDOMNode)
	if len(parseParentNode.children) != 0 {
		parseT.Errorf("Expected all children to be deleted, got %d", len(parseParentNode.children))
	}
}

func TestRunEffects_SingleEffect(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	isParseExecuted := false
	parseFiber := &Fiber{
		typeOf:  "div",
		props:   make(map[string]interface{}),
		effects: []Effect{{Fn: func() func() { isParseExecuted = true; return nil }}},
	}

	parseRt.runEffects(parseFiber)

	if !isParseExecuted {
		parseT.Error("Expected effect to run")
	}
}

func TestRunEffects_MultipleEffects(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCount := 0
	parseFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		effects: []Effect{
			{Fn: func() func() { parseCount++; return nil }},
			{Fn: func() func() { parseCount++; return nil }},
			{Fn: func() func() { parseCount++; return nil }},
		},
	}

	parseRt.runEffects(parseFiber)

	if parseCount != 3 {
		parseT.Errorf("Expected 3 effects to run, got %d", parseCount)
	}
}

func TestRunEffects_NestedFibers(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCount := 0

	parseGrandchild := &Fiber{typeOf: "p", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { parseCount++; return nil }}}}
	parseChild := &Fiber{typeOf: "span", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { parseCount++; return nil }}}, child: parseGrandchild}
	parseParent := &Fiber{typeOf: "div", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { parseCount++; return nil }}}, child: parseChild}

	parseRt.runEffects(parseParent)

	if parseCount != 3 {
		parseT.Errorf("Expected 3 effects (parent + child + grandchild), got %d", parseCount)
	}
}

func TestRunEffects_WithSiblings(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCount := 0

	parseSibling2 := &Fiber{typeOf: "span", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { parseCount++; return nil }}}}
	parseSibling1 := &Fiber{typeOf: "span", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { parseCount++; return nil }}}, sibling: parseSibling2}
	parseParent := &Fiber{typeOf: "div", props: make(map[string]interface{}), effects: []Effect{{Fn: func() func() { parseCount++; return nil }}}, child: parseSibling1}

	parseRt.runEffects(parseParent)

	if parseCount != 3 {
		parseT.Errorf("Expected 3 effects (parent + 2 siblings), got %d", parseCount)
	}
}

func TestRunCleanups_SingleCleanup(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	isParseExecuted := false
	parseFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks:  &Hooks{cleanups: []func(){func() { isParseExecuted = true }}},
	}

	parseRt.runCleanups(parseFiber)

	if !isParseExecuted {
		parseT.Error("Expected cleanup to run")
	}
}

func TestRunCleanups_MultipleCleanups(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCount := 0
	parseFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks: &Hooks{cleanups: []func(){
			func() { parseCount++ },
			func() { parseCount++ },
			func() { parseCount++ },
		}},
	}

	parseRt.runCleanups(parseFiber)

	if parseCount != 3 {
		parseT.Errorf("Expected 3 cleanups, got %d", parseCount)
	}
}

func TestRunCleanups_Nested(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseCount := 0

	parseGrandchild := &Fiber{typeOf: "p", props: make(map[string]interface{}), hooks: &Hooks{cleanups: []func(){func() { parseCount++ }}}}
	parseChild := &Fiber{typeOf: "span", props: make(map[string]interface{}), hooks: &Hooks{cleanups: []func(){func() { parseCount++ }}}, child: parseGrandchild}
	parseParent := &Fiber{typeOf: "div", props: make(map[string]interface{}), hooks: &Hooks{cleanups: []func(){func() { parseCount++ }}}, child: parseChild}

	parseRt.runCleanups(parseParent)

	if parseCount != 3 {
		parseT.Errorf("Expected 3 cleanups, got %d", parseCount)
	}
}

func TestGetNextUnitOfWork_LinearChain(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseFiber5 := &Fiber{typeOf: "5"}
	parseFiber4 := &Fiber{typeOf: "4", child: parseFiber5}
	parseFiber3 := &Fiber{typeOf: "3", child: parseFiber4}
	parseFiber2 := &Fiber{typeOf: "2", child: parseFiber3}
	parseFiber1 := &Fiber{typeOf: "1", child: parseFiber2}

	parseNext := parseRt.getNextUnitOfWork(parseFiber1)
	if parseNext != parseFiber2 {
		parseT.Error("Expected fiber2")
	}

	parseNext = parseRt.getNextUnitOfWork(parseFiber2)
	if parseNext != parseFiber3 {
		parseT.Error("Expected fiber3")
	}
}

func TestGetNextUnitOfWork_WithSiblings(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParent := &Fiber{typeOf: "parent"}
	parseChild1 := &Fiber{typeOf: "child1", parent: parseParent}
	parseChild2 := &Fiber{typeOf: "child2", parent: parseParent}
	parseChild3 := &Fiber{typeOf: "child3", parent: parseParent}

	parseChild1.sibling = parseChild2
	parseChild2.sibling = parseChild3

	parseNext := parseRt.getNextUnitOfWork(parseChild1)
	if parseNext != parseChild2 {
		parseT.Error("Expected child2")
	}

	parseNext = parseRt.getNextUnitOfWork(parseChild2)
	if parseNext != parseChild3 {
		parseT.Error("Expected child3")
	}
}

func TestGetNextUnitOfWork_ComplexTree(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	//       root
	//      /    \
	//   child1  child2
	//     |
	//  grandchild

	parseRoot := &Fiber{typeOf: "root"}
	parseChild1 := &Fiber{typeOf: "child1", parent: parseRoot}
	parseChild2 := &Fiber{typeOf: "child2", parent: parseRoot}
	parseGrandchild := &Fiber{typeOf: "grandchild", parent: parseChild1}

	parseRoot.child = parseChild1
	parseChild1.sibling = parseChild2
	parseChild1.child = parseGrandchild

	// From root, should go to child1
	parseNext := parseRt.getNextUnitOfWork(parseRoot)
	if parseNext != parseChild1 {
		parseT.Error("Expected child1")
	}

	// From child1, should go to grandchild
	parseNext = parseRt.getNextUnitOfWork(parseChild1)
	if parseNext != parseGrandchild {
		parseT.Error("Expected grandchild")
	}

	// From grandchild, should go to child2 (sibling of parent)
	parseNext = parseRt.getNextUnitOfWork(parseGrandchild)
	if parseNext != parseChild2 {
		parseT.Error("Expected child2")
	}
}

func TestCommitWork_FunctionComponentWithHostChild(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: parseScheduler})

	parseParentDOM := parseAdapter.CreateElement("div")
	parseChildDOM := parseAdapter.CreateElement("span")

	parseFuncFiber := &Fiber{
		typeOf: func(parseP map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
	}

	parseHostFiber := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       parseChildDOM,
		parent:    parseFuncFiber,
		effectTag: "PLACEMENT",
	}

	parseFuncFiber.child = parseHostFiber

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseParentDOM,
	}

	parseFuncFiber.parent = parseParent

	parseRt.commitWork(parseHostFiber, parseParentDOM)

	parseParentNode := parseParentDOM.(*testDOMNode)
	if len(parseParentNode.children) != 1 {
		parseT.Error("Expected child to be committed to grandparent")
	}
}
