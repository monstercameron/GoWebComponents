package runtime

import (
	"strings"
	"testing"
	"time"
)

func TestReconcileChildrenReportsMissingKeyWhenMixedWithKeyedSiblings(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := &Runtime{}
	parseParent := &Fiber{typeOf: "div"}
	parseElements := []interface{}{
		CreateElement("li", map[string]interface{}{"key": "a"}),
		CreateElement("li", nil),
	}

	parseRt.reconcileChildren(parseParent, parseElements)

	parseDiagnostics := GetDiagnostics()
	if len(parseDiagnostics) == 0 {
		parseT.Fatal("expected missing-key diagnostic to be reported")
	}
}

func TestReconcileChildrenSkipsMissingKeyDiagnosticForFullyUnkeyedList(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseRt := &Runtime{}
	parseParent := &Fiber{typeOf: "div"}
	parseElements := []interface{}{
		CreateElement("li", nil),
		CreateElement("li", nil),
	}

	parseRt.reconcileChildren(parseParent, parseElements)

	if parseDiagnostics := GetDiagnostics(); len(parseDiagnostics) != 0 {
		parseT.Fatalf("expected no missing-key diagnostic for fully unkeyed list, got %+v", parseDiagnostics)
	}
}

// Test mock DOM node
type testDOMNode struct {
	nodeType   string
	tag        string
	text       string
	attributes map[string]string
	properties map[string]interface{}
	styles     map[string]string
	children   []DOMNode
	parent     *testDOMNode
	isNull     bool
}

func (parseN *testDOMNode) IsNull() bool {
	return parseN == nil || parseN.isNull
}

func (parseN *testDOMNode) Equals(parseOther DOMNode) bool {
	if IsDOMNodeNull(parseN) {
		return IsDOMNodeNull(parseOther)
	}
	return parseN == parseOther
}

// Test mock DOM adapter
type testDOMAdapter struct {
	elements map[string]*testDOMNode
	nextID   int
}

func newTestDOMAdapter() *testDOMAdapter {
	return &testDOMAdapter{
		elements: make(map[string]*testDOMNode),
		nextID:   1,
	}
}

func (parseA *testDOMAdapter) CreateElement(parseTag string) DOMNode {
	parseNode := &testDOMNode{
		nodeType:   "element",
		tag:        parseTag,
		attributes: make(map[string]string),
		properties: make(map[string]interface{}),
		styles:     make(map[string]string),
		children:   make([]DOMNode, 0),
	}
	return parseNode
}

func (parseA *testDOMAdapter) CreateTextNode(parseText string) DOMNode {
	parseNode := &testDOMNode{
		nodeType:   "text",
		text:       parseText,
		attributes: make(map[string]string),
		properties: make(map[string]interface{}),
		styles:     make(map[string]string),
		children:   make([]DOMNode, 0),
	}
	return parseNode
}

func (parseA *testDOMAdapter) AppendChild(parseParent, parseChild DOMNode) {
	if parseP, parseOk := parseParent.(*testDOMNode); parseOk {
		parseP.children = append(parseP.children, parseChild)
		if parseC, parseOk2 := parseChild.(*testDOMNode); parseOk2 {
			parseC.parent = parseP
		}
	}
}

func (parseA *testDOMAdapter) RemoveChild(parseParent, parseChild DOMNode) {
	if parseP, parseOk := parseParent.(*testDOMNode); parseOk {
		for parseI, parseC := range parseP.children {
			if parseC.Equals(parseChild) {
				if parseRemoved, parseOk2 := parseC.(*testDOMNode); parseOk2 {
					parseRemoved.parent = nil
				}
				parseP.children = append(parseP.children[:parseI], parseP.children[parseI+1:]...)
				break
			}
		}
	}
}

func (parseA *testDOMAdapter) InsertBefore(parseParent, parseNewChild, parseRefChild DOMNode) {
	if parseP, parseOk := parseParent.(*testDOMNode); parseOk {
		for parseI, parseC := range parseP.children {
			if parseC.Equals(parseRefChild) {
				parseP.children = append(parseP.children[:parseI], append([]DOMNode{parseNewChild}, parseP.children[parseI:]...)...)
				if parseInserted, parseOk2 := parseNewChild.(*testDOMNode); parseOk2 {
					parseInserted.parent = parseP
				}
				return
			}
		}
		parseP.children = append(parseP.children, parseNewChild)
		if parseInserted2, parseOk3 := parseNewChild.(*testDOMNode); parseOk3 {
			parseInserted2.parent = parseP
		}
	}
}

func (parseA *testDOMAdapter) ReplaceChild(parseParent, parseNewChild, parseOldChild DOMNode) {
	if parseP, parseOk := parseParent.(*testDOMNode); parseOk {
		for parseI, parseC := range parseP.children {
			if parseC.Equals(parseOldChild) {
				parseP.children[parseI] = parseNewChild
				if parseReplaced, parseOk2 := parseOldChild.(*testDOMNode); parseOk2 {
					parseReplaced.parent = nil
				}
				if parseInserted, parseOk3 := parseNewChild.(*testDOMNode); parseOk3 {
					parseInserted.parent = parseP
				}
				return
			}
		}
	}
}

func (parseA *testDOMAdapter) GetParent(parseNode DOMNode) DOMNode {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		return parseN.parent
	}
	return nil
}

func (parseA *testDOMAdapter) GetParentNode(parseNode DOMNode) DOMNode {
	return nil
}

func (parseA *testDOMAdapter) GetFirstChild(parseNode DOMNode) DOMNode {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk && len(parseN.children) > 0 {
		return parseN.children[0]
	}
	return nil
}

func (parseA *testDOMAdapter) GetNextSibling(parseNode DOMNode) DOMNode {
	parseN, parseOk := parseNode.(*testDOMNode)
	if !parseOk || parseN.parent == nil {
		return nil
	}
	for parseIndex, parseChild := range parseN.parent.children {
		if parseChild.Equals(parseNode) && parseIndex+1 < len(parseN.parent.children) {
			return parseN.parent.children[parseIndex+1]
		}
	}
	return nil
}

func (parseA *testDOMAdapter) GetChildren(parseNode DOMNode) []DOMNode {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		return parseN.children
	}
	return nil
}

func (parseA *testDOMAdapter) SetAttribute(parseNode DOMNode, parseName, parseValue string) {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		parseN.attributes[parseName] = parseValue
	}
}

func (parseA *testDOMAdapter) RemoveAttribute(parseNode DOMNode, parseName string) {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		delete(parseN.attributes, parseName)
	}
}

func (parseA *testDOMAdapter) SetProperty(parseNode DOMNode, parseName string, parseValue interface{}) {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		parseN.properties[parseName] = parseValue
	}
}

func (parseA *testDOMAdapter) GetProperty(parseNode DOMNode, parseName string) interface{} {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		switch parseName {
		case "nodeType":
			if parseN.nodeType == "text" {
				return 3
			}
			return 1
		case "tagName":
			return strings.ToUpper(parseN.tag)
		case "nodeName":
			if parseN.nodeType == "text" {
				return "#text"
			}
			return strings.ToUpper(parseN.tag)
		case "textContent":
			if parseN.nodeType == "text" {
				return parseN.text
			}
			return parseN.text
		case "className":
			return parseN.attributes["class"]
		case "htmlFor":
			return parseN.attributes["for"]
		}
		if parseValue, parseOk2 := parseN.attributes[parseName]; parseOk2 {
			return parseValue
		}
		return parseN.properties[parseName]
	}
	return nil
}

func (parseA *testDOMAdapter) SetStyle(parseNode DOMNode, parseProperty, parseValue string) {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		parseN.styles[parseProperty] = parseValue
	}
}

func (parseA *testDOMAdapter) SetStyles(parseNode DOMNode, parseStyles map[string]string) {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		for parseK, parseV := range parseStyles {
			parseN.styles[parseK] = parseV
		}
	}
}

func (parseA *testDOMAdapter) SetInnerHTML(parseNode DOMNode, parseHtml string) {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		for _, parseChild := range parseN.children {
			if parseC, parseOk2 := parseChild.(*testDOMNode); parseOk2 {
				parseC.parent = nil
			}
		}
		parseN.children = make([]DOMNode, 0)
	}
}

func (parseA *testDOMAdapter) SetTextContent(parseNode DOMNode, parseText string) {
	if parseN, parseOk := parseNode.(*testDOMNode); parseOk {
		parseN.text = parseText
		if parseN.nodeType == "text" {
			return
		}
		if len(parseN.children) == 1 {
			if parseTextChild, parseOk2 := parseN.children[0].(*testDOMNode); parseOk2 && parseTextChild.nodeType == "text" {
				if parseText == "" {
					parseTextChild.parent = nil
					parseN.children = parseN.children[:0]
					return
				}
				parseTextChild.text = parseText
				return
			}
		}
		for _, parseChild := range parseN.children {
			if parseTextChild, parseOk2 := parseChild.(*testDOMNode); parseOk2 {
				parseTextChild.parent = nil
			}
		}
		parseN.children = parseN.children[:0]
		if parseText == "" {
			return
		}
		parseChildNode := &testDOMNode{
			nodeType:   "text",
			text:       parseText,
			attributes: make(map[string]string),
			properties: make(map[string]interface{}),
			styles:     make(map[string]string),
			children:   make([]DOMNode, 0),
			parent:     parseN,
		}
		parseN.children = append(parseN.children, parseChildNode)
	}
}

func (parseA *testDOMAdapter) WrapFunction(parseFn interface{}) interface{} {
	return parseFn
}

func TestCreateElement(parseT *testing.T) {
	parseElem := CreateElement("div", map[string]interface{}{"id": "test"}, "child1", "child2")

	if parseElem.Type != "div" {
		parseT.Errorf("Expected type 'div', got %v", parseElem.Type)
	}

	if parseElem.Props["id"] != "test" {
		parseT.Errorf("Expected id 'test', got %v", parseElem.Props["id"])
	}

	if len(parseElem.Children) != 2 {
		parseT.Errorf("Expected 2 children, got %d", len(parseElem.Children))
	}
}

func TestCreateElement_NilProps(parseT *testing.T) {
	parseElem := CreateElement("span", nil, "text")

	if parseElem.Type != "span" {
		parseT.Errorf("Expected type 'span', got %v", parseElem.Type)
	}

	if len(parseElem.Props) == 0 {
		parseT.Error("Expected props map to be initialized")
	}
}

func TestCreateElement_NoChildren(parseT *testing.T) {
	parseElem := CreateElement("input", map[string]interface{}{"type": "text"})

	if len(parseElem.Children) != 0 {
		parseT.Errorf("Expected 0 children, got %d", len(parseElem.Children))
	}
}

func TestIsSameType_StringTypes(parseT *testing.T) {
	if !isSameType("div", "div") {
		parseT.Error("Expected same string types to match")
	}

	if isSameType("div", "span") {
		parseT.Error("Expected different string types to not match")
	}

	if isSameType("div", 123) {
		parseT.Error("Expected string and non-string to not match")
	}
}

func TestIsSameType_FunctionTypes(parseT *testing.T) {
	parseFn1 := func(parseP map[string]interface{}) *Element { return nil }
	parseFn2 := func(parseP2 map[string]interface{}) *Element { return nil }
	parseClosureFactory := func(parseLabel string) func(map[string]interface{}) *Element {
		return func(parseProps map[string]interface{}) *Element {
			return CreateElement("div", nil, parseLabel)
		}
	}
	parseClosure1 := parseClosureFactory("one")
	parseClosure2 := parseClosureFactory("two")

	if !isSameType(parseFn1, parseFn1) {
		parseT.Error("Expected same function to match itself")
	}

	// Different function instances should not match
	if isSameType(parseFn1, parseFn2) {
		parseT.Error("Expected different function instances to not match")
	}

	if isSameType(parseClosure1, parseClosure2) {
		parseT.Error("Expected different closure instances to not match")
	}
}

func TestIsSameType_NilTypes(parseT *testing.T) {
	if isSameType(nil, nil) {
		parseT.Error("Expected nil types to not match")
	}

	if isSameType(nil, "div") {
		parseT.Error("Expected nil and non-nil to not match")
	}
}

func TestReconcileChildren_NewChildren(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	parseChild1 := CreateElement("span", map[string]interface{}{"id": "1"})
	parseChild2 := CreateElement("p", map[string]interface{}{"id": "2"})

	parseRt.reconcileChildren(parseParent, []interface{}{parseChild1, parseChild2})

	if parseParent.child == nil {
		parseT.Fatal("Expected parent to have a child")
	}

	if parseParent.child.typeOf != "span" {
		parseT.Errorf("Expected first child to be span, got %v", parseParent.child.typeOf)
	}

	if parseParent.child.effectTag != "PLACEMENT" {
		parseT.Errorf("Expected PLACEMENT tag, got %s", parseParent.child.effectTag)
	}

	if parseParent.child.sibling == nil {
		parseT.Fatal("Expected first child to have a sibling")
	}

	if parseParent.child.sibling.typeOf != "p" {
		parseT.Errorf("Expected second child to be p, got %v", parseParent.child.sibling.typeOf)
	}
}

func TestReconcileChildren_UpdateExisting(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseOldChild := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"className": "old"},
		dom:    parseMockDOM.CreateElement("div"),
	}

	parseParent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: parseOldChild,
		},
	}

	parseNewChild := CreateElement("div", map[string]interface{}{"className": "new"})

	parseRt.reconcileChildren(parseParent, []interface{}{parseNewChild})

	if parseParent.child == nil {
		parseT.Fatal("Expected parent to have a child")
	}

	if parseParent.child.effectTag != "UPDATE" {
		parseT.Errorf("Expected UPDATE tag for same type, got %s", parseParent.child.effectTag)
	}

	if parseParent.child.props["className"] != "new" {
		parseT.Errorf("Expected new props, got %v", parseParent.child.props["className"])
	}

	if parseParent.child.dom != parseOldChild.dom {
		parseT.Error("Expected DOM node to be reused")
	}
}

func TestReconcileChildren_DeleteOldChildren(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseOldChild1 := &Fiber{typeOf: "div", props: make(map[string]interface{})}
	parseOldChild2 := &Fiber{typeOf: "span", props: make(map[string]interface{})}
	parseOldChild1.sibling = parseOldChild2

	parseParent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: parseOldChild1,
		},
	}

	// Only one new child - second should be deleted
	parseNewChild := CreateElement("div", map[string]interface{}{})

	parseRt.reconcileChildren(parseParent, []interface{}{parseNewChild})

	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}

	if parseRt.deletions[0].typeOf != "span" {
		parseT.Errorf("Expected span to be deleted, got %v", parseRt.deletions[0].typeOf)
	}

	if parseRt.deletions[0].effectTag != "DELETION" {
		parseT.Errorf("Expected DELETION tag, got %s", parseRt.deletions[0].effectTag)
	}
}

func TestReconcileChildren_ReplaceWithDifferentType(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseOldChild := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseMockDOM.CreateElement("div"),
	}

	parseParent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: parseOldChild,
		},
	}

	parseNewChild := CreateElement("span", map[string]interface{}{})

	parseRt.reconcileChildren(parseParent, []interface{}{parseNewChild})

	// Old child should be marked for deletion
	if len(parseRt.deletions) != 1 {
		parseT.Errorf("Expected 1 deletion, got %d", len(parseRt.deletions))
	}

	if parseRt.deletions[0].typeOf != "div" {
		parseT.Errorf("Expected div to be deleted, got %v", parseRt.deletions[0].typeOf)
	}

	// New child should be placed
	if parseParent.child.typeOf != "span" {
		parseT.Errorf("Expected new child to be span, got %v", parseParent.child.typeOf)
	}

	if parseParent.child.effectTag != "PLACEMENT" {
		parseT.Errorf("Expected PLACEMENT tag, got %s", parseParent.child.effectTag)
	}
}

func TestCreateDom_Element(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"id": "test", "className": "myclass"},
	}

	parseDom := parseRt.createDom(parseFiber)

	if parseDom == nil || parseDom.IsNull() {
		parseT.Fatal("Expected DOM node to be created")
	}

	// Check that properties were applied
	parseMockNode := parseDom.(*testDOMNode)
	if parseMockNode.attributes["id"] != "test" {
		parseT.Errorf("Expected id='test', got %v", parseMockNode.attributes["id"])
	}
}

func TestCreateDom_TextElement(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseFiber := &Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]interface{}{"nodeValue": "Hello World"},
	}

	parseDom := parseRt.createDom(parseFiber)

	if parseDom == nil || parseDom.IsNull() {
		parseT.Fatal("Expected text node to be created")
	}

	parseMockNode := parseDom.(*testDOMNode)
	if parseMockNode.nodeType != "text" {
		parseT.Errorf("Expected text node type, got %s", parseMockNode.nodeType)
	}
}

func TestUpdateDomProperties_SetProperties(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseDom := parseMockDOM.CreateElement("div")

	parseOldProps := map[string]interface{}{}
	parseNewProps := map[string]interface{}{
		"id":        "test",
		"className": "myclass",
		"disabled":  true,
	}

	parseRt.updateDomProperties(parseDom, parseOldProps, parseNewProps)

	parseMockNode := parseDom.(*testDOMNode)
	if parseMockNode.attributes["id"] != "test" {
		parseT.Errorf("Expected id='test', got %v", parseMockNode.attributes["id"])
	}

	if parseMockNode.attributes["class"] != "myclass" {
		parseT.Errorf("Expected class='myclass', got %v", parseMockNode.attributes["class"])
	}
}

func TestUpdateDomProperties_RemoveOldProperties(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseDom := parseMockDOM.CreateElement("div")

	parseOldProps := map[string]interface{}{
		"id":      "old",
		"oldProp": "value",
	}
	parseNewProps := map[string]interface{}{
		"id": "new",
	}

	parseRt.updateDomProperties(parseDom, parseOldProps, parseNewProps)

	parseMockNode := parseDom.(*testDOMNode)
	if parseMockNode.attributes["id"] != "new" {
		parseT.Errorf("Expected id='new', got %v", parseMockNode.attributes["id"])
	}

	if _, parseExists := parseMockNode.attributes["oldProp"]; parseExists {
		parseT.Error("Expected oldProp to be removed")
	}
}

func TestUpdateDomProperties_Styles(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseDom := parseMockDOM.CreateElement("div")

	parseOldProps := map[string]interface{}{}
	parseNewProps := map[string]interface{}{
		"style": map[string]string{
			"color":      "red",
			"background": "blue",
		},
	}

	parseRt.updateDomProperties(parseDom, parseOldProps, parseNewProps)

	parseMockNode := parseDom.(*testDOMNode)
	if parseMockNode.styles["color"] != "red" {
		parseT.Errorf("Expected color='red', got %v", parseMockNode.styles["color"])
	}
}

func TestUpdateDomProperties_NilDom(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	// Should not panic with nil DOM
	parseRt.updateDomProperties(nil, map[string]interface{}{}, map[string]interface{}{"id": "test"})
}

func TestCommitRoot_ProcessesDeletions(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParentDOM := parseMockDOM.CreateElement("div")
	parseChildDOM := parseMockDOM.CreateElement("span")
	parseMockDOM.AppendChild(parseParentDOM, parseChildDOM)

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseParentDOM,
	}

	parseChild := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       parseChildDOM,
		parent:    parseParent,
		effectTag: "DELETION",
	}

	parseRt.deletions = []*Fiber{parseChild}
	parseRt.wipRoot = parseParent

	parseRt.commitRoot()

	if len(parseRt.deletions) != 0 {
		parseT.Errorf("Expected deletions to be cleared, got %d", len(parseRt.deletions))
	}

	// Check that child was removed
	parseMockParent := parseParentDOM.(*testDOMNode)
	if len(parseMockParent.children) != 0 {
		parseT.Errorf("Expected child to be removed, got %d children", len(parseMockParent.children))
	}
}

func TestCommitWork_Placement(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParentDOM := parseMockDOM.CreateElement("div")
	parseChildDOM := parseMockDOM.CreateElement("span")

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseParentDOM,
	}

	parseChild := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       parseChildDOM,
		parent:    parseParent,
		effectTag: "PLACEMENT",
	}

	parseRt.commitWork(parseChild, parseParentDOM)

	parseMockParent := parseParentDOM.(*testDOMNode)
	if len(parseMockParent.children) != 1 {
		parseT.Errorf("Expected 1 child, got %d", len(parseMockParent.children))
	}

	if parseMockParent.children[0] != parseChildDOM {
		parseT.Error("Expected child to be appended to parent")
	}
}

func TestCommitWork_Update(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseDom := parseMockDOM.CreateElement("div")

	parseParent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		dom:    parseMockDOM.CreateElement("root"),
	}

	parseAlternate := &Fiber{
		props: map[string]interface{}{"id": "old"},
	}

	parseFiber := &Fiber{
		typeOf:    "div",
		props:     map[string]interface{}{"id": "new"},
		dom:       parseDom,
		parent:    parseParent,
		alternate: parseAlternate,
		effectTag: "UPDATE",
	}

	parseRt.commitWork(parseFiber, parseParent.dom)

	parseMockNode := parseDom.(*testDOMNode)
	if parseMockNode.attributes["id"] != "new" {
		parseT.Errorf("Expected id='new', got %v", parseMockNode.attributes["id"])
	}
}

func TestCommitWork_RecursiveCommit(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParentDOM := parseMockDOM.CreateElement("div")
	parseChild1DOM := parseMockDOM.CreateElement("span")
	parseChild2DOM := parseMockDOM.CreateElement("p")

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parseParentDOM,
	}

	parseChild1 := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       parseChild1DOM,
		parent:    parseParent,
		effectTag: "PLACEMENT",
	}

	parseChild2 := &Fiber{
		typeOf:    "p",
		props:     make(map[string]interface{}),
		dom:       parseChild2DOM,
		parent:    parseParent,
		effectTag: "PLACEMENT",
	}

	parseChild1.sibling = parseChild2

	parseRt.commitWork(parseChild1, parseParentDOM)

	parseMockParent := parseParentDOM.(*testDOMNode)
	if len(parseMockParent.children) != 2 {
		parseT.Errorf("Expected 2 children, got %d", len(parseMockParent.children))
	}
}

func TestCommitDeletion_WithDOM(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParentDOM := parseMockDOM.CreateElement("div")
	parseChildDOM := parseMockDOM.CreateElement("span")
	parseMockDOM.AppendChild(parseParentDOM, parseChildDOM)

	parseFiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    parseChildDOM,
	}

	parseRt.commitDeletion(parseFiber, parseParentDOM)

	parseMockParent := parseParentDOM.(*testDOMNode)
	if len(parseMockParent.children) != 0 {
		parseT.Errorf("Expected child to be removed, got %d children", len(parseMockParent.children))
	}
}

func TestCommitDeletion_FunctionComponent(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParentDOM := parseMockDOM.CreateElement("div")
	parseChildDOM := parseMockDOM.CreateElement("span")
	parseMockDOM.AppendChild(parseParentDOM, parseChildDOM)

	// Function component with host component child
	parseFuncFiber := &Fiber{
		typeOf: func(parseP map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
	}

	parseHostFiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    parseChildDOM,
		parent: parseFuncFiber,
	}

	parseFuncFiber.child = parseHostFiber

	parseRt.commitDeletion(parseFuncFiber, parseParentDOM)

	parseMockParent := parseParentDOM.(*testDOMNode)
	if len(parseMockParent.children) != 0 {
		parseT.Errorf("Expected child to be removed, got %d children", len(parseMockParent.children))
	}
}

func TestCommitDeletion_FunctionComponentPreservesSiblingComponentDOM(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParentDOM := parseMockDOM.CreateElement("div")
	parseDeletedDOM := parseMockDOM.CreateElement("span")
	parseKeptDOM := parseMockDOM.CreateElement("p")
	parseMockDOM.AppendChild(parseParentDOM, parseDeletedDOM)
	parseMockDOM.AppendChild(parseParentDOM, parseKeptDOM)

	parseDeletedFiber := &Fiber{typeOf: func(parseP map[string]interface{}) *Element { return nil }, props: make(map[string]interface{})}
	parseDeletedChild := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: parseDeletedDOM, parent: parseDeletedFiber}
	parseDeletedFiber.child = parseDeletedChild

	parseKeptFiber := &Fiber{typeOf: func(parseP2 map[string]interface{}) *Element { return nil }, props: make(map[string]interface{})}
	parseKeptChild := &Fiber{typeOf: "p", props: make(map[string]interface{}), dom: parseKeptDOM, parent: parseKeptFiber}
	parseKeptFiber.child = parseKeptChild
	parseDeletedFiber.sibling = parseKeptFiber

	parseRt.commitDeletion(parseDeletedFiber, parseParentDOM)

	parseChildren := parseParentDOM.(*testDOMNode).children
	if len(parseChildren) != 1 {
		parseT.Fatalf("expected sibling component DOM to remain, got %d children", len(parseChildren))
	}
	if parseChildren[0] != parseKeptDOM {
		parseT.Fatal("expected sibling component DOM node to be preserved")
	}
}

func TestRunEffects(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	isParseExecuted1 := false
	isParseExecuted2 := false

	parseFiber1 := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		effects: []Effect{
			{Fn: func() func() { isParseExecuted1 = true; return nil }, CleanupIndex: 0},
		},
		hooks: &Hooks{cleanups: make([]func(), 1)},
	}

	parseFiber2 := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		effects: []Effect{
			{Fn: func() func() { isParseExecuted2 = true; return nil }, CleanupIndex: 0},
		},
		hooks: &Hooks{cleanups: make([]func(), 1)},
	}

	parseFiber1.child = parseFiber2

	parseRt.runEffects(parseFiber1)

	if !isParseExecuted1 {
		parseT.Error("Expected effect 1 to be executed")
	}

	if !isParseExecuted2 {
		parseT.Error("Expected effect 2 to be executed")
	}
}

func TestRunEffects_NilFiber(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	// Should not panic
	parseRt.runEffects(nil)
}

func TestRunEffects_ReportsSlowEffectDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseMockDOM, Scheduler: parseScheduler})

	parseFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		effects: []Effect{{
			Fn: func() func() {
				time.Sleep(5 * time.Millisecond)
				return nil
			},
			CleanupIndex: 0,
		}},
		hooks: &Hooks{cleanups: make([]func(), 1)},
	}

	parseRt.runEffects(parseFiber)
	if parseFiber.effectDurationNs < slowOperationDiagnosticThresholdNs {
		parseT.Fatalf("expected effect duration to be recorded, got %d", parseFiber.effectDurationNs)
	}
	if parseRt.profiling.effectExecutions != 1 {
		parseT.Fatalf("expected effect execution counter to increment, got %d", parseRt.profiling.effectExecutions)
	}
	isParseFound := false
	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, "slow effect") {
			isParseFound = true
			break
		}
	}
	if !isParseFound {
		parseT.Fatal("expected slow effect diagnostic to be reported")
	}
}

func TestRunCleanups_ReportsSlowCleanupDiagnostic(parseT *testing.T) {
	ClearDiagnostics()
	defer ClearDiagnostics()

	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseMockDOM, Scheduler: parseScheduler})

	parseFiber := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks: &Hooks{cleanups: []func(){func() {
			time.Sleep(5 * time.Millisecond)
		}}},
	}

	parseRt.runCleanups(parseFiber)
	if parseFiber.cleanupDurationNs < slowOperationDiagnosticThresholdNs {
		parseT.Fatalf("expected cleanup duration to be recorded, got %d", parseFiber.cleanupDurationNs)
	}
	if parseRt.profiling.cleanupExecutions != 1 {
		parseT.Fatalf("expected cleanup execution counter to increment, got %d", parseRt.profiling.cleanupExecutions)
	}
	isParseFound := false
	for _, parseDiagnostic := range GetDiagnostics() {
		if strings.Contains(parseDiagnostic.Message, "slow cleanup") {
			isParseFound = true
			break
		}
	}
	if !isParseFound {
		parseT.Fatal("expected slow cleanup diagnostic to be reported")
	}
}

func TestRunCleanups_ExecutesAllCleanups(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseMockDOM, Scheduler: parseScheduler})

	isParseExecuted1 := false
	isParseExecuted2 := false

	parseChild := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks: &Hooks{cleanups: []func(){
			func() { isParseExecuted1 = true },
		}},
	}

	parseParent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks: &Hooks{cleanups: []func(){
			func() { isParseExecuted2 = true },
		}},
		child: parseChild,
	}

	parseRt.runCleanups(parseParent)

	if !isParseExecuted1 {
		parseT.Error("Expected child cleanup to run")
	}
	if !isParseExecuted2 {
		parseT.Error("Expected parent cleanup to run")
	}
}

func TestCommitDeletion_RunsCleanupsAndCleansAtomSubs(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{DOMAdapter: parseMockDOM, Scheduler: parseScheduler})

	// Create a DOM tree
	parseParentDOM := parseMockDOM.CreateElement("div")
	parseChildDOM := parseMockDOM.CreateElement("span")
	parseMockDOM.AppendChild(parseParentDOM, parseChildDOM)

	// Create a function fiber that subscribes to an atom
	parseFuncFiber := &Fiber{typeOf: func(parseP map[string]interface{}) *Element { return nil }, props: make(map[string]interface{})}
	parseHostFiber := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: parseChildDOM, parent: parseFuncFiber}
	parseFuncFiber.child = parseHostFiber

	// Simulate using atom in child fiber via the runtime's registry
	parseRt.atomRegistry.InitAtom("test-atom", 0)
	parseRt.atomRegistry.Subscribe("test-atom", parseFuncFiber)

	// Ensure subscription is present
	if parseRt.atomRegistry.GetSubscriberCount("test-atom") != 1 {
		parseT.Fatal("expected 1 subscriber before deletion")
	}

	// Attach a cleanup to the function fiber
	isParseRan := false
	parseFuncFiber.hooks = &Hooks{cleanups: []func(){func() { isParseRan = true }}}

	parseRt.commitDeletion(parseFuncFiber, parseParentDOM)

	if !isParseRan {
		parseT.Error("Expected cleanup to run during commitDeletion")
	}

	if parseRt.atomRegistry.GetSubscriberCount("test-atom") != 0 {
		parseT.Error("Expected atom subscription to be removed during commitDeletion")
	}
}

func TestGetNextUnitOfWork_Child(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseChild := &Fiber{typeOf: "child"}
	parseParent := &Fiber{typeOf: "parent", child: parseChild}

	parseNext := parseRt.getNextUnitOfWork(parseParent)

	if parseNext != parseChild {
		parseT.Error("Expected child to be next unit of work")
	}
}

func TestGetNextUnitOfWork_Sibling(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseSibling := &Fiber{typeOf: "sibling"}
	parseParent := &Fiber{typeOf: "parent"}
	parseFiber := &Fiber{typeOf: "fiber", parent: parseParent, sibling: parseSibling}

	parseNext := parseRt.getNextUnitOfWork(parseFiber)

	if parseNext != parseSibling {
		parseT.Error("Expected sibling to be next unit of work")
	}
}

func TestGetNextUnitOfWork_ParentSibling(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParentSibling := &Fiber{typeOf: "parentSibling"}
	parseParent := &Fiber{typeOf: "parent", sibling: parseParentSibling}
	parseFiber := &Fiber{typeOf: "fiber", parent: parseParent}

	parseNext := parseRt.getNextUnitOfWork(parseFiber)

	if parseNext != parseParentSibling {
		parseT.Error("Expected parent's sibling to be next unit of work")
	}
}

func TestGetNextUnitOfWork_End(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseParent := &Fiber{typeOf: "parent"}
	parseFiber := &Fiber{typeOf: "fiber", parent: parseParent}

	parseNext := parseRt.getNextUnitOfWork(parseFiber)

	if parseNext != nil {
		parseT.Errorf("Expected nil at end of tree, got %v", parseNext)
	}
}

func TestPerformUnitOfWork_SkipNonDirty(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseSibling := &Fiber{typeOf: "sibling", dirty: true}
	parseFiber := &Fiber{typeOf: "test", dirty: false, sibling: parseSibling}

	parseNext := parseRt.performUnitOfWork(parseFiber)

	// Should skip to sibling
	if parseNext != parseSibling {
		parseT.Errorf("Expected to skip to sibling, got %v", parseNext)
	}
}

func TestPerformUnitOfWork_RootFiber(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseChild1 := CreateElement("div", map[string]interface{}{})
	parseChild2 := CreateElement("span", map[string]interface{}{})

	parseFiber := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{"children": []interface{}{parseChild1, parseChild2}},
		dirty:  true,
	}

	parseRt.performUnitOfWork(parseFiber)

	if parseFiber.child == nil {
		parseT.Fatal("Expected root to have children")
	}

	if parseFiber.child.typeOf != "div" {
		parseT.Errorf("Expected first child to be div, got %v", parseFiber.child.typeOf)
	}
}

func TestPerformUnitOfWork_HostComponent(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseChildElem := CreateElement("span", map[string]interface{}{})

	parseFiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"children": []interface{}{parseChildElem}},
		dirty:  true,
	}

	parseRt.performUnitOfWork(parseFiber)

	if parseFiber.dom == nil || parseFiber.dom.IsNull() {
		parseT.Fatal("Expected DOM node to be created")
	}

	if parseFiber.child == nil {
		parseT.Fatal("Expected fiber to have children")
	}

	if parseFiber.child.typeOf != "span" {
		parseT.Errorf("Expected child to be span, got %v", parseFiber.child.typeOf)
	}
}

func TestPerformUnitOfWork_FunctionComponent(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseComponentFn := func(parseProps map[string]interface{}) *Element {
		return CreateElement("div", map[string]interface{}{"id": "from-component"})
	}

	parseFiber := &Fiber{
		typeOf: parseComponentFn,
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	parseRt.performUnitOfWork(parseFiber)

	// Hooks are only initialized when a hook is called
	// This component doesn't use hooks, so fiber.hooks will be nil

	if parseFiber.child == nil {
		parseT.Fatal("Expected fiber to have children")
	}

	if parseFiber.child.typeOf != "div" {
		parseT.Errorf("Expected child to be div, got %v", parseFiber.child.typeOf)
	}
}

func TestPerformUnitOfWork_FunctionComponentNoProps(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseComponentFn := func() *Element {
		return CreateElement("div", map[string]interface{}{"id": "from-component"})
	}

	parseFiber := &Fiber{
		typeOf: parseComponentFn,
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	parseRt.performUnitOfWork(parseFiber)

	if parseFiber.child == nil {
		parseT.Fatal("Expected fiber to have children")
	}

	if parseFiber.child.typeOf != "div" {
		parseT.Errorf("Expected child to be div, got %v", parseFiber.child.typeOf)
	}
}

func TestPerformUnitOfWork_FunctionComponentWithHooks(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	// Component with previous render (has hooks state)
	parseOldHooks := &Hooks{
		states: []interface{}{42, 42, "test", "test"},
		deps:   make([][]interface{}, 0),
		memos:  make([]memoizedValue, 0),
		index:  0,
	}

	parseAlternate := &Fiber{
		hooks: parseOldHooks,
	}

	parseComponentFn := func(parseProps map[string]interface{}) *Element {
		return CreateElement("div", nil)
	}

	parseFiber := &Fiber{
		typeOf:    parseComponentFn,
		props:     make(map[string]interface{}),
		dirty:     true,
		alternate: parseAlternate,
	}

	parseRt.performUnitOfWork(parseFiber)

	if parseFiber.hooks == nil {
		parseT.Fatal("Expected hooks to be initialized")
	}

	if len(parseFiber.hooks.states) != 4 {
		parseT.Errorf("Expected hooks state to be preserved, got %d items", len(parseFiber.hooks.states))
	}

	if parseFiber.hooks.states[0] != 42 {
		parseT.Errorf("Expected first state to be 42, got %v", parseFiber.hooks.states[0])
	}
}

func TestPerformUnitOfWork_NilFiber(parseT *testing.T) {
	parseMockDOM := newTestDOMAdapter()
	parseScheduler := newTestScheduler()
	parseRt := NewRuntime(Config{
		DOMAdapter: parseMockDOM,
		Scheduler:  parseScheduler,
	})

	parseNext := parseRt.performUnitOfWork(nil)

	if parseNext != nil {
		parseT.Errorf("Expected nil for nil fiber, got %v", parseNext)
	}
}

func TestGetSetCurrentFiber(parseT *testing.T) {
	parseFiber := &Fiber{typeOf: "test"}

	SetCurrentFiber(parseFiber)

	if GetCurrentFiber() != parseFiber {
		parseT.Error("Expected to get the same fiber that was set")
	}

	SetCurrentFiber(nil)

	if GetCurrentFiber() != nil {
		parseT.Error("Expected nil after setting to nil")
	}
}

func TestFlattenFragments_NoFragments(parseT *testing.T) {
	parseElements := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseResult, _ := flattenFragments(parseElements)

	if len(parseResult) != 2 {
		parseT.Errorf("Expected 2 elements, got %d", len(parseResult))
	}

	if parseElem, parseOk := parseResult[0].(*Element); !parseOk || parseElem.Type != "div" {
		parseT.Error("Expected first element to be div")
	}

	if parseElem2, parseOk2 := parseResult[1].(*Element); !parseOk2 || parseElem2.Type != "span" {
		parseT.Error("Expected second element to be span")
	}
}

func TestFlattenFragments_WithFragment(parseT *testing.T) {
	// Create elements: div, Fragment(span, p), h1
	parseFragmentChildren := []interface{}{
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{Type: "p", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseElements := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": parseFragmentChildren},
			Children: []interface{}{},
		},
		&Element{Type: "h1", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseResult, _ := flattenFragments(parseElements)

	if len(parseResult) != 4 {
		parseT.Errorf("Expected 4 flattened elements, got %d", len(parseResult))
	}

	// Check order: div, span, p, h1
	parseExpected := []string{"div", "span", "p", "h1"}
	for parseI, parseExpectedType := range parseExpected {
		if parseElem, parseOk := parseResult[parseI].(*Element); !parseOk || parseElem.Type != parseExpectedType {
			parseT.Errorf("Expected element %d to be %s, got %T with type %v", parseI, parseExpectedType, parseResult[parseI], parseElem.Type)
		}
	}
}

func TestFlattenFragments_NestedFragments(parseT *testing.T) {
	// Create nested: Fragment(div, Fragment(span, p), h1)
	parseInnerFragmentChildren := []interface{}{
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{Type: "p", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseOuterFragmentChildren := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": parseInnerFragmentChildren},
			Children: []interface{}{},
		},
		&Element{Type: "h1", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseElements := []interface{}{
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": parseOuterFragmentChildren},
			Children: []interface{}{},
		},
	}

	parseResult, _ := flattenFragments(parseElements)

	if len(parseResult) != 4 {
		parseT.Errorf("Expected 4 flattened elements from nested fragments, got %d", len(parseResult))
	}

	// Check order: div, span, p, h1
	parseExpected := []string{"div", "span", "p", "h1"}
	for parseI, parseExpectedType := range parseExpected {
		if parseElem, parseOk := parseResult[parseI].(*Element); !parseOk || parseElem.Type != parseExpectedType {
			parseT.Errorf("Expected element %d to be %s, got %T with type %v", parseI, parseExpectedType, parseResult[parseI], parseElem.Type)
		}
	}
}

func TestFlattenFragments_EmptyFragment(parseT *testing.T) {
	parseElements := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{
			Type:     "FRAGMENT",
			Props:    make(map[string]interface{}),
			Children: []interface{}{},
		},
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseResult, _ := flattenFragments(parseElements)

	if len(parseResult) != 2 {
		parseT.Errorf("Expected 2 elements (empty fragment flattened away), got %d", len(parseResult))
	}

	// Check order: div, span
	parseExpected := []string{"div", "span"}
	for parseI, parseExpectedType := range parseExpected {
		if parseElem, parseOk := parseResult[parseI].(*Element); !parseOk || parseElem.Type != parseExpectedType {
			parseT.Errorf("Expected element %d to be %s, got %T with type %v", parseI, parseExpectedType, parseResult[parseI], parseElem.Type)
		}
	}
}

func TestFlattenFragments_OnlyFragments(parseT *testing.T) {
	// Only fragments containing elements
	parseFragment1Children := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseFragment2Children := []interface{}{
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	parseElements := []interface{}{
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": parseFragment1Children},
			Children: []interface{}{},
		},
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": parseFragment2Children},
			Children: []interface{}{},
		},
	}

	parseResult, _ := flattenFragments(parseElements)

	if len(parseResult) != 2 {
		parseT.Errorf("Expected 2 flattened elements, got %d", len(parseResult))
	}

	parseExpected := []string{"div", "span"}
	for parseI, parseExpectedType := range parseExpected {
		if parseElem, parseOk := parseResult[parseI].(*Element); !parseOk || parseElem.Type != parseExpectedType {
			parseT.Errorf("Expected element %d to be %s, got %T", parseI, parseExpectedType, parseResult[parseI])
		}
	}
}
