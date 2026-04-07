package runtime

import "testing"

type optimizationTestAdapter struct {
	*testDOMAdapter
	setAttributeCount    int
	removeAttributeCount int
	setPropertyCount     int
	setTextContentCount  int
	getFirstChildCount   int
	getNextSiblingCount  int
	insertBeforeCount    int
	beginBatchCount      int
	endBatchCount        int
}

func (parseA *optimizationTestAdapter) SetAttribute(parseNode DOMNode, parseName string, parseValue string) {
	parseA.setAttributeCount++
	parseA.testDOMAdapter.SetAttribute(parseNode, parseName, parseValue)
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

func (parseA *optimizationTestAdapter) BeginBatch(parseParent DOMNode) {
	parseA.beginBatchCount++
}

func (parseA *optimizationTestAdapter) EndBatch() {
	parseA.endBatchCount++
}

func (parseA *optimizationTestAdapter) resetCounts() {
	parseA.setAttributeCount = 0
	parseA.removeAttributeCount = 0
	parseA.setPropertyCount = 0
	parseA.setTextContentCount = 0
	parseA.getFirstChildCount = 0
	parseA.getNextSiblingCount = 0
	parseA.insertBeforeCount = 0
	parseA.beginBatchCount = 0
	parseA.endBatchCount = 0
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

func TestReconcileChildrenReusesHostUpdateWhenOnlyChildrenChange(parseT *testing.T) {
	parseRt := &Runtime{}
	parseOldHost := CreateElement("div", map[string]interface{}{"id": "host"}, "before")
	parseRoot := &Fiber{
		typeOf: "ROOT",
		props:  map[string]interface{}{},
		alternate: &Fiber{
			child: &Fiber{
				typeOf: parseOldHost.Type,
				props:  parseOldHost.Props,
				dom:    (&testDOMAdapter{}).CreateElement("div"),
			},
		},
	}

	parseRt.reconcileChildren(parseRoot, []interface{}{
		CreateElement("div", map[string]interface{}{"id": "host"}, "after"),
	})

	parseChild := parseRoot.child
	if parseChild == nil {
		parseT.Fatal("expected reconciled child fiber")
	}
	if parseChild.effectTag != "UPDATE" {
		parseT.Fatalf("expected reused host fiber to retain UPDATE effect, got %q", parseChild.effectTag)
	}
	if !parseChild.dirty {
		parseT.Fatal("expected child-only change to keep subtree dirty for child reconciliation")
	}
}

func TestRenderSkipsCommittedHostPropWritesWhenOnlyChildrenChange(parseT *testing.T) {
	parseAdapter := &optimizationTestAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRt := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseContainer := parseAdapter.CreateElement("root")

	parseRt.Render(CreateElement("div", map[string]interface{}{"id": "host"}, "before"), parseContainer)
	parseAdapter.resetCounts()

	parseRt.Render(CreateElement("div", map[string]interface{}{"id": "host"}, "after"), parseContainer)

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
		parseT.Fatalf("expected one text child, got %d", len(parseHost.children))
	}
	parseTextNode, parseOk2 := parseHost.children[0].(*testDOMNode)
	if !parseOk2 {
		parseT.Fatalf("expected text child type *testDOMNode, got %T", parseHost.children[0])
	}
	if parseTextNode.text != "after" {
		parseT.Fatalf("expected updated child text %q, got %q", "after", parseTextNode.text)
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
