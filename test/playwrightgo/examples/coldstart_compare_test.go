//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"runtime"
	"sort"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

type coldStartSample struct {
	DocMs      float64 `json:"docMs"`
	AssetMs    float64 `json:"assetMs"`
	AssetKB    float64 `json:"assetKB"`
	ReadyMs    float64 `json:"readyMs"`
	CompressOK bool    `json:"compressOK"`
}

const coldStartReadyPredicate = `() => {
	if (window.__example201SubjectError) { return true; }
	const parseApp = document.getElementById('app');
	return !!(parseApp && parseApp.children.length > 0);
}`

const coldStartCollectScript = `() => {
	const parseNav = performance.getEntriesByType('navigation')[0];
	const parseAssets = performance.getEntriesByType('resource');
	const parseAsset = parseAssets.find(r => r.name.indexOf('render-benchmark.wasm') >= 0)
		|| parseAssets.find(r => r.name.indexOf('react.latest') >= 0);
	return JSON.stringify({
		docMs: parseNav ? parseNav.responseEnd : 0,
		assetMs: parseAsset ? (parseAsset.responseEnd - parseAsset.startTime) : 0,
		assetKB: parseAsset ? ((parseAsset.transferSize || parseAsset.encodedBodySize) / 1024) : 0,
		readyMs: performance.now(),
		compressOK: parseAsset ? (parseAsset.encodedBodySize > 0 && parseAsset.decodedBodySize > parseAsset.encodedBodySize) : false
	});
}`

func measureColdStart(parseT *testing.T, parsePage playwright.Page, parseURL string) coldStartSample {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("goto %s: %v", parseURL, parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(coldStartReadyPredicate, nil, playwright.PageWaitForFunctionOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait subject ready %s: %v", parseURL, parseErr)
	}
	parseRaw, parseErr := parsePage.Evaluate(coldStartCollectScript)
	if parseErr != nil {
		parseT.Fatalf("collect metrics %s: %v", parseURL, parseErr)
	}
	var parseSample coldStartSample
	if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseSample); parseErr != nil {
		parseT.Fatalf("decode metrics: %v", parseErr)
	}
	return parseSample
}

func coldStartMedian(parseSamples []float64) float64 {
	sort.Float64s(parseSamples)
	return parseSamples[len(parseSamples)/2]
}

// TestColdStartReactVsRuntime1 measures the cold-start waterfall (document,
// primary asset transfer, subject-ready) for the React and GWC runtime1
// benchmark subjects, first-visit (fresh context) and warm (reload).
func TestColdStartReactVsRuntime1(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping cold start comparison in short mode")
	}
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18102")

	parseSubjects := []struct {
		Name string
		URL  string
	}{
		{Name: "react", URL: getBaseURL + "/examples/testing/render-benchmark/react/index.html"},
		{Name: "gwc-runtime1", URL: getBaseURL + "/examples/testing/render-benchmark/runtime/index.html"},
	}

	if parseErr := ensureExamplesBrowserInstalled("chromium"); parseErr != nil {
		parseT.Fatalf("install chromium: %v", parseErr)
	}
	parsePw, parseErr := playwright.Run(&playwright.RunOptions{Browsers: []string{"chromium"}})
	if parseErr != nil {
		parseT.Fatalf("run playwright: %v", parseErr)
	}
	defer func() { _ = parsePw.Stop() }()
	parseBrowser, parseErr := launchExamplesBrowser(parsePw, "chromium")
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer func() { _ = parseBrowser.Close() }()

	const parseIterations = 3
	for _, parseSubject := range parseSubjects {
		var parseColdReady, parseColdAsset, parseWarmReady []float64
		var parseLastCold coldStartSample
		for parseIdx := 0; parseIdx < parseIterations; parseIdx++ {
			parseCtx, parseCtxErr := parseBrowser.NewContext()
			if parseCtxErr != nil {
				parseT.Fatalf("new context: %v", parseCtxErr)
			}
			parsePage, parsePageErr := parseCtx.NewPage()
			if parsePageErr != nil {
				parseT.Fatalf("new page: %v", parsePageErr)
			}
			parseCold := measureColdStart(parseT, parsePage, parseSubject.URL)
			parseColdReady = append(parseColdReady, parseCold.ReadyMs)
			parseColdAsset = append(parseColdAsset, parseCold.AssetMs)
			parseLastCold = parseCold

			// Warm: reload within the same context so the HTTP cache applies.
			parseWarm := measureColdStart(parseT, parsePage, parseSubject.URL)
			parseWarmReady = append(parseWarmReady, parseWarm.ReadyMs)
			_ = parseCtx.Close()
		}
		parseT.Logf("%s cold: ready=%.0fms (asset %.0fms, %.0f KB, compressed=%v) | warm reload: ready=%.0fms",
			parseSubject.Name,
			coldStartMedian(parseColdReady),
			coldStartMedian(parseColdAsset),
			parseLastCold.AssetKB,
			parseLastCold.CompressOK,
			coldStartMedian(parseWarmReady),
		)
	}
}
