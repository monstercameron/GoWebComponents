package runtime2

import (
	"strings"
	"testing"
)

// TestCommitRegionAttrClassUpdateCommits verifies class-attr commits apply to the target node.
func TestCommitRegionAttrClassUpdateCommits(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 17, &RegionDOMNode{
		GetNodeID:    17,
		GetTag:       "div",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseCommitErr := parseDOMCommitter.CommitRegionAttr("region-42", 17, "class", "card hot")
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionAttr(class) error = %v", parseCommitErr)
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 17)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(host) error = %v", parseLookupErr)
	}
	if parseNode.GetAttrByKey["class"] != "card hot" {
		parseTesting.Fatalf("CommitRegionAttr(class) value = %q, want %q", parseNode.GetAttrByKey["class"], "card hot")
	}
}

// TestCommitRegionAttrDataUpdateCommits verifies data-attr commits apply to the target node.
func TestCommitRegionAttrDataUpdateCommits(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 17, &RegionDOMNode{
		GetNodeID:    17,
		GetTag:       "div",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseCommitErr := parseDOMCommitter.CommitRegionAttr("region-42", 17, "data-state", "healthy")
	if parseCommitErr != nil {
		parseTesting.Fatalf("CommitRegionAttr(data-*) error = %v", parseCommitErr)
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 17)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(host) error = %v", parseLookupErr)
	}
	if parseNode.GetAttrByKey["data-state"] != "healthy" {
		parseTesting.Fatalf("CommitRegionAttr(data-*) value = %q, want %q", parseNode.GetAttrByKey["data-state"], "healthy")
	}
}

// TestCommitRegionAttrUnsupportedKindFailsBeforeMutation verifies unsupported attrs fail before mutating node state.
func TestCommitRegionAttrUnsupportedKindFailsBeforeMutation(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 17, &RegionDOMNode{
		GetNodeID: 17,
		GetTag:    "div",
		GetAttrByKey: map[string]string{
			"class": "steady",
		},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseCommitErr := parseDOMCommitter.CommitRegionAttr("region-42", 17, "onclick", "alert(1)")
	if parseCommitErr == nil {
		parseTesting.Fatal("CommitRegionAttr(unsupported) error = nil, want error")
	}
	if !strings.Contains(parseCommitErr.Error(), "unsupported attr kind") {
		parseTesting.Fatalf("CommitRegionAttr(unsupported) error = %q, want unsupported-attr guidance", parseCommitErr.Error())
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 17)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(host) error = %v", parseLookupErr)
	}
	if parseNode.GetAttrByKey["class"] != "steady" {
		parseTesting.Fatalf("CommitRegionAttr(unsupported) changed class unexpectedly to %q", parseNode.GetAttrByKey["class"])
	}
}
