//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"net/url"
	"path/filepath"
	"runtime"
	"strings"
	"testing"
	"time"

	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	playwright "github.com/playwright-community/playwright-go"
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
	if _, parseErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("goto /app for full-stack demo login: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(
		`() => {
			if (document.querySelector("#chat-input") || document.querySelector("#auth-email-input")) {
				return true;
			}
			const bodyText = String((document.body && document.body.innerText) || "").toLowerCase();
			return bodyText.includes("log in");
		}`,
		nil,
	); parseErr != nil {
		parseT.Fatalf("wait login entry surface: %v", parseErr)
	}
	parseHasAuthEmailValue, parseErr := parsePage.Evaluate(`() => !!document.querySelector("#auth-email-input")`)
	if parseErr != nil {
		parseT.Fatalf("evaluate auth-email visibility before login click: %v", parseErr)
	}
	parseHasChatInputValue, parseErr := parsePage.Evaluate(`() => !!document.querySelector("#chat-input")`)
	if parseErr != nil {
		parseT.Fatalf("evaluate chat-input visibility before login click: %v", parseErr)
	}
	if parseHasAuthEmail, _ := parseHasAuthEmailValue.(bool); !parseHasAuthEmail {
		if parseHasChatInput, _ := parseHasChatInputValue.(bool); !parseHasChatInput {
			if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
				parseT.Fatalf("goto /login from marketing auth entry: %v", parseErr)
			}
			if _, parseErr := parsePage.WaitForFunction(`() => !!document.querySelector("#chat-input") || !!document.querySelector("#auth-email-input")`, nil); parseErr != nil {
				parseDebugValue, _ := parsePage.Evaluate(`() => ({
					path: window.location.pathname + window.location.search,
					title: document.title || "",
					hasChatInput: !!document.querySelector("#chat-input"),
					hasAuthEmail: !!document.querySelector("#auth-email-input"),
					body: ((document.body && document.body.innerText) || "").slice(0, 1800),
				})`)
				parseT.Fatalf("wait auth-or-chat after /login fallback: %v debug=%#v", parseErr, parseDebugValue)
			}
		}
	}

	parseHasChatInputValue, parseErr = parsePage.Evaluate(`() => !!document.querySelector("#chat-input")`)
	if parseErr != nil {
		parseT.Fatalf("evaluate chat-input visibility before auth submit: %v", parseErr)
	}
	if parseHasChatInput, _ := parseHasChatInputValue.(bool); !parseHasChatInput {
		if _, parseErr := parsePage.WaitForSelector("#auth-email-input"); parseErr != nil {
			parseDebugValue, _ := parsePage.Evaluate(`() => ({
				path: window.location.pathname + window.location.search,
				title: document.title || "",
				hasChatInput: !!document.querySelector("#chat-input"),
				hasAuthEmail: !!document.querySelector("#auth-email-input"),
				hasLandingLogin: !!document.querySelector("#landing-login-link"),
				hasLandingSignup: !!document.querySelector("#landing-signup-link"),
				body: ((document.body && document.body.innerText) || "").slice(0, 1800),
			})`)
			parseT.Fatalf("wait #auth-email-input for login: %v debug=%#v", parseErr, parseDebugValue)
		}
		parseSubmitExample100AuthCredentials(parseT, parsePage, parseEmail, parsePassword)
		if _, parseErr := parsePage.WaitForFunction(`() => !!document.querySelector("#chat-input") || !!document.querySelector("#auth-error-banner")`, nil); parseErr != nil {
			parseT.Fatalf("wait chat-or-auth-error after login submit: %v", parseErr)
		}
		parseHasAuthErrorValue, parseErr := parsePage.Evaluate(`() => !!document.querySelector("#auth-error-banner")`)
		if parseErr != nil {
			parseT.Fatalf("evaluate auth error visibility after login submit: %v", parseErr)
		}
		if parseHasAuthError, _ := parseHasAuthErrorValue.(bool); parseHasAuthError {
			parseAuthErrorValue, _ := parsePage.TextContent("#auth-error-banner")
			parseT.Fatalf("login failed with auth error: %s", strings.TrimSpace(parseAuthErrorValue))
		}
	}
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

	parseSuperuserToken := parseLoginExample100FullStackDemoUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)

	parsePrompt := fmt.Sprintf("full-stack-sweep-%d", time.Now().UTC().UnixNano()%1_000_000)
	if parseErr := parsePage.Click(`button:has-text("New chat")`); parseErr != nil {
		parseT.Fatalf("click New chat: %v", parseErr)
	}
	if parseErr := parsePage.Fill("#chat-input", parsePrompt); parseErr != nil {
		parseT.Fatalf("fill #chat-input: %v", parseErr)
	}
	if parseErr := parsePage.Click("#send-btn"); parseErr != nil {
		parseT.Fatalf("click #send-btn: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(fmt.Sprintf(`() => document.body && document.body.innerText.includes(%q)`, parsePrompt), nil); parseErr != nil {
		parseT.Fatalf("wait prompt visibility after send: %v", parseErr)
	}
	parseArtifact.HasFirstPromptVisible = true
	if _, parseErr := parsePage.WaitForFunction(`() => !document.getElementById("streaming-assistant-bubble")`, nil); parseErr != nil {
		parseT.Fatalf("wait streaming completion after send: %v", parseErr)
	}
	parseArtifact.HasFirstStreamComplete = true
	if _, parseErr := parsePage.WaitForFunction(
		`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
		playwright.PageWaitForFunctionOptions{Timeout: playwright.Float(12000)},
	); parseErr != nil {
		if _, parseFallbackErr := parsePage.Goto(parseBaseURL+"/app", playwright.PageGotoOptions{
			WaitUntil: playwright.WaitUntilStateDomcontentloaded,
		}); parseFallbackErr != nil {
			parseT.Fatalf("fallback goto /app for thread-route normalization: %v", parseFallbackErr)
		}
		parseConversationSelector := fmt.Sprintf(`#conversation-list button:has-text(%q)`, parsePrompt)
		if _, parseFallbackErr := parsePage.WaitForSelector(parseConversationSelector); parseFallbackErr != nil {
			parseDebugValue, _ := parsePage.Evaluate(`() => ({
				path: window.location.pathname + window.location.search,
				hasConversationList: !!document.querySelector("#conversation-list"),
				conversationRows: Array.from(document.querySelectorAll("#conversation-list button")).map((button) => String(button.textContent || "").trim()).slice(0, 12),
				hasStreamingBubble: !!document.querySelector("#streaming-assistant-bubble"),
				body: ((document.body && document.body.innerText) || "").slice(0, 1800),
			})`)
			parseT.Fatalf("wait canonical thread route after send: %v (fallback row wait failed: %v) debug=%#v", parseErr, parseFallbackErr, parseDebugValue)
		}
		if parseFallbackErr := parsePage.Click(parseConversationSelector); parseFallbackErr != nil {
			parseT.Fatalf("fallback click for thread-route normalization failed: %v", parseFallbackErr)
		}
		if _, parseFallbackErr := parsePage.WaitForFunction(
			`() => window.location.pathname.startsWith("/app/thread/") && window.location.pathname.length > "/app/thread/".length`,
			nil,
		); parseFallbackErr != nil {
			parseT.Fatalf("wait canonical thread route after send: %v (fallback normalization failed: %v)", parseErr, parseFallbackErr)
		}
	}
	parseArtifact.HasThreadRoute = true
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

	if _, parseErr := parsePage.Goto(parseBaseURL+"/app/dashboard", playwright.PageGotoOptions{WaitUntil: playwright.WaitUntilStateDomcontentloaded}); parseErr != nil {
		parseT.Fatalf("goto /app/dashboard: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForFunction(`() => window.location.pathname.startsWith("/app/dashboard")`, nil); parseErr != nil {
		parseT.Fatalf("wait /app/dashboard route: %v", parseErr)
	}
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait #chat-input on /app/dashboard: %v", parseErr)
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
	if _, parseErr := parsePage.WaitForSelector("#chat-input"); parseErr != nil {
		parseT.Fatalf("wait #chat-input after mutation dashboard route: %v", parseErr)
	}

	return parseArtifact
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
	grantExample100AdminJourneySuperuserRole(parseT, parseDBPath)
	buildExample100CustomerErrorClientArtifacts(parseT, parseRepoRoot)
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
		if !parseArtifact.HasFirstPromptVisible || !parseArtifact.HasFirstStreamComplete || !parseArtifact.HasThreadRoute {
			parseT.Fatalf("first chat path failed: %s", formatExample100FullStackDemoSummary(parseArtifact))
		}
		if !strings.HasPrefix(parseArtifact.GetThreadPath, "/app/thread/") {
			parseT.Fatalf("thread route missing after first chat: %s", formatExample100FullStackDemoSummary(parseArtifact))
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
