package main

import (
	"fmt"
	"os"
	"regexp"
	"runtime"
	"strings"
	"sync"
)

var (
	goCleanupWrapperPrefix = regexp.MustCompile(`(?i)^go (test|build|run)\b[^:]*:\s+`)
	goCleanupExitLine      = regexp.MustCompile(`^exit status \d+$`)
)

// isGoCleanupToleranceHost reports whether this host may hit the locked-exe cleanup
// failure; only Windows tolerates it, every other OS keeps strict exit handling.
var isGoCleanupToleranceHost = func() bool {
	return runtime.GOOS == "windows"
}

var (
	goCleanupWarningsMutex sync.Mutex
	goCleanupWarnings      []string
)

// normalizeChildGoOutput strips the "go test failed:" style wrapper prefixes and bare
// "exit status N" lines so the shared classifier sees raw `go` tool output.
func normalizeChildGoOutput(parseOutput string) string {
	parseLines := strings.Split(strings.ReplaceAll(parseOutput, "\r\n", "\n"), "\n")
	parseKept := make([]string, 0, len(parseLines))
	for _, parseRaw := range parseLines {
		parseLine := strings.TrimSpace(parseRaw)
		if goCleanupExitLine.MatchString(parseLine) {
			continue
		}
		parseKept = append(parseKept, goCleanupWrapperPrefix.ReplaceAllString(parseLine, ""))
	}
	return strings.Join(parseKept, "\n")
}

// isGoCleanupOnlyOutput reports whether parseOutput holds only unlinkat/Access-denied
// *.exe cleanup lines, as left by `go run`/`go build` after a successful run. Any other
// line, including "exit status N", means the child itself failed.
func isGoCleanupOnlyOutput(parseOutput string) bool {
	parseSeen := false
	for _, parseRaw := range strings.Split(strings.ReplaceAll(parseOutput, "\r\n", "\n"), "\n") {
		parseLine := strings.TrimSpace(parseRaw)
		if parseLine == "" {
			continue
		}
		if !benchBenignCleanupLine.MatchString(parseLine) {
			return false
		}
		parseSeen = true
	}
	return parseSeen
}

// classifyGoCleanupOutput returns the cleanup warning lines when parseOutput shows a
// passing run whose only problem is the locked *.exe cleanup failure. It is OS-neutral;
// FAIL, panic, build errors, a missing PASS line, or any other error line yield false.
func classifyGoCleanupOutput(parseOutput string) ([]string, bool) {
	parseNormalized := normalizeChildGoOutput(parseOutput)
	if parseWarnings, isBenign := classifyBenchBenignExit(parseNormalized); isBenign {
		return parseWarnings, true
	}
	// Cleanup-only output keeps "exit status N" lines so a failed child program is never masked.
	parseKept := strings.TrimSpace(goCleanupWrapperPrefix.ReplaceAllString(strings.TrimSpace(parseOutput), ""))
	if isGoCleanupOnlyOutput(parseKept) {
		return strings.Split(strings.ReplaceAll(parseKept, "\r\n", "\n"), "\n"), true
	}
	return nil, false
}

// isBenignGoCleanupFailure reports whether a failed `go` run only failed to delete its
// freshly built *.exe after the work itself succeeded. It is false off Windows and for
// a nil error.
func isBenignGoCleanupFailure(parseOutput string, parseErr error) bool {
	if parseErr == nil || !isGoCleanupToleranceHost() {
		return false
	}
	_, isBenign := classifyGoCleanupOutput(parseOutput)
	return isBenign
}

// recordGoCleanupWarnings stores parseWarnings for the next summary and prints a
// visible warning line to stderr.
func recordGoCleanupWarnings(parseCommand string, parseWarnings []string) {
	goCleanupWarningsMutex.Lock()
	defer goCleanupWarningsMutex.Unlock()
	for _, parseWarning := range parseWarnings {
		parseText := fmt.Sprintf("%s: tolerated Windows exe cleanup failure: %s", parseCommand, parseWarning)
		goCleanupWarnings = append(goCleanupWarnings, parseText)
		fmt.Fprintln(os.Stderr, "warning: "+parseText)
	}
}

// takeGoCleanupWarnings returns and clears the tolerated-cleanup warnings so far.
func takeGoCleanupWarnings() []string {
	goCleanupWarningsMutex.Lock()
	defer goCleanupWarningsMutex.Unlock()
	parseWarnings := goCleanupWarnings
	goCleanupWarnings = nil
	return parseWarnings
}

// tolerateGoCleanupFailure turns a benign cleanup failure into success plus a recorded
// warning; any other error is returned unchanged.
func tolerateGoCleanupFailure(parseCommand string, parseOutput string, parseErr error) error {
	parseCombined := parseOutput
	if strings.TrimSpace(parseCombined) == "" {
		parseCombined = parseErr.Error()
	}
	if !isBenignGoCleanupFailure(parseCombined, parseErr) {
		return parseErr
	}
	parseWarnings, _ := classifyGoCleanupOutput(parseCombined)
	recordGoCleanupWarnings(parseCommand, parseWarnings)
	return nil
}

// runLauncherGoCommand runs a `go` invocation through launcherRunCommand and tolerates
// the benign Windows exe-cleanup failure, recording a warning instead of failing.
func runLauncherGoCommand(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string) (string, error) {
	parseOutput, parseErr := launcherRunCommand(parseCommand, parseArgs, parseCwd, parseEnv)
	if parseErr == nil {
		return parseOutput, nil
	}
	return parseOutput, tolerateGoCleanupFailure("go "+strings.Join(parseArgs, " "), parseOutput, parseErr)
}

// printGoCleanupWarnings prints each tolerated-cleanup warning as a visible human line.
func printGoCleanupWarnings(parseWarnings []string) {
	for _, parseWarning := range parseWarnings {
		fmt.Printf("  warning:      %s\n", parseWarning)
	}
}
