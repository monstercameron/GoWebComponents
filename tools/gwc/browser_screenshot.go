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

// screenshotResult is the JSON payload returned by the screenshot command.
type screenshotResult struct {
	Path      string `json:"path"`
	URL       string `json:"url"`
	Selector  string `json:"selector,omitempty"`
	Width     int    `json:"width"`
	Height    int    `json:"height"`
	Bytes     int    `json:"bytes"`
	Headless  bool   `json:"headless"`
	NoSandbox bool   `json:"noSandbox"`
}

// captureScreenshot launches Chromium (via playwright-go, sandbox disabled,
// CDP-backed) against parseURL and writes a PNG of the page — or of a single
// selector — to parseOut. It is the testable core of `gwc screenshot`.
func captureScreenshot(parseURL string, parseSelector string, parseOut string, parseWidth int, parseHeight int, isHeadless bool) (screenshotResult, error) {
	if parseWidth <= 0 {
		parseWidth = 1280
	}
	if parseHeight <= 0 {
		parseHeight = 800
	}
	// Ensure the chromium build is present (no-op when already installed).
	_ = playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}})

	parsePw, parseErr := playwright.Run(&playwright.RunOptions{Browsers: []string{"chromium"}})
	if parseErr != nil {
		return screenshotResult{}, fmt.Errorf("start playwright: %w", parseErr)
	}
	defer func() { _ = parsePw.Stop() }()

	parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(isHeadless),
		Args:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
	})
	if parseErr != nil {
		return screenshotResult{}, fmt.Errorf("launch chromium: %w", parseErr)
	}
	defer func() { _ = parseBrowser.Close() }()

	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		return screenshotResult{}, fmt.Errorf("new page: %w", parseErr)
	}
	if parseErr := parsePage.SetViewportSize(parseWidth, parseHeight); parseErr != nil {
		return screenshotResult{}, fmt.Errorf("set viewport: %w", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); parseErr != nil {
		return screenshotResult{}, fmt.Errorf("goto %s: %w", parseURL, parseErr)
	}

	if parseDir := filepath.Dir(parseOut); parseDir != "" {
		_ = os.MkdirAll(parseDir, 0o755)
	}

	var parseBytes []byte
	parseSelector = strings.TrimSpace(parseSelector)
	if parseSelector != "" {
		parseBytes, parseErr = parsePage.Locator(parseSelector).Screenshot(playwright.LocatorScreenshotOptions{
			Path: playwright.String(parseOut),
			Type: playwright.ScreenshotTypePng,
		})
	} else {
		parseBytes, parseErr = parsePage.Screenshot(playwright.PageScreenshotOptions{
			Path:     playwright.String(parseOut),
			FullPage: playwright.Bool(true),
			Type:     playwright.ScreenshotTypePng,
		})
	}
	if parseErr != nil {
		return screenshotResult{}, fmt.Errorf("screenshot: %w", parseErr)
	}

	return screenshotResult{
		Path:      parseOut,
		URL:       parseURL,
		Selector:  parseSelector,
		Width:     parseWidth,
		Height:    parseHeight,
		Bytes:     len(parseBytes),
		Headless:  isHeadless,
		NoSandbox: true,
	}, nil
}

// runScreenshotCommand is the gwc CLI / MCP entry point for browser screenshots.
var runScreenshotCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("screenshot", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to open and capture")
	parseOut := parseFs.String("out", "bin/screenshot.png", "output PNG path")
	parseSelector := parseFs.String("selector", "", "optional CSS selector to capture instead of the full page")
	parseWidth := parseFs.Int("width", 1280, "viewport width")
	parseHeight := parseFs.Int("height", 800, "viewport height")
	parseHeadless := parseFs.Bool("headless", true, "run Chromium headless")
	parseFs.Bool("json", false, "emit the JSON envelope (always on for this command)")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseURL) == "" {
		return writeAgenticEnvelope("screenshot", false, nil, nil, fmt.Errorf("-url is required"))
	}
	parseResult, parseErr := captureScreenshot(*parseURL, *parseSelector, *parseOut, *parseWidth, *parseHeight, *parseHeadless)
	if parseErr != nil {
		return writeAgenticEnvelope("screenshot", false, nil, nil, parseErr)
	}
	return writeAgenticEnvelope("screenshot", true, parseResult, nil, nil)
}
