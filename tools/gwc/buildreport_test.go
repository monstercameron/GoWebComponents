package main

import "testing"

// actionGraphFixture mimics `go build -debug-actiongraph`: two rebuilt packages (NeedBuild
// with timings), one cached package, and non-build/synthetic actions that must be ignored.
const actionGraphFixture = `[
  {"Mode":"build","Package":"app/slow","NeedBuild":true,"TimeStart":"2026-06-27T18:00:00.000000000Z","TimeDone":"2026-06-27T18:00:00.300000000Z"},
  {"Mode":"build","Package":"app/fast","NeedBuild":true,"TimeStart":"2026-06-27T18:00:00.000000000Z","TimeDone":"2026-06-27T18:00:00.050000000Z"},
  {"Mode":"build","Package":"app/cached","NeedBuild":false,"TimeStart":"","TimeDone":""},
  {"Mode":"link","Package":"app/main","NeedBuild":true},
  {"Mode":"build","Package":"command-line-arguments","NeedBuild":true},
  {"Mode":""}
]`

// TestSummarizeActionGraphClassifiesAndRanks proves the report counts rebuilt vs cached
// packages (ignoring link/synthetic/std-meta actions) and ranks rebuilt ones by compile
// time — the honest "what rebuilt and why" picture from the cache-miss signal.
func TestSummarizeActionGraphClassifiesAndRanks(parseT *testing.T) {
	parseActions, parseErr := parseActionGraph([]byte(actionGraphFixture))
	if parseErr != nil {
		parseT.Fatalf("parseActionGraph: %v", parseErr)
	}
	parseReport := summarizeActionGraph(parseActions)

	if parseReport.TotalPackages != 3 {
		parseT.Fatalf("expected 3 real package builds (slow, fast, cached), got %d", parseReport.TotalPackages)
	}
	if parseReport.Rebuilt != 2 || parseReport.Cached != 1 {
		parseT.Fatalf("expected 2 rebuilt / 1 cached, got %d / %d", parseReport.Rebuilt, parseReport.Cached)
	}
	if len(parseReport.Slowest) != 2 || parseReport.Slowest[0].Package != "app/slow" {
		parseT.Fatalf("expected slow ranked first, got %+v", parseReport.Slowest)
	}
	if parseReport.Slowest[0].DurationMs < 290 || parseReport.Slowest[0].DurationMs > 310 {
		parseT.Fatalf("expected ~300ms for app/slow, got %.1f", parseReport.Slowest[0].DurationMs)
	}
}

// TestSummarizeActionGraphAllCached proves a fully-cached build reports nothing rebuilt.
func TestSummarizeActionGraphAllCached(parseT *testing.T) {
	parseActions, _ := parseActionGraph([]byte(`[
	  {"Mode":"build","Package":"a","NeedBuild":false},
	  {"Mode":"build","Package":"b","NeedBuild":false}
	]`))
	parseReport := summarizeActionGraph(parseActions)
	if parseReport.Rebuilt != 0 || parseReport.Cached != 2 {
		parseT.Fatalf("expected 0 rebuilt / 2 cached, got %d / %d", parseReport.Rebuilt, parseReport.Cached)
	}
	if len(parseReport.Slowest) != 0 {
		parseT.Fatalf("a fully-cached build should list no rebuilt packages, got %+v", parseReport.Slowest)
	}
}
