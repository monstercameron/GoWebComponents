//go:build playwrightgo
// +build playwrightgo

package main

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io/fs"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

func TestDevLoopBrowserPreservesStateAcrossSeveralHotReloads(parseT *testing.T) {
	if testing.Short() {
		parseT.Skip("skipping browser hot-reload e2e in short mode")
	}

	parseRepoRoot, parseErr := resolveRepoRoot()
	if parseErr != nil {
		parseT.Fatalf("resolve repo root: %v", parseErr)
	}

	parseAppRoot := filepath.Join(parseT.TempDir(), "hot-reload-browser-e2e")
	parseChangedPanelPath := filepath.Join(parseAppRoot, "widgets", "changed_panel.go")
	writeHotReloadBrowserFixture(parseT, parseRepoRoot, parseAppRoot, "v1")
	prepareHotReloadBrowserFixtureModule(parseT, parseAppRoot)

	parseRootURL, parseOutput, parseProcessExited, parseProcessErr := startHotReloadBrowserDevServer(parseT, parseRepoRoot, parseAppRoot)

	withLiveReloadClientPage(parseT, func(parsePage playwright.Page) {
		parsePage.OnConsole(func(parseMsg playwright.ConsoleMessage) {
			if parseMsg.Type() == "error" || strings.Contains(parseMsg.Text(), "[gwc]") {
				parseT.Logf("browser console [%s]: %s", parseMsg.Type(), parseMsg.Text())
			}
		})

		if _, parseErr2 := parsePage.Goto(parseRootURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr2 != nil {
			parseT.Fatalf("goto hot reload fixture: %v\n%s", parseErr2, parseOutput.String())
		}
		if _, parseErr2 := parsePage.WaitForSelector("#stable-increment", playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(120000)}); parseErr2 != nil {
			parseT.Fatalf("wait for initial wasm render: %v\n%s", parseErr2, parseOutput.String())
		}
		waitForHotReloadBrowserClient(parseT, parseRootURL, parseProcessExited, parseProcessErr, parseOutput)

		parseStableCount := 0
		for parseCycle := 1; parseCycle <= 3; parseCycle++ {
			parseStableCount++
			clickHotReloadBrowserButton(parseT, parsePage, "#stable-increment", 1)
			clickHotReloadBrowserButton(parseT, parsePage, "#changed-increment", 2)
			waitHotReloadBrowserDOM(parseT, parsePage, parseRootURL, parseOutput, parseStableCount, 2, fmt.Sprintf("v%d", parseCycle), false)

			parseNextVersion := fmt.Sprintf("v%d", parseCycle+1)
			writeHotReloadBrowserChangedPanel(parseT, parseChangedPanelPath, parseNextVersion)
			waitHotReloadBrowserDOM(parseT, parsePage, parseRootURL, parseOutput, parseStableCount, 0, parseNextVersion, true)
		}
	})
}

func startHotReloadBrowserDevServer(parseT *testing.T, parseRepoRoot string, parseAppRoot string) (string, *bytes.Buffer, <-chan struct{}, *error) {
	parseT.Helper()

	parsePort, parseErr := reserveTCPPort()
	if parseErr != nil {
		parseT.Fatalf("reserve tcp port: %v", parseErr)
	}

	parseLivereloadBinary := prepareHotReloadBrowserLivereloadModule(parseT, parseRepoRoot)
	parseCtx, parseCancel := context.WithCancel(context.Background())
	parseCmd := exec.CommandContext(parseCtx, parseLivereloadBinary,
		"-app", filepath.Join(parseAppRoot, "main.go"),
		"-root", parseAppRoot,
		"-html", filepath.Join(parseAppRoot, "index.html"),
		"-wasm", scaffoldWASMOutputPath(),
		"-host", "127.0.0.1",
		"-port", parsePort,
	)
	parseCmd.Dir = parseAppRoot
	var parseOutput bytes.Buffer
	parseCmd.Stdout = &parseOutput
	parseCmd.Stderr = &parseOutput
	if parseErr2 := parseCmd.Start(); parseErr2 != nil {
		parseCancel()
		parseT.Fatalf("start dev server: %v", parseErr2)
	}

	parseProcessExited := make(chan struct{})
	var parseProcessErr error
	go func() {
		parseProcessErr = parseCmd.Wait()
		close(parseProcessExited)
	}()
	parseT.Cleanup(func() {
		parseCancel()
		terminateProcessTree(parseCmd)
		select {
		case <-parseProcessExited:
		case <-time.After(5 * time.Second):
			if parseCmd.Process != nil {
				_ = parseCmd.Process.Kill()
			}
			select {
			case <-parseProcessExited:
			case <-time.After(5 * time.Second):
				parseT.Logf("livereload server cleanup timed out\n%s", parseOutput.String())
			}
		}
	})

	parseRootURL := "http://127.0.0.1:" + parsePort + "/"
	waitForHTTPBodyWithProcess(parseT, parseRootURL, 180*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		if !strings.Contains(parseBody, "__GWC_LIVERELOAD_CONFIG") {
			return fmt.Errorf("html shell missing livereload config")
		}
		return nil
	})
	parseWasmURL := parseRootURL + filepath.ToSlash(scaffoldWASMOutputPath())
	waitForHTTPBodyWithProcess(parseT, parseWasmURL, 180*time.Second, parseProcessExited, &parseProcessErr, &parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		if len(parseBody) == 0 {
			return fmt.Errorf("empty wasm response")
		}
		return nil
	})

	return parseRootURL, &parseOutput, parseProcessExited, &parseProcessErr
}

func waitForHotReloadBrowserClient(parseT *testing.T, parseRootURL string, parseProcessExited <-chan struct{}, parseProcessErr *error, parseOutput *bytes.Buffer) {
	parseT.Helper()

	parseStatusURL := strings.TrimRight(parseRootURL, "/") + "/__gwc/status"
	waitForHTTPBodyWithProcess(parseT, parseStatusURL, 30*time.Second, parseProcessExited, parseProcessErr, parseOutput, func(parseResp *http.Response, parseBody string) error {
		if parseResp.StatusCode != http.StatusOK {
			return fmt.Errorf("unexpected status %d", parseResp.StatusCode)
		}
		var parseStatus struct {
			ClientCount int `json:"clientCount"`
		}
		if parseErr := json.Unmarshal([]byte(parseBody), &parseStatus); parseErr != nil {
			return fmt.Errorf("decode dev status: %w", parseErr)
		}
		if parseStatus.ClientCount < 1 {
			return fmt.Errorf("waiting for browser websocket client, got %d", parseStatus.ClientCount)
		}
		return nil
	})
}

func prepareHotReloadBrowserLivereloadModule(parseT *testing.T, parseRepoRoot string) string {
	parseT.Helper()
	parseSourceRoot := filepath.Join(parseRepoRoot, "tools", "livereload")
	parseTargetRoot := filepath.Join(parseT.TempDir(), "livereload")
	if parseErr := filepath.WalkDir(parseSourceRoot, func(parsePath string, parseEntry fs.DirEntry, parseErr error) error {
		if parseErr != nil {
			return parseErr
		}
		parseRel, parseRelErr := filepath.Rel(parseSourceRoot, parsePath)
		if parseRelErr != nil {
			return parseRelErr
		}
		parseTargetPath := filepath.Join(parseTargetRoot, parseRel)
		if parseEntry.IsDir() {
			return os.MkdirAll(parseTargetPath, 0o755)
		}
		parseContent, parseReadErr := os.ReadFile(parsePath)
		if parseReadErr != nil {
			return parseReadErr
		}
		return os.WriteFile(parseTargetPath, parseContent, 0o644)
	}); parseErr != nil {
		parseT.Fatalf("copy livereload module: %v", parseErr)
	}

	parseModulePath, parseErr := (launcher{repoRoot: parseRepoRoot}).readRepoModulePath()
	if parseErr != nil {
		parseT.Fatalf("read repo module path: %v", parseErr)
	}
	parseGoModPath := filepath.Join(parseTargetRoot, "go.mod")
	parseGoModBytes, parseErr := os.ReadFile(parseGoModPath)
	if parseErr != nil {
		parseT.Fatalf("read copied livereload go.mod: %v", parseErr)
	}
	parseGoModLines := strings.Split(string(parseGoModBytes), "\n")
	parseSawReplace := false
	for parseIndex, parseLine := range parseGoModLines {
		if strings.HasPrefix(strings.TrimSpace(parseLine), "replace "+parseModulePath+" =>") {
			parseGoModLines[parseIndex] = "replace " + parseModulePath + " => " + filepath.ToSlash(parseRepoRoot)
			parseSawReplace = true
		}
	}
	if !parseSawReplace {
		parseGoModLines = append(parseGoModLines, "replace "+parseModulePath+" => "+filepath.ToSlash(parseRepoRoot))
	}
	if parseErr2 := os.WriteFile(parseGoModPath, []byte(strings.Join(parseGoModLines, "\n")), 0o644); parseErr2 != nil {
		parseT.Fatalf("write copied livereload go.mod: %v", parseErr2)
	}

	parseCmd := exec.Command("go", "mod", "tidy")
	parseCmd.Dir = parseTargetRoot
	parseOutput, parseErr := parseCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("tidy copied livereload module: %v\n%s", parseErr, string(parseOutput))
	}

	parseBinaryPath := filepath.Join(parseTargetRoot, "livereload-test")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseBuildCmd := exec.Command("go", "build", "-o", parseBinaryPath, ".")
	parseBuildCmd.Dir = parseTargetRoot
	parseBuildOutput, parseBuildErr := parseBuildCmd.CombinedOutput()
	if parseBuildErr != nil {
		parseT.Fatalf("build copied livereload module: %v\n%s", parseBuildErr, string(parseBuildOutput))
	}
	return parseBinaryPath
}

func clickHotReloadBrowserButton(parseT *testing.T, parsePage playwright.Page, parseSelector string, parseTimes int) {
	parseT.Helper()
	for parseIndex := 0; parseIndex < parseTimes; parseIndex++ {
		if parseErr := parsePage.Click(parseSelector); parseErr != nil {
			parseT.Fatalf("click %s: %v", parseSelector, parseErr)
		}
	}
}

func waitHotReloadBrowserDOM(parseT *testing.T, parsePage playwright.Page, parseRootURL string, parseOutput *bytes.Buffer, parseStableCount int, parseChangedCount int, parseVersion string, parseRequireSelectiveOutcome bool) {
	parseT.Helper()

	parseExpression := fmt.Sprintf(`() => {
		const stable = document.querySelector("#stable-count");
		const changed = document.querySelector("#changed-count");
		const version = document.querySelector("#changed-version");
		if (!stable || !changed || !version) return false;
		if (!stable.textContent.includes("Stable count: %d")) return false;
		if (!changed.textContent.includes("Changed count: %d")) return false;
		if (!version.textContent.includes("Changed subtree version: %s")) return false;
		if (!%t) return true;
		const outcome = window.GoLiveReload && window.GoLiveReload.getLastHotReloadOutcome ? window.GoLiveReload.getLastHotReloadOutcome() : null;
		return !!outcome && outcome.outcome === "restored-selective";
	}`, parseStableCount, parseChangedCount, parseVersion, parseRequireSelectiveOutcome)

	if _, parseErr := parsePage.WaitForFunction(parseExpression, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(180000)}); parseErr != nil {
		parseBody, _ := parsePage.TextContent("body")
		parseStatus := readHotReloadBrowserStatus(parseRootURL)
		parseT.Fatalf("wait for hot reload DOM state stable=%d changed=%d version=%s selective=%t: %v\nbody:\n%s\nstatus:\n%s\nserver:\n%s",
			parseStableCount, parseChangedCount, parseVersion, parseRequireSelectiveOutcome, parseErr, parseBody, parseStatus, parseOutput.String())
	}
}

func readHotReloadBrowserStatus(parseRootURL string) string {
	parseResp, parseErr := http.Get(strings.TrimRight(parseRootURL, "/") + "/__gwc/status")
	if parseErr != nil {
		return parseErr.Error()
	}
	defer parseResp.Body.Close()
	parseBuf := new(bytes.Buffer)
	_, _ = parseBuf.ReadFrom(parseResp.Body)
	return parseBuf.String()
}

func writeHotReloadBrowserFixture(parseT *testing.T, parseRepoRoot string, parseAppRoot string, parseVersion string) {
	parseT.Helper()
	if parseErr := os.MkdirAll(filepath.Join(parseAppRoot, "widgets"), 0o755); parseErr != nil {
		parseT.Fatalf("create fixture dirs: %v", parseErr)
	}

	parseModulePath, parseErr := (launcher{repoRoot: parseRepoRoot}).readRepoModulePath()
	if parseErr != nil {
		parseT.Fatalf("read repo module path: %v", parseErr)
	}
	parseGoMod := fmt.Sprintf("module example.com/gwc-hot-reload-browser-e2e\n\ngo 1.25.0\n\nrequire %s v0.0.0\n\nreplace %s => %s\n", parseModulePath, parseModulePath, filepath.ToSlash(parseRepoRoot))
	writeHotReloadBrowserFile(parseT, filepath.Join(parseAppRoot, "go.mod"), parseGoMod)
	writeHotReloadBrowserIndex(parseT, parseAppRoot)
	writeHotReloadBrowserMain(parseT, parseAppRoot, parseModulePath)
	writeHotReloadBrowserApp(parseT, parseAppRoot, parseModulePath)
	writeHotReloadBrowserChangedPanel(parseT, filepath.Join(parseAppRoot, "widgets", "changed_panel.go"), parseVersion)
	copyHotReloadBrowserWASMExec(parseT, parseAppRoot)
}

func prepareHotReloadBrowserFixtureModule(parseT *testing.T, parseAppRoot string) {
	parseT.Helper()
	parseCmd := exec.Command("go", "mod", "tidy")
	parseCmd.Dir = parseAppRoot
	parseCmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	parseOutput, parseErr := parseCmd.CombinedOutput()
	if parseErr != nil {
		parseT.Fatalf("tidy hot reload browser fixture: %v\n%s", parseErr, string(parseOutput))
	}
}

func writeHotReloadBrowserIndex(parseT *testing.T, parseAppRoot string) {
	parseT.Helper()
	writeHotReloadBrowserFile(parseT, filepath.Join(parseAppRoot, "index.html"), `<!doctype html>
<html lang="en">
<head>
	<meta charset="utf-8">
	<title>Hot Reload Browser E2E</title>
	<script src="./wasm_exec.js"></script>
</head>
<body>
	<div id="app"></div>
	<div id="boot-error"></div>
	<script>
		const go = new Go();
		WebAssembly.instantiateStreaming(fetch("./bin/main.wasm"), go.importObject)
			.then((result) => go.run(result.instance))
			.catch((error) => {
				document.getElementById("boot-error").textContent = String(error);
				console.error(error);
			});
	</script>
</body>
</html>
`)
}

func writeHotReloadBrowserMain(parseT *testing.T, parseAppRoot string, parseModulePath string) {
	parseT.Helper()
	writeHotReloadBrowserFile(parseT, filepath.Join(parseAppRoot, "main.go"), fmt.Sprintf(`//go:build js && wasm
// +build js,wasm

package main

import (
	"example.com/gwc-hot-reload-browser-e2e/widgets"
	"%s/hotreload"
	"%s/ui"
	"%s/utils"
)

func main() {
	hotreload.Enable()
	ui.Render(ui.CreateElement(widgets.App), "#app")
	utils.WaitForever()
}
`, parseModulePath, parseModulePath, parseModulePath))
}

func writeHotReloadBrowserApp(parseT *testing.T, parseAppRoot string, parseModulePath string) {
	parseT.Helper()
	writeHotReloadBrowserFile(parseT, filepath.Join(parseAppRoot, "widgets", "app.go"), fmt.Sprintf(`package widgets

import (
	"fmt"

	"%s/html"
	"%s/ui"
)

func App() ui.Node {
	return html.Div(html.Props{ID: "hot-reload-root"},
		ui.CreateElement(StableCounterPanel),
		ui.CreateElement(ChangedCounterPanel),
	)
}

func StableCounterPanel() ui.Node {
	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int { return parsePrevious + 1 })
	})

	return html.Div(html.Props{ID: "stable-panel"},
		html.P(html.Props{ID: "stable-count"}, html.Text(fmt.Sprintf("Stable count: %%d", parseCount.Get()))),
		html.Button(html.Props{ID: "stable-increment", OnClick: parseIncrement}, html.Text("Increment stable")),
	)
}
`, parseModulePath, parseModulePath))
}

func writeHotReloadBrowserChangedPanel(parseT *testing.T, parsePath string, parseVersion string) {
	parseT.Helper()
	writeHotReloadBrowserFile(parseT, parsePath, fmt.Sprintf(`package widgets

import (
	"fmt"

	"github.com/monstercameron/GoWebComponents/html"
	"github.com/monstercameron/GoWebComponents/ui"
)

func ChangedCounterPanel() ui.Node {
	const changedSubtreeVersion = %q

	parseCount := ui.UseState(0)
	parseIncrement := ui.UseEvent(func() {
		parseCount.Update(func(parsePrevious int) int { return parsePrevious + 1 })
	})

	return html.Div(html.Props{ID: "changed-panel"},
		html.P(html.Props{ID: "changed-version"}, html.Text("Changed subtree version: "+changedSubtreeVersion)),
		html.P(html.Props{ID: "changed-count"}, html.Text(fmt.Sprintf("Changed count: %%d", parseCount.Get()))),
		html.Button(html.Props{ID: "changed-increment", OnClick: parseIncrement}, html.Text("Increment changed")),
	)
}
`, parseVersion))
}

func copyHotReloadBrowserWASMExec(parseT *testing.T, parseAppRoot string) {
	parseT.Helper()
	parseSourcePath, parseErr := resolveWasmExecPath()
	if parseErr != nil {
		parseT.Fatalf("resolve wasm_exec.js: %v", parseErr)
	}
	parseContent, parseErr := os.ReadFile(parseSourcePath)
	if parseErr != nil {
		parseT.Fatalf("read wasm_exec.js: %v", parseErr)
	}
	writeHotReloadBrowserFile(parseT, filepath.Join(parseAppRoot, "wasm_exec.js"), string(parseContent))
}

func writeHotReloadBrowserFile(parseT *testing.T, parsePath string, parseContent string) {
	parseT.Helper()
	if parseErr := os.MkdirAll(filepath.Dir(parsePath), 0o755); parseErr != nil {
		parseT.Fatalf("create parent dir for %s: %v", parsePath, parseErr)
	}
	if parseErr := os.WriteFile(parsePath, []byte(parseContent), 0o644); parseErr != nil {
		parseT.Fatalf("write %s: %v", parsePath, parseErr)
	}
}
