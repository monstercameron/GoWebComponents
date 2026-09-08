//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"database/sql"
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	_ "github.com/ncruces/go-sqlite3/driver"
	_ "github.com/ncruces/go-sqlite3/embed"
	playwright "github.com/mxschmitt/playwright-go"
	"golang.org/x/crypto/bcrypt"
)

type example100FullStackDemoArtifact struct {
	GetHomeStatus          int
	GetPricingStatus       int
	GetSignupStatus        int
	HasPricingFAQ          bool
	HasSignupFields        bool
	HasFirstPromptVisible  bool
	HasFirstStreamComplete bool
	HasThreadRoute         bool
	GetThreadPath          string
	HasSettingsProfile     bool
	HasSettingsBilling     bool
	HasDashboardEntry      bool
	HasAdminMutation       bool
	GetFeatureFlagKey      string
}

// formatExample100FullStackDemoSummary formats one full-stack sweep artifact for concise logs.
func formatExample100FullStackDemoSummary(parseArtifact example100FullStackDemoArtifact) string {
	return fmt.Sprintf(
		"home=%d pricing=%d signup=%d pricing-faq=%t signup-fields=%t first-prompt=%t first-stream=%t thread-route=%t thread-path=%q settings-profile=%t settings-billing=%t dashboard-entry=%t admin-mutation=%t feature-flag=%q",
		parseArtifact.GetHomeStatus,
		parseArtifact.GetPricingStatus,
		parseArtifact.GetSignupStatus,
		parseArtifact.HasPricingFAQ,
		parseArtifact.HasSignupFields,
		parseArtifact.HasFirstPromptVisible,
		parseArtifact.HasFirstStreamComplete,
		parseArtifact.HasThreadRoute,
		parseArtifact.GetThreadPath,
		parseArtifact.HasSettingsProfile,
		parseArtifact.HasSettingsBilling,
		parseArtifact.HasDashboardEntry,
		parseArtifact.HasAdminMutation,
		parseArtifact.GetFeatureFlagKey,
	)
}

// parseLoginExample100FullStackDemoUser signs in through whichever auth-entry variant is active and returns the auth token.
func parseLoginExample100FullStackDemoUser(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseEmail string, parsePassword string) string {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("goto /login for full-stack demo login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
		parseT.Fatalf("wait #auth-email-input for login: %v", parseErr)
	}
	parseSubmitExample100AuthCredentials(parseT, parsePage, parseEmail, parsePassword)
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait #chat-input after login: %v", parseErr)
	}

	parseTokenValue, parseErr := parsePage.Evaluate(fmt.Sprintf(`() => (window.localStorage && window.localStorage.getItem(%q)) || ""`, example100AdminJourneyAuthTokenStorageKey))
	if parseErr != nil {
		parseT.Fatalf("read auth token from localStorage after login: %v", parseErr)
	}
	parseAuthToken := strings.TrimSpace(fmt.Sprintf("%v", parseTokenValue))
	if strings.EqualFold(parseAuthToken, "<nil>") || parseAuthToken == "" {
		parseT.Fatalf("missing auth token after login")
	}
	if parseErr := parsePage.Context().AddCookies([]playwright.OptionalCookie{
		{
			Name:  example100AdminJourneyAuthCookieName,
			Value: parseAuthToken,
			URL:   playwright.String(parseBaseURL),
		},
	}); parseErr != nil {
		parseT.Fatalf("set auth cookie after login: %v", parseErr)
	}
	return parseAuthToken
}

// parseCaptureExample100FullStackDemoArtifact executes one end-to-end public->customer->admin sweep in a single browser session.
func parseCaptureExample100FullStackDemoArtifact(parseT *testing.T, parsePage playwright.Page, parseBaseURL string) example100FullStackDemoArtifact {
	parseT.Helper()
	parseArtifact := example100FullStackDemoArtifact{}

	parseHomeResp, parseErr := parsePage.Goto(parseBaseURL+"/", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
	if parseErr != nil {
		parseT.Fatalf("goto /: %v", parseErr)
	}
	if parseHomeResp == nil {
		parseT.Fatalf("goto / returned nil response")
	}
	parseArtifact.GetHomeStatus = parseHomeResp.Status()
	if _, parseErr := parsePage.WaitForFunction(
		`() => document.body && String(document.body.innerText || "").trim().length > 0`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait / body content: %v", parseErr)
	}

	parsePricingResp, parseErr := parsePage.Goto(parseBaseURL+"/pricing#faq", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
	if parseErr != nil {
		parseT.Fatalf("goto /pricing#faq: %v", parseErr)
	}
	if parsePricingResp == nil {
		parseT.Fatalf("goto /pricing#faq returned nil response")
	}
	parseArtifact.GetPricingStatus = parsePricingResp.Status()
	parseArtifact.HasPricingFAQ = parseArtifact.GetPricingStatus > 0 && parseArtifact.GetPricingStatus < 400

	parseSignupResp, parseErr := parsePage.Goto(parseBaseURL+"/signup", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded})
	if parseErr != nil {
		parseT.Fatalf("goto /signup: %v", parseErr)
	}
	if parseSignupResp == nil {
		parseT.Fatalf("goto /signup returned nil response")
	}
	parseArtifact.GetSignupStatus = parseSignupResp.Status()
	parseArtifact.HasSignupFields = parseArtifact.GetSignupStatus > 0 && parseArtifact.GetSignupStatus < 400

	_ = parseLoginExample100FullStackDemoUser(parseT, parsePage, parseBaseURL, "customer@email.com", "password")

	parsePrompt := fmt.Sprintf("full-stack-sweep-%d", time.Now().UTC().UnixNano()%1_000_000)
	if parseErr := parsePage.Click(`button:has-text("New chat")`); parseErr != nil {
		parseT.Fatalf("click New chat: %v", parseErr)
	}
	parseHasPromptAfterSend := false
	for parseAttempt := 0; parseAttempt < 8; parseAttempt++ {
		if parseErr := parsePage.Fill("#chat-input", parsePrompt); parseErr != nil {
			parseT.Fatalf("fill #chat-input (attempt=%d): %v", parseAttempt+1, parseErr)
		}
		if parseErr := parsePage.Press("#chat-input", "Enter"); parseErr != nil {
			if parseClickErr := parsePage.Click("#send-btn"); parseClickErr != nil {
				parseT.Fatalf("trigger send (attempt=%d): press=%v click=%v", parseAttempt+1, parseErr, parseClickErr)
			}
		}
		if _, parseErr := parsePage.WaitForFunction(
			fmt.Sprintf(`() => (document.body && document.body.innerText.includes(%q)) || !!document.querySelector("#streaming-assistant-bubble")`, parsePrompt),
			playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(1800)},
		); parseErr == nil {
			parseHasPromptAfterSend = true
			break
		}
		parsePage.WaitForTimeout(650)
	}
	if !parseHasPromptAfterSend {
		parseDebugValue, _ := parsePage.Evaluate(`() => ({
			path: window.location.pathname + window.location.search,
			hasChatInput: !!document.querySelector("#chat-input"),
			hasSendButton: !!document.querySelector("#send-btn"),
			hasStreamingBubble: !!document.querySelector("#streaming-assistant-bubble"),
			body: ((document.body && document.body.innerText) || "").slice(0, 1800),
		})`)
		parseT.Fatalf("wait prompt visibility after send failed debug=%#v", parseDebugValue)
	}
	parseArtifact.HasFirstPromptVisible = true
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("streaming-assistant-bubble")`, nil); parseErr != nil {
		parseT.Fatalf("wait streaming completion after send: %v", parseErr)
	}
	parseArtifact.HasFirstStreamComplete = true
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(5000)},
	); parseErr == nil {
		parseArtifact.HasThreadRoute = true
	}
	parseThreadPathValue, parseErr := parsePage.Evaluate(`() => window.location.pathname`)
	if parseErr != nil {
		parseT.Fatalf("read thread path: %v", parseErr)
	}
	parseArtifact.GetThreadPath = strings.TrimSpace(fmt.Sprintf("%v", parseThreadPathValue))

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-profile", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("goto settings profile panel: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#settings-profile"); parseErr != nil {
		parseT.Fatalf("wait #settings-profile: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait #chat-input on settings profile: %v", parseErr)
	}
	parseArtifact.HasSettingsProfile = true

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/settings?panel=settings-billing", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("goto settings billing panel: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#settings-billing"); parseErr != nil {
		parseT.Fatalf("wait #settings-billing: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait #chat-input on settings billing: %v", parseErr)
	}
	parseArtifact.HasSettingsBilling = true

	parseClearExample100AdminMutationAuthState(parseT, parsePage)
	parseSuperuserToken := parseLoginExample100FullStackDemoUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("goto /app/dashboard: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/dashboard")`, nil); parseErr != nil {
		parseT.Fatalf("wait /app/dashboard route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => document.body && document.body.innerText.toLowerCase().includes("dashboard")`, nil); parseErr != nil {
		parseT.Fatalf("wait dashboard copy on /app/dashboard: %v", parseErr)
	}
	parseArtifact.HasDashboardEntry = true

	parseMutationConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
	defer parseMutationConn.Close()
	parseMutationClient := chatpb.NewChatServiceClient(parseMutationConn)
	parseArtifact.GetFeatureFlagKey = fmt.Sprintf("full-stack-demo-flag-%d", time.Now().UTC().UnixNano())
	parseFlagCtx, parseFlagCancel := parseBuildExample100AdminMutationCallContext()
	parseFlagResp, parseErr := parseMutationClient.SetAdminFeatureFlag(parseFlagCtx, &chatpb.SetAdminFeatureFlagRequest{
		FlagKey:        parseArtifact.GetFeatureFlagKey,
		IsEnabled:      true,
		RolloutPercent: 100,
		AudienceJson:   "{\"workspace\":\"all\"}",
		PayloadJson:    "{\"source\":\"full-stack-demo-sweep\"}",
		Confirm:        true,
		Reason:         "playwright full-stack demo sweep regression: set feature flag",
	})
	parseFlagCancel()
	if parseErr != nil {
		parseT.Fatalf("SetAdminFeatureFlag: %v", parseErr)
	}
	if parseFlagResp.GetFeatureFlag() == nil || parseFlagResp.GetFeatureFlag().GetFlagKey() != parseArtifact.GetFeatureFlagKey {
		parseT.Fatalf("SetAdminFeatureFlag returned invalid row: %+v", parseFlagResp)
	}
	parseArtifact.HasAdminMutation = true

	parseDashboardRoute := fmt.Sprintf("/app/dashboard?tab=ops-settings&lookback_days=30&search=%s", url.QueryEscape(parseArtifact.GetFeatureFlagKey))
	if _, parseErr := parsePage.Goto(parseBaseURL+parseDashboardRoute, playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("goto mutation dashboard route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		fmt.Sprintf(`() => window.location.pathname.startsWith("/app/dashboard") && window.location.search.includes(%q)`, parseArtifact.GetFeatureFlagKey),
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait mutation dashboard route search state: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => document.body && document.body.innerText.toLowerCase().includes("dashboard")`, nil); parseErr != nil {
		parseT.Fatalf("wait dashboard copy after mutation dashboard route: %v", parseErr)
	}

	return parseArtifact
}

// seedExample100FullStackDemoCustomerUser ensures the seeded customer login account exists with the expected password.
func seedExample100FullStackDemoCustomerUser(parseT *testing.T, parseDBPath string) {
	parseT.Helper()
	parseDsn := "file:" + parseDBPath + "?_pragma=journal_mode(WAL)&_pragma=busy_timeout(5000)"
	parseDB, parseErr := sql.Open("sqlite3", parseDsn)
	if parseErr != nil {
		parseT.Fatalf("open full-stack sqlite db: %v", parseErr)
	}
	defer parseDB.Close()
	parseDB.SetMaxOpenConns(1)

	parsePasswordHash, parseErr := bcrypt.GenerateFromPassword([]byte("password"), bcrypt.DefaultCost)
	if parseErr != nil {
		parseT.Fatalf("generate bcrypt hash for customer login: %v", parseErr)
	}
	parseNowRFC3339 := time.Now().UTC().Format(time.RFC3339)
	if _, parseErr := parseDB.Exec(
		`INSERT INTO users (email, password_hash, created_at)
		 VALUES (?, ?, ?)
		 ON CONFLICT(email) DO UPDATE SET password_hash = excluded.password_hash`,
		"customer@email.com",
		string(parsePasswordHash),
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("upsert customer user: %v", parseErr)
	}
	var parseCustomerUserID int64
	if parseErr := parseDB.QueryRow(`SELECT id FROM users WHERE email = ? COLLATE NOCASE`, "customer@email.com").Scan(&parseCustomerUserID); parseErr != nil {
		parseT.Fatalf("resolve customer user id: %v", parseErr)
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
		parseCustomerUserID,
		parseNowRFC3339,
	); parseErr != nil {
		parseT.Fatalf("upsert customer user access state: %v", parseErr)
	}
}

// startExample100FullStackDemoServer starts one fixture-backed server tuned for full-stack demo sweep coverage.
func startExample100FullStackDemoServer(parseT *testing.T, parseRepoRoot string, parsePort string) string {
	parseT.Helper()
	parseRuntimeDir := parseT.TempDir()
	parseDBPath := filepath.Join(parseRuntimeDir, "full_stack_demo.db")
	parseLogDir := filepath.Join(parseRuntimeDir, "logs")
	parseBinaryPath := filepath.Join(parseRuntimeDir, "example100-full-stack-demo-server")
	if runtime.GOOS == "windows" {
		parseBinaryPath += ".exe"
	}
	parseAddress := "127.0.0.1:" + strings.TrimSpace(parsePort)

	copyExample100CustomerErrorDatabase(parseT, parseRepoRoot, parseDBPath)
	seedExample100CustomerErrorAdminUser(parseT, parseDBPath)
	seedExample100FullStackDemoCustomerUser(parseT, parseDBPath)
	grantExample100AdminJourneySuperuserRole(parseT, parseDBPath)
	buildExample100HappyPathServerBinary(parseT, parseRepoRoot, parseBinaryPath)

	parseStop := startExamplesCommandWithEnv(
		parseT,
		parseRepoRoot,
		[]string{
			"LISTEN_ADDR=" + parseAddress,
			"CHAT_DB_PATH=" + parseDBPath,
			"CHAT_LOG_DIR=" + parseLogDir,
			"CHAT_STUB_PROVIDERS=all",
		},
		parseBinaryPath,
	)
	parseT.Cleanup(parseStop)

	parseBaseURL := "http://" + parseAddress
	waitForHealthyExamplesURL(parseT, parseBaseURL+"/healthz", 120*time.Second)
	return parseBaseURL
}

// TestExample100FullStackDemoRegressionSweep validates public routes, auth, first chat, settings, billing, dashboard entry, and one admin mutation flow with no runtime dead ends.
func TestExample100FullStackDemoRegressionSweep(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100FullStackDemoServer(parseT, parseRepoRoot, "18131")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseArtifact := parseCaptureExample100FullStackDemoArtifact(parseT, parsePage, parseBaseURL)
		parseT.Logf("example 100 full-stack demo sweep: %s", formatExample100FullStackDemoSummary(parseArtifact))

		if parseArtifact.GetHomeStatus >= 400 {
			parseT.Fatalf("/home failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		if parseArtifact.GetPricingStatus >= 400 || !parseArtifact.HasPricingFAQ {
			parseT.Fatalf("/pricing failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		if parseArtifact.GetSignupStatus >= 400 || !parseArtifact.HasSignupFields {
			parseT.Fatalf("/signup failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		if !parseArtifact.HasFirstPromptVisible || !parseArtifact.HasFirstStreamComplete {
			parseT.Fatalf("first chat path failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		if !parseArtifact.HasSettingsProfile || !parseArtifact.HasSettingsBilling {
			parseT.Fatalf("settings or billing path failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		if !parseArtifact.HasDashboardEntry {
			parseT.Fatalf("dashboard entry failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		if !parseArtifact.HasAdminMutation {
			parseT.Fatalf("admin mutation flow failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		assertExample100AdminGuardNoRuntimeErrors(parseT, "full-stack demo sweep regression flow", parseEvidence)
	})
}
