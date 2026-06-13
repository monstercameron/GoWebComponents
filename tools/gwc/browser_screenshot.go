//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"os"
	"path/filepath"
	"strings"

	playwright "github.com/playwright-community/playwright-go"
)

// screenshotOptions configures a single capture. When CDPEndpoint is set the
// capture attaches to an already-running browser (the engineer's headed window
// opened by `gwc browser`) instead of launching its own — so the screenshot is
// exactly what is on screen. Otherwise it launches a fresh headless Chromium.
type screenshotOptions struct {
	URL         string
	Selector    string
	Out         string
	CDPEndpoint string
	Width       int
	Height      int
	Headless    bool
}

// screenshotResult is the JSON payload returned by the screenshot command.
type screenshotResult struct {
	Path      string `json:"path"`
	URL       string `json:"url,omitempty"`
	Selector  string `json:"selector,omitempty"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Bytes     int    `json:"bytes"`
	Headless  bool   `json:"headless"`
	NoSandbox bool   `json:"noSandbox"`
	Attached  bool   `json:"attached"`
}

// firstExistingPage returns the front-most existing page of a connected browser,
// or nil when the browser has no open pages.
func firstExistingPage(parseBrowser playwright.Browser) playwright.Page {
	for _, parseCtx := range parseBrowser.Contexts() {
		parsePages := parseCtx.Pages()
		if len(parsePages) > 0 {
			return parsePages[len(parsePages)-1]
		}
	}
	return nil
}

// captureScreenshot writes a PNG of a page — full page or a single selector — to
// opts.Out. With opts.CDPEndpoint it attaches to the engineer's open browser
// window over CDP (capturing exactly what is on screen); otherwise it launches
// a fresh sandbox-disabled headless Chromium. It is the testable core of
// `gwc screenshot`.
func captureScreenshot(parseOpts screenshotOptions) (screenshotResult, error) {
	if parseOpts.Width <= 0 {
		parseOpts.Width = 1280
	}
	if parseOpts.Height <= 0 {
		parseOpts.Height = 800
	}
	// Ensure the chromium build is present (no-op when already installed).
	_ = playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}})

	parsePw, parseErr := playwright.Run(&playwright.RunOptions{Browsers: []string{"chromium"}})
	if parseErr != nil {
		return screenshotResult{}, fmt.Errorf("start playwright: %w", parseErr)
	}
	defer func() { _ = parsePw.Stop() }()

	var parsePage playwright.Page
	parseAttached := false

	if strings.TrimSpace(parseOpts.CDPEndpoint) != "" {
		// Attach to the engineer's already-open window over CDP. Do NOT close
		// this browser on exit — it belongs to the live `gwc browser` session.
		parseBrowser, parseErr := parsePw.Chromium.ConnectOverCDP(parseOpts.CDPEndpoint)
		if parseErr != nil {
			return screenshotResult{}, fmt.Errorf("connect over CDP %s: %w", parseOpts.CDPEndpoint, parseErr)
		}
		parseAttached = true
		parsePage = firstExistingPage(parseBrowser)
		if parsePage == nil {
			if strings.TrimSpace(parseOpts.URL) == "" {
				return screenshotResult{}, fmt.Errorf("attached browser has no open page and no -url to open")
			}
			parsePage, parseErr = parseBrowser.NewPage()
			if parseErr != nil {
				return screenshotResult{}, fmt.Errorf("new page on attached browser: %w", parseErr)
			}
		}
		if strings.TrimSpace(parseOpts.URL) != "" {
			if _, parseErr := parsePage.Goto(parseOpts.URL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateNetworkidle,
			}); parseErr != nil {
				return screenshotResult{}, fmt.Errorf("goto %s: %w", parseOpts.URL, parseErr)
			}
		}
	} else {
		// Launch our own sandbox-disabled Chromium.
		if strings.TrimSpace(parseOpts.URL) == "" {
			return screenshotResult{}, fmt.Errorf("-url is required when not attaching over -cdp")
		}
		parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
			Headless: playwright.Bool(parseOpts.Headless),
			Args:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
		})
		if parseErr != nil {
			return screenshotResult{}, fmt.Errorf("launch chromium: %w", parseErr)
		}
		defer func() { _ = parseBrowser.Close() }()

		parsePage, parseErr = parseBrowser.NewPage()
		if parseErr != nil {
			return screenshotResult{}, fmt.Errorf("new page: %w", parseErr)
		}
		if parseErr := parsePage.SetViewportSize(parseOpts.Width, parseOpts.Height); parseErr != nil {
			return screenshotResult{}, fmt.Errorf("set viewport: %w", parseErr)
		}
		if _, parseErr := parsePage.Goto(parseOpts.URL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateNetworkidle,
		}); parseErr != nil {
			return screenshotResult{}, fmt.Errorf("goto %s: %w", parseOpts.URL, parseErr)
		}
	}

	if parseDir := filepath.Dir(parseOpts.Out); parseDir != "" {
		_ = os.MkdirAll(parseDir, 0o755)
	}

	var parseBytes []byte
	parseSelector := strings.TrimSpace(parseOpts.Selector)
	if parseSelector != "" {
		parseBytes, parseErr = parsePage.Locator(parseSelector).Screenshot(playwright.LocatorScreenshotOptions{
			Path: playwright.String(parseOpts.Out),
			Type: playwright.ScreenshotTypePng,
		})
	} else {
		parseBytes, parseErr = parsePage.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(parseOpts.Out),
			FullPage: playwright.Bool(true),
			Type:     playwright.ScreenshotTypePng,
		})
	}
	if parseErr != nil {
		return screenshotResult{}, fmt.Errorf("screenshot: %w", parseErr)
	}

	return screenshotResult{
		Path:      parseOpts.Out,
		URL:       parseOpts.URL,
		Selector:  parseSelector,
		Width:     parseOpts.Width,
		Height:    parseOpts.Height,
		Bytes:     len(parseBytes),
		Headless:  parseOpts.Headless,
		NoSandbox: true,
		Attached:  parseAttached,
	}, nil
}

// runScreenshotCommand is the gwc CLI / MCP entry point for browser screenshots.
var runScreenshotCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("screenshot", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to open and capture (required unless -cdp is set)")
	parseOut := parseFs.String("out", "bin/screenshot.png", "output PNG path")
	parseSelector := parseFs.String("selector", "", "optional CSS selector to capture instead of the full page")
	parseCDP := parseFs.String("cdp", "", "attach to a running browser's CDP endpoint (e.g. http://127.0.0.1:9222) instead of launching one — captures the engineer's open `gwc browser` window")
	parseWidth := parseFs.Int("width", 1280, "viewport width")
	parseHeight := parseFs.Int("height", 800, "viewport height")
	parseHeadless := parseFs.Bool("headless", true, "run Chromium headless (ignored when -cdp is set)")
	parseFs.Bool("json", false, "emit the JSON envelope (always on for this command)")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseURL) == "" && strings.TrimSpace(*parseCDP) == "" {
		return writeAgenticEnvelope("screenshot", false, nil, nil, fmt.Errorf("-url is required (or attach to an open window with -cdp)"))
	}
	parseResult, parseErr := captureScreenshot(screenshotOptions{
		URL:         *parseURL,
		Selector:    *parseSelector,
		Out:         *parseOut,
		CDPEndpoint: *parseCDP,
		Width:       *parseWidth,
		Height:      *parseHeight,
		Headless:    *parseHeadless,
	})
	if parseErr != nil {
		return writeAgenticEnvelope("screenshot", false, nil, nil, parseErr)
	}
	return writeAgenticEnvelope("screenshot", true, parseResult, nil, nil)
}
