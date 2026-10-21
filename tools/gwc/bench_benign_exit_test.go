package main

import (
	"os"
	"path/filepath"
	"strings"
	"testing"
)

const benchFixtureUnlinkat = `go: unlinkat C:\Users\x\AppData\Local\Temp\go-build123\b001\state.test.exe: Access is denied.`

func TestBenchBenignExitPassWithUnlinkat(parseT *testing.T) {
	parseOut := "goos: windows\nBenchmarkX-8 \t 100\t 5 ns/op\nPASS\nok  \tpkg\t0.5s\n" + benchFixtureUnlinkat
	parseWarnings, isBenign := classifyBenchBenignExit(parseOut)
	if !isBenign || len(parseWarnings) != 1 || !strings.Contains(parseWarnings[0], "unlinkat") {
		parseT.Fatalf("want benign with 1 warning, got %v %v", isBenign, parseWarnings)
	}
}

func TestBenchBenignExitRealFailStaysFailed(parseT *testing.T) {
	parseOut := "--- FAIL: TestA (0.00s)\n    a_test.go:1: boom\nFAIL\nFAIL\tpkg\t0.1s\n" + benchFixtureUnlinkat
	if _, isBenign := classifyBenchBenignExit(parseOut); isBenign {
		parseT.Fatal("real FAIL must not be masked")
	}
}

func TestBenchBenignExitPanicStaysFailed(parseT *testing.T) {
	parseOut := "PASS\npanic: boom\n\ngoroutine 1 [running]:\n" + benchFixtureUnlinkat
	if _, isBenign := classifyBenchBenignExit(parseOut); isBenign {
		parseT.Fatal("panic must not be masked")
	}
}

func TestBenchBenignExitUnknownErrorStaysFailed(parseT *testing.T) {
	parseOut := "PASS\nsome other error: disk on fire\n" + benchFixtureUnlinkat
	if _, isBenign := classifyBenchBenignExit(parseOut); isBenign {
		parseT.Fatal("unknown error line must not be masked")
	}
}

func TestBenchBenignExitPassWithoutUnlinkatIsNotClassified(parseT *testing.T) {
	// A clean PASS exits zero, so the classifier is never consulted; with no cleanup line it
	// declines, leaving a non-zero exit failed.
	if _, isBenign := classifyBenchBenignExit("PASS\nok  \tpkg\t0.5s"); isBenign {
		parseT.Fatal("non-zero exit without a cleanup line must stay failed")
	}
}

func TestBenchRunPackagePassWithUnlinkatRecordsWarning(parseT *testing.T) {
	parseOriginal := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginal })
	launcherRunCommand = func(string, []string, string, []string) (string, error) {
		return "BenchmarkX-8 \t 100\t 5 ns/op\t 0 B/op\t 0 allocs/op\nPASS\nok  \tpkg\t0.5s\n" + benchFixtureUnlinkat, os.ErrPermission
	}
	parseRoot := parseT.TempDir()
	parseReport := (launcher{repoRoot: parseRoot}).runBenchmarkPackage(benchmarkConfig{rootPath: parseRoot, bench: ".", count: 1}, "native", ".")
	if !parseReport.OK || len(parseReport.Warnings) != 1 || parseReport.BenchmarkCount != 1 {
		parseT.Fatalf("want ok with warning, got %+v", parseReport)
	}
}

func TestResolveWasmExecPrefersGoHelperOnWindows(parseT *testing.T) {
	parseOriginal := buildWasmExecHelper
	parseT.Cleanup(func() { buildWasmExecHelper = parseOriginal })
	parseT.Setenv("GO_WASM_EXEC", "")
	parseWant := filepath.Join("x", "gowasmexec.exe")
	buildWasmExecHelper = func(string) (string, bool) { return parseWant, true }
	parseRoot := parseT.TempDir()
	parseGot, parseErr := resolveWasmTestExec(parseRoot)
	if parseErr != nil {
		parseT.Skipf("non-windows host: %v", parseErr)
	}
	if parseGot != parseWant {
		parseT.Fatalf("want %q got %q", parseWant, parseGot)
	}
}
