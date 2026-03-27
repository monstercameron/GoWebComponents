//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"fmt"
	"math"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

type example201ScenarioLabel struct {
	GetScenarioID       string `json:"getScenarioID"`
	GetScenarioLabel    string `json:"getScenarioLabel"`
	GetCategory         string `json:"getCategory"`
	GetRequestedWork    string `json:"getRequestedWork"`
	GetCorrectnessCheck string `json:"getCorrectnessCheck"`
	GetFinishLine       string `json:"getFinishLine"`
}

type example201ScenarioResult struct {
	GetFramework                 string    `json:"getFramework"`
	GetScenarioID                string    `json:"getScenarioID"`
	GetScenarioLabel             string    `json:"getScenarioLabel"`
	GetCategory                  string    `json:"getCategory"`
	GetRequestedWork             string    `json:"getRequestedWork"`
	GetCorrectnessCheck          string    `json:"getCorrectnessCheck"`
	GetFinishLine                string    `json:"getFinishLine"`
	GetIterationCount            int       `json:"getIterationCount"`
	GetDomReadyMeanMs            float64   `json:"getDomReadyMeanMs"`
	GetDomReadyMedianMs          float64   `json:"getDomReadyMedianMs"`
	GetDomReadyMinMs             float64   `json:"getDomReadyMinMs"`
	GetDomReadyMaxMs             float64   `json:"getDomReadyMaxMs"`
	GetDomReadyP95Ms             float64   `json:"getDomReadyP95Ms"`
	GetDomReadySamplesMs         []float64 `json:"getDomReadySamplesMs"`
	GetPaintVisibleMeanMs        float64   `json:"getPaintVisibleMeanMs"`
	GetPaintVisibleMedianMs      float64   `json:"getPaintVisibleMedianMs"`
	GetPaintVisibleMinMs         float64   `json:"getPaintVisibleMinMs"`
	GetPaintVisibleMaxMs         float64   `json:"getPaintVisibleMaxMs"`
	GetPaintVisibleP95Ms         float64   `json:"getPaintVisibleP95Ms"`
	GetPaintVisibleSamplesMs     []float64 `json:"getPaintVisibleSamplesMs"`
	GetPaintAfterDomMeanMs       float64   `json:"getPaintAfterDomMeanMs"`
	GetPaintAfterDomMedianMs     float64   `json:"getPaintAfterDomMedianMs"`
	GetPaintAfterDomP95Ms        float64   `json:"getPaintAfterDomP95Ms"`
	GetMutationRecordMean        float64   `json:"getMutationRecordMean"`
	GetChildListMutationMean     float64   `json:"getChildListMutationMean"`
	GetAttributeMutationMean     float64   `json:"getAttributeMutationMean"`
	GetCharacterDataMutationMean float64   `json:"getCharacterDataMutationMean"`
	GetAddedNodeMean             float64   `json:"getAddedNodeMean"`
	GetRemovedNodeMean           float64   `json:"getRemovedNodeMean"`
	GetLongTaskCountMean         float64   `json:"getLongTaskCountMean"`
	GetLongTaskDurationMeanMs    float64   `json:"getLongTaskDurationMeanMs"`
	HasHeapDelta                 bool      `json:"hasHeapDelta"`
	GetHeapDeltaMeanBytes        float64   `json:"getHeapDeltaMeanBytes"`
	GetHeapDeltaMedianBytes      float64   `json:"getHeapDeltaMedianBytes"`
	GetHeapDeltaP95Bytes         float64   `json:"getHeapDeltaP95Bytes"`
}

type example201FrameworkResult struct {
	GetFramework       string                     `json:"getFramework"`
	GetLabel           string                     `json:"getLabel"`
	GetScenarioResults []example201ScenarioResult `json:"getScenarioResults"`
}

type example201Report struct {
	GeneratedAt       string                      `json:"generatedAt"`
	GetIterations     int                         `json:"getIterations"`
	GetWarmups        int                         `json:"getWarmups"`
	GetSeed           int                         `json:"getSeed"`
	GetScenarioOrder  []string                    `json:"getScenarioOrder"`
	GetFrameworkOrder []string                    `json:"getFrameworkOrder"`
	GetScenarioLabels []example201ScenarioLabel   `json:"getScenarioLabels"`
	GetFrameworks     []example201FrameworkResult `json:"getFrameworks"`
}

type example201Artifact struct {
	GetGeneratedAt string           `json:"getGeneratedAt"`
	GetBrowser     string           `json:"getBrowser"`
	GetUserAgent   string           `json:"getUserAgent"`
	GetBaseURL     string           `json:"getBaseURL"`
	GetRoute       string           `json:"getRoute"`
	GetReport      example201Report `json:"getReport"`
}

type example201ScenarioRow struct {
	GetScenarioID       string
	GetScenarioLabel    string
	GetCategory         string
	GetRequestedWork    string
	GetCorrectnessCheck string
	GetFinishLine       string
	GetFrameworks       []example201ScenarioCell
}

type example201ScenarioCell struct {
	GetFramework                 string
	GetLabel                     string
	GetDomReadyMeanMs            float64
	GetDomReadyMedianMs          float64
	GetPaintVisibleMeanMs        float64
	GetPaintVisibleMedianMs      float64
	GetPaintAfterDomMeanMs       float64
	GetChildListMutationMean     float64
	GetAttributeMutationMean     float64
	GetCharacterDataMutationMean float64
	GetAddedNodeMean             float64
	GetRemovedNodeMean           float64
	GetLongTaskCountMean         float64
	GetLongTaskDurationMeanMs    float64
	GetRelativePaintVisible      float64
}

type example201CategoryRow struct {
	GetCategory      string
	GetScenarioCount int
	GetFrameworks    []example201CategoryCell
}

type example201CategoryCell struct {
	GetFramework                     string
	GetLabel                         string
	GetScenarioCount                 int
	GetScenarioWins                  int
	GetDomReadyMeanMs                float64
	GetPaintVisibleMeanMs            float64
	GetPaintVisibleGeometricRelative float64
}

// buildExample201BenchmarkWasm builds the shared Go benchmark subject through gwc into the examples wasm output directory.
func buildExample201BenchmarkWasm(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	getOutputPath := filepath.Join(parseRepoRoot, "bin", "examples", "render-benchmark.wasm")
	getCommand := exec.Command(
		"go",
		"run", "./tools/gwc", "build",
		"-app", "./examples/201-render-benchmark/main.go",
		"-root", "./examples/201-render-benchmark",
		"-out", "./bin/examples/render-benchmark.wasm",
	)
	getCommand.Dir = parseRepoRoot
	if getOutput, parseErr := getCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build example 201 wasm: %v\n%s", parseErr, string(getOutput))
	}
	return getOutputPath
}

// buildExample201BenchmarkWorkerWasm builds the Go WASM worker used by the runtime2 benchmark subjects.
func buildExample201BenchmarkWorkerWasm(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	getOutputPath := filepath.Join(parseRepoRoot, "bin", "examples", "render-benchmark-worker.wasm")
	getCommand := exec.Command("go", "build", "-o", getOutputPath, "./examples/201-render-benchmark/backgroundworker")
	getCommand.Dir = parseRepoRoot
	getCommand.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if getOutput, parseErr := getCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build example 201 worker wasm: %v\n%s", parseErr, string(getOutput))
	}
	return getOutputPath
}

// buildExample201Artifact reads the browser benchmark report from the example page and wraps it with browser metadata.
func buildExample201Artifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example201Artifact {
	parseT.Helper()
	getRoute := "/examples/201-render-benchmark/?iterations=7&warmups=2&seed=20101"
	if _, parseErr := parsePage.Goto(parseBaseURL+getRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto example 201 benchmark: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction("() => !!window.__example201Runner", nil); parseErr != nil {
		parseT.Fatalf("wait for benchmark runner: %v", parseErr)
	}
	getReportJSONValue, parseErr := parsePage.Evaluate("async () => JSON.stringify(await window.__example201Runner.runBenchmarks())")
	if parseErr != nil {
		parseT.Fatalf("run benchmark report: %v", parseErr)
	}
	getReportJSON, hasReportJSON := getReportJSONValue.(string)
	if !hasReportJSON || strings.TrimSpace(getReportJSON) == "" {
		parseT.Fatalf("expected benchmark report json string, got %#v", getReportJSONValue)
	}
	var getReport example201Report
	if parseErr2 := json.Unmarshal([]byte(getReportJSON), &getReport); parseErr2 != nil {
		parseT.Fatalf("decode benchmark report: %v\n%s", parseErr2, getReportJSON)
	}
	if len(getReport.GetFrameworks) != 4 {
		parseT.Fatalf("expected four framework results, got %#v", getReport.GetFrameworks)
	}
	getUserAgentValue, parseErr := parsePage.Evaluate("() => navigator.userAgent")
	if parseErr != nil {
		parseT.Fatalf("read user agent: %v", parseErr)
	}
	getUserAgent, hasUserAgent := getUserAgentValue.(string)
	if !hasUserAgent {
		getUserAgent = fmt.Sprintf("%v", getUserAgentValue)
	}
	return example201Artifact{
		GetGeneratedAt: time.Now().UTC().Format(time.RFC3339),
		GetBrowser:     "chromium",
		GetUserAgent:   getUserAgent,
		GetBaseURL:     parseBaseURL,
		GetRoute:       getRoute,
		GetReport:      getReport,
	}
}

// buildExample201ScenarioRows converts the raw framework-first payload into scenario-first comparison rows.
func buildExample201ScenarioRows(parseArtifact example201Artifact) []example201ScenarioRow {
	getRows := make([]example201ScenarioRow, 0, len(parseArtifact.GetReport.GetScenarioLabels))
	for _, getScenarioLabel := range parseArtifact.GetReport.GetScenarioLabels {
		getRow := example201ScenarioRow{
			GetScenarioID:       getScenarioLabel.GetScenarioID,
			GetScenarioLabel:    getScenarioLabel.GetScenarioLabel,
			GetCategory:         getScenarioLabel.GetCategory,
			GetRequestedWork:    getScenarioLabel.GetRequestedWork,
			GetCorrectnessCheck: getScenarioLabel.GetCorrectnessCheck,
			GetFinishLine:       getScenarioLabel.GetFinishLine,
		}
		getFastestPaintVisible := 0.0
		for _, getFramework := range parseArtifact.GetReport.GetFrameworks {
			for _, getScenario := range getFramework.GetScenarioResults {
				if getScenario.GetScenarioID != getScenarioLabel.GetScenarioID {
					continue
				}
				getCell := example201ScenarioCell{
					GetFramework:                 getFramework.GetFramework,
					GetLabel:                     getFramework.GetLabel,
					GetDomReadyMeanMs:            getScenario.GetDomReadyMeanMs,
					GetDomReadyMedianMs:          getScenario.GetDomReadyMedianMs,
					GetPaintVisibleMeanMs:        getScenario.GetPaintVisibleMeanMs,
					GetPaintVisibleMedianMs:      getScenario.GetPaintVisibleMedianMs,
					GetPaintAfterDomMeanMs:       getScenario.GetPaintAfterDomMeanMs,
					GetChildListMutationMean:     getScenario.GetChildListMutationMean,
					GetAttributeMutationMean:     getScenario.GetAttributeMutationMean,
					GetCharacterDataMutationMean: getScenario.GetCharacterDataMutationMean,
					GetAddedNodeMean:             getScenario.GetAddedNodeMean,
					GetRemovedNodeMean:           getScenario.GetRemovedNodeMean,
					GetLongTaskCountMean:         getScenario.GetLongTaskCountMean,
					GetLongTaskDurationMeanMs:    getScenario.GetLongTaskDurationMeanMs,
				}
				if getFastestPaintVisible == 0 || getScenario.GetPaintVisibleMeanMs < getFastestPaintVisible {
					getFastestPaintVisible = getScenario.GetPaintVisibleMeanMs
				}
				getRow.GetFrameworks = append(getRow.GetFrameworks, getCell)
			}
		}
		sort.Slice(getRow.GetFrameworks, func(parseLeft int, parseRight int) bool {
			return getRow.GetFrameworks[parseLeft].GetPaintVisibleMeanMs < getRow.GetFrameworks[parseRight].GetPaintVisibleMeanMs
		})
		for parseIndex := range getRow.GetFrameworks {
			if getFastestPaintVisible == 0 {
				getRow.GetFrameworks[parseIndex].GetRelativePaintVisible = 0
				continue
			}
			getRow.GetFrameworks[parseIndex].GetRelativePaintVisible = getRow.GetFrameworks[parseIndex].GetPaintVisibleMeanMs / getFastestPaintVisible
		}
		getRows = append(getRows, getRow)
	}
	return getRows
}

// buildExample201CategoryRows aggregates category-local paint-visible relative cost and scenario wins.
func buildExample201CategoryRows(parseScenarioRows []example201ScenarioRow) []example201CategoryRow {
	getRowsByCategory := map[string]*example201CategoryRow{}
	getCellsByCategoryAndFramework := map[string]map[string]*example201CategoryCell{}
	for _, getScenarioRow := range parseScenarioRows {
		getCategoryRow := getRowsByCategory[getScenarioRow.GetCategory]
		if getCategoryRow == nil {
			getCategoryRow = &example201CategoryRow{
				GetCategory: getScenarioRow.GetCategory,
			}
			getRowsByCategory[getScenarioRow.GetCategory] = getCategoryRow
			getCellsByCategoryAndFramework[getScenarioRow.GetCategory] = map[string]*example201CategoryCell{}
		}
		getCategoryRow.GetScenarioCount++
		for parseIndex, getFramework := range getScenarioRow.GetFrameworks {
			getFrameworkCell := getCellsByCategoryAndFramework[getScenarioRow.GetCategory][getFramework.GetFramework]
			if getFrameworkCell == nil {
				getFrameworkCell = &example201CategoryCell{
					GetFramework:                     getFramework.GetFramework,
					GetLabel:                         getFramework.GetLabel,
					GetPaintVisibleGeometricRelative: 1,
				}
				getCellsByCategoryAndFramework[getScenarioRow.GetCategory][getFramework.GetFramework] = getFrameworkCell
			}
			getFrameworkCell.GetScenarioCount++
			getFrameworkCell.GetDomReadyMeanMs += getFramework.GetDomReadyMeanMs
			getFrameworkCell.GetPaintVisibleMeanMs += getFramework.GetPaintVisibleMeanMs
			getFrameworkCell.GetPaintVisibleGeometricRelative *= math.Max(getFramework.GetRelativePaintVisible, 0.0001)
			if parseIndex == 0 || getFramework.GetRelativePaintVisible == 1 {
				getFrameworkCell.GetScenarioWins++
			}
		}
	}
	getRows := make([]example201CategoryRow, 0, len(getRowsByCategory))
	for getCategory, getRow := range getRowsByCategory {
		getFrameworkRows := make([]example201CategoryCell, 0, len(getCellsByCategoryAndFramework[getCategory]))
		for _, getFrameworkCell := range getCellsByCategoryAndFramework[getCategory] {
			getFrameworkCell.GetDomReadyMeanMs = getFrameworkCell.GetDomReadyMeanMs / float64(getFrameworkCell.GetScenarioCount)
			getFrameworkCell.GetPaintVisibleMeanMs = getFrameworkCell.GetPaintVisibleMeanMs / float64(getFrameworkCell.GetScenarioCount)
			getFrameworkCell.GetPaintVisibleGeometricRelative = math.Pow(getFrameworkCell.GetPaintVisibleGeometricRelative, 1/float64(getFrameworkCell.GetScenarioCount))
			getFrameworkRows = append(getFrameworkRows, *getFrameworkCell)
		}
		sort.Slice(getFrameworkRows, func(parseLeft int, parseRight int) bool {
			if getFrameworkRows[parseLeft].GetPaintVisibleGeometricRelative == getFrameworkRows[parseRight].GetPaintVisibleGeometricRelative {
				return getFrameworkRows[parseLeft].GetFramework < getFrameworkRows[parseRight].GetFramework
			}
			return getFrameworkRows[parseLeft].GetPaintVisibleGeometricRelative < getFrameworkRows[parseRight].GetPaintVisibleGeometricRelative
		})
		getRow.GetFrameworks = getFrameworkRows
		getRows = append(getRows, *getRow)
	}
	sort.Slice(getRows, func(parseLeft int, parseRight int) bool {
		return getRows[parseLeft].GetCategory < getRows[parseRight].GetCategory
	})
	return getRows
}

// formatExample201BenchmarkMarkdown renders the browser benchmark artifact into a human-readable Markdown report.
func formatExample201BenchmarkMarkdown(parseArtifact example201Artifact) string {
	var getBuilder strings.Builder
	getScenarioRows := buildExample201ScenarioRows(parseArtifact)
	getCategoryRows := buildExample201CategoryRows(getScenarioRows)
	getBuilder.WriteString("# Example 201 Browser Benchmark Report\n\n")
	getBuilder.WriteString(fmt.Sprintf("- Generated at: `%s`\n", parseArtifact.GetGeneratedAt))
	getBuilder.WriteString(fmt.Sprintf("- Browser: `%s`\n", parseArtifact.GetBrowser))
	getBuilder.WriteString(fmt.Sprintf("- Route: `%s`\n", parseArtifact.GetRoute))
	getBuilder.WriteString(fmt.Sprintf("- Iterations: `%d`\n", parseArtifact.GetReport.GetIterations))
	getBuilder.WriteString(fmt.Sprintf("- Warmups: `%d`\n", parseArtifact.GetReport.GetWarmups))
	getBuilder.WriteString(fmt.Sprintf("- Seed: `%d`\n", parseArtifact.GetReport.GetSeed))
	getBuilder.WriteString(fmt.Sprintf("- Scenario order: `%s`\n", strings.Join(parseArtifact.GetReport.GetScenarioOrder, ", ")))
	getBuilder.WriteString(fmt.Sprintf("- Framework order: `%s`\n", strings.Join(parseArtifact.GetReport.GetFrameworkOrder, ", ")))
	getBuilder.WriteString("- Important boundary: the `runtime2` subjects here still keep DOM ownership on the main thread.\n")
	getBuilder.WriteString("- Worker note: `Runtime 2` opens one Go WASM worker and `Runtime 2 (4 Workers)` opens four Go WASM workers to prepare core and content chunks before the local runtime2 shell commits DOM updates.\n")
	getBuilder.WriteString("- Non-worker note: deep-tree and hook-grid scenarios remain main-thread-owned today, so the worker-backed benefit is expected to concentrate in the core and content scenarios.\n")
	getBuilder.WriteString("- React subject note: the page uses vendored React 18 UMD files under `examples/201-render-benchmark/vendor/`, so the comparison stays local to the repo server.\n\n")
	getBuilder.WriteString("- Finish lines: `DOM Ready` means the scenario correctness contract became true. `Paint Proxy` means one `requestAnimationFrame` boundary after the DOM-ready checkpoint.\n")
	getBuilder.WriteString("- Primary comparison: category summaries and scenario tables use `Paint Proxy` as the lead timing. `DOM Ready` is reported alongside it so compute-or-DOM readiness stays visible.\n")
	getBuilder.WriteString("- Ratio discipline: there is no single mixed overall multiplier across all scenarios. Category-local geometric-relative scores are used instead.\n\n")
	getBuilder.WriteString("## Category Summary\n\n")
	getBuilder.WriteString("| Category | Framework | Avg DOM Ready | Avg Paint Proxy | Geom. Relative | Wins |\n")
	getBuilder.WriteString("| --- | --- | ---: | ---: | ---: | ---: |\n")
	for _, getCategoryRow := range getCategoryRows {
		for _, getFramework := range getCategoryRow.GetFrameworks {
			getBuilder.WriteString(fmt.Sprintf(
				"| %s | %s | %.3f ms | %.3f ms | %.3fx | %d |\n",
				getCategoryRow.GetCategory,
				getFramework.GetLabel,
				getFramework.GetDomReadyMeanMs,
				getFramework.GetPaintVisibleMeanMs,
				getFramework.GetPaintVisibleGeometricRelative,
				getFramework.GetScenarioWins,
			))
		}
	}
	getBuilder.WriteString("\n## Scenarios\n")
	for _, getScenarioRow := range getScenarioRows {
		getBuilder.WriteString(fmt.Sprintf("\n### %s\n\n", getScenarioRow.GetScenarioLabel))
		getBuilder.WriteString(fmt.Sprintf("- Category: `%s`\n", getScenarioRow.GetCategory))
		getBuilder.WriteString(fmt.Sprintf("- Requested work: %s\n", getScenarioRow.GetRequestedWork))
		getBuilder.WriteString(fmt.Sprintf("- Correctness check: %s\n", getScenarioRow.GetCorrectnessCheck))
		getBuilder.WriteString(fmt.Sprintf("- Finish line: %s\n\n", getScenarioRow.GetFinishLine))
		getBuilder.WriteString("| Framework | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Long Tasks | Relative Paint |\n")
		getBuilder.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
		for _, getFramework := range getScenarioRow.GetFrameworks {
			getBuilder.WriteString(fmt.Sprintf(
				"| %s | %.3f ms | %.3f ms | %.3f ms | %.3f ms | %.3f ms | %.1f / %.1f / %.1f | %.1f / %.1f | %.1f / %.3f ms | %.3fx |\n",
				getFramework.GetLabel,
				getFramework.GetDomReadyMeanMs,
				getFramework.GetDomReadyMedianMs,
				getFramework.GetPaintVisibleMeanMs,
				getFramework.GetPaintVisibleMedianMs,
				getFramework.GetPaintAfterDomMeanMs,
				getFramework.GetChildListMutationMean,
				getFramework.GetAttributeMutationMean,
				getFramework.GetCharacterDataMutationMean,
				getFramework.GetAddedNodeMean,
				getFramework.GetRemovedNodeMean,
				getFramework.GetLongTaskCountMean,
				getFramework.GetLongTaskDurationMeanMs,
				getFramework.GetRelativePaintVisible,
			))
		}
	}
	return getBuilder.String()
}

// storeExample201Artifact writes the JSON and Markdown benchmark reports under the repo-local browser test results directory.
func storeExample201Artifact(parseT *testing.T, parseRepoRoot string, parseArtifact example201Artifact) (string, string) {
	parseT.Helper()
	getOutputDir := filepath.Join(parseRepoRoot, "bin", "test-results", "example-201-browser-benchmark")
	if parseErr := os.MkdirAll(getOutputDir, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir example 201 report dir: %v", parseErr)
	}
	getJSONPath := filepath.Join(getOutputDir, "browser-benchmark-report.json")
	getMarkdownPath := filepath.Join(getOutputDir, "browser-benchmark-report.md")
	getJSONBytes, parseErr := json.MarshalIndent(parseArtifact, "", "  ")
	if parseErr != nil {
		parseT.Fatalf("marshal example 201 report: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(getJSONPath, getJSONBytes, 0o644); parseErr2 != nil {
		parseT.Fatalf("write example 201 json report: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(getMarkdownPath, []byte(formatExample201BenchmarkMarkdown(parseArtifact)), 0o644); parseErr3 != nil {
		parseT.Fatalf("write example 201 markdown report: %v", parseErr3)
	}
	return getJSONPath, getMarkdownPath
}

// TestExample201BrowserBenchmarkReport builds the benchmark subject, runs the browser comparison through gwc examples, and writes a local report.
func TestExample201BrowserBenchmarkReport(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	buildExample201BenchmarkWorkerWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18101")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		getArtifact := buildExample201Artifact(parseT, parsePage, getBaseURL)
		getJSONPath, getMarkdownPath := storeExample201Artifact(parseT, getRepoRoot, getArtifact)
		parseT.Logf("example 201 benchmark json report: %s", getJSONPath)
		parseT.Logf("example 201 benchmark markdown report: %s", getMarkdownPath)
	})
}
