//go:build playwrightgo
// +build playwrightgo

package playwrightgoexamples_test

import (
	"fmt"
	"net/url"
	"runtime"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v6/examples/server/ai-chat-wizard/proto"
	playwright "github.com/mxschmitt/playwright-go"
)

type example100LazyNamespaceEntrySpec struct {
	GetNamespace          string
	HasExpectedMessages   bool
	HasAllowEmptyMessages bool
}

type example100LazyNamespaceRouteSpec struct {
	GetRoutePath     string
	GetReadySelector string
	GetReadyJS       string
	GetNamespaces    []example100LazyNamespaceEntrySpec
}

type example100LazyNamespaceCacheState struct {
	GetVersion     string
	GetContentHash string
}

// parseLoginExample100LazyNamespaceUser authenticates through /login and mirrors the auth token into the deep-link auth cookie.
func parseLoginExample100LazyNamespaceUser(parseT *testing.T, parsePage playwright.Page, parseBaseURL string, parseEmail string, parsePassword string) string {
	parseT.Helper()
	if _, parseErr := parsePage.Goto(parseBaseURL+"/login", playwright.PageGotoOptions{
		WaitUntil: playwright.WaitUntilStateDomcontentloaded,
	}); parseErr != nil {
		parseT.Fatalf("goto /login for lazy-namespace login (%s): %v", parseEmail, parseErr)
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

// parseWaitExample100LazyNamespaceRouteReady waits for one route-specific readiness signal.
func parseWaitExample100LazyNamespaceRouteReady(parseT *testing.T, parsePage playwright.Page, parseRouteSpec example100LazyNamespaceRouteSpec) {
	parseT.Helper()
	parseSelector := strings.TrimSpace(parseRouteSpec.GetReadySelector)
	if parseSelector != "" {
		if _, parseErr := parsePage.WaitForSelector(parseSelector); parseErr != nil {
			parseT.Fatalf("wait for route selector %q on %s: %v", parseSelector, parseRouteSpec.GetRoutePath, parseErr)
		}
		return
	}
	parseReadyJS := strings.TrimSpace(parseRouteSpec.GetReadyJS)
	if parseReadyJS == "" {
		parseT.Fatalf("route %s is missing readiness selector/js", parseRouteSpec.GetRoutePath)
	}
	if _, parseErr := parsePage.WaitForFunction(parseReadyJS, nil); parseErr != nil {
		parseT.Fatalf("wait for route js readiness on %s: %v", parseRouteSpec.GetRoutePath, parseErr)
	}
}

// parseAssertExample100LazyNamespaceRouteState verifies route path and query are preserved exactly.
func parseAssertExample100LazyNamespaceRouteState(parseT *testing.T, parsePage playwright.Page, parseExpectedRoutePath string, parseStage string) {
	parseT.Helper()
	parseExpectedURL, parseErr := url.Parse(strings.TrimSpace(parseExpectedRoutePath))
	if parseErr != nil {
		parseT.Fatalf("%s parse expected route %q: %v", parseStage, parseExpectedRoutePath, parseErr)
	}
	parseExpectedState := strings.TrimSpace(parseExpectedURL.Path) + strings.TrimSpace(parseExpectedURL.RawQuery)
	if parseExpectedURL.RawQuery != "" {
		parseExpectedState = strings.TrimSpace(parseExpectedURL.Path) + "?" + strings.TrimSpace(parseExpectedURL.RawQuery)
	}
	parseCurrentRouteValue, parseErr := parsePage.Evaluate(`() => String(window.location.pathname || "") + String(window.location.search || "")`)
	if parseErr != nil {
		parseT.Fatalf("%s evaluate current route state: %v", parseStage, parseErr)
	}
	parseCurrentRouteState := strings.TrimSpace(fmt.Sprintf("%v", parseCurrentRouteValue))
	if parseCurrentRouteState != parseExpectedState {
		parseT.Fatalf("%s route state mismatch: got=%q want=%q", parseStage, parseCurrentRouteState, parseExpectedState)
	}
}

// parseFetchExample100CatalogNamespace fetches one catalog namespace and validates cache behavior.
func parseFetchExample100CatalogNamespace(
	parseT *testing.T,
	parseClient chatpb.ChatServiceClient,
	parseNamespaceSpec example100LazyNamespaceEntrySpec,
	parseLocale string,
	parseCacheByNamespace map[string]example100LazyNamespaceCacheState,
	parseRequireNotModified bool,
	parseStage string,
) {
	parseT.Helper()
	parseNamespace := strings.TrimSpace(parseNamespaceSpec.GetNamespace)
	if parseNamespace == "" {
		parseT.Fatalf("%s namespace is required", parseStage)
	}
	parseCacheState, hasParseCacheState := parseCacheByNamespace[parseNamespace]
	parseKnownVersion := ""
	parseKnownHash := ""
	if hasParseCacheState {
		parseKnownVersion = strings.TrimSpace(parseCacheState.GetVersion)
		parseKnownHash = strings.TrimSpace(parseCacheState.GetContentHash)
	}
	parseCallCtx, parseCallCancel := parseBuildExample100AdminMutationCallContext()
	parseResp, parseErr := parseClient.GetCatalogNamespace(parseCallCtx, &chatpb.GetCatalogNamespaceRequest{
		Namespace:    parseNamespace,
		Locale:       strings.TrimSpace(parseLocale),
		KnownVersion: parseKnownVersion,
		KnownHash:    parseKnownHash,
	})
	parseCallCancel()
	if parseErr != nil {
		parseT.Fatalf("%s GetCatalogNamespace(%q): %v", parseStage, parseNamespace, parseErr)
	}
	if parseRequireNotModified {
		if !parseResp.GetIsNotModified() {
			parseT.Fatalf("%s expected not-modified cache hit for namespace %q, got cache miss", parseStage, parseNamespace)
		}
		return
	}
	if parseResp.GetIsNotModified() {
		parseT.Fatalf("%s expected cache miss for namespace %q, got not-modified", parseStage, parseNamespace)
	}
	parseCatalog := parseResp.GetCatalog()
	if parseCatalog == nil {
		parseT.Fatalf("%s expected catalog payload for namespace %q", parseStage, parseNamespace)
	}
	parseVersion := strings.TrimSpace(parseCatalog.GetVersion())
	parseContentHash := strings.TrimSpace(parseCatalog.GetContentHash())
	if parseVersion == "" || parseContentHash == "" {
		parseT.Fatalf("%s namespace %q returned blank cache metadata (version=%q hash=%q)", parseStage, parseNamespace, parseVersion, parseContentHash)
	}
	parseMessageCount := len(parseCatalog.GetMessages())
	if parseNamespaceSpec.HasExpectedMessages && parseMessageCount == 0 {
		parseT.Fatalf("%s namespace %q expected message rows, got none", parseStage, parseNamespace)
	}
	if !parseNamespaceSpec.HasExpectedMessages && !parseNamespaceSpec.HasAllowEmptyMessages && parseMessageCount == 0 {
		parseT.Fatalf("%s namespace %q unexpectedly returned empty message rows", parseStage, parseNamespace)
	}
	parseCacheByNamespace[parseNamespace] = example100LazyNamespaceCacheState{
		GetVersion:     parseVersion,
		GetContentHash: parseContentHash,
	}
}

// TestExample100LazyNamespaceLoadingRegression verifies route/query stability plus one-fetch cache behavior for settings, billing, and dashboard catalog namespaces.
func TestExample100LazyNamespaceLoadingRegression(parseT *testing.T) {
	_, parseFile, _, _ := runtime.Caller(0)
	parseRepoRoot := examplesRepoRootFromFile(parseFile)
	parseBaseURL := startExample100AdminJourneyServer(parseT, parseRepoRoot, "18121")

	withExamplesPage(parseT, func(parsePage playwright.Page) {
		parseEvidence := captureExample100AdminGuardRuntimeEvidence(parsePage)
		parseSuperuserToken := parseLoginExample100LazyNamespaceUser(parseT, parsePage, parseBaseURL, example100AdminJourneyLoginEmail, example100AdminJourneyLoginPassword)
		parseConn := parseOpenExample100AdminMutationConn(parseT, parseBaseURL, parseSuperuserToken)
		defer parseConn.Close()
		parseClient := chatpb.NewChatServiceClient(parseConn)

		parseRouteSpecs := []example100LazyNamespaceRouteSpec{
			{
				GetRoutePath:     "/app/settings?panel=settings-profile&lookback_days=7",
				GetReadySelector: "#settings-profile",
				GetNamespaces: []example100LazyNamespaceEntrySpec{
					{GetNamespace: "chat", HasExpectedMessages: true},
					{GetNamespace: "settings", HasAllowEmptyMessages: true},
				},
			},
			{
				GetRoutePath:     "/app/settings?panel=settings-billing&lookback_days=30",
				GetReadySelector: "#settings-billing",
				GetNamespaces: []example100LazyNamespaceEntrySpec{
					{GetNamespace: "chat", HasExpectedMessages: true},
					{GetNamespace: "billing", HasAllowEmptyMessages: true},
				},
			},
			{
				GetRoutePath: "/app/dashboard/providers?lookback_days=30&tab=providers",
				GetReadyJS: `() => {
					const routeOK = window.location.pathname === "/app/dashboard/providers"
					const text = (document.body && document.body.innerText) || ""
					return routeOK && text.includes("Admin Dashboard") && text.includes("Providers")
				}`,
				GetNamespaces: []example100LazyNamespaceEntrySpec{
					{GetNamespace: "chat", HasExpectedMessages: true},
					{GetNamespace: "dashboard.providers", HasAllowEmptyMessages: true},
				},
			},
		}

		parseCacheByNamespace := map[string]example100LazyNamespaceCacheState{}
		for _, parseRouteSpec := range parseRouteSpecs {
			parseRouteURL := parseBaseURL + parseRouteSpec.GetRoutePath
			if _, parseErr := parsePage.Goto(parseRouteURL, playwright.PageGotoOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("goto route %s: %v", parseRouteSpec.GetRoutePath, parseErr)
			}
			parseWaitExample100LazyNamespaceRouteReady(parseT, parsePage, parseRouteSpec)
			parseAssertExample100LazyNamespaceRouteState(parseT, parsePage, parseRouteSpec.GetRoutePath, "after-open "+parseRouteSpec.GetRoutePath)

			for _, parseNamespaceSpec := range parseRouteSpec.GetNamespaces {
				parseNamespace := strings.TrimSpace(parseNamespaceSpec.GetNamespace)
				_, hasParseCacheState := parseCacheByNamespace[parseNamespace]
				parseFetchExample100CatalogNamespace(
					parseT,
					parseClient,
					parseNamespaceSpec,
					"en",
					parseCacheByNamespace,
					hasParseCacheState,
					"open "+parseRouteSpec.GetRoutePath,
				)
			}

			if _, parseErr := parsePage.Reload(playwright.PageReloadOptions{
				WaitUntil: playwright.WaitUntilStateDomcontentloaded,
			}); parseErr != nil {
				parseT.Fatalf("reload route %s: %v", parseRouteSpec.GetRoutePath, parseErr)
			}
			parseWaitExample100LazyNamespaceRouteReady(parseT, parsePage, parseRouteSpec)
			parseAssertExample100LazyNamespaceRouteState(parseT, parsePage, parseRouteSpec.GetRoutePath, "after-reload "+parseRouteSpec.GetRoutePath)

			for _, parseNamespaceSpec := range parseRouteSpec.GetNamespaces {
				parseFetchExample100CatalogNamespace(
					parseT,
					parseClient,
					parseNamespaceSpec,
					"en",
					parseCacheByNamespace,
					true,
					"reload "+parseRouteSpec.GetRoutePath,
				)
			}
		}

		assertExample100AdminGuardNoRuntimeErrors(parseT, "lazy-namespace loading regression", parseEvidence)
	})
}
