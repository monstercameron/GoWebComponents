//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

// Performance-budget gate for GoWebComponents examples.
//
// Budget file: testdata/perf_budgets.json — single source of truth.
// To ratchet budgets after a deliberate size change, measure the new artifact
// size with TestPerfBudgetWasmSize (logged as "actual"), update maxWasmBytes in
// perf_budgets.json to ~20-30% above the new measured value, and commit.
// (A -update-budgets flag that regenerates the file automatically could be
// added later, but the manual ratchet keeps diffs intentional and reviewable.)

import (
	"encoding/json"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

// perfBudget holds per-example thresholds used by the performance-budget gate.
//
// MaxWasmBytes is the strong, deterministic check: the on-disk app.wasm must
// not exceed this value. The gate would catch a large accidental dependency
// being linked in.
//
// MaxStartupMs is a generous browser-timing ceiling. It is measured but only
// fails on hardware-independent runaway slowdowns; normal machine variance
// will not trigger it.
type perfBudget struct {
	MaxWasmBytes int64 `json:"maxWasmBytes"`
	MaxStartupMs int64 `json:"maxStartupMs"`
}

// loadPerfBudgets reads and JSON-decodes the budget file at parsePath.
// It returns a map keyed by example slug (e.g. "counter").
func loadPerfBudgets(parsePath string) (map[string]perfBudget, error) {
	parseData, parseErr := os.ReadFile(parsePath)
	if parseErr != nil {
		return nil, parseErr
	}
	var parseBudgets map[string]perfBudget
	if parseErr := json.Unmarshal(parseData, &parseBudgets); parseErr != nil {
		return nil, parseErr
	}
	return parseBudgets, nil
}

// exceedsWasmBudget reports whether parseActualBytes is strictly greater than
// parseBudget. It is a pure function with no I/O, testable without any artifact
// on disk.
func exceedsWasmBudget(parseActualBytes, parseBudget int64) bool {
	return parseActualBytes > parseBudget
}

// wasmArtifactPath returns the on-disk path for the app.wasm of the given
// example slug under the public-examples-site asset tree.
func wasmArtifactPath(parseRepoRoot, parseSlug string) string {
	return filepath.Join(
		parseRepoRoot,
		"examples", "public-examples-site", "assets", "examples",
		parseSlug, "app.wasm",
	)
}

// budgetFilePath returns the path to the checked-in budget JSON relative to
// the test file's location (one level up from the test file).
func budgetFilePath(parseTestFile string) string {
	return filepath.Join(filepath.Dir(parseTestFile), "testdata", "perf_budgets.json")
}

// TestPerfBudgetWasmSize is the deterministic size gate. For each example in
// the budget file it stats the on-disk app.wasm and fails if the artifact
// exceeds MaxWasmBytes. No browser is needed.
//
// Budget rationale (counter):
//
//	Measured artifact size at time of gate creation: 6,368,199 bytes (~6.07 MB).
//	Budget set to 8,000,000 bytes (~7.63 MB), which is ~25.6% above the
//	measured size. This gives headroom for minor dependency growth while
//	catching a large regression (e.g. accidentally linking a new package).
func TestPerfBudgetWasmSize(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBudgetPath := budgetFilePath(parseFile)

	parseBudgets, parseLoadErr := loadPerfBudgets(parseBudgetPath)
	if parseLoadErr != nil {
		parseT.Fatalf("load perf budgets from %s: %v", parseBudgetPath, parseLoadErr)
	}

	for parseSlug, parseBudget := range parseBudgets {
		parseSlug := parseSlug
		parseBudget := parseBudget
		parseT.Run(parseSlug, func(parseT *testing.T) {
			parseArtifactPath := wasmArtifactPath(parseRepoRoot, parseSlug)
			parseInfo, parseStatErr := os.Stat(parseArtifactPath)
			if parseStatErr != nil {
				// The artifact is not present — it must be built first.
				// The public-examples-site tree is produced by `go run ./tools/gwc examples build`.
				// In CI it is always pre-built. Locally, run that command if this skips.
				parseT.Skipf(
					"wasm artifact not found at %s (run `go run ./tools/gwc examples build` to produce it): %v",
					parseArtifactPath, parseStatErr,
				)
				return
			}

			parseActual := parseInfo.Size()
			parseT.Logf("slug=%s  actual=%d bytes  budget=%d bytes", parseSlug, parseActual, parseBudget.MaxWasmBytes)

			if exceedsWasmBudget(parseActual, parseBudget.MaxWasmBytes) {
				parseT.Errorf(
					"OVER BUDGET: %s app.wasm is %d bytes, exceeds maxWasmBytes=%d (over by %d bytes)",
					parseSlug, parseActual, parseBudget.MaxWasmBytes, parseActual-parseBudget.MaxWasmBytes,
				)
			} else {
				parseT.Logf("within budget: %d / %d bytes (%.1f%% of budget)",
					parseActual, parseBudget.MaxWasmBytes,
					float64(parseActual)/float64(parseBudget.MaxWasmBytes)*100,
				)
			}
		})
	}
}

// TestPerfBudgetStartup measures the time from browser navigation to "app
// interactive" (when #app first has child elements). It logs the measured
// startup time and fails only if it exceeds MaxStartupMs.
//
// The test runs a few warm iterations and takes the minimum to reduce
// cold-cache variance. The MaxStartupMs ceiling in the budget file is
// deliberately generous (8 s default) so that normal machine-speed variation
// does not cause flakes.
func TestPerfBudgetStartup(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBudgetPath := budgetFilePath(parseFile)

	parseBudgets, parseLoadErr := loadPerfBudgets(parseBudgetPath)
	if parseLoadErr != nil {
		parseT.Fatalf("load perf budgets from %s: %v", parseBudgetPath, parseLoadErr)
	}

	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18277")

	if parseErr := ensureExamplesChromiumInstalled(); parseErr != nil {
		parseT.Fatalf("install chromium: %v", parseErr)
	}
	parsePw, parseErr := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if parseErr != nil {
		parseT.Fatalf("run playwright-go: %v", parseErr)
	}
	defer func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright-go: %v", parseStopErr)
		}
	}()

	parseBrowserHandle, parseErr := launchExamplesBrowser(parsePw, "chromium")
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer func() {
		if parseCloseErr := parseBrowserHandle.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	// wasmReadyJS returns true once #app has at least one child element,
	// which indicates the WASM runtime has finished its first render.
	const wasmReadyJS = `() => {
		const parseApp = document.getElementById('app');
		return !!(parseApp && parseApp.children.length > 0);
	}`

	// measureStartupMs loads parseURL in a fresh page and returns the
	// wall-clock milliseconds from Goto call to WaitForFunction return.
	measureStartupMs := func(parseSlug string) (int64, error) {
		parsePage, parsePageErr := parseBrowserHandle.NewPage()
		if parsePageErr != nil {
			return 0, parsePageErr
		}
		defer func() { _ = parsePage.Close() }()

		parseURL := parseBaseURL + "/examples/public-examples-site/assets/examples/" + parseSlug + "/"
		parseStart := time.Now()
		if _, parseNavErr := parsePage.Goto(parseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(90000),
		}); parseNavErr != nil {
			return 0, parseNavErr
		}
		if _, parseWaitErr := parsePage.WaitForFunction(wasmReadyJS, nil, playwright.PageWaitForFunctionOptions{
			Timeout: playwright.Float(90000),
		}); parseWaitErr != nil {
			return 0, parseWaitErr
		}
		return time.Since(parseStart).Milliseconds(), nil
	}

	const parseWarmIterations = 3

	for parseSlug, parseBudget := range parseBudgets {
		parseSlug := parseSlug
		parseBudget := parseBudget
		parseT.Run(parseSlug, func(parseT *testing.T) {
			// Check that the artifact is present; if not, skip the startup test
			// for this slug (the wasm-size test already explains why it is absent).
			parseArtifactPath := wasmArtifactPath(parseRepoRoot, parseSlug)
			if _, parseStatErr := os.Stat(parseArtifactPath); parseStatErr != nil {
				parseT.Skipf("wasm artifact not found for %s, skipping startup measurement", parseSlug)
				return
			}

			var parseMinMs int64 = -1
			for parseI := 0; parseI < parseWarmIterations; parseI++ {
				parseMs, parseMeasureErr := measureStartupMs(parseSlug)
				if parseMeasureErr != nil {
					parseT.Logf("iteration %d measurement error (non-fatal): %v", parseI+1, parseMeasureErr)
					continue
				}
				parseT.Logf("iteration %d: %d ms", parseI+1, parseMs)
				if parseMinMs < 0 || parseMs < parseMinMs {
					parseMinMs = parseMs
				}
			}

			if parseMinMs < 0 {
				parseT.Fatalf("all startup measurement iterations failed for %s", parseSlug)
			}

			parseT.Logf("slug=%s  min startup=%d ms  budget=%d ms", parseSlug, parseMinMs, parseBudget.MaxStartupMs)
			if parseMinMs > parseBudget.MaxStartupMs {
				parseT.Errorf(
					"startup OVER BUDGET: %s min=%d ms exceeds maxStartupMs=%d (over by %d ms)",
					parseSlug, parseMinMs, parseBudget.MaxStartupMs, parseMinMs-parseBudget.MaxStartupMs,
				)
			}
		})
	}
}

// TestPerfBudgetGateCatchesRegression is a self-test proving the gate logic
// works without requiring a browser or the wasm artifact.
//
// It constructs extreme budgets and asserts that exceedsWasmBudget correctly
// classifies them:
//   - A budget of 1 byte must be exceeded by any real artifact.
//   - A budget far above the real artifact must not be exceeded.
//
// This test is always expected to pass (it has no external dependencies) and
// proves that a green TestPerfBudgetWasmSize is meaningful, not the result of
// a broken gate.
func TestPerfBudgetGateCatchesRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// Locate the counter artifact for a real size reading.
	parseArtifactPath := wasmArtifactPath(parseRepoRoot, "counter")
	parseInfo, parseStatErr := os.Stat(parseArtifactPath)

	// --- Pure-function logic tests (no artifact required) ---

	// exceedsWasmBudget must return true when actual > budget.
	if !exceedsWasmBudget(100, 99) {
		parseT.Error("exceedsWasmBudget(100, 99) should be true (actual > budget)")
	}
	// exceedsWasmBudget must return false when actual == budget.
	if exceedsWasmBudget(100, 100) {
		parseT.Error("exceedsWasmBudget(100, 100) should be false (at limit is OK)")
	}
	// exceedsWasmBudget must return false when actual < budget.
	if exceedsWasmBudget(99, 100) {
		parseT.Error("exceedsWasmBudget(99, 100) should be false (under budget)")
	}

	parseT.Log("pure-function gate logic: PASS")

	// --- Artifact-based regression scenarios ---
	// These only run when the artifact is present; if not, the pure-function
	// assertions above are still enough to prove the gate logic is correct.
	if parseStatErr != nil {
		parseT.Logf(
			"counter artifact not found at %s; skipping artifact-based scenarios (pure-function checks already passed)",
			parseArtifactPath,
		)
		return
	}

	parseActualSize := parseInfo.Size()
	parseT.Logf("counter artifact size: %d bytes", parseActualSize)

	// Scenario 1: a deliberately tiny budget must be flagged as over-budget.
	const parseTinyBudget int64 = 1
	if !exceedsWasmBudget(parseActualSize, parseTinyBudget) {
		parseT.Errorf(
			"gate BROKEN: exceedsWasmBudget(%d, %d) returned false — a real artifact must exceed a 1-byte budget",
			parseActualSize, parseTinyBudget,
		)
	} else {
		parseT.Logf("scenario OVER-BUDGET (budget=1): correctly detected regression")
	}

	// Scenario 2: a budget far above the real size must not be flagged.
	const parseHighBudget int64 = 1_000_000_000 // 1 GB
	if exceedsWasmBudget(parseActualSize, parseHighBudget) {
		parseT.Errorf(
			"gate BROKEN: exceedsWasmBudget(%d, %d) returned true — real artifact must be within a 1 GB budget",
			parseActualSize, parseHighBudget,
		)
	} else {
		parseT.Logf("scenario UNDER-BUDGET (budget=1GB): correctly reported within budget")
	}

	parseT.Log("TestPerfBudgetGateCatchesRegression: all scenarios PASS")
}
