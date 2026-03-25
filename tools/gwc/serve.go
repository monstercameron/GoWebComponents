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

func (f *serveFixtureFlag) String() string {
	parts := make([]string, 0, len(f.values))
	for _, value := range f.values {
		parts = append(parts, value.route+"="+value.path)
	}
	return strings.Join(parts, ",")
}

func (f *serveFixtureFlag) Set(value string) error {
	raw := strings.TrimSpace(value)
	if raw == "" {
		return nil
	}
	parts := strings.SplitN(raw, "=", 2)
	if len(parts) != 2 {
		return fmt.Errorf("fixture route must use /route=path syntax: %q", value)
	}
	route := normalizeServeRoute(parts[0])
	if route == "" || route == "/" {
		return fmt.Errorf("fixture route must be a non-root absolute path: %q", value)
	}
	path := strings.TrimSpace(parts[1])
	if path == "" {
		return fmt.Errorf("fixture file path cannot be empty for route %q", route)
	}
	f.values = append(f.values, serveFixture{route: route, path: path})
	return nil
}

func (l launcher) runServe(args []string) error {
	fs := flag.NewFlagSet("serve", flag.ContinueOnError)
	fs.SetOutput(os.Stdout)
	root := fs.String("root", "", "Root directory to serve; defaults to the current working directory")
	index := fs.String("index", "index.html", "Index file served for / and directory requests")
	host := fs.String("host", defaultHost, "Host to bind")
	port := fs.String("port", defaultPort, "Port to bind")
	wasmRoute := fs.String("wasm-route", "/main.wasm", "Optional route path for a built wasm artifact")
	wasmFile := fs.String("wasm-file", "", "Optional path to the wasm artifact served at -wasm-route")
	wasmExecRoute := fs.String("wasm-exec-route", "/wasm_exec.js", "Route path for the active Go toolchain wasm_exec.js helper")
	disableWasmExec := fs.Bool("no-wasm-exec", false, "Disable automatic serving of wasm_exec.js")
	var fixtures serveFixtureFlag
	fs.Var(&fixtures, "fixture-json", "Serve a JSON fixture file at a route using /route=path; repeatable")
	if err := fs.Parse(args); err != nil {
		if errors.Is(err, flag.ErrHelp) {
			return nil
		}
		return err
	}

	config, err := resolveServeConfig(serveConfig{
		rootPath:      *root,
		indexFile:     *index,
		host:          *host,
		port:          *port,
		wasmRoute:     *wasmRoute,
		wasmFile:      *wasmFile,
		wasmExecRoute: *wasmExecRoute,
		fixtures:      fixtures.values,
	})
	if err != nil {
		return err
	}
	if *disableWasmExec {
		config.wasmExecRoute = ""
		config.wasmExecFile = ""
	}

	listener, err := net.Listen("tcp", joinHostPort(config.host, config.port))
	if err != nil {
		return fmt.Errorf("listen on %s: %w", joinHostPort(config.host, config.port), err)
	}
	defer listener.Close()

	fmt.Printf("GWC serve listening on http://%s\n", joinHostPort(config.host, config.port))
	fmt.Printf("  root: %s\n", config.rootPath)
	if config.wasmFile != "" {
		fmt.Printf("  wasm: %s -> %s\n", config.wasmRoute, config.wasmFile)
	}
	if config.wasmExecFile != "" {
		fmt.Printf("  wasm_exec.js: %s -> %s\n", config.wasmExecRoute, config.wasmExecFile)
	}
	for _, fixture := range config.fixtures {
		fmt.Printf("  fixture: %s -> %s\n", fixture.route, fixture.path)
	}

	server := &http.Server{
		Addr:    joinHostPort(config.host, config.port),
		Handler: config.newHandler(),
	}
	return server.Serve(listener)
}

func resolveServeConfig(config serveConfig) (serveConfig, error) {
	rootPath := strings.TrimSpace(config.rootPath)
	if rootPath == "" {
		cwd, err := os.Getwd()
		if err != nil {
			return serveConfig{}, fmt.Errorf("resolve serve root from cwd: %w", err)
		}
		rootPath = cwd
	}
	absRoot, err := filepath.Abs(rootPath)
	if err != nil {
		return serveConfig{}, fmt.Errorf("resolve serve root: %w", err)
	}
	info, err := os.Stat(absRoot)
	if err != nil {
		return serveConfig{}, fmt.Errorf("stat serve root: %w", err)
	}
	if !info.IsDir() {
		return serveConfig{}, fmt.Errorf("serve root is not a directory: %s", absRoot)
	}

	resolved := serveConfig{
		rootPath:  absRoot,
		indexFile: strings.TrimSpace(config.indexFile),
		host:      firstNonEmpty(strings.TrimSpace(config.host), defaultHost),
		port:      firstNonEmpty(strings.TrimSpace(config.port), defaultPort),
	}
	if resolved.indexFile == "" {
		resolved.indexFile = "index.html"
	}

	if route := normalizeServeRoute(config.wasmRoute); route != "" {
		resolved.wasmRoute = route
	}
	if file := strings.TrimSpace(config.wasmFile); file != "" {
		resolvedPath, err := resolveServeFilePath(absRoot, file)
		if err != nil {
			return serveConfig{}, fmt.Errorf("resolve wasm file: %w", err)
		}
		resolved.wasmFile = resolvedPath
		if resolved.wasmRoute == "" {
			resolved.wasmRoute = "/main.wasm"
		}
	}

	if route := normalizeServeRoute(config.wasmExecRoute); route != "" {
		resolved.wasmExecRoute = route
		wasmExecPath, err := serveResolveWasmExecPath()
		if err != nil {
			return serveConfig{}, fmt.Errorf("resolve wasm_exec.js for serve: %w", err)
		}
		resolved.wasmExecFile = wasmExecPath
	}

	if len(config.fixtures) > 0 {
		resolved.fixtures = make([]serveFixture, 0, len(config.fixtures))
		for _, fixture := range config.fixtures {
			resolvedPath, err := resolveServeFilePath(absRoot, fixture.path)
			if err != nil {
				return serveConfig{}, fmt.Errorf("resolve JSON fixture for %s: %w", fixture.route, err)
			}
			resolved.fixtures = append(resolved.fixtures, serveFixture{
				route: normalizeServeRoute(fixture.route),
				path:  resolvedPath,
			})
		}
	}

	return resolved, nil
}

func resolveServeFilePath(rootPath string, value string) (string, error) {
	trimmed := strings.TrimSpace(value)
	if trimmed == "" {
		return "", nil
	}
	if filepath.IsAbs(trimmed) {
		return filepath.Clean(trimmed), nil
	}
	return filepath.Clean(filepath.Join(rootPath, trimmed)), nil
}

func normalizeServeRoute(route string) string {
	trimmed := strings.TrimSpace(route)
	if trimmed == "" {
		return ""
	}
	if !strings.HasPrefix(trimmed, "/") {
		trimmed = "/" + trimmed
	}
	return trimmed
}

func (config serveConfig) newHandler() http.Handler {
	mux := http.NewServeMux()

	mux.HandleFunc("/healthz", func(w http.ResponseWriter, r *http.Request) {
		writeServeJSON(w, http.StatusOK, map[string]bool{"ok": true})
	})

	for _, fixture := range config.fixtures {
		fixture := fixture
		mux.HandleFunc(fixture.route, func(w http.ResponseWriter, r *http.Request) {
			content, err := os.ReadFile(fixture.path)
			if err != nil {
				http.Error(w, "fixture unavailable", http.StatusInternalServerError)
				return
			}
			w.Header().Set("Content-Type", "application/json")
			w.Header().Set("Cache-Control", "no-store")
			_, _ = w.Write(content)
		})
	}

	if config.wasmRoute != "" && config.wasmFile != "" {
		mux.HandleFunc(config.wasmRoute, func(w http.ResponseWriter, r *http.Request) {
			serveStaticFile(w, r, config.wasmFile, true)
		})
	}
	if config.wasmExecRoute != "" && config.wasmExecFile != "" {
		mux.HandleFunc(config.wasmExecRoute, func(w http.ResponseWriter, r *http.Request) {
			serveStaticFile(w, r, config.wasmExecFile, false)
		})
	}

	mux.HandleFunc("/", func(w http.ResponseWriter, r *http.Request) {
		targetPath, err := config.resolveRequestPath(r.URL.Path)
		if err != nil {
			http.Error(w, "forbidden", http.StatusForbidden)
			return
		}
		serveStaticFile(w, r, targetPath, strings.EqualFold(filepath.Ext(targetPath), ".wasm"))
	})

	return mux
}

func (config serveConfig) resolveRequestPath(requestPath string) (string, error) {
	cleaned := filepath.Clean(filepath.FromSlash("/" + strings.TrimPrefix(requestPath, "/")))
	relative := strings.TrimPrefix(cleaned, string(filepath.Separator))
	if relative == "." || relative == "" {
		relative = config.indexFile
	}
	target := filepath.Join(config.rootPath, relative)
	info, err := os.Stat(target)
	if err == nil && info.IsDir() {
		target = filepath.Join(target, config.indexFile)
	}
	resolvedRoot := filepath.Clean(config.rootPath)
	resolvedTarget := filepath.Clean(target)
	if resolvedTarget != resolvedRoot && !strings.HasPrefix(resolvedTarget, resolvedRoot+string(filepath.Separator)) {
		return "", fmt.Errorf("path %q escapes root", requestPath)
	}
	return resolvedTarget, nil
}

func serveStaticFile(w http.ResponseWriter, r *http.Request, path string, noStore bool) {
	info, err := os.Stat(path)
	if err != nil || info.IsDir() {
		http.NotFound(w, r)
		return
	}

	contentType := mime.TypeByExtension(strings.ToLower(filepath.Ext(path)))
	if strings.EqualFold(filepath.Ext(path), ".wasm") {
		contentType = "application/wasm"
	}
	if contentType != "" {
		w.Header().Set("Content-Type", contentType)
	}
	if noStore {
		w.Header().Set("Cache-Control", "no-store")
	} else {
		w.Header().Set("Cache-Control", "no-cache")
	}
	http.ServeFile(w, r, path)
}

func writeServeJSON(w http.ResponseWriter, status int, payload interface{}) {
	w.Header().Set("Content-Type", "application/json")
	w.Header().Set("Cache-Control", "no-store")
	w.WriteHeader(status)
	_ = json.NewEncoder(w).Encode(payload)
}
