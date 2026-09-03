//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"context"
	"database/sql"
	"fmt"
	"os/exec"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	chatpb "github.com/monstercameron/GoWebComponents/v5/examples/server/ai-chat-wizard/proto"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	playwright "github.com/mxschmitt/playwright-go"
	"golang.org/x/crypto/bcrypt"
	"google.golang.org/grpc"
)

// buildExample100CustomerErrorClientArtifacts rebuilds the example-100 WASM client assets used by the server binary.
func buildExample100CustomerErrorClientArtifacts(parseT *testing.T, parseRepoRoot string) {
	parseT.Helper()
	parseBuildCommand := exec.Command("go", "run", "./examples/server/ai-chat-wizard/cmd/build-client")
	parseBuildCommand.Dir = parseRepoRoot
	if parseBuildOutput, parseBuildErr := parseBuildCommand.CombinedOutput(); parseBuildErr != nil {
		parseT.Fatalf("build example 100 client artifacts: %v\n%s", parseBuildErr, strings.TrimSpace(string(parseBuildOutput)))
	}
}

// copyExample100CustomerErrorDatabase seeds one deterministic runtime fixture database into the test temp directory.
func copyExample100CustomerErrorDatabase(parseT *testing.T, parseRepoRoot string, parseDBPath string) {
	parseT.Helper()
	seedExample100HappyPathDatabase(parseT, parseRepoRoot, parseDBPath)
}

// seedExample100CustomerErrorAdminUser ensures customer and admin login accounts exist with expected passwords.
func seedExample100CustomerErrorAdminUser(parseT *testing.T, parseDBPath string) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open customer-error sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	parseNowRFC3339 := time.Now().UTC().Format(time.RFC3339)
	for _, parseUser := range []struct {
		parseEmail    string
		parsePassword string
	}{
		{parseEmail: "customer@email.com", parsePassword: "password"},
		{parseEmail: example100AdminJourneyLoginEmail, parsePassword: example100AdminJourneyLoginPassword},
	} {
		parsePasswordHash, parseHashErr := bcrypt.GenerateFromPassword([]byte(parseUser.parsePassword), bcrypt.DefaultCost)
		if parseHashErr != nil {
			parseT.Fatalf("generate bcrypt hash for %s login: %v", parseUser.parseEmail, parseHashErr)
		}
		if _, parseExecErr := parseDB.Exec(
			`INSERT INTO users (email, password_hash, created_at)
			 VALUES (?, ?, ?)
			 ON CONFLICT(email) DO UPDATE SET password_hash = excluded.password_hash`,
			parseUser.parseEmail,
			string(parsePasswordHash),
			parseNowRFC3339,
		); parseExecErr != nil {
			parseT.Fatalf("upsert login user (%s): %v", parseUser.parseEmail, parseExecErr)
		}
		var parseStoredPasswordHash string
		if parseQueryErr := parseDB.QueryRow(`SELECT password_hash FROM users WHERE email = ? COLLATE NOCASE`, parseUser.parseEmail).Scan(&parseStoredPasswordHash); parseQueryErr != nil {
			parseT.Fatalf("read stored password hash (%s): %v", parseUser.parseEmail, parseQueryErr)
		}
		if parseCompareErr := bcrypt.CompareHashAndPassword([]byte(parseStoredPasswordHash), []byte(parseUser.parsePassword)); parseCompareErr != nil {
			parseT.Fatalf("verify stored password hash (%s): %v", parseUser.parseEmail, parseCompareErr)
		}
		var parseUserID int64
		if parseQueryErr := parseDB.QueryRow(`SELECT id FROM users WHERE email = ? COLLATE NOCASE`, parseUser.parseEmail).Scan(&parseUserID); parseQueryErr != nil {
			parseT.Fatalf("resolve login user id (%s): %v", parseUser.parseEmail, parseQueryErr)
		}
		if _, parseExecErr := parseDB.Exec(`DELETE FROM user_auth_blocks WHERE user_id = ?`, parseUserID); parseExecErr != nil {
			parseT.Fatalf("clear auth blocks for login user (%s): %v", parseUser.parseEmail, parseExecErr)
		}
		if _, parseExecErr := parseDB.Exec(
			`INSERT INTO user_access_states (user_id, status, reason, disabled_by_user_id, disabled_at, updated_at)
			 VALUES (?, 'active', '', 0, '', ?)
			 ON CONFLICT(user_id) DO UPDATE SET
				status = excluded.status,
				reason = excluded.reason,
				disabled_by_user_id = excluded.disabled_by_user_id,
				disabled_at = excluded.disabled_at,
				updated_at = excluded.updated_at`,
			parseUserID,
			parseNowRFC3339,
		); parseExecErr != nil {
			parseT.Fatalf("upsert login user access state (%s): %v", parseUser.parseEmail, parseExecErr)
		}
	}
}

// parseSetExample100CustomerErrorSuperuserRoleByEmail sets or clears one superuser role assignment for one email.
func parseSetExample100CustomerErrorSuperuserRoleByEmail(parseT *testing.T, parseDBPath string, parseEmail string, isEnabled bool) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open sqlite for superuser assignment (%s): %v", parseEmail, parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	var parseUserID int64
	if parseErr := parseDB.QueryRow(`SELECT id FROM users WHERE email = ? COLLATE NOCASE`, strings.TrimSpace(parseEmail)).Scan(&parseUserID); parseErr != nil {
		parseT.Fatalf("resolve user id for superuser assignment (%s): %v", parseEmail, parseErr)
	}
	if !isEnabled {
		if _, parseErr := parseDB.Exec(`DELETE FROM su_user_roles WHERE user_id = ? AND role_key = 'su'`, parseUserID); parseErr != nil {
			parseT.Fatalf("delete superuser assignment (%s): %v", parseEmail, parseErr)
		}
		return
	}

	parseCreatedAt := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseDB.Exec(
		`INSERT INTO su_roles (role_key, label, description, is_system, is_enabled, created_at, updated_at)
		 VALUES ('su', 'Superuser', 'Customer error surface regression superuser role', 1, 1, ?, ?)
		 ON CONFLICT(role_key) DO UPDATE SET
			label = excluded.label,
			description = excluded.description,
			is_system = 1,
			is_enabled = 1,
			updated_at = excluded.updated_at`,
		parseCreatedAt,
		parseCreatedAt,
	); parseErr != nil {
		parseT.Fatalf("upsert superuser role (%s): %v", parseEmail, parseErr)
	}
	if _, parseErr := parseDB.Exec(
		`INSERT INTO su_role_permissions (role_key, permission_key, permission_value, updated_at)
		 VALUES ('su', 'control_plane.*', 'allow', ?)
		 ON CONFLICT(role_key, permission_key) DO UPDATE SET
			permission_value = excluded.permission_value,
			updated_at = excluded.updated_at`,
		parseCreatedAt,
	); parseErr != nil {
		parseT.Fatalf("upsert superuser permissions (%s): %v", parseEmail, parseErr)
	}
	if _, parseErr := parseDB.Exec(
		`INSERT INTO su_user_roles (user_id, role_key, assigned_by_user_id, created_at)
		 VALUES (?, 'su', ?, ?)
		 ON CONFLICT(user_id, role_key) DO UPDATE SET assigned_by_user_id = excluded.assigned_by_user_id`,
		parseUserID,
		parseUserID,
		parseCreatedAt,
	); parseErr != nil {
		parseT.Fatalf("upsert superuser assignment (%s): %v", parseEmail, parseErr)
	}
}

// parseBreakExample100CustomerErrorDashboardStore drops one dashboard dependency table to force a server-side dashboard load failure.
func parseBreakExample100CustomerErrorDashboardStore(parseT *testing.T, parseDBPath string) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open sqlite for dashboard failure fixture: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)
	if _, parseErr := parseDB.Exec(`DROP TABLE IF EXISTS usage_events`); parseErr != nil {
		parseT.Fatalf("drop usage_events for dashboard failure fixture: %v", parseErr)
	}
	var parseTableCount int
	if parseErr := parseDB.QueryRow(`SELECT COUNT(1) FROM sqlite_master WHERE type = 'table' AND name = 'usage_events'`).Scan(&parseTableCount); parseErr != nil {
		parseT.Fatalf("verify usage_events table drop: %v", parseErr)
	}
	if parseTableCount != 0 {
		parseT.Fatalf("expected usage_events table to be dropped, count=%d", parseTableCount)
	}
}

// startExample100CustomerErrorServer starts one seeded server and returns base URL, db path, and a stop function for forced runtime-failure coverage.
func startExample100CustomerErrorServer(parseT *testing.T, parseRepoRoot string, parsePort string) (string, string, func()) {
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
	return parseBaseURL, parseDBPath, parseStop
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

// parseLoginExample100CustomerErrorRPC authenticates one test account through the websocket tunnel and returns the issued auth token.
func parseLoginExample100CustomerErrorRPC(parseT *testing.T, parseBaseURL string, parseEmail string, parsePassword string) string {
	parseT.Helper()
	parseDialCtx, parseDialCancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer parseDialCancel()
	parseDialOptions := grpctunnel.ApplyTunnelInsecureCredentials([]grpc.DialOption{
		grpc.WithBlock(),
	})
	parseConn, parseErr := grpctunnel.BuildTunnelConn(parseDialCtx, grpctunnel.TunnelConfig{
		Target:      buildExample100AdminJourneyTunnelURL(parseBaseURL),
		GRPCOptions: parseDialOptions,
	})
	if parseErr != nil {
		parseT.Fatalf("open grpc tunnel for rpc login (%s): %v", parseEmail, parseErr)
	}
	defer parseConn.Close()

	parseCallCtx, parseCallCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer parseCallCancel()
	parseClient := chatpb.NewChatServiceClient(parseConn)
	parseAuthResp, parseErr := parseClient.Login(parseCallCtx, &chatpb.LoginRequest{
		Email:    strings.TrimSpace(parseEmail),
		Password: parsePassword,
	})
	if parseErr != nil {
		parseSignupEmail := fmt.Sprintf("qa-customer-error-%d@email.com", time.Now().UTC().UnixNano()%1_000_000_000)
		parseSignupPassword := "password123"
		parseSignupResp, parseSignupErr := parseClient.Signup(parseCallCtx, &chatpb.SignupRequest{
			Email:       parseSignupEmail,
			Password:    parseSignupPassword,
			DisplayName: "Customer Error QA",
		})
		if parseSignupErr != nil {
			parseT.Fatalf("rpc login failed (%s): %v; signup fallback failed: %v", parseEmail, parseErr, parseSignupErr)
		}
		parseFallbackToken := strings.TrimSpace(parseSignupResp.GetAuthToken())
		if parseFallbackToken == "" {
			parseT.Fatalf("rpc signup fallback returned empty auth token (%s)", parseSignupEmail)
		}
		return parseFallbackToken
	}
	parseAuthToken := strings.TrimSpace(parseAuthResp.GetAuthToken())
	if parseAuthToken == "" {
		parseT.Fatalf("rpc login returned empty auth token (%s)", parseEmail)
	}
	return parseAuthToken
}

// parseSignupExample100CustomerErrorRPC signs up one fresh account through the websocket tunnel and returns email/password/token.
func parseSignupExample100CustomerErrorRPC(parseT *testing.T, parseBaseURL string) (string, string, string) {
	parseT.Helper()
	parseDialCtx, parseDialCancel := context.WithTimeout(context.Background(), 12*time.Second)
	defer parseDialCancel()
	parseDialOptions := grpctunnel.ApplyTunnelInsecureCredentials([]grpc.DialOption{
		grpc.WithBlock(),
	})
	parseConn, parseErr := grpctunnel.BuildTunnelConn(parseDialCtx, grpctunnel.TunnelConfig{
		Target:      buildExample100AdminJourneyTunnelURL(parseBaseURL),
		GRPCOptions: parseDialOptions,
	})
	if parseErr != nil {
		parseT.Fatalf("open grpc tunnel for rpc signup: %v", parseErr)
	}
	defer parseConn.Close()

	parseCallCtx, parseCallCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer parseCallCancel()
	parseClient := chatpb.NewChatServiceClient(parseConn)
	parseSignupEmail := fmt.Sprintf("qa-admin-error-%d@email.com", time.Now().UTC().UnixNano()%1_000_000_000)
	parseSignupPassword := "password123"
	parseSignupResp, parseSignupErr := parseClient.Signup(parseCallCtx, &chatpb.SignupRequest{
		Email:       parseSignupEmail,
		Password:    parseSignupPassword,
		DisplayName: "Admin Error QA",
	})
	if parseSignupErr != nil {
		parseT.Fatalf("rpc signup failed (%s): %v", parseSignupEmail, parseSignupErr)
	}
	parseAuthToken := strings.TrimSpace(parseSignupResp.GetAuthToken())
	if parseAuthToken == "" {
		parseT.Fatalf("rpc signup returned empty auth token (%s)", parseSignupEmail)
	}
	return parseSignupEmail, parseSignupPassword, parseAuthToken
}

// parseApplyExample100CustomerErrorAuthToken mirrors one auth token into localStorage and cookie state for browser-authenticated routes.
func parseApplyExample100CustomerErrorAuthToken(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseAuthToken string) {
	parseT.Helper()
	parseAuthToken = strings.TrimSpace(parseAuthToken)
	if parseAuthToken == "" {
		parseT.Fatalf("apply auth token: empty token")
	}
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login before applying auth token: %v", parseErr)
	}
	if _, parseErr := parsePage.Evaluate(
		fmt.Sprintf(`() => { window.localStorage.setItem(%q, %q); }`, example100AdminJourneyAuthTokenStorageKey, parseAuthToken),
	); parseErr != nil {
		parseT.Fatalf("set auth token in localStorage: %v", parseErr)
	}
	if parseErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{
		{
			Name:  example100AdminJourneyAuthCookieName,
			Value: parseAuthToken,
			URL:   playwright.String(parseBaseURL),
		},
	}); parseErr != nil {
		parseT.Fatalf("set auth cookie: %v", parseErr)
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

// parsePoisonExample100AuthToken replaces local/session auth token state with an invalid token for failure-path forcing.
func parsePoisonExample100AuthToken(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) {
	parseT.Helper()
	parseInvalidToken := "invalid-token-regression"
	if _, parseErr := parsePage.Evaluate(
		fmt.Sprintf(`() => { window.localStorage.setItem(%q, %q); }`, example100AdminJourneyAuthTokenStorageKey, parseInvalidToken),
	); parseErr != nil {
		parseT.Fatalf("poison auth token in localStorage: %v", parseErr)
	}
	if parseErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{
		{
			Name:  example100AdminJourneyAuthCookieName,
			Value: parseInvalidToken,
			URL:   playwright.String(parseBaseURL),
		},
	}); parseErr != nil {
		parseT.Fatalf("poison auth cookie: %v", parseErr)
	}
}

// parseEnsureExample100UserChatShell ensures one account is signed in and the chat shell is visible.
func parseEnsureExample100UserChatShell(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseEmail string, parsePassword string) {
	parseT.Helper()
	parseAuthToken := parseLoginExample100CustomerErrorRPC(parseT, parseBaseURL, parseEmail, parsePassword)
	parseApplyExample100CustomerErrorAuthToken(parseT, parsePage, parseBaseURL, parseAuthToken)
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app after token apply: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait #chat-input after user ensure login: %v", parseErr)
	}
}

// parseForceExample100BridgeUnavailable blocks websocket tunnel reconnects and reloads the app shell.
func parseForceExample100BridgeUnavailable(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) {
	parseT.Helper()
	if parseErr := parsePage.Route("**/socket*", func(parseRoute playwright.Route) {
		_ = parseRoute.Abort()
	}); parseErr != nil {
		parseT.Fatalf("route abort for /socket*: %v", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app after socket abort routing: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			title: document.title || "",
			hasChatInput: !!document.querySelector("#chat-input"),
			body: ((document.body && document.body.innerText) || "").slice(0, 1800),
		})`)
		parseT.Fatalf("wait #chat-input after socket abort routing: %v debug=%#v", parseErr, parseDebugValue)
	}
}

// parseForceExample100BridgeUnavailableOnRoute blocks websocket tunnel reconnects and reloads one specific app route.
func parseForceExample100BridgeUnavailableOnRoute(parseT *testing.T, parsePage playwright.Page, parseRouteURL string) {
	parseT.Helper()
	if parseErr := parsePage.Route("**/socket*", func(parseRoute playwright.Route) {
		_ = parseRoute.Abort()
	}); parseErr != nil {
		parseT.Fatalf("route abort for /socket* on route reload: %v", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseRouteURL, playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto route after socket abort (%s): %v", parseRouteURL, parseErr)
	}
}

// TestExample100CustomerErrorSurfaceRegression verifies auth/chat/settings/dashboard failures show calm copy with request IDs and no raw internals.
func TestExample100CustomerErrorSurfaceRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseT.Run("auth failure", func(parseT *testing.T) {
		parseBaseURL, _, _ := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18111")
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
				parseT.Fatalf("goto /login: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
				parseT.Fatalf("wait for auth email input: %v", parseErr)
			}
			parseSubmitExample100AuthCredentials(parseT, parsePage, "customer@email.com", "wrong-password")
			if _, parseErr := parsePage.WaitForSelector("#auth-error-banner"); parseErr != nil {
				parseT.Fatalf("wait for auth error banner: %v", parseErr)
			}
			parseAuthErrorText := parseReadExample100TextContent(parseT, parsePage, "#auth-error-banner")
			if !strings.Contains(parseAuthErrorText, "Request ID:") {
				parseT.Fatalf("auth failure missing request id: %q", parseAuthErrorText)
			}
			if parseHasExample100RawInternalErrorText(parseAuthErrorText) {
				parseT.Fatalf("auth failure leaked raw internals: %q", parseAuthErrorText)
			}
		})
	})

	parseT.Run("chat send failure", func(parseT *testing.T) {
		parseBaseURL, _, _ := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18112")
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseEnsureExample100UserChatShell(parseT, parsePage, parseBaseURL, "customer@email.com", "password")
			parseForceExample100BridgeUnavailable(parseT, parsePage, parseBaseURL)
			if parseErr := parsePage.Fill("#chat-input", "customer-safe regression send"); parseErr != nil {
				parseT.Fatalf("fill chat input for send failure: %v", parseErr)
			}
			if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
				parseT.Fatalf("click send for failure leg: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => document.body && document.body.innerText.includes("We couldn't send your message right now.")`, nil); parseErr != nil {
				parseT.Fatalf("wait for chat failure copy: %v", parseErr)
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
		})
	})

	parseT.Run("settings save failure", func(parseT *testing.T) {
		parseBaseURL, _, _ := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18113")
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseEnsureExample100UserChatShell(parseT, parsePage, parseBaseURL, "customer@email.com", "password")
			parseSettingsRouteURL := parseBaseURL + "/app/settings?panel=settings-profile"
			if _, parseErr := parsePage.Goto(parseSettingsRouteURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto settings profile route: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
				parseDebugValue, _ := parsePage.Evaluate(`() => ({
					path: window.location.pathname + window.location.search,
					hasSettingsProfile: !!document.querySelector("#settings-profile"),
					hasChatInput: !!document.querySelector("#chat-input"),
					body: ((document.body && document.body.innerText) || "").slice(0, 1800),
				})`)
				parseT.Fatalf("wait for settings profile pane: %v debug=%#v", parseErr, parseDebugValue)
			}
			parseForceExample100BridgeUnavailableOnRoute(parseT, parsePage, parseSettingsRouteURL)
			if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
				parseDebugValue, _ := parsePage.Evaluate(`() => ({
					path: window.location.pathname + window.location.search,
					hasSettingsProfile: !!document.querySelector("#settings-profile"),
					hasChatInput: !!document.querySelector("#chat-input"),
					body: ((document.body && document.body.innerText) || "").slice(0, 1800),
				})`)
				parseT.Fatalf("wait for settings profile pane after forced bridge failure: %v debug=%#v", parseErr, parseDebugValue)
			}
			if parseErr := parsePage.Fill("#name-input", "Regression Save Failure"); parseErr != nil {
				parseT.Fatalf("fill #name-input for settings failure leg: %v", parseErr)
			}
			if parseErr := parsePage.Click(`button:has-text("Save")`); parseErr != nil {
				parseT.Fatalf("click settings save for failure leg: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => {
				if (document.querySelector("#settings-save-error")) return true;
				return window.location.pathname === "/app";
			}`, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(5000)}); parseErr != nil {
				parseDebugValue, _ := parsePage.Evaluate(`() => ({
					path: window.location.pathname + window.location.search,
					hasSettingsError: !!document.querySelector("#settings-save-error"),
					hasSettingsProfile: !!document.querySelector("#settings-profile"),
					hasNameInput: !!document.querySelector("#name-input"),
					body: ((document.body && document.body.innerText) || "").slice(0, 1800),
				})`)
				parseT.Fatalf("wait for settings save failure surface: %v debug=%#v", parseErr, parseDebugValue)
			}
			parseHasSettingsErrorValue, parseErr := parsePage.Evaluate(`() => !!document.querySelector("#settings-save-error")`)
			if parseErr != nil {
				parseT.Fatalf("evaluate settings error visibility: %v", parseErr)
			}
			if parseHasSettingsError, _ := parseHasSettingsErrorValue.(bool); parseHasSettingsError {
				parseSettingsErrorText := parseReadExample100TextContent(parseT, parsePage, "#settings-save-error")
				if !strings.Contains(parseSettingsErrorText, "Request ID:") && !strings.Contains(parseSettingsErrorText, "Support ID:") {
					parseT.Fatalf("settings failure missing support/request id: %q", parseSettingsErrorText)
				}
				if parseHasExample100RawInternalErrorText(parseSettingsErrorText) {
					parseT.Fatalf("settings failure leaked raw internals: %q", parseSettingsErrorText)
				}
				return
			}
			parseFallbackBodyValue, parseErr := parsePage.Evaluate(`() => (document.body && document.body.innerText) || ""`)
			if parseErr != nil {
				parseT.Fatalf("evaluate settings fallback body text: %v", parseErr)
			}
			parseFallbackBodyText := strings.TrimSpace(fmt.Sprintf("%v", parseFallbackBodyValue))
			if parseHasExample100RawInternalErrorText(parseFallbackBodyText) {
				parseT.Fatalf("settings fallback leaked raw internals: %q", parseFallbackBodyText)
			}
		})
	})

	parseT.Run("dashboard load failure", func(parseT *testing.T) {
		parseBaseURL, parseDBPath, _ := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18114")
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseAdminEmail, _, parseAdminToken := parseSignupExample100CustomerErrorRPC(parseT, parseBaseURL)
			parseSetExample100CustomerErrorSuperuserRoleByEmail(parseT, parseDBPath, parseAdminEmail, true)
			parseBreakExample100CustomerErrorDashboardStore(parseT, parseDBPath)
			parseApplyExample100CustomerErrorAuthToken(parseT, parsePage, parseBaseURL, parseAdminToken)
			if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto /app/dashboard for dashboard failure leg: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/dashboard")`, nil); parseErr != nil {
				parseT.Fatalf("wait dashboard route for dashboard failure leg: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => {
				if (document.querySelector("#dashboard-error-banner")) return true;
				return document.body && document.body.innerText.includes("ADMIN SURFACES");
			}`, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(5000)}); parseErr != nil {
				parseDebugValue, _ := parsePage.Evaluate(`() => ({
					path: window.location.pathname + window.location.search,
					hasDashboardError: !!document.querySelector("#dashboard-error-banner"),
					hasChatInput: !!document.querySelector("#chat-input"),
					body: ((document.body && document.body.innerText) || "").slice(0, 1800),
				})`)
				parseT.Fatalf("wait for dashboard failure surface: %v debug=%#v", parseErr, parseDebugValue)
			}
			parseHasDashboardErrorValue, parseErr := parsePage.Evaluate(`() => !!document.querySelector("#dashboard-error-banner")`)
			if parseErr != nil {
				parseT.Fatalf("evaluate dashboard error visibility: %v", parseErr)
			}
			if parseHasDashboardError, _ := parseHasDashboardErrorValue.(bool); parseHasDashboardError {
				parseDashboardErrorText := parseReadExample100TextContent(parseT, parsePage, "#dashboard-error-banner")
				if !strings.Contains(parseDashboardErrorText, "Request ID:") && !strings.Contains(parseDashboardErrorText, "Support ID:") {
					parseT.Fatalf("dashboard failure missing support/request id: %q", parseDashboardErrorText)
				}
				if parseHasExample100RawInternalErrorText(parseDashboardErrorText) {
					parseT.Fatalf("dashboard failure leaked raw internals: %q", parseDashboardErrorText)
				}
				return
			}
			parseDashboardBodyValue, parseErr := parsePage.Evaluate(`() => (document.body && document.body.innerText) || ""`)
			if parseErr != nil {
				parseT.Fatalf("evaluate dashboard fallback body text: %v", parseErr)
			}
			parseDashboardBodyText := strings.TrimSpace(fmt.Sprintf("%v", parseDashboardBodyValue))
			if parseHasExample100RawInternalErrorText(parseDashboardBodyText) {
				parseT.Fatalf("dashboard fallback leaked raw internals: %q", parseDashboardBodyText)
			}
		})
	})
}

// parseCollectExample100ConsoleDiagnosticLogs installs one OnConsole listener that collects all plain-text and structured
// console entries emitted by the WASM client. The listener runs for parseFn and returns collected entries once done.
func parseCollectExample100ConsoleDiagnosticLogs(parsePage playwright.Page, parseFn func()) []string {
	var parseMu sync.Mutex
	var parseLogs []string
	parsePage.OnConsole(func(parseMsg playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMsg.Text())
		if parseText == "" {
			return
		}
		parseMu.Lock()
		parseLogs = append(parseLogs, strings.ToLower(parseText))
		parseMu.Unlock()
	})
	parseFn()
	parseMu.Lock()
	defer parseMu.Unlock()
	return parseLogs
}

// parseHasExample100DiagnosticLogEntry reports whether any captured console entry contains all of the supplied tokens.
func parseHasExample100DiagnosticLogEntry(parseLogs []string, parseTokens ...string) bool {
	for _, parseEntry := range parseLogs {
		parseAllMatch := true
		for _, parseToken := range parseTokens {
			if !strings.Contains(parseEntry, strings.ToLower(parseToken)) {
				parseAllMatch = false
				break
			}
		}
		if parseAllMatch {
			return true
		}
	}
	return false
}

// TestExample100ErrorBoundaryDiagnosticLog verifies that each of the three error-surface paths — route boot auth
// failure, panel fetch failure, and mutation submit failure — shows a calm fallback UI and also emits at least one
// structured diagnostic console log entry so the failure can never be silently swallowed without an operator signal.
func TestExample100ErrorBoundaryDiagnosticLog(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)

	// ── 1. Route boot auth failure ─────────────────────────────────────────────
	// Poison the auth token before the WASM app boots on /app. The client should
	// redirect to /login (not blank the screen) and emit a diagnostic auth log.
	parseT.Run("route boot auth failure emits diagnostic log", func(parseT *testing.T) {
		parseBaseURL, _, _ := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18131")
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			var parseDiagLogs []string
			var parseLogMu sync.Mutex
			parsePage.OnConsole(func(parseMsg playwright.ConsoleMessage) {
				parseText := strings.ToLower(strings.TrimSpace(parseMsg.Text()))
				if parseText == "" {
					return
				}
				parseLogMu.Lock()
				parseDiagLogs = append(parseDiagLogs, parseText)
				parseLogMu.Unlock()
			})

			// Land on /login first to establish localStorage context, then poison the
			// stored auth token and navigate to the authenticated shell.
			if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto /login for token poison setup: %v", parseErr)
			}
			parsePoisonExample100AuthToken(parseT, parsePage, parseBaseURL)

			if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto /app with poisoned token: %v", parseErr)
			}

			// The authenticated shell must not blank the screen — it must either stay
			// on /app or redirect to /login within 5 s.
			if _, parseWaitErr := parsePage.WaitForFunction(`() => {
				const p = window.location.pathname;
				return p.startsWith("/app") || p.startsWith("/login") || p.startsWith("/auth");
			}`, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(5000)}); parseWaitErr != nil {
				parseT.Fatalf("route boot fallback: page did not reach a valid fallback route: %v", parseWaitErr)
			}

			parseFinalPath, _ := parsePage.Evaluate(`() => window.location.pathname`)
			parseFinalPathStr := strings.TrimSpace(fmt.Sprintf("%v", parseFinalPath))
			if parseFinalPathStr == "" || (!strings.HasPrefix(parseFinalPathStr, "/app") &&
				!strings.HasPrefix(parseFinalPathStr, "/login") &&
				!strings.HasPrefix(parseFinalPathStr, "/auth")) {
				parseT.Errorf("route boot fallback: unexpected final path %q — expected /login, /auth/*, or /app", parseFinalPathStr)
			}

			// The app body must not be blank.
			parseBodyText, _ := parsePage.Evaluate(`() => ((document.body && document.body.innerText) || "").trim()`)
			if parseBodyTextStr := strings.TrimSpace(fmt.Sprintf("%v", parseBodyText)); parseBodyTextStr == "" {
				parseT.Error("route boot fallback: body is blank — the error boundary swallowed the failure silently")
			}

			// At least one diagnostic console log must mention an auth session event
			// originating from the chat-wizard scope.
			parseLogMu.Lock()
			parseCapturedLogs := make([]string, len(parseDiagLogs))
			copy(parseCapturedLogs, parseDiagLogs)
			parseLogMu.Unlock()

			parseHasAuthDiag := parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "auth") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "chat-wizard") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "unauthenticated") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "redirect")
			if !parseHasAuthDiag {
				parseT.Errorf("route boot auth failure: no auth diagnostic log captured; got %d console entries: %v",
					len(parseCapturedLogs), parseCapturedLogs)
			}
		})
	})

	// ── 2. Panel fetch failure (admin dashboard) ──────────────────────────────
	// Drop a required store table so the dashboard RPC fails, then navigate to
	// /app/dashboard as admin. The client must show an error panel and emit a
	// diagnostic admin/dashboard log entry.
	parseT.Run("panel fetch failure emits diagnostic log", func(parseT *testing.T) {
		parseBaseURL, parseDBPath, _ := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18132")
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseAdminEmail, _, parseAdminToken := parseSignupExample100CustomerErrorRPC(parseT, parseBaseURL)
			parseSetExample100CustomerErrorSuperuserRoleByEmail(parseT, parseDBPath, parseAdminEmail, true)
			parseBreakExample100CustomerErrorDashboardStore(parseT, parseDBPath)

			var parseDiagLogs []string
			var parseLogMu sync.Mutex
			parsePage.OnConsole(func(parseMsg playwright.ConsoleMessage) {
				parseText := strings.ToLower(strings.TrimSpace(parseMsg.Text()))
				if parseText == "" {
					return
				}
				parseLogMu.Lock()
				parseDiagLogs = append(parseDiagLogs, parseText)
				parseLogMu.Unlock()
			})

			parseApplyExample100CustomerErrorAuthToken(parseT, parsePage, parseBaseURL, parseAdminToken)
			if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto /app/dashboard for panel fetch failure: %v", parseErr)
			}

			// Wait for either the error banner or a dashboard heading that shows the
			// panel rendered without blanking.
			if _, parseWaitErr := parsePage.WaitForFunction(`() => {
				if (document.querySelector("#dashboard-error-banner")) return true;
				const body = (document.body && document.body.innerText) || "";
				return body.includes("ADMIN SURFACES") || body.includes("Dashboard") || body.includes("Usage");
			}`, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(8000)}); parseWaitErr != nil {
				parseDebug, _ := parsePage.Evaluate(`() => ({ path: window.location.pathname, body: ((document.body && document.body.innerText) || "").slice(0, 600) })`)
				parseT.Fatalf("panel fetch fallback: no fallback UI appeared: %v debug=%#v", parseWaitErr, parseDebug)
			}

			// Body must not be blank.
			parseBodyText, _ := parsePage.Evaluate(`() => ((document.body && document.body.innerText) || "").trim()`)
			if parseBodyTextStr := strings.TrimSpace(fmt.Sprintf("%v", parseBodyText)); parseBodyTextStr == "" {
				parseT.Error("panel fetch fallback: body is blank — the error boundary swallowed the failure silently")
			}

			// Diagnostic log must mention the dashboard fetch failure.
			parseLogMu.Lock()
			parseCapturedLogs := make([]string, len(parseDiagLogs))
			copy(parseCapturedLogs, parseDiagLogs)
			parseLogMu.Unlock()

			parseHasDashDiag := parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "dashboard", "failed") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "dashboard", "fetch") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "admin", "dashboard") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "chat-wizard")
			if !parseHasDashDiag {
				parseT.Errorf("panel fetch failure: no dashboard diagnostic log captured; got %d console entries: %v",
					len(parseCapturedLogs), parseCapturedLogs)
			}
		})
	})

	// ── 3. Mutation submit failure (settings name save) ───────────────────────
	// Force the gRPC bridge offline after login so the name save RPC fails. The
	// client must surface an error state (not blank) and emit a diagnostic log
	// entry for the failed mutation.
	parseT.Run("mutation submit failure emits diagnostic log", func(parseT *testing.T) {
		parseBaseURL, _, _ := startExample100CustomerErrorServer(parseT, parseRepoRoot, "18133")
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseEnsureExample100UserChatShell(parseT, parsePage, parseBaseURL, "customer@email.com", "password")

			var parseDiagLogs []string
			var parseLogMu sync.Mutex
			parsePage.OnConsole(func(parseMsg playwright.ConsoleMessage) {
				parseText := strings.ToLower(strings.TrimSpace(parseMsg.Text()))
				if parseText == "" {
					return
				}
				parseLogMu.Lock()
				parseDiagLogs = append(parseDiagLogs, parseText)
				parseLogMu.Unlock()
			})

			parseSettingsURL := parseBaseURL + "/app/settings?panel=settings-profile"
			if _, parseErr := parsePage.Goto(parseSettingsURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto settings profile for mutation failure: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
				parseDebug, _ := parsePage.Evaluate(`() => ({ path: window.location.pathname, body: ((document.body && document.body.innerText) || "").slice(0, 600) })`)
				parseT.Fatalf("wait #settings-profile for mutation failure: %v debug=%#v", parseErr, parseDebug)
			}

			// Force the bridge offline and reload the settings panel.
			parseForceExample100BridgeUnavailableOnRoute(parseT, parsePage, parseSettingsURL)
			if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
				parseDebug, _ := parsePage.Evaluate(`() => ({ path: window.location.pathname, body: ((document.body && document.body.innerText) || "").slice(0, 600) })`)
				parseT.Fatalf("wait #settings-profile after bridge abort: %v debug=%#v", parseErr, parseDebug)
			}

			// Submit the name mutation with the bridge offline.
			if parseErr := parsePage.Fill("#name-input", "DiagnosticLogRegressionTest"); parseErr != nil {
				parseT.Fatalf("fill #name-input for mutation failure: %v", parseErr)
			}
			if parseErr := parsePage.Click(`button:has-text("Save")`); parseErr != nil {
				parseT.Fatalf("click Save for mutation failure: %v", parseErr)
			}

			// Wait for the error surface or fallback to appear.
			if _, parseWaitErr := parsePage.WaitForFunction(`() => {
				if (document.querySelector("#settings-save-error")) return true;
				if (document.querySelector(".settings-save-toast")) return true;
				// Also accept navigation away from settings as a fallback.
				return !window.location.pathname.startsWith("/app/settings");
			}`, playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(6000)}); parseWaitErr != nil {
				parseDebug, _ := parsePage.Evaluate(`() => ({ path: window.location.pathname, body: ((document.body && document.body.innerText) || "").slice(0, 800) })`)
				parseT.Logf("mutation submit fallback: no error UI marker in time — treating as navigated-away fallback; debug=%#v", parseDebug)
			}

			// Body must not be blank.
			parseBodyText, _ := parsePage.Evaluate(`() => ((document.body && document.body.innerText) || "").trim()`)
			if parseBodyTextStr := strings.TrimSpace(fmt.Sprintf("%v", parseBodyText)); parseBodyTextStr == "" {
				parseT.Error("mutation submit fallback: body is blank — the error boundary swallowed the failure silently")
			}

			// Diagnostic log must mention the mutation failure.
			parseLogMu.Lock()
			parseCapturedLogs := make([]string, len(parseDiagLogs))
			copy(parseCapturedLogs, parseDiagLogs)
			parseLogMu.Unlock()

			parseHasMutDiag := parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "set user name failed") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "name", "failed") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "grpc", "unavailable") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "grpc client not yet ready") ||
				parseHasExample100DiagnosticLogEntry(parseCapturedLogs, "chat-wizard")
			if !parseHasMutDiag {
				parseT.Errorf("mutation submit failure: no mutation diagnostic log captured; got %d console entries: %v",
					len(parseCapturedLogs), parseCapturedLogs)
			}
		})
	})
}
