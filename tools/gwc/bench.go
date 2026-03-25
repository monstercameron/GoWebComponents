package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"time"
)

var runBenchmarkCommand = func(l launcher, args []string) error {
	return l.runBenchmark(args)
}

var benchmarkResolveWasmExec = resolveWasmTestExec

type benchmarkConfig struct {
	rootPath  string
	lanes     []string
	bench     string
	benchtime string
	count     int
	json      bool
	outPath   string
}

type benchmarkReport struct {
	OK             bool                        `json:"ok"`
	Root           string                      `json:"root"`
	GeneratedAt    string                      `json:"generatedAt"`
	GoVersion      string                      `json:"goVersion"`
	GOOS           string                      `json:"goos"`
	GOARCH         string                      `json:"goarch"`
	SelectedLanes  []string                    `json:"selectedLanes"`
	Bench          string                      `json:"bench"`
	Count          int                         `json:"count"`
	Benchtime      string                      `json:"benchtime,omitempty"`
	ReportPath     string                      `json:"reportPath"`
	PackageCount   int                         `json:"packageCount"`
	BenchmarkCount int                         `json:"benchmarkCount"`
	FailedPackages int                         `json:"failedPackages"`
	Packages       []benchmarkPackageReport    `json:"packages"`
	Comparison     *benchmarkComparisonSummary `json:"comparison,omitempty"`
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
	Samples        []benchmarkSample  `json:"samples"`
	AverageMetrics map[string]float64 `json:"averageMetrics,omitempty"`
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

const benchmarkComparisonTolerancePct = 2.0

func (l launcher) runBenchmark(args []string) error {
	fs := flag.NewFlagSet("bench", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	root := fs.String("root", "", "Root directory to inspect for benchmark packages; defaults to the current working directory")
	benchPattern := fs.String("bench", ".", "Benchmark pattern passed to go test -bench")
	benchtime := fs.String("benchtime", "", "Optional benchtime forwarded to go test")
	count := fs.Int("count", 1, "Number of benchmark runs per package")
	outPath := fs.String("out", "", "JSON report path; defaults to docs/benchmarks/latest.json beneath the root")
	jsonOutput := fs.Bool("json", false, "Emit the JSON report to stdout after writing it to disk")
	var lanes stringListFlag
	fs.Var(&lanes, "lane", "Benchmark lanes: native, wasm, or all; repeatable")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveBenchmarkConfig(benchmarkConfig{
		rootPath:  *root,
		lanes:     lanes.Values(),
		bench:     *benchPattern,
		benchtime: *benchtime,
		count:     *count,
		json:      *jsonOutput,
		outPath:   *outPath,
	})
	if err != nil {
		return err
	}

	previous, _ := loadBenchmarkReport(config.outPath)
	report, runErr := l.executeBenchmark(config)
	report.ReportPath = benchmarkDisplayPath(config.rootPath, config.outPath)
	if previous.OK || previous.GeneratedAt != "" {
		report.Comparison = compareBenchmarkReports(previous, report)
	}
	if writeErr := writeBenchmarkReport(config.outPath, report); writeErr != nil {
		return writeErr
	}
	if config.json {
		encoder := json.NewEncoder(os.Stdout)
		encoder.SetIndent("", "  ")
		if err := encoder.Encode(report); err != nil {
			return err
		}
	} else {
		printBenchmarkReport(report)
	}
	if runErr != nil {
		return runErr
	}
	return nil
}

func resolveBenchmarkConfig(config benchmarkConfig) (benchmarkConfig, error) {
	rootPath := strings.TrimSpace(config.rootPath)
	if rootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return benchmarkConfig{}, fmt.Errorf("resolve benchmark root from cwd: %w", err)
		}
		rootPath = cwd
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return benchmarkConfig{}, fmt.Errorf("resolve benchmark root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return benchmarkConfig{}, fmt.Errorf("stat benchmark root: %w", err)
	}
	if !info.IsDir() {
		return benchmarkConfig{}, fmt.Errorf("benchmark root is not a directory: %s", absRoot)
	}
	lanes, err := normalizeBenchmarkLanes(config.lanes)
	if err != nil {
		return benchmarkConfig{}, err
	}
	count := config.count
	if count <= 0 {
		return benchmarkConfig{}, errors.New("benchmark count must be at least 1")
	}
	benchPattern := strings.TrimSpace(config.bench)
	if benchPattern == "" {
		benchPattern = "."
	}
	resolvedOutPath := strings.TrimSpace(config.outPath)
	if resolvedOutPath == "" {
		resolvedOutPath = filepath.Join(absRoot, "docs", "benchmarks", "latest.json")
	} else if !filepath.IsAbs(resolvedOutPath) {
		resolvedOutPath = filepath.Join(absRoot, resolvedOutPath)
	}
	return benchmarkConfig{
		rootPath:  absRoot,
		lanes:     lanes,
		bench:     benchPattern,
		benchtime: strings.TrimSpace(config.benchtime),
		count:     count,
		json:      config.json,
		outPath:   filepath.Clean(resolvedOutPath),
	}, nil
}

func normalizeBenchmarkLanes(requested []string) ([]string, error) {
	if len(requested) == 0 {
		return []string{"native", "wasm"}, nil
	}
	seen := map[string]struct{}{}
	normalized := []string{}
	appendLane := func(value string) {
		if _, ok := seen[value]; ok {
			return
		}
		seen[value] = struct{}{}
		normalized = append(normalized, value)
	}
	for _, lane := range requested {
		switch strings.ToLower(strings.TrimSpace(lane)) {
		case "all":
			appendLane("native")
			appendLane("wasm")
		case "native", "host", "go-native":
			appendLane("native")
		case "wasm", "js-wasm", "go-wasm":
			appendLane("wasm")
		default:
			return nil, fmt.Errorf("unknown benchmark lane %q", lane)
		}
	}
	return normalized, nil
}

func (l launcher) executeBenchmark(config benchmarkConfig) (benchmarkReport, error) {
	goVersion, err := launcherRunCommand("go", []string{"env", "GOVERSION"}, config.rootPath, buildNativeGoEnv())
	if err != nil {
		return benchmarkReport{}, fmt.Errorf("resolve Go version for benchmark report: %w", err)
	}
	nativePackages, wasmPackages, err := collectBenchmarkPackages(config.rootPath)
	if err != nil {
		return benchmarkReport{}, err
	}

	report := benchmarkReport{
		OK:            true,
		Root:          ".",
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		GoVersion:     strings.TrimSpace(goVersion),
		GOOS:          runtime.GOOS,
		GOARCH:        runtime.GOARCH,
		SelectedLanes: append([]string(nil), config.lanes...),
		Bench:         config.bench,
		Count:         config.count,
		Benchtime:     config.benchtime,
		Packages:      []benchmarkPackageReport{},
	}

	packageCountBefore := 0
	for _, lane := range config.lanes {
		var packages []string
		switch lane {
		case "native":
			packages = nativePackages
		case "wasm":
			packages = wasmPackages
		}
		packageCountBefore += len(packages)
		if lane == "wasm" && len(packages) == 0 {
			continue
		}
		for _, packagePath := range packages {
			packageReport := l.runBenchmarkPackage(config, lane, packagePath)
			report.Packages = append(report.Packages, packageReport)
			report.PackageCount++
			report.BenchmarkCount += packageReport.BenchmarkCount
			if !packageReport.OK {
				report.OK = false
				report.FailedPackages++
			}
		}
	}
	if packageCountBefore == 0 {
		return benchmarkReport{}, fmt.Errorf("no benchmark packages were discovered under %s", config.rootPath)
	}
	if !report.OK {
		return report, fmt.Errorf("one or more benchmark packages failed; see %s", config.outPath)
	}
	return report, nil
}

func (l launcher) runBenchmarkPackage(config benchmarkConfig, lane string, packagePath string) benchmarkPackageReport {
	workspace := config.rootPath
	if packagePath != "." {
		workspace = filepath.Join(config.rootPath, filepath.FromSlash(strings.TrimPrefix(packagePath, "./")))
	}
	args := []string{"test", "-run", "^$", "-bench", config.bench, "-benchmem", "-count", strconv.Itoa(config.count), "."}
	if config.benchtime != "" {
		args = append(args[:len(args)-1], "-benchtime", config.benchtime, ".")
	}
	env := buildNativeGoEnv()
	if lane == "wasm" {
		wasmExec, err := benchmarkResolveWasmExec(l.repoRoot)
		if err != nil {
			return benchmarkPackageReport{
				Lane:      lane,
				Package:   packagePath,
				Workspace: benchmarkDisplayPath(config.rootPath, workspace),
				Command:   benchmarkCommandString(args, false),
				OK:        false,
				Error:     err.Error(),
			}
		}
		args = []string{"test", "-exec", wasmExec, "-run", "^$", "-bench", config.bench, "-benchmem", "-count", strconv.Itoa(config.count), "."}
		if config.benchtime != "" {
			args = append(args[:len(args)-1], "-benchtime", config.benchtime, ".")
		}
		env = buildWasmGoEnv()
	}
	output, err := launcherRunCommand("go", args, workspace, env)
	commandText := benchmarkCommandString(args, lane == "wasm")
	if err != nil {
		return benchmarkPackageReport{
			Lane:      lane,
			Package:   packagePath,
			Workspace: benchmarkDisplayPath(config.rootPath, workspace),
			Command:   commandText,
			OK:        false,
			Error:     err.Error(),
			Output:    output,
		}
	}
	benchmarks := parseBenchmarkOutput(output)
	return benchmarkPackageReport{
		Lane:           lane,
		Package:        packagePath,
		Workspace:      benchmarkDisplayPath(config.rootPath, workspace),
		Command:        commandText,
		OK:             true,
		BenchmarkCount: len(benchmarks),
		Benchmarks:     benchmarks,
	}
}

func benchmarkDisplayPath(rootPath string, absolutePath string) string {
	root := strings.TrimSpace(rootPath)
	path := strings.TrimSpace(absolutePath)
	if root == "" || path == "" {
		return filepath.ToSlash(path)
	}
	relPath, err := filepath.Rel(root, path)
	if err != nil {
		return filepath.ToSlash(path)
	}
	relPath = filepath.ToSlash(relPath)
	if relPath == "." {
		return "."
	}
	if strings.HasPrefix(relPath, "../") {
		return relPath
	}
	return "./" + strings.TrimPrefix(relPath, "./")
}

func benchmarkCommandString(args []string, sanitizeWasmExec bool) string {
	parts := append([]string(nil), args...)
	if sanitizeWasmExec {
		for index := 0; index < len(parts)-1; index++ {
			if parts[index] == "-exec" {
				parts[index+1] = "<goWasmExec>"
				break
			}
		}
	}
	return "go " + strings.Join(parts, " ")
}

func collectBenchmarkPackages(rootPath string) ([]string, []string, error) {
	type benchmarkPresence struct {
		native bool
		wasm   bool
	}
	presence := map[string]*benchmarkPresence{}
	err := filepath.WalkDir(rootPath, func(path string, entry fs.DirEntry, walkErr error) error {
		if walkErr != nil {
			return walkErr
		}
		if entry.IsDir() {
			if shouldSkipBenchmarkWalkDir(entry.Name()) && path != rootPath {
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(entry.Name(), "_test.go") {
			return nil
		}
		content, err := os.ReadFile(path)
		if err != nil {
			return err
		}
		hasBenchmark, wasmOnly := benchmarkTestFileKind(entry.Name(), string(content))
		if !hasBenchmark {
			return nil
		}
		dir := filepath.Dir(path)
		record := presence[dir]
		if record == nil {
			record = &benchmarkPresence{}
			presence[dir] = record
		}
		if wasmOnly {
			record.wasm = true
		} else {
			record.native = true
		}
		return nil
	})
	if err != nil {
		return nil, nil, fmt.Errorf("collect benchmark packages: %w", err)
	}
	nativePackages := []string{}
	wasmPackages := []string{}
	for dir, record := range presence {
		relDir, err := filepath.Rel(rootPath, dir)
		if err != nil {
			return nil, nil, fmt.Errorf("resolve benchmark package path for %s: %w", dir, err)
		}
		packagePath := "."
		if relDir != "." {
			packagePath = "./" + filepath.ToSlash(relDir)
		}
		if record.native {
			nativePackages = append(nativePackages, packagePath)
		}
		if record.wasm {
			wasmPackages = append(wasmPackages, packagePath)
		}
	}
	sort.Strings(nativePackages)
	sort.Strings(wasmPackages)
	return nativePackages, wasmPackages, nil
}

func shouldSkipBenchmarkWalkDir(name string) bool {
	if shouldSkipTestWalkDir(name) {
		return true
	}
	switch name {
	case ".venv", ".vscode", "bin", "docs", "examples", "test", "third_party", "tools":
		return true
	default:
		return false
	}
}

func benchmarkTestFileKind(name string, content string) (bool, bool) {
	if !strings.Contains(content, "func Benchmark") {
		return false, false
	}
	wasmOnly := strings.HasSuffix(name, "_wasm_test.go") || benchmarkFileHasWasmBuildTag(content)
	return true, wasmOnly
}

func benchmarkFileHasWasmBuildTag(content string) bool {
	lines := strings.Split(content, "\n")
	limit := len(lines)
	if limit > 8 {
		limit = 8
	}
	for _, line := range lines[:limit] {
		trimmed := strings.TrimSpace(line)
		if strings.Contains(trimmed, "go:build js && wasm") || strings.Contains(trimmed, "+build js,wasm") {
			return true
		}
	}
	return false
}

func parseBenchmarkOutput(output string) []benchmarkResultReport {
	ordered := []string{}
	results := map[string]*benchmarkResultReport{}
	for _, rawLine := range strings.Split(output, "\n") {
		line := strings.TrimSpace(rawLine)
		if !strings.HasPrefix(line, "Benchmark") {
			continue
		}
		fields := strings.Fields(line)
		if len(fields) < 4 {
			continue
		}
		iterations, err := strconv.ParseInt(fields[1], 10, 64)
		if err != nil {
			continue
		}
		metrics := map[string]float64{}
		for index := 2; index+1 < len(fields); index += 2 {
			value, err := strconv.ParseFloat(fields[index], 64)
			if err != nil {
				break
			}
			metrics[fields[index+1]] = value
		}
		if len(metrics) == 0 {
			continue
		}
		name := fields[0]
		record := results[name]
		if record == nil {
			record = &benchmarkResultReport{Name: name}
			results[name] = record
			ordered = append(ordered, name)
		}
		record.Samples = append(record.Samples, benchmarkSample{
			Iterations: iterations,
			Metrics:    metrics,
			Raw:        line,
		})
	}
	report := make([]benchmarkResultReport, 0, len(ordered))
	for _, name := range ordered {
		record := results[name]
		record.AverageMetrics = averageBenchmarkMetrics(record.Samples)
		report = append(report, *record)
	}
	return report
}

func averageBenchmarkMetrics(samples []benchmarkSample) map[string]float64 {
	if len(samples) == 0 {
		return nil
	}
	totals := map[string]float64{}
	counts := map[string]int{}
	for _, sample := range samples {
		for metric, value := range sample.Metrics {
			totals[metric] += value
			counts[metric]++
		}
	}
	averages := map[string]float64{}
	for metric, total := range totals {
		averages[metric] = total / float64(counts[metric])
	}
	return averages
}

func loadBenchmarkReport(path string) (benchmarkReport, error) {
	content, err := os.ReadFile(path)
	if err != nil {
		return benchmarkReport{}, err
	}
	var report benchmarkReport
	if err := json.Unmarshal(content, &report); err != nil {
		return benchmarkReport{}, err
	}
	return report, nil
}

func compareBenchmarkReports(baseline benchmarkReport, current benchmarkReport) *benchmarkComparisonSummary {
	if baseline.GeneratedAt == "" {
		return nil
	}
	baselineMetrics := benchmarkMetricIndex(baseline)
	currentMetrics := benchmarkMetricIndex(current)
	entries := []benchmarkMetricComparison{}
	improved := 0
	regressed := 0
	unchanged := 0
	for key, baselineValue := range baselineMetrics {
		currentValue, ok := currentMetrics[key]
		if !ok {
			continue
		}
		delta := currentValue - baselineValue
		deltaPct := 0.0
		if baselineValue != 0 {
			deltaPct = (delta / baselineValue) * 100
		}
		direction := "unchanged"
		if math.Abs(deltaPct) >= benchmarkComparisonTolerancePct {
			if delta < 0 {
				direction = "improved"
				improved++
			} else if delta > 0 {
				direction = "regressed"
				regressed++
			}
		} else {
			unchanged++
		}
		entries = append(entries, benchmarkMetricComparison{
			Lane:      key.Lane,
			Package:   key.Package,
			Benchmark: key.Benchmark,
			Metric:    key.Metric,
			Baseline:  baselineValue,
			Current:   currentValue,
			Delta:     delta,
			DeltaPct:  deltaPct,
			Direction: direction,
		})
	}
	if len(entries) == 0 {
		return nil
	}
	sort.Slice(entries, func(i, j int) bool {
		left := math.Abs(entries[i].DeltaPct)
		right := math.Abs(entries[j].DeltaPct)
		if left == right {
			if entries[i].Lane == entries[j].Lane {
				if entries[i].Package == entries[j].Package {
					if entries[i].Benchmark == entries[j].Benchmark {
						return entries[i].Metric < entries[j].Metric
					}
					return entries[i].Benchmark < entries[j].Benchmark
				}
				return entries[i].Package < entries[j].Package
			}
			return entries[i].Lane < entries[j].Lane
		}
		return left > right
	})
	return &benchmarkComparisonSummary{
		BaselineGeneratedAt: baseline.GeneratedAt,
		TolerancePct:        benchmarkComparisonTolerancePct,
		MatchedMetrics:      len(entries),
		Improved:            improved,
		Regressed:           regressed,
		Unchanged:           unchanged,
		Entries:             entries,
	}
}

func benchmarkMetricIndex(report benchmarkReport) map[benchmarkMetricKey]float64 {
	index := map[benchmarkMetricKey]float64{}
	for _, packageReport := range report.Packages {
		if !packageReport.OK {
			continue
		}
		for _, benchmark := range packageReport.Benchmarks {
			for metric, value := range benchmark.AverageMetrics {
				if !benchmarkMetricComparable(metric) {
					continue
				}
				index[benchmarkMetricKey{
					Lane:      packageReport.Lane,
					Package:   packageReport.Package,
					Benchmark: benchmark.Name,
					Metric:    metric,
				}] = value
			}
		}
	}
	return index
}

func benchmarkMetricComparable(metric string) bool {
	switch strings.TrimSpace(metric) {
	case "ns/op", "B/op", "allocs/op":
		return true
	default:
		return false
	}
}

func writeBenchmarkReport(path string, report benchmarkReport) error {
	if err := os.MkdirAll(filepath.Dir(path), 0755); err != nil {
		return fmt.Errorf("create benchmark report directory: %w", err)
	}
	payload, err := json.MarshalIndent(report, "", "  ")
	if err != nil {
		return fmt.Errorf("marshal benchmark report: %w", err)
	}
	payload = append(payload, '\n')
	if err := os.WriteFile(path, payload, 0644); err != nil {
		return fmt.Errorf("write benchmark report: %w", err)
	}
	return nil
}

func printBenchmarkReport(report benchmarkReport) {
	fmt.Println("GWC bench")
	fmt.Printf("root:          %s\n", report.Root)
	fmt.Printf("report:        %s\n", report.ReportPath)
	fmt.Printf("go version:    %s\n", report.GoVersion)
	fmt.Printf("lanes:         %s\n", strings.Join(report.SelectedLanes, ", "))
	fmt.Printf("packages:      %d\n", report.PackageCount)
	fmt.Printf("benchmarks:    %d\n", report.BenchmarkCount)
	if report.FailedPackages > 0 {
		fmt.Printf("failures:      %d\n", report.FailedPackages)
	}
	if report.Comparison != nil {
		fmt.Printf("baseline:      %s\n", report.Comparison.BaselineGeneratedAt)
		fmt.Printf("comparison:    %d improved, %d regressed, %d unchanged (tolerance %.1f%%)\n", report.Comparison.Improved, report.Comparison.Regressed, report.Comparison.Unchanged, report.Comparison.TolerancePct)
	}
	for _, packageReport := range report.Packages {
		status := "ok"
		if !packageReport.OK {
			status = "fail"
		}
		fmt.Printf("[%s] %s %s (%d benchmarks)\n", status, packageReport.Lane, packageReport.Package, packageReport.BenchmarkCount)
		if packageReport.Error != "" {
			fmt.Printf("  error: %s\n", packageReport.Error)
		}
	}
}
