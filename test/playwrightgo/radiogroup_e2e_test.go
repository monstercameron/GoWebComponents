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

// TestRadioGroupE2E proves U5 in a real browser: an interactive radio group built
// on ui.UseCompositeNavigation has a single roving tab stop, arrow keys move the
// selection (selection-follows-focus), and focus follows the active radio.
func TestRadioGroupE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/v5/examples/public/radiogroup")
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
		parseW.Write([]byte(radioGroupIndexHTML))
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
	if _, parseErr := parsePage.WaitForSelector("#opt-0", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for radios: %v", parseErr)
	}

	parseAttr := func(parseID, parseAttr string) string {
		return evalString(parseT, parsePage,
			`(document.getElementById('`+parseID+`') && document.getElementById('`+parseID+`').getAttribute('`+parseAttr+`')) || ''`)
	}

	// Initial: opt-0 is the single tab stop, checked, and focused.
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.activeElement && document.activeElement.id === 'opt-0'`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("opt-0 not focused on mount: %v", parseErr)
	}
	if parseAttr("opt-0", "tabindex") != "0" || parseAttr("opt-1", "tabindex") != "-1" {
		parseT.Fatalf("initial roving tab stop wrong: opt-0=%q opt-1=%q", parseAttr("opt-0", "tabindex"), parseAttr("opt-1", "tabindex"))
	}
	if parseAttr("opt-0", "aria-checked") != "true" {
		parseT.Fatalf("opt-0 should be checked initially, got %q", parseAttr("opt-0", "aria-checked"))
	}

	// ArrowRight: selection + tab stop + focus move to opt-1.
	if parseErr := parsePage.Keyboard().Press("ArrowRight"); parseErr != nil {
		parseT.Fatalf("ArrowRight: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.activeElement && document.activeElement.id === 'opt-1'
		   && document.getElementById('opt-1').getAttribute('tabindex') === '0'
		   && document.getElementById('opt-1').getAttribute('aria-checked') === 'true'
		   && document.getElementById('opt-0').getAttribute('tabindex') === '-1'
		   && document.getElementById('opt-0').getAttribute('aria-checked') === 'false'`,
		nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("ArrowRight did not move selection/focus to opt-1: %v", parseErr)
	}

	// Exactly one tab stop at any time.
	parseZeros := evalString(parseT, parsePage,
		`String(Array.from(document.querySelectorAll('[role=radio]')).filter(e => e.getAttribute('tabindex') === '0').length)`)
	if parseZeros != "1" {
		parseT.Fatalf("expected exactly one roving tab stop, got %s", parseZeros)
	}

	// Loop wrap: opt-1 -> opt-2 -> opt-0.
	if parseErr := parsePage.Keyboard().Press("ArrowRight"); parseErr != nil {
		parseT.Fatalf("ArrowRight 2: %v", parseErr)
	}
	if parseErr := parsePage.Keyboard().Press("ArrowRight"); parseErr != nil {
		parseT.Fatalf("ArrowRight 3: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.activeElement && document.activeElement.id === 'opt-0'`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("arrow navigation did not wrap back to opt-0: %v", parseErr)
	}
}

const radioGroupIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>radiogroup-e2e</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
