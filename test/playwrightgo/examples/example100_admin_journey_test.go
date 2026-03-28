//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"context"
	"database/sql"
	"fmt"
	"path/filepath"
	"runtime"
	"strings"
	"sync"
	"testing"
	"time"

	"github.com/monstercameron/GoGRPCBridge/pkg/grpctunnel"
	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	playwright "github.com/playwright-community/playwright-go"
	"google.golang.org/grpc"
	"google.golang.org/grpc/metadata"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
)

const example100AdminJourneyAuthTokenStorageKey = "chat-wizard:auth-token"
const example100AdminJourneyAuthCookieName = "chat_wizard_auth"
const example100AdminJourneyLoginEmail = "demo@example.com"
const example100AdminJourneyLoginPassword = "password123"

type example100AdminJourneyRPCSnapshot struct {
	HasSessionResolved          bool
	HasAdminDashboardLoaded     bool
	HasAdminUsersSliceLoaded    bool
	HasAdminConvsSliceLoaded    bool
	GetSessionEmail             string
	GetRecentUsersCount         int
	GetRecentConversationsCount int
	GetDashboardErrorText       string
	GetUsersErrorText           string
	GetConversationsErrorText   string
}

type example100AdminJourneyArtifact struct {
	GetLandingStatus            int
	GetAuthStatus               int
	HasLandingBootMount         bool
	HasAdminLogin               bool
	HasAdminEntryVisible        bool
	HasDashboardHomeLoaded      bool
	HasDashboardSliceNavigation bool
	HasBackNavigation           bool
	HasRefreshPersistence       bool
	HasRoleResolution           bool
	HasDashboardRPCLoaded       bool
	HasDashboardSliceRPCLoaded  bool
	GetRoutePath                string
	GetConsoleErrorCount        int
	GetPageErrorCount           int
	GetConsoleSampleText        string
	GetPageErrorSampleText      string
	GetRPCSummary               string
}

// grantExample100AdminJourneySuperuserRole grants the seeded admin-journey account one superuser role in the test database.
func grantExample100AdminJourneySuperuserRole(parseT *testing.T, parseDBPath string) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open admin journey sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	var parseAdminUserID int64
	if parseErr := parseDB.QueryRow(`SELECT id FROM users WHERE email = ? COLLATE NOCASE`, example100AdminJourneyLoginEmail).Scan(&parseAdminUserID); parseErr != nil {
		parseT.Fatalf("resolve seeded admin-journey user id: %v", parseErr)
	}
	parseCreatedAt := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseDB.Exec(
		`INSERT INTO su_roles (role_key, label, description, is_system, is_enabled, created_at, updated_at)
		 VALUES ('su', 'Superuser', 'Admin journey browser regression role', 1, 1, ?, ?)
		 ON CONFLICT(role_key) DO UPDATE SET
			label = excluded.label,
			description = excluded.description,
			is_system = 1,
			is_enabled = 1,
			updated_at = excluded.updated_at`,
		parseCreatedAt,
		parseCreatedAt,
	); parseErr != nil {
		parseT.Fatalf("upsert seeded superuser role: %v", parseErr)
	}
	if _, parseErr := parseDB.Exec(
		`INSERT INTO su_role_permissions (role_key, permission_key, permission_value, updated_at)
		 VALUES ('su', 'control_plane.*', 'allow', ?)
		 ON CONFLICT(role_key, permission_key) DO UPDATE SET
			permission_value = excluded.permission_value,
			updated_at = excluded.updated_at`,
		parseCreatedAt,
	); parseErr != nil {
		parseT.Fatalf("upsert seeded superuser permission: %v", parseErr)
	}
	if _, parseErr := parseDB.Exec(
		`INSERT INTO su_user_roles (user_id, role_key, assigned_by_user_id, created_at) VALUES (?, 'su', ?, ?)
		 ON CONFLICT(user_id, role_key) DO UPDATE SET assigned_by_user_id = excluded.assigned_by_user_id`,
		parseAdminUserID,
		parseAdminUserID,
		parseCreatedAt,
	); parseErr != nil {
		parseT.Fatalf("grant seeded admin superuser role: %v", parseErr)
	}
}

// startExample100AdminJourneyServer starts one seeded server with admin@example.com granted superuser scope for dashboard RPC checks.
func startExample100AdminJourneyServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "admin_journey.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-admin-journey-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)

	seedExample100HappyPathDatabase(parseT, parseRepoRoot, parseDBPath)
	grantExample100AdminJourneySuperuserRole(parseT, parseDBPath)
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

// buildExample100AdminJourneyTunnelURL converts one HTTP base URL into the matching websocket tunnel URL.
func buildExample100AdminJourneyTunnelURL(parseBaseURL string) string {
	parseTrimmed := strings.TrimSpace(parseBaseURL)
	switch {
	case strings.HasPrefix(parseTrimmed, "https://"):
		return "wss://" + strings.TrimPrefix(parseTrimmed, "https://") + "/socket"
	case strings.HasPrefix(parseTrimmed, "http://"):
		return "ws://" + strings.TrimPrefix(parseTrimmed, "http://") + "/socket"
	default:
		return strings.TrimRight(parseTrimmed, "/") + "/socket"
	}
}

// captureExample100AdminJourneyRPCSnapshot fetches admin dashboard home + slices over the tunnel using one browser-auth token.
func captureExample100AdminJourneyRPCSnapshot(parseBaseURL, parseAuthToken string) example100AdminJourneyRPCSnapshot {
	parseSnapshot := example100AdminJourneyRPCSnapshot{}
	parseToken := strings.TrimSpace(parseAuthToken)
	if parseToken == "" {
		parseSnapshot.GetDashboardErrorText = "missing auth token"
		return parseSnapshot
	}

	parseDialCtx, parseCancelDial := context.WithTimeout(context.Background(), 12*time.Second)
	defer parseCancelDial()
	parseDialOptions := grpctunnel.ApplyTunnelInsecureCredentials([]grpc.DialOption{
		grpc.WithUnaryInterceptor(func(parseCtx context.Context, parseMethod string, parseReq interface{}, parseReply interface{}, parseConn *grpc.ClientConn, parseInvoker grpc.UnaryInvoker, parseOpts ...grpc.CallOption) error {
			parseAuthCtx := metadata.AppendToOutgoingContext(parseCtx, "authorization", "Bearer "+parseToken)
			return parseInvoker(parseAuthCtx, parseMethod, parseReq, parseReply, parseConn, parseOpts...)
		}),
		grpc.WithBlock(),
	})
	parseConn, parseErr := grpctunnel.BuildTunnelConn(parseDialCtx, grpctunnel.TunnelConfig{
		Target:      buildExample100AdminJourneyTunnelURL(parseBaseURL),
		GRPCOptions: parseDialOptions,
	})
	if parseErr != nil {
		parseSnapshot.GetDashboardErrorText = parseErr.Error()
		return parseSnapshot
	}
	defer parseConn.Close()

	parseClient := chatpb.NewChatServiceClient(parseConn)
	parseCallCtx, parseCancelCall := context.WithTimeout(context.Background(), 10*time.Second)
	defer parseCancelCall()

	parseSessionResp, parseSessionErr := parseClient.GetSession(parseCallCtx, &emptypb.Empty{})
	if parseSessionErr == nil && parseSessionResp.GetAuthenticated() {
		parseSnapshot.HasSessionResolved = true
		parseSnapshot.GetSessionEmail = strings.TrimSpace(parseSessionResp.GetEmail())
	}

	parseDashboardResp, parseDashboardErr := parseClient.GetAdminDashboard(parseCallCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 30,
		TopLimit:     8,
		RecentLimit:  8,
	})
	if parseDashboardErr != nil {
		parseSnapshot.GetDashboardErrorText = parseDashboardErr.Error()
	} else {
		parseSnapshot.HasAdminDashboardLoaded = true
		parseSnapshot.GetRecentUsersCount = len(parseDashboardResp.GetRecentUsers())
		parseSnapshot.GetRecentConversationsCount = len(parseDashboardResp.GetRecentConversations())
	}

	parseUsersResp, parseUsersErr := parseClient.ListAdminUsers(parseCallCtx, &chatpb.ListAdminUsersRequest{Limit: 12})
	if parseUsersErr != nil {
		parseSnapshot.GetUsersErrorText = parseUsersErr.Error()
	} else if len(parseUsersResp.GetUsers()) >= 1 {
		parseSnapshot.HasAdminUsersSliceLoaded = true
	}

	parseConversationsResp, parseConversationsErr := parseClient.ListAdminConversations(parseCallCtx, &chatpb.ListAdminConversationsRequest{Limit: 12})
	if parseConversationsErr != nil {
		parseSnapshot.GetConversationsErrorText = parseConversationsErr.Error()
	} else if len(parseConversationsResp.GetConversations()) >= 1 {
		parseSnapshot.HasAdminConvsSliceLoaded = true
	}

	return parseSnapshot
}

// formatExample100AdminJourneySummary formats one admin journey artifact for concise logs.
func formatExample100AdminJourneySummary(parseArtifact example100AdminJourneyArtifact) string {
	return fmt.Sprintf(
		"landing=%d auth=%d landing-boot=%t login=%t entry=%t dashboard-home=%t slices=%t back=%t refresh=%t role=%t dashboard-rpc=%t slice-rpc=%t route=%q console-errors=%d page-errors=%d rpc=%q console-samples=%q page-error-samples=%q",
		parseArtifact.GetLandingStatus,
		parseArtifact.GetAuthStatus,
		parseArtifact.HasLandingBootMount,
		parseArtifact.HasAdminLogin,
		parseArtifact.HasAdminEntryVisible,
		parseArtifact.HasDashboardHomeLoaded,
		parseArtifact.HasDashboardSliceNavigation,
		parseArtifact.HasBackNavigation,
		parseArtifact.HasRefreshPersistence,
		parseArtifact.HasRoleResolution,
		parseArtifact.HasDashboardRPCLoaded,
		parseArtifact.HasDashboardSliceRPCLoaded,
		parseArtifact.GetRoutePath,
		parseArtifact.GetConsoleErrorCount,
		parseArtifact.GetPageErrorCount,
		parseArtifact.GetRPCSummary,
		parseArtifact.GetConsoleSampleText,
		parseArtifact.GetPageErrorSampleText,
	)
}

// shouldIgnoreExample100ConsoleError reports whether one known benign browser console error should be excluded from failure counts.
func shouldIgnoreExample100ConsoleError(parseText string) bool {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseText))
	return strings.Contains(parseNormalized, "wasm failed to load") &&
		strings.Contains(parseNormalized, "webassembly compilation aborted") &&
		strings.Contains(parseNormalized, "response body loading was aborted")
}

// captureExample100AdminJourneyArtifact executes homepage -> admin login -> admin deep-link dashboard -> slice/back/refresh checks in one browser flow.
func captureExample100AdminJourneyArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100AdminJourneyArtifact {
	parseT.Helper()
	parseArtifact := example100AdminJourneyArtifact{}
	parseConsoleSamples := make([]string, 0, 140)
	parsePageErrorSamples := make([]string, 0, 12)
	var parseMu sync.Mutex

	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseMu.Lock()
		defer parseMu.Unlock()
		if len(parseConsoleSamples) < 140 {
			parseConsoleSamples = append(parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
		}
		if parseMessage.Type() == "error" {
			if shouldIgnoreExample100ConsoleError(parseText) {
				return
			}
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

	parseLandingResp, parseErr := parsePage.Goto(parseBaseURL+"/", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto homepage route: %v", parseErr)
	}
	if parseLandingResp == nil {
		parseT.Fatalf("homepage route returned nil response")
	}
	parseArtifact.GetLandingStatus = parseLandingResp.Status()
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			const app = document.getElementById("app");
			const boot = document.getElementById("boot-shell");
			const mounted = !!app && ((app.children && app.children.length > 0) || String(app.textContent || "").trim().length > 0);
			return mounted && !boot;
		}`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for homepage mount and boot-shell removal: %v", parseErr)
	}
	parseArtifact.HasLandingBootMount = true

	parseAuthResp, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	})
	if parseErr != nil {
		parseT.Fatalf("goto /app for admin login: %v", parseErr)
	}
	if parseAuthResp == nil {
		parseT.Fatalf("/app auth route returned nil response")
	}
	parseArtifact.GetAuthStatus = parseAuthResp.Status()
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for auth email input: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", example100AdminJourneyLoginEmail); parseErr != nil {
		parseT.Fatalf("fill admin-journey auth email: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", example100AdminJourneyLoginPassword); parseErr != nil {
		parseT.Fatalf("fill admin-journey auth password: %v", parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit admin auth form: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after admin login: %v", parseErr)
	}
	parseArtifact.HasAdminLogin = true

	parseTokenValue, parseErr := parsePage.Evaluate(fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey))
	if parseErr != nil {
		parseT.Fatalf("read auth token from localStorage after admin login: %v", parseErr)
	}
	parseAuthToken := strings.TrimSpace(fmt.Sprintf("%v", parseTokenValue))
	if strings.EqualFold(parseAuthToken, "<nil>") {
		parseAuthToken = ""
	}
	if parseAuthToken == "" {
		parseT.Fatalf("admin login token missing from localStorage")
	}
	if parseErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{
		{
			Name:  example100AdminJourneyAuthCookieName,
			Value: parseAuthToken,
			URL:   playwright.String(parseBaseURL),
		},
	}); parseErr != nil {
		parseT.Fatalf("set auth cookie for admin deep-link guard: %v", parseErr)
	}

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app/dashboard admin home route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input on /app/dashboard: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname === "/app/dashboard"`, nil); parseErr != nil {
		parseT.Fatalf("wait for /app/dashboard path after admin deep link: %v", parseErr)
	}
	parseArtifact.HasAdminEntryVisible = true
	parseArtifact.HasDashboardHomeLoaded = true

	parseRPCSnapshot := captureExample100AdminJourneyRPCSnapshot(parseBaseURL, parseAuthToken)
	parseArtifact.HasDashboardRPCLoaded = parseRPCSnapshot.HasAdminDashboardLoaded
	parseArtifact.HasDashboardSliceRPCLoaded = parseRPCSnapshot.HasAdminUsersSliceLoaded && parseRPCSnapshot.HasAdminConvsSliceLoaded
	parseArtifact.HasRoleResolution = parseRPCSnapshot.HasSessionResolved &&
		strings.EqualFold(strings.TrimSpace(parseRPCSnapshot.GetSessionEmail), example100AdminJourneyLoginEmail) &&
		parseRPCSnapshot.HasAdminDashboardLoaded
	parseArtifact.GetRPCSummary = fmt.Sprintf(
		"session=%t email=%q dashboard=%t users=%t conversations=%t dashboard-err=%q users-err=%q conversations-err=%q",
		parseRPCSnapshot.HasSessionResolved,
		parseRPCSnapshot.GetSessionEmail,
		parseRPCSnapshot.HasAdminDashboardLoaded,
		parseRPCSnapshot.HasAdminUsersSliceLoaded,
		parseRPCSnapshot.HasAdminConvsSliceLoaded,
		parseRPCSnapshot.GetDashboardErrorText,
		parseRPCSnapshot.GetUsersErrorText,
		parseRPCSnapshot.GetConversationsErrorText,
	)

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard/usage", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app/dashboard/usage admin slice route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/dashboard/usage")`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for /app/dashboard/usage admin slice route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input on /app/dashboard/usage: %v", parseErr)
	}
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/admin/users", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app/admin/users admin slice route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/admin/users")`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for /app/admin/users admin slice route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input on /app/admin/users: %v", parseErr)
	}
	parseArtifact.HasDashboardSliceNavigation = true

	if _, parseErr := parsePage.GoBack(playwright.PageGoBackOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("go back to prior admin slice route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/dashboard/usage")`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for /app/dashboard/usage route after back nav: %v", parseErr)
	}
	parseArtifact.HasBackNavigation = true

	if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("reload /app/dashboard/usage admin slice route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/dashboard/usage")`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait for /app/dashboard/usage route after reload: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after reloading /app/dashboard/usage: %v", parseErr)
	}
	parseArtifact.HasRefreshPersistence = true

	parseRouteValue, parseErr := parsePage.Evaluate(`() => window.location.pathname + window.location.search`)
	if parseErr == nil {
		parseArtifact.GetRoutePath = strings.TrimSpace(fmt.Sprintf("%v", parseRouteValue))
	}

	parseMu.Lock()
	parseArtifact.GetConsoleSampleText = strings.Join(parseConsoleSamples, " || ")
	parseArtifact.GetPageErrorSampleText = strings.Join(parsePageErrorSamples, " || ")
	parseMu.Unlock()
	return parseArtifact
}

// TestExample100AdminJourneyRegression verifies homepage->admin-login->admin deep links with role-scoped dashboard RPC access and stable slice back/refresh behavior.
func TestExample100AdminJourneyRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100AdminJourneyServer(parseT, parseRepoRoot, "18107")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseArtifact := captureExample100AdminJourneyArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 admin journey: %s", formatExample100AdminJourneySummary(parseArtifact))

		if parseArtifact.GetLandingStatus >= 400 || !parseArtifact.HasLandingBootMount {
			parseT.Fatalf("admin journey landing step failed: %s", formatExample100AdminJourneySummary(parseArtifact))
		}
		if parseArtifact.GetAuthStatus >= 400 || !parseArtifact.HasAdminLogin {
			parseT.Fatalf("admin journey login step failed: %s", formatExample100AdminJourneySummary(parseArtifact))
		}
		if !parseArtifact.HasAdminEntryVisible || !parseArtifact.HasDashboardHomeLoaded {
			parseT.Fatalf("admin journey entry/home step failed: %s", formatExample100AdminJourneySummary(parseArtifact))
		}
		if !parseArtifact.HasRoleResolution || !parseArtifact.HasDashboardRPCLoaded {
			parseT.Fatalf("admin journey role/dashboard-rpc step failed: %s", formatExample100AdminJourneySummary(parseArtifact))
		}
		if !parseArtifact.HasDashboardSliceNavigation || !parseArtifact.HasDashboardSliceRPCLoaded {
			parseT.Fatalf("admin journey slice step failed: %s", formatExample100AdminJourneySummary(parseArtifact))
		}
		if !parseArtifact.HasBackNavigation || !parseArtifact.HasRefreshPersistence {
			parseT.Fatalf("admin journey back/refresh step failed: %s", formatExample100AdminJourneySummary(parseArtifact))
		}
		if parseArtifact.GetConsoleErrorCount != 0 {
			parseT.Fatalf("console errors=%d: %s", parseArtifact.GetConsoleErrorCount, formatExample100AdminJourneySummary(parseArtifact))
		}
		if parseArtifact.GetPageErrorCount != 0 {
			parseT.Fatalf("page errors=%d: %s", parseArtifact.GetPageErrorCount, formatExample100AdminJourneySummary(parseArtifact))
		}
	})
}
