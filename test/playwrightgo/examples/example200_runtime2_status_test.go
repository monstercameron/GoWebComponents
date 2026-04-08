//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"os/exec"
	"regexp"
	"runtime"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

const (
	getExample200ExpectedWorkerCount  = 8
	getExample200AppLabelName         = "runtime2-app"
	getExample200InspectorLabelName   = "runtime2-inspector"
	getExample200OwnerPanelLabelName  = "runtime2-owner-panel"
	getExample200RegionLabelName      = "runtime2-region"
	getExample200TracePanelLabelName  = "runtime2-render-trace"
	getExample200WorkerFleetLabelName = "runtime2-worker-fleet"
	getExample200WorkbenchLabelName   = "runtime2-workbench"
)

type example200RuntimeExecutionArtifact struct {
	GetRoute                      string `json:"getRoute"`
	GetInitialAppRenderPass       int    `json:"getInitialAppRenderPass"`
	GetSettledAppRenderPass       int    `json:"getSettledAppRenderPass"`
	GetLoadRerenderDelta          int    `json:"getLoadRerenderDelta"`
	GetOwnerState                 int    `json:"getOwnerState"`
	HasRuntime2Proof              bool   `json:"hasRuntime2Proof"`
	HasRuntime2ActiveRoute        bool   `json:"hasRuntime2ActiveRoute"`
	GetWorkerBackedLine           string `json:"getWorkerBackedLine"`
	HasWorkerBatchSuccess         bool   `json:"hasWorkerBatchSuccess"`
	GetWorkerProbeDoneCount       int    `json:"getWorkerProbeDoneCount"`
	GetRuntime2LogCount           int    `json:"getRuntime2LogCount"`
	GetWorkerLogCount             int    `json:"getWorkerLogCount"`
	GetConsoleErrorCount          int    `json:"getConsoleErrorCount"`
	GetPageErrorCount             int    `json:"getPageErrorCount"`
	GetConsoleSampleText          string `json:"getConsoleSampleText"`
	GetPageErrorSampleText        string `json:"getPageErrorSampleText"`
	GetAppLabelCountBefore        int    `json:"getAppLabelCountBefore"`
	GetAppLabelCountBeforeIdle    int    `json:"getAppLabelCountBeforeIdle"`
	GetAppLabelCountAfterNoop     int    `json:"getAppLabelCountAfterNoop"`
	GetAppLabelCountAfter         int    `json:"getAppLabelCountAfter"`
	GetAppLabelCountAfterIdle     int    `json:"getAppLabelCountAfterIdle"`
	GetAppLabelCountAfterBurst    int    `json:"getAppLabelCountAfterBurst"`
	GetAppLabelCountBurstIdle     int    `json:"getAppLabelCountBurstIdle"`
	GetInspectorLabelBeforeIdle   int    `json:"getInspectorLabelBeforeIdle"`
	GetInspectorLabelAfterNoop    int    `json:"getInspectorLabelAfterNoop"`
	GetInspectorLabelAfter        int    `json:"getInspectorLabelAfter"`
	GetInspectorLabelAfterIdle    int    `json:"getInspectorLabelAfterIdle"`
	GetInspectorLabelBurstIdle    int    `json:"getInspectorLabelBurstIdle"`
	GetRegionLabelBeforeIdle      int    `json:"getRegionLabelBeforeIdle"`
	GetRegionLabelAfterNoop       int    `json:"getRegionLabelAfterNoop"`
	GetRegionLabelAfter           int    `json:"getRegionLabelAfter"`
	GetRegionLabelAfterIdle       int    `json:"getRegionLabelAfterIdle"`
	GetRegionLabelAfterBurst      int    `json:"getRegionLabelAfterBurst"`
	GetRegionLabelBurstIdle       int    `json:"getRegionLabelBurstIdle"`
	GetTracePanelLabelBeforeIdle  int    `json:"getTracePanelLabelBeforeIdle"`
	GetTracePanelLabelAfterNoop   int    `json:"getTracePanelLabelAfterNoop"`
	GetTracePanelLabelAfter       int    `json:"getTracePanelLabelAfter"`
	GetTracePanelLabelAfterIdle   int    `json:"getTracePanelLabelAfterIdle"`
	GetTracePanelLabelAfterBurst  int    `json:"getTracePanelLabelAfterBurst"`
	GetTracePanelLabelBurstIdle   int    `json:"getTracePanelLabelBurstIdle"`
	GetWorkerFleetLabelBeforeIdle int    `json:"getWorkerFleetLabelBeforeIdle"`
	GetWorkerFleetLabelAfterNoop  int    `json:"getWorkerFleetLabelAfterNoop"`
	GetWorkerFleetLabelAfter      int    `json:"getWorkerFleetLabelAfter"`
	GetWorkerFleetLabelAfterIdle  int    `json:"getWorkerFleetLabelAfterIdle"`
	GetWorkerFleetLabelAfterBurst int    `json:"getWorkerFleetLabelAfterBurst"`
	GetWorkerFleetLabelBurstIdle  int    `json:"getWorkerFleetLabelBurstIdle"`
	GetOwnerStateAfterBurst       int    `json:"getOwnerStateAfterBurst"`
	GetOwnerPanelLabelAfter       int    `json:"getOwnerPanelLabelAfter"`
	GetOwnerPanelLabelAfterIdle   int    `json:"getOwnerPanelLabelAfterIdle"`
	GetOwnerPanelLabelAfterBurst  int    `json:"getOwnerPanelLabelAfterBurst"`
	GetOwnerPanelLabelBurstIdle   int    `json:"getOwnerPanelLabelBurstIdle"`
	GetWorkbenchLabelAfter        int    `json:"getWorkbenchLabelAfter"`
	GetWorkbenchLabelAfterIdle    int    `json:"getWorkbenchLabelAfterIdle"`
	GetWorkbenchLabelAfterBurst   int    `json:"getWorkbenchLabelAfterBurst"`
	GetWorkbenchLabelBurstIdle    int    `json:"getWorkbenchLabelBurstIdle"`
	GetWorkbenchLabelBeforeNoop   int    `json:"getWorkbenchLabelBeforeNoop"`
	GetWorkbenchLabelAfterNoop    int    `json:"getWorkbenchLabelAfterNoop"`
}

var (
	getExample200AppRenderPassPattern = regexp.MustCompile(`App(?: committed)? render pass #(-?\d+)`)
	getExample200OwnerStatePattern    = regexp.MustCompile(`Owner State\s*(-?\d+)`)
	getExample200WorkerBackedPattern  = regexp.MustCompile(`Worker-backed:\s*([^\r\n]+)`)
	getExample200RenderLabelPattern   = regexp.MustCompile(`label=([a-z0-9-]+)\s+count=(\d+)`)
)

// buildExample200Runtime2MainWasm builds the example 200 main wasm artifact used by the runtime2 status page.
func buildExample200Runtime2MainWasm(parseT *testing.T, parseRepoRoot string) {
	parseT.Helper()
	getCommand := exec.Command(
		"go",
		"run", "./tools/gwc", "build",
		"-app", "./examples/testing/runtime2-status/main.go",
		"-root", "./examples/testing/runtime2-status",
		"-out", "./examples/static/bin/runtime2-status.wasm",
	)
	getCommand.Dir = parseRepoRoot
	if getOutput, parseErr := getCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build example 200 main wasm: %v\n%s", parseErr, string(getOutput))
	}
}

// buildExample200Runtime2WorkerWasm builds the worker wasm artifact used by the example 200 worker pool.
func buildExample200Runtime2WorkerWasm(parseT *testing.T, parseRepoRoot string) {
	parseT.Helper()
	getCommand := exec.Command(
		"go",
		"run", "./tools/gwc", "build",
		"-app", "./examples/testing/runtime2-status/backgroundworker/main.go",
		"-root", "./examples/testing/runtime2-status/backgroundworker",
		"-out", "./bin/runtime2-status-worker.wasm",
	)
	getCommand.Dir = parseRepoRoot
	if getOutput, parseErr := getCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build example 200 worker wasm: %v\n%s", parseErr, string(getOutput))
	}
}

// startExample200RuntimeServer starts a gwc static server from repo root so example 200 can resolve /examples, /bin, and /third_party paths.
func startExample200RuntimeServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseStop := startExamplesCommand(
		parseT,
		parseRepoRoot,
		"go",
		"run", "./tools/gwc", "serve",
		"-root", ".",
		"-host", "127.0.0.1",
		"-port", parsePort,
	)
	parseT.Cleanup(parseStop)
	parseBaseURL := "http://127.0.0.1:" + parsePort
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 30*time.Second)
	return parseBaseURL
}

// parseExample200AppRenderPass extracts the visible app render pass from page body text.
func parseExample200AppRenderPass(parseBodyText string) int {
	getMatches := getExample200AppRenderPassPattern.FindStringSubmatch(parseBodyText)
	if len(getMatches) < 2 {
		return -1
	}
	getValue, parseErr := strconv.Atoi(strings.TrimSpace(getMatches[1]))
	if parseErr != nil {
		return -1
	}
	return getValue
}

// parseExample200OwnerState extracts the visible owner state from page body text.
func parseExample200OwnerState(parseBodyText string) int {
	getMatches := getExample200OwnerStatePattern.FindStringSubmatch(parseBodyText)
	if len(getMatches) < 2 {
		return -1
	}
	getValue, parseErr := strconv.Atoi(strings.TrimSpace(getMatches[1]))
	if parseErr != nil {
		return -1
	}
	return getValue
}

// parseExample200WorkerBackedLine extracts the runtime2 proof Worker-backed line from page body text.
func parseExample200WorkerBackedLine(parseBodyText string) string {
	getMatches := getExample200WorkerBackedPattern.FindStringSubmatch(parseBodyText)
	if len(getMatches) < 2 {
		return ""
	}
	return strings.TrimSpace(getMatches[1])
}

// readExample200BodyText reads the rendered body text from the browser page.
func readExample200BodyText(parseT *testing.T, parsePage playwright.Page) string {
	parseT.Helper()
	getBodyText, parseErr := parsePage.TextContent("body")
	if parseErr != nil {
		parseT.Fatalf("read example 200 body text: %v", parseErr)
	}
	return getBodyText
}

// waitForExample200RuntimeReady polls until the app render pass and runtime2 proof route text appear in the page body.
func waitForExample200RuntimeReady(parseT *testing.T, parsePage playwright.Page, parseTimeout time.Duration) string {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	parseLastBodyText := ""
	for time.Now().Before(parseDeadline) {
		parseLastBodyText = readExample200BodyText(parseT, parsePage)
		if parseExample200AppRenderPass(parseLastBodyText) >= 1 &&
			strings.Contains(strings.ToLower(parseLastBodyText), "route: runtime2 active") {
			return parseLastBodyText
		}
		parsePage.WaitForTimeout(200)
	}
	parseT.Fatalf("wait for example 200 runtime ready timed out after %s; body=%q", parseTimeout, strings.TrimSpace(parseLastBodyText))
	return ""
}

// captureExample200RuntimeExecutionArtifact captures runtime2 execution signals for one example 200 browser run.
func captureExample200RuntimeExecutionArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example200RuntimeExecutionArtifact {
	parseT.Helper()
	getRoute := "/examples/testing/runtime2-status/runtime2-status.html"
	getArtifact := example200RuntimeExecutionArtifact{
		GetRoute:                getRoute,
		GetInitialAppRenderPass: -1,
		GetSettledAppRenderPass: -1,
		GetOwnerState:           -1,
	}

	var getLogMu sync.Mutex
	getConsoleSamples := make([]string, 0, 120)
	getPageErrorSamples := make([]string, 0, 8)
	getLabelCountByLabel := map[string]int{}
	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		getLogText := strings.TrimSpace(parseMessage.Text())
		if getLogText == "" {
			return
		}
		getLogMu.Lock()
		defer getLogMu.Unlock()
		if parseMessage.Type() == "error" {
			getArtifact.GetConsoleErrorCount++
		}
		if len(getConsoleSamples) < 120 {
			getConsoleSamples = append(getConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), getLogText))
		}
		if strings.Contains(getLogText, "[runtime2-status/runtime2]") {
			getArtifact.GetRuntime2LogCount++
		}
		if strings.Contains(getLogText, "[runtime2-status-worker/runtime2]") {
			getArtifact.GetWorkerLogCount++
			if strings.Contains(getLogText, "probe complete") {
				getArtifact.GetWorkerProbeDoneCount++
			}
		}
		if strings.Contains(getLogText, "worker batch success") {
			getArtifact.HasWorkerBatchSuccess = true
		}
		getLabelMatches := getExample200RenderLabelPattern.FindStringSubmatch(getLogText)
		if len(getLabelMatches) == 3 {
			getLabel := strings.TrimSpace(getLabelMatches[1])
			getCount, parseErr := strconv.Atoi(strings.TrimSpace(getLabelMatches[2]))
			if parseErr == nil && getCount > getLabelCountByLabel[getLabel] {
				getLabelCountByLabel[getLabel] = getCount
			}
		}
	})
	parsePage.OnPageError(func(parseErr error) {
		getLogMu.Lock()
		getArtifact.GetPageErrorCount++
		if parseErr != nil && len(getPageErrorSamples) < 8 {
			getPageErrorSamples = append(getPageErrorSamples, strings.TrimSpace(parseErr.Error()))
		}
		getLogMu.Unlock()
	})

	if _, parseErr := parsePage.Goto(parseBaseURL+getRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto example 200 route: %v", parseErr)
	}
	getInitialBodyText := waitForExample200RuntimeReady(parseT, parsePage, 90*time.Second)
	getArtifact.GetInitialAppRenderPass = parseExample200AppRenderPass(getInitialBodyText)
	getArtifact.GetOwnerState = parseExample200OwnerState(getInitialBodyText)
	getArtifact.HasRuntime2Proof = strings.Contains(getInitialBodyText, "Runtime2 Proof")
	getArtifact.HasRuntime2ActiveRoute = strings.Contains(strings.ToLower(getInitialBodyText), "route: runtime2 active")
	getArtifact.GetWorkerBackedLine = parseExample200WorkerBackedLine(getInitialBodyText)

	parsePage.WaitForTimeout(2000)
	getLogMu.Lock()
	getArtifact.GetAppLabelCountBefore = getLabelCountByLabel[getExample200AppLabelName]
	getLogMu.Unlock()
	parsePage.WaitForTimeout(2000)
	getLogMu.Lock()
	getArtifact.GetAppLabelCountBeforeIdle = getLabelCountByLabel[getExample200AppLabelName]
	getArtifact.GetInspectorLabelBeforeIdle = getLabelCountByLabel[getExample200InspectorLabelName]
	getArtifact.GetRegionLabelBeforeIdle = getLabelCountByLabel[getExample200RegionLabelName]
	getArtifact.GetTracePanelLabelBeforeIdle = getLabelCountByLabel[getExample200TracePanelLabelName]
	getArtifact.GetWorkerFleetLabelBeforeIdle = getLabelCountByLabel[getExample200WorkerFleetLabelName]
	getArtifact.GetWorkbenchLabelBeforeNoop = getLabelCountByLabel[getExample200WorkbenchLabelName]
	getLogMu.Unlock()
	if parseClickErr := parsePage.Click("button:has-text(\"Overview\")"); parseClickErr != nil {
		parseT.Fatalf("click no-op overview pane: %v", parseClickErr)
	}
	if parseClickErr := parsePage.Click("button:has-text(\"x1\")"); parseClickErr != nil {
		parseT.Fatalf("click no-op step x1: %v", parseClickErr)
	}
	parsePage.WaitForTimeout(600)
	getLogMu.Lock()
	getArtifact.GetAppLabelCountAfterNoop = getLabelCountByLabel[getExample200AppLabelName]
	getArtifact.GetInspectorLabelAfterNoop = getLabelCountByLabel[getExample200InspectorLabelName]
	getArtifact.GetRegionLabelAfterNoop = getLabelCountByLabel[getExample200RegionLabelName]
	getArtifact.GetTracePanelLabelAfterNoop = getLabelCountByLabel[getExample200TracePanelLabelName]
	getArtifact.GetWorkerFleetLabelAfterNoop = getLabelCountByLabel[getExample200WorkerFleetLabelName]
	getArtifact.GetWorkbenchLabelAfterNoop = getLabelCountByLabel[getExample200WorkbenchLabelName]
	getLogMu.Unlock()
	getSettledBodyText := readExample200BodyText(parseT, parsePage)
	getArtifact.GetSettledAppRenderPass = parseExample200AppRenderPass(getSettledBodyText)
	if getArtifact.GetInitialAppRenderPass >= 0 && getArtifact.GetSettledAppRenderPass >= 0 {
		getArtifact.GetLoadRerenderDelta = getArtifact.GetSettledAppRenderPass - getArtifact.GetInitialAppRenderPass
	}

	if parseClickErr := parsePage.Click("button:has-text(\"Increment\")"); parseClickErr != nil {
		parseT.Fatalf("click owner increment: %v", parseClickErr)
	}
	parseWorkerDeadline := time.Now().Add(15 * time.Second)
	hasWorkerReady := false
	parseLastBodyText := ""
	for time.Now().Before(parseWorkerDeadline) {
		parseLastBodyText = readExample200BodyText(parseT, parsePage)
		getLogMu.Lock()
		hasBatchSuccess := getArtifact.HasWorkerBatchSuccess
		getProbeDoneCount := getArtifact.GetWorkerProbeDoneCount
		getLogMu.Unlock()
		if strings.Contains(parseLastBodyText, "Probe 8") || (hasBatchSuccess && getProbeDoneCount >= getExample200ExpectedWorkerCount) {
			hasWorkerReady = true
			break
		}
		parsePage.WaitForTimeout(150)
	}
	if !hasWorkerReady {
		getLogMu.Lock()
		getArtifact.GetConsoleSampleText = strings.Join(getConsoleSamples, " || ")
		getArtifact.GetPageErrorSampleText = strings.Join(getPageErrorSamples, " || ")
		getConsoleSampleText := getArtifact.GetConsoleSampleText
		getPageErrorSampleText := getArtifact.GetPageErrorSampleText
		getLogMu.Unlock()
		parseT.Fatalf("wait for worker probe cards timed out; body=%q console=%q page-errors=%q", strings.TrimSpace(parseLastBodyText), getConsoleSampleText, getPageErrorSampleText)
	}
	parsePage.WaitForTimeout(600)
	getLogMu.Lock()
	getArtifact.GetAppLabelCountAfter = getLabelCountByLabel[getExample200AppLabelName]
	getArtifact.GetInspectorLabelAfter = getLabelCountByLabel[getExample200InspectorLabelName]
	getArtifact.GetOwnerPanelLabelAfter = getLabelCountByLabel[getExample200OwnerPanelLabelName]
	getArtifact.GetRegionLabelAfter = getLabelCountByLabel[getExample200RegionLabelName]
	getArtifact.GetTracePanelLabelAfter = getLabelCountByLabel[getExample200TracePanelLabelName]
	getArtifact.GetWorkerFleetLabelAfter = getLabelCountByLabel[getExample200WorkerFleetLabelName]
	getArtifact.GetWorkbenchLabelAfter = getLabelCountByLabel[getExample200WorkbenchLabelName]
	getLogMu.Unlock()
	parsePage.WaitForTimeout(2500)
	getLogMu.Lock()
	getArtifact.GetAppLabelCountAfterIdle = getLabelCountByLabel[getExample200AppLabelName]
	getArtifact.GetInspectorLabelAfterIdle = getLabelCountByLabel[getExample200InspectorLabelName]
	getArtifact.GetOwnerPanelLabelAfterIdle = getLabelCountByLabel[getExample200OwnerPanelLabelName]
	getArtifact.GetRegionLabelAfterIdle = getLabelCountByLabel[getExample200RegionLabelName]
	getArtifact.GetTracePanelLabelAfterIdle = getLabelCountByLabel[getExample200TracePanelLabelName]
	getArtifact.GetWorkerFleetLabelAfterIdle = getLabelCountByLabel[getExample200WorkerFleetLabelName]
	getArtifact.GetWorkbenchLabelAfterIdle = getLabelCountByLabel[getExample200WorkbenchLabelName]
	getArtifact.GetConsoleSampleText = strings.Join(getConsoleSamples, " || ")
	getArtifact.GetPageErrorSampleText = strings.Join(getPageErrorSamples, " || ")
	getLogMu.Unlock()
	for parseBurst := 0; parseBurst < 8; parseBurst++ {
		if parseClickErr := parsePage.Click("button:has-text(\"Increment\")"); parseClickErr != nil {
			parseT.Fatalf("click owner increment burst #%d: %v", parseBurst+1, parseClickErr)
		}
		parsePage.WaitForTimeout(250)
	}
	parsePage.WaitForTimeout(1000)
	getBurstBodyText := readExample200BodyText(parseT, parsePage)
	getArtifact.GetOwnerStateAfterBurst = parseExample200OwnerState(getBurstBodyText)
	getLogMu.Lock()
	getArtifact.GetAppLabelCountAfterBurst = getLabelCountByLabel[getExample200AppLabelName]
	getArtifact.GetOwnerPanelLabelAfterBurst = getLabelCountByLabel[getExample200OwnerPanelLabelName]
	getArtifact.GetRegionLabelAfterBurst = getLabelCountByLabel[getExample200RegionLabelName]
	getArtifact.GetTracePanelLabelAfterBurst = getLabelCountByLabel[getExample200TracePanelLabelName]
	getArtifact.GetWorkerFleetLabelAfterBurst = getLabelCountByLabel[getExample200WorkerFleetLabelName]
	getArtifact.GetWorkbenchLabelAfterBurst = getLabelCountByLabel[getExample200WorkbenchLabelName]
	getLogMu.Unlock()
	parsePage.WaitForTimeout(5200)
	getLogMu.Lock()
	getArtifact.GetAppLabelCountBurstIdle = getLabelCountByLabel[getExample200AppLabelName]
	getArtifact.GetInspectorLabelBurstIdle = getLabelCountByLabel[getExample200InspectorLabelName]
	getArtifact.GetOwnerPanelLabelBurstIdle = getLabelCountByLabel[getExample200OwnerPanelLabelName]
	getArtifact.GetRegionLabelBurstIdle = getLabelCountByLabel[getExample200RegionLabelName]
	getArtifact.GetTracePanelLabelBurstIdle = getLabelCountByLabel[getExample200TracePanelLabelName]
	getArtifact.GetWorkerFleetLabelBurstIdle = getLabelCountByLabel[getExample200WorkerFleetLabelName]
	getArtifact.GetWorkbenchLabelBurstIdle = getLabelCountByLabel[getExample200WorkbenchLabelName]
	getLogMu.Unlock()

	return getArtifact
}

// formatExample200RuntimeExecutionSummary formats one runtime2 execution artifact into concise test logs.
func formatExample200RuntimeExecutionSummary(parseArtifact example200RuntimeExecutionArtifact) string {
	return fmt.Sprintf(
		"route=%s initial-pass=%d settled-pass=%d load-delta=%d owner=%d runtime2-proof=%t runtime2-route-active=%t worker-backed=%q batch-success=%t worker-probe-complete=%d runtime2-logs=%d worker-logs=%d app-label-before=%d app-label-before-idle=%d app-label-after-noop=%d app-label-after=%d app-label-after-idle=%d app-label-after-burst=%d app-label-burst-idle=%d inspector-before-idle=%d inspector-after-noop=%d inspector-after=%d inspector-after-idle=%d inspector-burst-idle=%d region-before-idle=%d region-after-noop=%d region-after=%d region-after-idle=%d region-after-burst=%d region-burst-idle=%d trace-before-idle=%d trace-after-noop=%d trace-after=%d trace-after-idle=%d trace-after-burst=%d trace-burst-idle=%d fleet-before-idle=%d fleet-after-noop=%d fleet-after=%d fleet-after-idle=%d fleet-after-burst=%d fleet-burst-idle=%d owner-after-burst=%d owner-panel-after=%d owner-panel-after-idle=%d owner-panel-after-burst=%d owner-panel-burst-idle=%d workbench-label-before-noop=%d workbench-label-after-noop=%d workbench-label-after=%d workbench-label-after-idle=%d workbench-label-after-burst=%d workbench-label-burst-idle=%d console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetRoute,
		parseArtifact.GetInitialAppRenderPass,
		parseArtifact.GetSettledAppRenderPass,
		parseArtifact.GetLoadRerenderDelta,
		parseArtifact.GetOwnerState,
		parseArtifact.HasRuntime2Proof,
		parseArtifact.HasRuntime2ActiveRoute,
		parseArtifact.GetWorkerBackedLine,
		parseArtifact.HasWorkerBatchSuccess,
		parseArtifact.GetWorkerProbeDoneCount,
		parseArtifact.GetRuntime2LogCount,
		parseArtifact.GetWorkerLogCount,
		parseArtifact.GetAppLabelCountBefore,
		parseArtifact.GetAppLabelCountBeforeIdle,
		parseArtifact.GetAppLabelCountAfterNoop,
		parseArtifact.GetAppLabelCountAfter,
		parseArtifact.GetAppLabelCountAfterIdle,
		parseArtifact.GetAppLabelCountAfterBurst,
		parseArtifact.GetAppLabelCountBurstIdle,
		parseArtifact.GetInspectorLabelBeforeIdle,
		parseArtifact.GetInspectorLabelAfterNoop,
		parseArtifact.GetInspectorLabelAfter,
		parseArtifact.GetInspectorLabelAfterIdle,
		parseArtifact.GetInspectorLabelBurstIdle,
		parseArtifact.GetRegionLabelBeforeIdle,
		parseArtifact.GetRegionLabelAfterNoop,
		parseArtifact.GetRegionLabelAfter,
		parseArtifact.GetRegionLabelAfterIdle,
		parseArtifact.GetRegionLabelAfterBurst,
		parseArtifact.GetRegionLabelBurstIdle,
		parseArtifact.GetTracePanelLabelBeforeIdle,
		parseArtifact.GetTracePanelLabelAfterNoop,
		parseArtifact.GetTracePanelLabelAfter,
		parseArtifact.GetTracePanelLabelAfterIdle,
		parseArtifact.GetTracePanelLabelAfterBurst,
		parseArtifact.GetTracePanelLabelBurstIdle,
		parseArtifact.GetWorkerFleetLabelBeforeIdle,
		parseArtifact.GetWorkerFleetLabelAfterNoop,
		parseArtifact.GetWorkerFleetLabelAfter,
		parseArtifact.GetWorkerFleetLabelAfterIdle,
		parseArtifact.GetWorkerFleetLabelAfterBurst,
		parseArtifact.GetWorkerFleetLabelBurstIdle,
		parseArtifact.GetOwnerStateAfterBurst,
		parseArtifact.GetOwnerPanelLabelAfter,
		parseArtifact.GetOwnerPanelLabelAfterIdle,
		parseArtifact.GetOwnerPanelLabelAfterBurst,
		parseArtifact.GetOwnerPanelLabelBurstIdle,
		parseArtifact.GetWorkbenchLabelBeforeNoop,
		parseArtifact.GetWorkbenchLabelAfterNoop,
		parseArtifact.GetWorkbenchLabelAfter,
		parseArtifact.GetWorkbenchLabelAfterIdle,
		parseArtifact.GetWorkbenchLabelAfterBurst,
		parseArtifact.GetWorkbenchLabelBurstIdle,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// TestExample200Runtime2Execution runs example 200 through Playwright-Go and verifies core runtime2 execution signals.
func TestExample200Runtime2Execution(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample200Runtime2MainWasm(parseT, getRepoRoot)
	buildExample200Runtime2WorkerWasm(parseT, getRepoRoot)
	getBaseURL := startExample200RuntimeServer(parseT, getRepoRoot, "18102")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		getArtifact := captureExample200RuntimeExecutionArtifact(parseT, parsePage, getBaseURL)
		parseT.Logf("example 200 runtime2 execution: %s", formatExample200RuntimeExecutionSummary(getArtifact))

		if getArtifact.GetInitialAppRenderPass != 1 {
			parseT.Fatalf("expected initial app render pass to be exactly 1, got %d", getArtifact.GetInitialAppRenderPass)
		}
		if getArtifact.GetSettledAppRenderPass != getArtifact.GetInitialAppRenderPass {
			parseT.Fatalf("expected settled app render pass to stay at %d, got %d", getArtifact.GetInitialAppRenderPass, getArtifact.GetSettledAppRenderPass)
		}
		if getArtifact.GetLoadRerenderDelta != 0 {
			parseT.Fatalf("expected zero startup app rerenders, got load delta %d", getArtifact.GetLoadRerenderDelta)
		}
		if getArtifact.GetOwnerState != 0 {
			parseT.Fatalf("expected initial owner state 0, got %d", getArtifact.GetOwnerState)
		}
		if getArtifact.GetAppLabelCountBefore > 1 {
			parseT.Fatalf("expected no idle app rerender spam before interaction, got app label count %d", getArtifact.GetAppLabelCountBefore)
		}
		if getArtifact.GetAppLabelCountBeforeIdle > getArtifact.GetAppLabelCountBefore {
			parseT.Fatalf(
				"expected app label count to stay stable while idle before interaction, got before=%d before-idle=%d",
				getArtifact.GetAppLabelCountBefore,
				getArtifact.GetAppLabelCountBeforeIdle,
			)
		}
		if getArtifact.GetAppLabelCountAfterNoop > getArtifact.GetAppLabelCountBeforeIdle {
			parseT.Fatalf(
				"expected no-op controls to keep app label count stable, got before-idle=%d after-noop=%d",
				getArtifact.GetAppLabelCountBeforeIdle,
				getArtifact.GetAppLabelCountAfterNoop,
			)
		}
		if getArtifact.GetWorkbenchLabelAfterNoop > getArtifact.GetWorkbenchLabelBeforeNoop {
			parseT.Fatalf(
				"expected no-op controls to keep workbench label count stable, got before-noop=%d after-noop=%d",
				getArtifact.GetWorkbenchLabelBeforeNoop,
				getArtifact.GetWorkbenchLabelAfterNoop,
			)
		}
		if getArtifact.GetInspectorLabelAfterNoop > getArtifact.GetInspectorLabelBeforeIdle {
			parseT.Fatalf(
				"expected no-op controls to keep inspector label count stable, got before-idle=%d after-noop=%d",
				getArtifact.GetInspectorLabelBeforeIdle,
				getArtifact.GetInspectorLabelAfterNoop,
			)
		}
		if getArtifact.GetRegionLabelAfterNoop > getArtifact.GetRegionLabelBeforeIdle {
			parseT.Fatalf(
				"expected no-op controls to keep region label count stable, got before-idle=%d after-noop=%d",
				getArtifact.GetRegionLabelBeforeIdle,
				getArtifact.GetRegionLabelAfterNoop,
			)
		}
		if getArtifact.GetTracePanelLabelAfterNoop > getArtifact.GetTracePanelLabelBeforeIdle {
			parseT.Fatalf(
				"expected no-op controls to keep trace label count stable, got before-idle=%d after-noop=%d",
				getArtifact.GetTracePanelLabelBeforeIdle,
				getArtifact.GetTracePanelLabelAfterNoop,
			)
		}
		if getArtifact.GetWorkerFleetLabelAfterNoop > getArtifact.GetWorkerFleetLabelBeforeIdle {
			parseT.Fatalf(
				"expected no-op controls to keep worker-fleet label count stable, got before-idle=%d after-noop=%d",
				getArtifact.GetWorkerFleetLabelBeforeIdle,
				getArtifact.GetWorkerFleetLabelAfterNoop,
			)
		}
		if !getArtifact.HasRuntime2Proof || !getArtifact.HasRuntime2ActiveRoute {
			parseT.Fatalf("runtime2 proof missing or inactive: %s", formatExample200RuntimeExecutionSummary(getArtifact))
		}
		if getArtifact.GetConsoleErrorCount > 0 || getArtifact.GetPageErrorCount > 0 {
			parseT.Fatalf("unexpected browser errors: %s", formatExample200RuntimeExecutionSummary(getArtifact))
		}
		if !getArtifact.HasWorkerBatchSuccess {
			parseT.Fatalf("expected worker batch success log after increment: %s", formatExample200RuntimeExecutionSummary(getArtifact))
		}
		if getArtifact.GetWorkerProbeDoneCount < getExample200ExpectedWorkerCount {
			parseT.Fatalf("expected at least %d worker probe-complete logs, got %d", getExample200ExpectedWorkerCount, getArtifact.GetWorkerProbeDoneCount)
		}
		if getArtifact.GetAppLabelCountAfter != 1 {
			parseT.Fatalf("expected app shell to remain at committed render count 1 after interaction-isolated updates, got %d", getArtifact.GetAppLabelCountAfter)
		}
		if getArtifact.GetAppLabelCountAfterIdle > getArtifact.GetAppLabelCountAfter {
			parseT.Fatalf(
				"expected app label count to stay stable while idle after one interaction, got after=%d after-idle=%d",
				getArtifact.GetAppLabelCountAfter,
				getArtifact.GetAppLabelCountAfterIdle,
			)
		}
		if getArtifact.GetOwnerPanelLabelAfterIdle > getArtifact.GetOwnerPanelLabelAfter {
			parseT.Fatalf(
				"expected owner panel label count to stay stable while idle after one interaction, got after=%d after-idle=%d",
				getArtifact.GetOwnerPanelLabelAfter,
				getArtifact.GetOwnerPanelLabelAfterIdle,
			)
		}
		if getArtifact.GetWorkbenchLabelAfterIdle > getArtifact.GetWorkbenchLabelAfter {
			parseT.Fatalf(
				"expected workbench label count to stay stable while idle after one interaction, got after=%d after-idle=%d",
				getArtifact.GetWorkbenchLabelAfter,
				getArtifact.GetWorkbenchLabelAfterIdle,
			)
		}
		if getArtifact.GetInspectorLabelAfterIdle > getArtifact.GetInspectorLabelAfter {
			parseT.Fatalf(
				"expected inspector label count to stay stable while idle after one interaction, got after=%d after-idle=%d",
				getArtifact.GetInspectorLabelAfter,
				getArtifact.GetInspectorLabelAfterIdle,
			)
		}
		if getArtifact.GetRegionLabelAfterIdle > getArtifact.GetRegionLabelAfter {
			parseT.Fatalf(
				"expected region label count to stay stable while idle after one interaction, got after=%d after-idle=%d",
				getArtifact.GetRegionLabelAfter,
				getArtifact.GetRegionLabelAfterIdle,
			)
		}
		if getArtifact.GetTracePanelLabelAfterIdle > getArtifact.GetTracePanelLabelAfter {
			parseT.Fatalf(
				"expected trace label count to stay stable while idle after one interaction, got after=%d after-idle=%d",
				getArtifact.GetTracePanelLabelAfter,
				getArtifact.GetTracePanelLabelAfterIdle,
			)
		}
		if getArtifact.GetWorkerFleetLabelAfterIdle > getArtifact.GetWorkerFleetLabelAfter {
			parseT.Fatalf(
				"expected worker-fleet label count to stay stable while idle after one interaction, got after=%d after-idle=%d",
				getArtifact.GetWorkerFleetLabelAfter,
				getArtifact.GetWorkerFleetLabelAfterIdle,
			)
		}
		if getArtifact.GetOwnerStateAfterBurst < 9 {
			parseT.Fatalf("expected owner state >= 9 after burst increments, got %d", getArtifact.GetOwnerStateAfterBurst)
		}
		if getArtifact.GetAppLabelCountAfterBurst > getArtifact.GetAppLabelCountAfterIdle {
			parseT.Fatalf(
				"expected app label count to stay stable through burst increments, got after-idle=%d after-burst=%d",
				getArtifact.GetAppLabelCountAfterIdle,
				getArtifact.GetAppLabelCountAfterBurst,
			)
		}
		if getArtifact.GetAppLabelCountBurstIdle > getArtifact.GetAppLabelCountAfterBurst {
			parseT.Fatalf(
				"expected app label count to stay stable after burst idle window, got after-burst=%d burst-idle=%d",
				getArtifact.GetAppLabelCountAfterBurst,
				getArtifact.GetAppLabelCountBurstIdle,
			)
		}
		if getArtifact.GetOwnerPanelLabelBurstIdle > getArtifact.GetOwnerPanelLabelAfterBurst {
			parseT.Fatalf(
				"expected owner panel label count to stay stable after burst idle window, got after-burst=%d burst-idle=%d",
				getArtifact.GetOwnerPanelLabelAfterBurst,
				getArtifact.GetOwnerPanelLabelBurstIdle,
			)
		}
		if getArtifact.GetWorkbenchLabelBurstIdle > getArtifact.GetWorkbenchLabelAfterBurst {
			parseT.Fatalf(
				"expected workbench label count to stay stable after burst idle window, got after-burst=%d burst-idle=%d",
				getArtifact.GetWorkbenchLabelAfterBurst,
				getArtifact.GetWorkbenchLabelBurstIdle,
			)
		}
		if getArtifact.GetInspectorLabelBurstIdle > getArtifact.GetInspectorLabelAfterIdle {
			parseT.Fatalf(
				"expected inspector label count to stay stable during long post-burst idle window, got after-idle=%d burst-idle=%d",
				getArtifact.GetInspectorLabelAfterIdle,
				getArtifact.GetInspectorLabelBurstIdle,
			)
		}
		if getArtifact.GetRegionLabelBurstIdle > getArtifact.GetRegionLabelAfterBurst {
			parseT.Fatalf(
				"expected region label count to stay stable during long post-burst idle window, got after-burst=%d burst-idle=%d",
				getArtifact.GetRegionLabelAfterBurst,
				getArtifact.GetRegionLabelBurstIdle,
			)
		}
		if getArtifact.GetTracePanelLabelBurstIdle > getArtifact.GetTracePanelLabelAfterBurst {
			parseT.Fatalf(
				"expected trace label count to stay stable during long post-burst idle window, got after-burst=%d burst-idle=%d",
				getArtifact.GetTracePanelLabelAfterBurst,
				getArtifact.GetTracePanelLabelBurstIdle,
			)
		}
		if getArtifact.GetWorkerFleetLabelBurstIdle > getArtifact.GetWorkerFleetLabelAfterBurst {
			parseT.Fatalf(
				"expected worker-fleet label count to stay stable during long post-burst idle window, got after-burst=%d burst-idle=%d",
				getArtifact.GetWorkerFleetLabelAfterBurst,
				getArtifact.GetWorkerFleetLabelBurstIdle,
			)
		}
	})
}
