package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeConfigHelperBranches(parseT *testing.T) {
	if parseLoaded := parseLoadFirstDotEnv(func(parsePath string) error {
		if parsePath == "second.env" {
			return nil
		}
		return os.ErrNotExist
	}, []string{"first.env", "second.env", "third.env"}); parseLoaded != "second.env" {
		parseT.Fatalf("loadFirstDotEnv() = %q, want second.env", parseLoaded)
	}
	if parseLoaded2 := parseLoadFirstDotEnv(func(string) error { return os.ErrNotExist }, []string{"missing.env"}); parseLoaded2 != "" {
		parseT.Fatalf("loadFirstDotEnv() = %q, want empty string", parseLoaded2)
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
			return " openai, anthropic ,, cerebras "
		case "CHAT_MODEL":
			return " gpt-5.4-mini "
		case "LISTEN_ADDR":
			return " 0.0.0.0:9000 "
		case "CHAT_DB_PATH":
			return " runtime/chat.db "
		case "CHAT_AUTH_SECRET":
			return " secret "
		case "CHAT_USAGE_PREMIUM_PERCENT":
			return " 6.25 "
		default:
			return ""
		}
	})
	if parseConfig.openAIAPIKey != "openai-key" || parseConfig.anthropicAPIKey != "anthropic-key" || parseConfig.cerebrasAPIKey != "cerebras-key" {
		parseT.Fatalf("unexpected API key config: %+v", parseConfig)
	}
	if parseGot := strings.Join(parseConfig.stubProviders, ","); parseGot != "openai,anthropic,cerebras" {
		parseT.Fatalf("stubProviders = %q, want openai,anthropic,cerebras", parseGot)
	}
	if parseConfig.defaultModel != "gpt-5.4-mini" || parseConfig.addr != "0.0.0.0:9000" || parseConfig.dbPath != "runtime/chat.db" || parseConfig.authSecret != "secret" || parseConfig.usagePremiumPct != 6.25 {
		parseT.Fatalf("unexpected runtime config: %+v", parseConfig)
	}

	parseDefaults := parseReadServerRuntimeConfig(func(parseKey2 string) string {
		if parseKey2 == "OPENAI_MODEL" {
			return " gpt-5.4 "
		}
		return ""
	})
	if parseDefaults.defaultModel != "gpt-5.4" {
		parseT.Fatalf("default model fallback = %q, want gpt-5.4", parseDefaults.defaultModel)
	}
	if parseDefaults.addr != "127.0.0.1:8095" {
		parseT.Fatalf("default addr = %q, want 127.0.0.1:8095", parseDefaults.addr)
	}
	if parseDefaults.dbPath != "examples/100-ai-chat-wizard/bin/runtime/chat_history.db" {
		parseT.Fatalf("default dbPath = %q", parseDefaults.dbPath)
	}
	if parseDefaults.usagePremiumPct != 5 {
		parseT.Fatalf("default usage premium pct = %.2f, want 5.00", parseDefaults.usagePremiumPct)
	}

	if parseGot2 := parseSplitAndTrim(" one, two ,, three "); strings.Join(parseGot2, "|") != "one|two|three" {
		parseT.Fatalf("splitAndTrim() = %q, want one|two|three", strings.Join(parseGot2, "|"))
	}
	if parseGot3 := parseSplitAndTrim("   "); parseGot3 != nil {
		parseT.Fatalf("splitAndTrim(blank) = %#v, want nil", parseGot3)
	}
	if parseGot4 := parseUsagePremiumPercent("12.5", 5); parseGot4 != 12.5 {
		parseT.Fatalf("parseUsagePremiumPercent(valid) = %.2f, want 12.50", parseGot4)
	}
	if parseGot5 := parseUsagePremiumPercent("-2", 5); parseGot5 != 5 {
		parseT.Fatalf("parseUsagePremiumPercent(negative) = %.2f, want fallback 5.00", parseGot5)
	}
	if parseGot6 := parseUsagePremiumPercent("oops", 5); parseGot6 != 5 {
		parseT.Fatalf("parseUsagePremiumPercent(invalid) = %.2f, want fallback 5.00", parseGot6)
	}
	if parseGot7 := parseUsagePremiumPercent("5000", 5); parseGot7 != 1000 {
		parseT.Fatalf("parseUsagePremiumPercent(clamped) = %.2f, want 1000.00", parseGot7)
	}
}

func TestChatShellRoutingHelpers(parseT *testing.T) {
	parseT.Run("shouldServeClientShell", func(parseT2 *testing.T) {
		parseCases := map[string]bool{
			"":                   true,
			"/":                  true,
			"/app":               true,
			"/app/":              true,
			"/home":              true,
			"/pricing":           true,
			"/thread":            true,
			"/thread/123":        true,
			"/app/thread/123":    true,
			"/login":             true,
			"/signup":            true,
			"/logout":            true,
			"/static/app.css":    false,
			"/chat.wasm":         false,
			"/images/logo.svg":   false,
			"/unknown/deep/link": false,
		}
		for parsePath, parseWant := range parseCases {
			if parseGot := shouldServeClientShell(parsePath); parseGot != parseWant {
				parseT2.Fatalf("shouldServeClientShell(%q) = %v, want %v", parsePath, parseGot, parseWant)
			}
		}
	})

	parseT.Run("cloneRequestWithPath", func(parseT3 *testing.T) {
		parseReq := httptest.NewRequest(http.MethodGet, "http://example.com/old/path?br=true", nil)
		parseCloned := parseCloneRequestWithPath(parseReq, "/app/chat.wasm")
		if parseCloned == parseReq {
			parseT3.Fatal("expected cloneRequestWithPath to return a cloned request")
		}
		if parseCloned.URL.Path != "/app/chat.wasm" || parseCloned.URL.RawPath != "/app/chat.wasm" {
			parseT3.Fatalf("unexpected cloned path: %q raw=%q", parseCloned.URL.Path, parseCloned.URL.RawPath)
		}
		if parseCloned.RequestURI != "/app/chat.wasm?br=true" {
			parseT3.Fatalf("unexpected RequestURI: %q", parseCloned.RequestURI)
		}
		if parseReq.URL.Path != "/old/path" {
			parseT3.Fatalf("expected original request path to remain unchanged, got %q", parseReq.URL.Path)
		}
	})

	parseT.Run("chatShellHandler", func(parseT4 *testing.T) {
		var parseServedPaths []string
		parseFileServer := http.HandlerFunc(func(parseW http.ResponseWriter, parseR *http.Request) {
			parseServedPaths = append(parseServedPaths, parseR.URL.Path)
			_, _ = parseW.Write([]byte("file:" + parseR.URL.Path))
		})
		parseHandler := parseChatShellHandler(parseFileServer)

		parseBootstrapWriter := httptest.NewRecorder()
		setChatUsagePremiumPercent(8.25)
		parseHandler.ServeHTTP(parseBootstrapWriter, httptest.NewRequest(http.MethodGet, "http://example.com/chat-bootstrap.js", nil))
		if !strings.Contains(parseBootstrapWriter.Body.ParseString(), "loadChatWasm") {
			parseT4.Fatalf("expected bootstrap route to serve JS, got %q", parseBootstrapWriter.Body.ParseString())
		}
		if !strings.Contains(parseBootstrapWriter.Body.ParseString(), "window.__relaydesk_usage_premium_percent = 8.250000;") {
			parseT4.Fatalf("expected bootstrap route to include usage premium percent, got %q", parseBootstrapWriter.Body.ParseString())
		}

		parseAssetWriter := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseAssetWriter, httptest.NewRequest(http.MethodGet, "http://example.com/chat.wasm?br=true", nil))
		if len(parseServedPaths) == 0 || parseServedPaths[0] != "/app/chat.wasm" {
			parseT4.Fatalf("rewritten file server path = %#v, want /app/chat.wasm", parseServedPaths)
		}

		parseShellWriter := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseShellWriter, httptest.NewRequest(http.MethodGet, "http://example.com/thread/42", nil))
		if !strings.Contains(parseShellWriter.Body.ParseString(), "chat-bootstrap.js") {
			parseT4.Fatalf("expected shell route to return app shell, got %q", parseShellWriter.Body.ParseString())
		}

		parseStaticWriter := httptest.NewRecorder()
		parseHandler.ServeHTTP(parseStaticWriter, httptest.NewRequest(http.MethodGet, "http://example.com/static/app.css", nil))
		if len(parseServedPaths) < 2 || parseServedPaths[1] != "/static/app.css" {
			parseT4.Fatalf("fallback file server path = %#v, want /static/app.css", parseServedPaths)
		}
	})
}

func TestChatBootstrapLoaderTracksDownloadPhase(parseT *testing.T) {
	if !strings.Contains(chatShellHTML, `id="boot-heading"`) {
		parseT.Fatal("expected boot shell html to expose a dynamic heading target")
	}
	if !strings.Contains(chatShellHTML, `id="boot-stage"`) {
		parseT.Fatal("expected boot shell html to expose a dynamic stage target")
	}
	if !strings.Contains(chatShellHTML, `id="boot-detail"`) {
		parseT.Fatal("expected boot shell html to expose a dynamic detail target")
	}
	if !strings.Contains(chatShellHTML, `id="boot-fin-label"`) || !strings.Contains(chatShellHTML, `id="boot-fin-heading"`) {
		parseT.Fatal("expected boot shell html to expose finalizing text targets")
	}
	if !strings.Contains(chatBootstrapJS, `bootProgressFill.classList.toggle('is-indeterminate', isIndeterminate);`) {
		parseT.Fatal("expected bootstrap js to toggle the indeterminate progress state")
	}
	if !strings.Contains(chatBootstrapJS, `bootShell.classList.toggle('is-finalizing', isFinalizing);`) {
		parseT.Fatal("expected bootstrap js to gate finalizing separately from indeterminate loading")
	}
	if !strings.Contains(chatBootstrapJS, `setBootText('boot-heading', statusText);`) {
		parseT.Fatal("expected bootstrap js to update the boot heading text")
	}
	if !strings.Contains(chatBootstrapJS, `setBootText('boot-detail', detailText);`) {
		parseT.Fatal("expected bootstrap js to update the boot detail text")
	}
	if !strings.Contains(chatBootstrapJS, `setBootText('boot-fin-heading', statusText);`) {
		parseT.Fatal("expected bootstrap js to update the finalizing heading text")
	}
}

func TestResolveStaticDirectoriesAndLoadStoreQueries(parseT *testing.T) {
	parseWd, parseErr := os.Getwd()
	if parseErr != nil {
		parseT.Fatalf("Getwd: %v", parseErr)
	}
	parseTempDir := parseT.TempDir()
	if parseErr2 := os.MkdirAll(filepath.Join(parseTempDir, "examples", "100-ai-chat-wizard", "bin", "client"), 0o755); parseErr2 != nil {
		parseT.Fatalf("MkdirAll client dir: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(filepath.Join(parseTempDir, "examples", "static"), 0o755); parseErr3 != nil {
		parseT.Fatalf("MkdirAll static dir: %v", parseErr3)
	}
	if parseErr4 := os.Chdir(parseTempDir); parseErr4 != nil {
		parseT.Fatalf("Chdir temp dir: %v", parseErr4)
	}
	defer func() {
		if parseChdirErr := os.Chdir(parseWd); parseChdirErr != nil {
			parseT.Fatalf("restore cwd: %v", parseChdirErr)
		}
	}()

	parseClientDir, parseSharedDir := parseResolveStaticDirectories()
	if parseClientDir != "examples/100-ai-chat-wizard/bin/client" {
		parseT.Fatalf("clientDir = %q, want examples/100-ai-chat-wizard/bin/client", parseClientDir)
	}
	if parseSharedDir != "examples/static" {
		parseT.Fatalf("sharedDir = %q, want examples/static", parseSharedDir)
	}

	parseQueries, parseErr := parseLoadStoreQueries()
	if parseErr != nil {
		parseT.Fatalf("loadStoreQueries: %v", parseErr)
	}
	if !strings.Contains(strings.ToUpper(parseQueries.schema), "CREATE TABLE") {
		parseT.Fatalf("schema query missing CREATE TABLE: %q", parseQueries.schema)
	}
	if !strings.Contains(strings.ToUpper(parseQueries.parseCreateUser), "INSERT") {
		parseT.Fatalf("createUser query missing INSERT: %q", parseQueries.parseCreateUser)
	}
	if !strings.Contains(strings.ToUpper(parseQueries.parseListModelCatalog), "SELECT") {
		parseT.Fatalf("listModelCatalog query missing SELECT: %q", parseQueries.parseListModelCatalog)
	}
}
