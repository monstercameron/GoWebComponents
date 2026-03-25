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

func TestServeFixtureFlagSetAcceptsAndRejectsExpectedValues(parseT *testing.T) {
	parseT.Run("accepts normalized route", func(parseT2 *testing.T) {
		var parseFlag serveFixtureFlag
		if parseErr := parseFlag.Set("api/user/123=fixtures/user.json"); parseErr != nil {
			parseT2.Fatalf("set fixture flag: %v", parseErr)
		}
		if len(parseFlag.values) != 1 {
			parseT2.Fatalf("expected one fixture, got %+v", parseFlag.values)
		}
		if parseFlag.values[0].route != "/api/user/123" || parseFlag.values[0].path != "fixtures/user.json" {
			parseT2.Fatalf("unexpected normalized fixture value: %+v", parseFlag.values[0])
		}
		if parseGot := parseFlag.String(); parseGot != "/api/user/123=fixtures/user.json" {
			parseT2.Fatalf("unexpected fixture flag string %q", parseGot)
		}
	})

	parseT.Run("blank is ignored", func(parseT3 *testing.T) {
		var parseFlag2 serveFixtureFlag
		if parseErr2 := parseFlag2.Set("   "); parseErr2 != nil {
			parseT3.Fatalf("expected blank fixture to be ignored, got %v", parseErr2)
		}
		if len(parseFlag2.values) != 0 {
			parseT3.Fatalf("expected no fixtures after blank input, got %+v", parseFlag2.values)
		}
	})

	parseT.Run("rejects malformed values", func(parseT4 *testing.T) {
		parseCases := []string{
			"missing-separator",
			"/=fixture.json",
			"/api/user/123=",
		}
		for _, parseValue := range parseCases {
			var parseFlag3 serveFixtureFlag
			if parseErr3 := parseFlag3.Set(parseValue); parseErr3 == nil {
				parseT4.Fatalf("expected invalid fixture %q to fail", parseValue)
			}
		}
	})
}

func TestResolveServeConfigResolvesRuntimeAndFixturePaths(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseFixturePath := filepath.Join(parseRoot, "fixtures", "user-123.json")
	parseWasmPath := filepath.Join(parseRoot, "out", "main.wasm")
	if parseErr := os.MkdirAll(filepath.Dir(parseFixturePath), 0o755); parseErr != nil {
		parseT.Fatalf("mkdir fixture dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Dir(parseWasmPath), 0o755); parseErr2 != nil {
		parseT.Fatalf("mkdir wasm dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(parseFixturePath, []byte(`{"ok":true}`), 0o644); parseErr3 != nil {
		parseT.Fatalf("write fixture: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseWasmPath, []byte("wasm"), 0o644); parseErr4 != nil {
		parseT.Fatalf("write wasm: %v", parseErr4)
	}

	parsePreviousResolve := serveResolveWasmExecPath
	parseT.Cleanup(func() { serveResolveWasmExecPath = parsePreviousResolve })
	serveResolveWasmExecPath = func() (string, error) {
		return filepath.Join(parseRoot, "goroot", "wasm_exec.js"), nil
	}

	parseConfig, parseErr5 := resolveServeConfig(serveConfig{
		rootPath:      parseRoot,
		wasmFile:      filepath.Join("out", "main.wasm"),
		wasmExecRoute: "/wasm_exec.js",
		fixtures:      []serveFixture{{route: "api/user/123", path: filepath.Join("fixtures", "user-123.json")}},
		host:          "",
		port:          "",
		indexFile:     "",
	})
	if parseErr5 != nil {
		parseT.Fatalf("resolve serve config: %v", parseErr5)
	}

	if parseConfig.rootPath != parseRoot {
		parseT.Fatalf("expected root %q, got %q", parseRoot, parseConfig.rootPath)
	}
	if parseConfig.indexFile != "index.html" {
		parseT.Fatalf("expected default index.html, got %q", parseConfig.indexFile)
	}
	if parseConfig.host != defaultHost || parseConfig.port != defaultPort {
		parseT.Fatalf("expected default host/port %s:%s, got %s:%s", defaultHost, defaultPort, parseConfig.host, parseConfig.port)
	}
	if parseConfig.wasmRoute != "/main.wasm" || parseConfig.wasmFile != parseWasmPath {
		parseT.Fatalf("expected wasm route and path to resolve, got route=%q path=%q", parseConfig.wasmRoute, parseConfig.wasmFile)
	}
	if parseConfig.wasmExecRoute != "/wasm_exec.js" || !strings.HasSuffix(parseConfig.wasmExecFile, filepath.Join("goroot", "wasm_exec.js")) {
		parseT.Fatalf("expected wasm_exec route and path to resolve, got route=%q path=%q", parseConfig.wasmExecRoute, parseConfig.wasmExecFile)
	}
	if len(parseConfig.fixtures) != 1 || parseConfig.fixtures[0].route != "/api/user/123" || parseConfig.fixtures[0].path != parseFixturePath {
		parseT.Fatalf("expected fixture route to resolve, got %+v", parseConfig.fixtures)
	}
}

func TestResolveServeConfigRejectsInvalidInputsAndSupportsEdgePaths(parseT *testing.T) {
	parseT.Run("rejects non-directory root", func(parseT2 *testing.T) {
		parseRoot := parseT2.TempDir()
		parseFilePath := filepath.Join(parseRoot, "not-a-dir.txt")
		if parseErr := os.WriteFile(parseFilePath, []byte("x"), 0o644); parseErr != nil {
			parseT2.Fatalf("write file root fixture: %v", parseErr)
		}
		if _, parseErr2 := resolveServeConfig(serveConfig{rootPath: parseFilePath}); parseErr2 == nil || !strings.Contains(parseErr2.Error(), "not a directory") {
			parseT2.Fatalf("expected non-directory root error, got %v", parseErr2)
		}
	})

	parseT.Run("rejects wasm_exec resolution errors when enabled", func(parseT3 *testing.T) {
		parseRoot2 := parseT3.TempDir()
		parsePreviousResolve := serveResolveWasmExecPath
		parseT3.Cleanup(func() { serveResolveWasmExecPath = parsePreviousResolve })
		serveResolveWasmExecPath = func() (string, error) {
			return "", os.ErrNotExist
		}

		if _, parseErr3 := resolveServeConfig(serveConfig{rootPath: parseRoot2, wasmExecRoute: "/wasm_exec.js"}); parseErr3 == nil || !strings.Contains(parseErr3.Error(), "resolve wasm_exec.js for serve") {
			parseT3.Fatalf("expected wasm_exec resolution error, got %v", parseErr3)
		}
	})

	parseT.Run("directory requests resolve to index file", func(parseT4 *testing.T) {
		parseRoot3 := parseT4.TempDir()
		parseNestedDir := filepath.Join(parseRoot3, "nested")
		if parseErr4 := os.MkdirAll(parseNestedDir, 0o755); parseErr4 != nil {
			parseT4.Fatalf("mkdir nested dir: %v", parseErr4)
		}
		if parseErr5 := os.WriteFile(filepath.Join(parseNestedDir, "index.html"), []byte("nested"), 0o644); parseErr5 != nil {
			parseT4.Fatalf("write nested index: %v", parseErr5)
		}
		parseConfig := serveConfig{rootPath: parseRoot3, indexFile: "index.html"}
		parseGot, parseErr6 := parseConfig.resolveRequestPath("/nested/")
		if parseErr6 != nil {
			parseT4.Fatalf("resolve nested request path: %v", parseErr6)
		}
		parseWant := filepath.Join(parseNestedDir, "index.html")
		if parseGot != parseWant {
			parseT4.Fatalf("expected nested index path %q, got %q", parseWant, parseGot)
		}
	})
}

func TestServeHandlerServesStaticFixtureAndRuntimeAssets(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parseIndexPath := filepath.Join(parseRoot, "index.html")
	parseFixturePath := filepath.Join(parseRoot, "fixtures", "user-123.json")
	parseWasmPath := filepath.Join(parseRoot, "out", "main.wasm")
	parseWasmExecPath := filepath.Join(parseRoot, "runtime", "wasm_exec.js")
	if parseErr := os.MkdirAll(filepath.Dir(parseFixturePath), 0o755); parseErr != nil {
		parseT.Fatalf("mkdir fixture dir: %v", parseErr)
	}
	if parseErr2 := os.MkdirAll(filepath.Dir(parseWasmPath), 0o755); parseErr2 != nil {
		parseT.Fatalf("mkdir wasm dir: %v", parseErr2)
	}
	if parseErr3 := os.MkdirAll(filepath.Dir(parseWasmExecPath), 0o755); parseErr3 != nil {
		parseT.Fatalf("mkdir wasm_exec dir: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(parseIndexPath, []byte("<html><body>ok</body></html>"), 0o644); parseErr4 != nil {
		parseT.Fatalf("write index: %v", parseErr4)
	}
	if parseErr5 := os.WriteFile(parseFixturePath, []byte(`{"id":123}`), 0o644); parseErr5 != nil {
		parseT.Fatalf("write fixture: %v", parseErr5)
	}
	if parseErr6 := os.WriteFile(parseWasmPath, []byte("wasm"), 0o644); parseErr6 != nil {
		parseT.Fatalf("write wasm: %v", parseErr6)
	}
	if parseErr7 := os.WriteFile(parseWasmExecPath, []byte("console.log('runtime');"), 0o644); parseErr7 != nil {
		parseT.Fatalf("write wasm_exec: %v", parseErr7)
	}

	parseConfig := serveConfig{
		rootPath:      parseRoot,
		indexFile:     "index.html",
		host:          defaultHost,
		port:          defaultPort,
		wasmRoute:     "/main.wasm",
		wasmFile:      parseWasmPath,
		wasmExecRoute: "/wasm_exec.js",
		wasmExecFile:  parseWasmExecPath,
		fixtures:      []serveFixture{{route: "/api/user/123", path: parseFixturePath}},
	}
	parseHandler := parseConfig.newHandler()

	parseHealthz := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseHealthz, httptest.NewRequest(http.MethodGet, "/healthz", nil))
	if parseHealthz.Code != http.StatusOK || !strings.Contains(parseHealthz.Body.String(), `"ok":true`) {
		parseT.Fatalf("expected /healthz JSON ok payload, got code=%d body=%q", parseHealthz.Code, parseHealthz.Body.String())
	}

	parseFixture := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseFixture, httptest.NewRequest(http.MethodGet, "/api/user/123", nil))
	if parseFixture.Code != http.StatusOK || parseFixture.Body.String() != `{"id":123}` {
		parseT.Fatalf("expected fixture payload, got code=%d body=%q", parseFixture.Code, parseFixture.Body.String())
	}
	if parseFixture.Header().Get("Cache-Control") != "no-store" {
		parseT.Fatalf("expected fixture no-store cache header, got %q", parseFixture.Header().Get("Cache-Control"))
	}

	parseWasm := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseWasm, httptest.NewRequest(http.MethodGet, "/main.wasm", nil))
	if parseWasm.Code != http.StatusOK || parseWasm.Body.String() != "wasm" {
		parseT.Fatalf("expected wasm payload, got code=%d body=%q", parseWasm.Code, parseWasm.Body.String())
	}
	if parseWasm.Header().Get("Content-Type") != "application/wasm" {
		parseT.Fatalf("expected application/wasm content type, got %q", parseWasm.Header().Get("Content-Type"))
	}
	if parseWasm.Header().Get("Cache-Control") != "no-store" {
		parseT.Fatalf("expected wasm no-store cache header, got %q", parseWasm.Header().Get("Cache-Control"))
	}

	parseWasmExec := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseWasmExec, httptest.NewRequest(http.MethodGet, "/wasm_exec.js", nil))
	if parseWasmExec.Code != http.StatusOK || !strings.Contains(parseWasmExec.Body.String(), "runtime") {
		parseT.Fatalf("expected wasm_exec payload, got code=%d body=%q", parseWasmExec.Code, parseWasmExec.Body.String())
	}

	parseIndex := httptest.NewRecorder()
	parseHandler.ServeHTTP(parseIndex, httptest.NewRequest(http.MethodGet, "/", nil))
	if parseIndex.Code != http.StatusOK || !strings.Contains(parseIndex.Body.String(), "<body>ok</body>") {
		parseT.Fatalf("expected index payload, got code=%d body=%q", parseIndex.Code, parseIndex.Body.String())
	}

	parseResolvedPath, parseErr8 := parseConfig.resolveRequestPath("/../secret.txt")
	if parseErr8 != nil {
		parseT.Fatalf("expected cleaned path traversal request to stay inside root, got %v", parseErr8)
	}
	if parseResolvedPath != filepath.Join(parseRoot, "secret.txt") {
		parseT.Fatalf("expected cleaned request path to stay within root, got %q", parseResolvedPath)
	}
}

func TestRunServeNoWasmExecBypassesRuntimeResolution(parseT *testing.T) {
	parseRoot := parseT.TempDir()
	parsePreviousResolve := serveResolveWasmExecPath
	parseT.Cleanup(func() { serveResolveWasmExecPath = parsePreviousResolve })
	serveResolveWasmExecPath = func() (string, error) {
		return "", errors.New("should not resolve wasm_exec when disabled")
	}

	parseLauncher := launcher{}
	parseErr := parseLauncher.runServe([]string{"-root", parseRoot, "-no-wasm-exec", "-host", "[", "-port", "0"})
	if parseErr == nil {
		parseT.Fatal("expected invalid listen address error")
	}
	if strings.Contains(parseErr.Error(), "should not resolve wasm_exec") {
		parseT.Fatalf("expected -no-wasm-exec to bypass runtime resolution, got %v", parseErr)
	}
	if !strings.Contains(strings.ToLower(parseErr.Error()), "listen") {
		parseT.Fatalf("expected listen failure after bypassing runtime resolution, got %v", parseErr)
	}
}

func FuzzServeResolveRequestPathKeepsPathsContained(parseF *testing.F) {
	parseRoot, parseErr := os.MkdirTemp("", "gwc-serve-fuzz-")
	if parseErr != nil {
		parseF.Fatalf("mkdir temp root: %v", parseErr)
	}
	defer os.RemoveAll(parseRoot)
	if parseErr2 := os.MkdirAll(filepath.Join(parseRoot, "nested"), 0o755); parseErr2 != nil {
		parseF.Fatalf("mkdir nested dir: %v", parseErr2)
	}
	if parseErr3 := os.WriteFile(filepath.Join(parseRoot, "index.html"), []byte("root"), 0o644); parseErr3 != nil {
		parseF.Fatalf("write root index: %v", parseErr3)
	}
	if parseErr4 := os.WriteFile(filepath.Join(parseRoot, "nested", "index.html"), []byte("nested"), 0o644); parseErr4 != nil {
		parseF.Fatalf("write nested index: %v", parseErr4)
	}

	parseConfig := serveConfig{rootPath: parseRoot, indexFile: "index.html"}
	for _, parseSeed := range []string{"", "/", "/nested/", "/index.html", "/../secret.txt", `/nested\child.txt`, "/./a/../../index.html"} {
		parseF.Add(parseSeed)
	}

	parseF.Fuzz(func(parseT *testing.T, parseRequestPath string) {
		parseResolvedPath, parseErr5 := parseConfig.resolveRequestPath(parseRequestPath)
		if parseErr5 != nil {
			return
		}
		parseCleanRoot := filepath.Clean(parseRoot)
		parseCleanResolved := filepath.Clean(parseResolvedPath)
		if parseCleanResolved != parseCleanRoot && !strings.HasPrefix(parseCleanResolved, parseCleanRoot+string(filepath.Separator)) {
			parseT.Fatalf("resolved path escaped root: request=%q resolved=%q root=%q", parseRequestPath, parseCleanResolved, parseCleanRoot)
		}
	})
}

func FuzzServeFixtureFlagSetInvariants(parseF *testing.F) {
	for _, parseSeed := range []string{"", "api/user/123=fixtures/user.json", "/=x", "missing-separator", "/api=user.json"} {
		parseF.Add(parseSeed)
	}

	parseF.Fuzz(func(parseT *testing.T, parseValue string) {
		var parseFlag serveFixtureFlag
		parseErr := parseFlag.Set(parseValue)
		if parseErr != nil {
			return
		}
		for _, parseFixture := range parseFlag.values {
			if parseFixture.route == "" || !strings.HasPrefix(parseFixture.route, "/") || parseFixture.route == "/" {
				parseT.Fatalf("invalid accepted fixture route %#v for input %q", parseFixture, parseValue)
			}
			if strings.TrimSpace(parseFixture.path) == "" {
				parseT.Fatalf("invalid accepted fixture path %#v for input %q", parseFixture, parseValue)
			}
		}
	})
}
