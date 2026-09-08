package main

import "testing"

func TestNormalizeTestLanesDesktopIsExplicit(t *testing.T) {
	parseLanes, parseErr := normalizeTestLanes([]string{"desktop"})
	if parseErr != nil || len(parseLanes) != 1 || parseLanes[0] != "desktop" {
		t.Fatalf("desktop normalization = %#v, %v", parseLanes, parseErr)
	}
	parseAll, parseErr := normalizeTestLanes([]string{"all"})
	if parseErr != nil {
		t.Fatal(parseErr)
	}
	for _, parseLane := range parseAll {
		if parseLane == "desktop" {
			t.Fatal("desktop must remain opt-in and absent from all")
		}
	}
}
