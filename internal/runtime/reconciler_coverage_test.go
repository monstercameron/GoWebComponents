package runtime

import "testing"

func TestFlattenFragments_NestedFragmentsWithRawChildren(t *testing.T) {
	elements := []interface{}{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]interface{}{
				"children": []interface{}{
					&Element{Type: "div"},
					&Element{
						Type: "FRAGMENT",
						Props: map[string]interface{}{
							"children": []interface{}{nil, &Element{Type: "span"}, "raw"},
						},
					},
				},
			},
		},
	}

	flattened, allocated := flattenFragments(elements)

	if !allocated {
		t.Fatal("expected fragment flattening to allocate when wrappers are present")
	}
	if len(flattened) < 3 {
		t.Fatalf("expected at least 3 flattened children, got %d", len(flattened))
	}

	hasDiv := false
	hasSpan := false
	hasRaw := false
	for _, item := range flattened {
		switch v := item.(type) {
		case *Element:
			if v == nil {
				continue
			}
			if v.Type == "div" {
				hasDiv = true
			}
			if v.Type == "span" {
				hasSpan = true
			}
		case string:
			if v == "raw" {
				hasRaw = true
			}
		}
	}

	if !hasDiv || !hasSpan || !hasRaw {
		t.Fatal("expected flattened output to preserve non-fragment children")
	}
}

func TestFlattenFragments_SkipsNilAndTypedNilElements(t *testing.T) {
	var typedNil *Element
	elements := []interface{}{
		&Element{
			Type: "FRAGMENT",

			Props: map[string]interface{}{
				"children": []interface{}{nil, typedNil, "raw"},
			},
		},
	}

	flattened, allocated := flattenFragments(elements)

	if !allocated {
		t.Fatal("expected allocation when fragment wrapper is present")
	}

	foundRaw := false
	for _, item := range flattened {
		if str, ok := item.(string); ok && str == "raw" {
			foundRaw = true
		}
	}
	if !foundRaw {
		t.Fatal("expected non-nil child to survive flattening")
	}
}

func TestCreateDom_TextElementFromNodeValue(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})

	dom := rt.createDom(&Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]interface{}{"nodeValue": "fallback"},
	})

	node := dom.(*testDOMNode)
	if node.nodeType != "text" || node.text != "fallback" {
		t.Fatal("expected text node to be created from nodeValue fallback")
	}
}

func TestCreateDom_TextElementFromTextContent(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})

	dom := rt.createDom(&Fiber{
		typeOf:      "TEXT_ELEMENT",
		textContent: "from-text-content",
		props:       map[string]interface{}{},
	})

	node := dom.(*testDOMNode)
	if node.text != "from-text-content" {
		t.Fatal("expected text node to be created from textContent")
	}
}

func TestCreateDom_TextElementWithEmptyTextContentCreatesNode(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})

	dom := rt.createDom(&Fiber{
		typeOf:      "TEXT_ELEMENT",
		textContent: "",
		props:       map[string]interface{}{},
	})

	if dom == nil {
		t.Fatal("expected empty text node to be created")
	}

	node := dom.(*testDOMNode)
	if node.nodeType != "text" || node.text != "" {
		t.Fatalf("expected empty text node, got type=%q text=%q", node.nodeType, node.text)
	}
}

func TestCreateDom_FragmentReturnsNil(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	if dom := rt.createDom(&Fiber{typeOf: "FRAGMENT"}); dom != nil {
		t.Fatal("expected fragment fibers to produce no DOM node")
	}
}

func TestCreateDom_ElementAppliesProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})

	dom := rt.createDom(&Fiber{
		typeOf: "button",
		props:  map[string]interface{}{"id": "save", "className": "primary"},
	})

	node := dom.(*testDOMNode)
	if node.tag != "button" {
		t.Fatal("expected button element")
	}
	if node.attributes["id"] != "save" || node.attributes["class"] != "primary" {
		t.Fatal("expected element props to be applied during DOM creation")
	}
}

func TestCreateDom_UnsupportedTypeReturnsNil(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	if dom := rt.createDom(&Fiber{typeOf: 42}); dom != nil {
		t.Fatal("expected unsupported fiber type to produce nil DOM")
	}
}

func TestUpdateDomProperties_NilDomIsIgnored(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	rt.updateDomProperties(nil, map[string]interface{}{"id": "old"}, map[string]interface{}{"id": "new"})
}

func TestUpdateDomProperties_InitialRenderAppliesMixedValues(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("input")

	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"id":        "field",
		"className": "large",
		"value":     "abc",
		"style":     "color:red",
	})

	node := dom.(*testDOMNode)
	if node.attributes["id"] != "field" || node.attributes["class"] != "large" || node.attributes["style"] != "color:red" {
		t.Fatal("expected string attributes to be applied on initial render")
	}
	if node.properties["value"] != "abc" {
		t.Fatal("expected property updates to be applied on initial render")
	}
}

func TestUpdateDomProperties_InitialRenderStringStyleWithoutBatching(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("div")

	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"style": "color:blue",
	})

	if dom.(*testDOMNode).attributes["style"] != "color:blue" {
		t.Fatal("expected string style to be applied as attribute")
	}
}

func TestUpdateDomProperties_InitialRenderWithPropertyFallback(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("div")

	rt.updateDomProperties(dom, map[string]interface{}{}, map[string]interface{}{
		"tabIndex": 2,
		"hidden":   true,
	})

	node := dom.(*testDOMNode)
	if node.properties["tabIndex"] != 2 || node.properties["hidden"] != true {
		t.Fatal("expected non-string props to be set as properties")
	}
}

func TestUpdateDomProperties_RemoveOldProps(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("input")
	node := dom.(*testDOMNode)
	node.attributes["id"] = "old"
	node.attributes["class"] = "legacy"

	rt.updateDomProperties(dom, map[string]interface{}{"id": "old", "class": "legacy"}, map[string]interface{}{"class": "next"})

	if _, ok := node.attributes["id"]; ok {
		t.Fatal("expected removed attribute to be deleted")
	}
	if node.attributes["class"] != "next" {
		t.Fatal("expected retained attribute to be updated")
	}
}

func TestUpdateDomProperties_SkipUnchangedValue(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("div")

	handler := func() {}
	rt.updateDomProperties(dom, map[string]interface{}{"onclick": handler}, map[string]interface{}{"onclick": handler})

	if len(dom.(*testDOMNode).properties) != 0 {
		t.Fatal("expected unchanged properties to be skipped")
	}
}

func TestUpdateDomProperties_StringStyleAndSelectedProperty(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("option")

	rt.updateDomProperties(dom, map[string]interface{}{"selected": false}, map[string]interface{}{
		"style":    "font-weight:bold",
		"selected": true,
	})

	node := dom.(*testDOMNode)
	if node.attributes["style"] != "font-weight:bold" {
		t.Fatal("expected inline style string to be set as attribute")
	}
	if node.properties["selected"] != true {
		t.Fatal("expected selected to be set as property")
	}
}

func TestUpdateDomProperties_UpdateStyleMap(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("div")

	rt.updateDomProperties(dom, map[string]interface{}{"id": "x"}, map[string]interface{}{
		"id": "x",
		"style": map[string]string{
			"color": "green",
		},
	})

	if dom.(*testDOMNode).styles["color"] != "green" {
		t.Fatal("expected style map update branch to execute")
	}
}

func TestUpdateDomProperties_UpdateNonStringProperty(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	dom := rt.domAdapter.CreateElement("input")

	rt.updateDomProperties(dom, map[string]interface{}{"tabIndex": 1}, map[string]interface{}{"tabIndex": 5})

	if dom.(*testDOMNode).properties["tabIndex"] != 5 {
		t.Fatal("expected changed non-string property to be updated")
	}
}

func TestCommitWork_ResolvesParentFromAncestors(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	parentDOM := adapter.CreateElement("div")
	root := &Fiber{typeOf: "ROOT", dom: parentDOM}
	functionParent := &Fiber{typeOf: func(map[string]interface{}) *Element { return nil }, parent: root}
	child := &Fiber{typeOf: "span", dom: adapter.CreateElement("span"), parent: functionParent, effectTag: "PLACEMENT"}

	rt.commitWork(child, nil)

	if len(parentDOM.(*testDOMNode).children) != 1 {
		t.Fatal("expected placement to resolve DOM parent from ancestor chain")
	}
}

func TestCommitWork_TextUpdateUsesFallbackProps(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	textDOM := adapter.CreateTextNode("old")
	fiber := &Fiber{
		typeOf: "TEXT_ELEMENT",
		dom:    textDOM,
		props:  map[string]interface{}{"nodeValue": "new"},
		alternate: &Fiber{
			typeOf: "TEXT_ELEMENT",
			dom:    textDOM,
			props:  map[string]interface{}{"nodeValue": "old"},
		},
		effectTag: "UPDATE",
	}

	rt.commitWork(fiber, adapter.CreateElement("div"))

	if textDOM.(*testDOMNode).text != "new" {
		t.Fatal("expected text update to use nodeValue fallback path")
	}
}

func TestCommitWork_TraversesChildrenAndSiblings(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	parentDOM := adapter.CreateElement("div")
	parent := &Fiber{typeOf: "section", dom: parentDOM}
	child := &Fiber{typeOf: "span", dom: adapter.CreateElement("span"), parent: parent, effectTag: "PLACEMENT"}
	sibling := &Fiber{typeOf: "p", dom: adapter.CreateElement("p"), parent: parent, effectTag: "PLACEMENT"}
	parent.child = child
	child.sibling = sibling

	rt.commitWork(parent, parentDOM)

	if len(parentDOM.(*testDOMNode).children) != 2 {
		t.Fatal("expected commitWork to recurse into children and siblings")
	}
}

func TestCommitWork_UpdateWithoutAlternateDoesNotPanic(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	fiber := &Fiber{typeOf: "div", dom: rt.domAdapter.CreateElement("div"), effectTag: "UPDATE"}

	rt.commitWork(fiber, rt.domAdapter.CreateElement("div"))
}

func TestCommitDeletion_NilFiber(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	rt.commitDeletion(nil, rt.domAdapter.CreateElement("div"))
}

func TestCommitDeletion_FunctionComponentRemovesAllChildDOMSiblings(t *testing.T) {
	adapter := newTestDOMAdapter()
	rt := NewRuntime(Config{DOMAdapter: adapter, Scheduler: newTestScheduler()})
	parentDOM := adapter.CreateElement("div")
	firstDOM := adapter.CreateElement("span")
	secondDOM := adapter.CreateElement("p")
	adapter.AppendChild(parentDOM, firstDOM)
	adapter.AppendChild(parentDOM, secondDOM)

	funcFiber := &Fiber{typeOf: func(map[string]interface{}) *Element { return nil }}
	firstChild := &Fiber{typeOf: "span", dom: firstDOM, parent: funcFiber}
	secondChild := &Fiber{typeOf: "p", dom: secondDOM, parent: funcFiber}
	firstChild.sibling = secondChild
	funcFiber.child = firstChild

	rt.commitDeletion(funcFiber, parentDOM)

	if len(parentDOM.(*testDOMNode).children) != 0 {
		t.Fatal("expected all DOM descendants to be removed for function-component deletion")
	}
}

func TestRunCleanups_WithNilEntries(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	executed := false
	fiber := &Fiber{hooks: &Hooks{cleanups: []func(){nil, func() { executed = true }}}}

	rt.runCleanups(fiber)

	if !executed {
		t.Fatal("expected non-nil cleanup to run")
	}
}

func TestRunEffects_NoEffectsOnFiberStillTraversesChildren(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	executed := false
	child := &Fiber{
		hooks:   &Hooks{cleanups: make([]func(), 1)},
		effects: []Effect{{Fn: func() func() { executed = true; return nil }, CleanupIndex: 0}},
	}
	parent := &Fiber{child: child}

	rt.runEffects(parent)

	if !executed {
		t.Fatal("expected child effect to run even when parent has no effects")
	}
}

func TestPerformUnitOfWork_HostFiberWithoutChildrenProp(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	fiber := &Fiber{typeOf: "div", props: map[string]interface{}{}, dirty: true}

	next := rt.performUnitOfWork(fiber)

	if fiber.dom == nil {
		t.Fatal("expected host fiber DOM to be created")
	}
	if next != nil {
		t.Fatal("expected no further work without children or siblings")
	}
}

func TestPerformUnitOfWork_FunctionComponentReturningNil(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	fiber := &Fiber{
		typeOf: func(map[string]interface{}) *Element { return nil },
		props:  map[string]interface{}{},
		dirty:  true,
	}

	next := rt.performUnitOfWork(fiber)

	if fiber.child != nil {
		t.Fatal("expected no child fibers when component returns nil")
	}
	if next != nil {
		t.Fatal("expected no further work")
	}
}

func TestPerformUnitOfWork_FunctionComponentReturningNilDeletesPreviousChild(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	oldChild := &Fiber{
		typeOf: "div",
		props:  map[string]interface{}{"id": "old"},
		dom:    rt.domAdapter.CreateElement("div"),
	}
	fiber := &Fiber{
		typeOf: func(map[string]interface{}) *Element { return nil },
		props:  map[string]interface{}{},
		dirty:  true,
		alternate: &Fiber{
			child: oldChild,
		},
	}

	rt.performUnitOfWork(fiber)

	if len(rt.deletions) != 1 || rt.deletions[0] != oldChild {
		t.Fatal("expected previous child to be marked for deletion when component renders nil")
	}
}

func TestReconcileChildren_NilElementDeletesOldFiber(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	old := &Fiber{typeOf: "div", props: map[string]interface{}{}, dom: rt.domAdapter.CreateElement("div")}
	parent := &Fiber{alternate: &Fiber{child: old}}

	rt.reconcileChildren(parent, []interface{}{nil})

	if len(rt.deletions) != 1 || rt.deletions[0] != old {
		t.Fatal("expected nil element slot to mark old fiber for deletion")
	}
}

func TestRunCleanups_NilFiber(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	rt.runCleanups(nil)
}

func TestRunCleanups_TraversesSiblingChain(t *testing.T) {
	rt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	count := 0
	sibling := &Fiber{hooks: &Hooks{cleanups: []func(){func() { count++ }}}}
	child := &Fiber{hooks: &Hooks{cleanups: []func(){func() { count++ }}}, sibling: sibling}

	rt.runCleanups(child)

	if count != 2 {
		t.Fatal("expected cleanup traversal to include siblings")
	}
}
