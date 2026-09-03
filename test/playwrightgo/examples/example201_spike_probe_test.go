//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"runtime"
	"sort"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// Where the Initial Render spike comes from.
//
// The scored report puts content-render at ~14-17ms for GWC against React's
// ~2.2ms. That number is a mean over seven samples that look like
// [4, 4, 4.7, 5.1, 6, 6.1, 88.9]: a ~4ms median with one ~75ms outlier, and the
// outlier reproduces in every run. React's own outliers form a two-sample
// cluster, so the report's bimodal detection fires and scores React on its low
// cluster; GWC's lone spike is not a cluster, so GWC is scored on the mean. The
// asymmetry is worth roughly 3x on that scenario.
//
// What the spike IS could not be read off the existing reports:
//
//   - The long-task metric says zero for all 114 scenario rows across three
//     runs, including the row containing an 88.9ms sample. Headless Chromium
//     does not list "longtask" in supportedEntryTypes, so the probe disables
//     itself and reports 0. It cannot distinguish waiting from computing.
//   - The dev-build phase and GC probes show content-render at a 3.2ms mean,
//     4.4ms max, and zero Go GCs — no spike at all, under both GOGC settings.
//
// So the spike lives in something the scored run does and the probes do not.
// The two candidates are the BUILD PROFILE (scored runs use -profile benchmark,
// i.e. -tags production; probes use development) and the SHARED PAGE (scored
// runs alternate React and GWC subjects; probes load the GWC subject alone).
//
// This probe isolates the first: production build, GWC subject alone, and the
// per-iteration sequence printed IN ORDER rather than summarized. Order matters
// — a spike in position 0 is a cold-start cost, a periodic spike is collection
// or growth, and a spike at a random position is scheduling.
//
// WHAT THESE TWO PROBES ESTABLISHED (2026-07-25), so the next reader does not
// repeat the eliminations:
//
//   - The outlier is NOT React interference. It reproduces with the GWC subject
//     alone on the page: content-render, 40 samples, median 3.3ms, one 27.4ms
//     sample at position 33.
//   - It is NOT cold start. Position 33 of 40, and it moves between runs.
//   - It is NOT generic to GWC rendering. hooks-render, over the same 40
//     samples, produced no sample above 4x its median.
//   - It is probably NOT a Go collection, which was the leading hypothesis.
//     GOGC=50 collects over a heap roughly a third the size GOGC=300 allows, so
//     a collection-shaped outlier should have shrunk accordingly. Measured over
//     60 samples it went 28.4ms -> 20.7ms and still fired once per run. That is
//     too small a move to carry the explanation.
//
// Still unnamed, therefore. It is rare (~1 in 40-60), costs 20-30ms in
// isolation, and inflates to 75-90ms in the full scored run, where the page has
// hosted 19 scenarios across two frameworks. Naming it needs a profiler
// correlated to the specific iteration; the long-task path cannot help, because
// headless Chromium does not implement it.
//
// The REPORTING consequence is separable from the cause, and is the part worth
// acting on: with seven samples per scenario, one such outlier moves the mean by
// 3x, and the report scores GWC on its mean while scoring React on its low
// cluster. Across three runs the same-run geomean is 0.648/0.753/0.645 on the
// scored statistic and 0.692/0.827/0.722 on the median.

// TestExample201SpikeSequenceProductionAlone runs content-render one iteration
// at a time so each sample can be attributed to its position in the sequence.
func TestExample201SpikeSequenceProductionAlone(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18106")

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

		for _, getScenarioID := range []string{"content-render", "hooks-render"} {
			// iterations:1 / warmups:0 per call, so each returned mean IS one
			// sample and the loop preserves order. Asking the subject for 40
			// iterations would hand back a sorted summary, which is exactly the
			// information being sought here.
			getRaw, parseErr := parsePage.Evaluate(`async (parseScenarioID) => {
				const getSamples = [];
				for (let parseIndex = 0; parseIndex < 40; parseIndex++) {
					const getSummary = await window.__example201Subject.measureScenario(parseScenarioID, { iterations: 1, warmups: 0 });
					getSamples.push(getSummary.getDomReadyMeanMs);
				}
				return JSON.stringify(getSamples);
			}`, getScenarioID)
			if parseErr != nil {
				parseT.Fatalf("run %s: %v", getScenarioID, parseErr)
			}

			var getSamples []float64
			if parseErr2 := json.Unmarshal([]byte(getRaw.(string)), &getSamples); parseErr2 != nil {
				parseT.Fatalf("decode samples: %v", parseErr2)
			}
			if len(getSamples) == 0 {
				parseT.Fatalf("%s produced no samples", getScenarioID)
			}

			getSorted := append([]float64(nil), getSamples...)
			sort.Float64s(getSorted)
			getMedian := getSorted[len(getSorted)/2]

			parseT.Logf("%s, production build, GWC alone — 40 samples in order:", getScenarioID)
			parseT.Logf("  %v", roundAll(getSamples))
			parseT.Logf("  median=%.2fms  min=%.2fms  max=%.2fms", getMedian, getSorted[0], getSorted[len(getSorted)-1])

			// Report every sample more than 4x the median, with its position.
			// Position is the diagnosis: 0 means cold start, evenly spaced means
			// collection or growth, scattered means scheduling.
			getOutlierPositions := []int{}
			for parseIndex, getSample := range getSamples {
				if getSample > 4*getMedian {
					getOutlierPositions = append(getOutlierPositions, parseIndex)
				}
			}
			if len(getOutlierPositions) == 0 {
				parseT.Logf("  no sample exceeded 4x the median — the spike does NOT reproduce with the production build alone, so it needs the shared React page")
				continue
			}
			parseT.Logf("  outliers (>4x median) at positions %v of 0..39", getOutlierPositions)
		}
	})
}

// roundAll trims samples to two decimals so a 40-sample line stays readable.
func roundAll(parseSamples []float64) []float64 {
	getOut := make([]float64, len(parseSamples))
	for parseIndex, getSample := range parseSamples {
		getOut[parseIndex] = float64(int(getSample*100+0.5)) / 100
	}
	return getOut
}

// TestExample201SpikeAgainstGCPacing asks whether the outlier is a Go
// collection.
//
// TestExample201SpikeSequenceProductionAlone establishes the shape: in a
// production build with no React on the page, content-render is a 3.3ms median
// with a single ~27ms sample at position 33 of 40 — rare (about one in forty),
// expensive, and at no fixed position. hooks-render, which allocates less per
// iteration, produces no outlier at all. That profile fits a collection: rare
// because GOGC=300 lets the heap grow to four times live before collecting,
// expensive because the heap is large by then, and unpredictable in position
// because it depends on cumulative allocation rather than iteration count.
//
// The prediction is falsifiable. Lowering GOGC collects more often over a
// smaller heap, so if the outlier is a collection its height must fall; if it
// is anything else — scheduling, wasm memory growth, a V8 collection — GOGC
// moves it very little. The subject reads localStorage["gwc:gogc"] at boot, so
// the two arms differ by one reload.
func TestExample201SpikeAgainstGCPacing(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18107")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		for _, getMode := range []struct {
			GetLabel string
			GetGOGC  string
		}{{"gwc default (GOGC=300)", ""}, {"GOGC=50", "50"}} {
			if _, parseErr := parsePage.Goto(getBaseURL+"/examples/testing/render-benchmark/runtime/", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto subject: %v", parseErr)
			}
			if getMode.GetGOGC == "" {
				if _, parseErr := parsePage.Evaluate(`() => localStorage.removeItem('gwc:gogc')`); parseErr != nil {
					parseT.Fatalf("clear gogc: %v", parseErr)
				}
			} else if _, parseErr := parsePage.Evaluate(`(v) => localStorage.setItem('gwc:gogc', v)`, getMode.GetGOGC); parseErr != nil {
				parseT.Fatalf("seed gogc: %v", parseErr)
			}
			if _, parseErr := parsePage.Reload(); parseErr != nil {
				parseT.Fatalf("reload: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction("() => !!window.__example201Subject && window.__example201Subject.isReady", nil, playwright.PageWaitForFunctionOptions{
				Timeout: playwright.Float(30000),
			}); parseErr != nil {
				parseT.Fatalf("wait for subject: %v", parseErr)
			}

			getRaw, parseErr := parsePage.Evaluate(`async () => {
				const getSamples = [];
				for (let parseIndex = 0; parseIndex < 60; parseIndex++) {
					const getSummary = await window.__example201Subject.measureScenario("content-render", { iterations: 1, warmups: 0 });
					getSamples.push(getSummary.getDomReadyMeanMs);
				}
				return JSON.stringify(getSamples);
			}`)
			if parseErr != nil {
				parseT.Fatalf("run content-render: %v", parseErr)
			}
			var getSamples []float64
			if parseErr2 := json.Unmarshal([]byte(getRaw.(string)), &getSamples); parseErr2 != nil {
				parseT.Fatalf("decode samples: %v", parseErr2)
			}

			getSorted := append([]float64(nil), getSamples...)
			sort.Float64s(getSorted)
			getMedian := getSorted[len(getSorted)/2]
			getP95 := getSorted[int(0.95*float64(len(getSorted)))]
			getOver := 0
			for _, getSample := range getSamples {
				if getSample > 4*getMedian {
					getOver++
				}
			}
			parseT.Logf("%-24s median=%5.2fms  p95=%6.2fms  max=%6.2fms  samples over 4x median: %d of %d",
				getMode.GetLabel, getMedian, getP95, getSorted[len(getSorted)-1], getOver, len(getSamples))
		}
	})
}
