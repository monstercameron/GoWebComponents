//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"runtime"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// TestExample201LongFrameParity asks whether M2's budget of ZERO long frames is
// reachable by a framework at all.
//
// M2 is the only v5 budget written as an absolute with no tolerance; every other
// one is a percentile or a bound. Four long frames remain, and the scheduling
// levers are exhausted in both directions — slicing the work loop helped
// interaction latency, and aligning those slices to animation frames made
// everything worse (M2 4→17, M3 40→48ms, measured cool against cool).
//
// That leaves a question the plan cannot answer from GWC alone: is a frame over
// 50ms avoidable while doing this much work, or is it a property of the work?
// React 19 is the control. It runs the same nineteen scenarios on the same page
// with the same driver, so if React also produces frames over 50ms then zero is
// a claim about update size rather than about a framework, and M2's shape is
// wrong rather than GWC's performance.
//
// Deliberately NOT scored against the load harness: this isolates rendering,
// which is where the remaining long frames live.
func TestExample201LongFrameParity(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18108")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseT.Logf("%-10s %12s %10s %14s %12s", "subject", "longFrames", "worstMs", "totalScripted", "elapsedMs")
		for _, parseSubject := range []struct {
			Label string
			Path  string
		}{
			{"react", "/examples/testing/render-benchmark/react/"},
			{"gwc", "/examples/testing/render-benchmark/runtime/"},
		} {
			if _, parseErr := parsePage.Goto(getBaseURL+parseSubject.Path,
				playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
				parseT.Fatalf("goto %s: %v", parseSubject.Label, parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(
				"() => !!window.__example201Subject && window.__example201Subject.isReady", nil,
				playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(60000)}); parseErr != nil {
				parseT.Fatalf("wait for %s subject: %v", parseSubject.Label, parseErr)
			}

			parseRaw, parseErr := parsePage.Evaluate(`async () => {
				const frames = [];
				const observer = new PerformanceObserver((list) => {
					for (const e of list.getEntries()) {
						let scripted = 0;
						for (const s of (e.scripts ?? [])) scripted += s.duration;
						frames.push({ duration: e.duration, scripted });
					}
				});
				observer.observe({ type: 'long-animation-frame' });
				const started = performance.now();
				// A subset both subjects run standalone. enterprise-subtree-update
				// needs setup the scored runner provides and the bare subject page
				// does not, and it is the scenario, not the framework, that differs.
				await window.__example201Subject.measureAllScenarios({
					iterations: 7, warmups: 2, seed: 20101,
					scenarioIDs: ['content-render','content-update','content-refresh',
						'deep-render','deep-update','hooks-render',
						'primitive-render','primitive-append','primitive-remove'],
				});
				observer.disconnect();
				return JSON.stringify({ frames, elapsed: performance.now() - started });
			}`)
			if parseErr != nil {
				parseT.Fatalf("run %s: %v", parseSubject.Label, parseErr)
			}

			var parseResult struct {
				Frames []struct {
					Duration float64 `json:"duration"`
					Scripted float64 `json:"scripted"`
				} `json:"frames"`
				Elapsed float64 `json:"elapsed"`
			}
			if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseResult); parseErr != nil {
				parseT.Fatalf("decode %s: %v", parseSubject.Label, parseErr)
			}
			parseWorst, parseScripted := 0.0, 0.0
			for _, parseFrame := range parseResult.Frames {
				if parseFrame.Duration > parseWorst {
					parseWorst = parseFrame.Duration
				}
				parseScripted += parseFrame.Scripted
			}
			parseT.Logf("%-10s %12d %10.1f %14.1f %12.0f",
				parseSubject.Label, len(parseResult.Frames), parseWorst, parseScripted, parseResult.Elapsed)
		}
	})
}
