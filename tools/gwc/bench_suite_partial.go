package main

import (
	"bytes"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json"
	"errors"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"time"
)

// benchSuitePartialSchemaVersion is the schema id of the incremental `<out>.partial.json` file.
const benchSuitePartialSchemaVersion = "gwc.bench.suite.partial/v1"

// benchSuiteTimeoutMarker prefixes the error text of a target killed by -target-timeout.
const benchSuiteTimeoutMarker = "timed out after "

// benchSuiteTarget is the outcome of one (lane, package) target; Entries are only kept in the partial file.
type benchSuiteTarget struct {
	Lane       string            `json:"lane"`
	Package    string            `json:"package"`
	Status     string            `json:"status"`
	Error      string            `json:"error,omitempty"`
	DurationMs int64             `json:"durationMs"`
	Warnings   []string          `json:"warnings,omitempty"`
	Entries    []benchSuiteEntry `json:"entries,omitempty"`
	// GoOS and GoArch are the benchmark header values the target's go test printed.
	GoOS   string `json:"goos,omitempty"`
	GoArch string `json:"goarch,omitempty"`
}

// benchSuitePartial is the incrementally rewritten checkpoint of an unfinished suite run.
type benchSuitePartial struct {
	SchemaVersion string             `json:"schemaVersion"`
	ConfigHash    string             `json:"configHash"`
	Seed          int64              `json:"seed"`
	Targets       []benchSuiteTarget `json:"targets"`
	Size          *benchSuiteSize    `json:"size,omitempty"`
	Build         *benchSuiteBuild   `json:"build,omitempty"`
}

// getBenchSuiteTargetID returns the stable id lane/package of a target.
func getBenchSuiteTargetID(parseLane string, parsePackagePath string) string {
	return parseLane + "/" + parsePackagePath
}

// getBenchSuitePartialPath returns the partial checkpoint path for a report path.
func getBenchSuitePartialPath(parseOutPath string) string {
	return parseOutPath + ".partial.json"
}

// hashBenchSuiteConfig fingerprints every setting that changes what a sample means. The root and
// the timeout are excluded so a resumed run may move paths or allow more time.
func hashBenchSuiteConfig(parseConfig benchSuiteConfig) string {
	parseFields := struct {
		Suite        string   `json:"suite"`
		Lanes        []string `json:"lanes"`
		Count        int      `json:"count"`
		Warmup       int      `json:"warmup"`
		HasWarmup    bool     `json:"hasWarmup"`
		Benchtime    string   `json:"benchtime"`
		Bench        string   `json:"bench"`
		Seed         int64    `json:"seed"`
		Shuffle      bool     `json:"shuffle"`
		Include      []string `json:"include"`
		SizePackage  string   `json:"sizePackage"`
		BuildPattern string   `json:"buildPattern"`
		Goarch       string   `json:"goarch,omitempty"`
	}{parseConfig.suiteName, parseConfig.lanes, parseConfig.count, parseConfig.warmup, parseConfig.hasWarmup, parseConfig.benchtime,
		parseConfig.bench, parseConfig.seed, parseConfig.isShuffle, parseConfig.include, parseConfig.sizePackage, parseConfig.buildPattern, parseConfig.goarch}
	parsePayload, _ := json.Marshal(parseFields)
	parseSum := sha256.Sum256(parsePayload)
	return hex.EncodeToString(parseSum[:8])
}

// storeBenchSuitePartial atomically replaces the partial file: write a temp file, then rename it
// over the target (Go's rename replaces existing files on Windows), retrying briefly on lock errors.
func storeBenchSuitePartial(parsePath string, parsePartial benchSuitePartial) error {
	parsePartial.SchemaVersion = benchSuitePartialSchemaVersion
	parsePayload, parseErr := json.MarshalIndent(parsePartial, "", "  ")
	if parseErr != nil {
		return fmt.Errorf("marshal bench suite partial: %w", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr2 != nil {
		return fmt.Errorf("create bench suite partial directory: %w", parseErr2)
	}
	parseTemp := parsePath + ".tmp"
	if parseErr3 := os.WriteFile(parseTemp, append(parsePayload, '\n'), 0o644); parseErr3 != nil {
		return fmt.Errorf("write bench suite partial: %w", parseErr3)
	}
	var parseRenameErr error
	for parseAttempt := 0; parseAttempt < 5; parseAttempt++ {
		if parseRenameErr = os.Rename(parseTemp, parsePath); parseRenameErr == nil {
			return nil
		}
		time.Sleep(20 * time.Millisecond)
	}
	_ = os.Remove(parseTemp)
	return fmt.Errorf("replace bench suite partial: %w", parseRenameErr)
}

// loadBenchSuitePartial reads a partial file and rejects other schema versions.
func loadBenchSuitePartial(parsePath string) (benchSuitePartial, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return benchSuitePartial{}, parseErr
	}
	var parsePartial benchSuitePartial
	if parseErr2 := json.Unmarshal(parseContent, &parsePartial); parseErr2 != nil {
		return benchSuitePartial{}, fmt.Errorf("parse %s: %w", parsePath, parseErr2)
	}
	if parsePartial.SchemaVersion != benchSuitePartialSchemaVersion {
		return benchSuitePartial{}, fmt.Errorf("%s has schemaVersion %q, want %q", parsePath, parsePartial.SchemaVersion, benchSuitePartialSchemaVersion)
	}
	return parsePartial, nil
}

// checkBenchSuiteResume refuses to resume a partial file written with a different configuration.
func checkBenchSuiteResume(parsePartial benchSuitePartial, parseConfig benchSuiteConfig, parsePath string) error {
	parseWant := hashBenchSuiteConfig(parseConfig)
	if parsePartial.ConfigHash != parseWant {
		return fmt.Errorf("cannot resume: %s was written with a different configuration (hash %s, this run %s); "+
			"rerun with the same flags (including -seed %d) or delete the partial file to start over",
			parsePath, parsePartial.ConfigHash, parseWant, parsePartial.Seed)
	}
	return nil
}

// isBenchSuiteTimeoutError reports whether an error text came from -target-timeout.
func isBenchSuiteTimeoutError(parseText string) bool {
	return strings.HasPrefix(strings.TrimSpace(parseText), strings.TrimSpace(benchSuiteTimeoutMarker))
}

// benchmarkRunCommandTimeout runs a command like launcherRunCommand but kills its whole process
// tree and returns a timeout error after parseTimeout; tests replace it.
var benchmarkRunCommandTimeout = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string, parseTimeout time.Duration) (string, error) {
	parseCmd := exec.Command(parseCommand, parseArgs...)
	parseCmd.Dir = parseCwd
	parseCmd.Env = parseEnv
	parseCmd.WaitDelay = 10 * time.Second
	var parseBuffer bytes.Buffer
	parseCmd.Stdout = &parseBuffer
	parseCmd.Stderr = &parseBuffer
	if parseErr := parseCmd.Start(); parseErr != nil {
		return "", fmt.Errorf("%s %s failed: %w", parseCommand, strings.Join(parseArgs, " "), parseErr)
	}
	parseDone := make(chan error, 1)
	go func() { parseDone <- parseCmd.Wait() }()
	parseTimer := time.NewTimer(parseTimeout)
	defer parseTimer.Stop()
	select {
	case parseErr2 := <-parseDone:
		parseText := strings.TrimSpace(parseBuffer.String())
		if parseErr2 != nil {
			if parseText == "" {
				return "", fmt.Errorf("%s %s failed: %w", parseCommand, strings.Join(parseArgs, " "), parseErr2)
			}
			return parseText, fmt.Errorf("%s %s failed: %s", parseCommand, strings.Join(parseArgs, " "), parseText)
		}
		return parseText, nil
	case <-parseTimer.C:
		terminateLauncherProcessTree(parseCmd)
		<-parseDone
		return strings.TrimSpace(parseBuffer.String()), errors.New(benchSuiteTimeoutMarker + parseTimeout.String())
	}
}
