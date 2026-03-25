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
	buildSummary := buildInspectSummary(parseConfig)
	if parseConfig.json {
		renderEncoder := json.NewEncoder(os.Stdout)
		renderEncoder.SetIndent("", "  ")
		return renderEncoder.Encode(buildSummary)
	}
	renderInspectSummary(buildSummary)
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
func buildInspectSummary(buildConfig inspectConfig) inspectSummary {
	buildSummary := inspectSummary{
		OK:   true,
		Root: buildConfig.rootPath,
	}

	buildRoutes, buildRouteErr := buildInspectRoutes(buildConfig.rootPath)
	if buildRouteErr != nil {
		buildSummary.OK = false
		buildSummary.Errors = append(buildSummary.Errors, fmt.Sprintf("routes: %v", buildRouteErr))
	} else {
		buildSummary.Routes = buildRoutes
	}

	buildDependencies, buildDependencyErr := buildInspectDependencies(buildConfig.rootPath)
	if buildDependencyErr != nil {
		buildSummary.OK = false
		buildSummary.Errors = append(buildSummary.Errors, fmt.Sprintf("dependencies: %v", buildDependencyErr))
	} else {
		buildSummary.Dependencies = buildDependencies
	}

	buildOwnership, buildOwnershipErr := buildInspectOwnership(buildConfig.rootPath)
	if buildOwnershipErr != nil {
		buildSummary.OK = false
		buildSummary.Errors = append(buildSummary.Errors, fmt.Sprintf("ownership: %v", buildOwnershipErr))
	} else {
		buildSummary.Ownership = buildOwnership
	}

	buildFileTypes, buildFileTypeErr := buildInspectFileTypes(buildConfig.rootPath)
	if buildFileTypeErr != nil {
		buildSummary.OK = false
		buildSummary.Errors = append(buildSummary.Errors, fmt.Sprintf("file-types: %v", buildFileTypeErr))
	} else {
		buildSummary.FileTypes = buildFileTypes
	}

	return buildSummary
}

// buildInspectRoutes summarizes route registration and delivery shape signals.
func buildInspectRoutes(buildRootPath string) (inspectRouteReport, error) {
	buildFiles, buildErr := collectGoldenPathGoFiles(buildRootPath)
	if buildErr != nil {
		return inspectRouteReport{}, buildErr
	}
	buildReport := inspectRouteReport{}
	buildRouteFiles := map[string]struct{}{}
	for _, buildFile := range buildFiles {
		buildHits := strings.Count(buildFile.Content, "MustDefineRoute(")
		buildHits += strings.Count(buildFile.Content, "router.Register(")
		if buildHits > 0 {
			buildReport.Registrations += buildHits
			buildRouteFiles[buildFile.RelPath] = struct{}{}
		}
		if strings.Contains(buildFile.Content, "ui.Lazy(") || strings.Contains(buildFile.Content, "ui.CreateElement(ui.Lazy") {
			buildReport.HasLazySplit = true
		}
		buildLowered := strings.ToLower(buildFile.Content)
		if strings.Contains(buildLowered, "prerender") || strings.Contains(buildLowered, "static shell") {
			buildReport.HasPrerenderHint = true
		}
		if strings.Contains(buildFile.Content, `"/pricing"`) ||
			strings.Contains(buildFile.Content, `"/capabilities"`) ||
			strings.Contains(buildFile.Content, `"/about"`) ||
			strings.Contains(buildFile.Content, `"/docs"`) {
			buildReport.HasMarketingRoute = true
		}
	}
	for buildFile := range buildRouteFiles {
		buildReport.RouteFiles = append(buildReport.RouteFiles, buildFile)
	}
	sort.Strings(buildReport.RouteFiles)
	return buildReport, nil
}

// buildInspectDependencies resolves module identity and third-party dependency imports.
func buildInspectDependencies(buildRootPath string) (inspectDependencyView, error) {
	buildModuleCommand := exec.Command("go", "list", "-m")
	buildModuleCommand.Dir = buildRootPath
	buildModuleOutput, buildModuleErr := buildModuleCommand.CombinedOutput()
	if buildModuleErr != nil {
		return inspectDependencyView{}, fmt.Errorf("run go list -m: %w", buildModuleErr)
	}

	buildDependencyCommand := exec.Command("go", "list", "-deps", "-f", "{{if not .Standard}}{{.ImportPath}}{{end}}", "./...")
	buildDependencyCommand.Dir = buildRootPath
	buildDependencyOutput, buildDependencyErr := buildDependencyCommand.CombinedOutput()
	if buildDependencyErr != nil {
		return inspectDependencyView{}, fmt.Errorf("run go list -deps: %w", buildDependencyErr)
	}

	buildDependencySet := map[string]struct{}{}
	for _, buildLine := range strings.Split(string(buildDependencyOutput), "\n") {
		buildImportPath := strings.TrimSpace(buildLine)
		if buildImportPath == "" {
			continue
		}
		buildDependencySet[buildImportPath] = struct{}{}
	}
	buildDependencies := make([]string, 0, len(buildDependencySet))
	for buildImportPath := range buildDependencySet {
		buildDependencies = append(buildDependencies, buildImportPath)
	}
	sort.Strings(buildDependencies)

	buildReport := inspectDependencyView{
		ModulePath:   strings.TrimSpace(string(buildModuleOutput)),
		TotalImports: len(buildDependencies),
	}
	if len(buildDependencies) > 20 {
		buildReport.SampleImports = append(buildReport.SampleImports, buildDependencies[:20]...)
		buildReport.HasSampleLimit = true
		return buildReport, nil
	}
	buildReport.SampleImports = buildDependencies
	return buildReport, nil
}

// buildInspectOwnership mirrors doctor ownership checks in one inspect-friendly summary.
func buildInspectOwnership(buildRootPath string) (inspectOwnershipView, error) {
	buildImportCheck := buildDoctorOwnershipBoundaryCheck(buildRootPath)
	buildStateCheck := buildDoctorStateOwnershipCheck(buildRootPath)
	buildBoundaryFiles := map[string]struct{}{}
	for _, buildPath := range buildImportCheck.Locations {
		buildBoundaryFiles[buildPath] = struct{}{}
	}
	for _, buildPath := range buildStateCheck.Locations {
		buildBoundaryFiles[buildPath] = struct{}{}
	}
	buildReport := inspectOwnershipView{
		ImportBoundaryStatus: buildImportCheck.Status,
		ImportBoundaryNote:   buildImportCheck.Summary,
		StateBoundaryStatus:  buildStateCheck.Status,
		StateBoundaryNote:    buildStateCheck.Summary,
	}
	for buildPath := range buildBoundaryFiles {
		buildReport.BoundaryFiles = append(buildReport.BoundaryFiles, buildPath)
	}
	sort.Strings(buildReport.BoundaryFiles)
	return buildReport, nil
}

// buildInspectFileTypes counts files by extension and top-level directory.
func buildInspectFileTypes(buildRootPath string) (inspectFileTypesView, error) {
	buildExtensionCounts := map[string]int{}
	buildDirectoryCounts := map[string]int{}
	buildWalkErr := filepath.WalkDir(buildRootPath, func(buildPath string, buildEntry fs.DirEntry, buildEntryErr error) error {
		if buildEntryErr != nil {
			return buildEntryErr
		}
		if buildPath == buildRootPath {
			return nil
		}
		if buildEntry.IsDir() {
			if shouldSkipDoctorAuditDir(buildEntry.Name()) {
				return filepath.SkipDir
			}
			return nil
		}
		buildRelativePath, buildRelativeErr := filepath.Rel(buildRootPath, buildPath)
		if buildRelativeErr != nil {
			return buildRelativeErr
		}
		buildRelativePath = filepath.ToSlash(buildRelativePath)
		buildExtension := strings.ToLower(strings.TrimSpace(filepath.Ext(buildEntry.Name())))
		if buildExtension == "" {
			buildExtension = "<none>"
		}
		buildExtensionCounts[buildExtension]++
		buildTopDirectory := buildInspectTopDirectory(buildRelativePath)
		buildDirectoryCounts[buildTopDirectory]++
		return nil
	})
	if buildWalkErr != nil {
		return inspectFileTypesView{}, buildWalkErr
	}
	buildReport := inspectFileTypesView{
		ExtensionCounts: buildInspectCountViews(buildExtensionCounts),
		DirectoryCounts: buildInspectCountViews(buildDirectoryCounts),
	}
	for _, buildEntry := range buildReport.ExtensionCounts {
		buildReport.TotalFiles += buildEntry.Count
	}
	return buildReport, nil
}

// buildInspectTopDirectory extracts the first path segment used for directory-level counting.
func buildInspectTopDirectory(buildRelativePath string) string {
	buildSegments := strings.Split(strings.TrimSpace(buildRelativePath), "/")
	if len(buildSegments) == 0 {
		return "."
	}
	if strings.TrimSpace(buildSegments[0]) == "" {
		return "."
	}
	return buildSegments[0]
}

// buildInspectCountViews converts a count map into deterministic sorted output.
func buildInspectCountViews(buildCounts map[string]int) []inspectCountView {
	buildViews := make([]inspectCountView, 0, len(buildCounts))
	for buildName, buildCount := range buildCounts {
		buildViews = append(buildViews, inspectCountView{Name: buildName, Count: buildCount})
	}
	sort.Slice(buildViews, func(buildLeft int, buildRight int) bool {
		if buildViews[buildLeft].Count != buildViews[buildRight].Count {
			return buildViews[buildLeft].Count > buildViews[buildRight].Count
		}
		return buildViews[buildLeft].Name < buildViews[buildRight].Name
	})
	return buildViews
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
