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

// TestUseHeadE2E is the end-to-end proof of head.UseHead: it builds the use-head fixture to wasm,
// runs it in a real Chromium, and asserts that UseHead applied the Document's title, description,
// canonical, and OpenGraph tags to the LIVE document head (overriding the placeholder title), and
// that switching the logical page re-applies the new title + description without a reload.
func TestUseHeadE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/examples/public/use-head")
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
		parseW.Write([]byte(useHeadIndexHTML))
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
	if _, parseErr := parsePage.WaitForSelector("#app-root", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for #app-root: %v", parseErr)
	}

	// UseHead overrode the placeholder title and applied the head tags.
	if _, parseErr := parsePage.WaitForFunction(`() => document.title === 'Home Title'`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)}); parseErr != nil {
		parseT.Fatalf("UseHead did not set the document title: %v", parseErr)
	}
	parseChecks := map[string]string{
		`document.querySelector('meta[name="description"]')?.content`:           "home description",
		`document.querySelector('meta[property="og:title"]')?.content`:          "Home OG Title",
		`document.querySelector('link[rel="canonical"]')?.getAttribute('href')`: "https://example.com/home",
	}
	for parseExpr, parseWant := range parseChecks {
		parseVal, parseErr := parsePage.Evaluate(`() => ` + parseExpr)
		if parseErr != nil {
			parseT.Fatalf("eval %s: %v", parseExpr, parseErr)
		}
		if parseGot, _ := parseVal.(string); parseGot != parseWant {
			parseT.Fatalf("head tag %s = %q, want %q", parseExpr, parseGot, parseWant)
		}
	}

	// Switching the page re-applies the new title + description on the live document (no reload).
	if parseErr := parsePage.Click("#to-about"); parseErr != nil {
		parseT.Fatalf("click #to-about: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => document.title === 'About Title'`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)}); parseErr != nil {
		parseT.Fatalf("UseHead did not update the title on page change: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.querySelector('meta[name="description"]')?.content === 'about description'`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)}); parseErr != nil {
		parseT.Fatalf("UseHead did not update the description on page change: %v", parseErr)
	}
}

const useHeadIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>initial-placeholder</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`
