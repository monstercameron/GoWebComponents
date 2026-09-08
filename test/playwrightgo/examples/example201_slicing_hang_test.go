//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// TestExample201SlicingHang finds the mechanism behind the frame-budget hang.
//
// Enabling FrameBudgetMs makes the Example 201 core-* scenarios fail: 0 of 40
// items appear within a 5-second wait, in both builds. Five seconds for forty
// items is not slowness, and the native slicing tests pass with a budget set so
// small it yields after every unit — so the cause is browser-specific and
// invisible to the native suite.
//
// Two shapes fit "nothing rendered": the work loop never resumes after a yield
// (no further passes), or it resumes and makes no progress (passes climb, units
// do not). The dev build exposes __gwcPhaseTotals, which counts both, so
// sampling it while the scenario is stuck separates them.
func TestExample201SlicingHang(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasmDev(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18109")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(getBaseURL+"/examples/testing/render-benchmark/runtime/",
			playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
			parseT.Fatalf("goto subject: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(
			"() => !!window.__example201Subject && window.__example201Subject.isReady && typeof window.__gwcPhaseTotals === 'function'",
			nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(60000)}); parseErr != nil {
			parseT.Fatalf("development subject or required phase totals unavailable: %v", parseErr)
		}

		// Run core-render WITHOUT the harness's wait, sampling the runtime's own
		// counters while it should be working.
		parseRaw, parseErr := parsePage.Evaluate(`async () => {
			const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
			const read = () => {
				const t = JSON.parse(window.__gwcPhaseTotals());
				return {
					passes: t.workLoopPasses, units: t.processedUnits, commits: t.commitCount,
					items: document.querySelectorAll('.benchmark-core-item').length,
				};
			};
			const samples = [read()];
			// Kick the scenario but do not await it; its internal wait would throw
			// before anything could be sampled.
			const running = window.__example201Subject
				.measureScenario('core-render', { iterations: 1, warmups: 0 })
				.catch((e) => String(e));
			for (let i = 0; i < 6; i++) {
				await sleep(700);
				samples.push(read());
			}
			const outcome = await Promise.race([running, sleep(6000).then(() => 'still running')]);
			return JSON.stringify({ samples, outcome: String(outcome).slice(0, 160) });
		}`)
		if parseErr != nil {
			parseT.Fatalf("probe: %v", parseErr)
		}

		var parseResult struct {
			Samples []struct {
				Passes  int `json:"passes"`
				Units   int `json:"units"`
				Commits int `json:"commits"`
				Items   int `json:"items"`
			} `json:"samples"`
			Outcome string `json:"outcome"`
		}
		if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseResult); parseErr != nil {
			parseT.Fatalf("decode: %v", parseErr)
		}

		parseT.Logf("%-8s %10s %10s %10s %8s", "sample", "passes", "units", "commits", "items")
		for parseIndex, parseSample := range parseResult.Samples {
			parseT.Logf("%-8d %10d %10d %10d %8d",
				parseIndex, parseSample.Passes, parseSample.Units, parseSample.Commits, parseSample.Items)
		}
		parseT.Logf("outcome: %s", parseResult.Outcome)
	})
}
