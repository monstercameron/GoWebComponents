package main

import (
	"bytes"
	"errors"
	"fmt"
	"os"
	"path/filepath"
	"slices"
	"strings"
	"testing"
	"time"
)

// stubBenchSuiteRunPackage replaces benchSuiteRunPackage for one test.
func stubBenchSuiteRunPackage(parseT *testing.T, parseRun func(parseConfig benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport) {
	parseT.Helper()
	parseOriginal := benchSuiteRunPackage
	parseT.Cleanup(func() { benchSuiteRunPackage = parseOriginal })
	benchSuiteRunPackage = func(parseL launcher, parseConfig benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		return parseRun(parseConfig, parseLane, parsePackagePath)
	}
}

func TestBenchSuiteOutPathResolvesAgainstWorkingDirectory(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseConfig, parseErr := resolveBenchSuiteConfig(benchSuiteConfig{rootPath: parseRoot, outPath: filepath.Join("bin", "x.json"), count: 1})
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseWant, _ := filepath.Abs(filepath.Join("bin", "x.json"))
	if parseConfig.outPath != parseWant {
		parseT.Fatalf("relative -out must follow the working directory: got %s want %s", parseConfig.outPath, parseWant)
	}
	if strings.HasPrefix(parseConfig.outPath, parseRoot) {
		parseT.Fatalf("report path leaked into -root: %s", parseConfig.outPath)
	}
	if _, parseErr2 := resolveBenchSuiteConfig(benchSuiteConfig{rootPath: parseRoot, count: 1, goarch: "arm 64"}); parseErr2 == nil {
		parseT.Fatal("a malformed -goarch must be rejected")
	}
}

func TestBenchSuiteStartupLinesPrintAbsolutePaths(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	var parseConsole bytes.Buffer
	parseConfig.progressWriter = &parseConsole
	stubBenchSuiteRunPackage(parseT, func(benchmarkConfig, string, string) benchmarkPackageReport {
		return benchmarkPackageReport{OK: true}
	})
	if _, parseErr := (launcher{}).executeBenchSuite(parseConfig); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	for _, parseWant := range []string{"report: " + parseConfig.outPath, "log:     " + parseConfig.outPath + ".log", "partial: " + parseConfig.outPath + ".partial.json"} {
		if !strings.Contains(parseConsole.String(), parseWant) {
			parseT.Fatalf("missing %q in:\n%s", parseWant, parseConsole.String())
		}
	}
}

func TestBuildBenchmarkNativeEnvPassesRequestedGoarch(parseT *testing.T) {
	parseT.Setenv("GOARCH", "386")
	parseCount := func(parseEnv []string) (int, string) {
		parseSeen, parseValue := 0, ""
		for _, parseEntry := range parseEnv {
			if parseText, isFound := strings.CutPrefix(parseEntry, "GOARCH="); isFound {
				parseSeen++
				parseValue = parseText
			}
		}
		return parseSeen, parseValue
	}
	if parseSeen, _ := parseCount(buildBenchmarkNativeEnv("")); parseSeen != 0 {
		parseT.Fatal("an empty -goarch must leave the child default alone")
	}
	if parseSeen, parseValue := parseCount(buildBenchmarkNativeEnv(" arm64 ")); parseSeen != 1 || parseValue != "arm64" {
		parseT.Fatalf("want exactly GOARCH=arm64, got %d %q", parseSeen, parseValue)
	}
}

func TestBenchmarkPackageRunPassesGoarchAndReadsHeader(parseT *testing.T) {
	parseOriginal := benchmarkRunCommandTimeout
	parseT.Cleanup(func() { benchmarkRunCommandTimeout = parseOriginal })
	var parseEnvSeen []string
	benchmarkRunCommandTimeout = func(parseCommand string, parseArgs []string, parseCwd string, parseEnv []string, parseTimeout time.Duration) (string, error) {
		parseEnvSeen = parseEnv
		return "goos: windows\ngoarch: arm64\nBenchmarkX-8 10 100 ns/op 8 B/op 1 allocs/op\nPASS", nil
	}
	parseReport := launcher{}.runBenchmarkPackage(benchmarkConfig{rootPath: parseT.TempDir(), bench: ".", count: 1, goarch: "arm64", targetTimeout: time.Minute}, "native", ".")
	if !parseReport.OK || parseReport.GoArch != "arm64" || parseReport.GoOS != "windows" {
		parseT.Fatalf("header not captured: %+v", parseReport)
	}
	if !slices.Contains(parseEnvSeen, "GOARCH=arm64") {
		parseT.Fatalf("child env lacks GOARCH=arm64: %v", parseEnvSeen)
	}
}

func TestBenchSuiteGoarchMismatchFailsTargetAndRecordsObservedArch(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	parseConfig.progressWriter = &bytes.Buffer{}
	parseConfig.goarch = "arm64"
	stubBenchSuiteRunPackage(parseT, func(benchmarkConfig, string, string) benchmarkPackageReport {
		return benchmarkPackageReport{OK: true, GoOS: "windows", GoArch: "amd64", Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\n")}
	})
	parseReport, parseErr := launcher{}.executeBenchSuite(parseConfig)
	if parseErr == nil || !strings.Contains(parseErr.Error(), "requested -goarch arm64") {
		parseT.Fatalf("want a clear goarch mismatch error, got %v", parseErr)
	}
	if parseReport.Lanes[0].Status != benchSuiteStatusFailed || len(parseReport.Lanes[0].Entries) != 0 {
		parseT.Fatalf("mismatched lane must fail without entries: %+v", parseReport.Lanes[0])
	}
	if parseReport.Machine.Arch != "amd64" || parseReport.Machine.ArchSource != "benchmark-header" || parseReport.Machine.OS != "windows" {
		parseT.Fatalf("machine must state what actually ran: %+v", parseReport.Machine)
	}
}

func TestBenchSuiteMachineArchComesFromBenchmarkHeader(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	parseConfig.progressWriter = &bytes.Buffer{}
	parseConfig.goarch = "arm64"
	stubBenchSuiteRunPackage(parseT, func(benchmarkConfig, string, string) benchmarkPackageReport {
		return benchmarkPackageReport{OK: true, GoOS: "windows", GoArch: "arm64", Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\n")}
	})
	parseReport, parseErr := launcher{}.executeBenchSuite(parseConfig)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if parseReport.Machine.Arch != "arm64" || parseReport.Machine.ArchSource != "benchmark-header" || len(parseReport.Lanes[0].Entries) != 2 {
		parseT.Fatalf("unexpected machine/lane: %+v / %+v", parseReport.Machine, parseReport.Lanes[0])
	}
}

func TestBuildBenchSuiteMachineUsesRootToolchainAndSources(parseT *testing.T) {
	parseOriginal := benchmarkQueryCommandEnv
	parseOriginalPlain := benchmarkQueryCommand
	parseT.Cleanup(func() { benchmarkQueryCommandEnv = parseOriginal; benchmarkQueryCommand = parseOriginalPlain })
	parseRoot := parseT.TempDir()
	_ = os.WriteFile(filepath.Join(parseRoot, "go.mod"), []byte("module m\n\ngo 1.26.0\n\ntoolchain go1.26.3\n"), 0o644)
	_ = os.WriteFile(filepath.Join(parseRoot, ".gwc-source-sha"), []byte("deadbeef1234\n"), 0o644)
	benchmarkQueryCommand = func(string, string, ...string) string { return "PARENT-ENV-MUST-NOT-BE-USED" }
	benchmarkQueryCommandEnv = func(parseCwd string, parseEnv []string, parseName string, parseArgs ...string) string {
		if parseCwd != parseRoot || !slices.Contains(parseEnv, "GOARCH=arm64") {
			parseT.Errorf("query must run in the root under the child env: cwd=%s", parseCwd)
		}
		switch strings.Join(append([]string{parseName}, parseArgs...), " ") {
		case "go version":
			return "go version go1.26.3 windows/arm64"
		case "go env GOARCH":
			return "arm64"
		case "go env GOHOSTARCH":
			return "arm64"
		}
		return ""
	}
	parseMachine := buildBenchSuiteMachine(parseRoot, buildBenchmarkNativeEnv("arm64"))
	if parseMachine.GoVersion != "go1.26.3" || parseMachine.GoEnvGoarch != "arm64" || parseMachine.HostArch != "arm64" {
		parseT.Fatalf("toolchain fields: %+v", parseMachine)
	}
	if parseMachine.RootGoMod != "go 1.26.0; toolchain go1.26.3" || parseMachine.RootGitSha != "deadbeef1234" {
		parseT.Fatalf("root fields: %q %q", parseMachine.RootGoMod, parseMachine.RootGitSha)
	}
	applyBenchmarkObservedPlatform(parseMachine, "windows", "amd64")
	if parseMachine.Arch != "amd64" || parseMachine.GoEnvGoarch != "arm64" || parseMachine.ArchSource != "benchmark-header" {
		parseT.Fatalf("observed platform must override arch but keep goEnvGoarch: %+v", parseMachine)
	}
}

func TestGetBenchmarkRootGitShaIgnoresEnclosingRepository(parseT *testing.T) {
	parseOriginal := benchmarkQueryCommand
	parseT.Cleanup(func() { benchmarkQueryCommand = parseOriginal })
	benchmarkQueryCommand = func(parseCwd string, parseName string, parseArgs ...string) string { return "parentsha" }
	parseRoot := parseT.TempDir()
	if parseSha := getBenchmarkRootGitSha(parseRoot); parseSha != "" {
		parseT.Fatalf("an export without .git must not borrow the parent commit, got %q", parseSha)
	}
	_ = os.MkdirAll(filepath.Join(parseRoot, ".git"), 0o755)
	if parseSha := getBenchmarkRootGitSha(parseRoot); parseSha != "parentsha" {
		parseT.Fatalf("a root with .git asks git, got %q", parseSha)
	}
}

func TestClipBenchSuiteOutputKeepsHeadAndTail(parseT *testing.T) {
	parseLines := make([]string, 1000)
	for parseIndex := range parseLines {
		parseLines[parseIndex] = fmt.Sprintf("line %04d", parseIndex)
	}
	parseClipped := clipBenchSuiteOutput(strings.Join(parseLines, "\r\n"))
	parseGot := strings.Split(parseClipped, "\n")
	if len(parseGot) != 61 || parseGot[0] != "line 0000" || parseGot[19] != "line 0019" || parseGot[20] != "... [940 lines omitted] ..." || parseGot[21] != "line 0960" || parseGot[60] != "line 0999" {
		parseT.Fatalf("clip shape wrong (%d lines): %q", len(parseGot), parseClipped)
	}
	parseLong := clipBenchSuiteOutput(strings.Repeat("é", 50000))
	if len(parseLong) > benchSuiteClipMaxBytes+100 || !strings.Contains(parseLong, "bytes omitted") {
		parseT.Fatalf("long line not capped: %d bytes", len(parseLong))
	}
	if strings.ContainsRune(parseLong, '�') {
		parseT.Fatal("clip split a UTF-8 sequence")
	}
	if parseShort := clipBenchSuiteOutput("a\nb"); parseShort != "a\nb" {
		parseT.Fatalf("short text must pass through: %q", parseShort)
	}
}

func TestBenchSuiteFailedTargetWritesFullOutputToFile(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	var parseConsole bytes.Buffer
	parseConfig.progressWriter = &parseConsole
	parseFull := make([]string, 5000)
	for parseIndex := range parseFull {
		parseFull[parseIndex] = fmt.Sprintf("noise %05d", parseIndex)
	}
	parseText := strings.Join(parseFull, "\n")
	stubBenchSuiteRunPackage(parseT, func(parseRun benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		if parsePackagePath == "./a" {
			return benchmarkPackageReport{Error: "go test failed: " + parseText, Output: parseText}
		}
		return benchmarkPackageReport{OK: true, Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\n")}
	})
	parseReport, parseErr := launcher{}.executeBenchSuite(parseConfig)
	if parseErr == nil {
		parseT.Fatal("a failed target must fail the run")
	}
	if len(parseReport.Lanes[0].Reason) > benchSuiteClipMaxBytes+300 || len(parseErr.Error()) > benchSuiteClipMaxBytes+600 {
		parseT.Fatalf("report/error still carry the full output: reason %d error %d", len(parseReport.Lanes[0].Reason), len(parseErr.Error()))
	}
	parseFile := filepath.Join(parseConfig.outPath+".failed", "native-a.txt")
	parseSaved, parseReadErr := os.ReadFile(parseFile)
	if parseReadErr != nil || !strings.Contains(string(parseSaved), "noise 02500") {
		parseT.Fatalf("full output missing from %s: %v", parseFile, parseReadErr)
	}
	if !strings.Contains(parseConsole.String(), parseFile) || strings.Contains(parseConsole.String(), "noise 02500") {
		parseT.Fatalf("console must name the file and not embed the output:\n%.600s", parseConsole.String())
	}
	if !strings.Contains(parseReport.Lanes[0].Reason, "(full output: "+parseFile+")") {
		parseT.Fatalf("reason lacks the file pointer: %.300s", parseReport.Lanes[0].Reason)
	}
}

func TestBenchSuiteBenignWarningsAreClipped(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	parseConfig.progressWriter = &bytes.Buffer{}
	parseWarnings := make([]string, 200)
	for parseIndex := range parseWarnings {
		parseWarnings[parseIndex] = fmt.Sprintf("remove x%03d.exe: Access is denied.", parseIndex)
	}
	stubBenchSuiteRunPackage(parseT, func(benchmarkConfig, string, string) benchmarkPackageReport {
		return benchmarkPackageReport{OK: true, Warnings: parseWarnings}
	})
	parseReport, parseErr := launcher{}.executeBenchSuite(parseConfig)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	if len(parseReport.Lanes[0].Warnings) != 2*61 {
		parseT.Fatalf("each target keeps 61 lines (20 + marker + 40), lane has %d", len(parseReport.Lanes[0].Warnings))
	}
	if _, parseStatErr := os.Stat(filepath.Join(parseConfig.outPath+".failed", "native-a-warnings.txt")); parseStatErr != nil {
		parseT.Fatalf("full warnings file missing: %v", parseStatErr)
	}
}

func TestBenchSuiteTimeoutKeepsPartialEntries(parseT *testing.T) {
	parseConfig := makeBenchSuiteTargetsFixture(parseT)
	parseConfig.progressWriter = &bytes.Buffer{}
	parseConfig.warmup = 0
	stubBenchSuiteRunPackage(parseT, func(parseRun benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
		if parsePackagePath == "./a" {
			return benchmarkPackageReport{Error: benchSuiteTimeoutMarker + "1m0s", Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\nBenchmarkX-8 10 110 ns/op\n")}
		}
		return benchmarkPackageReport{OK: true, Benchmarks: parseBenchmarkOutput("BenchmarkX-8 10 100 ns/op\n")}
	})
	parseReport, parseErr := launcher{}.executeBenchSuite(parseConfig)
	if parseErr == nil {
		parseT.Fatal("a timeout still fails the run")
	}
	parseLane := parseReport.Lanes[0]
	if parseLane.Status != benchSuiteStatusTimeout || len(parseLane.Entries) != 2 {
		parseT.Fatalf("lane %+v", parseLane)
	}
	if !parseLane.Entries[0].Partial || parseLane.Entries[0].N != 2 || parseLane.Entries[1].Partial {
		parseT.Fatalf("partial flag wrong: %+v", parseLane.Entries)
	}
}

func TestBenchmarkPackageTimeoutParsesCompleteLinesOnly(parseT *testing.T) {
	parseOriginal := benchmarkRunCommandTimeout
	parseT.Cleanup(func() { benchmarkRunCommandTimeout = parseOriginal })
	benchmarkRunCommandTimeout = func(string, []string, string, []string, time.Duration) (string, error) {
		return "goos: windows\ngoarch: arm64\nBenchmarkA-8 10 100 ns/op 8 B/op 1 allocs/op\nBenchmarkB-8 10 55", errors.New(benchSuiteTimeoutMarker + "1m0s")
	}
	parseReport := launcher{}.runBenchmarkPackage(benchmarkConfig{rootPath: parseT.TempDir(), bench: ".", count: 1, targetTimeout: time.Minute}, "native", ".")
	if parseReport.OK || len(parseReport.Benchmarks) != 1 || parseReport.Benchmarks[0].Name != "BenchmarkA-8" || parseReport.GoArch != "arm64" {
		parseT.Fatalf("want one complete benchmark kept from the killed run: %+v", parseReport)
	}
}

func TestBenchSuiteTargetTimeoutOverrides(parseT *testing.T) {
	parseConfig := benchmarkConfig{targetTimeout: 15 * time.Minute}
	if parseGot := getBenchmarkTargetTimeout(parseConfig, "./internal/runtime"); parseGot != 75*time.Minute {
		parseT.Fatalf("built-in runtime default: %v", parseGot)
	}
	if parseGot := getBenchmarkTargetTimeout(parseConfig, "./sanitize"); parseGot != 15*time.Minute {
		parseT.Fatalf("global default: %v", parseGot)
	}
	parseMap, parseErr := parseBenchSuiteTimeoutList("./internal/runtime=90m, sanitize=2m")
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseConfig.targetTimeoutFor = parseMap
	if parseGot := getBenchmarkTargetTimeout(parseConfig, "./internal/runtime"); parseGot != 90*time.Minute {
		parseT.Fatalf("explicit override must win: %v", parseGot)
	}
	if parseGot := getBenchmarkTargetTimeout(parseConfig, "./sanitize"); parseGot != 2*time.Minute {
		parseT.Fatalf("keys are normalised: %v", parseGot)
	}
	if parseGot := getBenchmarkTargetTimeout(benchmarkConfig{}, "./internal/runtime"); parseGot != 0 {
		parseT.Fatalf("a disabled global timeout disables defaults: %v", parseGot)
	}
	for _, parseBad := range []string{"nopair", "./a=xyz", "=5m"} {
		if _, parseBadErr := parseBenchSuiteTimeoutList(parseBad); parseBadErr == nil {
			parseT.Fatalf("%q must be rejected", parseBad)
		}
	}
}

// makeBenchSuiteMergeFixture builds a report with the given targets; each OK target gets one entry.
func makeBenchSuiteMergeFixture(parseMedian float64, parseTargets ...benchSuiteTarget) benchSuiteReport {
	parseReport := makeBenchSuiteFixture(parseMedian, 1, "arm64", 400)
	parseReport.Lanes[0].Entries = nil
	parseReport.Targets = nil
	for _, parseTarget := range parseTargets {
		parseSummary := parseTarget
		parseSummary.Entries = nil
		parseReport.Targets = append(parseReport.Targets, parseSummary)
		if parseTarget.Status == benchSuiteStatusOK {
			parseReport.Lanes[0].Entries = append(parseReport.Lanes[0].Entries, benchSuiteEntry{
				ID: "native/" + parseTarget.Package + "/BenchmarkX/sub", Unit: "ns/op", Samples: []float64{parseMedian}, Median: parseMedian, N: 1})
		}
	}
	return parseReport
}

func TestMergeBenchSuiteReportsOverridesAndProvenance(parseT *testing.T) {
	parseOne := makeBenchSuiteMergeFixture(100,
		benchSuiteTarget{Lane: "native", Package: "./a", Status: benchSuiteStatusOK},
		benchSuiteTarget{Lane: "native", Package: "./a/b", Status: benchSuiteStatusOK},
		benchSuiteTarget{Lane: "native", Package: "./c", Status: benchSuiteStatusTimeout, Error: "timed out after 1s"})
	parseTwo := makeBenchSuiteMergeFixture(200,
		benchSuiteTarget{Lane: "native", Package: "./a", Status: benchSuiteStatusFailed, Error: "boom"},
		benchSuiteTarget{Lane: "native", Package: "./c", Status: benchSuiteStatusOK},
		benchSuiteTarget{Lane: "native", Package: "./d", Status: benchSuiteStatusOK})
	parseTwo.Size = []benchSuiteSize{{Artifact: "./app", RawBytes: 2000, GzipBytes: 300}}
	parseMerged, parseErr := mergeBenchSuiteReports([]string{"one.json", "two.json"}, []benchSuiteReport{parseOne, parseTwo}, false)
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseMedians := map[string]float64{}
	parseSources := map[string]string{}
	for _, parseEntry := range parseMerged.Lanes[0].Entries {
		parseMedians[parseEntry.ID] = parseEntry.Median
		parseSources[parseEntry.ID] = parseEntry.SourceReport
	}
	parseWant := map[string]float64{"native/./a/BenchmarkX/sub": 100, "native/./a/b/BenchmarkX/sub": 100, "native/./c/BenchmarkX/sub": 200, "native/./d/BenchmarkX/sub": 200}
	for parseID, parseValue := range parseWant {
		if parseMedians[parseID] != parseValue {
			parseT.Fatalf("%s median %v want %v (all: %v)", parseID, parseMedians[parseID], parseValue, parseMedians)
		}
	}
	if len(parseMedians) != 4 || parseSources["native/./a/BenchmarkX/sub"] != "one.json" || parseSources["native/./c/BenchmarkX/sub"] != "two.json" {
		parseT.Fatalf("provenance wrong: %v", parseSources)
	}
	// ./a failed in the later report but keeps the earlier ok data; ./c recovered from a timeout.
	if parseMerged.Lanes[0].Status != benchSuiteStatusOK || len(parseMerged.Targets) != 4 || parseMerged.Size[0].RawBytes != 2000 {
		parseT.Fatalf("merged lane/targets/size: %+v %+v %+v", parseMerged.Lanes[0], parseMerged.Targets, parseMerged.Size)
	}
	if len(parseMerged.Lanes) != 5 || parseMerged.Lanes[4].Lane != "browser" {
		parseT.Fatalf("all lanes must stay present: %+v", parseMerged.Lanes)
	}
}

func TestMergeBenchSuiteReportsRefusesMismatchUnlessForced(parseT *testing.T) {
	parseOne := makeBenchSuiteMergeFixture(100, benchSuiteTarget{Lane: "native", Package: "./a", Status: benchSuiteStatusOK})
	parseTwo := makeBenchSuiteMergeFixture(100, benchSuiteTarget{Lane: "native", Package: "./b", Status: benchSuiteStatusOK})
	parseTwo.Run.Seed = 99
	if _, parseErr := mergeBenchSuiteReports([]string{"a", "b"}, []benchSuiteReport{parseOne, parseTwo}, false); parseErr == nil || !strings.Contains(parseErr.Error(), "seed 7 vs 99") {
		parseT.Fatalf("want seed refusal, got %v", parseErr)
	}
	parseTwo.Run.Seed = 7
	parseTwo.Run.Count = 9
	if _, parseErr := mergeBenchSuiteReports([]string{"a", "b"}, []benchSuiteReport{parseOne, parseTwo}, false); parseErr == nil || !strings.Contains(parseErr.Error(), "count 5 vs 9") {
		parseT.Fatalf("want count refusal, got %v", parseErr)
	}
	if _, parseErr := mergeBenchSuiteReports([]string{"a", "b"}, []benchSuiteReport{parseOne, parseTwo}, true); parseErr != nil {
		parseT.Fatalf("-force must merge: %v", parseErr)
	}
}

func TestRunBenchSuiteMergeCommandWritesReport(parseT *testing.T) {
	parseDir := parseT.TempDir()
	parseA := filepath.Join(parseDir, "a.json")
	parseB := filepath.Join(parseDir, "b.json")
	parseOut := filepath.Join(parseDir, "merged.json")
	_ = writeBenchSuiteReport(parseA, makeBenchSuiteMergeFixture(100, benchSuiteTarget{Lane: "native", Package: "./a", Status: benchSuiteStatusOK}))
	_ = writeBenchSuiteReport(parseB, makeBenchSuiteMergeFixture(150, benchSuiteTarget{Lane: "native", Package: "./b", Status: benchSuiteStatusOK}))
	if parseErr := (launcher{}).runBenchmarkSuite([]string{"merge", "-o", parseOut, parseA, parseB}); parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseMerged, parseLoadErr := loadBenchSuiteReport(parseOut)
	if parseLoadErr != nil || len(parseMerged.Lanes[0].Entries) != 2 || parseMerged.Lanes[0].Entries[1].SourceReport != "b.json" {
		parseT.Fatalf("merged file wrong: %v %+v", parseLoadErr, parseMerged.Lanes[0].Entries)
	}
	if parseErr := (launcher{}).runBenchmarkSuite([]string{"merge", parseA}); parseErr == nil {
		parseT.Fatal("merge without -o and two inputs must print usage as an error")
	}
}
