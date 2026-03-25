package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"time"
)

var runBenchmarkCommand = func(l launcher, args []string) error {
	return l.runBenchmark(args)
}

var benchmarkResolveWasmExec = resolveWasmTestExec

var benchmarkLookPath = exec.LookPath

var benchmarkRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
	return launcherRunCommand(command, args, cwd, env)
}

type benchmarkConfig struct {
	rootPath      string
	lanes         []string
	bench         string
	benchtime     string
	count         int
	parallel      int
	json          bool
	outPath       string
	referencePath string
}

type benchmarkCompareConfig struct {
	baselinePath  string
	candidatePath string
}

type benchmarkCaptureConfig struct {
	packagePath string
	count       int
	bench       string
	outputPath  string
	execPath    string
	json        bool
}

type benchmarkReport struct {
	OK                 bool                        `json:"ok"`
	Root               string                      `json:"root"`
	GeneratedAt        string                      `json:"generatedAt"`
	GoVersion          string                      `json:"goVersion"`
	GOOS               string                      `json:"goos"`
	GOARCH             string                      `json:"goarch"`
	SelectedLanes      []string                    `json:"selectedLanes"`
	Bench              string                      `json:"bench"`
	Count              int                         `json:"count"`
	PackageParallelism int                         `json:"packageParallelism"`
	Benchtime          string                      `json:"benchtime,omitempty"`
	ReportPath         string                      `json:"reportPath"`
	PackageCount       int                         `json:"packageCount"`
	BenchmarkCount     int                         `json:"benchmarkCount"`
	FailedPackages     int                         `json:"failedPackages"`
	Packages           []benchmarkPackageReport    `json:"packages"`
	Scores             *benchmarkScoreSummary      `json:"scores,omitempty"`
	Comparison         *benchmarkComparisonSummary `json:"comparison,omitempty"`
}

type benchmarkCompareSummary struct {
	OK                 bool   `json:"ok"`
	BenchstatAvailable bool   `json:"benchstatAvailable"`
	BaselinePath       string `json:"baselinePath"`
	CandidatePath      string `json:"candidatePath"`
	Output             string `json:"output,omitempty"`
	Message            string `json:"message,omitempty"`
}

type benchmarkCaptureSummary struct {
	OK         bool   `json:"ok"`
	Package    string `json:"package"`
	Count      int    `json:"count"`
	Bench      string `json:"bench"`
	OutputPath string `json:"outputPath"`
	Exec       string `json:"exec,omitempty"`
	Command    string `json:"command"`
}

type benchmarkPackageReport struct {
	Lane           string                  `json:"lane"`
	Package        string                  `json:"package"`
	Workspace      string                  `json:"workspace"`
	Command        string                  `json:"command"`
	OK             bool                    `json:"ok"`
	BenchmarkCount int                     `json:"benchmarkCount"`
	Benchmarks     []benchmarkResultReport `json:"benchmarks,omitempty"`
	Error          string                  `json:"error,omitempty"`
	Output         string                  `json:"output,omitempty"`
}

type benchmarkResultReport struct {
	Name           string             `json:"name"`
	Bucket         string             `json:"bucket,omitempty"`
	Samples        []benchmarkSample  `json:"samples"`
	AverageMetrics map[string]float64 `json:"averageMetrics,omitempty"`
}

type benchmarkScoreSummary struct {
	Method               string                 `json:"method"`
	ReferencePath        string                 `json:"referencePath"`
	ReferenceGeneratedAt string                 `json:"referenceGeneratedAt,omitempty"`
	ReferenceMachine     string                 `json:"referenceMachine,omitempty"`
	MatchedBenchmarks    int                    `json:"matchedBenchmarks"`
	OverallScore         float64                `json:"overallScore"`
	Buckets              []benchmarkBucketScore `json:"buckets"`
}

type benchmarkBucketScore struct {
	ID                string  `json:"id"`
	Label             string  `json:"label"`
	MatchedBenchmarks int     `json:"matchedBenchmarks"`
	Score             float64 `json:"score"`
}

type benchmarkSample struct {
	Iterations int64              `json:"iterations"`
	Metrics    map[string]float64 `json:"metrics"`
	Raw        string             `json:"raw"`
}

type benchmarkComparisonSummary struct {
	BaselineGeneratedAt string                      `json:"baselineGeneratedAt,omitempty"`
	TolerancePct        float64                     `json:"tolerancePct"`
	MatchedMetrics      int                         `json:"matchedMetrics"`
	Improved            int                         `json:"improved"`
	Regressed           int                         `json:"regressed"`
	Unchanged           int                         `json:"unchanged"`
	Entries             []benchmarkMetricComparison `json:"entries,omitempty"`
}

type benchmarkMetricComparison struct {
	Lane      string  `json:"lane"`
	Package   string  `json:"package"`
	Benchmark string  `json:"benchmark"`
	Metric    string  `json:"metric"`
	Baseline  float64 `json:"baseline"`
	Current   float64 `json:"current"`
	Delta     float64 `json:"delta"`
	DeltaPct  float64 `json:"deltaPct"`
	Direction string  `json:"direction"`
}

type benchmarkMetricKey struct {
	Lane      string
	Package   string
	Benchmark string
	Metric    string
}

type benchmarkPackageJob struct {
	index       int
	lane        string
	packagePath string
}

type benchmarkPackageResult struct {
	index  int
	report benchmarkPackageReport
}

const benchmarkComparisonTolerancePct = 2.0
const benchmarkScoreMethod = "100 * geometric_mean(reference_ns / measured_ns)"

func (parseL launcher) runBenchmark(parseArgs []string) error {
	if len(parseArgs) > 0 && strings.EqualFold(strings.TrimSpace(parseArgs[0]), "compare") {
		return parseL.runBenchmarkCompare(parseArgs[1:])
	}
	if len(parseArgs) > 0 && strings.EqualFold(strings.TrimSpace(parseArgs[0]), "capture") {
		return parseL.runBenchmarkCapture(parseArgs[1:])
	}

	parseFs := flag.NewFlagSet("bench", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseRoot := parseFs.String("root", "", "Root directory to inspect for benchmark packages; defaults to the current working directory")
	parseBenchPattern := parseFs.String("bench", ".", "Benchmark pattern passed to go test -bench")
	parseBenchtime := parseFs.String("benchtime", "", "Optional benchtime forwarded to go test")
	parseCount := parseFs.Int("count", 1, "Number of benchmark runs per package")
	parseParallel := parseFs.Int("parallel", 1, "Maximum number of benchmark packages to run concurrently; keep 1 for the lowest-noise regression tracking")
	parseOutPath := parseFs.String("out", "", "JSON report path; defaults to docs/benchmarks/latest.json beneath the root")
	parseReferencePath := parseFs.String("reference", "", "Optional reference benchmark JSON used for normalized scoring; defaults to docs/benchmarks/reference.json beneath the root")
	parseJsonOutput := parseFs.Bool("json", false, "Emit the JSON report to stdout after writing it to disk")
	var parseLanes stringListFlag
	parseFs.Var(&parseLanes, "lane", "Benchmark lanes: native, wasm, or all; repeatable")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := resolveBenchmarkConfig(benchmarkConfig{
		rootPath:      *parseRoot,
		lanes:         parseLanes.Values(),
		bench:         *parseBenchPattern,
		benchtime:     *parseBenchtime,
		count:         *parseCount,
		parallel:      *parseParallel,
		json:          *parseJsonOutput,
		outPath:       *parseOutPath,
		referencePath: *parseReferencePath,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parsePrevious, _ := loadBenchmarkReport(parseConfig.outPath)
	parseReference, _ := loadBenchmarkReport(parseConfig.referencePath)
	parseReport, parseRunErr := parseL.executeBenchmark(parseConfig)
	parseReport.ReportPath = benchmarkDisplayPath(parseConfig.rootPath, parseConfig.outPath)
	if parseReference.GeneratedAt != "" {
		parseReport.Scores = buildBenchmarkScoreSummary(parseReference, parseReport, parseConfig)
	}
	if parsePrevious.OK || parsePrevious.GeneratedAt != "" {
		parseReport.Comparison = compareBenchmarkReports(parsePrevious, parseReport)
	}
	if parseWriteErr := writeBenchmarkReport(parseConfig.outPath, parseReport); parseWriteErr != nil {
		return parseWriteErr
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseErr3 := parseEncoder.Encode(parseReport); parseErr3 != nil {
			return parseErr3
		}
	} else {
		printBenchmarkReport(parseReport)
	}
	if parseRunErr != nil {
		return parseRunErr
	}
	return nil
}

func (parseL launcher) runBenchmarkCompare(parseArgs []string) error {
	parseFs := flag.NewFlagSet("bench compare", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseBaseline := parseFs.String("baseline", "", "Path to baseline benchmark output file")
	parseCandidate := parseFs.String("candidate", "", "Path to candidate benchmark output file")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parsePositional := parseFs.Args()
	if len(parsePositional) > 2 {
		return errors.New("bench compare accepts at most two positional arguments: <baseline> <candidate>")
	}
	parseBaselinePath := strings.TrimSpace(*parseBaseline)
	parseCandidatePath := strings.TrimSpace(*parseCandidate)
	if len(parsePositional) > 0 {
		if parseBaselinePath != "" && parseCandidatePath != "" {
			return errors.New("bench compare received positional arguments but -baseline and -candidate are already set")
		}
		switch len(parsePositional) {
		case 1:
			if parseBaselinePath == "" {
				parseBaselinePath = strings.TrimSpace(parsePositional[0])
			} else if parseCandidatePath == "" {
				parseCandidatePath = strings.TrimSpace(parsePositional[0])
			}
		case 2:
			if parseBaselinePath != "" || parseCandidatePath != "" {
				return errors.New("bench compare positional shortcuts require either no flags or exactly one missing path")
			}
			parseBaselinePath = strings.TrimSpace(parsePositional[0])
			parseCandidatePath = strings.TrimSpace(parsePositional[1])
		}
	}

	parseConfig, parseErr2 := resolveBenchmarkCompareConfig(benchmarkCompareConfig{
		baselinePath:  parseBaselinePath,
		candidatePath: parseCandidatePath,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseSummary := benchmarkCompareSummary{
		OK:            true,
		BaselinePath:  filepath.ToSlash(parseConfig.baselinePath),
		CandidatePath: filepath.ToSlash(parseConfig.candidatePath),
	}

	if _, parseErr3 := benchmarkLookPath("benchstat"); parseErr3 != nil {
		parseSummary.BenchstatAvailable = false
		parseSummary.Message = "benchstat is not installed. Install with: go install golang.org/x/perf/cmd/benchstat@latest"
		if *parseJsonOutput {
			parseEncoder := json.NewEncoder(os.Stdout)
			parseEncoder.SetIndent("", "  ")
			return parseEncoder.Encode(parseSummary)
		}
		printBenchmarkCompareSummary(parseSummary)
		return nil
	}

	parseOutput, parseErr2 := benchmarkRunCommand("benchstat", []string{parseConfig.baselinePath, parseConfig.candidatePath}, "", buildNativeGoEnv())
	if parseErr2 != nil {
		parseSummary.OK = false
		parseSummary.BenchstatAvailable = true
		parseSummary.Output = strings.TrimSpace(parseOutput)
		if strings.TrimSpace(parseSummary.Output) == "" {
			parseSummary.Message = parseErr2.Error()
		}
		if *parseJsonOutput {
			parseEncoder2 := json.NewEncoder(os.Stdout)
			parseEncoder2.SetIndent("", "  ")
			_ = parseEncoder2.Encode(parseSummary)
		}
		return parseErr2
	}

	parseSummary.BenchstatAvailable = true
	parseSummary.Output = strings.TrimSpace(parseOutput)
	if *parseJsonOutput {
		parseEncoder3 := json.NewEncoder(os.Stdout)
		parseEncoder3.SetIndent("", "  ")
		return parseEncoder3.Encode(parseSummary)
	}
	printBenchmarkCompareSummary(parseSummary)
	return nil
}

// runBenchmarkCapture executes one focused benchmark run and writes raw output to a file.
func (parseL launcher) runBenchmarkCapture(parseArgs []string) error {
	parseFs := flag.NewFlagSet("bench capture", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parsePackagePath := parseFs.String("package", "./internal/runtime", "Package passed to go test")
	parseCount := parseFs.Int("count", 5, "Benchmark sample count passed to go test -count")
	parseBenchPattern := parseFs.String("bench", ".", "Benchmark pattern passed to go test -bench")
	parseOutput := parseFs.String("output", "", "Output file path for raw benchmark output")
	parseExecPath := parseFs.String("exec", "", "Optional go test -exec helper path")
	parseJsonOutput := parseFs.Bool("json", false, "Emit machine-readable JSON output")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr2 := parseL.resolveBenchmarkCaptureConfig(benchmarkCaptureConfig{
		packagePath: *parsePackagePath,
		count:       *parseCount,
		bench:       *parseBenchPattern,
		outputPath:  *parseOutput,
		execPath:    *parseExecPath,
		json:        *parseJsonOutput,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	parseCwd, parseErr2 := os.Getwd()
	if parseErr2 != nil {
		return fmt.Errorf("resolve benchmark capture cwd: %w", parseErr2)
	}
	parseGoArgs := []string{"test"}
	if strings.TrimSpace(parseConfig.execPath) != "" {
		parseGoArgs = append(parseGoArgs, "-exec", parseConfig.execPath)
	}
	parseGoArgs = append(parseGoArgs, parseConfig.packagePath, "-run", "^$", "-bench", parseConfig.bench, "-benchmem", "-count", strconv.Itoa(parseConfig.count))
	parseOutputText, parseRunErr := launcherRunCommand("go", parseGoArgs, parseCwd, buildNativeGoEnv())
	if parseWriteErr := os.WriteFile(parseConfig.outputPath, []byte(strings.TrimSpace(parseOutputText)+"\n"), 0644); parseWriteErr != nil {
		return fmt.Errorf("write benchmark capture output: %w", parseWriteErr)
	}

	parseSummary := benchmarkCaptureSummary{
		OK:         parseRunErr == nil,
		Package:    parseConfig.packagePath,
		Count:      parseConfig.count,
		Bench:      parseConfig.bench,
		OutputPath: filepath.ToSlash(parseConfig.outputPath),
		Exec:       parseConfig.execPath,
		Command:    "go " + strings.Join(parseGoArgs, " "),
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		_ = parseEncoder.Encode(parseSummary)
	} else {
		printBenchmarkCaptureSummary(parseSummary)
	}
	if parseRunErr != nil {
		return parseRunErr
	}
	return nil
}

func resolveBenchmarkCompareConfig(parseConfig benchmarkCompareConfig) (benchmarkCompareConfig, error) {
	parseCwd, parseErr := os.Getwd()
	if parseErr != nil {
		return benchmarkCompareConfig{}, fmt.Errorf("resolve benchmark compare cwd: %w", parseErr)
	}

	parseBaselinePath, parseErr := normalizeExistingPath(parseCwd, strings.TrimSpace(parseConfig.baselinePath))
	if parseErr != nil {
		return benchmarkCompareConfig{}, fmt.Errorf("resolve baseline benchmark file: %w", parseErr)
	}
	parseCandidatePath, parseErr := normalizeExistingPath(parseCwd, strings.TrimSpace(parseConfig.candidatePath))
	if parseErr != nil {
		return benchmarkCompareConfig{}, fmt.Errorf("resolve candidate benchmark file: %w", parseErr)
	}

	return benchmarkCompareConfig{
		baselinePath:  parseBaselinePath,
		candidatePath: parseCandidatePath,
	}, nil
}

// resolveBenchmarkCaptureConfig validates and normalizes bench capture settings.
func (parseL launcher) resolveBenchmarkCaptureConfig(parseConfig benchmarkCaptureConfig) (benchmarkCaptureConfig, error) {
	parsePackagePath := strings.TrimSpace(parseConfig.packagePath)
	if parsePackagePath == "" {
		return benchmarkCaptureConfig{}, errors.New("benchmark capture package cannot be empty")
	}
	if parseConfig.count <= 0 {
		return benchmarkCaptureConfig{}, errors.New("benchmark capture count must be at least 1")
	}
	parseBenchPattern := strings.TrimSpace(parseConfig.bench)
	if parseBenchPattern == "" {
		parseBenchPattern = "."
	}
	parseOutputPath := strings.TrimSpace(parseConfig.outputPath)
	if parseOutputPath == "" {
		parseStamp := time.Now().Format("20060102-150405")
		parsePackageToken := benchmarkPackageToken(parsePackagePath)
		parseTargetDir := filepath.Join(strings.TrimSpace(parseL.repoRoot), "tools")
		if strings.TrimSpace(parseL.repoRoot) == "" {
			parseCwd, parseErr := os.Getwd()
			if parseErr != nil {
				return benchmarkCaptureConfig{}, fmt.Errorf("resolve benchmark capture cwd: %w", parseErr)
			}
			parseTargetDir = parseCwd
		}
		parseOutputPath = filepath.Join(parseTargetDir, fmt.Sprintf("bench-%s-%s.txt", parsePackageToken, parseStamp))
	} else if !filepath.IsAbs(parseOutputPath) {
		parseCwd2, parseErr2 := os.Getwd()
		if parseErr2 != nil {
			return benchmarkCaptureConfig{}, fmt.Errorf("resolve benchmark capture cwd: %w", parseErr2)
		}
		parseOutputPath = filepath.Join(parseCwd2, parseOutputPath)
	}
	if parseErr3 := os.MkdirAll(filepath.Dir(parseOutputPath), 0755); parseErr3 != nil {
		return benchmarkCaptureConfig{}, fmt.Errorf("create benchmark capture output directory: %w", parseErr3)
	}
	return benchmarkCaptureConfig{
		packagePath: parsePackagePath,
		count:       parseConfig.count,
		bench:       parseBenchPattern,
		outputPath:  filepath.Clean(parseOutputPath),
		execPath:    strings.TrimSpace(parseConfig.execPath),
		json:        parseConfig.json,
	}, nil
}

// benchmarkPackageToken converts a package path into a filename-safe token.
func benchmarkPackageToken(parsePackagePath string) string {
	parseTrimmed := strings.TrimSpace(parsePackagePath)
	if parseTrimmed == "" {
		return "package"
	}
	var parseBuilder strings.Builder
	for _, parseR := range parseTrimmed {
		isAllowed := (parseR >= 'a' && parseR <= 'z') || (parseR >= 'A' && parseR <= 'Z') || (parseR >= '0' && parseR <= '9') || parseR == '.' || parseR == '_' || parseR == '-'
		if isAllowed {
			parseBuilder.WriteRune(parseR)
		} else {
			parseBuilder.WriteRune('_')
		}
	}
	parseResult := parseBuilder.String()
	if strings.Trim(parseResult, "_") == "" {
		return "package"
	}
	return parseResult
}

func resolveBenchmarkConfig(parseConfig benchmarkConfig) (benchmarkConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCwd, parseErr := os.Getwd()
		if parseErr != nil {
			return benchmarkConfig{}, fmt.Errorf("resolve benchmark root from cwd: %w", parseErr)
		}
		parseRootPath = parseCwd
	}
	parseAbsRoot, parseErr2 := filepath.Abs(parseRootPath)
	if parseErr2 != nil {
		return benchmarkConfig{}, fmt.Errorf("resolve benchmark root: %w", parseErr2)
	}
	parseInfo, parseErr2 := os.Stat(parseAbsRoot)
	if parseErr2 != nil {
		return benchmarkConfig{}, fmt.Errorf("stat benchmark root: %w", parseErr2)
	}
	if !parseInfo.IsDir() {
		return benchmarkConfig{}, fmt.Errorf("benchmark root is not a directory: %s", parseAbsRoot)
	}
	parseLanes, parseErr2 := normalizeBenchmarkLanes(parseConfig.lanes)
	if parseErr2 != nil {
		return benchmarkConfig{}, parseErr2
	}
	parseCount := parseConfig.count
	if parseCount <= 0 {
		return benchmarkConfig{}, errors.New("benchmark count must be at least 1")
	}
	parseParallel := parseConfig.parallel
	if parseParallel <= 0 {
		return benchmarkConfig{}, errors.New("benchmark parallelism must be at least 1")
	}
	parseBenchPattern := strings.TrimSpace(parseConfig.bench)
	if parseBenchPattern == "" {
		parseBenchPattern = "."
	}
	parseResolvedOutPath := strings.TrimSpace(parseConfig.outPath)
	if parseResolvedOutPath == "" {
		parseResolvedOutPath = filepath.Join(parseAbsRoot, "docs", "benchmarks", "latest.json")
	} else if !filepath.IsAbs(parseResolvedOutPath) {
		parseResolvedOutPath = filepath.Join(parseAbsRoot, parseResolvedOutPath)
	}
	parseResolvedReferencePath := strings.TrimSpace(parseConfig.referencePath)
	if parseResolvedReferencePath == "" {
		parseResolvedReferencePath = filepath.Join(parseAbsRoot, "docs", "benchmarks", "reference.json")
	} else if !filepath.IsAbs(parseResolvedReferencePath) {
		parseResolvedReferencePath = filepath.Join(parseAbsRoot, parseResolvedReferencePath)
	}
	return benchmarkConfig{
		rootPath:      parseAbsRoot,
		lanes:         parseLanes,
		bench:         parseBenchPattern,
		benchtime:     strings.TrimSpace(parseConfig.benchtime),
		count:         parseCount,
		parallel:      parseParallel,
		json:          parseConfig.json,
		outPath:       filepath.Clean(parseResolvedOutPath),
		referencePath: filepath.Clean(parseResolvedReferencePath),
	}, nil
}

func printBenchmarkCompareSummary(parseSummary benchmarkCompareSummary) {
	fmt.Println("GWC bench compare")
	if !parseSummary.BenchstatAvailable {
		fmt.Println(parseSummary.Message)
		fmt.Printf("Baseline:  %s\n", parseSummary.BaselinePath)
		fmt.Printf("Candidate: %s\n", parseSummary.CandidatePath)
		return
	}
	fmt.Printf("Baseline:  %s\n", parseSummary.BaselinePath)
	fmt.Printf("Candidate: %s\n", parseSummary.CandidatePath)
	if strings.TrimSpace(parseSummary.Output) != "" {
		fmt.Println(parseSummary.Output)
	}
}

// printBenchmarkCaptureSummary prints a short human-readable summary for `gwc bench capture`.
func printBenchmarkCaptureSummary(parseSummary benchmarkCaptureSummary) {
	fmt.Println("GWC bench capture")
	fmt.Printf("Package: %s\n", parseSummary.Package)
	fmt.Printf("Count:   %d\n", parseSummary.Count)
	fmt.Printf("Bench:   %s\n", parseSummary.Bench)
	if strings.TrimSpace(parseSummary.Exec) != "" {
		fmt.Printf("Exec:    %s\n", parseSummary.Exec)
	}
	fmt.Printf("Output:  %s\n", parseSummary.OutputPath)
}

func normalizeBenchmarkLanes(parseRequested []string) ([]string, error) {
	if len(parseRequested) == 0 {
		return []string{"native", "wasm"}, nil
	}
	parseSeen := map[string]struct{}{}
	parseNormalized := []string{}
	parseAppendLane := func(parseValue string) {
		if _, parseOk := parseSeen[parseValue]; parseOk {
			return
		}
		parseSeen[parseValue] = struct{}{}
		parseNormalized = append(parseNormalized, parseValue)
	}
	for _, parseLane := range parseRequested {
		switch strings.ToLower(strings.TrimSpace(parseLane)) {
		case "all":
			parseAppendLane("native")
			parseAppendLane("wasm")
		case "native", "host", "go-native":
			parseAppendLane("native")
		case "wasm", "js-wasm", "go-wasm":
			parseAppendLane("wasm")
		default:
			return nil, fmt.Errorf("unknown benchmark lane %q", parseLane)
		}
	}
	return parseNormalized, nil
}

func (parseL launcher) executeBenchmark(parseConfig benchmarkConfig) (benchmarkReport, error) {
	parseGoVersion, parseErr := launcherRunCommand("go", []string{"env", "GOVERSION"}, parseConfig.rootPath, buildNativeGoEnv())
	if parseErr != nil {
		return benchmarkReport{}, fmt.Errorf("resolve Go version for benchmark report: %w", parseErr)
	}
	parseNativePackages, parseWasmPackages, parseErr := collectBenchmarkPackages(parseConfig.rootPath)
	if parseErr != nil {
		return benchmarkReport{}, parseErr
	}

	parseReport := benchmarkReport{
		OK:                 true,
		Root:               ".",
		GeneratedAt:        time.Now().UTC().Format(time.RFC3339),
		GoVersion:          strings.TrimSpace(parseGoVersion),
		GOOS:               runtime.GOOS,
		GOARCH:             runtime.GOARCH,
		SelectedLanes:      append([]string(nil), parseConfig.lanes...),
		Bench:              parseConfig.bench,
		Count:              parseConfig.count,
		PackageParallelism: parseConfig.parallel,
		Benchtime:          parseConfig.benchtime,
		Packages:           []benchmarkPackageReport{},
	}

	parseJobs := []benchmarkPackageJob{}
	for _, parseLane := range parseConfig.lanes {
		var parsePackages []string
		switch parseLane {
		case "native":
			parsePackages = parseNativePackages
		case "wasm":
			parsePackages = parseWasmPackages
		}
		if parseLane == "wasm" && len(parsePackages) == 0 {
			continue
		}
		for _, parsePackagePath := range parsePackages {
			parseJobs = append(parseJobs, benchmarkPackageJob{
				index:       len(parseJobs),
				lane:        parseLane,
				packagePath: parsePackagePath,
			})
		}
	}
	if len(parseJobs) == 0 {
		return benchmarkReport{}, fmt.Errorf("no benchmark packages were discovered under %s", parseConfig.rootPath)
	}
	parsePackageReports := parseL.runBenchmarkJobs(parseConfig, parseJobs)
	parseReport.Packages = parsePackageReports
	parseReport.PackageCount = len(parsePackageReports)
	for _, parsePackageReport := range parsePackageReports {
		parseReport.BenchmarkCount += parsePackageReport.BenchmarkCount
		if !parsePackageReport.OK {
			parseReport.OK = false
			parseReport.FailedPackages++
		}
	}
	if !parseReport.OK {
		return parseReport, fmt.Errorf("one or more benchmark packages failed; see %s", parseConfig.outPath)
	}
	return parseReport, nil
}

func (parseL launcher) runBenchmarkJobs(parseConfig benchmarkConfig, parseJobs []benchmarkPackageJob) []benchmarkPackageReport {
	if len(parseJobs) == 0 {
		return nil
	}
	if parseConfig.parallel <= 1 || len(parseJobs) == 1 {
		parseReports := make([]benchmarkPackageReport, len(parseJobs))
		for _, parseJob := range parseJobs {
			parseReports[parseJob.index] = parseL.runBenchmarkPackage(parseConfig, parseJob.lane, parseJob.packagePath)
		}
		return parseReports
	}
	parseWorkerCount := parseConfig.parallel
	if parseWorkerCount > len(parseJobs) {
		parseWorkerCount = len(parseJobs)
	}
	parseJobCh := make(chan benchmarkPackageJob)
	parseResultCh := make(chan benchmarkPackageResult, len(parseJobs))
	var parseWg sync.WaitGroup
	for parseI := 0; parseI < parseWorkerCount; parseI++ {
		parseWg.Add(1)
		go func() {
			defer parseWg.Done()
			for parseJob2 := range parseJobCh {
				parseResultCh <- benchmarkPackageResult{
					index:  parseJob2.index,
					report: parseL.runBenchmarkPackage(parseConfig, parseJob2.lane, parseJob2.packagePath),
				}
			}
		}()
	}
	for _, parseJob3 := range parseJobs {
		parseJobCh <- parseJob3
	}
	close(parseJobCh)
	parseWg.Wait()
	close(parseResultCh)
	parseReports2 := make([]benchmarkPackageReport, len(parseJobs))
	for parseResult := range parseResultCh {
		parseReports2[parseResult.index] = parseResult.report
	}
	return parseReports2
}

func (parseL launcher) runBenchmarkPackage(parseConfig benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
	parseWorkspace := parseConfig.rootPath
	if parsePackagePath != "." {
		parseWorkspace = filepath.Join(parseConfig.rootPath, filepath.FromSlash(strings.TrimPrefix(parsePackagePath, "./")))
	}
	parseArgs := []string{"test", "-run", "^$", "-bench", parseConfig.bench, "-benchmem", "-count", strconv.Itoa(parseConfig.count), "."}
	if parseConfig.benchtime != "" {
		parseArgs = append(parseArgs[:len(parseArgs)-1], "-benchtime", parseConfig.benchtime, ".")
	}
	parseEnv := buildNativeGoEnv()
	if parseLane == "wasm" {
		parseWasmExec, parseErr := benchmarkResolveWasmExec(parseL.repoRoot)
		if parseErr != nil {
			return benchmarkPackageReport{
				Lane:      parseLane,
				Package:   parsePackagePath,
				Workspace: benchmarkDisplayPath(parseConfig.rootPath, parseWorkspace),
				Command:   benchmarkCommandString(parseArgs, false),
				OK:        false,
				Error:     parseErr.Error(),
			}
		}
		parseArgs = []string{"test", "-exec", parseWasmExec, "-run", "^$", "-bench", parseConfig.bench, "-benchmem", "-count", strconv.Itoa(parseConfig.count), "."}
		if parseConfig.benchtime != "" {
			parseArgs = append(parseArgs[:len(parseArgs)-1], "-benchtime", parseConfig.benchtime, ".")
		}
		parseEnv = buildWasmGoEnv()
	}
	parseOutput, parseErr2 := launcherRunCommand("go", parseArgs, parseWorkspace, parseEnv)
	parseCommandText := benchmarkCommandString(parseArgs, parseLane == "wasm")
	if parseErr2 != nil {
		return benchmarkPackageReport{
			Lane:      parseLane,
			Package:   parsePackagePath,
			Workspace: benchmarkDisplayPath(parseConfig.rootPath, parseWorkspace),
			Command:   parseCommandText,
			OK:        false,
			Error:     parseErr2.Error(),
			Output:    parseOutput,
		}
	}
	parseBenchmarks := parseBenchmarkOutput(parseOutput)
	for parseIndex := range parseBenchmarks {
		parseBenchmarks[parseIndex].Bucket = benchmarkBucketID(parsePackagePath, parseBenchmarks[parseIndex].Name)
	}
	return benchmarkPackageReport{
		Lane:           parseLane,
		Package:        parsePackagePath,
		Workspace:      benchmarkDisplayPath(parseConfig.rootPath, parseWorkspace),
		Command:        parseCommandText,
		OK:             true,
		BenchmarkCount: len(parseBenchmarks),
		Benchmarks:     parseBenchmarks,
	}
}

func benchmarkDisplayPath(parseRootPath string, parseAbsolutePath string) string {
	parseRoot := strings.TrimSpace(parseRootPath)
	parsePath := strings.TrimSpace(parseAbsolutePath)
	if parseRoot == "" || parsePath == "" {
		return filepath.ToSlash(parsePath)
	}
	parseRelPath, parseErr := filepath.Rel(parseRoot, parsePath)
	if parseErr != nil {
		return filepath.ToSlash(parsePath)
	}
	parseRelPath = filepath.ToSlash(parseRelPath)
	if parseRelPath == "." {
		return "."
	}
	if strings.HasPrefix(parseRelPath, "../") {
		return parseRelPath
	}
	return "./" + strings.TrimPrefix(parseRelPath, "./")
}

func benchmarkCommandString(parseArgs []string, isSanitizeWasmExec bool) string {
	parseParts := append([]string(nil), parseArgs...)
	if isSanitizeWasmExec {
		for parseIndex := 0; parseIndex < len(parseParts)-1; parseIndex++ {
			if parseParts[parseIndex] == "-exec" {
				parseParts[parseIndex+1] = "<goWasmExec>"
				break
			}
		}
	}
	return "go " + strings.Join(parseParts, " ")
}

func benchmarkBucketID(parsePackagePath string, parseBenchmarkName string) string {
	parseLowerName := strings.ToLower(strings.TrimSpace(parseBenchmarkName))
	parseLowerPackage := strings.ToLower(strings.TrimSpace(parsePackagePath))
	switch {
	case strings.Contains(parseLowerName, "marshal"),
		strings.Contains(parseLowerName, "unmarshal"),
		strings.Contains(parseLowerName, "json"),
		strings.Contains(parseLowerName, "binary"),
		strings.Contains(parseLowerName, "decode"),
		strings.Contains(parseLowerName, "snapshot"),
		strings.Contains(parseLowerName, "bootstrap"),
		strings.Contains(parseLowerName, "clone"):
		return "memory"
	case strings.Contains(parseLowerName, "schedule"),
		strings.Contains(parseLowerName, "scheduler"),
		strings.Contains(parseLowerName, "queue"),
		strings.Contains(parseLowerName, "transition"),
		strings.Contains(parseLowerName, "subscribe"),
		strings.Contains(parseLowerName, "unsubscribe"),
		strings.Contains(parseLowerName, "listener"),
		strings.Contains(parseLowerName, "enabledisable"),
		strings.Contains(parseLowerName, "flushall"),
		strings.Contains(parseLowerName, "exchange"),
		strings.Contains(parseLowerName, "batch"):
		return "sync_concurrency"
	case strings.Contains(parseLowerName, "hydrate"),
		strings.Contains(parseLowerName, "navigation"),
		strings.Contains(parseLowerName, "viewport"),
		strings.Contains(parseLowerName, "multipartformdata"),
		strings.Contains(parseLowerName, "supportdiagnosticbundle"),
		strings.Contains(parseLowerName, "serviceworker"),
		strings.Contains(parseLowerName, "rendertostring"),
		strings.HasPrefix(parseLowerName, "benchmarkrenderto"),
		strings.Contains(parseLowerPackage, "/internal/platform/jsdom"):
		return "end_to_end"
	case strings.Contains(parseLowerPackage, "/internal/runtime"),
		strings.Contains(parseLowerName, "alloc"),
		strings.Contains(parseLowerName, "allocation"),
		strings.Contains(parseLowerName, "runtime"),
		strings.Contains(parseLowerName, "fiber"),
		strings.Contains(parseLowerName, "hook"),
		strings.Contains(parseLowerName, "atom"),
		strings.Contains(parseLowerName, "use"),
		strings.Contains(parseLowerName, "createelement"),
		strings.Contains(parseLowerName, "commit"),
		strings.Contains(parseLowerName, "propsequal"),
		strings.Contains(parseLowerName, "portal"),
		strings.Contains(parseLowerName, "shim"),
		strings.Contains(parseLowerName, "layout"),
		strings.Contains(parseLowerName, "refetch"):
		return "alloc_runtime"
	default:
		return "compute"
	}
}

func benchmarkBucketLabel(parseBucketID string) string {
	switch parseBucketID {
	case "compute":
		return "Compute"
	case "memory":
		return "Memory"
	case "alloc_runtime":
		return "Alloc/Runtime"
	case "sync_concurrency":
		return "Sync/Concurrency"
	case "end_to_end":
		return "End-to-End"
	default:
		return parseBucketID
	}
}

func benchmarkBucketOrder(parseBucketID string) int {
	switch parseBucketID {
	case "compute":
		return 0
	case "memory":
		return 1
	case "alloc_runtime":
		return 2
	case "sync_concurrency":
		return 3
	case "end_to_end":
		return 4
	default:
		return 100
	}
}

func buildBenchmarkScoreSummary(parseReference benchmarkReport, parseCurrent benchmarkReport, parseConfig benchmarkConfig) *benchmarkScoreSummary {
	parseReferenceIndex := benchmarkMetricIndex(parseReference)
	parseBucketRatios := map[string][]float64{}
	parseMatchedBenchmarks := 0
	for _, parsePackageReport := range parseCurrent.Packages {
		if !parsePackageReport.OK {
			continue
		}
		for _, parseBenchmark := range parsePackageReport.Benchmarks {
			parseMeasuredNS, parseOk := parseBenchmark.AverageMetrics["ns/op"]
			if !parseOk || parseMeasuredNS <= 0 {
				continue
			}
			parseReferenceNS, parseOk := parseReferenceIndex[benchmarkMetricKey{
				Lane:      parsePackageReport.Lane,
				Package:   parsePackageReport.Package,
				Benchmark: parseBenchmark.Name,
				Metric:    "ns/op",
			}]
			if !parseOk || parseReferenceNS <= 0 {
				continue
			}
			parseBucketID := parseBenchmark.Bucket
			if strings.TrimSpace(parseBucketID) == "" {
				parseBucketID = benchmarkBucketID(parsePackageReport.Package, parseBenchmark.Name)
			}
			parseBucketRatios[parseBucketID] = append(parseBucketRatios[parseBucketID], parseReferenceNS/parseMeasuredNS)
			parseMatchedBenchmarks++
		}
	}
	if parseMatchedBenchmarks == 0 {
		return nil
	}
	parseBuckets := make([]benchmarkBucketScore, 0, len(parseBucketRatios))
	parseBucketFactors := []float64{}
	for parseBucketID2, parseRatios := range parseBucketRatios {
		if len(parseRatios) == 0 {
			continue
		}
		parseFactor := geometricMean(parseRatios)
		if parseFactor <= 0 {
			continue
		}
		parseBucketFactors = append(parseBucketFactors, parseFactor)
		parseBuckets = append(parseBuckets, benchmarkBucketScore{
			ID:                parseBucketID2,
			Label:             benchmarkBucketLabel(parseBucketID2),
			MatchedBenchmarks: len(parseRatios),
			Score:             roundBenchmarkScore(100 * parseFactor),
		})
	}
	sort.Slice(parseBuckets, func(parseI, parseJ int) bool {
		parseLeft := benchmarkBucketOrder(parseBuckets[parseI].ID)
		parseRight := benchmarkBucketOrder(parseBuckets[parseJ].ID)
		if parseLeft == parseRight {
			return parseBuckets[parseI].ID < parseBuckets[parseJ].ID
		}
		return parseLeft < parseRight
	})
	parseOverall := 0.0
	if len(parseBucketFactors) > 0 {
		parseOverall = roundBenchmarkScore(100 * geometricMean(parseBucketFactors))
	}
	return &benchmarkScoreSummary{
		Method:               benchmarkScoreMethod,
		ReferencePath:        benchmarkDisplayPath(parseConfig.rootPath, parseConfig.referencePath),
		ReferenceGeneratedAt: parseReference.GeneratedAt,
		ReferenceMachine:     benchmarkMachineLabel(parseReference),
		MatchedBenchmarks:    parseMatchedBenchmarks,
		OverallScore:         parseOverall,
		Buckets:              parseBuckets,
	}
}

func geometricMean(parseValues []float64) float64 {
	if len(parseValues) == 0 {
		return 0
	}
	parseSum := 0.0
	parseCount := 0
	for _, parseValue := range parseValues {
		if parseValue <= 0 {
			continue
		}
		parseSum += math.Log(parseValue)
		parseCount++
	}
	if parseCount == 0 {
		return 0
	}
	return math.Exp(parseSum / float64(parseCount))
}

func roundBenchmarkScore(parseValue float64) float64 {
	return math.Round(parseValue*10) / 10
}

func benchmarkMachineLabel(parseReport benchmarkReport) string {
	parseGoos := strings.TrimSpace(parseReport.GOOS)
	parseGoarch := strings.TrimSpace(parseReport.GOARCH)
	parseGoVersion := strings.TrimSpace(parseReport.GoVersion)
	parseParts := []string{}
	if parseGoVersion != "" {
		parseParts = append(parseParts, parseGoVersion)
	}
	if parseGoos != "" || parseGoarch != "" {
		parseParts = append(parseParts, strings.TrimSpace(parseGoos+"/"+parseGoarch))
	}
	return strings.Join(parseParts, " ")
}

func collectBenchmarkPackages(parseRootPath string) ([]string, []string, error) {
	type benchmarkPresence struct {
		native bool
		wasm   bool
	}
	parsePresence := map[string]*benchmarkPresence{}
	parseErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseWalkErr error) error {
		if parseWalkErr != nil {
			return parseWalkErr
		}
		if parseEntry.IsDir() {
			if shouldSkipBenchmarkWalkDir(parseEntry.Name()) && parsePath != parseRootPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(parseEntry.Name(), "_test.go") {
			return nil
		}
		parseContent, parseErr2 := os.ReadFile(parsePath)
		if parseErr2 != nil {
			return parseErr2
		}
		hasBenchmark, parseWasmOnly := benchmarkTestFileKind(parseEntry.Name(), string(parseContent))
		if !hasBenchmark {
			return nil
		}
		parseDir := filepath.Dir(parsePath)
		parseRecord := parsePresence[parseDir]
		if parseRecord == nil {
			parseRecord = &benchmarkPresence{}
			parsePresence[parseDir] = parseRecord
		}
		if parseWasmOnly {
			parseRecord.wasm = true
		} else {
			parseRecord.native = true
		}
		return nil
	})
	if parseErr != nil {
		return nil, nil, fmt.Errorf("collect benchmark packages: %w", parseErr)
	}
	parseNativePackages := []string{}
	parseWasmPackages := []string{}
	for parseDir2, parseRecord2 := range parsePresence {
		parseRelDir, parseErr3 := filepath.Rel(parseRootPath, parseDir2)
		if parseErr3 != nil {
			return nil, nil, fmt.Errorf("resolve benchmark package path for %s: %w", parseDir2, parseErr3)
		}
		parsePackagePath := "."
		if parseRelDir != "." {
			parsePackagePath = "./" + filepath.ToSlash(parseRelDir)
		}
		if parseRecord2.native {
			parseNativePackages = append(parseNativePackages, parsePackagePath)
		}
		if parseRecord2.wasm {
			parseWasmPackages = append(parseWasmPackages, parsePackagePath)
		}
	}
	sort.Strings(parseNativePackages)
	sort.Strings(parseWasmPackages)
	return parseNativePackages, parseWasmPackages, nil
}

func shouldSkipBenchmarkWalkDir(parseName string) bool {
	if shouldSkipTestWalkDir(parseName) {
		return true
	}
	switch parseName {
	case ".venv", ".vscode", "bin", "docs", "examples", "test", "third_party", "tools":
		return true
	default:
		return false
	}
}

func benchmarkTestFileKind(parseName string, parseContent string) (bool, bool) {
	if !strings.Contains(parseContent, "func Benchmark") {
		return false, false
	}
	isParseWasmOnly := strings.HasSuffix(parseName, "_wasm_test.go") || benchmarkFileHasWasmBuildTag(parseContent)
	return true, isParseWasmOnly
}

func benchmarkFileHasWasmBuildTag(parseContent string) bool {
	parseLines := strings.Split(parseContent, "\n")
	parseLimit := len(parseLines)
	if parseLimit > 8 {
		parseLimit = 8
	}
	for _, parseLine := range parseLines[:parseLimit] {
		parseTrimmed := strings.TrimSpace(parseLine)
		if strings.Contains(parseTrimmed, "go:build js && wasm") || strings.Contains(parseTrimmed, "+build js,wasm") {
			return true
		}
	}
	return false
}

func parseBenchmarkOutput(parseOutput string) []benchmarkResultReport {
	parseOrdered := []string{}
	parseResults := map[string]*benchmarkResultReport{}
	for _, parseRawLine := range strings.Split(parseOutput, "\n") {
		parseLine := strings.TrimSpace(parseRawLine)
		if !strings.HasPrefix(parseLine, "Benchmark") {
			continue
		}
		parseFields := strings.Fields(parseLine)
		if len(parseFields) < 4 {
			continue
		}
		parseIterations, parseErr := strconv.ParseInt(parseFields[1], 10, 64)
		if parseErr != nil {
			continue
		}
		parseMetrics := map[string]float64{}
		for parseIndex := 2; parseIndex+1 < len(parseFields); parseIndex += 2 {
			parseValue, parseErr2 := strconv.ParseFloat(parseFields[parseIndex], 64)
			if parseErr2 != nil {
				break
			}
			parseMetrics[parseFields[parseIndex+1]] = parseValue
		}
		if len(parseMetrics) == 0 {
			continue
		}
		parseName := parseFields[0]
		parseRecord := parseResults[parseName]
		if parseRecord == nil {
			parseRecord = &benchmarkResultReport{Name: parseName}
			parseResults[parseName] = parseRecord
			parseOrdered = append(parseOrdered, parseName)
		}
		parseRecord.Samples = append(parseRecord.Samples, benchmarkSample{
			Iterations: parseIterations,
			Metrics:    parseMetrics,
			Raw:        parseLine,
		})
	}
	parseReport := make([]benchmarkResultReport, 0, len(parseOrdered))
	for _, parseName2 := range parseOrdered {
		parseRecord2 := parseResults[parseName2]
		parseRecord2.AverageMetrics = averageBenchmarkMetrics(parseRecord2.Samples)
		parseReport = append(parseReport, *parseRecord2)
	}
	return parseReport
}

func averageBenchmarkMetrics(parseSamples []benchmarkSample) map[string]float64 {
	if len(parseSamples) == 0 {
		return nil
	}
	parseTotals := map[string]float64{}
	parseCounts := map[string]int{}
	for _, parseSample := range parseSamples {
		for parseMetric, parseValue := range parseSample.Metrics {
			parseTotals[parseMetric] += parseValue
			parseCounts[parseMetric]++
		}
	}
	parseAverages := map[string]float64{}
	for parseMetric2, parseTotal := range parseTotals {
		parseAverages[parseMetric2] = parseTotal / float64(parseCounts[parseMetric2])
	}
	return parseAverages
}

func loadBenchmarkReport(parsePath string) (benchmarkReport, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return benchmarkReport{}, parseErr
	}
	var parseReport benchmarkReport
	if parseErr2 := json.Unmarshal(parseContent, &parseReport); parseErr2 != nil {
		return benchmarkReport{}, parseErr2
	}
	return parseReport, nil
}

func compareBenchmarkReports(parseBaseline benchmarkReport, parseCurrent benchmarkReport) *benchmarkComparisonSummary {
	if parseBaseline.GeneratedAt == "" {
		return nil
	}
	parseBaselineMetrics := benchmarkMetricIndex(parseBaseline)
	parseCurrentMetrics := benchmarkMetricIndex(parseCurrent)
	parseEntries := []benchmarkMetricComparison{}
	parseImproved := 0
	parseRegressed := 0
	parseUnchanged := 0
	for parseKey, parseBaselineValue := range parseBaselineMetrics {
		parseCurrentValue, parseOk := parseCurrentMetrics[parseKey]
		if !parseOk {
			continue
		}
		parseDelta := parseCurrentValue - parseBaselineValue
		parseDeltaPct := 0.0
		if parseBaselineValue != 0 {
			parseDeltaPct = (parseDelta / parseBaselineValue) * 100
		}
		parseDirection := "unchanged"
		if math.Abs(parseDeltaPct) >= benchmarkComparisonTolerancePct {
			if parseDelta < 0 {
				parseDirection = "improved"
				parseImproved++
			} else if parseDelta > 0 {
				parseDirection = "regressed"
				parseRegressed++
			}
		} else {
			parseUnchanged++
		}
		parseEntries = append(parseEntries, benchmarkMetricComparison{
			Lane:      parseKey.Lane,
			Package:   parseKey.Package,
			Benchmark: parseKey.Benchmark,
			Metric:    parseKey.Metric,
			Baseline:  parseBaselineValue,
			Current:   parseCurrentValue,
			Delta:     parseDelta,
			DeltaPct:  parseDeltaPct,
			Direction: parseDirection,
		})
	}
	if len(parseEntries) == 0 {
		return nil
	}
	sort.Slice(parseEntries, func(parseI, parseJ int) bool {
		parseLeft := math.Abs(parseEntries[parseI].DeltaPct)
		parseRight := math.Abs(parseEntries[parseJ].DeltaPct)
		if parseLeft == parseRight {
			if parseEntries[parseI].Lane == parseEntries[parseJ].Lane {
				if parseEntries[parseI].Package == parseEntries[parseJ].Package {
					if parseEntries[parseI].Benchmark == parseEntries[parseJ].Benchmark {
						return parseEntries[parseI].Metric < parseEntries[parseJ].Metric
					}
					return parseEntries[parseI].Benchmark < parseEntries[parseJ].Benchmark
				}
				return parseEntries[parseI].Package < parseEntries[parseJ].Package
			}
			return parseEntries[parseI].Lane < parseEntries[parseJ].Lane
		}
		return parseLeft > parseRight
	})
	return &benchmarkComparisonSummary{
		BaselineGeneratedAt: parseBaseline.GeneratedAt,
		TolerancePct:        benchmarkComparisonTolerancePct,
		MatchedMetrics:      len(parseEntries),
		Improved:            parseImproved,
		Regressed:           parseRegressed,
		Unchanged:           parseUnchanged,
		Entries:             parseEntries,
	}
}

func benchmarkMetricIndex(parseReport benchmarkReport) map[benchmarkMetricKey]float64 {
	parseIndex := map[benchmarkMetricKey]float64{}
	for _, parsePackageReport := range parseReport.Packages {
		if !parsePackageReport.OK {
			continue
		}
		for _, parseBenchmark := range parsePackageReport.Benchmarks {
			for parseMetric, parseValue := range parseBenchmark.AverageMetrics {
				if !benchmarkMetricComparable(parseMetric) {
					continue
				}
				parseIndex[benchmarkMetricKey{
					Lane:      parsePackageReport.Lane,
					Package:   parsePackageReport.Package,
					Benchmark: parseBenchmark.Name,
					Metric:    parseMetric,
				}] = parseValue
			}
		}
	}
	return parseIndex
}

func benchmarkMetricComparable(parseMetric string) bool {
	switch strings.TrimSpace(parseMetric) {
	case "ns/op", "B/op", "allocs/op":
		return true
	default:
		return false
	}
}

func writeBenchmarkReport(parsePath string, parseReport benchmarkReport) error {
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0755); parseErr != nil {
		return fmt.Errorf("create benchmark report directory: %w", parseErr)
	}
	parsePayload, parseErr2 := json.MarshalIndent(parseReport, "", "  ")
	if parseErr2 != nil {
		return fmt.Errorf("marshal benchmark report: %w", parseErr2)
	}
	parsePayload = append(parsePayload, '\n')
	if parseErr3 := os.WriteFile(parsePath, parsePayload, 0644); parseErr3 != nil {
		return fmt.Errorf("write benchmark report: %w", parseErr3)
	}
	return nil
}

func printBenchmarkReport(parseReport benchmarkReport) {
	fmt.Println("GWC bench")
	fmt.Printf("root:          %s\n", parseReport.Root)
	fmt.Printf("report:        %s\n", parseReport.ReportPath)
	fmt.Printf("go version:    %s\n", parseReport.GoVersion)
	fmt.Printf("lanes:         %s\n", strings.Join(parseReport.SelectedLanes, ", "))
	fmt.Printf("parallel:      %d\n", parseReport.PackageParallelism)
	fmt.Printf("packages:      %d\n", parseReport.PackageCount)
	fmt.Printf("benchmarks:    %d\n", parseReport.BenchmarkCount)
	if parseReport.FailedPackages > 0 {
		fmt.Printf("failures:      %d\n", parseReport.FailedPackages)
	}
	if parseReport.Scores != nil {
		fmt.Printf("reference:     %s", parseReport.Scores.ReferencePath)
		if strings.TrimSpace(parseReport.Scores.ReferenceGeneratedAt) != "" {
			fmt.Printf(" (%s", parseReport.Scores.ReferenceGeneratedAt)
			if parseMachine := strings.TrimSpace(parseReport.Scores.ReferenceMachine); parseMachine != "" {
				fmt.Printf(", %s", parseMachine)
			}
			fmt.Print(")")
		}
		fmt.Println()
		for _, parseBucket := range parseReport.Scores.Buckets {
			fmt.Printf("%-15s %5.1f %s (%d benchmarks)\n", parseBucket.Label+" Score:", parseBucket.Score, benchmarkScoreGraph(parseBucket.Score, 20), parseBucket.MatchedBenchmarks)
		}
		fmt.Printf("Overall Score: %5.1f %s\n", parseReport.Scores.OverallScore, benchmarkScoreGraph(parseReport.Scores.OverallScore, 20))
	}
	if parseReport.Comparison != nil {
		fmt.Printf("baseline:      %s\n", parseReport.Comparison.BaselineGeneratedAt)
		fmt.Printf("comparison:    %d improved, %d regressed, %d unchanged (tolerance %.1f%%)\n", parseReport.Comparison.Improved, parseReport.Comparison.Regressed, parseReport.Comparison.Unchanged, parseReport.Comparison.TolerancePct)
	}
	for _, parsePackageReport := range parseReport.Packages {
		parseStatus := "ok"
		if !parsePackageReport.OK {
			parseStatus = "fail"
		}
		fmt.Printf("[%s] %s %s (%d benchmarks)\n", parseStatus, parsePackageReport.Lane, parsePackageReport.Package, parsePackageReport.BenchmarkCount)
		if parsePackageReport.Error != "" {
			fmt.Printf("  error: %s\n", parsePackageReport.Error)
		}
	}
}

func benchmarkScoreGraph(parseScore float64, parseWidth int) string {
	if parseWidth <= 0 {
		return ""
	}
	parseClamped := parseScore
	if parseClamped < 0 {
		parseClamped = 0
	}
	if parseClamped > 200 {
		parseClamped = 200
	}
	parseFilled := int(math.Round((parseClamped / 200) * float64(parseWidth)))
	if parseFilled < 0 {
		parseFilled = 0
	}
	if parseFilled > parseWidth {
		parseFilled = parseWidth
	}
	return "[" + strings.Repeat("#", parseFilled) + strings.Repeat("-", parseWidth-parseFilled) + "]"
}
