package main

import (
	"encoding/json"
	"math"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
)

func TestBuildBenchmarkMachineRecordsEffectiveAndHostArch(parseT *testing.T) {
	parseOriginal := benchmarkQueryCommand
	parseT.Cleanup(func() { benchmarkQueryCommand = parseOriginal })
	benchmarkQueryCommand = func(parseCwd string, parseName string, parseArgs ...string) string {
		switch strings.Join(append([]string{parseName}, parseArgs...), " ") {
		case "go env GOARCH":
			return "amd64"
		case "go env GOHOSTARCH":
			return "arm64"
		case "go env GOFLAGS":
			return "-mod=mod -tags=a,b"
		case "git rev-parse HEAD":
			return "abc123"
		case "git status --porcelain":
			return " M file.go"
		}
		return ""
	}
	parseMachine := buildBenchmarkMachine(parseT.TempDir(), "go1.26.3")
	if parseMachine.Arch != "amd64" || parseMachine.HostArch != "arm64" {
		parseT.Fatalf("unexpected arch fields: %#v", parseMachine)
	}
	if parseMachine.BuildTags != "a,b" || parseMachine.GitSHA != "abc123" || !parseMachine.GitDirty || parseMachine.GoVersion != "go1.26.3" {
		parseT.Fatalf("unexpected machine fields: %#v", parseMachine)
	}
	if parseMachine.LogicalCPUs <= 0 || parseMachine.GOMAXPROCS <= 0 {
		parseT.Fatalf("expected cpu counts: %#v", parseMachine)
	}
	if !strings.Contains(formatBenchmarkMachine(parseMachine), "emulated") {
		parseT.Fatalf("expected emulated marker: %q", formatBenchmarkMachine(parseMachine))
	}
}

func TestBenchmarkReportMachineOmittedForOldJSON(parseT *testing.T) {
	parseOld := []byte(`{"ok":true,"generatedAt":"x","packages":[{"lane":"native","package":"./a","ok":true,"benchmarks":[{"name":"B","samples":[],"averageMetrics":{"ns/op":1}}]}]}`)
	var parseReport benchmarkReport
	if parseErr := json.Unmarshal(parseOld, &parseReport); parseErr != nil {
		parseT.Fatalf("unmarshal old report: %v", parseErr)
	}
	if parseReport.Machine != nil {
		parseT.Fatalf("expected nil machine for old JSON")
	}
	parseOut, _ := json.Marshal(parseReport)
	for _, parseKey := range []string{`"machine"`, `"median"`, `"mad"`, `"n"`} {
		if strings.Contains(string(parseOut), parseKey) {
			parseT.Fatalf("old report round trip gained %s: %s", parseKey, parseOut)
		}
	}
}

func TestParseBenchmarkOutputRecordsMedianMADAndN(parseT *testing.T) {
	parseResults := parseBenchmarkOutput(strings.Join([]string{
		"BenchmarkX-8 1 100 ns/op",
		"BenchmarkX-8 1 110 ns/op",
		"BenchmarkX-8 1 400 ns/op",
		"BenchmarkSingle-8 1 50 ns/op",
	}, "\n"))
	parseX := parseResults[0]
	if parseX.N != 3 || parseX.Median["ns/op"] != 110 || parseX.MAD["ns/op"] != 10 {
		parseT.Fatalf("unexpected spread: n=%d median=%v mad=%v", parseX.N, parseX.Median, parseX.MAD)
	}
	if math.Abs(parseX.AverageMetrics["ns/op"]-203.3333) > 0.01 {
		parseT.Fatalf("mean changed: %v", parseX.AverageMetrics)
	}
	if parseResults[1].N != 0 || parseResults[1].Median != nil {
		parseT.Fatalf("single sample must not record spread: %#v", parseResults[1])
	}
	if medianBenchmarkValues([]float64{1, 2, 3, 4}) != 2.5 {
		parseT.Fatalf("even-length median wrong")
	}
}

func TestBenchmarkStatSelectsMeanOrMedian(parseT *testing.T) {
	parseBuild := func(parseMean float64, parseMedian float64) benchmarkReport {
		return benchmarkReport{GeneratedAt: "x", Packages: []benchmarkPackageReport{{
			Lane: "native", Package: "./a", OK: true,
			Benchmarks: []benchmarkResultReport{{
				Name: "BenchmarkX", AverageMetrics: map[string]float64{"ns/op": parseMean},
				Median: map[string]float64{"ns/op": parseMedian},
			}},
		}}}
	}
	parseBaseline := parseBuild(100, 100)
	parseCurrent := parseBuild(200, 100)
	if parseMean := compareBenchmarkReportsByStat(parseBaseline, parseCurrent, benchmarkStatMean); parseMean.Regressed != 1 {
		parseT.Fatalf("mean path should regress: %#v", parseMean)
	}
	if parseMedian := compareBenchmarkReportsByStat(parseBaseline, parseCurrent, benchmarkStatMedian); parseMedian.Regressed != 0 || parseMedian.Unchanged != 1 {
		parseT.Fatalf("median path should be unchanged: %#v", parseMedian)
	}
	if parseDefault := compareBenchmarkReports(parseBaseline, parseCurrent); parseDefault.Regressed != 1 {
		parseT.Fatalf("default must stay mean: %#v", parseDefault)
	}
	parseRoot := parseT.TempDir()
	parseScoreMean := buildBenchmarkScoreSummary(parseBaseline, parseCurrent, benchmarkConfig{rootPath: parseRoot, referencePath: "r.json"})
	parseScoreMedian := buildBenchmarkScoreSummary(parseBaseline, parseCurrent, benchmarkConfig{rootPath: parseRoot, referencePath: "r.json", stat: benchmarkStatMedian})
	if parseScoreMean.OverallScore != 50 || parseScoreMedian.OverallScore != 100 {
		parseT.Fatalf("scores mean=%v median=%v", parseScoreMean.OverallScore, parseScoreMedian.OverallScore)
	}
	if _, parseErr := normalizeBenchmarkStat("mode"); parseErr == nil {
		parseT.Fatal("expected error for unknown stat")
	}
}

func TestCollectBenchmarkPackagesWithIncludesOptsInSkippedDirs(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBench := "package p\nimport \"testing\"\nfunc BenchmarkX(b *testing.B) {}\n"
	for _, parsePath := range []string{
		"ui/a_test.go",
		"test/render/r_test.go",
		"test/other/o_test.go",
		"test/top_test.go",
		"examples/testing/atlas-perf/p_test.go",
		"examples/public/counter/c_test.go",
		"test/render/tmp/skip_test.go",
	} {
		parseFull := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFull), 0o755); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseErr := os.WriteFile(parseFull, []byte(parseBench), 0o644); parseErr != nil {
			parseT.Fatal(parseErr)
		}
	}
	parseDefault, _, parseErr := collectBenchmarkPackages(parseRoot)
	if parseErr != nil || !slices.Equal(parseDefault, []string{"./ui"}) {
		parseT.Fatalf("default discovery changed: %#v err=%v", parseDefault, parseErr)
	}
	parseIncluded, _, parseErr2 := collectBenchmarkPackagesWithIncludes(parseRoot, []string{"./test/render", "examples/testing/atlas-perf/"})
	if parseErr2 != nil {
		parseT.Fatal(parseErr2)
	}
	parseWant := []string{"./examples/testing/atlas-perf", "./test/render", "./ui"}
	if !slices.Equal(parseIncluded, parseWant) {
		parseT.Fatalf("included discovery = %#v, want %#v", parseIncluded, parseWant)
	}
}
