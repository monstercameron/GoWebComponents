package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"sync"
	"sync/atomic"
	"testing"
	"time"
)

func TestCollectBenchmarkPackagesClassifiesNativeAndWasm(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for parsePath, parseContent := range map[string]string{
		"ui/micro_benchmark_test.go": `package ui
import "testing"
func BenchmarkRenderMicro(b *testing.B) {}
`,
		"fetch/micro_benchmark_wasm_test.go": `//go:build js && wasm
package fetch
import "testing"
func BenchmarkFetchMicro(b *testing.B) {}
`,
		"examples/public/counter/micro_benchmark_test.go": `package ignore
import "testing"
func BenchmarkIgnored(b *testing.B) {}
`,
	} {
		parseFullPath := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFullPath), 0755); parseErr != nil {
			parseT.Fatalf("mkdir %q: %v", parsePath, parseErr)
		}
		if parseErr2 := os.WriteFile(parseFullPath, []byte(parseContent), 0644); parseErr2 != nil {
			parseT.Fatalf("write %q: %v", parsePath, parseErr2)
		}
	}

	parseNativePackages, parseWasmPackages, parseErr3 := collectBenchmarkPackages(parseRoot)
	if parseErr3 != nil {
		parseT.Fatalf("collect benchmark packages: %v", parseErr3)
	}
	if !slices.Equal(parseNativePackages, []string{"./ui"}) {
		parseT.Fatalf("unexpected native packages: %#v", parseNativePackages)
	}
	if !slices.Equal(parseWasmPackages, []string{"./fetch"}) {
		parseT.Fatalf("unexpected wasm packages: %#v", parseWasmPackages)
	}
}

func TestParseBenchmarkOutputGroupsSamples(parseT *testing.T) {
	parseOutput := strings.Join([]string{
		"goos: windows",
		"BenchmarkRenderMicro-8  10  100 ns/op  20 B/op  2 allocs/op",
		"BenchmarkRenderMicro-8  12  110 ns/op  18 B/op  1 allocs/op",
		"BenchmarkOther-8  5  50 ns/op",
	}, "\n")

	parseResults := parseBenchmarkOutput(parseOutput)
	if len(parseResults) != 2 {
		parseT.Fatalf("expected 2 benchmark entries, got %#v", parseResults)
	}
	if parseResults[0].Name != "BenchmarkRenderMicro-8" || len(parseResults[0].Samples) != 2 {
		parseT.Fatalf("expected grouped samples for first benchmark, got %#v", parseResults[0])
	}
	if parseGot := parseResults[0].AverageMetrics["ns/op"]; parseGot != 105 {
		parseT.Fatalf("expected averaged ns/op of 105, got %v", parseGot)
	}
	if parseGot2 := parseResults[1].AverageMetrics["ns/op"]; parseGot2 != 50 {
		parseT.Fatalf("expected second ns/op average of 50, got %v", parseGot2)
	}
}

func TestBenchmarkBucketIDClassifiesExpectedFamilies(parseT *testing.T) {
	parseTests := []struct {
		packagePath string
		benchmark   string
		want        string
	}{
		{packagePath: "./prerender", benchmark: "BenchmarkNormalizeRoutePathMicro-8", want: "compute"},
		{packagePath: "./ui", benchmark: "BenchmarkMarshalSSRBootstrapJSON-8", want: "memory"},
		{packagePath: "./internal/runtime", benchmark: "BenchmarkFiberAllocation-8", want: "alloc_runtime"},
		{packagePath: "./internal/runtime", benchmark: "BenchmarkScheduleUpdate-8", want: "sync_concurrency"},
		{packagePath: "./internal/platform/jsdom", benchmark: "BenchmarkWASMDOMAdapterAppendChild-8", want: "end_to_end"},
	}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.benchmark, func(parseT2 *testing.T) {
			if parseGot := benchmarkBucketID(parseTest.packagePath, parseTest.benchmark); parseGot != parseTest.want {
				parseT2.Fatalf("expected bucket %q, got %q", parseTest.want, parseGot)
			}
		})
	}
}

func TestRunBenchmarkComparePrintsInstallHintWhenBenchstatMissing(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "bench-before.txt")
	parseCandidatePath := filepath.Join(parseRoot, "bench-after.txt")
	if parseErr := os.WriteFile(parseBaselinePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr != nil {
		parseT.Fatalf("write baseline file: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate file: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := benchmarkLookPath
	parseT.Cleanup(func() {
		benchmarkLookPath = parseOriginalLookPath
	})
	benchmarkLookPath = func(parseFile string) (string, error) {
		return "", os.ErrNotExist
	}

	if parseErr4 := (launcher{}).runBenchmark([]string{"compare", "-baseline", parseBaselinePath, "-candidate", parseCandidatePath}); parseErr4 != nil {
		parseT.Fatalf("run benchmark compare without benchstat: %v", parseErr4)
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	if !strings.Contains(parseOutput, "benchstat is not installed") {
		parseT.Fatalf("expected install hint output, got %q", parseOutput)
	}
	if !strings.Contains(parseOutput, filepath.ToSlash(parseBaselinePath)) {
		parseT.Fatalf("expected baseline path in output, got %q", parseOutput)
	}
	if !strings.Contains(parseOutput, filepath.ToSlash(parseCandidatePath)) {
		parseT.Fatalf("expected candidate path in output, got %q", parseOutput)
	}
}

func TestRunBenchmarkCompareRunsBenchstatWhenAvailable(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "bench-before.txt")
	parseCandidatePath := filepath.Join(parseRoot, "bench-after.txt")
	if parseErr := os.WriteFile(parseBaselinePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr != nil {
		parseT.Fatalf("write baseline file: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate file: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := benchmarkLookPath
	parseOriginalRunCommand := benchmarkRunCommand
	parseT.Cleanup(func() {
		benchmarkLookPath = parseOriginalLookPath
		benchmarkRunCommand = parseOriginalRunCommand
	})

	benchmarkLookPath = func(parseFile string) (string, error) {
		if parseFile != "benchstat" {
			parseT.Fatalf("expected benchstat lookup, got %q", parseFile)
		}
		return "/usr/bin/benchstat", nil
	}

	isParseCalled := false
	benchmarkRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		isParseCalled = true
		if parseCommand != "benchstat" {
			parseT.Fatalf("expected benchstat command, got %q", parseCommand)
		}
		if !slices.Equal(parseArgs, []string{parseBaselinePath, parseCandidatePath}) {
			parseT.Fatalf("unexpected benchstat args: %#v", parseArgs)
		}
		return "name old time/op new time/op delta\nBenchmarkX 1ns 0.9ns -10%", nil
	}

	if parseErr4 := (launcher{}).runBenchmark([]string{"compare", "-baseline", parseBaselinePath, "-candidate", parseCandidatePath}); parseErr4 != nil {
		parseT.Fatalf("run benchmark compare with benchstat: %v", parseErr4)
	}
	if !isParseCalled {
		parseT.Fatal("expected benchstat command to run")
	}

	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	if !strings.Contains(parseOutput, "GWC bench compare") {
		parseT.Fatalf("expected bench compare header, got %q", parseOutput)
	}
	if !strings.Contains(parseOutput, "BenchmarkX") {
		parseT.Fatalf("expected benchstat table output, got %q", parseOutput)
	}
}

func TestRunBenchmarkCompareAcceptsPositionalPaths(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "bench-before.txt")
	parseCandidatePath := filepath.Join(parseRoot, "bench-after.txt")
	if parseErr := os.WriteFile(parseBaselinePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr != nil {
		parseT.Fatalf("write baseline file: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate file: %v", parseErr2)
	}

	parseStdout, parseRestoreStdout, parseErr3 := captureExamplesStdout()
	if parseErr3 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr3)
	}
	defer parseRestoreStdout()

	parseOriginalLookPath := benchmarkLookPath
	parseOriginalRunCommand := benchmarkRunCommand
	parseT.Cleanup(func() {
		benchmarkLookPath = parseOriginalLookPath
		benchmarkRunCommand = parseOriginalRunCommand
	})

	benchmarkLookPath = func(parseFile string) (string, error) {
		if parseFile != "benchstat" {
			parseT.Fatalf("expected benchstat lookup, got %q", parseFile)
		}
		return "/usr/bin/benchstat", nil
	}
	benchmarkRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "benchstat" {
			parseT.Fatalf("expected benchstat command, got %q", parseCommand)
		}
		if !slices.Equal(parseArgs, []string{parseBaselinePath, parseCandidatePath}) {
			parseT.Fatalf("unexpected benchstat args: %#v", parseArgs)
		}
		return "BenchmarkX", nil
	}

	if parseErr4 := (launcher{}).runBenchmark([]string{"compare", parseBaselinePath, parseCandidatePath}); parseErr4 != nil {
		parseT.Fatalf("run benchmark compare with positional paths: %v", parseErr4)
	}
	parseOutput, parseErr3 := parseStdout()
	if parseErr3 != nil {
		parseT.Fatalf("read stdout: %v", parseErr3)
	}
	if !strings.Contains(parseOutput, "BenchmarkX") {
		parseT.Fatalf("expected benchstat output, got %q", parseOutput)
	}
}

func TestRunBenchmarkCompareAcceptsSinglePositionalPathWhenOneFlagMissing(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseBaselinePath := filepath.Join(parseRoot, "bench-before.txt")
	parseCandidatePath := filepath.Join(parseRoot, "bench-after.txt")
	if parseErr := os.WriteFile(parseBaselinePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr != nil {
		parseT.Fatalf("write baseline file: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(parseCandidatePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); parseErr2 != nil {
		parseT.Fatalf("write candidate file: %v", parseErr2)
	}

	parseOriginalLookPath := benchmarkLookPath
	parseOriginalRunCommand := benchmarkRunCommand
	parseT.Cleanup(func() {
		benchmarkLookPath = parseOriginalLookPath
		benchmarkRunCommand = parseOriginalRunCommand
	})
	benchmarkLookPath = func(parseFile string) (string, error) { return "/usr/bin/benchstat", nil }
	benchmarkRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if !slices.Equal(parseArgs, []string{parseBaselinePath, parseCandidatePath}) {
			parseT.Fatalf("unexpected benchstat args: %#v", parseArgs)
		}
		return "", nil
	}

	if parseErr3 := (launcher{}).runBenchmark([]string{"compare", "-baseline", parseBaselinePath, parseCandidatePath}); parseErr3 != nil {
		parseT.Fatalf("run benchmark compare with mixed flag and positional path: %v", parseErr3)
	}
}

func TestRunBenchmarkCaptureWritesRawOutputFile(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseOutputPath := filepath.Join(parseRoot, "bench-output.txt")

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
	})

	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("expected go command, got %q", parseCommand)
		}
		parseWant := []string{"test", "-exec", "./tools/go_js_wasm_exec.bat", "./internal/platform/jsdom", "-run", "^$", "-bench", "BenchmarkRuntime", "-benchmem", "-count", "2"}
		if !slices.Equal(parseArgs, parseWant) {
			parseT.Fatalf("unexpected go test args: %#v", parseArgs)
		}
		return "BenchmarkRuntime-8 2 100 ns/op 8 B/op 1 allocs/op", nil
	}

	parseStdout, parseRestoreStdout, parseErr := captureExamplesStdout()
	if parseErr != nil {
		parseT.Fatalf("capture stdout: %v", parseErr)
	}
	defer parseRestoreStdout()

	if parseErr2 := (launcher{}).runBenchmark([]string{"capture", "-package", "./internal/platform/jsdom", "-count", "2", "-bench", "BenchmarkRuntime", "-exec", "./tools/go_js_wasm_exec.bat", "-output", parseOutputPath}); parseErr2 != nil {
		parseT.Fatalf("run benchmark capture: %v", parseErr2)
	}

	parseContent, parseErr := os.ReadFile(parseOutputPath)
	if parseErr != nil {
		parseT.Fatalf("read benchmark capture output: %v", parseErr)
	}
	if !strings.Contains(string(parseContent), "BenchmarkRuntime-8") {
		parseT.Fatalf("expected benchmark output in file, got %q", string(parseContent))
	}

	parseHumanOutput, parseErr := parseStdout()
	if parseErr != nil {
		parseT.Fatalf("read stdout: %v", parseErr)
	}
	if !strings.Contains(parseHumanOutput, "GWC bench capture") {
		parseT.Fatalf("expected capture summary output, got %q", parseHumanOutput)
	}
}

func TestRunBenchmarkCaptureUsesDefaultOutputPathUnderRepoTools(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	if parseErr := os.MkdirAll(filepath.Join(parseRoot, "tools"), 0755); parseErr != nil {
		parseT.Fatalf("mkdir tools dir: %v", parseErr)
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
	})
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		return "BenchmarkRuntime-8 1 100 ns/op", nil
	}

	parseStdout, parseRestoreStdout, parseErr2 := captureExamplesStdout()
	if parseErr2 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr2)
	}
	defer parseRestoreStdout()

	if parseErr3 := (launcher{repoRoot: parseRoot}).runBenchmark([]string{"capture", "-package", "./internal/runtime", "-json"}); parseErr3 != nil {
		parseT.Fatalf("run benchmark capture with default output: %v", parseErr3)
	}
	parseOutput, parseErr2 := parseStdout()
	if parseErr2 != nil {
		parseT.Fatalf("read stdout: %v", parseErr2)
	}
	var parseSummary benchmarkCaptureSummary
	if parseErr4 := json.Unmarshal([]byte(parseOutput), &parseSummary); parseErr4 != nil {
		parseT.Fatalf("decode capture summary: %v\n%s", parseErr4, parseOutput)
	}
	if !strings.Contains(parseSummary.OutputPath, "/tools/bench-") {
		parseT.Fatalf("expected output path under tools dir, got %#v", parseSummary)
	}
	if _, parseErr5 := os.Stat(filepath.FromSlash(parseSummary.OutputPath)); parseErr5 != nil {
		parseT.Fatalf("expected output file to exist: %v", parseErr5)
	}
}

func TestBenchmarkScoreGraphUsesExpectedScale(parseT *testing.T) {
	parseTests := []struct {
		name  string
		score float64
		want  string
	}{
		{name: "zero", score: 0, want: "[--------------------]"},
		{name: "reference", score: 100, want: "[##########----------]"},
		{name: "high", score: 200, want: "[####################]"},
	}
	for _, parseTest := range parseTests {
		parseT.Run(parseTest.name, func(parseT2 *testing.T) {
			if parseGot := benchmarkScoreGraph(parseTest.score, 20); parseGot != parseTest.want {
				parseT2.Fatalf("benchmarkScoreGraph(%v, 20) = %q, want %q", parseTest.score, parseGot, parseTest.want)
			}
		})
	}
}

func TestRunBenchmarkWritesJSONReport(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for parsePath, parseContent := range map[string]string{
		"ui/micro_benchmark_test.go": `package ui
import "testing"
func BenchmarkRenderMicro(b *testing.B) {}
`,
		"fetch/micro_benchmark_wasm_test.go": `//go:build js && wasm
package fetch
import "testing"
func BenchmarkFetchMicro(b *testing.B) {}
`,
	} {
		parseFullPath := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFullPath), 0755); parseErr != nil {
			parseT.Fatalf("mkdir %q: %v", parsePath, parseErr)
		}
		if parseErr2 := os.WriteFile(parseFullPath, []byte(parseContent), 0644); parseErr2 != nil {
			parseT.Fatalf("write %q: %v", parsePath, parseErr2)
		}
	}
	parseReferenceReport := benchmarkReport{
		GeneratedAt: "2026-03-24T00:00:00Z",
		GoVersion:   "go1.24.9",
		GOOS:        "windows",
		GOARCH:      "amd64",
		Packages: []benchmarkPackageReport{
			{
				Lane:    "native",
				Package: "./ui",
				OK:      true,
				Benchmarks: []benchmarkResultReport{{
					Name:           "BenchmarkRenderMicro-8",
					Bucket:         "compute",
					AverageMetrics: map[string]float64{"ns/op": 200},
				}},
			},
			{
				Lane:    "wasm",
				Package: "./fetch",
				OK:      true,
				Benchmarks: []benchmarkResultReport{{
					Name:           "BenchmarkFetchMicro-8",
					Bucket:         "compute",
					AverageMetrics: map[string]float64{"ns/op": 100},
				}},
			},
		},
	}
	parseReferencePath := filepath.Join(parseRoot, "docs", "benchmarks", "reference.json")
	if parseErr3 := os.MkdirAll(filepath.Dir(parseReferencePath), 0755); parseErr3 != nil {
		parseT.Fatalf("mkdir reference dir: %v", parseErr3)
	}
	parseReferencePayload, parseErr4 := json.Marshal(parseReferenceReport)
	if parseErr4 != nil {
		parseT.Fatalf("marshal reference report: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseReferencePath, parseReferencePayload, 0644); parseErr5 != nil {
		parseT.Fatalf("write reference report: %v", parseErr5)
	}

	parseStdout, parseRestoreStdout, parseErr4 := captureExamplesStdout()
	if parseErr4 != nil {
		parseT.Fatalf("capture stdout: %v", parseErr4)
	}
	defer parseRestoreStdout()

	parseOriginalRunCommand := launcherRunCommand
	parseOriginalResolveWasmExec := benchmarkResolveWasmExec
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
		benchmarkResolveWasmExec = parseOriginalResolveWasmExec
	})

	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("unexpected command %q", parseCommand)
		}
		if len(parseArgs) >= 2 && parseArgs[0] == "env" && parseArgs[1] == "GOVERSION" {
			return "go1.25.0", nil
		}
		if slices.Contains(parseArgs, "-exec") {
			return "BenchmarkFetchMicro-8  5  50 ns/op  10 B/op  1 allocs/op", nil
		}
		return "BenchmarkRenderMicro-8  10  100 ns/op  20 B/op  2 allocs/op", nil
	}
	benchmarkResolveWasmExec = func(parseRepoRoot string) (string, error) {
		return filepath.Join(parseRepoRoot, "tools", "go_js_wasm_exec.bat"), nil
	}

	parseReportPath := filepath.Join(parseRoot, "docs", "benchmarks", "latest.json")
	if parseErr6 := (launcher{repoRoot: parseRoot}).runBenchmark([]string{"-root", parseRoot, "-out", parseReportPath, "-json"}); parseErr6 != nil {
		parseT.Fatalf("run benchmark: %v", parseErr6)
	}

	parseOutput, parseErr4 := parseStdout()
	if parseErr4 != nil {
		parseT.Fatalf("read stdout: %v", parseErr4)
	}
	var parseReport benchmarkReport
	if parseErr7 := json.Unmarshal([]byte(parseOutput), &parseReport); parseErr7 != nil {
		parseT.Fatalf("decode benchmark report: %v\n%s", parseErr7, parseOutput)
	}
	if !parseReport.OK || parseReport.PackageCount != 2 || parseReport.BenchmarkCount != 2 {
		parseT.Fatalf("unexpected benchmark report: %#v", parseReport)
	}
	if parseReport.Root != "." || parseReport.ReportPath != "./docs/benchmarks/latest.json" {
		parseT.Fatalf("expected repo-relative root and report path, got %#v", parseReport)
	}
	if parseReport.PackageParallelism != 1 {
		parseT.Fatalf("expected default package parallelism of 1, got %#v", parseReport)
	}
	if parseReport.Scores == nil || parseReport.Scores.ReferencePath != "./docs/benchmarks/reference.json" || parseReport.Scores.OverallScore != 200 {
		parseT.Fatalf("expected reference-backed score summary, got %#v", parseReport)
	}
	if _, parseErr8 := os.Stat(parseReportPath); parseErr8 != nil {
		parseT.Fatalf("expected report file at %s: %v", parseReportPath, parseErr8)
	}
	parseContent2, parseErr4 := os.ReadFile(parseReportPath)
	if parseErr4 != nil {
		parseT.Fatalf("read report file: %v", parseErr4)
	}
	var parseFileReport benchmarkReport
	if parseErr9 := json.Unmarshal(parseContent2, &parseFileReport); parseErr9 != nil {
		parseT.Fatalf("decode written report file: %v", parseErr9)
	}
	if parseFileReport.ReportPath != "./docs/benchmarks/latest.json" {
		parseT.Fatalf("expected written report path %q, got %#v", parseReportPath, parseFileReport)
	}
	if parseFileReport.Scores == nil || parseFileReport.Scores.ReferencePath != "./docs/benchmarks/reference.json" {
		parseT.Fatalf("expected written report scores, got %#v", parseFileReport)
	}
}

func TestExecuteBenchmarkRunsPackagesConcurrentlyAndKeepsOrder(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	for parsePath, parseContent := range map[string]string{
		"pkg/a/micro_benchmark_test.go": `package a
import "testing"
func BenchmarkA(b *testing.B) {}
`,
		"pkg/b/micro_benchmark_test.go": `package b
import "testing"
func BenchmarkB(b *testing.B) {}
`,
	} {
		parseFullPath := filepath.Join(parseRoot, filepath.FromSlash(parsePath))
		if parseErr := os.MkdirAll(filepath.Dir(parseFullPath), 0755); parseErr != nil {
			parseT.Fatalf("mkdir %q: %v", parsePath, parseErr)
		}
		if parseErr2 := os.WriteFile(parseFullPath, []byte(parseContent), 0644); parseErr2 != nil {
			parseT.Fatalf("write %q: %v", parsePath, parseErr2)
		}
	}

	parseOriginalRunCommand := launcherRunCommand
	parseT.Cleanup(func() {
		launcherRunCommand = parseOriginalRunCommand
	})

	var parseActive int32
	var parseSawConcurrent int32
	var parseMu sync.Mutex
	parseSeen := []string{}
	launcherRunCommand = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
		if parseCommand != "go" {
			parseT.Fatalf("unexpected command %q", parseCommand)
		}
		if len(parseArgs) >= 2 && parseArgs[0] == "env" && parseArgs[1] == "GOVERSION" {
			return "go1.25.0", nil
		}
		parseCurrent := atomic.AddInt32(&parseActive, 1)
		if parseCurrent > 1 {
			atomic.StoreInt32(&parseSawConcurrent, 1)
		}
		time.Sleep(40 * time.Millisecond)
		atomic.AddInt32(&parseActive, -1)
		parseMu.Lock()
		parseSeen = append(parseSeen, filepath.Base(parseCwd))
		parseMu.Unlock()
		return "BenchmarkRender-8  10  100 ns/op  20 B/op  2 allocs/op", nil
	}

	parseConfig, parseErr3 := resolveBenchmarkConfig(benchmarkConfig{
		rootPath: parseRoot,
		lanes:    []string{"native"},
		count:    1,
		parallel: 2,
	})
	if parseErr3 != nil {
		parseT.Fatalf("resolve benchmark config: %v", parseErr3)
	}

	parseReport, parseErr3 := (launcher{repoRoot: parseRoot}).executeBenchmark(parseConfig)
	if parseErr3 != nil {
		parseT.Fatalf("execute benchmark: %v", parseErr3)
	}
	if atomic.LoadInt32(&parseSawConcurrent) == 0 {
		parseT.Fatalf("expected concurrent benchmark execution, saw calls %#v", parseSeen)
	}
	if parseWant := []string{"./pkg/a", "./pkg/b"}; !slices.Equal([]string{parseReport.Packages[0].Package, parseReport.Packages[1].Package}, parseWant) {
		parseT.Fatalf("expected deterministic package order %#v, got %#v", parseWant, parseReport.Packages)
	}
	if parseReport.PackageParallelism != 2 {
		parseT.Fatalf("expected report to record parallelism 2, got %#v", parseReport)
	}
}

func TestCompareBenchmarkReportsFlagsMeaningfulDeltas(parseT *testing.T) {
	parseBaseline := benchmarkReport{
		GeneratedAt: "2026-03-24T00:00:00Z",
		Packages: []benchmarkPackageReport{{
			Lane:    "native",
			Package: "./ui",
			OK:      true,
			Benchmarks: []benchmarkResultReport{{
				Name:           "BenchmarkRenderMicro-8",
				AverageMetrics: map[string]float64{"ns/op": 100, "B/op": 20},
			}},
		}},
	}
	parseCurrent := benchmarkReport{
		GeneratedAt: "2026-03-25T00:00:00Z",
		Packages: []benchmarkPackageReport{{
			Lane:    "native",
			Package: "./ui",
			OK:      true,
			Benchmarks: []benchmarkResultReport{{
				Name:           "BenchmarkRenderMicro-8",
				AverageMetrics: map[string]float64{"ns/op": 110, "B/op": 19},
			}},
		}},
	}

	parseComparison := compareBenchmarkReports(parseBaseline, parseCurrent)
	if parseComparison == nil {
		parseT.Fatal("expected comparison summary")
	}
	if parseComparison.MatchedMetrics != 2 {
		parseT.Fatalf("expected 2 matched metrics, got %#v", parseComparison)
	}
	if parseComparison.Regressed != 1 || parseComparison.Improved != 1 {
		parseT.Fatalf("expected one regression and one improvement, got %#v", parseComparison)
	}
}

func TestBuildBenchmarkScoreSummaryUsesReferenceNormalizedGeomean(parseT *testing.T) {
	parseReference := benchmarkReport{
		GeneratedAt: "2026-03-24T00:00:00Z",
		GoVersion:   "go1.25.0",
		GOOS:        "windows",
		GOARCH:      "amd64",
		Packages: []benchmarkPackageReport{
			{
				Lane:    "native",
				Package: "./prerender",
				OK:      true,
				Benchmarks: []benchmarkResultReport{{
					Name:           "BenchmarkNormalizeRoutePathMicro-8",
					Bucket:         "compute",
					AverageMetrics: map[string]float64{"ns/op": 100},
				}},
			},
			{
				Lane:    "native",
				Package: "./ui",
				OK:      true,
				Benchmarks: []benchmarkResultReport{{
					Name:           "BenchmarkMarshalSSRBootstrapJSON-8",
					Bucket:         "memory",
					AverageMetrics: map[string]float64{"ns/op": 200},
				}},
			},
		},
	}
	parseCurrent := benchmarkReport{
		Packages: []benchmarkPackageReport{
			{
				Lane:    "native",
				Package: "./prerender",
				OK:      true,
				Benchmarks: []benchmarkResultReport{{
					Name:           "BenchmarkNormalizeRoutePathMicro-8",
					Bucket:         "compute",
					AverageMetrics: map[string]float64{"ns/op": 50},
				}},
			},
			{
				Lane:    "native",
				Package: "./ui",
				OK:      true,
				Benchmarks: []benchmarkResultReport{{
					Name:           "BenchmarkMarshalSSRBootstrapJSON-8",
					Bucket:         "memory",
					AverageMetrics: map[string]float64{"ns/op": 400},
				}},
			},
		},
	}
	parseConfig := benchmarkConfig{
		rootPath:      `C:\repo`,
		referencePath: `C:\repo\docs\benchmarks\reference.json`,
	}

	parseSummary := buildBenchmarkScoreSummary(parseReference, parseCurrent, parseConfig)
	if parseSummary == nil {
		parseT.Fatal("expected score summary")
	}
	if parseSummary.ReferencePath != "./docs/benchmarks/reference.json" {
		parseT.Fatalf("unexpected reference path: %#v", parseSummary)
	}
	if parseSummary.MatchedBenchmarks != 2 {
		parseT.Fatalf("expected 2 matched benchmarks, got %#v", parseSummary)
	}
	if len(parseSummary.Buckets) != 2 {
		parseT.Fatalf("expected 2 bucket scores, got %#v", parseSummary)
	}
	if parseSummary.Buckets[0].ID != "compute" || parseSummary.Buckets[0].Score != 200 {
		parseT.Fatalf("expected compute score 200, got %#v", parseSummary.Buckets)
	}
	if parseSummary.Buckets[1].ID != "memory" || parseSummary.Buckets[1].Score != 50 {
		parseT.Fatalf("expected memory score 50, got %#v", parseSummary.Buckets)
	}
	if parseSummary.OverallScore != 100 {
		parseT.Fatalf("expected overall score 100 from bucket geomean, got %#v", parseSummary)
	}
}
