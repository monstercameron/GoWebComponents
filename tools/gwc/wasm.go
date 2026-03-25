package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strconv"
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
	Variants    map[string]interface{}     `json:"variants"`
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
	parsePackagePath := parseFs.String("package", "./examples/21-ui-render", "Package path to build")
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
	parsePackagePath := parseFs.String("package", "./examples/21-ui-render", "Package path to build")
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
	var parseBaselineData interface{}
	var parseCandidateData interface{}
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
