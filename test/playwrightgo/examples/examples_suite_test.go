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
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

var installExamplesChromiumOnce sync.Once
var installExamplesChromiumErr error

func ensureExamplesChromiumInstalled() error {
	installExamplesChromiumOnce.Do(func() {
		installExamplesChromiumErr = playwright.Install(&playwright.RunOptions{
			Browsers: []string{"chromium"},
			Verbose:  false,
		})
	})
	return installExamplesChromiumErr
}

func examplesRepoRootFromFile(testFile string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", "..", ".."))
}

func startExamplesCommand(t *testing.T, dir string, name string, args ...string) (stop func()) {
	t.Helper()
	cmd := exec.Command(name, args...)
	cmd.Dir = dir
	cmd.Stdout = io.Discard
	cmd.Stderr = io.Discard
	if err := cmd.Start(); err != nil {
		t.Fatalf("start %s %v: %v", name, args, err)
	}
	done := make(chan error, 1)
	go func() {
		done <- cmd.Wait()
	}()
	return func() {
		if cmd.Process != nil {
			_ = cmd.Process.Kill()
		}
		select {
		case <-done:
		case <-time.After(5 * time.Second):
		}
	}
}

func waitForHealthyExamplesURL(t *testing.T, healthURL string, timeout time.Duration) {
	t.Helper()
	deadline := time.Now().Add(timeout)
	for time.Now().Before(deadline) {
		resp, err := http.Get(healthURL)
		if err == nil {
			_ = resp.Body.Close()
			if resp.StatusCode >= 200 && resp.StatusCode < 500 {
				return
			}
		}
		time.Sleep(200 * time.Millisecond)
	}
	t.Fatalf("health check timed out: %s", healthURL)
}

func startExamplesCatalogServer(t *testing.T, repoRoot string, port string) string {
	t.Helper()
	stop := startExamplesCommand(
		t,
		repoRoot,
		"go",
		"run", "./tools/gwc", "examples",
		"-host", "127.0.0.1",
		"-port", port,
	)
	t.Cleanup(stop)
	baseURL := "http://127.0.0.1:" + port
	waitForHealthyExamplesURL(t, baseURL+"/healthz", 30*time.Second)
	return baseURL
}

func withExamplesPage(t *testing.T, fn func(page playwright.Page)) {
	t.Helper()
	if err := ensureExamplesChromiumInstalled(); err != nil {
		t.Fatalf("install playwright chromium: %v", err)
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
		t.Fatalf("new page: %v", err)
	}
	fn(page)
}

func discoverExampleRoutes(t *testing.T, repoRoot string, prefixes []string) []string {
	t.Helper()
	examplesRoot := filepath.Join(repoRoot, "examples")
	entries, err := os.ReadDir(examplesRoot)
	if err != nil {
		t.Fatalf("read examples dir: %v", err)
	}

	prefixSet := map[string]struct{}{}
	for _, p := range prefixes {
		prefixSet[p] = struct{}{}
	}

	var exampleDirs []string
	for _, entry := range entries {
		if !entry.IsDir() {
			continue
		}
		name := entry.Name()
		for prefix := range prefixSet {
			if strings.HasPrefix(name, prefix+"-") {
				exampleDirs = append(exampleDirs, filepath.Join(examplesRoot, name))
				break
			}
		}
	}
	sort.Strings(exampleDirs)

	var routes []string
	for _, dir := range exampleDirs {
		indexPath := filepath.Join(dir, "index.html")
		if _, err := os.Stat(indexPath); err == nil {
			rel, relErr := filepath.Rel(examplesRoot, indexPath)
			if relErr == nil {
				routes = append(routes, "/examples/"+filepath.ToSlash(rel))
			}
			continue
		}

		var htmlFiles []string
		_ = filepath.WalkDir(dir, func(path string, d os.DirEntry, walkErr error) error {
			if walkErr != nil {
				return nil
			}
			if d.IsDir() {
				return nil
			}
			if strings.EqualFold(filepath.Ext(path), ".html") {
				htmlFiles = append(htmlFiles, path)
			}
			return nil
		})
		sort.Strings(htmlFiles)
		if len(htmlFiles) == 0 {
			continue
		}
		rel, relErr := filepath.Rel(examplesRoot, htmlFiles[0])
		if relErr != nil {
			continue
		}
		routes = append(routes, "/examples/"+filepath.ToSlash(rel))
	}
	return routes
}

func visitRouteAndAssertSuccess(t *testing.T, page playwright.Page, baseURL string, route string) {
	t.Helper()
	resp, err := page.Goto(baseURL+route, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if err != nil {
		t.Fatalf("goto %s: %v", route, err)
	}
	if resp == nil {
		t.Fatalf("nil response for route %s", route)
	}
	if resp.Status() >= 400 {
		t.Fatalf("route %s returned status %d", route, resp.Status())
	}
}

func TestCatalog(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18090")

	withExamplesPage(t, func(page playwright.Page) {
		visitRouteAndAssertSuccess(t, page, baseURL, "/examples")
		bodyText, err := page.TextContent("body")
		if err != nil {
			t.Fatalf("read catalog body text: %v", err)
		}
		if !strings.Contains(strings.ToLower(bodyText), "examples") {
			t.Fatalf("catalog page does not include expected text")
		}
	})
}

func TestLinks(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18091")
	prefixes := []string{"00", "01", "02", "05", "06", "07", "08", "10", "12", "13"}
	routes := discoverExampleRoutes(t, repoRoot, prefixes)
	if len(routes) == 0 {
		t.Fatal("no example routes discovered for links smoke")
	}

	withExamplesPage(t, func(page playwright.Page) {
		for _, route := range routes {
			visitRouteAndAssertSuccess(t, page, baseURL, route)
		}
	})
}

func TestSSRServerRouting(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18092")
	routes := discoverExampleRoutes(t, repoRoot, []string{"18"})
	if len(routes) == 0 {
		t.Fatal("no route discovered for example 18")
	}
	withExamplesPage(t, func(page playwright.Page) {
		visitRouteAndAssertSuccess(t, page, baseURL, routes[0])
	})
}

func TestAtlasSSR(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18093")
	routes := discoverExampleRoutes(t, repoRoot, []string{"86"})
	if len(routes) == 0 {
		t.Fatal("no route discovered for example 86")
	}
	withExamplesPage(t, func(page playwright.Page) {
		visitRouteAndAssertSuccess(t, page, baseURL, routes[0])
	})
}

func TestStartup(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18094")
	routes := discoverExampleRoutes(t, repoRoot, []string{"21", "56"})
	if len(routes) == 0 {
		t.Fatal("no startup routes discovered")
	}
	withExamplesPage(t, func(page playwright.Page) {
		for _, route := range routes {
			visitRouteAndAssertSuccess(t, page, baseURL, route)
		}
	})
}

func TestVirtualization(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18095")
	routes := discoverExampleRoutes(t, repoRoot, []string{"103"})
	if len(routes) == 0 {
		t.Fatal("no route discovered for example 103")
	}
	withExamplesPage(t, func(page playwright.Page) {
		visitRouteAndAssertSuccess(t, page, baseURL, routes[0])
	})
}

func TestAtlasStartup(t *testing.T) {
	TestAtlasSSR(t)
}

func TestBrowserCompat(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18096")
	routes := discoverExampleRoutes(t, repoRoot, []string{"71", "73", "101"})
	if len(routes) == 0 {
		t.Fatal("no browser-compat routes discovered")
	}
	withExamplesPage(t, func(page playwright.Page) {
		for _, route := range routes {
			visitRouteAndAssertSuccess(t, page, baseURL, route)
		}
	})
}

func TestChatWizard(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := examplesRepoRootFromFile(file)
	baseURL := startExamplesCatalogServer(t, repoRoot, "18097")
	chatExampleDir := filepath.Join(repoRoot, "examples", "100-ai-chat-wizard")
	if _, err := os.Stat(chatExampleDir); err != nil {
		t.Fatalf("chat wizard example directory missing: %v", err)
	}
	routes := discoverExampleRoutes(t, repoRoot, []string{"100"})

	withExamplesPage(t, func(page playwright.Page) {
		visitRouteAndAssertSuccess(t, page, baseURL, "/examples")
		if len(routes) > 0 {
			visitRouteAndAssertSuccess(t, page, baseURL, routes[0])
		}
	})
}

func TestExamplesAll(t *testing.T) {
	TestCatalog(t)
	TestLinks(t)
	TestSSRServerRouting(t)
	TestAtlasSSR(t)
	TestStartup(t)
	TestVirtualization(t)
	TestAtlasStartup(t)
	TestBrowserCompat(t)
	TestChatWizard(t)
}
