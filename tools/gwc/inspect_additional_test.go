package main

import (
	"strings"
	"testing"
)

// TestInspectSummaryTextAndHelpers verifies text rendering and deterministic helper ordering.
func TestInspectSummaryTextAndHelpers(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("captureExamplesStdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	renderInspectSummary(inspectSummary{
		OK:   false,
		Root: "/workspace/app",
		Routes: inspectRouteReport{
			Registrations:     3,
			RouteFiles:        []string{"main.go", "routes.go"},
			HasLazySplit:      true,
			HasPrerenderHint:  true,
			HasMarketingRoute: false,
		},
		Dependencies: inspectDependencyView{
			ModulePath:   "example.com/app",
			TotalImports: 5,
		},
		Ownership: inspectOwnershipView{
			ImportBoundaryStatus: "warn",
			StateBoundaryStatus:  "ok",
		},
		FileTypes: inspectFileTypesView{
			TotalFiles:      7,
			ExtensionCounts: []inspectCountView{{Name: ".go", Count: 4}, {Name: ".html", Count: 3}},
		},
		Errors: []string{"routes: boom", "dependencies: unavailable"},
	})

	parseOutput, parseErr2 := parseStdout()
	if parseErr2 != nil {
		parseT.Fatalf("read captured stdout: %v", parseErr2)
	}
	for _, parseNeedle := range []string{"GWC inspect", "/workspace/app", "3 registrations across 2 files", "route lazy split:   true", "errors:", "routes: boom"} {
		if !strings.Contains(parseOutput, parseNeedle) {
			parseT.Fatalf("expected inspect summary to contain %q\n%s", parseNeedle, parseOutput)
		}
	}

	if parseGot := buildInspectTopDirectory(""); parseGot != "." {
		parseT.Fatalf("expected top directory fallback '.', got %q", parseGot)
	}
	if parseGot2 := buildInspectTopDirectory("web/index.html"); parseGot2 != "web" {
		parseT.Fatalf("expected top directory web, got %q", parseGot2)
	}

	parseViews := buildInspectCountViews(map[string]int{"css": 2, "go": 4, "html": 4})
	if len(parseViews) != 3 || parseViews[0].Name != "go" || parseViews[1].Name != "html" || parseViews[2].Name != "css" {
		parseT.Fatalf("unexpected inspect count ordering %#v", parseViews)
	}
}
