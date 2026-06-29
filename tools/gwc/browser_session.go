//go:build playwrightgo

package main

import (
	"flag"
	"fmt"
	"os"
	"os/signal"
	"strings"
	"syscall"

	playwright "github.com/playwright-community/playwright-go"
)

// browserSessionResult is the JSON payload describing an opened headed window.
type browserSessionResult struct {
	URL         string `json:"url"`
	CDPEndpoint string `json:"cdpEndpoint"`
	CDPPort     int    `json:"cdpPort"`
	Width       int    `json:"width"`
	Height      int    `json:"height"`
	Headed      bool   `json:"headed"`
	NoSandbox   bool   `json:"noSandbox"`
	Waiting     bool   `json:"waiting"`
}

// browserSession holds the live playwright handles for an open headed window so
// the caller can keep it alive and close it cleanly.
type browserSession struct {
	pw      *playwright.Playwright
	browser playwright.Browser
	Result  browserSessionResult
}

// close tears the session down (browser then driver), best-effort.
func (parseS *browserSession) close() {
	if parseS == nil {
		return
	}
	if parseS.browser != nil {
		_ = parseS.browser.Close()
	}
	if parseS.pw != nil {
		_ = parseS.pw.Stop()
	}
}

// openBrowserSession launches a VISIBLE (headed), sandbox-disabled Chromium with
// a CDP debugging port, navigates it to url, and returns the live session. The
// window stays open until the caller closes it — it is what the engineer watches
// during copilot dev while the agent drives the app and attaches over CDP. It is
// the testable core of `gwc browser`.
func openBrowserSession(parseURL string, parseCDPPort int, parseWidth int, parseHeight int) (*browserSession, error) {
	if strings.TrimSpace(parseURL) == "" {
		return nil, fmt.Errorf("-url is required")
	}
	if parseCDPPort <= 0 {
		parseCDPPort = 9222
	}
	if parseWidth <= 0 {
		parseWidth = 1280
	}
	if parseHeight <= 0 {
		parseHeight = 800
	}

	_ = playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}})

	parsePw, parseErr := playwright.Run(&playwright.RunOptions{Browsers: []string{"chromium"}})
	if parseErr != nil {
		return nil, fmt.Errorf("start playwright: %w", parseErr)
	}

	parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(false),
		Args: []string{
			"--no-sandbox",
			"--disable-dev-shm-usage",
			fmt.Sprintf("--remote-debugging-port=%d", parseCDPPort),
		},
	})
	if parseErr != nil {
		_ = parsePw.Stop()
		return nil, fmt.Errorf("launch headed chromium: %w", parseErr)
	}

	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		_ = parseBrowser.Close()
		_ = parsePw.Stop()
		return nil, fmt.Errorf("new page: %w", parseErr)
	}
	if parseErr := parsePage.SetViewportSize(parseWidth, parseHeight); parseErr != nil {
		_ = parseBrowser.Close()
		_ = parsePw.Stop()
		return nil, fmt.Errorf("set viewport: %w", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateNetworkidle,
	}); parseErr != nil {
		_ = parseBrowser.Close()
		_ = parsePw.Stop()
		return nil, fmt.Errorf("goto %s: %w", parseURL, parseErr)
	}

	return &browserSession{
		pw:      parsePw,
		browser: parseBrowser,
		Result: browserSessionResult{
			URL:         parseURL,
			CDPEndpoint: fmt.Sprintf("http://127.0.0.1:%d", parseCDPPort),
			CDPPort:     parseCDPPort,
			Width:       parseWidth,
			Height:      parseHeight,
			Headed:      true,
			NoSandbox:   true,
		},
	}, nil
}

// runBrowserCommand opens a headed Chromium window on -url and keeps it open
// until interrupted (Ctrl+C / SIGTERM) so the engineer can watch the app while
// copilot drives it. Its CDP endpoint is reported so `gwc screenshot -cdp <ep>`
// (and future click/type tools) can attach to this exact window.
var runBrowserCommand = func(parseL launcher, parseArgs []string) error {
	parseFs := flag.NewFlagSet("browser", flag.ContinueOnError)
	parseURL := parseFs.String("url", "", "URL to open in the visible window (required)")
	parseCDPPort := parseFs.Int("cdp-port", 9222, "CDP debugging port to expose for copilot attach")
	parseWidth := parseFs.Int("width", 1280, "window width")
	parseHeight := parseFs.Int("height", 800, "window height")
	parseWait := parseFs.Bool("wait", true, "keep the window open until interrupted (Ctrl+C)")
	parseFs.Bool("json", false, "emit the JSON envelope (always on for this command)")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		return parseErr
	}
	if strings.TrimSpace(*parseURL) == "" {
		return writeAgenticEnvelope("browser", false, nil, nil, fmt.Errorf("-url is required"))
	}

	parseSession, parseErr := openBrowserSession(*parseURL, *parseCDPPort, *parseWidth, *parseHeight)
	if parseErr != nil {
		return writeAgenticEnvelope("browser", false, nil, nil, parseErr)
	}
	defer parseSession.close()

	parseResult := parseSession.Result
	parseResult.Waiting = *parseWait
	if parseErr := writeAgenticEnvelope("browser", true, parseResult, nil, nil); parseErr != nil {
		return parseErr
	}

	if !*parseWait {
		return nil
	}

	// Human-facing hint goes to stderr so stdout stays a clean envelope.
	fmt.Fprintf(os.Stderr, "\ngwc browser: window open at %s (CDP %s). Copilot can attach with:\n  gwc screenshot -cdp %s -out bin/shot.png\nPress Ctrl+C to close.\n",
		parseResult.URL, parseResult.CDPEndpoint, parseResult.CDPEndpoint)

	parseSig := make(chan os.Signal, 1)
	signal.Notify(parseSig, os.Interrupt, syscall.SIGTERM)
	<-parseSig
	fmt.Fprintln(os.Stderr, "gwc browser: closing window.")
	return nil
}
