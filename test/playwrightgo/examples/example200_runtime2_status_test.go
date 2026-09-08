//go:build playwrightgo

package playwrightgoexamples_test

import (
	"net"
	"os"
	"path/filepath"
	"runtime"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

// TestExample200RetiredRuntime2RouteIsUnavailable protects the v5 P5.2 migration boundary, not retired renderer execution.
func TestExample200RetiredRuntime2RouteIsUnavailable(parseTest *testing.T) {
	// Commit eeda966e intentionally removed this renderer and its example. Rebuilding
	// an unrelated runtime under its label would fabricate historical coverage.
	_, parseFile, _, _ := runtime.Caller(0)
	parseRoot := examplesRepoRootFromFile(parseFile)
	for _, parsePath := range []string{"examples/testing/runtime2-status/main.go", "examples/testing/runtime2-status/backgroundworker/main.go"} {
		if _, parseErr := os.Stat(filepath.Join(parseRoot, filepath.FromSlash(parsePath))); !os.IsNotExist(parseErr) {
			parseTest.Fatalf("retired runtime2 source unexpectedly present or unreadable: %s: %v", parsePath, parseErr)
		}
	}
	parseListener, parseErr := net.Listen("tcp", "127.0.0.1:0")
	if parseErr != nil {
		parseTest.Fatal(parseErr)
	}
	_, parsePort, _ := net.SplitHostPort(parseListener.Addr().String())
	_ = parseListener.Close()
	parseStop := startExamplesCommand(parseTest, parseRoot, "go", "run", "./tools/gwc", "serve", "-root", ".", "-host", "127.0.0.1", "-port", parsePort)
	parseTest.Cleanup(parseStop)
	parseBaseURL := "http://127.0.0.1:" + parsePort
	waitForHealthyExamplesURL(parseTest, parseBaseURL+"/healthz", 30*time.Second)
	withExamplesPage(parseTest, func(parsePage playwright.Page) {
		parseResponse, parseErr := parsePage.Goto(parseBaseURL+"/examples/testing/runtime2-status/runtime2-status.html", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
		if parseErr != nil || parseResponse == nil || parseResponse.Status() != 404 {
			parseTest.Fatalf("retired renderer route must explicitly return 404, got response=%v error=%v", parseResponse, parseErr)
		}
	})
}
