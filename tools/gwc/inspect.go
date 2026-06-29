package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"io/fs"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"strings"
)

var runInspectCommand = func(l launcher, args []string) error {
	return l.runInspect(args)
}

type inspectConfig struct {
	rootPath string
	json     bool
}

type inspectSummary struct {
	OK           bool                  `json:"ok"`
	Root         string                `json:"root"`
	Routes       inspectRouteReport    `json:"routes"`
	Dependencies inspectDependencyView `json:"dependencies"`
	Ownership    inspectOwnershipView  `json:"ownership"`
	FileTypes    inspectFileTypesView  `json:"fileTypes"`
	Errors       []string              `json:"errors,omitempty"`
}

type inspectRouteReport struct {
	Registrations     int      `json:"registrations"`
	RouteFiles        []string `json:"routeFiles,omitempty"`
	HasLazySplit      bool     `json:"hasLazySplit"`
	HasPrerenderHint  bool     `json:"hasPrerenderHint"`
	HasMarketingRoute bool     `json:"hasMarketingRoute"`
}

type inspectDependencyView struct {
	ModulePath     string   `json:"modulePath,omitempty"`
	TotalImports   int      `json:"totalImports"`
	SampleImports  []string `json:"sampleImports,omitempty"`
	HasSampleLimit bool     `json:"hasSampleLimit,omitempty"`
}

type inspectOwnershipView struct {
	ImportBoundaryStatus string   `json:"importBoundaryStatus"`
	ImportBoundaryNote   string   `json:"importBoundaryNote"`
	StateBoundaryStatus  string   `json:"stateBoundaryStatus"`
	StateBoundaryNote    string   `json:"stateBoundaryNote"`
	BoundaryFiles        []string `json:"boundaryFiles,omitempty"`
}

type inspectFileTypesView struct {
	TotalFiles      int                `json:"totalFiles"`
	ExtensionCounts []inspectCountView `json:"extensionCounts,omitempty"`
	DirectoryCounts []inspectCountView `json:"directoryCounts,omitempty"`
}

type inspectCountView struct {
	Name  string `json:"name"`
	Count int    `json:"count"`
}

// runInspect parses launcher flags and prints higher-level project inspection reports.
func (parseL launcher) runInspect(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("inspect", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parseRoot := parseFlags.String("root", "", "Root directory to inspect; defaults to the current working directory")
	parseJSON := parseFlags.Bool("json", false, "Emit a machine-readable JSON report")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseConfig, parseErr := parseInspectConfig(inspectConfig{
		rootPath: *parseRoot,
		json:     *parseJSON,
	})
	if parseErr != nil {
		return parseErr
	}
	parseSummary := buildInspectSummary(parseConfig)
	if parseConfig.json {
		renderEncoder := json.NewEncoder(os.Stdout)
		renderEncoder.SetIndent("", "  ")
		return renderEncoder.Encode(parseSummary)
	}
	renderInspectSummary(parseSummary)
	return nil
}

// parseInspectConfig resolves and validates inspect command configuration.
func parseInspectConfig(parseConfig inspectConfig) (inspectConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCWD, parseErr := os.Getwd()
		if parseErr != nil {
			return inspectConfig{}, fmt.Errorf("resolve inspect root from cwd: %w", parseErr)
		}
		parseRootPath = parseCWD
	}
	parseAbsoluteRoot, parseErr := filepath.Abs(parseRootPath)
	if parseErr != nil {
		return inspectConfig{}, fmt.Errorf("resolve inspect root: %w", parseErr)
	}
	parseInfo, parseErr := os.Stat(parseAbsoluteRoot)
	if parseErr != nil {
		return inspectConfig{}, fmt.Errorf("stat inspect root: %w", parseErr)
	}
	if !parseInfo.IsDir() {
		return inspectConfig{}, fmt.Errorf("inspect root is not a directory: %s", parseAbsoluteRoot)
	}
	return inspectConfig{
		rootPath: parseAbsoluteRoot,
		json:     parseConfig.json,
	}, nil
}

// buildInspectSummary collects route, dependency, ownership, and file-type reports.
func buildInspectSummary(parseConfig inspectConfig) inspectSummary {
	parseSummary := inspectSummary{
		OK:   true,
		Root: parseConfig.rootPath,
	}

	parseRoutes, parseRouteErr := buildInspectRoutes(parseConfig.rootPath)
	if parseRouteErr != nil {
		parseSummary.OK = false
		parseSummary.Errors = append(parseSummary.Errors, fmt.Sprintf("routes: %v", parseRouteErr))
	} else {
		parseSummary.Routes = parseRoutes
	}

	parseDependencies, parseDependencyErr := buildInspectDependencies(parseConfig.rootPath)
	if parseDependencyErr != nil {
		parseSummary.OK = false
		parseSummary.Errors = append(parseSummary.Errors, fmt.Sprintf("dependencies: %v", parseDependencyErr))
	} else {
		parseSummary.Dependencies = parseDependencies
	}

	parseOwnership, parseOwnershipErr := buildInspectOwnership(parseConfig.rootPath)
	if parseOwnershipErr != nil {
		parseSummary.OK = false
		parseSummary.Errors = append(parseSummary.Errors, fmt.Sprintf("ownership: %v", parseOwnershipErr))
	} else {
		parseSummary.Ownership = parseOwnership
	}

	parseFileTypes, parseFileTypeErr := buildInspectFileTypes(parseConfig.rootPath)
	if parseFileTypeErr != nil {
		parseSummary.OK = false
		parseSummary.Errors = append(parseSummary.Errors, fmt.Sprintf("file-types: %v", parseFileTypeErr))
	} else {
		parseSummary.FileTypes = parseFileTypes
	}

	return parseSummary
}

// buildInspectRoutes summarizes route registration and delivery shape signals.
func buildInspectRoutes(parseRootPath string) (inspectRouteReport, error) {
	parseFiles, parseErr := collectGoldenPathGoFiles(parseRootPath)
	if parseErr != nil {
		return inspectRouteReport{}, parseErr
	}
	parseReport := inspectRouteReport{}
	parseRouteFiles := map[string]struct{}{}
	for _, parseFile := range parseFiles {
		parseHits := strings.Count(parseFile.Content, "MustDefineRoute(")
		parseHits += strings.Count(parseFile.Content, "router.Register(")
		if parseHits > 0 {
			parseReport.Registrations += parseHits
			parseRouteFiles[parseFile.RelPath] = struct{}{}
		}
		if strings.Contains(parseFile.Content, "ui.Lazy(") || strings.Contains(parseFile.Content, "ui.CreateElement(ui.Lazy") {
			parseReport.HasLazySplit = true
		}
		parseLowered := strings.ToLower(parseFile.Content)
		if strings.Contains(parseLowered, "prerender") || strings.Contains(parseLowered, "static shell") {
			parseReport.HasPrerenderHint = true
		}
		if strings.Contains(parseFile.Content, `"/pricing"`) ||
			strings.Contains(parseFile.Content, `"/capabilities"`) ||
			strings.Contains(parseFile.Content, `"/about"`) ||
			strings.Contains(parseFile.Content, `"/docs"`) {
			parseReport.HasMarketingRoute = true
		}
	}
	for parseFile := range parseRouteFiles {
		parseReport.RouteFiles = append(parseReport.RouteFiles, parseFile)
	}
	sort.Strings(parseReport.RouteFiles)
	return parseReport, nil
}

// buildInspectDependencies resolves module identity and third-party dependency imports.
func buildInspectDependencies(parseRootPath string) (inspectDependencyView, error) {
	parseModuleCommand := exec.Command("go", "list", "-m")
	parseModuleCommand.Dir = parseRootPath
	parseModuleOutput, parseModuleErr := parseModuleCommand.CombinedOutput()
	if parseModuleErr != nil {
		return inspectDependencyView{}, fmt.Errorf("run go list -m: %w", parseModuleErr)
	}

	parseDependencyCommand := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./...")
	parseDependencyCommand.Dir = parseRootPath
	parseDependencyOutput, parseDependencyErr := parseDependencyCommand.CombinedOutput()
	if parseDependencyErr != nil {
		return inspectDependencyView{}, fmt.Errorf("run go list -deps: %w", parseDependencyErr)
	}

	parseDependencySet := map[string]struct{}{}
	for parseLine := range strings.SplitSeq(string(parseDependencyOutput), "\n") {
		parseImportPath := strings.TrimSpace(parseLine)
		if parseImportPath == "" {
			continue
		}
		parseDependencySet[parseImportPath] = struct{}{}
	}
	parseDependencies := make([]string, 0, len(parseDependencySet))
	for parseImportPath := range parseDependencySet {
		parseDependencies = append(parseDependencies, parseImportPath)
	}
	sort.Strings(parseDependencies)

	parseReport := inspectDependencyView{
		ModulePath:   strings.TrimSpace(string(parseModuleOutput)),
		TotalImports: len(parseDependencies),
	}
	if len(parseDependencies) > 20 {
		parseReport.SampleImports = append(parseReport.SampleImports, parseDependencies[:20]...)
		parseReport.HasSampleLimit = true
		return parseReport, nil
	}
	parseReport.SampleImports = parseDependencies
	return parseReport, nil
}

// buildInspectOwnership mirrors doctor ownership checks in one inspect-friendly summary.
func buildInspectOwnership(parseRootPath string) (inspectOwnershipView, error) {
	parseImportCheck := buildDoctorOwnershipBoundaryCheck(parseRootPath)
	parseStateCheck := buildDoctorStateOwnershipCheck(parseRootPath)
	parseBoundaryFiles := map[string]struct{}{}
	for _, parsePath := range parseImportCheck.Locations {
		parseBoundaryFiles[parsePath] = struct{}{}
	}
	for _, parsePath := range parseStateCheck.Locations {
		parseBoundaryFiles[parsePath] = struct{}{}
	}
	parseReport := inspectOwnershipView{
		ImportBoundaryStatus: parseImportCheck.Status,
		ImportBoundaryNote:   parseImportCheck.Summary,
		StateBoundaryStatus:  parseStateCheck.Status,
		StateBoundaryNote:    parseStateCheck.Summary,
	}
	for parsePath := range parseBoundaryFiles {
		parseReport.BoundaryFiles = append(parseReport.BoundaryFiles, parsePath)
	}
	sort.Strings(parseReport.BoundaryFiles)
	return parseReport, nil
}

// buildInspectFileTypes counts files by extension and top-level directory.
func buildInspectFileTypes(parseRootPath string) (inspectFileTypesView, error) {
	parseExtensionCounts := map[string]int{}
	parseDirectoryCounts := map[string]int{}
	parseWalkErr := filepath.WalkDir(parseRootPath, func(parsePath string, parseEntry fs.DirEntry, parseEntryErr error) error {
		if parseEntryErr != nil {
			return parseEntryErr
		}
		if parsePath == parseRootPath {
			return nil
		}
		if parseEntry.IsDir() {
			if shouldSkipDoctorAuditDir(parseEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		parseRelativePath, parseRelativeErr := filepath.Rel(parseRootPath, parsePath)
		if parseRelativeErr != nil {
			return parseRelativeErr
		}
		parseRelativePath = filepath.ToSlash(parseRelativePath)
		parseExtension := strings.ToLower(strings.TrimSpace(filepath.Ext(parseEntry.Name())))
		if parseExtension == "" {
			parseExtension = "<none>"
		}
		parseExtensionCounts[parseExtension]++
		parseTopDirectory := buildInspectTopDirectory(parseRelativePath)
		parseDirectoryCounts[parseTopDirectory]++
		return nil
	})
	if parseWalkErr != nil {
		return inspectFileTypesView{}, parseWalkErr
	}
	parseReport := inspectFileTypesView{
		ExtensionCounts: buildInspectCountViews(parseExtensionCounts),
		DirectoryCounts: buildInspectCountViews(parseDirectoryCounts),
	}
	for _, parseEntry := range parseReport.ExtensionCounts {
		parseReport.TotalFiles += parseEntry.Count
	}
	return parseReport, nil
}

// buildInspectTopDirectory extracts the first path segment used for directory-level counting.
func buildInspectTopDirectory(parseRelativePath string) string {
	parseSegments := strings.Split(strings.TrimSpace(parseRelativePath), "/")
	if len(parseSegments) == 0 {
		return "."
	}
	if strings.TrimSpace(parseSegments[0]) == "" {
		return "."
	}
	return parseSegments[0]
}

// buildInspectCountViews converts a count map into deterministic sorted output.
func buildInspectCountViews(parseCounts map[string]int) []inspectCountView {
	parseViews := make([]inspectCountView, 0, len(parseCounts))
	for parseName, parseCount := range parseCounts {
		parseViews = append(parseViews, inspectCountView{Name: parseName, Count: parseCount})
	}
	sort.Slice(parseViews, func(parseLeft int, parseRight int) bool {
		if parseViews[parseLeft].Count != parseViews[parseRight].Count {
			return parseViews[parseLeft].Count > parseViews[parseRight].Count
		}
		return parseViews[parseLeft].Name < parseViews[parseRight].Name
	})
	return parseViews
}

// renderInspectSummary prints a concise multi-report inspection summary.
func renderInspectSummary(renderSummary inspectSummary) {
	fmt.Println("GWC inspect")
	fmt.Printf("  root:               %s\n", renderSummary.Root)
	fmt.Printf("  routes:             %d registrations across %d files\n", renderSummary.Routes.Registrations, len(renderSummary.Routes.RouteFiles))
	fmt.Printf("  route lazy split:   %t\n", renderSummary.Routes.HasLazySplit)
	fmt.Printf("  route prerender:    %t\n", renderSummary.Routes.HasPrerenderHint)
	fmt.Printf("  marketing routes:   %t\n", renderSummary.Routes.HasMarketingRoute)
	fmt.Printf("  dependencies:       %d imports (%s)\n", renderSummary.Dependencies.TotalImports, firstNonEmpty(renderSummary.Dependencies.ModulePath, "<unknown module>"))
	fmt.Printf("  ownership imports:  %s\n", firstNonEmpty(renderSummary.Ownership.ImportBoundaryStatus, "unknown"))
	fmt.Printf("  ownership state:    %s\n", firstNonEmpty(renderSummary.Ownership.StateBoundaryStatus, "unknown"))
	fmt.Printf("  file types:         %d files (%d extensions)\n", renderSummary.FileTypes.TotalFiles, len(renderSummary.FileTypes.ExtensionCounts))
	if len(renderSummary.Errors) == 0 {
		return
	}
	fmt.Println("  errors:")
	for _, renderError := range renderSummary.Errors {
		fmt.Printf("    - %s\n", renderError)
	}
}
