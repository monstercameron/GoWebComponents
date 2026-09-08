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

// TestTypedCSSE2E is the end-to-end check for the F3 typed-CSS package: it builds
// the css demo to wasm, serves it, loads it in a real Chromium, and asserts that
// the runtime <style> injection sink produced rules and that the browser actually
// computed the typed styles (display:flex, the Sky-500 background, the :hover
// rule). This proves the whole Layer-1 path end-to-end in a browser.
func TestTypedCSSE2E(parseT *testing.T) {
	parseRoot := repoRootFromTest(parseT)
	parseDir := parseT.TempDir()

	// 1. Build the demo to wasm.
	parseWasmPath := filepath.Join(parseDir, "app.wasm")
	parseBuild := exec.Command("go", "build", "-o", parseWasmPath, "github.com/monstercameron/GoWebComponents/v6/examples/public/typed-css")
	parseBuild.Dir = parseRoot
	parseBuild.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if parseOut, parseErr := parseBuild.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("build wasm demo: %v\n%s", parseErr, parseOut)
	}

	// 2. Copy the matching wasm_exec.js shim from GOROOT.
	parseGoRoot := strings.TrimSpace(runGo(parseT, "env", "GOROOT"))
	parseShimSrc := filepath.Join(parseGoRoot, "lib", "wasm", "wasm_exec.js")
	parseShim, parseErr := os.ReadFile(parseShimSrc)
	if parseErr != nil {
		parseT.Fatalf("read wasm_exec.js: %v", parseErr)
	}

	// 3. Serve index.html + wasm_exec.js + app.wasm.
	parseMux := http.NewServeMux()
	parseMux.HandleFunc("/", func(parseW http.ResponseWriter, parseReq *http.Request) {
		parseW.Header().Set("Content-Type", "text/html; charset=utf-8")
		parseW.Write([]byte(typedCSSIndexHTML))
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

	// 4. Drive Chromium.
	parseRunOptions := &playwright.RunOptions{Browsers: []string{"chromium"}}
	if parseErr := playwright.Install(parseRunOptions); parseErr != nil {
		parseT.Fatalf("install playwright: %v", parseErr)
	}
	parsePw, parseErr := playwright.Run(parseRunOptions)
	if parseErr != nil {
		parseT.Fatalf("run playwright: %v", parseErr)
	}
	defer parsePw.Stop()

	parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
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

	// Wait for the wasm app to mount the styled card.
	if _, parseErr := parsePage.WaitForSelector("#card", playwright.PageWaitForSelectorOptions{
		Timeout: playwright.Float(120000),
	}); parseErr != nil {
		parseT.Fatalf("wait for #card: %v", parseErr)
	}

	// Assert the injection sink produced a managed <style> with rules.
	parseStyleText, parseErr := parsePage.Evaluate(`() => {
		const el = document.getElementById('gwc-css');
		return el ? el.textContent : '';
	}`)
	if parseErr != nil {
		parseT.Fatalf("read style element: %v", parseErr)
	}
	parseCSS, _ := parseStyleText.(string)
	for _, parseNeedle := range []string{"display:flex", ":hover", "@media", "> span"} {
		if !strings.Contains(parseCSS, parseNeedle) {
			parseT.Fatalf("injected stylesheet missing %q:\n%s", parseNeedle, parseCSS)
		}
	}

	// Card: typed display + Slate-900 background computed by the browser.
	if parseDisplay := evalString(parseT, parsePage, `getComputedStyle(document.getElementById('card')).display`); parseDisplay != "flex" {
		parseT.Fatalf("expected computed display flex, got %q", parseDisplay)
	}
	if parseBg := evalString(parseT, parsePage, `getComputedStyle(document.getElementById('card')).backgroundColor`); parseBg != "rgb(15, 23, 42)" {
		parseT.Fatalf("expected Slate-900 card background, got %q", parseBg)
	}

	// Selector composition: the label <span> is styled ONLY via the parent card's
	// css.Child(css.El("span"), …) rule — so a Slate-400 uppercase label proves the
	// descendant combinator composed and applied in a real browser.
	if parseColor := evalString(parseT, parsePage, `getComputedStyle(document.getElementById('label')).color`); parseColor != "rgb(148, 163, 184)" {
		parseT.Fatalf("child-combinator color not applied to label, got %q", parseColor)
	}
	if parseTransform := evalString(parseT, parsePage, `getComputedStyle(document.getElementById('label')).textTransform`); parseTransform != "uppercase" {
		parseT.Fatalf("child-combinator text-transform not applied, got %q", parseTransform)
	}

	// Interaction: clicking +1 updates the rendered count.
	if parseErr := parsePage.Click("#inc"); parseErr != nil {
		parseT.Fatalf("click +1: %v", parseErr)
	}
	if parseErr := parsePage.Click("#inc"); parseErr != nil {
		parseT.Fatalf("click +1: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.body.innerText.includes('2')`, nil,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(10000)},
	); parseErr != nil {
		parseT.Fatalf("count did not reach 2 after two clicks: %v", parseErr)
	}

	if len(distinctClasses(parseCSS)) < 2 {
		parseT.Fatalf("expected at least two generated classes in the sheet:\n%s", parseCSS)
	}
}

func evalString(parseT *testing.T, parsePage playwright.Page, parseExpr string) string {
	parseT.Helper()
	parseVal, parseErr := parsePage.Evaluate(`() => ` + parseExpr)
	if parseErr != nil {
		parseT.Fatalf("eval %q: %v", parseExpr, parseErr)
	}
	parseStr, _ := parseVal.(string)
	return parseStr
}

const typedCSSIndexHTML = `<!doctype html>
<html><head><meta charset="utf-8"><title>typed-css-e2e</title></head>
<body><div id="app"></div>
<script src="wasm_exec.js"></script>
<script>
const go = new Go();
WebAssembly.instantiateStreaming(fetch("app.wasm"), go.importObject).then(r => { go.run(r.instance); });
</script></body></html>`

func runGo(parseT *testing.T, parseArgs ...string) string {
	parseT.Helper()
	parseOut, parseErr := exec.Command("go", parseArgs...).Output()
	if parseErr != nil {
		parseT.Fatalf("go %s: %v", strings.Join(parseArgs, " "), parseErr)
	}
	return string(parseOut)
}

// repoRootFromTest walks up from the working directory to the module root (the
// directory containing go.mod).
func repoRootFromTest(parseT *testing.T) string {
	parseT.Helper()
	parseDir, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("getwd: %v", parseErr)
	}
	for {
		if _, parseErr := os.Stat(filepath.Join(parseDir, "go.mod")); parseErr == nil {
			return parseDir
		}
		parseParent := filepath.Dir(parseDir)
		if parseParent == parseDir {
			parseT.Fatal("could not find module root (go.mod)")
		}
		parseDir = parseParent
	}
}

// distinctClasses extracts the generated "c-…" class names from a stylesheet.
func distinctClasses(parseCSS string) []string {
	parseSeen := map[string]bool{}
	var parseOut []string
	for parseIdx := strings.Index(parseCSS, ".c-"); parseIdx >= 0; {
		parseRest := parseCSS[parseIdx+1:]
		parseEnd := strings.IndexAny(parseRest, "{:: ,>")
		if parseEnd < 0 {
			break
		}
		parseName := parseRest[:parseEnd]
		if parseName != "" && !parseSeen[parseName] {
			parseSeen[parseName] = true
			parseOut = append(parseOut, parseName)
		}
		parseNext := strings.Index(parseCSS[parseIdx+1:], ".c-")
		if parseNext < 0 {
			break
		}
		parseIdx = parseIdx + 1 + parseNext
	}
	return parseOut
}
