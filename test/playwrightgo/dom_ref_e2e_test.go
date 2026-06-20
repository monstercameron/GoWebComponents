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

	playwright "github.com/playwright-community/playwright-go"
)

// TestDOMRefE2E is the end-to-end proof of the G2 DOM-ref primitive: it builds
// the dom-ref fixture to wasm, runs it in a real Chromium, and asserts that
// ui.UseDOMRef resolved to the live <input> (the on-mount Focus() landed focus on
// it) and that unmounting the input detaches the ref without error.
func TestDOMRefE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/examples/public/dom-ref")
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
		parseW.Write([]byte(domRefIndexHTML))
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
	if _, parseErr := parsePage.WaitForSelector("#target", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for #target: %v", parseErr)
	}

	// The ref resolved to the live input AND Focus() worked → activeElement is it.
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.activeElement && document.activeElement.id === 'target'`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("ref did not resolve / focus the input: %v", parseErr)
	}
	if parseStatus := evalString(parseT, parsePage, `document.getElementById('status').textContent`); parseStatus != "focused" {
		parseT.Fatalf("expected status 'focused' (ref.Mounted() true on mount), got %q", parseStatus)
	}

	// Unmount the input — the ref detaches and nothing throws; the field is gone.
	if parseErr := parsePage.Click("#toggle"); parseErr != nil {
		parseT.Fatalf("click toggle: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.getElementById('target') === null`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("input not unmounted after toggle: %v", parseErr)
	}
}

const domRefIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>dom-ref-e2e</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
