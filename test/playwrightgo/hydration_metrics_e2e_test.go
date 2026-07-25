//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"fmt"
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/monstercameron/GoWebComponents/v5/examples/shared/hydrationprobe"
	"github.com/monstercameron/GoWebComponents/v5/ui"
	playwright "github.com/playwright-community/playwright-go"
)

// TestHydrationAdoptsServerDOME2E closes the long-standing gap (#37): before this,
// no test proved in a REAL browser that ui.Hydrate resumes server-rendered DOM
// instead of silently client-rendering it. It server-renders the shared probe,
// serves that markup, hydrates it in Chromium, and asserts the framework-reported
// hydration metrics show ZERO fallbacks and ZERO discarded nodes — i.e. the server
// DOM was adopted, not thrown away.
func TestHydrationAdoptsServerDOME2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	// The exact server markup the client will hydrate — rendered from the SAME
	// component the wasm fixture hydrates, so a correct hydration adopts it verbatim.
	parseServerMarkup, parseRenderErr := ui.RenderToString(ui.CreateElement(hydrationprobe.HydrationProbe))
	if parseRenderErr != nil {
		parseT.Fatalf("server render probe: %v", parseRenderErr)
	}

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/v5/examples/public/hydration-metrics")
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

	parseIndexHTML := fmt.Sprintf(hydrationMetricsIndexTemplate, parseServerMarkup)

	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseReq *http.Request) {
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		parseW.Write([]byte(parseIndexHTML))
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

	// Hydration finished (the fixture surfaces this after ui.Hydrate).
	if _, parseErr := parsePage.WaitForFunction(`() => window.__hydrationDone === true`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(30000)}); parseErr != nil {
		parseT.Fatalf("hydration did not complete: %v", parseErr)
	}

	// A hydration error is a hard fail with its message.
	if parseErrVal, parseEvalErr := parsePage.Evaluate(`() => window.__hydrationError || ""`); parseEvalErr == nil {
		if parseMsg, parseOk := parseErrVal.(string); parseOk && strings.TrimSpace(parseMsg) != "" {
			parseT.Fatalf("ui.Hydrate reported an error: %s", parseMsg)
		}
	}

	parseInt := func(parseName string) int {
		parseVal, parseEvalErr := parsePage.Evaluate(fmt.Sprintf(`() => window.%s`, parseName))
		if parseEvalErr != nil {
			parseT.Fatalf("read %s: %v", parseName, parseEvalErr)
		}
		switch parseTyped := parseVal.(type) {
		case int:
			return parseTyped
		case float64:
			return int(parseTyped)
		default:
			parseT.Fatalf("%s was not numeric: %T (%v)", parseName, parseVal, parseVal)
			return -1
		}
	}

	// The core assertion: hydration ADOPTED the server DOM.
	if parseFallback := parseInt("__hydrationFallbackCount"); parseFallback != 0 {
		parseT.Fatalf("hydration fell back to client render %d time(s) — server DOM was not adopted", parseFallback)
	}
	if parseDiscarded := parseInt("__hydrationDiscarded"); parseDiscarded != 0 {
		parseT.Fatalf("hydration discarded %d server node(s) — server DOM was not fully adopted", parseDiscarded)
	}
	// The server DOM was actually present to adopt (guards against a false pass on an
	// empty root, which would trivially report zero fallbacks).
	if parseExisting := parseInt("__hydrationExisting"); parseExisting == 0 {
		parseT.Fatal("hydration saw zero existing server DOM nodes — the fixture did not exercise adoption")
	}

	// The resumed tree is the server's node, not a remount: the server-rendered
	// count text is still present in the live DOM.
	if _, parseErr := parsePage.WaitForFunction(`() => {
		const el = document.getElementById("probe-count");
		return el && el.textContent === "count: 7";
	}`, nil, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)}); parseErr != nil {
		parseT.Fatalf("hydrated tree lost the server-rendered content: %v", parseErr)
	}
}

const hydrationMetricsIndexTemplate = `<!doctype html>
<html><head><meta charset="utf-8"><title>hydration-metrics-e2e</title></head>
<body><div id="app">%s</div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
