package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

// TestLifecycleHelperBranchesCoverRootsPresetsModulesAndFeatures covers pure lifecycle helper branches.
func TestLifecycleHelperBranchesCoverRootsPresetsModulesAndFeatures(parseT *testing.T) {
	parseCWD, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("get cwd: %v", parseErr)
	}
	if parseRootPath, parseRootErr := parseLifecycleRootPath(""); parseRootErr != nil || parseRootPath != parseCWD {
		parseT.Fatalf("expected empty root path to resolve cwd %q, got path=%q err=%v", parseCWD, parseRootPath, parseRootErr)
	}

	parseTempRoot := parseT.TempDir()
	parseFilePath := filepath.Join(parseTempRoot, "root.txt")
	if parseWriteErr := os.WriteFile(parseFilePath, []byte("root"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write file root fixture: %v", parseWriteErr)
	}
	if _, parseRootErr := parseLifecycleRootPath(parseFilePath); parseRootErr == nil || !strings.Contains(parseRootErr.Error(), "not a directory") {
		parseT.Fatalf("expected non-directory root error, got %v", parseRootErr)
	}
	if _, parseRootErr := parseLifecycleRootPath(filepath.Join(parseTempRoot, "missing")); parseRootErr == nil || !strings.Contains(parseRootErr.Error(), "stat root path") {
		parseT.Fatalf("expected missing root stat error, got %v", parseRootErr)
	}

	if parsePreset, parseOK := parseLifecyclePreset(" MINIMAL-CLIENT "); !parseOK || parsePreset.Key != "minimal-client" {
		parseT.Fatalf("expected trimmed preset match, got preset=%#v ok=%t", parsePreset, parseOK)
	}
	if _, parseOK := parseLifecyclePreset("unknown-preset"); parseOK {
		parseT.Fatal("expected unknown preset lookup to fail")
	}

	if parseModulePath := parseLifecycleModulePath(parseTempRoot); parseModulePath != "" {
		parseT.Fatalf("expected missing go.mod to produce empty module path, got %q", parseModulePath)
	}
	parseNoModuleRoot := parseT.TempDir()
	if parseWriteErr := os.WriteFile(filepath.Join(parseNoModuleRoot, "go.mod"), []byte("go 1.25\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write no-module go.mod: %v", parseWriteErr)
	}
	if parseModulePath := parseLifecycleModulePath(parseNoModuleRoot); parseModulePath != "" {
		parseT.Fatalf("expected go.mod without module directive to produce empty module path, got %q", parseModulePath)
	}
	parseModuleRoot := parseT.TempDir()
	if parseWriteErr := os.WriteFile(filepath.Join(parseModuleRoot, "go.mod"), []byte("module example.com/lifecycle-fixture\n\ngo 1.25\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write module go.mod: %v", parseWriteErr)
	}
	if parseModulePath := parseLifecycleModulePath(parseModuleRoot); parseModulePath != "example.com/lifecycle-fixture" {
		parseT.Fatalf("expected module path to be parsed, got %q", parseModulePath)
	}

	parseEmptyMatrix := buildLifecycleFeatureMatrix(scaffoldMetadata{})
	if !strings.Contains(parseEmptyMatrix, "No preset features were recorded") {
		parseT.Fatalf("expected empty feature matrix fallback text, got %q", parseEmptyMatrix)
	}
	parseMatrix := buildLifecycleFeatureMatrix(scaffoldMetadata{
		Preset:     scaffoldPresetMetadata{Features: []string{"router", "router", "forms"}},
		Enterprise: scaffoldEnterpriseMetadata{Features: []string{"audit", "forms"}},
	})
	if strings.Count(parseMatrix, "`forms`") != 1 || !strings.Contains(parseMatrix, "- `audit`\n- `forms`\n- `router`\n") {
		parseT.Fatalf("expected deduplicated sorted feature matrix, got %q", parseMatrix)
	}
}

// TestLifecycleHelperBranchesCoverInitUpgradeAndRuntimeAssets covers init and upgrade branches that write runtime assets.
func TestLifecycleHelperBranchesCoverInitUpgradeAndRuntimeAssets(parseT *testing.T) {
	parseOriginalResolveWasmExecPath := scaffoldResolveWasmExecPath
	parseT.Cleanup(func() {
		scaffoldResolveWasmExecPath = parseOriginalResolveWasmExecPath
	})

	parseRootPath := parseT.TempDir()
	parseAppPath := filepath.Join(parseRootPath, "main.go")
	if parseWriteErr := os.WriteFile(parseAppPath, []byte("package main\n\nfunc main() {}\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr)
	}
	if parseWriteErr := os.WriteFile(filepath.Join(parseRootPath, "go.mod"), []byte("module example.com/lifecycle-upgrade\n\ngo 1.25\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
	}

	parseWasmExecSource := filepath.Join(parseT.TempDir(), "wasm_exec.js")
	if parseWriteErr := os.WriteFile(parseWasmExecSource, []byte("// wasm exec runtime\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write wasm_exec.js source: %v", parseWriteErr)
	}
	scaffoldResolveWasmExecPath = func() (string, error) {
		return parseWasmExecSource, nil
	}

	parseLauncher := launcher{}
	parseInitSummary, parseInitErr := parseLauncher.applyLifecycleInit(lifecycleInitConfig{
		rootPath:    parseRootPath,
		appPath:     parseAppPath,
		wasmPath:    "dist/app.wasm",
		modulePath:  "example.com/lifecycle-upgrade",
		projectName: "lifecycle-upgrade",
		projectMode: string(scaffoldProjectModeStandalone),
		presetKey:   "minimal-client",
	})
	if parseInitErr != nil {
		parseT.Fatalf("apply lifecycle init: %v", parseInitErr)
	}
	if parseInitSummary.RuntimeAssetPath == "" || !fileExists(parseInitSummary.RuntimeAssetPath) {
		parseT.Fatalf("expected runtime asset path from init, got %#v", parseInitSummary)
	}
	if parseRuntimeAssetBytes, parseReadErr := os.ReadFile(parseInitSummary.RuntimeAssetPath); parseReadErr != nil || string(parseRuntimeAssetBytes) != "// wasm exec runtime\n" {
		parseT.Fatalf("expected copied wasm runtime asset, got bytes=%q err=%v", string(parseRuntimeAssetBytes), parseReadErr)
	}
	if _, parseInitErr = parseLauncher.applyLifecycleInit(lifecycleInitConfig{
		rootPath:    parseRootPath,
		appPath:     parseAppPath,
		wasmPath:    "dist/app.wasm",
		modulePath:  "example.com/lifecycle-upgrade",
		projectName: "lifecycle-upgrade",
		projectMode: string(scaffoldProjectModeStandalone),
		presetKey:   "minimal-client",
	}); parseInitErr == nil || !strings.Contains(parseInitErr.Error(), "already exists") {
		parseT.Fatalf("expected existing metadata init error, got %v", parseInitErr)
	}

	parseMetadataPath := filepath.Join(parseRootPath, "gwc-start.json")
	parseMatrixPath := filepath.Join(parseRootPath, "FEATURE_MATRIX.md")
	if parseRemoveErr := os.Remove(parseMatrixPath); parseRemoveErr != nil {
		parseT.Fatalf("remove feature matrix: %v", parseRemoveErr)
	}
	parseMetadata := scaffoldMetadata{
		ProjectName: "lifecycle-upgrade",
		Tooling: scaffoldToolingMetadata{
			AppPath: "main.go",
		},
	}
	parseMetadataBytes, parseMarshalErr := json.MarshalIndent(parseMetadata, "", "  ")
	if parseMarshalErr != nil {
		parseT.Fatalf("marshal metadata fixture: %v", parseMarshalErr)
	}
	parseMetadataBytes = append(parseMetadataBytes, '\n')
	if parseWriteErr := os.WriteFile(parseMetadataPath, parseMetadataBytes, 0644); parseWriteErr != nil {
		parseT.Fatalf("write metadata fixture: %v", parseWriteErr)
	}
	if parseRemoveErr := os.Remove(filepath.Join(parseRootPath, "wasm_exec.js")); parseRemoveErr != nil {
		parseT.Fatalf("remove existing runtime asset: %v", parseRemoveErr)
	}

	parseUpgradeSummary, parseUpgradeErr := parseLauncher.applyLifecycleUpgrade(lifecycleUpgradeConfig{rootPath: parseRootPath})
	if parseUpgradeErr != nil {
		parseT.Fatalf("apply lifecycle upgrade: %v", parseUpgradeErr)
	}
	if parseUpgradeSummary.FeatureMatrixPath == "" || !fileExists(parseUpgradeSummary.FeatureMatrixPath) {
		parseT.Fatalf("expected upgrade to create feature matrix, got %#v", parseUpgradeSummary)
	}
	if parseUpgradeSummary.RuntimeAssetPath == "" || !fileExists(parseUpgradeSummary.RuntimeAssetPath) {
		parseT.Fatalf("expected upgrade to write runtime asset, got %#v", parseUpgradeSummary)
	}
	parseUpgradedMetadata, parseFoundMetadata, parseLoadErr := loadScaffoldMetadata(parseRootPath)
	if parseLoadErr != nil {
		parseT.Fatalf("load upgraded metadata: %v", parseLoadErr)
	}
	if !parseFoundMetadata {
		parseT.Fatal("expected upgraded metadata to exist")
	}
	if parseUpgradedMetadata.SchemaVersion != currentScaffoldMetadataSchemaVersion || strings.TrimSpace(parseUpgradedMetadata.Tooling.DevHost) == "" || strings.TrimSpace(parseUpgradedMetadata.Tooling.ReleaseCompression) == "" {
		parseT.Fatalf("expected upgrade defaults to be filled, got %#v", parseUpgradedMetadata)
	}
}

// TestLifecycleHelperBranchesCoverErrorsAndMigrationFindings covers remaining lifecycle error and migration helper branches.
func TestLifecycleHelperBranchesCoverErrorsAndMigrationFindings(parseT *testing.T) {
	parseRootPath := parseT.TempDir()
	parseAppPath := filepath.Join(parseRootPath, "main.go")
	if parseWriteErr := os.WriteFile(parseAppPath, []byte("package main\n\nfunc main() {}\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write main.go: %v", parseWriteErr)
	}
	if _, parseErr := parseLifecycleInitConfig(lifecycleInitConfig{
		rootPath:    parseRootPath,
		appPath:     parseAppPath,
		presetKey:   "unknown-preset",
		projectMode: string(scaffoldProjectModeStandalone),
	}); parseErr == nil || !strings.Contains(parseErr.Error(), "unknown init preset") {
		parseT.Fatalf("expected unknown preset init-config error, got %v", parseErr)
	}
	if _, parseErr := parseLifecycleInitConfig(lifecycleInitConfig{
		rootPath:    parseRootPath,
		appPath:     parseAppPath,
		presetKey:   "minimal-client",
		projectMode: "unknown-mode",
	}); parseErr == nil || !strings.Contains(parseErr.Error(), "unknown project mode") {
		parseT.Fatalf("expected unknown project-mode init-config error, got %v", parseErr)
	}
	if _, parseErr := parseLifecycleUpgradeConfig(lifecycleUpgradeConfig{rootPath: filepath.Join(parseRootPath, "missing")}); parseErr == nil {
		parseT.Fatal("expected invalid upgrade root to fail")
	}
	if _, parseErr := parseLifecycleMigrateConfig(lifecycleMigrateConfig{rootPath: filepath.Join(parseRootPath, "missing")}); parseErr == nil {
		parseT.Fatal("expected invalid migrate root to fail")
	}

	parseOriginalResolveWasmExecPath := scaffoldResolveWasmExecPath
	parseT.Cleanup(func() {
		scaffoldResolveWasmExecPath = parseOriginalResolveWasmExecPath
	})
	scaffoldResolveWasmExecPath = func() (string, error) {
		return "", os.ErrNotExist
	}
	if _, parseErr := applyLifecycleRuntimeAsset(parseRootPath); parseErr == nil || !strings.Contains(parseErr.Error(), "file does not exist") {
		parseT.Fatalf("expected runtime asset resolve error, got %v", parseErr)
	}
	scaffoldResolveWasmExecPath = func() (string, error) {
		return filepath.Join(parseRootPath, "missing-wasm-exec.js"), nil
	}
	if _, parseErr := applyLifecycleRuntimeAsset(parseRootPath); parseErr == nil || !strings.Contains(parseErr.Error(), "read wasm_exec.js") {
		parseT.Fatalf("expected runtime asset read error, got %v", parseErr)
	}

	parseGoModPath := filepath.Join(parseRootPath, "go.mod")
	if parseWriteErr := os.WriteFile(parseGoModPath, []byte("module example.com/migrate\n\ngo 1.25\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write go.mod: %v", parseWriteErr)
	}
	parseNestedDir := filepath.Join(parseRootPath, "internal")
	if parseMkdirErr := os.MkdirAll(parseNestedDir, 0755); parseMkdirErr != nil {
		parseT.Fatalf("mkdir nested dir: %v", parseMkdirErr)
	}
	if parseWriteErr := os.WriteFile(filepath.Join(parseRootPath, "b_routes.go"), []byte("package main\n// GoRegisterRoute(\"/b\", nil)\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write b_routes.go: %v", parseWriteErr)
	}
	if parseWriteErr := os.WriteFile(filepath.Join(parseNestedDir, "a_routes.go"), []byte("package internal\n// GoGetRoute()\n// GoRegisterRoute(\"/a\", nil)\n"), 0644); parseWriteErr != nil {
		parseT.Fatalf("write a_routes.go: %v", parseWriteErr)
	}

	parseFindings, parseFindingsErr := buildLifecycleMigrateFindings(parseRootPath)
	if parseFindingsErr != nil {
		parseT.Fatalf("build migrate findings: %v", parseFindingsErr)
	}
	if len(parseFindings) != 3 {
		parseT.Fatalf("expected three migration findings, got %#v", parseFindings)
	}
	if parseFindings[0].Path != "b_routes.go" || parseFindings[1].Path != filepath.ToSlash(filepath.Join("internal", "a_routes.go")) || parseFindings[2].LegacyCall != "GoRegisterRoute(" {
		parseT.Fatalf("expected sorted migration findings, got %#v", parseFindings)
	}

	parseLauncher := launcher{}
	if _, parseErr := parseLauncher.applyLifecycleUpgrade(lifecycleUpgradeConfig{rootPath: parseT.TempDir()}); parseErr == nil || !strings.Contains(parseErr.Error(), "run `gwc init` first") {
		parseT.Fatalf("expected upgrade without metadata to fail, got %v", parseErr)
	}
	if _, parseErr := parseLauncher.applyLifecycleMigrate(lifecycleMigrateConfig{rootPath: parseT.TempDir()}); parseErr == nil || !strings.Contains(parseErr.Error(), "run `gwc init` first") {
		parseT.Fatalf("expected migrate without metadata to fail, got %v", parseErr)
	}
}
