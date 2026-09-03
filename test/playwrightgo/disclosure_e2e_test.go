//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

// TestDisclosureE2E proves the WAI-ARIA disclosure pattern (the `gwc add disclosure` /
// shorthand interaction) in a real browser: the trigger starts collapsed (aria-expanded
// false, region absent), a click expands it (aria-expanded true, region revealed), and a
// second click collapses it again.
func TestDisclosureE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/v5/examples/public/disclosure-demo")
	parseBuild.Dir = parseRoot
	parseBuild.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOut, parseErr := parseBuild.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build wasm fixture: %v\n%s", parseErr, parseOut)
	}

	parseGoRoot := strings.TrimSpace(runGo(parseT, "env", "GOROOT"))
	parseShim, parseErr := os.ReadFile(filepath.Join(parseGoRoot, "lib", "wasm", "wasm_exec.js"))
	if parseErr != nil {
		parseT.Fatalf("read wasm_exec.js: %v", parseErr)
	}

	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseReq *http.Request) {
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		parseW.Write([]byte(disclosureIndexHTML))
	})
	parseMux.HandleFunc("/wasm_exec.js", func(parseW http.ResponseWriter, parseReq *http.Request) {
		parseW.Header().Set("Content-Type", "application/javascript")
		parseW.Write(parseShim)
	})
	parseMux.HandleFunc("/app.wasm", func(parseW http.ResponseWriter, parseReq *http.Request) {
		parseW.Header().Set("Content-Type", "application/wasm")
		http.ServeFile(parseW, parseReq, parseWasmPath)
	})
	parseServer := httptest.NewServer(parseMux)
	defer parseServer.Close()

	parseRunOptions := &playwright.RunOptions{Browsers: []string{"chromium"}}
	if parseErr := playwright.Install(parseRunOptions); parseErr != nil {
		parseT.Fatalf("install playwright: %v", parseErr)
	}
	parsePw, parseErr := playwright.Run(parseRunOptions)
	if parseErr != nil {
		parseT.Fatalf("run playwright: %v", parseErr)
	}
	defer parsePw.Stop()

	parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer parseBrowser.Close()

	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		parseT.Fatalf("new page: %v", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseServer.URL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("goto: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#disclosure-trigger", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for trigger: %v", parseErr)
	}

	// Collapsed initially: aria-expanded false, region absent.
	parseExpanded := func() string {
		return evalString(parseT, parsePage,
			`(document.getElementById('disclosure-trigger') && document.getElementById('disclosure-trigger').getAttribute('aria-expanded')) || ''`)
	}
	parseRegionText := func() string {
		return evalString(parseT, parsePage,
			`(document.getElementById('disclosure-region') && document.getElementById('disclosure-region').textContent) || ''`)
	}
	if parseExpanded() != "false" {
		parseT.Fatalf("disclosure should start collapsed, aria-expanded=%q", parseExpanded())
	}
	if parseRegionText() != "" {
		parseT.Fatalf("region should be absent while collapsed, got %q", parseRegionText())
	}

	// Click → expanded: aria-expanded true, region revealed.
	if parseErr := parsePage.Click("#disclosure-trigger"); parseErr != nil {
		parseT.Fatalf("click to expand: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.getElementById('disclosure-trigger').getAttribute('aria-expanded') === 'true'
		   && document.getElementById('disclosure-region')
		   && document.getElementById('disclosure-region').textContent.indexOf('Hidden content revealed') !== -1`,
		nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("click did not expand the disclosure: %v", parseErr)
	}

	// Click again → collapsed.
	if parseErr := parsePage.Click("#disclosure-trigger"); parseErr != nil {
		parseT.Fatalf("click to collapse: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.getElementById('disclosure-trigger').getAttribute('aria-expanded') === 'false'
		   && !document.getElementById('disclosure-region')`,
		nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("second click did not collapse the disclosure: %v", parseErr)
	}
}

const disclosureIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>disclosure-e2e</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
