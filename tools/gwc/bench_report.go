package main

import (
	"encoding/json"
	"fmt"
	"io/fs"
	"math"
	"os"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
)

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
	parseLimit := min(len(parseLines), 8)
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
	for parseRawLine := range strings.SplitSeq(parseOutput, "\n") {
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
	parseFilled := min(max(int(math.Round((parseClamped/200)*float64(parseWidth))), 0), parseWidth)
	return "[" + strings.Repeat("#", parseFilled) + strings.Repeat("-", parseWidth-parseFilled) + "]"
}
