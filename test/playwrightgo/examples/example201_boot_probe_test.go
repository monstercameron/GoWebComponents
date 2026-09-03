//go:build playwrightgo

package playwrightgoexamples_test

import (
	"encoding/json"
	"fmt"
	"runtime"
	"sort"
	"testing"

	"github.com/mxschmitt/playwright-go"
)

// TestExample201BootProbe measures cold-start cost per framework: navigation
// to subject-ready (TTI proxy), plus the GWC-specific wasm fetch and
// instantiate+run split from resource timing. Diagnostic tooling, not an
// assertion — Example 201's scenario benchmarks measure steady state only,
// and boot is the classic wasm weak flank vs JS frameworks.
func TestExample201BootProbe(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	getRepoRoot := examplesRepoRootFromFile(parseFile)
	buildExample201BenchmarkWasm(parseT, getRepoRoot)
	getBaseURL := startExamplesCatalogServer(parseT, getRepoRoot, "18104")

	getTargets := []struct {
		GetLabel string
		GetPath  string
	}{
		{GetLabel: "runtime1", GetPath: "/examples/testing/render-benchmark/runtime/"},
		{GetLabel: "react", GetPath: "/examples/testing/render-benchmark/react/"},
	}
	const parseLoads = 5

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseT.Logf("%-10s %8s | %10s %10s %12s", "framework", "load", "ttiMs", "wasmFetch", "wasmBootMs")
		for _, getTarget := range getTargets {
			getTTIs := make([]float64, 0, parseLoads)
			for parseLoad := 0; parseLoad < parseLoads; parseLoad++ {
				if _, parseErr := parsePage.Goto(getBaseURL+getTarget.GetPath, playwright.PageGotoOptions{
					WaitUntil: playwright.WaitUntilStateDomcontentloaded,
				}); parseErr != nil {
					parseT.Fatalf("goto %s subject: %v", getTarget.GetLabel, parseErr)
				}
				// Stamp readiness the moment the predicate first sees it so the
				// polling interval does not inflate the reading.
				if _, parseErr := parsePage.WaitForFunction(
					"() => { const ok = !!window.__example201Subject && window.__example201Subject.isReady; if (ok && !window.__bootReadyAt) { window.__bootReadyAt = performance.now(); } return ok; }",
					nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(60000)},
				); parseErr != nil {
					parseT.Fatalf("wait for %s subject: %v", getTarget.GetLabel, parseErr)
				}
				getProbeValue, parseErr := parsePage.Evaluate(`() => {
					const getNav = performance.getEntriesByType('navigation')[0];
					const getWasm = performance.getEntriesByType('resource').find((e) => e.name.endsWith('.wasm'));
					return JSON.stringify({
						tti: window.__bootReadyAt || 0,
						navDom: getNav ? getNav.domContentLoadedEventEnd : 0,
						wasmStart: getWasm ? getWasm.startTime : -1,
						wasmEnd: getWasm ? getWasm.responseEnd : -1,
						wasmBytes: getWasm ? getWasm.transferSize : 0
					});
				}`)
				if parseErr != nil {
					parseT.Fatalf("read %s boot timings: %v", getTarget.GetLabel, parseErr)
				}
				getJSON, hasJSON := getProbeValue.(string)
				if !hasJSON {
					parseT.Fatalf("expected boot json, got %#v", getProbeValue)
				}
				var getProbe struct {
					TTI       float64 `json:"tti"`
					NavDom    float64 `json:"navDom"`
					WasmStart float64 `json:"wasmStart"`
					WasmEnd   float64 `json:"wasmEnd"`
					WasmBytes float64 `json:"wasmBytes"`
				}
				if parseErr2 := json.Unmarshal([]byte(getJSON), &getProbe); parseErr2 != nil {
					parseT.Fatalf("decode boot probe: %v\n%s", parseErr2, getJSON)
				}
				getTTIs = append(getTTIs, getProbe.TTI)
				getFetch, getBoot := "-", "-"
				if getProbe.WasmStart >= 0 {
					getFetch = fmt.Sprintf("%.1f", getProbe.WasmEnd-getProbe.WasmStart)
					getBoot = fmt.Sprintf("%.1f", getProbe.TTI-getProbe.WasmEnd)
				}
				parseT.Logf("%-10s %8d | %10.1f %10s %12s", getTarget.GetLabel, parseLoad+1, getProbe.TTI, getFetch, getBoot)
			}
			sort.Float64s(getTTIs)
			parseT.Logf("%-10s   median | %10.1f", getTarget.GetLabel, getTTIs[len(getTTIs)/2])
		}
	})
}
