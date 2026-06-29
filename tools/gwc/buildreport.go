package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"os"
	"os/exec"
	"path/filepath"
	"sort"
	"time"
)

// runBuildReportCommand routes the `gwc buildreport` build-cache analysis.
var runBuildReportCommand = func(parseL launcher, parseArgs []string) error {
	return parseL.runBuildReport(parseArgs)
}

// buildAction is one node of `go build -debug-actiongraph` output.
type buildAction struct {
	Mode      string `json:"Mode"`
	Package   string `json:"Package"`
	NeedBuild bool   `json:"NeedBuild"`
	TimeStart string `json:"TimeStart"`
	TimeDone  string `json:"TimeDone"`
}

// packageBuild is the per-package outcome in a build report.
type packageBuild struct {
	Package    string  `json:"package"`
	Rebuilt    bool    `json:"rebuilt"`
	DurationMs float64 `json:"durationMs"`
}

// buildReport is the "what was rebuilt and why" summary derived from the action graph: the
// honest cache-miss picture the C2 platform-honest target asks for (NeedBuild is the actual
// cache-miss signal, not a vague changed-file list).
type buildReport struct {
	TotalPackages int            `json:"totalPackages"`
	Rebuilt       int            `json:"rebuilt"`
	Cached        int            `json:"cached"`
	Slowest       []packageBuild `json:"slowestRebuilt"`
}

// runBuildReport parses `gwc buildreport [-pattern ./...] [-rebuild] [-json]`, runs a build
// capturing its action graph, and prints what was rebuilt vs served from cache.
func (parseL launcher) runBuildReport(parseArgs []string) error {
	parseFlags := flag.NewFlagSet("buildreport", flag.ContinueOnError)
	parseFlags.SetOutput(os.Stdout)
	parsePattern := parseFlags.String("pattern", "./...", "Package pattern to build")
	parseRebuild := parseFlags.Bool("rebuild", false, "Force a full rebuild (cold cache) instead of an incremental one")
	parseJSON := parseFlags.Bool("json", false, "Emit the report as JSON")
	if parseErr := parseFlags.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}

	parseGraphPath := filepath.Join(os.TempDir(), "gwc-actiongraph.json")
	parseElapsed, parseErr := runBuildWithActionGraph(*parsePattern, parseGraphPath, *parseRebuild)
	if parseErr != nil {
		return parseErr
	}

	parseData, parseErr := os.ReadFile(parseGraphPath)
	if parseErr != nil {
		return fmt.Errorf("read action graph: %w", parseErr)
	}
	parseActions, parseErr := parseActionGraph(parseData)
	if parseErr != nil {
		return parseErr
	}
	parseReport := summarizeActionGraph(parseActions)

	if *parseJSON {
		parseOut, _ := json.MarshalIndent(parseReport, "", "  ")
		fmt.Println(string(parseOut))
		return nil
	}

	parseCacheKind := "warm"
	if *parseRebuild {
		parseCacheKind = "cold"
	}
	fmt.Printf("GWC buildreport (%s cache): %d package(s) — %d rebuilt, %d cached — wall %s\n",
		parseCacheKind, parseReport.TotalPackages, parseReport.Rebuilt, parseReport.Cached, parseElapsed.Round(time.Millisecond))
	if parseReport.Rebuilt == 0 {
		fmt.Println("  nothing rebuilt — fully served from the build cache")
		return nil
	}
	fmt.Println("  slowest rebuilt packages (cache miss = changed or a dependency changed):")
	for _, parsePkg := range parseReport.Slowest {
		fmt.Printf("    %7.1fms  %s\n", parsePkg.DurationMs, parsePkg.Package)
	}
	return nil
}

// runBuildWithActionGraph runs the build, writing its action graph to graphPath, and
// returns the wall-clock build time.
func runBuildWithActionGraph(parsePattern, parseGraphPath string, parseRebuild bool) (time.Duration, error) {
	parseBuildArgs := []string{"build", "-debug-actiongraph=" + parseGraphPath}
	if parseRebuild {
		parseBuildArgs = append(parseBuildArgs, "-a")
	}
	parseBuildArgs = append(parseBuildArgs, parsePattern)

	parseCmd := exec.Command("go", parseBuildArgs...)
	parseStart := time.Now()
	parseOutput, parseErr := parseCmd.CombinedOutput()
	parseElapsed := time.Since(parseStart)
	if parseErr != nil {
		return parseElapsed, fmt.Errorf("build failed: %w\n%s", parseErr, parseOutput)
	}
	return parseElapsed, nil
}

// parseActionGraph decodes the action-graph JSON array.
func parseActionGraph(parseData []byte) ([]buildAction, error) {
	var parseActions []buildAction
	if parseErr := json.Unmarshal(parseData, &parseActions); parseErr != nil {
		return nil, fmt.Errorf("parse action graph: %w", parseErr)
	}
	return parseActions, nil
}

// summarizeActionGraph classifies the build actions into rebuilt vs cached packages and
// ranks the rebuilt ones by compile time. Only real package "build" actions are counted
// (skipping link/run/std-meta actions and the synthetic root).
func summarizeActionGraph(parseActions []buildAction) buildReport {
	var parseRebuilt []packageBuild
	parseReport := buildReport{}
	for _, parseAction := range parseActions {
		if parseAction.Mode != "build" || parseAction.Package == "" || parseAction.Package == "command-line-arguments" {
			continue
		}
		parseReport.TotalPackages++
		if !parseAction.NeedBuild {
			parseReport.Cached++
			continue
		}
		parseReport.Rebuilt++
		parseRebuilt = append(parseRebuilt, packageBuild{
			Package:    parseAction.Package,
			Rebuilt:    true,
			DurationMs: actionDurationMs(parseAction),
		})
	}

	sort.Slice(parseRebuilt, func(parseA, parseB int) bool {
		return parseRebuilt[parseA].DurationMs > parseRebuilt[parseB].DurationMs
	})
	if len(parseRebuilt) > 15 {
		parseRebuilt = parseRebuilt[:15]
	}
	parseReport.Slowest = parseRebuilt
	return parseReport
}

// actionDurationMs returns an action's compile time in milliseconds, or 0 when timestamps
// are absent (cached actions).
func actionDurationMs(parseAction buildAction) float64 {
	if parseAction.TimeStart == "" || parseAction.TimeDone == "" {
		return 0
	}
	parseStart, parseErrStart := time.Parse(time.RFC3339Nano, parseAction.TimeStart)
	parseDone, parseErrDone := time.Parse(time.RFC3339Nano, parseAction.TimeDone)
	if parseErrStart != nil || parseErrDone != nil {
		return 0
	}
	return float64(parseDone.Sub(parseStart).Microseconds()) / 1000.0
}
