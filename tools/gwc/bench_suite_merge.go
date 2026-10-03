package main

import (
	"errors"
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"
	"time"
)

// benchSuiteLaneOrder is the fixed order of lanes in every report.
var benchSuiteLaneOrder = []string{benchSuiteLaneNative, benchSuiteLaneWasm, benchSuiteLaneSize, benchSuiteLaneBuild, benchSuiteLaneBrowser}

// benchSuiteMergedTarget is one target during a merge: its summary, entries and source file.
type benchSuiteMergedTarget struct {
	target  benchSuiteTarget
	entries []benchSuiteEntry
}

// runBenchSuiteMerge implements `gwc bench suite merge -o out.json a.json b.json ...`.
func runBenchSuiteMerge(parseArgs []string) error {
	parseFs := flag.NewFlagSet("bench suite merge", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseFs.Usage = func() {
		fmt.Fprintln(parseFs.Output(), "Usage: gwc bench suite merge [-force] -o <out.json> <a.json> <b.json> ...\n\n"+
			"Merges lanes, entries, targets, size and build of several suite reports into one so a baseline can be\n"+
			"assembled from separate runs. Later files override the same target id (a failed or timed-out target never\n"+
			"replaces an earlier ok one). Every entry gets a sourceReport field naming its file. Reports must share\n"+
			"seed, count and machine arch/hostArch/cpuModel unless -force is given. A relative -o is resolved against\n"+
			"the directory gwc was started in.")
		parseFs.PrintDefaults()
	}
	parseOut := parseFs.String("o", "", "Merged report path (required)")
	isForce := parseFs.Bool("force", false, "Merge even when seed, count or machine differ")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parsePaths := parseFs.Args()
	if strings.TrimSpace(*parseOut) == "" || len(parsePaths) < 2 {
		return errors.New("usage: gwc bench suite merge [-force] -o <out.json> <a.json> <b.json> ...")
	}
	parseAbsOut, parseErr2 := filepath.Abs(*parseOut)
	if parseErr2 != nil {
		return fmt.Errorf("resolve merge output path: %w", parseErr2)
	}
	parseReports := make([]benchSuiteReport, 0, len(parsePaths))
	parseNames := make([]string, 0, len(parsePaths))
	for _, parsePath := range parsePaths {
		parseReport, parseLoadErr := loadBenchSuiteReport(parsePath)
		if parseLoadErr != nil {
			return parseLoadErr
		}
		parseReports = append(parseReports, parseReport)
		parseNames = append(parseNames, filepath.Base(parsePath))
	}
	parseMerged, parseErr3 := mergeBenchSuiteReports(parseNames, parseReports, *isForce)
	if parseErr3 != nil {
		return parseErr3
	}
	if parseWriteErr := writeBenchSuiteReport(parseAbsOut, parseMerged); parseWriteErr != nil {
		return parseWriteErr
	}
	parseEntries := 0
	for _, parseLane := range parseMerged.Lanes {
		parseEntries += len(parseLane.Entries)
	}
	fmt.Printf("merged %d reports into %s (%d targets, %d entries)\n", len(parseReports), parseAbsOut, len(parseMerged.Targets), parseEntries)
	return nil
}

// mergeBenchSuiteReports combines parseReports (named by parseNames) into one report. The first
// report supplies suite, run, machine and refs; later reports override same-id targets.
func mergeBenchSuiteReports(parseNames []string, parseReports []benchSuiteReport, isForce bool) (benchSuiteReport, error) {
	if len(parseReports) == 0 {
		return benchSuiteReport{}, errors.New("nothing to merge")
	}
	parseFirst := parseReports[0]
	for parseIndex := 1; parseIndex < len(parseReports); parseIndex++ {
		parseOther := parseReports[parseIndex]
		parseProblems := []string{}
		if parseOther.Run.Seed != parseFirst.Run.Seed {
			parseProblems = append(parseProblems, fmt.Sprintf("seed %d vs %d", parseFirst.Run.Seed, parseOther.Run.Seed))
		}
		if parseOther.Run.Count != parseFirst.Run.Count {
			parseProblems = append(parseProblems, fmt.Sprintf("count %d vs %d", parseFirst.Run.Count, parseOther.Run.Count))
		}
		parseProblems = append(parseProblems, getBenchSuiteMachineMismatch(parseFirst.Machine, parseOther.Machine)...)
		if len(parseProblems) > 0 && !isForce {
			return benchSuiteReport{}, fmt.Errorf("refusing to merge %s with %s (%s); pass -force to merge anyway",
				parseNames[0], parseNames[parseIndex], strings.Join(parseProblems, "; "))
		}
	}

	parseOrder := []string{}
	parseByKey := map[string]*benchSuiteMergedTarget{}
	parseOrphans := map[string][]benchSuiteEntry{}
	parseRunOrder := append([]string{}, parseFirst.Run.Order...)
	parseSizes := []benchSuiteSize{}
	var parseBuild *benchSuiteBuild
	parseLaneChoice := map[string]benchSuiteLane{}
	for parseIndex, parseReport := range parseReports {
		parseSource := parseNames[parseIndex]
		for _, parseID := range parseReport.Run.Order {
			if !containsBenchSuiteString(parseRunOrder, parseID) {
				parseRunOrder = append(parseRunOrder, parseID)
			}
		}
		parseAssigned, parseLoose := assignBenchSuiteEntries(parseReport)
		for _, parseTarget := range parseReport.Targets {
			parseKey := getBenchSuiteTargetID(parseTarget.Lane, parseTarget.Package)
			parseCandidate := &benchSuiteMergedTarget{target: parseTarget}
			parseCandidate.target.Entries = nil
			for _, parseEntry := range parseAssigned[parseKey] {
				if parseEntry.SourceReport == "" {
					parseEntry.SourceReport = parseSource
				}
				parseCandidate.entries = append(parseCandidate.entries, parseEntry)
			}
			parseExisting, isKnown := parseByKey[parseKey]
			switch {
			case !isKnown:
				parseByKey[parseKey] = parseCandidate
				parseOrder = append(parseOrder, parseKey)
			case parseExisting.target.Status == benchSuiteStatusOK && parseCandidate.target.Status != benchSuiteStatusOK:
				// A later failure must not erase good data collected earlier.
			default:
				parseByKey[parseKey] = parseCandidate
			}
		}
		for _, parseEntry := range parseLoose {
			if parseEntry.SourceReport == "" {
				parseEntry.SourceReport = parseSource
			}
			parseLane := strings.SplitN(parseEntry.ID, "/", 2)[0]
			parseOrphans[parseLane] = upsertBenchSuiteEntry(parseOrphans[parseLane], parseEntry)
		}
		for _, parseSize := range parseReport.Size {
			parseSizes = upsertBenchSuiteSize(parseSizes, parseSize)
		}
		if parseReport.Build != nil {
			parseCopy := *parseReport.Build
			parseBuild = &parseCopy
		}
		for _, parseLane := range parseReport.Lanes {
			if _, isSeen := parseLaneChoice[parseLane.Lane]; !isSeen || parseLane.Status != benchSuiteStatusNotRun {
				parseLaneChoice[parseLane.Lane] = parseLane
			}
		}
	}

	parseMerged := benchSuiteReport{
		SchemaVersion: benchSuiteSchemaVersion,
		Suite:         parseFirst.Suite,
		GeneratedAt:   time.Now().UTC().Format(time.RFC3339),
		Machine:       parseFirst.Machine,
		Run:           parseFirst.Run,
		Size:          parseSizes,
		Build:         parseBuild,
		Refs:          map[string]any{},
		Health:        map[string]any{},
	}
	parseMerged.Run.Order = parseRunOrder
	for parseKey, parseValue := range parseFirst.Refs {
		parseMerged.Refs[parseKey] = parseValue
	}
	for parseKey, parseValue := range parseFirst.Health {
		parseMerged.Health[parseKey] = parseValue
	}
	parseMerged.Health["merge"] = map[string]any{"sources": parseNames, "force": isForce}

	for _, parseKey := range parseOrder {
		parseSummary := parseByKey[parseKey].target
		parseSummary.Entries = nil
		parseMerged.Targets = append(parseMerged.Targets, parseSummary)
	}
	for _, parseName := range benchSuiteLaneOrder {
		parseLane := parseLaneChoice[parseName]
		if parseLane.Lane == "" {
			parseLane = benchSuiteLane{Lane: parseName, Layer: getBenchSuiteLayer(parseName), Status: benchSuiteStatusNotRun, Reason: "lane not selected for this run"}
		}
		if parseName == benchSuiteLaneNative || parseName == benchSuiteLaneWasm {
			parseRebuilt, isRebuilt := rebuildBenchSuiteLane(parseName, parseOrder, parseByKey, parseOrphans[parseName])
			if isRebuilt {
				parseLane = parseRebuilt
			}
		}
		if parseLane.Entries == nil {
			parseLane.Entries = []benchSuiteEntry{}
		}
		parseMerged.Lanes = append(parseMerged.Lanes, parseLane)
	}
	return parseMerged, nil
}

// rebuildBenchSuiteLane recomputes a micro lane from the merged targets: entries in target order,
// the worst target status, and one reason per target that is not ok. It reports false when the
// lane has no targets and no loose entries.
func rebuildBenchSuiteLane(parseName string, parseOrder []string, parseByKey map[string]*benchSuiteMergedTarget, parseLoose []benchSuiteEntry) (benchSuiteLane, bool) {
	parseLane := benchSuiteLane{Lane: parseName, Layer: getBenchSuiteLayer(parseName), Status: benchSuiteStatusOK, Entries: []benchSuiteEntry{}}
	hasContent := false
	for _, parseKey := range parseOrder {
		parseMerged := parseByKey[parseKey]
		if parseMerged.target.Lane != parseName {
			continue
		}
		hasContent = true
		applyBenchSuiteTarget(&parseLane, benchSuiteTarget{
			Lane: parseName, Package: parseMerged.target.Package, Status: parseMerged.target.Status, Error: parseMerged.target.Error,
			Warnings: parseMerged.target.Warnings, Entries: parseMerged.entries,
		})
	}
	if len(parseLoose) > 0 {
		hasContent = true
		parseLane.Entries = append(parseLane.Entries, parseLoose...)
	}
	return parseLane, hasContent
}

// assignBenchSuiteEntries groups a report's entries by target id (lane/package) using the longest
// matching id prefix, because package paths and sub-benchmark names both contain slashes. Entries
// that match no target are returned separately.
func assignBenchSuiteEntries(parseReport benchSuiteReport) (map[string][]benchSuiteEntry, []benchSuiteEntry) {
	parseAssigned := map[string][]benchSuiteEntry{}
	parseLoose := []benchSuiteEntry{}
	for _, parseLane := range parseReport.Lanes {
		for _, parseEntry := range parseLane.Entries {
			parseBest := ""
			for _, parseTarget := range parseReport.Targets {
				parseKey := getBenchSuiteTargetID(parseTarget.Lane, parseTarget.Package)
				if strings.HasPrefix(parseEntry.ID, parseKey+"/") && len(parseKey) > len(parseBest) {
					parseBest = parseKey
				}
			}
			if parseBest == "" {
				parseLoose = append(parseLoose, parseEntry)
				continue
			}
			parseAssigned[parseBest] = append(parseAssigned[parseBest], parseEntry)
		}
	}
	return parseAssigned, parseLoose
}

// upsertBenchSuiteEntry replaces the entry with the same id or appends parseEntry.
func upsertBenchSuiteEntry(parseEntries []benchSuiteEntry, parseEntry benchSuiteEntry) []benchSuiteEntry {
	for parseIndex := range parseEntries {
		if parseEntries[parseIndex].ID == parseEntry.ID {
			parseEntries[parseIndex] = parseEntry
			return parseEntries
		}
	}
	return append(parseEntries, parseEntry)
}

// upsertBenchSuiteSize replaces the size row of the same artifact or appends parseSize.
func upsertBenchSuiteSize(parseSizes []benchSuiteSize, parseSize benchSuiteSize) []benchSuiteSize {
	for parseIndex := range parseSizes {
		if parseSizes[parseIndex].Artifact == parseSize.Artifact {
			parseSizes[parseIndex] = parseSize
			return parseSizes
		}
	}
	return append(parseSizes, parseSize)
}

// containsBenchSuiteString reports whether parseValues holds parseWant.
func containsBenchSuiteString(parseValues []string, parseWant string) bool {
	for _, parseValue := range parseValues {
		if parseValue == parseWant {
			return true
		}
	}
	return false
}
