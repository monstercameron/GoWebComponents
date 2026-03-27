package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestWasmHelpersResetDirectoryAndEnv verifies directory reset and env restore branches.
func TestWasmHelpersResetDirectoryAndEnv(parseT *testing.T) {
	if parseErr := resetWasmDirectory(""); parseErr == nil || !strings.Contains(parseErr.Error(), "empty directory path") {
		parseT.Fatalf("expected empty path error, got %v", parseErr)
	}

	parseRoot := parseT.TempDir()
	parseDir := filepath.Join(parseRoot, "cache")
	if parseErr := os.MkdirAll(filepath.Join(parseDir, "nested"), 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll nested: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseDir, "nested", "artifact.txt"), []byte("stale"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile artifact: %v", parseErr2)
	}
	if parseErr3 := resetWasmDirectory(parseDir); parseErr3 != nil {
		parseT.Fatalf("resetWasmDirectory: %v", parseErr3)
	}
	parseEntries, parseErr4 := os.ReadDir(parseDir)
	if parseErr4 != nil {
		parseT.Fatalf("ReadDir cache: %v", parseErr4)
	}
	if len(parseEntries) != 0 {
		parseT.Fatalf("expected empty reset directory, got %d entries", len(parseEntries))
	}

	parseEnvName := "GWC_WASM_HELPER_ENV"
	parseOriginalValue, hasParseOriginal := os.LookupEnv(parseEnvName)
	parseT.Cleanup(func() {
		if hasParseOriginal {
			_ = os.Setenv(parseEnvName, parseOriginalValue)
			return
		}
		_ = os.Unsetenv(parseEnvName)
	})

	_ = os.Unsetenv(parseEnvName)
	parseRestore := setWasmEnv(parseEnvName, "warm")
	if parseGot := os.Getenv(parseEnvName); parseGot != "warm" {
		parseT.Fatalf("setWasmEnv without prior value = %q, want warm", parseGot)
	}
	parseRestore()
	if _, hasParseValue := os.LookupEnv(parseEnvName); hasParseValue {
		parseT.Fatalf("expected env to be unset after restore")
	}

	if parseErr5 := os.Setenv(parseEnvName, "cold"); parseErr5 != nil {
		parseT.Fatalf("Setenv original: %v", parseErr5)
	}
	parseRestore = setWasmEnv(parseEnvName, "hot")
	if parseGot := os.Getenv(parseEnvName); parseGot != "hot" {
		parseT.Fatalf("setWasmEnv with prior value = %q, want hot", parseGot)
	}
	parseRestore()
	if parseGot := os.Getenv(parseEnvName); parseGot != "cold" {
		parseT.Fatalf("restored env = %q, want cold", parseGot)
	}
}

// TestWasmHelpersResolveEditableFileAndSmallEdit verifies editable file selection and temporary edits.
func TestWasmHelpersResolveEditableFileAndSmallEdit(parseT *testing.T) {
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

	parsePackageDir := filepath.Join(parseRoot, "pkg")
	if parseErr4 := os.MkdirAll(parsePackageDir, 0o755); parseErr4 != nil {
		parseT.Fatalf("MkdirAll package: %v", parseErr4)
	}
	parseAlphaPath := filepath.Join(parsePackageDir, "alpha.go")
	parseAlphaContent := "package pkg\n\nfunc Value() int { return 1 }\n"
	if parseErr5 := os.WriteFile(parseAlphaPath, []byte(parseAlphaContent), 0o644); parseErr5 != nil {
		parseT.Fatalf("WriteFile alpha.go: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(filepath.Join(parsePackageDir, "zeta.go"), []byte("package pkg\n"), 0o644); parseErr6 != nil {
		parseT.Fatalf("WriteFile zeta.go: %v", parseErr6)
	}
	if parseErr7 := os.WriteFile(filepath.Join(parsePackageDir, "alpha_test.go"), []byte("package pkg\n"), 0o644); parseErr7 != nil {
		parseT.Fatalf("WriteFile alpha_test.go: %v", parseErr7)
	}
	if parseErr8 := os.WriteFile(filepath.Join(parsePackageDir, "notes.txt"), []byte("ignore"), 0o644); parseErr8 != nil {
		parseT.Fatalf("WriteFile notes.txt: %v", parseErr8)
	}

	parseEditablePath, parseErr9 := resolveWasmCacheEditableFile("./pkg")
	if parseErr9 != nil {
		parseT.Fatalf("resolveWasmCacheEditableFile: %v", parseErr9)
	}
	if parseEditablePath != parseAlphaPath {
		parseT.Fatalf("editable path = %q, want %q", parseEditablePath, parseAlphaPath)
	}

	parseRestore, parseEditedFile, parseErr10 := applyWasmSmallEdit("./pkg", true)
	if parseErr10 != nil {
		parseT.Fatalf("applyWasmSmallEdit enabled: %v", parseErr10)
	}
	if parseEditedFile != parseAlphaPath {
		parseT.Fatalf("edited file = %q, want %q", parseEditedFile, parseAlphaPath)
	}
	parseUpdatedContent, parseErr11 := os.ReadFile(parseAlphaPath)
	if parseErr11 != nil {
		parseT.Fatalf("ReadFile edited source: %v", parseErr11)
	}
	if !strings.Contains(string(parseUpdatedContent), "cache experiment marker") {
		parseT.Fatalf("expected temporary marker in edited file, got %q", string(parseUpdatedContent))
	}
	parseRestore()
	parseRestoredContent, parseErr12 := os.ReadFile(parseAlphaPath)
	if parseErr12 != nil {
		parseT.Fatalf("ReadFile restored source: %v", parseErr12)
	}
	if string(parseRestoredContent) != parseAlphaContent {
		parseT.Fatalf("restored content mismatch: %q", string(parseRestoredContent))
	}

	parseRestore, parseEditedFile, parseErr10 = applyWasmSmallEdit("./pkg", false)
	if parseErr10 != nil {
		parseT.Fatalf("applyWasmSmallEdit disabled: %v", parseErr10)
	}
	if parseRestore != nil || parseEditedFile != "" {
		parseT.Fatalf("expected disabled small edit to be a no-op, got restore=%v file=%q", parseRestore != nil, parseEditedFile)
	}

	if _, parseErr13 := resolveWasmCacheEditableFile(parseAlphaPath); parseErr13 == nil || !strings.Contains(parseErr13.Error(), "requires a package directory path") {
		parseT.Fatalf("expected package directory error, got %v", parseErr13)
	}

	parseEmptyDir := filepath.Join(parseRoot, "empty")
	if parseErr14 := os.MkdirAll(parseEmptyDir, 0o755); parseErr14 != nil {
		parseT.Fatalf("MkdirAll empty: %v", parseErr14)
	}
	if _, parseErr15 := resolveWasmCacheEditableFile("./empty"); parseErr15 == nil || !strings.Contains(parseErr15.Error(), "no editable Go source file found") {
		parseT.Fatalf("expected no editable file error, got %v", parseErr15)
	}
}

// TestWasmHelpersParityChecks verifies file and artifact parity success and failure branches.
func TestWasmHelpersParityChecks(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseSourcePath := filepath.Join(parseRoot, "source.wasm")
	parseTargetPath := filepath.Join(parseRoot, "target.wasm")
	if parseErr := os.WriteFile(parseSourcePath, []byte("wasm-bytes"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile source: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseTargetPath, []byte("wasm-bytes"), 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile target: %v", parseErr2)
	}
	if parseErr3 := assertWasmFileParity(parseSourcePath, parseTargetPath); parseErr3 != nil {
		parseT.Fatalf("assertWasmFileParity success: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseTargetPath, []byte("changed"), 0o644); parseErr4 != nil {
		parseT.Fatalf("WriteFile mismatch target: %v", parseErr4)
	}
	if parseErr5 := assertWasmFileParity(parseSourcePath, parseTargetPath); parseErr5 == nil || !strings.Contains(parseErr5.Error(), "parity mismatch") {
		parseT.Fatalf("expected parity mismatch, got %v", parseErr5)
	}
	if parseErr6 := assertWasmFileParity(filepath.Join(parseRoot, "missing-source.wasm"), parseTargetPath); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "read source artifact") {
		parseT.Fatalf("expected source read error, got %v", parseErr6)
	}
	if parseErr7 := assertWasmFileParity(parseSourcePath, filepath.Join(parseRoot, "missing-target.wasm")); parseErr7 == nil || !strings.Contains(parseErr7.Error(), "read target artifact") {
		parseT.Fatalf("expected target read error, got %v", parseErr7)
	}

	parseArtifact := releaseArtifactRecord{Path: "app.wasm", Bytes: 8, SHA256: "abc123"}
	if parseErr8 := assertWasmArtifactParity("optimized", parseArtifact, parseArtifact); parseErr8 != nil {
		parseT.Fatalf("assertWasmArtifactParity success: %v", parseErr8)
	}
	if parseErr9 := assertWasmArtifactParity("optimized", parseArtifact, releaseArtifactRecord{Path: "app.wasm", Bytes: 7, SHA256: "abc123"}); parseErr9 == nil || !strings.Contains(parseErr9.Error(), "bytes") {
		parseT.Fatalf("expected bytes mismatch error, got %v", parseErr9)
	}
	if parseErr10 := assertWasmArtifactParity("optimized", parseArtifact, releaseArtifactRecord{Path: "app.wasm", Bytes: 8, SHA256: ""}); parseErr10 == nil || !strings.Contains(parseErr10.Error(), "non-empty sha256") {
		parseT.Fatalf("expected empty sha error, got %v", parseErr10)
	}
	if parseErr11 := assertWasmArtifactParity("optimized", parseArtifact, releaseArtifactRecord{Path: "app.wasm", Bytes: 8, SHA256: "xyz789"}); parseErr11 == nil || !strings.Contains(parseErr11.Error(), "sha256") {
		parseT.Fatalf("expected sha mismatch error, got %v", parseErr11)
	}
}

// TestWasmMetricHelpers verifies metric categorization and threshold selection.
func TestWasmMetricHelpers(parseT *testing.T) {
	parseConfig := wasmCompareConfig{
		timingRegressionPercent: 15,
		sizeRegressionPercent:   7,
		otherRegressionPercent:  3,
	}

	if parseCategory := wasmMetricCategory("phases.go_build_ms"); parseCategory != "timing" {
		parseT.Fatalf("wasmMetricCategory timing = %q, want timing", parseCategory)
	}
	if parseCategory := wasmMetricCategory("variants.module_download_ms"); parseCategory != "timing" {
		parseT.Fatalf("wasmMetricCategory module download = %q, want timing", parseCategory)
	}
	if parseCategory := wasmMetricCategory("artifacts.wasm.bytes"); parseCategory != "size" {
		parseT.Fatalf("wasmMetricCategory size = %q, want size", parseCategory)
	}
	if parseCategory := wasmMetricCategory("counts.total"); parseCategory != "other" {
		parseT.Fatalf("wasmMetricCategory other = %q, want other", parseCategory)
	}

	if parseThreshold := wasmMetricThreshold("phases.go_build_ms", parseConfig); parseThreshold != 15 {
		parseT.Fatalf("wasmMetricThreshold timing = %v, want 15", parseThreshold)
	}
	if parseThreshold := wasmMetricThreshold("artifacts.wasm.bytes", parseConfig); parseThreshold != 7 {
		parseT.Fatalf("wasmMetricThreshold size = %v, want 7", parseThreshold)
	}
	if parseThreshold := wasmMetricThreshold("counts.total", parseConfig); parseThreshold != 3 {
		parseT.Fatalf("wasmMetricThreshold other = %v, want 3", parseThreshold)
	}
}

// TestExecuteWasmCacheVariantCoversFailureBranches verifies direct cache-variant failure paths.
func TestExecuteWasmCacheVariantCoversFailureBranches(parseT *testing.T) {
	parseOriginalRunCommand := wasmRunCommand
	parseT.Cleanup(func() {
		wasmRunCommand = parseOriginalRunCommand
	})

	parseT.Run("module download failure", func(parseT *testing.T) {
		wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			if parseCommand == "go" && len(parseArgs) == 2 && parseArgs[0] == "mod" && parseArgs[1] == "download" {
				return "", os.ErrPermission
			}
			parseT.Fatalf("unexpected command %q args %#v", parseCommand, parseArgs)
			return "", nil
		}

		parseResult, parseErr := executeWasmCacheVariant(wasmCacheConfig{
			packagePath: ".",
			outDir:      parseT.TempDir(),
			binaryName:  "app.wasm",
		}, wasmCacheVariantSpec{
			key:                "ci-style-cold",
			label:              "ci style cold",
			goCache:            filepath.Join(parseT.TempDir(), "gocache"),
			goModCache:         filepath.Join(parseT.TempDir(), "gomodcache"),
			note:               "Fresh caches.",
			prepareModuleCache: true,
		})
		if parseErr != nil {
			parseT.Fatalf("executeWasmCacheVariant module download failure: %v", parseErr)
		}
		if parseResult.Status != "failed" || parseResult.Error != "go mod download failed" {
			parseT.Fatalf("unexpected module download failure result: %#v", parseResult)
		}
		if parseResult.ModuleDownloadMS == nil {
			parseT.Fatalf("expected module download timing in failure result")
		}
	})

	parseT.Run("small edit failure", func(parseT *testing.T) {
		wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			parseT.Fatalf("unexpected command %q args %#v", parseCommand, parseArgs)
			return "", nil
		}

		parsePackageDir := parseT.TempDir()
		parseResult, parseErr := executeWasmCacheVariant(wasmCacheConfig{
			packagePath: parsePackageDir,
			outDir:      parseT.TempDir(),
			binaryName:  "app.wasm",
		}, wasmCacheVariantSpec{
			key:        "shared-cache-small-edit",
			label:      "shared cache small edit",
			goCache:    filepath.Join(parseT.TempDir(), "gocache"),
			goModCache: filepath.Join(parseT.TempDir(), "gomodcache"),
			note:       "Small edit rebuild.",
			smallEdit:  true,
		})
		if parseErr != nil {
			parseT.Fatalf("executeWasmCacheVariant small edit failure: %v", parseErr)
		}
		if parseResult.Status != "failed" || !strings.Contains(parseResult.Error, "no editable Go source file found") {
			parseT.Fatalf("unexpected small edit failure result: %#v", parseResult)
		}
	})

	parseT.Run("measurement failure", func(parseT *testing.T) {
		wasmRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
			if len(parseArgs) > 0 && parseArgs[0] == "build" {
				return "", os.ErrInvalid
			}
			parseT.Fatalf("unexpected command %q args %#v", parseCommand, parseArgs)
			return "", nil
		}

		parseResult, parseErr := executeWasmCacheVariant(wasmCacheConfig{
			packagePath: ".",
			outDir:      parseT.TempDir(),
			binaryName:  "app.wasm",
		}, wasmCacheVariantSpec{
			key:        "shared-cache-cold",
			label:      "shared cache cold",
			goCache:    filepath.Join(parseT.TempDir(), "gocache"),
			goModCache: filepath.Join(parseT.TempDir(), "gomodcache"),
			note:       "Cold build.",
		})
		if parseErr != nil {
			parseT.Fatalf("executeWasmCacheVariant measurement failure: %v", parseErr)
		}
		if parseResult.Status != "failed" || parseResult.Error != "wasm measurement failed" {
			parseT.Fatalf("unexpected measurement failure result: %#v", parseResult)
		}
	})
}
