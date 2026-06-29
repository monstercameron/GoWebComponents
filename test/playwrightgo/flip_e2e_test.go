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

// TestKeyedListFLIPE2E proves keyed-list FLIP in a real browser (audit FA5): clicking
// "shuffle" reorders the keyed list (deterministic) and each moved item records a NON-ZERO
// FLIP invert translateY in data-flip-dy — computed by anim.ComputeFLIP from the item's
// measured before/after rects. The attribute is persisted (not animated away), so the
// assertion is not timing-sensitive.
func TestKeyedListFLIPE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/v4/examples/public/flip-demo")
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
		parseW.Write([]byte(flipIndexHTML))
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
	if _, parseErr := parsePage.WaitForSelector("#shuffle", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for shuffle: %v", parseErr)
	}

	order := func() string {
		return evalString(parseT, parsePage,
			`Array.from(document.querySelectorAll('.flip-item')).map(e => e.id).join(',')`)
	}
	// Initial keyed order.
	if got := order(); got != "item-a,item-b,item-c" {
		parseT.Fatalf("initial order = %q, want item-a,item-b,item-c", got)
	}

	if parseErr := parsePage.Click("#shuffle"); parseErr != nil {
		parseT.Fatalf("click shuffle: %v", parseErr)
	}

	// Reorder is deterministic (rotate), and at least one moved item must have a non-zero
	// FLIP invert translateY — proving anim.ComputeFLIP measured the move in a real browser.
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			const ids = Array.from(document.querySelectorAll('.flip-item')).map(e => e.id).join(',');
			if (ids !== 'item-c,item-a,item-b') return false;
			return Array.from(document.querySelectorAll('.flip-item'))
				.some(e => Math.abs(parseFloat(e.getAttribute('data-flip-dy') || '0')) > 0.5);
		}`,
		nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("keyed reorder + non-zero FLIP invert not observed: %v (order=%q)", parseErr, order())
	}
}

const flipIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>flip-e2e</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
