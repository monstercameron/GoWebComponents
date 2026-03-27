package main

import (
	"errors"
	"strings"
	"testing"
)

// TestBenchmarkPrinterIncludesScoresAndFailures verifies the human-readable benchmark report printer.
func TestBenchmarkPrinterIncludesScoresAndFailures(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	printBenchmarkReport(benchmarkReport{
		Root:               ".",
		ReportPath:         "./docs/benchmarks/latest.json",
		GoVersion:          "go1.26.0",
		SelectedLanes:      []string{"native", "wasm"},
		PackageParallelism: 2,
		PackageCount:       2,
		BenchmarkCount:     3,
		FailedPackages:     1,
		Scores: &benchmarkScoreSummary{
			ReferencePath:        "./docs/benchmarks/reference.json",
			ReferenceGeneratedAt: "2026-03-26T00:00:00Z",
			ReferenceMachine:     "builder-1",
			OverallScore:         111.5,
			Buckets: []benchmarkBucketScore{{
				Label:             "Compute",
				Score:             120,
				MatchedBenchmarks: 2,
			}},
		},
		Comparison: &benchmarkComparisonSummary{
			BaselineGeneratedAt: "2026-03-25T00:00:00Z",
			Improved:            1,
			Regressed:           1,
			Unchanged:           1,
			TolerancePct:        2.0,
		},
		Packages: []benchmarkPackageReport{
			{Lane: "native", Package: "./ui", OK: true, BenchmarkCount: 2},
			{Lane: "wasm", Package: "./fetch", OK: false, BenchmarkCount: 1, Error: "bench failed"},
		},
	})

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	for _, parseWant := range []string{
		"GWC bench",
		"reference:     ./docs/benchmarks/reference.json (2026-03-26T00:00:00Z, builder-1)",
		"Compute Score:",
		"Overall Score:",
		"comparison:    1 improved, 1 regressed, 1 unchanged",
		"[fail] wasm ./fetch",
		"error: bench failed",
	} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Fatalf("expected benchmark output to contain %q, got %q", parseWant, parseOutput)
		}
	}
}

// TestWasmSummaryPrintersEmitReadableOutput verifies the wasm summary printers.
func TestWasmSummaryPrintersEmitReadableOutput(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	printWasmMeasureSummary(wasmMeasureSummary{
		OutDir:       "./out",
		ManifestPath: "./out/manifest.json",
		Manifest: wasmMeasureManifest{
			Package:    "./ui",
			Profile:    "release",
			GoVersion:  "go1.26.0",
			Phases:     map[string]int64{"go_build_ms": 123},
			Artifacts:  map[string]releaseArtifactRecord{"wasm": {Path: "app.wasm", Bytes: 42}},
			GoExecutable: "go",
		},
	})
	printWasmCompareSummary(wasmCompareSummary{
		Baseline:  "baseline.json",
		Candidate: "candidate.json",
		Counts: wasmCompareCounts{
			Total:           3,
			Improved:        1,
			Regressed:       1,
			Unchanged:       1,
			WithinThreshold: 0,
		},
	})
	printWasmCacheSummary(wasmCacheSummary{
		Package:     "./ui",
		GeneratedAt: "2026-03-26T00:00:00Z",
		Environment: wasmCacheEnvironment{GoVersion: "go1.26.0", DefaultGoCache: "C:/gocache", DefaultGoMod: "C:/gomodcache"},
		SummaryPath: "./cache-summary.json",
		Variants: map[string]wasmCacheVariantResult{
			"shared-cache-cold": {Status: "ok"},
			"ci-style-cold":     {Status: "error", Error: "cache miss"},
		},
	})
	printWasmToolchainSummary(wasmToolchainSummary{
		Package:            "./ui",
		ComparedAt:         "2026-03-26T00:00:00Z",
		Baseline:           wasmToolchainParty{GoExecutable: "go1.25"},
		Candidate:          wasmToolchainParty{GoExecutable: "go1.26"},
		Comparison:         "./toolchain-comparison.json",
		RegressionExitCode: 1,
		SummaryPath:        "./toolchain-summary.json",
	})
	printWasmCompressionSummary(wasmCompressionSummary{
		Package:     "./ui",
		GeneratedAt: "2026-03-26T00:00:00Z",
		Environment: wasmCompressionEnvironment{
			GoVersion:        "go1.26.0",
			BrotliSupported:  true,
			WasmOptAvailable: true,
			WasmOptPath:      "wasm-opt@1.0",
		},
		SummaryPath: "./compression-summary.json",
		Variants: map[string]interface{}{
			"plain_raw":            map[string]any{},
			"optimized_compressed": map[string]any{},
		},
	})

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	for _, parseWant := range []string{
		"GWC wasm measure",
		"go build ms:  123",
		"GWC wasm compare",
		"GWC wasm compare-cache",
		"error: cache miss",
		"GWC wasm compare-toolchain",
		"regressions:  1",
		"GWC wasm compare-compression",
		"wasm-opt id: wasm-opt@1.0",
		"variant:     optimized_compressed",
	} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Fatalf("expected wasm output to contain %q, got %q", parseWant, parseOutput)
		}
	}
}

// TestTextSummaryPrintersEmitReadableOutput verifies env and prerender summary printers.
func TestTextSummaryPrintersEmitReadableOutput(parseT *testing.T) {
	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	printLauncherEnvSummary(launcherEnvSummary{
		Variables: []launcherEnvVariableRecord{
			{Name: "OPENAI_API_KEY", Set: true, Value: "[redacted len=12]", Description: "Used for OpenAI", DefaultValue: "unset"},
			{Name: "PLAYWRIGHT_WORKERS", Set: false},
		},
	})
	renderPrerenderSummary(prerenderSummary{
		Root:          ".",
		OutDir:        "./dist",
		AppPath:       "./main.go",
		HTMLPath:      "./index.html",
		WASMArtifact:  "./dist/app.wasm",
		RuntimeAsset:  "./dist/wasm_exec.js",
		Routes:        []string{"/", "/about"},
		HTMLFiles:     []string{"index.html", "about/index.html"},
		AssetFiles:    []string{"app.wasm", "wasm_exec.js"},
		ManifestPath:  "./dist/prerender.json",
		BuildExecuted: true,
		BuildProfile:  "release",
	})

	parseOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	for _, parseWant := range []string{
		"GWC env",
		"OPENAI_API_KEY [set]",
		"default: unset",
		"PLAYWRIGHT_WORKERS [unset]",
		"GWC prerender",
		"routes:        /, /about",
		"assets copied: 2",
		"build:         executed (release profile)",
	} {
		if !strings.Contains(parseOutput, parseWant) {
			parseT.Fatalf("expected text output to contain %q, got %q", parseWant, parseOutput)
		}
	}
}

// TestProcessDoneClassifierRecognizesFinishedProcesses verifies process-done error matching.
func TestProcessDoneClassifierRecognizesFinishedProcesses(parseT *testing.T) {
	if errorsIsProcessDone(nil) {
		parseT.Fatal("expected nil error to report false")
	}
	if !errorsIsProcessDone(assertErrorMessage("process already finished")) {
		parseT.Fatal("expected process-done text to report true")
	}
	if errorsIsProcessDone(assertErrorMessage("access denied")) {
		parseT.Fatal("expected unrelated error text to report false")
	}
	terminateLauncherProcessTree(nil)
}

// assertErrorMessage returns an error value for classifier tests.
func assertErrorMessage(parseMessage string) error {
	return errors.New(parseMessage)
}
