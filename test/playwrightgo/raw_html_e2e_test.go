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

// TestRawHTMLE2E proves the G3 markup nodes in a real browser: RawHTML builds
// real DOM elements (not escaped text), the trusted path renders inline SVG in
// the SVG namespace, and the stripped <script> never executes.
func TestRawHTMLE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/v4/examples/public/raw-html")
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
		parseW.Write([]byte(rawHTMLIndexHTML))
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
	if _, parseErr := parsePage.WaitForSelector("#bold", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for #bold: %v", parseErr)
	}

	// RawHTML produced a real <b> element (not escaped text).
	if parseTag := evalString(parseT, parsePage, `document.getElementById('bold').tagName`); parseTag != "B" {
		parseT.Fatalf("expected real <b> element, got tagName %q", parseTag)
	}
	if parseText := evalString(parseT, parsePage, `document.getElementById('bold').textContent`); parseText != "world" {
		parseT.Fatalf("expected <b>world</b>, got %q", parseText)
	}

	// The trusted SVG circle is a real SVG element in the SVG namespace.
	if parseNS := evalString(parseT, parsePage, `document.getElementById('dot').namespaceURI`); parseNS != "http://www.w3.org/2000/svg" {
		parseT.Fatalf("svg circle not in SVG namespace, got %q", parseNS)
	}

	// The stripped <script> must not have executed.
	parseWindowPwned, parseErr := parsePage.Evaluate(`() => window.__pwned === undefined`)
	if parseErr != nil {
		parseT.Fatalf("eval pwned check: %v", parseErr)
	}
	if parseWindowPwned != true {
		parseT.Fatal("sanitized <script> executed — XSS not prevented")
	}

	// And no literal "<script" leaked into the rendered rich-text node.
	if parseRichHTML := evalString(parseT, parsePage, `document.getElementById('rich').innerHTML`); strings.Contains(parseRichHTML, "<script") {
		parseT.Fatalf("script markup leaked into the DOM: %s", parseRichHTML)
	}
}

const rawHTMLIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>raw-html-e2e</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
