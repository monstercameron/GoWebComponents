//go:build playwrightgo

package atlascommerceostests

import (
	"os"
	"strings"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

func TestCaptureAtlasScreenshotBaselines(parseT *testing.T) {
	if os.Getenv("ATLAS_CAPTURE_SCREENSHOTS") != "1" {
		parseT.Skip("set ATLAS_CAPTURE_SCREENSHOTS=1 and ATLAS_BASE_URL to refresh Atlas screenshot baselines")
	}
	parseBaseURL := strings.TrimRight(strings.TrimSpace(os.Getenv("ATLAS_BASE_URL")), "/")
	if parseBaseURL == "" {
		parseT.Fatal("ATLAS_BASE_URL is required when ATLAS_CAPTURE_SCREENSHOTS=1")
	}
	parseManifest := loadManifest(parseT)
	if parseErr := playwright.Install(&playwright.RunOptions{Browsers: []string{"chromium"}, Verbose: false}); parseErr != nil {
		parseT.Fatalf("install chromium: %v", parseErr)
	}
	parsePW, parseErr := playwright.Run(&playwright.RunOptions{Browsers: []string{"chromium"}, Verbose: false})
	if parseErr != nil {
		parseT.Fatalf("run playwright: %v", parseErr)
	}
	defer func() {
		if parseStopErr := parsePW.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright: %v", parseStopErr)
		}
	}()
	parseBrowser, parseErr := parsePW.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	if parseErr != nil {
		parseT.Fatalf("launch chromium: %v", parseErr)
	}
	defer func() {
		if parseCloseErr := parseBrowser.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()
	for _, parseBaseline := range parseManifest.ScreenshotBaselines {
		for _, parseViewport := range parseBaseline.Viewports {
			parseSize := screenshotViewportSize(parseViewport)
			parseContext, parseErr2 := parseBrowser.NewContext(playwright.BrowserNewContextOptions{
				Viewport: &parseSize,
			})
			if parseErr2 != nil {
				parseT.Fatalf("new context for %s: %v", parseViewport, parseErr2)
			}
			if parseBaseline.Surface == "internal" {
				if parseCookieErr := parseContext.AddCookies([]playwright.OptionalCookie{
					{Name: "atlas_mock_role", Value: "inventory_manager", URL: playwright.String(parseBaseURL)},
				}); parseCookieErr != nil {
					parseT.Fatalf("add internal mock cookie: %v", parseCookieErr)
				}
			}
			parsePage, parseErr2 := parseContext.NewPage()
			if parseErr2 != nil {
				parseT.Fatalf("new page for %s: %v", parseViewport, parseErr2)
			}
			for _, parseTheme := range parseBaseline.Themes {
				captureAtlasBaseline(parseT, parsePage, parseBaseURL, parseManifest, parseBaseline, parseTheme, parseViewport)
			}
			if parseCloseErr := parseContext.Close(); parseCloseErr != nil {
				parseT.Errorf("close context for %s: %v", parseViewport, parseCloseErr)
			}
		}
	}
}

func captureAtlasBaseline(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseManifest parseManifest, parseBaseline parseScreenshotBaseline, parseTheme string, parseViewport string) {
	parseT.Helper()
	parseColorScheme := playwright.ColorSchemeDark
	if parseTheme == "light" {
		parseColorScheme = playwright.ColorSchemeLight
	}
	if parseErr := parsePage.EmulateMedia(playwright.PageEmulateMediaOptions{ColorScheme: parseColorScheme}); parseErr != nil {
		parseT.Fatalf("emulate %s media: %v", parseTheme, parseErr)
	}
	if _, parseErr := parsePage.Goto(parseBaseURL+parseBaseline.Route, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateLoad,
		Timeout:   playwright.Float(45000),
	}); parseErr != nil {
		parseT.Fatalf("goto %s: %v", parseBaseline.Route, parseErr)
	}
	if _, parseErr := parsePage.Evaluate(`(theme) => {
		document.documentElement.classList.remove('atlas-theme-dark', 'atlas-theme-light');
		document.documentElement.classList.add('atlas-theme-' + theme);
		document.documentElement.style.colorScheme = theme;
		const style = document.createElement('style');
		style.textContent = '*, *::before, *::after { animation-duration: 0s !important; transition-duration: 0s !important; caret-color: transparent !important; scroll-behavior: auto !important; }';
		document.head.appendChild(style);
	}`, parseTheme); parseErr != nil {
		parseT.Fatalf("prepare theme %s for %s: %v", parseTheme, parseBaseline.Route, parseErr)
	}
	time.Sleep(250 * time.Millisecond)
	parsePath := screenshotBaselinePath(parseManifest.ScreenshotConventions.Directory, parseBaseline, parseTheme, parseViewport)
	if _, parseErr := parsePage.Screenshot(playwright.PageScreenshotOptions{
		Path:     playwright.String(parsePath),
		FullPage: playwright.Bool(true),
		Type:     playwright.ScreenshotTypePng,
		Scale:    playwright.ScreenshotScaleCss,
	}); parseErr != nil {
		parseT.Fatalf("capture %s %s %s: %v", parseBaseline.ID, parseTheme, parseViewport, parseErr)
	}
	assertPNGNonBlank(parseT, parsePath)
}

func screenshotViewportSize(parseViewport string) playwright.Size {
	if parseViewport == "mobile" {
		return playwright.Size{Width: 390, Height: 844}
	}
	return playwright.Size{Width: 1440, Height: 1100}
}
