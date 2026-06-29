package main

import (
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

// TestResolveBenchmarkConfigHelpers verifies benchmark config normalization and validation.
func TestResolveBenchmarkConfigHelpers(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseResolvedConfig, parseErr := resolveBenchmarkConfig(benchmarkConfig{
		rootPath:      parseRoot,
		lanes:         []string{"all", "native"},
		bench:         "",
		count:         2,
		parallel:      3,
		outPath:       "docs/latest.json",
		referencePath: "docs/reference.json",
	})
	if parseErr != nil {
		parseT.Fatalf("resolveBenchmarkConfig: %v", parseErr)
	}
	if !slices.Equal(parseResolvedConfig.lanes, []string{"native", "wasm"}) {
		parseT.Fatalf("normalized lanes = %#v, want native+wasm", parseResolvedConfig.lanes)
	}
	if parseResolvedConfig.bench != "." || parseResolvedConfig.count != 2 || parseResolvedConfig.parallel != 3 {
		parseT.Fatalf("unexpected resolved benchmark config: %#v", parseResolvedConfig)
	}
	if !strings.HasSuffix(filepath.ToSlash(parseResolvedConfig.outPath), "/docs/latest.json") {
		parseT.Fatalf("unexpected benchmark out path: %q", parseResolvedConfig.outPath)
	}
	if !strings.HasSuffix(filepath.ToSlash(parseResolvedConfig.referencePath), "/docs/reference.json") {
		parseT.Fatalf("unexpected benchmark reference path: %q", parseResolvedConfig.referencePath)
	}

	if _, parseErr2 := normalizeBenchmarkLanes([]string{"mystery"}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "unknown benchmark lane") {
		parseT.Fatalf("expected unknown lane error, got %v", parseErr2)
	}
	if parseGot := benchmarkPackageToken("./internal/runtime"); parseGot != "._internal_runtime" {
		parseT.Fatalf("benchmarkPackageToken = %q, want ._internal_runtime", parseGot)
	}
	if benchmarkBucketLabel("end_to_end") != "End-to-End" || benchmarkBucketOrder("sync_concurrency") != 3 || benchmarkBucketOrder("custom") != 100 {
		parseT.Fatalf("unexpected benchmark bucket helpers")
	}
	if _, parseErr3 := resolveBenchmarkConfig(benchmarkConfig{rootPath: parseRoot, parallel: 0, count: 1}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "parallelism") {
		parseT.Fatalf("expected parallelism error, got %v", parseErr3)
	}
	if _, parseErr4 := resolveBenchmarkConfig(benchmarkConfig{rootPath: parseRoot, parallel: 1, count: 0}); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "count") {
		parseT.Fatalf("expected count error, got %v", parseErr4)
	}
}

// TestResolveBenchmarkCompareAndCaptureConfig verifies compare and capture config branches.
func TestResolveBenchmarkCompareAndCaptureConfig(parseT *testing.T) {
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

	parseBaselinePath := filepath.Join(parseRoot, "baseline.txt")
	parseCandidatePath := filepath.Join(parseRoot, "candidate.txt")
	if parseErr4 := os.WriteFile(parseBaselinePath, []byte("bench"), 0o644); parseErr4 != nil {
		parseT.Fatalf("WriteFile baseline: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseCandidatePath, []byte("bench"), 0o644); parseErr5 != nil {
		parseT.Fatalf("WriteFile candidate: %v", parseErr5)
	}

	parseCompareConfig, parseErr := resolveBenchmarkCompareConfig(benchmarkCompareConfig{
		baselinePath:  "baseline.txt",
		candidatePath: "candidate.txt",
	})
	if parseErr != nil {
		parseT.Fatalf("resolveBenchmarkCompareConfig: %v", parseErr)
	}
	if parseCompareConfig.baselinePath != parseBaselinePath || parseCompareConfig.candidatePath != parseCandidatePath {
		parseT.Fatalf("unexpected compare config: %#v", parseCompareConfig)
	}
	if _, parseErr2 := resolveBenchmarkCompareConfig(benchmarkCompareConfig{baselinePath: "missing.txt", candidatePath: "candidate.txt"}); parseErr2 == nil {
		parseT.Fatal("expected missing baseline error")
	}

	parseLauncher := launcher{repoRoot: parseRoot}
	parseCaptureConfig, parseErr := parseLauncher.resolveBenchmarkCaptureConfig(benchmarkCaptureConfig{
		packagePath: "./internal/runtime",
		count:       2,
	})
	if parseErr != nil {
		parseT.Fatalf("resolveBenchmarkCaptureConfig default output: %v", parseErr)
	}
	if parseCaptureConfig.bench != "." || !strings.Contains(filepath.ToSlash(parseCaptureConfig.outputPath), "/tools/bench-._internal_runtime-") {
		parseT.Fatalf("unexpected capture config: %#v", parseCaptureConfig)
	}
	if _, parseErr3 := parseLauncher.resolveBenchmarkCaptureConfig(benchmarkCaptureConfig{count: 1}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "package cannot be empty") {
		parseT.Fatalf("expected missing package error, got %v", parseErr3)
	}
	if _, parseErr4 := parseLauncher.resolveBenchmarkCaptureConfig(benchmarkCaptureConfig{packagePath: "./ui", count: 0}); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "count must be at least 1") {
		parseT.Fatalf("expected count error, got %v", parseErr4)
	}
}

// TestResolveWasmConfigHelpers verifies wasm config normalization and validation helpers.
func TestResolveWasmConfigHelpers(parseT *testing.T) {
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

	parseBaselinePath := filepath.Join(parseRoot, "baseline.json")
	parseCandidatePath := filepath.Join(parseRoot, "candidate.json")
	if parseErr4 := os.WriteFile(parseBaselinePath, []byte("{}"), 0o644); parseErr4 != nil {
		parseT.Fatalf("WriteFile baseline: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseCandidatePath, []byte("{}"), 0o644); parseErr5 != nil {
		parseT.Fatalf("WriteFile candidate: %v", parseErr5)
	}

	parseMeasureConfig, parseErr := resolveWasmMeasureConfig(wasmMeasureConfig{})
	if parseErr != nil {
		parseT.Fatalf("resolveWasmMeasureConfig: %v", parseErr)
	}
	if parseMeasureConfig.packagePath != "." || filepath.Base(parseMeasureConfig.outDir) != "wasm-build-experiment" || parseMeasureConfig.binaryName != "app.wasm" || parseMeasureConfig.manifestName != "wasm-build-experiment.json" || parseMeasureConfig.goExecutable != "go" {
		parseT.Fatalf("unexpected measure config: %#v", parseMeasureConfig)
	}

	parseCompareConfig, parseErr := resolveWasmCompareConfig(wasmCompareConfig{
		baselinePath:  "baseline.json",
		candidatePath: "candidate.json",
		outFile:       "reports/compare.json",
	})
	if parseErr != nil {
		parseT.Fatalf("resolveWasmCompareConfig: %v", parseErr)
	}
	if parseCompareConfig.baselinePath != parseBaselinePath || parseCompareConfig.candidatePath != parseCandidatePath || !strings.HasSuffix(filepath.ToSlash(parseCompareConfig.outFile), "/reports/compare.json") {
		parseT.Fatalf("unexpected compare config: %#v", parseCompareConfig)
	}
	if _, parseErr2 := resolveWasmCompareConfig(wasmCompareConfig{baselinePath: "missing.json", candidatePath: "candidate.json"}); parseErr2 == nil {
		parseT.Fatal("expected missing wasm baseline error")
	}

	parseCompressionConfig, parseErr := resolveWasmCompressionConfig(wasmCompressionConfig{})
	if parseErr != nil {
		parseT.Fatalf("resolveWasmCompressionConfig: %v", parseErr)
	}
	if parseCompressionConfig.packagePath != "." || filepath.Base(parseCompressionConfig.outDir) != "wasm-compression-comparison" || parseCompressionConfig.binaryName != "app.wasm" {
		parseT.Fatalf("unexpected compression config: %#v", parseCompressionConfig)
	}

	parseCacheConfig, parseErr := resolveWasmCacheConfig(wasmCacheConfig{})
	if parseErr != nil {
		parseT.Fatalf("resolveWasmCacheConfig: %v", parseErr)
	}
	if parseCacheConfig.packagePath != "." || filepath.Base(parseCacheConfig.outDir) != "wasm-build-cache-comparison" || parseCacheConfig.summaryName != "wasm-build-cache-comparison.json" {
		parseT.Fatalf("unexpected cache config: %#v", parseCacheConfig)
	}

	if _, parseErr3 := resolveWasmToolchainConfig(wasmToolchainConfig{}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "-package") {
		parseT.Fatalf("expected missing package error, got %v", parseErr3)
	}
	if _, parseErr4 := resolveWasmToolchainConfig(wasmToolchainConfig{packagePath: "./ui"}); parseErr4 == nil || !strings.Contains(parseErr4.Error(), "-baseline-go") {
		parseT.Fatalf("expected missing baseline-go error, got %v", parseErr4)
	}
	if _, parseErr5 := resolveWasmToolchainConfig(wasmToolchainConfig{packagePath: "./ui", baselineGoExecutable: "go1.25"}); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "-candidate-go") {
		parseT.Fatalf("expected missing candidate-go error, got %v", parseErr5)
	}
	parseToolchainConfig, parseErr := resolveWasmToolchainConfig(wasmToolchainConfig{
		packagePath:           "./ui",
		baselineGoExecutable:  "go1.25",
		candidateGoExecutable: "go1.26",
	})
	if parseErr != nil {
		parseT.Fatalf("resolveWasmToolchainConfig: %v", parseErr)
	}
	if parseToolchainConfig.binaryName != "app.wasm" || filepath.Base(parseToolchainConfig.outDir) != "wasm-toolchain-comparison" {
		parseT.Fatalf("unexpected toolchain config: %#v", parseToolchainConfig)
	}
}
