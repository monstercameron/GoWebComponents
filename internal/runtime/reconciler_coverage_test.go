package runtime

import "testing"

func TestFlattenFragments_NestedFragmentsWithRawChildren(parseT *testing.T) {
	parseElements := []any{
		&Element{
			Type: "FRAGMENT",
			Props: map[string]any{
				"children": []any{
					&Element{Type: "div"},
					&Element{
						Type: "FRAGMENT",
						Props: map[string]any{
							"children": []any{nil, &Element{Type: "span"}, "raw"},
						},
					},
				},
			},
		},
	}

	parseFlattened, parseAllocated := flattenFragments(parseElements)

	if !parseAllocated {
		parseT.Fatal("expected fragment flattening to allocate when wrappers are present")
	}
	if len(parseFlattened) < 3 {
		parseT.Fatalf("expected at least 3 flattened children, got %d", len(parseFlattened))
	}

	hasDiv := false
	hasSpan := false
	hasRaw := false
	for _, parseItem := range parseFlattened {
		switch parseV := parseItem.(type) {
		case *Element:
			if parseV == nil {
				continue
			}
			if parseV.Type == "div" {
				hasDiv = true
			}
			if parseV.Type == "span" {
				hasSpan = true
			}
		case string:
			if parseV == "raw" {
				hasRaw = true
			}
		}
	}

	if !hasDiv || !hasSpan || !hasRaw {
		parseT.Fatal("expected flattened output to preserve non-fragment children")
	}
}

func TestFlattenFragments_SkipsNilAndTypedNilElements(parseT *testing.T) {
	var parseTypedNil *Element
	parseElements := []any{
		&Element{
			Type: "FRAGMENT",

			Props: map[string]any{
				"children": []any{nil, parseTypedNil, "raw"},
			},
		},
	}

	parseFlattened, parseAllocated := flattenFragments(parseElements)

	if !parseAllocated {
		parseT.Fatal("expected allocation when fragment wrapper is present")
	}

	isParseFoundRaw := false
	for _, parseItem := range parseFlattened {
		if parseStr, parseOk := parseItem.(string); parseOk && parseStr == "raw" {
			isParseFoundRaw = true
		}
	}
	if !isParseFoundRaw {
		parseT.Fatal("expected non-nil child to survive flattening")
	}
}

func TestCreateDom_TextElementFromNodeValue(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseDom := parseRt.createDom(&Fiber{
		typeOf: "TEXT_ELEMENT",
		props:  map[string]any{"nodeValue": "fallback"},
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.nodeType != "text" || parseNode.text != "fallback" {
		parseT.Fatal("expected text node to be created from nodeValue fallback")
	}
}

func TestCreateDom_TextElementFromTextContent(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseDom := parseRt.createDom(&Fiber{
		typeOf:      "TEXT_ELEMENT",
		textContent: "from-text-content",
		props:       map[string]any{},
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.text != "from-text-content" {
		parseT.Fatal("expected text node to be created from textContent")
	}
}

func TestCreateDom_TextElementWithEmptyTextContentCreatesNode(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseDom := parseRt.createDom(&Fiber{
		typeOf:      "TEXT_ELEMENT",
		textContent: "",
		props:       map[string]any{},
	})

	if parseDom == nil {
		parseT.Fatal("expected empty text node to be created")
	}

	parseNode := parseDom.(*testDOMNode)
	if parseNode.nodeType != "text" || parseNode.text != "" {
		parseT.Fatalf("expected empty text node, got type=%q text=%q", parseNode.nodeType, parseNode.text)
	}
}

func TestCreateDom_FragmentReturnsNil(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	if parseDom := parseRt.createDom(&Fiber{typeOf: "FRAGMENT"}); parseDom != nil {
		parseT.Fatal("expected fragment fibers to produce no DOM node")
	}
}

func TestCreateDom_ElementAppliesProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})

	parseDom := parseRt.createDom(&Fiber{
		typeOf: "button",
		props:  map[string]any{"id": "save", "className": "primary"},
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.tag != "button" {
		parseT.Fatal("expected button element")
	}
	if parseNode.attributes["id"] != "save" || parseNode.attributes["class"] != "primary" {
		parseT.Fatal("expected element props to be applied during DOM creation")
	}
}

func TestCreateDom_UnsupportedTypeReturnsNil(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})

	if parseDom := parseRt.createDom(&Fiber{typeOf: 42}); parseDom != nil {
		parseT.Fatal("expected unsupported fiber type to produce nil DOM")
	}
}

func TestUpdateDomProperties_NilDomIsIgnored(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRt.updateDomProperties(nil, map[string]any{"id": "old"}, map[string]any{"id": "new"})
}

func TestUpdateDomProperties_InitialRenderAppliesMixedValues(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("input")

	parseRt.updateDomProperties(parseDom, map[string]any{}, map[string]any{
		"id":        "field",
		"className": "large",
		"value":     "abc",
		"style":     "color:red",
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["id"] != "field" || parseNode.attributes["class"] != "large" || parseNode.attributes["style"] != "color:red" {
		parseT.Fatal("expected string attributes to be applied on initial render")
	}
	if parseNode.properties["value"] != "abc" {
		parseT.Fatal("expected property updates to be applied on initial render")
	}
}

func TestUpdateDomProperties_InitialRenderStringStyleWithoutBatching(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("div")

	parseRt.updateDomProperties(parseDom, map[string]any{}, map[string]any{
		"style": "color:blue",
	})

	if parseDom.(*testDOMNode).attributes["style"] != "color:blue" {
		parseT.Fatal("expected string style to be applied as attribute")
	}
}

func TestUpdateDomProperties_InitialRenderWithPropertyFallback(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("div")

	parseRt.updateDomProperties(parseDom, map[string]any{}, map[string]any{
		"tabIndex": 2,
		"hidden":   true,
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.properties["tabIndex"] != 2 || parseNode.properties["hidden"] != true {
		parseT.Fatal("expected non-string props to be set as properties")
	}
}

func TestUpdateDomProperties_RemoveOldProps(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("input")
	parseNode := parseDom.(*testDOMNode)
	parseNode.attributes["id"] = "old"
	parseNode.attributes["class"] = "legacy"

	parseRt.updateDomProperties(parseDom, map[string]any{"id": "old", "class": "legacy"}, map[string]any{"class": "next"})

	if _, parseOk := parseNode.attributes["id"]; parseOk {
		parseT.Fatal("expected removed attribute to be deleted")
	}
	if parseNode.attributes["class"] != "next" {
		parseT.Fatal("expected retained attribute to be updated")
	}
}

func TestUpdateDomProperties_SkipUnchangedValue(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("div")

	parseHandler := func() {}
	parseRt.updateDomProperties(parseDom, map[string]any{"onclick": parseHandler}, map[string]any{"onclick": parseHandler})

	if len(parseDom.(*testDOMNode).properties) != 0 {
		parseT.Fatal("expected unchanged properties to be skipped")
	}
}

func TestUpdateDomProperties_StringStyleAndSelectedProperty(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("option")

	parseRt.updateDomProperties(parseDom, map[string]any{"selected": false}, map[string]any{
		"style":    "font-weight:bold",
		"selected": true,
	})

	parseNode := parseDom.(*testDOMNode)
	if parseNode.attributes["style"] != "font-weight:bold" {
		parseT.Fatal("expected inline style string to be set as attribute")
	}
	if parseNode.properties["selected"] != true {
		parseT.Fatal("expected selected to be set as property")
	}
}

func TestUpdateDomProperties_UpdateStyleMap(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("div")

	parseRt.updateDomProperties(parseDom, map[string]any{"id": "x"}, map[string]any{
		"id": "x",
		"style": map[string]string{
			"color": "green",
		},
	})

	if parseDom.(*testDOMNode).styles["color"] != "green" {
		parseT.Fatal("expected style map update branch to execute")
	}
}

func TestUpdateDomProperties_UpdateNonStringProperty(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseDom := parseRt.domAdapter.CreateElement("input")

	parseRt.updateDomProperties(parseDom, map[string]any{"tabIndex": 1}, map[string]any{"tabIndex": 5})

	if parseDom.(*testDOMNode).properties["tabIndex"] != 5 {
		parseT.Fatal("expected changed non-string property to be updated")
	}
}

func TestCommitWork_ResolvesParentFromAncestors(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseParentDOM := parseAdapter.CreateElement("div")
	parseRoot := &Fiber{typeOf: "ROOT", dom: parseParentDOM}
	parseFunctionParent := &Fiber{typeOf: func(map[string]any) *Element { return nil }, parent: parseRoot}
	parseChild := &Fiber{typeOf: "span", dom: parseAdapter.CreateElement("span"), parent: parseFunctionParent, effectTag: effectTagPlacement}

	parseRt.commitWork(parseChild, nil)

	if len(parseParentDOM.(*testDOMNode).children) != 1 {
		parseT.Fatal("expected placement to resolve DOM parent from ancestor chain")
	}
}

func TestCommitWork_TextUpdateUsesFallbackProps(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseTextDOM := parseAdapter.CreateTextNode("old")
	parseFiber := &Fiber{
		typeOf: "TEXT_ELEMENT",
		dom:    parseTextDOM,
		props:  map[string]any{"nodeValue": "new"},
		alternate: &Fiber{
			typeOf: "TEXT_ELEMENT",
			dom:    parseTextDOM,
			props:  map[string]any{"nodeValue": "old"},
		},
		effectTag: effectTagUpdate,
	}

	parseRt.commitWork(parseFiber, parseAdapter.CreateElement("div"))

	if parseTextDOM.(*testDOMNode).text != "new" {
		parseT.Fatal("expected text update to use nodeValue fallback path")
	}
}

func TestCommitWork_TraversesChildrenAndSiblings(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseParentDOM := parseAdapter.CreateElement("div")
	parseParent := &Fiber{typeOf: "section", dom: parseParentDOM}
	parseChild := &Fiber{typeOf: "span", dom: parseAdapter.CreateElement("span"), parent: parseParent, effectTag: effectTagPlacement}
	parseSibling := &Fiber{typeOf: "p", dom: parseAdapter.CreateElement("p"), parent: parseParent, effectTag: effectTagPlacement}
	parseParent.child = parseChild
	parseChild.sibling = parseSibling

	parseRt.commitWork(parseParent, parseParentDOM)

	if len(parseParentDOM.(*testDOMNode).children) != 2 {
		parseT.Fatal("expected commitWork to recurse into children and siblings")
	}
}

func TestCommitWork_UpdateWithoutAlternateDoesNotPanic(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseFiber := &Fiber{typeOf: "div", dom: parseRt.domAdapter.CreateElement("div"), effectTag: effectTagUpdate}

	parseRt.commitWork(parseFiber, parseRt.domAdapter.CreateElement("div"))
}

func TestCommitDeletion_NilFiber(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRt.commitDeletion(nil, parseRt.domAdapter.CreateElement("div"))
}

func TestCommitDeletion_FunctionComponentRemovesAllChildDOMSiblings(parseT *testing.T) {
	parseAdapter := newTestDOMAdapter()
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter, Scheduler: newTestScheduler()})
	parseParentDOM := parseAdapter.CreateElement("div")
	parseFirstDOM := parseAdapter.CreateElement("span")
	parseSecondDOM := parseAdapter.CreateElement("p")
	parseAdapter.AppendChild(parseParentDOM, parseFirstDOM)
	parseAdapter.AppendChild(parseParentDOM, parseSecondDOM)

	parseFuncFiber := &Fiber{typeOf: func(map[string]any) *Element { return nil }}
	parseFirstChild := &Fiber{typeOf: "span", dom: parseFirstDOM, parent: parseFuncFiber}
	parseSecondChild := &Fiber{typeOf: "p", dom: parseSecondDOM, parent: parseFuncFiber}
	parseFirstChild.sibling = parseSecondChild
	parseFuncFiber.child = parseFirstChild

	parseRt.commitDeletion(parseFuncFiber, parseParentDOM)

	if len(parseParentDOM.(*testDOMNode).children) != 0 {
		parseT.Fatal("expected all DOM descendants to be removed for function-component deletion")
	}
}

func TestRunCleanups_WithNilEntries(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	isParseExecuted := false
	parseFiber := &Fiber{hooks: &Hooks{cleanups: []func(){nil, func() { isParseExecuted = true }}}}

	parseRt.runCleanups(parseFiber)

	if !isParseExecuted {
		parseT.Fatal("expected non-nil cleanup to run")
	}
}

func TestRunEffects_NoEffectsOnFiberStillTraversesChildren(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	isParseExecuted := false
	parseChild := &Fiber{
		hooks:   &Hooks{cleanups: make([]func(), 1)},
		effects: []Effect{{Fn: func() func() { isParseExecuted = true; return nil }, CleanupIndex: 0}},
	}
	parseParent := &Fiber{child: parseChild}

	parseRt.runEffects(parseParent)

	if !isParseExecuted {
		parseT.Fatal("expected child effect to run even when parent has no effects")
	}
}

func TestPerformUnitOfWork_HostFiberWithoutChildrenProp(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseFiber := &Fiber{typeOf: "div", props: map[string]any{}, dirty: true}

	parseNext := parseRt.performUnitOfWork(parseFiber)

	if parseFiber.dom == nil {
		parseT.Fatal("expected host fiber DOM to be created")
	}
	if parseNext != nil {
		parseT.Fatal("expected no further work without children or siblings")
	}
}

func TestPerformUnitOfWork_FunctionComponentReturningNil(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseFiber := &Fiber{
		typeOf: func(map[string]any) *Element { return nil },
		props:  map[string]any{},
		dirty:  true,
	}

	parseNext := parseRt.performUnitOfWork(parseFiber)

	if parseFiber.child != nil {
		parseT.Fatal("expected no child fibers when component returns nil")
	}
	if parseNext != nil {
		parseT.Fatal("expected no further work")
	}
}

func TestPerformUnitOfWork_FunctionComponentReturningNilDeletesPreviousChild(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseOldChild := &Fiber{
		typeOf: "div",
		props:  map[string]any{"id": "old"},
		dom:    parseRt.domAdapter.CreateElement("div"),
	}
	parseFiber := &Fiber{
		typeOf: func(map[string]any) *Element { return nil },
		props:  map[string]any{},
		dirty:  true,
		alternate: &Fiber{
			child: parseOldChild,
		},
	}

	parseRt.performUnitOfWork(parseFiber)

	if len(parseRt.deletions) != 1 || parseRt.deletions[0] != parseOldChild {
		parseT.Fatal("expected previous child to be marked for deletion when component renders nil")
	}
}

func TestReconcileChildren_NilElementDeletesOldFiber(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseOld := &Fiber{typeOf: "div", props: map[string]any{}, dom: parseRt.domAdapter.CreateElement("div")}
	parseParent := &Fiber{alternate: &Fiber{child: parseOld}}

	parseRt.reconcileChildren(parseParent, []any{nil})

	if len(parseRt.deletions) != 1 || parseRt.deletions[0] != parseOld {
		parseT.Fatal("expected nil element slot to mark old fiber for deletion")
	}
}

func TestRunCleanups_NilFiber(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseRt.runCleanups(nil)
}

func TestRunCleanups_DoesNotTraverseArgumentSiblings(parseT *testing.T) {
	parseRt := NewRuntime(Config{DOMAdapter: newTestDOMAdapter(), Scheduler: newTestScheduler()})
	parseCount := 0
	parseSiblingCount := 0
	parseSibling := &Fiber{hooks: &Hooks{cleanups: []func(){func() { parseSiblingCount++ }}}}
	parseChild := &Fiber{hooks: &Hooks{cleanups: []func(){func() { parseCount++ }}}, sibling: parseSibling}

	// The argument fiber's own sibling chain must NOT be cleaned: commitDeletion
	// passes a deleted fiber whose old-tree siblings may still be live.
	parseRt.runCleanups(parseChild)
	if parseCount != 1 || parseSiblingCount != 0 {
		parseT.Fatalf("expected only the argument fiber cleaned, got self=%d sibling=%d", parseCount, parseSiblingCount)
	}

	// But sibling chains of CHILDREN (inside the subtree) are traversed.
	parseCount = 0
	parseSiblingCount = 0
	parseParent := &Fiber{child: parseChild}
	parseChild.hooks = &Hooks{cleanups: []func(){func() { parseCount++ }}}
	parseSibling.hooks = &Hooks{cleanups: []func(){func() { parseSiblingCount++ }}}
	parseRt.runCleanups(parseParent)
	if parseCount != 1 || parseSiblingCount != 1 {
		parseT.Fatalf("expected full child subtree cleaned, got child=%d childSibling=%d", parseCount, parseSiblingCount)
	}
}
