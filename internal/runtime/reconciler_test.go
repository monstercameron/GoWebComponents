package runtime

import (
	"testing"
)

// Test mock DOM node
type testDOMNode struct {
	nodeType   string
	tag        string
	text       string
	attributes map[string]string
	properties map[string]interface{}
	styles     map[string]string
	children   []DOMNode
	isNull     bool
}

func (n *testDOMNode) IsNull() bool {
	return n == nil || n.isNull
}

func (n *testDOMNode) Equals(other DOMNode) bool {
	if other == nil {
		return n == nil
	}
	return n == other
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

func (a *testDOMAdapter) CreateElement(tag string) DOMNode {
	node := &testDOMNode{
		nodeType:   "element",
		tag:        tag,
		attributes: make(map[string]string),
		properties: make(map[string]interface{}),
		styles:     make(map[string]string),
		children:   make([]DOMNode, 0),
	}
	return node
}

func (a *testDOMAdapter) CreateTextNode(text string) DOMNode {
	node := &testDOMNode{
		nodeType:   "text",
		text:       text,
		attributes: make(map[string]string),
		properties: make(map[string]interface{}),
		styles:     make(map[string]string),
		children:   make([]DOMNode, 0),
	}
	return node
}

func (a *testDOMAdapter) AppendChild(parent, child DOMNode) {
	if p, ok := parent.(*testDOMNode); ok {
		p.children = append(p.children, child)
	}
}

func (a *testDOMAdapter) RemoveChild(parent, child DOMNode) {
	if p, ok := parent.(*testDOMNode); ok {
		for i, c := range p.children {
			if c.Equals(child) {
				p.children = append(p.children[:i], p.children[i+1:]...)
				break
			}
		}
	}
}

func (a *testDOMAdapter) InsertBefore(parent, newChild, refChild DOMNode) {
	if p, ok := parent.(*testDOMNode); ok {
		for i, c := range p.children {
			if c.Equals(refChild) {
				p.children = append(p.children[:i], append([]DOMNode{newChild}, p.children[i:]...)...)
				return
			}
		}
		p.children = append(p.children, newChild)
	}
}

func (a *testDOMAdapter) ReplaceChild(parent, newChild, oldChild DOMNode) {
	if p, ok := parent.(*testDOMNode); ok {
		for i, c := range p.children {
			if c.Equals(oldChild) {
				p.children[i] = newChild
				return
			}
		}
	}
}

func (a *testDOMAdapter) GetParent(node DOMNode) DOMNode {
	return nil
}

func (a *testDOMAdapter) GetParentNode(node DOMNode) DOMNode {
	return nil
}

func (a *testDOMAdapter) GetFirstChild(node DOMNode) DOMNode {
	if n, ok := node.(*testDOMNode); ok && len(n.children) > 0 {
		return n.children[0]
	}
	return nil
}

func (a *testDOMAdapter) GetNextSibling(node DOMNode) DOMNode {
	return nil
}

func (a *testDOMAdapter) GetChildren(node DOMNode) []DOMNode {
	if n, ok := node.(*testDOMNode); ok {
		return n.children
	}
	return nil
}

func (a *testDOMAdapter) SetAttribute(node DOMNode, name, value string) {
	if n, ok := node.(*testDOMNode); ok {
		n.attributes[name] = value
	}
}

func (a *testDOMAdapter) RemoveAttribute(node DOMNode, name string) {
	if n, ok := node.(*testDOMNode); ok {
		delete(n.attributes, name)
	}
}

func (a *testDOMAdapter) SetProperty(node DOMNode, name string, value interface{}) {
	if n, ok := node.(*testDOMNode); ok {
		n.properties[name] = value
	}
}

func (a *testDOMAdapter) GetProperty(node DOMNode, name string) interface{} {
	if n, ok := node.(*testDOMNode); ok {
		return n.properties[name]
	}
	return nil
}

func (a *testDOMAdapter) SetStyle(node DOMNode, property, value string) {
	if n, ok := node.(*testDOMNode); ok {
		n.styles[property] = value
	}
}

func (a *testDOMAdapter) SetStyles(node DOMNode, styles map[string]string) {
	if n, ok := node.(*testDOMNode); ok {
		for k, v := range styles {
			n.styles[k] = v
		}
	}
}

func (a *testDOMAdapter) SetInnerHTML(node DOMNode, html string) {
	if n, ok := node.(*testDOMNode); ok {
		n.children = make([]DOMNode, 0)
	}
}

func (a *testDOMAdapter) SetTextContent(node DOMNode, text string) {
	if n, ok := node.(*testDOMNode); ok {
		n.text = text
	}
}

func (a *testDOMAdapter) WrapFunction(fn interface{}) interface{} {
	return fn
}

func TestCreateElement(t *testing.T) {
	elem := CreateElement("div", map[string]interface{}{"id": "test"}, "child1", "child2")

	if elem.Type != "div" {
		t.Errorf("Expected type 'div', got %v", elem.Type)
	}

	if elem.Props["id"] != "test" {
		t.Errorf("Expected id 'test', got %v", elem.Props["id"])
	}

	if len(elem.Children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(elem.Children))
	}
}

func TestCreateElement_NilProps(t *testing.T) {
	elem := CreateElement("span", nil, "text")

	if elem.Type != "span" {
		t.Errorf("Expected type 'span', got %v", elem.Type)
	}

	if len(elem.Props) == 0 {
		t.Error("Expected props map to be initialized")
	}
}

func TestCreateElement_NoChildren(t *testing.T) {
	elem := CreateElement("input", map[string]interface{}{"type": "text"})

	if len(elem.Children) != 0 {
		t.Errorf("Expected 0 children, got %d", len(elem.Children))
	}
}

func TestIsSameType_StringTypes(t *testing.T) {
	if !isSameType("div", "div") {
		t.Error("Expected same string types to match")
	}

	if isSameType("div", "span") {
		t.Error("Expected different string types to not match")
	}

	if isSameType("div", 123) {
		t.Error("Expected string and non-string to not match")
	}
}

func TestIsSameType_FunctionTypes(t *testing.T) {
	fn1 := func(p map[string]interface{}) *Element { return nil }
	fn2 := func(p map[string]interface{}) *Element { return nil }

	if !isSameType(fn1, fn1) {
		t.Error("Expected same function to match itself")
	}

	// Different function instances should not match
	if isSameType(fn1, fn2) {
		t.Error("Expected different function instances to not match")
	}
}

func TestIsSameType_NilTypes(t *testing.T) {
	if isSameType(nil, nil) {
		t.Error("Expected nil types to not match")
	}

	if isSameType(nil, "div") {
		t.Error("Expected nil and non-nil to not match")
	}
}

func TestReconcileChildren_NewChildren(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
	}

	child1 := CreateElement("span", map[string]interface{}{"id": "1"})
	child2 := CreateElement("p", map[string]interface{}{"id": "2"})

	rt.reconcileChildren(parent, []interface{}{child1, child2})

	if parent.child == nil {
		t.Fatal("Expected parent to have a child")
	}

	if parent.child.typeOf != "span" {
		t.Errorf("Expected first child to be span, got %v", parent.child.typeOf)
	}

	if parent.child.effectTag != "PLACEMENT" {
		t.Errorf("Expected PLACEMENT tag, got %s", parent.child.effectTag)
	}

	if parent.child.sibling == nil {
		t.Fatal("Expected first child to have a sibling")
	}

	if parent.child.sibling.typeOf != "p" {
		t.Errorf("Expected second child to be p, got %v", parent.child.sibling.typeOf)
	}
}

func TestReconcileChildren_UpdateExisting(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	oldChild := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"className": "old"},
		dom:    mockDOM.CreateElement("div"),
	}

	parent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: oldChild,
		},
	}

	newChild := CreateElement("div", map[string]interface{}{"className": "new"})

	rt.reconcileChildren(parent, []interface{}{newChild})

	if parent.child == nil {
		t.Fatal("Expected parent to have a child")
	}

	if parent.child.effectTag != "UPDATE" {
		t.Errorf("Expected UPDATE tag for same type, got %s", parent.child.effectTag)
	}

	if parent.child.props["className"] != "new" {
		t.Errorf("Expected new props, got %v", parent.child.props["className"])
	}

	if parent.child.dom != oldChild.dom {
		t.Error("Expected DOM node to be reused")
	}
}

func TestReconcileChildren_DeleteOldChildren(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	oldChild1 := &Fiber{typeOf: "div", props: make(map[string]interface{})}
	oldChild2 := &Fiber{typeOf: "span", props: make(map[string]interface{})}
	oldChild1.sibling = oldChild2

	parent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: oldChild1,
		},
	}

	// Only one new child - second should be deleted
	newChild := CreateElement("div", map[string]interface{}{})

	rt.reconcileChildren(parent, []interface{}{newChild})

	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}

	if rt.deletions[0].typeOf != "span" {
		t.Errorf("Expected span to be deleted, got %v", rt.deletions[0].typeOf)
	}

	if rt.deletions[0].effectTag != "DELETION" {
		t.Errorf("Expected DELETION tag, got %s", rt.deletions[0].effectTag)
	}
}

func TestReconcileChildren_ReplaceWithDifferentType(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	oldChild := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    mockDOM.CreateElement("div"),
	}

	parent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		alternate: &Fiber{
			child: oldChild,
		},
	}

	newChild := CreateElement("span", map[string]interface{}{})

	rt.reconcileChildren(parent, []interface{}{newChild})

	// Old child should be marked for deletion
	if len(rt.deletions) != 1 {
		t.Errorf("Expected 1 deletion, got %d", len(rt.deletions))
	}

	if rt.deletions[0].typeOf != "div" {
		t.Errorf("Expected div to be deleted, got %v", rt.deletions[0].typeOf)
	}

	// New child should be placed
	if parent.child.typeOf != "span" {
		t.Errorf("Expected new child to be span, got %v", parent.child.typeOf)
	}

	if parent.child.effectTag != "PLACEMENT" {
		t.Errorf("Expected PLACEMENT tag, got %s", parent.child.effectTag)
	}
}

func TestCreateDom_Element(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	fiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"id": "test", "className": "myclass"},
	}

	dom := rt.createDom(fiber)

	if dom == nil || dom.IsNull() {
		t.Fatal("Expected DOM node to be created")
	}

	// Check that properties were applied
	mockNode := dom.(*testDOMNode)
	if mockNode.attributes["id"] != "test" {
		t.Errorf("Expected id='test', got %v", mockNode.attributes["id"])
	}
}

func TestCreateDom_TextElement(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	fiber := &Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]interface{}{"nodeValue": "Hello World"},
	}

	dom := rt.createDom(fiber)

	if dom == nil || dom.IsNull() {
		t.Fatal("Expected text node to be created")
	}

	mockNode := dom.(*testDOMNode)
	if mockNode.nodeType != "text" {
		t.Errorf("Expected text node type, got %s", mockNode.nodeType)
	}
}

func TestUpdateDomProperties_SetProperties(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	dom := mockDOM.CreateElement("div")

	oldProps := map[string]interface{}{}
	newProps := map[string]interface{}{
		"id":        "test",
		"className": "myclass",
		"disabled":  true,
	}

	rt.updateDomProperties(dom, oldProps, newProps)

	mockNode := dom.(*testDOMNode)
	if mockNode.attributes["id"] != "test" {
		t.Errorf("Expected id='test', got %v", mockNode.attributes["id"])
	}

	if mockNode.attributes["class"] != "myclass" {
		t.Errorf("Expected class='myclass', got %v", mockNode.attributes["class"])
	}
}

func TestUpdateDomProperties_RemoveOldProperties(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	dom := mockDOM.CreateElement("div")

	oldProps := map[string]interface{}{
		"id":      "old",
		"oldProp": "value",
	}
	newProps := map[string]interface{}{
		"id": "new",
	}

	rt.updateDomProperties(dom, oldProps, newProps)

	mockNode := dom.(*testDOMNode)
	if mockNode.attributes["id"] != "new" {
		t.Errorf("Expected id='new', got %v", mockNode.attributes["id"])
	}

	if _, exists := mockNode.attributes["oldProp"]; exists {
		t.Error("Expected oldProp to be removed")
	}
}

func TestUpdateDomProperties_Styles(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	dom := mockDOM.CreateElement("div")

	oldProps := map[string]interface{}{}
	newProps := map[string]interface{}{
		"style": map[string]string{
			"color":      "red",
			"background": "blue",
		},
	}

	rt.updateDomProperties(dom, oldProps, newProps)

	mockNode := dom.(*testDOMNode)
	if mockNode.styles["color"] != "red" {
		t.Errorf("Expected color='red', got %v", mockNode.styles["color"])
	}
}

func TestUpdateDomProperties_NilDom(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	// Should not panic with nil DOM
	rt.updateDomProperties(nil, map[string]interface{}{}, map[string]interface{}{"id": "test"})
}

func TestCommitRoot_ProcessesDeletions(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parentDOM := mockDOM.CreateElement("div")
	childDOM := mockDOM.CreateElement("span")
	mockDOM.AppendChild(parentDOM, childDOM)

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parentDOM,
	}

	child := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       childDOM,
		parent:    parent,
		effectTag: "DELETION",
	}

	rt.deletions = []*Fiber{child}
	rt.wipRoot = parent

	rt.commitRoot()

	if len(rt.deletions) != 0 {
		t.Errorf("Expected deletions to be cleared, got %d", len(rt.deletions))
	}

	// Check that child was removed
	mockParent := parentDOM.(*testDOMNode)
	if len(mockParent.children) != 0 {
		t.Errorf("Expected child to be removed, got %d children", len(mockParent.children))
	}
}

func TestCommitWork_Placement(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parentDOM := mockDOM.CreateElement("div")
	childDOM := mockDOM.CreateElement("span")

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parentDOM,
	}

	child := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       childDOM,
		parent:    parent,
		effectTag: "PLACEMENT",
	}

	rt.commitWork(child, parentDOM)

	mockParent := parentDOM.(*testDOMNode)
	if len(mockParent.children) != 1 {
		t.Errorf("Expected 1 child, got %d", len(mockParent.children))
	}

	if mockParent.children[0] != childDOM {
		t.Error("Expected child to be appended to parent")
	}
}

func TestCommitWork_Update(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	dom := mockDOM.CreateElement("div")

	parent := &Fiber{
		typeOf: "root",
		props:  make(map[string]interface{}),
		dom:    mockDOM.CreateElement("root"),
	}

	alternate := &Fiber{
		props: map[string]interface{}{"id": "old"},
	}

	fiber := &Fiber{
		typeOf:    "div",
		props:     map[string]interface{}{"id": "new"},
		dom:       dom,
		parent:    parent,
		alternate: alternate,
		effectTag: "UPDATE",
	}

	rt.commitWork(fiber, parent.dom)

	mockNode := dom.(*testDOMNode)
	if mockNode.attributes["id"] != "new" {
		t.Errorf("Expected id='new', got %v", mockNode.attributes["id"])
	}
}

func TestCommitWork_RecursiveCommit(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parentDOM := mockDOM.CreateElement("div")
	child1DOM := mockDOM.CreateElement("span")
	child2DOM := mockDOM.CreateElement("p")

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		dom:    parentDOM,
	}

	child1 := &Fiber{
		typeOf:    "span",
		props:     make(map[string]interface{}),
		dom:       child1DOM,
		parent:    parent,
		effectTag: "PLACEMENT",
	}

	child2 := &Fiber{
		typeOf:    "p",
		props:     make(map[string]interface{}),
		dom:       child2DOM,
		parent:    parent,
		effectTag: "PLACEMENT",
	}

	child1.sibling = child2

	rt.commitWork(child1, parentDOM)

	mockParent := parentDOM.(*testDOMNode)
	if len(mockParent.children) != 2 {
		t.Errorf("Expected 2 children, got %d", len(mockParent.children))
	}
}

func TestCommitDeletion_WithDOM(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parentDOM := mockDOM.CreateElement("div")
	childDOM := mockDOM.CreateElement("span")
	mockDOM.AppendChild(parentDOM, childDOM)

	fiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    childDOM,
	}

	rt.commitDeletion(fiber, parentDOM)

	mockParent := parentDOM.(*testDOMNode)
	if len(mockParent.children) != 0 {
		t.Errorf("Expected child to be removed, got %d children", len(mockParent.children))
	}
}

func TestCommitDeletion_FunctionComponent(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parentDOM := mockDOM.CreateElement("div")
	childDOM := mockDOM.CreateElement("span")
	mockDOM.AppendChild(parentDOM, childDOM)

	// Function component with host component child
	funcFiber := &Fiber{
		typeOf: func(p map[string]interface{}) *Element { return nil },
		props:  make(map[string]interface{}),
	}

	hostFiber := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		dom:    childDOM,
		parent: funcFiber,
	}

	funcFiber.child = hostFiber

	rt.commitDeletion(funcFiber, parentDOM)

	mockParent := parentDOM.(*testDOMNode)
	if len(mockParent.children) != 0 {
		t.Errorf("Expected child to be removed, got %d children", len(mockParent.children))
	}
}

func TestRunEffects(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	executed1 := false
	executed2 := false

	fiber1 := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		effects: []Effect{
			{Fn: func() func() { executed1 = true; return nil }, CleanupIndex: 0},
		},
		hooks: &Hooks{cleanups: make([]func(), 1)},
	}

	fiber2 := &Fiber{
		typeOf: "span",
		props:  make(map[string]interface{}),
		effects: []Effect{
			{Fn: func() func() { executed2 = true; return nil }, CleanupIndex: 0},
		},
		hooks: &Hooks{cleanups: make([]func(), 1)},
	}

	fiber1.child = fiber2

	rt.runEffects(fiber1)

	if !executed1 {
		t.Error("Expected effect 1 to be executed")
	}

	if !executed2 {
		t.Error("Expected effect 2 to be executed")
	}
}

func TestRunEffects_NilFiber(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	// Should not panic
	rt.runEffects(nil)
}

func TestRunCleanups_ExecutesAllCleanups(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: mockDOM, Scheduler: scheduler})

	executed1 := false
	executed2 := false

	child := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks: &Hooks{cleanups: []func(){
			func() { executed1 = true },
		}},
	}

	parent := &Fiber{
		typeOf: "div",
		props:  make(map[string]interface{}),
		hooks: &Hooks{cleanups: []func(){
			func() { executed2 = true },
		}},
		child: child,
	}

	rt.runCleanups(parent)

	if !executed1 {
		t.Error("Expected child cleanup to run")
	}
	if !executed2 {
		t.Error("Expected parent cleanup to run")
	}
}

func TestCommitDeletion_RunsCleanupsAndCleansAtomSubs(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{DOMAdapter: mockDOM, Scheduler: scheduler})

	// Create a DOM tree
	parentDOM := mockDOM.CreateElement("div")
	childDOM := mockDOM.CreateElement("span")
	mockDOM.AppendChild(parentDOM, childDOM)

	// Create a function fiber that subscribes to an atom
	funcFiber := &Fiber{typeOf: func(p map[string]interface{}) *Element { return nil }, props: make(map[string]interface{})}
	hostFiber := &Fiber{typeOf: "span", props: make(map[string]interface{}), dom: childDOM, parent: funcFiber}
	funcFiber.child = hostFiber

	// Simulate using atom in child fiber via the runtime's registry
	rt.atomRegistry.InitAtom("test-atom", 0)
	rt.atomRegistry.Subscribe("test-atom", funcFiber)

	// Ensure subscription is present
	if rt.atomRegistry.GetSubscriberCount("test-atom") != 1 {
		t.Fatal("expected 1 subscriber before deletion")
	}

	// Attach a cleanup to the function fiber
	ran := false
	funcFiber.hooks = &Hooks{cleanups: []func(){func() { ran = true }}}

	rt.commitDeletion(funcFiber, parentDOM)

	if !ran {
		t.Error("Expected cleanup to run during commitDeletion")
	}

	if rt.atomRegistry.GetSubscriberCount("test-atom") != 0 {
		t.Error("Expected atom subscription to be removed during commitDeletion")
	}
}

func TestGetNextUnitOfWork_Child(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	child := &Fiber{typeOf: "child"}
	parent := &Fiber{typeOf: "parent", child: child}

	next := rt.getNextUnitOfWork(parent)

	if next != child {
		t.Error("Expected child to be next unit of work")
	}
}

func TestGetNextUnitOfWork_Sibling(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	sibling := &Fiber{typeOf: "sibling"}
	parent := &Fiber{typeOf: "parent"}
	fiber := &Fiber{typeOf: "fiber", parent: parent, sibling: sibling}

	next := rt.getNextUnitOfWork(fiber)

	if next != sibling {
		t.Error("Expected sibling to be next unit of work")
	}
}

func TestGetNextUnitOfWork_ParentSibling(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parentSibling := &Fiber{typeOf: "parentSibling"}
	parent := &Fiber{typeOf: "parent", sibling: parentSibling}
	fiber := &Fiber{typeOf: "fiber", parent: parent}

	next := rt.getNextUnitOfWork(fiber)

	if next != parentSibling {
		t.Error("Expected parent's sibling to be next unit of work")
	}
}

func TestGetNextUnitOfWork_End(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	parent := &Fiber{typeOf: "parent"}
	fiber := &Fiber{typeOf: "fiber", parent: parent}

	next := rt.getNextUnitOfWork(fiber)

	if next != nil {
		t.Errorf("Expected nil at end of tree, got %v", next)
	}
}

func TestPerformUnitOfWork_SkipNonDirty(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	sibling := &Fiber{typeOf: "sibling", dirty: true}
	fiber := &Fiber{typeOf: "test", dirty: false, sibling: sibling}

	next := rt.performUnitOfWork(fiber)

	// Should skip to sibling
	if next != sibling {
		t.Errorf("Expected to skip to sibling, got %v", next)
	}
}

func TestPerformUnitOfWork_RootFiber(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	child1 := CreateElement("div", map[string]interface{}{})
	child2 := CreateElement("span", map[string]interface{}{})

	fiber := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{"children": []interface{}{child1, child2}},
		dirty:  true,
	}

	rt.performUnitOfWork(fiber)

	if fiber.child == nil {
		t.Fatal("Expected root to have children")
	}

	if fiber.child.typeOf != "div" {
		t.Errorf("Expected first child to be div, got %v", fiber.child.typeOf)
	}
}

func TestPerformUnitOfWork_HostComponent(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	childElem := CreateElement("span", map[string]interface{}{})

	fiber := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"children": []interface{}{childElem}},
		dirty:  true,
	}

	rt.performUnitOfWork(fiber)

	if fiber.dom == nil || fiber.dom.IsNull() {
		t.Fatal("Expected DOM node to be created")
	}

	if fiber.child == nil {
		t.Fatal("Expected fiber to have children")
	}

	if fiber.child.typeOf != "span" {
		t.Errorf("Expected child to be span, got %v", fiber.child.typeOf)
	}
}

func TestPerformUnitOfWork_FunctionComponent(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	componentFn := func(props map[string]interface{}) *Element {
		return CreateElement("div", map[string]interface{}{"id": "from-component"})
	}

	fiber := &Fiber{
		typeOf: componentFn,
		props:  make(map[string]interface{}),
		dirty:  true,
	}

	rt.performUnitOfWork(fiber)

	if fiber.hooks == nil {
		t.Fatal("Expected hooks to be initialized")
	}

	if fiber.child == nil {
		t.Fatal("Expected fiber to have children")
	}

	if fiber.child.typeOf != "div" {
		t.Errorf("Expected child to be div, got %v", fiber.child.typeOf)
	}
}

func TestPerformUnitOfWork_FunctionComponentWithHooks(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	// Component with previous render (has hooks state)
	oldHooks := &Hooks{
		states: []interface{}{42, 42, "test", "test"},
		deps:   make([][]interface{}, 0),
		memos:  make([]memoizedValue, 0),
		index:  0,
	}

	alternate := &Fiber{
		hooks: oldHooks,
	}

	componentFn := func(props map[string]interface{}) *Element {
		return CreateElement("div", nil)
	}

	fiber := &Fiber{
		typeOf:    componentFn,
		props:     make(map[string]interface{}),
		dirty:     true,
		alternate: alternate,
	}

	rt.performUnitOfWork(fiber)

	if fiber.hooks == nil {
		t.Fatal("Expected hooks to be initialized")
	}

	if len(fiber.hooks.states) != 4 {
		t.Errorf("Expected hooks state to be preserved, got %d items", len(fiber.hooks.states))
	}

	if fiber.hooks.states[0] != 42 {
		t.Errorf("Expected first state to be 42, got %v", fiber.hooks.states[0])
	}
}

func TestPerformUnitOfWork_NilFiber(t *testing.T) {
	mockDOM := newTestDOMAdapter()
	scheduler := newTestScheduler()
	rt := NewRuntime(Config{
		DOMAdapter: mockDOM,
		Scheduler:  scheduler,
	})

	next := rt.performUnitOfWork(nil)

	if next != nil {
		t.Errorf("Expected nil for nil fiber, got %v", next)
	}
}

func TestGetSetCurrentFiber(t *testing.T) {
	fiber := &Fiber{typeOf: "test"}

	SetCurrentFiber(fiber)

	if GetCurrentFiber() != fiber {
		t.Error("Expected to get the same fiber that was set")
	}

	SetCurrentFiber(nil)

	if GetCurrentFiber() != nil {
		t.Error("Expected nil after setting to nil")
	}
}

func TestFlattenFragments_NoFragments(t *testing.T) {
	elements := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	result, _ := flattenFragments(elements)

	if len(result) != 2 {
		t.Errorf("Expected 2 elements, got %d", len(result))
	}

	if elem, ok := result[0].(*Element); !ok || elem.Type != "div" {
		t.Error("Expected first element to be div")
	}

	if elem, ok := result[1].(*Element); !ok || elem.Type != "span" {
		t.Error("Expected second element to be span")
	}
}

func TestFlattenFragments_WithFragment(t *testing.T) {
	// Create elements: div, Fragment(span, p), h1
	fragmentChildren := []interface{}{
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{Type: "p", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	elements := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": fragmentChildren},
			Children: []interface{}{},
		},
		&Element{Type: "h1", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	result, _ := flattenFragments(elements)

	if len(result) != 4 {
		t.Errorf("Expected 4 flattened elements, got %d", len(result))
	}

	// Check order: div, span, p, h1
	expected := []string{"div", "span", "p", "h1"}
	for i, expectedType := range expected {
		if elem, ok := result[i].(*Element); !ok || elem.Type != expectedType {
			t.Errorf("Expected element %d to be %s, got %T with type %v", i, expectedType, result[i], elem.Type)
		}
	}
}

func TestFlattenFragments_NestedFragments(t *testing.T) {
	// Create nested: Fragment(div, Fragment(span, p), h1)
	innerFragmentChildren := []interface{}{
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{Type: "p", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	outerFragmentChildren := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": innerFragmentChildren},
			Children: []interface{}{},
		},
		&Element{Type: "h1", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	elements := []interface{}{
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": outerFragmentChildren},
			Children: []interface{}{},
		},
	}

	result, _ := flattenFragments(elements)

	if len(result) != 4 {
		t.Errorf("Expected 4 flattened elements from nested fragments, got %d", len(result))
	}

	// Check order: div, span, p, h1
	expected := []string{"div", "span", "p", "h1"}
	for i, expectedType := range expected {
		if elem, ok := result[i].(*Element); !ok || elem.Type != expectedType {
			t.Errorf("Expected element %d to be %s, got %T with type %v", i, expectedType, result[i], elem.Type)
		}
	}
}

func TestFlattenFragments_EmptyFragment(t *testing.T) {
	elements := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
		&Element{
			Type:     "FRAGMENT",
			Props:    make(map[string]interface{}),
			Children: []interface{}{},
		},
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	result, _ := flattenFragments(elements)

	if len(result) != 2 {
		t.Errorf("Expected 2 elements (empty fragment flattened away), got %d", len(result))
	}

	// Check order: div, span
	expected := []string{"div", "span"}
	for i, expectedType := range expected {
		if elem, ok := result[i].(*Element); !ok || elem.Type != expectedType {
			t.Errorf("Expected element %d to be %s, got %T with type %v", i, expectedType, result[i], elem.Type)
		}
	}
}

func TestFlattenFragments_OnlyFragments(t *testing.T) {
	// Only fragments containing elements
	fragment1Children := []interface{}{
		&Element{Type: "div", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	fragment2Children := []interface{}{
		&Element{Type: "span", Props: make(map[string]interface{}), Children: []interface{}{}},
	}

	elements := []interface{}{
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": fragment1Children},
			Children: []interface{}{},
		},
		&Element{
			Type:     "FRAGMENT",
			Props:    map[string]interface{}{"children": fragment2Children},
			Children: []interface{}{},
		},
	}

	result, _ := flattenFragments(elements)

	if len(result) != 2 {
		t.Errorf("Expected 2 flattened elements, got %d", len(result))
	}

	expected := []string{"div", "span"}
	for i, expectedType := range expected {
		if elem, ok := result[i].(*Element); !ok || elem.Type != expectedType {
			t.Errorf("Expected element %d to be %s, got %T", i, expectedType, result[i])
		}
	}
}
