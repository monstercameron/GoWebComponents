package main

import (
	"fmt"
	"os"
	"path"
	"path/filepath"
	"regexp"
	"sort"
	"strings"
	"time"
	"unicode/utf8"
)

// Limits of the failure text kept in a report and on the console; the full text goes to a file.
const (
	benchSuiteClipHeadLines = 20
	benchSuiteClipTailLines = 40
	benchSuiteClipMaxBytes  = 8 * 1024
)

// benchSuiteDefaultTargetTimeouts lists packages whose benchmark set needs far longer than the
// global -target-timeout (102 benchmarks x 5 samples + warm-up is about 40 minutes).
var benchSuiteDefaultTargetTimeouts = map[string]time.Duration{
	"./internal/runtime": 75 * time.Minute,
}

// benchSuiteGoarchPattern accepts the GOARCH spellings the go command knows (arm64, amd64, 386).
var benchSuiteGoarchPattern = regexp.MustCompile(`^[a-z0-9]+$`)

// benchSuiteSlugPattern matches the runs of characters replaced when naming a file after a package.
var benchSuiteSlugPattern = regexp.MustCompile(`[^A-Za-z0-9]+`)

// buildBenchmarkNativeEnv returns the child environment of a native-lane go test. The inherited
// GOOS and GOARCH are always dropped; a requested parseGoarch is then passed explicitly so it does
// not depend on what the parent exported or on `go env -w`.
func buildBenchmarkNativeEnv(parseGoarch string) []string {
	parseEnv := buildNativeGoEnv()
	if strings.TrimSpace(parseGoarch) != "" {
		parseEnv = append(parseEnv, "GOARCH="+strings.TrimSpace(parseGoarch))
	}
	return parseEnv
}

// normalizeBenchSuitePackageKey turns docs-style spellings (internal/runtime, .\internal\runtime\)
// into the ./internal/runtime form used by package targets.
func normalizeBenchSuitePackageKey(parseKey string) string {
	parseValue := strings.TrimSpace(filepath.ToSlash(parseKey))
	parseValue = strings.TrimSuffix(strings.TrimPrefix(parseValue, "./"), "/")
	if parseValue == "" {
		return "."
	}
	return "./" + parseValue
}

// parseBenchSuiteTimeoutList parses "pkg=dur,pkg=dur" into a per-package timeout map.
func parseBenchSuiteTimeoutList(parseText string) (map[string]time.Duration, error) {
	parseResult := map[string]time.Duration{}
	for _, parseItem := range strings.Split(parseText, ",") {
		parseItem = strings.TrimSpace(parseItem)
		if parseItem == "" {
			continue
		}
		parseKey, parseValue, isFound := strings.Cut(parseItem, "=")
		if !isFound || strings.TrimSpace(parseKey) == "" {
			return nil, fmt.Errorf("target-timeout-for entry %q must look like ./pkg=75m", parseItem)
		}
		parseDuration, parseErr := time.ParseDuration(strings.TrimSpace(parseValue))
		if parseErr != nil || parseDuration < 0 {
			return nil, fmt.Errorf("target-timeout-for entry %q has an invalid duration", parseItem)
		}
		parseResult[normalizeBenchSuitePackageKey(parseKey)] = parseDuration
	}
	return parseResult, nil
}

// getBenchmarkTargetTimeout resolves the timeout of one package: an explicit per-package value
// wins, then a built-in longer default (never shorter than the global value), then the global
// value. A disabled global timeout (0) also disables the built-in defaults.
func getBenchmarkTargetTimeout(parseConfig benchmarkConfig, parsePackagePath string) time.Duration {
	parseKey := normalizeBenchSuitePackageKey(parsePackagePath)
	if parseValue, isFound := parseConfig.targetTimeoutFor[parseKey]; isFound {
		return parseValue
	}
	if parseConfig.targetTimeout <= 0 {
		return 0
	}
	if parseValue, isFound := benchSuiteDefaultTargetTimeouts[parseKey]; isFound && parseValue > parseConfig.targetTimeout {
		return parseValue
	}
	return parseConfig.targetTimeout
}

// parseBenchmarkPlatformHeader returns the first goos and goarch header lines of go test output.
func parseBenchmarkPlatformHeader(parseOutput string) (string, string) {
	var parseGoOS, parseGoArch string
	for _, parseLine := range strings.Split(parseOutput, "\n") {
		parseLine = strings.TrimSpace(parseLine)
		if parseValue, isFound := strings.CutPrefix(parseLine, "goos:"); isFound && parseGoOS == "" {
			parseGoOS = strings.TrimSpace(parseValue)
		}
		if parseValue, isFound := strings.CutPrefix(parseLine, "goarch:"); isFound && parseGoArch == "" {
			parseGoArch = strings.TrimSpace(parseValue)
		}
		if parseGoOS != "" && parseGoArch != "" {
			break
		}
	}
	return parseGoOS, parseGoArch
}

// trimBenchmarkIncompleteLine drops a final benchmark result line that a kill cut in half, so a
// truncated sample is never parsed as a measurement.
func trimBenchmarkIncompleteLine(parseOutput string) string {
	parseText := strings.TrimRight(parseOutput, "\r\n ")
	parseIndex := strings.LastIndex(parseText, "\n")
	parseLast := parseText[parseIndex+1:]
	if strings.HasPrefix(strings.TrimSpace(parseLast), "Benchmark") && !strings.Contains(parseLast, "allocs/op") {
		if parseIndex < 0 {
			return ""
		}
		return parseText[:parseIndex]
	}
	return parseText
}

// clipBenchSuiteOutput keeps the first 20 and last 40 lines of parseText, at most 8 KB in total.
func clipBenchSuiteOutput(parseText string) string {
	parseLines := strings.Split(strings.ReplaceAll(strings.TrimSpace(parseText), "\r\n", "\n"), "\n")
	parseClipped := strings.Join(parseLines, "\n")
	if len(parseLines) > benchSuiteClipHeadLines+benchSuiteClipTailLines {
		parseOmitted := len(parseLines) - benchSuiteClipHeadLines - benchSuiteClipTailLines
		parseKept := append([]string{}, parseLines[:benchSuiteClipHeadLines]...)
		parseKept = append(parseKept, fmt.Sprintf("... [%d lines omitted] ...", parseOmitted))
		parseKept = append(parseKept, parseLines[len(parseLines)-benchSuiteClipTailLines:]...)
		parseClipped = strings.Join(parseKept, "\n")
	}
	if len(parseClipped) <= benchSuiteClipMaxBytes {
		return parseClipped
	}
	// Few but very long lines: cut by bytes, never inside a UTF-8 sequence.
	parseHead := 3000
	parseTail := 4900
	for parseHead > 0 && !utf8.RuneStart(parseClipped[parseHead]) {
		parseHead--
	}
	parseStart := len(parseClipped) - parseTail
	for parseStart < len(parseClipped) && !utf8.RuneStart(parseClipped[parseStart]) {
		parseStart++
	}
	return fmt.Sprintf("%s\n... [%d bytes omitted] ...\n%s", parseClipped[:parseHead], parseStart-parseHead, parseClipped[parseStart:])
}

// getBenchSuiteFailedDir returns the directory that holds the full output of failed targets.
func getBenchSuiteFailedDir(parseOutPath string) string {
	if strings.TrimSpace(parseOutPath) == "" {
		return ""
	}
	return parseOutPath + ".failed"
}

// getBenchSuiteSlug turns a package path into a file-name-safe slug ("./internal/runtime" -> "internal-runtime").
func getBenchSuiteSlug(parsePackagePath string) string {
	parseSlug := strings.Trim(benchSuiteSlugPattern.ReplaceAllString(parsePackagePath, "-"), "-")
	if parseSlug == "" {
		return "root"
	}
	return parseSlug
}

// storeBenchSuiteFailedOutput writes the full text of one target to <dir>/<lane>-<slug><suffix>.txt
// and returns the path; it returns an empty path when parseDir is empty or the write fails.
func storeBenchSuiteFailedOutput(parseDir string, parseLane string, parsePackagePath string, parseSuffix string, parseText string) string {
	if parseDir == "" {
		return ""
	}
	if parseErr := os.MkdirAll(parseDir, 0o755); parseErr != nil {
		return ""
	}
	parsePath := filepath.Join(parseDir, parseLane+"-"+getBenchSuiteSlug(parsePackagePath)+parseSuffix+".txt")
	if parseErr := os.WriteFile(parsePath, []byte(parseText+"\n"), 0o644); parseErr != nil {
		return ""
	}
	return parsePath
}

// clipBenchSuiteWarnings keeps the first 20 and last 40 warning lines; the full list goes to a
// file when it had to be cut.
func clipBenchSuiteWarnings(parseWarnings []string, parseDir string, parseLane string, parsePackagePath string) ([]string, string) {
	if len(parseWarnings) <= benchSuiteClipHeadLines+benchSuiteClipTailLines {
		return parseWarnings, ""
	}
	parsePath := storeBenchSuiteFailedOutput(parseDir, parseLane, parsePackagePath, "-warnings", strings.Join(parseWarnings, "\n"))
	parseKept := append([]string{}, parseWarnings[:benchSuiteClipHeadLines]...)
	parseKept = append(parseKept, fmt.Sprintf("... [%d warnings omitted] ...", len(parseWarnings)-benchSuiteClipHeadLines-benchSuiteClipTailLines))
	parseKept = append(parseKept, parseWarnings[len(parseWarnings)-benchSuiteClipTailLines:]...)
	return parseKept, parsePath
}

// benchSuiteTagPattern accepts one go build tag: letters, digits, and the punctuation build tags use.
var benchSuiteTagPattern = regexp.MustCompile(`^!?[A-Za-z0-9_.]+$`)

// splitBenchSuiteList splits a comma-separated flag value and drops empty items.
func splitBenchSuiteList(parseText string) []string {
	parseItems := []string{}
	for _, parseItem := range strings.Split(parseText, ",") {
		if parseTrimmed := strings.TrimSpace(parseItem); parseTrimmed != "" {
			parseItems = append(parseItems, parseTrimmed)
		}
	}
	return parseItems
}

// normalizeBenchSuiteTags validates a comma (or space) separated tag list and returns it sorted,
// deduplicated and comma-joined so equal tag sets hash and compare equal.
func normalizeBenchSuiteTags(parseText string) (string, error) {
	parseSeen := map[string]bool{}
	parseTags := []string{}
	for _, parseTag := range splitBenchSuiteList(strings.ReplaceAll(parseText, " ", ",")) {
		if !benchSuiteTagPattern.MatchString(parseTag) {
			return "", fmt.Errorf("bench suite tag %q is not a build tag (use letters, digits, _ and . in a comma list such as production,foo)", parseTag)
		}
		if !parseSeen[parseTag] {
			parseSeen[parseTag] = true
			parseTags = append(parseTags, parseTag)
		}
	}
	sort.Strings(parseTags)
	return strings.Join(parseTags, ","), nil
}

// normalizeBenchSuitePackageList puts every -packages entry in the ./pkg form and drops repeats.
func normalizeBenchSuitePackageList(parseItems []string) []string {
	var parseResult []string
	parseSeen := map[string]bool{}
	for _, parseItem := range parseItems {
		if strings.TrimSpace(parseItem) == "" {
			continue
		}
		parseKey := normalizeBenchSuitePackageKey(parseItem)
		if !parseSeen[parseKey] {
			parseSeen[parseKey] = true
			parseResult = append(parseResult, parseKey)
		}
	}
	return parseResult
}

// isBenchSuitePackageMatch reports whether one discovered package path matches a -packages entry:
// an exact path, a path.Match glob (* does not cross a slash), or a prefix ending in /...
func isBenchSuitePackageMatch(parsePattern string, parsePackagePath string) bool {
	if parsePattern == parsePackagePath {
		return true
	}
	if parsePrefix, isTree := strings.CutSuffix(parsePattern, "/..."); isTree {
		return parsePackagePath == parsePrefix || strings.HasPrefix(parsePackagePath, parsePrefix+"/")
	}
	if !strings.ContainsAny(parsePattern, "*?[") {
		return false
	}
	parseMatched, parseErr := path.Match(parsePattern, parsePackagePath)
	return parseErr == nil && parseMatched
}

// filterBenchSuitePackages keeps only the discovered packages selected by parsePatterns. An empty
// pattern list keeps everything. A pattern that matches no package of the selected lanes is an
// error that lists the valid package paths.
func filterBenchSuitePackages(parseNative []string, parseWasm []string, parseLanes []string, parsePatterns []string) ([]string, []string, error) {
	if len(parsePatterns) == 0 {
		return parseNative, parseWasm, nil
	}
	parseKnown := map[string]bool{}
	if hasBenchSuiteLane(parseLanes, benchSuiteLaneNative) {
		for _, parsePackage := range parseNative {
			parseKnown[parsePackage] = true
		}
	}
	if hasBenchSuiteLane(parseLanes, benchSuiteLaneWasm) {
		for _, parsePackage := range parseWasm {
			parseKnown[parsePackage] = true
		}
	}
	parseValid := make([]string, 0, len(parseKnown))
	for parsePackage := range parseKnown {
		parseValid = append(parseValid, parsePackage)
	}
	sort.Strings(parseValid)
	parseUnknown := []string{}
	for _, parsePattern := range parsePatterns {
		isMatched := false
		for _, parsePackage := range parseValid {
			if isBenchSuitePackageMatch(parsePattern, parsePackage) {
				isMatched = true
				break
			}
		}
		if !isMatched {
			parseUnknown = append(parseUnknown, parsePattern)
		}
	}
	if len(parseUnknown) > 0 {
		return nil, nil, fmt.Errorf("bench suite -packages entries match no benchmark package: %s; valid packages: %s", strings.Join(parseUnknown, ", "), strings.Join(parseValid, ", "))
	}
	parseKeep := func(parsePackages []string) []string {
		parseKept := []string{}
		for _, parsePackage := range parsePackages {
			for _, parsePattern := range parsePatterns {
				if isBenchSuitePackageMatch(parsePattern, parsePackage) {
					parseKept = append(parseKept, parsePackage)
					break
				}
			}
		}
		return parseKept
	}
	return parseKeep(parseNative), parseKeep(parseWasm), nil
}

// runBenchSuiteWithBuildTags runs parseRun with GOFLAGS carrying -tags=<parseTags> so the size and
// build lanes (whose go build argument lists live in other files) compile the same tagged variant
// as the micro lanes. An empty tag list calls parseRun unchanged. The previous GOFLAGS value is
// restored afterwards and any -tags already in it is replaced.
func runBenchSuiteWithBuildTags[T any](parseTags string, parseRun func() (T, error)) (T, error) {
	if parseTags == "" {
		return parseRun()
	}
	parseOld, hasOld := os.LookupEnv("GOFLAGS")
	parseKept := []string{}
	for _, parseField := range strings.Fields(parseOld) {
		if !strings.HasPrefix(parseField, "-tags=") {
			parseKept = append(parseKept, parseField)
		}
	}
	parseKept = append(parseKept, "-tags="+parseTags)
	_ = os.Setenv("GOFLAGS", strings.Join(parseKept, " "))
	defer func() {
		if hasOld {
			_ = os.Setenv("GOFLAGS", parseOld)
		} else {
			_ = os.Unsetenv("GOFLAGS")
		}
	}()
	return parseRun()
}
