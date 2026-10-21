package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io"
	"math"
	"math/rand/v2"
	"os"
	"path/filepath"
	"regexp"
	"strings"
	"time"
)

// benchSuiteSchemaVersion is the schema id written into every suite report.
const benchSuiteSchemaVersion = "gwc.bench.suite/v1"

// Lane identifiers of the suite. Browser is reserved for lanes added in a later phase and is
// always written as status "not-run" so reports never silently omit a lane.
const (
	benchSuiteLaneNative  = "native"
	benchSuiteLaneWasm    = "wasm"
	benchSuiteLaneSize    = "size"
	benchSuiteLaneBuild   = "build"
	benchSuiteLaneBrowser = "browser"
)

// Lane statuses recorded in lanes[].status.
const (
	benchSuiteStatusOK     = "ok"
	benchSuiteStatusFailed = "failed"
	benchSuiteStatusNotRun = "not-run"
	// benchSuiteStatusTimeout marks a lane or target whose go test was killed by -target-timeout.
	benchSuiteStatusTimeout = "timeout"
)

// benchSuiteReport is the single versioned JSON document produced by `gwc bench suite`.
type benchSuiteReport struct {
	SchemaVersion string            `json:"schemaVersion"`
	Suite         string            `json:"suite"`
	GeneratedAt   string            `json:"generatedAt"`
	Machine       *benchmarkMachine `json:"machine,omitempty"`
	Run           benchSuiteRun     `json:"run"`
	Lanes         []benchSuiteLane  `json:"lanes"`
	// Targets lists every native or wasm package target with its status and duration, so a
	// timeout or failure is visible per package. Entries live in Lanes, not here.
	Targets []benchSuiteTarget `json:"targets,omitempty"`
	Size    []benchSuiteSize   `json:"size"`
	Build   *benchSuiteBuild   `json:"build,omitempty"`
	// Refs is a placeholder for reference-implementation results (for example React or other
	// framework baselines) that later lanes fill in. It is always serialized, empty for now.
	Refs map[string]any `json:"refs"`
	// Health is a placeholder for run-health signals (thermal throttling, background load, clock
	// drift) that later lanes fill in. It is always serialized, empty for now.
	Health map[string]any `json:"health"`
}

// benchSuiteRun records how the samples were collected so a run can be reproduced.
type benchSuiteRun struct {
	Count     int      `json:"count"`
	Warmup    int      `json:"warmup"`
	Benchtime string   `json:"benchtime,omitempty"`
	Seed      int64    `json:"seed"`
	Order     []string `json:"order"`
	Parallel  int      `json:"parallel"`
}

// benchSuiteLane groups the entries of one lane; Layer is the unit of per-layer geomeans.
type benchSuiteLane struct {
	Lane     string            `json:"lane"`
	Layer    string            `json:"layer"`
	Status   string            `json:"status"`
	Reason   string            `json:"reason,omitempty"`
	Warnings []string          `json:"warnings,omitempty"`
	Entries  []benchSuiteEntry `json:"entries"`
}

// benchSuiteEntry is one measured benchmark: raw samples plus median and MAD of the unit.
type benchSuiteEntry struct {
	ID          string    `json:"id"`
	Unit        string    `json:"unit"`
	Samples     []float64 `json:"samples"`
	Median      float64   `json:"median"`
	MAD         float64   `json:"mad"`
	N           int       `json:"n"`
	BytesPerOp  *float64  `json:"bytesPerOp,omitempty"`
	AllocsPerOp *float64  `json:"allocsPerOp,omitempty"`
	// Partial marks an entry parsed from a target killed by -target-timeout: it may hold fewer
	// than -count samples.
	Partial bool `json:"partial,omitempty"`
	// SourceReport names the report file the entry came from; `bench suite merge` fills it.
	SourceReport string `json:"sourceReport,omitempty"`
}

// benchSuiteSize is the byte size of one built artifact in raw, gzip and brotli form.
type benchSuiteSize struct {
	Artifact    string `json:"artifact"`
	RawBytes    int64  `json:"rawBytes"`
	GzipBytes   int64  `json:"gzipBytes"`
	BrotliBytes int64  `json:"brotliBytes"`
}

// benchSuiteBuild holds the first-build and immediate-rebuild wall time of the build lane.
type benchSuiteBuild struct {
	Pattern string `json:"pattern"`
	Target  string `json:"target"`
	ColdMs  int64  `json:"coldMs"`
	WarmMs  int64  `json:"warmMs"`
}

// benchSuiteConfig is the resolved configuration of one suite run.
type benchSuiteConfig struct {
	rootPath     string
	outPath      string
	suiteName    string
	lanes        []string
	count        int
	warmup       int
	hasWarmup    bool
	benchtime    string
	bench        string
	skip         string
	seed         int64
	isShuffle    bool
	include      []string
	json         bool
	sizePackage  string
	buildPattern string
	// isQuiet suppresses console progress lines; the log file is still written.
	isQuiet bool
	// isResume skips targets already completed in the partial file.
	isResume bool
	// targetTimeout kills one package's go test process tree after this long; 0 disables it.
	targetTimeout time.Duration
	// targetTimeoutFor overrides targetTimeout per package (keys like ./internal/runtime).
	targetTimeoutFor map[string]time.Duration
	// goarch is passed to native-lane children as GOARCH; empty keeps the child default.
	goarch string
	// tags is the normalized comma list of build tags passed to every micro-lane go test and to the
	// size and build lanes; empty adds no -tags flag.
	tags string
	// packages restricts the micro lanes to these package paths or globs; empty runs every package.
	packages []string
	// resumeFrom is the validated partial file to resume from, set by runBenchmarkSuite.
	resumeFrom *benchSuitePartial
	// progressWriter overrides the console sink (stderr); tests set it.
	progressWriter io.Writer
}

// benchSuiteJob is one (lane, package) unit of work in the randomizable run order.
type benchSuiteJob struct {
	lane        string
	packagePath string
}

// benchSuiteRunPackage runs one package for one lane; tests replace it.
var benchSuiteRunPackage = func(parseL launcher, parseConfig benchmarkConfig, parseLane string, parsePackagePath string) benchmarkPackageReport {
	return parseL.runBenchmarkPackage(parseConfig, parseLane, parsePackagePath)
}

// benchSuiteMeasureSize builds one wasm artifact and reports its sizes; tests replace it.
var benchSuiteMeasureSize = func(parsePackagePath string, parseOutDir string) (benchSuiteSize, error) {
	parseConfig, parseErr := resolveWasmMeasureConfig(wasmMeasureConfig{
		packagePath:    parsePackagePath,
		outDir:         parseOutDir,
		ldflags:        "-s -w",
		releaseProfile: true,
	})
	if parseErr != nil {
		return benchSuiteSize{}, parseErr
	}
	parseSummary, parseErr2 := executeWasmMeasure(parseConfig)
	if parseErr2 != nil {
		return benchSuiteSize{}, parseErr2
	}
	return benchSuiteSize{
		Artifact:    parsePackagePath,
		RawBytes:    parseSummary.Manifest.Artifacts["wasm"].Bytes,
		GzipBytes:   parseSummary.Manifest.Artifacts["gzip"].Bytes,
		BrotliBytes: parseSummary.Manifest.Artifacts["brotli"].Bytes,
	}, nil
}

// benchSuiteMeasureBuild times a first build and an immediate rebuild; tests replace it.
var benchSuiteMeasureBuild = func(parseRoot string, parsePattern string, parseTarget string) (benchSuiteBuild, error) {
	parseCold, parseErr := warmBuildOnce(parseRoot, parsePattern, parseTarget)
	if parseErr != nil {
		return benchSuiteBuild{}, parseErr
	}
	parseWarm, parseErr2 := warmBuildOnce(parseRoot, parsePattern, parseTarget)
	if parseErr2 != nil {
		return benchSuiteBuild{}, parseErr2
	}
	return benchSuiteBuild{Pattern: parsePattern, Target: parseTarget, ColdMs: parseCold.Milliseconds(), WarmMs: parseWarm.Milliseconds()}, nil
}

// runBenchmarkSuite implements `gwc bench suite` and its `compare` subcommand.
func (parseL launcher) runBenchmarkSuite(parseArgs []string) error {
	if len(parseArgs) > 0 && strings.EqualFold(strings.TrimSpace(parseArgs[0]), "compare") {
		return runBenchSuiteCompare(parseArgs[1:])
	}
	if len(parseArgs) > 0 && strings.EqualFold(strings.TrimSpace(parseArgs[0]), "merge") {
		return runBenchSuiteMerge(parseArgs[1:])
	}
	parseFs := flag.NewFlagSet("bench suite", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseFs.Usage = func() {
		fmt.Fprintln(parseFs.Output(), "Usage: gwc bench suite [flags]\n       gwc bench suite compare [-max-regress-pct N] [-min-baseline-ns N] [-allow-machine-mismatch] [-json] <baseline.json> <candidate.json>\n       gwc bench suite merge [-force] -o <out.json> <a.json> <b.json> ...\n\n"+
			"Runs the selected lanes and writes ONE versioned JSON report (schema "+benchSuiteSchemaVersion+").\n"+
			"Lanes: native, wasm (micro benchmarks), size (release wasm bytes), build (first build vs rebuild).\n"+
			"Browser lanes are not implemented yet and are always written as status \"not-run\".\n"+
			"The size lane builds relative to the current directory; run from the repository root.")
		parseFs.PrintDefaults()
	}
	parseRoot := parseFs.String("root", "", "Root directory to inspect for benchmark packages; defaults to the current working directory")
	parseOut := parseFs.String("out", filepath.Join("bin", "bench-suite.json"), "JSON report path; a relative path is resolved against the directory gwc was started in (NOT -root). The log, .partial.json and .failed/ output sit next to it")
	parseSuite := parseFs.String("suite", "gwc-v6-baseline", "Suite name recorded in the report")
	parseCount := parseFs.Int("count", 5, "Samples per benchmark (go test -count)")
	parseBench := parseFs.String("bench", ".", "Benchmark regex passed to go test -bench")
	parseSkip := parseFs.String("skip", "", "Regex passed to go test -skip for every target (excludes benchmarks that cannot run on one side of a comparison)")
	parseBenchtime := parseFs.String("benchtime", "", "Optional benchtime forwarded to go test (for example 200ms or 1x)")
	parseSeed := parseFs.Int64("seed", 0, "Seed for -shuffle and the recorded run; 0 picks one from the clock and records it")
	parseShuffle := parseFs.Bool("shuffle", false, "Randomize package run order with -seed and record the order in run.order")
	parseWarmup := parseFs.Int("warmup", 1, "Discarded warm-up runs per package; applies to the native lane (wasm only when set explicitly)")
	parseSizePackage := parseFs.String("size-package", "./examples/public/counter", "Package built for the size lane")
	parseBuildPattern := parseFs.String("build-pattern", "./examples/public/counter", "go build pattern timed by the build lane (built for js/wasm)")
	parseJSON := parseFs.Bool("json", false, "Emit the JSON report to stdout after writing it to disk")
	parseQuiet := parseFs.Bool("quiet", false, "Suppress progress lines on stderr; the JSON report and <out>.log are still written")
	parseResume := parseFs.Bool("resume", false, "Skip targets already completed in <out>.partial.json; refuses when the configuration differs")
	parseTimeout := parseFs.Duration("target-timeout", 15*time.Minute, "Kill one package's go test process tree after this long and mark it timeout; 0 disables")
	parseTimeoutFor := parseFs.String("target-timeout-for", "", "Per-package timeouts as pkg=duration,pkg=duration (for example ./internal/runtime=75m); overrides -target-timeout. ./internal/runtime defaults to 75m")
	parseTags := parseFs.String("tags", "", "Comma-separated build tags passed as -tags to every go test of the micro lanes and to the size and build lane go build (for example production, the build wasm users receive); recorded in machine.buildTags")
	parsePackages := parseFs.String("packages", "", "Comma-separated package paths or globs the micro lanes are limited to (for example ./internal/runtime,./ui,./internal/platform/*); an entry matching no benchmark package is an error that lists the valid ones; empty runs every package")
	parseGoarch := parseFs.String("goarch", "", "GOARCH passed explicitly to native-lane go test children (for example arm64); empty keeps the child toolchain default. The architecture the benchmarks actually printed is recorded in machine.arch; a mismatch fails the lane")
	var parseLanes stringListFlag
	parseFs.Var(&parseLanes, "lane", "Lanes to run among native,wasm,size,build; repeatable or comma-separated (default native,wasm,size,build)")
	var parseInclude stringListFlag
	parseFs.Var(&parseInclude, "include", "Root-relative directory to opt in even if skipped by default (for example test/render); repeatable or comma-separated")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	hasWarmup := false
	parseFs.Visit(func(parseCurrent *flag.Flag) {
		if parseCurrent.Name == "warmup" {
			hasWarmup = true
		}
	})
	parseTimeoutMap, parseTimeoutErr := parseBenchSuiteTimeoutList(*parseTimeoutFor)
	if parseTimeoutErr != nil {
		return parseTimeoutErr
	}
	parseConfig, parseErr2 := resolveBenchSuiteConfig(benchSuiteConfig{
		targetTimeoutFor: parseTimeoutMap,
		goarch:           *parseGoarch,
		tags:             *parseTags,
		packages:         splitBenchSuiteList(*parsePackages),
		rootPath:         *parseRoot,
		outPath:          *parseOut,
		suiteName:        *parseSuite,
		lanes:            parseLanes.Values(),
		count:            *parseCount,
		warmup:           *parseWarmup,
		hasWarmup:        hasWarmup,
		benchtime:        *parseBenchtime,
		bench:            *parseBench,
		skip:             *parseSkip,
		seed:             *parseSeed,
		isShuffle:        *parseShuffle,
		include:          parseInclude.Values(),
		json:             *parseJSON,
		sizePackage:      *parseSizePackage,
		buildPattern:     *parseBuildPattern,
		isQuiet:          *parseQuiet,
		isResume:         *parseResume,
		targetTimeout:    *parseTimeout,
	})
	if parseErr2 != nil {
		return parseErr2
	}
	parsePartialPath := getBenchSuitePartialPath(parseConfig.outPath)
	if parseConfig.isResume {
		parsePartial, parseLoadErr := loadBenchSuitePartial(parsePartialPath)
		if parseLoadErr != nil {
			return fmt.Errorf("cannot resume: %w", parseLoadErr)
		}
		// An unset seed is random, so adopt the partial file's seed before comparing configurations.
		if *parseSeed == 0 {
			parseConfig.seed = parsePartial.Seed
		}
		if parseCheckErr := checkBenchSuiteResume(parsePartial, parseConfig, parsePartialPath); parseCheckErr != nil {
			return parseCheckErr
		}
		parseConfig.resumeFrom = &parsePartial
	}
	parseReport, parseRunErr := parseL.executeBenchSuite(parseConfig)
	if parseWriteErr := writeBenchSuiteReport(parseConfig.outPath, parseReport); parseWriteErr != nil {
		return parseWriteErr
	}
	if parseRunErr == nil {
		// The final report supersedes the checkpoint; a failed run keeps it for -resume.
		_ = os.Remove(parsePartialPath)
	}
	if parseConfig.json {
		parseEncoder := json.NewEncoder(os.Stdout)
		parseEncoder.SetIndent("", "  ")
		if parseEncodeErr := parseEncoder.Encode(parseReport); parseEncodeErr != nil {
			return parseEncodeErr
		}
	} else {
		printBenchSuiteReport(parseReport, parseConfig.outPath)
	}
	return parseRunErr
}

// resolveBenchSuiteConfig validates flags and resolves paths and the lane list.
func resolveBenchSuiteConfig(parseConfig benchSuiteConfig) (benchSuiteConfig, error) {
	parseRoot := strings.TrimSpace(parseConfig.rootPath)
	if parseRoot == "" {
		parseCwd, parseErr := os.Getwd()
		if parseErr != nil {
			return benchSuiteConfig{}, fmt.Errorf("resolve bench suite root: %w", parseErr)
		}
		parseRoot = parseCwd
	}
	parseAbsRoot, parseErr2 := filepath.Abs(parseRoot)
	if parseErr2 != nil {
		return benchSuiteConfig{}, fmt.Errorf("resolve bench suite root: %w", parseErr2)
	}
	if parseInfo, parseErr3 := os.Stat(parseAbsRoot); parseErr3 != nil || !parseInfo.IsDir() {
		return benchSuiteConfig{}, fmt.Errorf("bench suite root is not a directory: %s", parseAbsRoot)
	}
	if parseConfig.count <= 0 {
		return benchSuiteConfig{}, errors.New("bench suite count must be at least 1")
	}
	if parseConfig.warmup < 0 {
		return benchSuiteConfig{}, errors.New("bench suite warmup cannot be negative")
	}
	if parseConfig.targetTimeout < 0 {
		return benchSuiteConfig{}, errors.New("bench suite target-timeout cannot be negative")
	}
	parseLanes, parseErr4 := normalizeBenchSuiteLanes(parseConfig.lanes)
	if parseErr4 != nil {
		return benchSuiteConfig{}, parseErr4
	}
	parseGoarch := strings.ToLower(strings.TrimSpace(parseConfig.goarch))
	if parseGoarch != "" && !benchSuiteGoarchPattern.MatchString(parseGoarch) {
		return benchSuiteConfig{}, fmt.Errorf("bench suite goarch %q is not a GOARCH value (for example arm64 or amd64)", parseConfig.goarch)
	}
	parseOut := strings.TrimSpace(parseConfig.outPath)
	if parseOut == "" {
		parseOut = filepath.Join("bin", "bench-suite.json")
	}
	// A relative -out belongs to the directory gwc was started in, not to -root: with -root pointing
	// at an exported tree the report must not land inside that tree.
	parseAbsOut, parseErr5 := filepath.Abs(parseOut)
	if parseErr5 != nil {
		return benchSuiteConfig{}, fmt.Errorf("resolve bench suite out path: %w", parseErr5)
	}
	parseOut = parseAbsOut
	parseBench := strings.TrimSpace(parseConfig.bench)
	if parseBench == "" {
		parseBench = "."
	}
	parseSeed := parseConfig.seed
	if parseSeed == 0 {
		parseSeed = time.Now().UnixNano()
	}
	parseSuite := strings.TrimSpace(parseConfig.suiteName)
	if parseSuite == "" {
		parseSuite = "gwc-v6-baseline"
	}
	parseConfig.rootPath = parseAbsRoot
	parseConfig.outPath = filepath.Clean(parseOut)
	parseConfig.goarch = parseGoarch
	parseTags, parseTagsErr := normalizeBenchSuiteTags(parseConfig.tags)
	if parseTagsErr != nil {
		return benchSuiteConfig{}, parseTagsErr
	}
	parseConfig.tags = parseTags
	parseConfig.packages = normalizeBenchSuitePackageList(parseConfig.packages)
	parseConfig.suiteName = parseSuite
	parseConfig.lanes = parseLanes
	parseConfig.bench = parseBench
	parseConfig.benchtime = strings.TrimSpace(parseConfig.benchtime)
	parseConfig.seed = parseSeed
	parseConfig.include = normalizeBenchmarkIncludes(parseConfig.include)
	return parseConfig, nil
}

// normalizeBenchSuiteLanes validates the lane list; an empty list selects every runnable lane.
func normalizeBenchSuiteLanes(parseRequested []string) ([]string, error) {
	parseAll := []string{benchSuiteLaneNative, benchSuiteLaneWasm, benchSuiteLaneSize, benchSuiteLaneBuild}
	if len(parseRequested) == 0 {
		return parseAll, nil
	}
	parseSeen := map[string]bool{}
	for _, parseLane := range parseRequested {
		parseName := strings.ToLower(strings.TrimSpace(parseLane))
		switch parseName {
		case "all":
			for _, parseKnown := range parseAll {
				parseSeen[parseKnown] = true
			}
		case benchSuiteLaneNative, benchSuiteLaneWasm, benchSuiteLaneSize, benchSuiteLaneBuild:
			parseSeen[parseName] = true
		case benchSuiteLaneBrowser:
			return nil, errors.New("browser lanes are not implemented yet; they are recorded as not-run in the report")
		default:
			return nil, fmt.Errorf("unknown bench suite lane %q (want native, wasm, size, build)", parseLane)
		}
	}
	parseOrdered := []string{}
	for _, parseKnown := range parseAll {
		if parseSeen[parseKnown] {
			parseOrdered = append(parseOrdered, parseKnown)
		}
	}
	return parseOrdered, nil
}

// getBenchSuiteLayer maps a lane to the layer its entries are aggregated under.
func getBenchSuiteLayer(parseLane string) string {
	switch parseLane {
	case benchSuiteLaneNative:
		return "native-micro"
	case benchSuiteLaneWasm:
		return "wasm-micro"
	default:
		return parseLane
	}
}

// shuffleBenchSuiteOrder returns parseIDs permuted deterministically from parseSeed.
func shuffleBenchSuiteOrder(parseIDs []string, parseSeed int64) []string {
	parseOrder := append([]string(nil), parseIDs...)
	parseRng := rand.New(rand.NewPCG(uint64(parseSeed), 0x9e3779b97f4a7c15))
	parseRng.Shuffle(len(parseOrder), func(parseI, parseJ int) {
		parseOrder[parseI], parseOrder[parseJ] = parseOrder[parseJ], parseOrder[parseI]
	})
	return parseOrder
}

// hasBenchSuiteLane reports whether parseLane is in the selected list.
func hasBenchSuiteLane(parseLanes []string, parseLane string) bool {
	for _, parseCandidate := range parseLanes {
		if parseCandidate == parseLane {
			return true
		}
	}
	return false
}

// executeBenchSuite runs every selected lane and assembles the report. The report is always
// returned (even with lane failures) so partial results are still written to disk.
func (parseL launcher) executeBenchSuite(parseConfig benchSuiteConfig) (benchSuiteReport, error) {
	// The machine block describes the child toolchain: go version and go env run in the measured
	// root under the same environment the benchmarks use.
	parseReport := benchSuiteReport{
		SchemaVersion: benchSuiteSchemaVersion,
		Suite:         parseConfig.suiteName,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Machine:       buildBenchSuiteMachine(parseConfig.rootPath, buildBenchmarkNativeEnv(parseConfig.goarch)),
		Run: benchSuiteRun{
			Count:     parseConfig.count,
			Warmup:    parseConfig.warmup,
			Benchtime: parseConfig.benchtime,
			Seed:      parseConfig.seed,
			Order:     []string{},
			Parallel:  1,
		},
		Size:   []benchSuiteSize{},
		Refs:   map[string]any{},
		Health: map[string]any{},
	}
	// The tags are part of what was measured, so they replace whatever GOFLAGS reported.
	if parseConfig.tags != "" && parseReport.Machine != nil {
		parseReport.Machine.BuildTags = parseConfig.tags
	}
	parseMicro := benchmarkConfig{
		rootPath:  parseConfig.rootPath,
		bench:     parseConfig.bench,
		skip:      parseConfig.skip,
		benchtime: parseConfig.benchtime,
		count:     parseConfig.count,
		parallel:  1,
		include:   parseConfig.include,

		targetTimeout:    parseConfig.targetTimeout,
		targetTimeoutFor: parseConfig.targetTimeoutFor,
		goarch:           parseConfig.goarch,
		tags:             parseConfig.tags,
	}
	var parseErrs []string
	isArchObserved := false

	parsePartialPath := getBenchSuitePartialPath(parseConfig.outPath)
	parsePartial := benchSuitePartial{ConfigHash: hashBenchSuiteConfig(parseConfig), Seed: parseConfig.seed}
	parseResumed := map[string]benchSuiteTarget{}
	if parseConfig.resumeFrom != nil {
		for _, parseTarget := range parseConfig.resumeFrom.Targets {
			if parseTarget.Status == benchSuiteStatusOK {
				parseResumed[getBenchSuiteTargetID(parseTarget.Lane, parseTarget.Package)] = parseTarget
			}
		}
		parsePartial.Size = parseConfig.resumeFrom.Size
		parsePartial.Build = parseConfig.resumeFrom.Build
	}
	var parseProgress *benchSuiteProgress
	// storeProgress checkpoints finished targets so an interrupted run keeps its results.
	storeProgress := func() {
		if strings.TrimSpace(parseConfig.outPath) == "" {
			return
		}
		if parseStoreErr := storeBenchSuitePartial(parsePartialPath, parsePartial); parseStoreErr != nil {
			parseProgress.writeLine("warning: " + parseStoreErr.Error())
		}
	}
	var parseConsole io.Writer = os.Stderr
	if parseConfig.progressWriter != nil {
		parseConsole = parseConfig.progressWriter
	}
	if parseConfig.isQuiet {
		parseConsole = nil
	}
	var parseLogWriter io.Writer
	if parseLog, parseLogErr := openBenchSuiteLog(getBenchSuiteLogPath(parseConfig.outPath), parseConfig.resumeFrom != nil); parseLogErr == nil && parseLog != nil {
		defer parseLog.Close()
		parseLogWriter = parseLog
	}
	parseProgress = buildBenchSuiteProgress(parseConsole, parseLogWriter, 0, nil)
	// Absolute paths up front so a relative -out can never leave the reader guessing where files land.
	parseProgress.writeLine(fmt.Sprintf("report: %s", parseConfig.outPath))
	parseProgress.writeLine(fmt.Sprintf("log:     %s", getBenchSuiteLogPath(parseConfig.outPath)))
	parseProgress.writeLine(fmt.Sprintf("partial: %s", parsePartialPath))
	parseProgress.writeLine(fmt.Sprintf("failed:  %s (full output of failed targets)", getBenchSuiteFailedDir(parseConfig.outPath)))
	parseUnits := 0
	if hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneSize) {
		parseUnits++
	}
	if hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneBuild) {
		parseUnits++
	}

	parseLaneByName := map[string]*benchSuiteLane{}
	for _, parseName := range []string{benchSuiteLaneNative, benchSuiteLaneWasm, benchSuiteLaneSize, benchSuiteLaneBuild, benchSuiteLaneBrowser} {
		parseLaneByName[parseName] = &benchSuiteLane{
			Lane:    parseName,
			Layer:   getBenchSuiteLayer(parseName),
			Status:  benchSuiteStatusNotRun,
			Reason:  "lane not selected for this run",
			Entries: []benchSuiteEntry{},
		}
	}
	parseLaneByName[benchSuiteLaneBrowser].Reason = "browser lanes are not implemented yet; added in a later v7 phase"

	hasMicro := hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneNative) || hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneWasm)
	if hasMicro {
		parseNative, parseWasm, parseErr := collectBenchmarkPackagesWithIncludes(parseConfig.rootPath, parseConfig.include)
		if parseErr != nil {
			return parseReport, parseErr
		}
		parseNative, parseWasm, parseErr = filterBenchSuitePackages(parseNative, parseWasm, parseConfig.lanes, parseConfig.packages)
		if parseErr != nil {
			return parseReport, parseErr
		}
		parseJobs := []benchSuiteJob{}
		if hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneNative) {
			for _, parsePackage := range parseNative {
				parseJobs = append(parseJobs, benchSuiteJob{lane: benchSuiteLaneNative, packagePath: parsePackage})
			}
		}
		if hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneWasm) {
			for _, parsePackage := range parseWasm {
				parseJobs = append(parseJobs, benchSuiteJob{lane: benchSuiteLaneWasm, packagePath: parsePackage})
			}
		}
		parseIDs := make([]string, len(parseJobs))
		parseByID := map[string]benchSuiteJob{}
		for parseIndex, parseJob := range parseJobs {
			parseIDs[parseIndex] = parseJob.lane + "/" + parseJob.packagePath
			parseByID[parseIDs[parseIndex]] = parseJob
		}
		if parseConfig.isShuffle {
			parseIDs = shuffleBenchSuiteOrder(parseIDs, parseConfig.seed)
		}
		parseReport.Run.Order = parseIDs
		for _, parseLane := range []string{benchSuiteLaneNative, benchSuiteLaneWasm} {
			if hasBenchSuiteLane(parseConfig.lanes, parseLane) {
				parseLaneByName[parseLane].Status = benchSuiteStatusOK
				parseLaneByName[parseLane].Reason = ""
			}
		}
		parseProgress.setTotal(len(parseJobs) + parseUnits)
		parseProgress.writeLine(fmt.Sprintf("bench suite: %d package targets + %d size/build steps, lanes %s, count %d, warmup %d, seed %d, target-timeout %s",
			len(parseJobs), parseUnits, strings.Join(parseConfig.lanes, ","), parseConfig.count, parseConfig.warmup, parseConfig.seed, parseConfig.targetTimeout))
		if parseConfig.tags != "" {
			parseProgress.writeLine("build tags: " + parseConfig.tags)
		}
		if len(parseConfig.packages) > 0 {
			parseProgress.writeLine("packages: " + strings.Join(parseConfig.packages, ","))
		}
		if parseConfig.resumeFrom != nil {
			parseProgress.writeLine(fmt.Sprintf("resuming: %d completed targets restored from %s", len(parseResumed), parsePartialPath))
		}
		for _, parseID := range parseIDs {
			parseJob := parseByID[parseID]
			parseLaneState := parseLaneByName[parseJob.lane]
			parseLabel := parseJob.lane + " " + parseJob.packagePath
			parseTarget, isResumed := parseResumed[parseID]
			if isResumed {
				parseProgress.logSkip(parseLabel, len(parseTarget.Entries), time.Duration(parseTarget.DurationMs)*time.Millisecond)
			} else {
				parseTarget = runBenchSuiteTarget(parseL, parseMicro, parseConfig, parseProgress, parseJob)
			}
			// The first native target that printed a header decides the architecture the report claims.
			if parseJob.lane == benchSuiteLaneNative && !isArchObserved && parseTarget.GoArch != "" {
				applyBenchmarkObservedPlatform(parseReport.Machine, parseTarget.GoOS, parseTarget.GoArch)
				isArchObserved = true
			}
			applyBenchSuiteTarget(parseLaneState, parseTarget)
			if parseTarget.Status != benchSuiteStatusOK {
				parseMessage := fmt.Sprintf("%s: %s", parseJob.packagePath, strings.TrimSpace(parseTarget.Error))
				parseErrs = append(parseErrs, parseJob.lane+" "+parseMessage)
			}
			parsePartial.Targets = append(parsePartial.Targets, parseTarget)
			if !isResumed {
				storeProgress()
			}
		}
	}
	if !hasMicro {
		parseProgress.setTotal(parseUnits)
	}
	for _, parseTarget := range parsePartial.Targets {
		parseSummaryTarget := parseTarget
		parseSummaryTarget.Entries = nil
		parseReport.Targets = append(parseReport.Targets, parseSummaryTarget)
	}

	if hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneSize) {
		parseLaneState := parseLaneByName[benchSuiteLaneSize]
		parseLabel := benchSuiteLaneSize + " " + parseConfig.sizePackage
		if parseConfig.resumeFrom != nil && parseConfig.resumeFrom.Size != nil && parseConfig.resumeFrom.Size.Artifact == parseConfig.sizePackage {
			parseProgress.logSkip(parseLabel, 1, 0)
			parseLaneState.Status = benchSuiteStatusOK
			parseLaneState.Reason = ""
			parseReport.Size = append(parseReport.Size, *parseConfig.resumeFrom.Size)
		} else {
			parseOutDir := filepath.Join(parseConfig.rootPath, "bin", "bench-suite-size")
			parseIndex := parseProgress.beginUnit(parseLabel, "release wasm build")
			parseStop := parseProgress.startHeartbeat(parseLabel, benchSuiteHeartbeatInterval)
			parseStart := time.Now()
			parseSize, parseErr := runBenchSuiteWithBuildTags(parseConfig.tags, func() (benchSuiteSize, error) {
				return benchSuiteMeasureSize(parseConfig.sizePackage, parseOutDir)
			})
			parseStop()
			if parseErr != nil {
				parseLaneState.Status = benchSuiteStatusFailed
				parseLaneState.Reason = parseErr.Error()
				parseErrs = append(parseErrs, "size "+parseErr.Error())
				parseProgress.finishUnit(parseIndex, benchSuiteRowFail, parseLabel, 0, getBenchSuiteFirstError(parseErr.Error()), time.Since(parseStart))
			} else {
				parseLaneState.Status = benchSuiteStatusOK
				parseLaneState.Reason = ""
				parseReport.Size = append(parseReport.Size, parseSize)
				parsePartial.Size = &parseSize
				storeProgress()
				parseProgress.finishUnit(parseIndex, benchSuiteRowOK, parseLabel, 1, fmt.Sprintf("raw %d gzip %d brotli %d bytes", parseSize.RawBytes, parseSize.GzipBytes, parseSize.BrotliBytes), time.Since(parseStart))
			}
		}
	}
	if hasBenchSuiteLane(parseConfig.lanes, benchSuiteLaneBuild) {
		parseLaneState := parseLaneByName[benchSuiteLaneBuild]
		parseLabel := benchSuiteLaneBuild + " " + parseConfig.buildPattern
		if parseConfig.resumeFrom != nil && parseConfig.resumeFrom.Build != nil && parseConfig.resumeFrom.Build.Pattern == parseConfig.buildPattern {
			parseProgress.logSkip(parseLabel, 1, 0)
			parseLaneState.Status = benchSuiteStatusOK
			parseLaneState.Reason = ""
			parseBuild := *parseConfig.resumeFrom.Build
			parseReport.Build = &parseBuild
		} else {
			parseIndex := parseProgress.beginUnit(parseLabel, "first build + rebuild")
			parseStop := parseProgress.startHeartbeat(parseLabel, benchSuiteHeartbeatInterval)
			parseStart := time.Now()
			parseBuild, parseErr := runBenchSuiteWithBuildTags(parseConfig.tags, func() (benchSuiteBuild, error) {
				return benchSuiteMeasureBuild(parseConfig.rootPath, parseConfig.buildPattern, "wasm")
			})
			parseStop()
			if parseErr != nil {
				parseLaneState.Status = benchSuiteStatusFailed
				parseLaneState.Reason = parseErr.Error()
				parseErrs = append(parseErrs, "build "+parseErr.Error())
				parseProgress.finishUnit(parseIndex, benchSuiteRowFail, parseLabel, 0, getBenchSuiteFirstError(parseErr.Error()), time.Since(parseStart))
			} else {
				parseLaneState.Status = benchSuiteStatusOK
				parseLaneState.Reason = ""
				parseReport.Build = &parseBuild
				parsePartial.Build = &parseBuild
				storeProgress()
				parseProgress.finishUnit(parseIndex, benchSuiteRowOK, parseLabel, 1, fmt.Sprintf("first %dms rebuild %dms", parseBuild.ColdMs, parseBuild.WarmMs), time.Since(parseStart))
			}
		}
	}
	parseProgress.writeSummary()

	for _, parseName := range []string{benchSuiteLaneNative, benchSuiteLaneWasm, benchSuiteLaneSize, benchSuiteLaneBuild, benchSuiteLaneBrowser} {
		parseReport.Lanes = append(parseReport.Lanes, *parseLaneByName[parseName])
	}
	if len(parseErrs) > 0 {
		return parseReport, fmt.Errorf("bench suite lanes failed: %s", strings.Join(parseErrs, "; "))
	}
	return parseReport, nil
}

// runBenchSuiteTarget runs the warm-ups and the measured run of one package target with START,
// warmup, heartbeat and OK/FAIL/TIMEOUT progress lines, and classifies the outcome.
func runBenchSuiteTarget(parseL launcher, parseMicro benchmarkConfig, parseConfig benchSuiteConfig, parseProgress *benchSuiteProgress, parseJob benchSuiteJob) benchSuiteTarget {
	parseLabel := parseJob.lane + " " + parseJob.packagePath
	parseWarmups := parseConfig.warmup
	if parseJob.lane == benchSuiteLaneWasm && !parseConfig.hasWarmup {
		parseWarmups = 0
	}
	parseIndex := parseProgress.beginUnit(parseLabel, fmt.Sprintf("warmup %d + count %d", parseWarmups, parseConfig.count))
	parseStop := parseProgress.startHeartbeat(parseLabel, benchSuiteHeartbeatInterval)
	parseStart := time.Now()
	parseWarmConfig := parseMicro
	parseWarmConfig.count = 1
	var parsePackageReport benchmarkPackageReport
	isAborted := false
	for parseWarmup := 1; parseWarmup <= parseWarmups; parseWarmup++ {
		parseProgress.logWarmup(parseIndex, parseWarmup, parseWarmups, parseLabel)
		// Warm-up failures are ignored as before, except a timeout which would only repeat.
		parseWarmReport := benchSuiteRunPackage(parseL, parseWarmConfig, parseJob.lane, parseJob.packagePath)
		if !parseWarmReport.OK && isBenchSuiteTimeoutError(parseWarmReport.Error) {
			parsePackageReport = parseWarmReport
			isAborted = true
			break
		}
	}
	if !isAborted {
		parsePackageReport = benchSuiteRunPackage(parseL, parseMicro, parseJob.lane, parseJob.packagePath)
	}
	parseStop()
	parseDuration := time.Since(parseStart)

	// With build tags the exact go test command line is logged so the tags are visibly forwarded.
	if parseConfig.tags != "" && parsePackageReport.Command != "" {
		parseProgress.writeLine("      command: " + parsePackageReport.Command + "  (dir " + parsePackageReport.Workspace + ")")
	}
	parseTarget := benchSuiteTarget{Lane: parseJob.lane, Package: parseJob.packagePath, DurationMs: parseDuration.Milliseconds(),
		GoOS: parsePackageReport.GoOS, GoArch: parsePackageReport.GoArch}
	parseFailedDir := getBenchSuiteFailedDir(parseConfig.outPath)
	// A native run that printed a different goarch than requested measured the wrong machine
	// configuration, so its numbers must not enter the report.
	isArchMismatch := parsePackageReport.OK && parseJob.lane == benchSuiteLaneNative && parseConfig.goarch != "" &&
		parsePackageReport.GoArch != "" && parsePackageReport.GoArch != parseConfig.goarch
	switch {
	case isArchMismatch:
		parsePackageReport.OK = false
		parsePackageReport.Error = fmt.Sprintf("requested -goarch %s but the benchmarks ran with goarch: %s", parseConfig.goarch, parsePackageReport.GoArch)
		parsePackageReport.Output = ""
		fallthrough
	case !parsePackageReport.OK && !isBenchSuiteTimeoutError(parsePackageReport.Error):
		parseTarget.Status = benchSuiteStatusFailed
		parseTarget.Error = storeBenchSuiteTargetFailure(parseProgress, parseFailedDir, parseJob, parsePackageReport)
		parseProgress.finishUnit(parseIndex, benchSuiteRowFail, parseLabel, 0, getBenchSuiteFirstError(parsePackageReport.Error), parseDuration)
	case parsePackageReport.OK:
		parseTarget.Status = benchSuiteStatusOK
		parseTarget.Warnings = parsePackageReport.Warnings
		if parseKept, parseWarningPath := clipBenchSuiteWarnings(parsePackageReport.Warnings, parseFailedDir, parseJob.lane, parseJob.packagePath); parseWarningPath != "" {
			parseTarget.Warnings = parseKept
			parseProgress.writeLine("      all warnings of " + parseLabel + ": " + parseWarningPath)
		}
		parseTarget.Entries = []benchSuiteEntry{}
		for _, parseBenchmark := range parsePackageReport.Benchmarks {
			parseTarget.Entries = append(parseTarget.Entries, buildBenchSuiteEntry(parseJob.lane, parseJob.packagePath, parseBenchmark))
		}
		parseProgress.finishUnit(parseIndex, benchSuiteRowOK, parseLabel, len(parseTarget.Entries), fmt.Sprintf("%d benchmarks", len(parseTarget.Entries)), parseDuration)
	default:
		// Timeout: the child was killed, but the complete benchmark lines it printed are kept so a
		// long run is not wasted; every kept entry is flagged partial.
		parseTarget.Status = benchSuiteStatusTimeout
		parseTarget.Error = storeBenchSuiteTargetFailure(parseProgress, parseFailedDir, parseJob, parsePackageReport)
		for _, parseBenchmark := range parsePackageReport.Benchmarks {
			parseEntry := buildBenchSuiteEntry(parseJob.lane, parseJob.packagePath, parseBenchmark)
			parseEntry.Partial = true
			parseTarget.Entries = append(parseTarget.Entries, parseEntry)
		}
		parseDetail := strings.TrimSpace(parsePackageReport.Error) + ", killed, continuing"
		if len(parseTarget.Entries) > 0 {
			parseDetail = fmt.Sprintf("%s, killed, kept %d partial benchmarks", strings.TrimSpace(parsePackageReport.Error), len(parseTarget.Entries))
		}
		parseProgress.finishUnit(parseIndex, benchSuiteRowTimeout, parseLabel, len(parseTarget.Entries), parseDetail, parseDuration)
	}
	return parseTarget
}

// storeBenchSuiteTargetFailure writes the full output of a failed or timed-out target to the
// failed directory, announces the path, and returns the clipped text kept in the report.
func storeBenchSuiteTargetFailure(parseProgress *benchSuiteProgress, parseFailedDir string, parseJob benchSuiteJob, parseReport benchmarkPackageReport) string {
	parseError := strings.TrimSpace(parseReport.Error)
	parseFull := strings.TrimSpace(parseReport.Output)
	if parseFull == "" {
		parseFull = parseError
	}
	// The error text normally embeds the whole output; only a long one is worth a file.
	parseClipped := clipBenchSuiteOutput(parseError)
	parsePath := ""
	if parseClipped != parseError || len(strings.Split(parseFull, "\n")) > benchSuiteClipHeadLines+benchSuiteClipTailLines || len(parseFull) > benchSuiteClipMaxBytes {
		parsePath = storeBenchSuiteFailedOutput(parseFailedDir, parseJob.lane, parseJob.packagePath, "", parseFull)
	}
	if parsePath == "" {
		return parseClipped
	}
	parseProgress.writeLine("      full output of " + parseJob.lane + " " + parseJob.packagePath + ": " + parsePath)
	return parseClipped + "\n(full output: " + parsePath + ")"
}

// applyBenchSuiteTarget folds one finished target into its lane: entries and warnings on success,
// status and reason on failure or timeout (a failure outranks a timeout).
func applyBenchSuiteTarget(parseLaneState *benchSuiteLane, parseTarget benchSuiteTarget) {
	if parseTarget.Status != benchSuiteStatusOK {
		if parseTarget.Status == benchSuiteStatusFailed || parseLaneState.Status != benchSuiteStatusFailed {
			parseLaneState.Status = parseTarget.Status
		}
		parseMessage := fmt.Sprintf("%s: %s", parseTarget.Package, parseTarget.Error)
		parseLaneState.Reason = strings.TrimPrefix(strings.TrimSpace(parseLaneState.Reason+"; "+parseMessage), "; ")
		// A timeout keeps the partial entries it parsed; a failure never has any.
		parseLaneState.Entries = append(parseLaneState.Entries, parseTarget.Entries...)
		return
	}
	for _, parseWarning := range parseTarget.Warnings {
		parseLaneState.Warnings = append(parseLaneState.Warnings, parseTarget.Package+": "+parseWarning)
	}
	parseLaneState.Entries = append(parseLaneState.Entries, parseTarget.Entries...)
}

var benchSuiteCPUSuffix = regexp.MustCompile(`-\d+$`)

// buildBenchSuiteEntry converts one parsed benchmark into a suite entry with ns/op samples.
// The GOMAXPROCS suffix is stripped from the name so ids stay stable across core counts.
func buildBenchSuiteEntry(parseLane string, parsePackagePath string, parseBenchmark benchmarkResultReport) benchSuiteEntry {
	parseSamples := []float64{}
	parseBytes := []float64{}
	parseAllocs := []float64{}
	for _, parseSample := range parseBenchmark.Samples {
		if parseValue, isFound := parseSample.Metrics["ns/op"]; isFound {
			parseSamples = append(parseSamples, parseValue)
		}
		if parseValue, isFound := parseSample.Metrics["B/op"]; isFound {
			parseBytes = append(parseBytes, parseValue)
		}
		if parseValue, isFound := parseSample.Metrics["allocs/op"]; isFound {
			parseAllocs = append(parseAllocs, parseValue)
		}
	}
	parseMedian, parseMAD := summarizeBenchSuiteSamples(parseSamples)
	parseEntry := benchSuiteEntry{
		ID:      parseLane + "/" + parsePackagePath + "/" + benchSuiteCPUSuffix.ReplaceAllString(parseBenchmark.Name, ""),
		Unit:    "ns/op",
		Samples: parseSamples,
		Median:  parseMedian,
		MAD:     parseMAD,
		N:       len(parseSamples),
	}
	if len(parseBytes) > 0 {
		parseValue := medianBenchmarkValues(parseBytes)
		parseEntry.BytesPerOp = &parseValue
	}
	if len(parseAllocs) > 0 {
		parseValue := medianBenchmarkValues(parseAllocs)
		parseEntry.AllocsPerOp = &parseValue
	}
	return parseEntry
}

// summarizeBenchSuiteSamples returns the median and median absolute deviation of parseSamples.
func summarizeBenchSuiteSamples(parseSamples []float64) (float64, float64) {
	if len(parseSamples) == 0 {
		return 0, 0
	}
	parseMedian := medianBenchmarkValues(parseSamples)
	parseDeviations := make([]float64, len(parseSamples))
	for parseIndex, parseValue := range parseSamples {
		parseDeviations[parseIndex] = math.Abs(parseValue - parseMedian)
	}
	return parseMedian, medianBenchmarkValues(parseDeviations)
}

// writeBenchSuiteReport writes the report as indented JSON, creating parent directories.
func writeBenchSuiteReport(parsePath string, parseReport benchSuiteReport) error {
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr != nil {
		return fmt.Errorf("create bench suite report directory: %w", parseErr)
	}
	parsePayload, parseErr2 := json.MarshalIndent(parseReport, "", "  ")
	if parseErr2 != nil {
		return fmt.Errorf("marshal bench suite report: %w", parseErr2)
	}
	return os.WriteFile(parsePath, append(parsePayload, '\n'), 0o644)
}

// loadBenchSuiteReport reads a suite report and rejects other schema versions.
func loadBenchSuiteReport(parsePath string) (benchSuiteReport, error) {
	parseContent, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return benchSuiteReport{}, parseErr
	}
	var parseReport benchSuiteReport
	if parseErr2 := json.Unmarshal(parseContent, &parseReport); parseErr2 != nil {
		return benchSuiteReport{}, fmt.Errorf("parse %s: %w", parsePath, parseErr2)
	}
	if parseReport.SchemaVersion != benchSuiteSchemaVersion {
		return benchSuiteReport{}, fmt.Errorf("%s has schemaVersion %q, want %q", parsePath, parseReport.SchemaVersion, benchSuiteSchemaVersion)
	}
	return parseReport, nil
}

// printBenchSuiteReport prints a short human-readable run summary.
func printBenchSuiteReport(parseReport benchSuiteReport, parsePath string) {
	fmt.Println("GWC bench suite")
	fmt.Printf("report:  %s\n", parsePath)
	if parseMachine := formatBenchmarkMachine(parseReport.Machine); parseMachine != "" {
		fmt.Printf("machine: %s\n", parseMachine)
	}
	fmt.Printf("run:     count=%d warmup=%d seed=%d\n", parseReport.Run.Count, parseReport.Run.Warmup, parseReport.Run.Seed)
	for _, parseLane := range parseReport.Lanes {
		parseLine := fmt.Sprintf("[%s] %s (%d entries)", parseLane.Status, parseLane.Lane, len(parseLane.Entries))
		if parseLane.Reason != "" {
			parseLine += ": " + parseLane.Reason
		}
		fmt.Println(parseLine)
	}
	for _, parseSize := range parseReport.Size {
		fmt.Printf("size %s: raw %d, gzip %d, brotli %d\n", parseSize.Artifact, parseSize.RawBytes, parseSize.GzipBytes, parseSize.BrotliBytes)
	}
	if parseReport.Build != nil {
		fmt.Printf("build %s (%s): first %dms, rebuild %dms\n", parseReport.Build.Pattern, parseReport.Build.Target, parseReport.Build.ColdMs, parseReport.Build.WarmMs)
	}
}
