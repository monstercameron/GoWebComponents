package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunWasmComparePrintsSummaryAndWritesOutFile verifies the non-JSON compare path and summary artifact output.
func TestRunWasmComparePrintsSummaryAndWritesOutFile(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "baseline.json")
	parseCandidatePath := filepath.Join(parseRoot, "candidate.json")
	parseOutPath := filepath.Join(parseRoot, "reports", "compare.json")
	if parseErr := os.WriteFile(parseBaselinePath, []byte(`{"phases":{"go_build_ms":100,"steady_ms":5}}`), 0644); parseErr != nil {
		parseT.Fatalf("write baseline: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte(`{"phases":{"go_build_ms":90,"steady_ms":5}}`), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	if parseErr4 := (launcher{}).runWasmCompare([]string{"-baseline", parseBaselinePath, "-candidate", parseCandidatePath, "-out-file", parseOutPath}); parseErr4 != nil {
		parseT.Fatalf("runWasmCompare: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	for _, parseWant := range []string{"GWC wasm compare", "improved:    1", "thresholded: 0"} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Fatalf("expected compare output to contain %q\n%s", parseWant, parseOutput)
		}
	}

	parseSummaryBytes, parseErr5 := os.ReadFile(parseOutPath)
	if parseErr5 != nil {
		parseT.Fatalf("read compare summary: %v", parseErr5)
	}
	var parseSummary wasmCompareSummary
	if parseErr6 := json.Unmarshal(parseSummaryBytes, &parseSummary); parseErr6 != nil {
		parseT.Fatalf("decode compare summary: %v", parseErr6)
	}
	if !parseSummary.OK || parseSummary.Counts.Improved != 1 || parseSummary.Counts.Unchanged != 1 {
		parseT.Fatalf("unexpected compare summary %#v", parseSummary)
	}
}

// TestExecuteWasmCompareCoversStatusesAndFailures verifies added, removed, improved, thresholded, and failure branches.
func TestExecuteWasmCompareCoversStatusesAndFailures(parseT *testing.T) {
	parseT.Run("classifies mixed metric statuses and writes summary", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseBaselinePath := filepath.Join(parseRoot, "baseline.json")
		parseCandidatePath := filepath.Join(parseRoot, "candidate.json")
		parseOutPath := filepath.Join(parseRoot, "compare", "summary.json")
		if parseErr := os.WriteFile(parseBaselinePath, []byte(`{
  "phases": {
    "improved_ms": 100,
    "unchanged_ms": 5,
    "removed_ms": 9,
    "zero_growth_ms": 0
  }
}`), 0644); parseErr != nil {
			parseT2.Fatalf("write baseline: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(parseCandidatePath, []byte(`{
  "phases": {
    "improved_ms": 80,
    "unchanged_ms": 5,
    "added_ms": 7,
    "zero_growth_ms": 3
  }
}`), 0644); parseErr2 != nil {
			parseT2.Fatalf("write candidate: %v", parseErr2)
		}

		parseSummary, parseRegressionCount, parseErr3 := executeWasmCompare(wasmCompareConfig{
			baselinePath:            parseBaselinePath,
			candidatePath:           parseCandidatePath,
			outFile:                 parseOutPath,
			timingRegressionPercent: 10,
			sizeRegressionPercent:   0,
			otherRegressionPercent:  0,
		})
		if parseErr3 != nil {
			parseT2.Fatalf("executeWasmCompare: %v", parseErr3)
		}
		if parseRegressionCount != 0 || !parseSummary.OK {
			parseT2.Fatalf("expected no regressions, got summary=%#v count=%d", parseSummary, parseRegressionCount)
		}
		if parseSummary.Counts.Added != 1 || parseSummary.Counts.Removed != 1 || parseSummary.Counts.Improved != 1 || parseSummary.Counts.Unchanged != 1 || parseSummary.Counts.WithinThreshold != 1 {
			parseT2.Fatalf("unexpected compare counts %#v", parseSummary.Counts)
		}

		parseStatusByPath := map[string]string{}
		for _, parseMetric := range parseSummary.Metrics {
			parseStatusByPath[parseMetric.Path] = parseMetric.Status
		}
		parseWantStatuses := map[string]string{
			"phases.added_ms":       "added",
			"phases.improved_ms":    "improved",
			"phases.removed_ms":     "removed",
			"phases.unchanged_ms":   "unchanged",
			"phases.zero_growth_ms": "within-threshold",
		}
		for parsePath, parseWant := range parseWantStatuses {
			if parseStatusByPath[parsePath] != parseWant {
				parseT2.Fatalf("expected %s status %q, got %#v", parsePath, parseWant, parseSummary.Metrics)
			}
		}
		if _, parseErr4 := os.Stat(parseOutPath); parseErr4 != nil {
			parseT2.Fatalf("expected compare output file: %v", parseErr4)
		}
	})

	parseT.Run("reports baseline read and candidate parse failures", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseCandidatePath := filepath.Join(parseRoot, "candidate.json")
		if parseErr := os.WriteFile(parseCandidatePath, []byte(`{invalid`), 0644); parseErr != nil {
			parseT2.Fatalf("write invalid candidate: %v", parseErr)
		}

		_, _, parseErr := executeWasmCompare(wasmCompareConfig{
			baselinePath:  filepath.Join(parseRoot, "missing.json"),
			candidatePath: parseCandidatePath,
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "read baseline manifest") {
			parseT2.Fatalf("expected baseline read failure, got %v", parseErr)
		}

		parseBaselinePath := filepath.Join(parseRoot, "baseline.json")
		if parseErr2 := os.WriteFile(parseBaselinePath, []byte(`{}`), 0644); parseErr2 != nil {
			parseT2.Fatalf("write baseline: %v", parseErr2)
		}
		_, _, parseErr = executeWasmCompare(wasmCompareConfig{
			baselinePath:  parseBaselinePath,
			candidatePath: parseCandidatePath,
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "parse candidate manifest") {
			parseT2.Fatalf("expected candidate parse failure, got %v", parseErr)
		}
	})

	parseT.Run("reports compare output directory and write failures", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseBaselinePath := filepath.Join(parseRoot, "baseline.json")
		parseCandidatePath := filepath.Join(parseRoot, "candidate.json")
		if parseErr := os.WriteFile(parseBaselinePath, []byte(`{"phases":{"go_build_ms":1}}`), 0644); parseErr != nil {
			parseT2.Fatalf("write baseline: %v", parseErr)
		}
		if parseErr2 := os.WriteFile(parseCandidatePath, []byte(`{"phases":{"go_build_ms":1}}`), 0644); parseErr2 != nil {
			parseT2.Fatalf("write candidate: %v", parseErr2)
		}

		parseBlockedPath := filepath.Join(parseRoot, "blocked")
		if parseErr3 := os.WriteFile(parseBlockedPath, []byte("not-a-dir"), 0644); parseErr3 != nil {
			parseT2.Fatalf("write blocked path: %v", parseErr3)
		}
		_, _, parseErr := executeWasmCompare(wasmCompareConfig{
			baselinePath:  parseBaselinePath,
			candidatePath: parseCandidatePath,
			outFile:       filepath.Join(parseBlockedPath, "summary.json"),
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "create compare output dir") {
			parseT2.Fatalf("expected compare output dir failure, got %v", parseErr)
		}

		parseWriteBlockedDir := filepath.Join(parseRoot, "write-blocked")
		if parseErr4 := os.MkdirAll(parseWriteBlockedDir, 0755); parseErr4 != nil {
			parseT2.Fatalf("mkdir write-blocked dir: %v", parseErr4)
		}
		_, _, parseErr = executeWasmCompare(wasmCompareConfig{
			baselinePath:  parseBaselinePath,
			candidatePath: parseCandidatePath,
			outFile:       parseWriteBlockedDir,
		})
		if parseErr == nil || !strings.Contains(parseErr.Error(), "write compare summary") {
			parseT2.Fatalf("expected compare summary write failure, got %v", parseErr)
		}
	})
}
