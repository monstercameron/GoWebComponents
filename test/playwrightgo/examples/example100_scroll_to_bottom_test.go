//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"runtime"
	"strings"
	"sync"
	"testing"

	playwright "github.com/playwright-community/playwright-go"
)

type example100ScrollToBottomArtifact struct {
	HasOverflowAfterScrollTopReset bool
	HasScrollToBottomButton        bool
	HasReachedScrollBottom         bool
	GetScrollDelta                 float64
	GetScrollHeight                float64
	GetClientHeight                float64
	GetScrollTop                   float64
	GetConsoleErrorCount           int
	GetPageErrorCount              int
	GetConsoleSampleText           string
	GetPageErrorSampleText         string
}

// buildExample100ScrollStressPrompt builds one deterministic multi-line prompt that forces vertical overflow.
func buildExample100ScrollStressPrompt() string {
	var parseBuilder strings.Builder
	parseBuilder.WriteString("scroll reliability probe\n")
	for parseLine := 1; parseLine <= 160; parseLine++ {
		parseBuilder.WriteString(fmt.Sprintf("line %03d: verify scroll-to-bottom behavior under long thread content.\n", parseLine))
	}
	return parseBuilder.String()
}

// formatExample100ScrollToBottomSummary formats one scroll artifact for concise test logs.
func formatExample100ScrollToBottomSummary(parseArtifact example100ScrollToBottomArtifact) string {
	return fmt.Sprintf(
		"overflow=%t button=%t reached-bottom=%t delta=%.2f scroll-height=%.2f client-height=%.2f scroll-top=%.2f console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.HasOverflowAfterScrollTopReset,
		parseArtifact.HasScrollToBottomButton,
		parseArtifact.HasReachedScrollBottom,
		parseArtifact.GetScrollDelta,
		parseArtifact.GetScrollHeight,
		parseArtifact.GetClientHeight,
		parseArtifact.GetScrollTop,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100ScrollToBottomArtifact executes login, creates one long-message thread, and verifies jump-to-bottom behavior.
func captureExample100ScrollToBottomArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100ScrollToBottomArtifact {
	parseT.Helper()
	parseArtifact := example100ScrollToBottomArtifact{}
	parseConsoleSamples := make([]string, 0, 80)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		parseMu.Lock()
		defer parseMu.Unlock()
		if parseMessage.Type() == "error" {
			parseArtifact.GetConsoleErrorCount++
		}
		if parseText == "" {
			return
		}
		if len(parseConsoleSamples) < 80 {
			parseConsoleSamples = append(parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
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

	if parseErr := parsePage.SetViewportSize(1280, 480); parseErr != nil {
		parseT.Fatalf("set viewport for scroll overflow: %v", parseErr)
	}

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

	if parseErr := parsePage.Click(`button:has-text("New chat")`); parseErr != nil {
		parseT.Fatalf("click new chat: %v", parseErr)
	}
	parseForceTopAndMeasureOverflow := func() {
		parseOverflowValue, parseErr := parsePage.Evaluate(`() => {
		const list = document.getElementById("message-list");
		if (!list) return { overflow: false, scrollTop: 0, scrollHeight: 0, clientHeight: 0 };
		list.scrollTop = 0;
		list.dispatchEvent(new Event("scroll", { bubbles: true }));
		return {
			overflow: (list.scrollHeight - list.clientHeight) > 20,
			scrollTop: list.scrollTop,
			scrollHeight: list.scrollHeight,
			clientHeight: list.clientHeight,
		};
	}`)
		if parseErr != nil {
			parseT.Fatalf("force message-list top scroll: %v", parseErr)
		}
		parseMetrics, parseOk := parseOverflowValue.(map[string]interface{})
		if !parseOk {
			parseT.Fatalf("unexpected overflow metrics payload: %T", parseOverflowValue)
		}
		parseArtifact.HasOverflowAfterScrollTopReset = parseMetrics["overflow"] == true
		if parseValue, parseOk2 := parseMetrics["scrollTop"].(float64); parseOk2 {
			parseArtifact.GetScrollTop = parseValue
		}
		if parseValue2, parseOk3 := parseMetrics["scrollHeight"].(float64); parseOk3 {
			parseArtifact.GetScrollHeight = parseValue2
		}
		if parseValue3, parseOk4 := parseMetrics["clientHeight"].(float64); parseOk4 {
			parseArtifact.GetClientHeight = parseValue3
		}
	}

	parsePromptBase := "scroll reliability probe"
	parsePrompt := buildExample100ScrollStressPrompt()
	for parseAttempt := 1; parseAttempt <= 4; parseAttempt++ {
		parseAttemptPrompt := parsePrompt
		if parseAttempt > 1 {
			parseAttemptPrompt = fmt.Sprintf("%s attempt-%d\n%s", parsePromptBase, parseAttempt, parsePrompt)
		}
		if parseErr := parsePage.Fill("#chat-input", parseAttemptPrompt); parseErr != nil {
			parseT.Fatalf("fill long chat input attempt %d: %v", parseAttempt, parseErr)
		}
		if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
			parseT.Fatalf("click send attempt %d: %v", parseAttempt, parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(
			`(() => window.location.pathname.startsWith('/app/thread/'))()`,
			nil,
		); parseErr != nil {
			parseT.Fatalf("wait for thread route after send attempt %d: %v", parseAttempt, parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(
			fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parsePromptBase),
			nil,
		); parseErr != nil {
			parseT.Fatalf("wait for prompt visibility attempt %d: %v", parseAttempt, parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("streaming-assistant-bubble")`, nil); parseErr != nil {
			parseT.Fatalf("wait for stream completion attempt %d: %v", parseAttempt, parseErr)
		}
		parseForceTopAndMeasureOverflow()
		if parseArtifact.HasOverflowAfterScrollTopReset {
			break
		}
	}

	if !parseArtifact.HasOverflowAfterScrollTopReset {
		parseT.Fatalf("message list did not expose enough overflow after reset: %s", formatExample100ScrollToBottomSummary(parseArtifact))
	}

	if _, parseErr := parsePage.WaitForSelector("#scroll-to-bottom-btn"); parseErr != nil {
		parseT.Fatalf("wait for scroll-to-bottom button: %v", parseErr)
	}
	parseArtifact.HasScrollToBottomButton = true
	if parseErr := parsePage.Click("#scroll-to-bottom-btn"); parseErr != nil {
		parseT.Fatalf("click scroll-to-bottom button: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => {
		const list = document.getElementById("message-list");
		if (!list) return false;
		const delta = (list.scrollHeight - list.clientHeight) - list.scrollTop;
		return Math.abs(delta) <= 2;
	}`, nil); parseErr != nil {
		parseT.Fatalf("wait for message-list bottom alignment: %v", parseErr)
	}
	parseArtifact.HasReachedScrollBottom = true

	parseDeltaValue, parseErr := parsePage.Evaluate(`() => {
		const list = document.getElementById("message-list");
		if (!list) return -1;
		return (list.scrollHeight - list.clientHeight) - list.scrollTop;
	}`)
	if parseErr == nil {
		switch parseDelta := parseDeltaValue.(type) {
		case float64:
			parseArtifact.GetScrollDelta = parseDelta
		case int:
			parseArtifact.GetScrollDelta = float64(parseDelta)
		}
	}

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100ScrollToBottomButton verifies the jump-to-bottom action lands at the true bottom of the active thread.
func TestExample100ScrollToBottomButton(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18105")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100ScrollToBottomArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 scroll-to-bottom: %s", formatExample100ScrollToBottomSummary(parseArtifact))

		if !parseArtifact.HasOverflowAfterScrollTopReset {
			parseT.Fatalf("expected scroll overflow in active thread: %s", formatExample100ScrollToBottomSummary(parseArtifact))
		}
		if !parseArtifact.HasScrollToBottomButton {
			parseT.Fatalf("expected scroll-to-bottom button visibility: %s", formatExample100ScrollToBottomSummary(parseArtifact))
		}
		if !parseArtifact.HasReachedScrollBottom {
			parseT.Fatalf("expected bottom alignment after jump-to-bottom click: %s", formatExample100ScrollToBottomSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100ScrollToBottomSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100ScrollToBottomSummary(parseArtifact))
		}
	})
}
