package app

import "testing"

// TestProviderUsageDailyRollups verifies provider usage daily rollups refresh from usage events and remain queryable.
func TestProviderUsageDailyRollups(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseSeedAdminDashboardTestData(parseT, parseStore)

	parseSince := "2000-01-01T00:00:00Z"
	if parseErr := parseStore.parseRefreshProviderUsageDailyRollups(parseSince); parseErr != nil {
		parseT.Fatalf("parseRefreshProviderUsageDailyRollups: %v", parseErr)
	}
	parseRows, parseErr := parseStore.parseListProviderUsageDailyRollups(parseSince, 100)
	if parseErr != nil {
		parseT.Fatalf("parseListProviderUsageDailyRollups: %v", parseErr)
	}
	if len(parseRows) == 0 {
		parseT.Fatalf("expected provider usage rollup rows")
	}

	parseTotalUsageEvents := int64(0)
	parseProviderSeen := map[string]struct{}{}
	for _, parseRow := range parseRows {
		parseTotalUsageEvents += parseRow.UsageEventCount
		parseProviderSeen[parseRow.ProviderID] = struct{}{}
	}
	if parseTotalUsageEvents < 3 {
		parseT.Fatalf("expected rollup usage count >= 3 from seeded usage events, got %d", parseTotalUsageEvents)
	}
	if _, hasParseOpenAI := parseProviderSeen["openai"]; !hasParseOpenAI {
		parseT.Fatalf("expected openai provider rollup row, got %+v", parseRows)
	}
	if _, hasParseFake := parseProviderSeen["fake"]; !hasParseFake {
		parseT.Fatalf("expected fake provider rollup row, got %+v", parseRows)
	}
}
