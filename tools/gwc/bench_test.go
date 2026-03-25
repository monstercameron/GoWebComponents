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

func TestCollectBenchmarkPackagesClassifiesNativeAndWasm(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"ui/micro_benchmark_test.go": `package ui
import "testing"
func BenchmarkRenderMicro(b *testing.B) {}
`,
		"fetch/micro_benchmark_wasm_test.go": `//go:build js && wasm
package fetch
import "testing"
func BenchmarkFetchMicro(b *testing.B) {}
`,
		"examples/01-counter/micro_benchmark_test.go": `package ignore
import "testing"
func BenchmarkIgnored(b *testing.B) {}
`,
	} {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir %q: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	nativePackages, wasmPackages, err := collectBenchmarkPackages(root)
	if err != nil {
		t.Fatalf("collect benchmark packages: %v", err)
	}
	if !slices.Equal(nativePackages, []string{"./ui"}) {
		t.Fatalf("unexpected native packages: %#v", nativePackages)
	}
	if !slices.Equal(wasmPackages, []string{"./fetch"}) {
		t.Fatalf("unexpected wasm packages: %#v", wasmPackages)
	}
}

func TestParseBenchmarkOutputGroupsSamples(t *testing.T) {
	output := strings.Join([]string{
		"goos: windows",
		"BenchmarkRenderMicro-8  10  100 ns/op  20 B/op  2 allocs/op",
		"BenchmarkRenderMicro-8  12  110 ns/op  18 B/op  1 allocs/op",
		"BenchmarkOther-8  5  50 ns/op",
	}, "\n")

	results := parseBenchmarkOutput(output)
	if len(results) != 2 {
		t.Fatalf("expected 2 benchmark entries, got %#v", results)
	}
	if results[0].Name != "BenchmarkRenderMicro-8" || len(results[0].Samples) != 2 {
		t.Fatalf("expected grouped samples for first benchmark, got %#v", results[0])
	}
	if got := results[0].AverageMetrics["ns/op"]; got != 105 {
		t.Fatalf("expected averaged ns/op of 105, got %v", got)
	}
	if got := results[1].AverageMetrics["ns/op"]; got != 50 {
		t.Fatalf("expected second ns/op average of 50, got %v", got)
	}
}

func TestBenchmarkBucketIDClassifiesExpectedFamilies(t *testing.T) {
	tests := []struct {
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
	for _, test := range tests {
		t.Run(test.benchmark, func(t *testing.T) {
			if got := benchmarkBucketID(test.packagePath, test.benchmark); got != test.want {
				t.Fatalf("expected bucket %q, got %q", test.want, got)
			}
		})
	}
}

func TestRunBenchmarkComparePrintsInstallHintWhenBenchstatMissing(t *testing.T) {
	root := t.TempDir()
	baselinePath := filepath.Join(root, "bench-before.txt")
	candidatePath := filepath.Join(root, "bench-after.txt")
	if err := os.WriteFile(baselinePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); err != nil {
		t.Fatalf("write baseline file: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); err != nil {
		t.Fatalf("write candidate file: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalLookPath := benchmarkLookPath
	t.Cleanup(func() {
		benchmarkLookPath = originalLookPath
	})
	benchmarkLookPath = func(file string) (string, error) {
		return "", os.ErrNotExist
	}

	if err := (launcher{}).runBenchmark([]string{"compare", "-baseline", baselinePath, "-candidate", candidatePath}); err != nil {
		t.Fatalf("run benchmark compare without benchstat: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(output, "benchstat is not installed") {
		t.Fatalf("expected install hint output, got %q", output)
	}
	if !strings.Contains(output, filepath.ToSlash(baselinePath)) {
		t.Fatalf("expected baseline path in output, got %q", output)
	}
	if !strings.Contains(output, filepath.ToSlash(candidatePath)) {
		t.Fatalf("expected candidate path in output, got %q", output)
	}
}

func TestRunBenchmarkCompareRunsBenchstatWhenAvailable(t *testing.T) {
	root := t.TempDir()
	baselinePath := filepath.Join(root, "bench-before.txt")
	candidatePath := filepath.Join(root, "bench-after.txt")
	if err := os.WriteFile(baselinePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); err != nil {
		t.Fatalf("write baseline file: %v", err)
	}
	if err := os.WriteFile(candidatePath, []byte("BenchmarkX 1 1 ns/op\n"), 0644); err != nil {
		t.Fatalf("write candidate file: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalLookPath := benchmarkLookPath
	originalRunCommand := benchmarkRunCommand
	t.Cleanup(func() {
		benchmarkLookPath = originalLookPath
		benchmarkRunCommand = originalRunCommand
	})

	benchmarkLookPath = func(file string) (string, error) {
		if file != "benchstat" {
			t.Fatalf("expected benchstat lookup, got %q", file)
		}
		return "/usr/bin/benchstat", nil
	}

	called := false
	benchmarkRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		called = true
		if command != "benchstat" {
			t.Fatalf("expected benchstat command, got %q", command)
		}
		if !slices.Equal(args, []string{baselinePath, candidatePath}) {
			t.Fatalf("unexpected benchstat args: %#v", args)
		}
		return "name old time/op new time/op delta\nBenchmarkX 1ns 0.9ns -10%", nil
	}

	if err := (launcher{}).runBenchmark([]string{"compare", "-baseline", baselinePath, "-candidate", candidatePath}); err != nil {
		t.Fatalf("run benchmark compare with benchstat: %v", err)
	}
	if !called {
		t.Fatal("expected benchstat command to run")
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(output, "GWC bench compare") {
		t.Fatalf("expected bench compare header, got %q", output)
	}
	if !strings.Contains(output, "BenchmarkX") {
		t.Fatalf("expected benchstat table output, got %q", output)
	}
}

func TestRunBenchmarkCaptureWritesRawOutputFile(t *testing.T) {
	root := t.TempDir()
	outputPath := filepath.Join(root, "bench-output.txt")

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() {
		launcherRunCommand = originalRunCommand
	})

	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("expected go command, got %q", command)
		}
		want := []string{"test", "-exec", "./tools/go_js_wasm_exec.bat", "./internal/platform/jsdom", "-run", "^$", "-bench", "BenchmarkRuntime", "-benchmem", "-count", "2"}
		if !slices.Equal(args, want) {
			t.Fatalf("unexpected go test args: %#v", args)
		}
		return "BenchmarkRuntime-8 2 100 ns/op 8 B/op 1 allocs/op", nil
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{}).runBenchmark([]string{"capture", "-package", "./internal/platform/jsdom", "-count", "2", "-bench", "BenchmarkRuntime", "-exec", "./tools/go_js_wasm_exec.bat", "-output", outputPath}); err != nil {
		t.Fatalf("run benchmark capture: %v", err)
	}

	content, err := os.ReadFile(outputPath)
	if err != nil {
		t.Fatalf("read benchmark capture output: %v", err)
	}
	if !strings.Contains(string(content), "BenchmarkRuntime-8") {
		t.Fatalf("expected benchmark output in file, got %q", string(content))
	}

	humanOutput, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	if !strings.Contains(humanOutput, "GWC bench capture") {
		t.Fatalf("expected capture summary output, got %q", humanOutput)
	}
}

func TestRunBenchmarkCaptureUsesDefaultOutputPathUnderRepoTools(t *testing.T) {
	root := t.TempDir()
	if err := os.MkdirAll(filepath.Join(root, "tools"), 0755); err != nil {
		t.Fatalf("mkdir tools dir: %v", err)
	}

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() {
		launcherRunCommand = originalRunCommand
	})
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		return "BenchmarkRuntime-8 1 100 ns/op", nil
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	if err := (launcher{repoRoot: root}).runBenchmark([]string{"capture", "-package", "./internal/runtime", "-json"}); err != nil {
		t.Fatalf("run benchmark capture with default output: %v", err)
	}
	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var summary benchmarkCaptureSummary
	if err := json.Unmarshal([]byte(output), &summary); err != nil {
		t.Fatalf("decode capture summary: %v\n%s", err, output)
	}
	if !strings.Contains(summary.OutputPath, "/tools/bench-") {
		t.Fatalf("expected output path under tools dir, got %#v", summary)
	}
	if _, err := os.Stat(filepath.FromSlash(summary.OutputPath)); err != nil {
		t.Fatalf("expected output file to exist: %v", err)
	}
}

func TestBenchmarkScoreGraphUsesExpectedScale(t *testing.T) {
	tests := []struct {
		name  string
		score float64
		want  string
	}{
		{name: "zero", score: 0, want: "[--------------------]"},
		{name: "reference", score: 100, want: "[##########----------]"},
		{name: "high", score: 200, want: "[####################]"},
	}
	for _, test := range tests {
		t.Run(test.name, func(t *testing.T) {
			if got := benchmarkScoreGraph(test.score, 20); got != test.want {
				t.Fatalf("benchmarkScoreGraph(%v, 20) = %q, want %q", test.score, got, test.want)
			}
		})
	}
}

func TestRunBenchmarkWritesJSONReport(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
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
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir %q: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}
	referenceReport := benchmarkReport{
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
	referencePath := filepath.Join(root, "docs", "benchmarks", "reference.json")
	if err := os.MkdirAll(filepath.Dir(referencePath), 0755); err != nil {
		t.Fatalf("mkdir reference dir: %v", err)
	}
	referencePayload, err := json.Marshal(referenceReport)
	if err != nil {
		t.Fatalf("marshal reference report: %v", err)
	}
	if err := os.WriteFile(referencePath, referencePayload, 0644); err != nil {
		t.Fatalf("write reference report: %v", err)
	}

	stdout, restoreStdout, err := captureExamplesStdout()
	if err != nil {
		t.Fatalf("capture stdout: %v", err)
	}
	defer restoreStdout()

	originalRunCommand := launcherRunCommand
	originalResolveWasmExec := benchmarkResolveWasmExec
	t.Cleanup(func() {
		launcherRunCommand = originalRunCommand
		benchmarkResolveWasmExec = originalResolveWasmExec
	})

	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("unexpected command %q", command)
		}
		if len(args) >= 2 && args[0] == "env" && args[1] == "GOVERSION" {
			return "go1.25.0", nil
		}
		if slices.Contains(args, "-exec") {
			return "BenchmarkFetchMicro-8  5  50 ns/op  10 B/op  1 allocs/op", nil
		}
		return "BenchmarkRenderMicro-8  10  100 ns/op  20 B/op  2 allocs/op", nil
	}
	benchmarkResolveWasmExec = func(repoRoot string) (string, error) {
		return filepath.Join(repoRoot, "tools", "go_js_wasm_exec.bat"), nil
	}

	reportPath := filepath.Join(root, "docs", "benchmarks", "latest.json")
	if err := (launcher{repoRoot: root}).runBenchmark([]string{"-root", root, "-out", reportPath, "-json"}); err != nil {
		t.Fatalf("run benchmark: %v", err)
	}

	output, err := stdout()
	if err != nil {
		t.Fatalf("read stdout: %v", err)
	}
	var report benchmarkReport
	if err := json.Unmarshal([]byte(output), &report); err != nil {
		t.Fatalf("decode benchmark report: %v\n%s", err, output)
	}
	if !report.OK || report.PackageCount != 2 || report.BenchmarkCount != 2 {
		t.Fatalf("unexpected benchmark report: %#v", report)
	}
	if report.Root != "." || report.ReportPath != "./docs/benchmarks/latest.json" {
		t.Fatalf("expected repo-relative root and report path, got %#v", report)
	}
	if report.PackageParallelism != 1 {
		t.Fatalf("expected default package parallelism of 1, got %#v", report)
	}
	if report.Scores == nil || report.Scores.ReferencePath != "./docs/benchmarks/reference.json" || report.Scores.OverallScore != 200 {
		t.Fatalf("expected reference-backed score summary, got %#v", report)
	}
	if _, err := os.Stat(reportPath); err != nil {
		t.Fatalf("expected report file at %s: %v", reportPath, err)
	}
	content, err := os.ReadFile(reportPath)
	if err != nil {
		t.Fatalf("read report file: %v", err)
	}
	var fileReport benchmarkReport
	if err := json.Unmarshal(content, &fileReport); err != nil {
		t.Fatalf("decode written report file: %v", err)
	}
	if fileReport.ReportPath != "./docs/benchmarks/latest.json" {
		t.Fatalf("expected written report path %q, got %#v", reportPath, fileReport)
	}
	if fileReport.Scores == nil || fileReport.Scores.ReferencePath != "./docs/benchmarks/reference.json" {
		t.Fatalf("expected written report scores, got %#v", fileReport)
	}
}

func TestExecuteBenchmarkRunsPackagesConcurrentlyAndKeepsOrder(t *testing.T) {
	root := t.TempDir()
	for path, content := range map[string]string{
		"pkg/a/micro_benchmark_test.go": `package a
import "testing"
func BenchmarkA(b *testing.B) {}
`,
		"pkg/b/micro_benchmark_test.go": `package b
import "testing"
func BenchmarkB(b *testing.B) {}
`,
	} {
		fullPath := filepath.Join(root, filepath.FromSlash(path))
		if err := os.MkdirAll(filepath.Dir(fullPath), 0755); err != nil {
			t.Fatalf("mkdir %q: %v", path, err)
		}
		if err := os.WriteFile(fullPath, []byte(content), 0644); err != nil {
			t.Fatalf("write %q: %v", path, err)
		}
	}

	originalRunCommand := launcherRunCommand
	t.Cleanup(func() {
		launcherRunCommand = originalRunCommand
	})

	var active int32
	var sawConcurrent int32
	var mu sync.Mutex
	seen := []string{}
	launcherRunCommand = func(command string, args []string, cwd string, env []string) (string, error) {
		if command != "go" {
			t.Fatalf("unexpected command %q", command)
		}
		if len(args) >= 2 && args[0] == "env" && args[1] == "GOVERSION" {
			return "go1.25.0", nil
		}
		current := atomic.AddInt32(&active, 1)
		if current > 1 {
			atomic.StoreInt32(&sawConcurrent, 1)
		}
		time.Sleep(40 * time.Millisecond)
		atomic.AddInt32(&active, -1)
		mu.Lock()
		seen = append(seen, filepath.Base(cwd))
		mu.Unlock()
		return "BenchmarkRender-8  10  100 ns/op  20 B/op  2 allocs/op", nil
	}

	config, err := resolveBenchmarkConfig(benchmarkConfig{
		rootPath: root,
		lanes:    []string{"native"},
		count:    1,
		parallel: 2,
	})
	if err != nil {
		t.Fatalf("resolve benchmark config: %v", err)
	}

	report, err := (launcher{repoRoot: root}).executeBenchmark(config)
	if err != nil {
		t.Fatalf("execute benchmark: %v", err)
	}
	if atomic.LoadInt32(&sawConcurrent) == 0 {
		t.Fatalf("expected concurrent benchmark execution, saw calls %#v", seen)
	}
	if want := []string{"./pkg/a", "./pkg/b"}; !slices.Equal([]string{report.Packages[0].Package, report.Packages[1].Package}, want) {
		t.Fatalf("expected deterministic package order %#v, got %#v", want, report.Packages)
	}
	if report.PackageParallelism != 2 {
		t.Fatalf("expected report to record parallelism 2, got %#v", report)
	}
}

func TestCompareBenchmarkReportsFlagsMeaningfulDeltas(t *testing.T) {
	baseline := benchmarkReport{
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
	current := benchmarkReport{
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

	comparison := compareBenchmarkReports(baseline, current)
	if comparison == nil {
		t.Fatal("expected comparison summary")
	}
	if comparison.MatchedMetrics != 2 {
		t.Fatalf("expected 2 matched metrics, got %#v", comparison)
	}
	if comparison.Regressed != 1 || comparison.Improved != 1 {
		t.Fatalf("expected one regression and one improvement, got %#v", comparison)
	}
}

func TestBuildBenchmarkScoreSummaryUsesReferenceNormalizedGeomean(t *testing.T) {
	reference := benchmarkReport{
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
	current := benchmarkReport{
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
	config := benchmarkConfig{
		rootPath:      `C:\repo`,
		referencePath: `C:\repo\docs\benchmarks\reference.json`,
	}

	summary := buildBenchmarkScoreSummary(reference, current, config)
	if summary == nil {
		t.Fatal("expected score summary")
	}
	if summary.ReferencePath != "./docs/benchmarks/reference.json" {
		t.Fatalf("unexpected reference path: %#v", summary)
	}
	if summary.MatchedBenchmarks != 2 {
		t.Fatalf("expected 2 matched benchmarks, got %#v", summary)
	}
	if len(summary.Buckets) != 2 {
		t.Fatalf("expected 2 bucket scores, got %#v", summary)
	}
	if summary.Buckets[0].ID != "compute" || summary.Buckets[0].Score != 200 {
		t.Fatalf("expected compute score 200, got %#v", summary.Buckets)
	}
	if summary.Buckets[1].ID != "memory" || summary.Buckets[1].Score != 50 {
		t.Fatalf("expected memory score 50, got %#v", summary.Buckets)
	}
	if summary.OverallScore != 100 {
		t.Fatalf("expected overall score 100 from bucket geomean, got %#v", summary)
	}
}
