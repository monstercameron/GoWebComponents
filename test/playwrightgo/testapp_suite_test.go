//go:build playwrightgo
// +build playwrightgo

package playwrightgo_test

import (
	"io"
	"net/http"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

var installChromiumOnce sync.Once
var installChromiumErr error

func ensureChromiumInstalled() error {
	installChromiumOnce.Do(func() {
		installChromiumErr = playwright.Install(&playwright.RunOptions{
			Browsers: []string{"chromium"},
			Verbose:  false,
		})
	})
	return installChromiumErr
}

func repoRootFromFile(testFile string) string {
	return filepath.Clean(filepath.Join(filepath.Dir(testFile), "..", ".."))
}

func startCommand(t *testing.T, dir string, name string, args ...string) (stop func()) {
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

func waitForHealthyURL(t *testing.T, healthURL string, timeout time.Duration) {
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

func buildTestAppWasm(t *testing.T, repoRoot string) string {
	t.Helper()
	outputDir := filepath.Join(repoRoot, "bin", "test", "testapp")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir output dir: %v", err)
	}

	outputWasm := filepath.Join(outputDir, "main.wasm")
	cmd := exec.Command("go", "build", "-o", outputWasm, ".")
	cmd.Dir = filepath.Join(repoRoot, "test", "testapp")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build testapp wasm: %v\n%s", err, string(out))
	}
	return outputWasm
}

func runChromiumPage(t *testing.T, fn func(page playwright.Page)) {
	t.Helper()
	if err := ensureChromiumInstalled(); err != nil {
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

func startTestAppServer(t *testing.T, repoRoot string, port string) string {
	t.Helper()
	wasmPath := buildTestAppWasm(t, repoRoot)
	fixturePath := filepath.Join(repoRoot, "test", "fixtures", "user-123.json")
	stop := startCommand(
		t,
		repoRoot,
		"go",
		"run", "./tools/gwc", "serve",
		"-root", "./test/testapp",
		"-host", "127.0.0.1",
		"-port", port,
		"-wasm-route", "/main.wasm",
		"-wasm-file", wasmPath,
		"-fixture-json", "/api/user/123="+fixturePath,
	)
	t.Cleanup(stop)

	baseURL := "http://127.0.0.1:" + port
	waitForHealthyURL(t, baseURL+"/healthz", 30*time.Second)
	return baseURL
}

func TestComponents(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := repoRootFromFile(file)
	baseURL := startTestAppServer(t, repoRoot, "18083")

	runChromiumPage(t, func(page playwright.Page) {
		if _, err := page.Goto(baseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			t.Fatalf("goto test app: %v", err)
		}
		if _, err := page.WaitForSelector("#main-heading"); err != nil {
			t.Fatalf("wait for main heading: %v", err)
		}
		if _, err := page.WaitForSelector("button[aria-label=\"Increment main\"]"); err != nil {
			t.Fatalf("wait for increment button: %v", err)
		}
		heading, err := page.TextContent("#main-heading")
		if err != nil {
			t.Fatalf("read heading: %v", err)
		}
		if !strings.Contains(heading, "GoWebComponents Test") {
			t.Fatalf("unexpected heading: %q", heading)
		}

		if err := page.Click("button[aria-label=\"Increment main\"]"); err != nil {
			t.Fatalf("click increment main: %v", err)
		}
		if _, err := page.WaitForFunction(
			"() => document.querySelector('[data-testid=\"count-display\"]')?.textContent?.includes('Count: 1')",
			nil,
		); err != nil {
			t.Fatalf("wait for count display update: %v", err)
		}
		countDisplay, err := page.TextContent("[data-testid=\"count-display\"]")
		if err != nil {
			t.Fatalf("read count display: %v", err)
		}
		if !strings.Contains(countDisplay, "Count: 1") {
			t.Fatalf("unexpected count display: %q", countDisplay)
		}
	})
}

func TestIntegration(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := repoRootFromFile(file)
	baseURL := startTestAppServer(t, repoRoot, "18084")

	runChromiumPage(t, func(page playwright.Page) {
		if _, err := page.Goto(baseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			t.Fatalf("goto test app: %v", err)
		}
		if _, err := page.WaitForSelector("#todo-input"); err != nil {
			t.Fatalf("wait for todo input: %v", err)
		}

		if err := page.Fill("#todo-input", "go-playwright todo"); err != nil {
			t.Fatalf("fill todo input: %v", err)
		}
		if err := page.Click("#todo-add"); err != nil {
			t.Fatalf("click todo add: %v", err)
		}

		todoText, err := page.TextContent("#todo-text-1")
		if err != nil {
			t.Fatalf("read first todo: %v", err)
		}
		if !strings.Contains(todoText, "go-playwright todo") {
			t.Fatalf("unexpected todo text: %q", todoText)
		}

		if err := page.Click("button:has-text(\"Toggle Theme\")"); err != nil {
			t.Fatalf("click toggle theme: %v", err)
		}
		theme, err := page.GetAttribute("body", "data-theme")
		if err != nil {
			t.Fatalf("read body theme attr: %v", err)
		}
		if theme == "" {
			t.Fatalf("expected non-empty body theme attr after toggle")
		}
	})
}

func TestState(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := repoRootFromFile(file)
	baseURL := startTestAppServer(t, repoRoot, "18085")

	runChromiumPage(t, func(page playwright.Page) {
		if _, err := page.Goto(baseURL, playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			t.Fatalf("goto test app: %v", err)
		}
		if _, err := page.WaitForSelector("#stress-plus-5"); err != nil {
			t.Fatalf("wait for stress controls: %v", err)
		}
		if err := page.Click("#stress-plus-5"); err != nil {
			t.Fatalf("click stress plus 5: %v", err)
		}
		if err := page.Click("#stress-plus-25"); err != nil {
			t.Fatalf("click stress plus 25: %v", err)
		}

		stressCount, err := page.TextContent("#stress-count")
		if err != nil {
			t.Fatalf("read stress count: %v", err)
		}
		if !strings.Contains(stressCount, "Stress Count: 5") && !strings.Contains(stressCount, "Stress Count: 30") {
			t.Fatalf("unexpected stress count: %q", stressCount)
		}
	})
}

func buildBenchmarkWasm(t *testing.T, repoRoot string) string {
	t.Helper()
	outputDir := filepath.Join(repoRoot, "bin", "test", "benchmark")
	if err := os.MkdirAll(outputDir, 0o755); err != nil {
		t.Fatalf("mkdir benchmark output dir: %v", err)
	}

	outputWasm := filepath.Join(outputDir, "benchmark.wasm")
	cmd := exec.Command("go", "build", "-o", outputWasm, ".")
	cmd.Dir = filepath.Join(repoRoot, "test", "benchmark")
	cmd.Env = append(os.Environ(), "GOOS=js", "GOARCH=wasm")
	if out, err := cmd.CombinedOutput(); err != nil {
		t.Fatalf("build benchmark wasm: %v\n%s", err, string(out))
	}
	return outputWasm
}

func TestBenchmark(t *testing.T) {
	_, file, _, _ := runtime.Caller(0)
	repoRoot := repoRootFromFile(file)
	wasmPath := buildBenchmarkWasm(t, repoRoot)
	stop := startCommand(
		t,
		repoRoot,
		"go",
		"run", "./tools/gwc", "serve",
		"-root", "./test/benchmark",
		"-host", "127.0.0.1",
		"-port", "18082",
		"-wasm-route", "/bin/benchmark.wasm",
		"-wasm-file", wasmPath,
	)
	t.Cleanup(stop)
	baseURL := "http://127.0.0.1:18082"
	waitForHealthyURL(t, baseURL+"/healthz", 30*time.Second)

	runChromiumPage(t, func(page playwright.Page) {
		if _, err := page.Goto(baseURL+"/benchmark.html", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); err != nil {
			t.Fatalf("goto benchmark page: %v", err)
		}
		if _, err := page.WaitForSelector("#btn-render"); err != nil {
			t.Fatalf("wait for render button: %v", err)
		}
		if err := page.Click("#btn-render"); err != nil {
			t.Fatalf("click render button: %v", err)
		}
		if _, err := page.WaitForSelector(".core-list-item"); err != nil {
			t.Fatalf("wait for core list item: %v", err)
		}
		coreCount, err := page.TextContent("#core-count")
		if err != nil {
			t.Fatalf("read core count: %v", err)
		}
		if !strings.Contains(coreCount, "Core Count: 40") {
			t.Fatalf("unexpected core count text: %q", coreCount)
		}
	})
}

func TestMainSuite(t *testing.T) {
	for _, name := range []string{"components", "integration", "state"} {
		name := name
		t.Run(name, func(t *testing.T) {
			switch name {
			case "components":
				TestComponents(t)
			case "integration":
				TestIntegration(t)
			case "state":
				TestState(t)
			default:
				t.Fatalf("unknown suite: %s", name)
			}
		})
	}
}
