//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"net/url"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestMainSuite(t *testing.T) {
	if err := playwright.Install(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}); err != nil {
		t.Fatalf("install playwright-go chromium: %v", err)
	}

	pw, err := playwright.Run(&playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	})
	if err != nil {
		t.Fatalf("run playwright-go: %v", err)
	}
	defer func() {
		if stopErr := pw.Stop(); stopErr != nil {
			t.Errorf("stop playwright-go: %v", stopErr)
		}
	}()

	browser, err := pw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
	})
	if err != nil {
		t.Fatalf("launch chromium: %v", err)
	}
	defer func() {
		if closeErr := browser.Close(); closeErr != nil {
			t.Errorf("close chromium: %v", closeErr)
		}
	}()

	page, err := browser.NewPage()
	if err != nil {
		t.Fatalf("create browser page: %v", err)
	}

	projectName := "golden-reference-app"
	html := "<html><body><h1 id='starter-heading'>" + projectName + "</h1></body></html>"
	dataURL := "data:text/html," + url.PathEscape(html)
	if _, err := page.Goto(dataURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("goto starter smoke page: %v", err)
	}

	heading, err := page.TextContent("#starter-heading")
	if err != nil {
		t.Fatalf("read starter heading: %v", err)
	}
	if !strings.EqualFold(strings.TrimSpace(heading), projectName) {
		t.Fatalf("unexpected starter heading: got %q want %q", heading, projectName)
	}
}
