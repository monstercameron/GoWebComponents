package benchmarks

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const latestReportRel = "latest.json"
const driftNoteRel = "DRIFT_NOTE.md"

func TestLoadReportReadsCommittedBenchmarkReport(parseT *testing.T) {
	parseReport, parseErr := LoadReport(filepath.FromSlash(latestReportRel))
	if parseErr != nil {
		parseT.Fatalf("load latest report: %v", parseErr)
	}
	if !parseReport.OK {
		parseT.Fatal("latest benchmark report is not marked ok")
	}
	if parseReport.GeneratedAt == "" {
		parseT.Fatal("latest benchmark report has no generatedAt")
	}
	if parseReport.Comparison == nil {
		parseT.Fatal("latest benchmark report has no comparison summary")
	}
	if parseReport.Comparison.TolerancePct <= 0 {
		parseT.Fatalf("expected positive comparison tolerance, got %v", parseReport.Comparison.TolerancePct)
	}
}

func TestRenderDriftNoteSummarizesRegressionsAndImprovements(parseT *testing.T) {
	parseNote := RenderDriftNote(BenchmarkReport{
		OK:             true,
		GeneratedAt:    "2026-06-12T00:00:00Z",
		GoVersion:      "go1.26.0",
		GOOS:           "windows",
		GOARCH:         "amd64",
		SelectedLanes:  []string{"native"},
		ReportPath:     "./docs/benchmarks/latest.json",
		PackageCount:   1,
		BenchmarkCount: 2,
		Scores: &BenchmarkScoreSummary{
			Method:               "score method",
			ReferencePath:        "./docs/benchmarks/reference.json",
			ReferenceGeneratedAt: "2026-06-11T00:00:00Z",
			MatchedBenchmarks:    2,
			OverallScore:         101.25,
		},
		Comparison: &BenchmarkComparisonSummary{
			BaselineGeneratedAt: "2026-06-11T00:00:00Z",
			TolerancePct:        2,
			MatchedMetrics:      2,
			Improved:            1,
			Regressed:           1,
			Entries: []BenchmarkMetricComparison{
				{Lane: "native", Package: "./pkg", Benchmark: "BenchmarkSlow", Metric: "ns/op", Baseline: 10, Current: 12.5, Delta: 2.5, DeltaPct: 25, Direction: "regressed"},
				{Lane: "native", Package: "./pkg", Benchmark: "BenchmarkFast", Metric: "ns/op", Baseline: 10, Current: 8, Delta: -2, DeltaPct: -20, Direction: "improved"},
			},
		},
	})

	for _, parseWant := range []string{
		"drift beyond tolerance detected",
		"Top Regressions",
		"BenchmarkSlow",
		"Top Improvements",
		"BenchmarkFast",
		"101.2",
	} {
		if !strings.Contains(parseNote, parseWant) {
			parseT.Fatalf("expected drift note to contain %q:\n%s", parseWant, parseNote)
		}
	}
}

// TestBenchmarkDriftNoteIsGenerated is the CI drift guard: the committed note
// must byte-match the current benchmark report. Run with BENCHMARKS_WRITE=1 to
// regenerate after updating docs/benchmarks/latest.json.
func TestBenchmarkDriftNoteIsGenerated(parseT *testing.T) {
	parseReport, parseErr := LoadReport(filepath.FromSlash(latestReportRel))
	if parseErr != nil {
		parseT.Fatalf("load latest benchmark report: %v", parseErr)
	}
	parseExpected := RenderDriftNote(parseReport)
	parsePath := filepath.FromSlash(driftNoteRel)

	if os.Getenv("BENCHMARKS_WRITE") != "" {
		if parseWriteErr := os.WriteFile(parsePath, []byte(parseExpected), 0o644); parseWriteErr != nil {
			parseT.Fatalf("write benchmark drift note: %v", parseWriteErr)
		}
		parseT.Logf("regenerated %s", parsePath)
		return
	}

	parseActual, parseReadErr := os.ReadFile(parsePath)
	if parseReadErr != nil {
		parseT.Fatalf("read benchmark drift note (regenerate with BENCHMARKS_WRITE=1 go test ./docs/benchmarks/): %v", parseReadErr)
	}
	if strings.ReplaceAll(string(parseActual), "\r\n", "\n") != parseExpected {
		parseT.Fatalf("benchmark drift note is stale; regenerate with BENCHMARKS_WRITE=1 go test ./docs/benchmarks/")
	}
}
