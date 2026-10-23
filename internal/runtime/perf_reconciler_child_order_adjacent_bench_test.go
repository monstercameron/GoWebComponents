package runtime

import "testing"

type adjacentOrderBenchmarkNode struct {
	parent *adjacentOrderBenchmarkNode
	first  *adjacentOrderBenchmarkNode
	last   *adjacentOrderBenchmarkNode
	prev   *adjacentOrderBenchmarkNode
	next   *adjacentOrderBenchmarkNode
}

func (parseN *adjacentOrderBenchmarkNode) IsNull() bool { return parseN == nil }

func (parseN *adjacentOrderBenchmarkNode) Equals(parseOther DOMNode) bool {
	parseOtherNode, parseOk := parseOther.(*adjacentOrderBenchmarkNode)
	return parseOk && parseN == parseOtherNode
}

type adjacentOrderBenchmarkAdapter struct{ *testDOMAdapter }

// CreateElement creates an O(1)-linked benchmark element.
func (parseA *adjacentOrderBenchmarkAdapter) CreateElement(string) DOMNode {
	return &adjacentOrderBenchmarkNode{}
}

// CreateTextNode creates an O(1)-linked benchmark text node.
func (parseA *adjacentOrderBenchmarkAdapter) CreateTextNode(string) DOMNode {
	return &adjacentOrderBenchmarkNode{}
}

// AppendChild attaches one benchmark node after the current last child.
func (parseA *adjacentOrderBenchmarkAdapter) AppendChild(parseParent, parseChild DOMNode) {
	parseParentNode, parseParentOk := parseParent.(*adjacentOrderBenchmarkNode)
	parseChildNode, parseChildOk := parseChild.(*adjacentOrderBenchmarkNode)
	if !parseParentOk || !parseChildOk {
		return
	}
	parseA.detachAdjacentOrderBenchmarkNode(parseChildNode)
	parseChildNode.parent = parseParentNode
	parseChildNode.prev = parseParentNode.last
	parseChildNode.next = nil
	if parseParentNode.last == nil {
		parseParentNode.first = parseChildNode
	} else {
		parseParentNode.last.next = parseChildNode
	}
	parseParentNode.last = parseChildNode
}

// InsertBefore moves one benchmark node before a sibling in constant time.
func (parseA *adjacentOrderBenchmarkAdapter) InsertBefore(parseParent, parseNewNode, parseRefNode DOMNode) {
	parseParentNode, parseParentOk := parseParent.(*adjacentOrderBenchmarkNode)
	parseNewNodeNode, parseNewOk := parseNewNode.(*adjacentOrderBenchmarkNode)
	parseRefNodeNode, parseRefOk := parseRefNode.(*adjacentOrderBenchmarkNode)
	if !parseParentOk || !parseNewOk || !parseRefOk || parseRefNodeNode.parent != parseParentNode {
		return
	}
	parseA.detachAdjacentOrderBenchmarkNode(parseNewNodeNode)
	parseNewNodeNode.parent = parseParentNode
	parseNewNodeNode.prev = parseRefNodeNode.prev
	parseNewNodeNode.next = parseRefNodeNode
	if parseRefNodeNode.prev == nil {
		parseParentNode.first = parseNewNodeNode
	} else {
		parseRefNodeNode.prev.next = parseNewNodeNode
	}
	parseRefNodeNode.prev = parseNewNodeNode
}

// detachAdjacentOrderBenchmarkNode unlinks one benchmark node from its parent.
func (parseA *adjacentOrderBenchmarkAdapter) detachAdjacentOrderBenchmarkNode(parseNode *adjacentOrderBenchmarkNode) {
	if parseNode.parent == nil {
		return
	}
	if parseNode.prev == nil {
		parseNode.parent.first = parseNode.next
	} else {
		parseNode.prev.next = parseNode.next
	}
	if parseNode.next == nil {
		parseNode.parent.last = parseNode.prev
	} else {
		parseNode.next.prev = parseNode.prev
	}
	parseNode.parent = nil
	parseNode.prev = nil
	parseNode.next = nil
}

// GetFirstChild returns the first linked benchmark child.
func (parseA *adjacentOrderBenchmarkAdapter) GetFirstChild(parseNode DOMNode) DOMNode {
	parseNodeValue, parseOk := parseNode.(*adjacentOrderBenchmarkNode)
	if !parseOk || parseNodeValue.first == nil {
		return nil
	}
	return parseNodeValue.first
}

// GetNextSibling returns the next linked benchmark sibling.
func (parseA *adjacentOrderBenchmarkAdapter) GetNextSibling(parseNode DOMNode) DOMNode {
	parseNodeValue, parseOk := parseNode.(*adjacentOrderBenchmarkNode)
	if !parseOk || parseNodeValue.next == nil {
		return nil
	}
	return parseNodeValue.next
}

// buildAdjacentOrderBenchmarkFixture creates a linked child list and a target order with one middle swap.
func buildAdjacentOrderBenchmarkFixture(parseCount int) (*Runtime, DOMNode, []DOMNode, []*adjacentOrderBenchmarkNode) {
	parseAdapter := &adjacentOrderBenchmarkAdapter{testDOMAdapter: newTestDOMAdapter()}
	parseRuntime := NewRuntime(Config{DOMAdapter: parseAdapter})
	parseParent := parseAdapter.CreateElement("section")
	parseNodes := make([]*adjacentOrderBenchmarkNode, parseCount)
	parseExpected := make([]DOMNode, parseCount)
	for parseIndex := range parseNodes {
		parseNode := parseAdapter.CreateElement("div").(*adjacentOrderBenchmarkNode)
		parseNodes[parseIndex] = parseNode
		parseAdapter.AppendChild(parseParent, parseNode)
		parseExpected[parseIndex] = parseNode
	}
	parseSwapIndex := parseCount / 2
	parseExpected[parseSwapIndex], parseExpected[parseSwapIndex+1] = parseExpected[parseSwapIndex+1], parseExpected[parseSwapIndex]
	return parseRuntime, parseParent, parseExpected, parseNodes
}

// resetAdjacentOrderBenchmarkFixture restores the original linked child order without allocating.
func resetAdjacentOrderBenchmarkFixture(parseParent DOMNode, parseNodes []*adjacentOrderBenchmarkNode) {
	parseParentNode := parseParent.(*adjacentOrderBenchmarkNode)
	parseParentNode.first = parseNodes[0]
	parseParentNode.last = parseNodes[len(parseNodes)-1]
	for parseIndex, parseNode := range parseNodes {
		parseNode.parent = parseParentNode
		parseNode.prev = nil
		parseNode.next = nil
		if parseIndex > 0 {
			parseNode.prev = parseNodes[parseIndex-1]
			parseNodes[parseIndex-1].next = parseNode
		}
	}
}

// BenchmarkApplyCommittedChildOrderAdjacentSwap256 measures one middle adjacent swap with O(1) DOM sibling traversal.
func BenchmarkApplyCommittedChildOrderAdjacentSwap256(parseB *testing.B) {
	benchmarkAdjacentOrderSwap(parseB, 256)
}

// BenchmarkApplyCommittedChildOrderAdjacentSwap1000 measures one middle adjacent swap with O(1) DOM sibling traversal.
func BenchmarkApplyCommittedChildOrderAdjacentSwap1000(parseB *testing.B) {
	benchmarkAdjacentOrderSwap(parseB, 1000)
}

// benchmarkAdjacentOrderSwap runs one adjacent swap at the requested list size.
func benchmarkAdjacentOrderSwap(parseB *testing.B, parseCount int) {
	parseRuntime, parseParent, parseExpected, parseNodes := buildAdjacentOrderBenchmarkFixture(parseCount)
	parseB.ReportAllocs()
	parseB.ResetTimer()
	for parseB.Loop() {
		parseB.StopTimer()
		resetAdjacentOrderBenchmarkFixture(parseParent, parseNodes)
		parseB.StartTimer()
		parseRuntime.applyCommittedChildOrder(parseParent, parseExpected)
	}
}
