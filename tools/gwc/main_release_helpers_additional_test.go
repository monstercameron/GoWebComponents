package main

import (
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestReleaseApplyPostLinkOptimizationCoversModesAndFailures verifies optimizer mode handling, command failures, and success replacement.
func TestReleaseApplyPostLinkOptimizationCoversModesAndFailures(parseT *testing.T) {
	parseOriginalLookPath := releaseLookPath
	parseOriginalRunCommand := releaseRunCommand
	parseT.Cleanup(func() {
		releaseLookPath = parseOriginalLookPath
		releaseRunCommand = parseOriginalRunCommand
	})

	parseT.Run("skips none mode", func(parseT2 *testing.T) {
		parseRecord, parseErr := releaseApplyPostLinkOptimization("none", filepath.Join(parseT2.TempDir(), "app.wasm"))
		if parseErr != nil {
			parseT2.Fatalf("releaseApplyPostLinkOptimization(none): %v", parseErr)
		}
		if parseRecord != nil {
			parseT2.Fatalf("expected nil optimizer record, got %#v", parseRecord)
		}
	})

	parseT.Run("reports optimizer command failure", func(parseT2 *testing.T) {
		parseWasmPath := filepath.Join(parseT2.TempDir(), "app.wasm")
		if parseErr := os.WriteFile(parseWasmPath, []byte("wasm"), 0644); parseErr != nil {
			parseT2.Fatalf("write wasm artifact: %v", parseErr)
		}
		releaseLookPath = func(parseFile string) (string, error) {
			return filepath.Join(parseT2.TempDir(), "wasm-opt.exe"), nil
		}
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return "", errors.New("optimizer failed")
		}

		_, parseErr := releaseApplyPostLinkOptimization("wasm-opt", parseWasmPath)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "run post-link optimizer") {
			parseT2.Fatalf("expected optimizer command failure, got %v", parseErr)
		}
	})

	parseT.Run("requires optimized artifact output", func(parseT2 *testing.T) {
		parseWasmPath := filepath.Join(parseT2.TempDir(), "app.wasm")
		if parseErr := os.WriteFile(parseWasmPath, []byte("wasm"), 0644); parseErr != nil {
			parseT2.Fatalf("write wasm artifact: %v", parseErr)
		}
		releaseLookPath = func(parseFile string) (string, error) {
			return filepath.Join(parseT2.TempDir(), "wasm-opt.exe"), nil
		}
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return "", nil
		}

		_, parseErr := releaseApplyPostLinkOptimization("wasm-opt", parseWasmPath)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "read optimized wasm artifact") {
			parseT2.Fatalf("expected optimized artifact read failure, got %v", parseErr)
		}
	})

	parseT.Run("replaces artifact with optimized output", func(parseT2 *testing.T) {
		parseWasmPath := filepath.Join(parseT2.TempDir(), "app.wasm")
		if parseErr := os.WriteFile(parseWasmPath, []byte("raw-wasm"), 0644); parseErr != nil {
			parseT2.Fatalf("write wasm artifact: %v", parseErr)
		}
		parseResolvedTool := filepath.Join(parseT2.TempDir(), "wasm-opt.exe")
		releaseLookPath = func(parseFile string) (string, error) {
			return parseResolvedTool, nil
		}
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			parseOptimizedPath := parseWasmPath + ".opt"
			if parseCommand != "wasm-opt" {
				parseT2.Fatalf("expected wasm-opt command, got %q", parseCommand)
			}
			if parseErr := os.WriteFile(parseOptimizedPath, []byte("optimized-wasm"), 0644); parseErr != nil {
				parseT2.Fatalf("write optimized artifact: %v", parseErr)
			}
			return "", nil
		}

		parseRecord, parseErr := releaseApplyPostLinkOptimization("wasm-opt", parseWasmPath)
		if parseErr != nil {
			parseT2.Fatalf("releaseApplyPostLinkOptimization(success): %v", parseErr)
		}
		if parseRecord == nil || parseRecord.Mode != "wasm-opt" || parseRecord.Tool != parseResolvedTool {
			parseT2.Fatalf("unexpected optimizer record %#v", parseRecord)
		}
		parseBytes, parseErr2 := os.ReadFile(parseWasmPath)
		if parseErr2 != nil {
			parseT2.Fatalf("read optimized wasm artifact: %v", parseErr2)
		}
		if string(parseBytes) != "optimized-wasm" {
			parseT2.Fatalf("expected optimized wasm bytes, got %q", string(parseBytes))
		}
		if _, parseErr3 := os.Stat(parseWasmPath + ".opt"); !os.IsNotExist(parseErr3) {
			parseT2.Fatalf("expected temporary optimized artifact cleanup, stat err=%v", parseErr3)
		}
	})
}

// TestReleaseWriteSizeAttributionCoversSuccessAndFailures verifies release attribution writing, marshal failures, and output collisions.
func TestReleaseWriteSizeAttributionCoversSuccessAndFailures(parseT *testing.T) {
	parseOriginalRunCommand := releaseRunCommand
	parseOriginalMarshalIndent := releaseMarshalIndent
	parseT.Cleanup(func() {
		releaseRunCommand = parseOriginalRunCommand
		releaseMarshalIndent = parseOriginalMarshalIndent
	})

	parsePackageRoot := parseT.TempDir()
	parseAlphaDir := filepath.Join(parsePackageRoot, "alpha")
	parseBetaDir := filepath.Join(parsePackageRoot, "beta")
	parseZeroDir := filepath.Join(parsePackageRoot, "zero")
	for _, parseDir := range []string{parseAlphaDir, parseBetaDir, parseZeroDir} {
		if parseErr := os.MkdirAll(parseDir, 0755); parseErr != nil {
			parseT.Fatalf("mkdir package dir: %v", parseErr)
		}
	}
	parseAlphaArchive := filepath.Join(parsePackageRoot, "alpha.a")
	if parseErr := os.WriteFile(parseAlphaArchive, []byte("archive-bytes"), 0644); parseErr != nil {
		parseT.Fatalf("write alpha archive: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseAlphaDir, "alpha.go"), []byte("package alpha\n"), 0644); parseErr != nil {
		parseT.Fatalf("write alpha.go: %v", parseErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseBetaDir, "beta.go"), []byte("package beta\n"), 0644); parseErr != nil {
		parseT.Fatalf("write beta.go: %v", parseErr)
	}

	parseOutput := buildReleasePackageJSONStream(parseT, []releaseGoListPackage{
		{ImportPath: "", Dir: parsePackageRoot},
		{ImportPath: "example.com/zero", Dir: parseZeroDir},
		{ImportPath: "example.com/beta", Dir: parseBetaDir, GoFiles: []string{"beta.go"}},
		{ImportPath: "example.com/alpha", Dir: parseAlphaDir, Export: parseAlphaArchive, GoFiles: []string{"alpha.go"}},
	})

	parseT.Run("skips none mode", func(parseT2 *testing.T) {
		parseRecord, parseErr := releaseWriteSizeAttribution("none", parsePackageRoot, parseT2.TempDir())
		if parseErr != nil {
			parseT2.Fatalf("releaseWriteSizeAttribution(none): %v", parseErr)
		}
		if parseRecord != nil {
			parseT2.Fatalf("expected nil attribution record, got %#v", parseRecord)
		}
	})

	parseT.Run("writes sorted attribution artifact", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return parseOutput, nil
		}
		releaseMarshalIndent = json.MarshalIndent

		parseRecord, parseErr := releaseWriteSizeAttribution("packages", parsePackageRoot, parseOutDir)
		if parseErr != nil {
			parseT2.Fatalf("releaseWriteSizeAttribution(success): %v", parseErr)
		}
		if parseRecord == nil || parseRecord.Mode != "packages" || parseRecord.PackageCount != 2 {
			parseT2.Fatalf("unexpected attribution record %#v", parseRecord)
		}
		parseArtifactPath := filepath.Join(parseOutDir, "wasm-package-size-attribution.json")
		parseBytes, parseErr2 := os.ReadFile(parseArtifactPath)
		if parseErr2 != nil {
			parseT2.Fatalf("read attribution artifact: %v", parseErr2)
		}
		if !strings.Contains(string(parseBytes), `"packages"`) || !strings.Contains(string(parseBytes), "example.com/alpha") {
			parseT2.Fatalf("expected attribution artifact to contain package payload, got %s", string(parseBytes))
		}
	})

	parseT.Run("reports marshal failure", func(parseT2 *testing.T) {
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return parseOutput, nil
		}
		releaseMarshalIndent = func(parseV any, parsePrefix string, parseIndent string) ([]byte, error) {
			return nil, errors.New("marshal failed")
		}

		_, parseErr := releaseWriteSizeAttribution("packages", parsePackageRoot, parseT2.TempDir())
		if parseErr == nil || !strings.Contains(parseErr.Error(), "encode release size attribution") {
			parseT2.Fatalf("expected attribution marshal failure, got %v", parseErr)
		}
	})

	parseT.Run("reports write failure", func(parseT2 *testing.T) {
		parseOutDir := parseT2.TempDir()
		parseCollisionPath := filepath.Join(parseOutDir, "wasm-package-size-attribution.json")
		if parseErr := os.MkdirAll(parseCollisionPath, 0755); parseErr != nil {
			parseT2.Fatalf("mkdir collision path: %v", parseErr)
		}
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return parseOutput, nil
		}
		releaseMarshalIndent = json.MarshalIndent

		_, parseErr := releaseWriteSizeAttribution("packages", parsePackageRoot, parseOutDir)
		if parseErr == nil || !strings.Contains(parseErr.Error(), "write release size attribution") {
			parseT2.Fatalf("expected attribution write failure, got %v", parseErr)
		}
	})
}

// TestReleaseCollectPackageSizeAttributionCoversSortingAndRecordErrors verifies command, decode, sorting, and package record failures.
func TestReleaseCollectPackageSizeAttributionCoversSortingAndRecordErrors(parseT *testing.T) {
	parseOriginalRunCommand := releaseRunCommand
	parseT.Cleanup(func() {
		releaseRunCommand = parseOriginalRunCommand
	})

	parseT.Run("reports command and decode failures", func(parseT2 *testing.T) {
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return "", errors.New("go list failed")
		}
		if _, parseErr := releaseCollectPackageSizeAttribution(parseT2.TempDir()); parseErr == nil || !strings.Contains(parseErr.Error(), "collect release package attribution") {
			parseT2.Fatalf("expected go list failure, got %v", parseErr)
		}

		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return "{", nil
		}
		if _, parseErr := releaseCollectPackageSizeAttribution(parseT2.TempDir()); parseErr == nil || !strings.Contains(parseErr.Error(), "decode release package attribution") {
			parseT2.Fatalf("expected decode failure, got %v", parseErr)
		}
	})

	parseT.Run("sorts records and skips empty packages", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseAlphaDir := filepath.Join(parseRoot, "alpha")
		parseBetaDir := filepath.Join(parseRoot, "beta")
		parseZeroDir := filepath.Join(parseRoot, "zero")
		for _, parseDir := range []string{parseAlphaDir, parseBetaDir, parseZeroDir} {
			if parseErr := os.MkdirAll(parseDir, 0755); parseErr != nil {
				parseT2.Fatalf("mkdir package dir: %v", parseErr)
			}
		}
		parseAlphaArchive := filepath.Join(parseRoot, "alpha.a")
		if parseErr := os.WriteFile(parseAlphaArchive, []byte("archive-bytes"), 0644); parseErr != nil {
			parseT2.Fatalf("write alpha archive: %v", parseErr)
		}
		if parseErr := os.WriteFile(filepath.Join(parseAlphaDir, "alpha.go"), []byte("package alpha\n"), 0644); parseErr != nil {
			parseT2.Fatalf("write alpha.go: %v", parseErr)
		}
		if parseErr := os.WriteFile(filepath.Join(parseBetaDir, "beta.go"), []byte("package beta\n"), 0644); parseErr != nil {
			parseT2.Fatalf("write beta.go: %v", parseErr)
		}

		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			return buildReleasePackageJSONStream(parseT2, []releaseGoListPackage{
				{ImportPath: "example.com/zero", Dir: parseZeroDir},
				{ImportPath: "", Dir: parseRoot},
				{ImportPath: "example.com/beta", Dir: parseBetaDir, GoFiles: []string{"beta.go"}},
				{ImportPath: "example.com/alpha", Dir: parseAlphaDir, Export: parseAlphaArchive, GoFiles: []string{"alpha.go"}},
			}), nil
		}

		parsePackages, parseErr := releaseCollectPackageSizeAttribution(parseRoot)
		if parseErr != nil {
			parseT2.Fatalf("releaseCollectPackageSizeAttribution: %v", parseErr)
		}
		if len(parsePackages) != 2 {
			parseT2.Fatalf("expected two non-empty packages, got %#v", parsePackages)
		}
		if parsePackages[0].ImportPath != "example.com/alpha" || parsePackages[1].ImportPath != "example.com/beta" {
			parseT2.Fatalf("expected archive-heavy package to sort first, got %#v", parsePackages)
		}
	})

	parseT.Run("unions multiple entrypoint reachability graphs", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseAppADir := filepath.Join(parseRoot, "cmd", "app-a")
		parseAppBDir := filepath.Join(parseRoot, "cmd", "app-b")
		parseSharedDir := filepath.Join(parseRoot, "shared")
		parseAOnlyDir := filepath.Join(parseRoot, "aonly")
		parseBOnlyDir := filepath.Join(parseRoot, "bonly")
		for _, parseDir := range []string{parseAppADir, parseAppBDir, parseSharedDir, parseAOnlyDir, parseBOnlyDir} {
			if parseErr := os.MkdirAll(parseDir, 0755); parseErr != nil {
				parseT2.Fatalf("mkdir package dir: %v", parseErr)
			}
		}
		parseSharedArchiveA := filepath.Join(parseRoot, "shared-a.a")
		parseSharedArchiveB := filepath.Join(parseRoot, "shared-b.a")
		if parseErr := os.WriteFile(parseSharedArchiveA, []byte("shared"), 0644); parseErr != nil {
			parseT2.Fatalf("write shared archive a: %v", parseErr)
		}
		if parseErr := os.WriteFile(parseSharedArchiveB, []byte("shared-archive"), 0644); parseErr != nil {
			parseT2.Fatalf("write shared archive b: %v", parseErr)
		}
		if parseErr := os.WriteFile(filepath.Join(parseSharedDir, "shared.go"), []byte("package shared\n"), 0644); parseErr != nil {
			parseT2.Fatalf("write shared.go: %v", parseErr)
		}
		if parseErr := os.WriteFile(filepath.Join(parseAOnlyDir, "a.go"), []byte("package aonly\n"), 0644); parseErr != nil {
			parseT2.Fatalf("write a.go: %v", parseErr)
		}
		if parseErr := os.WriteFile(filepath.Join(parseBOnlyDir, "b.go"), []byte("package bonly\n"), 0644); parseErr != nil {
			parseT2.Fatalf("write b.go: %v", parseErr)
		}

		parseCalls := []string{}
		releaseRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			parseCalls = append(parseCalls, parseCwd)
			if parseCommand != "go" || strings.Join(parseArgs, " ") != "list -deps -json -export ." {
				parseT2.Fatalf("unexpected go list invocation: command=%q args=%v", parseCommand, parseArgs)
			}
			switch parseCwd {
			case parseAppADir:
				return buildReleasePackageJSONStream(parseT2, []releaseGoListPackage{
					{ImportPath: "example.com/app/shared", Dir: parseSharedDir, Export: parseSharedArchiveA, GoFiles: []string{"shared.go"}},
					{ImportPath: "example.com/app/aonly", Dir: parseAOnlyDir, GoFiles: []string{"a.go"}},
				}), nil
			case parseAppBDir:
				return buildReleasePackageJSONStream(parseT2, []releaseGoListPackage{
					{ImportPath: "example.com/app/shared", Dir: parseSharedDir, Export: parseSharedArchiveB, GoFiles: []string{"shared.go"}},
					{ImportPath: "example.com/app/bonly", Dir: parseBOnlyDir, GoFiles: []string{"b.go"}},
				}), nil
			default:
				parseT2.Fatalf("unexpected package dir %q", parseCwd)
				return "", nil
			}
		}

		parsePackages, parseErr := releaseCollectPackageSizeAttributionForEntrypoints([]string{parseAppADir, parseAppBDir, parseAppADir})
		if parseErr != nil {
			parseT2.Fatalf("releaseCollectPackageSizeAttributionForEntrypoints: %v", parseErr)
		}
		if strings.Join(parseCalls, "|") != strings.Join([]string{parseAppADir, parseAppBDir}, "|") {
			parseT2.Fatalf("expected one go list per unique entrypoint, got %v", parseCalls)
		}
		if len(parsePackages) != 3 {
			parseT2.Fatalf("expected union to contain three unique packages, got %#v", parsePackages)
		}
		parseByImportPath := map[string]releasePackageSizeRecord{}
		for _, parsePackage := range parsePackages {
			parseByImportPath[parsePackage.ImportPath] = parsePackage
		}
		if _, parseOk := parseByImportPath["example.com/app/aonly"]; !parseOk {
			parseT2.Fatalf("expected a-only package in union, got %#v", parsePackages)
		}
		if _, parseOk := parseByImportPath["example.com/app/bonly"]; !parseOk {
			parseT2.Fatalf("expected b-only package in union, got %#v", parsePackages)
		}
		parseShared := parseByImportPath["example.com/app/shared"]
		if parseShared.ArchiveBytes != int64(len("shared-archive")) {
			parseT2.Fatalf("expected duplicate shared package to keep richest archive size, got %#v", parseShared)
		}
	})

	parseT.Run("reports export and source stat failures", func(parseT2 *testing.T) {
		if _, parseErr := releaseBuildPackageSizeRecord(releaseGoListPackage{
			ImportPath: "example.com/missing-export",
			Dir:        parseT2.TempDir(),
			Export:     filepath.Join(parseT2.TempDir(), "missing.a"),
		}); parseErr == nil || !strings.Contains(parseErr.Error(), "inspect export archive") {
			parseT2.Fatalf("expected export stat failure, got %v", parseErr)
		}

		parseSourceDir := parseT2.TempDir()
		if _, parseErr := releaseBuildPackageSizeRecord(releaseGoListPackage{
			ImportPath: "example.com/missing-source",
			Dir:        parseSourceDir,
			GoFiles:    []string{"missing.go"},
		}); parseErr == nil || !strings.Contains(parseErr.Error(), "inspect source file") {
			parseT2.Fatalf("expected source stat failure, got %v", parseErr)
		}
	})
}

// buildReleasePackageJSONStream marshals sequential go list package records the same way `go list -json` emits them.
func buildReleasePackageJSONStream(parseT *testing.T, parsePackages []releaseGoListPackage) string {
	parseT.Helper()

	parseParts := make([]string, 0, len(parsePackages))
	for _, parsePackage := range parsePackages {
		parseBytes, parseErr := json.Marshal(parsePackage)
		if parseErr != nil {
			parseT.Fatalf("marshal go list package: %v", parseErr)
		}
		parseParts = append(parseParts, string(parseBytes))
	}
	return strings.Join(parseParts, "\n")
}
