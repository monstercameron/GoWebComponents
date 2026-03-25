package main

import (
	"encoding/json"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
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
