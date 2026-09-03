//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/mxschmitt/playwright-go"
)

type example100SidebarThreadStateArtifact struct {
	GetPrompt                 string
	GetThreadPath             string
	HasConvListEntryAfterSend bool
	HasSidebarNavToThread     bool
	HasNewChatEmptyState      bool
	HasSidebarReturnToThread  bool
	HasReturnedPromptVisible  bool
	GetConsoleErrorCount      int
	GetPageErrorCount         int
	GetConsoleSampleText      string
	GetPageErrorSampleText    string
}

// formatExample100SidebarThreadStateSummary formats one sidebar/thread artifact for concise test logs.
func formatExample100SidebarThreadStateSummary(parseArtifact example100SidebarThreadStateArtifact) string {
	return fmt.Sprintf(
		"prompt=%q thread-path=%q conv-entry=%t sidebar-nav=%t new-chat-empty=%t sidebar-return=%t returned-prompt=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetPrompt,
		parseArtifact.GetThreadPath,
		parseArtifact.HasConvListEntryAfterSend,
		parseArtifact.HasSidebarNavToThread,
		parseArtifact.HasNewChatEmptyState,
		parseArtifact.HasSidebarReturnToThread,
		parseArtifact.HasReturnedPromptVisible,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100SidebarThreadStateArtifact exercises sidebar population, sidebar navigation, new-chat
// transition, and sidebar return-to-persisted-thread on one seeded server instance.
func captureExample100SidebarThreadStateArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100SidebarThreadStateArtifact {
	parseT.Helper()
	parseArtifact := example100SidebarThreadStateArtifact{
		GetPrompt: fmt.Sprintf("e2e-sidebar-%d", time.Now().UnixNano()%1_000_000),
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

	// ── Leg 1: login ─────────────────────────────────────────────────────────
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for email input: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", "customer@email.com"); parseErr != nil {
		parseT.Fatalf("fill email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", "password"); parseErr != nil {
		parseT.Fatalf("fill password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			body: ((document.body && document.body.innerText) || "").slice(0, 800),
		})`)
		parseT.Fatalf("wait for chat input after login: %v debug=%#v", parseErr, parseDebugValue)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("boot-shell")`, nil); parseErr != nil {
		parseT.Fatalf("wait for boot shell removal: %v", parseErr)
	}

	// ── Leg 2: send a message to create a persisted conversation ─────────────
	if parseErr := parsePage.Fill("#chat-input", parseArtifact.GetPrompt); parseErr != nil {
		parseT.Fatalf("fill chat input: %v", parseErr)
	}
	if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
		parseT.Fatalf("click send: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#streaming-assistant-bubble"); parseErr != nil {
		parseT.Fatalf("wait for streaming bubble: %v", parseErr)
	}
	// wait for stream completion and thread route
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("streaming-assistant-bubble")`, nil); parseErr != nil {
		parseT.Fatalf("wait for stream completion: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
		nil,
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => window.location.pathname`)
		parseT.Fatalf("wait for thread route after send: %v path=%v", parseErr, parseDebugValue)
	}
	parsePathValue, parseErr := parsePage.Evaluate(`() => window.location.pathname`)
	if parseErr != nil {
		parseT.Fatalf("read thread pathname: %v", parseErr)
	}
	parseArtifact.GetThreadPath = strings.TrimSpace(fmt.Sprintf("%v", parsePathValue))

	// ── Leg 3: verify sidebar conversation list has at least one entry ────────
	// The sidebar populates #conversation-list with [data-convrow] items.
	// After a successful send, the new conversation must appear in the list.
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.querySelectorAll("[data-convrow]").length >= 1`,
		nil,
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => document.getElementById("conversation-list") ? document.getElementById("conversation-list").innerHTML.slice(0,600) : "missing"`)
		parseT.Fatalf("wait for sidebar conv-row entry: %v list-html=%v", parseErr, parseDebugValue)
	}
	parseArtifact.HasConvListEntryAfterSend = true

	// ── Leg 4: click "New chat" to reset to root, then click the sidebar entry ─
	// First navigate to /app so the sidebar is fresh and not mid-thread.
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app for new-chat baseline: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input at /app: %v", parseErr)
	}
	// Sidebar retains the saved conversation; confirm the entry is still present.
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.querySelectorAll("[data-convrow]").length >= 1`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for sidebar entry after /app navigation: %v", parseErr)
	}
	// Click the first sidebar conv-row to navigate to the persisted thread.
	if parseErr := parsePage.Click("[data-convrow]:first-of-type"); parseErr != nil {
		parseT.Fatalf("click sidebar conv-row: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => window.location.pathname === %q`, parseArtifact.GetThreadPath),
		nil,
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => window.location.pathname`)
		parseT.Fatalf("wait for thread route after sidebar click: %v path=%v", parseErr, parseDebugValue)
	}
	parseArtifact.HasSidebarNavToThread = true

	// ── Leg 5: click "New chat" to transition to new-thread empty state ───────
	if parseErr := parsePage.Click(`button:has-text("New chat")`); parseErr != nil {
		parseT.Fatalf("click new chat button: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname === "/app"`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for root route after new chat: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#empty-state"); parseErr != nil {
		parseT.Fatalf("wait for empty state after new chat: %v", parseErr)
	}
	parseArtifact.HasNewChatEmptyState = true

	// ── Leg 6: return to the persisted thread via the sidebar ─────────────────
	// The persisted conv-row must still exist after clicking new chat.
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.querySelectorAll("[data-convrow]").length >= 1`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for sidebar entry after new-chat click: %v", parseErr)
	}
	if parseErr := parsePage.Click("[data-convrow]:first-of-type"); parseErr != nil {
		parseT.Fatalf("click sidebar conv-row to return: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => window.location.pathname === %q`, parseArtifact.GetThreadPath),
		nil,
	); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => window.location.pathname`)
		parseT.Fatalf("wait for thread route after sidebar return: %v path=%v", parseErr, parseDebugValue)
	}
	parseArtifact.HasSidebarReturnToThread = true
	// The original prompt must still be visible in the message list.
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parseArtifact.GetPrompt),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for returned prompt text: %v", parseErr)
	}
	parseArtifact.HasReturnedPromptVisible = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100SidebarThreadStateRegression verifies sidebar population, sidebar navigation, new-chat
// transition, and sidebar return-to-persisted-thread so the core chat shell does not regress under list churn.
func TestExample100SidebarThreadStateRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18128")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100SidebarThreadStateArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 sidebar/thread state: %s", formatExample100SidebarThreadStateSummary(parseArtifact))

		if !parseArtifact.HasConvListEntryAfterSend {
			parseT.Fatalf("sidebar conversation list empty after send: %s", formatExample100SidebarThreadStateSummary(parseArtifact))
		}
		if !parseArtifact.HasSidebarNavToThread {
			parseT.Fatalf("sidebar click did not navigate to thread: %s", formatExample100SidebarThreadStateSummary(parseArtifact))
		}
		if !parseArtifact.HasNewChatEmptyState {
			parseT.Fatalf("new-chat button did not produce empty state at /app: %s", formatExample100SidebarThreadStateSummary(parseArtifact))
		}
		if !parseArtifact.HasSidebarReturnToThread {
			parseT.Fatalf("sidebar did not return to persisted thread: %s", formatExample100SidebarThreadStateSummary(parseArtifact))
		}
		if !parseArtifact.HasReturnedPromptVisible {
			parseT.Fatalf("returned thread does not show original prompt: %s", formatExample100SidebarThreadStateSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount > 0 {
			parseT.Fatalf("page errors during sidebar/thread regression (%d): %s", parseArtifact.GetPageErrorCount, formatExample100SidebarThreadStateSummary(parseArtifact))
		}
	})
}
