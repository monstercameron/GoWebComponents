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
	Valid   bool `json:"valid"`
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
			Count  int     `json:"count"`
			WorstMs float64 `json:"worstMs"`
			Source string  `json:"source"`
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

// buildV5HarnessWasm compiles the subject app for the browser.
func buildV5HarnessWasm(parseT *testing.T, parseRepoRoot string, parseOutPath string) {
	parseT.Helper()
	parseCmd := exec.Command("go", "build", "-o", parseOutPath, "./examples/testing/v5-load-harness")
	parseCmd.Dir = parseRepoRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOut, parseErr := parseCmd.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build harness wasm: %v\n%s", parseErr, parseOut)
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

	if !parseVerdict.Passed {
		for _, parseFailure := range parseVerdict.Failures {
			parseT.Logf("baseline misses %s: %s", parseFailure.Metric, parseFailure.Reason)
		}
		fmt.Println("baseline captured with failing targets, as expected for v4")
	}
}
