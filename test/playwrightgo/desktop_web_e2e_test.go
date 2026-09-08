//go:build playwrightgo

package playwrightgo_test

import (
	"net/http"
	"net/http/httptest"
	"os"
	"os/exec"
	"path/filepath"
	"strings"
	"sync"
	"testing"

	playwright "github.com/mxschmitt/playwright-go"
)

// TestDesktopLabWebE2E proves the real lab's web artifact has no native bootstrap.
func TestDesktopLabWebE2E(parseTest *testing.T) {
	parseRoot := repoRootFromTest(parseTest)
	// A fresh scaffold prevents old helper-generated assets from masking a broken CLI build.
	parseExample := filepath.Join(parseTest.TempDir(), "lab")
	parseCommand := exec.Command("go", "run", "./tools/gwc", "build", "-target", "web", "-root", parseExample, "-json")
	parseCommand.Dir = parseRoot
	for _, parseEntry := range os.Environ() {
		parseKey, _, _ := strings.Cut(parseEntry, "=")
		switch strings.ToUpper(parseKey) {
		case "GOOS", "GOARCH", "GOFLAGS", "CGO_ENABLED":
			continue
		}
		parseCommand.Env = append(parseCommand.Env, parseEntry)
	}
	parseInit := exec.Command("go", "run", "./tools/gwc", "desktop", "init", "-root", parseExample, "-json")
	parseInit.Dir = parseRoot
	parseInit.Env = parseCommand.Env
	if parseOutput, parseErr := parseInit.CombinedOutput(); parseErr != nil {
		parseTest.Fatalf("scaffold web lab: %v\n%s", parseErr, parseOutput)
	}
	if parseOutput, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
		parseTest.Fatalf("build web lab: %v\n%s", parseErr, parseOutput)
	}
	parseServer := httptest.NewServer(http.FileServer(http.Dir(filepath.Join(parseExample, "assets", "web"))))
	defer parseServer.Close()
	runChromiumPage(parseTest, func(parsePage playwright.Page) {
		var parseMutex sync.Mutex
		var parseProblems []string
		parsePage.On("pageerror", func(parseError error) {
			parseMutex.Lock()
			defer parseMutex.Unlock()
			parseProblems = append(parseProblems, parseError.Error())
		})
		parsePage.On("request", func(parseRequest playwright.Request) {
			if strings.Contains(parseRequest.URL(), "/wails/") || strings.Contains(parseRequest.URL(), "/bindings/") {
				parseMutex.Lock()
				defer parseMutex.Unlock()
				parseProblems = append(parseProblems, "web requested native module: "+parseRequest.URL())
			}
		})
		if _, parseErr := parsePage.Goto(parseServer.URL + "/#/tester"); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#api-open-file", playwright.PageWaitForSelectorOptions{Timeout: playwright.Float(60000)}); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		for _, parseID := range []string{"api-open-file", "api-open-files", "api-open-directory", "api-save-path", "api-message-info", "api-clipboard-write", "api-clipboard-read", "api-window-info", "api-screens", "api-export"} {
			if isDisabled, parseErr := parsePage.Locator("#" + parseID).IsDisabled(); parseErr != nil || !isDisabled {
				parseTest.Errorf("web native control %s must be disabled: %v", parseID, parseErr)
			}
		}
		if parseErr := parsePage.Locator("#api-editable").Fill("Web café 雪 123"); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		if parseValue, parseErr := parsePage.Locator("#api-editable").InputValue(); parseErr != nil || parseValue != "Web café 雪 123" {
			parseTest.Fatalf("web text roundtrip: %q %v", parseValue, parseErr)
		}
		parseScreenshot := filepath.Join(parseRoot, "bin", "v6-web-lab.png")
		if _, parseErr := parsePage.Screenshot(playwright.PageScreenshotOptions{Path: playwright.String(parseScreenshot), FullPage: playwright.Bool(true)}); parseErr != nil {
			parseTest.Fatalf("capture web tester evidence: %v", parseErr)
		}
		if _, parseErr := parsePage.Goto(parseServer.URL + "/#/counter"); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		if parseErr := parsePage.Locator("#local-increment").Click(); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.querySelector('#local-count')?.textContent === 'Count: 1'`, nil); parseErr != nil {
			parseTest.Fatal(parseErr)
		}
		if parseValue, parseErr := parsePage.Evaluate(`() => typeof globalThis.__gwcDesktop`); parseErr != nil || parseValue != "undefined" {
			parseTest.Errorf("web must not install native transport: %v %v", parseValue, parseErr)
		}
		parseMutex.Lock()
		defer parseMutex.Unlock()
		if len(parseProblems) != 0 {
			parseTest.Fatalf("web runtime errors: %v", parseProblems)
		}
	})
}
