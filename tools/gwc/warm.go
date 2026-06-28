package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"strings"
	"time"
)

// runWarmCommand routes the build-warming daemon.
var runWarmCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runWarm(parseArgs)
}

// warmRunCommand executes the go build and is overridden by tests.
var warmRunCommand = func(parseArgs []string, parseRoot string, parseEnv []string) (string, error) {
	return launcherRunCommand("go", parseArgs, parseRoot, parseEnv)
}

// warmReport is the build daemon's running summary: the cold build time, the most recent (warm)
// build time, and how many warm cycles have run. Published as JSON it gives CI a cold-vs-warm
// timing pair to gate on.
type warmReport struct {
	OK             bool   `json:"ok"`
	Root           string `json:"root"`
	Pattern        string `json:"pattern"`
	Target         string `json:"target"`
	Cycles         int    `json:"cycles"`
	ColdDurationMs int64  `json:"coldDurationMs"`
	LastDurationMs int64  `json:"lastDurationMs"`
	LastError      string `json:"lastError,omitempty"`
}

// runWarm implements `gwc warm`: a persistent build daemon that keeps the Go build cache hot.
// It runs an initial COLD build (timing it), then watches the project and re-runs the build on
// every save so the cache stays warm — meaning the developer's own `go build`/`gwc build` after
// a save hits a populated cache and returns fast. -once warms a single time (for CI), -json emits
// the cold/warm timing report. It complements `gwc buildreport` (what rebuilt & why) by keeping
// the cache primed in the first place.
func (parseL launcher) runWarm(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("warm", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Project root to warm and watch; defaults to the current working directory")
	parsePattern := parseFlags.String("pattern", "./...", "go build package pattern to keep warm")
	parseTarget := parseFlags.String("target", "native", "Build target to warm: native or wasm")
	parseDebounce := parseFlags.Duration("debounce", 500*time.Millisecond, "Change-detection debounce interval")
	parseOnce := parseFlags.Bool("once", false, "Warm the cache once and exit (CI mode)")
	parseJSON := parseFlags.Bool("json", false, "Emit the cold/warm timing report as JSON")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseRootPath, parseErr := resolveAgenticToolRoot(*parseRoot)
	if parseErr != nil {
		return parseErr
	}
	parseTargetName := strings.ToLower(strings.TrimSpace(*parseTarget))
	if parseTargetName != "native" && parseTargetName != "wasm" {
		return fmt.Errorf("invalid -target %q: want native or wasm", *parseTarget)
	}

	parseReport := warmReport{Root: parseRootPath, Pattern: *parsePattern, Target: parseTargetName}

	// Cold build: populate the cache from a clean state.
	parseColdDuration, parseColdErr := warmBuildOnce(parseRootPath, *parsePattern, parseTargetName)
	parseReport.Cycles = 1
	parseReport.ColdDurationMs = parseColdDuration.Milliseconds()
	parseReport.LastDurationMs = parseColdDuration.Milliseconds()
	parseReport.OK = parseColdErr == nil
	if parseColdErr != nil {
		parseReport.LastError = parseColdErr.Error()
	}

	if *parseOnce || *parseJSON {
		if *parseJSON {
			_ = writeAgenticEnvelope("warm", parseReport.OK, parseReport, nil, parseColdErr)
		} else {
			fmt.Printf("gwc warm: cold build %s in %dms\n", parseReport.Pattern, parseReport.ColdDurationMs)
		}
		return parseColdErr
	}

	fmt.Printf("gwc warm: cold build %s in %dms — watching for changes (cache stays hot)\n", parseReport.Pattern, parseReport.ColdDurationMs)
	parseLastFingerprint := fingerprintWatchTree(parseRootPath)
	for {
		time.Sleep(*parseDebounce)
		parseNextFingerprint := fingerprintWatchTree(parseRootPath)
		if parseNextFingerprint == parseLastFingerprint {
			continue
		}
		parseLastFingerprint = parseNextFingerprint

		parseDuration, parseWarmErr := warmBuildOnce(parseRootPath, *parsePattern, parseTargetName)
		parseReport.Cycles++
		parseReport.LastDurationMs = parseDuration.Milliseconds()
		if parseWarmErr != nil {
			parseReport.OK = false
			parseReport.LastError = parseWarmErr.Error()
			fmt.Printf("gwc warm: rebuild failed in %dms: %v\n", parseDuration.Milliseconds(), parseWarmErr)
			continue
		}
		parseReport.OK = true
		parseReport.LastError = ""
		fmt.Printf("gwc warm: cache hot — rebuilt in %dms (cycle %d)\n", parseDuration.Milliseconds(), parseReport.Cycles)
	}
}

// warmBuildOnce runs one `go build` over the pattern for the requested target and returns how
// long it took. The first call after a change is a real compile; subsequent unchanged calls are
// near-instant cache hits — which is the daemon's whole point.
func warmBuildOnce(parseRoot string, parsePattern string, parseTarget string) (time.Duration, error) {
	parseEnv := buildNativeGoEnv()
	if parseTarget == "wasm" {
		parseEnv = buildWasmGoEnv()
	}
	parseStart := time.Now()
	_, parseErr := warmRunCommand([]string{"build", parsePattern}, parseRoot, parseEnv)
	return time.Since(parseStart), parseErr
}
