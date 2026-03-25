//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"net/url"
	"strings"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

func TestPlaywrightGoChromiumSmoke(t *testing.T) {
	runOptions := &playwright.RunOptions{
		Browsers: []string{"chromium"},
		Verbose:  false,
	}

	if err := playwright.Install(runOptions); err != nil {
		t.Fatalf("install playwright-go driver/browser: %v", err)
	}

	pw, err := playwright.Run(runOptions)
	if err != nil {
		t.Fatalf("run playwright-go driver: %v", err)
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
		t.Fatalf("new page: %v", err)
	}

	html := "<html><head><title>pwgo-smoke</title></head><body><h1 id='ready'>ready</h1></body></html>"
	dataURL := "data:text/html," + url.PathEscape(html)
	if _, err := page.Goto(dataURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); err != nil {
		t.Fatalf("goto smoke data url: %v", err)
	}

	title, err := page.Title()
	if err != nil {
		t.Fatalf("read page title: %v", err)
	}
	if title != "pwgo-smoke" {
		t.Fatalf("unexpected title: got %q want %q", title, "pwgo-smoke")
	}

	ready, err := page.TextContent("#ready")
	if err != nil {
		t.Fatalf("read #ready text: %v", err)
	}
	if strings.TrimSpace(ready) != "ready" {
		t.Fatalf("unexpected #ready text: got %q want %q", ready, "ready")
	}
}
