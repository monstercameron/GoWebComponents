package main

import (
	"encoding/json"
	"errors"
	"flag"
	"fmt"
	"mime"
	"net"
	"net/http"
	"os"
	"path/filepath"
	"strings"
)

var runServeCommand = func(l launcher, args []string) error {
	return l.runServe(args)
}

var serveResolveWasmExecPath = resolveWasmExecPath

type serveConfig struct {
	rootPath      string
	indexFile     string
	host          string
	port          string
	wasmRoute     string
	wasmFile      string
	wasmExecRoute string
	wasmExecFile  string
	fixtures      []serveFixture
}

type serveFixture struct {
	route string
	path  string
}

type serveFixtureFlag struct {
	values []serveFixture
}

func (parseF *serveFixtureFlag) String() string {
	parseParts := make([]string, 0, len(parseF.values))
	for _, parseValue := range parseF.values {
		parseParts = append(parseParts, parseValue.route+"="+parseValue.path)
	}
	return strings.Join(parseParts, ",")
}

func (parseF *serveFixtureFlag) Set(parseValue string) error {
	parseRaw := strings.TrimSpace(parseValue)
	if parseRaw == "" {
		return nil
	}
	parseParts := strings.SplitN(parseRaw, "=", 2)
	if len(parseParts) != 2 {
		return fmt.Errorf("fixture route must use /route=path syntax: %q", parseValue)
	}
	parseRoute := normalizeServeRoute(parseParts[0])
	if parseRoute == "" || parseRoute == "/" {
		return fmt.Errorf("fixture route must be a non-root absolute path: %q", parseValue)
	}
	parsePath := strings.TrimSpace(parseParts[1])
	if parsePath == "" {
		return fmt.Errorf("fixture file path cannot be empty for route %q", parseRoute)
	}
	parseF.values = append(parseF.values, serveFixture{route: parseRoute, path: parsePath})
	return nil
}

func (parseL launcher) runServe(parseArgs []string) error {
	parseFs := flag.NewFlagSet("serve", flag.ContinueOnError)
	parseFs.SetOutput(os.Stdout)
	parseRoot := parseFs.String("root", "", "Root directory to serve; defaults to the current working directory")
	parseIndex := parseFs.String("index", "index.html", "Index file served for / and directory requests")
	parseHost := parseFs.String("host", defaultHost, "Host to bind")
	parsePort := parseFs.String("port", defaultPort, "Port to bind")
	parseWasmRoute := parseFs.String("wasm-route", "/main.wasm", "Optional route path for a built wasm artifact")
	parseWasmFile := parseFs.String("wasm-file", "", "Optional path to the wasm artifact served at -wasm-route")
	parseWasmExecRoute := parseFs.String("wasm-exec-route", "/wasm_exec.js", "Route path for the active Go toolchain wasm_exec.js helper")
	parseDisableWasmExec := parseFs.Bool("no-wasm-exec", false, "Disable automatic serving of wasm_exec.js")
	parseJsonOutput := parseFs.Bool("json", false, "Emit the serve startup result as JSON when managed by the launcher envelope")
	_ = parseJsonOutput
	var parseFixtures serveFixtureFlag
	parseFs.Var(&parseFixtures, "fixture-json", "Serve a JSON fixture file at a route using /route=path; repeatable")
	if parseErr := parseFs.Parse(parseArgs); parseErr != nil {
		if errors.Is(parseErr, flag.ErrHelp) {
			return nil
		}
		return parseErr
	}
	parseResolvedWasmExecRoute := *parseWasmExecRoute
	if *parseDisableWasmExec {
		parseResolvedWasmExecRoute = ""
	}

	parseConfig, parseErr2 := resolveServeConfig(serveConfig{
		rootPath:      *parseRoot,
		indexFile:     *parseIndex,
		host:          *parseHost,
		port:          *parsePort,
		wasmRoute:     *parseWasmRoute,
		wasmFile:      *parseWasmFile,
		wasmExecRoute: parseResolvedWasmExecRoute,
		fixtures:      parseFixtures.values,
	})
	if parseErr2 != nil {
		return parseErr2
	}

	// Enforce one live serve per (host, port): terminate any predecessor still
	// holding the port instead of failing to bind alongside a runaway orphan.
	parseSingleton, parseErr2 := parseL.acquireLauncherSingleton("serve", parseConfig.host, parseConfig.port)
	if parseErr2 != nil {
		return parseErr2
	}
	defer parseSingleton.Release()

	parseListener, parseErr2 := net.Listen("tcp", joinHostPort(parseConfig.host, parseConfig.port))
	if parseErr2 != nil {
		return fmt.Errorf("listen on %s: %w", joinHostPort(parseConfig.host, parseConfig.port), parseErr2)
	}
	defer parseListener.Close()

	fmt.Printf("GWC serve listening on http://%s\n", joinHostPort(parseConfig.host, parseConfig.port))
	fmt.Printf("  root: %s\n", parseConfig.rootPath)
	if parseConfig.wasmFile != "" {
		fmt.Printf("  wasm: %s -> %s\n", parseConfig.wasmRoute, parseConfig.wasmFile)
	}
	if parseConfig.wasmExecFile != "" {
		fmt.Printf("  wasm_exec.js: %s -> %s\n", parseConfig.wasmExecRoute, parseConfig.wasmExecFile)
	}
	for _, parseFixture := range parseConfig.fixtures {
		fmt.Printf("  fixture: %s -> %s\n", parseFixture.route, parseFixture.path)
	}

	parseServer := &http.Server{
		Addr:    joinHostPort(parseConfig.host, parseConfig.port),
		Handler: parseConfig.newHandler(),
	}
	return parseServer.Serve(parseListener)
}

func resolveServeConfig(parseConfig serveConfig) (serveConfig, error) {
	parseRootPath := strings.TrimSpace(parseConfig.rootPath)
	if parseRootPath == "" {
		parseCwd, parseErr := os.Getwd()
		if parseErr != nil {
			return serveConfig{}, fmt.Errorf("resolve serve root from cwd: %w", parseErr)
		}
		parseRootPath = parseCwd
	}
	parseAbsRoot, parseErr2 := filepath.Abs(parseRootPath)
	if parseErr2 != nil {
		return serveConfig{}, fmt.Errorf("resolve serve root: %w", parseErr2)
	}
	parseInfo, parseErr2 := os.Stat(parseAbsRoot)
	if parseErr2 != nil {
		return serveConfig{}, fmt.Errorf("stat serve root: %w", parseErr2)
	}
	if !parseInfo.IsDir() {
		return serveConfig{}, fmt.Errorf("serve root is not a directory: %s", parseAbsRoot)
	}

	parseResolved := serveConfig{
		rootPath:  parseAbsRoot,
		indexFile: strings.TrimSpace(parseConfig.indexFile),
		host:      firstNonEmpty(strings.TrimSpace(parseConfig.host), defaultHost),
		port:      firstNonEmpty(strings.TrimSpace(parseConfig.port), defaultPort),
	}
	if parseResolved.indexFile == "" {
		parseResolved.indexFile = "index.html"
	}

	if parseRoute := normalizeServeRoute(parseConfig.wasmRoute); parseRoute != "" {
		parseResolved.wasmRoute = parseRoute
	}
	if parseFile := strings.TrimSpace(parseConfig.wasmFile); parseFile != "" {
		parseResolvedPath, parseErr3 := resolveServeFilePath(parseAbsRoot, parseFile)
		if parseErr3 != nil {
			return serveConfig{}, fmt.Errorf("resolve wasm file: %w", parseErr3)
		}
		parseResolved.wasmFile = parseResolvedPath
		if parseResolved.wasmRoute == "" {
			parseResolved.wasmRoute = "/main.wasm"
		}
	}

	if parseRoute2 := normalizeServeRoute(parseConfig.wasmExecRoute); parseRoute2 != "" {
		parseResolved.wasmExecRoute = parseRoute2
		parseWasmExecPath, parseErr4 := serveResolveWasmExecPath()
		if parseErr4 != nil {
			return serveConfig{}, fmt.Errorf("resolve wasm_exec.js for serve: %w", parseErr4)
		}
		parseResolved.wasmExecFile = parseWasmExecPath
	}

	if len(parseConfig.fixtures) > 0 {
		parseResolved.fixtures = make([]serveFixture, 0, len(parseConfig.fixtures))
		for _, parseFixture := range parseConfig.fixtures {
			parseResolvedPath2, parseErr5 := resolveServeFilePath(parseAbsRoot, parseFixture.path)
			if parseErr5 != nil {
				return serveConfig{}, fmt.Errorf("resolve JSON fixture for %s: %w", parseFixture.route, parseErr5)
			}
			parseResolved.fixtures = append(parseResolved.fixtures, serveFixture{
				route: normalizeServeRoute(parseFixture.route),
				path:  parseResolvedPath2,
			})
		}
	}

	return parseResolved, nil
}

func resolveServeFilePath(parseRootPath string, parseValue string) (string, error) {
	parseTrimmed := strings.TrimSpace(parseValue)
	if parseTrimmed == "" {
		return "", nil
	}
	if filepath.IsAbs(parseTrimmed) {
		return filepath.Clean(parseTrimmed), nil
	}
	return filepath.Clean(filepath.Join(parseRootPath, parseTrimmed)), nil
}

func normalizeServeRoute(parseRoute string) string {
	parseTrimmed := strings.TrimSpace(parseRoute)
	if parseTrimmed == "" {
		return ""
	}
	if !strings.HasPrefix(parseTrimmed, "/") {
		parseTrimmed = "/" + parseTrimmed
	}
	return parseTrimmed
}

func (parseConfig serveConfig) newHandler() http.Handler {
	parseMux := http.NewServeMux()

	parseMux.HandleFunc("/healthz", func(parseW http.ResponseWriter, parseR *http.Request) {
		writeServeJSON(parseW, http.StatusOK, map[string]bool{"ok": true})
	})

	for _, parseFixture := range parseConfig.fixtures {
		parseFixture2 := parseFixture
		parseMux.HandleFunc(parseFixture2.route, func(parseW2 http.ResponseWriter, parseR2 *http.Request) {
			parseContent, parseErr := os.ReadFile(parseFixture2.path)
			if parseErr != nil {
				http.Error(parseW2, "fixture unavailable", http.StatusInternalServerError)
				return
			}
			parseW2.Header().Set("Content-Type", "application/json")
			parseW2.Header().Set("Cache-Control", "no-store")
			_, _ = parseW2.Write(parseContent)
		})
	}

	if parseConfig.wasmRoute != "" && parseConfig.wasmFile != "" {
		parseMux.HandleFunc(parseConfig.wasmRoute, func(parseW3 http.ResponseWriter, parseR3 *http.Request) {
			serveStaticFile(parseW3, parseR3, parseConfig.wasmFile, true)
		})
	}
	if parseConfig.wasmExecRoute != "" && parseConfig.wasmExecFile != "" {
		parseMux.HandleFunc(parseConfig.wasmExecRoute, func(parseW4 http.ResponseWriter, parseR4 *http.Request) {
			serveStaticFile(parseW4, parseR4, parseConfig.wasmExecFile, false)
		})
	}

	parseMux.HandleFunc("/", func(parseW5 http.ResponseWriter, parseR5 *http.Request) {
		parseTargetPath, parseErr2 := parseConfig.resolveRequestPath(parseR5.URL.Path)
		if parseErr2 != nil {
			http.Error(parseW5, "forbidden", http.StatusForbidden)
			return
		}
		serveStaticFile(parseW5, parseR5, parseTargetPath, strings.EqualFold(filepath.Ext(parseTargetPath), ".wasm"))
	})

	return parseMux
}

func (parseConfig serveConfig) resolveRequestPath(parseRequestPath string) (string, error) {
	parseCleaned := filepath.Clean(filepath.FromSlash("/" + strings.TrimPrefix(parseRequestPath, "/")))
	parseRelative := strings.TrimPrefix(parseCleaned, string(filepath.Separator))
	if parseRelative == "." || parseRelative == "" {
		parseRelative = parseConfig.indexFile
	}
	parseTarget := filepath.Join(parseConfig.rootPath, parseRelative)
	parseInfo, parseErr := os.Stat(parseTarget)
	if parseErr == nil && parseInfo.IsDir() {
		parseTarget = filepath.Join(parseTarget, parseConfig.indexFile)
	}
	parseResolvedRoot := filepath.Clean(parseConfig.rootPath)
	parseResolvedTarget := filepath.Clean(parseTarget)
	if parseResolvedTarget != parseResolvedRoot && !strings.HasPrefix(parseResolvedTarget, parseResolvedRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes root", parseRequestPath)
	}
	// Resolve symlinks to prevent a symlink inside the root from pointing
	// outside it and escaping the prefix guard above.
	parseSymlinkRoot, parseRootSymlinkErr := filepath.EvalSymlinks(parseResolvedRoot)
	if parseRootSymlinkErr != nil {
		parseSymlinkRoot = parseResolvedRoot
	}
	parseSymlinkTarget, parseTargetSymlinkErr := filepath.EvalSymlinks(parseResolvedTarget)
	if parseTargetSymlinkErr != nil {
		// Target does not exist yet (e.g. requested file is missing); skip the
		// symlink check and let the caller's os.Stat produce a 404.
		return parseResolvedTarget, nil
	}
	parseSymlinkTarget = filepath.Clean(parseSymlinkTarget)
	if parseSymlinkTarget != parseSymlinkRoot && !strings.HasPrefix(parseSymlinkTarget, parseSymlinkRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes root via symlink", parseRequestPath)
	}
	return parseResolvedTarget, nil
}

func serveStaticFile(parseW http.ResponseWriter, parseR *http.Request, parsePath string, isNoStore bool) {
	parseInfo, parseErr := os.Stat(parsePath)
	if parseErr != nil || parseInfo.IsDir() {
		http.NotFound(parseW, parseR)
		return
	}

	parseContentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(parsePath)))
	if strings.EqualFold(filepath.Ext(parsePath), ".wasm") {
		parseContentType = "application/wasm"
	}
	if parseContentType != "" {
		parseW.Header().Set("Content-Type", parseContentType)
	}
	if isNoStore {
		parseW.Header().Set("Cache-Control", "no-store")
	} else {
		parseW.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeFile(parseW, parseR, parsePath)
}

func writeServeJSON(parseW http.ResponseWriter, parseStatus int, parsePayload any) {
	parseW.Header().Set("Content-Type", "application/json")
	parseW.Header().Set("Cache-Control", "no-store")
	parseW.WriteHeader(parseStatus)
	_ = json.NewEncoder(parseW).Encode(parsePayload)
}
