package runtime2

import (
	"strings"
	"testing"
)

// TestCommitRegionPatchTransactionFullyValidCommitsAllOps verifies valid streams commit every operation.
func TestCommitRegionPatchTransactionFullyValidCommitsAllOps(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:    1,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 1,
				GetText:   "after",
			},
			{
				GetKind:      RegionPatchOpKindSetAttr,
				GetNodeID:    1,
				GetAttrKey:   "class",
				GetAttrValue: "card",
			},
		},
	})
	if parseTransactionErr != nil {
		parseTesting.Fatalf("CommitRegionPatchTransaction(valid) error = %v", parseTransactionErr)
	}
	if parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(valid) expected no fallback")
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(host) error = %v", parseLookupErr)
	}
	if parseNode.GetText != "after" {
		parseTesting.Fatalf("GetRegionDOMNode(host) text = %q, want %q", parseNode.GetText, "after")
	}
	if parseNode.GetAttrByKey["class"] != "card" {
		parseTesting.Fatalf("GetRegionDOMNode(host) class = %q, want %q", parseNode.GetAttrByKey["class"], "card")
	}
}

// TestCommitRegionPatchTransactionInvalidMidStreamTriggersFallback verifies invalid mid-stream operations trigger fallback.
func TestCommitRegionPatchTransactionInvalidMidStreamTriggersFallback(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:    1,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	parseTransactionResult, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 1,
				GetText:   "after",
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 999,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(invalid mid-stream) error = nil, want error")
	}
	if !parseTransactionResult.HasFallbackEntered {
		parseTesting.Fatal("CommitRegionPatchTransaction(invalid mid-stream) expected fallback result")
	}
}

// TestCommitRegionPatchTransactionRollsBackPartialState verifies failed streams do not leave partial commit effects active.
func TestCommitRegionPatchTransactionRollsBackPartialState(parseTesting *testing.T) {
	parseRegionDOMIndex := BuildRegionDOMIndex()
	parseInsertErr := parseRegionDOMIndex.SetRegionDOMNode("region-42", 1, &RegionDOMNode{
		GetNodeID:    1,
		GetTag:       "div",
		GetText:      "before",
		GetAttrByKey: map[string]string{},
	})
	if parseInsertErr != nil {
		parseTesting.Fatalf("SetRegionDOMNode(host) error = %v", parseInsertErr)
	}
	parseDOMCommitter := BuildDOMCommitter(parseRegionDOMIndex)
	_, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps: []RegionPatchOp{
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 1,
				GetText:   "after",
			},
			{
				GetKind:   RegionPatchOpKindSetText,
				GetNodeID: 999,
				GetText:   "broken",
			},
		},
	})
	if parseTransactionErr == nil {
		parseTesting.Fatal("CommitRegionPatchTransaction(invalid mid-stream) error = nil, want error")
	}
	parseNode, parseLookupErr := parseRegionDOMIndex.GetRegionDOMNode("region-42", 1)
	if parseLookupErr != nil {
		parseTesting.Fatalf("GetRegionDOMNode(host) error = %v", parseLookupErr)
	}
	if parseNode.GetText != "before" {
		parseTesting.Fatalf("GetRegionDOMNode(host) text after rollback = %q, want %q", parseNode.GetText, "before")
	}
}

// TestCommitRegionPatchTransactionRejectsExcessiveOpCount verifies hard guard limits fail fast before deep snapshot cloning work.
func TestCommitRegionPatchTransactionRejectsExcessiveOpCount(parseTesting *testing.T) {
	parseDOMCommitter := BuildDOMCommitter(BuildRegionDOMIndex())
	parseOps := make([]RegionPatchOp, getDOMCommitTransactionOpHardLimit+1)
	_, parseTransactionErr := parseDOMCommitter.CommitRegionPatchTransaction(RegionPatchTransaction{
		GetRegionID: "region-42",
		GetOps:      parseOps,
	})
	if parseTransactionErr == nil || !strings.Contains(parseTransactionErr.Error(), "exceeds guard limit") {
		parseTesting.Fatalf("CommitRegionPatchTransaction(guard) error = %v, want guard-limit error", parseTransactionErr)
	}
}
