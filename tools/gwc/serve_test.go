package main

import (
	"errors"
	"net/http"
	"net/http/httptest"
	"os"
	"path/filepath"
	"strings"
	"testing"
)

func TestServeFixtureFlagSetAcceptsAndRejectsExpectedValues(t *testing.T) {
	t.Run("accepts normalized route", func(t *testing.T) {
		var flag serveFixtureFlag
		if err := flag.Set("api/user/123=fixtures/user.json"); err != nil {
			t.Fatalf("set fixture flag: %v", err)
		}
		if len(flag.values) != 1 {
			t.Fatalf("expected one fixture, got %+v", flag.values)
		}
		if flag.values[0].route != "/api/user/123" || flag.values[0].path != "fixtures/user.json" {
			t.Fatalf("unexpected normalized fixture value: %+v", flag.values[0])
		}
		if got := flag.String(); got != "/api/user/123=fixtures/user.json" {
			t.Fatalf("unexpected fixture flag string %q", got)
		}
	})

	t.Run("blank is ignored", func(t *testing.T) {
		var flag serveFixtureFlag
		if err := flag.Set("   "); err != nil {
			t.Fatalf("expected blank fixture to be ignored, got %v", err)
		}
		if len(flag.values) != 0 {
			t.Fatalf("expected no fixtures after blank input, got %+v", flag.values)
		}
	})

	t.Run("rejects malformed values", func(t *testing.T) {
		cases := []string{
			"missing-separator",
			"/=fixture.json",
			"/api/user/123=",
		}
		for _, value := range cases {
			var flag serveFixtureFlag
			if err := flag.Set(value); err == nil {
				t.Fatalf("expected invalid fixture %q to fail", value)
			}
		}
	})
}

func TestResolveServeConfigResolvesRuntimeAndFixturePaths(t *testing.T) {
	root := t.TempDir()
	fixturePath := filepath.Join(root, "fixtures", "user-123.json")
	wasmPath := filepath.Join(root, "out", "main.wasm")
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(wasmPath), 0o755); err != nil {
		t.Fatalf("mkdir wasm dir: %v", err)
	}
	if err := os.WriteFile(fixturePath, []byte(`{"ok":true}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.WriteFile(wasmPath, []byte("wasm"), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}

	previousResolve := serveResolveWasmExecPath
	t.Cleanup(func() { serveResolveWasmExecPath = previousResolve })
	serveResolveWasmExecPath = func() (string, error) {
		return filepath.Join(root, "goroot", "wasm_exec.js"), nil
	}

	config, err := resolveServeConfig(serveConfig{
		rootPath:      root,
		wasmFile:      filepath.Join("out", "main.wasm"),
		wasmExecRoute: "/wasm_exec.js",
		fixtures:      []serveFixture{{route: "api/user/123", path: filepath.Join("fixtures", "user-123.json")}},
		host:          "",
		port:          "",
		indexFile:     "",
	})
	if err != nil {
		t.Fatalf("resolve serve config: %v", err)
	}

	if config.rootPath != root {
		t.Fatalf("expected root %q, got %q", root, config.rootPath)
	}
	if config.indexFile != "index.html" {
		t.Fatalf("expected default index.html, got %q", config.indexFile)
	}
	if config.host != defaultHost || config.port != defaultPort {
		t.Fatalf("expected default host/port %s:%s, got %s:%s", defaultHost, defaultPort, config.host, config.port)
	}
	if config.wasmRoute != "/main.wasm" || config.wasmFile != wasmPath {
		t.Fatalf("expected wasm route and path to resolve, got route=%q path=%q", config.wasmRoute, config.wasmFile)
	}
	if config.wasmExecRoute != "/wasm_exec.js" || !strings.HasSuffix(config.wasmExecFile, filepath.Join("goroot", "wasm_exec.js")) {
		t.Fatalf("expected wasm_exec route and path to resolve, got route=%q path=%q", config.wasmExecRoute, config.wasmExecFile)
	}
	if len(config.fixtures) != 1 || config.fixtures[0].route != "/api/user/123" || config.fixtures[0].path != fixturePath {
		t.Fatalf("expected fixture route to resolve, got %+v", config.fixtures)
	}
}

func TestResolveServeConfigRejectsInvalidInputsAndSupportsEdgePaths(t *testing.T) {
	t.Run("rejects non-directory root", func(t *testing.T) {
		root := t.TempDir()
		filePath := filepath.Join(root, "not-a-dir.txt")
		if err := os.WriteFile(filePath, []byte("x"), 0o644); err != nil {
			t.Fatalf("write file root fixture: %v", err)
		}
		if _, err := resolveServeConfig(serveConfig{rootPath: filePath}); err == nil || !strings.Contains(err.Error(), "not a directory") {
			t.Fatalf("expected non-directory root error, got %v", err)
		}
	})

	t.Run("rejects wasm_exec resolution errors when enabled", func(t *testing.T) {
		root := t.TempDir()
		previousResolve := serveResolveWasmExecPath
		t.Cleanup(func() { serveResolveWasmExecPath = previousResolve })
		serveResolveWasmExecPath = func() (string, error) {
			return "", os.ErrNotExist
		}

		if _, err := resolveServeConfig(serveConfig{rootPath: root, wasmExecRoute: "/wasm_exec.js"}); err == nil || !strings.Contains(err.Error(), "resolve wasm_exec.js for serve") {
			t.Fatalf("expected wasm_exec resolution error, got %v", err)
		}
	})

	t.Run("directory requests resolve to index file", func(t *testing.T) {
		root := t.TempDir()
		nestedDir := filepath.Join(root, "nested")
		if err := os.MkdirAll(nestedDir, 0o755); err != nil {
			t.Fatalf("mkdir nested dir: %v", err)
		}
		if err := os.WriteFile(filepath.Join(nestedDir, "index.html"), []byte("nested"), 0o644); err != nil {
			t.Fatalf("write nested index: %v", err)
		}
		config := serveConfig{rootPath: root, indexFile: "index.html"}
		got, err := config.resolveRequestPath("/nested/")
		if err != nil {
			t.Fatalf("resolve nested request path: %v", err)
		}
		want := filepath.Join(nestedDir, "index.html")
		if got != want {
			t.Fatalf("expected nested index path %q, got %q", want, got)
		}
	})
}

func TestServeHandlerServesStaticFixtureAndRuntimeAssets(t *testing.T) {
	root := t.TempDir()
	indexPath := filepath.Join(root, "index.html")
	fixturePath := filepath.Join(root, "fixtures", "user-123.json")
	wasmPath := filepath.Join(root, "out", "main.wasm")
	wasmExecPath := filepath.Join(root, "runtime", "wasm_exec.js")
	if err := os.MkdirAll(filepath.Dir(fixturePath), 0o755); err != nil {
		t.Fatalf("mkdir fixture dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(wasmPath), 0o755); err != nil {
		t.Fatalf("mkdir wasm dir: %v", err)
	}
	if err := os.MkdirAll(filepath.Dir(wasmExecPath), 0o755); err != nil {
		t.Fatalf("mkdir wasm_exec dir: %v", err)
	}
	if err := os.WriteFile(indexPath, []byte("<html><body>ok</body></html>"), 0o644); err != nil {
		t.Fatalf("write index: %v", err)
	}
	if err := os.WriteFile(fixturePath, []byte(`{"id":123}`), 0o644); err != nil {
		t.Fatalf("write fixture: %v", err)
	}
	if err := os.WriteFile(wasmPath, []byte("wasm"), 0o644); err != nil {
		t.Fatalf("write wasm: %v", err)
	}
	if err := os.WriteFile(wasmExecPath, []byte("console.log('runtime');"), 0o644); err != nil {
		t.Fatalf("write wasm_exec: %v", err)
	}

	config := serveConfig{
		rootPath:      root,
		indexFile:     "index.html",
		host:          defaultHost,
		port:          defaultPort,
		wasmRoute:     "/main.wasm",
		wasmFile:      wasmPath,
		wasmExecRoute: "/wasm_exec.js",
		wasmExecFile:  wasmExecPath,
		fixtures:      []serveFixture{{route: "/api/user/123", path: fixturePath}},
	}
	handler := config.newHandler()

	healthz := httptest.NewRecorder()
	handler.ServeHTTP(healthz, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if healthz.Code != http.StatusOK || !strings.Contains(healthz.Body.String(), `"ok":true`) {
		t.Fatalf("expected /healthz JSON ok payload, got code=%d body=%q", healthz.Code, healthz.Body.String())
	}

	fixture := httptest.NewRecorder()
	handler.ServeHTTP(fixture, httptest.NewRequest(http.MethodGet, "/api/user/123", nil))
	if fixture.Code != http.StatusOK || fixture.Body.String() != `{"id":123}` {
		t.Fatalf("expected fixture payload, got code=%d body=%q", fixture.Code, fixture.Body.String())
	}
	if fixture.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected fixture no-store cache header, got %q", fixture.Header().Get("Cache-Control"))
	}

	wasm := httptest.NewRecorder()
	handler.ServeHTTP(wasm, httptest.NewRequest(http.MethodGet, "/main.wasm", nil))
	if wasm.Code != http.StatusOK || wasm.Body.String() != "wasm" {
		t.Fatalf("expected wasm payload, got code=%d body=%q", wasm.Code, wasm.Body.String())
	}
	if wasm.Header().Get("Content-Type") != "application/wasm" {
		t.Fatalf("expected application/wasm content type, got %q", wasm.Header().Get("Content-Type"))
	}
	if wasm.Header().Get("Cache-Control") != "no-store" {
		t.Fatalf("expected wasm no-store cache header, got %q", wasm.Header().Get("Cache-Control"))
	}

	wasmExec := httptest.NewRecorder()
	handler.ServeHTTP(wasmExec, httptest.NewRequest(http.MethodGet, "/wasm_exec.js", nil))
	if wasmExec.Code != http.StatusOK || !strings.Contains(wasmExec.Body.String(), "runtime") {
		t.Fatalf("expected wasm_exec payload, got code=%d body=%q", wasmExec.Code, wasmExec.Body.String())
	}

	index := httptest.NewRecorder()
	handler.ServeHTTP(index, httptest.NewRequest(http.MethodGet, "/", nil))
	if index.Code != http.StatusOK || !strings.Contains(index.Body.String(), "<body>ok</body>") {
		t.Fatalf("expected index payload, got code=%d body=%q", index.Code, index.Body.String())
	}

	resolvedPath, err := config.resolveRequestPath("/../secret.txt")
	if err != nil {
		t.Fatalf("expected cleaned path traversal request to stay inside root, got %v", err)
	}
	if resolvedPath != filepath.Join(root, "secret.txt") {
		t.Fatalf("expected cleaned request path to stay within root, got %q", resolvedPath)
	}
}

func TestRunServeNoWasmExecBypassesRuntimeResolution(t *testing.T) {
	root := t.TempDir()
	previousResolve := serveResolveWasmExecPath
	t.Cleanup(func() { serveResolveWasmExecPath = previousResolve })
	serveResolveWasmExecPath = func() (string, error) {
		return "", errors.New("should not resolve wasm_exec when disabled")
	}

	launcher := launcher{}
	err := launcher.runServe([]string{"-root", root, "-no-wasm-exec", "-host", "[", "-port", "0"})
	if err == nil {
		t.Fatal("expected invalid listen address error")
	}
	if strings.Contains(err.Error(), "should not resolve wasm_exec") {
		t.Fatalf("expected -no-wasm-exec to bypass runtime resolution, got %v", err)
	}
	if !strings.Contains(strings.ToLower(err.Error()), "listen") {
		t.Fatalf("expected listen failure after bypassing runtime resolution, got %v", err)
	}
}

func FuzzServeResolveRequestPathKeepsPathsContained(f *testing.F) {
	root, err := os.MkdirTemp("", "gwc-serve-fuzz-")
	if err != nil {
		f.Fatalf("mkdir temp root: %v", err)
	}
	defer os.RemoveAll(root)
	if err := os.MkdirAll(filepath.Join(root, "nested"), 0o755); err != nil {
		f.Fatalf("mkdir nested dir: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "index.html"), []byte("root"), 0o644); err != nil {
		f.Fatalf("write root index: %v", err)
	}
	if err := os.WriteFile(filepath.Join(root, "nested", "index.html"), []byte("nested"), 0o644); err != nil {
		f.Fatalf("write nested index: %v", err)
	}

	config := serveConfig{rootPath: root, indexFile: "index.html"}
	for _, seed := range []string{"", "/", "/nested/", "/index.html", "/../secret.txt", `/nested\child.txt`, "/./a/../../index.html"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, requestPath string) {
		resolvedPath, err := config.resolveRequestPath(requestPath)
		if err != nil {
			return
		}
		cleanRoot := filepath.Clean(root)
		cleanResolved := filepath.Clean(resolvedPath)
		if cleanResolved != cleanRoot && !strings.HasPrefix(cleanResolved, cleanRoot+string(filepath.Separator)) {
			t.Fatalf("resolved path escaped root: request=%q resolved=%q root=%q", requestPath, cleanResolved, cleanRoot)
		}
	})
}

func FuzzServeFixtureFlagSetInvariants(f *testing.F) {
	for _, seed := range []string{"", "api/user/123=fixtures/user.json", "/=x", "missing-separator", "/api=user.json"} {
		f.Add(seed)
	}

	f.Fuzz(func(t *testing.T, value string) {
		var flag serveFixtureFlag
		err := flag.Set(value)
		if err != nil {
			return
		}
		for _, fixture := range flag.values {
			if fixture.route == "" || !strings.HasPrefix(fixture.route, "/") || fixture.route == "/" {
				t.Fatalf("invalid accepted fixture route %#v for input %q", fixture, value)
			}
			if strings.TrimSpace(fixture.path) == "" {
				t.Fatalf("invalid accepted fixture path %#v for input %q", fixture, value)
			}
		}
	})
}
