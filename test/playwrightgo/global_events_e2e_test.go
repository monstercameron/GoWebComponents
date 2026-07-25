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

// TestGlobalEventsE2E proves G9 (ui.UseGlobalKey) in a real browser: the
// document keydown listener fires while mounted, and — critically — is cleaned
// up on unmount. A leaked js.Func would survive remount and double-count a single
// keypress; the test asserts exactly-once counting after an unmount/remount.
func TestGlobalEventsE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/v5/examples/public/global-events")
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
		parseW.Write([]byte(globalEventsIndexHTML))
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
	if _, parseErr := parsePage.WaitForSelector("#listener-marker", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for listener: %v", parseErr)
	}

	parseExpectCount := func(parseWant string) {
		parseT.Helper()
		if _, parseErr := parsePage.WaitForFunction(
			`() => document.getElementById('count') && document.getElementById('count').textContent === '`+parseWant+`'`,
			nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
		); parseErr != nil {
			parseT.Fatalf("expected count %s: %v (got %q)", parseWant, parseErr,
				evalString(parseT, parsePage, `document.getElementById('count') && document.getElementById('count').textContent`))
		}
	}

	// Mounted: a global keydown increments the count.
	if parseErr := parsePage.Keyboard().Press("a"); parseErr != nil {
		parseT.Fatalf("press: %v", parseErr)
	}
	parseExpectCount("1")

	// Unmount the listener, then remount it.
	if parseErr := parsePage.Click("#toggle"); parseErr != nil {
		parseT.Fatalf("toggle off: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => document.getElementById('listener-marker') === null`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)}); parseErr != nil {
		parseT.Fatalf("listener not unmounted: %v", parseErr)
	}
	if parseErr := parsePage.Click("#toggle"); parseErr != nil {
		parseT.Fatalf("toggle on: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#listener-marker", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(10000),
	}); parseErr != nil {
		parseT.Fatalf("listener did not remount: %v", parseErr)
	}

	// One keypress after remount must increment by exactly 1 (=> 2). If the old
	// listener leaked on unmount, two listeners would fire and the count would be 3.
	if parseErr := parsePage.Keyboard().Press("b"); parseErr != nil {
		parseT.Fatalf("press after remount: %v", parseErr)
	}
	parseExpectCount("2")
}

const globalEventsIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>global-events-e2e</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
