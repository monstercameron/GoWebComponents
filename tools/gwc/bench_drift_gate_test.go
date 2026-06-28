package main

import "testing"

// TestBenchmarkRegressionGate proves the CI drift gate's policy: it fails only when enabled AND a
// baseline comparison reports a regression beyond tolerance; otherwise it is a no-op so ordinary
// local runs (and runs with no baseline) are never broken.
func TestBenchmarkRegressionGate(parseT *testing.T) {
	parseRegressed := &benchmarkComparisonSummary{Regressed: 2, TolerancePct: 2.0}
	parseClean := &benchmarkComparisonSummary{Regressed: 0, TolerancePct: 2.0}

	if parseErr := benchmarkRegressionGate(parseRegressed, true); parseErr == nil {
		parseT.Fatal("gate enabled with regressions must fail")
	}
	if parseErr := benchmarkRegressionGate(parseRegressed, false); parseErr != nil {
		parseT.Fatalf("gate disabled must never fail, got %v", parseErr)
	}
	if parseErr := benchmarkRegressionGate(parseClean, true); parseErr != nil {
		parseT.Fatalf("gate enabled with no regressions must pass, got %v", parseErr)
	}
	if parseErr := benchmarkRegressionGate(nil, true); parseErr != nil {
		parseT.Fatalf("gate enabled with no baseline comparison must pass, got %v", parseErr)
	}
}
