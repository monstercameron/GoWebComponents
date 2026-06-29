package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
	"time"
)

var runWasmCommand = func(l launcher, args []string) error {
	return l.runWasm(args)
}

var wasmRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
	return launcherRunCommand(command, args, cwd, env)
}

var wasmLookPath = exec.LookPath

var wasmWriteGzipSidecar = writeGzipSidecar

var wasmWriteBrotliSidecar = writeBrotliSidecar

var wasmReleaseArtifactRecordForPath = releaseArtifactRecordForPath

type wasmMeasureConfig struct {
	packagePath     string
	outDir          string
	binaryName      string
	manifestName    string
	goExecutable    string
	ldflags         string
	releaseProfile  bool
	skipCompression bool
	serveReloadMs   int64
	hasServeReload  bool
	json            bool
}

type wasmMeasureManifest struct {
	Package      string                           `json:"package"`
	Profile      string                           `json:"profile"`
	GoExecutable string                           `json:"go_executable"`
	GoVersion    string                           `json:"go_version"`
	GOOS         string                           `json:"goos"`
	GOARCH       string                           `json:"goarch"`
	BuildArgs    []string                         `json:"build_args"`
	Phases       map[string]int64                 `json:"phases"`
	Artifacts    map[string]releaseArtifactRecord `json:"artifacts"`
}

type wasmMeasureSummary struct {
	OK           bool                `json:"ok"`
	OutDir       string              `json:"outDir"`
	ManifestPath string              `json:"manifestPath"`
	Manifest     wasmMeasureManifest `json:"manifest"`
}

type wasmCompareConfig struct {
	baselinePath            string
	candidatePath           string
	outFile                 string
	timingRegressionPercent float64
	sizeRegressionPercent   float64
	otherRegressionPercent  float64
	json                    bool
}

type wasmCompareThresholds struct {
	TimingRegressionPercent float64 `json:"timing_regression_percent"`
	SizeRegressionPercent   float64 `json:"size_regression_percent"`
	OtherRegressionPercent  float64 `json:"other_regression_percent"`
}

type wasmCompareCounts struct {
	Total           int `json:"total"`
	Improved        int `json:"improved"`
	Unchanged       int `json:"unchanged"`
	WithinThreshold int `json:"within_threshold"`
	Regressed       int `json:"regressed"`
	Added           int `json:"added"`
	Removed         int `json:"removed"`
}

type wasmCompareMetric struct {
	Path             string   `json:"path"`
	Status           string   `json:"status"`
	Category         string   `json:"category"`
	Baseline         *float64 `json:"baseline,omitempty"`
	Candidate        *float64 `json:"candidate,omitempty"`
	Delta            *float64 `json:"delta,omitempty"`
	DeltaPercent     *float64 `json:"delta_percent,omitempty"`
	ThresholdPercent *float64 `json:"threshold_percent,omitempty"`
}

type wasmCompareSummary struct {
	OK         bool                  `json:"ok"`
	Baseline   string                `json:"baseline"`
	Candidate  string                `json:"candidate"`
	ComparedAt string                `json:"compared_at"`
	Thresholds wasmCompareThresholds `json:"thresholds"`
	Counts     wasmCompareCounts     `json:"counts"`
	Metrics    []wasmCompareMetric   `json:"metrics"`
}

type wasmCompressionConfig struct {
	packagePath string
	outDir      string
	binaryName  string
	summaryName string
	json        bool
}

type wasmCacheConfig struct {
	packagePath    string
	outDir         string
	binaryName     string
	summaryName    string
	releaseProfile bool
	json           bool
}

type wasmCacheEnvironment struct {
	GoVersion      string `json:"go_version"`
	DefaultGoCache string `json:"default_gocache"`
	DefaultGoMod   string `json:"default_gomodcache"`
}

type wasmCacheVariantResult struct {
	Label            string               `json:"label"`
	GoCache          string               `json:"gocache"`
	GoModCache       string               `json:"gomodcache"`
	Status           string               `json:"status"`
	Notes            []string             `json:"notes,omitempty"`
	ModuleDownloadMS *int64               `json:"module_download_ms,omitempty"`
	EditedFile       string               `json:"edited_file,omitempty"`
	Measurement      *wasmMeasureManifest `json:"measurement,omitempty"`
	Error            string               `json:"error,omitempty"`
}

type wasmCacheSummary struct {
	OK          bool                              `json:"ok"`
	Package     string                            `json:"package"`
	GeneratedAt string                            `json:"generated_at"`
	Environment wasmCacheEnvironment              `json:"environment"`
	Variants    map[string]wasmCacheVariantResult `json:"variants"`
	SummaryPath string                            `json:"summaryPath,omitempty"`
}

type wasmCacheVariantSpec struct {
	key                string
	label              string
	goCache            string
	goModCache         string
	note               string
	smallEdit          bool
	prepareModuleCache bool
	resetBeforeRun     bool
}

type wasmToolchainConfig struct {
	packagePath             string
	baselineGoExecutable    string
	candidateGoExecutable   string
	binaryName              string
	outDir                  string
	timingRegressionPercent float64
	sizeRegressionPercent   float64
	otherRegressionPercent  float64
	releaseProfile          bool
	skipCompression         bool
	json                    bool
}

type wasmToolchainParty struct {
	GoExecutable string `json:"go_executable"`
	GoVersion    string `json:"go_version"`
	Manifest     string `json:"manifest"`
}

type wasmToolchainSummary struct {
	OK                 bool                  `json:"ok"`
	Package            string                `json:"package"`
	ComparedAt         string                `json:"compared_at"`
	Baseline           wasmToolchainParty    `json:"baseline"`
	Candidate          wasmToolchainParty    `json:"candidate"`
	Thresholds         wasmCompareThresholds `json:"thresholds"`
	Comparison         string                `json:"comparison"`
	RegressionExitCode int                   `json:"regression_exit_code"`
	SummaryPath        string                `json:"summaryPath,omitempty"`
}

type wasmCompressionEnvironment struct {
	GoVersion        string `json:"go_version"`
	BrotliSupported  bool   `json:"brotli_supported"`
	WasmOptAvailable bool   `json:"wasm_opt_available"`
	WasmOptPath      string `json:"wasm_opt_path,omitempty"`
}

type wasmCompressionSummary struct {
	OK          bool                       `json:"ok"`
	Package     string                     `json:"package"`
	GeneratedAt string                     `json:"generated_at"`
	Environment wasmCompressionEnvironment `json:"environment"`
	Variants    map[string]any             `json:"variants"`
	Unsupported map[string]string          `json:"unsupported,omitempty"`
	SummaryPath string                     `json:"summaryPath,omitempty"`
}

type wasmOptimizerCommand struct {
	Available  bool
	Command    string
	PrefixArgs []string
	Label      string
}

// runWasm routes wasm-focused helper subcommands.
func (parseL launcher) runWasm(parseArgs []string) error {
	if len(parseArgs) == 0 {
		return errors.New("wasm requires a subcommand: measure, compare, compare-compression, compare-cache, compare-toolchain")
	}
	switch strings.ToLower(strings.TrimSpace(parseArgs[0])) {
	case "measure":
		return parseL.runWasmMeasure(parseArgs[1:])
	case "compare":
		return parseL.runWasmCompare(parseArgs[1:])
	case "compare-compression":
		return parseL.runWasmCompareCompression(parseArgs[1:])
	case "compare-cache", "compare-build-cache":
		return parseL.runWasmCompareCache(parseArgs[1:])
	case "compare-toolchain", "compare-go-toolchain":
		return parseL.runWasmCompareToolchain(parseArgs[1:])
	default:
		return fmt.Errorf("unknown wasm subcommand %q", parseArgs[0])
	}
}

// runWasmMeasure builds a wasm package and writes phase-attributed build metadata.
func (parseL launcher) runWasmMeasure(parseArgs []string) error {
	parseFs := flag.NewFlagSet("wasm measure", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parsePackagePath := parseFs.String("package", ".", "Package path to build for js/wasm")
	parseOutDir := parseFs.String("out-dir", filepath.Join("bin", "wasm-build-experiment"), "Output directory for artifacts and manifest")
	parseBinaryName := parseFs.String("binary-name", "app.wasm", "Wasm artifact filename")
	parseManifestName := parseFs.String("manifest-name", "wasm-build-experiment.json", "Manifest filename")
	parseGoExecutable := parseFs.String("go-executable", "go", "Go executable used for build and version checks")
	parseLdflags := parseFs.String("ldflags", "-s -w", "Release profile ldflags value")
	parseReleaseProfile := parseFs.Bool("release-profile", false, "Use release-style build flags (-trimpath, -ldflags, -buildvcs=false)")
	parseSkipCompression := parseFs.Bool("skip-compression", false, "Skip gzip and brotli sidecar generation")
	parseServeReloadMs := parseFs.Int64("serve-reload-ms", 0, "Optional external reload timing in milliseconds")
	parseMaxGzipBytes := parseFs.Int64("max-gzip-bytes", 0, "Fail (non-zero exit) if the gzip artifact exceeds this many bytes; 0 disables the budget gate (CI size budget)")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	hasServeReload := false
	parseFs.Visit(func(parseCurrent *flag.Flag) {
		if parseCurrent.Name == "serve-reload-ms" {
			hasServeReload = true
		}
	})
	parseConfig, parseErr2 := resolveWasmMeasureConfig(wasmMeasureConfig{
		packagePath:     *parsePackagePath,
		outDir:          *parseOutDir,
		binaryName:      *parseBinaryName,
		manifestName:    *parseManifestName,
		goExecutable:    *parseGoExecutable,
		ldflags:         *parseLdflags,
		releaseProfile:  *parseReleaseProfile,
		skipCompression: *parseSkipCompression,
		serveReloadMs:   *parseServeReloadMs,
		hasServeReload:  hasServeReload,
		json:            *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseErr2 := executeWasmMeasure(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseEncodeErr := parseEncoder.Encode(parseSummary); parseEncodeErr != nil {
			return parseEncodeErr
		}
	} else {
		printWasmMeasureSummary(parseSummary)
	}
	if parseErr2 != nil {
		return parseErr2
	}
	// CI size-budget gate: a non-zero -max-gzip-bytes fails the build when the gzip artifact
	// exceeds the budget, so a starter (or any app) catches bundle-size regressions in CI.
	if parseBudgetErr := checkWasmGzipBudget(parseSummary, *parseMaxGzipBytes); parseBudgetErr != nil {
		return parseBudgetErr
	}
	return nil
}

// checkWasmGzipBudget enforces a gzip-size budget against a measured summary. A budget of 0 (or
// negative) disables the gate. When the gzip artifact is missing it cannot be checked, so the
// gate is a no-op rather than a false failure. Pure and testable.
func checkWasmGzipBudget(parseSummary wasmMeasureSummary, parseMaxGzipBytes int64) error {
	if parseMaxGzipBytes <= 0 {
		return nil
	}
	parseGzip, parseOk := parseSummary.Manifest.Artifacts["gzip"]
	if !parseOk {
		return nil
	}
	if parseGzip.Bytes > parseMaxGzipBytes {
		return fmt.Errorf("wasm gzip size %d bytes exceeds budget %d bytes", parseGzip.Bytes, parseMaxGzipBytes)
	}
	return nil
}

// runWasmCompare compares two wasm experiment manifests and reports threshold regressions.
func (parseL launcher) runWasmCompare(parseArgs []string) error {
	parseFs := flag.NewFlagSet("wasm compare", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseBaseline := parseFs.String("baseline", "", "Baseline wasm manifest path")
	parseCandidate := parseFs.String("candidate", "", "Candidate wasm manifest path")
	parseOutFile := parseFs.String("out-file", "", "Optional comparison summary JSON path")
	parseTimingThreshold := parseFs.Float64("timing-regression-percent", 10, "Allowed timing regression percentage")
	parseSizeThreshold := parseFs.Float64("size-regression-percent", 0, "Allowed size regression percentage")
	parseOtherThreshold := parseFs.Float64("other-regression-percent", 0, "Allowed fallback regression percentage")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveWasmCompareConfig(wasmCompareConfig{
		baselinePath:            *parseBaseline,
		candidatePath:           *parseCandidate,
		outFile:                 *parseOutFile,
		timingRegressionPercent: *parseTimingThreshold,
		sizeRegressionPercent:   *parseSizeThreshold,
		otherRegressionPercent:  *parseOtherThreshold,
		json:                    *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseRegressionCount, parseCompareErr := executeWasmCompare(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		_ = parseEncoder.Encode(parseSummary)
	} else {
		printWasmCompareSummary(parseSummary)
	}
	if parseCompareErr != nil {
		return parseCompareErr
	}
	if parseRegressionCount > 0 {
		return fmt.Errorf("detected %d metric regressions beyond configured thresholds", parseRegressionCount)
	}
	return nil
}

// runWasmCompareCompression compares plain, stripped, compressed, and optimized wasm variants.
func (parseL launcher) runWasmCompareCompression(parseArgs []string) error {
	parseFs := flag.NewFlagSet("wasm compare-compression", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parsePackagePath := parseFs.String("package", "./examples/public/ui-render", "Package path to build")
	parseOutDir := parseFs.String("out-dir", filepath.Join("bin", "wasm-compression-comparison"), "Output directory for comparison artifacts")
	parseBinaryName := parseFs.String("binary-name", "app.wasm", "Wasm artifact filename")
	parseSummaryName := parseFs.String("summary-name", "wasm-compression-comparison.json", "Summary JSON filename")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr2 := resolveWasmCompressionConfig(wasmCompressionConfig{
		packagePath: *parsePackagePath,
		outDir:      *parseOutDir,
		binaryName:  *parseBinaryName,
		summaryName: *parseSummaryName,
		json:        *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseErr2 := executeWasmCompareCompression(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseEncodeErr := parseEncoder.Encode(parseSummary); parseEncodeErr != nil {
			return parseEncodeErr
		}
	} else {
		printWasmCompressionSummary(parseSummary)
	}
	return parseErr2
}

// runWasmCompareCache compares wasm build behavior across cache topologies.
func (parseL launcher) runWasmCompareCache(parseArgs []string) error {
	parseFs := flag.NewFlagSet("wasm compare-cache", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parsePackagePath := parseFs.String("package", "./examples/public/ui-render", "Package path to build")
	parseOutDir := parseFs.String("out-dir", filepath.Join("bin", "wasm-build-cache-comparison"), "Output directory for comparison artifacts")
	parseBinaryName := parseFs.String("binary-name", "app.wasm", "Wasm artifact filename")
	parseSummaryName := parseFs.String("summary-name", "wasm-build-cache-comparison.json", "Summary JSON filename")
	parseReleaseProfile := parseFs.Bool("release-profile", false, "Use release-style wasm build profile")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr2 := resolveWasmCacheConfig(wasmCacheConfig{
		packagePath:    *parsePackagePath,
		outDir:         *parseOutDir,
		binaryName:     *parseBinaryName,
		summaryName:    *parseSummaryName,
		releaseProfile: *parseReleaseProfile,
		json:           *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseErr2 := executeWasmCompareCache(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseEncodeErr := parseEncoder.Encode(parseSummary); parseEncodeErr != nil {
			return parseEncodeErr
		}
	} else {
		printWasmCacheSummary(parseSummary)
	}
	return parseErr2
}

// runWasmCompareToolchain compares wasm build outputs across two Go toolchains.
func (parseL launcher) runWasmCompareToolchain(parseArgs []string) error {
	parseFs := flag.NewFlagSet("wasm compare-toolchain", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parsePackagePath := parseFs.String("package", "", "Package path to build (required)")
	parseBaselineGo := parseFs.String("baseline-go", "", "Baseline Go executable (required)")
	parseCandidateGo := parseFs.String("candidate-go", "", "Candidate Go executable (required)")
	parseBinaryName := parseFs.String("binary-name", "app.wasm", "Wasm artifact filename")
	parseOutDir := parseFs.String("out-dir", filepath.Join("bin", "wasm-toolchain-comparison"), "Output directory for comparison artifacts")
	parseTimingThreshold := parseFs.Float64("timing-regression-percent", 10, "Allowed timing regression percentage")
	parseSizeThreshold := parseFs.Float64("size-regression-percent", 0, "Allowed size regression percentage")
	parseOtherThreshold := parseFs.Float64("other-regression-percent", 0, "Allowed fallback regression percentage")
	parseReleaseProfile := parseFs.Bool("release-profile", false, "Use release-style wasm build profile")
	parseSkipCompression := parseFs.Bool("skip-compression", false, "Skip gzip and brotli sidecar generation during measurement")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseConfig, parseErr2 := resolveWasmToolchainConfig(wasmToolchainConfig{
		packagePath:             *parsePackagePath,
		baselineGoExecutable:    *parseBaselineGo,
		candidateGoExecutable:   *parseCandidateGo,
		binaryName:              *parseBinaryName,
		outDir:                  *parseOutDir,
		timingRegressionPercent: *parseTimingThreshold,
		sizeRegressionPercent:   *parseSizeThreshold,
		otherRegressionPercent:  *parseOtherThreshold,
		releaseProfile:          *parseReleaseProfile,
		skipCompression:         *parseSkipCompression,
		json:                    *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary, parseErr2 := executeWasmCompareToolchain(parseConfig)
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseEncodeErr := parseEncoder.Encode(parseSummary); parseEncodeErr != nil {
			return parseEncodeErr
		}
	} else {
		printWasmToolchainSummary(parseSummary)
	}
	return parseErr2
}

// resolveWasmMeasureConfig validates and normalizes wasm measurement configuration.
func resolveWasmMeasureConfig(parseConfig wasmMeasureConfig) (wasmMeasureConfig, error) {
	parsePackagePath := strings.TrimSpace(parseConfig.packagePath)
	if parsePackagePath == "" {
		parsePackagePath = "."
	}
	parseOutDir := strings.TrimSpace(parseConfig.outDir)
	if parseOutDir == "" {
		parseOutDir = filepath.Join("bin", "wasm-build-experiment")
	}
	parseCwd, parseErr := os.Getwd()
	if parseErr != nil {
		return wasmMeasureConfig{}, fmt.Errorf("resolve wasm measure cwd: %w", parseErr)
	}
	if !filepath.IsAbs(parseOutDir) {
		parseOutDir = filepath.Join(parseCwd, parseOutDir)
	}
	if parseErr2 := os.MkdirAll(parseOutDir, 0755); parseErr2 != nil {
		return wasmMeasureConfig{}, fmt.Errorf("create wasm measure out dir: %w", parseErr2)
	}
	parseBinaryName := strings.TrimSpace(parseConfig.binaryName)
	if parseBinaryName == "" {
		parseBinaryName = "app.wasm"
	}
	parseManifestName := strings.TrimSpace(parseConfig.manifestName)
	if parseManifestName == "" {
		parseManifestName = "wasm-build-experiment.json"
	}
	parseGoExecutable := strings.TrimSpace(parseConfig.goExecutable)
	if parseGoExecutable == "" {
		parseGoExecutable = "go"
	}
	return wasmMeasureConfig{
		packagePath:     parsePackagePath,
		outDir:          filepath.Clean(parseOutDir),
		binaryName:      filepath.Base(parseBinaryName),
		manifestName:    filepath.Base(parseManifestName),
		goExecutable:    parseGoExecutable,
		ldflags:         strings.TrimSpace(parseConfig.ldflags),
		releaseProfile:  parseConfig.releaseProfile,
		skipCompression: parseConfig.skipCompression,
		serveReloadMs:   parseConfig.serveReloadMs,
		hasServeReload:  parseConfig.hasServeReload,
		json:            parseConfig.json,
	}, nil
}

// resolveWasmCompareConfig validates and normalizes wasm compare configuration.
func resolveWasmCompareConfig(parseConfig wasmCompareConfig) (wasmCompareConfig, error) {
	parseCwd, parseErr := os.Getwd()
	if parseErr != nil {
		return wasmCompareConfig{}, fmt.Errorf("resolve wasm compare cwd: %w", parseErr)
	}
	parseBaselinePath, parseErr := normalizeExistingPath(parseCwd, strings.TrimSpace(parseConfig.baselinePath))
	if parseErr != nil {
		return wasmCompareConfig{}, fmt.Errorf("resolve baseline path: %w", parseErr)
	}
	parseCandidatePath, parseErr := normalizeExistingPath(parseCwd, strings.TrimSpace(parseConfig.candidatePath))
	if parseErr != nil {
		return wasmCompareConfig{}, fmt.Errorf("resolve candidate path: %w", parseErr)
	}
	parseOutFile := strings.TrimSpace(parseConfig.outFile)
	if parseOutFile != "" && !filepath.IsAbs(parseOutFile) {
		parseOutFile = filepath.Join(parseCwd, parseOutFile)
	}
	if strings.TrimSpace(parseOutFile) != "" {
		parseOutFile = filepath.Clean(parseOutFile)
	}
	return wasmCompareConfig{
		baselinePath:            parseBaselinePath,
		candidatePath:           parseCandidatePath,
		outFile:                 parseOutFile,
		timingRegressionPercent: parseConfig.timingRegressionPercent,
		sizeRegressionPercent:   parseConfig.sizeRegressionPercent,
		otherRegressionPercent:  parseConfig.otherRegressionPercent,
		json:                    parseConfig.json,
	}, nil
}

// resolveWasmCompressionConfig validates and normalizes wasm compression comparison settings.
func resolveWasmCompressionConfig(parseConfig wasmCompressionConfig) (wasmCompressionConfig, error) {
	parseCwd, parseErr := os.Getwd()
	if parseErr != nil {
		return wasmCompressionConfig{}, fmt.Errorf("resolve wasm compression cwd: %w", parseErr)
	}
	parseOutDir := strings.TrimSpace(parseConfig.outDir)
	if parseOutDir == "" {
		parseOutDir = filepath.Join("bin", "wasm-compression-comparison")
	}
	if !filepath.IsAbs(parseOutDir) {
		parseOutDir = filepath.Join(parseCwd, parseOutDir)
	}
	if parseErr2 := os.MkdirAll(parseOutDir, 0755); parseErr2 != nil {
		return wasmCompressionConfig{}, fmt.Errorf("create wasm compression out dir: %w", parseErr2)
	}
	parsePackagePath := strings.TrimSpace(parseConfig.packagePath)
	if parsePackagePath == "" {
		parsePackagePath = "."
	}
	parseBinaryName := strings.TrimSpace(parseConfig.binaryName)
	if parseBinaryName == "" {
		parseBinaryName = "app.wasm"
	}
	parseSummaryName := strings.TrimSpace(parseConfig.summaryName)
	if parseSummaryName == "" {
		parseSummaryName = "wasm-compression-comparison.json"
	}
	return wasmCompressionConfig{
		packagePath: parsePackagePath,
		outDir:      filepath.Clean(parseOutDir),
		binaryName:  filepath.Base(parseBinaryName),
		summaryName: filepath.Base(parseSummaryName),
		json:        parseConfig.json,
	}, nil
}

// resolveWasmCacheConfig validates and normalizes wasm cache comparison settings.
func resolveWasmCacheConfig(parseConfig wasmCacheConfig) (wasmCacheConfig, error) {
	parseCwd, parseErr := os.Getwd()
	if parseErr != nil {
		return wasmCacheConfig{}, fmt.Errorf("resolve wasm cache cwd: %w", parseErr)
	}
	parseOutDir := strings.TrimSpace(parseConfig.outDir)
	if parseOutDir == "" {
		parseOutDir = filepath.Join("bin", "wasm-build-cache-comparison")
	}
	if !filepath.IsAbs(parseOutDir) {
		parseOutDir = filepath.Join(parseCwd, parseOutDir)
	}
	if parseErr2 := os.MkdirAll(parseOutDir, 0755); parseErr2 != nil {
		return wasmCacheConfig{}, fmt.Errorf("create wasm cache out dir: %w", parseErr2)
	}
	parsePackagePath := strings.TrimSpace(parseConfig.packagePath)
	if parsePackagePath == "" {
		parsePackagePath = "."
	}
	parseBinaryName := strings.TrimSpace(parseConfig.binaryName)
	if parseBinaryName == "" {
		parseBinaryName = "app.wasm"
	}
	parseSummaryName := strings.TrimSpace(parseConfig.summaryName)
	if parseSummaryName == "" {
		parseSummaryName = "wasm-build-cache-comparison.json"
	}
	return wasmCacheConfig{
		packagePath:    parsePackagePath,
		outDir:         filepath.Clean(parseOutDir),
		binaryName:     filepath.Base(parseBinaryName),
		summaryName:    filepath.Base(parseSummaryName),
		releaseProfile: parseConfig.releaseProfile,
		json:           parseConfig.json,
	}, nil
}

// resolveWasmToolchainConfig validates and normalizes toolchain comparison settings.
func resolveWasmToolchainConfig(parseConfig wasmToolchainConfig) (wasmToolchainConfig, error) {
	parseCwd, parseErr := os.Getwd()
	if parseErr != nil {
		return wasmToolchainConfig{}, fmt.Errorf("resolve wasm toolchain cwd: %w", parseErr)
	}
	parsePackagePath := strings.TrimSpace(parseConfig.packagePath)
	if parsePackagePath == "" {
		return wasmToolchainConfig{}, errors.New("wasm compare-toolchain requires -package")
	}
	parseBaselineGo := strings.TrimSpace(parseConfig.baselineGoExecutable)
	if parseBaselineGo == "" {
		return wasmToolchainConfig{}, errors.New("wasm compare-toolchain requires -baseline-go")
	}
	parseCandidateGo := strings.TrimSpace(parseConfig.candidateGoExecutable)
	if parseCandidateGo == "" {
		return wasmToolchainConfig{}, errors.New("wasm compare-toolchain requires -candidate-go")
	}
	parseBinaryName := strings.TrimSpace(parseConfig.binaryName)
	if parseBinaryName == "" {
		parseBinaryName = "app.wasm"
	}
	parseOutDir := strings.TrimSpace(parseConfig.outDir)
	if parseOutDir == "" {
		parseOutDir = filepath.Join("bin", "wasm-toolchain-comparison")
	}
	if !filepath.IsAbs(parseOutDir) {
		parseOutDir = filepath.Join(parseCwd, parseOutDir)
	}
	if parseErr2 := os.MkdirAll(parseOutDir, 0755); parseErr2 != nil {
		return wasmToolchainConfig{}, fmt.Errorf("create wasm toolchain out dir: %w", parseErr2)
	}
	return wasmToolchainConfig{
		packagePath:             parsePackagePath,
		baselineGoExecutable:    parseBaselineGo,
		candidateGoExecutable:   parseCandidateGo,
		binaryName:              filepath.Base(parseBinaryName),
		outDir:                  filepath.Clean(parseOutDir),
		timingRegressionPercent: parseConfig.timingRegressionPercent,
		sizeRegressionPercent:   parseConfig.sizeRegressionPercent,
		otherRegressionPercent:  parseConfig.otherRegressionPercent,
		releaseProfile:          parseConfig.releaseProfile,
		skipCompression:         parseConfig.skipCompression,
		json:                    parseConfig.json,
	}, nil
}

// executeWasmMeasure runs the measured build and writes the manifest artifact.
func executeWasmMeasure(parseConfig wasmMeasureConfig) (wasmMeasureSummary, error) {
	parseWasmPath := filepath.Join(parseConfig.outDir, parseConfig.binaryName)
	parseManifestPath := filepath.Join(parseConfig.outDir, parseConfig.manifestName)
	buildArgs := []string{"build", "-o", parseWasmPath}
	if parseConfig.releaseProfile {
		buildArgs = append(buildArgs, "-trimpath")
		if strings.TrimSpace(parseConfig.ldflags) != "" {
			buildArgs = append(buildArgs, "-ldflags="+parseConfig.ldflags)
		}
		buildArgs = append(buildArgs, "-buildvcs=false")
	}
	buildArgs = append(buildArgs, parseConfig.packagePath)

	parseTotalStart := time.Now()
	buildStart := time.Now()
	_, buildErr := wasmRunCommand(parseConfig.goExecutable, buildArgs, "", append(buildNativeGoEnv(), "GOOS=js", "GOARCH=wasm"))
	buildDurationMs := time.Since(buildStart).Milliseconds()
	if buildErr != nil {
		return wasmMeasureSummary{}, buildErr
	}

	parseArtifacts := map[string]releaseArtifactRecord{}
	parseWasmArtifact, parseErr := wasmReleaseArtifactRecordForPath(parseConfig.outDir, parseWasmPath)
	if parseErr != nil {
		return wasmMeasureSummary{}, parseErr
	}
	parseArtifacts["wasm"] = parseWasmArtifact

	parsePhases := map[string]int64{
		"go_build_ms":          buildDurationMs,
		"compression_total_ms": 0,
	}
	if !parseConfig.skipCompression {
		parseCompressionStart := time.Now()
		parseGzipPath := parseWasmPath + ".gz"
		parseGzipStart := time.Now()
		if parseErr2 := wasmWriteGzipSidecar(parseWasmPath, parseGzipPath); parseErr2 != nil {
			return wasmMeasureSummary{}, parseErr2
		}
		parsePhases["gzip_ms"] = time.Since(parseGzipStart).Milliseconds()
		parseGzipArtifact, parseErr3 := wasmReleaseArtifactRecordForPath(parseConfig.outDir, parseGzipPath)
		if parseErr3 != nil {
			return wasmMeasureSummary{}, parseErr3
		}
		parseArtifacts["gzip"] = parseGzipArtifact

		parseBrotliPath := parseWasmPath + ".br"
		parseBrotliStart := time.Now()
		if parseErr4 := wasmWriteBrotliSidecar(parseWasmPath, parseBrotliPath); parseErr4 != nil {
			return wasmMeasureSummary{}, parseErr4
		}
		parsePhases["brotli_ms"] = time.Since(parseBrotliStart).Milliseconds()
		parseBrotliArtifact, parseErr3 := wasmReleaseArtifactRecordForPath(parseConfig.outDir, parseBrotliPath)
		if parseErr3 != nil {
			return wasmMeasureSummary{}, parseErr3
		}
		parseArtifacts["brotli"] = parseBrotliArtifact
		parsePhases["compression_total_ms"] = time.Since(parseCompressionStart).Milliseconds()
	}
	if parseConfig.hasServeReload {
		parsePhases["serve_reload_ms"] = parseConfig.serveReloadMs
	}
	parsePhases["total_wall_ms"] = time.Since(parseTotalStart).Milliseconds()

	parseGoVersionOutput, parseVersionErr := wasmRunCommand(parseConfig.goExecutable, []string{"version"}, "", buildNativeGoEnv())
	if parseVersionErr != nil {
		return wasmMeasureSummary{}, parseVersionErr
	}
	parseManifest := wasmMeasureManifest{
		Package:      parseConfig.packagePath,
		Profile:      wasmMeasureProfileName(parseConfig.releaseProfile),
		GoExecutable: parseConfig.goExecutable,
		GoVersion:    strings.TrimSpace(parseGoVersionOutput),
		GOOS:         "js",
		GOARCH:       "wasm",
		BuildArgs:    append([]string(nil), buildArgs...),
		Phases:       parsePhases,
		Artifacts:    parseArtifacts,
	}
	parsePayload, parseErr := json.MarshalIndent(parseManifest, "", "  ")
	if parseErr != nil {
		return wasmMeasureSummary{}, fmt.Errorf("marshal wasm measure manifest: %w", parseErr)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr5 := os.WriteFile(parseManifestPath, parsePayload, 0644); parseErr5 != nil {
		return wasmMeasureSummary{}, fmt.Errorf("write wasm measure manifest: %w", parseErr5)
	}
	return wasmMeasureSummary{
		OK:           true,
		OutDir:       filepath.ToSlash(parseConfig.outDir),
		ManifestPath: filepath.ToSlash(parseManifestPath),
		Manifest:     parseManifest,
	}, nil
}

// executeWasmCompare compares numeric metrics between two manifest files.
func executeWasmCompare(parseConfig wasmCompareConfig) (wasmCompareSummary, int, error) {
	parseBaselinePayload, parseErr := os.ReadFile(parseConfig.baselinePath)
	if parseErr != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("read baseline manifest: %w", parseErr)
	}
	parseCandidatePayload, parseErr := os.ReadFile(parseConfig.candidatePath)
	if parseErr != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("read candidate manifest: %w", parseErr)
	}
	var parseBaselineData any
	var parseCandidateData any
	if parseErr2 := json.Unmarshal(parseBaselinePayload, &parseBaselineData); parseErr2 != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("parse baseline manifest: %w", parseErr2)
	}
	if parseErr3 := json.Unmarshal(parseCandidatePayload, &parseCandidateData); parseErr3 != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("parse candidate manifest: %w", parseErr3)
	}

	parseBaselineMetrics := map[string]float64{}
	parseCandidateMetrics := map[string]float64{}
	collectWasmNumericMetrics(parseBaselineData, "", parseBaselineMetrics)
	collectWasmNumericMetrics(parseCandidateData, "", parseCandidateMetrics)

	parsePathSet := map[string]struct{}{}
	for parseKey := range parseBaselineMetrics {
		parsePathSet[parseKey] = struct{}{}
	}
	for parseKey2 := range parseCandidateMetrics {
		parsePathSet[parseKey2] = struct{}{}
	}
	parsePaths := make([]string, 0, len(parsePathSet))
	for parseKey3 := range parsePathSet {
		parsePaths = append(parsePaths, parseKey3)
	}
	sort.Strings(parsePaths)

	parseMetrics := make([]wasmCompareMetric, 0, len(parsePaths))
	parseCounts := wasmCompareCounts{}
	for _, parsePath := range parsePaths {
		parseCounts.Total++
		parseBaselineValue, hasBaseline := parseBaselineMetrics[parsePath]
		parseCandidateValue, hasCandidate := parseCandidateMetrics[parsePath]
		parseCategory := wasmMetricCategory(parsePath)
		if !hasBaseline {
			parseCandidateCopy := parseCandidateValue
			parseMetrics = append(parseMetrics, wasmCompareMetric{
				Path:      parsePath,
				Status:    "added",
				Category:  parseCategory,
				Candidate: &parseCandidateCopy,
			})
			parseCounts.Added++
			continue
		}
		if !hasCandidate {
			parseBaselineCopy := parseBaselineValue
			parseMetrics = append(parseMetrics, wasmCompareMetric{
				Path:     parsePath,
				Status:   "removed",
				Category: parseCategory,
				Baseline: &parseBaselineCopy,
			})
			parseCounts.Removed++
			continue
		}
		parseDelta := parseCandidateValue - parseBaselineValue
		parseThreshold := wasmMetricThreshold(parsePath, parseConfig)
		var parseDeltaPercent *float64
		if parseBaselineValue == 0 {
			if parseCandidateValue == 0 {
				parseZero := 0.0
				parseDeltaPercent = &parseZero
			}
		} else {
			parseValue := math.Round(((parseDelta/parseBaselineValue)*100)*10000) / 10000
			parseDeltaPercent = &parseValue
		}
		parseStatus := "unchanged"
		switch {
		case parseDelta < 0:
			parseStatus = "improved"
			parseCounts.Improved++
		case parseDelta > 0:
			if parseDeltaPercent != nil && *parseDeltaPercent > parseThreshold {
				parseStatus = "regressed"
				parseCounts.Regressed++
			} else {
				parseStatus = "within-threshold"
				parseCounts.WithinThreshold++
			}
		default:
			parseCounts.Unchanged++
		}
		parseBaselineCopy2 := parseBaselineValue
		parseCandidateCopy2 := parseCandidateValue
		parseDeltaCopy := parseDelta
		parseThresholdCopy := parseThreshold
		parseMetrics = append(parseMetrics, wasmCompareMetric{
			Path:             parsePath,
			Status:           parseStatus,
			Category:         parseCategory,
			Baseline:         &parseBaselineCopy2,
			Candidate:        &parseCandidateCopy2,
			Delta:            &parseDeltaCopy,
			DeltaPercent:     parseDeltaPercent,
			ThresholdPercent: &parseThresholdCopy,
		})
	}

	parseSummary := wasmCompareSummary{
		OK:         parseCounts.Regressed == 0,
		Baseline:   filepath.ToSlash(parseConfig.baselinePath),
		Candidate:  filepath.ToSlash(parseConfig.candidatePath),
		ComparedAt: time.Now().UTC().Format(time.RFC3339),
		Thresholds: wasmCompareThresholds{
			TimingRegressionPercent: parseConfig.timingRegressionPercent,
			SizeRegressionPercent:   parseConfig.sizeRegressionPercent,
			OtherRegressionPercent:  parseConfig.otherRegressionPercent,
		},
		Counts:  parseCounts,
		Metrics: parseMetrics,
	}
	if strings.TrimSpace(parseConfig.outFile) != "" {
		if parseErr4 := os.MkdirAll(filepath.Dir(parseConfig.outFile), 0755); parseErr4 != nil {
			return wasmCompareSummary{}, 0, fmt.Errorf("create compare output dir: %w", parseErr4)
		}
		parsePayload, parseErr5 := json.MarshalIndent(parseSummary, "", "  ")
		if parseErr5 != nil {
			return wasmCompareSummary{}, 0, fmt.Errorf("marshal compare summary: %w", parseErr5)
		}
		parsePayload = append(parsePayload, '\n')
		if parseErr6 := os.WriteFile(parseConfig.outFile, parsePayload, 0644); parseErr6 != nil {
			return wasmCompareSummary{}, 0, fmt.Errorf("write compare summary: %w", parseErr6)
		}
	}
	return parseSummary, parseCounts.Regressed, nil
}

// executeWasmCompareCompression runs multiple wasm build variants and saves a comparison summary.
