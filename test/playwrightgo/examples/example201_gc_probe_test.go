//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestExample201GCTuningProbe A/Bs Go GC aggressiveness in the browser
// subject: default GOGC=100 vs the localStorage["benchmark:gogc"] override.
// Reports per-scenario GC collections, total pause, and DOM-ready mean/max —
// the tail-latency question, not the mean. Diagnostic only.
func TestExample201GCTuningProbe(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasmDev(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18105")

	getScenarios := []string{"core-update", "hooks-render", "core-stress-update", "content-render"}

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		for _, getMode := range []struct {
			GetLabel string
			GetGOGC  string
		}{{"go-default", "off"}, {"gwc-default", ""}} {
			if _, parseErr := parsePage.Goto(getBaseURL+"/examples/testing/render-benchmark/runtime/", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto subject: %v", parseErr)
			}
			if getMode.GetGOGC != "" {
				if _, parseErr := parsePage.Evaluate(`(v) => localStorage.setItem('gwc:gogc', v)`, getMode.GetGOGC); parseErr != nil {
					parseT.Fatalf("seed gogc: %v", parseErr)
				}
			} else {
				if _, parseErr := parsePage.Evaluate(`() => localStorage.removeItem('gwc:gogc')`); parseErr != nil {
					parseT.Fatalf("clear gogc: %v", parseErr)
				}
			}
			if _, parseErr := parsePage.Reload(); parseErr != nil {
				parseT.Fatalf("reload for gc mode: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction("() => !!window.__example201Subject && window.__example201Subject.isReady && typeof window.__gwcPhaseTotals === 'function'", nil, playwright.PageWaitForFunctionOptions{
				Timeout: playwright.Float(30000),
			}); parseErr != nil {
				parseT.Fatalf("wait for subject: %v", parseErr)
			}
			parseT.Logf("[%s] %-22s %10s %10s | %5s %10s", getMode.GetLabel, "scenario", "meanMs", "maxMs", "gc", "pauseMs")
			for _, getScenarioID := range getScenarios {
				getResultValue, parseErr := parsePage.Evaluate(`async (parseScenarioID) => {
					const getBefore = JSON.parse(window.__gwcPhaseTotals());
					const getSummary = await window.__example201Subject.measureScenario(parseScenarioID, { iterations: 9, warmups: 2 });
					const getAfter = JSON.parse(window.__gwcPhaseTotals());
					return JSON.stringify({
						mean: getSummary.getDomReadyMeanMs,
						max: getSummary.getDomReadyMaxMs,
						numGC: (getAfter.numGC || 0) - (getBefore.numGC || 0),
						pauseNs: (getAfter.gcPauseNs || 0) - (getBefore.gcPauseNs || 0)
					});
				}`, getScenarioID)
				if parseErr != nil {
					parseT.Fatalf("measure %s: %v", getScenarioID, parseErr)
				}
				getJSON, hasJSON := getResultValue.(string)
				if !hasJSON {
					parseT.Fatalf("expected probe json, got %#v", getResultValue)
				}
				var getProbe struct {
					Mean    float64 `json:"mean"`
					Max     float64 `json:"max"`
					NumGC   int64   `json:"numGC"`
					PauseNs int64   `json:"pauseNs"`
				}
				if parseErr2 := json.Unmarshal([]byte(getJSON), &getProbe); parseErr2 != nil {
					parseT.Fatalf("decode probe: %v\n%s", parseErr2, getJSON)
				}
				parseT.Logf("[%s] %-22s %10.1f %10.1f | %5d %10.2f", getMode.GetLabel, getScenarioID, getProbe.Mean, getProbe.Max, getProbe.NumGC, float64(getProbe.PauseNs)/1e6)
			}
		}
		// Leave no override behind for later tests sharing the origin.
		_, _ = parsePage.Evaluate(`() => localStorage.removeItem('gwc:gogc')`)
	})
}
