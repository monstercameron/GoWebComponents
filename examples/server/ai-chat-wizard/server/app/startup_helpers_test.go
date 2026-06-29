package app

import (
	"context"
	"net/http"
	"net/http/httptest"
	"net/url"
	"strings"
	"testing"

	chatpb "github.com/monstercameron/GoWebComponents/v4/examples/server/ai-chat-wizard/proto"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

func TestLoadFirstDotEnvAndRuntimeConfig(parseT *testing.T) {
	parseLoaded := []string{}
	parseChosen := parseLoadFirstDotEnv(func(parsePath string) error {
		parseLoaded = append(parseLoaded, parsePath)
		if parsePath == "../.env" {
			return nil
		}
		return url.EscapeError(parsePath)
	}, []string{".env", "../.env", "ignored.env"})
	if parseChosen != "../.env" {
		parseT.Fatalf("expected ../.env to be selected, got %q", parseChosen)
	}
	if len(parseLoaded) != 2 {
		parseT.Fatalf("expected loader to stop after first success, got %v", parseLoaded)
	}
	if parseGot := parseLoadFirstDotEnv(func(string) error { return url.EscapeError("miss") }, []string{"a.env"}); parseGot != "" {
		parseT.Fatalf("expected empty result when no env files load, got %q", parseGot)
	}

	parseConfig := parseReadServerRuntimeConfig(func(parseKey string) string {
		switch parseKey {
		case "OPENAI_API_KEY":
			return " openai-key "
		case "ANTHROPIC_API_KEY":
			return " anthropic-key "
		case "CEREBRAS_API_KEY":
			return " cerebras-key "
		case "CHAT_PROVIDER_STUBS":
			return " anthropic, cerebras "
		case "CHAT_MODEL":
			return "  "
		case "OPENAI_MODEL":
			return " gpt-5.4-mini "
		case "LISTEN_ADDR":
			return " 0.0.0.0:9999 "
		case "CHAT_DB_PATH":
			return " ./chat.db "
		case "CHAT_AUTH_SECRET":
			return " secret "
		case "CHAT_USAGE_PREMIUM_PERCENT":
			return " 7.5 "
		case "CHAT_PLATFORM_FEE_USD":
			return " 33.0 "
		default:
			return ""
		}
	})
	if parseConfig.openAIAPIKey != "openai-key" || parseConfig.anthropicAPIKey != "anthropic-key" || parseConfig.cerebrasAPIKey != "cerebras-key" {
		parseT.Fatalf("unexpected API key trimming: %+v", parseConfig)
	}
	if len(parseConfig.stubProviders) != 2 || parseConfig.stubProviders[0] != "anthropic" || parseConfig.stubProviders[1] != "cerebras" {
		parseT.Fatalf("unexpected stub provider config: %+v", parseConfig)
	}
	if parseConfig.defaultModel != "gpt-5.4-mini" || parseConfig.addr != "0.0.0.0:9999" || parseConfig.dbPath != "./chat.db" || parseConfig.authSecret != "secret" || parseConfig.usagePremiumPct != 7.5 || parseConfig.platformFeeUSD != 33 {
		parseT.Fatalf("unexpected runtime config values: %+v", parseConfig)
	}

	parseDefaultConfig := parseReadServerRuntimeConfig(func(string) string { return "" })
	if parseDefaultConfig.addr != "127.0.0.1:8095" || parseDefaultConfig.dbPath != "examples/server/ai-chat-wizard/bin/runtime/chat_history.db" || parseDefaultConfig.defaultModel != "" || parseDefaultConfig.usagePremiumPct != 5 || parseDefaultConfig.platformFeeUSD != 29 {
		parseT.Fatalf("unexpected default runtime config: %+v", parseDefaultConfig)
	}
	if len(parseDefaultConfig.stubProviders) != 0 {
		parseT.Fatalf("expected default config to omit provider stubs, got %+v", parseDefaultConfig)
	}
}

func TestNewChatServiceServerSupportsProviderStubs(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseServer := parseNewChatServiceServer("", "", "", "", store, parseNewTestLogger(), "anthropic", "cerebras")

	parseResp, parseErr := parseServer.ListModelOptions(context.Background(), &chatpb.ListModelOptionsRequest{})
	if parseErr != nil {
		parseT.Fatalf("ListModelOptions: %v", parseErr)
	}
	if len(parseResp.GetModels()) == 0 {
		parseT.Fatalf("expected stub-backed model options, got %+v", parseResp)
	}
	if parseResp.GetDefaultModel() == "" {
		parseT.Fatalf("expected default model from stub providers, got %+v", parseResp)
	}
	parseProviders := map[string]struct{}{}
	for _, parseModel := range parseResp.GetModels() {
		parseProviders[parseModel.GetCapabilities().GetProviderId()] = struct{}{}
	}
	if _, parseOk := parseProviders["anthropic"]; !parseOk {
		parseT.Fatalf("expected anthropic stub provider in model catalog, got %+v", parseResp)
	}
	if _, parseOk2 := parseProviders["cerebras"]; !parseOk2 {
		parseT.Fatalf("expected cerebras stub provider in model catalog, got %+v", parseResp)
	}
}

func TestProviderStubRuntimeSupportsCrossProviderSelection(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, store, "stub-switch@example.com")
	parseServer := parseNewChatServiceServer("", "", "", "", store, parseNewTestLogger(), "anthropic", "cerebras")
	parseCtx := parseBindAuthUser(parseServer, "peer-stub-switch", parseUser.ID, parseUser.Email)
	parseT.Cleanup(func() { parseServer.parseUnbindAuthenticatedPeer("peer-stub-switch") })

	if _, parseErr := parseServer.SetSelectedModel(parseCtx, wrapperspb.String("claude-sonnet-4-5")); parseErr != nil {
		parseT.Fatalf("SetSelectedModel anthropic stub: %v", parseErr)
	}
	parseSelectedAnthropic, parseErr2 := parseServer.GetSelectedModel(parseCtx, &emptypb.Empty{})
	if parseErr2 != nil {
		parseT.Fatalf("GetSelectedModel anthropic stub: %v", parseErr2)
	}
	if parseSelectedAnthropic.GetValue() != "claude-sonnet-4-5" {
		parseT.Fatalf("expected anthropic stub model selection, got %+v", parseSelectedAnthropic)
	}

	if _, parseErr3 := parseServer.SetSelectedModel(parseCtx, wrapperspb.String("gpt-oss-120b")); parseErr3 != nil {
		parseT.Fatalf("SetSelectedModel cerebras stub: %v", parseErr3)
	}
	parseSelectedCerebras, parseErr2 := parseServer.GetSelectedModel(parseCtx, &emptypb.Empty{})
	if parseErr2 != nil {
		parseT.Fatalf("GetSelectedModel cerebras stub: %v", parseErr2)
	}
	if parseSelectedCerebras.GetValue() != "gpt-oss-120b" {
		parseT.Fatalf("expected cerebras stub model selection, got %+v", parseSelectedCerebras)
	}
}

func TestChatServerAuthRPCs(parseT *testing.T) {
	store := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", store, parseNewTestLogger())
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, store, parseNewTestLogger())
	parseServer.authManager = parseAuth

	parseSignupResp, parseErr := parseServer.Signup(context.Background(), &chatpb.SignupRequest{
		Email:       "startup@example.com",
		Password:    "password123",
		DisplayName: "Startup",
	})
	if parseErr != nil {
		parseT.Fatalf("Signup: %v", parseErr)
	}
	if parseSignupResp.GetAuthToken() == "" || parseSignupResp.GetEmail() != "startup@example.com" || parseSignupResp.GetDisplayName() != "Startup" {
		parseT.Fatalf("unexpected signup response: %+v", parseSignupResp)
	}

	_, parseErr = parseServer.Signup(context.Background(), &chatpb.SignupRequest{
		Email:    "startup@example.com",
		Password: "password123",
	})
	if status.Code(parseErr) != codes.AlreadyExists {
		parseT.Fatalf("expected duplicate signup to return AlreadyExists, got %v", status.Code(parseErr))
	}

	_, parseErr = parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "startup@example.com",
		Password: "wrong",
	})
	if status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected invalid login to return Unauthenticated, got %v", status.Code(parseErr))
	}

	parseLoginResp, parseErr := parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "startup@example.com",
		Password: "password123",
	})
	if parseErr != nil {
		parseT.Fatalf("Login: %v", parseErr)
	}
	if parseLoginResp.GetAuthToken() == "" || parseLoginResp.GetEmail() != "startup@example.com" {
		parseT.Fatalf("unexpected login response: %+v", parseLoginResp)
	}

	parseUnauthSession, parseErr := parseServer.GetSession(context.Background(), &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSession unauthenticated: %v", parseErr)
	}
	if parseUnauthSession.GetAuthenticated() {
		parseT.Fatalf("expected empty session without metadata auth, got %+v", parseUnauthSession)
	}
	if parseUnauthSession.GetSessionStatus() != "unauthenticated" {
		parseT.Fatalf("expected unauthenticated session status, got %+v", parseUnauthSession)
	}

	parseAuthCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseLoginResp.GetAuthToken()))
	parseSession, parseErr := parseServer.GetSession(parseAuthCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSession authenticated: %v", parseErr)
	}
	if !parseSession.GetAuthenticated() || parseSession.GetEmail() != "startup@example.com" || parseSession.GetDisplayName() != "Startup" {
		parseT.Fatalf("unexpected authenticated session: %+v", parseSession)
	}
	if parseSession.GetSessionStatus() != "authenticated" || parseSession.GetSessionId() == "" || parseSession.GetTokenVersion() <= 0 || parseSession.GetExpiresAt() == "" || parseSession.GetExpiresInSeconds() <= 0 {
		parseT.Fatalf("expected typed auth bootstrap session metadata, got %+v", parseSession)
	}
	if parseRoleSummary := parseSession.GetRoleSummary(); parseRoleSummary == nil || parseRoleSummary.GetScope() != "user" || parseRoleSummary.GetCanAccessAdmin() || parseRoleSummary.GetIsSuperuser() {
		parseT.Fatalf("expected non-admin role summary, got %+v", parseRoleSummary)
	}
	parseMustEnsureWorkspaceMembership(parseT, store, parseSignupResp.GetUserId(), "ws-startup-auth-rpcs")
	parseInstrumentedLoginResp, parseErr := parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "startup@example.com",
		Password: "password123",
	})
	if parseErr != nil {
		parseT.Fatalf("Login with workspace membership: %v", parseErr)
	}
	parseAuthCtx = metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseInstrumentedLoginResp.GetAuthToken()))
	if _, parseErr = parseServer.GetSession(parseAuthCtx, &emptypb.Empty{}); parseErr != nil {
		parseT.Fatalf("GetSession after workspace membership: %v", parseErr)
	}
	parseAnalyticsRows, parseErr := store.parseListProductAnalyticsEvents(100)
	if parseErr != nil {
		parseT.Fatalf("parseListProductAnalyticsEvents: %v", parseErr)
	}
	isParseHasAuthStarted := false
	isParseHasAuthCompleted := false
	isParseHasAppBooted := false
	for _, parseAnalyticsRow := range parseAnalyticsRows {
		if parseAnalyticsRow.FunnelKey != parseFirstChatFunnelKey {
			continue
		}
		switch parseAnalyticsRow.StepKey {
		case parseFirstChatStepAuthStarted:
			isParseHasAuthStarted = true
		case parseFirstChatStepAuthCompleted:
			isParseHasAuthCompleted = true
		case parseFirstChatStepAppBooted:
			isParseHasAppBooted = true
		}
	}
	if !isParseHasAuthStarted || !isParseHasAuthCompleted || !isParseHasAppBooted {
		parseT.Fatalf("expected auth funnel steps, got rows=%+v", parseAnalyticsRows)
	}

	parseRefreshResp, parseErr := parseServer.RefreshSession(parseAuthCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("RefreshSession authenticated: %v", parseErr)
	}
	if parseRefreshResp.GetAuthToken() == "" || parseRefreshResp.GetEmail() != "startup@example.com" || parseRefreshResp.GetDisplayName() != "Startup" {
		parseT.Fatalf("unexpected refresh response: %+v", parseRefreshResp)
	}

	_, parseErr = parseServer.RefreshSession(context.Background(), &emptypb.Empty{})
	if status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected RefreshSession without auth to be unauthenticated, got %v", status.Code(parseErr))
	}

	if _, parseErr2 := parseServer.Logout(parseAuthCtx, &emptypb.Empty{}); parseErr2 != nil {
		parseT.Fatalf("Logout: %v", parseErr2)
	}
	if _, parseErr = parseServer.GetSession(parseAuthCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected revoked session to return Unauthenticated, got %v", status.Code(parseErr))
	}

	parseMalformedCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer malformed.token.value"))
	if _, parseErr = parseServer.GetSession(parseMalformedCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected malformed session token to return Unauthenticated, got %v", status.Code(parseErr))
	}

	parsePeerBoundCtx := parseBindAuthUser(parseServer, "peer-session-token-precedence", parseSignupResp.GetUserId(), parseSignupResp.GetEmail())
	parseTokenPriorityCtx := metadata.NewIncomingContext(parsePeerBoundCtx, metadata.Pairs(authMetadataKey, "Bearer malformed.token.value"))
	if _, parseErr = parseServer.GetSession(parseTokenPriorityCtx, &emptypb.Empty{}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected malformed metadata token to override peer fallback and return Unauthenticated, got %v", status.Code(parseErr))
	}
}

func TestAdminAccessInvalidatesAfterRoleDowngradeAndSessionRevocation(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseServer := parseNewChatServiceServer("", "", "", modelGPT54Mini, parseStore, parseNewTestLogger())
	parseServer.authManager = parseAuth

	parseSignupResp, parseErr := parseServer.Signup(context.Background(), &chatpb.SignupRequest{
		Email:       "admin-access-cycle@example.com",
		Password:    "password123",
		DisplayName: "Admin Access Cycle",
	})
	if parseErr != nil {
		parseT.Fatalf("Signup: %v", parseErr)
	}
	parseGrantSuperuserRole(parseT, parseStore, parseSignupResp.GetUserId())

	parseLoginResp, parseErr := parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "admin-access-cycle@example.com",
		Password: "password123",
	})
	if parseErr != nil {
		parseT.Fatalf("Login: %v", parseErr)
	}
	parseAuthCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseLoginResp.GetAuthToken()))
	if _, parseErr := parseServer.GetAdminDashboard(parseAuthCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 7,
		TopLimit:     5,
		RecentLimit:  5,
	}); parseErr != nil {
		parseT.Fatalf("GetAdminDashboard baseline superuser: %v", parseErr)
	}
	parseSessionResp, parseErr := parseServer.GetSession(parseAuthCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSession baseline superuser: %v", parseErr)
	}
	if parseRoleSummary := parseSessionResp.GetRoleSummary(); parseRoleSummary == nil || !parseRoleSummary.GetCanAccessAdmin() || !parseRoleSummary.GetIsSuperuser() {
		parseT.Fatalf("expected baseline superuser role summary, got %+v", parseRoleSummary)
	}

	if _, parseErr := parseStore.db.Exec(`DELETE FROM su_user_roles WHERE user_id = ? AND role_key = ?`, parseSignupResp.GetUserId(), "su"); parseErr != nil {
		parseT.Fatalf("delete su_user_roles for downgrade: %v", parseErr)
	}
	parseSessionResp, parseErr = parseServer.GetSession(parseAuthCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("GetSession after role downgrade: %v", parseErr)
	}
	if parseRoleSummary := parseSessionResp.GetRoleSummary(); parseRoleSummary == nil || parseRoleSummary.GetCanAccessAdmin() || parseRoleSummary.GetIsSuperuser() {
		parseT.Fatalf("expected downgraded non-admin role summary, got %+v", parseRoleSummary)
	}
	if _, parseErr := parseServer.GetAdminDashboard(parseAuthCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 7,
		TopLimit:     5,
		RecentLimit:  5,
	}); status.Code(parseErr) != codes.PermissionDenied {
		parseT.Fatalf("expected role downgrade to deny admin dashboard immediately, got %v", status.Code(parseErr))
	}

	parseGrantSuperuserRole(parseT, parseStore, parseSignupResp.GetUserId())
	parseRefreshResp, parseErr := parseServer.RefreshSession(parseAuthCtx, &emptypb.Empty{})
	if parseErr != nil {
		parseT.Fatalf("RefreshSession after role re-grant: %v", parseErr)
	}
	parseRefreshedCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseRefreshResp.GetAuthToken()))
	if _, parseErr := parseServer.GetAdminDashboard(parseRefreshedCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 7,
		TopLimit:     5,
		RecentLimit:  5,
	}); parseErr != nil {
		parseT.Fatalf("GetAdminDashboard after role re-grant: %v", parseErr)
	}

	if parseErr := parseStore.parseRevokeAuthSessionsByUser(parseSignupResp.GetUserId()); parseErr != nil {
		parseT.Fatalf("parseRevokeAuthSessionsByUser: %v", parseErr)
	}
	if _, parseErr := parseServer.GetAdminDashboard(parseRefreshedCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 7,
		TopLimit:     5,
		RecentLimit:  5,
	}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected session revocation to invalidate admin dashboard access, got %v", status.Code(parseErr))
	}

	parseReloginResp, parseErr := parseServer.Login(context.Background(), &chatpb.LoginRequest{
		Email:    "admin-access-cycle@example.com",
		Password: "password123",
	})
	if parseErr != nil {
		parseT.Fatalf("Login after revocation: %v", parseErr)
	}
	parseReloginCtx := metadata.NewIncomingContext(context.Background(), metadata.Pairs(authMetadataKey, "Bearer "+parseReloginResp.GetAuthToken()))
	if _, parseErr := parseServer.GetAdminDashboard(parseReloginCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 7,
		TopLimit:     5,
		RecentLimit:  5,
	}); parseErr != nil {
		parseT.Fatalf("GetAdminDashboard after relogin: %v", parseErr)
	}
	if _, parseErr := parseServer.Logout(parseReloginCtx, &emptypb.Empty{}); parseErr != nil {
		parseT.Fatalf("Logout: %v", parseErr)
	}
	if _, parseErr := parseServer.GetAdminDashboard(parseReloginCtx, &chatpb.GetAdminDashboardRequest{
		LookbackDays: 7,
		TopLimit:     5,
		RecentLimit:  5,
	}); status.Code(parseErr) != codes.Unauthenticated {
		parseT.Fatalf("expected logout to invalidate admin dashboard access immediately, got %v", status.Code(parseErr))
	}
}

func TestChatShellHandlerTracksFirstChatFunnelSteps(parseT *testing.T) {
	parseLandingSteps := parseCollectFirstChatFunnelStepsForPath("/")
	if len(parseLandingSteps) != 1 || parseLandingSteps[0] != parseFirstChatStepLandingViewed {
		parseT.Fatalf("landing steps mismatch: %+v", parseLandingSteps)
	}
	parsePricingSteps := parseCollectFirstChatFunnelStepsForPath("/pricing")
	if len(parsePricingSteps) != 1 || parsePricingSteps[0] != parseFirstChatStepPricingViewed {
		parseT.Fatalf("pricing steps mismatch: %+v", parsePricingSteps)
	}
	parsePlansSteps := parseCollectFirstChatFunnelStepsForPath("/plans")
	if len(parsePlansSteps) != 1 || parsePlansSteps[0] != parseFirstChatStepPricingViewed {
		parseT.Fatalf("plans steps mismatch: %+v", parsePlansSteps)
	}
	parseAuthSteps := parseCollectFirstChatFunnelStepsForPath("/login")
	if len(parseAuthSteps) != 2 || parseAuthSteps[0] != parseFirstChatStepCTAClicked || parseAuthSteps[1] != parseFirstChatStepAuthStarted {
		parseT.Fatalf("auth route steps mismatch: %+v", parseAuthSteps)
	}
}

func TestChatShellHandler(parseT *testing.T) {
	parseFileServer := http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if _, parseErr := parseW.Write([]byte("asset:" + parseR.URL.Path)); parseErr != nil {
			parseT.Fatalf("write asset response: %v", parseErr)
		}
	})
	parseHandler := parseChatShellHandler(parseFileServer)

	parseUnauthResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseUnauthResp, httptest.NewRequest(http.MethodGet, "http://example.com/", nil))
	parseUnauthBody := parseUnauthResp.Body.String()
	if parseUnauthResp.Code != http.StatusOK || !strings.Contains(parseUnauthBody, `id="boot-shell"`) || !strings.Contains(parseUnauthBody, "chat-bootstrap.js") {
		parseT.Fatalf("expected public shell content, got code=%d body=%q", parseUnauthResp.Code, parseUnauthResp.Body.String())
	}

	parseBootstrapReq := httptest.NewRequest(http.MethodGet, "http://example.com/chat-bootstrap.js", nil)
	parseBootstrapResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseBootstrapResp, parseBootstrapReq)
	if parseBootstrapResp.Code != http.StatusOK || !strings.Contains(parseBootstrapResp.Body.String(), "loadChatWasm") {
		parseT.Fatalf("expected bootstrap response, got code=%d body=%q", parseBootstrapResp.Code, parseBootstrapResp.Body.String())
	}

	parseDeepLinkReq := httptest.NewRequest(http.MethodGet, "http://example.com/thread/42", nil)
	parseDeepLinkResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseDeepLinkResp, parseDeepLinkReq)
	parseDeepLinkBody := parseDeepLinkResp.Body.String()
	if parseDeepLinkResp.Code != http.StatusOK || !strings.Contains(parseDeepLinkBody, `id="boot-shell"`) || !strings.Contains(parseDeepLinkBody, "chat-bootstrap.js") {
		parseT.Fatalf("expected thread deep-link shell content, got code=%d body=%q", parseDeepLinkResp.Code, parseDeepLinkResp.Body.String())
	}

	parseAssetReq := httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm", nil)
	parseAssetResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseAssetResp, parseAssetReq)
	if parseAssetResp.Code != http.StatusOK || parseAssetResp.Body.String() != "asset:/app/chat.wasm" {
		parseT.Fatalf("expected file server fallback, got code=%d body=%q", parseAssetResp.Code, parseAssetResp.Body.String())
	}

	parseLegacyAssetReq := httptest.NewRequest(http.MethodGet, "http://example.com/chat.wasm?br=true", nil)
	parseLegacyAssetResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseLegacyAssetResp, parseLegacyAssetReq)
	if parseLegacyAssetResp.Code != http.StatusOK || parseLegacyAssetResp.Body.String() != "asset:/app/chat.wasm" {
		parseT.Fatalf("expected legacy wasm path rewrite, got code=%d body=%q", parseLegacyAssetResp.Code, parseLegacyAssetResp.Body.String())
	}

	parseLegacyRouteReq := httptest.NewRequest(http.MethodGet, "http://example.com/login", nil)
	parseLegacyRouteResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseLegacyRouteResp, parseLegacyRouteReq)
	parseLegacyRouteBody := parseLegacyRouteResp.Body.String()
	if parseLegacyRouteResp.Code != http.StatusOK || !strings.Contains(parseLegacyRouteBody, `id="boot-shell"`) || !strings.Contains(parseLegacyRouteBody, "chat-bootstrap.js") {
		parseT.Fatalf("expected legacy auth route to resolve to the client shell, got code=%d body=%q", parseLegacyRouteResp.Code, parseLegacyRouteResp.Body.String())
	}

	parsePublicInfoReq := httptest.NewRequest(http.MethodGet, "http://example.com/about", nil)
	parsePublicInfoResp := httptest.NewRecorder()
	parseHandler.ServeHTTP(parsePublicInfoResp, parsePublicInfoReq)
	parsePublicInfoBody := parsePublicInfoResp.Body.String()
	if parsePublicInfoResp.Code != http.StatusOK || !strings.Contains(parsePublicInfoBody, `id="boot-shell"`) || !strings.Contains(parsePublicInfoBody, "chat-bootstrap.js") {
		parseT.Fatalf("expected public info route to resolve to the client shell, got code=%d body=%q", parsePublicInfoResp.Code, parsePublicInfoResp.Body.String())
	}
}
