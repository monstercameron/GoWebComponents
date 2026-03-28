//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"database/sql"
	"os"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	playwright "github.com/playwright-community/playwright-go"
	"golang.org/x/crypto/bcrypt"
)

// buildExample100CustomerErrorClientArtifacts rebuilds the example-100 WASM client assets used by the server binary.
func buildExample100CustomerErrorClientArtifacts(parseT *testing.T, parseRepoRoot string) {
	parseT.Helper()
	parseBuildCommand := exec.Command("go", "run", "./examples/100-ai-chat-wizard/cmd/build-client")
	parseBuildCommand.Dir = parseRepoRoot
	if parseBuildOutput, parseBuildErr := parseBuildCommand.CombinedOutput(); parseBuildErr != nil {
		parseT.Fatalf("build example 100 client artifacts: %v\n%s", parseBuildErr, strings.TrimSpace(string(parseBuildOutput)))
	}
}

// copyExample100CustomerErrorDatabase copies one existing runtime fixture database into the test temp directory.
func copyExample100CustomerErrorDatabase(parseT *testing.T, parseRepoRoot string, parseDBPath string) {
	parseT.Helper()
	parseSourceDBPath := filepath.Join(parseRepoRoot, "examples", "100-ai-chat-wizard", "bin", "runtime", "test_chat.db")
	parseBytes, parseErr := os.ReadFile(parseSourceDBPath)
	if parseErr != nil {
		parseT.Fatalf("read example 100 fixture db: %v", parseErr)
	}
	if parseErr := os.WriteFile(parseDBPath, parseBytes, 0o644); parseErr != nil {
		parseT.Fatalf("write example 100 test db copy: %v", parseErr)
	}
}

// seedExample100CustomerErrorAdminUser ensures the admin journey login account exists with the expected password.
func seedExample100CustomerErrorAdminUser(parseT *testing.T, parseDBPath string) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open customer-error sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	parsePasswordHash, parseErr := bcrypt.GenerateFromPassword([]byte(example100AdminJourneyLoginPassword), bcrypt.DefaultCost)
	if parseErr != nil {
		parseT.Fatalf("generate bcrypt hash for admin login: %v", parseErr)
	}
	parseNowRFC3339 := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseDB.Exec(
		`INSERT INTO users (email, password_hash, created_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(email) DO UPDATE SET password_hash = excluded.password_hash`,
		example100AdminJourneyLoginEmail,
		string(parsePasswordHash),
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("upsert admin journey user: %v", parseErr)
	}
	var parseAdminUserID int64
	if parseErr := parseDB.QueryRow(`SELECT id FROM users WHERE email = ? COLLATE NOCASE`, example100AdminJourneyLoginEmail).Scan(&parseAdminUserID); parseErr != nil {
		parseT.Fatalf("resolve admin journey user id: %v", parseErr)
	}
	if _, parseErr := parseDB.Exec(
		`INSERT INTO user_access_states (user_id, status, reason, disabled_by_user_id, disabled_at, updated_at)
		 VALUES (?, 'active', '', 0, '', ?)
		 ON CONFLICT(user_id) DO UPDATE SET
			status = excluded.status,
			reason = excluded.reason,
			disabled_by_user_id = excluded.disabled_by_user_id,
			disabled_at = excluded.disabled_at,
			updated_at = excluded.updated_at`,
		parseAdminUserID,
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("upsert admin journey user access state: %v", parseErr)
	}
}

// startExample100CustomerErrorServer starts one seeded server and returns a stop function for forced runtime-failure coverage.
func startExample100CustomerErrorServer(parseT *testing.T, parseRepoRoot string, parsePort string) (string, func()) {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "customer_error_surface.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-customer-error-surface-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)

	copyExample100CustomerErrorDatabase(parseT, parseRepoRoot, parseDBPath)
	seedExample100CustomerErrorAdminUser(parseT, parseDBPath)
	grantExample100AdminJourneySuperuserRole(parseT, parseDBPath)
	buildExample100CustomerErrorClientArtifacts(parseT, parseRepoRoot)
	buildExample100HappyPathServerBinary(parseT, parseRepoRoot, parseBinaryPath)

	parseStopRaw := startExamplesCommandWithEnv(
		parseT,
		parseRepoRoot,
		[]string{
			"LISTEN_ADDR=" + parseAddress,
			"CHAT_DB_PATH=" + parseDBPath,
			"CHAT_LOG_DIR=" + parseLogDir,
		},
		parseBinaryPath,
	)
	var parseStopOnce sync.Once
	parseStop := func() {
		parseStopOnce.Do(parseStopRaw)
	}
	parseT.Cleanup(parseStop)

	parseBaseURL := "http://" + parseAddress
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 120*time.Second)
	return parseBaseURL, parseStop
}

// parseHasExample100RawInternalErrorText reports whether one visible UI error includes raw backend/internal wording.
func parseHasExample100RawInternalErrorText(parseText string) bool {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseText))
	parseBanned := []string{
		"rpc error:",
		"code =",
		"desc =",
		"stack trace",
		"panic",
		"sql",
		"sqlite",
		"grpc",
		"dial tcp",
		"no such host",
	}
	for _, parseToken := range parseBanned {
		if strings.Contains(parseNormalized, parseToken) {
			return true
		}
	}
	return false
}

// parseReadExample100TextContent reads one selector text and trims surrounding whitespace.
func parseReadExample100TextContent(parseT *testing.T, parsePage playwright.Page, parseSelector string) string {
	parseT.Helper()
	parseValue, parseErr := parsePage.TextContent(parseSelector)
	if parseErr != nil {
		parseT.Fatalf("read selector text (%s): %v", parseSelector, parseErr)
	}
	return strings.TrimSpace(parseValue)
}

// parseWaitExample100AuthSubmitReady waits for one enabled auth submit button.
func parseWaitExample100AuthSubmitReady(parseT *testing.T, parsePage playwright.Page) {
	parseT.Helper()
	if _, parseErr := parsePage.WaitForFunction(`() => {
		const submitButton = document.querySelector("button.w-full");
		return !!submitButton && !submitButton.disabled;
	}`, nil); parseErr != nil {
		parseT.Fatalf("wait for enabled auth submit button: %v", parseErr)
	}
}

// parseSubmitExample100AuthCredentials fills auth credentials and submits with the visible login button.
func parseSubmitExample100AuthCredentials(parseT *testing.T, parsePage playwright.Page, parseEmail string, parsePassword string) {
	parseT.Helper()
	if parseErr := parsePage.Fill("#auth-email-input", parseEmail); parseErr != nil {
		parseT.Fatalf("fill auth email (%s): %v", parseEmail, parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", parsePassword); parseErr != nil {
		parseT.Fatalf("fill auth password (%s): %v", parseEmail, parseErr)
	}
	parseWaitExample100AuthSubmitReady(parseT, parsePage)
	if _, parseErr := parsePage.Evaluate(`() => {
		const submitButton = document.querySelector("button.w-full:not([disabled])");
		if (!submitButton) {
			throw new Error("enabled auth submit button not found");
		}
		submitButton.click();
	}`); parseErr != nil {
		parseT.Fatalf("click auth submit button (%s): %v", parseEmail, parseErr)
	}
}

// TestExample100CustomerErrorSurfaceRegression verifies auth/chat/settings/dashboard failures show calm copy with request IDs and no raw internals.
func TestExample100CustomerErrorSurfaceRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL, parseStopServer := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18111")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseErr != nil {
			parseT.Fatalf("goto /login: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
			parseT.Fatalf("wait for auth email input: %v", parseErr)
		}

		parseSubmitExample100AuthCredentials(parseT, parsePage, "customer@email.com", "wrong-password")
		if _, parseErr := parsePage.WaitForFunction(
			`() => !!document.querySelector("#auth-error-banner") || !!document.querySelector("#chat-input")`,
			nil,
		); parseErr != nil {
			parseDebugValue, _ := parsePage.Evaluate(`() => ({
				path: window.location.pathname + window.location.search,
				title: document.title || "",
				buttons: Array.from(document.querySelectorAll("button")).map((button) => ({
					text: String(button.textContent || "").trim(),
					disabled: !!button.disabled,
					className: String(button.className || ""),
				})).slice(0, 8),
				hasAuthError: !!document.querySelector("#auth-error-banner"),
				hasChatInput: !!document.querySelector("#chat-input"),
				body: ((document.body && document.body.innerText) || "").slice(0, 1400),
			})`)
			parseT.Fatalf("wait for auth failure/shell state: %v debug=%#v", parseErr, parseDebugValue)
		}
		parseHasChatAfterFailureValue, parseErr := parsePage.Evaluate(`() => !!document.querySelector("#chat-input")`)
		if parseErr != nil {
			parseT.Fatalf("evaluate chat shell after auth failure leg: %v", parseErr)
		}
		if parseHasChatAfterFailure, _ := parseHasChatAfterFailureValue.(bool); parseHasChatAfterFailure {
			parseT.Fatal("wrong-password auth failure leg unexpectedly reached authenticated chat shell")
		}
		parseAuthErrorText := parseReadExample100TextContent(parseT, parsePage, "#auth-error-banner")
		if !strings.Contains(parseAuthErrorText, "Request ID:") {
			parseT.Fatalf("auth failure missing request id: %q", parseAuthErrorText)
		}
		if parseHasExample100RawInternalErrorText(parseAuthErrorText) {
			parseT.Fatalf("auth failure leaked raw internals: %q", parseAuthErrorText)
		}

		parseSubmitExample100AuthCredentials(parseT, parsePage, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)
		if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
			parseDebugValue, _ := parsePage.Evaluate(`() => ({
				path: window.location.pathname + window.location.search,
				title: document.title || "",
				hasAuthError: !!document.querySelector("#auth-error-banner"),
				authErrorText: (document.querySelector("#auth-error-banner") && document.querySelector("#auth-error-banner").textContent) || "",
				body: ((document.body && document.body.innerText) || "").slice(0, 1400),
			})`)
			parseT.Fatalf("wait for chat shell after admin login: %v debug=%#v", parseErr, parseDebugValue)
		}
		if _, parseErr := parsePage.WaitForSelector("#open-admin-dashboard-btn"); parseErr != nil {
			parseT.Fatalf("wait for admin dashboard entry button: %v", parseErr)
		}

		parseStopServer()
		time.Sleep(1200 * time.Millisecond)

		if parseErr := parsePage.Fill("#chat-input", "customer-safe regression send"); parseErr != nil {
			parseT.Fatalf("fill chat input for send failure: %v", parseErr)
		}
		if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
			parseT.Fatalf("click send for failure leg: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => document.body && document.body.innerText.includes("We couldn't send your message right now.")`, nil); parseErr != nil {
			parseDebugValue, _ := parsePage.Evaluate(`() => ({
				path: window.location.pathname + window.location.search,
				hasStreamingBubble: !!document.querySelector("#streaming-assistant-bubble"),
				hasSendButton: !!document.querySelector("#send-btn"),
				hasSettingsButton: !!document.querySelector("#open-settings-btn"),
				body: ((document.body && document.body.innerText) || "").slice(0, 1800),
			})`)
			parseT.Fatalf("wait for chat failure copy: %v debug=%#v", parseErr, parseDebugValue)
		}
		parseChatFailureTextValue, parseErr := parsePage.Evaluate(`() => {
			const bodyText = (document.body && document.body.innerText) || "";
			const marker = "We couldn't send your message right now.";
			const idx = bodyText.indexOf(marker);
			if (idx < 0) return "";
			return bodyText.slice(idx, idx + 260);
		}`)
		if parseErr != nil {
			parseT.Fatalf("evaluate chat failure text: %v", parseErr)
		}
		parseChatFailureText := strings.TrimSpace(parseChatFailureTextValue.(string))
		if !strings.Contains(parseChatFailureText, "Request ID:") {
			parseT.Fatalf("chat failure missing request id: %q", parseChatFailureText)
		}
		if parseHasExample100RawInternalErrorText(parseChatFailureText) {
			parseT.Fatalf("chat failure leaked raw internals: %q", parseChatFailureText)
		}

		if parseErr := parsePage.Click("#open-settings-btn"); parseErr != nil {
			parseT.Fatalf("open settings modal: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
			parseT.Fatalf("wait for settings profile pane: %v", parseErr)
		}
		if parseErr := parsePage.Click(`button:has-text("Save")`); parseErr != nil {
			parseT.Fatalf("click settings save for failure leg: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#settings-save-error"); parseErr != nil {
			parseT.Fatalf("wait for settings save error: %v", parseErr)
		}
		parseSettingsErrorText := parseReadExample100TextContent(parseT, parsePage, "#settings-save-error")
		if !strings.Contains(parseSettingsErrorText, "Request ID:") {
			parseT.Fatalf("settings failure missing request id: %q", parseSettingsErrorText)
		}
		if parseHasExample100RawInternalErrorText(parseSettingsErrorText) {
			parseT.Fatalf("settings failure leaked raw internals: %q", parseSettingsErrorText)
		}

		if parseErr := parsePage.Click(`button:has-text("Cancel")`); parseErr != nil {
			parseT.Fatalf("close settings modal: %v", parseErr)
		}
		if parseErr := parsePage.Click("#open-admin-dashboard-btn"); parseErr != nil {
			parseT.Fatalf("open admin dashboard: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/dashboard")`, nil); parseErr != nil {
			parseT.Fatalf("wait for dashboard route: %v", parseErr)
		}
		if _, parseErr := parsePage.WaitForSelector("#dashboard-error-banner"); parseErr != nil {
			parseT.Fatalf("wait for dashboard error banner: %v", parseErr)
		}
		parseDashboardErrorText := parseReadExample100TextContent(parseT, parsePage, "#dashboard-error-banner")
		if !strings.Contains(parseDashboardErrorText, "Request ID:") {
			parseT.Fatalf("dashboard failure missing request id: %q", parseDashboardErrorText)
		}
		if parseHasExample100RawInternalErrorText(parseDashboardErrorText) {
			parseT.Fatalf("dashboard failure leaked raw internals: %q", parseDashboardErrorText)
		}
	})
}
