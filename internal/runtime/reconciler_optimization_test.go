package runtime

import "testing"

type optimizationTestAdapter struct {
	*testDOMAdapter
	setAttributeCount      int
	batchSetAttributeCount int
	removeAttributeCount   int
	createPreparedCount    int
	setPropertyCount       int
	setTextContentCount    int
	getFirstChildCount     int
	getNextSiblingCount    int
	insertBeforeCount      int
	replaceChildrenCount   int
	beginBatchCount        int
	endBatchCount          int
}

func (parseA *optimizationTestAdapter) SetAttribute(parseNode DOMNode, parseName string, parseValue string) {
	parseA.setAttributeCount++
	parseA.testDOMAdapter.SetAttribute(parseNode, parseName, parseValue)
}

func (parseA *optimizationTestAdapter) BatchSetAttributes(parseNode DOMNode, parseAttrs map[string]string) {
	parseA.batchSetAttributeCount++
	for parseName, parseValue := range parseAttrs {
		parseA.testDOMAdapter.SetAttribute(parseNode, parseName, parseValue)
	}
}

// CreatePreparedElement records one compact host prepared mount for optimization coverage.
func (parseA *optimizationTestAdapter) CreatePreparedElement(parseTag string, parseAttrs []HostAttr, parseText string) DOMNode {
	parseA.createPreparedCount++
	parseNode := parseA.testDOMAdapter.CreateElement(parseTag)
	for _, parseAttr := range parseAttrs {
		parseA.testDOMAdapter.SetAttribute(parseNode, parseAttr.Name, parseAttr.Value)
	}
	if parseText != "" {
		parseA.testDOMAdapter.SetTextContent(parseNode, parseText)
	}
	return parseNode
}

func (parseA *optimizationTestAdapter) RemoveAttribute(parseNode DOMNode, parseName string) {
	parseA.removeAttributeCount++
	parseA.testDOMAdapter.RemoveAttribute(parseNode, parseName)
}

func (parseA *optimizationTestAdapter) SetProperty(parseNode DOMNode, parseName string, parseValue interface{}) {
	parseA.setPropertyCount++
	parseA.testDOMAdapter.SetProperty(parseNode, parseName, parseValue)
}

func (parseA *optimizationTestAdapter) SetTextContent(parseNode DOMNode, parseText string) {
	parseA.setTextContentCount++
	parseA.testDOMAdapter.SetTextContent(parseNode, parseText)
}

func (parseA *optimizationTestAdapter) GetFirstChild(parseNode DOMNode) DOMNode {
	parseA.getFirstChildCount++
	return parseA.testDOMAdapter.GetFirstChild(parseNode)
}

func (parseA *optimizationTestAdapter) GetNextSibling(parseNode DOMNode) DOMNode {
	parseA.getNextSiblingCount++
	return parseA.testDOMAdapter.GetNextSibling(parseNode)
}

func (parseA *optimizationTestAdapter) InsertBefore(parseParent DOMNode, parseNewChild DOMNode, parseRefChild DOMNode) {
	parseA.insertBeforeCount++
	parseA.testDOMAdapter.InsertBefore(parseParent, parseNewChild, parseRefChild)
}

// ReplaceChildren records one wholesale child replacement for optimization coverage.
func (parseA *optimizationTestAdapter) ReplaceChildren(parseParent DOMNode, parseChildren []DOMNode) {
	parseA.replaceChildrenCount++
	parseA.testDOMAdapter.ReplaceChildren(parseParent, parseChildren)
}

func (parseA *optimizationTestAdapter) BeginBatch(parseParent DOMNode) {
	parseA.beginBatchCount++
}

func (parseA *optimizationTestAdapter) EndBatch() {
	parseA.endBatchCount++
}

func (parseA *optimizationTestAdapter) resetCounts() {
	parseA.setAttributeCount = 0
	parseA.batchSetAttributeCount = 0
	parseA.removeAttributeCount = 0
	parseA.createPreparedCount = 0
	parseA.setPropertyCount = 0
	parseA.setTextContentCount = 0
	parseA.getFirstChildCount = 0
	parseA.getNextSiblingCount = 0
	parseA.insertBeforeCount = 0
	parseA.replaceChildrenCount = 0
	parseA.beginBatchCount = 0
	parseA.endBatchCount = 0
}

// TestCreateDomUsesPreparedCompactHostMount verifies compact host mounts take the prepared fast path instead of generic attr/text writes.
func TestCreateDomUsesPreparedCompactHostMount(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElement("div", map[string]interface{}{
		"id":         "row-1",
		"className":  "card",
		"data-state": "live",
	}, "hello"), parseContainer)

	if parseAdapter.createPreparedCount != 1 {
		parseT.Fatalf("expected one prepared host mount, got %d", parseAdapter.createPreparedCount)
	}
	if parseAdapter.setAttributeCount != 0 {
		parseT.Fatalf("expected prepared host mount to skip generic attr writes, got %d", parseAdapter.setAttributeCount)
	}
	if parseAdapter.setTextContentCount != 0 {
		parseT.Fatalf("expected prepared host mount to skip generic text writes, got %d", parseAdapter.setTextContentCount)
	}
}

func TestCreateElementCompactHostOwnedUsesProvidedAttrs(parseT *testing.T) {
	parseProps := map[string]interface{}{
		"id":    "row-1",
		"class": "card",
	}
	parseElem := CreateElementCompactHostOwned("div", parseProps, []HostAttr{
		{Name: "id", Value: "row-1"},
		{Name: "class", Value: "card"},
	}, "hello")

	if !parseElem.isCompactHostProps {
		parseT.Fatal("expected compact host props")
	}
	if len(parseElem.getHostAttrs) != 2 {
		parseT.Fatalf("expected two provided compact attrs, got %#v", parseElem.getHostAttrs)
	}
	if parseElem.getHostAttrs[0].Name != "id" || parseElem.getHostAttrs[1].Name != "class" {
		parseT.Fatalf("expected provided attrs to be preserved, got %#v", parseElem.getHostAttrs)
	}
	if !parseElem.hasDirectText || parseElem.TextContent != "hello" {
		parseT.Fatalf("expected direct text child, got direct=%v text=%q", parseElem.hasDirectText, parseElem.TextContent)
	}
	parseProps["id"] = "row-2"
	if parseElem.Props["id"] != "row-2" {
		parseT.Fatal("expected owned props map to be retained")
	}
}

func TestCreateElementCompactHostOwnedRendersPreparedMount(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElementCompactHostOwned("div", map[string]interface{}{
		"id":    "row-1",
		"class": "card",
	}, []HostAttr{
		{Name: "id", Value: "row-1"},
		{Name: "class", Value: "card"},
	}, "hello"), parseContainer)

	if parseAdapter.createPreparedCount != 1 {
		parseT.Fatalf("expected compact constructor to use prepared mount, got %d", parseAdapter.createPreparedCount)
	}
	if parseAdapter.setAttributeCount != 0 {
		parseT.Fatalf("expected prepared compact constructor mount to skip generic attrs, got %d", parseAdapter.setAttributeCount)
	}
}

func TestRefreshElementHostPropsAfterCompactConstructorMutation(parseT *testing.T) {
	parseElem := CreateElementCompactHostOwned("input", map[string]interface{}{
		"id": "email",
	}, []HostAttr{{Name: "id", Value: "email"}})

	parseElem.Props["value"] = "cam@example.test"
	RefreshElementHostProps(parseElem)

	if parseElem.isCompactHostProps {
		parseT.Fatal("expected special property mutation to disable compact host props")
	}
	if len(parseElem.getHostAttrs) != 0 {
		parseT.Fatalf("expected compact attrs to be cleared after noncompact mutation, got %#v", parseElem.getHostAttrs)
	}
}

func TestPropsEqualIgnoringChildrenTreatsChildSliceChangesAsStable(parseT *testing.T) {
	parsePrev := map[string]interface{}{
		"id":       "host",
		"children": []interface{}{"before"},
	}
	parseNext := map[string]interface{}{
		"id":       "host",
		"children": []interface{}{"after"},
	}

	if !propsEqualIgnoringChildren(parsePrev, parseNext) {
		parseT.Fatal("expected child-only prop changes to compare equal")
	}

	parseChanged := map[string]interface{}{
		"id":       "host-2",
		"children": []interface{}{"after"},
	}
	if propsEqualIgnoringChildren(parsePrev, parseChanged) {
		parseT.Fatal("expected non-child prop change to compare different")
	}
}

func TestReconcileChildrenSeparatesHostChildReconcileWhenOnlyChildrenChange(parseT *testing.T) {
	parseRt := &Runtime{}
	parseOldHost := CreateElement("div", map[string]interface{}{"id": "host"}, CreateElement("span", nil, "before"))
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{},
		alternate: &Fiber{
			child: &Fiber{
				typeOf:   parseOldHost.Type,
				props:    parseOldHost.Props,
				children: getElementChildren(parseOldHost),
				dom:      (&testDOMAdapter{}).CreateElement("div"),
			},
		},
	}

	parseRt.reconcileChildren(parseRoot, []interface{}{
		CreateElement("div", map[string]interface{}{"id": "host"}, CreateElement("span", nil, "after")),
	})

	parseChild := parseRoot.child
	if parseChild == nil {
		parseT.Fatal("expected reconciled child fiber")
	}
	if parseChild.effectTag != effectTagNone {
		parseT.Fatalf("expected child-only host reconcile to avoid UPDATE effect, got %q", parseChild.effectTag)
	}
	if parseChild.dirty {
		parseT.Fatal("expected child-only change to avoid self-dirty host updates")
	}
	if !parseChild.needsChildReconcile {
		parseT.Fatal("expected child-only change to request child reconciliation")
	}
}

func TestRenderSkipsCommittedHostPropWritesWhenOnlyChildrenChange(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElement("div", map[string]interface{}{"id": "host"}, CreateElement("span", nil, "before")), parseContainer)
	parseAdapter.resetCounts()

	parseRt.Render(CreateElement("div", map[string]interface{}{"id": "host"}, CreateElement("span", nil, "after")), parseContainer)

	if parseAdapter.setAttributeCount != 0 {
		parseT.Fatalf("expected no attribute writes on stable host props, got %d", parseAdapter.setAttributeCount)
	}
	if parseAdapter.removeAttributeCount != 0 {
		parseT.Fatalf("expected no attribute removals on stable host props, got %d", parseAdapter.removeAttributeCount)
	}
	if parseAdapter.setPropertyCount != 0 {
		parseT.Fatalf("expected no property writes on stable host props, got %d", parseAdapter.setPropertyCount)
	}
	parseRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseRootChildren) != 1 {
		parseT.Fatalf("expected one host root child, got %d", len(parseRootChildren))
	}
	parseHost, parseOk := parseRootChildren[0].(*testDOMNode)
	if !parseOk {
		parseT.Fatalf("expected host node type *testDOMNode, got %T", parseRootChildren[0])
	}
	if len(parseHost.children) != 1 {
		parseT.Fatalf("expected one span child, got %d", len(parseHost.children))
	}
	parseSpanNode, parseOk2 := parseHost.children[0].(*testDOMNode)
	if !parseOk2 {
		parseT.Fatalf("expected span child type *testDOMNode, got %T", parseHost.children[0])
	}
	if len(parseSpanNode.children) != 1 {
		parseT.Fatalf("expected span text child, got %d", len(parseSpanNode.children))
	}
	parseTextNode, parseOk3 := parseSpanNode.children[0].(*testDOMNode)
	if !parseOk3 {
		parseT.Fatalf("expected span text child type *testDOMNode, got %T", parseSpanNode.children[0])
	}
	if parseTextNode.text != "after" {
		parseT.Fatalf("expected updated nested text %q, got %q", "after", parseTextNode.text)
	}
}

func TestCreateElementStoresDirectHostTextChild(parseT *testing.T) {
	parseElem := CreateElement("span", map[string]interface{}{"id": "label"}, "hello")

	if !parseElem.hasDirectText {
		parseT.Fatal("expected single host string child to use direct text storage")
	}
	if parseElem.TextContent != "hello" {
		parseT.Fatalf("expected direct text %q, got %q", "hello", parseElem.TextContent)
	}
	if len(getElementChildren(parseElem)) != 0 {
		parseT.Fatalf("expected no structural children, got %d", len(getElementChildren(parseElem)))
	}
	parseChildren, parseOk := parseElem.Props["children"].([]interface{})
	if !parseOk || len(parseChildren) != 1 {
		parseT.Fatalf("expected legacy props children to remain visible, got %#v", parseElem.Props["children"])
	}
}

func TestRenderUsesDirectHostTextWithoutTextFiber(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElement("div", map[string]interface{}{"id": "host"}, "hello"), parseContainer)

	if parseRt.currentRoot == nil || parseRt.currentRoot.child == nil {
		parseT.Fatal("expected committed host fiber")
	}
	parseHostFiber := parseRt.currentRoot.child
	if !parseHostFiber.hasDirectText {
		parseT.Fatal("expected committed host fiber to keep direct text flag")
	}
	if parseHostFiber.child != nil {
		parseT.Fatal("expected direct host text to avoid text child fibers")
	}
	parseRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseRootChildren) != 1 {
		parseT.Fatalf("expected one host child, got %d", len(parseRootChildren))
	}
	parseHostNode, parseOk := parseRootChildren[0].(*testDOMNode)
	if !parseOk {
		parseT.Fatalf("expected host node type *testDOMNode, got %T", parseRootChildren[0])
	}
	if len(parseHostNode.children) != 1 {
		parseT.Fatalf("expected one text DOM child, got %d", len(parseHostNode.children))
	}
	parseTextNode, parseOk2 := parseHostNode.children[0].(*testDOMNode)
	if !parseOk2 || parseTextNode.text != "hello" {
		parseT.Fatalf("expected host text DOM child %q, got %#v", "hello", parseHostNode.children[0])
	}
}

func TestRenderTransitionsDirectHostTextToElementChildren(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElement("div", map[string]interface{}{"id": "host"}, "before"), parseContainer)
	parseAdapter.resetCounts()
	parseRt.Render(CreateElement("div", map[string]interface{}{"id": "host"}, CreateElement("span", map[string]interface{}{"id": "next"}, "after")), parseContainer)

	parseRootChildren := parseAdapter.GetChildren(parseContainer)
	parseHostNode, parseOk := parseRootChildren[0].(*testDOMNode)
	if !parseOk {
		parseT.Fatalf("expected host node type *testDOMNode, got %T", parseRootChildren[0])
	}
	if len(parseHostNode.children) != 1 {
		parseT.Fatalf("expected direct text to be cleared before child placement, got %d children", len(parseHostNode.children))
	}
	parseSpanNode, parseOk2 := parseHostNode.children[0].(*testDOMNode)
	if !parseOk2 || parseSpanNode.tag != "span" {
		parseT.Fatalf("expected span child after direct-text transition, got %#v", parseHostNode.children[0])
	}
	if parseAdapter.setTextContentCount == 0 {
		parseT.Fatal("expected direct text transition to clear host text content")
	}
}

func TestHydrateUsesDirectHostTextWithoutTextFiber(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")
	parseServerNode := parseAdapter.CreateElement("p")
	parseAdapter.SetTextContent(parseServerNode, "server")
	parseAdapter.AppendChild(parseContainer, parseServerNode)

	parseRt.Hydrate(CreateElement("p", map[string]interface{}{"id": "greeting"}, "client"), parseContainer)

	if parseRt.currentRoot == nil || parseRt.currentRoot.child == nil {
		parseT.Fatal("expected hydrated host fiber")
	}
	parseHostFiber := parseRt.currentRoot.child
	if !parseHostFiber.hasDirectText {
		parseT.Fatal("expected hydrated host fiber to keep direct text flag")
	}
	if parseHostFiber.child != nil {
		parseT.Fatal("expected hydrated direct host text to avoid text child fibers")
	}
	parseRootChildren := parseAdapter.GetChildren(parseContainer)
	parseHostNode, parseOk := parseRootChildren[0].(*testDOMNode)
	if !parseOk {
		parseT.Fatalf("expected hydrated host node type *testDOMNode, got %T", parseRootChildren[0])
	}
	if len(parseHostNode.children) != 1 {
		parseT.Fatalf("expected one hydrated text DOM child, got %d", len(parseHostNode.children))
	}
	parseTextNode, parseOk2 := parseHostNode.children[0].(*testDOMNode)
	if !parseOk2 || parseTextNode.text != "client" {
		parseT.Fatalf("expected hydrated text %q, got %#v", "client", parseHostNode.children[0])
	}
}

func TestApplyInitialDomPropsSkipsBatchMapForTwoStringAttrs(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseDom := parseAdapter.CreateElement("div")

	parseRt.updateDomProperties(parseDom, nil, map[string]interface{}{
		"id":        "row-1",
		"className": "card",
	})

	if parseAdapter.batchSetAttributeCount != 0 {
		parseT.Fatalf("expected small attr mount path to avoid batch map, got %d batch calls", parseAdapter.batchSetAttributeCount)
	}
	if parseAdapter.setAttributeCount != 2 {
		parseT.Fatalf("expected direct attribute writes for two attrs, got %d", parseAdapter.setAttributeCount)
	}
}

func TestApplyInitialDomPropsBatchesThreeStringAttrs(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseDom := parseAdapter.CreateElement("div")

	parseRt.updateDomProperties(parseDom, nil, map[string]interface{}{
		"id":         "row-1",
		"className":  "card",
		"data-state": "live",
	})

	if parseAdapter.batchSetAttributeCount != 1 {
		parseT.Fatalf("expected three-attr mount path to use one batch call, got %d", parseAdapter.batchSetAttributeCount)
	}
}

func TestCommitRootSkipsChildOrderRepairWhenOrderIsStable(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRenderList := func(parseAlpha string, parseBeta string, parseGamma string) *Element {
		return CreateElement("section", map[string]interface{}{"id": "host"},
			CreateElement("div", map[string]interface{}{"id": "a", "key": "a"}, parseAlpha),
			CreateElement("div", map[string]interface{}{"id": "b", "key": "b"}, parseBeta),
			CreateElement("div", map[string]interface{}{"id": "c", "key": "c"}, parseGamma),
		)
	}

	parseRt.Render(parseRenderList("a1", "b1", "c1"), parseContainer)
	parseAdapter.resetCounts()

	parseRt.Render(parseRenderList("a2", "b2", "c2"), parseContainer)

	if parseAdapter.getFirstChildCount != 0 || parseAdapter.getNextSiblingCount != 0 || parseAdapter.insertBeforeCount != 0 {
		parseT.Fatalf("expected stable order commit to skip repair scans, got first=%d next=%d insert=%d", parseAdapter.getFirstChildCount, parseAdapter.getNextSiblingCount, parseAdapter.insertBeforeCount)
	}
}

func TestCommitRootRepairsChildOrderWhenKeyedChildrenMove(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRenderList := func(parseIDs ...string) *Element {
		parseChildren := make([]interface{}, 0, len(parseIDs))
		for _, parseID := range parseIDs {
			parseChildren = append(parseChildren, CreateElement("div", map[string]interface{}{"id": parseID, "key": parseID}, parseID))
		}
		return CreateElement("section", map[string]interface{}{"id": "host"}, parseChildren...)
	}

	parseRt.Render(parseRenderList("a", "b", "c"), parseContainer)
	parseAdapter.resetCounts()

	parseRt.Render(parseRenderList("c", "a", "b"), parseContainer)

	if parseRt.currentRoot == nil || parseRt.currentRoot.child == nil {
		parseT.Fatal("expected committed root child after keyed reorder render")
	}
	if !parseRt.currentRoot.child.needsChildOrder {
		parseT.Fatal("expected keyed reorder to mark the host fiber for child-order repair")
	}

	if parseAdapter.getFirstChildCount == 0 && parseAdapter.getNextSiblingCount == 0 && parseAdapter.insertBeforeCount == 0 {
		parseT.Fatal("expected keyed reorder to trigger child-order repair work")
	}

	parseRootChildren := parseAdapter.GetChildren(parseContainer)
	if len(parseRootChildren) != 1 {
		parseT.Fatalf("expected one host root child, got %d", len(parseRootChildren))
	}
	parseHost, parseOk := parseRootChildren[0].(*testDOMNode)
	if !parseOk {
		parseT.Fatalf("expected host node type *testDOMNode, got %T", parseRootChildren[0])
	}
	parseExpectedIDs := []string{"c", "a", "b"}
	for parseIndex, parseExpectedID := range parseExpectedIDs {
		parseChild, parseOk2 := parseHost.children[parseIndex].(*testDOMNode)
		if !parseOk2 {
			parseT.Fatalf("expected child %d to be *testDOMNode, got %T", parseIndex, parseHost.children[parseIndex])
		}
		if parseChild.attributes["id"] != parseExpectedID {
			parseT.Fatalf("expected child order %v, got id %q at index %d", parseExpectedIDs, parseChild.attributes["id"], parseIndex)
		}
	}
}

func TestApplyCommittedChildOrderReusesObservedChildScan(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseParent := parseAdapter.CreateElement("section")
	parseNodeA := parseAdapter.CreateElement("div")
	parseNodeB := parseAdapter.CreateElement("div")
	parseNodeC := parseAdapter.CreateElement("div")

	parseAdapter.AppendChild(parseParent, parseNodeC)
	parseAdapter.AppendChild(parseParent, parseNodeA)
	parseAdapter.AppendChild(parseParent, parseNodeB)
	parseAdapter.resetCounts()

	parseRt.applyCommittedChildOrder(parseParent, []DOMNode{parseNodeA, parseNodeB, parseNodeC})

	if parseAdapter.getFirstChildCount != 1 {
		parseT.Fatalf("expected one observed-child scan, got %d first-child reads", parseAdapter.getFirstChildCount)
	}
	if parseAdapter.getNextSiblingCount != 3 {
		parseT.Fatalf("expected one observed-child walk across three children, got %d next-sibling reads", parseAdapter.getNextSiblingCount)
	}
	parseChildren := parseAdapter.GetChildren(parseParent)
	if len(parseChildren) != 3 {
		parseT.Fatalf("expected one repaired child set, got %d children", len(parseChildren))
	}
	if !IsSameDOMNode(parseChildren[0], parseNodeA) || !IsSameDOMNode(parseChildren[1], parseNodeB) || !IsSameDOMNode(parseChildren[2], parseNodeC) {
		parseT.Fatalf("expected repaired child order A,B,C, got %#v", parseChildren)
	}
}

// TestApplyCommittedChildOrderUsesReplaceChildrenFastPath verifies large reorder repairs collapse to one replaceChildren call when the observed child set matches.
func TestApplyCommittedChildOrderUsesReplaceChildrenFastPath(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseParent := parseAdapter.CreateElement("section")
	getExpected := make([]DOMNode, 0, getCommittedChildReplaceThreshold)

	for getIndex := 0; getIndex < getCommittedChildReplaceThreshold; getIndex++ {
		getNode := parseAdapter.CreateElement("div")
		getExpected = append(getExpected, getNode)
	}
	for getIndex := len(getExpected) - 1; getIndex >= 0; getIndex-- {
		parseAdapter.AppendChild(parseParent, getExpected[getIndex])
	}
	parseAdapter.resetCounts()

	parseRt.applyCommittedChildOrder(parseParent, getExpected)

	if parseAdapter.replaceChildrenCount != 1 {
		parseT.Fatalf("expected one replaceChildren fast path call, got %d", parseAdapter.replaceChildrenCount)
	}
	if parseAdapter.insertBeforeCount != 0 {
		parseT.Fatalf("expected replaceChildren fast path to skip incremental inserts, got %d", parseAdapter.insertBeforeCount)
	}
	parseChildren := parseAdapter.GetChildren(parseParent)
	if len(parseChildren) != len(getExpected) {
		parseT.Fatalf("expected %d repaired children, got %d", len(getExpected), len(parseChildren))
	}
	for getIndex, getExpectedNode := range getExpected {
		if !IsSameDOMNode(parseChildren[getIndex], getExpectedNode) {
			parseT.Fatalf("expected child %d to match repaired order", getIndex)
		}
	}
}

// TestApplyCommittedChildOrderSkipsReplaceChildrenWithUnexpectedObservedNode verifies the wholesale fast path does not drop unexpected siblings that the incremental repair logic preserves.
func TestApplyCommittedChildOrderSkipsReplaceChildrenWithUnexpectedObservedNode(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseParent := parseAdapter.CreateElement("section")
	getExpected := make([]DOMNode, 0, getCommittedChildReplaceThreshold)

	for getIndex := 0; getIndex < getCommittedChildReplaceThreshold; getIndex++ {
		getNode := parseAdapter.CreateElement("div")
		getExpected = append(getExpected, getNode)
	}
	getUnexpected := parseAdapter.CreateElement("aside")
	parseAdapter.AppendChild(parseParent, getUnexpected)
	for getIndex := len(getExpected) - 1; getIndex >= 0; getIndex-- {
		parseAdapter.AppendChild(parseParent, getExpected[getIndex])
	}
	parseAdapter.resetCounts()

	parseRt.applyCommittedChildOrder(parseParent, getExpected)

	if parseAdapter.replaceChildrenCount != 0 {
		parseT.Fatalf("expected unexpected observed child to skip replaceChildren, got %d calls", parseAdapter.replaceChildrenCount)
	}
	parseChildren := parseAdapter.GetChildren(parseParent)
	if len(parseChildren) != len(getExpected)+1 {
		parseT.Fatalf("expected unexpected child to be preserved, got %d children", len(parseChildren))
	}
	if !IsSameDOMNode(parseChildren[len(parseChildren)-1], getUnexpected) {
		parseT.Fatal("expected unexpected child to remain attached")
	}
}

// TestShouldBatchCommittedPlacementsKeepsBatchingWithChildOrderRepair verifies keyed placement subtrees can still use append batching before the post-commit child-order repair pass.
func TestShouldBatchCommittedPlacementsKeepsBatchingWithChildOrderRepair(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseParent := parseAdapter.CreateElement("section")
	parseFiber := &Fiber{
		needsChildOrder: true,
		child: &Fiber{
			effectTag: effectTagPlacement,
			dom:       parseAdapter.CreateElement("div"),
			sibling: &Fiber{
				effectTag: effectTagPlacement,
				dom:       parseAdapter.CreateElement("div"),
			},
		},
	}

	if !parseRt.shouldBatchCommittedPlacements(parseFiber, parseParent) {
		parseT.Fatal("expected child-order repair subtree to keep placement batching enabled")
	}
}

func TestCommitRootBatchesStablePlacementChildren(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElement("section", map[string]interface{}{"id": "host"},
		CreateElement("div", map[string]interface{}{"id": "a"}, "a"),
		CreateElement("div", map[string]interface{}{"id": "b"}, "b"),
		CreateElement("div", map[string]interface{}{"id": "c"}, "c"),
	), parseContainer)

	if parseAdapter.beginBatchCount == 0 || parseAdapter.endBatchCount == 0 {
		parseT.Fatalf("expected stable child placements to use batch adapter, got begin=%d end=%d", parseAdapter.beginBatchCount, parseAdapter.endBatchCount)
	}
	if parseAdapter.beginBatchCount != parseAdapter.endBatchCount {
		parseT.Fatalf("expected balanced batch calls, got begin=%d end=%d", parseAdapter.beginBatchCount, parseAdapter.endBatchCount)
	}
}

func TestCommitRootBatchesRootLevelPlacements(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElement("FRAGMENT", nil,
		CreateElement("div", map[string]interface{}{"id": "a"}, "a"),
		CreateElement("div", map[string]interface{}{"id": "b"}, "b"),
		CreateElement("div", map[string]interface{}{"id": "c"}, "c"),
	), parseContainer)

	if parseAdapter.beginBatchCount == 0 || parseAdapter.endBatchCount == 0 {
		parseT.Fatalf("expected root-level placements to use batch adapter, got begin=%d end=%d", parseAdapter.beginBatchCount, parseAdapter.endBatchCount)
	}
	if parseAdapter.beginBatchCount != parseAdapter.endBatchCount {
		parseT.Fatalf("expected balanced root batch calls, got begin=%d end=%d", parseAdapter.beginBatchCount, parseAdapter.endBatchCount)
	}
}

func TestPerformUnitOfWorkReusesCleanSubtreeWithoutCloning(parseT *testing.T) {
	parseRt := &Runtime{}
	parseOldGrandchild := &Fiber{typeOf: "span"}
	parseOldChild := &Fiber{typeOf: "div", child: parseOldGrandchild}
	parseOldGrandchild.parent = parseOldChild
	parseOldParent := &Fiber{typeOf: "section", child: parseOldChild}
	parseOldChild.parent = parseOldParent

	parseNewParent := &Fiber{
		typeOf:    "section",
		alternate: parseOldParent,
	}

	parseNext := parseRt.performUnitOfWork(parseNewParent)

	if parseNext != nil {
		parseT.Fatalf("expected clean subtree bailout to skip descendants, got next %v", parseNext)
	}
	if parseNewParent.child != parseOldChild {
		parseT.Fatal("expected clean subtree bailout to reuse committed child chain")
	}
	if parseOldChild.parent != parseNewParent {
		parseT.Fatal("expected reused child chain to relink to the new parent fiber")
	}
	if parseOldGrandchild.parent != parseOldChild {
		parseT.Fatal("expected reused descendants to preserve their existing parent chain")
	}
}

func TestRunPendingEffectsPreservesQueuedOrder(parseT *testing.T) {
	parseRt := &Runtime{}
	parseCalls := make([]string, 0, 2)

	parseParent := &Fiber{
		hooks: &Hooks{cleanups: make([]func(), 1)},
		effects: []Effect{{
			Fn: func() func() {
				parseCalls = append(parseCalls, "parent")
				return nil
			},
			CleanupIndex: 0,
		}},
	}
	parseChild := &Fiber{
		hooks: &Hooks{cleanups: make([]func(), 1)},
		effects: []Effect{{
			Fn: func() func() {
				parseCalls = append(parseCalls, "child")
				return nil
			},
			CleanupIndex: 0,
		}},
	}

	parseRt.queuePendingEffectFiber(nil)
	parseRt.queuePendingEffectFiber(&Fiber{})
	parseRt.queuePendingEffectFiber(parseParent)
	parseRt.queuePendingEffectFiber(parseChild)
	parseRt.runPendingEffects()

	if len(parseCalls) != 2 {
		parseT.Fatalf("expected two queued effects to run, got %d", len(parseCalls))
	}
	if parseCalls[0] != "parent" || parseCalls[1] != "child" {
		parseT.Fatalf("expected queued effect order [parent child], got %v", parseCalls)
	}
	if len(parseRt.pendingEffectFibers) != 0 {
		parseT.Fatalf("expected pending effect queue to be cleared, got %d entries", len(parseRt.pendingEffectFibers))
	}
}
