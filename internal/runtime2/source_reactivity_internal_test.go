package runtime2

import "testing"

// TestSourceReactivityRejectsInvalidInputsAndNilReceivers verifies source-reactivity helpers reject invalid inputs and tolerate nil receivers as no-ops.
func TestSourceReactivityRejectsInvalidInputsAndNilReceivers(parseT *testing.T) {
	var parseNilSourceReactivity *SourceReactivity
	if parseErr := parseNilSourceReactivity.SetRegionDeclaredSources("region-1", []string{"status"}); parseErr != nil {
		parseT.Fatalf("expected nil source reactivity set to be ignored, got %v", parseErr)
	}
	if parseNilSourceReactivity.HandleSourceChange("status") {
		parseT.Fatal("expected nil source reactivity source change to be ignored")
	}
	if parseQueue := parseNilSourceReactivity.GetRegionUpdateQueue(); parseQueue != nil {
		parseT.Fatalf("expected nil source reactivity queue to be nil, got %+v", parseQueue)
	}

	parseSourceReactivity := BuildSourceReactivity()
	if parseErr := parseSourceReactivity.SetRegionDeclaredSources("", []string{"status"}); parseErr == nil {
		parseT.Fatal("expected empty region ID to fail")
	}
	if parseErr := parseSourceReactivity.SetRegionDeclaredSources("region-1", []string{" status "}); parseErr == nil {
		parseT.Fatal("expected surrounding-whitespace source ID to fail")
	}
	if parseSourceReactivity.HandleSourceChange(" status ") {
		parseT.Fatal("expected surrounding-whitespace source change to be ignored")
	}
	if parseSourceReactivity.HandleSourceChange("") {
		parseT.Fatal("expected empty source change to be ignored")
	}
}

// TestSourceReactivityRebindsClearsAndQueuesInSortedOrder verifies exact-match rebinds are no-ops, source clearing prunes reverse maps, and queued region updates are returned in sorted order.
func TestSourceReactivityRebindsClearsAndQueuesInSortedOrder(parseT *testing.T) {
	parseSourceReactivity := BuildSourceReactivity()
	if parseErr := parseSourceReactivity.SetRegionDeclaredSources("region-2", []string{"status"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources(region-2) returned error: %v", parseErr)
	}
	if parseErr := parseSourceReactivity.SetRegionDeclaredSources("region-1", []string{"count", "status"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources(region-1) returned error: %v", parseErr)
	}
	if parseErr := parseSourceReactivity.SetRegionDeclaredSources("region-1", []string{"status", "count"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources(region-1 exact rebind) returned error: %v", parseErr)
	}
	if parseRegions, hasCount := parseSourceReactivity.storeRegionIDsBySourceID["count"]; !hasCount || len(parseRegions) != 1 {
		parseT.Fatalf("expected count reverse binding to remain after exact rebind, got %+v", parseRegions)
	}
	if parseErr := parseSourceReactivity.SetRegionDeclaredSources("region-1", []string{"status"}); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources(region-1 prune count) returned error: %v", parseErr)
	}
	if _, hasCount := parseSourceReactivity.storeRegionIDsBySourceID["count"]; hasCount {
		parseT.Fatal("expected count reverse binding to be removed after region-1 dropped count")
	}
	if parseRegions, hasStatus := parseSourceReactivity.storeRegionIDsBySourceID["status"]; !hasStatus || len(parseRegions) != 2 {
		parseT.Fatalf("expected status reverse binding to include both regions, got %+v", parseRegions)
	}
	if parseErr := parseSourceReactivity.SetRegionDeclaredSources("region-1", nil); parseErr != nil {
		parseT.Fatalf("SetRegionDeclaredSources(region-1 clear) returned error: %v", parseErr)
	}
	if _, hasRegion := parseSourceReactivity.storeSourceIDsByRegionID["region-1"]; hasRegion {
		parseT.Fatal("expected region-1 declared sources to be removed after clear")
	}
	if parseRegions, hasStatus := parseSourceReactivity.storeRegionIDsBySourceID["status"]; !hasStatus || len(parseRegions) != 1 {
		parseT.Fatalf("expected status reverse binding to retain region-2 only, got %+v", parseRegions)
	}

	parseSourceReactivity.storeSourceReactivityQueueRegionUpdate("region-b")
	parseSourceReactivity.storeSourceReactivityQueueRegionUpdate("region-a")
	parseSourceReactivity.storeSourceReactivityQueueRegionUpdate("region-a")
	parseQueuedRegions := parseSourceReactivity.GetRegionUpdateQueue()
	if len(parseQueuedRegions) != 2 {
		parseT.Fatalf("expected two queued regions, got %+v", parseQueuedRegions)
	}
	if parseQueuedRegions[0] != "region-a" || parseQueuedRegions[1] != "region-b" {
		parseT.Fatalf("expected sorted queued regions [region-a region-b], got %+v", parseQueuedRegions)
	}
	if parseQueue := parseSourceReactivity.GetRegionUpdateQueue(); parseQueue != nil {
		parseT.Fatalf("expected drained queue to return nil, got %+v", parseQueue)
	}
}
