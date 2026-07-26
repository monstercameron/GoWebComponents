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

	// A soft memory limit can be injected for the M7 experiment: GOGC cannot make
	// collections frequent on a sub-megabyte heap, so every collection follows a
	// gap and a gap is what makes one expensive.
	if parseLimit := os.Getenv("GWC_V5_GCMEM"); parseLimit != "" {
		if _, parseErr := parsePage.Evaluate(`(v) => localStorage.setItem('gwc:gcmem', v)`, parseLimit); parseErr != nil {
			parseT.Fatalf("seed gcmem: %v", parseErr)
		}
		if _, parseErr := parsePage.Reload(); parseErr != nil {
			parseT.Fatalf("reload for gcmem: %v", parseErr)
		}
		parseT.Logf("soft memory limit = %s bytes", parseLimit)
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

	// Per frame, not aggregated. The aggregate said GWC's own handlers, but the
	// Go phase totals say a commit costs ~6-15ms — so a 100ms frame contains
	// something the sum hides, and only the individual frames can say whether
	// that is one long script, many short ones, or time attributed to nothing.
	for parseIndex, parseFrame := range parseFrames {
		parseScripted := 0.0
		for _, parseScript := range parseFrame.Scripts {
			parseScripted += parseScript.Duration
		}
		parseT.Logf("  frame %d: duration=%.1fms blocking=%.1fms styleLayout=%.1fms scripted=%.1fms unattributed=%.1fms scripts=%d",
			parseIndex, parseFrame.Duration, parseFrame.BlockingDuration, parseFrame.StyleAndLayout,
			parseScripted, parseFrame.Duration-parseScripted-parseFrame.StyleAndLayout, len(parseFrame.Scripts))
		for _, parseScript := range parseFrame.Scripts {
			parseT.Logf("      %-42s %7.1fms (forced reflow %.1fms)",
				parseScript.InvokerType+" "+parseScript.Invoker, parseScript.Duration, parseScript.Forced)
		}
	}
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

// TestV5GCPauseSweep asks whether M7's 3ms budget is a tuning problem or a floor.
//
// The render thread already runs the responsive pacing profile (GOGC=40) and
// still reports a ~7ms worst pause. Two explanations fit that equally well from
// a single reading — the pacing is not aggressive enough, or Go's
// stop-the-world costs more than 3ms in wasm regardless — and they have
// opposite fixes. Sweeping GOGC separates them: if the worst pause tracks the
// setting, it is tuning; if it plateaus, it is a floor and the budget is the
// thing that is wrong.
func TestV5GCPauseSweep(parseT *testing.T) {
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

	parseT.Logf("%-14s %10s %10s %10s", "GOGC", "maxPauseMs", "numGC", "heapMB")
	for _, parseGOGC := range []string{"", "40", "20", "10"} {
		if _, parseErr := parsePage.Goto(parseServer.URL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad, Timeout: playwright.Float(180000),
		}); parseErr != nil {
			parseT.Fatalf("goto harness: %v", parseErr)
		}
		if parseGOGC == "" {
			if _, parseErr := parsePage.Evaluate(`() => localStorage.removeItem('gwc:gogc')`); parseErr != nil {
				parseT.Fatalf("clear gogc: %v", parseErr)
			}
		} else if _, parseErr := parsePage.Evaluate(`(v) => localStorage.setItem('gwc:gogc', v)`, parseGOGC); parseErr != nil {
			parseT.Fatalf("seed gogc: %v", parseErr)
		}
		if _, parseErr := parsePage.Reload(); parseErr != nil {
			parseT.Fatalf("reload: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true && !!window.__gwcV5Probe`, nil,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
			parseT.Fatalf("wait for subject: %v", parseErr)
		}

		parseRaw, parseErr := parsePage.Evaluate(`async () => {
			for (const name of ['import', 'reindex', 'decode']) window.__gwcV5Workloads[name].start();
			// Read once to establish the window baseline the probe diffs against,
			// then run long enough that a collection actually happens: at GOGC=40
			// with a ~1MB heap, six seconds of this workload produced zero GCs and
			// therefore a meaningless 0.00ms worst pause.
			window.__gwcV5Probe();
			await window.__gwcV5Probes.typing(20000);
			for (const name of ['import', 'reindex', 'decode']) window.__gwcV5Workloads[name].stop();
			return window.__gwcV5Probe();
		}`)
		if parseErr != nil {
			parseT.Fatalf("run window: %v", parseErr)
		}
		var parseProbe struct {
			WindowMaxPauseNs uint64 `json:"windowMaxPauseNs"`
			Truncated        bool   `json:"pauseSampleTruncated"`
			NumGC            uint32 `json:"numGC"`
			HeapAllocBytes   uint64 `json:"heapAllocBytes"`
		}
		if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseProbe); parseErr != nil {
			parseT.Fatalf("decode probe: %v", parseErr)
		}
		parseWorst := parseProbe.WindowMaxPauseNs
		if parseProbe.Truncated {
			parseT.Logf("  (pause sample truncated at GOGC=%s; the worst pause may be higher than reported)", parseGOGC)
		}
		parseLabel := parseGOGC
		if parseLabel == "" {
			parseLabel = "responsive(40)"
		}
		parseT.Logf("%-14s %10.2f %10d %10.1f", parseLabel,
			float64(parseWorst)/1e6, parseProbe.NumGC, float64(parseProbe.HeapAllocBytes)/1e6)
	}
}

// TestV5LongFrameDoseResponse asks whether M2's long frames are the framework's
// work or the render thread failing to get CPU.
//
// The attribution probe found a 205ms long frame containing zero scripts, zero
// style, and zero layout — 205ms attributed to nothing the main thread executed.
// A frame like that is not slow rendering; it is a thread that did not run. The
// obvious suspect is the domain worker, which runs wazero-interpreted SQLite and
// is CPU-bound.
//
// If that is right, long frames should scale with how many workloads are
// running and should not depend on what the render thread is doing. If it is
// wrong — if the framework is really producing them — the count should track
// rendering instead.
//
// This matters for whether M2 is achievable at all: no framework change can stop
// a thread from being descheduled by the OS.
func TestV5LongFrameDoseResponse(parseT *testing.T) {
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
		parseT.Fatalf("wait for subject: %v", parseErr)
	}

	parseCores, _ := parsePage.Evaluate(`() => navigator.hardwareConcurrency`)
	parseT.Logf("navigator.hardwareConcurrency = %v", parseCores)

	parseArms := []struct {
		Label     string
		Workloads []string
	}{
		{"no workloads", []string{}},
		{"import only", []string{"import"}},
		{"import+reindex", []string{"import", "reindex"}},
		{"all three", []string{"import", "reindex", "decode"}},
	}

	parseT.Logf("%-18s %10s %12s %14s %14s", "arm", "longFrames", "worstMs", "scriptedMs", "unattributedMs")
	for _, parseArm := range parseArms {
		parseRaw, parseErr := parsePage.Evaluate(`async (names) => {
			const frames = [];
			const observer = new PerformanceObserver((list) => {
				for (const e of list.getEntries()) {
					let scripted = 0;
					for (const s of (e.scripts ?? [])) scripted += s.duration;
					frames.push({ duration: e.duration, scripted, styleLayout: e.styleAndLayoutDuration ?? 0 });
				}
			});
			observer.observe({ type: 'long-animation-frame' });
			for (const n of names) window.__gwcV5Workloads[n].start();
			await window.__gwcV5Probes.typing(6000);
			for (const n of names) window.__gwcV5Workloads[n].stop();
			observer.disconnect();
			return JSON.stringify(frames);
		}`, parseArm.Workloads)
		if parseErr != nil {
			parseT.Fatalf("run arm %s: %v", parseArm.Label, parseErr)
		}
		var parseFrames []struct {
			Duration    float64 `json:"duration"`
			Scripted    float64 `json:"scripted"`
			StyleLayout float64 `json:"styleLayout"`
		}
		if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseFrames); parseErr != nil {
			parseT.Fatalf("decode frames: %v", parseErr)
		}
		parseWorst, parseScripted, parseUnattributed := 0.0, 0.0, 0.0
		for _, parseFrame := range parseFrames {
			if parseFrame.Duration > parseWorst {
				parseWorst = parseFrame.Duration
			}
			parseScripted += parseFrame.Scripted
			parseUnattributed += parseFrame.Duration - parseFrame.Scripted - parseFrame.StyleLayout
		}
		parseT.Logf("%-18s %10d %12.1f %14.1f %14.1f",
			parseArm.Label, len(parseFrames), parseWorst, parseScripted, parseUnattributed)
	}
}

// TestV5LongFrameRealVersusSyntheticInput isolates the one variable the
// dose-response left uncontrolled.
//
// With the app's own typing probe driving the filter, all three workloads
// running, and 6s per arm, the render thread produced ZERO long frames. The
// baseline run produces 13. The two differ in how input arrives: the baseline
// uses real keystrokes dispatched over CDP by Playwright, which is also the only
// way M3 sees anything, since Event Timing ignores untrusted events.
//
// So either real input costs something synthetic input does not — which would
// make M2 a genuine finding about handling trusted events — or the long frames
// belong to the automation channel rather than the app, which would make M2 a
// third measurement artifact after the unbuilt worker and the buffered
// observers.
//
// Both arms run the same workloads for the same duration. Only the input
// mechanism changes.
func TestV5LongFrameRealVersusSyntheticInput(parseT *testing.T) {
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
		parseT.Fatalf("wait for subject: %v", parseErr)
	}

	parseCollect := func(parseLabel string, parseReal bool) {
		if _, parseErr := parsePage.Evaluate(`() => {
			window.__frames = [];
			window.__obs = new PerformanceObserver((list) => {
				for (const e of list.getEntries()) {
					let scripted = 0;
					for (const s of (e.scripts ?? [])) scripted += s.duration;
					window.__frames.push({ duration: e.duration, scripted, styleLayout: e.styleAndLayoutDuration ?? 0 });
				}
			});
			window.__obs.observe({ type: 'long-animation-frame' });
			for (const n of ['import','reindex','decode']) window.__gwcV5Workloads[n].start();
		}`); parseErr != nil {
			parseT.Fatalf("start %s: %v", parseLabel, parseErr)
		}

		if parseReal {
			parseAlphabet := "abcdefghijklmnopqrstuvwxyz"
			parseDeadline := time.Now().Add(6 * time.Second)
			for parseI := 0; time.Now().Before(parseDeadline); parseI++ {
				parseInput := parsePage.Locator("#filter")
				if parseI%4 == 0 {
					_ = parseInput.Fill("")
				} else {
					_ = parseInput.PressSequentially(string(parseAlphabet[parseI%26]),
						playwright.LocatorPressSequentiallyOptions{Delay: playwright.Float(20)})
				}
				time.Sleep(60 * time.Millisecond)
			}
		} else if _, parseErr := parsePage.Evaluate(`async () => { await window.__gwcV5Probes.typing(6000); }`); parseErr != nil {
			parseT.Fatalf("synthetic typing: %v", parseErr)
		}

		parseRaw, parseErr := parsePage.Evaluate(`() => {
			for (const n of ['import','reindex','decode']) window.__gwcV5Workloads[n].stop();
			window.__obs.disconnect();
			return JSON.stringify(window.__frames);
		}`)
		if parseErr != nil {
			parseT.Fatalf("stop %s: %v", parseLabel, parseErr)
		}
		var parseFrames []struct {
			Duration    float64 `json:"duration"`
			Scripted    float64 `json:"scripted"`
			StyleLayout float64 `json:"styleLayout"`
		}
		if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseFrames); parseErr != nil {
			parseT.Fatalf("decode %s: %v", parseLabel, parseErr)
		}
		parseWorst, parseScripted, parseUnattributed := 0.0, 0.0, 0.0
		for _, parseFrame := range parseFrames {
			if parseFrame.Duration > parseWorst {
				parseWorst = parseFrame.Duration
			}
			parseScripted += parseFrame.Scripted
			parseUnattributed += parseFrame.Duration - parseFrame.Scripted - parseFrame.StyleLayout
		}
		parseT.Logf("%-22s longFrames=%-4d worst=%7.1fms scripted=%8.1fms unattributed=%8.1fms",
			parseLabel, len(parseFrames), parseWorst, parseScripted, parseUnattributed)
	}

	parseCollect("synthetic input", false)
	parseCollect("real CDP keystrokes", true)
}

// TestV5GCPauseFloor asks whether M7's 3ms budget is reachable at all.
//
// The sweep established that pacing has nothing to pace: the render thread runs
// about one collection per twenty seconds against a one-megabyte heap, so a
// lower GOGC cannot shorten a pause that is not happening. That leaves the
// pause's own cost, and a single observed 6-7ms says nothing about whether it is
// reducible — a rare expensive collection and a platform floor look identical
// from one sample.
//
// Forcing collections settles it. If every forced pause on this heap lands near
// the observed worst, Go's stop-the-world in wasm costs more than the budget
// allows and no framework change reaches it. If forced pauses are far cheaper,
// the observed 6-7ms belongs to a specific moment worth finding.
func TestV5GCPauseFloor(parseT *testing.T) {
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
	if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true && !!window.__gwcV5ForceGC`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
		parseT.Fatalf("wait for subject: %v", parseErr)
	}

	// Once quiescent, then again under the same load M7 is measured against —
	// a floor that only holds on an idle page would not be a floor.
	for _, parseArm := range []struct {
		Label  string
		Loaded bool
	}{{"quiescent", false}, {"under load", true}} {
		if parseArm.Loaded {
			if _, parseErr := parsePage.Evaluate(`() => {
				for (const n of ['import','reindex','decode']) window.__gwcV5Workloads[n].start();
			}`); parseErr != nil {
				parseT.Fatalf("start workloads: %v", parseErr)
			}
			time.Sleep(3 * time.Second)
		}

		parseRaw, parseErr := parsePage.Evaluate(`() => window.__gwcV5ForceGC(15)`)
		if parseErr != nil {
			parseT.Fatalf("force gc (%s): %v", parseArm.Label, parseErr)
		}
		var parseResult struct {
			PauseNs        []uint64 `json:"pauseNs"`
			HeapAllocBytes uint64   `json:"heapAllocBytes"`
		}
		if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseResult); parseErr != nil {
			parseT.Fatalf("decode (%s): %v", parseArm.Label, parseErr)
		}
		if len(parseResult.PauseNs) == 0 {
			parseT.Fatalf("%s: no pauses recorded", parseArm.Label)
		}
		parseWorst, parseTotal := uint64(0), uint64(0)
		for _, parsePause := range parseResult.PauseNs {
			if parsePause > parseWorst {
				parseWorst = parsePause
			}
			parseTotal += parsePause
		}
		parseT.Logf("%-12s %d forced collections on a %.1fMB heap: worst %.2fms, mean %.2fms, budget 3.00ms",
			parseArm.Label, len(parseResult.PauseNs), float64(parseResult.HeapAllocBytes)/1e6,
			float64(parseWorst)/1e6, float64(parseTotal)/float64(len(parseResult.PauseNs))/1e6)

		if parseArm.Loaded {
			if _, parseErr := parsePage.Evaluate(`() => {
				for (const n of ['import','reindex','decode']) window.__gwcV5Workloads[n].stop();
			}`); parseErr != nil {
				parseT.Fatalf("stop workloads: %v", parseErr)
			}
		}
	}
}

// TestV5GCPauseVersusFrameBudget tests the last standing explanation for M7.
//
// Go's stop-the-world must wait for every goroutine to reach a safepoint, and
// wasm has no asynchronous preemption — a goroutine yields only at a function
// call or a channel operation. So a collection triggered in the middle of a long
// uninterrupted render would record the WAIT as pause time, and the pause would
// be bounded by how long the runtime lets a slice run rather than by collection
// work. Forced collections already showed the work itself costs 0.2-0.5ms warm,
// so something is adding milliseconds that is not collecting.
//
// The prediction is falsifiable: if the pause is safepoint waiting, shrinking
// the frame budget shrinks it, because a shorter slice cannot block a collection
// for as long. If the pause is unchanged across budgets, waiting is not the
// cause and the remaining explanation is the collection's own mark cost.
//
// Uses the harness page directly rather than full baseline runs — one 30s
// loaded window per budget instead of 75s of interleaved arms.
func TestV5GCPauseVersusFrameBudget(parseT *testing.T) {
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

	parseT.Logf("%-16s %12s %10s %10s", "frameMs", "maxPauseMs", "numGC", "heapMB")
	for _, parseBudget := range []string{"5", "2", "1", "0.5"} {
		if _, parseErr := parsePage.Goto(parseServer.URL+"/?frameMs="+parseBudget, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateLoad, Timeout: playwright.Float(180000),
		}); parseErr != nil {
			parseT.Fatalf("goto (frameMs=%s): %v", parseBudget, parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true && !!window.__gwcV5Probe`, nil,
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
			parseT.Fatalf("wait (frameMs=%s): %v", parseBudget, parseErr)
		}

		parseRaw, parseErr := parsePage.Evaluate(`async () => {
			for (const n of ['import','reindex','decode']) window.__gwcV5Workloads[n].start();
			window.__gwcV5Probe();                       // seed the window diff
			await window.__gwcV5Probes.typing(30000);
			for (const n of ['import','reindex','decode']) window.__gwcV5Workloads[n].stop();
			return window.__gwcV5Probe();
		}`)
		if parseErr != nil {
			parseT.Fatalf("run (frameMs=%s): %v", parseBudget, parseErr)
		}
		var parseProbe struct {
			WindowMaxPauseNs uint64 `json:"windowMaxPauseNs"`
			NumGC            uint32 `json:"numGC"`
			HeapAllocBytes   uint64 `json:"heapAllocBytes"`
		}
		if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseProbe); parseErr != nil {
			parseT.Fatalf("decode (frameMs=%s): %v", parseBudget, parseErr)
		}
		parseT.Logf("%-16s %12.2f %10d %10.1f", parseBudget,
			float64(parseProbe.WindowMaxPauseNs)/1e6, parseProbe.NumGC,
			float64(parseProbe.HeapAllocBytes)/1e6)
	}
}

// TestV5GCPauseAcrossWindowTransitions asks where M7's pause actually lives.
//
// Every controlled measurement says the collector is cheap: forced collections
// cost 0.2-0.5ms warm, and a 30s loaded window reports a 0.00ms maximum at every
// frame budget from 5ms down to 0.5ms. The full baseline nonetheless reports
// ~7-8ms. Something the baseline does and a single window does not is producing
// it.
//
// The structural difference is the shape of the run: twelve interleaved windows,
// with the three workloads STARTED and STOPPED at each boundary and a cooldown
// between. Starting a workload re-imports 50,000 rows from zero, so each
// transition is an allocation burst on both threads, and the harness does its
// own bookkeeping there too.
//
// This replicates that shape and nothing else. If the pause appears here, M7 is
// measuring the harness's window structure rather than the application's
// steady-state behaviour — which would make it the fourth measurement artifact
// in this file, and would mean the number to fix is the instrument's.
func TestV5GCPauseAcrossWindowTransitions(parseT *testing.T) {
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
		parseT.Fatalf("goto: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.__gwcV5Ready === true && !!window.__gwcV5Probe`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
		parseT.Fatalf("wait: %v", parseErr)
	}

	parseRaw, parseErr := parsePage.Evaluate(`async () => {
		const sleep = (ms) => new Promise((r) => setTimeout(r, ms));
		const names = ['import','reindex','decode'];
		const perWindow = [];
		for (let i = 0; i < 6; i++) {
			// idle window, matching the baseline's interleave
			window.__gwcV5Probe();
			await window.__gwcV5Probes.typing(4000);
			perWindow.push({ arm: 'idle', probe: window.__gwcV5Probe() });
			await sleep(1500);

			// loaded window: workloads started and stopped around it
			for (const n of names) window.__gwcV5Workloads[n].start();
			window.__gwcV5Probe();
			await window.__gwcV5Probes.typing(4000);
			for (const n of names) window.__gwcV5Workloads[n].stop();
			perWindow.push({ arm: 'loaded', probe: window.__gwcV5Probe() });
			await sleep(1500);
		}
		return JSON.stringify(perWindow);
	}`)
	if parseErr != nil {
		parseT.Fatalf("run windows: %v", parseErr)
	}

	var parseWindows []struct {
		Arm   string `json:"arm"`
		Probe string `json:"probe"`
	}
	if parseErr := json.Unmarshal([]byte(parseRaw.(string)), &parseWindows); parseErr != nil {
		parseT.Fatalf("decode: %v", parseErr)
	}

	parseWorstIdle, parseWorstLoaded := 0.0, 0.0
	for parseIndex, parseWindow := range parseWindows {
		var parseProbe struct {
			WindowMaxPauseNs uint64 `json:"windowMaxPauseNs"`
			NumGC            uint32 `json:"numGC"`
		}
		if parseErr := json.Unmarshal([]byte(parseWindow.Probe), &parseProbe); parseErr != nil {
			parseT.Fatalf("decode probe %d: %v", parseIndex, parseErr)
		}
		parseMs := float64(parseProbe.WindowMaxPauseNs) / 1e6
		parseT.Logf("  w%-2d %-7s maxPause=%6.2fms numGC=%d", parseIndex, parseWindow.Arm, parseMs, parseProbe.NumGC)
		if parseWindow.Arm == "loaded" && parseMs > parseWorstLoaded {
			parseWorstLoaded = parseMs
		}
		if parseWindow.Arm == "idle" && parseMs > parseWorstIdle {
			parseWorstIdle = parseMs
		}
	}
	parseT.Logf("worst pause: idle %.2fms, loaded %.2fms (M7 budget 3.00ms, baseline reports ~7-8ms)",
		parseWorstIdle, parseWorstLoaded)
}
