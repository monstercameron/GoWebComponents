package app

import (
	"bytes"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
	"time"

	"github.com/gorilla/websocket"
	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/metadata"
	"google.golang.org/grpc/status"
	emptypb "google.golang.org/protobuf/types/known/emptypb"
	wrapperspb "google.golang.org/protobuf/types/known/wrapperspb"
)

// TestLoggerHelperBranches covers the remaining logger helper branches.
func TestLoggerHelperBranches(parseT *testing.T) {
	if parseBoundary := parseErrorBoundaryFromMessage("   "); parseBoundary != "" {
		parseT.Fatalf("blank error boundary = %q, want empty", parseBoundary)
	}

	parseStringRecord := slog.NewRecord(time.Now(), slog.LevelError, "record message", 0)
	parseStringRecord.AddAttrs(slog.Int("error", 7))
	parseErrorMessage, parseErrorType := parseErrorDetailsFromRecord(parseStringRecord)
	if parseErrorMessage != "7" || parseErrorType != "" {
		parseT.Fatalf("int error attr = (%q, %q), want (\"7\", \"\")", parseErrorMessage, parseErrorType)
	}

	parseAnyRecord := slog.NewRecord(time.Now(), slog.LevelError, "any message", 0)
	parseAnyRecord.AddAttrs(slog.Any("error", map[string]string{"reason": "denied"}))
	parseErrorMessage, parseErrorType = parseErrorDetailsFromRecord(parseAnyRecord)
	if parseErrorMessage != "map[reason:denied]" {
		parseT.Fatalf("map error attr message = %q, want map[reason:denied]", parseErrorMessage)
	}
	if parseErrorType != "map[string]string" {
		parseT.Fatalf("map error attr type = %q, want map[string]string", parseErrorType)
	}

	parseNilRecord := slog.NewRecord(time.Now(), slog.LevelError, "nil message", 0)
	parseNilRecord.AddAttrs(slog.Any("error", nil))
	parseErrorMessage, parseErrorType = parseErrorDetailsFromRecord(parseNilRecord)
	if parseErrorMessage != "" || parseErrorType != "" {
		parseT.Fatalf("nil error attr = (%q, %q), want empty strings", parseErrorMessage, parseErrorType)
	}

	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, "")
	parseLogger.Error("  fallback message  ", slog.String("detail", "ignored"))

	parseLine := strings.TrimSpace(parseOutput.String())
	if !strings.Contains(parseLine, `"error.message":"fallback message"`) {
		parseT.Fatalf("expected fallback error message in log line, got %q", parseLine)
	}
	if strings.Contains(parseLine, `"service.name"`) {
		parseT.Fatalf("did not expect service.name for blank service name, got %q", parseLine)
	}
}

// TestServerLoggerFailureBranches covers directory creation and file open failures.
func TestServerLoggerFailureBranches(parseT *testing.T) {
	parseOriginalWD, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("Getwd: %v", parseErr)
	}
	parseTempDir := parseT.TempDir()
	if parseErr2 := os.Chdir(parseTempDir); parseErr2 != nil {
		parseT.Fatalf("Chdir temp dir: %v", parseErr2)
	}
	defer func() {
		if parseErr3 := os.Chdir(parseOriginalWD); parseErr3 != nil {
			parseT.Fatalf("restore cwd: %v", parseErr3)
		}
	}()

	if parseErr4 := os.WriteFile(serverLogDir, []byte("not a directory"), 0o644); parseErr4 != nil {
		parseT.Fatalf("WriteFile log sentinel: %v", parseErr4)
	}
	if _, _, parseErr5 := parseNewServerLogger(); parseErr5 == nil {
		parseT.Fatal("expected parseNewServerLogger to fail when log path is a file")
	}

	if parseErr6 := os.Remove(serverLogDir); parseErr6 != nil {
		parseT.Fatalf("Remove log sentinel: %v", parseErr6)
	}
	if parseErr7 := os.MkdirAll(filepath.Join(serverLogDir, serverLogFilename), 0o755); parseErr7 != nil {
		parseT.Fatalf("MkdirAll log filename dir: %v", parseErr7)
	}
	if _, _, parseErr8 := parseNewServerLogger(); parseErr8 == nil {
		parseT.Fatal("expected parseNewServerLogger to fail when log filename is a directory")
	}
}

// TestAuthManagerFailureBranches covers nil, invalid, and closed-store auth paths.
func TestAuthManagerFailureBranches(parseT *testing.T) {
	var parseNilAuth *authManager
	if _, parseOk := parseNilAuth.parseAuthenticatedUserFromContext(metadata.NewIncomingContext(parseNewAuthenticatedContext("nil-auth"), metadata.Pairs(authMetadataKey, "Bearer token"))); parseOk {
		parseT.Fatal("expected nil auth manager context lookup to fail")
	}
	if _, parseOk2 := parseNilAuth.parseAuthenticatedUserFromRequest(httptest.NewRequest(http.MethodGet, "http://example.com", nil)); parseOk2 {
		parseT.Fatal("expected nil auth manager request lookup to fail")
	}

	parseStore := parseNewTestStore(parseT)
	parseAuth := parseNewAuthManager("test-secret", parseStore, parseNewTestLogger())
	parseUser, parseErr := parseAuth.parseSignup("branch@example.com", "password123", "Branch")
	if parseErr != nil {
		parseT.Fatalf("signup: %v", parseErr)
	}

	parseZeroToken, parseErr := parseAuth.issueToken(authUser{ID: 0, Email: "zero@example.com"})
	if parseErr != nil {
		parseT.Fatalf("issueToken zero user: %v", parseErr)
	}
	if _, parseErr2 := parseAuth.parseToken(parseZeroToken); !errors.Is(parseErr2, errInvalidCredentials) {
		parseT.Fatalf("expected invalid credentials for zero user token, got %v", parseErr2)
	}

	if _, parseOk3 := parseAuth.parseAuthenticatedUserFromContext(parseNewAuthenticatedContext("no-metadata")); parseOk3 {
		parseT.Fatal("expected context without metadata to fail authentication")
	}
	if _, parseOk4 := parseAuth.parseAuthenticatedUserFromContext(metadata.NewIncomingContext(parseNewAuthenticatedContext("bad-token"), metadata.Pairs(authMetadataKey, "Bearer not-a-jwt"))); parseOk4 {
		parseT.Fatal("expected invalid metadata token to fail authentication")
	}
	if parseTokenValue, parseOk5 := parseAuthTokenFromAuthorizationValue("Bearer"); !parseOk5 || parseTokenValue != "Bearer" {
		parseT.Fatalf("unexpected bare Bearer token parsing: ok=%v token=%q", parseOk5, parseTokenValue)
	}

	parseBadRequest := httptest.NewRequest(http.MethodGet, "http://example.com", nil)
	parseBadRequest.AddCookie(&http.Cookie{Name: authCookieName, Value: "not-a-jwt"})
	if _, parseOk6 := parseAuth.parseAuthenticatedUserFromRequest(parseBadRequest); parseOk6 {
		parseT.Fatal("expected invalid cookie token to fail authentication")
	}

	parseStore.parseClose()
	if _, parseOk7 := parseAuth.parseValidateActiveUser(parseUser, "closed-store"); parseOk7 {
		parseT.Fatal("expected closed store validation to fail")
	}
	if _, parseErr3 := parseAuth.parseLogin(parseUser.Email, "password123"); !errors.Is(parseErr3, errInvalidCredentials) {
		parseT.Fatalf("expected invalid credentials when store lookup fails, got %v", parseErr3)
	}
}

// TestModelCatalogLoadingBranches covers nil-store loading and fallback catalog selection.
func TestModelCatalogLoadingBranches(parseT *testing.T) {
	parseRows, parseErr := parseLoadModelCatalogRows(nil)
	if parseErr != nil {
		parseT.Fatalf("parseLoadModelCatalogRows(nil): %v", parseErr)
	}
	if len(parseRows) == 0 {
		parseT.Fatal("expected embedded model catalog rows")
	}

	parseStore := parseNewTestStore(parseT)
	if _, parseErr2 := parseStore.db.Exec(`DELETE FROM model_catalog`); parseErr2 != nil {
		parseT.Fatalf("delete model catalog: %v", parseErr2)
	}
	if _, parseErr3 := parseStore.db.Exec(`
		INSERT INTO model_catalog (
			id, provider_id, provider_label, label, note, description,
			supports_thinking, supports_speech,
			input_cost_per_million_usd, output_cost_per_million_usd, pricing_currency,
			max_output_tokens, throughput_tokens_per_second, onboarding_ready,
			is_default, use_for_title_generation, use_for_memory_extraction, sort_order
		) VALUES
			('gpt-5.4-2026-03-17', 'openai', 'OpenAI', 'GPT-5.4 dated', '', '', 1, 1, 0, 0, 'USD', 0, 0, 1, 0, 0, 0, 10),
			('title-model', 'openai', 'OpenAI', 'Title model', '', '', 1, 0, 0, 0, 'USD', 0, 0, 1, 0, 1, 0, 20),
			('ignored-model', '   ', 'OpenAI', 'Ignored provider', '', '', 0, 0, 0, 0, 'USD', 0, 0, 1, 0, 0, 0, 30)
	`); parseErr3 != nil {
		parseT.Fatalf("insert model catalog rows: %v", parseErr3)
	}

	parseConfig, parseErr4 := parseLoadModelCatalogConfig(parseStore)
	if parseErr4 != nil {
		parseT.Fatalf("parseLoadModelCatalogConfig: %v", parseErr4)
	}
	if parseConfig.DefaultModel != modelGPT54 {
		parseT.Fatalf("default model = %q, want %q", parseConfig.DefaultModel, modelGPT54)
	}
	if parseConfig.MemoryExtractionModel != modelGPT54 {
		parseT.Fatalf("memory extraction model = %q, want %q", parseConfig.MemoryExtractionModel, modelGPT54)
	}

	parseOpenAICatalog, parseOk := parseConfig.ProviderCatalogs["openai"]
	if !parseOk {
		parseT.Fatalf("expected openai catalog, got %+v", parseConfig.ProviderCatalogs)
	}
	if parseOpenAICatalog.DefaultModel != modelGPT54 {
		parseT.Fatalf("openai default model = %q, want %q", parseOpenAICatalog.DefaultModel, modelGPT54)
	}
	if parseOpenAICatalog.TitleModel != "title-model" {
		parseT.Fatalf("openai title model = %q, want title-model", parseOpenAICatalog.TitleModel)
	}
	if len(parseConfig.ProviderCatalogs) != 1 {
		parseT.Fatalf("provider catalog count = %d, want 1", len(parseConfig.ProviderCatalogs))
	}
}

// TestTunnelHandlerHooksLogConnectAndDisconnect covers the live websocket tunnel hooks.
func TestTunnelHandlerHooksLogConnectAndDisconnect(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName)
	parseServer := httptest.NewServer(parseNewGRPCTunnelHandler(grpc.NewServer(), parseLogger))
	defer parseServer.Close()

	parseSocketURL := "ws" + strings.TrimPrefix(parseServer.URL, "http")
	parseClientSocket, _, parseErr := websocket.DefaultDialer.Dial(parseSocketURL, nil)
	if parseErr != nil {
		parseT.Fatalf("websocket dial: %v", parseErr)
	}
	if parseErr2 := parseClientSocket.Close(); parseErr2 != nil {
		parseT.Fatalf("client close: %v", parseErr2)
	}

	parseDeadline := time.Now().Add(2 * time.Second)
	for time.Now().Before(parseDeadline) {
		parseLogs := parseOutput.String()
		if strings.Contains(parseLogs, "tunnel: client connected") && strings.Contains(parseLogs, "tunnel: client disconnected") {
			return
		}
		time.Sleep(20 * time.Millisecond)
	}

	parseT.Fatalf("expected connect and disconnect logs, got %q", parseOutput.String())
}

// TestPreferenceRPCFailureBranches covers invalid and closed-store preference RPC branches.
func TestPreferenceRPCFailureBranches(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "preference-errors@example.com")
	parseServer := parseNewFakeChatServer(parseStore, parseNewFakeProvider())
	parseCtx := parseBindAuthUser(parseServer, "peer-preference-errors", parseUser.ID, parseUser.Email)

	if _, parseErr := parseServer.SetSelectedModel(parseCtx, wrapperspb.String("missing-model")); status.Code(parseErr) != codes.InvalidArgument {
		parseT.Fatalf("expected invalid argument for unsupported selected model, got %v", status.Code(parseErr))
	}

	parseStore.parseClose()

	if _, parseErr2 := parseServer.SetSelectedModel(parseCtx, wrapperspb.String(modelGPT54Mini)); status.Code(parseErr2) != codes.Internal {
		parseT.Fatalf("expected internal set selected model error after store close, got %v", status.Code(parseErr2))
	}
	if _, parseErr3 := parseServer.GetSelectedModel(parseCtx, &emptypb.Empty{}); status.Code(parseErr3) != codes.Internal {
		parseT.Fatalf("expected internal get selected model error after store close, got %v", status.Code(parseErr3))
	}
	if _, parseErr4 := parseServer.SetSelectedTone(parseCtx, wrapperspb.String("professional")); status.Code(parseErr4) != codes.Internal {
		parseT.Fatalf("expected internal set selected tone error after store close, got %v", status.Code(parseErr4))
	}
	if _, parseErr5 := parseServer.GetSelectedTone(parseCtx, &emptypb.Empty{}); status.Code(parseErr5) != codes.Internal {
		parseT.Fatalf("expected internal get selected tone error after store close, got %v", status.Code(parseErr5))
	}
	if _, parseErr6 := parseServer.SetSelectedThinkingEnabled(parseCtx, wrapperspb.Bool(true)); status.Code(parseErr6) != codes.Internal {
		parseT.Fatalf("expected internal set selected thinking enabled error after store close, got %v", status.Code(parseErr6))
	}
	if _, parseErr7 := parseServer.SetSelectedThinkingEffort(parseCtx, wrapperspb.String("high")); status.Code(parseErr7) != codes.Internal {
		parseT.Fatalf("expected internal set selected thinking effort error after store close, got %v", status.Code(parseErr7))
	}

	parseServer.parseUnbindAuthenticatedPeer("peer-preference-errors")
}
