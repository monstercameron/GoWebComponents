package main

import (
	"math"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestBenchmarkDisplayAndBucketHelpersCoverBranches verifies benchmark helper switch and path branches.
func TestBenchmarkDisplayAndBucketHelpersCoverBranches(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseSiblingRoot := parseT.TempDir()
	parseNestedPath := filepath.Join(parseRoot, "reports", "latest.json")

	if parseGot := benchmarkDisplayPath("", parseNestedPath); parseGot != filepath.ToSlash(parseNestedPath) {
		parseT.Fatalf("benchmarkDisplayPath empty root = %q, want %q", parseGot, filepath.ToSlash(parseNestedPath))
	}
	if parseGot := benchmarkDisplayPath(parseRoot, ""); parseGot != "" {
		parseT.Fatalf("benchmarkDisplayPath empty path = %q, want empty", parseGot)
	}
	if parseGot := benchmarkDisplayPath(parseRoot, parseRoot); parseGot != "." {
		parseT.Fatalf("benchmarkDisplayPath root path = %q, want .", parseGot)
	}
	if parseGot := benchmarkDisplayPath(parseRoot, parseNestedPath); parseGot != "./reports/latest.json" {
		parseT.Fatalf("benchmarkDisplayPath nested path = %q", parseGot)
	}
	parseOutsidePath := filepath.Join(parseSiblingRoot, "outside.json")
	if parseGot := benchmarkDisplayPath(parseRoot, parseOutsidePath); !strings.HasPrefix(parseGot, "../") {
		parseT.Fatalf("benchmarkDisplayPath outside root = %q, want ../ prefix", parseGot)
	}

	parseBucketCases := []struct {
		parseBucketID string
		parseLabel    string
		parseOrder    int
	}{
		{parseBucketID: "compute", parseLabel: "Compute", parseOrder: 0},
		{parseBucketID: "memory", parseLabel: "Memory", parseOrder: 1},
		{parseBucketID: "alloc_runtime", parseLabel: "Alloc/Runtime", parseOrder: 2},
		{parseBucketID: "sync_concurrency", parseLabel: "Sync/Concurrency", parseOrder: 3},
		{parseBucketID: "end_to_end", parseLabel: "End-to-End", parseOrder: 4},
		{parseBucketID: "custom", parseLabel: "custom", parseOrder: 100},
	}
	for _, parseCase := range parseBucketCases {
		if parseLabel := benchmarkBucketLabel(parseCase.parseBucketID); parseLabel != parseCase.parseLabel {
			parseT.Fatalf("benchmarkBucketLabel(%q) = %q, want %q", parseCase.parseBucketID, parseLabel, parseCase.parseLabel)
		}
		if parseOrder := benchmarkBucketOrder(parseCase.parseBucketID); parseOrder != parseCase.parseOrder {
			parseT.Fatalf("benchmarkBucketOrder(%q) = %d, want %d", parseCase.parseBucketID, parseOrder, parseCase.parseOrder)
		}
	}

	if !benchmarkMetricComparable("ns/op") || !benchmarkMetricComparable("B/op") || !benchmarkMetricComparable("allocs/op") {
		parseT.Fatal("expected benchmarkMetricComparable to accept benchmark metrics")
	}
	if benchmarkMetricComparable("MB/s") {
		parseT.Fatal("expected benchmarkMetricComparable to reject unsupported metrics")
	}

	if parseToken := benchmarkPackageToken(" ./internal/runtime/jsdom "); parseToken != "._internal_runtime_jsdom" {
		parseT.Fatalf("benchmarkPackageToken normalized = %q", parseToken)
	}
	if parseToken := benchmarkPackageToken("///"); parseToken != "package" {
		parseT.Fatalf("benchmarkPackageToken slash-only = %q, want package", parseToken)
	}
}

// TestBenchmarkComparisonAndScoringHelpersCoverEdgeCases verifies nil, unchanged, and graph/clamp branches.
func TestBenchmarkComparisonAndScoringHelpersCoverEdgeCases(parseT *testing.T) {
	if parseSummary := compareBenchmarkReports(benchmarkReport{}, benchmarkReport{}); parseSummary != nil {
		parseT.Fatalf("expected nil comparison for missing baseline timestamp, got %#v", parseSummary)
	}

	parseBaseline := benchmarkReport{
		GeneratedAt: "2026-03-26T00:00:00Z",
		Packages: []benchmarkPackageReport{{
			Lane:    "native",
			Package: "./ui",
			OK:      true,
			Benchmarks: []benchmarkResultReport{{
				Name:           "BenchmarkRenderMicro-8",
				AverageMetrics: map[string]float64{"ns/op": 100},
			}},
		}},
	}
	parseCurrent := benchmarkReport{
		Packages: []benchmarkPackageReport{{
			Lane:    "native",
			Package: "./ui",
			OK:      true,
			Benchmarks: []benchmarkResultReport{{
				Name:           "BenchmarkRenderMicro-8",
				AverageMetrics: map[string]float64{"ns/op": 101},
			}},
		}},
	}
	parseComparison := compareBenchmarkReports(parseBaseline, parseCurrent)
	if parseComparison == nil {
		parseT.Fatal("expected comparison summary for overlapping metrics")
	}
	if parseComparison.Unchanged != 1 || parseComparison.Improved != 0 || parseComparison.Regressed != 0 {
		parseT.Fatalf("expected unchanged comparison within tolerance, got %#v", parseComparison)
	}

	parseUnmatchedCurrent := benchmarkReport{
		Packages: []benchmarkPackageReport{{
			Lane:    "wasm",
			Package: "./other",
			OK:      true,
			Benchmarks: []benchmarkResultReport{{
				Name:           "BenchmarkOther-8",
				AverageMetrics: map[string]float64{"ns/op": 50},
			}},
		}},
	}
	if parseSummary := compareBenchmarkReports(parseBaseline, parseUnmatchedCurrent); parseSummary != nil {
		parseT.Fatalf("expected nil comparison for unmatched metrics, got %#v", parseSummary)
	}

	if parseSummary := buildBenchmarkScoreSummary(parseBaseline, parseUnmatchedCurrent, benchmarkConfig{rootPath: parseT.TempDir(), referencePath: "reference.json"}); parseSummary != nil {
		parseT.Fatalf("expected nil score summary for unmatched benchmarks, got %#v", parseSummary)
	}

	parseReference := benchmarkReport{
		GeneratedAt: "2026-03-25T00:00:00Z",
		Packages: []benchmarkPackageReport{{
			Lane:    "native",
			Package: "./internal/runtime",
			OK:      true,
			Benchmarks: []benchmarkResultReport{{
				Name:           "BenchmarkRuntimeQueue-8",
				AverageMetrics: map[string]float64{"ns/op": 120},
			}},
		}},
	}
	parseScoredCurrent := benchmarkReport{
		Packages: []benchmarkPackageReport{{
			Lane:    "native",
			Package: "./internal/runtime",
			OK:      true,
			Benchmarks: []benchmarkResultReport{{
				Name:           "BenchmarkRuntimeQueue-8",
				AverageMetrics: map[string]float64{"ns/op": 60},
			}},
		}},
	}
	parseScoreSummary := buildBenchmarkScoreSummary(parseReference, parseScoredCurrent, benchmarkConfig{
		rootPath:      parseT.TempDir(),
		referencePath: filepath.Join(parseT.TempDir(), "reference.json"),
	})
	if parseScoreSummary == nil || len(parseScoreSummary.Buckets) != 1 {
		parseT.Fatalf("expected score summary with one fallback bucket, got %#v", parseScoreSummary)
	}
	if parseScoreSummary.Buckets[0].ID != "sync_concurrency" {
		parseT.Fatalf("expected fallback benchmark bucket classification, got %#v", parseScoreSummary.Buckets)
	}

	if parseMean := geometricMean(nil); parseMean != 0 {
		parseT.Fatalf("geometricMean(nil) = %v, want 0", parseMean)
	}
	if parseMean := geometricMean([]float64{-1, 0}); parseMean != 0 {
		parseT.Fatalf("geometricMean(non-positive only) = %v, want 0", parseMean)
	}
	if parseMean := geometricMean([]float64{4, -2, 16}); math.Abs(parseMean-8) > 0.0000001 {
		parseT.Fatalf("geometricMean(filtered positives) = %v, want 8", parseMean)
	}

	if parseGraph := benchmarkScoreGraph(100, 0); parseGraph != "" {
		parseT.Fatalf("benchmarkScoreGraph zero width = %q, want empty", parseGraph)
	}
	if parseGraph := benchmarkScoreGraph(-10, 10); parseGraph != "[----------]" {
		parseT.Fatalf("benchmarkScoreGraph negative clamp = %q", parseGraph)
	}
	if parseGraph := benchmarkScoreGraph(250, 10); parseGraph != "[##########]" {
		parseT.Fatalf("benchmarkScoreGraph high clamp = %q", parseGraph)
	}
}

// TestBenchmarkCaptureConfigAndReportHelpersCoverErrors verifies output-path normalization and report write branches.
func TestBenchmarkCaptureConfigAndReportHelpersCoverErrors(parseT *testing.T) {
	parseOriginalWD, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("Getwd: %v", parseErr)
	}
	parseRoot := parseT.TempDir()
	if parseErr2 := os.Chdir(parseRoot); parseErr2 != nil {
		parseT.Fatalf("Chdir temp root: %v", parseErr2)
	}
	defer func() {
		if parseErr3 := os.Chdir(parseOriginalWD); parseErr3 != nil {
			parseT.Fatalf("restore cwd: %v", parseErr3)
		}
	}()

	parseLauncher := launcher{repoRoot: parseRoot}
	parseCaptureConfig, parseErr := parseLauncher.resolveBenchmarkCaptureConfig(benchmarkCaptureConfig{
		packagePath: "./internal/runtime",
		count:       2,
		bench:       "",
		outputPath:  filepath.Join("reports", "capture.txt"),
	})
	if parseErr != nil {
		parseT.Fatalf("resolveBenchmarkCaptureConfig relative output: %v", parseErr)
	}
	if parseCaptureConfig.bench != "." {
		parseT.Fatalf("expected blank bench to normalize to '.', got %q", parseCaptureConfig.bench)
	}
	if !filepath.IsAbs(parseCaptureConfig.outputPath) || !strings.HasSuffix(filepath.ToSlash(parseCaptureConfig.outputPath), "/reports/capture.txt") {
		parseT.Fatalf("unexpected capture output path: %q", parseCaptureConfig.outputPath)
	}

	parseBlockingFile := filepath.Join(parseRoot, "blocked")
	if parseErr2 := os.WriteFile(parseBlockingFile, []byte("file"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile blocking path: %v", parseErr2)
	}
	if _, parseErr3 := parseLauncher.resolveBenchmarkCaptureConfig(benchmarkCaptureConfig{
		packagePath: "./internal/runtime",
		count:       1,
		bench:       ".",
		outputPath:  filepath.Join(parseBlockingFile, "capture.txt"),
	}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "create benchmark capture output directory") {
		parseT.Fatalf("expected capture directory creation error, got %v", parseErr3)
	}

	parseReportPath := filepath.Join(parseRoot, "docs", "benchmarks", "latest.json")
	parseReport := benchmarkReport{
		OK:            true,
		Root:          parseRoot,
		GeneratedAt:   "2026-03-26T00:00:00Z",
		SelectedLanes: []string{"native"},
		Packages: []benchmarkPackageReport{{
			Lane:    "native",
			Package: "./ui",
			OK:      true,
		}},
	}
	if parseErr4 := writeBenchmarkReport(parseReportPath, parseReport); parseErr4 != nil {
		parseT.Fatalf("writeBenchmarkReport success: %v", parseErr4)
	}
	parseLoadedReport, parseErr := loadBenchmarkReport(parseReportPath)
	if parseErr != nil {
		parseT.Fatalf("loadBenchmarkReport written file: %v", parseErr)
	}
	if parseLoadedReport.GeneratedAt != parseReport.GeneratedAt || parseLoadedReport.Root != parseReport.Root {
		parseT.Fatalf("unexpected loaded benchmark report: %#v", parseLoadedReport)
	}

	parseReportBlockingFile := filepath.Join(parseRoot, "report-parent")
	if parseErr5 := os.WriteFile(parseReportBlockingFile, []byte("file"), 0o644); parseErr5 != nil {
		parseT.Fatalf("WriteFile report blocking path: %v", parseErr5)
	}
	if parseErr6 := writeBenchmarkReport(filepath.Join(parseReportBlockingFile, "latest.json"), parseReport); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "create benchmark report directory") {
		parseT.Fatalf("expected report directory creation error, got %v", parseErr6)
	}
}
