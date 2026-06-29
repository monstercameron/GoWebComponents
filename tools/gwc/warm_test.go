package main

import (
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"
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

// TestEvaluateWarmRatio proves the cache-effectiveness gate: a fast warm build passes, a warm
// build that is too close to the cold time fails, and a zero cold build is treated as no-signal.
func TestEvaluateWarmRatio(parseT *testing.T) {
	parseCases := []struct {
		name     string
		cold     time.Duration
		warm     time.Duration
		maxRatio float64
		wantOK   bool
	}{
		{"hot cache passes", 1000 * time.Millisecond, 100 * time.Millisecond, 0.6, true},
		{"warm too slow fails", 1000 * time.Millisecond, 800 * time.Millisecond, 0.6, false},
		{"exactly at threshold passes", 1000 * time.Millisecond, 600 * time.Millisecond, 0.6, true},
		{"zero cold is no-signal", 0, 0, 0.6, true},
	}
	for _, parseCase := range parseCases {
		parseRatio, parseOK := evaluateWarmRatio(parseCase.cold, parseCase.warm, parseCase.maxRatio)
		if parseOK != parseCase.wantOK {
			parseT.Fatalf("%s: evaluateWarmRatio(%v,%v,%v) ok=%v (ratio %.3f), want %v",
				parseCase.name, parseCase.cold, parseCase.warm, parseCase.maxRatio, parseOK, parseRatio, parseCase.wantOK)
		}
	}
}

// TestWarmPublishesBaseline proves -baseline writes the cold/warm report to disk so the timings
// are published (the "published" half of A2/C2's platform-honest 10).
func TestWarmPublishesBaseline(parseT *testing.T) {
	parseOriginal := warmRunCommand
	defer func() { warmRunCommand = parseOriginal }()
	warmRunCommand = func([]string, string, []string) (string, error) { return "", nil }

	parsePath := filepath.Join(parseT.TempDir(), "build-times.json")
	parseLauncher := launcher{}
	if parseErr := parseLauncher.runWarm([]string{"-root", parseT.TempDir(), "-once", "-baseline", parsePath}); parseErr != nil {
		parseT.Fatalf("runWarm -baseline: %v", parseErr)
	}
	parseBytes, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		parseT.Fatalf("baseline file not written: %v", parseErr)
	}
	if !strings.Contains(string(parseBytes), "speedupRatio") || !strings.Contains(string(parseBytes), "coldDurationMs") {
		parseT.Fatalf("baseline JSON missing timing fields: %s", parseBytes)
	}
}
