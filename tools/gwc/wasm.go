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
func (l launcher) runWasm(args []string) error {
	if len(args) == 0 {
		return errors.New("wasm requires a subcommand: measure, compare, compare-compression, compare-cache, compare-toolchain")
	}
	switch strings.ToLower(strings.TrimSpace(args[0])) {
	case "measure":
		return l.runWasmMeasure(args[1:])
	case "compare":
		return l.runWasmCompare(args[1:])
	case "compare-compression":
		return l.runWasmCompareCompression(args[1:])
	case "compare-cache", "compare-build-cache":
		return l.runWasmCompareCache(args[1:])
	case "compare-toolchain", "compare-go-toolchain":
		return l.runWasmCompareToolchain(args[1:])
	default:
		return fmt.Errorf("unknown wasm subcommand %q", args[0])
	}
}

// runWasmMeasure builds a wasm package and writes phase-attributed build metadata.
func (l launcher) runWasmMeasure(args []string) error {
	fs := flag.NewFlagSet("wasm measure", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	packagePath := fs.String("package", ".", "Package path to build for js/wasm")
	outDir := fs.String("out-dir", filepath.Join("bin", "wasm-build-experiment"), "Output directory for artifacts and manifest")
	binaryName := fs.String("binary-name", "app.wasm", "Wasm artifact filename")
	manifestName := fs.String("manifest-name", "wasm-build-experiment.json", "Manifest filename")
	goExecutable := fs.String("go-executable", "go", "Go executable used for build and version checks")
	ldflags := fs.String("ldflags", "-s -w", "Release profile ldflags value")
	releaseProfile := fs.Bool("release-profile", false, "Use release-style build flags (-trimpath, -ldflags, -buildvcs=false)")
	skipCompression := fs.Bool("skip-compression", false, "Skip gzip and brotli sidecar generation")
	serveReloadMs := fs.Int64("serve-reload-ms", 0, "Optional external reload timing in milliseconds")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	hasServeReload := false
	fs.Visit(func(current *flag.Flag) {
		if current.Name == "serve-reload-ms" {
			hasServeReload = true
		}
	})
	config, err := resolveWasmMeasureConfig(wasmMeasureConfig{
		packagePath:     *packagePath,
		outDir:          *outDir,
		binaryName:      *binaryName,
		manifestName:    *manifestName,
		goExecutable:    *goExecutable,
		ldflags:         *ldflags,
		releaseProfile:  *releaseProfile,
		skipCompression: *skipCompression,
		serveReloadMs:   *serveReloadMs,
		hasServeReload:  hasServeReload,
		json:            *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := executeWasmMeasure(config)
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(summary); encodeErr != nil {
			return encodeErr
		}
	} else {
		printWasmMeasureSummary(summary)
	}
	if err != nil {
		return err
	}
	return nil
}

// runWasmCompare compares two wasm experiment manifests and reports threshold regressions.
func (l launcher) runWasmCompare(args []string) error {
	fs := flag.NewFlagSet("wasm compare", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	baseline := fs.String("baseline", "", "Baseline wasm manifest path")
	candidate := fs.String("candidate", "", "Candidate wasm manifest path")
	outFile := fs.String("out-file", "", "Optional comparison summary JSON path")
	timingThreshold := fs.Float64("timing-regression-percent", 10, "Allowed timing regression percentage")
	sizeThreshold := fs.Float64("size-regression-percent", 0, "Allowed size regression percentage")
	otherThreshold := fs.Float64("other-regression-percent", 0, "Allowed fallback regression percentage")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveWasmCompareConfig(wasmCompareConfig{
		baselinePath:            *baseline,
		candidatePath:           *candidate,
		outFile:                 *outFile,
		timingRegressionPercent: *timingThreshold,
		sizeRegressionPercent:   *sizeThreshold,
		otherRegressionPercent:  *otherThreshold,
		json:                    *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, regressionCount, compareErr := executeWasmCompare(config)
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		_ = encoder.Encode(summary)
	} else {
		printWasmCompareSummary(summary)
	}
	if compareErr != nil {
		return compareErr
	}
	if regressionCount > 0 {
		return fmt.Errorf("detected %d metric regressions beyond configured thresholds", regressionCount)
	}
	return nil
}

// runWasmCompareCompression compares plain, stripped, compressed, and optimized wasm variants.
func (l launcher) runWasmCompareCompression(args []string) error {
	fs := flag.NewFlagSet("wasm compare-compression", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	packagePath := fs.String("package", "./examples/21-ui-render", "Package path to build")
	outDir := fs.String("out-dir", filepath.Join("bin", "wasm-compression-comparison"), "Output directory for comparison artifacts")
	binaryName := fs.String("binary-name", "app.wasm", "Wasm artifact filename")
	summaryName := fs.String("summary-name", "wasm-compression-comparison.json", "Summary JSON filename")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	config, err := resolveWasmCompressionConfig(wasmCompressionConfig{
		packagePath: *packagePath,
		outDir:      *outDir,
		binaryName:  *binaryName,
		summaryName: *summaryName,
		json:        *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := executeWasmCompareCompression(config)
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(summary); encodeErr != nil {
			return encodeErr
		}
	} else {
		printWasmCompressionSummary(summary)
	}
	return err
}

// runWasmCompareCache compares wasm build behavior across cache topologies.
func (l launcher) runWasmCompareCache(args []string) error {
	fs := flag.NewFlagSet("wasm compare-cache", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	packagePath := fs.String("package", "./examples/21-ui-render", "Package path to build")
	outDir := fs.String("out-dir", filepath.Join("bin", "wasm-build-cache-comparison"), "Output directory for comparison artifacts")
	binaryName := fs.String("binary-name", "app.wasm", "Wasm artifact filename")
	summaryName := fs.String("summary-name", "wasm-build-cache-comparison.json", "Summary JSON filename")
	releaseProfile := fs.Bool("release-profile", false, "Use release-style wasm build profile")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	config, err := resolveWasmCacheConfig(wasmCacheConfig{
		packagePath:    *packagePath,
		outDir:         *outDir,
		binaryName:     *binaryName,
		summaryName:    *summaryName,
		releaseProfile: *releaseProfile,
		json:           *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := executeWasmCompareCache(config)
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(summary); encodeErr != nil {
			return encodeErr
		}
	} else {
		printWasmCacheSummary(summary)
	}
	return err
}

// runWasmCompareToolchain compares wasm build outputs across two Go toolchains.
func (l launcher) runWasmCompareToolchain(args []string) error {
	fs := flag.NewFlagSet("wasm compare-toolchain", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	packagePath := fs.String("package", "", "Package path to build (required)")
	baselineGo := fs.String("baseline-go", "", "Baseline Go executable (required)")
	candidateGo := fs.String("candidate-go", "", "Candidate Go executable (required)")
	binaryName := fs.String("binary-name", "app.wasm", "Wasm artifact filename")
	outDir := fs.String("out-dir", filepath.Join("bin", "wasm-toolchain-comparison"), "Output directory for comparison artifacts")
	timingThreshold := fs.Float64("timing-regression-percent", 10, "Allowed timing regression percentage")
	sizeThreshold := fs.Float64("size-regression-percent", 0, "Allowed size regression percentage")
	otherThreshold := fs.Float64("other-regression-percent", 0, "Allowed fallback regression percentage")
	releaseProfile := fs.Bool("release-profile", false, "Use release-style wasm build profile")
	skipCompression := fs.Bool("skip-compression", false, "Skip gzip and brotli sidecar generation during measurement")
	jsonOutput := fs.Bool("json", false, "Emit machine-readable JSON output")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}
	config, err := resolveWasmToolchainConfig(wasmToolchainConfig{
		packagePath:             *packagePath,
		baselineGoExecutable:    *baselineGo,
		candidateGoExecutable:   *candidateGo,
		binaryName:              *binaryName,
		outDir:                  *outDir,
		timingRegressionPercent: *timingThreshold,
		sizeRegressionPercent:   *sizeThreshold,
		otherRegressionPercent:  *otherThreshold,
		releaseProfile:          *releaseProfile,
		skipCompression:         *skipCompression,
		json:                    *jsonOutput,
	})
	if err != nil {
		return err
	}

	summary, err := executeWasmCompareToolchain(config)
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if encodeErr := encoder.Encode(summary); encodeErr != nil {
			return encodeErr
		}
	} else {
		printWasmToolchainSummary(summary)
	}
	return err
}

// resolveWasmMeasureConfig validates and normalizes wasm measurement configuration.
func resolveWasmMeasureConfig(config wasmMeasureConfig) (wasmMeasureConfig, error) {
	packagePath := strings.TrimSpace(config.packagePath)
	if packagePath == "" {
		packagePath = "."
	}
	outDir := strings.TrimSpace(config.outDir)
	if outDir == "" {
		outDir = filepath.Join("bin", "wasm-build-experiment")
	}
	cwd, err := os.Getwd()
	if err != nil {
		return wasmMeasureConfig{}, fmt.Errorf("resolve wasm measure cwd: %w", err)
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(cwd, outDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return wasmMeasureConfig{}, fmt.Errorf("create wasm measure out dir: %w", err)
	}
	binaryName := strings.TrimSpace(config.binaryName)
	if binaryName == "" {
		binaryName = "app.wasm"
	}
	manifestName := strings.TrimSpace(config.manifestName)
	if manifestName == "" {
		manifestName = "wasm-build-experiment.json"
	}
	goExecutable := strings.TrimSpace(config.goExecutable)
	if goExecutable == "" {
		goExecutable = "go"
	}
	return wasmMeasureConfig{
		packagePath:     packagePath,
		outDir:          filepath.Clean(outDir),
		binaryName:      filepath.Base(binaryName),
		manifestName:    filepath.Base(manifestName),
		goExecutable:    goExecutable,
		ldflags:         strings.TrimSpace(config.ldflags),
		releaseProfile:  config.releaseProfile,
		skipCompression: config.skipCompression,
		serveReloadMs:   config.serveReloadMs,
		hasServeReload:  config.hasServeReload,
		json:            config.json,
	}, nil
}

// resolveWasmCompareConfig validates and normalizes wasm compare configuration.
func resolveWasmCompareConfig(config wasmCompareConfig) (wasmCompareConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return wasmCompareConfig{}, fmt.Errorf("resolve wasm compare cwd: %w", err)
	}
	baselinePath, err := normalizeExistingPath(cwd, strings.TrimSpace(config.baselinePath))
	if err != nil {
		return wasmCompareConfig{}, fmt.Errorf("resolve baseline path: %w", err)
	}
	candidatePath, err := normalizeExistingPath(cwd, strings.TrimSpace(config.candidatePath))
	if err != nil {
		return wasmCompareConfig{}, fmt.Errorf("resolve candidate path: %w", err)
	}
	outFile := strings.TrimSpace(config.outFile)
	if outFile != "" && !filepath.IsAbs(outFile) {
		outFile = filepath.Join(cwd, outFile)
	}
	if strings.TrimSpace(outFile) != "" {
		outFile = filepath.Clean(outFile)
	}
	return wasmCompareConfig{
		baselinePath:            baselinePath,
		candidatePath:           candidatePath,
		outFile:                 outFile,
		timingRegressionPercent: config.timingRegressionPercent,
		sizeRegressionPercent:   config.sizeRegressionPercent,
		otherRegressionPercent:  config.otherRegressionPercent,
		json:                    config.json,
	}, nil
}

// resolveWasmCompressionConfig validates and normalizes wasm compression comparison settings.
func resolveWasmCompressionConfig(config wasmCompressionConfig) (wasmCompressionConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return wasmCompressionConfig{}, fmt.Errorf("resolve wasm compression cwd: %w", err)
	}
	outDir := strings.TrimSpace(config.outDir)
	if outDir == "" {
		outDir = filepath.Join("bin", "wasm-compression-comparison")
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(cwd, outDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return wasmCompressionConfig{}, fmt.Errorf("create wasm compression out dir: %w", err)
	}
	packagePath := strings.TrimSpace(config.packagePath)
	if packagePath == "" {
		packagePath = "."
	}
	binaryName := strings.TrimSpace(config.binaryName)
	if binaryName == "" {
		binaryName = "app.wasm"
	}
	summaryName := strings.TrimSpace(config.summaryName)
	if summaryName == "" {
		summaryName = "wasm-compression-comparison.json"
	}
	return wasmCompressionConfig{
		packagePath: packagePath,
		outDir:      filepath.Clean(outDir),
		binaryName:  filepath.Base(binaryName),
		summaryName: filepath.Base(summaryName),
		json:        config.json,
	}, nil
}

// resolveWasmCacheConfig validates and normalizes wasm cache comparison settings.
func resolveWasmCacheConfig(config wasmCacheConfig) (wasmCacheConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return wasmCacheConfig{}, fmt.Errorf("resolve wasm cache cwd: %w", err)
	}
	outDir := strings.TrimSpace(config.outDir)
	if outDir == "" {
		outDir = filepath.Join("bin", "wasm-build-cache-comparison")
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(cwd, outDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return wasmCacheConfig{}, fmt.Errorf("create wasm cache out dir: %w", err)
	}
	packagePath := strings.TrimSpace(config.packagePath)
	if packagePath == "" {
		packagePath = "."
	}
	binaryName := strings.TrimSpace(config.binaryName)
	if binaryName == "" {
		binaryName = "app.wasm"
	}
	summaryName := strings.TrimSpace(config.summaryName)
	if summaryName == "" {
		summaryName = "wasm-build-cache-comparison.json"
	}
	return wasmCacheConfig{
		packagePath:    packagePath,
		outDir:         filepath.Clean(outDir),
		binaryName:     filepath.Base(binaryName),
		summaryName:    filepath.Base(summaryName),
		releaseProfile: config.releaseProfile,
		json:           config.json,
	}, nil
}

// resolveWasmToolchainConfig validates and normalizes toolchain comparison settings.
func resolveWasmToolchainConfig(config wasmToolchainConfig) (wasmToolchainConfig, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return wasmToolchainConfig{}, fmt.Errorf("resolve wasm toolchain cwd: %w", err)
	}
	packagePath := strings.TrimSpace(config.packagePath)
	if packagePath == "" {
		return wasmToolchainConfig{}, errors.New("wasm compare-toolchain requires -package")
	}
	baselineGo := strings.TrimSpace(config.baselineGoExecutable)
	if baselineGo == "" {
		return wasmToolchainConfig{}, errors.New("wasm compare-toolchain requires -baseline-go")
	}
	candidateGo := strings.TrimSpace(config.candidateGoExecutable)
	if candidateGo == "" {
		return wasmToolchainConfig{}, errors.New("wasm compare-toolchain requires -candidate-go")
	}
	binaryName := strings.TrimSpace(config.binaryName)
	if binaryName == "" {
		binaryName = "app.wasm"
	}
	outDir := strings.TrimSpace(config.outDir)
	if outDir == "" {
		outDir = filepath.Join("bin", "wasm-toolchain-comparison")
	}
	if !filepath.IsAbs(outDir) {
		outDir = filepath.Join(cwd, outDir)
	}
	if err := os.MkdirAll(outDir, 0755); err != nil {
		return wasmToolchainConfig{}, fmt.Errorf("create wasm toolchain out dir: %w", err)
	}
	return wasmToolchainConfig{
		packagePath:             packagePath,
		baselineGoExecutable:    baselineGo,
		candidateGoExecutable:   candidateGo,
		binaryName:              filepath.Base(binaryName),
		outDir:                  filepath.Clean(outDir),
		timingRegressionPercent: config.timingRegressionPercent,
		sizeRegressionPercent:   config.sizeRegressionPercent,
		otherRegressionPercent:  config.otherRegressionPercent,
		releaseProfile:          config.releaseProfile,
		skipCompression:         config.skipCompression,
		json:                    config.json,
	}, nil
}

// executeWasmMeasure runs the measured build and writes the manifest artifact.
func executeWasmMeasure(config wasmMeasureConfig) (wasmMeasureSummary, error) {
	wasmPath := filepath.Join(config.outDir, config.binaryName)
	manifestPath := filepath.Join(config.outDir, config.manifestName)
	buildArgs := []string{"build", "-o", wasmPath}
	if config.releaseProfile {
		buildArgs = append(buildArgs, "-trimpath")
		if strings.TrimSpace(config.ldflags) != "" {
			buildArgs = append(buildArgs, "-ldflags="+config.ldflags)
		}
		buildArgs = append(buildArgs, "-buildvcs=false")
	}
	buildArgs = append(buildArgs, config.packagePath)

	totalStart := time.Now()
	buildStart := time.Now()
	_, buildErr := wasmRunCommand(config.goExecutable, buildArgs, "", append(buildNativeGoEnv(), "GOOS=js", "GOARCH=wasm"))
	buildDurationMs := time.Since(buildStart).Milliseconds()
	if buildErr != nil {
		return wasmMeasureSummary{}, buildErr
	}

	artifacts := map[string]releaseArtifactRecord{}
	wasmArtifact, err := wasmReleaseArtifactRecordForPath(config.outDir, wasmPath)
	if err != nil {
		return wasmMeasureSummary{}, err
	}
	artifacts["wasm"] = wasmArtifact

	phases := map[string]int64{
		"go_build_ms":          buildDurationMs,
		"compression_total_ms": 0,
	}
	if !config.skipCompression {
		compressionStart := time.Now()
		gzipPath := wasmPath + ".gz"
		gzipStart := time.Now()
		if err := wasmWriteGzipSidecar(wasmPath, gzipPath); err != nil {
			return wasmMeasureSummary{}, err
		}
		phases["gzip_ms"] = time.Since(gzipStart).Milliseconds()
		gzipArtifact, err := wasmReleaseArtifactRecordForPath(config.outDir, gzipPath)
		if err != nil {
			return wasmMeasureSummary{}, err
		}
		artifacts["gzip"] = gzipArtifact

		brotliPath := wasmPath + ".br"
		brotliStart := time.Now()
		if err := wasmWriteBrotliSidecar(wasmPath, brotliPath); err != nil {
			return wasmMeasureSummary{}, err
		}
		phases["brotli_ms"] = time.Since(brotliStart).Milliseconds()
		brotliArtifact, err := wasmReleaseArtifactRecordForPath(config.outDir, brotliPath)
		if err != nil {
			return wasmMeasureSummary{}, err
		}
		artifacts["brotli"] = brotliArtifact
		phases["compression_total_ms"] = time.Since(compressionStart).Milliseconds()
	}
	if config.hasServeReload {
		phases["serve_reload_ms"] = config.serveReloadMs
	}
	phases["total_wall_ms"] = time.Since(totalStart).Milliseconds()

	goVersionOutput, versionErr := wasmRunCommand(config.goExecutable, []string{"version"}, "", buildNativeGoEnv())
	if versionErr != nil {
		return wasmMeasureSummary{}, versionErr
	}
	manifest := wasmMeasureManifest{
		Package:      config.packagePath,
		Profile:      wasmMeasureProfileName(config.releaseProfile),
		GoExecutable: config.goExecutable,
		GoVersion:    strings.TrimSpace(goVersionOutput),
		GOOS:         "js",
		GOARCH:       "wasm",
		BuildArgs:    append([]string(nil), buildArgs...),
		Phases:       phases,
		Artifacts:    artifacts,
	}
	payload, err := json.MarshalIndent(manifest, "", "  ")
	if err != nil {
		return wasmMeasureSummary{}, fmt.Errorf("marshal wasm measure manifest: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(manifestPath, payload, 0644); err != nil {
		return wasmMeasureSummary{}, fmt.Errorf("write wasm measure manifest: %w", err)
	}
	return wasmMeasureSummary{
		OK:           true,
		OutDir:       filepath.ToSlash(config.outDir),
		ManifestPath: filepath.ToSlash(manifestPath),
		Manifest:     manifest,
	}, nil
}

// executeWasmCompare compares numeric metrics between two manifest files.
func executeWasmCompare(config wasmCompareConfig) (wasmCompareSummary, int, error) {
	baselinePayload, err := os.ReadFile(config.baselinePath)
	if err != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("read baseline manifest: %w", err)
	}
	candidatePayload, err := os.ReadFile(config.candidatePath)
	if err != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("read candidate manifest: %w", err)
	}
	var baselineData interface{}
	var candidateData interface{}
	if err := json.Unmarshal(baselinePayload, &baselineData); err != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("parse baseline manifest: %w", err)
	}
	if err := json.Unmarshal(candidatePayload, &candidateData); err != nil {
		return wasmCompareSummary{}, 0, fmt.Errorf("parse candidate manifest: %w", err)
	}

	baselineMetrics := map[string]float64{}
	candidateMetrics := map[string]float64{}
	collectWasmNumericMetrics(baselineData, "", baselineMetrics)
	collectWasmNumericMetrics(candidateData, "", candidateMetrics)

	pathSet := map[string]struct{}{}
	for key := range baselineMetrics {
		pathSet[key] = struct{}{}
	}
	for key := range candidateMetrics {
		pathSet[key] = struct{}{}
	}
	paths := make([]string, 0, len(pathSet))
	for key := range pathSet {
		paths = append(paths, key)
	}
	sort.Strings(paths)

	metrics := make([]wasmCompareMetric, 0, len(paths))
	counts := wasmCompareCounts{}
	for _, path := range paths {
		counts.Total++
		baselineValue, hasBaseline := baselineMetrics[path]
		candidateValue, hasCandidate := candidateMetrics[path]
		category := wasmMetricCategory(path)
		if !hasBaseline {
			candidateCopy := candidateValue
			metrics = append(metrics, wasmCompareMetric{
				Path:      path,
				Status:    "added",
				Category:  category,
				Candidate: &candidateCopy,
			})
			counts.Added++
			continue
		}
		if !hasCandidate {
			baselineCopy := baselineValue
			metrics = append(metrics, wasmCompareMetric{
				Path:     path,
				Status:   "removed",
				Category: category,
				Baseline: &baselineCopy,
			})
			counts.Removed++
			continue
		}
		delta := candidateValue - baselineValue
		threshold := wasmMetricThreshold(path, config)
		var deltaPercent *float64
		if baselineValue == 0 {
			if candidateValue == 0 {
				zero := 0.0
				deltaPercent = &zero
			}
		} else {
			value := math.Round(((delta/baselineValue)*100)*10000) / 10000
			deltaPercent = &value
		}
		status := "unchanged"
		switch {
		case delta < 0:
			status = "improved"
			counts.Improved++
		case delta > 0:
			if deltaPercent != nil && *deltaPercent > threshold {
				status = "regressed"
				counts.Regressed++
			} else {
				status = "within-threshold"
				counts.WithinThreshold++
			}
		default:
			counts.Unchanged++
		}
		baselineCopy := baselineValue
		candidateCopy := candidateValue
		deltaCopy := delta
		thresholdCopy := threshold
		metrics = append(metrics, wasmCompareMetric{
			Path:             path,
			Status:           status,
			Category:         category,
			Baseline:         &baselineCopy,
			Candidate:        &candidateCopy,
			Delta:            &deltaCopy,
			DeltaPercent:     deltaPercent,
			ThresholdPercent: &thresholdCopy,
		})
	}

	summary := wasmCompareSummary{
		OK:         counts.Regressed == 0,
		Baseline:   filepath.ToSlash(config.baselinePath),
		Candidate:  filepath.ToSlash(config.candidatePath),
		ComparedAt: time.Now().UTC().Format(time.RFC3339),
		Thresholds: wasmCompareThresholds{
			TimingRegressionPercent: config.timingRegressionPercent,
			SizeRegressionPercent:   config.sizeRegressionPercent,
			OtherRegressionPercent:  config.otherRegressionPercent,
		},
		Counts:  counts,
		Metrics: metrics,
	}
	if strings.TrimSpace(config.outFile) != "" {
		if err := os.MkdirAll(filepath.Dir(config.outFile), 0755); err != nil {
			return wasmCompareSummary{}, 0, fmt.Errorf("create compare output dir: %w", err)
		}
		payload, err := json.MarshalIndent(summary, "", "  ")
		if err != nil {
			return wasmCompareSummary{}, 0, fmt.Errorf("marshal compare summary: %w", err)
		}
		payload = append(payload, '\n')
		if err := os.WriteFile(config.outFile, payload, 0644); err != nil {
			return wasmCompareSummary{}, 0, fmt.Errorf("write compare summary: %w", err)
		}
	}
	return summary, counts.Regressed, nil
}

// executeWasmCompareCompression runs multiple wasm build variants and saves a comparison summary.
func executeWasmCompareCompression(config wasmCompressionConfig) (wasmCompressionSummary, error) {
	variants := map[string]interface{}{}

	plainRaw, err := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     config.packagePath,
		outDir:          filepath.Join(config.outDir, "plain-raw"),
		binaryName:      config.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  false,
		skipCompression: true,
	})
	if err != nil {
		return wasmCompressionSummary{}, err
	}
	variants["plain_raw"] = plainRaw.Manifest

	strippedRaw, err := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     config.packagePath,
		outDir:          filepath.Join(config.outDir, "stripped-raw"),
		binaryName:      config.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  true,
		skipCompression: true,
	})
	if err != nil {
		return wasmCompressionSummary{}, err
	}
	variants["stripped_raw"] = strippedRaw.Manifest

	strippedCompressed, err := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     config.packagePath,
		outDir:          filepath.Join(config.outDir, "stripped-compressed"),
		binaryName:      config.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  true,
		skipCompression: false,
	})
	if err != nil {
		return wasmCompressionSummary{}, err
	}
	variants["stripped_compressed"] = strippedCompressed.Manifest
	_, brotliSupported := strippedCompressed.Manifest.Artifacts["brotli"]

	optimizerCommand, err := resolveWasmOptimizerCommand()
	if err != nil {
		return wasmCompressionSummary{}, err
	}
	unsupported := map[string]string{}
	if brotliSupported {
		unsupported["brotli_delivery"] = "supported"
	} else {
		unsupported["brotli_delivery"] = "brotli artifact unavailable on current host"
	}
	if optimizerCommand.Available {
		unsupported["optimized_wasm"] = "supported"
	} else {
		unsupported["optimized_wasm"] = "wasm-opt is unavailable on PATH"
	}
	if optimizerCommand.Available {
		optimizedRawVariant, optimizedCompressedVariant, variantErr := executeWasmOptimizedVariants(config, strippedRaw, optimizerCommand, brotliSupported)
		if variantErr != nil {
			return wasmCompressionSummary{}, variantErr
		}
		variants["optimized_raw"] = optimizedRawVariant
		variants["optimized_compressed"] = optimizedCompressedVariant
	}

	goVersionOutput, err := wasmRunCommand("go", []string{"version"}, "", buildNativeGoEnv())
	if err != nil {
		return wasmCompressionSummary{}, err
	}
	summaryPath := filepath.Join(config.outDir, config.summaryName)
	summary := wasmCompressionSummary{
		OK:          true,
		Package:     config.packagePath,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Environment: wasmCompressionEnvironment{
			GoVersion:        strings.TrimSpace(goVersionOutput),
			BrotliSupported:  brotliSupported,
			WasmOptAvailable: optimizerCommand.Available,
			WasmOptPath:      optimizerCommand.Label,
		},
		Variants:    variants,
		Unsupported: unsupported,
		SummaryPath: filepath.ToSlash(summaryPath),
	}
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return wasmCompressionSummary{}, fmt.Errorf("marshal wasm compression summary: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(summaryPath, payload, 0644); err != nil {
		return wasmCompressionSummary{}, fmt.Errorf("write wasm compression summary: %w", err)
	}
	return summary, nil
}

// executeWasmCompareCache runs shared-cache and CI-style cache variants for wasm builds.
func executeWasmCompareCache(config wasmCacheConfig) (wasmCacheSummary, error) {
	sharedGoCache := filepath.Join(config.outDir, "shared-gocache")
	ciGoCache := filepath.Join(config.outDir, "ci-gocache")
	ciGoModCache := filepath.Join(config.outDir, "ci-gomodcache")
	isolatedGoCache := filepath.Join(config.outDir, "isolated-gocache")
	for _, path := range []string{sharedGoCache, ciGoCache, ciGoModCache, isolatedGoCache} {
		if err := resetWasmDirectory(path); err != nil {
			return wasmCacheSummary{}, err
		}
	}

	defaultGoCacheOutput, err := wasmRunCommand("go", []string{"env", "GOCACHE"}, "", buildNativeGoEnv())
	if err != nil {
		return wasmCacheSummary{}, err
	}
	defaultGoModOutput, err := wasmRunCommand("go", []string{"env", "GOMODCACHE"}, "", buildNativeGoEnv())
	if err != nil {
		return wasmCacheSummary{}, err
	}
	defaultGoCache := strings.TrimSpace(defaultGoCacheOutput)
	defaultGoMod := strings.TrimSpace(defaultGoModOutput)

	variantSpecs := []wasmCacheVariantSpec{
		{
			key:        "shared-cache-cold",
			label:      "shared cache cold",
			goCache:    sharedGoCache,
			goModCache: defaultGoMod,
			note:       "Empty dedicated build cache with the normal module cache.",
		},
		{
			key:        "shared-cache-warm",
			label:      "shared cache warm",
			goCache:    sharedGoCache,
			goModCache: defaultGoMod,
			note:       "Repeat build with the same dedicated build cache and reused module cache.",
		},
		{
			key:        "shared-cache-small-edit",
			label:      "shared cache small edit",
			goCache:    sharedGoCache,
			goModCache: defaultGoMod,
			note:       "Small edit rebuild with the warmed shared build cache and reused module cache.",
			smallEdit:  true,
		},
		{
			key:        "isolated-build-cache",
			label:      "isolated build cache",
			goCache:    isolatedGoCache,
			goModCache: defaultGoMod,
			note:       "Fresh build cache with the normal module cache, approximating a cold compile on a prepared machine.",
		},
		{
			key:                "ci-style-cold",
			label:              "ci-style cold",
			goCache:            ciGoCache,
			goModCache:         ciGoModCache,
			note:               "Fresh build cache and fresh module cache, approximating a clean CI worker.",
			prepareModuleCache: true,
			resetBeforeRun:     true,
		},
		{
			key:        "ci-style-warm",
			label:      "ci-style warm",
			goCache:    ciGoCache,
			goModCache: ciGoModCache,
			note:       "Repeat build after the CI-style caches were hydrated once in the same comparison run.",
		},
		{
			key:        "ci-style-small-edit",
			label:      "ci-style small edit",
			goCache:    ciGoCache,
			goModCache: ciGoModCache,
			note:       "Small edit rebuild after the CI-style caches were hydrated once in the same comparison run.",
			smallEdit:  true,
		},
	}

	variants := map[string]wasmCacheVariantResult{}
	overallOK := true
	for _, spec := range variantSpecs {
		if spec.resetBeforeRun {
			if err := resetWasmDirectory(spec.goCache); err != nil {
				return wasmCacheSummary{}, err
			}
			if err := resetWasmDirectory(spec.goModCache); err != nil {
				return wasmCacheSummary{}, err
			}
		}
		result, err := executeWasmCacheVariant(config, spec)
		if err != nil {
			return wasmCacheSummary{}, err
		}
		variants[spec.key] = result
		if result.Status != "ok" {
			overallOK = false
		}
	}

	goVersionOutput, err := wasmRunCommand("go", []string{"version"}, "", buildNativeGoEnv())
	if err != nil {
		return wasmCacheSummary{}, err
	}
	summaryPath := filepath.Join(config.outDir, config.summaryName)
	summary := wasmCacheSummary{
		OK:          overallOK,
		Package:     config.packagePath,
		GeneratedAt: time.Now().UTC().Format(time.RFC3339),
		Environment: wasmCacheEnvironment{
			GoVersion:      strings.TrimSpace(goVersionOutput),
			DefaultGoCache: defaultGoCache,
			DefaultGoMod:   defaultGoMod,
		},
		Variants:    variants,
		SummaryPath: filepath.ToSlash(summaryPath),
	}
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return wasmCacheSummary{}, fmt.Errorf("marshal wasm cache summary: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(summaryPath, payload, 0644); err != nil {
		return wasmCacheSummary{}, fmt.Errorf("write wasm cache summary: %w", err)
	}
	if !summary.OK {
		return summary, errors.New("one or more cache variants failed to measure")
	}
	return summary, nil
}

// executeWasmCompareToolchain measures baseline and candidate Go toolchains and compares results.
func executeWasmCompareToolchain(config wasmToolchainConfig) (wasmToolchainSummary, error) {
	baselineOutDir := filepath.Join(config.outDir, "baseline")
	candidateOutDir := filepath.Join(config.outDir, "candidate")
	comparisonPath := filepath.Join(config.outDir, "toolchain-comparison.json")
	summaryPath := filepath.Join(config.outDir, "wasm-toolchain-comparison.json")

	_, err := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     config.packagePath,
		outDir:          baselineOutDir,
		binaryName:      config.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    config.baselineGoExecutable,
		ldflags:         "-s -w",
		releaseProfile:  config.releaseProfile,
		skipCompression: config.skipCompression,
	})
	if err != nil {
		return wasmToolchainSummary{}, err
	}
	_, err = executeWasmMeasure(wasmMeasureConfig{
		packagePath:     config.packagePath,
		outDir:          candidateOutDir,
		binaryName:      config.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    config.candidateGoExecutable,
		ldflags:         "-s -w",
		releaseProfile:  config.releaseProfile,
		skipCompression: config.skipCompression,
	})
	if err != nil {
		return wasmToolchainSummary{}, err
	}

	baselineManifestPath := filepath.Join(baselineOutDir, "wasm-build-experiment.json")
	candidateManifestPath := filepath.Join(candidateOutDir, "wasm-build-experiment.json")
	_, regressionCount, compareErr := executeWasmCompare(wasmCompareConfig{
		baselinePath:            baselineManifestPath,
		candidatePath:           candidateManifestPath,
		outFile:                 comparisonPath,
		timingRegressionPercent: config.timingRegressionPercent,
		sizeRegressionPercent:   config.sizeRegressionPercent,
		otherRegressionPercent:  config.otherRegressionPercent,
	})
	regressionExitCode := 0
	if compareErr != nil || regressionCount > 0 {
		regressionExitCode = 1
	}

	baselineVersion, err := wasmRunCommand(config.baselineGoExecutable, []string{"version"}, "", buildNativeGoEnv())
	if err != nil {
		return wasmToolchainSummary{}, err
	}
	candidateVersion, err := wasmRunCommand(config.candidateGoExecutable, []string{"version"}, "", buildNativeGoEnv())
	if err != nil {
		return wasmToolchainSummary{}, err
	}

	summary := wasmToolchainSummary{
		OK:         regressionExitCode == 0,
		Package:    config.packagePath,
		ComparedAt: time.Now().UTC().Format(time.RFC3339),
		Baseline: wasmToolchainParty{
			GoExecutable: config.baselineGoExecutable,
			GoVersion:    strings.TrimSpace(baselineVersion),
			Manifest:     "baseline/wasm-build-experiment.json",
		},
		Candidate: wasmToolchainParty{
			GoExecutable: config.candidateGoExecutable,
			GoVersion:    strings.TrimSpace(candidateVersion),
			Manifest:     "candidate/wasm-build-experiment.json",
		},
		Thresholds: wasmCompareThresholds{
			TimingRegressionPercent: config.timingRegressionPercent,
			SizeRegressionPercent:   config.sizeRegressionPercent,
			OtherRegressionPercent:  config.otherRegressionPercent,
		},
		Comparison:         "toolchain-comparison.json",
		RegressionExitCode: regressionExitCode,
		SummaryPath:        filepath.ToSlash(summaryPath),
	}
	payload, err := json.MarshalIndent(summary, "", "  ")
	if err != nil {
		return wasmToolchainSummary{}, fmt.Errorf("marshal wasm toolchain summary: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(summaryPath, payload, 0644); err != nil {
		return wasmToolchainSummary{}, fmt.Errorf("write wasm toolchain summary: %w", err)
	}
	if compareErr != nil {
		return summary, compareErr
	}
	if regressionCount > 0 {
		return summary, fmt.Errorf("detected %d metric regressions beyond configured thresholds", regressionCount)
	}
	return summary, nil
}

// executeWasmCacheVariant runs one cache topology variant and returns its structured result.
func executeWasmCacheVariant(config wasmCacheConfig, spec wasmCacheVariantSpec) (wasmCacheVariantResult, error) {
	variantOutDir := filepath.Join(config.outDir, spec.key)
	if err := os.MkdirAll(variantOutDir, 0755); err != nil {
		return wasmCacheVariantResult{}, fmt.Errorf("create cache variant output dir: %w", err)
	}
	result := wasmCacheVariantResult{
		Label:      spec.label,
		GoCache:    spec.goCache,
		GoModCache: spec.goModCache,
		Status:     "pending",
		Notes:      []string{spec.note},
	}

	restoreGoCache := setWasmEnv("GOCACHE", spec.goCache)
	defer restoreGoCache()
	restoreGoMod := setWasmEnv("GOMODCACHE", spec.goModCache)
	defer restoreGoMod()

	if spec.prepareModuleCache {
		downloadStart := time.Now()
		_, err := wasmRunCommand("go", []string{"mod", "download"}, "", buildNativeGoEnv())
		downloadMs := time.Since(downloadStart).Milliseconds()
		result.ModuleDownloadMS = &downloadMs
		if err != nil {
			result.Status = "failed"
			result.Error = "go mod download failed"
			return result, nil
		}
	}

	restoreEdit, editedFile, editErr := applyWasmSmallEdit(config.packagePath, spec.smallEdit)
	if editErr != nil {
		result.Status = "failed"
		result.Error = editErr.Error()
		return result, nil
	}
	if restoreEdit != nil {
		defer restoreEdit()
		result.EditedFile = filepath.ToSlash(editedFile)
		result.Notes = append(result.Notes, "A temporary comment edit was applied and restored to measure rebuild invalidation.")
	}

	measurement, err := executeWasmMeasure(wasmMeasureConfig{
		packagePath:     config.packagePath,
		outDir:          variantOutDir,
		binaryName:      config.binaryName,
		manifestName:    "wasm-build-experiment.json",
		goExecutable:    "go",
		ldflags:         "-s -w",
		releaseProfile:  config.releaseProfile,
		skipCompression: false,
	})
	if err != nil {
		result.Status = "failed"
		result.Error = "wasm measurement failed"
		return result, nil
	}
	result.Status = "ok"
	result.Measurement = &measurement.Manifest
	return result, nil
}

// resetWasmDirectory clears and recreates one directory.
func resetWasmDirectory(path string) error {
	if strings.TrimSpace(path) == "" {
		return errors.New("cannot reset an empty directory path")
	}
	if err := os.RemoveAll(path); err != nil {
		return fmt.Errorf("reset directory %s: %w", path, err)
	}
	if err := os.MkdirAll(path, 0755); err != nil {
		return fmt.Errorf("create directory %s: %w", path, err)
	}
	return nil
}

// setWasmEnv sets one environment variable and returns a restore function.
func setWasmEnv(name string, value string) func() {
	previous, hadPrevious := os.LookupEnv(name)
	_ = os.Setenv(name, value)
	return func() {
		if hadPrevious {
			_ = os.Setenv(name, previous)
			return
		}
		_ = os.Unsetenv(name)
	}
}

// applyWasmSmallEdit appends a temporary marker comment and returns a restore closure.
func applyWasmSmallEdit(packagePath string, enabled bool) (func(), string, error) {
	if !enabled {
		return nil, "", nil
	}
	editableFile, err := resolveWasmCacheEditableFile(packagePath)
	if err != nil {
		return nil, "", err
	}
	originalBytes, err := os.ReadFile(editableFile)
	if err != nil {
		return nil, "", fmt.Errorf("read editable file: %w", err)
	}
	updatedBytes := append(append([]byte(nil), originalBytes...), []byte("\n// cache experiment marker\n")...)
	if err := os.WriteFile(editableFile, updatedBytes, 0644); err != nil {
		return nil, "", fmt.Errorf("apply temporary cache edit: %w", err)
	}
	restore := func() {
		_ = os.WriteFile(editableFile, originalBytes, 0644)
	}
	return restore, editableFile, nil
}

// resolveWasmCacheEditableFile picks the first non-test Go file for small-edit variants.
func resolveWasmCacheEditableFile(packagePath string) (string, error) {
	cwd, err := os.Getwd()
	if err != nil {
		return "", fmt.Errorf("resolve wasm cache package cwd: %w", err)
	}
	resolvedPath, err := normalizePath(cwd, packagePath)
	if err != nil {
		return "", fmt.Errorf("resolve wasm cache package path: %w", err)
	}
	info, err := os.Stat(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("inspect wasm cache package path: %w", err)
	}
	if !info.IsDir() {
		return "", fmt.Errorf("small-edit measurement requires a package directory path: %s", packagePath)
	}
	entries, err := os.ReadDir(resolvedPath)
	if err != nil {
		return "", fmt.Errorf("read wasm cache package directory: %w", err)
	}
	names := make([]string, 0, len(entries))
	for _, entry := range entries {
		if entry.IsDir() {
			continue
		}
		name := entry.Name()
		if !strings.HasSuffix(name, ".go") || strings.HasSuffix(name, "_test.go") {
			continue
		}
		names = append(names, name)
	}
	sort.Strings(names)
	if len(names) == 0 {
		return "", fmt.Errorf("no editable Go source file found for package: %s", packagePath)
	}
	return filepath.Join(resolvedPath, names[0]), nil
}

// executeWasmOptimizedVariants creates optimized wasm variants from a stripped baseline.
func executeWasmOptimizedVariants(config wasmCompressionConfig, strippedRaw wasmMeasureSummary, command wasmOptimizerCommand, includeBrotli bool) (map[string]interface{}, map[string]interface{}, error) {
	strippedRawPath := filepath.Join(filepath.FromSlash(strippedRaw.OutDir), config.binaryName)
	optimizedRawDir := filepath.Join(config.outDir, "optimized-raw")
	optimizedCompressedDir := filepath.Join(config.outDir, "optimized-compressed")
	if err := os.MkdirAll(optimizedRawDir, 0755); err != nil {
		return nil, nil, err
	}
	if err := os.MkdirAll(optimizedCompressedDir, 0755); err != nil {
		return nil, nil, err
	}
	optimizedRawPath := filepath.Join(optimizedRawDir, config.binaryName)
	optimizedCompressedPath := filepath.Join(optimizedCompressedDir, config.binaryName)

	optStart := time.Now()
	if err := runWasmOptimizer(command, strippedRawPath, optimizedRawPath); err != nil {
		return nil, nil, err
	}
	wasmOptMs := time.Since(optStart).Milliseconds()
	goBuildMs := strippedRaw.Manifest.Phases["go_build_ms"]

	optimizedRawArtifact, err := wasmReleaseArtifactRecordForPath(optimizedRawDir, optimizedRawPath)
	if err != nil {
		return nil, nil, err
	}
	optimizedRaw := map[string]interface{}{
		"package":    config.packagePath,
		"profile":    "release-optimized",
		"go_version": strippedRaw.Manifest.GoVersion,
		"goos":       "js",
		"goarch":     "wasm",
		"build_args": strippedRaw.Manifest.BuildArgs,
		"optimizer":  map[string]interface{}{"tool": command.Label, "args": append(append([]string{}, command.PrefixArgs...), strippedRawPath, "-Oz", "-o", optimizedRawPath)},
		"phases":     map[string]int64{"go_build_ms": goBuildMs, "wasm_opt_ms": wasmOptMs, "compression_total_ms": 0, "total_wall_ms": goBuildMs + wasmOptMs},
		"artifacts":  map[string]releaseArtifactRecord{"wasm": optimizedRawArtifact},
	}

	optimizedBytes, err := os.ReadFile(optimizedRawPath)
	if err != nil {
		return nil, nil, err
	}
	if err := os.WriteFile(optimizedCompressedPath, optimizedBytes, 0644); err != nil {
		return nil, nil, err
	}
	if err := assertWasmFileParity(optimizedRawPath, optimizedCompressedPath); err != nil {
		return nil, nil, err
	}
	gzipStart := time.Now()
	optimizedGzipPath := optimizedCompressedPath + ".gz"
	if err := wasmWriteGzipSidecar(optimizedCompressedPath, optimizedGzipPath); err != nil {
		return nil, nil, err
	}
	gzipMs := time.Since(gzipStart).Milliseconds()
	compressionTotalMs := gzipMs

	artifacts := map[string]releaseArtifactRecord{}
	optimizedCompressedArtifact, err := wasmReleaseArtifactRecordForPath(optimizedCompressedDir, optimizedCompressedPath)
	if err != nil {
		return nil, nil, err
	}
	if err := assertWasmArtifactParity("optimized raw -> optimized compressed wasm copy", optimizedRawArtifact, optimizedCompressedArtifact); err != nil {
		return nil, nil, err
	}
	artifacts["wasm"] = optimizedCompressedArtifact
	optimizedGzipArtifact, err := wasmReleaseArtifactRecordForPath(optimizedCompressedDir, optimizedGzipPath)
	if err != nil {
		return nil, nil, err
	}
	artifacts["gzip"] = optimizedGzipArtifact

	phases := map[string]int64{
		"go_build_ms":          goBuildMs,
		"wasm_opt_ms":          wasmOptMs,
		"gzip_ms":              gzipMs,
		"compression_total_ms": compressionTotalMs,
	}
	if includeBrotli {
		brotliStart := time.Now()
		optimizedBrotliPath := optimizedCompressedPath + ".br"
		if err := wasmWriteBrotliSidecar(optimizedCompressedPath, optimizedBrotliPath); err != nil {
			return nil, nil, err
		}
		brotliMs := time.Since(brotliStart).Milliseconds()
		phases["brotli_ms"] = brotliMs
		compressionTotalMs += brotliMs
		phases["compression_total_ms"] = compressionTotalMs
		brotliArtifact, err := wasmReleaseArtifactRecordForPath(optimizedCompressedDir, optimizedBrotliPath)
		if err != nil {
			return nil, nil, err
		}
		artifacts["brotli"] = brotliArtifact
	}
	phases["total_wall_ms"] = goBuildMs + wasmOptMs + compressionTotalMs

	optimizedCompressed := map[string]interface{}{
		"package":       config.packagePath,
		"profile":       "release-optimized",
		"go_version":    strippedRaw.Manifest.GoVersion,
		"goos":          "js",
		"goarch":        "wasm",
		"build_args":    strippedRaw.Manifest.BuildArgs,
		"optimizer":     map[string]interface{}{"tool": command.Label, "args": append(append([]string{}, command.PrefixArgs...), strippedRawPath, "-Oz", "-o", optimizedCompressedPath)},
		"phases":        phases,
		"artifacts":     artifacts,
		"parity_checks": map[string]interface{}{"raw_to_delivery_copy": map[string]interface{}{"ok": true, "source_bytes": optimizedRawArtifact.Bytes, "target_bytes": optimizedCompressedArtifact.Bytes, "source_sha256": optimizedRawArtifact.SHA256, "target_sha256": optimizedCompressedArtifact.SHA256}},
	}
	return optimizedRaw, optimizedCompressed, nil
}

// assertWasmFileParity validates byte-for-byte equality for two artifact paths.
func assertWasmFileParity(sourcePath string, targetPath string) error {
	sourceBytes, err := os.ReadFile(sourcePath)
	if err != nil {
		return fmt.Errorf("read source artifact for parity check: %w", err)
	}
	targetBytes, err := os.ReadFile(targetPath)
	if err != nil {
		return fmt.Errorf("read target artifact for parity check: %w", err)
	}
	if !bytes.Equal(sourceBytes, targetBytes) {
		return fmt.Errorf("optimized wasm parity mismatch between %s and %s", filepath.ToSlash(sourcePath), filepath.ToSlash(targetPath))
	}
	return nil
}

// assertWasmArtifactParity validates byte-size and hash parity for two artifact records.
func assertWasmArtifactParity(label string, source releaseArtifactRecord, target releaseArtifactRecord) error {
	if source.Bytes != target.Bytes {
		return fmt.Errorf("%s parity mismatch: bytes %d != %d", label, source.Bytes, target.Bytes)
	}
	if strings.TrimSpace(source.SHA256) == "" || strings.TrimSpace(target.SHA256) == "" {
		return fmt.Errorf("%s parity check requires non-empty sha256 records", label)
	}
	if source.SHA256 != target.SHA256 {
		return fmt.Errorf("%s parity mismatch: sha256 %s != %s", label, source.SHA256, target.SHA256)
	}
	return nil
}

// resolveWasmOptimizerCommand discovers wasm-opt from PATH.
func resolveWasmOptimizerCommand() (wasmOptimizerCommand, error) {
	if path, err := wasmLookPath("wasm-opt"); err == nil {
		return wasmOptimizerCommand{
			Available: true,
			Command:   "wasm-opt",
			Label:     path,
		}, nil
	}
	return wasmOptimizerCommand{}, nil
}

// runWasmOptimizer executes wasm-opt for one artifact pair.
func runWasmOptimizer(command wasmOptimizerCommand, sourcePath string, targetPath string) error {
	args := append([]string{}, command.PrefixArgs...)
	args = append(args, sourcePath, "-Oz", "-o", targetPath)
	_, err := wasmRunCommand(command.Command, args, filepath.Dir(sourcePath), buildNativeGoEnv())
	if err != nil {
		return fmt.Errorf("run wasm optimizer: %w", err)
	}
	return nil
}

// collectWasmNumericMetrics flattens numeric JSON values into a path-to-value map.
func collectWasmNumericMetrics(value interface{}, path string, metrics map[string]float64) {
	switch typed := value.(type) {
	case map[string]interface{}:
		for key, child := range typed {
			childPath := key
			if strings.TrimSpace(path) != "" {
				childPath = path + "." + key
			}
			collectWasmNumericMetrics(child, childPath, metrics)
		}
	case []interface{}:
		for index, child := range typed {
			childPath := fmt.Sprintf("[%d]", index)
			if strings.TrimSpace(path) != "" {
				childPath = fmt.Sprintf("%s[%d]", path, index)
			}
			collectWasmNumericMetrics(child, childPath, metrics)
		}
	case float64:
		metrics[path] = typed
	}
}

// wasmMetricCategory classifies a flattened metric path.
func wasmMetricCategory(path string) string {
	trimmed := strings.TrimSpace(path)
	if strings.HasSuffix(trimmed, "_ms") || strings.Contains(trimmed, "module_download_ms") {
		return "timing"
	}
	if strings.HasSuffix(trimmed, "bytes") {
		return "size"
	}
	return "other"
}

// wasmMetricThreshold resolves the regression threshold for one metric path.
func wasmMetricThreshold(path string, config wasmCompareConfig) float64 {
	switch wasmMetricCategory(path) {
	case "timing":
		return config.timingRegressionPercent
	case "size":
		return config.sizeRegressionPercent
	default:
		return config.otherRegressionPercent
	}
}

// wasmMeasureProfileName returns the serialized profile label for wasm measurement manifests.
func wasmMeasureProfileName(releaseProfile bool) string {
	if releaseProfile {
		return "release"
	}
	return "debug"
}

// printWasmMeasureSummary prints a concise human-readable wasm measure result.
func printWasmMeasureSummary(summary wasmMeasureSummary) {
	fmt.Println("GWC wasm measure")
	fmt.Printf("  out dir:      %s\n", summary.OutDir)
	fmt.Printf("  manifest:     %s\n", summary.ManifestPath)
	fmt.Printf("  package:      %s\n", summary.Manifest.Package)
	fmt.Printf("  profile:      %s\n", summary.Manifest.Profile)
	fmt.Printf("  go version:   %s\n", summary.Manifest.GoVersion)
	if buildMs, ok := summary.Manifest.Phases["go_build_ms"]; ok {
		fmt.Printf("  go build ms:  %s\n", strconv.FormatInt(buildMs, 10))
	}
	keys := make([]string, 0, len(summary.Manifest.Artifacts))
	for key := range summary.Manifest.Artifacts {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		record := summary.Manifest.Artifacts[key]
		fmt.Printf("  artifact[%s]: %s (%d bytes)\n", key, record.Path, record.Bytes)
	}
}

// printWasmCompareSummary prints a concise human-readable comparison summary.
func printWasmCompareSummary(summary wasmCompareSummary) {
	fmt.Println("GWC wasm compare")
	fmt.Printf("  baseline:    %s\n", summary.Baseline)
	fmt.Printf("  candidate:   %s\n", summary.Candidate)
	fmt.Printf("  total:       %d\n", summary.Counts.Total)
	fmt.Printf("  improved:    %d\n", summary.Counts.Improved)
	fmt.Printf("  regressed:   %d\n", summary.Counts.Regressed)
	fmt.Printf("  unchanged:   %d\n", summary.Counts.Unchanged)
	fmt.Printf("  thresholded: %d\n", summary.Counts.WithinThreshold)
}

// printWasmCacheSummary prints a concise human-readable cache comparison summary.
func printWasmCacheSummary(summary wasmCacheSummary) {
	fmt.Println("GWC wasm compare-cache")
	fmt.Printf("  package:     %s\n", summary.Package)
	fmt.Printf("  generated:   %s\n", summary.GeneratedAt)
	fmt.Printf("  go version:  %s\n", summary.Environment.GoVersion)
	fmt.Printf("  gocache:     %s\n", summary.Environment.DefaultGoCache)
	fmt.Printf("  gomodcache:  %s\n", summary.Environment.DefaultGoMod)
	if strings.TrimSpace(summary.SummaryPath) != "" {
		fmt.Printf("  summary:     %s\n", summary.SummaryPath)
	}
	keys := make([]string, 0, len(summary.Variants))
	for key := range summary.Variants {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		variant := summary.Variants[key]
		fmt.Printf("  [%s] %s\n", variant.Status, key)
		if strings.TrimSpace(variant.Error) != "" {
			fmt.Printf("    error: %s\n", variant.Error)
		}
	}
}

// printWasmToolchainSummary prints a concise human-readable toolchain comparison summary.
func printWasmToolchainSummary(summary wasmToolchainSummary) {
	fmt.Println("GWC wasm compare-toolchain")
	fmt.Printf("  package:      %s\n", summary.Package)
	fmt.Printf("  compared:     %s\n", summary.ComparedAt)
	fmt.Printf("  baseline go:  %s\n", summary.Baseline.GoExecutable)
	fmt.Printf("  candidate go: %s\n", summary.Candidate.GoExecutable)
	fmt.Printf("  comparison:   %s\n", summary.Comparison)
	fmt.Printf("  regressions:  %d\n", summary.RegressionExitCode)
	if strings.TrimSpace(summary.SummaryPath) != "" {
		fmt.Printf("  summary:      %s\n", summary.SummaryPath)
	}
}

// printWasmCompressionSummary prints a concise human-readable compression comparison summary.
func printWasmCompressionSummary(summary wasmCompressionSummary) {
	fmt.Println("GWC wasm compare-compression")
	fmt.Printf("  package:     %s\n", summary.Package)
	fmt.Printf("  generated:   %s\n", summary.GeneratedAt)
	fmt.Printf("  go version:  %s\n", summary.Environment.GoVersion)
	fmt.Printf("  brotli:      %t\n", summary.Environment.BrotliSupported)
	fmt.Printf("  wasm-opt:    %t\n", summary.Environment.WasmOptAvailable)
	if strings.TrimSpace(summary.Environment.WasmOptPath) != "" {
		fmt.Printf("  wasm-opt id: %s\n", summary.Environment.WasmOptPath)
	}
	if strings.TrimSpace(summary.SummaryPath) != "" {
		fmt.Printf("  summary:     %s\n", summary.SummaryPath)
	}
	keys := make([]string, 0, len(summary.Variants))
	for key := range summary.Variants {
		keys = append(keys, key)
	}
	sort.Strings(keys)
	for _, key := range keys {
		fmt.Printf("  variant:     %s\n", key)
	}
}
