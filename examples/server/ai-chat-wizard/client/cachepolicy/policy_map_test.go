package cachepolicy

import (
	"testing"
)

// TestBuildExampleCachePolicyMapCoverage verifies all expected example-100 cache/outbox resources are represented.
func TestBuildExampleCachePolicyMapCoverage(parseT *testing.T) {
	parseEntries := BuildExampleCachePolicyMap()
	if len(parseEntries) != 11 {
		parseT.Fatalf("expected 11 policy rows, got %d", len(parseEntries))
	}
	parseExpectedIDs := map[string]bool{
		"locale_catalog":         false,
		"undelivered_log_outbox": false,
		"unsent_message_outbox":  false,
		"thread_draft":           false,
		"conversation_list_page": false,
		"thread_history":         false,
		"settings_snapshot":      false,
		"model_catalog_metadata": false,
		"billing_summary":        false,
		"admin_dashboard":        false,
		"canvas_session":         false,
	}
	for _, parseEntry := range parseEntries {
		if _, isParseExpected := parseExpectedIDs[parseEntry.ResourceID]; !isParseExpected {
			parseT.Fatalf("unexpected cache policy resource id %q", parseEntry.ResourceID)
		}
		parseExpectedIDs[parseEntry.ResourceID] = true
		if parseEntry.ResourceKeyPattern == "" || parseEntry.ScopeKeyShape == "" || parseEntry.OfflineBehavior == "" || parseEntry.ConsistencyMode == "" {
			parseT.Fatalf("policy row missing required metadata: %+v", parseEntry)
		}
		if len(parseEntry.InvalidationTriggers) == 0 {
			parseT.Fatalf("policy row missing invalidation triggers: %+v", parseEntry)
		}
	}
	for parseID, isParseSeen := range parseExpectedIDs {
		if !isParseSeen {
			parseT.Fatalf("missing policy map entry for %q", parseID)
		}
	}
}

// TestGetExampleCachePolicyEntry verifies resource-id lookup behavior.
func TestGetExampleCachePolicyEntry(parseT *testing.T) {
	parseEntry, isParseFound := GetExampleCachePolicyEntry("billing_summary")
	if !isParseFound {
		parseT.Fatal("expected billing_summary entry")
	}
	if parseEntry.ResourceID != "billing_summary" {
		parseT.Fatalf("unexpected entry returned: %+v", parseEntry)
	}
	if _, isParseFound = GetExampleCachePolicyEntry("unknown"); isParseFound {
		parseT.Fatal("expected unknown entry lookup to fail")
	}
}
