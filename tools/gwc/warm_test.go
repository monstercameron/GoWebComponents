package main

import (
	"errors"
	"strings"
	"testing"
)

// TestWarmOnceReportsColdBuild proves `gwc warm -once` runs a single cold build over the
// pattern/target and exits, invoking `go build` with the right arguments.
func TestWarmOnceReportsColdBuild(parseT *testing.T) {
	parseOriginal := warmRunCommand
	defer func() { warmRunCommand = parseOriginal }()

	var parseGotArgs []string
	var parseGotEnv []string
	warmRunCommand = func(parseArgs []string, parseRoot string, parseEnv []string) (string, error) {
		parseGotArgs = parseArgs
		parseGotEnv = parseEnv
		return "", nil
	}

	parseLauncher := launcher{}
	if parseErr := parseLauncher.runWarm([]string{"-root", parseT.TempDir(), "-pattern", "./...", "-once"}); parseErr != nil {
		parseT.Fatalf("runWarm -once: %v", parseErr)
	}
	if len(parseGotArgs) != 2 || parseGotArgs[0] != "build" || parseGotArgs[1] != "./..." {
		parseT.Fatalf("expected `go build ./...`, got %v", parseGotArgs)
	}
	// Native target must NOT carry a wasm GOOS override.
	if joinedEnv := strings.Join(parseGotEnv, " "); strings.Contains(joinedEnv, "GOOS=js") {
		parseT.Fatalf("native warm must not set GOOS=js, env was %v", parseGotEnv)
	}
}

// TestWarmWasmTargetUsesWasmEnv proves -target wasm warms the cache for the browser build (the
// slow one developers most want hot).
func TestWarmWasmTargetUsesWasmEnv(parseT *testing.T) {
	parseOriginal := warmRunCommand
	defer func() { warmRunCommand = parseOriginal }()

	var parseGotEnv []string
	warmRunCommand = func(parseArgs []string, parseRoot string, parseEnv []string) (string, error) {
		parseGotEnv = parseEnv
		return "", nil
	}

	parseLauncher := launcher{}
	if parseErr := parseLauncher.runWarm([]string{"-root", parseT.TempDir(), "-target", "wasm", "-once"}); parseErr != nil {
		parseT.Fatalf("runWarm wasm: %v", parseErr)
	}
	if joinedEnv := strings.Join(parseGotEnv, " "); !strings.Contains(joinedEnv, "GOOS=js") {
		parseT.Fatalf("wasm warm must set GOOS=js, env was %v", parseGotEnv)
	}
}

// TestWarmSurfacesBuildFailure proves a failed cold build is surfaced (non-zero), so a CI
// `gwc warm -once` step fails on a broken build.
func TestWarmSurfacesBuildFailure(parseT *testing.T) {
	parseOriginal := warmRunCommand
	defer func() { warmRunCommand = parseOriginal }()
	warmRunCommand = func([]string, string, []string) (string, error) {
		return "compile error", errors.New("build failed")
	}

	parseLauncher := launcher{}
	if parseErr := parseLauncher.runWarm([]string{"-root", parseT.TempDir(), "-once"}); parseErr == nil {
		parseT.Fatal("expected warm -once to fail when the build fails")
	}
}

// TestWarmRejectsBadTarget proves an unknown target is rejected before any build.
func TestWarmRejectsBadTarget(parseT *testing.T) {
	parseLauncher := launcher{}
	if parseErr := parseLauncher.runWarm([]string{"-root", parseT.TempDir(), "-target", "android", "-once"}); parseErr == nil {
		parseT.Fatal("expected an invalid -target to be rejected")
	}
}
