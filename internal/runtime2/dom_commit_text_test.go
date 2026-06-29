package runtime2

import (
	"strings"
	"testing"
)

// TestCommitRegionTextUpdateChangesTargetNode verifies set-text commit updates the expected node.
func TestCommitRegionTextUpdateChangesTargetNode(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 17, &RegionDOMNode{
		GetNodeID: 17,
		GetTag:    "text",
		GetText:   "before",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(text) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseCommitResult, parseCommitErr := parseDOMCommitter.CommitRegionText("region-42", 17, "after")
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionText(valid) error = %v", parseCommitErr)
	}
	if parseCommitResult.IsNoOp {
		parseTesting.Fatal("CommitRegionText(valid) expected non-no-op result")
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 17)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(updated text node) error = %v", parseLookupErr)
	}
	if parseNode.GetText != "after" {
		parseTesting.Fatalf("GetRegionDOMNode(updated text node) text = %q, want %q", parseNode.GetText, "after")
	}
}

// TestCommitRegionTextUnknownNodeFails verifies set-text commit fails clearly for unknown nodes.
func TestCommitRegionTextUnknownNodeFails(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseCommitErr := parseDOMCommitter.CommitRegionText("region-42", 999, "after")
	if parseCommitErr == nil {
		parseTesting.Fatal("CommitRegionText(unknown node) error = nil, want error")
	}
	if !strings.Contains(parseCommitErr.Error(), "missing node ID") {
		parseTesting.Fatalf("CommitRegionText(unknown node) error = %q, want missing-node guidance", parseCommitErr.Error())
	}
}

// TestCommitRegionTextDuplicateValueReturnsNoOp verifies repeated same-value text updates return an explicit no-op.
func TestCommitRegionTextDuplicateValueReturnsNoOp(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 17, &RegionDOMNode{
		GetNodeID: 17,
		GetTag:    "text",
		GetText:   "steady",
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(text) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseCommitResult, parseCommitErr := parseDOMCommitter.CommitRegionText("region-42", 17, "steady")
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionText(duplicate) error = %v", parseCommitErr)
	}
	if !parseCommitResult.IsNoOp {
		parseTesting.Fatal("CommitRegionText(duplicate) expected no-op result")
	}
}
