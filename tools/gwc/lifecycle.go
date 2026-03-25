package main

import (
	"bufio"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"sort"
	"strings"
)

var runInitCommand = func(l launcher, args []string) error {
	return l.runInit(args)
}

var runUpgradeCommand = func(l launcher, args []string) error {
	return l.runUpgrade(args)
}

var runMigrateCommand = func(l launcher, args []string) error {
	return l.runMigrate(args)
}

type lifecycleInitConfig struct {
	rootPath          string
	appPath           string
	htmlPath          string
	wasmPath          string
	modulePath        string
	projectName       string
	projectMode       string
	presetKey         string
	skipRuntimeAssets bool
	forceMetadata     bool
	json              bool
}

type lifecycleUpgradeConfig struct {
	rootPath          string
	skipRuntimeAssets bool
	json              bool
}

type lifecycleMigrateConfig struct {
	rootPath          string
	skipRuntimeAssets bool
	json              bool
}

type lifecycleInitSummary struct {
	OK                bool   `json:"ok"`
	Root              string `json:"root"`
	MetadataPath      string `json:"metadataPath"`
	FeatureMatrixPath string `json:"featureMatrixPath"`
	RuntimeAssetPath  string `json:"runtimeAssetPath,omitempty"`
	AppPath           string `json:"appPath"`
	HTMLPath          string `json:"htmlPath,omitempty"`
	WASMPath          string `json:"wasmPath"`
}

type lifecycleUpgradeSummary struct {
	OK                bool   `json:"ok"`
	Root              string `json:"root"`
	MetadataPath      string `json:"metadataPath"`
	FeatureMatrixPath string `json:"featureMatrixPath,omitempty"`
	RuntimeAssetPath  string `json:"runtimeAssetPath,omitempty"`
	SchemaVersion     int    `json:"schemaVersion"`
}

type lifecycleMigrateSummary struct {
	OK                bool                      `json:"ok"`
	Root              string                    `json:"root"`
	MetadataPath      string                    `json:"metadataPath"`
	FeatureMatrixPath string                    `json:"featureMatrixPath,omitempty"`
	RuntimeAssetPath  string                    `json:"runtimeAssetPath,omitempty"`
	ReportPath        string                    `json:"reportPath"`
	FindingCount      int                       `json:"findingCount"`
	Findings          []lifecycleMigrateFinding `json:"findings,omitempty"`
}

type lifecycleMigrateFinding struct {
	Path           string `json:"path"`
	Line           int    `json:"line"`
	LegacyCall     string `json:"legacyCall"`
	Replacement    string `json:"replacement"`
	Recommendation string `json:"recommendation"`
}

// runInit executes a non-interactive project initialization flow.
func (parseL launcher) runInit(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("init", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to initialize; defaults to the current working directory")
	parseApp := parseFlags.String("app", "", "App entrypoint path relative to -root; defaults to main.go or cmd/web/main.go")
	parseHTML := parseFlags.String("html", "", "HTML shell path relative to -root; defaults to index.html if present")
	parseWASM := parseFlags.String("wasm", "", "WASM output path recorded in gwc-start.json")
	parseModule := parseFlags.String("module", "", "Go module path; defaults to existing go.mod module path when available")
	parseName := parseFlags.String("name", "", "Project name recorded in gwc-start.json")
	parseMode := parseFlags.String("mode", string(scaffoldProjectModeStandalone), "Project mode: standalone or contributor-linked")
	parsePreset := parseFlags.String("preset", "minimal-client", "Starter preset metadata label stored in gwc-start.json")
	parseSkipRuntime := parseFlags.Bool("skip-runtime-assets", false, "Skip writing wasm_exec.js into the project root")
	parseForce := parseFlags.Bool("force", false, "Overwrite existing gwc-start.json when present")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON summary")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr := parseLifecycleInitConfig(lifecycleInitConfig{
		rootPath:          *parseRoot,
		appPath:           *parseApp,
		htmlPath:          *parseHTML,
		wasmPath:          *parseWASM,
		modulePath:        *parseModule,
		projectName:       *parseName,
		projectMode:       *parseMode,
		presetKey:         *parsePreset,
		skipRuntimeAssets: *parseSkipRuntime,
		forceMetadata:     *parseForce,
		json:              *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	applySummary, applyErr := parseL.applyLifecycleInit(parseConfig)
	if applyErr != nil {
		return applyErr
	}
	if parseConfig.json {
		renderEncoder := json.NewEncoder(os.Stdout)
		renderEncoder.SetIndent("", "  ")
		return renderEncoder.Encode(applySummary)
	}
	renderLifecycleInitSummary(applySummary)
	return nil
}

// runUpgrade executes a non-interactive metadata and runtime asset upgrade flow.
func (parseL launcher) runUpgrade(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("upgrade", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to upgrade; defaults to the current working directory")
	parseSkipRuntime := parseFlags.Bool("skip-runtime-assets", false, "Skip writing wasm_exec.js into the project root")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON summary")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr := parseLifecycleUpgradeConfig(lifecycleUpgradeConfig{
		rootPath:          *parseRoot,
		skipRuntimeAssets: *parseSkipRuntime,
		json:              *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	applySummary, applyErr := parseL.applyLifecycleUpgrade(parseConfig)
	if applyErr != nil {
		return applyErr
	}
	if parseConfig.json {
		renderEncoder := json.NewEncoder(os.Stdout)
		renderEncoder.SetIndent("", "  ")
		return renderEncoder.Encode(applySummary)
	}
	renderLifecycleUpgradeSummary(applySummary)
	return nil
}

// runMigrate executes a non-interactive migration helper flow with legacy API findings.
func (parseL launcher) runMigrate(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("migrate", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to migrate; defaults to the current working directory")
	parseSkipRuntime := parseFlags.Bool("skip-runtime-assets", false, "Skip writing wasm_exec.js into the project root")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON summary")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr := parseLifecycleMigrateConfig(lifecycleMigrateConfig{
		rootPath:          *parseRoot,
		skipRuntimeAssets: *parseSkipRuntime,
		json:              *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	applySummary, applyErr := parseL.applyLifecycleMigrate(parseConfig)
	if applyErr != nil {
		return applyErr
	}
	if parseConfig.json {
		renderEncoder := json.NewEncoder(os.Stdout)
		renderEncoder.SetIndent("", "  ")
		return renderEncoder.Encode(applySummary)
	}
	renderLifecycleMigrateSummary(applySummary)
	return nil
}

// parseLifecycleInitConfig resolves and validates init command configuration.
func parseLifecycleInitConfig(parseConfig lifecycleInitConfig) (lifecycleInitConfig, error) {
	parseRootPath, parseErr := parseLifecycleRootPath(parseConfig.rootPath)
	if parseErr != nil {
		return lifecycleInitConfig{}, parseErr
	}
	parsePreset, parsePresetOK := parseLifecyclePreset(parseConfig.presetKey)
	if !parsePresetOK {
		return lifecycleInitConfig{}, fmt.Errorf("unknown init preset %q", parseConfig.presetKey)
	}
	parseMode, parseModeOK := normalizeScaffoldProjectMode(parseConfig.projectMode)
	if !parseModeOK {
		return lifecycleInitConfig{}, fmt.Errorf("unknown project mode %q", parseConfig.projectMode)
	}

	parseAppPath := strings.TrimSpace(parseConfig.appPath)
	if parseAppPath == "" {
		parseDetectedApp, parseDetectErr := detectAppPath(parseRootPath)
		if parseDetectErr != nil {
			return lifecycleInitConfig{}, fmt.Errorf("detect app path for init: %w", parseDetectErr)
		}
		parseAppPath = parseDetectedApp
	}
	parseAppPath, parseErr = normalizeExistingPath(parseRootPath, parseAppPath)
	if parseErr != nil {
		return lifecycleInitConfig{}, fmt.Errorf("resolve init app path: %w", parseErr)
	}

	parseHTMLPath := strings.TrimSpace(parseConfig.htmlPath)
	if parseHTMLPath == "" {
		parseHTMLPath = detectHTMLPath(parseRootPath)
	}
	if parseHTMLPath != "" {
		parseHTMLPath, parseErr = normalizeExistingPath(parseRootPath, parseHTMLPath)
		if parseErr != nil {
			return lifecycleInitConfig{}, fmt.Errorf("resolve init html path: %w", parseErr)
		}
	}

	parseModulePath := strings.TrimSpace(parseConfig.modulePath)
	if parseModulePath == "" {
		parseModulePath = parseLifecycleModulePath(parseRootPath)
	}
	if parseModulePath == "" {
		parseModulePath = "example.com/" + filepath.Base(parseRootPath)
	}
	parseProjectName := strings.TrimSpace(parseConfig.projectName)
	if parseProjectName == "" {
		parseProjectName = filepath.Base(parseRootPath)
	}
	parseWASMPath := strings.TrimSpace(parseConfig.wasmPath)
	if parseWASMPath == "" {
		parseWASMPath = filepath.ToSlash(scaffoldWASMOutputPath())
	}

	return lifecycleInitConfig{
		rootPath:          parseRootPath,
		appPath:           parseAppPath,
		htmlPath:          parseHTMLPath,
		wasmPath:          filepath.ToSlash(parseWASMPath),
		modulePath:        parseModulePath,
		projectName:       parseProjectName,
		projectMode:       string(parseMode),
		presetKey:         parsePreset.Key,
		skipRuntimeAssets: parseConfig.skipRuntimeAssets,
		forceMetadata:     parseConfig.forceMetadata,
		json:              parseConfig.json,
	}, nil
}

// parseLifecycleUpgradeConfig resolves and validates upgrade command configuration.
func parseLifecycleUpgradeConfig(parseConfig lifecycleUpgradeConfig) (lifecycleUpgradeConfig, error) {
	parseRootPath, parseErr := parseLifecycleRootPath(parseConfig.rootPath)
	if parseErr != nil {
		return lifecycleUpgradeConfig{}, parseErr
	}
	return lifecycleUpgradeConfig{
		rootPath:          parseRootPath,
		skipRuntimeAssets: parseConfig.skipRuntimeAssets,
		json:              parseConfig.json,
	}, nil
}

// parseLifecycleMigrateConfig resolves and validates migrate command configuration.
func parseLifecycleMigrateConfig(parseConfig lifecycleMigrateConfig) (lifecycleMigrateConfig, error) {
	parseRootPath, parseErr := parseLifecycleRootPath(parseConfig.rootPath)
	if parseErr != nil {
		return lifecycleMigrateConfig{}, parseErr
	}
	return lifecycleMigrateConfig{
		rootPath:          parseRootPath,
		skipRuntimeAssets: parseConfig.skipRuntimeAssets,
		json:              parseConfig.json,
	}, nil
}

// parseLifecycleRootPath resolves a command root path and verifies it is a directory.
func parseLifecycleRootPath(parseRootPath string) (string, error) {
	parseRootPath = strings.TrimSpace(parseRootPath)
	if parseRootPath == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return "", fmt.Errorf("resolve root from cwd: %w", parseErr)
		}
		parseRootPath = parseCWD
	}
	parseAbsoluteRoot, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return "", fmt.Errorf("resolve root path: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbsoluteRoot)
	if parseErr != nil {
		return "", fmt.Errorf("stat root path: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		return "", fmt.Errorf("root path is not a directory: %s", parseAbsoluteRoot)
	}
	return parseAbsoluteRoot, nil
}

// parseLifecyclePreset resolves a preset key from launcher defaults.
func parseLifecyclePreset(parseKey string) (startPreset, bool) {
	parseKey = strings.ToLower(strings.TrimSpace(parseKey))
	for _, parsePreset := range defaultStartPresets() {
		if strings.ToLower(strings.TrimSpace(parsePreset.Key)) == parseKey {
			return parsePreset, true
		}
	}
	return startPreset{}, false
}

// parseLifecycleModulePath reads a module path from the project go.mod when present.
func parseLifecycleModulePath(parseRootPath string) string {
	parseGoModPath := filepath.Join(parseRootPath, "go.mod")
	parseContent, parseErr := os.ReadFile(parseGoModPath)
	if parseErr != nil {
		return ""
	}
	parseScanner := bufio.NewScanner(strings.NewReader(string(parseContent)))
	for parseScanner.Scan() {
		parseLine := strings.TrimSpace(parseScanner.Text())
		if strings.HasPrefix(parseLine, "module ") {
			return strings.TrimSpace(strings.TrimPrefix(parseLine, "module "))
		}
	}
	return ""
}

// applyLifecycleInit writes non-interactive starter metadata and baseline project artifacts.
func (parseL launcher) applyLifecycleInit(applyConfig lifecycleInitConfig) (lifecycleInitSummary, error) {
	applyMetadataPath := filepath.Join(applyConfig.rootPath, "gwc-start.json")
	if !applyConfig.forceMetadata && fileExists(applyMetadataPath) {
		return lifecycleInitSummary{}, fmt.Errorf("gwc-start.json already exists at %s; rerun with -force to overwrite", applyMetadataPath)
	}

	applyPreset, _ := parseLifecyclePreset(applyConfig.presetKey)
	applyMode, _ := normalizeScaffoldProjectMode(applyConfig.projectMode)
	applySelection := startSelection{
		Preset:      applyPreset,
		ProjectMode: applyMode,
		ProjectName: applyConfig.projectName,
		ModulePath:  applyConfig.modulePath,
		Author:      "unknown",
		Version:     "0.1.0",
		Description: fmt.Sprintf("%s initialized with gwc init.", applyConfig.projectName),
		TargetDir:   applyConfig.rootPath,
	}
	applyMetadata := defaultScaffoldMetadata(applySelection)
	applyMetadata.TargetDir = applyConfig.rootPath
	applyMetadata.Tooling.AppPath = applyLifecycleRelativePath(applyConfig.rootPath, applyConfig.appPath)
	applyMetadata.Tooling.HTMLPath = applyLifecycleRelativePath(applyConfig.rootPath, applyConfig.htmlPath)
	applyMetadata.Tooling.WASMPath = filepath.ToSlash(applyConfig.wasmPath)
	if applyErr := applyLifecycleMetadata(applyMetadataPath, applyMetadata); applyErr != nil {
		return lifecycleInitSummary{}, applyErr
	}

	applyFeatureMatrixPath := filepath.Join(applyConfig.rootPath, "FEATURE_MATRIX.md")
	applyFeatureMatrix := renderScaffoldFeatureMatrix(applySelection)
	if applyErr := os.WriteFile(applyFeatureMatrixPath, []byte(applyFeatureMatrix), 0644); applyErr != nil {
		return lifecycleInitSummary{}, fmt.Errorf("write FEATURE_MATRIX.md: %w", applyErr)
	}
	applyRuntimeAssetPath := ""
	if !applyConfig.skipRuntimeAssets {
		applyPath, applyErr := applyLifecycleRuntimeAsset(applyConfig.rootPath)
		if applyErr != nil {
			return lifecycleInitSummary{}, applyErr
		}
		applyRuntimeAssetPath = applyPath
	}

	return lifecycleInitSummary{
		OK:                true,
		Root:              applyConfig.rootPath,
		MetadataPath:      applyMetadataPath,
		FeatureMatrixPath: applyFeatureMatrixPath,
		RuntimeAssetPath:  applyRuntimeAssetPath,
		AppPath:           applyConfig.appPath,
		HTMLPath:          applyConfig.htmlPath,
		WASMPath:          filepath.ToSlash(applyConfig.wasmPath),
	}, nil
}

// applyLifecycleUpgrade upgrades metadata schema, defaults, and runtime assets for an existing project.
func (parseL launcher) applyLifecycleUpgrade(applyConfig lifecycleUpgradeConfig) (lifecycleUpgradeSummary, error) {
	applyMetadataPath := filepath.Join(applyConfig.rootPath, "gwc-start.json")
	applyMetadata, applyFound, applyErr := loadScaffoldMetadata(applyConfig.rootPath)
	if applyErr != nil {
		return lifecycleUpgradeSummary{}, applyErr
	}
	if !applyFound {
		return lifecycleUpgradeSummary{}, fmt.Errorf("gwc-start.json not found under %s; run `gwc init` first", applyConfig.rootPath)
	}
	if strings.TrimSpace(applyMetadata.TargetDir) == "" {
		applyMetadata.TargetDir = applyConfig.rootPath
	}
	if strings.TrimSpace(applyMetadata.ProjectName) == "" {
		applyMetadata.ProjectName = filepath.Base(applyConfig.rootPath)
	}
	if strings.TrimSpace(applyMetadata.ModulePath) == "" {
		applyMetadata.ModulePath = parseLifecycleModulePath(applyConfig.rootPath)
	}
	if strings.TrimSpace(applyMetadata.Tooling.AppPath) == "" {
		applyDetectedApp, parseDetectErr := detectAppPath(applyConfig.rootPath)
		if parseDetectErr == nil {
			applyMetadata.Tooling.AppPath = applyLifecycleRelativePath(applyConfig.rootPath, applyDetectedApp)
		}
	}
	if strings.TrimSpace(applyMetadata.Tooling.HTMLPath) == "" {
		applyDetectedHTML := detectHTMLPath(applyConfig.rootPath)
		applyMetadata.Tooling.HTMLPath = applyLifecycleRelativePath(applyConfig.rootPath, applyDetectedHTML)
	}
	if strings.TrimSpace(applyMetadata.Tooling.WASMPath) == "" {
		applyMetadata.Tooling.WASMPath = filepath.ToSlash(scaffoldWASMOutputPath())
	}
	if strings.TrimSpace(applyMetadata.Tooling.DevHost) == "" {
		applyMetadata.Tooling.DevHost = defaultHost
	}
	if strings.TrimSpace(applyMetadata.Tooling.DevPort) == "" {
		applyMetadata.Tooling.DevPort = "8080"
	}
	if strings.TrimSpace(applyMetadata.Tooling.DefaultBuildProfile) == "" {
		applyMetadata.Tooling.DefaultBuildProfile = defaultScaffoldBuildProfile()
	}
	if strings.TrimSpace(applyMetadata.Tooling.ReleaseOutDir) == "" {
		applyMetadata.Tooling.ReleaseOutDir = defaultScaffoldReleaseOutDir()
	}
	if strings.TrimSpace(applyMetadata.Tooling.ReleaseBinaryName) == "" {
		applyMetadata.Tooling.ReleaseBinaryName = defaultScaffoldReleaseBinaryName()
	}
	if strings.TrimSpace(applyMetadata.Tooling.ReleaseCompression) == "" {
		applyMetadata.Tooling.ReleaseCompression = defaultScaffoldReleaseCompression()
	}
	if strings.TrimSpace(applyMetadata.Ownership.ProjectOwnership) == "" {
		applyMetadata.Ownership.ProjectOwnership = "standalone"
	}
	if strings.TrimSpace(applyMetadata.Ownership.FrameworkSourceMode) == "" {
		applyMetadata.Ownership.FrameworkSourceMode = "module-proxy"
	}
	applyMetadata.SchemaVersion = currentScaffoldMetadataSchemaVersion
	if applyErr := applyLifecycleMetadata(applyMetadataPath, applyMetadata); applyErr != nil {
		return lifecycleUpgradeSummary{}, applyErr
	}
	applyFeatureMatrixPath := ""
	if !fileExists(filepath.Join(applyConfig.rootPath, "FEATURE_MATRIX.md")) {
		applyFeatureMatrixPath = filepath.Join(applyConfig.rootPath, "FEATURE_MATRIX.md")
		applyFeatureMatrix := buildLifecycleFeatureMatrix(applyMetadata)
		if applyErr := os.WriteFile(applyFeatureMatrixPath, []byte(applyFeatureMatrix), 0644); applyErr != nil {
			return lifecycleUpgradeSummary{}, fmt.Errorf("write FEATURE_MATRIX.md: %w", applyErr)
		}
	}
	applyRuntimeAssetPath := ""
	if !applyConfig.skipRuntimeAssets {
		applyPath, applyErr := applyLifecycleRuntimeAsset(applyConfig.rootPath)
		if applyErr != nil {
			return lifecycleUpgradeSummary{}, applyErr
		}
		applyRuntimeAssetPath = applyPath
	}
	return lifecycleUpgradeSummary{
		OK:                true,
		Root:              applyConfig.rootPath,
		MetadataPath:      applyMetadataPath,
		FeatureMatrixPath: applyFeatureMatrixPath,
		RuntimeAssetPath:  applyRuntimeAssetPath,
		SchemaVersion:     applyMetadata.SchemaVersion,
	}, nil
}

// applyLifecycleMigrate upgrades metadata and writes a legacy API migration report.
func (parseL launcher) applyLifecycleMigrate(applyConfig lifecycleMigrateConfig) (lifecycleMigrateSummary, error) {
	applyUpgradeSummary, applyUpgradeErr := parseL.applyLifecycleUpgrade(lifecycleUpgradeConfig{
		rootPath:          applyConfig.rootPath,
		skipRuntimeAssets: applyConfig.skipRuntimeAssets,
		json:              false,
	})
	if applyUpgradeErr != nil {
		return lifecycleMigrateSummary{}, applyUpgradeErr
	}
	applyFindings, applyFindingsErr := buildLifecycleMigrateFindings(applyConfig.rootPath)
	if applyFindingsErr != nil {
		return lifecycleMigrateSummary{}, applyFindingsErr
	}
	applyReportPath := filepath.Join(applyConfig.rootPath, "bin", "gwc-migrate-report.json")
	if applyErr := os.MkdirAll(filepath.Dir(applyReportPath), 0755); applyErr != nil {
		return lifecycleMigrateSummary{}, fmt.Errorf("create migration report directory: %w", applyErr)
	}
	applyReportPayload := map[string]interface{}{
		"ok":       true,
		"root":     applyConfig.rootPath,
		"findings": applyFindings,
	}
	applyReportBytes, applyErr := json.MarshalIndent(applyReportPayload, "", "  ")
	if applyErr != nil {
		return lifecycleMigrateSummary{}, fmt.Errorf("encode migration report: %w", applyErr)
	}
	applyReportBytes = append(applyReportBytes, '\n')
	if applyErr := os.WriteFile(applyReportPath, applyReportBytes, 0644); applyErr != nil {
		return lifecycleMigrateSummary{}, fmt.Errorf("write migration report: %w", applyErr)
	}
	return lifecycleMigrateSummary{
		OK:                true,
		Root:              applyConfig.rootPath,
		MetadataPath:      applyUpgradeSummary.MetadataPath,
		FeatureMatrixPath: applyUpgradeSummary.FeatureMatrixPath,
		RuntimeAssetPath:  applyUpgradeSummary.RuntimeAssetPath,
		ReportPath:        applyReportPath,
		FindingCount:      len(applyFindings),
		Findings:          applyFindings,
	}, nil
}

// applyLifecycleMetadata writes scaffold metadata with stable formatting.
func applyLifecycleMetadata(applyMetadataPath string, applyMetadata scaffoldMetadata) error {
	applyBytes, applyErr := json.MarshalIndent(applyMetadata, "", "  ")
	if applyErr != nil {
		return fmt.Errorf("encode gwc-start.json: %w", applyErr)
	}
	applyBytes = append(applyBytes, '\n')
	if applyErr := os.WriteFile(applyMetadataPath, applyBytes, 0644); applyErr != nil {
		return fmt.Errorf("write gwc-start.json: %w", applyErr)
	}
	return nil
}

// applyLifecycleRuntimeAsset writes wasm_exec.js into a project root.
func applyLifecycleRuntimeAsset(applyRootPath string) (string, error) {
	applySourcePath, applyResolveErr := scaffoldResolveWasmExecPath()
	if applyResolveErr != nil {
		return "", applyResolveErr
	}
	applyBytes, applyReadErr := os.ReadFile(applySourcePath)
	if applyReadErr != nil {
		return "", fmt.Errorf("read wasm_exec.js: %w", applyReadErr)
	}
	applyTargetPath := filepath.Join(applyRootPath, "wasm_exec.js")
	if applyWriteErr := os.WriteFile(applyTargetPath, applyBytes, 0644); applyWriteErr != nil {
		return "", fmt.Errorf("write wasm_exec.js: %w", applyWriteErr)
	}
	return applyTargetPath, nil
}

// applyLifecycleRelativePath converts an absolute file path into a root-relative slash path.
func applyLifecycleRelativePath(applyRootPath string, applyPath string) string {
	applyPath = strings.TrimSpace(applyPath)
	if applyPath == "" {
		return ""
	}
	applyRelativePath, applyErr := filepath.Rel(applyRootPath, applyPath)
	if applyErr != nil {
		return filepath.ToSlash(applyPath)
	}
	return filepath.ToSlash(applyRelativePath)
}

// buildLifecycleFeatureMatrix builds a minimal feature matrix for upgrade-only projects.
func buildLifecycleFeatureMatrix(buildMetadata scaffoldMetadata) string {
	buildFeatures := append([]string{}, buildMetadata.Preset.Features...)
	buildFeatures = append(buildFeatures, buildMetadata.Enterprise.Features...)
	buildFeatureSet := map[string]struct{}{}
	for _, buildFeature := range buildFeatures {
		buildFeature = strings.TrimSpace(buildFeature)
		if buildFeature == "" {
			continue
		}
		buildFeatureSet[buildFeature] = struct{}{}
	}
	buildSortedFeatures := make([]string, 0, len(buildFeatureSet))
	for buildFeature := range buildFeatureSet {
		buildSortedFeatures = append(buildSortedFeatures, buildFeature)
	}
	sort.Strings(buildSortedFeatures)
	var buildBuilder strings.Builder
	buildBuilder.WriteString("# Feature Matrix\n\n")
	buildBuilder.WriteString("Generated by `gwc upgrade`.\n\n")
	if len(buildSortedFeatures) == 0 {
		buildBuilder.WriteString("- No preset features were recorded in metadata.\n")
		return buildBuilder.String()
	}
	for _, buildFeature := range buildSortedFeatures {
		buildBuilder.WriteString("- `")
		buildBuilder.WriteString(buildFeature)
		buildBuilder.WriteString("`\n")
	}
	return buildBuilder.String()
}

// buildLifecycleMigrateFindings scans Go files for legacy compatibility route API usage.
func buildLifecycleMigrateFindings(buildRootPath string) ([]lifecycleMigrateFinding, error) {
	buildFindings := []lifecycleMigrateFinding{}
	buildFiles, buildErr := collectGoldenPathGoFiles(buildRootPath)
	if buildErr != nil {
		return nil, buildErr
	}
	buildSignals := []struct {
		legacyCall     string
		replacement    string
		recommendation string
	}{
		{
			legacyCall:     "GoRegisterRoute(",
			replacement:    "router.Register(",
			recommendation: "Replace compatibility route registration with the primary router API.",
		},
		{
			legacyCall:     "GoGetRoute(",
			replacement:    "router.Current(",
			recommendation: "Replace compatibility route lookup with the primary router API.",
		},
	}

	for _, buildFile := range buildFiles {
		buildScanner := bufio.NewScanner(strings.NewReader(buildFile.Content))
		buildLineNumber := 0
		for buildScanner.Scan() {
			buildLineNumber++
			buildLine := buildScanner.Text()
			for _, buildSignal := range buildSignals {
				if strings.Contains(buildLine, buildSignal.legacyCall) {
					buildFindings = append(buildFindings, lifecycleMigrateFinding{
						Path:           buildFile.RelPath,
						Line:           buildLineNumber,
						LegacyCall:     buildSignal.legacyCall,
						Replacement:    buildSignal.replacement,
						Recommendation: buildSignal.recommendation,
					})
				}
			}
		}
	}
	sort.Slice(buildFindings, func(buildLeft int, buildRight int) bool {
		if buildFindings[buildLeft].Path != buildFindings[buildRight].Path {
			return buildFindings[buildLeft].Path < buildFindings[buildRight].Path
		}
		if buildFindings[buildLeft].Line != buildFindings[buildRight].Line {
			return buildFindings[buildLeft].Line < buildFindings[buildRight].Line
		}
		return buildFindings[buildLeft].LegacyCall < buildFindings[buildRight].LegacyCall
	})
	return buildFindings, nil
}

// renderLifecycleInitSummary prints a human-readable init command summary.
func renderLifecycleInitSummary(renderSummary lifecycleInitSummary) {
	fmt.Println("GWC init")
	fmt.Printf("  root:           %s\n", renderSummary.Root)
	fmt.Printf("  metadata:       %s\n", renderSummary.MetadataPath)
	fmt.Printf("  feature matrix: %s\n", renderSummary.FeatureMatrixPath)
	if strings.TrimSpace(renderSummary.RuntimeAssetPath) != "" {
		fmt.Printf("  wasm_exec.js:   %s\n", renderSummary.RuntimeAssetPath)
	}
	fmt.Printf("  app:            %s\n", renderSummary.AppPath)
	if strings.TrimSpace(renderSummary.HTMLPath) != "" {
		fmt.Printf("  html:           %s\n", renderSummary.HTMLPath)
	}
	fmt.Printf("  wasm output:    %s\n", renderSummary.WASMPath)
}

// renderLifecycleUpgradeSummary prints a human-readable upgrade command summary.
func renderLifecycleUpgradeSummary(renderSummary lifecycleUpgradeSummary) {
	fmt.Println("GWC upgrade")
	fmt.Printf("  root:           %s\n", renderSummary.Root)
	fmt.Printf("  metadata:       %s\n", renderSummary.MetadataPath)
	fmt.Printf("  schema version: %d\n", renderSummary.SchemaVersion)
	if strings.TrimSpace(renderSummary.FeatureMatrixPath) != "" {
		fmt.Printf("  feature matrix: %s\n", renderSummary.FeatureMatrixPath)
	}
	if strings.TrimSpace(renderSummary.RuntimeAssetPath) != "" {
		fmt.Printf("  wasm_exec.js:   %s\n", renderSummary.RuntimeAssetPath)
	}
}

// renderLifecycleMigrateSummary prints a human-readable migrate command summary.
func renderLifecycleMigrateSummary(renderSummary lifecycleMigrateSummary) {
	fmt.Println("GWC migrate")
	fmt.Printf("  root:           %s\n", renderSummary.Root)
	fmt.Printf("  metadata:       %s\n", renderSummary.MetadataPath)
	fmt.Printf("  report:         %s\n", renderSummary.ReportPath)
	fmt.Printf("  findings:       %d\n", renderSummary.FindingCount)
	if strings.TrimSpace(renderSummary.RuntimeAssetPath) != "" {
		fmt.Printf("  wasm_exec.js:   %s\n", renderSummary.RuntimeAssetPath)
	}
}
