//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"database/sql"
	"fmt"
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

const example100AdminGuardWorkspaceEmail = "workspace-admin@example.com"
const example100AdminGuardWorkspacePassword = "password123"
const example100AdminGuardNormalEmail = "admin@example.com"
const example100AdminGuardNormalPassword = "password"

type example100AdminGuardRuntimeEvidence struct {
	GetConsoleErrorCount      int
	GetPageErrorCount         int
	GetConsoleSampleText      string
	GetPageErrorSampleText    string
	parseConsoleSamples       []string
	parsePageErrorSamples     []string
	parseRuntimeEvidenceMutex sync.Mutex
}

// captureExample100AdminGuardRuntimeEvidence attaches browser listeners and collects console/page-error evidence.
func captureExample100AdminGuardRuntimeEvidence(parsePage playwright.Page) *example100AdminGuardRuntimeEvidence {
	parseEvidence := &example100AdminGuardRuntimeEvidence{
		parseConsoleSamples:   make([]string, 0, 120),
		parsePageErrorSamples: make([]string, 0, 12),
	}
	parsePage.OnConsole(func(parseMessage playwright.ConsoleMessage) {
		parseText := strings.TrimSpace(parseMessage.Text())
		if parseText == "" {
			return
		}
		parseEvidence.parseRuntimeEvidenceMutex.Lock()
		defer parseEvidence.parseRuntimeEvidenceMutex.Unlock()
		if len(parseEvidence.parseConsoleSamples) < 120 {
			parseEvidence.parseConsoleSamples = append(parseEvidence.parseConsoleSamples, fmt.Sprintf("%s: %s", parseMessage.Type(), parseText))
		}
		if parseMessage.Type() == "error" && !shouldIgnoreExample100ConsoleError(parseText) {
			parseEvidence.GetConsoleErrorCount++
		}
	})
	parsePage.OnPageError(func(parseErr error) {
		parseEvidence.parseRuntimeEvidenceMutex.Lock()
		defer parseEvidence.parseRuntimeEvidenceMutex.Unlock()
		parseEvidence.GetPageErrorCount++
		if parseErr != nil && len(parseEvidence.parsePageErrorSamples) < 12 {
			parseEvidence.parsePageErrorSamples = append(parseEvidence.parsePageErrorSamples, strings.TrimSpace(parseErr.Error()))
		}
	})
	return parseEvidence
}

// parseFinalizeRuntimeEvidence materializes compact sample strings after one browser flow finishes.
func parseFinalizeRuntimeEvidence(parseEvidence *example100AdminGuardRuntimeEvidence) {
	if parseEvidence == nil {
		return
	}
	parseEvidence.parseRuntimeEvidenceMutex.Lock()
	defer parseEvidence.parseRuntimeEvidenceMutex.Unlock()
	parseEvidence.GetConsoleSampleText = strings.Join(parseEvidence.parseConsoleSamples, " || ")
	parseEvidence.GetPageErrorSampleText = strings.Join(parseEvidence.parsePageErrorSamples, " || ")
}

// assertExample100AdminGuardNoRuntimeErrors fails one flow when actionable browser console/page errors are present.
func assertExample100AdminGuardNoRuntimeErrors(parseT *testing.T, parseLabel string, parseEvidence *example100AdminGuardRuntimeEvidence) {
	parseT.Helper()
	parseFinalizeRuntimeEvidence(parseEvidence)
	if parseEvidence.GetConsoleErrorCount != 0 {
		parseT.Fatalf("%s console errors=%d: console-samples=%q page-error-samples=%q", parseLabel, parseEvidence.GetConsoleErrorCount, parseEvidence.GetConsoleSampleText, parseEvidence.GetPageErrorSampleText)
	}
	if parseEvidence.GetPageErrorCount != 0 {
		parseT.Fatalf("%s page errors=%d: console-samples=%q page-error-samples=%q", parseLabel, parseEvidence.GetPageErrorCount, parseEvidence.GetConsoleSampleText, parseEvidence.GetPageErrorSampleText)
	}
}

// seedExample100AdminGuardWorkspaceAdminUser inserts one workspace-admin account with an empty workspace scope.
func seedExample100AdminGuardWorkspaceAdminUser(parseT *testing.T, parseDBPath string) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open admin-guard sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	parsePasswordHash, parseErr := bcrypt.GenerateFromPassword([]byte(example100AdminGuardWorkspacePassword), bcrypt.DefaultCost)
	if parseErr != nil {
		parseT.Fatalf("generate workspace-admin password hash: %v", parseErr)
	}

	parseNow := time.Now().UTC()
	parseNowRFC3339 := parseNow.Format(time.RFC3339)
	parseUserInsertResult, parseErr := parseDB.Exec(
		`INSERT INTO users (email, password_hash, created_at) VALUES (?, ?, ?)`,
		example100AdminGuardWorkspaceEmail,
		string(parsePasswordHash),
		parseNowRFC3339,
	)
	if parseErr != nil {
		parseT.Fatalf("insert workspace-admin user: %v", parseErr)
	}
	parseWorkspaceAdminUserID, parseErr := parseUserInsertResult.LastInsertId()
	if parseErr != nil {
		parseT.Fatalf("read workspace-admin user id: %v", parseErr)
	}

	if _, parseErr := parseDB.Exec(
		`INSERT INTO user_profile (user_id, name, updated_at, selected_model, selected_tone, selected_thinking_enabled, selected_thinking_effort)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		parseWorkspaceAdminUserID,
		"Workspace Admin",
		parseNow.Unix(),
		"gpt-5.4-mini",
		"balanced",
		1,
		"medium",
	); parseErr != nil {
		parseT.Fatalf("insert workspace-admin profile: %v", parseErr)
	}

	parseWorkspaceInsertResult, parseErr := parseDB.Exec(
		`INSERT INTO workspaces (workspace_key, slug, name, plan_code, status, owner_user_id, settings_json, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		"ws-admin-guard-empty",
		"ws-admin-guard-empty",
		"Admin Guard Empty Scope",
		"free",
		"active",
		parseWorkspaceAdminUserID,
		"{}",
		parseNowRFC3339,
		parseNowRFC3339,
	)
	if parseErr != nil {
		parseT.Fatalf("insert workspace-admin workspace: %v", parseErr)
	}
	parseWorkspaceID, parseErr := parseWorkspaceInsertResult.LastInsertId()
	if parseErr != nil {
		parseT.Fatalf("read workspace id: %v", parseErr)
	}

	if _, parseErr := parseDB.Exec(
		`INSERT INTO workspace_memberships (workspace_id, user_id, role_key, status, invited_by_user_id, created_at, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?)`,
		parseWorkspaceID,
		parseWorkspaceAdminUserID,
		"admin",
		"active",
		parseWorkspaceAdminUserID,
		parseNowRFC3339,
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("insert workspace-admin membership: %v", parseErr)
	}
}

// startExample100AdminGuardServer starts one seeded server with superuser and workspace-admin role fixtures for guard regression coverage.
func startExample100AdminGuardServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "admin_guard.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-admin-guard-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)

	seedExample100HappyPathDatabase(parseT, parseRepoRoot, parseDBPath)
	grantExample100AdminJourneySuperuserRole(parseT, parseDBPath)
	seedExample100AdminGuardWorkspaceAdminUser(parseT, parseDBPath)
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

// loginExample100AdminGuardUser authenticates one user in the browser and mirrors the auth token into the cookie used by deep-link guards.
func loginExample100AdminGuardUser(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseEmail string, parsePassword string) string {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /app for login (%s): %v", parseEmail, parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait for auth email input (%s): %v", parseEmail, parseErr)
	}
	if parseErr := parsePage.Fill("#auth-email-input", parseEmail); parseErr != nil {
		parseT.Fatalf("fill auth email (%s): %v", parseEmail, parseErr)
	}
	if parseErr := parsePage.Fill("#auth-password-input", parsePassword); parseErr != nil {
		parseT.Fatalf("fill auth password (%s): %v", parseEmail, parseErr)
	}
	if parseErr := parsePage.Press("#auth-password-input", "Enter"); parseErr != nil {
		parseT.Fatalf("submit auth form (%s): %v", parseEmail, parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait for chat input after login (%s): %v", parseEmail, parseErr)
	}
	parseTokenValue, parseErr := parsePage.Evaluate(fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey))
	if parseErr != nil {
		parseT.Fatalf("read auth token from localStorage (%s): %v", parseEmail, parseErr)
	}
	parseAuthToken := strings.TrimSpace(fmt.Sprintf("%v", parseTokenValue))
	if strings.EqualFold(parseAuthToken, "<nil>") {
		parseAuthToken = ""
	}
	if parseAuthToken == "" {
		parseT.Fatalf("missing auth token after login (%s)", parseEmail)
	}
	if parseErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{
		{
			Name:  example100AdminJourneyAuthCookieName,
			Value: parseAuthToken,
			URL:   playwright.String(parseBaseURL),
		},
	}); parseErr != nil {
		parseT.Fatalf("set auth cookie after login (%s): %v", parseEmail, parseErr)
	}
	return parseAuthToken
}

// hasExample100PermissionDeniedError checks whether one RPC error summary carries a permission-denied signal.
func hasExample100PermissionDeniedError(parseErrText string) bool {
	parseNormalized := strings.ToLower(strings.TrimSpace(parseErrText))
	return strings.Contains(parseNormalized, "permissiondenied") ||
		strings.Contains(parseNormalized, "permission denied") ||
		strings.Contains(parseNormalized, "admin dashboard role required")
}

// TestExample100AdminRoleGuardsAndDeepLinks adds focused guard/deep-link regressions for superuser vs workspace-admin routes and empty/denied dashboard outcomes.
func TestExample100AdminRoleGuardsAndDeepLinks(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100AdminGuardServer(parseT, parseRepoRoot, "18108")

	parseT.Run("superuser route guard allows deep links", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
			parseAuthToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

			if _, parseErr := parsePage.Goto(parseBaseURL+"/app/admin/users", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto /app/admin/users superuser deep link: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/admin/users")`, nil); parseErr != nil {
				parseT.Fatalf("wait for superuser deep link route: %v", parseErr)
			}

			parseSnapshot := captureExample100AdminJourneyRPCSnapshot(parseBaseURL, parseAuthToken)
			if !parseSnapshot.HasAdminDashboardLoaded || !parseSnapshot.HasAdminUsersSliceLoaded || !parseSnapshot.HasAdminConvsSliceLoaded {
				parseT.Fatalf("superuser admin snapshot incomplete: %+v", parseSnapshot)
			}
			if parseSnapshot.GetRecentConversationsCount <= 0 {
				parseT.Fatalf("superuser expected non-empty conversations, got %+v", parseSnapshot)
			}
			assertExample100AdminGuardNoRuntimeErrors(parseT, "superuser deep link flow", parseEvidence)
		})
	})

	parseT.Run("workspace-admin deep links keep scoped empty states", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
			parseAuthToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminGuardWorkspaceEmail, example100AdminGuardWorkspacePassword)

			if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard/usage", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto /app/dashboard/usage workspace-admin deep link: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/dashboard/usage")`, nil); parseErr != nil {
				parseT.Fatalf("wait for workspace-admin deep link route: %v", parseErr)
			}

			parseSnapshot := captureExample100AdminJourneyRPCSnapshot(parseBaseURL, parseAuthToken)
			if !parseSnapshot.HasAdminDashboardLoaded || !parseSnapshot.HasAdminUsersSliceLoaded {
				parseT.Fatalf("workspace-admin dashboard/users snapshot incomplete: %+v", parseSnapshot)
			}
			if parseSnapshot.HasAdminConvsSliceLoaded || parseSnapshot.GetRecentConversationsCount != 0 {
				parseT.Fatalf("workspace-admin expected empty scoped conversations, got %+v", parseSnapshot)
			}
			if strings.TrimSpace(parseSnapshot.GetDashboardErrorText) != "" || strings.TrimSpace(parseSnapshot.GetUsersErrorText) != "" || strings.TrimSpace(parseSnapshot.GetConversationsErrorText) != "" {
				parseT.Fatalf("workspace-admin expected empty-state without RPC errors, got %+v", parseSnapshot)
			}
			assertExample100AdminGuardNoRuntimeErrors(parseT, "workspace-admin deep link flow", parseEvidence)
		})
	})

	parseT.Run("normal user deep links fail closed", func(parseT *testing.T) {
		withExamplesPage(parseT, func(parsePage playwright.Page) {
			parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
			parseAuthToken := loginExample100AdminGuardUser(parseT, parsePage, parseBaseURL, example100AdminGuardNormalEmail, example100AdminGuardNormalPassword)

			if _, parseErr := parsePage.Goto(parseBaseURL+"/app/admin/users", playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto /app/admin/users normal-user deep link: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname === "/app"`, nil); parseErr != nil {
				parseT.Fatalf("wait for normal-user redirect to /app: %v", parseErr)
			}

			parseSnapshot := captureExample100AdminJourneyRPCSnapshot(parseBaseURL, parseAuthToken)
			if parseSnapshot.HasAdminDashboardLoaded || parseSnapshot.HasAdminUsersSliceLoaded || parseSnapshot.HasAdminConvsSliceLoaded {
				parseT.Fatalf("normal-user expected denied admin snapshot, got %+v", parseSnapshot)
			}
			if !hasExample100PermissionDeniedError(parseSnapshot.GetDashboardErrorText) {
				parseT.Fatalf("normal-user expected permission denied dashboard error, got %+v", parseSnapshot)
			}
			assertExample100AdminGuardNoRuntimeErrors(parseT, "normal-user denied flow", parseEvidence)
		})
	})
}
