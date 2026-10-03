package mockdom

import "testing"

// getChildIDs lists a node's child ids in order for compact assertions.
func getChildIDs(parseNode *MockDOMNode) []int {
	parseIDs := make([]int, 0, len(parseNode.Children))
	for _, parseChild := range parseNode.Children {
		parseIDs = append(parseIDs, parseChild.ID)
	}
	return parseIDs
}

// TestInsertBeforeSelfKeepsNode verifies insertBefore(node, node) leaves the
// node in place, as Element.before(self) does in the browser adapter.
func TestInsertBeforeSelfKeepsNode(parseT *testing.T) {
	parseA := NewMockDOMAdapter()
	parseParent := parseA.CreateElement("div").(*MockDOMNode)
	parseX := parseA.CreateElement("span").(*MockDOMNode)
	parseY := parseA.CreateElement("span").(*MockDOMNode)
	parseA.AppendChild(parseParent, parseX)
	parseA.AppendChild(parseParent, parseY)
	parseA.InsertBefore(parseParent, parseX, parseX)
	if len(parseParent.Children) != 2 || parseX.Parent != parseParent {
		parseT.Fatalf("node lost by self insertBefore: children=%v parent=%v", getChildIDs(parseParent), parseX.Parent)
	}
}

// TestReplaceChildDetachesNewNodeFromOldParent verifies a replacement node that
// already lives elsewhere is moved, not duplicated into two parents.
func TestReplaceChildDetachesNewNodeFromOldParent(parseT *testing.T) {
	parseA := NewMockDOMAdapter()
	parseP := parseA.CreateElement("div").(*MockDOMNode)
	parseQ := parseA.CreateElement("div").(*MockDOMNode)
	parseOld := parseA.CreateElement("b").(*MockDOMNode)
	parseNew := parseA.CreateElement("i").(*MockDOMNode)
	parseA.AppendChild(parseP, parseOld)
	parseA.AppendChild(parseQ, parseNew)
	parseA.ReplaceChild(parseP, parseNew, parseOld)
	if len(parseQ.Children) != 0 {
		parseT.Fatalf("new node still listed under its old parent: %v", getChildIDs(parseQ))
	}
	if len(parseP.Children) != 1 || parseP.Children[0] != parseNew || parseOld.Parent != nil {
		parseT.Fatalf("replace result wrong: %v", getChildIDs(parseP))
	}
}

// TestReplaceChildWithSiblingDoesNotDuplicate verifies replacing a child with
// a sibling of the same parent leaves one copy of that sibling.
func TestReplaceChildWithSiblingDoesNotDuplicate(parseT *testing.T) {
	parseA := NewMockDOMAdapter()
	parseP := parseA.CreateElement("div").(*MockDOMNode)
	parseX := parseA.CreateElement("b").(*MockDOMNode)
	parseY := parseA.CreateElement("i").(*MockDOMNode)
	parseZ := parseA.CreateElement("u").(*MockDOMNode)
	parseA.AppendChild(parseP, parseX)
	parseA.AppendChild(parseP, parseY)
	parseA.AppendChild(parseP, parseZ)
	parseA.ReplaceChild(parseP, parseZ, parseX)
	if parseGot := getChildIDs(parseP); len(parseGot) != 2 || parseGot[0] != parseZ.ID || parseGot[1] != parseY.ID {
		parseT.Fatalf("expected [z y], got %v", parseGot)
	}
}

// TestReplaceChildIgnoresNonChild verifies replacing a node that is not a child
// of parent changes nothing, matching the browser adapter's guard.
func TestReplaceChildIgnoresNonChild(parseT *testing.T) {
	parseA := NewMockDOMAdapter()
	parseP := parseA.CreateElement("div").(*MockDOMNode)
	parseQ := parseA.CreateElement("div").(*MockDOMNode)
	parseOld := parseA.CreateElement("b").(*MockDOMNode)
	parseNew := parseA.CreateElement("i").(*MockDOMNode)
	parseA.AppendChild(parseQ, parseOld)
	parseA.AppendChild(parseQ, parseNew)
	parseA.ReplaceChild(parseP, parseNew, parseOld)
	if len(parseP.Children) != 0 || len(parseQ.Children) != 2 {
		parseT.Fatalf("non-child replace mutated the tree: p=%v q=%v", getChildIDs(parseP), getChildIDs(parseQ))
	}
}
