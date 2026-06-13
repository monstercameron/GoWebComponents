//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"sort"
	"strconv"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

var installExamplesChromiumOnce sync.Once
var installExamplesChromiumErr error
var installExamplesBrowserMu sync.Mutex
var installExamplesBrowserErrByName = map[string]error{}

func ensureExamplesBrowserInstalled(parseBrowser string) error {
	parseKey := strings.TrimSpace(strings.ToLower(parseBrowser))
	if parseKey == "" {
		return nil
	}
	installExamplesBrowserMu.Lock()
	parseErr, parseExists := installExamplesBrowserErrByName[parseKey]
	installExamplesBrowserMu.Unlock()
	if parseExists {
		return parseErr
	}
	parseInstallErr := playwright.Install(&playwright.RunOptions{
		Browsers: []string{parseKey},
		Verbose:  false,
	})
	installExamplesBrowserMu.Lock()
	installExamplesBrowserErrByName[parseKey] = parseInstallErr
	installExamplesBrowserMu.Unlock()
	return parseInstallErr
}

func ensureExamplesChromiumInstalled() error {
	installExamplesChromiumOnce.Do(func() {
		installExamplesChromiumErr = ensureExamplesBrowserInstalled("chromium")
	})
	return installExamplesChromiumErr
}

func examplesRepoRootFromFile(parseTestFile string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(parseTestFile), "..", "..", ".."))
}

// terminateExamplesProcessTree terminates a command process and, on Windows, its descendant processes.
func terminateExamplesProcessTree(parseCmd *exec.Cmd) {
	if parseCmd == nil || parseCmd.Process == nil {
		return
	}
	if runtime.GOOS == "windows" {
		_ = exec.Command("taskkill", "/T", "/F", "/PID", strconv.Itoa(parseCmd.Process.Pid)).Run()
		return
	}
	_ = parseCmd.Process.Kill()
}

func startExamplesCommandWithEnv(parseT *testing.T, parseDir string, parseEnv []string, parseName string, parseArgs ...string) (parseStop func()) {
	parseT.Helper()
	parseCmd := exec.Command(parseName, parseArgs...)
	parseCmd.Dir = parseDir
	if len(parseEnv) > 0 {
		parseCmd.Env = append(os.Environ(), parseEnv...)
	}
	if os.Getenv("GWC_EXAMPLES_COMMAND_TRACE") == "1" {
		parseCmd.Stdout = os.Stdout
		parseCmd.Stderr = os.Stderr
	} else {
		parseCmd.Stdout = io.Discard
		parseCmd.Stderr = io.Discard
	}
	if parseErr := parseCmd.Start(); parseErr != nil {
		parseT.Fatalf("start %s %v: %v", parseName, parseArgs, parseErr)
	}
	parseDone := make(chan error, 1)
	go func() {
		parseDone <- parseCmd.Wait()
	}()
	return func() {
		terminateExamplesProcessTree(parseCmd)
		select {
		case <-parseDone:
		case <-time.After(5 * time.Second):
		}
	}
}

func startExamplesCommand(parseT *testing.T, parseDir string, parseName string, parseArgs ...string) (parseStop func()) {
	return startExamplesCommandWithEnv(parseT, parseDir, nil, parseName, parseArgs...)
}

func waitForHealthyExamplesURL(parseT *testing.T, parseHealthURL string, parseTimeout time.Duration) {
	parseT.Helper()
	parseDeadline := time.Now().Add(parseTimeout)
	for time.Now().Before(parseDeadline) {
		parseResp, parseErr := http.Get(parseHealthURL)
		if parseErr == nil {
			_ = parseResp.Body.Close()
			if parseResp.StatusCode >= 200 && parseResp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	parseT.Fatalf("health check timed out: %s", parseHealthURL)
}

func startExamplesCatalogServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseStop := startExamplesCommand(
		parseT,
		parseRepoRoot,
		"go",
		"run", "./tools/gwc", "examples",
		"-host", "127.0.0.1",
		"-port", parsePort,
	)
	parseT.Cleanup(parseStop)
	parseBaseURL := "http://127.0.0.1:" + parsePort
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 30*time.Second)
	return parseBaseURL
}

func launchExamplesBrowser(parsePw *playwright.Playwright, parseBrowser string) (playwright.Browser, error) {
	switch strings.TrimSpace(strings.ToLower(parseBrowser)) {
	case "chromium":
		return parsePw.Chromium.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	case "firefox":
		return parsePw.Firefox.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	case "webkit":
		return parsePw.WebKit.Launch(playwright.BrowserTypeLaunchOptions{Headless: playwright.Bool(true)})
	default:
		return nil, exec.ErrNotFound
	}
}

func withExamplesBrowserPage(parseT *testing.T, parseBrowser string, parseFn func(page playwright.Page)) {
	parseT.Helper()
	parseBrowserKey := strings.TrimSpace(strings.ToLower(parseBrowser))
	if parseBrowserKey == "" {
		parseBrowserKey = "chromium"
	}
	if parseErr := ensureExamplesBrowserInstalled(parseBrowserKey); parseErr != nil {
		parseT.Fatalf("install playwright browser (%s): %v", parseBrowserKey, parseErr)
	}
	parsePw, parseErr2 := playwright.Run(&playwright.RunOptions{
		Browsers: []string{parseBrowserKey},
		Verbose:  false,
	})
	if parseErr2 != nil {
		parseT.Fatalf("run playwright-go: %v", parseErr2)
	}
	defer func() {
		if parseStopErr := parsePw.Stop(); parseStopErr != nil {
			parseT.Errorf("stop playwright-go: %v", parseStopErr)
		}
	}()

	parseBrowserHandle, parseErr2 := launchExamplesBrowser(parsePw, parseBrowserKey)
	if parseErr2 != nil {
		parseT.Fatalf("launch %s: %v", parseBrowserKey, parseErr2)
	}
	defer func() {
		if parseCloseErr := parseBrowserHandle.Close(); parseCloseErr != nil {
			parseT.Errorf("close chromium: %v", parseCloseErr)
		}
	}()

	parsePage, parseErr2 := parseBrowserHandle.NewPage()
	if parseErr2 != nil {
		parseT.Fatalf("new page: %v", parseErr2)
	}
	parseFn(parsePage)
}

func withExamplesPage(parseT *testing.T, parseFn func(page playwright.Page)) {
	withExamplesBrowserPage(parseT, "chromium", parseFn)
}

func startAtlasExamplesServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseAddress := "127.0.0.1:" + parsePort
	parseStop := startExamplesCommandWithEnv(
		parseT,
		parseRepoRoot,
		[]string{"ATLAS_ADDR=" + parseAddress},
		"go",
		"run", "./examples/server/atlas-commerce-os/server",
	)
	parseT.Cleanup(parseStop)
	parseBaseURL := "http://" + parseAddress
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 45*time.Second)
	return parseBaseURL
}

// discoverPublicExampleRoutes resolves catalog routes for public examples.
// With an empty slug list it returns every example under examples/public;
// otherwise it validates each requested slug exists and fails loudly when one
// is missing so renames surface as test failures instead of silent skips.
func discoverPublicExampleRoutes(parseT *testing.T, parseRepoRoot string, parseSlugs []string) []string {
	parseT.Helper()
	parsePublicRoot := filepath.Join(parseRepoRoot, "examples", "public")

	if len(parseSlugs) == 0 {
		parseEntries, parseErr := os.ReadDir(parsePublicRoot)
		if parseErr != nil {
			parseT.Fatalf("read examples/public dir: %v", parseErr)
		}
		for _, parseEntry := range parseEntries {
			if parseEntry.IsDir() {
				parseSlugs = append(parseSlugs, parseEntry.Name())
			}
		}
	}

	var parseRoutes []string
	for _, parseSlug := range parseSlugs {
		if parseInfo, parseErr := os.Stat(filepath.Join(parsePublicRoot, parseSlug)); parseErr != nil || !parseInfo.IsDir() {
			parseT.Fatalf("public example slug %q not found under examples/public", parseSlug)
		}
		parseRoutes = append(parseRoutes, "/examples/public/"+parseSlug+"/")
	}
	sort.Strings(parseRoutes)
	return parseRoutes
}

func visitRouteAndAssertSuccess(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseRoute string) {
	parseT.Helper()
	parseResp, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto %s: %v", parseRoute, parseErr)
	}
	if parseResp == nil {
		parseT.Fatalf("nil response for route %s", parseRoute)
	}
	if parseResp.Status() >= 400 {
		parseT.Fatalf("route %s returned status %d", parseRoute, parseResp.Status())
	}
}

func visitRouteAndAssertAtlasSSR(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseRoute string, parseTitleContains string) {
	parseT.Helper()
	parseResp, parseErr := parsePage.Goto(parseBaseURL+parseRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto %s: %v", parseRoute, parseErr)
	}
	if parseResp == nil {
		parseT.Fatalf("nil response for route %s", parseRoute)
	}
	if parseResp.Status() >= 400 {
		parseT.Fatalf("route %s returned status %d", parseRoute, parseResp.Status())
	}
	parseTitle, parseErr := parsePage.Title()
	if parseErr != nil {
		parseT.Fatalf("read title for %s: %v", parseRoute, parseErr)
	}
	if !strings.Contains(parseTitle, parseTitleContains) {
		parseT.Fatalf("route %s expected title containing %q, got %q", parseRoute, parseTitleContains, parseTitle)
	}
	parseHTML, parseErr := parsePage.Content()
	if parseErr != nil {
		parseT.Fatalf("read content for %s: %v", parseRoute, parseErr)
	}
	if !strings.Contains(parseHTML, "__ATLAS_BOOTSTRAP__") {
		parseT.Fatalf("route %s expected SSR bootstrap script marker", parseRoute)
	}
}

func TestCatalog(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18090")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, "/examples")
		parseBodyText, parseErr := parsePage.TextContent("body")
		if parseErr != nil {
			parseT.Fatalf("read catalog body text: %v", parseErr)
		}
		if !strings.Contains(strings.ToLower(parseBodyText), "examples") {
			parseT.Fatalf("catalog page does not include expected text")
		}
	})
}

func TestLinks(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18091")
	parseRoutes := discoverPublicExampleRoutes(parseT, parseRepoRoot, []string{
		"counter", "text-input", "toggle", "form", "fetch",
		"hash-router", "portals", "todo-basic", "use-state", "web-components",
	})

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		for _, parseRoute := range parseRoutes {
			visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoute)
		}
	})
}

func TestSSRServerRouting(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18092")
	parseRoutes := discoverPublicExampleRoutes(parseT, parseRepoRoot, []string{"static-server-side-rendering-routing"})
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoutes[0])
	})
}

func TestAtlasSSR(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startAtlasExamplesServer(parseT, parseRepoRoot, "18093")
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertAtlasSSR(parseT, parsePage, parseBaseURL, "/shop", "Atlas Shop")
	})
}

func TestStartup(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18094")
	parseRoutes := discoverPublicExampleRoutes(parseT, parseRepoRoot, []string{"counter", "hydration"})
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		for _, parseRoute := range parseRoutes {
			visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoute)
		}
	})
}

func TestVirtualization(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18095")
	parseRoutes := discoverPublicExampleRoutes(parseT, parseRepoRoot, []string{"virtualized-feed"})
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoutes[0])
	})
}

func TestAtlasStartup(parseT *testing.T) {
	TestAtlasSSR(parseT)
}

func TestAtlasCrossBrowserSmoke(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startAtlasExamplesServer(parseT, parseRepoRoot, "18098")
	parseBrowsers := []string{"chromium", "firefox", "webkit"}
	parsePublicRoutes := []struct {
		route string
		title string
	}{
		{route: "/shop", title: "Atlas Shop"},
		{route: "/shop/frame-desk", title: "Atlas"},
		{route: "/warehouses", title: "Atlas Delivery Regions"},
	}
	parseInternalRoutes := []struct {
		route string
		title string
	}{
		{route: "/app/dashboard", title: "Atlas Ops Dashboard"},
		{route: "/app/inventory", title: "Atlas Inventory"},
	}

	for _, parseBrowser := range parseBrowsers {
		parseBrowser := parseBrowser
		parseT.Run(parseBrowser, func(parseT *testing.T) {
			withExamplesBrowserPage(parseT, parseBrowser, func(parsePage playwright.Page) {
				for _, parseRoute := range parsePublicRoutes {
					visitRouteAndAssertAtlasSSR(parseT, parsePage, parseBaseURL, parseRoute.route, parseRoute.title)
				}
				parseCookieErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{
					{Name: "atlas_mock_role", Value: "inventory_manager", URL: playwright.String(parseBaseURL)},
				})
				if parseCookieErr != nil {
					parseT.Fatalf("add atlas mock cookie: %v", parseCookieErr)
				}
				for _, parseRoute := range parseInternalRoutes {
					visitRouteAndAssertAtlasSSR(parseT, parsePage, parseBaseURL, parseRoute.route, parseRoute.title)
				}
			})
		})
	}
}

func TestBrowserCompat(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18096")
	parseRoutes := discoverPublicExampleRoutes(parseT, parseRepoRoot, nil)
	if len(parseRoutes) == 0 {
		parseT.Fatal("no browser-compat routes discovered")
	}
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		for _, parseRoute := range parseRoutes {
			visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoute)
		}
	})
}

func TestChatWizard(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18097")
	parseChatExampleDir := filepath.Join(parseRepoRoot, "examples", "server", "ai-chat-wizard")
	if _, parseErr := os.Stat(parseChatExampleDir); parseErr != nil {
		parseT.Fatalf("chat wizard example directory missing: %v", parseErr)
	}

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, "/examples")
	})
}

func TestExamplesAll(parseT *testing.T) {
	TestCatalog(parseT)
	TestLinks(parseT)
	TestSSRServerRouting(parseT)
	TestAtlasSSR(parseT)
	TestAtlasCrossBrowserSmoke(parseT)
	TestStartup(parseT)
	TestVirtualization(parseT)
	TestAtlasStartup(parseT)
	TestBrowserCompat(parseT)
	TestChatWizard(parseT)
}
