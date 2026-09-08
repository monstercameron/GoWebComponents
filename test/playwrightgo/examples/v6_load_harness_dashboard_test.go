//go:build playwrightgo

package playwrightgoexamples_test

import (
	"path/filepath"
	"runtime"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

// TestV6LoadHarnessDashboard verifies real GWC controls against a controlled measurement boundary.
func TestV6LoadHarnessDashboard(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRoot := examplesRepoRootFromFile(parseFile)
	parseArtifacts := parseT.TempDir()
	buildV5HarnessWasm(parseT, parseRoot, filepath.Join(parseArtifacts, "v5harness.wasm"))
	parseServer := serveV5Harness(parseT, parseRoot, parseArtifacts)
	defer parseServer.Close()
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
			if parseMessage.Type() == "error" {
				parseT.Logf("dashboard console error: %s", parseMessage.Text())
			}
		})
		parsePage.OnPageError(func(parseErr error) { parseT.Logf("dashboard page error: %v", parseErr) })
		if parseErr := parsePage.AddInitScript(playwright.Script{Content: playwright.String(`window.harnessInputTrace = []; for (const kind of ['beforeinput', 'input', 'change', 'pointerdown', 'pointerup', 'click']) document.addEventListener(kind, event => { if (event.target.id === 'filter' || event.target.id === 'scroll') window.harnessInputTrace.push({kind, target:event.target.id, value:event.target.value, inputType:event.inputType}); }, true);`)}); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.Goto(parseServer.URL); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#harness-dashboard #run"); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseErr := parsePage.Locator("#filter").Fill("row-004"); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelectorAll('#rows li').length === 10 && document.querySelector('#rows li').textContent.startsWith('row-0040')`, nil); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseErr := parsePage.Locator("#filter").Fill(""); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseErr := parsePage.GetByRole("button", playwright.PageGetByRoleOptions{Name: "advance window"}).Click(); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.getElementById('filter').value === '' && document.querySelectorAll('#rows li').length === 40 && document.querySelector('#rows li').textContent.startsWith('row-0040')`, nil); parseErr != nil {
			parseDebug, _ := parsePage.Evaluate(`() => ({value:document.getElementById('filter').value, rows:document.querySelectorAll('#rows li').length, first:document.querySelector('#rows li')?.textContent, state:document.querySelector('#app main').dataset, trace:window.harnessInputTrace})`)
			parseT.Fatalf("clear and advance lost controlled state: %v debug=%#v", parseErr, parseDebug)
		}
		// The measurement algorithm has independent seam tests. Here a pending
		// promise lets the real Go/Wasm UI prove busy, failure and retry behavior.
		if _, parseErr := parsePage.Evaluate(`() => { window.__gwcRunDashboardComparison = () => new Promise((resolve, reject) => { window.resolveDashboard = resolve; window.rejectDashboard = reject; }); }`); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseErr := parsePage.Locator("#run").Click(); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.getElementById('run').disabled && document.getElementById('status').textContent.startsWith('Running') && !!window.rejectDashboard`, nil); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.Evaluate(`() => window.rejectDashboard(new Error('measurement unavailable'))`); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById('run').disabled && document.getElementById('status').textContent === 'ERROR' && document.getElementById('out').textContent.includes('measurement unavailable')`, nil); parseErr != nil {
			parseDebug, _ := parsePage.Evaluate(`() => ({disabled: document.getElementById('run').disabled, status: document.getElementById('status').textContent, output: document.getElementById('out').textContent})`)
			parseT.Fatalf("error state: %v debug=%#v", parseErr, parseDebug)
		}
		if _, parseErr := parsePage.Evaluate(`() => { window.resolveDashboard = null; }`); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if parseErr := parsePage.Locator("#run").Click(); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => !!window.resolveDashboard`, nil); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.Evaluate(`() => window.resolveDashboard(JSON.stringify({status:'PASS', output:'<img src=x onerror=alert(1)>'}))`); parseErr != nil {
			parseT.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.getElementById('status').textContent === 'PASS' && !document.getElementById('run').disabled && document.getElementById('out').textContent === '<img src=x onerror=alert(1)>' && !document.querySelector('#out img') && !!document.getElementById('filter')`, nil); parseErr != nil {
			parseT.Fatal(parseErr)
		}
	})
}
