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

	playwright "github.com/playwright-community/playwright-go"
)

type example100CanvasSurfaceArtifact struct {
	GetThreadPath                string
	HasChatInputBeforeCanvas     bool
	HasCanvasRouteGraceful       bool
	HasChatInputAfterCanvasRoute bool
	HasChatInputAfterReload      bool
	HasRootReturnFromCanvas      bool
	GetConsoleErrorCount         int
	GetPageErrorCount            int
	GetConsoleSampleText         string
	GetPageErrorSampleText       string
}

// formatExample100CanvasSurfaceSummary formats one canvas surface artifact for concise test logs.
func formatExample100CanvasSurfaceSummary(parseArtifact example100CanvasSurfaceArtifact) string {
	return fmt.Sprintf(
		"thread-path=%q chat-before=%t canvas-graceful=%t chat-after-canvas=%t chat-after-reload=%t root-return=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetThreadPath,
		parseArtifact.HasChatInputBeforeCanvas,
		parseArtifact.HasCanvasRouteGraceful,
		parseArtifact.HasChatInputAfterCanvasRoute,
		parseArtifact.HasChatInputAfterReload,
		parseArtifact.HasRootReturnFromCanvas,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// captureExample100CanvasSurfaceArtifact exercises the canvas route infrastructure: login,
// send a message to obtain a real thread route, navigate to the canvas URL sub-path for that
// thread, confirm the app shell does not crash and remains functional, reload from the canvas
// URL, and finally return to the root route — covering route preservation, graceful
// handling of an unknown canvas artifact ID, and absence of runtime errors.
func captureExample100CanvasSurfaceArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100CanvasSurfaceArtifact {
	parseT.Helper()
	parseArtifact := example100CanvasSurfaceArtifact{}
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
	parseArtifact.HasChatInputBeforeCanvas = true

	// ── Leg 2: send a message to create a real thread route ───────────────────
	parsePrompt := fmt.Sprintf("e2e-canvas-%d", time.Now().UnixNano()%1_000_000)
	if parseErr := parsePage.Fill("#chat-input", parsePrompt); parseErr != nil {
		parseT.Fatalf("fill chat input: %v", parseErr)
	}
	if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
		parseT.Fatalf("click send: %v", parseErr)
	}
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
	if !strings.HasPrefix(parseArtifact.GetThreadPath, "/app/thread/") {
		parseT.Fatalf("thread pathname = %q, want /app/thread/:publicID", parseArtifact.GetThreadPath)
	}

	// ── Leg 3: navigate to the canvas URL sub-path for this thread ────────────
	// Use a stub canvas ID to exercise the canvas route infrastructure without a
	// real artifact — verifies the route registers, the app does not crash, and
	// the shell normalizes gracefully (redirecting back to the thread route or
	// keeping the canvas URL until re-render).
	parseCanvasURL := parseBaseURL + parseArtifact.GetThreadPath + "/canvas/stub-canvas-id"
	if _, parseErr := parsePage.Goto(parseCanvasURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto canvas URL: %v", parseErr)
	}
	// The app must not crash: #chat-input or the thread screen must still be accessible.
	// The normalization effect redirects to the thread route when the canvas session is
	// not active (no real artifact found), so we wait for the chat shell to be ready.
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname,
			body: ((document.body && document.body.innerText) || "").slice(0, 800),
		})`)
		parseT.Fatalf("wait for chat input after canvas route: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasCanvasRouteGraceful = true
	parseArtifact.HasChatInputAfterCanvasRoute = true

	// ── Leg 4: reload from the current URL and confirm the shell survives ─────
	if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("reload from canvas/thread route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname,
			body: ((document.body && document.body.innerText) || "").slice(0, 800),
		})`)
		parseT.Fatalf("wait for chat input after reload: %v debug=%#v", parseErr, parseDebugValue)
	}
	parseArtifact.HasChatInputAfterReload = true

	// ── Leg 5: return to root from the thread/canvas URL ─────────────────────
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app to return to root: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after root return: %v", parseErr)
	}
	parseArtifact.HasRootReturnFromCanvas = true

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100CanvasSurfaceRegression verifies the canvas route infrastructure does not crash the
// app shell, the chat surface remains reachable after visiting a canvas URL, and a reload from the
// canvas/thread URL does not emit runtime errors — covering route preservation and graceful handling
// of the current canvas surface even before the larger canvas redesign lands.
func TestExample100CanvasSurfaceRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100HappyPathServer(parseT, parseRepoRoot, "18129")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100CanvasSurfaceArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 canvas surface: %s", formatExample100CanvasSurfaceSummary(parseArtifact))

		if !parseArtifact.HasChatInputBeforeCanvas {
			parseT.Fatalf("chat input not ready before canvas test: %s", formatExample100CanvasSurfaceSummary(parseArtifact))
		}
		if !parseArtifact.HasCanvasRouteGraceful {
			parseT.Fatalf("canvas route caused app crash: %s", formatExample100CanvasSurfaceSummary(parseArtifact))
		}
		if !parseArtifact.HasChatInputAfterCanvasRoute {
			parseT.Fatalf("chat input not reachable after canvas route: %s", formatExample100CanvasSurfaceSummary(parseArtifact))
		}
		if !parseArtifact.HasChatInputAfterReload {
			parseT.Fatalf("chat input not reachable after reload from canvas URL: %s", formatExample100CanvasSurfaceSummary(parseArtifact))
		}
		if !parseArtifact.HasRootReturnFromCanvas {
			parseT.Fatalf("root return from canvas URL failed: %s", formatExample100CanvasSurfaceSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount > 0 {
			parseT.Fatalf("page errors during canvas surface regression (%d): %s", parseArtifact.GetPageErrorCount, formatExample100CanvasSurfaceSummary(parseArtifact))
		}
	})
}
