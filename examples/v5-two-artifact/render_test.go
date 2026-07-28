//go:build playwrightgo

// The assertion the packaging tests structurally cannot make.
//
// packaging_test.go proves which SYMBOLS land in which binary — that app.wasm
// does not link the engine and services.wasm does. Those are good checks and they
// are also why this example shipped with its return path missing: every symbol
// assertion passed while the page rendered an empty <div>. The domain worker
// executed the command, committed the row, logged success, and no state ever
// crossed back, because nothing published deltas and nothing applied them. A
// symbol check cannot see that. Only rendering can.
//
// So this test asserts the one thing that actually matters about the two-artifact
// split: a row written to SQLite INSIDE THE WORKER appears on the render thread.
//
// Behind the `playwrightgo` tag because it needs a browser, matching the rest of
// the repo's browser suites. Run with:
//
//	go test -tags playwrightgo ./examples/v5-two-artifact/ -run TestWorkerOwnedStateReachesTheRenderThread -v
package twoartifact_test

import (
	"net"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"testing"

	"github.com/playwright-community/playwright-go"
)

// stageTwoArtifactApp builds both binaries and assembles a servable directory.
func stageTwoArtifactApp(parseT *testing.T) string {
	parseT.Helper()
	parseDir := parseT.TempDir()

	for parsePackage, parseName := range map[string]string{
		"./app":      "app.wasm",
		"./services": "services.wasm",
	} {
		parseCommand := exec.Command("go", "build", "-o", filepath.Join(parseDir, parseName), parsePackage)
		parseCommand.Env = append(parseCommand.Environ(), "GOOS=js", "GOARCH=wasm")
		if parseOutput, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
			parseT.Fatalf("build %s: %v\n%s", parsePackage, parseErr, parseOutput)
		}
	}

	for _, parseName := range []string{"index.html", "worker.js"} {
		parseBytes, parseErr := os.ReadFile(parseName)
		if parseErr != nil {
			parseT.Fatalf("read %s: %v", parseName, parseErr)
		}
		if parseErr := os.WriteFile(filepath.Join(parseDir, parseName), parseBytes, 0o644); parseErr != nil {
			parseT.Fatalf("stage %s: %v", parseName, parseErr)
		}
	}

	// wasm_exec.js ships with the toolchain, so it is copied from GOROOT rather
	// than vendored — a stale copy is version-skewed against the compiler and
	// fails in ways that look like an application bug.
	parseGoroot, parseErr := exec.Command("go", "env", "GOROOT").Output()
	if parseErr != nil {
		parseT.Fatalf("go env GOROOT: %v", parseErr)
	}
	parseShim := filepath.Join(strings.TrimSpace(string(parseGoroot)), "lib", "wasm", "wasm_exec.js")
	parseShimBytes, parseShimErr := os.ReadFile(parseShim)
	if parseShimErr != nil {
		parseT.Skipf("wasm_exec.js not found at %s: %v", parseShim, parseShimErr)
	}
	if parseErr := os.WriteFile(filepath.Join(parseDir, "wasm_exec.js"), parseShimBytes, 0o644); parseErr != nil {
		parseT.Fatalf("stage wasm_exec.js: %v", parseErr)
	}
	return parseDir
}

// serveStaged serves the directory with the correct wasm MIME type.
//
// WebAssembly.instantiateStreaming REJECTS a response that is not
// application/wasm — the spec makes the browser refuse rather than sniff — so a
// server that gets this wrong produces "expected magic word 00 61 73 6d" on
// perfectly good bytes and the example looks broken for a reason unrelated to it.
func serveStaged(parseT *testing.T, parseDir string) string {
	parseT.Helper()
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseT.Fatalf("listen: %v", parseErr)
	}
	parseFiles := http.FileServer(http.Dir(parseDir))
	parseServer := &http.Server{Handler: http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if strings.HasSuffix(parseR.URL.Path, ".wasm") {
			parseW.Header().Set("Content-Type", "application/wasm")
		}
		parseFiles.ServeHTTP(parseW, parseR)
	})}
	go func() { _ = parseServer.Serve(parseListener) }()
	parseT.Cleanup(func() { _ = parseServer.Close() })
	return "http://" + parseListener.Addr().String()
}

func TestWorkerOwnedStateReachesTheRenderThread(parseT *testing.T) {
	parseDir := stageTwoArtifactApp(parseT)
	parseBase := serveStaged(parseT, parseDir)

	parsePW, parseErr := playwright.Run()
	if parseErr != nil {
		parseT.Skipf("playwright unavailable: %v", parseErr)
	}
	defer parsePW.Stop()
	parseBrowser, parseBrowserErr := parsePW.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseBrowserErr != nil {
		parseT.Skipf("chromium unavailable: %v", parseBrowserErr)
	}
	defer parseBrowser.Close()

	parseContext, _ := parseBrowser.NewContext()
	parsePage, parsePageErr := parseContext.NewPage()
	if parsePageErr != nil {
		parseT.Fatalf("new page: %v", parsePageErr)
	}

	// Console and page errors are captured so a failure NAMES ITS CAUSE. Every
	// defect this example has had failed by producing nothing; a test that can only
	// say "waited 30 seconds" would have been no help finding any of them.
	var parseDiagnostics []string
	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		if parseMessage.Type() == "error" || parseMessage.Type() == "warning" {
			parseDiagnostics = append(parseDiagnostics, parseMessage.Type()+": "+parseMessage.Text())
		}
	})
	parsePage.OnPageError(func(parsePageError error) {
		parseDiagnostics = append(parseDiagnostics, "pageerror: "+parsePageError.Error())
	})

	if _, parseErr := parsePage.Goto(parseBase+"/index.html", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(60000),
	}); parseErr != nil {
		parseT.Fatalf("navigate: %v\ndiagnostics: %s", parseErr, strings.Join(parseDiagnostics, " | "))
	}

	// THE ASSERTION. Not "#app has children" — an empty <div> satisfies that, and
	// an empty <div> is exactly what this example rendered for as long as the
	// publication path was missing. This waits for a row whose TEXT came from a
	// value the worker wrote to SQLite.
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			const rows = document.querySelectorAll('#app .row');
			for (const row of rows) {
				if ((row.textContent || '').includes('first')) return true;
			}
			return false;
		}`,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(60000)},
	); parseErr != nil {
		parseHTML, _ := parsePage.Evaluate(`() => document.getElementById('app')?.innerHTML ?? '(no #app)'`)
		parseT.Fatalf("worker-owned state never reached the render thread.\n"+
			"  #app: %v\n"+
			"  diagnostics: %s\n"+
			"  The command path can succeed while this fails: check that services.wasm "+
			"publishes deltas and that the app applies them (projection.Apply) inside ui.PostAsync.",
			parseHTML, strings.Join(parseDiagnostics, " | "))
	}

	parseRowCount, _ := parsePage.Evaluate(`() => document.querySelectorAll('#app .row').length`)
	parseT.Logf("rendered %v row(s) published by the domain worker", parseRowCount)

	for _, parseDiagnostic := range parseDiagnostics {
		// AbortError from a skipped view transition is a known browser artifact.
		if strings.Contains(parseDiagnostic, "AbortError") {
			continue
		}
		parseT.Errorf("unexpected browser diagnostic: %s", parseDiagnostic)
	}
}
