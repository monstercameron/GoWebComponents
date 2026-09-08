//go:build playwrightgo

package playwrightgoexamples_test

import (
	"net"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

// TestV6ServerInteractive verifies GWC controls reconcile real server SSE snapshots.
func TestV6ServerInteractive(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRoot := examplesRepoRootFromFile(parseFile)
	parseWasm := filepath.Join(parseT.TempDir(), "client.wasm")
	buildV5Wasm(parseT, parseRoot, parseWasm, "./examples/server/server-interactive-proof-of-concept")
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatal(parseErr)
	}
	parseAddress := parseListener.Addr().String()
	parseListener.Close()
	parseStop := startExamplesCommandWithEnv(parseT, parseRoot, []string{"GWC_INTERACTIVE_ADDRESS=" + parseAddress, "GWC_INTERACTIVE_WASM=" + parseWasm}, "go", "run", "./examples/server/server-interactive-proof-of-concept")
	defer parseStop()
	parseURL := "http://" + parseAddress
	waitForHealthyExamplesURL(parseT, parseURL, 30*time.Second)
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parsePage.OnPageError(func(parseErr error) { parseT.Errorf("page error: %v", parseErr) })
		parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
			if parseMessage.Type() == "error" {
				parseT.Logf("console error: %s", parseMessage.Text())
			}
		})
		if _, parseErr := parsePage.Goto(parseURL); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelector('#status')?.textContent.includes('Snapshot v1')`, nil); parseErr != nil {
			parseDebug, _ := parsePage.Evaluate(`() => ({statuses: [...document.querySelectorAll('#status')].map(node => node.textContent), body: document.body.innerText, resources: performance.getEntriesByType('resource').map(entry => entry.name)})`)
			parseT.Fatalf("first SSE snapshot: %v; debug=%#v", parseErr, parseDebug)
		}
		if parseCount, parseErr := parsePage.Locator("#status").Count(); parseErr != nil || parseCount != 1 {
			parseT.Fatalf("expected hydration to preserve one status node, count=%d err=%v", parseCount, parseErr)
		}
		for _, parseAction := range []struct{ ID, Metric, Expected string }{
			{"add-user", "Active Users", "20"}, {"resolve-approval", "Pending Approvals", "6"},
			{"add-incident", "Incidents Today", "3"}, {"clear-incidents", "Incidents Today", "0"},
		} {
			if parseErr := parsePage.Locator(`button[data-action="` + parseAction.ID + `"]`).Click(); parseErr != nil {
				parseT.Fatal(parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`([label, expected]) => [...document.querySelectorAll('.metric')].some(tile => tile.querySelector('.metric-label').textContent === label && tile.querySelector('.metric-value').textContent === expected)`, []string{parseAction.Metric, parseAction.Expected}); parseErr != nil {
				parseT.Fatal(parseErr)
			}
		}
		if _, parseErr := parsePage.Reload(); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelector('#status')?.textContent.includes('Snapshot v5') && document.querySelectorAll('button[data-action]').length === 4`, nil); parseErr != nil {
			parseT.Fatal(parseErr)
		}
	})
}
