//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	playwright "github.com/playwright-community/playwright-go"
)

type example100StartupBootArtifact struct {
	GetRoute                   string
	GetHTTPStatus              int
	GetPageTitle               string
	GetChatWasmResponseCount   int
	GetWorkerWasmResponseCount int
	GetConsoleErrorCount       int
	GetPageErrorCount          int
	HasGRPCReadyLog            bool
	HasWorkerReadyLog          bool
	HasWorkerPoolReadyLog      bool
	HasWorkerFallbackLog       bool
	GetConsoleSampleText       string
	GetPageErrorSampleText     string
}

// startExample100StartupServer starts the example-100 chat server on one dedicated test port.
func startExample100StartupServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "chat_startup_smoke.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)

	parseStop := startExamplesCommandWithEnv(
		parseT,
		parseRepoRoot,
		[]string{
			"LISTEN_ADDR=" + parseAddress,
			"CHAT_DB_PATH=" + parseDBPath,
			"CHAT_LOG_DIR=" + parseLogDir,
		},
		"go",
		"run", "./examples/server/ai-chat-wizard/cmd/server",
	)
	parseT.Cleanup(parseStop)

	parseBaseURL := "http://" + parseAddress
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 120*time.Second)
	return parseBaseURL
}

// captureExample100StartupBootArtifact captures startup signals for server health, WASM mount, and worker boot.
func captureExample100StartupBootArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100StartupBootArtifact {
	parseT.Helper()
	parseArtifact := example100StartupBootArtifact{
		GetRoute: "/",
	}
	parseConsoleSamples := make([]string, 0, 80)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseLowerText := strings.ToLower(parseText)
		parseMu.Lock()
		defer parseMu.Unlock()
		if len(parseConsoleSamples) < 80 {
			parseConsoleSamples = append(parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
		}
		if parseMessage.Type() == "error" {
			parseArtifact.GetConsoleErrorCount++
		}
		if strings.Contains(parseLowerText, "grpc ready") {
			parseArtifact.HasGRPCReadyLog = true
		}
		if strings.Contains(parseLowerText, "message: worker ready") {
			parseArtifact.HasWorkerReadyLog = true
		}
		if strings.Contains(parseLowerText, "background render worker pool ready") {
			parseArtifact.HasWorkerPoolReadyLog = true
		}
		if strings.Contains(parseLowerText, "background worker unavailable") ||
			strings.Contains(parseLowerText, "continuing without worker") ||
			strings.Contains(parseLowerText, "falling back to main-thread work") {
			parseArtifact.HasWorkerFallbackLog = true
		}
	})
	parsePage.OnResponse(func(parseResponse playwright.Response) {
		parseLowerURL := strings.ToLower(strings.TrimSpace(parseResponse.URL()))
		parseMu.Lock()
		defer parseMu.Unlock()
		if strings.Contains(parseLowerURL, "/app/chat.wasm") && parseResponse.Status() < 400 {
			parseArtifact.GetChatWasmResponseCount++
		}
		if strings.Contains(parseLowerURL, "/worker/background-worker.wasm") && parseResponse.Status() < 400 {
			parseArtifact.GetWorkerWasmResponseCount++
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

	parseResponse, parseErr := parsePage.Goto(parseBaseURL+parseArtifact.GetRoute, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto startup route: %v", parseErr)
	}
	if parseResponse == nil {
		parseT.Fatalf("goto startup route returned nil response")
	}
	parseArtifact.GetHTTPStatus = parseResponse.Status()
	parsePage.WaitForTimeout(200)

	if _, parseErr = parsePage.WaitForFunction(
		`() => {
			const root = document.getElementById("app");
			const shell = document.getElementById("boot-shell");
			const mounted = !!root && ((root.children && root.children.length > 0) || String(root.textContent || "").trim().length > 0);
			return !shell && mounted;
		}`,
		nil,
	); parseErr != nil {
		parseMu.Lock()
		parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
		parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
		parseMu.Unlock()
		parseT.Fatalf(
			"wait for wasm mount and boot-shell removal failed: %v console=%q page-errors=%q",
			parseErr,
			parseArtifact.GetConsoleSampleText,
			parseArtifact.GetPageErrorSampleText,
		)
	}

	parseDeadline := time.Now().Add(25 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseMu.Lock()
		hasBootSignals := parseArtifact.HasGRPCReadyLog &&
			parseArtifact.HasWorkerReadyLog &&
			parseArtifact.HasWorkerPoolReadyLog &&
			parseArtifact.GetChatWasmResponseCount > 0 &&
			parseArtifact.GetWorkerWasmResponseCount > 0
		parseMu.Unlock()
		if hasBootSignals {
			break
		}
		time.Sleep(200 * time.Millisecond)
	}

	parseTitle, parseErr := parsePage.Title()
	if parseErr == nil {
		parseArtifact.GetPageTitle = parseTitle
	}

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// formatExample100StartupBootSummary formats one startup artifact for concise test logs.
func formatExample100StartupBootSummary(parseArtifact example100StartupBootArtifact) string {
	return fmt.Sprintf(
		"route=%s status=%d title=%q grpc-ready=%t wasm-responses=%d worker-wasm-responses=%d worker-ready=%t worker-pool-ready=%t worker-fallback=%t console-errors=%d page-errors=%d console-samples=%q page-error-samples=%q",
		parseArtifact.GetRoute,
		parseArtifact.GetHTTPStatus,
		parseArtifact.GetPageTitle,
		parseArtifact.HasGRPCReadyLog,
		parseArtifact.GetChatWasmResponseCount,
		parseArtifact.GetWorkerWasmResponseCount,
		parseArtifact.HasWorkerReadyLog,
		parseArtifact.HasWorkerPoolReadyLog,
		parseArtifact.HasWorkerFallbackLog,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// TestExample100StartupBoot verifies server start, WASM shell boot, and background worker boot on the root route.
func TestExample100StartupBoot(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100StartupServer(parseT, parseRepoRoot, "18103")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100StartupBootArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 startup boot: %s", formatExample100StartupBootSummary(parseArtifact))

		if parseArtifact.GetHTTPStatus >= 400 {
			parseT.Fatalf("startup route returned status %d", parseArtifact.GetHTTPStatus)
		}
		if !strings.Contains(parseArtifact.GetPageTitle, "RelayDesk") {
			parseT.Fatalf("startup page title = %q, expected RelayDesk", parseArtifact.GetPageTitle)
		}
		if parseArtifact.GetChatWasmResponseCount < 1 {
			parseT.Fatalf("expected at least one /app/chat.wasm response, got %d", parseArtifact.GetChatWasmResponseCount)
		}
		if parseArtifact.GetWorkerWasmResponseCount < 1 {
			parseT.Fatalf("expected at least one /worker/background-worker.wasm response, got %d", parseArtifact.GetWorkerWasmResponseCount)
		}
		if !parseArtifact.HasGRPCReadyLog {
			parseT.Fatalf("missing grpc-ready startup log; summary=%s", formatExample100StartupBootSummary(parseArtifact))
		}
		if !parseArtifact.HasWorkerReadyLog {
			parseT.Fatalf("missing worker-ready startup log; summary=%s", formatExample100StartupBootSummary(parseArtifact))
		}
		if !parseArtifact.HasWorkerPoolReadyLog {
			parseT.Fatalf("missing worker-pool-ready startup log; summary=%s", formatExample100StartupBootSummary(parseArtifact))
		}
		if parseArtifact.HasWorkerFallbackLog {
			parseT.Fatalf("worker fallback detected during startup; summary=%s", formatExample100StartupBootSummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("startup console errors=%d; summary=%s", parseArtifact.GetConsoleErrorCount, formatExample100StartupBootSummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("startup page errors=%d; summary=%s", parseArtifact.GetPageErrorCount, formatExample100StartupBootSummary(parseArtifact))
		}
	})
}
