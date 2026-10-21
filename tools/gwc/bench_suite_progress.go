package main

import (
	"fmt"
	"io"
	"os"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"sync"
	"time"
)

// benchSuiteHeartbeatInterval is how often a long-running unit of work prints a still-running line.
const benchSuiteHeartbeatInterval = 60 * time.Second

// Row statuses printed in progress lines and the final summary table.
const (
	benchSuiteRowOK      = "OK"
	benchSuiteRowFail    = "FAIL"
	benchSuiteRowTimeout = "TIMEOUT"
	benchSuiteRowSkip    = "SKIP"
)

// benchSuiteSummaryRow is one finished unit of work kept for the final summary table.
type benchSuiteSummaryRow struct {
	label    string
	status   string
	entries  int
	duration time.Duration
}

// benchSuiteProgress is the single mutex-protected logger of a benchmark run. Every line goes to the
// log file and, unless the console writer is nil (quiet), to the console, so output from the
// heartbeat goroutine never interleaves with the main loop.
type benchSuiteProgress struct {
	mu        sync.Mutex
	console   io.Writer
	logFile   io.Writer
	now       func() time.Time
	startedAt time.Time
	total     int
	started   int
	finished  int
	durations []time.Duration
	rows      []benchSuiteSummaryRow
}

// buildBenchSuiteProgress creates a progress logger; a nil console or logFile disables that sink.
func buildBenchSuiteProgress(parseConsole io.Writer, parseLogFile io.Writer, parseTotal int, parseNow func() time.Time) *benchSuiteProgress {
	if parseNow == nil {
		parseNow = time.Now
	}
	return &benchSuiteProgress{console: parseConsole, logFile: parseLogFile, now: parseNow, startedAt: parseNow(), total: parseTotal}
}

// openBenchSuiteLog opens the run log next to the report, truncating it unless isAppend is set.
func openBenchSuiteLog(parsePath string, isAppend bool) (*os.File, error) {
	if strings.TrimSpace(parsePath) == "" {
		return nil, nil
	}
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr != nil {
		return nil, fmt.Errorf("create bench log directory: %w", parseErr)
	}
	parseFlags := os.O_CREATE | os.O_WRONLY | os.O_TRUNC
	if isAppend {
		parseFlags = os.O_CREATE | os.O_WRONLY | os.O_APPEND
	}
	return os.OpenFile(parsePath, parseFlags, 0o644)
}

// formatBenchSuiteDuration renders a duration as 41s, 2m13s or 1h05m02s with whole seconds.
func formatBenchSuiteDuration(parseDuration time.Duration) string {
	parseTotal := int64(parseDuration.Round(time.Second) / time.Second)
	if parseTotal < 0 {
		parseTotal = 0
	}
	parseHours := parseTotal / 3600
	parseMinutes := (parseTotal % 3600) / 60
	parseSeconds := parseTotal % 60
	switch {
	case parseHours > 0:
		return fmt.Sprintf("%dh%02dm%02ds", parseHours, parseMinutes, parseSeconds)
	case parseMinutes > 0:
		return fmt.Sprintf("%dm%02ds", parseMinutes, parseSeconds)
	default:
		return fmt.Sprintf("%ds", parseSeconds)
	}
}

// formatBenchSuiteIndex renders the [ 7/38] counter padded to the width of the total.
func formatBenchSuiteIndex(parseIndex int, parseTotal int) string {
	return fmt.Sprintf("[%*d/%d]", len(fmt.Sprint(parseTotal)), parseIndex, parseTotal)
}

// formatBenchSuiteETA estimates the time left as the mean completed duration times the units
// remaining; it returns "n/a" until one unit has completed.
func formatBenchSuiteETA(parseDurations []time.Duration, parseRemaining int) string {
	if len(parseDurations) == 0 {
		return "n/a"
	}
	if parseRemaining <= 0 {
		return "done"
	}
	var parseSum time.Duration
	for _, parseDuration := range parseDurations {
		parseSum += parseDuration
	}
	parseEstimate := parseSum / time.Duration(len(parseDurations)) * time.Duration(parseRemaining)
	parseMinutes := int64(parseEstimate.Round(time.Minute) / time.Minute)
	switch {
	case parseMinutes <= 0:
		return "~<1m"
	case parseMinutes >= 60:
		return fmt.Sprintf("~%dh%02dm", parseMinutes/60, parseMinutes%60)
	default:
		return fmt.Sprintf("~%dm", parseMinutes)
	}
}

// getBenchSuiteFirstError reduces a failed run's error text to one readable line: the first line
// that looks like an error, else the first non-empty line.
func getBenchSuiteFirstError(parseText string) string {
	parseFirst := ""
	for _, parseRaw := range strings.Split(strings.ReplaceAll(parseText, "\r\n", "\n"), "\n") {
		parseLine := strings.TrimSpace(parseRaw)
		parseLine = strings.TrimPrefix(parseLine, "go test failed: ")
		if parseLine == "" {
			continue
		}
		if parseFirst == "" {
			parseFirst = parseLine
		}
		if benchSuiteErrorLine.MatchString(parseLine) {
			parseFirst = parseLine
			break
		}
	}
	if len(parseFirst) > 200 {
		parseFirst = parseFirst[:200] + "..."
	}
	if parseFirst == "" {
		return "unknown error"
	}
	return parseFirst
}

// writeLine timestamps and writes one line to every sink under the logger mutex.
func (parseP *benchSuiteProgress) writeLine(parseMessage string) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.writeLineLocked(parseMessage)
}

// writeLineLocked is writeLine for callers that already hold the mutex.
func (parseP *benchSuiteProgress) writeLineLocked(parseMessage string) {
	parseLine := fmt.Sprintf("[%s] %s\n", parseP.now().Format("15:04:05"), parseMessage)
	if parseP.logFile != nil {
		_, _ = io.WriteString(parseP.logFile, parseLine)
	}
	if parseP.console != nil {
		_, _ = io.WriteString(parseP.console, parseLine)
	}
}

// setTotal records the number of units once discovery has finished.
func (parseP *benchSuiteProgress) setTotal(parseTotal int) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.total = parseTotal
}

// beginUnit logs a START line and returns the 1-based index of the unit.
func (parseP *benchSuiteProgress) beginUnit(parseLabel string, parseNote string) int {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.started++
	parseLine := fmt.Sprintf("%s START %s", formatBenchSuiteIndex(parseP.started, parseP.total), parseLabel)
	if parseNote != "" {
		parseLine += " (" + parseNote + ")"
	}
	parseP.writeLineLocked(parseLine)
	return parseP.started
}

// logWarmup logs a short warm-up counter line for a unit.
func (parseP *benchSuiteProgress) logWarmup(parseIndex int, parseWarmup int, parseWarmups int, parseLabel string) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.writeLineLocked(fmt.Sprintf("%s warmup %d/%d %s", formatBenchSuiteIndex(parseIndex, parseP.total), parseWarmup, parseWarmups, parseLabel))
}

// finishUnit logs the OK, FAIL or TIMEOUT line of a unit with duration, elapsed time and ETA.
func (parseP *benchSuiteProgress) finishUnit(parseIndex int, parseStatus string, parseLabel string, parseEntries int, parseDetail string, parseDuration time.Duration) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.finished++
	parseP.durations = append(parseP.durations, parseDuration)
	parseP.rows = append(parseP.rows, benchSuiteSummaryRow{label: parseLabel, status: parseStatus, entries: parseEntries, duration: parseDuration})
	parseElapsed := parseP.now().Sub(parseP.startedAt)
	parseP.writeLineLocked(fmt.Sprintf("%s %-5s %s  %s  %s  elapsed %s  ETA %s",
		formatBenchSuiteIndex(parseIndex, parseP.total), parseStatus, parseLabel, parseDetail,
		formatBenchSuiteDuration(parseDuration), formatBenchSuiteDuration(parseElapsed),
		formatBenchSuiteETA(parseP.durations, parseP.total-parseP.finished)))
}

// logSkip records a unit restored from a partial file; it counts as done but not toward the ETA.
func (parseP *benchSuiteProgress) logSkip(parseLabel string, parseEntries int, parseDuration time.Duration) {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseP.started++
	parseP.finished++
	parseP.rows = append(parseP.rows, benchSuiteSummaryRow{label: parseLabel, status: benchSuiteRowSkip, entries: parseEntries, duration: parseDuration})
	parseP.writeLineLocked(fmt.Sprintf("%s %-5s %s  resumed from partial file  %d benchmarks", formatBenchSuiteIndex(parseP.started, parseP.total), benchSuiteRowSkip, parseLabel, parseEntries))
}

// startHeartbeat prints a still-running line every parseInterval until the returned stop is called.
func (parseP *benchSuiteProgress) startHeartbeat(parseLabel string, parseInterval time.Duration) func() {
	parseStart := parseP.now()
	parseStop := make(chan struct{})
	parseDone := make(chan struct{})
	go func() {
		defer close(parseDone)
		parseTicker := time.NewTicker(parseInterval)
		defer parseTicker.Stop()
		for {
			select {
			case <-parseStop:
				return
			case <-parseTicker.C:
				parseP.writeLine(fmt.Sprintf("... still running %s (%s)", parseLabel, formatBenchSuiteDuration(parseP.now().Sub(parseStart))))
			}
		}
	}()
	return func() {
		close(parseStop)
		<-parseDone
	}
}

// writeSummary prints the final table of every unit, slowest first.
func (parseP *benchSuiteProgress) writeSummary() {
	parseP.mu.Lock()
	defer parseP.mu.Unlock()
	parseRows := append([]benchSuiteSummaryRow(nil), parseP.rows...)
	sort.SliceStable(parseRows, func(parseI, parseJ int) bool { return parseRows[parseI].duration > parseRows[parseJ].duration })
	parseWidth := len("TARGET")
	for _, parseRow := range parseRows {
		parseWidth = max(parseWidth, len(parseRow.label))
	}
	parseP.writeLineLocked(fmt.Sprintf("summary: %d targets, slowest first, total %s", len(parseRows), formatBenchSuiteDuration(parseP.now().Sub(parseP.startedAt))))
	parseP.writeLineLocked(fmt.Sprintf("  %-*s  %-7s  %7s  %s", parseWidth, "TARGET", "STATUS", "ENTRIES", "DURATION"))
	for _, parseRow := range parseRows {
		parseP.writeLineLocked(fmt.Sprintf("  %-*s  %-7s  %7d  %s", parseWidth, parseRow.label, parseRow.status, parseRow.entries, formatBenchSuiteDuration(parseRow.duration)))
	}
}

// getBenchSuiteLogPath returns the run log path next to a report; empty when there is no report path.
func getBenchSuiteLogPath(parseOutPath string) string {
	if strings.TrimSpace(parseOutPath) == "" {
		return ""
	}
	return parseOutPath + ".log"
}

// benchSuiteErrorLine picks the line of a failed run's output that states the cause.
var benchSuiteErrorLine = regexp.MustCompile(`(?i)error|undefined|cannot|unable|panic|fatal|denied|syntax|not found|no required module|timed out`)
