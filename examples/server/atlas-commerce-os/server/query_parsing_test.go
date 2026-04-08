package main

import (
	"context"
	"net/url"
	"testing"
)

// TestParsePositiveIntReturnsFallback verifies fallback behavior for empty, invalid, and non-positive values.
func TestParsePositiveIntReturnsFallback(parseT *testing.T) {
	parseT.Parallel()

	if parseGot := parsePositiveInt("", 7); parseGot != 7 {
		parseT.Fatalf("parsePositiveInt(empty) = %d, want 7", parseGot)
	}
	if parseGot := parsePositiveInt("abc", 7); parseGot != 7 {
		parseT.Fatalf("parsePositiveInt(invalid) = %d, want 7", parseGot)
	}
	if parseGot := parsePositiveInt("0", 7); parseGot != 7 {
		parseT.Fatalf("parsePositiveInt(zero) = %d, want 7", parseGot)
	}
	if parseGot := parsePositiveInt("-2", 7); parseGot != 7 {
		parseT.Fatalf("parsePositiveInt(negative) = %d, want 7", parseGot)
	}
	if parseGot := parsePositiveInt("9", 7); parseGot != 9 {
		parseT.Fatalf("parsePositiveInt(valid) = %d, want 9", parseGot)
	}
}

// TestParsePublicWarehouseProductQueryDefaultsSort verifies query trimming and default sort canonicalization.
func TestParsePublicWarehouseProductQueryDefaultsSort(parseT *testing.T) {
	parseT.Parallel()

	parseQuery := parsePublicWarehouseProductQuery(url.Values{
		"q":        {"  frame  "},
		"category": {"  desks "},
		"status":   {" low_stock "},
	})
	if parseQuery.Search != "frame" {
		parseT.Fatalf("search trim mismatch: %q", parseQuery.Search)
	}
	if parseQuery.Category != "desks" {
		parseT.Fatalf("category trim mismatch: %q", parseQuery.Category)
	}
	if parseQuery.Status != "low_stock" {
		parseT.Fatalf("status trim mismatch: %q", parseQuery.Status)
	}
	if parseQuery.Sort != "volume" {
		parseT.Fatalf("expected default sort \"volume\", got %q", parseQuery.Sort)
	}

	parseExplicit := parsePublicWarehouseProductQuery(url.Values{"sort": {" updated "}})
	if parseExplicit.Sort != "updated" {
		parseT.Fatalf("expected explicit trimmed sort \"updated\", got %q", parseExplicit.Sort)
	}
}

// TestAtlasDataQueryFiltersFrameworkKeys verifies bootstrap-only query keys are removed while business filters remain.
func TestAtlasDataQueryFiltersFrameworkKeys(parseT *testing.T) {
	parseT.Parallel()

	parseFiltered := atlasDataQuery(url.Values{
		"q":               {"desk"},
		"warehouse":       {"new-jersey-hub"},
		"atlas_notice":    {"saved"},
		"atlas_bootstrap": {"external"},
		"Atlas_Notice":    {"mixed-case"},
	})
	if parseFiltered.Get("q") != "desk" {
		parseT.Fatalf("expected q to remain, got %q", parseFiltered.Get("q"))
	}
	if parseFiltered.Get("warehouse") != "new-jersey-hub" {
		parseT.Fatalf("expected warehouse to remain, got %q", parseFiltered.Get("warehouse"))
	}
	if parseFiltered.Get("atlas_notice") != "" {
		parseT.Fatalf("expected atlas_notice to be filtered, got %q", parseFiltered.Get("atlas_notice"))
	}
	if parseFiltered.Get("atlas_bootstrap") != "" {
		parseT.Fatalf("expected atlas_bootstrap to be filtered, got %q", parseFiltered.Get("atlas_bootstrap"))
	}
}

// TestNormalizeWarehouseItemSortAndDirection verifies invalid sort and direction values canonicalize to supported defaults.
func TestNormalizeWarehouseItemSortAndDirection(parseT *testing.T) {
	parseT.Parallel()

	if parseGot := normalizeWarehouseItemSort(" inbound "); parseGot != "inbound" {
		parseT.Fatalf("normalizeWarehouseItemSort(inbound) = %q, want inbound", parseGot)
	}
	if parseGot := normalizeWarehouseItemSort("not-real"); parseGot != "updated" {
		parseT.Fatalf("normalizeWarehouseItemSort(invalid) = %q, want updated", parseGot)
	}
	if parseGot := normalizeWarehouseItemDirection("warehouse", ""); parseGot != "asc" {
		parseT.Fatalf("normalizeWarehouseItemDirection(warehouse, empty) = %q, want asc", parseGot)
	}
	if parseGot := normalizeWarehouseItemDirection("updated", ""); parseGot != "desc" {
		parseT.Fatalf("normalizeWarehouseItemDirection(updated, empty) = %q, want desc", parseGot)
	}
	if parseGot := normalizeWarehouseItemDirection("updated", " asc "); parseGot != "asc" {
		parseT.Fatalf("normalizeWarehouseItemDirection(updated, asc) = %q, want asc", parseGot)
	}
	if parseGot := normalizeWarehouseItemDirection("updated", "invalid"); parseGot != "desc" {
		parseT.Fatalf("normalizeWarehouseItemDirection(updated, invalid) = %q, want desc", parseGot)
	}
}

// TestInternalInventoryPageDataParsesFiltersAndSort verifies inventory query parsing populates filter echoes and sort behavior.
func TestInternalInventoryPageDataParsesFiltersAndSort(parseT *testing.T) {
	parseServer, parseCleanup := newTestAtlasServer(parseT)
	defer parseCleanup()

	parseData, parseErr := parseServer.internalInventoryPageData(context.Background(), url.Values{
		"q":         {" frame "},
		"warehouse": {" new-jersey-hub "},
		"status":    {" promise_risk "},
		"sort":      {"inbound"},
	})
	if parseErr != nil {
		parseT.Fatalf("internalInventoryPageData: %v", parseErr)
	}
	if parseData.Filters["q"] != "frame" {
		parseT.Fatalf("expected trimmed q filter, got %q", parseData.Filters["q"])
	}
	if parseData.Filters["warehouse"] != "new-jersey-hub" {
		parseT.Fatalf("expected trimmed warehouse filter, got %q", parseData.Filters["warehouse"])
	}
	if parseData.Filters["status"] != "promise_risk" {
		parseT.Fatalf("expected trimmed status filter, got %q", parseData.Filters["status"])
	}
	if parseData.Filters["sort"] != "inbound" {
		parseT.Fatalf("expected sort filter echo, got %q", parseData.Filters["sort"])
	}
	if len(parseData.Items) == 0 {
		parseT.Fatal("expected inventory rows for parsed filters")
	}
	for parseIndex := 1; parseIndex < len(parseData.Items); parseIndex++ {
		if parseData.Items[parseIndex-1].Inbound < parseData.Items[parseIndex].Inbound {
			parseT.Fatalf("expected inbound-desc sort at index %d: %d < %d", parseIndex, parseData.Items[parseIndex-1].Inbound, parseData.Items[parseIndex].Inbound)
		}
	}
}
