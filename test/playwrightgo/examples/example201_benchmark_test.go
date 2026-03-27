//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"fmt"
	"math"
	"net/url"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
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
	HasWorkerMetrics             bool      `json:"hasWorkerMetrics"`
	GetWorkerCountMean           float64   `json:"getWorkerCountMean"`
	GetWorkerBatchCountMean      float64   `json:"getWorkerBatchCountMean"`
	GetWorkerBatchMeanMs         float64   `json:"getWorkerBatchMeanMs"`
	GetWorkerPreparedItemsMean   float64   `json:"getWorkerPreparedItemsMean"`
}

type example201FrameworkResult struct {
	GetFramework       string                     `json:"getFramework"`
	GetLabel           string                     `json:"getLabel"`
	GetScenarioResults []example201ScenarioResult `json:"getScenarioResults"`
}

type example201Report struct {
	GeneratedAt                string                      `json:"generatedAt"`
	GetIterations              int                         `json:"getIterations"`
	GetWarmups                 int                         `json:"getWarmups"`
	GetSeed                    int                         `json:"getSeed"`
	GetScenarioOrder           []string                    `json:"getScenarioOrder"`
	GetFrameworkOrder          []string                    `json:"getFrameworkOrder"`
	GetScenarioFrameworkOrders map[string][]string         `json:"getScenarioFrameworkOrders,omitempty"`
	GetScenarioLabels          []example201ScenarioLabel   `json:"getScenarioLabels"`
	GetFrameworks              []example201FrameworkResult `json:"getFrameworks"`
}

type example201Artifact struct {
	GetGeneratedAt string                `json:"getGeneratedAt"`
	GetBrowser     string                `json:"getBrowser"`
	GetUserAgent   string                `json:"getUserAgent"`
	GetBaseURL     string                `json:"getBaseURL"`
	GetRoute       string                `json:"getRoute"`
	GetReport      example201Report      `json:"getReport"`
	GetScalingRun  *example201ScalingRun `json:"getScalingRun,omitempty"`
}

type example201ScalingRun struct {
	GetDescription string           `json:"getDescription"`
	GetRoute       string           `json:"getRoute"`
	GetReport      example201Report `json:"getReport"`
}

type example201ScoreReference struct {
	GetGeneratedAt       string                             `json:"getGeneratedAt"`
	GetRoute             string                             `json:"getRoute"`
	GetReferenceLabel    string                             `json:"getReferenceLabel"`
	GetReferenceBrowser  string                             `json:"getReferenceBrowser"`
	GetScenarioReference []example201ScoreReferenceScenario `json:"getScenarioReference"`
}

type example201ScoreReferenceScenario struct {
	GetScenarioID     string  `json:"getScenarioID"`
	GetScenarioLabel  string  `json:"getScenarioLabel"`
	GetCategory       string  `json:"getCategory"`
	GetDomReadyMeanMs float64 `json:"getDomReadyMeanMs"`
	IsWorkerRelevant  bool    `json:"isWorkerRelevant"`
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
	GetRelativeDomReady          float64
	GetRelativePaintVisible      float64
	HasDomScoreReference         bool
	IsWorkerRelevant             bool
	GetDomReadyReferenceMs       float64
	GetDomReadyScoreFactor       float64
	HasReactBaseline             bool
	GetDomReadyDeltaVsReactMs    float64
	GetDomReadySpeedupVsReact    float64
	GetDomScore                  int
	HasWorkerMetrics             bool
	GetWorkerCountMean           float64
	GetWorkerBatchCountMean      float64
	GetWorkerBatchMeanMs         float64
	GetWorkerPreparedItemsMean   float64
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
	GetScoreScenarioCount            int
	GetScenarioWins                  int
	GetReactScenarioCount            int
	GetDomReadyMeanMs                float64
	GetPaintVisibleMeanMs            float64
	GetDomReadyDeltaVsReactMs        float64
	GetDomReadyGeometricRelative     float64
	GetPaintVisibleGeometricRelative float64
	GetDomReadyGeometricSpeedup      float64
	GetDomScore                      int
	HasReactBaseline                 bool
}

type example201ScalingRow struct {
	GetScenarioID    string
	GetScenarioLabel string
	GetCategory      string
	GetFrameworks    []example201ScalingCell
}

type example201ScalingCell struct {
	GetFramework               string
	GetLabel                   string
	GetWorkerCountMean         float64
	GetWorkerBatchMeanMs       float64
	GetWorkerPreparedItemMean  float64
	GetDomReadyMeanMs          float64
	GetPaintVisibleMeanMs      float64
	GetRelativeDomReadySpeedup float64
	GetRelativeBatchSpeedup    float64
}

type example201OverallScoreCell struct {
	GetFramework          string
	GetLabel              string
	GetScenarioCount      int
	GetScoreScenarioCount int
	GetDomGeometricSpeed  float64
	GetDomScore           int
}

// buildExample201ReactScenarioByID returns the React benchmark row for each scenario ID when the mixed-framework report includes React.
func buildExample201ReactScenarioByID(parseReport example201Report) map[string]example201ScenarioResult {
	getScenarioByID := map[string]example201ScenarioResult{}
	for _, getFramework := range parseReport.GetFrameworks {
		if getFramework.GetFramework != "react" {
			continue
		}
		for _, getScenario := range getFramework.GetScenarioResults {
			getScenarioByID[getScenario.GetScenarioID] = getScenario
		}
		break
	}
	return getScenarioByID
}

// buildExample201IntegerScore rounds one speedup factor into the stable integer score used by the report.
func buildExample201IntegerScore(parseSpeedup float64) int {
	if parseSpeedup <= 0 {
		return 0
	}
	return int(math.Round(parseSpeedup * 100))
}

// buildExample201ScoreReferencePath returns the checked-in Example 201 score reference path.
func buildExample201ScoreReferencePath(parseRepoRoot string) string {
	return filepath.Join(parseRepoRoot, "examples", "201-render-benchmark", "score-reference.json")
}

// loadExample201ScoreReference reads the checked-in Example 201 score reference profile.
func loadExample201ScoreReference(parseT *testing.T, parseRepoRoot string) example201ScoreReference {
	parseT.Helper()
	getReferencePath := buildExample201ScoreReferencePath(parseRepoRoot)
	getReferenceBytes, parseErr := os.ReadFile(getReferencePath)
	if parseErr != nil {
		parseT.Fatalf("read example 201 score reference: %v", parseErr)
	}
	var getReference example201ScoreReference
	if parseErr2 := json.Unmarshal(getReferenceBytes, &getReference); parseErr2 != nil {
		parseT.Fatalf("decode example 201 score reference: %v", parseErr2)
	}
	return getReference
}

// buildExample201ScoreReferenceScenarioByID returns the fixed score reference for each benchmark scenario.
func buildExample201ScoreReferenceScenarioByID(parseReference example201ScoreReference) map[string]example201ScoreReferenceScenario {
	getScenarioByID := map[string]example201ScoreReferenceScenario{}
	for _, getScenario := range parseReference.GetScenarioReference {
		getScenarioByID[getScenario.GetScenarioID] = getScenario
	}
	return getScenarioByID
}

// buildExample201ScoreFactor returns the fixed-reference DOM-ready ratio used for integer scoring.
func buildExample201ScoreFactor(parseReferenceMs float64, parseMeasuredMs float64) float64 {
	if parseReferenceMs <= 0 || parseMeasuredMs <= 0 {
		return 0
	}
	return parseReferenceMs / parseMeasuredMs
}

// buildExample201ExpectedWorkerCounts resolves the runtime2 worker counts implied by one benchmark route.
func buildExample201ExpectedWorkerCounts(parseRoute string) []int {
	getRoute := parseRoute
	if strings.TrimSpace(getRoute) == "" {
		getRoute = "/examples/201-render-benchmark/?iterations=7&warmups=2&seed=20101"
	}
	getParsedRoute, parseErr := url.Parse(getRoute)
	if parseErr != nil {
		return []int{1, 2, 4, 8}
	}
	getQuery := getParsedRoute.Query()
	getCountSet := map[int]struct{}{}
	storeExample201Count := func(parseCount int) {
		if parseCount < 1 {
			return
		}
		getCountSet[parseCount] = struct{}{}
	}
	getWorkerCountsText := strings.TrimSpace(getQuery.Get("runtime2WorkerCounts"))
	getLegacyCountText := strings.TrimSpace(getQuery.Get("runtime2Workers"))
	switch {
	case getWorkerCountsText != "":
		for _, getCountToken := range strings.Split(getWorkerCountsText, ",") {
			getCount, parseCountErr := strconv.Atoi(strings.TrimSpace(getCountToken))
			if parseCountErr != nil {
				continue
			}
			storeExample201Count(getCount)
		}
	case getLegacyCountText != "":
		storeExample201Count(1)
		getCount, parseCountErr := strconv.Atoi(getLegacyCountText)
		if parseCountErr == nil {
			storeExample201Count(getCount)
		}
	default:
		storeExample201Count(1)
		storeExample201Count(2)
		storeExample201Count(4)
		storeExample201Count(8)
	}
	if len(getCountSet) == 0 {
		return []int{1, 2, 4, 8}
	}
	getCounts := make([]int, 0, len(getCountSet))
	for getCount := range getCountSet {
		getCounts = append(getCounts, getCount)
	}
	sort.Ints(getCounts)
	return getCounts
}

// buildExample201ExpectedFrameworkIDs resolves the framework IDs implied by one benchmark route.
func buildExample201ExpectedFrameworkIDs(parseRoute string) []string {
	getRoute := parseRoute
	if strings.TrimSpace(getRoute) == "" {
		getRoute = "/examples/201-render-benchmark/?iterations=7&warmups=2&seed=20101"
	}
	getParsedRoute, parseErr := url.Parse(getRoute)
	if parseErr != nil {
		return []string{"react", "runtime1", "runtime2-workers1", "runtime2-workers2", "runtime2-workers4", "runtime2-workers8"}
	}
	getQuery := getParsedRoute.Query()
	getWorkerCounts := buildExample201ExpectedWorkerCounts(getRoute)
	getFrameworkIDs := make([]string, 0, len(getWorkerCounts)+2)
	if strings.TrimSpace(getQuery.Get("subjectSet")) != "runtime2-scaling" {
		getFrameworkIDs = append(getFrameworkIDs, "react", "runtime1")
	}
	for _, getWorkerCount := range getWorkerCounts {
		getFrameworkIDs = append(getFrameworkIDs, fmt.Sprintf("runtime2-workers%d", getWorkerCount))
	}
	return getFrameworkIDs
}

// buildExample201ScalingRoute returns the RT2-only stress route used to expose worker-count scaling beyond one-frame paint quantization.
func buildExample201ScalingRoute() string {
	return buildExample201ScalingRouteWithDispatch("batch")
}

// buildExample201ScalingRouteWithDispatch returns the RT2-only stress route with one explicit worker-dispatch strategy.
func buildExample201ScalingRouteWithDispatch(parseDispatch string) string {
	getDispatch := strings.TrimSpace(strings.ToLower(parseDispatch))
	if getDispatch == "" {
		getDispatch = "batch"
	}
	getQuery := url.Values{}
	getQuery.Set("iterations", "5")
	getQuery.Set("warmups", "1")
	getQuery.Set("seed", "20101")
	getQuery.Set("subjectSet", "runtime2-scaling")
	getQuery.Set("runtime2WorkerCounts", "1,2,4,8")
	getQuery.Set("runtime2WorkScale", "12")
	getQuery.Set("runtime2Dispatch", getDispatch)
	return "/examples/201-render-benchmark/?" + getQuery.Encode()
}

// buildExample201OneAndFourWorkerRoute returns the mixed-framework route limited to React, runtime1, RT2x1, and RT2x4.
func buildExample201OneAndFourWorkerRoute() string {
	return "/examples/201-render-benchmark/?iterations=7&warmups=2&seed=20101&runtime2WorkerCounts=1,4"
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

// buildExample201Artifact reads the browser benchmark report from one example route and wraps it with browser metadata.
func buildExample201Artifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseRoute string) example201Artifact {
	parseT.Helper()
	getRoute := parseRoute
	if strings.TrimSpace(getRoute) == "" {
		getRoute = "/examples/201-render-benchmark/?iterations=7&warmups=2&seed=20101"
	}
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
	getExpectedFrameworkIDs := buildExample201ExpectedFrameworkIDs(getRoute)
	if len(getReport.GetFrameworks) != len(getExpectedFrameworkIDs) {
		parseT.Fatalf("expected %d framework results, got %#v", len(getExpectedFrameworkIDs), getReport.GetFrameworks)
	}
	getFrameworkSet := map[string]struct{}{}
	for _, getFramework := range getReport.GetFrameworks {
		getFrameworkSet[getFramework.GetFramework] = struct{}{}
	}
	for _, getFrameworkID := range getExpectedFrameworkIDs {
		if _, hasFramework := getFrameworkSet[getFrameworkID]; !hasFramework {
			parseT.Fatalf("expected framework %q in report, got %#v", getFrameworkID, getReport.GetFrameworkOrder)
		}
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

// buildExample201ScenarioRowsFromReport converts the raw framework-first payload into scenario-first comparison rows.
func buildExample201ScenarioRowsFromReport(parseReport example201Report, parseReference example201ScoreReference) []example201ScenarioRow {
	getReactScenarioByID := buildExample201ReactScenarioByID(parseReport)
	getReferenceScenarioByID := buildExample201ScoreReferenceScenarioByID(parseReference)
	getRows := make([]example201ScenarioRow, 0, len(parseReport.GetScenarioLabels))
	for _, getScenarioLabel := range parseReport.GetScenarioLabels {
		getRow := example201ScenarioRow{
			GetScenarioID:       getScenarioLabel.GetScenarioID,
			GetScenarioLabel:    getScenarioLabel.GetScenarioLabel,
			GetCategory:         getScenarioLabel.GetCategory,
			GetRequestedWork:    getScenarioLabel.GetRequestedWork,
			GetCorrectnessCheck: getScenarioLabel.GetCorrectnessCheck,
			GetFinishLine:       getScenarioLabel.GetFinishLine,
		}
		getReactScenario, hasReactScenario := getReactScenarioByID[getScenarioLabel.GetScenarioID]
		getReferenceScenario, hasReferenceScenario := getReferenceScenarioByID[getScenarioLabel.GetScenarioID]
		getFastestDomReady := 0.0
		getFastestPaintVisible := 0.0
		for _, getFramework := range parseReport.GetFrameworks {
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
					HasWorkerMetrics:             getScenario.HasWorkerMetrics,
					GetWorkerCountMean:           getScenario.GetWorkerCountMean,
					GetWorkerBatchCountMean:      getScenario.GetWorkerBatchCountMean,
					GetWorkerBatchMeanMs:         getScenario.GetWorkerBatchMeanMs,
					GetWorkerPreparedItemsMean:   getScenario.GetWorkerPreparedItemsMean,
				}
				if hasReferenceScenario {
					getCell.HasDomScoreReference = getReferenceScenario.GetDomReadyMeanMs > 0
					getCell.IsWorkerRelevant = getReferenceScenario.IsWorkerRelevant
					getCell.GetDomReadyReferenceMs = getReferenceScenario.GetDomReadyMeanMs
					getCell.GetDomReadyScoreFactor = buildExample201ScoreFactor(getReferenceScenario.GetDomReadyMeanMs, getScenario.GetDomReadyMeanMs)
					getCell.GetDomScore = buildExample201IntegerScore(getCell.GetDomReadyScoreFactor)
				}
				if hasReactScenario && getReactScenario.GetDomReadyMeanMs > 0 && getScenario.GetDomReadyMeanMs > 0 {
					getCell.HasReactBaseline = true
					getCell.GetDomReadyDeltaVsReactMs = getReactScenario.GetDomReadyMeanMs - getScenario.GetDomReadyMeanMs
					getCell.GetDomReadySpeedupVsReact = getReactScenario.GetDomReadyMeanMs / getScenario.GetDomReadyMeanMs
				}
				if getFastestDomReady == 0 || getScenario.GetDomReadyMeanMs < getFastestDomReady {
					getFastestDomReady = getScenario.GetDomReadyMeanMs
				}
				if getFastestPaintVisible == 0 || getScenario.GetPaintVisibleMeanMs < getFastestPaintVisible {
					getFastestPaintVisible = getScenario.GetPaintVisibleMeanMs
				}
				getRow.GetFrameworks = append(getRow.GetFrameworks, getCell)
			}
		}
		sort.Slice(getRow.GetFrameworks, func(parseLeft int, parseRight int) bool {
			if getRow.GetFrameworks[parseLeft].GetDomReadyMeanMs == getRow.GetFrameworks[parseRight].GetDomReadyMeanMs {
				return getRow.GetFrameworks[parseLeft].GetPaintVisibleMeanMs < getRow.GetFrameworks[parseRight].GetPaintVisibleMeanMs
			}
			return getRow.GetFrameworks[parseLeft].GetDomReadyMeanMs < getRow.GetFrameworks[parseRight].GetDomReadyMeanMs
		})
		for parseIndex := range getRow.GetFrameworks {
			if getFastestDomReady == 0 {
				getRow.GetFrameworks[parseIndex].GetRelativeDomReady = 0
			} else {
				getRow.GetFrameworks[parseIndex].GetRelativeDomReady = getRow.GetFrameworks[parseIndex].GetDomReadyMeanMs / getFastestDomReady
			}
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

// buildExample201ScenarioRows converts the stored artifact payload into scenario-first comparison rows.
func buildExample201ScenarioRows(parseArtifact example201Artifact, parseReference example201ScoreReference) []example201ScenarioRow {
	return buildExample201ScenarioRowsFromReport(parseArtifact.GetReport, parseReference)
}

// buildExample201CategoryRows aggregates category-local DOM-ready relative cost and scenario wins.
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
					GetDomReadyGeometricRelative:     1,
					GetPaintVisibleGeometricRelative: 1,
					GetDomReadyGeometricSpeedup:      1,
				}
				getCellsByCategoryAndFramework[getScenarioRow.GetCategory][getFramework.GetFramework] = getFrameworkCell
			}
			getFrameworkCell.GetScenarioCount++
			getFrameworkCell.GetDomReadyMeanMs += getFramework.GetDomReadyMeanMs
			getFrameworkCell.GetPaintVisibleMeanMs += getFramework.GetPaintVisibleMeanMs
			getFrameworkCell.GetDomReadyGeometricRelative *= math.Max(getFramework.GetRelativeDomReady, 0.0001)
			getFrameworkCell.GetPaintVisibleGeometricRelative *= math.Max(getFramework.GetRelativePaintVisible, 0.0001)
			if getFramework.HasDomScoreReference {
				getFrameworkCell.GetScoreScenarioCount++
				getFrameworkCell.GetDomReadyGeometricSpeedup *= math.Max(getFramework.GetDomReadyScoreFactor, 0.0001)
			}
			if getFramework.HasReactBaseline {
				getFrameworkCell.HasReactBaseline = true
				getFrameworkCell.GetReactScenarioCount++
				getFrameworkCell.GetDomReadyDeltaVsReactMs += getFramework.GetDomReadyDeltaVsReactMs
			}
			if parseIndex == 0 || getFramework.GetRelativeDomReady == 1 {
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
			getFrameworkCell.GetDomReadyGeometricRelative = math.Pow(getFrameworkCell.GetDomReadyGeometricRelative, 1/float64(getFrameworkCell.GetScenarioCount))
			getFrameworkCell.GetPaintVisibleGeometricRelative = math.Pow(getFrameworkCell.GetPaintVisibleGeometricRelative, 1/float64(getFrameworkCell.GetScenarioCount))
			if getFrameworkCell.GetScoreScenarioCount > 0 {
				getFrameworkCell.GetDomReadyGeometricSpeedup = math.Pow(getFrameworkCell.GetDomReadyGeometricSpeedup, 1/float64(getFrameworkCell.GetScoreScenarioCount))
				getFrameworkCell.GetDomScore = buildExample201IntegerScore(getFrameworkCell.GetDomReadyGeometricSpeedup)
			}
			if getFrameworkCell.GetReactScenarioCount > 0 {
				getFrameworkCell.GetDomReadyDeltaVsReactMs = getFrameworkCell.GetDomReadyDeltaVsReactMs / float64(getFrameworkCell.GetReactScenarioCount)
			}
			getFrameworkRows = append(getFrameworkRows, *getFrameworkCell)
		}
		sort.Slice(getFrameworkRows, func(parseLeft int, parseRight int) bool {
			if getFrameworkRows[parseLeft].GetScoreScenarioCount != getFrameworkRows[parseRight].GetScoreScenarioCount {
				return getFrameworkRows[parseLeft].GetScoreScenarioCount > getFrameworkRows[parseRight].GetScoreScenarioCount
			}
			if getFrameworkRows[parseLeft].GetDomScore == getFrameworkRows[parseRight].GetDomScore {
				return getFrameworkRows[parseLeft].GetFramework < getFrameworkRows[parseRight].GetFramework
			}
			return getFrameworkRows[parseLeft].GetDomScore > getFrameworkRows[parseRight].GetDomScore
		})
		getRow.GetFrameworks = getFrameworkRows
		getRows = append(getRows, *getRow)
	}
	sort.Slice(getRows, func(parseLeft int, parseRight int) bool {
		return getRows[parseLeft].GetCategory < getRows[parseRight].GetCategory
	})
	return getRows
}

// buildExample201ScalingRows builds RT2-only worker-scaling rows from the scenario comparison payload.
func buildExample201ScalingRows(parseScenarioRows []example201ScenarioRow) []example201ScalingRow {
	getRows := make([]example201ScalingRow, 0, len(parseScenarioRows))
	for _, getScenarioRow := range parseScenarioRows {
		getFrameworks := make([]example201ScalingCell, 0, len(getScenarioRow.GetFrameworks))
		for _, getFramework := range getScenarioRow.GetFrameworks {
			if !getFramework.HasWorkerMetrics {
				continue
			}
			getFrameworks = append(getFrameworks, example201ScalingCell{
				GetFramework:              getFramework.GetFramework,
				GetLabel:                  getFramework.GetLabel,
				GetWorkerCountMean:        getFramework.GetWorkerCountMean,
				GetWorkerBatchMeanMs:      getFramework.GetWorkerBatchMeanMs,
				GetWorkerPreparedItemMean: getFramework.GetWorkerPreparedItemsMean,
				GetDomReadyMeanMs:         getFramework.GetDomReadyMeanMs,
				GetPaintVisibleMeanMs:     getFramework.GetPaintVisibleMeanMs,
			})
		}
		if len(getFrameworks) == 0 {
			continue
		}
		hasExample201PositiveBatch := false
		for _, getFramework := range getFrameworks {
			if getFramework.GetWorkerBatchMeanMs > 0 {
				hasExample201PositiveBatch = true
				break
			}
		}
		if !hasExample201PositiveBatch {
			continue
		}
		sort.Slice(getFrameworks, func(parseLeft int, parseRight int) bool {
			return getFrameworks[parseLeft].GetWorkerCountMean < getFrameworks[parseRight].GetWorkerCountMean
		})
		getBaselineBatchMeanMs := getFrameworks[0].GetWorkerBatchMeanMs
		getBaselineDomReadyMeanMs := getFrameworks[0].GetDomReadyMeanMs
		for parseIndex := range getFrameworks {
			if getBaselineDomReadyMeanMs > 0 && getFrameworks[parseIndex].GetDomReadyMeanMs > 0 {
				getFrameworks[parseIndex].GetRelativeDomReadySpeedup = getBaselineDomReadyMeanMs / getFrameworks[parseIndex].GetDomReadyMeanMs
			}
			if getBaselineBatchMeanMs > 0 && getFrameworks[parseIndex].GetWorkerBatchMeanMs > 0 {
				getFrameworks[parseIndex].GetRelativeBatchSpeedup = getBaselineBatchMeanMs / getFrameworks[parseIndex].GetWorkerBatchMeanMs
			}
		}
		getRows = append(getRows, example201ScalingRow{
			GetScenarioID:    getScenarioRow.GetScenarioID,
			GetScenarioLabel: getScenarioRow.GetScenarioLabel,
			GetCategory:      getScenarioRow.GetCategory,
			GetFrameworks:    getFrameworks,
		})
	}
	return getRows
}

// buildExample201OverallScoreRows aggregates one report-wide geometric DOM-ready score from the fixed reference profile for each framework.
func buildExample201OverallScoreRows(parseScenarioRows []example201ScenarioRow, parseRequireWorkerRelevant bool) []example201OverallScoreCell {
	getCellByFramework := map[string]*example201OverallScoreCell{}
	for _, getScenarioRow := range parseScenarioRows {
		for _, getFramework := range getScenarioRow.GetFrameworks {
			getFrameworkCell := getCellByFramework[getFramework.GetFramework]
			if getFrameworkCell == nil {
				getFrameworkCell = &example201OverallScoreCell{
					GetFramework:         getFramework.GetFramework,
					GetLabel:             getFramework.GetLabel,
					GetDomGeometricSpeed: 1,
				}
				getCellByFramework[getFramework.GetFramework] = getFrameworkCell
			}
			getFrameworkCell.GetScenarioCount++
			if !getFramework.HasDomScoreReference {
				continue
			}
			if parseRequireWorkerRelevant && !getFramework.IsWorkerRelevant {
				continue
			}
			getFrameworkCell.GetScoreScenarioCount++
			getFrameworkCell.GetDomGeometricSpeed *= math.Max(getFramework.GetDomReadyScoreFactor, 0.0001)
		}
	}
	getRows := make([]example201OverallScoreCell, 0, len(getCellByFramework))
	for _, getFrameworkCell := range getCellByFramework {
		if getFrameworkCell.GetScoreScenarioCount > 0 {
			getFrameworkCell.GetDomGeometricSpeed = math.Pow(getFrameworkCell.GetDomGeometricSpeed, 1/float64(getFrameworkCell.GetScoreScenarioCount))
			getFrameworkCell.GetDomScore = buildExample201IntegerScore(getFrameworkCell.GetDomGeometricSpeed)
		}
		getRows = append(getRows, *getFrameworkCell)
	}
	sort.Slice(getRows, func(parseLeft int, parseRight int) bool {
		if getRows[parseLeft].GetScoreScenarioCount != getRows[parseRight].GetScoreScenarioCount {
			return getRows[parseLeft].GetScoreScenarioCount > getRows[parseRight].GetScoreScenarioCount
		}
		if getRows[parseLeft].GetDomScore == getRows[parseRight].GetDomScore {
			return getRows[parseLeft].GetFramework < getRows[parseRight].GetFramework
		}
		return getRows[parseLeft].GetDomScore > getRows[parseRight].GetDomScore
	})
	return getRows
}

// formatExample201SignedMilliseconds renders one signed millisecond delta for the report tables.
func formatExample201SignedMilliseconds(parseValue float64) string {
	return fmt.Sprintf("%+.3f ms", parseValue)
}

// formatExample201ReactDomText renders one DOM-ready comparison against the React baseline for mixed-framework scenario tables.
func formatExample201ReactDomText(parseCell example201ScenarioCell) string {
	if !parseCell.HasReactBaseline {
		return "n/a"
	}
	return fmt.Sprintf("%s / %.3fx", formatExample201SignedMilliseconds(parseCell.GetDomReadyDeltaVsReactMs), parseCell.GetDomReadySpeedupVsReact)
}

// formatExample201BenchmarkMarkdown renders the browser benchmark artifact into a human-readable Markdown report.
func formatExample201BenchmarkMarkdown(parseArtifact example201Artifact, parseReference example201ScoreReference) string {
	var getBuilder strings.Builder
	getScenarioRows := buildExample201ScenarioRows(parseArtifact, parseReference)
	getOverallScoreRows := buildExample201OverallScoreRows(getScenarioRows, false)
	getWorkerOverallScoreRows := buildExample201OverallScoreRows(getScenarioRows, true)
	getCategoryRows := buildExample201CategoryRows(getScenarioRows)
	getScalingRows := buildExample201ScalingRows(getScenarioRows)
	getBuilder.WriteString("# Example 201 Browser Benchmark Report\n\n")
	getBuilder.WriteString(fmt.Sprintf("- Generated at: `%s`\n", parseArtifact.GetGeneratedAt))
	getBuilder.WriteString(fmt.Sprintf("- Browser: `%s`\n", parseArtifact.GetBrowser))
	getBuilder.WriteString(fmt.Sprintf("- Route: `%s`\n", parseArtifact.GetRoute))
	getBuilder.WriteString(fmt.Sprintf("- Iterations: `%d`\n", parseArtifact.GetReport.GetIterations))
	getBuilder.WriteString(fmt.Sprintf("- Warmups: `%d`\n", parseArtifact.GetReport.GetWarmups))
	getBuilder.WriteString(fmt.Sprintf("- Seed: `%d`\n", parseArtifact.GetReport.GetSeed))
	getBuilder.WriteString(fmt.Sprintf("- Scenario order: `%s`\n", strings.Join(parseArtifact.GetReport.GetScenarioOrder, ", ")))
	getBuilder.WriteString(fmt.Sprintf("- Framework order: `%s`\n", strings.Join(parseArtifact.GetReport.GetFrameworkOrder, ", ")))
	if len(parseArtifact.GetReport.GetScenarioFrameworkOrders) > 0 {
		getBuilder.WriteString("- Fairness note: framework order rotates per scenario, so no framework keeps the same warm-cache or JIT slot across the whole run.\n")
	}
	getBuilder.WriteString("- Important boundary: the `runtime2` subjects here still keep DOM ownership on the main thread.\n")
	getBuilder.WriteString("- Worker note: each `Runtime 2 (N Workers)` subject opens the requested Go WASM worker count to prepare core and content chunks before the local runtime2 shell commits DOM updates.\n")
	getBuilder.WriteString("- Non-worker note: deep-tree, primitive, hook-grid, and enterprise-workspace subtree scenarios remain main-thread-owned today, so the worker-backed benefit is expected to concentrate in the core and content scenarios.\n")
	getBuilder.WriteString("- React subject note: the page uses vendored React 18 UMD files under `examples/201-render-benchmark/vendor/`, so the comparison stays local to the repo server.\n\n")
	getBuilder.WriteString("- Finish lines: `DOM Ready` means the scenario correctness contract became true. `Paint Proxy` means one `requestAnimationFrame` boundary after the DOM-ready checkpoint.\n")
	getBuilder.WriteString("- Primary comparison: category summaries and scenario ordering use `DOM Ready` as the lead timing. `Paint Proxy` stays in the report as secondary frame-bound context only.\n")
	getBuilder.WriteString(fmt.Sprintf("- Score reference: `%s` on `%s` from route `%s`. `DOM Score` is `100 * geometric_mean(reference DOM Ready / measured DOM Ready)`.\n", parseReference.GetReferenceLabel, parseReference.GetReferenceBrowser, parseReference.GetRoute))
	getBuilder.WriteString("- Mixed-framework score direction: higher `DOM Score` is faster than the fixed reference profile. `DOM vs React` stays as a separate same-run diagnostic column.\n")
	getBuilder.WriteString("- Worker diagnostics: worker-backed subjects also report whether the measured run triggered chunk preparation, the mean batch count, the last-batch duration, and the prepared-item count for that run window.\n")
	getBuilder.WriteString("- RT2 scaling view: the dedicated scaling section compares worker-preparation batch time first, because paint-proxy is often frame-quantized and can hide real worker-count differences.\n")
	if parseArtifact.GetScalingRun != nil {
		getBuilder.WriteString(fmt.Sprintf("- RT2 stress route: `%s` reruns RT2-only scaling with heavier worker prep so the end-to-end timing spreads beyond one frame when possible.\n", parseArtifact.GetScalingRun.GetRoute))
	}
	getBuilder.WriteString("- Headline scope note: worker-relevant overall score excludes deep-tree, primitive, hook-grid, and enterprise-workspace subtree scenarios because those paths are still main-thread-owned in runtime2 today.\n\n")
	getBuilder.WriteString("## Overall DOM Score (Worker-Relevant)\n\n")
	getBuilder.WriteString("- Score contract: `100` equals the checked-in reference profile for the worker-relevant scenarios only.\n\n")
	getBuilder.WriteString("| Framework | DOM Score | Geom. DOM Score Factor | Scored Scenarios | Total Scenarios |\n")
	getBuilder.WriteString("| --- | ---: | ---: | ---: | ---: |\n")
	for _, getFramework := range getWorkerOverallScoreRows {
		getBuilder.WriteString(fmt.Sprintf(
			"| %s | %d | %.3fx | %d | %d |\n",
			getFramework.GetLabel,
			getFramework.GetDomScore,
			getFramework.GetDomGeometricSpeed,
			getFramework.GetScoreScenarioCount,
			getFramework.GetScenarioCount,
		))
	}
	getBuilder.WriteString("\n## Overall DOM Score (Full Surface)\n\n")
	getBuilder.WriteString("- Full-surface score includes every active scenario, including primitive, deep-tree, hook-grid, and enterprise-workspace subtree paths.\n\n")
	getBuilder.WriteString("| Framework | DOM Score | Geom. DOM Score Factor | Scored Scenarios | Total Scenarios |\n")
	getBuilder.WriteString("| --- | ---: | ---: | ---: | ---: |\n")
	for _, getFramework := range getOverallScoreRows {
		getBuilder.WriteString(fmt.Sprintf(
			"| %s | %d | %.3fx | %d | %d |\n",
			getFramework.GetLabel,
			getFramework.GetDomScore,
			getFramework.GetDomGeometricSpeed,
			getFramework.GetScoreScenarioCount,
			getFramework.GetScenarioCount,
		))
	}
	getBuilder.WriteString("\n")
	getBuilder.WriteString("## Category Summary\n\n")
	getBuilder.WriteString("| Category | Framework | DOM Score | Avg DOM Ready | Avg DOM vs React | Avg Paint Proxy | Geom. DOM Score Factor | Wins |\n")
	getBuilder.WriteString("| --- | --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
	for _, getCategoryRow := range getCategoryRows {
		for _, getFramework := range getCategoryRow.GetFrameworks {
			getDomVsReactText := "n/a"
			getGeomSpeedText := "n/a"
			if getFramework.HasReactBaseline {
				getDomVsReactText = formatExample201SignedMilliseconds(getFramework.GetDomReadyDeltaVsReactMs)
			}
			if getFramework.GetScoreScenarioCount > 0 {
				getGeomSpeedText = fmt.Sprintf("%.3fx", getFramework.GetDomReadyGeometricSpeedup)
			}
			getBuilder.WriteString(fmt.Sprintf(
				"| %s | %s | %d | %.3f ms | %s | %.3f ms | %s | %d |\n",
				getCategoryRow.GetCategory,
				getFramework.GetLabel,
				getFramework.GetDomScore,
				getFramework.GetDomReadyMeanMs,
				getDomVsReactText,
				getFramework.GetPaintVisibleMeanMs,
				getGeomSpeedText,
				getFramework.GetScenarioWins,
			))
		}
	}
	getBuilder.WriteString("\n## RT2 Worker Scaling\n\n")
	for _, getScalingRow := range getScalingRows {
		getBuilder.WriteString(fmt.Sprintf("### %s\n\n", getScalingRow.GetScenarioLabel))
		getBuilder.WriteString(fmt.Sprintf("- Category: `%s`\n", getScalingRow.GetCategory))
		getBuilder.WriteString("- RT2 scaling view: DOM-ready is the primary end-to-end signal here, worker batch shows off-thread prep cost, and paint-proxy stays secondary because it can quantize to one frame.\n\n")
		getBuilder.WriteString("| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |\n")
		getBuilder.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
		for _, getFramework := range getScalingRow.GetFrameworks {
			getBuilder.WriteString(fmt.Sprintf(
				"| %s | %.3f ms | %.3fx | %.3f ms | %.3fx | %.1f | %.3f ms |\n",
				getFramework.GetLabel,
				getFramework.GetDomReadyMeanMs,
				getFramework.GetRelativeDomReadySpeedup,
				getFramework.GetWorkerBatchMeanMs,
				getFramework.GetRelativeBatchSpeedup,
				getFramework.GetWorkerPreparedItemMean,
				getFramework.GetPaintVisibleMeanMs,
			))
		}
		getBuilder.WriteString("\n")
	}
	if parseArtifact.GetScalingRun != nil {
		getStressRows := buildExample201ScalingRows(buildExample201ScenarioRowsFromReport(parseArtifact.GetScalingRun.GetReport, parseReference))
		getBuilder.WriteString("\n## RT2 Worker Scaling Stress\n\n")
		getBuilder.WriteString(fmt.Sprintf("- Description: %s\n", parseArtifact.GetScalingRun.GetDescription))
		getBuilder.WriteString(fmt.Sprintf("- Route: `%s`\n", parseArtifact.GetScalingRun.GetRoute))
		getBuilder.WriteString("- This route is RT2-only and uses a higher worker work scale so DOM-ready and paint metrics stop collapsing into the same one-frame bucket.\n\n")
		for _, getScalingRow := range getStressRows {
			getBuilder.WriteString(fmt.Sprintf("### %s\n\n", getScalingRow.GetScenarioLabel))
			getBuilder.WriteString(fmt.Sprintf("- Category: `%s`\n", getScalingRow.GetCategory))
			getBuilder.WriteString("| Framework | DOM Ready Mean | DOM Ready Speedup vs Smallest Worker Count | Worker Batch Mean | Worker Batch Speedup vs Smallest Worker Count | Prepared Items | Paint Proxy Mean |\n")
			getBuilder.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: |\n")
			for _, getFramework := range getScalingRow.GetFrameworks {
				getBuilder.WriteString(fmt.Sprintf(
					"| %s | %.3f ms | %.3fx | %.3f ms | %.3fx | %.1f | %.3f ms |\n",
					getFramework.GetLabel,
					getFramework.GetDomReadyMeanMs,
					getFramework.GetRelativeDomReadySpeedup,
					getFramework.GetWorkerBatchMeanMs,
					getFramework.GetRelativeBatchSpeedup,
					getFramework.GetWorkerPreparedItemMean,
					getFramework.GetPaintVisibleMeanMs,
				))
			}
			getBuilder.WriteString("\n")
		}
	}
	getBuilder.WriteString("\n## Scenarios\n")
	for _, getScenarioRow := range getScenarioRows {
		getBuilder.WriteString(fmt.Sprintf("\n### %s\n\n", getScenarioRow.GetScenarioLabel))
		getBuilder.WriteString(fmt.Sprintf("- Category: `%s`\n", getScenarioRow.GetCategory))
		getBuilder.WriteString(fmt.Sprintf("- Requested work: %s\n", getScenarioRow.GetRequestedWork))
		getBuilder.WriteString(fmt.Sprintf("- Correctness check: %s\n", getScenarioRow.GetCorrectnessCheck))
		getBuilder.WriteString(fmt.Sprintf("- Finish line: %s\n\n", getScenarioRow.GetFinishLine))
		getBuilder.WriteString("| Framework | DOM Score | DOM Ready Mean | DOM Ready Median | Paint Proxy Mean | Paint Proxy Median | Paint-After-DOM | Mutations C/A/T | Nodes + / - | Worker Batches / Last Batch / Items | Long Tasks | DOM vs React |\n")
		getBuilder.WriteString("| --- | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: | ---: |\n")
		for _, getFramework := range getScenarioRow.GetFrameworks {
			getWorkerText := "n/a"
			if getFramework.HasWorkerMetrics {
				getWorkerText = fmt.Sprintf("%.1f / %.3f ms / %.1f", getFramework.GetWorkerBatchCountMean, getFramework.GetWorkerBatchMeanMs, getFramework.GetWorkerPreparedItemsMean)
			}
			getBuilder.WriteString(fmt.Sprintf(
				"| %s | %d | %.3f ms | %.3f ms | %.3f ms | %.3f ms | %.3f ms | %.1f / %.1f / %.1f | %.1f / %.1f | %s | %.1f / %.3f ms | %s |\n",
				getFramework.GetLabel,
				getFramework.GetDomScore,
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
				getWorkerText,
				getFramework.GetLongTaskCountMean,
				getFramework.GetLongTaskDurationMeanMs,
				formatExample201ReactDomText(getFramework),
			))
		}
	}
	return getBuilder.String()
}

// storeExample201ArtifactNamed writes one named JSON and Markdown benchmark report under the repo-local browser test results directory.
func storeExample201ArtifactNamed(parseT *testing.T, parseRepoRoot string, parseArtifact example201Artifact, parseReportName string) (string, string) {
	parseT.Helper()
	getOutputDir := filepath.Join(parseRepoRoot, "bin", "test-results", "example-201-browser-benchmark")
	if parseErr := os.MkdirAll(getOutputDir, 0o755); parseErr != nil {
		parseT.Fatalf("mkdir example 201 report dir: %v", parseErr)
	}
	getReportName := strings.TrimSpace(parseReportName)
	if getReportName == "" {
		getReportName = "browser-benchmark-report"
	}
	getJSONPath := filepath.Join(getOutputDir, getReportName+".json")
	getMarkdownPath := filepath.Join(getOutputDir, getReportName+".md")
	getJSONBytes, parseErr := json.MarshalIndent(parseArtifact, "", "  ")
	if parseErr != nil {
		parseT.Fatalf("marshal example 201 report: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(getJSONPath, getJSONBytes, 0o644); parseErr2 != nil {
		parseT.Fatalf("write example 201 json report: %v", parseErr2)
	}
	getScoreReference := loadExample201ScoreReference(parseT, parseRepoRoot)
	if parseErr3 := os.WriteFile(getMarkdownPath, []byte(formatExample201BenchmarkMarkdown(parseArtifact, getScoreReference)), 0o644); parseErr3 != nil {
		parseT.Fatalf("write example 201 markdown report: %v", parseErr3)
	}
	return getJSONPath, getMarkdownPath
}

// storeExample201Artifact writes the default JSON and Markdown benchmark reports under the repo-local browser test results directory.
func storeExample201Artifact(parseT *testing.T, parseRepoRoot string, parseArtifact example201Artifact) (string, string) {
	parseT.Helper()
	return storeExample201ArtifactNamed(parseT, parseRepoRoot, parseArtifact, "browser-benchmark-report")
}

// buildExample201WorkerMetricMeans returns mean worker-batch and DOM-ready timings from worker-metric scenarios only.
func buildExample201WorkerMetricMeans(parseReport example201Report) (float64, float64, int) {
	getWorkerBatchSum := 0.0
	getDomReadySum := 0.0
	getSampleCount := 0
	for _, getFramework := range parseReport.GetFrameworks {
		if !strings.HasPrefix(getFramework.GetFramework, "runtime2-workers") {
			continue
		}
		for _, getScenario := range getFramework.GetScenarioResults {
			if !getScenario.HasWorkerMetrics {
				continue
			}
			getWorkerBatchSum += getScenario.GetWorkerBatchMeanMs
			getDomReadySum += getScenario.GetDomReadyMeanMs
			getSampleCount++
		}
	}
	if getSampleCount < 1 {
		return 0, 0, 0
	}
	return getWorkerBatchSum / float64(getSampleCount), getDomReadySum / float64(getSampleCount), getSampleCount
}

// TestBuildExample201ScenarioRowsFromReportUsesFixedReferenceScore verifies the mixed-framework scenario rows use the checked score reference instead of recentering on React.
func TestBuildExample201ScenarioRowsFromReportUsesFixedReferenceScore(parseT *testing.T) {
	getReference := example201ScoreReference{
		GetScenarioReference: []example201ScoreReferenceScenario{
			{
				GetScenarioID:     "core-render",
				GetScenarioLabel:  "Core Render",
				GetCategory:       "Initial Render",
				GetDomReadyMeanMs: 20,
				IsWorkerRelevant:  true,
			},
		},
	}
	getRows := buildExample201ScenarioRowsFromReport(example201Report{
		GetScenarioLabels: []example201ScenarioLabel{
			{
				GetScenarioID:    "core-render",
				GetScenarioLabel: "Core Render",
				GetCategory:      "Initial Render",
			},
		},
		GetFrameworks: []example201FrameworkResult{
			{
				GetFramework: "react",
				GetLabel:     "React 18",
				GetScenarioResults: []example201ScenarioResult{
					{
						GetScenarioID:         "core-render",
						GetDomReadyMeanMs:     25,
						GetPaintVisibleMeanMs: 33,
					},
				},
			},
			{
				GetFramework: "runtime1",
				GetLabel:     "Runtime 1",
				GetScenarioResults: []example201ScenarioResult{
					{
						GetScenarioID:         "core-render",
						GetDomReadyMeanMs:     10,
						GetPaintVisibleMeanMs: 30,
					},
				},
			},
			{
				GetFramework: "runtime2-workers1",
				GetLabel:     "Runtime 2 (1 Worker)",
				GetScenarioResults: []example201ScenarioResult{
					{
						GetScenarioID:         "core-render",
						GetDomReadyMeanMs:     40,
						GetPaintVisibleMeanMs: 45,
					},
				},
			},
		},
	}, getReference)
	if len(getRows) != 1 {
		parseT.Fatalf("expected one scenario row, got %d", len(getRows))
	}
	getCellsByFramework := map[string]example201ScenarioCell{}
	for _, getFramework := range getRows[0].GetFrameworks {
		getCellsByFramework[getFramework.GetFramework] = getFramework
	}
	if !getCellsByFramework["runtime1"].HasReactBaseline {
		parseT.Fatalf("expected runtime1 to include a React baseline")
	}
	if !getCellsByFramework["runtime1"].HasDomScoreReference {
		parseT.Fatalf("expected runtime1 to include a score reference")
	}
	if getCellsByFramework["runtime1"].GetDomScore != 200 {
		parseT.Fatalf("expected runtime1 score to be 200, got %d", getCellsByFramework["runtime1"].GetDomScore)
	}
	if getCellsByFramework["runtime1"].GetDomReadyDeltaVsReactMs != 15 {
		parseT.Fatalf("expected runtime1 delta to be +15ms, got %.3f", getCellsByFramework["runtime1"].GetDomReadyDeltaVsReactMs)
	}
	if getCellsByFramework["react"].GetDomScore != 80 {
		parseT.Fatalf("expected React score to follow the fixed reference, got %d", getCellsByFramework["react"].GetDomScore)
	}
	if getCellsByFramework["react"].GetDomReadySpeedupVsReact != 1 {
		parseT.Fatalf("expected React row to stay at 1.0x versus the same-run React baseline, got %.3f", getCellsByFramework["react"].GetDomReadySpeedupVsReact)
	}
	if getCellsByFramework["runtime2-workers1"].GetDomScore != 50 {
		parseT.Fatalf("expected runtime2 score to be 50, got %d", getCellsByFramework["runtime2-workers1"].GetDomScore)
	}
	if !getCellsByFramework["runtime1"].IsWorkerRelevant {
		parseT.Fatalf("expected core render to stay worker-relevant")
	}
}

// TestBuildExample201CategoryRowsUseFixedReferenceScore verifies the category summary uses fixed-reference geometric score ordering.
func TestBuildExample201CategoryRowsUseFixedReferenceScore(parseT *testing.T) {
	getRows := buildExample201CategoryRows([]example201ScenarioRow{
		{
			GetScenarioID:    "core-render",
			GetScenarioLabel: "Core Render",
			GetCategory:      "Initial Render",
			GetFrameworks: []example201ScenarioCell{
				{GetFramework: "runtime1", GetLabel: "Runtime 1", GetDomReadyMeanMs: 10, GetPaintVisibleMeanMs: 30, GetRelativeDomReady: 1, HasDomScoreReference: true, GetDomReadyScoreFactor: 2, HasReactBaseline: true, GetDomReadyDeltaVsReactMs: 15, GetDomReadySpeedupVsReact: 2.5},
				{GetFramework: "react", GetLabel: "React 18", GetDomReadyMeanMs: 25, GetPaintVisibleMeanMs: 33, GetRelativeDomReady: 2.5, HasDomScoreReference: true, GetDomReadyScoreFactor: 0.8, HasReactBaseline: true, GetDomReadyDeltaVsReactMs: 0, GetDomReadySpeedupVsReact: 1},
			},
		},
		{
			GetScenarioID:    "content-render",
			GetScenarioLabel: "Content Render",
			GetCategory:      "Initial Render",
			GetFrameworks: []example201ScenarioCell{
				{GetFramework: "runtime1", GetLabel: "Runtime 1", GetDomReadyMeanMs: 9, GetPaintVisibleMeanMs: 29, GetRelativeDomReady: 1, HasDomScoreReference: true, GetDomReadyScoreFactor: 2, HasReactBaseline: true, GetDomReadyDeltaVsReactMs: 9, GetDomReadySpeedupVsReact: 2},
				{GetFramework: "react", GetLabel: "React 18", GetDomReadyMeanMs: 18, GetPaintVisibleMeanMs: 31, GetRelativeDomReady: 2, HasDomScoreReference: true, GetDomReadyScoreFactor: 1, HasReactBaseline: true, GetDomReadyDeltaVsReactMs: 0, GetDomReadySpeedupVsReact: 1},
			},
		},
	})
	if len(getRows) != 1 {
		parseT.Fatalf("expected one category row, got %d", len(getRows))
	}
	getFrameworks := getRows[0].GetFrameworks
	if len(getFrameworks) != 2 {
		parseT.Fatalf("expected two frameworks, got %d", len(getFrameworks))
	}
	if getFrameworks[0].GetFramework != "runtime1" {
		parseT.Fatalf("expected runtime1 to sort first by reference score, got %s", getFrameworks[0].GetFramework)
	}
	if getFrameworks[0].GetDomReadyGeometricSpeedup != 2 {
		parseT.Fatalf("expected runtime1 geometric score factor to be 2.0x, got %.3f", getFrameworks[0].GetDomReadyGeometricSpeedup)
	}
	if getFrameworks[0].GetDomScore != 200 {
		parseT.Fatalf("expected runtime1 category score to be 200, got %d", getFrameworks[0].GetDomScore)
	}
	if getFrameworks[0].GetDomReadyDeltaVsReactMs != 12 {
		parseT.Fatalf("expected runtime1 average delta to be +12ms, got %.3f", getFrameworks[0].GetDomReadyDeltaVsReactMs)
	}
	if getFrameworks[1].GetFramework != "react" {
		parseT.Fatalf("expected React row second, got %s", getFrameworks[1].GetFramework)
	}
	if getFrameworks[1].GetDomReadyGeometricSpeedup < 0.89 || getFrameworks[1].GetDomReadyGeometricSpeedup > 0.90 {
		parseT.Fatalf("expected React geometric score factor near 0.894x, got %.3f", getFrameworks[1].GetDomReadyGeometricSpeedup)
	}
	if getFrameworks[1].GetDomScore != 89 {
		parseT.Fatalf("expected React category score to be 89, got %d", getFrameworks[1].GetDomScore)
	}
}

// TestExample201BrowserBenchmarkReport builds the benchmark subject, runs the browser comparison through gwc examples, and writes a local report.
func TestExample201BrowserBenchmarkReport(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	buildExample201BenchmarkWorkerWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18101")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		getArtifact := buildExample201Artifact(parseT, parsePage, getBaseURL, "")
		getArtifact.GetScalingRun = &example201ScalingRun{
			GetDescription: "RT2-only stress run with runtime2WorkScale=12 to expose worker-count scaling beyond frame-quantized paint-proxy timing.",
			GetRoute:       buildExample201ScalingRoute(),
			GetReport:      buildExample201Artifact(parseT, parsePage, getBaseURL, buildExample201ScalingRoute()).GetReport,
		}
		getJSONPath, getMarkdownPath := storeExample201Artifact(parseT, getRepoRoot, getArtifact)
		parseT.Logf("example 201 benchmark json report: %s", getJSONPath)
		parseT.Logf("example 201 benchmark markdown report: %s", getMarkdownPath)
	})
}

// TestExample201BrowserBenchmarkReportRuntime2OneAndFourWorkers runs the mixed-framework browser report for React, runtime1, RT2x1, and RT2x4 only.
func TestExample201BrowserBenchmarkReportRuntime2OneAndFourWorkers(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	buildExample201BenchmarkWorkerWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18101")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		getArtifact := buildExample201Artifact(parseT, parsePage, getBaseURL, buildExample201OneAndFourWorkerRoute())
		getJSONPath, getMarkdownPath := storeExample201ArtifactNamed(parseT, getRepoRoot, getArtifact, "browser-benchmark-report-rt2-1-4")
		parseT.Logf("example 201 1x4 benchmark json report: %s", getJSONPath)
		parseT.Logf("example 201 1x4 benchmark markdown report: %s", getMarkdownPath)
	})
}

// TestExample201BrowserBenchmarkDispatchCompare runs RT2 stress routes for legacy per-chunk and optimized lane-batch dispatch.
func TestExample201BrowserBenchmarkDispatchCompare(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	buildExample201BenchmarkWorkerWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18101")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		getChunkRoute := buildExample201ScalingRouteWithDispatch("chunk")
		getBatchRoute := buildExample201ScalingRouteWithDispatch("batch")
		getChunkArtifact := buildExample201Artifact(parseT, parsePage, getBaseURL, getChunkRoute)
		getBatchArtifact := buildExample201Artifact(parseT, parsePage, getBaseURL, getBatchRoute)
		getChunkJSONPath, getChunkMarkdownPath := storeExample201ArtifactNamed(parseT, getRepoRoot, getChunkArtifact, "browser-benchmark-report-rt2-dispatch-chunk")
		getBatchJSONPath, getBatchMarkdownPath := storeExample201ArtifactNamed(parseT, getRepoRoot, getBatchArtifact, "browser-benchmark-report-rt2-dispatch-batch")
		parseT.Logf("example 201 dispatch chunk benchmark json report: %s", getChunkJSONPath)
		parseT.Logf("example 201 dispatch chunk benchmark markdown report: %s", getChunkMarkdownPath)
		parseT.Logf("example 201 dispatch batch benchmark json report: %s", getBatchJSONPath)
		parseT.Logf("example 201 dispatch batch benchmark markdown report: %s", getBatchMarkdownPath)

		getChunkWorkerBatchMean, getChunkDomReadyMean, getChunkSampleCount := buildExample201WorkerMetricMeans(getChunkArtifact.GetReport)
		getBatchWorkerBatchMean, getBatchDomReadyMean, getBatchSampleCount := buildExample201WorkerMetricMeans(getBatchArtifact.GetReport)
		if getChunkSampleCount < 1 || getBatchSampleCount < 1 {
			parseT.Fatalf("expected worker metric samples for dispatch compare, got chunk=%d batch=%d", getChunkSampleCount, getBatchSampleCount)
		}
		parseT.Logf("example 201 dispatch chunk means: workerBatch=%.3fms domReady=%.3fms samples=%d", getChunkWorkerBatchMean, getChunkDomReadyMean, getChunkSampleCount)
		parseT.Logf("example 201 dispatch batch means: workerBatch=%.3fms domReady=%.3fms samples=%d", getBatchWorkerBatchMean, getBatchDomReadyMean, getBatchSampleCount)
		if getChunkWorkerBatchMean > 0 {
			parseT.Logf("example 201 dispatch worker-batch delta: %.2f%%", ((getChunkWorkerBatchMean-getBatchWorkerBatchMean)/getChunkWorkerBatchMean)*100)
		}
		if getChunkDomReadyMean > 0 {
			parseT.Logf("example 201 dispatch dom-ready delta: %.2f%%", ((getChunkDomReadyMean-getBatchDomReadyMean)/getChunkDomReadyMean)*100)
		}
		if getChunkWorkerBatchMean > 0 && getBatchWorkerBatchMean > getChunkWorkerBatchMean*1.35 {
			parseT.Fatalf("expected lane-batch dispatch to avoid large regression; chunk=%.3fms batch=%.3fms", getChunkWorkerBatchMean, getBatchWorkerBatchMean)
		}
	})
}

// TestExample201BrowserBenchmarkHonorsConfiguredWorkerCounts verifies the configurable RT2 worker matrix reports the requested worker counts.
func TestExample201BrowserBenchmarkHonorsConfiguredWorkerCounts(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	buildExample201BenchmarkWorkerWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18101")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		getArtifact := buildExample201Artifact(parseT, parsePage, getBaseURL, "/examples/201-render-benchmark/?iterations=2&warmups=1&seed=20101&runtime2WorkerCounts=2,8")
		getExpectedLabels := map[string]string{
			"runtime2-workers2": "Runtime 2 (2 Workers)",
			"runtime2-workers8": "Runtime 2 (8 Workers)",
		}
		for parseFrameworkIndex := range getArtifact.GetReport.GetFrameworks {
			getFramework := getArtifact.GetReport.GetFrameworks[parseFrameworkIndex]
			getExpectedLabel, hasExpectedLabel := getExpectedLabels[getFramework.GetFramework]
			if !hasExpectedLabel {
				continue
			}
			if getFramework.GetLabel != getExpectedLabel {
				parseT.Fatalf("expected configurable worker label %q, got %q", getExpectedLabel, getFramework.GetLabel)
			}
			getExpectedWorkerCount := 0.0
			if strings.HasSuffix(getFramework.GetFramework, "2") {
				getExpectedWorkerCount = 2
			} else if strings.HasSuffix(getFramework.GetFramework, "8") {
				getExpectedWorkerCount = 8
			}
			for _, getScenario := range getFramework.GetScenarioResults {
				if !getScenario.HasWorkerMetrics {
					continue
				}
				if getScenario.GetWorkerCountMean != getExpectedWorkerCount {
					parseT.Fatalf("expected worker metrics to report %.1f workers for %s, got %.1f", getExpectedWorkerCount, getScenario.GetScenarioID, getScenario.GetWorkerCountMean)
				}
			}
		}
	})
}
