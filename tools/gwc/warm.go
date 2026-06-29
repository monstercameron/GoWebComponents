package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
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
// timing pair to gate on. SpeedupRatio = warm/cold is the machine-independent invariant CI
// gates on (a hot cache must make the warm rebuild dramatically faster than the cold one);
// absolute milliseconds vary by hardware and are published for trend visibility, not gated.
type warmReport struct {
	OK             bool    `json:"ok"`
	Root           string  `json:"root"`
	Pattern        string  `json:"pattern"`
	Target         string  `json:"target"`
	Cycles         int     `json:"cycles"`
	ColdDurationMs int64   `json:"coldDurationMs"`
	WarmDurationMs int64   `json:"warmDurationMs"`
	LastDurationMs int64   `json:"lastDurationMs"`
	SpeedupRatio   float64 `json:"speedupRatio"`
	MaxWarmRatio   float64 `json:"maxWarmRatio,omitempty"`
	LastError      string  `json:"lastError,omitempty"`
}

// evaluateWarmRatio computes warm/cold and reports whether the cache delivered an acceptable
// speedup (warm at most maxRatio of cold). A zero/near-zero cold build can't be divided
// meaningfully, so it is treated as "no signal" → passes. This is the pure, machine-independent
// core the CI gate and the unit test both exercise.
func evaluateWarmRatio(parseCold time.Duration, parseWarm time.Duration, parseMaxRatio float64) (float64, bool) {
	if parseCold <= 0 {
		return 0, true
	}
	parseRatio := float64(parseWarm) / float64(parseCold)
	return parseRatio, parseRatio <= parseMaxRatio
}

// writeBuildTimesBaseline publishes the cold/warm report to a JSON file (e.g.
// docs/benchmarks/build-times.json) so the numbers are visible and trend over time. The
// machine-specific absolute Root is cleared so the committed snapshot is portable and doesn't
// leak a local path — the gate recomputes timings fresh anyway and gates on the ratio.
func writeBuildTimesBaseline(parsePath string, parseReport warmReport) error {
	parseReport.Root = ""
	parseBytes, parseErr := json.MarshalIndent(parseReport, "", "  ")
	if parseErr != nil {
		return parseErr
	}
	if parseDir := filepath.Dir(parsePath); parseDir != "" {
		_ = os.MkdirAll(parseDir, 0o755)
	}
	return os.WriteFile(parsePath, append(parseBytes, '\n'), 0o644)
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
	parseOnce := parseFlags.Bool("once", false, "Warm the cache once and exit (CI mode): cold build then an immediate warm rebuild")
	parseJSON := parseFlags.Bool("json", false, "Emit the cold/warm timing report as JSON")
	parseBaseline := parseFlags.String("baseline", "", "Path to publish the cold/warm timing report (e.g. docs/benchmarks/build-times.json)")
	parseMaxWarmRatio := parseFlags.Float64("max-warm-ratio", 0.6, "Maximum acceptable warm/cold build-time ratio (cache-effectiveness gate)")
	parseFailOnRegression := parseFlags.Bool("fail-on-regression", false, "Exit non-zero if the warm/cold speedup is worse than -max-warm-ratio")
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

	parseReport := warmReport{Root: parseRootPath, Pattern: *parsePattern, Target: parseTargetName, MaxWarmRatio: *parseMaxWarmRatio}

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
		// Warm rebuild: with the cache now populated, an immediate rebuild measures the warm
		// (cache-hit) time. The cold/warm pair is the whole point of `gwc warm -once` in CI.
		parseGateErr := parseColdErr
		if parseColdErr == nil {
			parseWarmDuration, parseWarmErr := warmBuildOnce(parseRootPath, *parsePattern, parseTargetName)
			parseReport.Cycles = 2
			parseReport.WarmDurationMs = parseWarmDuration.Milliseconds()
			parseReport.LastDurationMs = parseWarmDuration.Milliseconds()
			parseReport.OK = parseWarmErr == nil
			if parseWarmErr != nil {
				parseReport.LastError = parseWarmErr.Error()
				parseGateErr = parseWarmErr
			} else {
				parseRatio, parseRatioOK := evaluateWarmRatio(parseColdDuration, parseWarmDuration, *parseMaxWarmRatio)
				parseReport.SpeedupRatio = parseRatio
				if *parseFailOnRegression && !parseRatioOK {
					parseReport.OK = false
					parseGateErr = fmt.Errorf("build-cache speedup regressed: warm/cold ratio %.3f exceeds max %.3f (cold %dms, warm %dms)",
						parseRatio, *parseMaxWarmRatio, parseReport.ColdDurationMs, parseReport.WarmDurationMs)
					parseReport.LastError = parseGateErr.Error()
				}
			}
		}

		if *parseBaseline != "" && parseReport.OK {
			if parseWriteErr := writeBuildTimesBaseline(*parseBaseline, parseReport); parseWriteErr != nil {
				return parseWriteErr
			}
		}

		if *parseJSON {
			_ = writeAgenticEnvelope("warm", parseReport.OK, parseReport, nil, parseGateErr)
		} else {
			fmt.Printf("gwc warm: cold %dms → warm %dms (%.1f%% of cold) for %s\n",
				parseReport.ColdDurationMs, parseReport.WarmDurationMs, parseReport.SpeedupRatio*100, parseReport.Pattern)
		}
		return parseGateErr
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
