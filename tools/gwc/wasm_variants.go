package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"time"
)

func executeWasmCompareCompression(parseConfig wasmCompressionConfig) (wasmCompressionSummary, error) {
	parseVariants := map[string]interface{}{}

	parsePlainRaw, parseErr := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     parseConfig.packagePath,
		outDir:          filepath.Join(parseConfig.outDir, "plain-raw"),
		binaryName:      parseConfig.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  false,
		skipCompression: true,
	})
	if parseErr != nil {
		return wasmCompressionSummary{}, parseErr
	}
	parseVariants["plain_raw"] = parsePlainRaw.Manifest

	parseStrippedRaw, parseErr := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     parseConfig.packagePath,
		outDir:          filepath.Join(parseConfig.outDir, "stripped-raw"),
		binaryName:      parseConfig.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  true,
		skipCompression: true,
	})
	if parseErr != nil {
		return wasmCompressionSummary{}, parseErr
	}
	parseVariants["stripped_raw"] = parseStrippedRaw.Manifest

	parseStrippedCompressed, parseErr := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     parseConfig.packagePath,
		outDir:          filepath.Join(parseConfig.outDir, "stripped-compressed"),
		binaryName:      parseConfig.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  true,
		skipCompression: false,
	})
	if parseErr != nil {
		return wasmCompressionSummary{}, parseErr
	}
	parseVariants["stripped_compressed"] = parseStrippedCompressed.Manifest
	_, parseBrotliSupported := parseStrippedCompressed.Manifest.Artifacts["brotli"]

	parseOptimizerCommand, parseErr := resolveWasmOptimizerCommand()
	if parseErr != nil {
		return wasmCompressionSummary{}, parseErr
	}
	parseUnsupported := map[string]string{}
	if parseBrotliSupported {
		parseUnsupported["brotli_delivery"] = "supported"
	} else {
		parseUnsupported["brotli_delivery"] = "brotli artifact unavailable on current host"
	}
	if parseOptimizerCommand.Available {
		parseUnsupported["optimized_wasm"] = "supported"
	} else {
		parseUnsupported["optimized_wasm"] = "wasm-opt is unavailable on PATH"
	}
	if parseOptimizerCommand.Available {
		parseOptimizedRawVariant, parseOptimizedCompressedVariant, parseVariantErr := executeWasmOptimizedVariants(parseConfig, parseStrippedRaw, parseOptimizerCommand, parseBrotliSupported)
		if parseVariantErr != nil {
			return wasmCompressionSummary{}, parseVariantErr
		}
		parseVariants["optimized_raw"] = parseOptimizedRawVariant
		parseVariants["optimized_compressed"] = parseOptimizedCompressedVariant
	}

	parseGoVersionOutput, parseErr := wasmRunCommand("go", []string{"version"}, "", buildNativeGoEnv())
	if parseErr != nil {
		return wasmCompressionSummary{}, parseErr
	}
	parseSummaryPath := filepath.Join(parseConfig.outDir, parseConfig.summaryName)
	parseSummary := wasmCompressionSummary{
		OK:          true,
		Package:     parseConfig.packagePath,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Environment: wasmCompressionEnvironment{
			GoVersion:        strings.TrimSpace(parseGoVersionOutput),
			BrotliSupported:  parseBrotliSupported,
			WasmOptAvailable: parseOptimizerCommand.Available,
			WasmOptPath:      parseOptimizerCommand.Label,
		},
		Variants:    parseVariants,
		Unsupported: parseUnsupported,
		SummaryPath: filepath.ToSlash(parseSummaryPath),
	}
	parsePayload, parseErr := json.MarshalIndent(parseSummary, "", "  ")
	if parseErr != nil {
		return wasmCompressionSummary{}, fmt.Errorf("marshal wasm compression summary: %w", parseErr)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr2 := os.WriteFile(parseSummaryPath, parsePayload, 0644); parseErr2 != nil {
		return wasmCompressionSummary{}, fmt.Errorf("write wasm compression summary: %w", parseErr2)
	}
	return parseSummary, nil
}

// executeWasmCompareCache runs shared-cache and CI-style cache variants for wasm builds.
func executeWasmCompareCache(parseConfig wasmCacheConfig) (wasmCacheSummary, error) {
	parseSharedGoCache := filepath.Join(parseConfig.outDir, "shared-gocache")
	parseCiGoCache := filepath.Join(parseConfig.outDir, "ci-gocache")
	parseCiGoModCache := filepath.Join(parseConfig.outDir, "ci-gomodcache")
	parseIsolatedGoCache := filepath.Join(parseConfig.outDir, "isolated-gocache")
	for _, parsePath := range []string{parseSharedGoCache, parseCiGoCache, parseCiGoModCache, parseIsolatedGoCache} {
		if parseErr := resetWasmDirectory(parsePath); parseErr != nil {
			return wasmCacheSummary{}, parseErr
		}
	}

	parseDefaultGoCacheOutput, parseErr2 := wasmRunCommand("go", []string{"env", "GOCACHE"}, "", buildNativeGoEnv())
	if parseErr2 != nil {
		return wasmCacheSummary{}, parseErr2
	}
	parseDefaultGoModOutput, parseErr2 := wasmRunCommand("go", []string{"env", "GOMODCACHE"}, "", buildNativeGoEnv())
	if parseErr2 != nil {
		return wasmCacheSummary{}, parseErr2
	}
	parseDefaultGoCache := strings.TrimSpace(parseDefaultGoCacheOutput)
	parseDefaultGoMod := strings.TrimSpace(parseDefaultGoModOutput)

	parseVariantSpecs := []wasmCacheVariantSpec{
		{
			key:        "shared-cache-cold",
			label:      "shared cache cold",
			goCache:    parseSharedGoCache,
			goModCache: parseDefaultGoMod,
			note:       "Empty dedicated build cache with the normal module cache.",
		},
		{
			key:        "shared-cache-warm",
			label:      "shared cache warm",
			goCache:    parseSharedGoCache,
			goModCache: parseDefaultGoMod,
			note:       "Repeat build with the same dedicated build cache and reused module cache.",
		},
		{
			key:        "shared-cache-small-edit",
			label:      "shared cache small edit",
			goCache:    parseSharedGoCache,
			goModCache: parseDefaultGoMod,
			note:       "Small edit rebuild with the warmed shared build cache and reused module cache.",
			smallEdit:  true,
		},
		{
			key:        "isolated-build-cache",
			label:      "isolated build cache",
			goCache:    parseIsolatedGoCache,
			goModCache: parseDefaultGoMod,
			note:       "Fresh build cache with the normal module cache, approximating a cold compile on a prepared machine.",
		},
		{
			key:                "ci-style-cold",
			label:              "ci-style cold",
			goCache:            parseCiGoCache,
			goModCache:         parseCiGoModCache,
			note:               "Fresh build cache and fresh module cache, approximating a clean CI worker.",
			prepareModuleCache: true,
			resetBeforeRun:     true,
		},
		{
			key:        "ci-style-warm",
			label:      "ci-style warm",
			goCache:    parseCiGoCache,
			goModCache: parseCiGoModCache,
			note:       "Repeat build after the CI-style caches were hydrated once in the same comparison run.",
		},
		{
			key:        "ci-style-small-edit",
			label:      "ci-style small edit",
			goCache:    parseCiGoCache,
			goModCache: parseCiGoModCache,
			note:       "Small edit rebuild after the CI-style caches were hydrated once in the same comparison run.",
			smallEdit:  true,
		},
	}

	parseVariants := map[string]wasmCacheVariantResult{}
	isParseOverallOK := true
	for _, parseSpec := range parseVariantSpecs {
		if parseSpec.resetBeforeRun {
			if parseErr3 := resetWasmDirectory(parseSpec.goCache); parseErr3 != nil {
				return wasmCacheSummary{}, parseErr3
			}
			if parseErr4 := resetWasmDirectory(parseSpec.goModCache); parseErr4 != nil {
				return wasmCacheSummary{}, parseErr4
			}
		}
		parseResult, parseErr5 := executeWasmCacheVariant(parseConfig, parseSpec)
		if parseErr5 != nil {
			return wasmCacheSummary{}, parseErr5
		}
		parseVariants[parseSpec.key] = parseResult
		if parseResult.Status != "ok" {
			isParseOverallOK = false
		}
	}

	parseGoVersionOutput, parseErr2 := wasmRunCommand("go", []string{"version"}, "", buildNativeGoEnv())
	if parseErr2 != nil {
		return wasmCacheSummary{}, parseErr2
	}
	parseSummaryPath := filepath.Join(parseConfig.outDir, parseConfig.summaryName)
	parseSummary := wasmCacheSummary{
		OK:          isParseOverallOK,
		Package:     parseConfig.packagePath,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Environment: wasmCacheEnvironment{
			GoVersion:      strings.TrimSpace(parseGoVersionOutput),
			DefaultGoCache: parseDefaultGoCache,
			DefaultGoMod:   parseDefaultGoMod,
		},
		Variants:    parseVariants,
		SummaryPath: filepath.ToSlash(parseSummaryPath),
	}
	parsePayload, parseErr2 := json.MarshalIndent(parseSummary, "", "  ")
	if parseErr2 != nil {
		return wasmCacheSummary{}, fmt.Errorf("marshal wasm cache summary: %w", parseErr2)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr6 := os.WriteFile(parseSummaryPath, parsePayload, 0644); parseErr6 != nil {
		return wasmCacheSummary{}, fmt.Errorf("write wasm cache summary: %w", parseErr6)
	}
	if !parseSummary.OK {
		return parseSummary, errors.New("one or more cache variants failed to measure")
	}
	return parseSummary, nil
}

// executeWasmCompareToolchain measures baseline and candidate Go toolchains and compares results.
func executeWasmCompareToolchain(parseConfig wasmToolchainConfig) (wasmToolchainSummary, error) {
	parseBaselineOutDir := filepath.Join(parseConfig.outDir, "baseline")
	parseCandidateOutDir := filepath.Join(parseConfig.outDir, "candidate")
	parseComparisonPath := filepath.Join(parseConfig.outDir, "toolchain-comparison.json")
	parseSummaryPath := filepath.Join(parseConfig.outDir, "wasm-toolchain-comparison.json")

	_, parseErr := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     parseConfig.packagePath,
		outDir:          parseBaselineOutDir,
		binaryName:      parseConfig.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    parseConfig.baselineGoExecutable,
		ldflags:         "-s -w",
		releaseProfile:  parseConfig.releaseProfile,
		skipCompression: parseConfig.skipCompression,
	})
	if parseErr != nil {
		return wasmToolchainSummary{}, parseErr
	}
	_, parseErr = executeWasmMeasure(wasmMeasureConfig{
		packagePath:     parseConfig.packagePath,
		outDir:          parseCandidateOutDir,
		binaryName:      parseConfig.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    parseConfig.candidateGoExecutable,
		ldflags:         "-s -w",
		releaseProfile:  parseConfig.releaseProfile,
		skipCompression: parseConfig.skipCompression,
	})
	if parseErr != nil {
		return wasmToolchainSummary{}, parseErr
	}

	parseBaselineManifestPath := filepath.Join(parseBaselineOutDir, "wasm-build-experiment.json")
	parseCandidateManifestPath := filepath.Join(parseCandidateOutDir, "wasm-build-experiment.json")
	_, parseRegressionCount, parseCompareErr := executeWasmCompare(wasmCompareConfig{
		baselinePath:            parseBaselineManifestPath,
		candidatePath:           parseCandidateManifestPath,
		outFile:                 parseComparisonPath,
		timingRegressionPercent: parseConfig.timingRegressionPercent,
		sizeRegressionPercent:   parseConfig.sizeRegressionPercent,
		otherRegressionPercent:  parseConfig.otherRegressionPercent,
	})
	parseRegressionExitCode := 0
	if parseCompareErr != nil || parseRegressionCount > 0 {
		parseRegressionExitCode = 1
	}

	parseBaselineVersion, parseErr := wasmRunCommand(parseConfig.baselineGoExecutable, []string{"version"}, "", buildNativeGoEnv())
	if parseErr != nil {
		return wasmToolchainSummary{}, parseErr
	}
	parseCandidateVersion, parseErr := wasmRunCommand(parseConfig.candidateGoExecutable, []string{"version"}, "", buildNativeGoEnv())
	if parseErr != nil {
		return wasmToolchainSummary{}, parseErr
	}

	parseSummary := wasmToolchainSummary{
		OK:         parseRegressionExitCode == 0,
		Package:    parseConfig.packagePath,
		ComparedAt: time.Now().UTC().Format(time.RFC3339),
		Baseline: wasmToolchainParty{
			GoExecutable: parseConfig.baselineGoExecutable,
			GoVersion:    strings.TrimSpace(parseBaselineVersion),
			Manifest:     "baseline/wasm-build-experiment.json",
		},
		Candidate: wasmToolchainParty{
			GoExecutable: parseConfig.candidateGoExecutable,
			GoVersion:    strings.TrimSpace(parseCandidateVersion),
			Manifest:     "candidate/wasm-build-experiment.json",
		},
		Thresholds: wasmCompareThresholds{
			TimingRegressionPercent: parseConfig.timingRegressionPercent,
			SizeRegressionPercent:   parseConfig.sizeRegressionPercent,
			OtherRegressionPercent:  parseConfig.otherRegressionPercent,
		},
		Comparison:         "toolchain-comparison.json",
		RegressionExitCode: parseRegressionExitCode,
		SummaryPath:        filepath.ToSlash(parseSummaryPath),
	}
	parsePayload, parseErr := json.MarshalIndent(parseSummary, "", "  ")
	if parseErr != nil {
		return wasmToolchainSummary{}, fmt.Errorf("marshal wasm toolchain summary: %w", parseErr)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr2 := os.WriteFile(parseSummaryPath, parsePayload, 0644); parseErr2 != nil {
		return wasmToolchainSummary{}, fmt.Errorf("write wasm toolchain summary: %w", parseErr2)
	}
	if parseCompareErr != nil {
		return parseSummary, parseCompareErr
	}
	if parseRegressionCount > 0 {
		return parseSummary, fmt.Errorf("detected %d metric regressions beyond configured thresholds", parseRegressionCount)
	}
	return parseSummary, nil
}

// executeWasmCacheVariant runs one cache topology variant and returns its structured result.
func executeWasmCacheVariant(parseConfig wasmCacheConfig, parseSpec wasmCacheVariantSpec) (wasmCacheVariantResult, error) {
	parseVariantOutDir := filepath.Join(parseConfig.outDir, parseSpec.key)
	if parseErr := os.MkdirAll(parseVariantOutDir, 0755); parseErr != nil {
		return wasmCacheVariantResult{}, fmt.Errorf("create cache variant output dir: %w", parseErr)
	}
	parseResult := wasmCacheVariantResult{
		Label:      parseSpec.label,
		GoCache:    parseSpec.goCache,
		GoModCache: parseSpec.goModCache,
		Status:     "pending",
		Notes:      []string{parseSpec.note},
	}

	parseRestoreGoCache := setWasmEnv("GOCACHE", parseSpec.goCache)
	defer parseRestoreGoCache()
	parseRestoreGoMod := setWasmEnv("GOMODCACHE", parseSpec.goModCache)
	defer parseRestoreGoMod()

	if parseSpec.prepareModuleCache {
		parseDownloadStart := time.Now()
		_, parseErr2 := wasmRunCommand("go", []string{"mod", "download"}, "", buildNativeGoEnv())
		parseDownloadMs := time.Since(parseDownloadStart).Milliseconds()
		parseResult.ModuleDownloadMS = &parseDownloadMs
		if parseErr2 != nil {
			parseResult.Status = "failed"
			parseResult.Error = "go mod download failed"
			return parseResult, nil
		}
	}

	parseRestoreEdit, parseEditedFile, parseEditErr := applyWasmSmallEdit(parseConfig.packagePath, parseSpec.smallEdit)
	if parseEditErr != nil {
		parseResult.Status = "failed"
		parseResult.Error = parseEditErr.Error()
		return parseResult, nil
	}
	if parseRestoreEdit != nil {
		defer parseRestoreEdit()
		parseResult.EditedFile = filepath.ToSlash(parseEditedFile)
		parseResult.Notes = append(parseResult.Notes, "A temporary comment edit was applied and restored to measure rebuild invalidation.")
	}

	parseMeasurement, parseErr3 := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     parseConfig.packagePath,
		outDir:          parseVariantOutDir,
		binaryName:      parseConfig.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  parseConfig.releaseProfile,
		skipCompression: false,
	})
	if parseErr3 != nil {
		parseResult.Status = "failed"
		parseResult.Error = "wasm measurement failed"
		return parseResult, nil
	}
	parseResult.Status = "ok"
	parseResult.Measurement = &parseMeasurement.Manifest
	return parseResult, nil
}

// resetWasmDirectory clears and recreates one directory.
func resetWasmDirectory(parsePath string) error {
	if strings.TrimSpace(parsePath) == "" {
		return errors.New("cannot reset an empty directory path")
	}
	if parseErr := os.RemoveAll(parsePath); parseErr != nil {
		return fmt.Errorf("reset directory %s: %w", parsePath, parseErr)
	}
	if parseErr2 := os.MkdirAll(parsePath, 0755); parseErr2 != nil {
		return fmt.Errorf("create directory %s: %w", parsePath, parseErr2)
	}
	return nil
}

// setWasmEnv sets one environment variable and returns a restore function.
func setWasmEnv(parseName string, parseValue string) func() {
	parsePrevious, parseHadPrevious := os.LookupEnv(parseName)
	_ = os.Setenv(parseName, parseValue)
	return func() {
		if parseHadPrevious {
			_ = os.Setenv(parseName, parsePrevious)
			return
		}
		_ = os.Unsetenv(parseName)
	}
}

// applyWasmSmallEdit appends a temporary marker comment and returns a restore closure.
func applyWasmSmallEdit(parsePackagePath string, isEnabled bool) (func(), string, error) {
	if !isEnabled {
		return nil, "", nil
	}
	parseEditableFile, parseErr := resolveWasmCacheEditableFile(parsePackagePath)
	if parseErr != nil {
		return nil, "", parseErr
	}
	parseOriginalBytes, parseErr := os.ReadFile(parseEditableFile)
	if parseErr != nil {
		return nil, "", fmt.Errorf("read editable file: %w", parseErr)
	}
	parseUpdatedBytes := append(append([]byte(nil), parseOriginalBytes...), []byte("\n// cache experiment marker\n")...)
	if parseErr2 := os.WriteFile(parseEditableFile, parseUpdatedBytes, 0644); parseErr2 != nil {
		return nil, "", fmt.Errorf("apply temporary cache edit: %w", parseErr2)
	}
	parseRestore := func() {
		_ = os.WriteFile(parseEditableFile, parseOriginalBytes, 0644)
	}
	return parseRestore, parseEditableFile, nil
}

// resolveWasmCacheEditableFile picks the first non-test Go file for small-edit variants.
func resolveWasmCacheEditableFile(parsePackagePath string) (string, error) {
	parseCwd, parseErr := os.Getwd()
	if parseErr != nil {
		return "", fmt.Errorf("resolve wasm cache package cwd: %w", parseErr)
	}
	parseResolvedPath, parseErr := normalizePath(parseCwd, parsePackagePath)
	if parseErr != nil {
		return "", fmt.Errorf("resolve wasm cache package path: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseResolvedPath)
	if parseErr != nil {
		return "", fmt.Errorf("inspect wasm cache package path: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		return "", fmt.Errorf("small-edit measurement requires a package directory path: %s", parsePackagePath)
	}
	parseEntries, parseErr := os.ReadDir(parseResolvedPath)
	if parseErr != nil {
		return "", fmt.Errorf("read wasm cache package directory: %w", parseErr)
	}
	parseNames := make([]string, 0, len(parseEntries))
	for _, parseEntry := range parseEntries {
		if parseEntry.IsDir() {
			continue
		}
		parseName := parseEntry.Name()
		if !strings.HasSuffix(parseName, ".go") || strings.HasSuffix(parseName, "_test.go") {
			continue
		}
		parseNames = append(parseNames, parseName)
	}
	sort.Strings(parseNames)
	if len(parseNames) == 0 {
		return "", fmt.Errorf("no editable Go source file found for package: %s", parsePackagePath)
	}
	return filepath.Join(parseResolvedPath, parseNames[0]), nil
}

// executeWasmOptimizedVariants creates optimized wasm variants from a stripped baseline.
func executeWasmOptimizedVariants(parseConfig wasmCompressionConfig, parseStrippedRaw wasmMeasureSummary, parseCommand wasmOptimizerCommand, isIncludeBrotli bool) (map[string]interface{}, map[string]interface{}, error) {
	parseStrippedRawPath := filepath.Join(filepath.FromSlash(parseStrippedRaw.OutDir), parseConfig.binaryName)
	parseOptimizedRawDir := filepath.Join(parseConfig.outDir, "optimized-raw")
	parseOptimizedCompressedDir := filepath.Join(parseConfig.outDir, "optimized-compressed")
	if parseErr := os.MkdirAll(parseOptimizedRawDir, 0755); parseErr != nil {
		return nil, nil, parseErr
	}
	if parseErr2 := os.MkdirAll(parseOptimizedCompressedDir, 0755); parseErr2 != nil {
		return nil, nil, parseErr2
	}
	parseOptimizedRawPath := filepath.Join(parseOptimizedRawDir, parseConfig.binaryName)
	parseOptimizedCompressedPath := filepath.Join(parseOptimizedCompressedDir, parseConfig.binaryName)

	parseOptStart := time.Now()
	if parseErr3 := runWasmOptimizer(parseCommand, parseStrippedRawPath, parseOptimizedRawPath); parseErr3 != nil {
		return nil, nil, parseErr3
	}
	parseWasmOptMs := time.Since(parseOptStart).Milliseconds()
	parseGoBuildMs := parseStrippedRaw.Manifest.Phases["go_build_ms"]

	parseOptimizedRawArtifact, parseErr4 := wasmReleaseArtifactRecordForPath(parseOptimizedRawDir, parseOptimizedRawPath)
	if parseErr4 != nil {
		return nil, nil, parseErr4
	}
	parseOptimizedRaw := map[string]interface{}{
		"package":    parseConfig.packagePath,
		"profile":    "release-optimized",
		"go_version": parseStrippedRaw.Manifest.GoVersion,
		"goos":       "js",
		"goarch":     "wasm",
		"build_args": parseStrippedRaw.Manifest.BuildArgs,
		"optimizer":  map[string]interface{}{"tool": parseCommand.Label, "args": append(append([]string{}, parseCommand.PrefixArgs...), parseStrippedRawPath, "-Oz", "-o", parseOptimizedRawPath)},
		"phases":     map[string]int64{"go_build_ms": parseGoBuildMs, "wasm_opt_ms": parseWasmOptMs, "compression_total_ms": 0, "total_wall_ms": parseGoBuildMs + parseWasmOptMs},
		"artifacts":  map[string]releaseArtifactRecord{"wasm": parseOptimizedRawArtifact},
	}

	parseOptimizedBytes, parseErr4 := os.ReadFile(parseOptimizedRawPath)
	if parseErr4 != nil {
		return nil, nil, parseErr4
	}
	if parseErr5 := os.WriteFile(parseOptimizedCompressedPath, parseOptimizedBytes, 0644); parseErr5 != nil {
		return nil, nil, parseErr5
	}
	if parseErr6 := assertWasmFileParity(parseOptimizedRawPath, parseOptimizedCompressedPath); parseErr6 != nil {
		return nil, nil, parseErr6
	}
	parseGzipStart := time.Now()
	parseOptimizedGzipPath := parseOptimizedCompressedPath + ".gz"
	if parseErr7 := wasmWriteGzipSidecar(parseOptimizedCompressedPath, parseOptimizedGzipPath); parseErr7 != nil {
		return nil, nil, parseErr7
	}
	parseGzipMs := time.Since(parseGzipStart).Milliseconds()
	parseCompressionTotalMs := parseGzipMs

	parseArtifacts := map[string]releaseArtifactRecord{}
	parseOptimizedCompressedArtifact, parseErr4 := wasmReleaseArtifactRecordForPath(parseOptimizedCompressedDir, parseOptimizedCompressedPath)
	if parseErr4 != nil {
		return nil, nil, parseErr4
	}
	if parseErr8 := assertWasmArtifactParity("optimized raw -> optimized compressed wasm copy", parseOptimizedRawArtifact, parseOptimizedCompressedArtifact); parseErr8 != nil {
		return nil, nil, parseErr8
	}
	parseArtifacts["wasm"] = parseOptimizedCompressedArtifact
	parseOptimizedGzipArtifact, parseErr4 := wasmReleaseArtifactRecordForPath(parseOptimizedCompressedDir, parseOptimizedGzipPath)
	if parseErr4 != nil {
		return nil, nil, parseErr4
	}
	parseArtifacts["gzip"] = parseOptimizedGzipArtifact

	parsePhases := map[string]int64{
		"go_build_ms":          parseGoBuildMs,
		"wasm_opt_ms":          parseWasmOptMs,
		"gzip_ms":              parseGzipMs,
		"compression_total_ms": parseCompressionTotalMs,
	}
	if isIncludeBrotli {
		parseBrotliStart := time.Now()
		parseOptimizedBrotliPath := parseOptimizedCompressedPath + ".br"
		if parseErr9 := wasmWriteBrotliSidecar(parseOptimizedCompressedPath, parseOptimizedBrotliPath); parseErr9 != nil {
			return nil, nil, parseErr9
		}
		parseBrotliMs := time.Since(parseBrotliStart).Milliseconds()
		parsePhases["brotli_ms"] = parseBrotliMs
		parseCompressionTotalMs += parseBrotliMs
		parsePhases["compression_total_ms"] = parseCompressionTotalMs
		parseBrotliArtifact, parseErr10 := wasmReleaseArtifactRecordForPath(parseOptimizedCompressedDir, parseOptimizedBrotliPath)
		if parseErr10 != nil {
			return nil, nil, parseErr10
		}
		parseArtifacts["brotli"] = parseBrotliArtifact
	}
	parsePhases["total_wall_ms"] = parseGoBuildMs + parseWasmOptMs + parseCompressionTotalMs

	parseOptimizedCompressed := map[string]interface{}{
		"package":       parseConfig.packagePath,
		"profile":       "release-optimized",
		"go_version":    parseStrippedRaw.Manifest.GoVersion,
		"goos":          "js",
		"goarch":        "wasm",
		"build_args":    parseStrippedRaw.Manifest.BuildArgs,
		"optimizer":     map[string]interface{}{"tool": parseCommand.Label, "args": append(append([]string{}, parseCommand.PrefixArgs...), parseStrippedRawPath, "-Oz", "-o", parseOptimizedCompressedPath)},
		"phases":        parsePhases,
		"artifacts":     parseArtifacts,
		"parity_checks": map[string]interface{}{"raw_to_delivery_copy": map[string]interface{}{"ok": true, "source_bytes": parseOptimizedRawArtifact.Bytes, "target_bytes": parseOptimizedCompressedArtifact.Bytes, "source_sha256": parseOptimizedRawArtifact.SHA256, "target_sha256": parseOptimizedCompressedArtifact.SHA256}},
	}
	return parseOptimizedRaw, parseOptimizedCompressed, nil
}

// assertWasmFileParity validates byte-for-byte equality for two artifact paths.
func assertWasmFileParity(parseSourcePath string, parseTargetPath string) error {
	parseSourceBytes, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		return fmt.Errorf("read source artifact for parity check: %w", parseErr)
	}
	parseTargetBytes, parseErr := os.ReadFile(parseTargetPath)
	if parseErr != nil {
		return fmt.Errorf("read target artifact for parity check: %w", parseErr)
	}
	if !bytes.Equal(parseSourceBytes, parseTargetBytes) {
		return fmt.Errorf("optimized wasm parity mismatch between %s and %s", filepath.ToSlash(parseSourcePath), filepath.ToSlash(parseTargetPath))
	}
	return nil
}

// assertWasmArtifactParity validates byte-size and hash parity for two artifact records.
func assertWasmArtifactParity(parseLabel string, parseSource releaseArtifactRecord, parseTarget releaseArtifactRecord) error {
	if parseSource.Bytes != parseTarget.Bytes {
		return fmt.Errorf("%s parity mismatch: bytes %d != %d", parseLabel, parseSource.Bytes, parseTarget.Bytes)
	}
	if strings.TrimSpace(parseSource.SHA256) == "" || strings.TrimSpace(parseTarget.SHA256) == "" {
		return fmt.Errorf("%s parity check requires non-empty sha256 records", parseLabel)
	}
	if parseSource.SHA256 != parseTarget.SHA256 {
		return fmt.Errorf("%s parity mismatch: sha256 %s != %s", parseLabel, parseSource.SHA256, parseTarget.SHA256)
	}
	return nil
}

// resolveWasmOptimizerCommand discovers wasm-opt from PATH.
func resolveWasmOptimizerCommand() (wasmOptimizerCommand, error) {
	if parsePath, parseErr := wasmLookPath("wasm-opt"); parseErr == nil {
		return wasmOptimizerCommand{
			Available: true,
			Command:   "wasm-opt",
			Label:     parsePath,
		}, nil
	}
	return wasmOptimizerCommand{}, nil
}

// runWasmOptimizer executes wasm-opt for one artifact pair.
func runWasmOptimizer(parseCommand wasmOptimizerCommand, parseSourcePath string, parseTargetPath string) error {
	parseArgs := append([]string{}, parseCommand.PrefixArgs...)
	parseArgs = append(parseArgs, parseSourcePath, "-Oz", "-o", parseTargetPath)
	_, parseErr := wasmRunCommand(parseCommand.Command, parseArgs, filepath.Dir(parseSourcePath), buildNativeGoEnv())
	if parseErr != nil {
		return fmt.Errorf("run wasm optimizer: %w", parseErr)
	}
	return nil
}

// collectWasmNumericMetrics flattens numeric JSON values into a path-to-value map.
func collectWasmNumericMetrics(parseValue interface{}, parsePath string, parseMetrics map[string]float64) {
	switch parseTyped := parseValue.(type) {
	case map[string]interface{}:
		for parseKey, parseChild := range parseTyped {
			parseChildPath := parseKey
			if strings.TrimSpace(parsePath) != "" {
				parseChildPath = parsePath + "." + parseKey
			}
			collectWasmNumericMetrics(parseChild, parseChildPath, parseMetrics)
		}
	case []interface{}:
		for parseIndex, parseChild2 := range parseTyped {
			parseChildPath2 := fmt.Sprintf("[%d]", parseIndex)
			if strings.TrimSpace(parsePath) != "" {
				parseChildPath2 = fmt.Sprintf("%s[%d]", parsePath, parseIndex)
			}
			collectWasmNumericMetrics(parseChild2, parseChildPath2, parseMetrics)
		}
	case float64:
		parseMetrics[parsePath] = parseTyped
	}
}

// wasmMetricCategory classifies a flattened metric path.
func wasmMetricCategory(parsePath string) string {
	parseTrimmed := strings.TrimSpace(parsePath)
	if strings.HasSuffix(parseTrimmed, "_ms") || strings.Contains(parseTrimmed, "module_download_ms") {
		return "timing"
	}
	if strings.HasSuffix(parseTrimmed, "bytes") {
		return "size"
	}
	return "other"
}

// wasmMetricThreshold resolves the regression threshold for one metric path.
func wasmMetricThreshold(parsePath string, parseConfig wasmCompareConfig) float64 {
	switch wasmMetricCategory(parsePath) {
	case "timing":
		return parseConfig.timingRegressionPercent
	case "size":
		return parseConfig.sizeRegressionPercent
	default:
		return parseConfig.otherRegressionPercent
	}
}

// wasmMeasureProfileName returns the serialized profile label for wasm measurement manifests.
func wasmMeasureProfileName(isReleaseProfile bool) string {
	if isReleaseProfile {
		return "release"
	}
	return "debug"
}

// printWasmMeasureSummary prints a concise human-readable wasm measure result.
func printWasmMeasureSummary(parseSummary wasmMeasureSummary) {
	fmt.Println("GWC wasm measure")
	fmt.Printf("  out dir:      %s\n", parseSummary.OutDir)
	fmt.Printf("  manifest:     %s\n", parseSummary.ManifestPath)
	fmt.Printf("  package:      %s\n", parseSummary.Manifest.Package)
	fmt.Printf("  profile:      %s\n", parseSummary.Manifest.Profile)
	fmt.Printf("  go version:   %s\n", parseSummary.Manifest.GoVersion)
	if buildMs, parseOk := parseSummary.Manifest.Phases["go_build_ms"]; parseOk {
		fmt.Printf("  go build ms:  %s\n", strconv.FormatInt(buildMs, 10))
	}
	parseKeys := make([]string, 0, len(parseSummary.Manifest.Artifacts))
	for parseKey := range parseSummary.Manifest.Artifacts {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	for _, parseKey2 := range parseKeys {
		parseRecord := parseSummary.Manifest.Artifacts[parseKey2]
		fmt.Printf("  artifact[%s]: %s (%d bytes)\n", parseKey2, parseRecord.Path, parseRecord.Bytes)
	}
}

// printWasmCompareSummary prints a concise human-readable comparison summary.
func printWasmCompareSummary(parseSummary wasmCompareSummary) {
	fmt.Println("GWC wasm compare")
	fmt.Printf("  baseline:    %s\n", parseSummary.Baseline)
	fmt.Printf("  candidate:   %s\n", parseSummary.Candidate)
	fmt.Printf("  total:       %d\n", parseSummary.Counts.Total)
	fmt.Printf("  improved:    %d\n", parseSummary.Counts.Improved)
	fmt.Printf("  regressed:   %d\n", parseSummary.Counts.Regressed)
	fmt.Printf("  unchanged:   %d\n", parseSummary.Counts.Unchanged)
	fmt.Printf("  thresholded: %d\n", parseSummary.Counts.WithinThreshold)
}

// printWasmCacheSummary prints a concise human-readable cache comparison summary.
func printWasmCacheSummary(parseSummary wasmCacheSummary) {
	fmt.Println("GWC wasm compare-cache")
	fmt.Printf("  package:     %s\n", parseSummary.Package)
	fmt.Printf("  generated:   %s\n", parseSummary.GeneratedAt)
	fmt.Printf("  go version:  %s\n", parseSummary.Environment.GoVersion)
	fmt.Printf("  gocache:     %s\n", parseSummary.Environment.DefaultGoCache)
	fmt.Printf("  gomodcache:  %s\n", parseSummary.Environment.DefaultGoMod)
	if strings.TrimSpace(parseSummary.SummaryPath) != "" {
		fmt.Printf("  summary:     %s\n", parseSummary.SummaryPath)
	}
	parseKeys := make([]string, 0, len(parseSummary.Variants))
	for parseKey := range parseSummary.Variants {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	for _, parseKey2 := range parseKeys {
		parseVariant := parseSummary.Variants[parseKey2]
		fmt.Printf("  [%s] %s\n", parseVariant.Status, parseKey2)
		if strings.TrimSpace(parseVariant.Error) != "" {
			fmt.Printf("    error: %s\n", parseVariant.Error)
		}
	}
}

// printWasmToolchainSummary prints a concise human-readable toolchain comparison summary.
func printWasmToolchainSummary(parseSummary wasmToolchainSummary) {
	fmt.Println("GWC wasm compare-toolchain")
	fmt.Printf("  package:      %s\n", parseSummary.Package)
	fmt.Printf("  compared:     %s\n", parseSummary.ComparedAt)
	fmt.Printf("  baseline go:  %s\n", parseSummary.Baseline.GoExecutable)
	fmt.Printf("  candidate go: %s\n", parseSummary.Candidate.GoExecutable)
	fmt.Printf("  comparison:   %s\n", parseSummary.Comparison)
	fmt.Printf("  regressions:  %d\n", parseSummary.RegressionExitCode)
	if strings.TrimSpace(parseSummary.SummaryPath) != "" {
		fmt.Printf("  summary:      %s\n", parseSummary.SummaryPath)
	}
}

// printWasmCompressionSummary prints a concise human-readable compression comparison summary.
func printWasmCompressionSummary(parseSummary wasmCompressionSummary) {
	fmt.Println("GWC wasm compare-compression")
	fmt.Printf("  package:     %s\n", parseSummary.Package)
	fmt.Printf("  generated:   %s\n", parseSummary.GeneratedAt)
	fmt.Printf("  go version:  %s\n", parseSummary.Environment.GoVersion)
	fmt.Printf("  brotli:      %t\n", parseSummary.Environment.BrotliSupported)
	fmt.Printf("  wasm-opt:    %t\n", parseSummary.Environment.WasmOptAvailable)
	if strings.TrimSpace(parseSummary.Environment.WasmOptPath) != "" {
		fmt.Printf("  wasm-opt id: %s\n", parseSummary.Environment.WasmOptPath)
	}
	if strings.TrimSpace(parseSummary.SummaryPath) != "" {
		fmt.Printf("  summary:     %s\n", parseSummary.SummaryPath)
	}
	parseKeys := make([]string, 0, len(parseSummary.Variants))
	for parseKey := range parseSummary.Variants {
		parseKeys = append(parseKeys, parseKey)
	}
	sort.Strings(parseKeys)
	for _, parseKey2 := range parseKeys {
		fmt.Printf("  variant:     %s\n", parseKey2)
	}
}
