package app

import (
	"bytes"
	"database/sql"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"

	"github.com/google/uuid"
	"github.com/monstercameron/GoWebComponents/examples/100-ai-chat-wizard/server/provider"
	"google.golang.org/grpc"
)

// TestServerLoggerCreatesFileSink verifies the file-backed logger setup and close hook.
func TestServerLoggerCreatesFileSink(parseT *testing.T) {
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

	parseLogger, parseCloseLogger, parseErr4 := parseNewServerLogger()
	if parseErr4 != nil {
		parseT.Fatalf("parseNewServerLogger: %v", parseErr4)
	}
	parseLogger.Info("server logger test")
	parseCloseLogger()

	parseLogPath := filepath.Join(parseTempDir, serverLogDir, serverLogFilename)
	parseLogBytes, parseErr5 := os.ReadFile(parseLogPath)
	if parseErr5 != nil {
		parseT.Fatalf("ReadFile(%q): %v", parseLogPath, parseErr5)
	}
	if !strings.Contains(string(parseLogBytes), "server logger test") {
		parseT.Fatalf("expected log file to contain test message, got %q", string(parseLogBytes))
	}
}

// TestOTELLoggerWithAttrsAndGroup verifies grouped attributes through the wrapper handler.
func TestOTELLoggerWithAttrsAndGroup(parseT *testing.T) {
	var parseOutput bytes.Buffer
	parseLogger := parseNewOTELLogger(&parseOutput, serverServiceName).
		With(slog.String("scope", "chat")).
		WithGroup("request")
	parseLogger.Info("grouped log", slog.String("path", "/chat"))

	var parsePayload map[string]any
	if parseErr := json.Unmarshal(parseOutput.Bytes(), &parsePayload); parseErr != nil {
		parseT.Fatalf("json.Unmarshal: %v", parseErr)
	}
	if parsePayload["scope"] != "chat" {
		parseT.Fatalf("scope = %v, want chat", parsePayload["scope"])
	}
	parseGroupPayload, parseOk := parsePayload["request"].(map[string]any)
	if !parseOk {
		parseT.Fatalf("request group = %T, want map", parsePayload["request"])
	}
	if parseGroupPayload["path"] != "/chat" {
		parseT.Fatalf("request.path = %v, want /chat", parseGroupPayload["path"])
	}
}

// TestStubProviderHelpersNormalizeAndSelect verifies stub normalization and provider selection.
func TestStubProviderHelpersNormalizeAndSelect(parseT *testing.T) {
	parseNormalized := parseNormalizeStubProviders([]string{" openai,anthropic ", "all", " cerebras "})
	if len(parseNormalized) != 3 {
		parseT.Fatalf("normalizeStubProviders size = %d, want 3", len(parseNormalized))
	}
	for _, parseProviderID := range []string{"openai", "anthropic", "cerebras"} {
		if _, parseOk := parseNormalized[parseProviderID]; !parseOk {
			parseT.Fatalf("expected normalized provider %q to be present", parseProviderID)
		}
	}

	parseCatalog := provider.Catalog{
		DefaultModel: modelGPT54Mini,
		Models: []provider.ModelMetadata{{
			ID:            modelGPT54Mini,
			DisplayName:   "GPT-5.4 mini",
			ProviderID:    "openai",
			ProviderLabel: "OpenAI",
		}},
	}
	if parseSelected := parseSelectRuntimeProvider("openai", "test-key", nil, parseCatalog); parseSelected == nil || parseSelected.ParseID() != "openai" {
		parseT.Fatalf("expected keyed openai provider, got %#v", parseSelected)
	}
	if parseSelected := parseSelectRuntimeProvider("anthropic", "", map[string]struct{}{"anthropic": {}}, parseCatalog); parseSelected == nil || parseSelected.ParseID() != "anthropic" {
		parseT.Fatalf("expected stub anthropic provider, got %#v", parseSelected)
	}
	if parseSelected := parseSelectRuntimeProvider("cerebras", "", nil, parseCatalog); parseSelected == nil || parseSelected.ParseID() != "cerebras" {
		parseT.Fatalf("expected catalog cerebras provider, got %#v", parseSelected)
	}
	if parseSelected := parseSelectRuntimeProvider("unknown", "", nil, parseCatalog); parseSelected != nil {
		parseT.Fatalf("expected unknown provider to return nil, got %#v", parseSelected)
	}
}

// TestUserFacingStreamErrorVariants verifies user-facing stream error rewrites.
func TestUserFacingStreamErrorVariants(parseT *testing.T) {
	if parseGot := parseUserFacingStreamError("", "", errors.New("")); parseGot != "model provider stream failed" {
		parseT.Fatalf("empty provider error = %q, want default", parseGot)
	}
	if parseGot := parseUserFacingStreamError("openai", "", errors.New("")); parseGot != "openai stream failed" {
		parseT.Fatalf("provider fallback error = %q, want openai stream failed", parseGot)
	}
	parseCerebrasError := errors.New("404 Not Found: missing model")
	parseGot := parseUserFacingStreamError("cerebras", "gpt-oss-120b", parseCerebrasError)
	if !strings.Contains(parseGot, `Cerebras model "gpt-oss-120b" is unavailable`) {
		parseT.Fatalf("cerebras rewrite = %q", parseGot)
	}
	if parseGot2 := parseUserFacingStreamError("anthropic", "claude", errors.New("provider timeout")); parseGot2 != parsePublicErrorProviderMessage {
		parseT.Fatalf("plain provider error = %q, want %q", parseGot2, parsePublicErrorProviderMessage)
	}
}

// TestStoreRecoveryHelpersVerifyBranches covers public ID repair and recovery helpers.
func TestStoreRecoveryHelpersVerifyBranches(parseT *testing.T) {
	parseStore := parseNewTestStore(parseT)
	parseUser := parseMustCreateUser(parseT, parseStore, "coverage-store@example.com")
	parseConversationID, parseErr := parseStore.parseCreateConversation(parseUser.ID)
	if parseErr != nil {
		parseT.Fatalf("parseCreateConversation: %v", parseErr)
	}
	if _, parseErr2 := parseStore.db.Exec(`UPDATE conversations SET public_id = '' WHERE id = ?`, parseConversationID); parseErr2 != nil {
		parseT.Fatalf("clear public_id: %v", parseErr2)
	}
	if parseErr3 := parseEnsureConversationPublicIDs(parseStore.db); parseErr3 != nil {
		parseT.Fatalf("parseEnsureConversationPublicIDs: %v", parseErr3)
	}
	parseConversations, parseErr4 := parseStore.parseListConversations(parseUser.ID)
	if parseErr4 != nil {
		parseT.Fatalf("parseListConversations: %v", parseErr4)
	}
	if len(parseConversations) != 1 {
		parseT.Fatalf("conversation count = %d, want 1", len(parseConversations))
	}
	if parseConversations[0].PublicID == "" {
		parseT.Fatal("expected repaired public ID")
	}
	if _, parseErr5 := uuid.Parse(parseConversations[0].PublicID); parseErr5 != nil {
		parseT.Fatalf("expected repaired UUID, got %q: %v", parseConversations[0].PublicID, parseErr5)
	}

	parseOriginalGenerator := newConversationPublicID
	defer func() { newConversationPublicID = parseOriginalGenerator }()
	newConversationPublicID = func() string { return "   " }
	if parseErr6 := parseAssignConversationPublicID(parseStore.db, parseConversationID); parseErr6 == nil || !strings.Contains(parseErr6.Error(), "exhausted retries") {
		parseT.Fatalf("expected exhausted retries error, got %v", parseErr6)
	}

	parseBackupDir := parseT.TempDir()
	parseBackupPath := filepath.Join(parseBackupDir, "chat.db")
	for _, parseSuffix := range []string{"", "-wal", "-shm"} {
		if parseErr7 := os.WriteFile(parseBackupPath+parseSuffix, []byte("db"), 0o644); parseErr7 != nil {
			parseT.Fatalf("WriteFile(%q): %v", parseBackupPath+parseSuffix, parseErr7)
		}
	}
	parseBackupResult, parseErr8 := parseBackupIncompatibleStoreFiles(parseBackupPath)
	if parseErr8 != nil {
		parseT.Fatalf("parseBackupIncompatibleStoreFiles: %v", parseErr8)
	}
	if _, parseErr9 := os.Stat(parseBackupResult); parseErr9 != nil {
		parseT.Fatalf("expected backup file to exist: %v", parseErr9)
	}
	if _, parseErr10 := os.Stat(parseBackupResult + "-wal"); parseErr10 != nil {
		parseT.Fatalf("expected backup wal file to exist: %v", parseErr10)
	}
	if _, parseErr11 := os.Stat(parseBackupResult + "-shm"); parseErr11 != nil {
		parseT.Fatalf("expected backup shm file to exist: %v", parseErr11)
	}

	parseLegacyPath := filepath.Join(parseT.TempDir(), "legacy.db")
	parseLegacyDB, parseErr12 := sql.Open("sqlite3", "file:"+parseLegacyPath)
	if parseErr12 != nil {
		parseT.Fatalf("sql.Open legacy: %v", parseErr12)
	}
	if _, parseErr13 := parseLegacyDB.Exec(`CREATE TABLE user_profile (id INTEGER PRIMARY KEY, name TEXT NOT NULL);`); parseErr13 != nil {
		_ = parseLegacyDB.Close()
		parseT.Fatalf("create legacy schema: %v", parseErr13)
	}
	if parseErr14 := parseLegacyDB.Close(); parseErr14 != nil {
		parseT.Fatalf("close legacy db: %v", parseErr14)
	}
	if _, parseErr15 := parseOpenChatStoreWithRecovery(parseLegacyPath, false); parseErr15 == nil {
		parseT.Fatal("expected incompatible legacy schema without recovery to fail")
	}
	parseBackupMatches, parseErr16 := filepath.Glob(parseLegacyPath + ".incompatible-*.bak")
	if parseErr16 != nil {
		parseT.Fatalf("Glob legacy backups: %v", parseErr16)
	}
	if len(parseBackupMatches) != 0 {
		parseT.Fatalf("expected no recovery backup when recovery disabled, got %v", parseBackupMatches)
	}
}

// TestStoreRecoveryPredicatesVerifyBranches covers direct incompatible-store predicates.
func TestStoreRecoveryPredicatesVerifyBranches(parseT *testing.T) {
	parseMissingPath := filepath.Join(parseT.TempDir(), "missing.db")
	if shouldResetIncompatibleStore("", errors.New("malformed")) {
		parseT.Fatal("expected blank path to reject recovery reset")
	}
	if shouldResetIncompatibleStore(parseMissingPath, errors.New("malformed")) {
		parseT.Fatal("expected missing file to reject recovery reset")
	}

	parseExistingPath := filepath.Join(parseT.TempDir(), "existing.db")
	if parseErr := os.WriteFile(parseExistingPath, []byte("db"), 0o644); parseErr != nil {
		parseT.Fatalf("WriteFile existing db: %v", parseErr)
	}
	if !shouldResetIncompatibleStore(parseExistingPath, errors.New("table messages has no column named foo")) {
		parseT.Fatal("expected incompatible schema error to request reset")
	}
	if shouldResetIncompatibleStore(parseExistingPath, errors.New("permission denied")) {
		parseT.Fatal("expected unrelated error to reject reset")
	}
}

// TestTransportHelpersCoverTunnelAndWasmResolution verifies tunnel and wasm helper branches.
func TestTransportHelpersCoverTunnelAndWasmResolution(parseT *testing.T) {
	parseHandler := parseNewGRPCTunnelHandler(grpc.NewServer(), parseNewTestLogger())
	if parseHandler == nil {
		parseT.Fatal("expected grpc tunnel handler")
	}
	parseResp := httptest.NewRecorder()
	parseReq := httptest.NewRequest(http.MethodGet, "http://example.com/socket", nil)
	parseHandler.ServeHTTP(parseResp, parseReq)
	if parseResp.Code == http.StatusNotFound {
		parseT.Fatalf("unexpected not found response from tunnel handler: %d", parseResp.Code)
	}

	parseOriginalWD, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("Getwd: %v", parseErr)
	}
	parseTempDir := parseT.TempDir()
	parseAssetDir := filepath.Join(parseTempDir, "third_party", "GoGRPCBridge", "examples", "_shared", "public")
	if parseErr2 := os.MkdirAll(parseAssetDir, 0o755); parseErr2 != nil {
		parseT.Fatalf("MkdirAll wasm_exec dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseAssetDir, "wasm_exec.js"), []byte("console.log('wasm')"), 0o644); parseErr3 != nil {
		parseT.Fatalf("WriteFile wasm_exec.js: %v", parseErr3)
	}
	if parseErr4 := os.Chdir(parseTempDir); parseErr4 != nil {
		parseT.Fatalf("Chdir temp dir: %v", parseErr4)
	}
	parseFoundPath := filepath.ToSlash(parseResolveWasmExecPath())
	if parseFoundPath != "third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js" {
		parseT.Fatalf("resolveWasmExecPath found path = %q", parseFoundPath)
	}
	if parseErr5 := os.Chdir(parseOriginalWD); parseErr5 != nil {
		parseT.Fatalf("restore cwd: %v", parseErr5)
	}

	parseMissingDir := parseT.TempDir()
	if parseErr6 := os.Chdir(parseMissingDir); parseErr6 != nil {
		parseT.Fatalf("Chdir missing dir: %v", parseErr6)
	}
	defer func() {
		if parseErr7 := os.Chdir(parseOriginalWD); parseErr7 != nil {
			parseT.Fatalf("restore cwd: %v", parseErr7)
		}
	}()
	parseFallbackPath := filepath.ToSlash(parseResolveWasmExecPath())
	if parseFallbackPath != "third_party/GoGRPCBridge/examples/_shared/public/wasm_exec.js" {
		parseT.Fatalf("resolveWasmExecPath fallback = %q", parseFallbackPath)
	}
}

// TestBrotliServingRejectsUnsupportedRequests verifies remaining Brotli helper guard rails.
func TestBrotliServingRejectsUnsupportedRequests(parseT *testing.T) {
	parseRootDir := parseT.TempDir()
	parseArtifactDir := filepath.Join(parseRootDir, "app")
	if parseErr := os.MkdirAll(parseArtifactDir, 0o755); parseErr != nil {
		parseT.Fatalf("MkdirAll artifact dir: %v", parseErr)
	}
	if parseErr2 := os.WriteFile(filepath.Join(parseArtifactDir, "chat.wasm.br"), []byte{}, 0o644); parseErr2 != nil {
		parseT.Fatalf("WriteFile zero br: %v", parseErr2)
	}

	for _, parseCase := range []struct {
		name string
		req  *http.Request
	}{
		{name: "post method", req: httptest.NewRequest(http.MethodPost, "http://example.com/app/chat.wasm", nil)},
		{name: "non wasm", req: httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.js", nil)},
		{name: "invalid br query", req: httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm?br=maybe", nil)},
		{name: "path traversal", req: httptest.NewRequest(http.MethodGet, "http://example.com/../chat.wasm", nil)},
		{name: "zero length artifact", req: httptest.NewRequest(http.MethodGet, "http://example.com/app/chat.wasm", nil)},
	} {
		if parseServed := parseTryServeBrotliWASM(httptest.NewRecorder(), parseCase.req, parseRootDir); parseServed {
			parseT.Fatalf("%s: expected Brotli helper to reject request", parseCase.name)
		}
	}
}
