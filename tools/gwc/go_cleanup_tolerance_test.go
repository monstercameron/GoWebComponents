package main

import (
	"errors"
	"strings"
	"testing"
)

const cleanupFixtureUnlinkat = `go: unlinkat C:\Users\x\AppData\Local\Temp\go-build1\b001\pkg.test.exe: Access is denied.`

// useCleanupToleranceHost forces the Windows-only tolerance on or off for one test.
func useCleanupToleranceHost(parseT *testing.T, isWindows bool) {
	parseT.Helper()
	parseOriginal := isGoCleanupToleranceHost
	isGoCleanupToleranceHost = func() bool { return isWindows }
	parseT.Cleanup(func() { isGoCleanupToleranceHost = parseOriginal })
	takeGoCleanupWarnings()
}

func TestGoCleanupBenignPassWithUnlinkat(parseT *testing.T) {
	useCleanupToleranceHost(parseT, true)
	parseOut := "ok  \tpkg\t0.5s\n" + cleanupFixtureUnlinkat
	if !isBenignGoCleanupFailure(parseOut, errors.New("exit status 1")) {
		parseT.Fatal("pass plus unlinkat must be benign")
	}
}

func TestGoCleanupBenignRejectsRealFailures(parseT *testing.T) {
	useCleanupToleranceHost(parseT, true)
	parseCases := map[string]string{
		"fail":        "--- FAIL: TestA (0s)\nFAIL\nFAIL\tpkg\t0.1s\n" + cleanupFixtureUnlinkat,
		"panic":       "PASS\npanic: boom\n" + cleanupFixtureUnlinkat,
		"build error": "# pkg\n./a.go:3:1: undefined: x\nFAIL\tpkg [build failed]\n" + cleanupFixtureUnlinkat,
		"no pass":     cleanupFixtureUnlinkat + "\nexit status 1",
		"not an exe":  "ok  \tpkg\t0.5s\ngo: unlinkat C:/Temp/notes.txt: Access is denied.",
	}
	for parseName, parseOut := range parseCases {
		if isBenignGoCleanupFailure(parseOut, errors.New("exit status 1")) {
			parseT.Fatalf("%s must stay an error", parseName)
		}
	}
}

func TestGoCleanupBenignIsWindowsOnly(parseT *testing.T) {
	useCleanupToleranceHost(parseT, false)
	if isBenignGoCleanupFailure("ok  \tpkg\t0.5s\n"+cleanupFixtureUnlinkat, errors.New("exit status 1")) {
		parseT.Fatal("non-Windows hosts must keep strict behavior")
	}
	useCleanupToleranceHost(parseT, true)
	if isBenignGoCleanupFailure("ok  \tpkg\t0.5s\n"+cleanupFixtureUnlinkat, nil) {
		parseT.Fatal("a nil error is never a cleanup failure")
	}
}

func TestGoCleanupRunnerToleratesAndWarns(parseT *testing.T) {
	useCleanupToleranceHost(parseT, true)
	parseOriginal := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginal })
	launcherRunCommand = func(string, []string, string, []string) (string, error) {
		return "ok  \tpkg\t0.5s\n" + cleanupFixtureUnlinkat, errors.New("go test failed")
	}
	parseOut, parseErr := runLauncherGoCommand("go", []string{"test", "./..."}, "", nil)
	if parseErr != nil || !strings.Contains(parseOut, "ok") {
		parseT.Fatalf("want tolerated success, got %q %v", parseOut, parseErr)
	}
	parseWarnings := takeGoCleanupWarnings()
	if len(parseWarnings) != 1 || !strings.Contains(parseWarnings[0], "unlinkat") {
		parseT.Fatalf("want one warning, got %v", parseWarnings)
	}
	if len(takeGoCleanupWarnings()) != 0 {
		parseT.Fatal("warnings must drain")
	}
}

func TestGoCleanupRunnerKeepsRealFailure(parseT *testing.T) {
	useCleanupToleranceHost(parseT, true)
	parseOriginal := launcherRunCommand
	parseT.Cleanup(func() { launcherRunCommand = parseOriginal })
	launcherRunCommand = func(string, []string, string, []string) (string, error) {
		return "FAIL\tpkg\t0.1s\n" + cleanupFixtureUnlinkat, errors.New("go test failed")
	}
	if _, parseErr := runLauncherGoCommand("go", []string{"test", "./..."}, "", nil); parseErr == nil {
		parseT.Fatal("FAIL must remain an error")
	}
	if len(takeGoCleanupWarnings()) != 0 {
		parseT.Fatal("no warning for a real failure")
	}
}

func TestGoCleanupBuildOnlyUnlinkatIsBenign(parseT *testing.T) {
	useCleanupToleranceHost(parseT, true)
	if !isBenignGoCleanupFailure(cleanupFixtureUnlinkat, errors.New("exit status 1")) {
		parseT.Fatal("cleanup-only output of go build/run is benign")
	}
	if isBenignGoCleanupFailure(cleanupFixtureUnlinkat+"\nexit status 3", errors.New("exit status 1")) {
		parseT.Fatal("a child exit status line is a real failure")
	}
}
