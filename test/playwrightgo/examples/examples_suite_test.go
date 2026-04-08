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
	parseCmd.Stdout = io.Discard
	parseCmd.Stderr = io.Discard
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

func discoverExampleRoutes(parseT *testing.T, parseRepoRoot string, parsePrefixes []string) []string {
	parseT.Helper()
	parseExamplesRoot := filepath.Join(parseRepoRoot, "examples")
	parseEntries, parseErr := os.ReadDir(parseExamplesRoot)
	if parseErr != nil {
		parseT.Fatalf("read examples dir: %v", parseErr)
	}

	parsePrefixSet := map[string]struct{}{}
	for _, parseP := range parsePrefixes {
		parsePrefixSet[parseP] = struct{}{}
	}

	var parseExampleDirs []string
	for _, parseEntry := range parseEntries {
		if !parseEntry.IsDir() {
			continue
		}
		parseName := parseEntry.Name()
		for parsePrefix := range parsePrefixSet {
			if strings.HasPrefix(parseName, parsePrefix+"-") {
				parseExampleDirs = append(parseExampleDirs, filepath.Join(parseExamplesRoot, parseName))
				break
			}
		}
	}
	sort.Strings(parseExampleDirs)

	var parseRoutes []string
	for _, parseDir := range parseExampleDirs {
		parseIndexPath := filepath.Join(parseDir, "index.html")
		if _, parseErr2 := os.Stat(parseIndexPath); parseErr2 == nil {
			parseRel, parseRelErr := filepath.Rel(parseExamplesRoot, parseIndexPath)
			if parseRelErr == nil {
				parseRoutes = append(parseRoutes, "/examples/"+filepath.ToSlash(parseRel))
			}
			continue
		}

		var parseHtmlFiles []string
		_ = filepath.WalkDir(parseDir, func(parsePath string, parseD os.DirEntry, parseWalkErr error) error {
			if parseWalkErr != nil {
				return nil
			}
			if parseD.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(parsePath), ".html") {
				parseHtmlFiles = append(parseHtmlFiles, parsePath)
			}
			return nil
		})
		sort.Strings(parseHtmlFiles)
		if len(parseHtmlFiles) == 0 {
			continue
		}
		parseRel2, parseRelErr2 := filepath.Rel(parseExamplesRoot, parseHtmlFiles[0])
		if parseRelErr2 != nil {
			continue
		}
		parseRoutes = append(parseRoutes, "/examples/"+filepath.ToSlash(parseRel2))
	}
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
	parsePrefixes := []string{"00", "01", "02", "05", "06", "07", "08", "10", "12", "13"}
	parseRoutes := discoverExampleRoutes(parseT, parseRepoRoot, parsePrefixes)
	if len(parseRoutes) == 0 {
		parseT.Fatal("no example routes discovered for links smoke")
	}

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
	parseRoutes := discoverExampleRoutes(parseT, parseRepoRoot, []string{"18"})
	if len(parseRoutes) == 0 {
		parseT.Fatal("no route discovered for example 18")
	}
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoutes[0])
	})
}

func TestAtlasSSR(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18093")
	parseRoutes := discoverExampleRoutes(parseT, parseRepoRoot, []string{"86"})
	if len(parseRoutes) == 0 {
		parseT.Fatal("no route discovered for example 86")
	}
	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoutes[0])
	})
}

func TestStartup(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExamplesCatalogServer(parseT, parseRepoRoot, "18094")
	parseRoutes := discoverExampleRoutes(parseT, parseRepoRoot, []string{"21", "56"})
	if len(parseRoutes) == 0 {
		parseT.Fatal("no startup routes discovered")
	}
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
	parseRoutes := discoverExampleRoutes(parseT, parseRepoRoot, []string{"103"})
	if len(parseRoutes) == 0 {
		parseT.Fatal("no route discovered for example 103")
	}
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
	parseRoutes := discoverExampleRoutes(parseT, parseRepoRoot, []string{"71", "73", "101"})
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
	parseChatExampleDir := filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard")
	if _, parseErr := os.Stat(parseChatExampleDir); parseErr != nil {
		parseT.Fatalf("chat wizard example directory missing: %v", parseErr)
	}
	parseRoutes := discoverExampleRoutes(parseT, parseRepoRoot, []string{"100"})

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, "/examples")
		if len(parseRoutes) > 0 {
			visitRouteAndAssertSuccess(parseT, parsePage, parseBaseURL, parseRoutes[0])
		}
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
