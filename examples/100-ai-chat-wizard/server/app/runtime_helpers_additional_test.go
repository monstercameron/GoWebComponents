package app

import (
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestRuntimeConfigHelperBranches(t *testing.T) {
	if loaded := loadFirstDotEnv(func(path string) error {
		if path == "second.env" {
			return nil
		}
		return os.ErrNotExist
	}, []string{"first.env", "second.env", "third.env"}); loaded != "second.env" {
		t.Fatalf("loadFirstDotEnv() = %q, want second.env", loaded)
	}
	if loaded := loadFirstDotEnv(func(string) error { return os.ErrNotExist }, []string{"missing.env"}); loaded != "" {
		t.Fatalf("loadFirstDotEnv() = %q, want empty string", loaded)
	}

	config := readServerRuntimeConfig(func(key string) string {
		switch key {
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
	if config.openAIAPIKey != "openai-key" || config.anthropicAPIKey != "anthropic-key" || config.cerebrasAPIKey != "cerebras-key" {
		t.Fatalf("unexpected API key config: %+v", config)
	}
	if got := strings.Join(config.stubProviders, ","); got != "openai,anthropic,cerebras" {
		t.Fatalf("stubProviders = %q, want openai,anthropic,cerebras", got)
	}
	if config.defaultModel != "gpt-5.4-mini" || config.addr != "0.0.0.0:9000" || config.dbPath != "runtime/chat.db" || config.authSecret != "secret" || config.usagePremiumPct != 6.25 {
		t.Fatalf("unexpected runtime config: %+v", config)
	}

	defaults := readServerRuntimeConfig(func(key string) string {
		if key == "OPENAI_MODEL" {
			return " gpt-5.4 "
		}
		return ""
	})
	if defaults.defaultModel != "gpt-5.4" {
		t.Fatalf("default model fallback = %q, want gpt-5.4", defaults.defaultModel)
	}
	if defaults.addr != "127.0.0.1:8095" {
		t.Fatalf("default addr = %q, want 127.0.0.1:8095", defaults.addr)
	}
	if defaults.dbPath != "examples/100-ai-chat-wizard/bin/runtime/chat_history.db" {
		t.Fatalf("default dbPath = %q", defaults.dbPath)
	}
	if defaults.usagePremiumPct != 5 {
		t.Fatalf("default usage premium pct = %.2f, want 5.00", defaults.usagePremiumPct)
	}

	if got := splitAndTrim(" one, two ,, three "); strings.Join(got, "|") != "one|two|three" {
		t.Fatalf("splitAndTrim() = %q, want one|two|three", strings.Join(got, "|"))
	}
	if got := splitAndTrim("   "); got != nil {
		t.Fatalf("splitAndTrim(blank) = %#v, want nil", got)
	}
	if got := parseUsagePremiumPercent("12.5", 5); got != 12.5 {
		t.Fatalf("parseUsagePremiumPercent(valid) = %.2f, want 12.50", got)
	}
	if got := parseUsagePremiumPercent("-2", 5); got != 5 {
		t.Fatalf("parseUsagePremiumPercent(negative) = %.2f, want fallback 5.00", got)
	}
	if got := parseUsagePremiumPercent("oops", 5); got != 5 {
		t.Fatalf("parseUsagePremiumPercent(invalid) = %.2f, want fallback 5.00", got)
	}
	if got := parseUsagePremiumPercent("5000", 5); got != 1000 {
		t.Fatalf("parseUsagePremiumPercent(clamped) = %.2f, want 1000.00", got)
	}
}

func TestChatShellRoutingHelpers(t *testing.T) {
	t.Run("shouldServeClientShell", func(t *testing.T) {
		cases := map[string]bool{
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
		for path, want := range cases {
			if got := shouldServeClientShell(path); got != want {
				t.Fatalf("shouldServeClientShell(%q) = %v, want %v", path, got, want)
			}
		}
	})

	t.Run("cloneRequestWithPath", func(t *testing.T) {
		req := httptest.NewRequest(http.MethodGet, "http://example.com/old/path?br=true", nil)
		cloned := cloneRequestWithPath(req, "/app/chat.wasm")
		if cloned == req {
			t.Fatal("expected cloneRequestWithPath to return a cloned request")
		}
		if cloned.URL.Path != "/app/chat.wasm" || cloned.URL.RawPath != "/app/chat.wasm" {
			t.Fatalf("unexpected cloned path: %q raw=%q", cloned.URL.Path, cloned.URL.RawPath)
		}
		if cloned.RequestURI != "/app/chat.wasm?br=true" {
			t.Fatalf("unexpected RequestURI: %q", cloned.RequestURI)
		}
		if req.URL.Path != "/old/path" {
			t.Fatalf("expected original request path to remain unchanged, got %q", req.URL.Path)
		}
	})

	t.Run("chatShellHandler", func(t *testing.T) {
		var servedPaths []string
		fileServer := http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
			servedPaths = append(servedPaths, r.URL.Path)
			_, _ = w.Write([]byte("file:" + r.URL.Path))
		})
		handler := chatShellHandler(fileServer)

		bootstrapWriter := httptest.NewRecorder()
		setChatUsagePremiumPercent(8.25)
		handler.ServeHTTP(bootstrapWriter, httptest.NewRequest(http.MethodGet, "http://example.com/chat-bootstrap.js", nil))
		if !strings.Contains(bootstrapWriter.Body.String(), "loadChatWasm") {
			t.Fatalf("expected bootstrap route to serve JS, got %q", bootstrapWriter.Body.String())
		}
		if !strings.Contains(bootstrapWriter.Body.String(), "window.__relaydesk_usage_premium_percent = 8.250000;") {
			t.Fatalf("expected bootstrap route to include usage premium percent, got %q", bootstrapWriter.Body.String())
		}

		assetWriter := httptest.NewRecorder()
		handler.ServeHTTP(assetWriter, httptest.NewRequest(http.MethodGet, "http://example.com/chat.wasm?br=true", nil))
		if len(servedPaths) == 0 || servedPaths[0] != "/app/chat.wasm" {
			t.Fatalf("rewritten file server path = %#v, want /app/chat.wasm", servedPaths)
		}

		shellWriter := httptest.NewRecorder()
		handler.ServeHTTP(shellWriter, httptest.NewRequest(http.MethodGet, "http://example.com/thread/42", nil))
		if !strings.Contains(shellWriter.Body.String(), "chat-bootstrap.js") {
			t.Fatalf("expected shell route to return app shell, got %q", shellWriter.Body.String())
		}

		staticWriter := httptest.NewRecorder()
		handler.ServeHTTP(staticWriter, httptest.NewRequest(http.MethodGet, "http://example.com/static/app.css", nil))
		if len(servedPaths) < 2 || servedPaths[1] != "/static/app.css" {
			t.Fatalf("fallback file server path = %#v, want /static/app.css", servedPaths)
		}
	})
}

func TestChatBootstrapLoaderTracksDownloadPhase(t *testing.T) {
	if !strings.Contains(chatShellHTML, `id="boot-heading"`) {
		t.Fatal("expected boot shell html to expose a dynamic heading target")
	}
	if !strings.Contains(chatShellHTML, `id="boot-stage"`) {
		t.Fatal("expected boot shell html to expose a dynamic stage target")
	}
	if !strings.Contains(chatShellHTML, `id="boot-detail"`) {
		t.Fatal("expected boot shell html to expose a dynamic detail target")
	}
	if !strings.Contains(chatShellHTML, `id="boot-fin-label"`) || !strings.Contains(chatShellHTML, `id="boot-fin-heading"`) {
		t.Fatal("expected boot shell html to expose finalizing text targets")
	}
	if !strings.Contains(chatBootstrapJS, `bootProgressFill.classList.toggle('is-indeterminate', isIndeterminate);`) {
		t.Fatal("expected bootstrap js to toggle the indeterminate progress state")
	}
	if !strings.Contains(chatBootstrapJS, `bootShell.classList.toggle('is-finalizing', isFinalizing);`) {
		t.Fatal("expected bootstrap js to gate finalizing separately from indeterminate loading")
	}
	if !strings.Contains(chatBootstrapJS, `setBootText('boot-heading', statusText);`) {
		t.Fatal("expected bootstrap js to update the boot heading text")
	}
	if !strings.Contains(chatBootstrapJS, `setBootText('boot-detail', detailText);`) {
		t.Fatal("expected bootstrap js to update the boot detail text")
	}
	if !strings.Contains(chatBootstrapJS, `setBootText('boot-fin-heading', statusText);`) {
		t.Fatal("expected bootstrap js to update the finalizing heading text")
	}
}

func TestResolveStaticDirectoriesAndLoadStoreQueries(t *testing.T) {
	wd, err := os.Getwd()
	if err != nil {
		t.Fatalf("Getwd: %v", err)
	}
	tempDir := t.TempDir()
	if err := os.MkdirAll(filepath.Join(tempDir, "examples", "100-ai-chat-wizard", "bin", "client"), 0o755); err != nil {
		t.Fatalf("MkdirAll client dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Join(tempDir, "examples", "static"), 0o755); err != nil {
		t.Fatalf("MkdirAll static dir: %v", err)
	}
	if err := os.Chdir(tempDir); err != nil {
		t.Fatalf("Chdir temp dir: %v", err)
	}
	defer func() {
		if chdirErr := os.Chdir(wd); chdirErr != nil {
			t.Fatalf("restore cwd: %v", chdirErr)
		}
	}()

	clientDir, sharedDir := resolveStaticDirectories()
	if clientDir != "examples/100-ai-chat-wizard/bin/client" {
		t.Fatalf("clientDir = %q, want examples/100-ai-chat-wizard/bin/client", clientDir)
	}
	if sharedDir != "examples/static" {
		t.Fatalf("sharedDir = %q, want examples/static", sharedDir)
	}

	queries, err := loadStoreQueries()
	if err != nil {
		t.Fatalf("loadStoreQueries: %v", err)
	}
	if !strings.Contains(strings.ToUpper(queries.schema), "CREATE TABLE") {
		t.Fatalf("schema query missing CREATE TABLE: %q", queries.schema)
	}
	if !strings.Contains(strings.ToUpper(queries.createUser), "INSERT") {
		t.Fatalf("createUser query missing INSERT: %q", queries.createUser)
	}
	if !strings.Contains(strings.ToUpper(queries.listModelCatalog), "SELECT") {
		t.Fatalf("listModelCatalog query missing SELECT: %q", queries.listModelCatalog)
	}
}
