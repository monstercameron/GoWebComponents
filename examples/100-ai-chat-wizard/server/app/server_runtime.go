package app

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"os"
	"os/signal"
	"path/filepath"
	"strconv"
	"strings"
	"syscall"
	"time"

	"github.com/joho/godotenv"
	chatpb "github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/proto"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/status"
)

// parseLoadFirstDotEnv loads the first available .env file from the candidate list.

func parseLoadFirstDotEnv(parseLoad func(string) error, parseCandidates []string) string {
	for _, parseCandidate := range parseCandidates {
		if parseErr := parseLoad(parseCandidate); parseErr == nil {
			return parseCandidate
		}
	}
	return ""
}

type serverRuntimeConfig struct {
	openAIAPIKey                    string
	anthropicAPIKey                 string
	cerebrasAPIKey                  string
	stubProviders                   []string
	defaultModel                    string
	addr                            string
	dbPath                          string
	authSecret                      string
	environment                     string
	allowInsecureAuthSecretFallback bool
	usagePremiumPct                 float64
	platformFeeUSD                  float64
}

func parseReadServerRuntimeConfig(parseGetenv func(string) string) serverRuntimeConfig {
	parseDefaultModel := strings.TrimSpace(parseGetenv("CHAT_MODEL"))
	if parseDefaultModel == "" {
		parseDefaultModel = strings.TrimSpace(parseGetenv("OPENAI_MODEL"))
	}
	parseAddr := strings.TrimSpace(parseGetenv("LISTEN_ADDR"))
	if parseAddr == "" {
		parseAddr = "127.0.0.1:8095"
	}
	parseDbPath := strings.TrimSpace(parseGetenv("CHAT_DB_PATH"))
	if parseDbPath == "" {
		parseDbPath = "examples/100-ai-chat-wizard/bin/runtime/chat_history.db"
	}
	parseUsagePremiumPct := parseUsagePremiumPercent(parseGetenv("CHAT_USAGE_PREMIUM_PERCENT"), 5.0)
	parsePlatformFeeValue := parsePlatformFeeUSD(parseGetenv("CHAT_PLATFORM_FEE_USD"), 29.0)
	return serverRuntimeConfig{
		openAIAPIKey:                    strings.TrimSpace(parseGetenv("OPENAI_API_KEY")),
		anthropicAPIKey:                 strings.TrimSpace(parseGetenv("ANTHROPIC_API_KEY")),
		cerebrasAPIKey:                  strings.TrimSpace(parseGetenv("CEREBRAS_API_KEY")),
		stubProviders:                   parseSplitAndTrim(parseGetenv("CHAT_PROVIDER_STUBS")),
		defaultModel:                    parseDefaultModel,
		addr:                            parseAddr,
		dbPath:                          parseDbPath,
		authSecret:                      strings.TrimSpace(parseGetenv("CHAT_AUTH_SECRET")),
		environment:                     parseResolveRuntimeEnvironment(parseGetenv("CHAT_ENV")),
		allowInsecureAuthSecretFallback: parseResolveBooleanEnv(parseGetenv("CHAT_ALLOW_INSECURE_AUTH_FALLBACK")),
		usagePremiumPct:                 parseUsagePremiumPct,
		platformFeeUSD:                  parsePlatformFeeValue,
	}
}

func parseSplitAndTrim(parseValue string) []string {
	if strings.TrimSpace(parseValue) == "" {
		return nil
	}
	parseParts := strings.Split(parseValue, ",")
	parseTrimmed := make([]string, 0, len(parseParts))
	for _, parsePart := range parseParts {
		parseResolved := strings.TrimSpace(parsePart)
		if parseResolved != "" {
			parseTrimmed = append(parseTrimmed, parseResolved)
		}
	}
	return parseTrimmed
}

func parseUsagePremiumPercent(parseRawValue string, parseFallback float64) float64 {
	parseTrimmed := strings.TrimSpace(parseRawValue)
	if parseTrimmed == "" {
		return parseFallback
	}
	parseParsed, parseErr := strconv.ParseFloat(parseTrimmed, 64)
	if parseErr != nil || parseParsed < 0 {
		return parseFallback
	}
	if parseParsed > 1000 {
		return 1000
	}
	return parseParsed
}

func parsePlatformFeeUSD(parseRawValue string, parseFallback float64) float64 {
	parseTrimmed := strings.TrimSpace(parseRawValue)
	if parseTrimmed == "" {
		return parseFallback
	}
	parseParsed, parseErr := strconv.ParseFloat(parseTrimmed, 64)
	if parseErr != nil || parseParsed < 0 {
		return parseFallback
	}
	if parseParsed > 1_000_000 {
		return 1_000_000
	}
	return parseParsed
}

// parseResolveRuntimeEnvironment normalizes one chat runtime environment label.
func parseResolveRuntimeEnvironment(parseRawEnvironment string) string {
	parseEnvironment := strings.ToLower(strings.TrimSpace(parseRawEnvironment))
	if parseEnvironment == "" {
		return "development"
	}
	return parseEnvironment
}

// parseResolveBooleanEnv parses one environment toggle using permissive truthy literals.
func parseResolveBooleanEnv(parseRawValue string) bool {
	switch strings.ToLower(strings.TrimSpace(parseRawValue)) {
	case "1", "true", "yes", "on":
		return true
	default:
		return false
	}
}

// parseValidateAuthSecretConfig validates runtime auth-secret requirements before server startup.
func parseValidateAuthSecretConfig(parseConfig serverRuntimeConfig) error {
	if strings.TrimSpace(parseConfig.authSecret) != "" {
		return nil
	}
	if parseConfig.allowInsecureAuthSecretFallback {
		return nil
	}
	if parseConfig.environment == "production" {
		return errors.New("CHAT_AUTH_SECRET is required when CHAT_ENV=production")
	}
	return nil
}

// parseChatShellHandler serves the chat shell and static assets without one chat-service context.
func parseChatShellHandler(parseFileServer http.Handler) http.Handler {
	return parseChatShellHandlerForServer(nil, parseFileServer)
}

// parseChatShellHandlerForServer serves the chat shell and static assets with optional admin deep-link authz checks.
func parseChatShellHandlerForServer(parseS *chatServer, parseFileServer http.Handler) http.Handler {
	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		switch parseR.URL.Path {
		case "/chat-bootstrap.js", "/app/chat-bootstrap.js":
			parseServeChatBootstrapJS(parseW, parseR)
			return
		}
		if parseS != nil {
			parseS.parseTrackFirstChatFunnelShellRequest(parseR)
		}
		if parseHandleAdminDashboardDeepLinkGuard(parseW, parseR, parseS) {
			return
		}
		parseRewritten := parseRewriteLegacyClientAssetRequest(parseR)
		if parseRewritten != parseR {
			parseFileServer.ServeHTTP(parseW, parseRewritten)
			return
		}
		if shouldServeClientShell(parseR.URL.Path) {
			parseServeChatShell(parseW, parseR)
			return
		}
		parseFileServer.ServeHTTP(parseW, parseR)
	})
}

// parseHandleAdminDashboardDeepLinkGuard fail-closes unauthorized admin deep links by redirecting to the app root.
func parseHandleAdminDashboardDeepLinkGuard(parseW http.ResponseWriter, parseR *http.Request, parseS *chatServer) bool {
	if parseS == nil || parseR == nil {
		return false
	}
	if !parseIsAdminDashboardDeepLinkRequest(parseR) {
		return false
	}
	if parseS.authManager == nil {
		if parseS.logger != nil {
			parseS.logger.With(parseBuildLogFieldAttrs(parseR.Context(), parseLogFieldSpec{ParseRoute: strings.TrimSpace(parseR.URL.Path)})...).Warn(
				"http.admin deep link denied",
				slog.String("code", codes.Unauthenticated.String()),
				slog.String("error", "auth manager unavailable"),
				slog.String("next_action", "initialize auth manager and retry admin route entry"),
			)
		}
		http.Redirect(parseW, parseR, "/app", http.StatusSeeOther)
		return true
	}
	parseUser, parseIsAuthenticated := parseS.authManager.parseAuthenticatedUserFromRequest(parseR)
	if !parseIsAuthenticated || parseUser.ID <= 0 {
		if parseS.logger != nil {
			parseS.logger.With(parseBuildLogFieldAttrs(parseR.Context(), parseLogFieldSpec{ParseRoute: strings.TrimSpace(parseR.URL.Path), ParseActorUserID: parseUser.ID})...).Warn(
				"http.admin deep link denied",
				slog.String("code", codes.Unauthenticated.String()),
				slog.String("error", "authentication required"),
				slog.String("next_action", "sign in and retry admin route entry"),
			)
		}
		parseS.authManager.clearAuthCookie(parseW, parseR)
		http.Redirect(parseW, parseR, "/app", http.StatusSeeOther)
		return true
	}
	parseScope, parseErr := parseS.parseResolveAdminAccessScopeForUserID(parseUser.ID)
	if parseErr != nil {
		if parseS.logger != nil {
			parseS.logger.With(parseBuildLogFieldAttrs(parseR.Context(), parseLogFieldSpec{ParseRoute: strings.TrimSpace(parseR.URL.Path), ParseActorUserID: parseUser.ID})...).Warn(
				"http.admin deep link denied",
				slog.String("code", status.Code(parseErr).String()),
				slog.String("error", parseErr.Error()),
				slog.String("next_action", "verify role scope before retrying admin route entry"),
			)
		}
		http.Redirect(parseW, parseR, "/app", http.StatusSeeOther)
		return true
	}
	if parseS.logger != nil {
		parseScopeType := "workspace"
		if parseScope.isPlatformScope {
			parseScopeType = "platform"
		}
		parseS.logger.With(parseBuildLogFieldAttrs(parseR.Context(), parseLogFieldSpec{ParseRoute: strings.TrimSpace(parseR.URL.Path), ParseActorUserID: parseUser.ID, ParseWorkspaceID: parseResolveAdminScopeWorkspaceID(parseScope.workspaceIDs)})...).Info(
			"http.admin deep link allowed",
			slog.String("scope", parseScopeType),
			slog.Int("workspace_count", len(parseScope.workspaceIDs)),
		)
	}
	return false
}

// parseIsAdminDashboardDeepLinkRequest reports whether one HTTP request targets one dashboard/admin deep link.
func parseIsAdminDashboardDeepLinkRequest(parseR *http.Request) bool {
	if parseR == nil || parseR.URL == nil {
		return false
	}
	if parseIsAdminDashboardDeepLinkPath(parseR.URL.Path) {
		return true
	}
	parseCleanedPath := filepath.ToSlash(filepath.Clean("/" + strings.TrimLeft(strings.TrimSpace(parseR.URL.Path), "/")))
	if parseCleanedPath != "/app/settings" {
		return false
	}
	return parseIsAdminDashboardPanelKey(parseR.URL.Query().Get("panel"))
}

// parseIsAdminDashboardDeepLinkPath reports whether one HTTP path resolves to one admin/dashboard deep-link namespace.
func parseIsAdminDashboardDeepLinkPath(parseRequestPath string) bool {
	parseTrimmedPath := strings.TrimSpace(parseRequestPath)
	if parseTrimmedPath == "" {
		return false
	}
	parseCleanedPath := filepath.ToSlash(filepath.Clean("/" + strings.TrimLeft(parseTrimmedPath, "/")))
	switch parseCleanedPath {
	case "/admin", "/dashboard", "/su", "/app/admin", "/app/dashboard", "/app/su":
		return true
	default:
		return strings.HasPrefix(parseCleanedPath, "/admin/") ||
			strings.HasPrefix(parseCleanedPath, "/dashboard/") ||
			strings.HasPrefix(parseCleanedPath, "/su/") ||
			strings.HasPrefix(parseCleanedPath, "/app/admin/") ||
			strings.HasPrefix(parseCleanedPath, "/app/dashboard/") ||
			strings.HasPrefix(parseCleanedPath, "/app/su/")
	}
}

// parseIsAdminDashboardPanelKey reports whether one settings panel key represents one dashboard/admin surface.
func parseIsAdminDashboardPanelKey(parsePanelKey string) bool {
	parseNormalizedKey := strings.ToLower(strings.TrimSpace(parsePanelKey))
	if parseNormalizedKey == "" {
		return false
	}
	return strings.Contains(parseNormalizedKey, "admin") ||
		strings.Contains(parseNormalizedKey, "dashboard") ||
		strings.Contains(parseNormalizedKey, "superuser") ||
		strings.HasPrefix(parseNormalizedKey, "su-")
}

func parseRewriteLegacyClientAssetRequest(parseR *http.Request) *http.Request {
	switch parseR.URL.Path {
	case "/chat.wasm":
		return parseCloneRequestWithPath(parseR, "/app/chat.wasm")
	case "/background-worker.wasm":
		return parseCloneRequestWithPath(parseR, "/worker/background-worker.wasm")
	default:
		return parseR
	}
}

func shouldServeClientShell(parseRequestPath string) bool {
	parseTrimmedPath := strings.TrimSpace(parseRequestPath)
	if parseTrimmedPath == "" || parseTrimmedPath == "/" {
		return true
	}
	parseCleanedPath := filepath.ToSlash(filepath.Clean("/" + strings.TrimLeft(parseTrimmedPath, "/")))
	if parseCleanedPath == "/" || parseCleanedPath == "/app" || parseCleanedPath == "/app/" {
		return true
	}
	if parseCleanedPath == "/home" ||
		parseCleanedPath == "/capabilities" ||
		parseCleanedPath == "/pricing" ||
		parseCleanedPath == "/plans" ||
		parseCleanedPath == "/about" ||
		parseCleanedPath == "/contact" ||
		parseCleanedPath == "/privacy" ||
		parseCleanedPath == "/terms" ||
		parseCleanedPath == "/security" ||
		parseCleanedPath == "/status" {
		return true
	}
	if strings.HasPrefix(parseCleanedPath, "/app/") && filepath.Ext(parseCleanedPath) == "" {
		return true
	}
	if parseCleanedPath == "/thread" || strings.HasPrefix(parseCleanedPath, "/thread/") {
		return true
	}
	if strings.HasPrefix(parseCleanedPath, "/app/thread/") {
		return true
	}
	if filepath.Ext(parseCleanedPath) != "" {
		return false
	}
	switch parseCleanedPath {
	case "/login", "/signup", "/logout":
		return true
	default:
		return false
	}
}

func parseCloneRequestWithPath(parseR *http.Request, parsePath string) *http.Request {
	parseCloned := parseR.Clone(parseR.Context())
	if parseCloned.URL != nil {
		parseUrlCopy := *parseCloned.URL
		parseCloned.URL = &parseUrlCopy
	}
	parseCloned.URL.Path = parsePath
	parseCloned.URL.RawPath = parsePath
	parseCloned.RequestURI = parsePath
	if parseCloned.URL.RawQuery != "" {
		parseCloned.RequestURI += "?" + parseCloned.URL.RawQuery
	}
	return parseCloned
}

// ParseRun starts the chat wizard server entrypoint.
func ParseRun() {
	parseLoggers, parseLoggerErr := parseNewServerLoggers()
	parseLogger := parseLoggers.parseServerLogger
	parseClientLogger := parseLoggers.parseClientLogger
	parseCloseLogger := parseLoggers.parseClose
	if parseLoggerErr != nil {
		parseLogger = parseNewOTELLogger(os.Stderr, serverServiceName)
		parseClientLogger = parseNewOTELLogger(os.Stderr, clientLogServiceName)
		parseCloseLogger = func() {}
		parseLogger.Error("logging: failed to initialize file sink; continuing with stderr only",
			slog.String("error", parseLoggerErr.Error()),
			slog.String("log.dir", serverLogDir),
		)
	}
	defer parseCloseLogger()
	slog.SetDefault(parseLogger)
	// Load .env from the example directory, the server directory, or the repo
	// root, whichever is found first. Existing environment variables are never
	// overwritten, so explicit exports always take precedence.
	parseLoadedEnvPath := parseLoadFirstDotEnv(func(parsePath string) error { return godotenv.Load(parsePath) }, []string{
		".env",
		"../.env",
		"examples/100-ai-chat-wizard/.env",
	})
	if parseLoadedEnvPath != "" {
		parseLogger.Info("env: loaded .env file", slog.String("path", parseLoadedEnvPath))
	}

	parseConfig := parseReadServerRuntimeConfig(os.Getenv)
	if parseErr := parseValidateAuthSecretConfig(parseConfig); parseErr != nil {
		parseLogger.Error("auth: startup validation failed",
			slog.String("error", parseErr.Error()),
			slog.String("chat_env", parseConfig.environment),
		)
		os.Exit(1)
	}
	if strings.TrimSpace(parseConfig.authSecret) == "" {
		parseLogger.Warn("auth: CHAT_AUTH_SECRET not set; permitting development fallback secret",
			slog.String("chat_env", parseConfig.environment),
			slog.Bool("allow_insecure_auth_fallback", parseConfig.allowInsecureAuthSecretFallback),
		)
	}
	setChatUsagePremiumPercent(parseConfig.usagePremiumPct)
	setChatPlatformFeeUSD(parseConfig.platformFeeUSD)
	parseLogger.Info("billing: usage premium configured", slog.Float64("usage_premium_percent", parseConfig.usagePremiumPct))
	parseLogger.Info("billing: platform fee configured", slog.Float64("platform_fee_usd", parseConfig.platformFeeUSD))
	parseStubSet := parseNormalizeStubProviders(parseConfig.stubProviders)

	parseOpenAIAPIKey := parseConfig.openAIAPIKey
	if parseOpenAIAPIKey == "" {
		if _, parseOk := parseStubSet["openai"]; parseOk {
			parseLogger.Info("env: OPENAI_API_KEY not set; using OpenAI stub provider for local development")
		} else {
			parseLogger.Warn("env: OPENAI_API_KEY not set - chat RPCs will return Unavailable until configured")
		}
	}
	parseAnthropicAPIKey := parseConfig.anthropicAPIKey
	if parseAnthropicAPIKey == "" {
		if _, parseOk2 := parseStubSet["anthropic"]; parseOk2 {
			parseLogger.Info("env: ANTHROPIC_API_KEY not set; using Anthropic stub provider for local development")
		} else {
			parseLogger.Warn("env: ANTHROPIC_API_KEY not set - Claude models will be unavailable until configured")
		}
	}

	parseCerebrasAPIKey := parseConfig.cerebrasAPIKey
	if parseCerebrasAPIKey == "" {
		if _, parseOk3 := parseStubSet["cerebras"]; parseOk3 {
			parseLogger.Info("env: CEREBRAS_API_KEY not set; using Cerebras stub provider for local development")
		} else {
			parseLogger.Warn("env: CEREBRAS_API_KEY not set - Cerebras models will be unavailable until configured")
		}
	}

	parseDefaultModel := parseConfig.defaultModel
	if parseDefaultModel == "" {
		parseLogger.Debug("env: CHAT_MODEL not set - server default will be chosen from available providers")
	} else {
		parseLogger.Info("env: model override", slog.String("model", parseDefaultModel))
	}

	parseAddr := parseConfig.addr

	// Open SQLite persistence before wiring shared services.
	parseDbPath := parseConfig.dbPath
	store, parseDbErr := parseOpenChatStore(parseDbPath)
	if parseDbErr != nil {
		parseLogger.Error("db: failed to open - running without persistence",
			slog.String("path", parseDbPath),
			slog.String("error", parseDbErr.Error()),
		)
		store = nil
	} else {
		parseLogger.Info("db: opened", slog.String("path", parseDbPath))
		defer store.parseClose()
	}
	parseAuthManager := parseNewAuthManager(parseConfig.authSecret, store, parseLogger.With(slog.String("component", "auth")))

	// Build the gRPC server before wiring the browser bridge.
	parseGrpcSrv := grpc.NewServer(
		grpc.UnaryInterceptor(parseBuildCustomerSafeErrorUnaryInterceptor(parseLogger)),
		grpc.StreamInterceptor(parseBuildCustomerSafeErrorStreamInterceptor(parseLogger)),
	)
	parseSvcLog := parseLogger.With(slog.String("component", "chat-service"))
	parseChatService := parseNewChatServiceServer(parseOpenAIAPIKey, parseAnthropicAPIKey, parseCerebrasAPIKey, parseDefaultModel, store, parseSvcLog, parseConfig.stubProviders...)
	parseChatService.clientLogger = parseClientLogger.With(slog.String("component", "client-log"))
	parseChatService.authManager = parseAuthManager
	parseChatService.runtimeEnvironment = parseConfig.environment
	chatpb.RegisterChatServiceServer(parseGrpcSrv, parseChatService)

	// Use GoGRPCBridge to expose gRPC over WebSocket for the browser client.
	parseTunnelHandler := parseNewGRPCTunnelHandler(parseGrpcSrv, parseLogger)

	parseMux := http.NewServeMux()
	parseMux.Handle("/socket", parseTunnelHandler)
	parseMux.Handle("/socket/", parseTunnelHandler)
	parseMux.HandleFunc("/api/public/auth/session/sync", parseChatService.parseHandlePublicAuthSessionSync)
	parseMux.HandleFunc("/api/public/auth/password-reset/request", parseChatService.parseHandlePublicPasswordResetRequest)
	parseMux.HandleFunc("/api/public/auth/password-reset/consume", parseChatService.parseHandlePublicPasswordResetConsume)
	parseMux.HandleFunc("/api/public/auth/signup-verification/resend", parseChatService.parseHandlePublicSignupVerificationResend)
	parseMux.HandleFunc("/api/public/auth/signup-verification/consume", parseChatService.parseHandlePublicSignupVerificationConsume)

	parseClientDir, parseSharedDir := parseResolveStaticDirectories()
	if parseSharedDir != "" {
		parseMux.Handle("/static/", http.StripPrefix("/static/", parseNewPrecompressedWASMFileServer(parseSharedDir)))
	}
	// wasm_exec.js is served from its known location in third_party.
	parseWasmExecPath := parseResolveWasmExecPath()
	parseMux.HandleFunc("/static/script/wasm_exec.js", func(parseW http.ResponseWriter, parseR *http.Request) {
		parseW.Header().Set("Content-Type", "application/javascript")
		http.ServeFile(parseW, parseR, parseWasmExecPath)
	})
	parseFileServer := parseNewPrecompressedWASMFileServer(parseClientDir)
	parseMux.HandleFunc("/favicon.ico", func(parseW2 http.ResponseWriter, _ *http.Request) {
		parseW2.WriteHeader(http.StatusNoContent)
	})
	parseMux.HandleFunc("/healthz", func(parseW3 http.ResponseWriter, _ *http.Request) {
		fmt.Fprintln(parseW3, "ok")
	})
	// Marketing pages (public, no auth required).
	// Catch-all: bare "/" serves the marketing home; all other paths use the chat shell.
	parseMux.Handle("/", parseChatShellHandlerForServer(parseChatService, parseFileServer))

	parseLogger.Info("server: starting",
		slog.String("addr", parseAddr),
		slog.String("client_dir", parseClientDir),
		slog.String("static_dir", parseSharedDir),
		slog.String("grpc_ws", "ws://"+parseAddr+"/socket"),
	)

	parseSrv := &http.Server{
		Addr:              parseAddr,
		Handler:           parseMux,
		ReadHeaderTimeout: 10 * time.Second,
	}

	parseQuit := make(chan os.Signal, 1)
	signal.Notify(parseQuit, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		parseSig := <-parseQuit
		parseLogger.Info("server: shutdown signal received", slog.String("signal", parseSig.String()))
		parseCtx, parseCancel := context.WithTimeout(context.Background(), 10*time.Second)
		defer parseCancel()
		parseGrpcSrv.GracefulStop()
		if parseErr := parseSrv.Shutdown(parseCtx); parseErr != nil {
			parseLogger.Error("server: graceful shutdown failed", slog.String("error", parseErr.Error()))
		} else {
			parseLogger.Info("server: shutdown complete")
		}
	}()

	if parseErr2 := parseSrv.ListenAndServe(); parseErr2 != nil && parseErr2 != http.ErrServerClosed {
		parseLogger.Error("server: ListenAndServe failed",
			slog.String("addr", parseAddr),
			slog.String("error", parseErr2.Error()),
		)
		os.Exit(1)
	}
}

// resolveWasmExecPath locates wasm_exec.js relative to common invocation roots.
func parseResolveWasmExecPath() string {
	parseCandidates := []string{
		"third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js",
		"../../../third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js",
		"../../../../third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js",
	}
	for _, parseC := range parseCandidates {
		if _, parseErr := os.Stat(parseC); parseErr == nil {
			return parseC
		}
	}
	return parseCandidates[0]
}

// resolveStaticDirectories returns (clientDir, sharedStaticDir).
// clientDir contains the WASM binaries and related browser assets.
// sharedStaticDir contains the examples-wide static assets (tailwind, wasm_exec.js).
// Both resolve relative to common invocation roots (from repo root or from
// the server/ subdirectory).
func parseResolveStaticDirectories() (parseClientDir, parseSharedDir string) {
	parseClientCandidates := []string{
		"../bin/client",
		"bin/client",
		"examples/100-ai-chat-wizard/bin/client",
	}
	for _, parseC := range parseClientCandidates {
		if parseInfo, parseErr := os.Stat(parseC); parseErr == nil && parseInfo.IsDir() {
			parseClientDir = parseC
			break
		}
	}
	if parseClientDir == "" {
		parseClientDir = "bin/client"
	}

	parseSharedCandidates := []string{
		"../../static",
		"../../../static",
		"examples/static",
	}
	for _, parseC2 := range parseSharedCandidates {
		if parseInfo2, parseErr2 := os.Stat(parseC2); parseErr2 == nil && parseInfo2.IsDir() {
			parseSharedDir = parseC2
			break
		}
	}
	return
}

func parseNewPrecompressedWASMFileServer(parseRootDir string) http.Handler {
	parseFileServer := http.FileServer(http.Dir(parseRootDir))

	return http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
		if parseR != nil && filepath.Ext(parseR.URL.Path) == ".wasm" {
			parseSetNoStoreResponseHeaders(parseW)
		}
		if parseTryServeBrotliWASM(parseW, parseR, parseRootDir) {
			return
		}
		parseFileServer.ServeHTTP(parseW, parseR)
	})
}

func parseTryServeBrotliWASM(parseW http.ResponseWriter, parseR *http.Request, parseRootDir string) bool {
	if parseR.Method != http.MethodGet && parseR.Method != http.MethodHead {
		return false
	}
	if filepath.Ext(parseR.URL.Path) != ".wasm" {
		return false
	}
	parseQueryValue := strings.ToLower(strings.TrimSpace(parseR.URL.Query().Get("br")))
	if parseQueryValue == "false" || parseQueryValue == "0" {
		return false
	}
	if parseQueryValue != "" && parseQueryValue != "true" && parseQueryValue != "1" {
		return false
	}

	parseRelativePath, parseOk := parseResolveRelativeAssetPath(parseR.URL.Path)
	if !parseOk {
		return false
	}
	if !parseShouldServeFreshBrotliArtifact(parseRootDir, parseRelativePath) {
		return false
	}
	parseBrotliPath := filepath.Join(parseRootDir, parseRelativePath) + ".br"
	parseArtifactInfo, parseErr := os.Stat(parseBrotliPath)
	if parseErr != nil || parseArtifactInfo.IsDir() {
		return false
	}
	if parseArtifactInfo.Size() <= 0 {
		return false
	}

	parseOutputFile, parseErr := os.Open(parseBrotliPath)
	if parseErr != nil {
		return false
	}
	defer parseOutputFile.Close()

	parseSetNoStoreResponseHeaders(parseW)
	parseW.Header().Set("Content-Encoding", "br")
	parseW.Header().Set("Content-Type", "application/wasm")
	parseW.Header().Set("Vary", "Accept-Encoding")
	http.ServeContent(parseW, parseR, filepath.Base(parseR.URL.Path), parseArtifactInfo.ModTime(), parseOutputFile)
	return true
}

// parseShouldServeFreshBrotliArtifact reports whether the Brotli sidecar exists and is at least as fresh as the raw wasm artifact.
func parseShouldServeFreshBrotliArtifact(parseRootDir string, parseRelativePath string) bool {
	parseBrotliPath := filepath.Join(parseRootDir, parseRelativePath) + ".br"
	parseBrotliInfo, parseErr := os.Stat(parseBrotliPath)
	if parseErr != nil || parseBrotliInfo.IsDir() || parseBrotliInfo.Size() <= 0 {
		return false
	}
	parseRawPath := filepath.Join(parseRootDir, parseRelativePath)
	parseRawInfo, parseErr := os.Stat(parseRawPath)
	if parseErr != nil || parseRawInfo.IsDir() || parseRawInfo.Size() <= 0 {
		return true
	}
	return !parseRawInfo.ModTime().After(parseBrotliInfo.ModTime())
}

func parseResolveRelativeAssetPath(parseRequestPath string) (string, bool) {
	parseCleanedPath := filepath.ToSlash(filepath.Clean("/" + parseRequestPath))
	if strings.Contains(parseCleanedPath, "..") {
		return "", false
	}
	parseCleanedPath = strings.TrimLeft(parseCleanedPath, "/")
	parseCleanedPath = strings.TrimPrefix(parseCleanedPath, "./")
	return parseCleanedPath, parseCleanedPath != ""
}
