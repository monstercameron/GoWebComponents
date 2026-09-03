//go:build playwrightgo

package main

import (
	"fmt"
	"strings"

	playwright "github.com/mxschmitt/playwright-go"
)

// proxyPage is a resolved page for a browser-proxy verb plus the cleanup needed
// to release playwright. When attached over CDP, closeBrowser is false so the
// engineer's live `gwc browser` window is left running.
type proxyPage struct {
	pw          *playwright.Playwright
	browser     playwright.Browser
	page        playwright.Page
	attached    bool
	closeBrowser bool
}

// close releases the page's resources, leaving an attached engineer window open.
func (parseP *proxyPage) close() {
	if parseP == nil {
		return
	}
	if parseP.closeBrowser && parseP.browser != nil {
		_ = parseP.browser.Close()
	}
	if parseP.pw != nil {
		_ = parseP.pw.Stop()
	}
}

// openProxyPage resolves a page for a proxy verb: with cdpEndpoint it attaches
// to the engineer's running window (front page) over CDP; otherwise it launches
// a fresh sandbox-disabled headless Chromium against url. Shared by the input,
// console/network, and dom/eval verbs.
func openProxyPage(parseCDPEndpoint string, parseURL string, parseWidth int, parseHeight int) (*proxyPage, error) {
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

	if strings.TrimSpace(parseCDPEndpoint) != "" {
		parseBrowser, parseErr := parsePw.Chromium.ConnectOverCDP(parseCDPEndpoint)
		if parseErr != nil {
			_ = parsePw.Stop()
			return nil, fmt.Errorf("connect over CDP %s: %w", parseCDPEndpoint, parseErr)
		}
		parsePage := firstExistingPage(parseBrowser)
		if parsePage == nil {
			if strings.TrimSpace(parseURL) == "" {
				_ = parseBrowser.Close()
				_ = parsePw.Stop()
				return nil, fmt.Errorf("attached browser has no open page and no -url to open")
			}
			parsePage, parseErr = parseBrowser.NewPage()
			if parseErr != nil {
				_ = parseBrowser.Close()
				_ = parsePw.Stop()
				return nil, fmt.Errorf("new page on attached browser: %w", parseErr)
			}
		}
		if strings.TrimSpace(parseURL) != "" {
			if _, parseErr := parsePage.Goto(parseURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle}); parseErr != nil {
				_ = parsePw.Stop()
				return nil, fmt.Errorf("goto %s: %w", parseURL, parseErr)
			}
		}
		return &proxyPage{pw: parsePw, browser: parseBrowser, page: parsePage, attached: true, closeBrowser: false}, nil
	}

	if strings.TrimSpace(parseURL) == "" {
		_ = parsePw.Stop()
		return nil, fmt.Errorf("-url is required when not attaching over -cdp")
	}
	parseBrowser, parseErr := parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{
		Headless: playwright.Bool(true),
		Args:     []string{"--no-sandbox", "--disable-dev-shm-usage"},
	})
	if parseErr != nil {
		_ = parsePw.Stop()
		return nil, fmt.Errorf("launch chromium: %w", parseErr)
	}
	parsePage, parseErr := parseBrowser.NewPage()
	if parseErr != nil {
		_ = parseBrowser.Close()
		_ = parsePw.Stop()
		return nil, fmt.Errorf("new page: %w", parseErr)
	}
	_ = parsePage.SetViewportSize(parseWidth, parseHeight)
	if _, parseErr := parsePage.Goto(parseURL, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateNetworkidle}); parseErr != nil {
		_ = parseBrowser.Close()
		_ = parsePw.Stop()
		return nil, fmt.Errorf("goto %s: %w", parseURL, parseErr)
	}
	return &proxyPage{pw: parsePw, browser: parseBrowser, page: parsePage, attached: false, closeBrowser: true}, nil
}
