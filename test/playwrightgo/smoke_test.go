//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"net/url"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestPlaywrightGoChromiumSmoke(parseT *testing.T) {
	parseRunOptions := &playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}

	if parseErr := playwright.Install(parseRunOptions); parseErr != nil {
		parseT.Fatalf("install playwright-go driver/browser: %v", parseErr)
	}

	parsePw, parseErr2 := playwright.Run(parseRunOptions)
	if parseErr2 != nil {
		parseT.Fatalf("run playwright-go driver: %v", parseErr2)
	}
	defer func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright-go: %v", parseStopErr)
		}
	}()

	parseBrowser, parseErr2 := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if parseErr2 != nil {
		parseT.Fatalf("launch chromium: %v", parseErr2)
	}
	defer func() {
		if parseCloseErr := parseBrowser.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	parsePage, parseErr2 := parseBrowser.NewPage()
	if parseErr2 != nil {
		parseT.Fatalf("new page: %v", parseErr2)
	}

	parseHtml := "<html><head><title>pwgo-smoke</title></head><body><h1 id='ready'>ready</h1></body></html>"
	parseDataURL := "data:text/html," + url.PathEscape(parseHtml)
	if _, parseErr3 := parsePage.Goto(parseDataURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr3 != nil {
		parseT.Fatalf("goto smoke data url: %v", parseErr3)
	}

	parseTitle, parseErr2 := parsePage.Title()
	if parseErr2 != nil {
		parseT.Fatalf("read page title: %v", parseErr2)
	}
	if parseTitle != "pwgo-smoke" {
		parseT.Fatalf("unexpected title: got %q want %q", parseTitle, "pwgo-smoke")
	}

	parseReady, parseErr2 := parsePage.TextContent("#ready")
	if parseErr2 != nil {
		parseT.Fatalf("read #ready text: %v", parseErr2)
	}
	if strings.TrimSpace(parseReady) != "ready" {
		parseT.Fatalf("unexpected #ready text: got %q want %q", parseReady, "ready")
	}
}
