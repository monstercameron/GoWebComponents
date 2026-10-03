package main

import (
	"bytes"
	"encoding/json"
	"errors"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"testing/synctest"
	"time"
)

func TestProgressFormatterIndexAndDuration(parseT *testing.T) {
	if parseGot := formatBenchSuiteIndex(7, 38); parseGot != "[ 7/38]" {
		parseT.Fatalf("index %q", parseGot)
	}
	if parseGot := formatBenchSuiteIndex(1, 5); parseGot != "[1/5]" {
		parseT.Fatalf("index %q", parseGot)
	}
	for parseDuration, parseWant := range map[time.Duration]string{
		41 * time.Second:                          "41s",
		2*time.Minute + 13*time.Second:            "2m13s",
		18*time.Minute + 2*time.Second:            "18m02s",
		time.Hour + 5*time.Minute + 2*time.Second: "1h05m02s",
		-time.Second:                              "0s",
	} {
		if parseGot := formatBenchSuiteDuration(parseDuration); parseGot != parseWant {
			parseT.Fatalf("duration %v: got %q want %q", parseDuration, parseGot, parseWant)
		}
	}
}

func TestProgressFormatterETAMath(parseT *testing.T) {
	if parseGot := formatBenchSuiteETA(nil, 10); parseGot != "n/a" {
		parseT.Fatalf("first unit must be n/a, got %q", parseGot)
	}
	parseDurations := []time.Duration{2 * time.Minute, 4 * time.Minute}
	if parseGot := formatBenchSuiteETA(parseDurations, 3); parseGot != "~9m" {
		parseT.Fatalf("mean 3m x 3 = 9m, got %q", parseGot)
	}
	if parseGot := formatBenchSuiteETA([]time.Duration{10 * time.Second}, 2); parseGot != "~<1m" {
		parseT.Fatalf("20s remaining, got %q", parseGot)
	}
	if parseGot := formatBenchSuiteETA([]time.Duration{30 * time.Minute}, 3); parseGot != "~1h30m" {
		parseT.Fatalf("90m remaining, got %q", parseGot)
	}
	if parseGot := formatBenchSuiteETA(parseDurations, 0); parseGot != "done" {
		parseT.Fatalf("nothing remaining, got %q", parseGot)
	}
}

func TestProgressLinesStartFinishAndSummary(parseT *testing.T) {
	var parseConsole bytes.Buffer
	parseClock := time.Date(2026, 1, 2, 3, 4, 5, 0, time.UTC)
	parseProgress := buildBenchSuiteProgress(&parseConsole, nil, 2, func() time.Time { return parseClock })

	parseIndex := parseProgress.beginUnit("native ./router", "warmup 1 + count 5")
	parseProgress.logWarmup(parseIndex, 1, 1, "native ./router")
	parseClock = parseClock.Add(2 * time.Minute)
	parseProgress.finishUnit(parseIndex, benchSuiteRowOK, "native ./router", 41, "41 benchmarks", 2*time.Minute)
	parseIndex2 := parseProgress.beginUnit("native ./slow", "")
	parseClock = parseClock.Add(10 * time.Minute)
	parseProgress.finishUnit(parseIndex2, benchSuiteRowFail, "native ./slow", 0, "boom", 10*time.Minute)
	parseProgress.writeSummary()

	parseText := parseConsole.String()
	for _, parseWant := range []string{
		"[03:04:05] [1/2] START native ./router (warmup 1 + count 5)",
		"[1/2] warmup 1/1 native ./router",
		"[03:06:05] [1/2] OK    native ./router  41 benchmarks  2m00s  elapsed 2m00s  ETA ~2m",
		"[2/2] FAIL  native ./slow  boom  10m00s  elapsed 12m00s  ETA done",
		"summary: 2 targets, slowest first",
	} {
		if !strings.Contains(parseText, parseWant) {
			parseT.Fatalf("missing %q in:\n%s", parseWant, parseText)
		}
	}
	parseSummary := parseText[strings.Index(parseText, "summary:"):]
	if strings.Index(parseSummary, "native ./slow") > strings.Index(parseSummary, "native ./router") {
		parseT.Fatalf("summary must list the slowest target first: %s", parseText)
	}
}

func TestProgressFirstErrorLine(parseT *testing.T) {
	parseText := "go test failed: # pkg\n./x.go:3:1: undefined: foo\nFAIL pkg [build failed]"
	if parseGot := getBenchSuiteFirstError(parseText); parseGot != "./x.go:3:1: undefined: foo" {
		parseT.Fatalf("got %q", parseGot)
	}
	if parseGot := getBenchSuiteFirstError(""); parseGot != "unknown error" {
		parseT.Fatalf("got %q", parseGot)
	}
}

func TestProgressHeartbeatFakeClock(parseT *testing.T) {
	synctest.Test(parseT, func(parseT *testing.T) {
		var parseLog bytes.Buffer
		parseProgress := buildBenchSuiteProgress(nil, &parseLog, 1, nil)
		parseStop := parseProgress.startHeartbeat("native ./internal/runtime", benchSuiteHeartbeatInterval)
		time.Sleep(59 * time.Second)
		if parseLog.Len() != 0 {
			parseT.Fatalf("no heartbeat before 60s, got %q", parseLog.String())
		}
		time.Sleep(2*time.Second + 2*time.Minute)
		parseStop()
		parseText := parseLog.String()
		for _, parseWant := range []string{"... still running native ./internal/runtime (1m00s)", "(2m00s)", "(3m00s)"} {
			if !strings.Contains(parseText, parseWant) {
				parseT.Fatalf("missing %q in:\n%s", parseWant, parseText)
			}
		}
		if strings.Count(parseText, "still running") != 3 {
			parseT.Fatalf("want 3 heartbeats in 181s:\n%s", parseText)
		}
	})
}

func makeBenchSuitePartialFixture(parseConfig benchSuiteConfig) benchSuitePartial {
	return benchSuitePartial{
		ConfigHash: hashBenchSuiteConfig(parseConfig),
		Seed:       parseConfig.seed,
		Targets: []benchSuiteTarget{
			{Lane: "native", Package: "./a", Status: benchSuiteStatusOK, DurationMs: 1500, Entries: []benchSuiteEntry{{ID: "native/./a/BenchmarkA", Unit: "ns/op", Samples: []float64{5}, Median: 5, N: 1}}},
			{Lane: "native", Package: "./b", Status: benchSuiteStatusTimeout, Error: "timed out after 1s"},
		},
	}
}

func TestBenchSuitePartialAtomicWriteAndResumeMismatch(parseT *testing.T) {
	parseConfig := benchSuiteConfig{suiteName: "s", lanes: []string{"native"}, count: 5, warmup: 1, seed: 7, bench: "."}
	parsePath := filepath.Join(parseT.TempDir(), "out", "suite.json.partial.json")
	parsePartial := makeBenchSuitePartialFixture(parseConfig)
	for range 3 {
		if parseErr := storeBenchSuitePartial(parsePath, parsePartial); parseErr != nil {
			parseT.Fatal(parseErr)
		}
	}
	if _, parseErr := os.Stat(parsePath + ".tmp"); !os.IsNotExist(parseErr) {
		parseT.Fatalf("temp file must not remain: %v", parseErr)
	}
	parseLoaded, parseErr2 := loadBenchSuitePartial(parsePath)
	if parseErr2 != nil {
		parseT.Fatal(parseErr2)
	}
	if len(parseLoaded.Targets) != 2 || parseLoaded.Targets[0].Entries[0].Median != 5 || parseLoaded.SchemaVersion != benchSuitePartialSchemaVersion {
		parseT.Fatalf("round trip lost data: %+v", parseLoaded)
	}
	if parseErr3 := checkBenchSuiteResume(parseLoaded, parseConfig, parsePath); parseErr3 != nil {
		parseT.Fatalf("same config must resume: %v", parseErr3)
	}
	parseChanged := parseConfig
	parseChanged.count = 6
	parseErr4 := checkBenchSuiteResume(parseLoaded, parseChanged, parsePath)
	if parseErr4 == nil || !strings.Contains(parseErr4.Error(), "different configuration") || !strings.Contains(parseErr4.Error(), "-seed 7") {
		parseT.Fatalf("want clear mismatch refusal, got %v", parseErr4)
	}
	parseTimeoutOnly := parseConfig
	parseTimeoutOnly.targetTimeout = time.Hour
	if parseErr5 := checkBenchSuiteResume(parseLoaded, parseTimeoutOnly, parsePath); parseErr5 != nil {
		parseT.Fatalf("timeout is not part of the config hash: %v", parseErr5)
	}
	_ = os.WriteFile(parsePath, []byte(`{"schemaVersion":"other"}`), 0o644)
	if _, parseErr6 := loadBenchSuitePartial(parsePath); parseErr6 == nil {
		parseT.Fatal("wrong schema must be rejected")
	}
}

// makeBenchSuiteTargetsFixture writes two bench packages and returns the resolved config.
func makeBenchSuiteTargetsFixture(parseT *testing.T) benchSuiteConfig {
	parseT.Helper()
	parseRoot := parseT.TempDir()
	for _, parseName := range []string{"a", "b"} {
		_ = os.MkdirAll(filepath.Join(parseRoot, parseName), 0o755)
		_ = os.WriteFile(filepath.Join(parseRoot, parseName, "x_test.go"), []byte("package "+parseName+"\nfunc BenchmarkX() {}\n"), 0o644)
	}
	parseConfig, parseErr := resolveBenchSuiteConfig(benchSuiteConfig{rootPath: parseRoot, count: 2, warmup: 1, lanes: []string{"native"}, seed: 9, targetTimeout: time.Minute})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	return parseConfig
}

func TestBenchSuiteTimeoutClassificationContinues(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	var parseConsole bytes.Buffer
	parseConfig.progressWriter = &parseConsole
	parseOriginal := benchSuiteRunPackage
	parseT.Cleanup(func() { benchSuiteRunPackage = parseOriginal })
	var parseCalls []string
	benchSuiteRunPackage = func(parseL launcher, parseRun benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		parseCalls = append(parseCalls, parsePackagePath)
		if parseRun.targetTimeout != time.Minute {
			parseT.Errorf("target timeout not forwarded: %v", parseRun.targetTimeout)
		}
		if parsePackagePath == "./a" {
			return benchmarkPackageReport{Lane: parseLane, Package: parsePackagePath, Error: benchSuiteTimeoutMarker + "1m0s"}
		}
		return benchmarkPackageReport{Lane: parseLane, Package: parsePackagePath, OK: true, Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\n")}
	}
	parseReport, parseErr := launcher{}.executeBenchSuite(parseConfig)
	if parseErr == nil || !strings.Contains(parseErr.Error(), "timed out after") {
		parseT.Fatalf("a timeout must fail the run, got %v", parseErr)
	}
	// The timed-out warm-up aborts ./a before its measured run, then ./b still runs.
	if strings.Join(parseCalls, ",") != "./a,./b,./b" {
		parseT.Fatalf("calls %v", parseCalls)
	}
	parseLane := parseReport.Lanes[0]
	if parseLane.Status != benchSuiteStatusTimeout || len(parseLane.Entries) != 1 {
		parseT.Fatalf("lane %+v", parseLane)
	}
	if len(parseReport.Targets) != 2 || parseReport.Targets[0].Status != benchSuiteStatusTimeout || parseReport.Targets[1].Status != benchSuiteStatusOK || parseReport.Targets[1].Entries != nil {
		parseT.Fatalf("targets %+v", parseReport.Targets)
	}
	for _, parseWant := range []string{"TIMEOUT native ./a", "OK    native ./b", "summary:"} {
		if !strings.Contains(parseConsole.String(), parseWant) {
			parseT.Fatalf("missing %q in:\n%s", parseWant, parseConsole.String())
		}
	}
	if _, parseStatErr := os.Stat(getBenchSuiteLogPath(parseConfig.outPath)); parseStatErr != nil {
		parseT.Fatalf("log file missing: %v", parseStatErr)
	}
	parsePartial, parseLoadErr := loadBenchSuitePartial(getBenchSuitePartialPath(parseConfig.outPath))
	if parseLoadErr != nil || len(parsePartial.Targets) != 2 {
		parseT.Fatalf("partial not checkpointed: %v %+v", parseLoadErr, parsePartial)
	}
}

func TestBenchSuiteResumeSkipsCompletedTargets(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	parseConfig.progressWriter = &bytes.Buffer{}
	parsePartial := makeBenchSuitePartialFixture(parseConfig)
	parseConfig.resumeFrom = &parsePartial
	parseOriginal := benchSuiteRunPackage
	parseT.Cleanup(func() { benchSuiteRunPackage = parseOriginal })
	var parseCalls []string
	benchSuiteRunPackage = func(parseL launcher, parseRun benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		parseCalls = append(parseCalls, parsePackagePath)
		return benchmarkPackageReport{Lane: parseLane, Package: parsePackagePath, OK: true, Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\n")}
	}
	parseReport, parseErr := launcher{}.executeBenchSuite(parseConfig)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	// ./a was restored from the partial file; the timed-out ./b is retried (1 warm-up + 1 run).
	if strings.Join(parseCalls, ",") != "./b,./b" {
		parseT.Fatalf("calls %v", parseCalls)
	}
	parseIDs := []string{}
	for _, parseEntry := range parseReport.Lanes[0].Entries {
		parseIDs = append(parseIDs, parseEntry.ID)
	}
	if strings.Join(parseIDs, ",") != "native/./a/BenchmarkA,native/./b/BenchmarkX" || parseReport.Lanes[0].Status != benchSuiteStatusOK {
		parseT.Fatalf("lane %+v ids %v", parseReport.Lanes[0], parseIDs)
	}
	parseLogBytes, _ := os.ReadFile(getBenchSuiteLogPath(parseConfig.outPath))
	if !strings.Contains(string(parseLogBytes), "SKIP  native ./a") {
		parseT.Fatalf("log must record the skip:\n%s", parseLogBytes)
	}
}

func TestBenchSuiteQuietStillWritesLog(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	var parseConsole bytes.Buffer
	parseConfig.progressWriter = &parseConsole
	parseConfig.isQuiet = true
	parseOriginal := benchSuiteRunPackage
	parseT.Cleanup(func() { benchSuiteRunPackage = parseOriginal })
	benchSuiteRunPackage = func(parseL launcher, parseRun benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		return benchmarkPackageReport{Lane: parseLane, Package: parsePackagePath, OK: true}
	}
	if _, parseErr := (launcher{}).executeBenchSuite(parseConfig); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseLogBytes, _ := os.ReadFile(getBenchSuiteLogPath(parseConfig.outPath))
	if parseConsole.Len() != 0 || !strings.Contains(string(parseLogBytes), "START native ./a") {
		parseT.Fatalf("quiet console=%q log=%q", parseConsole.String(), parseLogBytes)
	}
}

func TestBenchSuiteReportJSONKeepsTargetsWithoutEntries(parseT *testing.T) {
	parseReport := makeBenchSuiteFixture(100, 1, "amd64", 400)
	parseReport.Targets = []benchSuiteTarget{{Lane: "native", Package: "./a", Status: benchSuiteStatusTimeout, Error: "timed out after 1s", DurationMs: 1000}}
	parsePayload, parseErr := json.Marshal(parseReport)
	if parseErr != nil || !strings.Contains(string(parsePayload), `"status":"timeout"`) || strings.Contains(string(parsePayload), `"entries":null`) {
		parseT.Fatalf("payload %s err %v", parsePayload, parseErr)
	}
}

func TestBenchSuiteHelperProcess(parseT *testing.T) {
	if os.Getenv("GWC_BENCH_HELPER") != "sleep" {
		return
	}
	time.Sleep(time.Minute)
}

func TestBenchSuiteTimedRunnerKillsOverrunningCommand(parseT *testing.T) {
	parseStart := time.Now()
	parseOutput, parseErr := benchmarkRunCommandTimeout(os.Args[0], []string{"-test.run=^TestBenchSuiteHelperProcess$"}, "", append(os.Environ(), "GWC_BENCH_HELPER=sleep"), 400*time.Millisecond)
	if parseErr == nil || !isBenchSuiteTimeoutError(parseErr.Error()) {
		parseT.Fatalf("want timeout error, output %q err %v", parseOutput, parseErr)
	}
	if time.Since(parseStart) > 20*time.Second {
		parseT.Fatalf("runner did not return promptly: %v", time.Since(parseStart))
	}
	if isBenchSuiteTimeoutError(errors.New("go test failed: boom").Error()) {
		parseT.Fatal("ordinary failures must not classify as timeouts")
	}
}
