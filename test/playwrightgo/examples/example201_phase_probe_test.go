//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"runtime"
	"sort"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestExample201PhaseTotalsProbe drives the runtime1 benchmark subject directly
// and splits each scenario's time into reconciler phases (render / diff /
// commit / effects) using the subject's __gwcPhaseTotals probe. Diagnostic
// tooling for optimization work: it reports where DOM-ready time goes rather
// than asserting perf numbers.
func TestExample201PhaseTotalsProbe(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasmDev(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18103")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(getBaseURL+"/examples/testing/render-benchmark/runtime/", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto runtime1 subject: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction("() => !!window.__example201Subject && window.__example201Subject.isReady && typeof window.__gwcPhaseTotals === 'function'", nil, playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(30000),
		}); parseErr != nil {
			parseT.Fatalf("wait for subject + probe: %v", parseErr)
		}

		getScenarios := []string{
			"core-render", "core-update", "core-stress-update", "core-append", "core-refresh", "content-refresh",
			"content-render", "content-update",
			"primitive-render", "primitive-text-update", "primitive-attribute-update", "primitive-remove",
			"deep-render", "deep-update",
			"enterprise-subtree-update", "hooks-render",
		}
		parseT.Logf("%-26s %9s | %9s %9s %9s %9s %9s | %6s %5s", "scenario", "domReady", "render", "diff", "commit", "effect", "phaseSum", "units", "iters")
		for _, getScenarioID := range getScenarios {
			getResultValue, parseErr := parsePage.Evaluate(`async (parseScenarioID) => {
				const getBefore = JSON.parse(window.__gwcPhaseTotals());
				const getSummary = await window.__example201Subject.measureScenario(parseScenarioID, { iterations: 7, warmups: 2 });
				const getAfter = JSON.parse(window.__gwcPhaseTotals());
				return JSON.stringify({
					domReadyMeanMs: getSummary.getDomReadyMeanMs,
					iterations: getSummary.getIterationCount,
					renderNs: getAfter.renderNs - getBefore.renderNs,
					diffNs: getAfter.diffNs - getBefore.diffNs,
					commitNs: getAfter.commitNs - getBefore.commitNs,
					effectNs: getAfter.effectNs - getBefore.effectNs,
					processedUnits: getAfter.processedUnits - getBefore.processedUnits,
					commitCount: getAfter.commitCount - getBefore.commitCount,
					numGC: (getAfter.numGC || 0) - (getBefore.numGC || 0),
					gcPauseNs: (getAfter.gcPauseNs || 0) - (getBefore.gcPauseNs || 0)
				});
			}`, getScenarioID)
			if parseErr != nil {
				parseT.Fatalf("measure %s: %v", getScenarioID, parseErr)
			}
			getResultJSON, hasResultJSON := getResultValue.(string)
			if !hasResultJSON {
				parseT.Fatalf("expected probe json for %s, got %#v", getScenarioID, getResultValue)
			}
			var getProbe struct {
				DomReadyMeanMs float64 `json:"domReadyMeanMs"`
				Iterations     int     `json:"iterations"`
				RenderNs       int64   `json:"renderNs"`
				DiffNs         int64   `json:"diffNs"`
				CommitNs       int64   `json:"commitNs"`
				EffectNs       int64   `json:"effectNs"`
				ProcessedUnits int64   `json:"processedUnits"`
				CommitCount    int64   `json:"commitCount"`
				NumGC          int64   `json:"numGC"`
				GCPauseNs      int64   `json:"gcPauseNs"`
			}
			if parseErr2 := json.Unmarshal([]byte(getResultJSON), &getProbe); parseErr2 != nil {
				parseT.Fatalf("decode probe for %s: %v\n%s", getScenarioID, parseErr2, getResultJSON)
			}
			// Phase deltas cover warmups + iterations; normalize per run so they
			// compare against the per-iteration DOM-ready mean.
			getRuns := float64(getProbe.Iterations + 2)
			getPerRunMs := func(parseNs int64) float64 { return float64(parseNs) / getRuns / 1e6 }
			getPhaseSum := getPerRunMs(getProbe.RenderNs) + getPerRunMs(getProbe.DiffNs) + getPerRunMs(getProbe.CommitNs) + getPerRunMs(getProbe.EffectNs)
			parseT.Logf("%-26s %8.3fms | %8.3fms %8.3fms %8.3fms %8.3fms %8.3fms | %6.0f %5d | gc %d (%.2fms)",
				getScenarioID,
				getProbe.DomReadyMeanMs,
				getPerRunMs(getProbe.RenderNs),
				getPerRunMs(getProbe.DiffNs),
				getPerRunMs(getProbe.CommitNs),
				getPerRunMs(getProbe.EffectNs),
				getPhaseSum,
				float64(getProbe.ProcessedUnits)/getRuns,
				getProbe.Iterations,
				getProbe.NumGC,
				float64(getProbe.GCPauseNs)/1e6,
			)
		}
	})
}

// TestExample201CPUProfileProbe captures a V8 CPU profile of the runtime1
// subject while it repeats one scenario, then reports the hottest functions by
// self time. Go wasm exports function names, so this attributes browser-side
// cost to reconciler internals. Diagnostic tooling for optimization work.
func TestExample201CPUProfileProbe(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasmDev(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18104")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(getBaseURL+"/examples/testing/render-benchmark/runtime/", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto runtime1 subject: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction("() => !!window.__example201Subject && window.__example201Subject.isReady", nil, playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(30000),
		}); parseErr != nil {
			parseT.Fatalf("wait for subject: %v", parseErr)
		}
		getSession, parseErr := parsePage.Context().NewCDPSession(parsePage)
		if parseErr != nil {
			parseT.Fatalf("cdp session: %v", parseErr)
		}
		if _, parseErr2 := getSession.Send("Profiler.enable", nil); parseErr2 != nil {
			parseT.Fatalf("profiler enable: %v", parseErr2)
		}
		if _, parseErr2 := getSession.Send("Profiler.setSamplingInterval", map[string]interface{}{"interval": 100}); parseErr2 != nil {
			parseT.Fatalf("profiler interval: %v", parseErr2)
		}
		// content-render, not core-render.
		//
		// Two reasons, and the first is that this probe was RED: the standalone
		// subject page cannot run the core-* scenarios — core-count comes back
		// zero without setup only the scored runner performs — so the profile
		// never ran. The second is that content-render is the scenario that
		// actually costs: it is GWC's worst against React and the one whose long
		// frames M2 counts.
		getScenario := "content-render"
		if _, parseErr2 := getSession.Send("Profiler.start", nil); parseErr2 != nil {
			parseT.Fatalf("profiler start: %v", parseErr2)
		}
		if _, parseErr2 := parsePage.Evaluate(`async (parseScenarioID) => {
			await window.__example201Subject.measureScenario(parseScenarioID, { iterations: 40, warmups: 3 });
		}`, getScenario); parseErr2 != nil {
			parseT.Fatalf("run scenario for profile: %v", parseErr2)
		}
		getProfileResult, parseErr := getSession.Send("Profiler.stop", nil)
		if parseErr != nil {
			parseT.Fatalf("profiler stop: %v", parseErr)
		}
		getProfileJSON, parseErr := json.Marshal(getProfileResult)
		if parseErr != nil {
			parseT.Fatalf("marshal profile: %v", parseErr)
		}
		var getPayload struct {
			Profile struct {
				Nodes []struct {
					ID        int `json:"id"`
					HitCount  int `json:"hitCount"`
					CallFrame struct {
						FunctionName string `json:"functionName"`
						URL          string `json:"url"`
					} `json:"callFrame"`
				} `json:"nodes"`
				StartTime float64 `json:"startTime"`
				EndTime   float64 `json:"endTime"`
				Samples   []int   `json:"samples"`
			} `json:"profile"`
		}
		if parseErr2 := json.Unmarshal(getProfileJSON, &getPayload); parseErr2 != nil {
			parseT.Fatalf("decode profile: %v", parseErr2)
		}
		getTotalHits := 0
		getHitsByName := map[string]int{}
		for _, getNode := range getPayload.Profile.Nodes {
			if getNode.HitCount == 0 {
				continue
			}
			getTotalHits += getNode.HitCount
			getName := getNode.CallFrame.FunctionName
			if getName == "" {
				getName = "(anonymous)"
			}
			getHitsByName[getName] += getNode.HitCount
		}
		type nameHits struct {
			GetName string
			GetHits int
		}
		getRanked := make([]nameHits, 0, len(getHitsByName))
		for getName, getHits := range getHitsByName {
			getRanked = append(getRanked, nameHits{GetName: getName, GetHits: getHits})
		}
		sort.Slice(getRanked, func(parseI, parseJ int) bool { return getRanked[parseI].GetHits > getRanked[parseJ].GetHits })
		parseT.Logf("scenario=%s total sampled hits=%d duration=%.1fms", getScenario, getTotalHits, (getPayload.Profile.EndTime-getPayload.Profile.StartTime)/1000)
		for parseIndex, getEntry := range getRanked {
			if parseIndex >= 45 {
				break
			}
			parseT.Logf("%6.2f%% %7d  %s", 100*float64(getEntry.GetHits)/float64(getTotalHits), getEntry.GetHits, getEntry.GetName)
		}
	})
}
