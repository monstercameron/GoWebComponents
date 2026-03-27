package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestRunBenchmarkCompareCoversPositionalAndFailureBranches verifies ambiguous positional inputs, missing benchstat, and benchstat failures.
func TestRunBenchmarkCompareCoversPositionalAndFailureBranches(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "baseline.txt")
	parseCandidatePath := filepath.Join(parseRoot, "candidate.txt")
	if parseErr := os.WriteFile(parseBaselinePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr != nil {
		parseT.Fatalf("write baseline: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate: %v", parseErr2)
	}

	parseT.Run("rejects ambiguous positional arguments", func(parseT2 *testing.T) {
		if parseErr := (launcher{}).runBenchmarkCompare([]string{"one", "two", "three"}); parseErr == nil || !strings.Contains(parseErr.Error(), "at most two positional arguments") {
			parseT2.Fatalf("expected too-many-positional error, got %v", parseErr)
		}
		if parseErr := (launcher{}).runBenchmarkCompare([]string{"-baseline", parseBaselinePath, "-candidate", parseCandidatePath, parseBaselinePath}); parseErr == nil || !strings.Contains(parseErr.Error(), "positional arguments but -baseline and -candidate are already set") {
			parseT2.Fatalf("expected conflicting positional/flag error, got %v", parseErr)
		}
		if parseErr := (launcher{}).runBenchmarkCompare([]string{"-baseline", parseBaselinePath, parseBaselinePath, parseCandidatePath}); parseErr == nil || !strings.Contains(parseErr.Error(), "positional shortcuts require either no flags or exactly one missing path") {
			parseT2.Fatalf("expected positional shortcut error, got %v", parseErr)
		}
	})

	parseT.Run("reports missing benchstat in json mode", func(parseT2 *testing.T) {
		parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
		if parseErr != nil {
			parseT2.Fatalf("capture stdout: %v", parseErr)
		}
		defer parseRestoreStdout()

		parseOriginalLookPath := benchmarkLookPath
		parseOriginalRunCommand := benchmarkRunCommand
		parseT2.Cleanup(func() {
			benchmarkLookPath = parseOriginalLookPath
			benchmarkRunCommand = parseOriginalRunCommand
		})
		benchmarkLookPath = func(parseFile string) (string, error) {
			return "", errors.New("not installed")
		}
		benchmarkRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			parseT2.Fatalf("did not expect benchstat to run when missing")
			return "", nil
		}

		if parseErr2 := (launcher{}).runBenchmarkCompare([]string{"-baseline", parseBaselinePath, "-candidate", parseCandidatePath, "-json"}); parseErr2 != nil {
			parseT2.Fatalf("runBenchmarkCompare missing benchstat: %v", parseErr2)
		}

		parseOutput, parseErr := parseStdout()
		if parseErr != nil {
			parseT2.Fatalf("read stdout: %v", parseErr)
		}
		var parseSummary benchmarkCompareSummary
		if parseErr3 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr3 != nil {
			parseT2.Fatalf("decode compare summary: %v\n%s", parseErr3, parseOutput)
		}
		if parseSummary.BenchstatAvailable || !strings.Contains(parseSummary.Message, "benchstat is not installed") {
			parseT2.Fatalf("unexpected missing-benchstat summary %#v", parseSummary)
		}
	})

	parseT.Run("returns benchstat command failure after emitting json summary", func(parseT2 *testing.T) {
		parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
		if parseErr != nil {
			parseT2.Fatalf("capture stdout: %v", parseErr)
		}
		defer parseRestoreStdout()

		parseOriginalLookPath := benchmarkLookPath
		parseOriginalRunCommand := benchmarkRunCommand
		parseT2.Cleanup(func() {
			benchmarkLookPath = parseOriginalLookPath
			benchmarkRunCommand = parseOriginalRunCommand
		})
		benchmarkLookPath = func(parseFile string) (string, error) {
			return "/usr/bin/benchstat", nil
		}
		benchmarkRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return "partial output", errors.New("benchstat failed")
		}

		parseErr2 := (launcher{}).runBenchmarkCompare([]string{"-baseline", parseBaselinePath, "-candidate", parseCandidatePath, "-json"})
		if parseErr2 == nil || !strings.Contains(parseErr2.Error(), "benchstat failed") {
			parseT2.Fatalf("expected benchstat failure, got %v", parseErr2)
		}

		parseOutput, parseErr := parseStdout()
		if parseErr != nil {
			parseT2.Fatalf("read stdout: %v", parseErr)
		}
		var parseSummary benchmarkCompareSummary
		if parseErr3 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr3 != nil {
			parseT2.Fatalf("decode compare summary: %v\n%s", parseErr3, parseOutput)
		}
		if parseSummary.OK || !parseSummary.BenchstatAvailable || parseSummary.Output != "partial output" {
			parseT2.Fatalf("unexpected failed compare summary %#v", parseSummary)
		}
	})
}

// TestRunBenchmarkCapturePropagatesRunFailureAfterWritingOutputs verifies failed capture runs still emit summaries and write raw output.
func TestRunBenchmarkCapturePropagatesRunFailureAfterWritingOutputs(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutputPath := filepath.Join(parseRoot, "bench-output.txt")

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
	})
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		return "BenchmarkRuntime-8 1 100 ns/op", errors.New("go test failed")
	}

	parseErr = (launcher{}).runBenchmarkCapture([]string{"-package", "./internal/runtime", "-count", "2", "-output", parseOutputPath, "-json"})
	if parseErr == nil || !strings.Contains(parseErr.Error(), "go test failed") {
		parseT.Fatalf("expected benchmark capture failure, got %v", parseErr)
	}

	parseRawOutput, parseErr := os.ReadFile(parseOutputPath)
	if parseErr != nil {
		parseT.Fatalf("read benchmark capture output: %v", parseErr)
	}
	if !strings.Contains(string(parseRawOutput), "BenchmarkRuntime-8") {
		parseT.Fatalf("expected raw benchmark output to be written, got %q", string(parseRawOutput))
	}

	parseJSONOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	var parseSummary benchmarkCaptureSummary
	if parseErr2 := json.Unmarshal([]byte(parseJSONOutput), &parseSummary); parseErr2 != nil {
		parseT.Fatalf("decode capture summary: %v\n%s", parseErr2, parseJSONOutput)
	}
	if parseSummary.OK || parseSummary.OutputPath != filepath.ToSlash(parseOutputPath) {
		parseT.Fatalf("unexpected capture failure summary %#v", parseSummary)
	}
}
