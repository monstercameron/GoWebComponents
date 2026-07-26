//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	"github.com/playwright-community/playwright-go"
)

// v5 P0.3 — capture the baseline the whole plan is sized against.
//
// Every target in the plan's success-criteria table currently reads
// "unmeasured". This drives the P0.2 subject app through the harness in real
// Chromium and writes the numbers to docs/benchmarks/v5-baseline.json.
//
// Run:
//   go test -tags playwrightgo ./test/playwrightgo/examples -run TestV5LoadHarnessBaseline -v
//
// It is deliberately NOT part of the default test lane: a full run is several
// minutes of deliberately heavy load, and R2 keeps every v5 flag off until this
// baseline exists to justify flipping it.

type v5HarnessVerdict struct {
	Passed   bool `json:"passed"`
	Failures []struct {
		Metric string `json:"metric"`
		Reason string `json:"reason"`
	} `json:"failures"`
}

type v5HarnessReport struct {
	Valid bool `json:"valid"`
	// WorkloadThroughput is how much background work the LOADED arm completed.
	//
	// Parsed because M1 is conditional on it. M1 asks whether a loaded frame is
	// equivalent to an idle one, and a loaded arm that ran no load answers yes
	// perfectly — a worker that failed to instantiate, a workload that errored
	// on its first call, or a message posted before onmessage existed all
	// produce a flawless result that measured an idle page twice.
	WorkloadThroughput []struct {
		Name      string   `json:"name"`
		Completed int      `json:"completed"`
		Windows   int      `json:"windows"`
		Errors    []string `json:"errors"`
	} `json:"workloadThroughput"`
	Metrics struct {
		M1 struct {
			Equivalent bool    `json:"equivalent"`
			MarginMs   float64 `json:"marginMs"`
			Idle       struct {
				N   int     `json:"n"`
				P50 float64 `json:"p50"`
				P95 float64 `json:"p95"`
				Max float64 `json:"max"`
			} `json:"idle"`
			Loaded struct {
				N   int     `json:"n"`
				P50 float64 `json:"p50"`
				P95 float64 `json:"p95"`
				Max float64 `json:"max"`
			} `json:"loaded"`
		} `json:"m1_frameTimeEquivalence"`
		M2 struct {
			Count   int     `json:"count"`
			WorstMs float64 `json:"worstMs"`
			Source  string  `json:"source"`
		} `json:"m2_longFrames"`
		M3 struct {
			N   int     `json:"n"`
			P95 float64 `json:"p95"`
			Max float64 `json:"max"`
		} `json:"m3_interactionLatency"`
		M7          float64 `json:"m7_maxGCPauseMs"`
		M7Truncated bool    `json:"m7_sampleTruncated"`
	} `json:"metrics"`
}

// buildV5HarnessWasm compiles the subject app AND its domain worker.
//
// Both, and that is the whole point. worker.js fetches ./v5services.wasm, and
// nothing ever built it — so the worker failed to instantiate on every run, the
// three background workloads never started, and the harness compared an idle
// page against an idle page. M1 reported perfect frame-time equivalence, which
// is exactly what measuring nothing twice looks like.
//
// The missing binary was invisible because the only evidence of it was a
// workload throughput of zero, and nothing read that.
func buildV5HarnessWasm(parseT *testing.T, parseRepoRoot string, parseOutPath string) {
	parseT.Helper()
	buildV5Wasm(parseT, parseRepoRoot, parseOutPath, "./examples/testing/v5-load-harness")
	buildV5Wasm(parseT, parseRepoRoot,
		filepath.Join(filepath.Dir(parseOutPath), "v5services.wasm"),
		"./examples/testing/v5-load-harness/services")
}

// buildV5Wasm compiles one package for js/wasm.
func buildV5Wasm(parseT *testing.T, parseRepoRoot string, parseOutPath string, parsePackage string) {
	parseT.Helper()
	parseCmd := exec.Command("go", "build", "-o", parseOutPath, parsePackage)
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOut, parseErr := parseCmd.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build %s for wasm: %v\n%s", parsePackage, parseErr, parseOut)
	}
}

// serveV5Harness serves the harness directory plus Go's wasm_exec.js shim.
//
// Self-contained rather than reusing the example catalog server: the harness
// needs exactly two things on one origin, and a local server keeps the run
// independent of catalog wiring.
func serveV5Harness(parseT *testing.T, parseRepoRoot string) *httptest.Server {
	parseT.Helper()
	parseDir := filepath.Join(parseRepoRoot, "examples", "testing", "v5-load-harness")
	parseShim := filepath.Join(runtime.GOROOT(), "lib", "wasm", "wasm_exec.js")
	if _, parseErr := os.Stat(parseShim); parseErr != nil {
		parseShim = filepath.Join(runtime.GOROOT(), "misc", "wasm", "wasm_exec.js")
	}

	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/wasm_exec.js", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "text/javascript")
		http.ServeFile(parseW, parseR, parseShim)
	})
	parseMux.Handle("/", http.FileServer(http.Dir(parseDir)))
	return httptest.NewServer(parseMux)
}

func TestV5LoadHarnessBaseline(parseT *testing.T) {
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", "..", ".."))
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}

	parseWasmPath := filepath.Join(parseRepoRoot, "examples", "testing", "v5-load-harness", "v5harness.wasm")
	buildV5HarnessWasm(parseT, parseRepoRoot, parseWasmPath)

	parseServer := serveV5Harness(parseT, parseRepoRoot)
	defer parseServer.Close()

	parsePW, parseErr := playwright.Run()
	if parseErr != nil {
		parseT.Skipf("playwright unavailable: %v", parseErr)
	}
	defer parsePW.Stop()

	parseBrowser, parseErr := parsePW.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer parseBrowser.Close()

	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new page: %v", parseErr)
	}

	parseHarnessDone := make(chan struct{})

	if _, parseErr := parsePage.Goto(parseServer.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(180000),
	}); parseErr != nil {
		parseT.Fatalf("goto harness: %v", parseErr)
	}

	// The app sets __gwcV5Ready after its first render, so the harness never
	// measures a tree that does not exist yet.
	if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
		parseT.Fatalf("wait for subject ready: %v", parseErr)
	}

	// Drive REAL keystrokes for the whole run.
	//
	// Event Timing only records trusted events, so the app cannot generate its
	// own interactions: a synthesized dispatchEvent yields no interactionId and
	// M3 measures an empty set. Playwright's keyboard produces genuine input,
	// which is the only way M3 sees anything at all.
	parseTypingDone := make(chan struct{})
	go func() {
		defer close(parseTypingDone)
		parseAlphabet := "abcdefghijklmnopqrstuvwxyz"
		for parseI := 0; ; parseI++ {
			select {
			case <-parseHarnessDone:
				return
			default:
			}
			parseInput := parsePage.Locator("#filter")
			// Alternate a matching prefix with a clear, so the filter actually
			// changes the visible set instead of no-opping.
			if parseI%4 == 0 {
				_ = parseInput.Fill("")
			} else {
				_ = parseInput.PressSequentially(string(parseAlphabet[parseI%26]),
					playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(20)})
			}
			time.Sleep(60 * time.Millisecond)
		}
	}()

	if parseErr := parsePage.Locator("#run").Click(); parseErr != nil {
		parseT.Fatalf("click run: %v", parseErr)
	}

	// Generous: 6 interleaved arms of 4s windows plus warmup and cooldowns,
	// under deliberately heavy load.
	if _, parseErr := parsePage.WaitForFunction(`() => !!window.__gwcV5Report`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(600000)}); parseErr != nil {
		parseT.Fatalf("wait for harness report: %v", parseErr)
	}

	close(parseHarnessDone)
	<-parseTypingDone

	parseRawReport, parseErr := parsePage.Evaluate(`() => JSON.stringify(window.__gwcV5Report)`)
	if parseErr != nil {
		parseT.Fatalf("read report: %v", parseErr)
	}
	parseRawVerdict, parseErr := parsePage.Evaluate(`() => JSON.stringify(window.__gwcV5Verdict)`)
	if parseErr != nil {
		parseT.Fatalf("read verdict: %v", parseErr)
	}

	var parseReport v5HarnessReport
	if parseErr := json.Unmarshal([]byte(parseRawReport.(string)), &parseReport); parseErr != nil {
		parseT.Fatalf("decode report: %v", parseErr)
	}
	var parseVerdict v5HarnessVerdict
	if parseErr := json.Unmarshal([]byte(parseRawVerdict.(string)), &parseVerdict); parseErr != nil {
		parseT.Fatalf("decode verdict: %v", parseErr)
	}

	parseT.Logf("valid=%t  M1 equivalent=%t  idle p95=%.2fms  loaded p95=%.2fms",
		parseReport.Valid, parseReport.Metrics.M1.Equivalent,
		parseReport.Metrics.M1.Idle.P95, parseReport.Metrics.M1.Loaded.P95)
	parseT.Logf("M2 long frames=%d (worst %.1fms, via %s)  M3 p95=%.1fms max=%.1fms n=%d  M7 max GC=%.2fms",
		parseReport.Metrics.M2.Count, parseReport.Metrics.M2.WorstMs, parseReport.Metrics.M2.Source,
		parseReport.Metrics.M3.P95, parseReport.Metrics.M3.Max, parseReport.Metrics.M3.N,
		parseReport.Metrics.M7)

	// Persist the baseline. This is the artifact, not the pass/fail: R2 keeps
	// every v5 flag off until these numbers exist, and a v4 run is EXPECTED to
	// fail its targets — that gap is the reason the version exists.
	parseBaselinePath := filepath.Join(parseRepoRoot, "docs", "benchmarks", "v5-baseline.json")
	parsePayload := map[string]any{
		"note":    "v4 baseline captured by P0.3. Failing targets here are expected; they are the gap v5 closes.",
		"valid":   parseReport.Valid,
		"verdict": parseVerdict,
		"report":  json.RawMessage(parseRawReport.(string)),
	}
	parseEncoded, parseErr := json.MarshalIndent(parsePayload, "", "  ")
	if parseErr != nil {
		parseT.Fatalf("encode baseline: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseBaselinePath, parseEncoded, 0o644); parseErr != nil {
		parseT.Fatalf("write baseline: %v", parseErr)
	}
	parseT.Logf("baseline written to %s", parseBaselinePath)

	// The run must be VALID. A drifted run says nothing about the code, and
	// recording one as a baseline would poison every later comparison.
	if !parseReport.Valid {
		parseT.Fatal("harness reported thermal/background drift; baseline discarded — re-run on a cooler machine")
	}
	// A blind observer would make M2 meaningless.
	if parseReport.Metrics.M2.Source == "none" {
		parseT.Fatal("no long-frame observer available in this browser; M2 is unmeasurable")
	}
	// No interactions means the probe never drove input, so M3 measured nothing.
	if parseReport.Metrics.M3.N == 0 {
		parseT.Fatal("probe recorded zero interactions; M3 measured nothing")
	}

	// And the load must have been real. This is checked LAST but matters MOST:
	// it is the only assertion here whose failure mode looks like success. Every
	// metric above passes at its best when the loaded arm did nothing, so
	// without this the strongest possible report is also the emptiest one.
	parseTotalCompleted := 0
	for _, parseWorkload := range parseReport.WorkloadThroughput {
		parseTotalCompleted += parseWorkload.Completed
		parseT.Logf("workload %-8s completed=%-8d windows=%d errors=%v",
			parseWorkload.Name, parseWorkload.Completed, parseWorkload.Windows, parseWorkload.Errors)
	}
	if len(parseReport.WorkloadThroughput) == 0 || parseTotalCompleted == 0 {
		parseT.Fatalf("the loaded arm completed no background work (%d workloads reporting); M1's equivalence is vacuous because there was no load to be equivalent to",
			len(parseReport.WorkloadThroughput))
	}
	parseT.Logf("loaded arm completed %d units of background work across %d workloads",
		parseTotalCompleted, len(parseReport.WorkloadThroughput))

	if !parseVerdict.Passed {
		for _, parseFailure := range parseVerdict.Failures {
			parseT.Logf("baseline misses %s: %s", parseFailure.Metric, parseFailure.Reason)
		}
		fmt.Println("baseline captured with failing targets, as expected for v4")
	}
}

// TestV5SchedulingComparison runs the harness twice on one machine -- v4
// behavior, then every v5 scheduling flag on -- and reports the delta.
//
// Back to back and in that order on purpose: this chassis is fanless and warms
// during a run, so running v5 second means any thermal drift works AGAINST the
// change rather than for it. A win measured that way is a floor, not a ceiling.
//
// R2 says a flag flips once its acceptance test passes. This is that test.
//
//	go test -tags playwrightgo ./test/playwrightgo/examples //	  -run TestV5SchedulingComparison -v -timeout 30m
func TestV5SchedulingComparison(parseT *testing.T) {
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", "..", ".."))
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}

	parseWasmPath := filepath.Join(parseRepoRoot, "examples", "testing", "v5-load-harness", "v5harness.wasm")
	buildV5HarnessWasm(parseT, parseRepoRoot, parseWasmPath)

	parseServer := serveV5Harness(parseT, parseRepoRoot)
	defer parseServer.Close()

	parsePW, parseErr := playwright.Run()
	if parseErr != nil {
		parseT.Skipf("playwright unavailable: %v", parseErr)
	}
	defer parsePW.Stop()

	parseBrowser, parseErr := parsePW.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer parseBrowser.Close()

	type armResult struct {
		label  string
		report v5HarnessReport
	}
	parseArms := []struct {
		label string
		query string
	}{
		{"v4 (flags off)", ""},
		{"v5 (all flags on)", "?v5=1"},
	}

	parseResults := make([]armResult, 0, len(parseArms))
	for _, parseArm := range parseArms {
		parsePage, parseErr := parseBrowser.NewPage()
		if parseErr != nil {
			parseT.Fatalf("new page: %v", parseErr)
		}

		parseHarnessDone := make(chan struct{})
		parseTypingDone := make(chan struct{})

		if _, parseErr := parsePage.Goto(parseServer.URL+parseArm.query, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad,
			Timeout:   playwright.Float(180000),
		}); parseErr != nil {
			parseT.Fatalf("goto %s: %v", parseArm.label, parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true`, nil,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
			parseT.Fatalf("wait ready %s: %v", parseArm.label, parseErr)
		}

		// Confirm the arm actually got the configuration it asked for, so a
		// flag that silently failed to apply cannot be reported as a win.
		parseRawConfig, parseErr := parsePage.Evaluate(`() => JSON.stringify(window.__gwcV5Config)`)
		if parseErr != nil {
			parseT.Fatalf("read config %s: %v", parseArm.label, parseErr)
		}
		parseT.Logf("%s config: %s", parseArm.label, parseRawConfig.(string))

		go func() {
			defer close(parseTypingDone)
			parseAlphabet := "abcdefghijklmnopqrstuvwxyz"
			for parseI := 0; ; parseI++ {
				select {
				case <-parseHarnessDone:
					return
				default:
				}
				parseInput := parsePage.Locator("#filter")
				if parseI%4 == 0 {
					_ = parseInput.Fill("")
				} else {
					_ = parseInput.PressSequentially(string(parseAlphabet[parseI%26]),
						playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(20)})
				}
				time.Sleep(60 * time.Millisecond)
			}
		}()

		if parseErr := parsePage.Locator("#run").Click(); parseErr != nil {
			parseT.Fatalf("click run %s: %v", parseArm.label, parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => !!window.__gwcV5Report`, nil,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(600000)}); parseErr != nil {
			parseT.Fatalf("wait report %s: %v", parseArm.label, parseErr)
		}
		close(parseHarnessDone)
		<-parseTypingDone

		parseRaw, parseErr := parsePage.Evaluate(`() => JSON.stringify(window.__gwcV5Report)`)
		if parseErr != nil {
			parseT.Fatalf("read report %s: %v", parseArm.label, parseErr)
		}
		var parseReport v5HarnessReport
		if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseReport); parseErr != nil {
			parseT.Fatalf("decode report %s: %v", parseArm.label, parseErr)
		}
		parseResults = append(parseResults, armResult{label: parseArm.label, report: parseReport})
		_ = parsePage.Close()
	}

	for _, parseResult := range parseResults {
		parseT.Logf("%-20s valid=%t  idle p95=%7.2fms  loaded p95=%9.2fms  longFrames=%4d  worst=%8.1fms  M3 p95=%7.1fms (n=%d)  GC max=%.2fms",
			parseResult.label,
			parseResult.report.Valid,
			parseResult.report.Metrics.M1.Idle.P95,
			parseResult.report.Metrics.M1.Loaded.P95,
			parseResult.report.Metrics.M2.Count,
			parseResult.report.Metrics.M2.WorstMs,
			parseResult.report.Metrics.M3.P95,
			parseResult.report.Metrics.M3.N,
			parseResult.report.Metrics.M7)
	}

	if len(parseResults) == 2 {
		parseV4, parseV5 := parseResults[0].report, parseResults[1].report
		parseT.Logf("delta loaded p95: %.2fms -> %.2fms", parseV4.Metrics.M1.Loaded.P95, parseV5.Metrics.M1.Loaded.P95)
		parseT.Logf("delta long frames: %d -> %d (worst %.1fms -> %.1fms)",
			parseV4.Metrics.M2.Count, parseV5.Metrics.M2.Count,
			parseV4.Metrics.M2.WorstMs, parseV5.Metrics.M2.WorstMs)
		parseT.Logf("delta interaction p95: %.1fms -> %.1fms", parseV4.Metrics.M3.P95, parseV5.Metrics.M3.P95)
	}
}

// TestV5WorkerMessageRate measures the half of "move the work off-thread" that
// moving the work does not fix.
//
// Relocating a workload to a worker moves the COMPUTE. It does not move the
// NOTIFICATION: every progress message is a wasm callback on the render thread
// plus one boundary crossing per field read. A worker that reports progress per
// batch, in a loop whose only pause is setTimeout(0), can therefore cost the
// render thread more than the work it took away — and it costs it in exactly the
// currency M2 and M3 are denominated in, small frequent interruptions rather
// than one long block.
//
// M1 stays green throughout, because a flood of 1 ms interruptions does not
// lengthen the p95 FRAME; it lengthens the time an interaction waits to be
// serviced. That is why this is measured separately rather than inferred from
// the frame metric.
func TestV5WorkerMessageRate(parseT *testing.T) {
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", "..", ".."))
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseWasmPath := filepath.Join(parseRepoRoot, "examples", "testing", "v5-load-harness", "v5harness.wasm")
	buildV5HarnessWasm(parseT, parseRepoRoot, parseWasmPath)

	parseServer := serveV5Harness(parseT, parseRepoRoot)
	defer parseServer.Close()

	parsePW, parseErr := playwright.Run()
	if parseErr != nil {
		parseT.Skipf("playwright unavailable: %v", parseErr)
	}
	defer parsePW.Stop()

	parseBrowser, parseErr := parsePW.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer parseBrowser.Close()

	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new page: %v", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseServer.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad, Timeout: playwright.Float(180000),
	}); parseErr != nil {
		parseT.Fatalf("goto harness: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true && !!window.__gwcV5WorkerMessages`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
		parseT.Fatalf("wait for subject ready: %v", parseErr)
	}

	parseRaw, parseErr := parsePage.Evaluate(`async () => {
		const before = window.__gwcV5WorkerMessages();
		for (const name of ['import', 'reindex', 'decode']) {
			window.__gwcV5Workloads[name].start();
		}
		await new Promise((r) => setTimeout(r, 4000));
		const after = window.__gwcV5WorkerMessages();
		for (const name of ['import', 'reindex', 'decode']) {
			window.__gwcV5Workloads[name].stop();
		}
		return JSON.stringify({ before, after });
	}`)
	if parseErr != nil {
		parseT.Fatalf("run workloads: %v", parseErr)
	}
	var parseCounts struct {
		Before int `json:"before"`
		After  int `json:"after"`
	}
	if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseCounts); parseErr != nil {
		parseT.Fatalf("decode counts: %v", parseErr)
	}

	parseDelivered := parseCounts.After - parseCounts.Before
	parsePerSecond := float64(parseDelivered) / 4.0
	parseT.Logf("worker delivered %d progress messages in 4s = %.0f/s to the render thread (about %.0f boundary crossings/s at ~5 reads each)",
		parseDelivered, parsePerSecond, parsePerSecond*5)

	if parseDelivered == 0 {
		parseT.Fatal("no progress messages were delivered; the workloads did not run, so this measured nothing")
	}
}

// TestV5LongFrameAttribution asks what is actually IN the long frames M2 counts.
//
// The workloads run in a worker, so the render thread should not be executing
// them. Something still produced 29 long frames on it during the loaded arm
// against 1 during the idle arm, and a count cannot say what. Long Animation
// Frame entries carry per-script attribution — name, invoker, duration, and the
// forced style/layout time that is the classic cause of a long frame no phase
// total explains — so this reads that rather than inferring from timing.
func TestV5LongFrameAttribution(parseT *testing.T) {
	parseRepoRoot, parseErr := filepath.Abs(filepath.Join("..", "..", ".."))
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}
	parseWasmPath := filepath.Join(parseRepoRoot, "examples", "testing", "v5-load-harness", "v5harness.wasm")
	buildV5HarnessWasm(parseT, parseRepoRoot, parseWasmPath)

	parseServer := serveV5Harness(parseT, parseRepoRoot)
	defer parseServer.Close()

	parsePW, parseErr := playwright.Run()
	if parseErr != nil {
		parseT.Skipf("playwright unavailable: %v", parseErr)
	}
	defer parsePW.Stop()
	parseBrowser, parseErr := parsePW.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer parseBrowser.Close()
	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new page: %v", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseServer.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad, Timeout: playwright.Float(180000),
	}); parseErr != nil {
		parseT.Fatalf("goto harness: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
		parseT.Fatalf("wait for subject ready: %v", parseErr)
	}

	// Collect from the moment the workloads start, and drive typing throughout so
	// the render thread is doing what it does during a measured window.
	parseStopTyping := make(chan struct{})
	parseTypingDone := make(chan struct{})
	go func() {
		defer close(parseTypingDone)
		parseAlphabet := "abcdefghijklmnopqrstuvwxyz"
		for parseI := 0; ; parseI++ {
			select {
			case <-parseStopTyping:
				return
			default:
			}
			parseInput := parsePage.Locator("#filter")
			if parseI%4 == 0 {
				_ = parseInput.Fill("")
			} else {
				_ = parseInput.PressSequentially(string(parseAlphabet[parseI%26]),
					playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(20)})
			}
			time.Sleep(60 * time.Millisecond)
		}
	}()

	parseRaw, parseErr := parsePage.Evaluate(`async () => {
		const frames = [];
		const observer = new PerformanceObserver((list) => {
			for (const entry of list.getEntries()) {
				frames.push({
					duration: entry.duration,
					blockingDuration: entry.blockingDuration ?? 0,
					styleAndLayout: entry.styleAndLayoutDuration ?? 0,
					scripts: (entry.scripts ?? []).map((s) => ({
						name: s.name, invoker: s.invoker, invokerType: s.invokerType,
						duration: s.duration, forced: s.forcedStyleAndLayoutDuration ?? 0,
					})),
				});
			}
		});
		observer.observe({ type: 'long-animation-frame' });
		for (const name of ['import', 'reindex', 'decode']) window.__gwcV5Workloads[name].start();
		await new Promise((r) => setTimeout(r, 8000));
		for (const name of ['import', 'reindex', 'decode']) window.__gwcV5Workloads[name].stop();
		observer.disconnect();
		return JSON.stringify(frames);
	}`)
	close(parseStopTyping)
	<-parseTypingDone
	if parseErr != nil {
		parseT.Fatalf("collect long frames: %v", parseErr)
	}

	var parseFrames []struct {
		Duration         float64 `json:"duration"`
		BlockingDuration float64 `json:"blockingDuration"`
		StyleAndLayout   float64 `json:"styleAndLayout"`
		Scripts          []struct {
			Name        string  `json:"name"`
			Invoker     string  `json:"invoker"`
			InvokerType string  `json:"invokerType"`
			Duration    float64 `json:"duration"`
			Forced      float64 `json:"forced"`
		} `json:"scripts"`
	}
	if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseFrames); parseErr != nil {
		parseT.Fatalf("decode frames: %v", parseErr)
	}

	parseT.Logf("%d long animation frames on the render thread during 8s of loaded typing", len(parseFrames))
	parseByInvoker := map[string]float64{}
	parseCountByInvoker := map[string]int{}
	parseTotalForced := 0.0
	for _, parseFrame := range parseFrames {
		parseTotalForced += parseFrame.StyleAndLayout
		for _, parseScript := range parseFrame.Scripts {
			parseKey := parseScript.InvokerType + " " + parseScript.Invoker
			parseByInvoker[parseKey] += parseScript.Duration
			parseCountByInvoker[parseKey]++
			parseTotalForced += parseScript.Forced
		}
	}
	for parseKey, parseMs := range parseByInvoker {
		parseT.Logf("  %-58s %6.1f ms across %d scripts", parseKey, parseMs, parseCountByInvoker[parseKey])
	}
	parseT.Logf("  total style/layout + forced reflow time: %.1f ms", parseTotalForced)
	if len(parseFrames) == 0 {
		parseT.Skip("no long frames observed in this run; nothing to attribute")
	}
}
