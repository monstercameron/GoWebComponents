//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
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

type example100HappyPathArtifact struct {
	GetPrompt                 string
	GetThreadPath             string
	HasThreadRouteAfterSend   bool
	HasPromptAfterSend        bool
	HasPromptAfterReload      bool
	HasThreadRouteAfterReload bool
	HasThreadRouteAfterReopen bool
	HasPromptAfterReopen      bool
	GetConsoleErrorCount      int
	GetPageErrorCount         int
	GetConsoleSampleText      string
	GetPageErrorSampleText    string
}

// seedExample100HappyPathDatabase seeds one sqlite file with deterministic demo credentials and conversations.
func seedExample100HappyPathDatabase(parseT *testing.T, parseRepoRoot string, parseDBPath string) {
	parseT.Helper()
	parseCommand := exec.Command("go", "run", "./examples/100-ai-chat-wizard/cmd/seed-test-db")
	parseCommand.Dir = parseRepoRoot
	parseCommand.Env = append(os.Environ(), "CHAT_DB_PATH="+parseDBPath)
	if parseOutput, parseErr := parseCommand.CombinedOutput(); parseErr != nil {
		parseT.Fatalf("seed example 100 test db: %v\n%s", parseErr, strings.TrimSpace(string(parseOutput)))
	}
}

// buildExample100HappyPathServerBinary builds one local server executable for stable startup timing in browser tests.
func buildExample100HappyPathServerBinary(parseT *testing.T, parseRepoRoot string, parseBinaryPath string) {
	parseT.Helper()
	parseBuildCommand := exec.Command("go", "build", "-o", parseBinaryPath, "./examples/100-ai-chat-wizard/cmd/server")
	parseBuildCommand.Dir = parseRepoRoot
	if parseBuildOutput, parseBuildErr := parseBuildCommand.CombinedOutput(); parseBuildErr != nil {
		parseT.Fatalf("build example 100 server binary: %v\n%s", parseBuildErr, strings.TrimSpace(string(parseBuildOutput)))
	}
}

// startExample100HappyPathServer starts the example-100 server with one seeded test database.
func startExample100HappyPathServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "happy_path.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-happy-path-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)
	seedExample100HappyPathDatabase(parseT, parseRepoRoot, parseDBPath)
	buildExample100HappyPathServerBinary(parseT, parseRepoRoot, parseBinaryPath)

	parseStop := startExamplesCommandWithEnv(
		parseT,
		parseRepoRoot,
		[]string{
			"LISTEN_ADDR=" + parseAddress,
			"CHAT_DB_PATH=" + parseDBPath,
			"CHAT_LOG_DIR=" + parseLogDir,
		},
		parseBinaryPath,
	)
	parseT.Cleanup(parseStop)

	parseBaseURL := "http://" + parseAddress
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 120*time.Second)
	return parseBaseURL
}

// formatExample100HappyPathSummary formats one authenticated-flow artifact for concise test logs.
func formatExample100HappyPathSummary(parseArtifact example100HappyPathArtifact) string {
	return fmt.Sprintf(
		"prompt=%q thread-path=%q send-route=%t send-prompt=%t reload-route=%t reload-prompt=%t reopen-route=%t reopen-prompt=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetPrompt,
		parseArtifact.GetThreadPath,
		parseArtifact.HasThreadRouteAfterSend,
		parseArtifact.HasPromptAfterSend,
		parseArtifact.HasThreadRouteAfterReload,
		parseArtifact.HasPromptAfterReload,
		parseArtifact.HasThreadRouteAfterReopen,
		parseArtifact.HasPromptAfterReopen,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100AuthenticatedHappyPath executes login, send/stream, refresh, and reopen on one seeded server.
func captureExample100AuthenticatedHappyPath(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100HappyPathArtifact {
	parseT.Helper()
	parseArtifact := example100HappyPathArtifact{
		GetPrompt: fmt.Sprintf("e2e-happy-%d", time.Now().UnixNano()%1_000_000),
	}
	parseConsoleSamples := make([]string, 0, 120)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseMu.Lock()
		defer parseMu.Unlock()
		if len(parseConsoleSamples) < 120 {
			parseConsoleSamples = append(parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
		}
		if parseMessage.Type() == "error" {
			parseArtifact.GetConsoleErrorCount++
		}
	})
	parsePage.OnPageError(func(parseErr error) {
		parseMu.Lock()
		defer parseMu.Unlock()
		parseArtifact.GetPageErrorCount++
		if parseErr != nil && len(parsePageErrorSamples) < 12 {
			parsePageErrorSamples = append(parsePageErrorSamples, strings.TrimSpace(parseErr.Error()))
		}
	})

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app for login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for auth email input: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", "demo@example.com"); parseErr != nil {
		parseT.Fatalf("fill auth email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", "password123"); parseErr != nil {
		parseT.Fatalf("fill auth password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit auth form: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
		parseT.Fatalf("wait for boot shell removal after login: %v", parseErr)
	}

	if parseErr := parsePage.Click(`button:has-text("New chat")`); parseErr != nil {
		parseT.Fatalf("click new chat: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#chat-input", parseArtifact.GetPrompt); parseErr != nil {
		parseT.Fatalf("fill chat input: %v", parseErr)
	}
	if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
		parseT.Fatalf("click send: %v", parseErr)
	}

	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parseArtifact.GetPrompt),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for sent prompt text: %v", parseErr)
	}
	parseArtifact.HasPromptAfterSend = true
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.querySelectorAll("#message-list .rounded-br-md").length >= 1 && document.querySelectorAll("#message-list .rounded-bl-md").length >= 1`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for user+assistant message bubbles: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for thread route after send: %v", parseErr)
	}
	parseArtifact.HasThreadRouteAfterSend = true
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("streaming-assistant-bubble")`, nil); parseErr != nil {
		parseT.Fatalf("wait for stream completion: %v", parseErr)
	}
	parsePathValue, parseErr := parsePage.Evaluate(`() => window.location.pathname`)
	if parseErr != nil {
		parseT.Fatalf("read thread pathname after send: %v", parseErr)
	}
	parseArtifact.GetThreadPath = strings.TrimSpace(fmt.Sprintf("%v", parsePathValue))
	if !strings.HasPrefix(parseArtifact.GetThreadPath, "/app/thread/") {
		parseT.Fatalf("thread pathname after send = %q, want /app/thread/:publicID", parseArtifact.GetThreadPath)
	}

	if _, parseErr = parsePage.Reload(playwright.PageReloadOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("reload thread route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after reload: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => window.location.pathname === %q`, parseArtifact.GetThreadPath),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for same thread route after reload: %v", parseErr)
	}
	parseArtifact.HasThreadRouteAfterReload = true
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parseArtifact.GetPrompt),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for prompt after reload: %v", parseErr)
	}
	parseArtifact.HasPromptAfterReload = true

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app before reopen: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input before reopen: %v", parseErr)
	}
	parseConversationSelector := fmt.Sprintf(`#conversation-list button:has-text(%q)`, parseArtifact.GetPrompt)
	if _, parseErr := parsePage.WaitForSelector(parseConversationSelector); parseErr != nil {
		parseT.Fatalf("wait for seeded conversation row containing prompt: %v", parseErr)
	}
	if parseErr := parsePage.Click(parseConversationSelector); parseErr != nil {
		parseT.Fatalf("click conversation row for reopen: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => window.location.pathname === %q`, parseArtifact.GetThreadPath),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for reopened thread route: %v", parseErr)
	}
	parseArtifact.HasThreadRouteAfterReopen = true
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parseArtifact.GetPrompt),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for prompt after reopen: %v", parseErr)
	}
	parseArtifact.HasPromptAfterReopen = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100AuthenticatedHappyPath verifies login, new-thread send/stream, reload persistence, and manual thread reopen.
func TestExample100AuthenticatedHappyPath(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18104")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100AuthenticatedHappyPath(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 authenticated happy path: %s", formatExample100HappyPathSummary(parseArtifact))

		if !parseArtifact.HasThreadRouteAfterSend {
			parseT.Fatalf("missing thread route after send: %s", formatExample100HappyPathSummary(parseArtifact))
		}
		if !parseArtifact.HasPromptAfterSend {
			parseT.Fatalf("missing prompt visibility after send: %s", formatExample100HappyPathSummary(parseArtifact))
		}
		if !parseArtifact.HasThreadRouteAfterReload || !parseArtifact.HasPromptAfterReload {
			parseT.Fatalf("missing route/prompt persistence after reload: %s", formatExample100HappyPathSummary(parseArtifact))
		}
		if !parseArtifact.HasThreadRouteAfterReopen || !parseArtifact.HasPromptAfterReopen {
			parseT.Fatalf("missing route/prompt after manual reopen: %s", formatExample100HappyPathSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100HappyPathSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100HappyPathSummary(parseArtifact))
		}
	})
}
